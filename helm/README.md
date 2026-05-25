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
| `image.tag` | Container image tag | `0.2.0` |
| `namespace` | Kubernetes namespace for ConfigMap | `default` |
| `configMapName` | Name of ConfigMap with scripts | `microcron-scripts` |
| `secretMounts` | List of secrets to mount (by name) | `[]` |
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
