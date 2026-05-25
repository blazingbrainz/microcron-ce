# Microcron-CE - Kubernetes Native Cron Job Runner

A lightweight, cloud-native microservice for running scheduled bash/shell scripts in Kubernetes using cron expressions. Scripts are managed through ConfigMaps, making it easy to update schedules and scripts without redeploying.

## Overview

Microcron-CE solves the problem of running scheduled tasks in Kubernetes environments. Build specifically for environments/organizations following k8s/container centric deployments. This microservice retains the familarity with the traditional bash scripts + cron syntax while taking full advantage for k8s infrastructures:

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
├── helm/                        # Helm chart v0.2.0           
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

## Features

### Core Features

✅ **ConfigMap-Based Script Management**: Scripts mounted as volume  
✅ **Cron Schedule Support**: Full cron expression support (5 fields)  
✅ **Automatic Log Rotation**: Daily logs with configurable retention  
✅ **Persistent Logging**: Optional PVC for durable log storage  
✅ **Hot Script Reloading**: ConfigMap updates detected via polling  
✅ **Security**: Non-root user, read-only filesystem (except logs)  
✅ **Health Checks**: Liveness and readiness probes  
✅ **Production Ready**: Multi-stage Docker build, resource limits, minimal footprint

### Script Format

Scripts in ConfigMaps must follow this format:

```bash
#!/bin/bash
# 0 * * * *

# Your script content here
echo "Script executed at $(date)"
```

**Cron Schedule Line**: The first non-shebang comment line must contain a valid 5-field cron expression:
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

**Optional Secret Reference** (new): The second non-shebang comment line can specify Kubernetes secrets:

```bash
#!/bin/bash
# 0 * * * *
# my-db-secret: DB_USER, DB_PASS, DB_HOST

echo "Connecting as $DB_USER to $DB_HOST"
```

Format: `# <secretname>: <key1>, <key2>, ..., <keyN>`

The secret must be pre-created in the same namespace and mounted via `secretMounts` in values.yaml. The referenced keys are loaded as environment variables into the script.

### Secrets Management

Scripts can reference Kubernetes opaque secrets to access sensitive values like passwords and API keys.

**Creating a Secret**:

```bash
kubectl create secret generic my-db-secret \
  --from-literal=DB_USER=admin \
  --from-literal=DB_PASS=s3cr3t \
  --from-literal=DB_HOST=postgres.svc \
  -n microcron-ce
```

**Configuring Secret Mounts** (in `values.yaml`):

```yaml
secretMounts:
  - name: my-db-secret
  - name: api-credentials
```

**Using Secrets in Scripts**:

```bash
#!/bin/bash
# 0 * * * *
# my-db-secret: DB_USER, DB_PASS, DB_HOST

psql -h "$DB_HOST" -U "$DB_USER" -c "SELECT version();" << EOF
$DB_PASS
EOF
```

Each referenced key is available as an environment variable. The secret keys must match exactly (case-sensitive). Mounted secrets are read-only; no Kubernetes RBAC permissions are required.

## Getting Started

### Prerequisites

- Kubernetes 1.24+
- Helm 3.7+ (for OCI registry support)
- kubectl configured with cluster access
- GitHub Personal Access Token (for pulling from ghcr.io)

### Quick Start - Deploy from OCI Registry

```bash
# 1. Create namespace
kubectl create namespace microcron-ce

# 2. Create image pull secret (if private registry)
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_TOKEN \
  -n microcron-ce

# 3. Add Helm repo and install
helm pull oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce --version 0.2.0

# 4. Install chart
helm install microcron-ce ./microcron-ce-0.2.0.tgz \
  --namespace microcron-ce \
  --set image.repository=ghcr.io/blazingbrainz/microcron-ce \
  --set image.pullPolicy=IfNotPresent

# 5. Verify deployment
kubectl get pods -n microcron-ce
kubectl logs -f deployment/microcron-ce -n microcron-ce
```

### Deployment Examples

**Production Deployment**
```bash
helm install microcron-ce oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce \
  --version 0.2.0 \
  --namespace production \
  --create-namespace \
  --values values-production.yaml
```

**Development Deployment**
```bash
helm install microcron-ce oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce \
  --version 0.2.0 \
  --namespace dev \
  --create-namespace \
  --set logging.retentionDays=3 \
  --set persistence.enabled=false
```

### Container Images

