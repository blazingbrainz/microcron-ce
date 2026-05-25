# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-05-25

### Added
- Sidecar containers support for utility tools
- Shared volume (`/opt/microcron-tools`) for tool sharing between main and sidecar containers
- Pre-built utilities Docker image with curl, wget, jq, awscli, psql, mysql, git, ssh
- Makefile targets for building and publishing utilities sidecar image
- Security context configuration for sidecars (non-root by default)

### Changed
- Main container PATH automatically includes sidecar tools when sidecars enabled
- Improved security: sidecars run as 'nobody' user with dropped capabilities
- Updated documentation with sidecar usage examples

### Security
- Sidecars run as non-root user (UID 65534)
- Tools mounted read-only to main container
- No additional RBAC permissions required

## [0.2.0] - 2026-05-25

### Added
- Optional Kubernetes opaque secret mounting for scripts
- Secret reference syntax in scripts: `# secretname: KEY1, KEY2`
- Automatic secret value injection as environment variables
- `secretMounts` configuration in Helm values for mounting secrets
- Support for secret key rotation (values loaded at execution time, not schedule time)

### Changed
- Script struct now includes optional `SecretRefs` field
- Executor accepts optional environment variables for scripts
- Scheduler loads secret values at job execution time

### Security
- Secrets mounted read-only
- No additional RBAC permissions required (kubelet-managed mounts)
- Secret keys accessible as environment variables to bash scripts

## [0.1.0] - 2026-05-22

### Added
- Initial release with core functionality
- ConfigMap script loading via volume mount
- Full cron expression support (5 fields)
- Automatic log rotation with configurable retention
- Kubernetes deployment via Helm chart
- Docker multi-stage container build
- Kubernetes ServiceAccount and RBAC support
- Liveness and readiness health checks
- Non-root user security context
- PersistentVolume support for durable log storage
- Example scripts in Helm values
- Hot script reloading (30-second polling)
- Graceful shutdown handling

### Technical Details
- **Language**: Go 1.26
- **Dependencies**: robfig/cron/v3 only
- **Container Base**: alpine:latest (minimal image)
- **Log Directory**: `/var/log/microcron-ce`
- **Default Script Mount**: `/etc/microcron-ce/scripts`

### Security Features
- Runs as non-root user (UID 1000)
- No privilege escalation allowed
- All Linux capabilities dropped
- ConfigMap mounted read-only
- Service account for pod identity

