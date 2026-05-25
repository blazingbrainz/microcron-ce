## Architecture


### Components


```
microcron-ce/
├── cmd/
│   └── main.go                  # Application entry point
├── pkg/
│   ├── cron/
│   │   └── scheduler.go         # Cron job scheduling logic
│   ├── configmap/
│   │   └── configmap.go         # ConfigMap script loader
│   ├── executor/
│   │   └── executor.go          # Script execution engine
│   └── logger/
│       └── logger.go            # Log rotation handler
├── helm/                        # Helm chart v0.2.1           
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── templates/
│   └── README.md
├── Dockerfile                    # Container image
├── go.mod                        # Go module definition
└── README.md                     # This file
```


### Project Structure


**cmd/main.go**: Main application entry point
- Loads configuration from flags
- Initializes logger, ConfigMap loader, and scheduler
- Watches ConfigMap for updates
- Handles graceful shutdown


**pkg/cron/**: Cron job scheduling
- Manages scheduled jobs using robfig/cron library
- Handles job creation, update, and removal
- Thread-safe job management


**pkg/configmap/**: ConfigMap integration
- Reads scripts from mounted ConfigMap volume
- Extracts cron schedule from script comments
- Polls for ConfigMap updates every 30 seconds
- No Kubernetes API calls required


**pkg/executor/**: Script execution
- Executes shell scripts via os/exec
- Captures stdout/stderr
- Formats execution results with timestamps


**pkg/logger/**: Log management
- Daily log file rotation
- Logs to stdout and persistent storage
- Automatic cleanup of old logs based on retention policy
- Thread-safe logging

