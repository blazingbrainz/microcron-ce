# Makefile Setup & First Run Guide

Complete step-by-step guide for running the Makefile in WSL/Ubuntu for the first time.

## Step 1: Prerequisites Installation

### 1.1 Open WSL/Ubuntu Terminal

```bash
# If you're on Windows, open PowerShell and run:
wsl
# Or open Ubuntu from Windows Start Menu
```

### 1.2 Update Package Manager

```bash
sudo apt-get update
sudo apt-get upgrade -y
```

### 1.3 Install Build Tools

```bash
sudo apt-get install -y make git
```

Verify installation:
```bash
make --version
# Output: GNU Make 4.x.x
```

### 1.4 Install Docker

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
```

Add your user to docker group (avoid sudo):
```bash
sudo usermod -aG docker $USER
newgrp docker
```

Verify Docker:
```bash
docker --version
# Output: Docker version 24.x.x
```

### 1.5 Install Helm

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

Verify Helm:
```bash
helm version
# Output: version.BuildInfo{Version:"v3.x.x", ...}
```

### 1.6 Install oras

```bash
# Create bin directory
mkdir -p ~/bin

# Download latest oras release
VERSION=1.1.0
cd ~/bin
wget https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_linux_amd64.tar.gz

# Extract
tar xzf oras_${VERSION}_linux_amd64.tar.gz
rm oras_${VERSION}_linux_amd64.tar.gz

# Add to PATH
export PATH=$HOME/bin:$PATH
echo 'export PATH=$HOME/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

Verify oras:
```bash
oras version
# Output: Version:  v1.1.0
```

---

## Step 2: Clone Repository

```bash
# Navigate to your workspace
cd /mnt/d/workspace/blazingbrainz  # Or your desired location

# Clone the repository
git clone https://github.com/blazingbrainz/microcron-ce.git
cd microcron-ce

# Verify Makefile exists
ls -la Makefile
# Output: -rw-r--r-- 1 user user ... Makefile
```

---

## Step 3: Verify Makefile Setup

### 3.1 Check Help

```bash
make help
```

Expected output:
```
Microcron-CE Build & Publish Makefile

Usage:
  make publish GITHUB_USERNAME=<username> GITHUB_PAT=<token>

Available targets:
  versions          - Display app and chart versions
  docker-build      - Build Docker image
  docker-push       - Push Docker image to GHCR
  helm-package      - Package Helm chart as .tgz
  helm-publish      - Publish Helm chart as OCI artifact
  publish           - Build, package, and publish everything
  clean             - Remove local build artifacts
```

### 3.2 Display Current Versions

```bash
make versions
```

Expected output:
```
Version Information:
  Chart File:        helm/Chart.yaml
  App Version:       0.2.0
  Chart Version:     0.2.0

Docker Image:
  ghcr.io/blazingbrainz/microcron-ce:0.2.0

OCI Artifact:
  ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0
```

---

## Step 4: Prepare GitHub Credentials

### 4.1 Create Personal Access Token (PAT)

1. Go to GitHub: https://github.com/settings/tokens
2. Click "Generate new token" → "Generate new token (classic)"
3. Enter name: `microcron-ce-publish`
4. Select scopes:
   - ✅ `write:packages` - Push packages to GHCR
   - ✅ `read:packages` - Read packages from GHCR
