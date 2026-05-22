# Microcron-CE - Kubernetes Native Cron Job Runner

A lightweight, cloud-native microservice for running scheduled bash/shell scripts in Kubernetes using cron expressions. Scripts are managed through ConfigMaps, making it easy to update schedules and scripts without redeploying.

## Overview

Microcron-CE solves the problem of running scheduled tasks in Kubernetes environments. Unlike traditional cron jobs that run on a single node, Microcron-CE:

- Loads scripts from Kubernetes ConfigMaps
- Executes scripts according to cron schedules
- Logs all output to both stdout and date-rotated log files
- Supports automatic log rotation with configurable retention
- Provides proper Kubernetes RBAC and security contexts
- Includes comprehensive Helm chart for easy deployment

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
├── helm/                        # Helm chart v0.1.0           
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

**pkg/configmap/**: Kubernetes ConfigMap integration
- Loads scripts from ConfigMaps
- Extracts cron schedule from script comments
- Watches for ConfigMap updates
- In-cluster Kubernetes authentication

**pkg/executor/**: Script execution
- Executes shell scripts via os/exec
- Captures stdout/stderr
- Formats execution results with timestamps

**pkg/logger/**: Log management
- Daily log file rotation
- Logs to stdout and persistent storage
- Automatic cleanup of old logs based on retention policy
- Thread-safe logging

## Features

### Core Features

✅ **ConfigMap-Based Script Management**: Scripts stored as ConfigMap data  
✅ **Cron Schedule Support**: Full cron expression support (5 fields)  
✅ **Automatic Log Rotation**: Daily logs with configurable retention  
✅ **Persistent Logging**: Optional PVC for durable log storage  
✅ **Hot Script Reloading**: ConfigMap updates detected automatically  
✅ **Kubernetes RBAC**: Proper service accounts and role bindings  
✅ **Security**: Non-root user, read-only filesystem (except logs)  
✅ **Health Checks**: Liveness and readiness probes  
✅ **Production Ready**: Multi-stage Docker build, resource limits

### Script Format

Scripts in ConfigMaps must follow this format:

```bash
#!/bin/bash
# 0 * * * *

# Your script content here
echo "Script executed at $(date)"
```

The first non-shebang comment line must contain a valid 5-field cron expression:
- Minute (0-59)
- Hour (0-23)
- Day of Month (1-31)
- Month (1-12)
- Day of Week (0-6, where 0=Sunday)

Examples:
- `# 0 * * * *` - Top of every hour
- `# 0 2 * * *` - Every day at 2 AM
- `# */5 * * * *` - Every 5 minutes
- `# 0 0 1 * *` - First day of every month

## Getting Started

### Prerequisites

- Go 1.22+
- Kubernetes 1.24+
- Helm 3.0+
- Docker (for building images)

### Local Development

1. **Build the application**
   ```bash
   go build -o microcron-ce ./cmd/microcron-ce
   ```

2. **Create a test ConfigMap**
   ```bash
   kubectl create configmap microcron-scripts \
     --from-literal='test.sh=#!/bin/bash\n# */5 * * * *\necho "Hello from script"'
   ```

3. **Run the application**
   ```bash
   ./microcron-ce \
     --namespace=default \
     --configmap=microcron-scripts \
     --log-dir=./logs \
     --retention-days=7
   ```

### Deployment with Helm

1. **Install with default values**
   ```bash
   helm install microcron-ce ./helm/microcron-ce \
     --namespace microcron-ce \
     --create-namespace
   ```

2. **Create scripts ConfigMap**
   ```bash
   kubectl create configmap microcron-scripts \
     --from-file=backup.sh \
     --from-file=health-check.sh \
     -n microcron-ce
   ```

3. **Verify deployment**
   ```bash
   kubectl get pods -n microcron-ce
   kubectl logs -f deployment/microcron-ce -n microcron-ce
   ```

### Docker Build

1. **Build image**
   ```bash
   docker build -t microcron-ce:0.1.0 .
   ```

2. **Run in container**
   ```bash
   docker run \
     -v $(pwd)/logs:/var/log/microcron-ce \
     -e KUBECONFIG=/root/.kube/config \
     -v $HOME/.kube/config:/root/.kube/config \
     microcron-ce:0.1.0
   ```

## Configuration

### Command-Line Flags

```
--namespace=default           # Kubernetes namespace for ConfigMaps
--configmap=microcron-scripts # Name of ConfigMap with scripts
--log-dir=/var/log/microcron  # Directory for log files
--retention-days=7            # Days to retain log files
```

### Helm Values

Key Helm values (see `helm/microcron-ce/values.yaml` for all):

```yaml
replicaCount: 1
image:
  repository: blazingbrainz/microcron-ce
  tag: "0.1.0"

namespace: default
configMapName: microcron-scripts

logging:
  logDir: /var/log/microcron-ce
  retentionDays: 7

persistence:
  enabled: true
  size: 10Gi

resources:
  requests:
    memory: 128Mi
    cpu: 100m
  limits:
    memory: 512Mi
    cpu: 500m
```

## Helm Chart Details

The Helm chart includes:

