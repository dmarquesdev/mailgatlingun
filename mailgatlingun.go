package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/mailgun/mailgun-go/v4"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Domain  string `yaml:"domain"`
	Sender  string `yaml:"sender"`
	Subject string `yaml:"subject"`
	APIKey  string `yaml:"apiKey"`
}

func main() {
	configFilePath := flag.String("config", "", "Path to the configuration file")
	targetFilePath := flag.String("targets", "", "Path to the target file")
	threads := flag.Int("threads", 1, "Number of concurrent threads")
	delay := flag.Int("delay", 0, "Delay between each email sent in seconds")
	mode := flag.String("mode", "template", "Operation mode: 'template' or 'file'")
	templateName := flag.String("template", "", "Mailgun template name (required if mode is 'template')")
	messageFilePath := flag.String("messageFile", "", "Path to the message file (required if mode is 'file')")
	flag.Parse()

	// Checking required parameters based on the mode
	if *mode == "file" && *messageFilePath == "" {
		log.Fatal("Message file path is required when mode is 'file'.")
	} else if *mode != "file" && *templateName == "" {
		// If mode is not "file" and template name is missing, we treat it as "template" mode requiring a template name.
		log.Fatal("Template name is required when mode is 'template'.")
	}

	if *configFilePath == "" || *targetFilePath == "" {
		log.Fatal("Config file and target file are required.")
	}

	config, err := loadConfig(*configFilePath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Count the number of target lines to set the progress bar total
	totalTargets, err := countLines(*targetFilePath)
	if err != nil {
		log.Fatalf("Failed to count lines in target file: %v", err)
	}

	targetsChan := make(chan [2]string)
	go loadTargets(*targetFilePath, targetsChan)

	if err != nil {
		log.Fatalf("Error loading targets: %v", err)
	}

	var messageContent string
	if *mode == "file" {
		content, err := os.ReadFile(*messageFilePath)
		if err != nil {
			log.Fatalf("Failed to read message file: %v", err)
		}
		messageContent = string(content)
	}

	html := strings.HasSuffix(*messageFilePath, ".html")

	sendEmails(config, targetsChan, *threads, *delay, *mode, *templateName, messageContent, html, totalTargets)
}

// countLines returns the number of lines in the given file.
func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		if scanner.Text() != "" { // Count non-empty lines only
			lineCount++
		}
	}
	return lineCount, scanner.Err()
}

func loadConfig(path string) (Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return config, nil
}

// loadTargets reads target entries from the given file path and sends them through a channel.
func loadTargets(path string, targetsChan chan<- [2]string) {
	defer close(targetsChan)

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open target file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 1 {
			parts = append(parts, "") // No custom URL
		}
		targetsChan <- [2]string{parts[0], parts[1]}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading targets: %v", err)
	}
}

func sendEmails(config Config, targets <-chan [2]string, threads int, delay int, mode string, templateName string, messageContent string, html bool, totalTargets int) {
	mg := mailgun.NewMailgun(config.Domain, config.APIKey)
	var wg sync.WaitGroup
	bar := pb.StartNew(totalTargets)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range targets {
				var err error
				if mode == "file" {
					err = sendEmailWithFile(mg, config, target, messageContent, html)
				} else {
					// Default to template mode
					err = sendEmailWithTemplate(mg, config, target, templateName)
				}
				if err != nil {
					log.Printf("Failed to send email to %s: %v", target[0], err)
				} else {
					log.Printf("Email sent to %s", target[0])
				}
				bar.Increment()
				time.Sleep(time.Duration(delay) * time.Second)
			}
		}()
	}

	wg.Wait()
	bar.Finish()
}

func sendEmailWithTemplate(mg mailgun.Mailgun, config Config, target [2]string, templateName string) error {
	recipient, customURL := target[0], target[1]
	message := mg.NewMessage(config.Sender, config.Subject, "", recipient)
	message.SetTemplate(templateName)

	if customURL != "" {
		message.AddTemplateVariable("URL", customURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := mg.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("mailgun send error: %w", err)
	}
	return nil
}

func sendEmailWithFile(mg mailgun.Mailgun, config Config, target [2]string, messageContent string, html bool) error {
	recipient, customURL := target[0], target[1]
	// Replace the {{URL}} placeholder with the custom URL, if provided
	personalizedContent := strings.Replace(messageContent, "{{URL}}", customURL, -1)

	message := mg.NewMessage(config.Sender, config.Subject, personalizedContent, recipient)
	if html {
		message.SetHtml(personalizedContent)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, _, err := mg.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("mailgun send error: %w", err)
	}
	return nil
}