**Docker Image**: `ghcr.io/blazingbrainz/microcron-ce:0.2.0`
**Helm Chart**: `oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0`

Both available at GitHub Container Registry (GHCR)

## Configuration

### Command-Line Flags

```
--namespace=default           # Kubernetes namespace for ConfigMaps
--configmap=microcron-scripts # Name of ConfigMap with scripts
--log-dir=/var/log/microcron  # Directory for log files
--retention-days=7            # Days to retain log files
```

### Helm Values

Key Helm values (see `helm/values.yaml` for all):

```yaml
replicaCount: 1

image:
  repository: ghcr.io/blazingbrainz/microcron-ce
  pullPolicy: IfNotPresent
  tag: "0.2.0"

imagePullSecrets: []
  # - name: ghcr-secret

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

- **Chart.yaml**: Chart metadata (v0.2.0)
- **values.yaml**: Default configuration
- **values-production.yaml**: Production example
- **Deployment**: Main pod deployment with init containers and health checks
- **ServiceAccount**: RBAC identity
- **ClusterRole/ClusterRoleBinding**: Permissions to read ConfigMaps
- **PersistentVolumeClaim**: Optional log storage
- **ConfigMap**: template for script cofigmap

### Helm Chart Management

**Install from OCI Registry**
```bash
helm install microcron-ce oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce \
  --version 0.2.0 \
  --namespace microcron-ce \
  --create-namespace
```

**Upgrade to newer version**
```bash
helm upgrade microcron-ce oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce \
  --version 0.1.1 \
  --namespace microcron-ce
```

**Uninstall**
```bash
helm uninstall microcron-ce -n microcron-ce
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
- **ServiceAccount**: microcron-ce (for pod identity)
- **ClusterRole**: Empty (no API permissions required)
- **ClusterRoleBinding**: Minimal binding for pod identity


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

### Example 2: Every 5 Minutes Log Check

```bash
#!/bin/bash
# */5 * * * *

ERROR_COUNT=$(grep -c "ERROR" /var/log/app.log)
echo "Current error count: $ERROR_COUNT"
```

## Publishing & Development (contributors only section)

### For Contributors - Building from Source

**Prerequisites**: Go 1.26+, Docker, Helm 3.7+, oras

```bash
# Clone and build
git clone https://github.com/blazingbrainz/microcron-ce.git
cd microcron-ce

# Build Docker image
docker build -t ghcr.io/blazingbrainz/microcron-ce:dev .

# Test locally
docker run ghcr.io/blazingbrainz/microcron-ce:dev

# Package Helm chart
cd helm
helm package .
```

### Publishing to GHCR (optional - only for authors)

**Prerequisites**:
- GitHub Personal Access Token with `write:packages` permission
- `docker` and `oras` CLI tools installed

```bash
# 1. Authenticate to GHCR
echo YOUR_GITHUB_PAT | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# 2. Build and push Docker image
#    update namespace if needed
docker build -t ghcr.io/blazingbrainz/microcron-ce:0.2.0 .
docker push ghcr.io/blazingbrainz/microcron-ce:0.2.0

# 3. Push Helm chart as OCI artifact using oras
#    Update namespace to your own/target namespace
cd helm
oras login -u YOUR_GITHUB_USERNAME -p YOUR_GITHUB_PAT ghcr.io
oras push ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0 \
  microcron-ce-0.2.0.tgz:application/vnd.cncf.helm.chart.v1.tar+gzip

# 4. Verify both are published
docker images | grep microcron-ce
oras repo tags ghcr.io/blazingbrainz/helm-charts/microcron-ce
```

**Note**: Use `oras` instead of `helm push` for reliable OCI artifact publishing to GHCR.


## License

MIT License - See LICENSE file for details

## Support

For issues, questions, or feature requests, please contact:
- Email:  mailfrmsoyuz@rocketmail.com
- Issues: GitHub Issues

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.


## Roadmap

Future enhancements:
- [x] Tokenized secrets in scripts via Kubernetes Secrets
- [ ] Job execution history and metrics
- [ ] Prometheus metrics export
- [ ] Script execution timeout configuration
- [ ] Webhook notifications on job completion
- [ ] React-based UI dashboard for job monitoring
- [ ] SAML/OAuth role-based authentication
- [ ] Ingress support
- [ ] Multi-namespace support
- [ ] Script templates with variable substitution 