- **Chart.yaml**: Chart metadata (v0.1.0)
- **values.yaml**: Default configuration
- **values-production.yaml**: Production example
- **Deployment**: Main pod deployment with init containers and health checks
- **ServiceAccount**: RBAC identity
- **ClusterRole/ClusterRoleBinding**: Permissions to read ConfigMaps
- **PersistentVolumeClaim**: Optional log storage
- **ConfigMap example**: Sample scripts (optional)

### Install Helm Chart

```bash
# Install with defaults
helm install microcron-ce ./helm/microcron-ce

# Install with custom values
helm install microcron-ce ./helm/microcron-ce \
  -f helm/microcron-ce/values-production.yaml

# Upgrade
helm upgrade microcron-ce ./helm/microcron-ce

# Uninstall
helm uninstall microcron-ce
```

## Security

### Security Features

- **Non-root user**: Runs as UID 1000 (microcron)
- **Security context**: No privilege escalation, dropped capabilities
- **RBAC**: Minimal permissions (read ConfigMaps only)
- **Read-only filesystem**: Root filesystem read-only (except /tmp and logs)
- **Network policies**: Can be configured via Ingress/NetworkPolicy

### Kubernetes RBAC

The chart creates:
- **ServiceAccount**: microcron-ce
- **ClusterRole**: Permission to get/list/watch ConfigMaps
- **ClusterRoleBinding**: Binds ServiceAccount to ClusterRole

## Logs

### Log Location

- **Stdout**: Real-time logs to console
- **Files**: `/var/log/microcron-ce/microcron-ce-YYYY-MM-DD.log`
- **Rotation**: Daily, automatic cleanup after retention period

### View Logs

```bash
# Tail pod logs
kubectl logs -f deployment/microcron-ce -n microcron-ce

# Get specific script execution output
kubectl logs deployment/microcron-ce -n microcron-ce | grep "script-name"

# Access log files from persistent volume
kubectl exec deployment/microcron-ce -n microcron-ce -- \
  ls -la /var/log/microcron-ce/
```

## Monitoring & Troubleshooting

### Health Checks

- **Liveness probe**: Checks if process is running
- **Readiness probe**: Same as liveness (pod is ready immediately)

### Common Issues

**Scripts not executing:**
- Verify ConfigMap exists: `kubectl get configmap microcron-scripts`
- Check script format has valid cron expression in first comment
- Check pod logs: `kubectl logs deployment/microcron-ce`

**Pod not starting:**
- Check RBAC permissions: `kubectl describe clusterrole microcron-ce`
- Verify storage class exists: `kubectl get storageclass`
- Check pod events: `kubectl describe pod <pod-name>`

**Log files not persisting:**
- Verify PVC is bound: `kubectl get pvc`
- Check pod volume mounts: `kubectl describe pod <pod-name>`

## Development Guide

### Adding a New Package

1. Create package directory: `pkg/newpackage/`
2. Create implementation: `pkg/newpackage/newpackage.go`
3. Add tests: `pkg/newpackage/newpackage_test.go`
4. Update imports in `cmd/microcron-ce/main.go`

### Testing

```bash
go test ./...
go test -v ./pkg/cron
go test -cover ./...
```

### Code Style

- Follow Go conventions
- Use meaningful variable names
- Add comments for exported functions
- Keep packages focused and atomic

## Examples

### Example 1: Hourly Health Check

```bash
#!/bin/bash
# 0 * * * *

STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$STATUS" = "200" ]; then
  echo "Service is healthy"
else
  echo "Service health check failed: $STATUS"
fi
```

### Example 2: Daily Database Backup

```bash
#!/bin/bash
# 0 2 * * *

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/db_backup_$DATE.sql"

pg_dump -h db-host -U db-user -d db-name > "$BACKUP_FILE"
gzip "$BACKUP_FILE"
echo "Backup completed: ${BACKUP_FILE}.gz"
```

### Example 3: Every 5 Minutes Log Check

```bash
#!/bin/bash
# */5 * * * *

ERROR_COUNT=$(grep -c "ERROR" /var/log/app.log)
echo "Current error count: $ERROR_COUNT"
```

## Deployment Examples

### Development Deployment

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace dev \
  --create-namespace \
  --set logging.retentionDays=3
```

### Production Deployment

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace production \
  --create-namespace \
  -f helm/microcron-ce/values-production.yaml
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write tests
5. Submit a pull request

## License

MIT License - See LICENSE file for details

## Support

For issues, questions, or feature requests, please contact:
- Email: dev@blazingbrainz.com
- Issues: GitHub Issues

## Changelog

### v0.1.0 (Initial Release)
- Initial release with core functionality
- ConfigMap script loading
- Cron schedule support
- Log rotation
- Helm chart deployment
- Docker containerization
- Kubernetes RBAC

## Roadmap

Future enhancements:
- [ ] Job execution history/metrics
- [ ] Prometheus metrics export
- [ ] Script execution timeout configuration
- [ ] Webhook notifications on job completion
- [ ] UI dashboard for job monitoring
- [ ] Multi-namespace support
- [ ] Script templates
