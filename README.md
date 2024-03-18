# MailGatlinGun - A phishing mail sender using MailGun API

## Building
After installing Go, simply run
```sh
go build
```

## Running
MailGun Template Mode
```sh
./mailgatlingun -config config.yaml -targets targets.txt -template my-mailgun-template -delay 300 -threads 2 -startTime "2024-03-18 14:00:00" -timeZone "EST"
```

MailGun File Mode
```sh
./mailgatlingun -config config.yaml -targets targets.txt -mode file -messageFile template.html -delay 300 -threads 2 -startTime "2024-03-18 14:00:00" -timeZone "EST"
```

## Parameters
- `-config`: The YAML configuration file to run the script (example inside examples/config_example.yaml)
- `-targets`: A text file with the targets (recipients) that will receive the phishing email (example inside examples/targets_example.txt)
- `-mode`: Mode can be "template" (that will use a template file) or "file" (that will use a HTML or text file). This will set how the message body will be constructed - Default: template
- `-template`: The template name registered in MailGun (for "template" mode only)
- `-messageFile`: The template file that will be sent as the email body (for "file" mode only)
- `-delay`: Delay between each email in each thread (in seconds) - Default: 0
- `-threads`: Number of threads to run the script. This will define how many emails are sent at once - Default: 1
- `-startTime`: Start time in 'YYYY-MM-DD HH:mm:ss' format. Starts immediately if not provided.
- `-timeZone`: Timezone (e.g., 'EST', 'UTC'). Defaults to the OS timezone if not provided.

## Custom URL replacement
In the case of having a custom URL for each target, you can provide them in the targets file in the following format
```
email@domain.com,https://mycustomurl.com
```
MailGatlinGun will look for a `{{URL}}` placeholder inside the template (in both modes) and replace with the `https://mycustomurl.com` 
in this example, giving the flexibility to send a custom URL for each target. This can be useful for JS Injection and tracking

If Custom URL is not provided, the Phishing URL parameter will be used (read from the config YAML file).