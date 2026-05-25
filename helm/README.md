# Microcron-CE Helm Chart

A Kubernetes-native cron job runner that loads scripts from ConfigMaps and executes them according to cron schedules.

## Features

- **ConfigMap-based Script Management**: Load multiple bash/shell scripts from a Kubernetes ConfigMap
- **Cron Schedule Support**: Define cron schedules in script comments (first comment line)
- **Log Rotation**: Automatic daily log rotation with configurable retention period
- **Persistent Storage**: Optional PersistentVolumeClaim for log storage
- **Kubernetes RBAC**: Proper service account and role bindings included
- **Health Checks**: Liveness and readiness probes for pod health monitoring

## Installation

### Prerequisites

- Kubernetes 1.24+
- Helm 3.7+

### Install the Chart

```bash
# Clone the repository
git clone https://github.com/blazingbrainz/microcron-ce.git
cd microcron-ce

# Install the chart
helm install microcron-ce ./helm/microcron-ce \
  --namespace microcron-ce \
  --create-namespace
```

### Quick Start with Example Scripts

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace microcron-ce \
  --create-namespace \
  --set createExampleConfigMap=true
```

## Configuration

### Basic Configuration

```bash
helm install microcron-ce ./helm/microcron-ce \
  --set namespace=default \
  --set configMapName=microcron-scripts \
  --set logging.retentionDays=7
```

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image repository | `blazingbrainz/microcron-ce` |
| `image.tag` | Container image tag | `0.2.1` |
| `namespace` | Kubernetes namespace for ConfigMap | `default` |
| `configMapName` | Name of ConfigMap with scripts | `microcron-scripts` |
| `secretMounts` | List of secrets to mount (by name) | `[]` |
| `sidecars` | List of sidecar containers | `[]` |
| `logging.retentionDays` | Days to retain log files | `7` |
| `persistence.enabled` | Enable persistent volume for logs | `true` |
| `persistence.size` | PVC size | `10Gi` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.limits.cpu` | CPU limit | `500m` |

### Full Parameters

See `values.yaml` for all available configuration options.

## Creating Scripts

Scripts are loaded from a Kubernetes ConfigMap. Each script must have a cron schedule defined in its first comment line.

### Script Format

```bash
#!/bin/bash
# 0 * * * *

echo "This runs at the top of every hour"
```

The format is:
```
#!/bin/bash
# MINUTE HOUR DAY MONTH DAY_OF_WEEK
# [optional] secretname: KEY1, KEY2
script content...
```

## Sidecar Containers for Utility Tools

The deployment supports optional sidecar containers that share all mounts with the main cron scheduler. This allows cron scripts to access CLI tools from sidecars.

### Included Tools

**Default utilities image includes:**
- **Network**: curl, wget, jq, dnsutils
- **AWS**: awscli
- **Databases**: postgresql-client, mysql-client
- **Utilities**: git, ssh, python3, bash

### How Shared Tools Work

When sidecars are enabled:
1. A shared EmptyDir volume is created at `/opt/microcron-tools`
2. Sidecars install utilities to `/opt/microcron-tools/bin`
3. Main container mounts the same volume (read-only)
4. Main container's PATH is automatically updated to include the tools
5. Bash scripts can directly call tools: `curl`, `aws`, `psql`, `mysql`, `git`, etc.

Sidecars are defined in `values.yaml` and automatically share:
- Script volumes (ConfigMap)
- Secret mounts
- Log volumes
- **Tools volume** (EmptyDir at `/opt/microcron-tools`)
- Network namespace (can communicate via localhost)

Example 1: **Use pre-built utilities image (recommended)**

```yaml
sidecars:
  - name: utilities
    image: ghcr.io/blazingbrainz/microcron-ce-utilities:0.2.1
    imagePullPolicy: IfNotPresent
    resources:
      requests:
        memory: 256Mi
        cpu: 100m
      limits:
        memory: 512Mi
        cpu: 500m
```

By default, sidecars run as the `nobody` user (UID 65534) with dropped capabilities for security.

Example 2: **Production with explicit security context**

```yaml
sidecars:
  - name: utilities
    image: ghcr.io/blazingbrainz/microcron-ce-utilities:0.2.1
    imagePullPolicy: IfNotPresent
    resources:
      requests:
        memory: 256Mi
        cpu: 100m
      limits:
        memory: 512Mi
        cpu: 500m
    # Explicit security context (non-root, no privilege escalation)
    securityContext:
      runAsNonRoot: true
      runAsUser: 65534  # 'nobody' user
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
```

Example 3: **Custom utilities image**

```yaml
sidecars:
  - name: utilities
    image: ghcr.io/yourusername/microcron-utilities:latest
    imagePullPolicy: IfNotPresent
    resources:
      requests:
        memory: 256Mi
        cpu: 100m
      limits:
        memory: 512Mi
        cpu: 500m
```

### Using Sidecar Tools in Scripts

Tools from sidecars are **directly available** in the main container via a shared volume. The main container's PATH is automatically updated to include `/opt/microcron-tools/bin`.

Example: **AWS CLI via sidecar**

```bash
#!/bin/bash
# 0 * * * *

# AWS CLI tools are directly available
aws s3 ls > /var/log/microcron-ce/s3-listing.log
aws ec2 describe-instances --region us-east-1
```

Example: **psql via sidecar**

```bash
#!/bin/bash
# 0 2 * * *

# PostgreSQL tools are directly available
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM users;" \
  >> /var/log/microcron-ce/db-count.log
```