5. Click "Generate token"
6. **Copy the token immediately** (you won't see it again!)

### 4.2 Store Credentials Securely (Optional but Recommended)

Create a `.env` file (do NOT commit this):

```bash
# In the microcron-ce directory
cat > .env << 'EOF'
export GITHUB_USERNAME=your-github-username
export GITHUB_PAT=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
EOF

chmod 600 .env  # Restrict permissions
```

⚠️ **Important**: The `export` keyword is **required** so credentials are inherited by make!

Then load it before running make:
```bash
source .env
make debug          # Verify credentials are loaded
make publish        # Should work without passing credentials
```

**Or** pass credentials directly (less secure, visible in shell history):
```bash
make publish GITHUB_USERNAME=your-username GITHUB_PAT=ghp_xxxx
```

**Or** set as environment variables manually:
```bash
export GITHUB_USERNAME=your-username
export GITHUB_PAT=ghp_xxxx
make publish
```

---

## Step 4.3: Verify Credentials Are Loaded

After sourcing `.env`, verify the credentials are properly loaded:

```bash
source .env
make debug
```

Expected output:
```
Debug Information:
  GITHUB_USERNAME: ✓ Set to 'your-username'
  GITHUB_PAT:      ✓ Set (hidden for security)

Troubleshooting:
  ✓ All credentials are set! Ready to publish.
```

If you see "✗ NOT SET", run:
```bash
# Make sure .env has export keyword
cat .env

# Try sourcing again
source .env
make debug
```

---

## Step 5: Test Build (Without Publishing)

### 5.1 Test Docker Build Only

```bash
make docker-build
```

What happens:
1. Reads versions from `helm/Chart.yaml`
2. Builds Docker image: `ghcr.io/blazingbrainz/microcron-ce:0.2.0`
3. Takes ~2-5 minutes depending on internet speed

Expected output:
```
Building Docker image: ghcr.io/blazingbrainz/microcron-ce:0.2.0
[+] Building 45.2s (13/13) FINISHED
...
✓ Docker image built successfully
```

Verify image:
```bash
docker images | grep microcron-ce
```

### 5.2 Test Helm Package Only

```bash
make helm-package
```

What happens:
1. Reads versions from `helm/Chart.yaml`
2. Packages Helm chart: `helm/microcron-ce-0.2.0.tgz`
3. Takes ~1 second

Expected output:
```
Packaging Helm chart...
Successfully packaged chart and saved it to: .../helm/microcron-ce-0.2.0.tgz
✓ Helm chart packaged: helm/microcron-ce-0.2.0.tgz
```

Verify package:
```bash
ls -lh helm/microcron-ce-0.2.0.tgz
```

---

## Step 6: Full Publish Workflow (First Time)

### 6.1 Prepare Environment

```bash
# Option 1: Load from .env file (if you created it)
source .env

# Option 2: Set as variables
export GITHUB_USERNAME="your-github-username"
export GITHUB_PAT="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### 6.2 Run Full Publish

```bash
make publish GITHUB_USERNAME=$GITHUB_USERNAME GITHUB_PAT=$GITHUB_PAT
```

Or directly:
```bash
make publish GITHUB_USERNAME=your-username GITHUB_PAT=ghp_xxxx
```

What happens:
1. Displays versions
2. Builds Docker image
3. Logs into GHCR
4. Pushes Docker image
5. Packages Helm chart
6. Pushes Helm chart as OCI artifact
7. Logs out of GHCR
8. Displays summary

Total time: ~5-10 minutes (first time, longer due to Docker build)

Expected final output:
```
========================================
Microcron-CE Build & Publish Pipeline
========================================

Building Docker image: ghcr.io/blazingbrainz/microcron-ce:0.2.0
...
✓ Docker image pushed: ghcr.io/blazingbrainz/microcron-ce:0.2.0

Packaging Helm chart...
✓ Helm chart packaged: helm/microcron-ce-0.2.0.tgz

Publishing Helm chart as OCI artifact...
✓ Helm chart published: ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0

========================================
✓ Publish Complete!
========================================

Artifacts Published:
  Docker Image: ghcr.io/blazingbrainz/microcron-ce:0.2.0
  OCI Artifact: ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0

Pull commands:
  docker pull ghcr.io/blazingbrainz/microcron-ce:0.2.0
  helm pull oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0
```

### 6.3 Verify Published Artifacts

```bash
# Check Docker image in GHCR
curl -H "Authorization: token $GITHUB_PAT" \
  https://ghcr.io/v2/blazingbrainz/microcron-ce/tags/list

# Check Helm chart in GHCR
oras repo tags ghcr.io/blazingbrainz/helm-charts/microcron-ce
```

---

## Step 7: Update Version & Republish (Next Releases)

### 7.1 Update Version in helm/Chart.yaml

```bash
# Edit the Chart.yaml file
nano helm/Chart.yaml

# Change both version and appVersion to new version
# version: 0.3.0
# appVersion: "0.3.0"
```

### 7.2 Run Publish Again

```bash
make publish GITHUB_USERNAME=$GITHUB_USERNAME GITHUB_PAT=$GITHUB_PAT
```

The Makefile will automatically:
- Detect new versions
- Build new images
- Package new artifacts
- Publish to GHCR

---

## Troubleshooting

### Error: `make: command not found`

```bash
sudo apt-get install -y make
```

### Error: `docker: permission denied`

```bash
sudo usermod -aG docker $USER
newgrp docker
# Log out and log back in to WSL
```

### Error: `oras: command not found`

```bash
# Ensure oras is in PATH
export PATH=$HOME/bin:$PATH
oras version
```

### Error: `Error: GITHUB_USERNAME and GITHUB_PAT are required`

```bash
# Provide credentials
make publish GITHUB_USERNAME=your-user GITHUB_PAT=your-token
```

### Error: `No such file or directory: helm/Chart.yaml`

```bash
# Ensure you're in the correct directory
pwd  # Should show: .../microcron-ce
ls helm/Chart.yaml  # Should exist
```

### Docker Build Takes Very Long

This is normal for the first build. Subsequent builds are much faster due to caching.

### GHCR Login Fails

```bash
# Verify token has write:packages permission
# Verify username is correct (not email)
# Try logging in manually first:
echo $GITHUB_PAT | docker login ghcr.io -u $GITHUB_USERNAME --password-stdin
```

---

## Quick Reference Commands

```bash
# Check versions before publishing
make versions

# Build only (no push)
make docker-build

# Package only (no push)
make helm-package

# Full publish
make publish GITHUB_USERNAME=user GITHUB_PAT=token

# Clean local artifacts
make clean

# View all options
make help
```

---

## Security Best Practices

⚠️ **DO NOT:**
- Commit `.env` file (add to `.gitignore`)
- Share your PAT in chat, email, or documentation
- Use PAT in shell history (use `.env` file instead)
- Give PAT more permissions than needed

✅ **DO:**
- Use fine-grained personal access tokens when available
- Rotate tokens periodically
- Restrict token to specific repositories if possible
- Delete unused tokens from GitHub settings

---

## Next Steps

After first successful publish:

1. **Tag the release in Git:**
   ```bash
   git tag -a v0.2.0 -m "Release 0.2.0: Add secret mounting support"
   git push origin v0.2.0
   ```

2. **Create GitHub Release:**
   - Go to: https://github.com/blazingbrainz/microcron-ce/releases
   - Click "Create a new release"
   - Tag version: `v0.2.0`
   - Title: `Release 0.2.0: Add secret mounting support`
   - Description: Add changelog details

3. **Update Deployment:**
   ```bash
   # Pull new image version
   docker pull ghcr.io/blazingbrainz/microcron-ce:0.2.0
   
   # Deploy with helm
   helm install microcron-ce \
     oci://ghcr.io/blazingbrainz/helm-charts/microcron-ce:0.2.0 \
     -n microcron-ce --create-namespace
   ```

---

## Support

For issues or questions:
- Check this guide's Troubleshooting section
- Review Makefile help: `make help`
- Check README.md for detailed documentation
- Review CHANGELOG.md for version history
