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


## Getting Started

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


### Cron Script Format - to be loaded via configmap

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

The secret must be pre-created in the same namespace and mounted via `secretMounts` in values.yaml. The referenced keys are loaded as environment variables into the 
script.


### Secrets Management


Scripts can optionally reference Kubernetes opaque secrets to access sensitive values like passwords and API keys.


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

Each referenced key is available as an environment variable. The secret keys must match exactly (case-sensitive). Mounted secrets are read-only; no Kubernetes RBAC 
permissions are required.


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

### Using the Makefile for Publishing

A `Makefile` automates the complete build and publish workflow. Versions are automatically extracted from `helm/Chart.yaml`.

**👉 For detailed first-time setup and step-by-step instructions, see [MAKEFILE_SETUP.md](MAKEFILE_SETUP.md)**


## License

MIT License - See LICENSE file for details

## Support

For issues, questions, or feature requests, please use:
- Issues: GitHub Issues

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.


## Roadmap

Future enhancements:
- [ ] Sidecar container support for additional cli tools. 
- [ ] Add support for k8s secret creation via cloud secret stores
- [ ] Job execution history and metrics
- [ ] Prometheus metrics export
- [ ] Script execution timeout configuration
- [ ] Webhook notifications on job completion
- [ ] React-based UI dashboard for job monitoring
- [ ] SAML/OAuth role-based authentication
- [ ] Ingress support