Example: **Multiple tools**

```bash
#!/bin/bash
# 0 * * * *

# Use curl to fetch data
DATA=$(curl -s https://api.example.com/data)

# Parse with jq
COUNT=$(echo "$DATA" | jq '.items | length')

# Store in database
echo "Fetched $COUNT items" | psql -h $DB_HOST -U $DB_USER
```

### Creating a Custom Utilities Image

To extend the utilities image with additional tools, create a custom Dockerfile:

**Example Dockerfile** (add Azure CLI and Google Cloud CLI):

```dockerfile
FROM ghcr.io/blazingbrainz/microcron-ce-utilities:0.2.1

# Add additional tools on top of base utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    azure-cli \
    google-cloud-cli \
    && rm -rf /var/lib/apt/lists/*
```

Build and push:

```bash
docker build -t ghcr.io/yourusername/microcron-utilities:latest .
docker push ghcr.io/yourusername/microcron-utilities:latest
```

Then reference in `values.yaml`:

```yaml
sidecars:
  - name: utilities
    image: ghcr.io/yourusername/microcron-utilities:latest
```

**Note**: The pre-built `microcron-ce-utilities` image is recommended as a base for custom images - it's optimized for cron script use cases.

## Script Secrets

Scripts can reference Kubernetes opaque secrets to access sensitive data.

### Creating and Using Secrets

1. **Create a Kubernetes secret**:
```bash
kubectl create secret generic db-credentials \
  --from-literal=DB_USER=admin \
  --from-literal=DB_PASS=password123 \
  -n microcron-ce
```

2. **Configure secret mounts in values.yaml**:
```yaml
secretMounts:
  - name: db-credentials
```

3. **Use the secret in a script**:
```bash
#!/bin/bash
# 0 * * * *
# db-credentials: DB_USER, DB_PASS

echo "Database user is $DB_USER"
```

4. **Deploy with the updated values**:
```bash
helm upgrade microcron-ce ./helm/microcron-ce \
  -f values.yaml
```

The referenced keys become environment variables in the script context.

### Create ConfigMap with Scripts

```bash
kubectl create configmap microcron-scripts \
  --from-file=backup.sh \
  --from-file=health-check.sh \
  -n microcron-ce
```

### Example Scripts

```bash
# Every hour at the top of the hour
# 0 * * * *

# Every day at 2 AM
# 0 2 * * *

# Every 5 minutes
# */5 * * * *

# Every Monday at 9 AM
# 0 9 * * 1

# Every 1st day of month at midnight
# 0 0 1 * *
```

### Update Scripts

After updating the ConfigMap, the scheduler will automatically reload scripts within 30 seconds:

```bash
kubectl edit configmap microcron-scripts -n microcron-ce
```

## Viewing Logs

### Check Pod Logs

```bash
kubectl logs -f deployment/microcron-ce -n microcron-ce
```

### Access Log Files

If persistence is enabled:

```bash
kubectl exec -it deployment/microcron-ce -n microcron-ce -- sh
ls -la /var/log/microcron-ce/
cat /var/log/microcron-ce/microcron-ce-2024-01-01.log
```

### Port Forward to Access Logs

```bash
kubectl port-forward svc/microcron-ce 8080:8080 -n microcron-ce
```

## Examples

### Install with Persistent Volume

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace microcron-ce \
  --create-namespace \
  --set persistence.enabled=true \
  --set persistence.size=20Gi \
  --set logging.retentionDays=30
```

### Install with Example Scripts

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace microcron-ce \
  --create-namespace \
  --set createExampleConfigMap=true
```

### Deploy to Production

```bash
helm install microcron-ce ./helm/microcron-ce \
  --namespace production \
  --create-namespace \
  --set replicaCount=1 \
  --set resources.limits.memory=1Gi \
  --set resources.limits.cpu=1000m \
  --set resources.requests.memory=512Mi \
  --set resources.requests.cpu=200m \
  --set persistence.enabled=true \
  --set persistence.size=50Gi \
  --set logging.retentionDays=30 \
  --set persistence.storageClassName=fast-ssd
```

## Upgrading

```bash
helm upgrade microcron-ce ./helm/microcron-ce \
  --namespace microcron-ce \
  --set logging.retentionDays=14
```

## Uninstalling

```bash
helm uninstall microcron-ce -n microcron-ce
kubectl delete namespace microcron-ce
```

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod -l app.kubernetes.io/name=microcron-ce -n microcron-ce
kubectl logs deployment/microcron-ce -n microcron-ce
```

### Scripts not running

1. Check if scripts are in ConfigMap:
```bash
kubectl get configmap microcron-scripts -n microcron-ce -o yaml
```

2. Verify script format (must have cron schedule in first comment):
```bash
kubectl get configmap microcron-scripts -n microcron-ce -o jsonpath='{.data.your-script\.sh}' | head -2
```

3. Check pod logs for errors:
```bash
kubectl logs deployment/microcron-ce -n microcron-ce
```

### Permissions issues

Ensure RBAC is properly configured:
```bash
kubectl get clusterrole | grep microcron-ce
kubectl get clusterrolebinding | grep microcron-ce
```

## Development

### Build Image Locally

```bash
docker build -t microcron-ce:latest .
```

### Test Locally

```bash
docker run -v $(pwd)/scripts:/etc/microcron-ce/scripts \
  -v $(pwd)/logs:/var/log/microcron-ce \
  microcron-ce:latest
```


## License

MIT License - See LICENSE file for details
