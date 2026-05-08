# Release Process

This document describes the release process for k8s-pod-postmortem.

## Versioning

k8s-pod-postmortem follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, backward compatible

## Release Workflow

### Prerequisites

1. Ensure all tests pass on `main` branch
2. Ensure all changes are documented in CHANGELOG.md
3. Ensure you have push access to the repository
4. Ensure you have Docker Hub/GHCR access for image publishing

### Step 1: Prepare Release

1. **Update CHANGELOG.md**

```bash
# Create a new version section
## [v1.0.0] - 2024-01-15

### Added
- New feature X
- New feature Y

### Changed
- Improvement Z

### Fixed
- Bug fix A
- Bug fix B

### Breaking Changes
- Breaking change description (if any)
```

2. **Update version files**

```bash
# Update Chart.yaml
sed -i 's/appVersion: "0.1.0"/appVersion: "1.0.0"/' charts/k8s-pod-postmortem/Chart.yaml
sed -i 's/version: 0.1.0/version: 1.0.0/' charts/k8s-pod-postmortem/Chart.yaml

# Update action.yaml if needed
# Update README.md version references
```

3. **Commit changes**

```bash
git add .
git commit -s -m "chore: prepare release v1.0.0"
git push origin main
```

### Step 2: Create Release Tag

1. **Create and push tag**

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

2. **Or use GitHub UI**

- Go to https://github.com/bansikah22/k8s-pod-postmortem/releases/new
- Select "Choose a tag" and enter `v1.0.0`
- Click "Generate release notes"
- Add any additional notes
- Click "Publish release"

### Step 3: Automated Release Process

When a tag is pushed, the CI/CD pipeline automatically:

1. **Builds the binary**

```yaml
# .github/workflows/release.yaml
- name: Build binary
  run: |
    GOOS=linux GOARCH=amd64 go build -o bin/postmortem-linux-amd64 ./cmd/postmortem
    GOOS=linux GOARCH=arm64 go build -o bin/postmortem-linux-arm64 ./cmd/postmortem
```

2. **Builds and pushes Docker image**

```yaml
- name: Build and push Docker image
  uses: docker/build-push-action@v5
  with:
    push: true
    tags: |
      ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0
      ghcr.io/bansikah22/k8s-pod-postmortem:latest
```

3. **Publishes Helm chart**

```yaml
- name: Package Helm chart
  run: |
    helm package charts/k8s-pod-postmortem
    helm push k8s-pod-postmortem-*.tgz oci://ghcr.io/bansikah22/charts
```

4. **Creates GitHub release**

```yaml
- name: Create GitHub release
  uses: softprops/action-gh-release@v1
  with:
    files: bin/*
    generate_release_notes: true
```

### Step 4: Verify Release

1. **Check GitHub Release**

```bash
# View release
open https://github.com/bansikah22/k8s-pod-postmortem/releases/tag/v1.0.0
```

2. **Verify Docker Image**

```bash
# Pull and test image
docker pull ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0
docker run --rm ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0 --help
```

3. **Verify Helm Chart**

```bash
# Add repository
helm repo add k8s-pod-postmortem https://bansikah22.github.io/k8s-pod-postmortem
helm repo update

# Search for chart
helm search repo k8s-pod-postmortem --versions
```

4. **Test in Kubernetes**

```bash
# Install new version
helm upgrade --install k8s-pod-postmortem \
  k8s-pod-postmortem/k8s-pod-postmortem \
  --namespace actions-runners \
  --version 1.0.0
```

## Release Checklist

### Pre-Release

- [ ] All tests pass on CI
- [ ] CHANGELOG.md updated
- [ ] Version bumped in all files
- [ ] Documentation updated
- [ ] Breaking changes documented
- [ ] Migration guide created (if breaking changes)

### Release

- [ ] Tag created
- [ ] Release notes generated
- [ ] Docker image published
- [ ] Helm chart published
- [ ] GitHub release published

### Post-Release

- [ ] Verify Docker image
- [ ] Verify Helm chart
- [ ] Test installation
- [ ] Update documentation site (if applicable)
- [ ] Announce release

## Automated Release Workflow

The complete release workflow is defined in `.github/workflows/release.yaml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Build binaries
        run: |
          GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/postmortem-linux-amd64 ./cmd/postmortem
          GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/postmortem-linux-arm64 ./cmd/postmortem
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: bin/

  docker:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/bansikah22/k8s-pod-postmortem:${{ github.ref_name }}
            ghcr.io/bansikah22/k8s-pod-postmortem:latest

  helm:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Helm
        uses: azure/setup-helm@v3
      
      - name: Login to GHCR
        run: |
          echo ${{ secrets.GITHUB_TOKEN }} | helm registry login ghcr.io -u ${{ github.actor }} --password-stdin
      
      - name: Package chart
        run: |
          helm dependency update charts/k8s-pod-postmortem
          helm package charts/k8s-pod-postmortem
      
      - name: Push chart
        run: |
          helm push k8s-pod-postmortem-*.tgz oci://ghcr.io/bansikah22/charts

  release:
    runs-on: ubuntu-latest
    needs: [build, docker, helm]
    steps:
      - uses: actions/checkout@v4
      
      - name: Download binaries
        uses: actions/download-artifact@v4
        with:
          name: binaries
          path: bin/
      
      - name: Create release
        uses: softprops/action-gh-release@v1
        with:
          files: bin/*
          generate_release_notes: true
```

## Hotfix Releases

For critical bug fixes:

1. **Create hotfix branch**

```bash
git checkout -b hotfix/v1.0.1 v1.0.0
```

2. **Apply fix**

```bash
# Make necessary changes
git add .
git commit -s -m "fix: critical bug description"
```

3. **Create hotfix tag**

```bash
git tag -a v1.0.1 -m "Hotfix v1.0.1"
git push origin v1.0.1
```

4. **Merge back to main**

```bash
git checkout main
git merge hotfix/v1.0.1
git push origin main
```

## Pre-Release Versions

For testing new features:

1. **Create pre-release tag**

```bash
git tag -a v1.1.0-beta.1 -m "Pre-release v1.1.0-beta.1"
git push origin v1.1.0-beta.1
```

2. **Mark as pre-release in GitHub**

When creating the release, check "This is a pre-release" option.

3. **Use pre-release version**

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1.1.0-beta.1
```

## Rollback

If a release has critical issues:

1. **Revert to previous version**

```bash
# Delete the tag
git tag -d v1.0.1
git push origin :refs/tags/v1.0.1

# Delete the release from GitHub UI
```

2. **Re-release previous version**

```bash
# Re-tag the previous version as latest
git tag -f v1.0.0 v1.0.0
git push origin v1.0.0 --force
```

3. **Re-publish Docker image**

```bash
# Re-tag and push previous image
docker pull ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0
docker tag ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0 ghcr.io/bansikah22/k8s-pod-postmortem:latest
docker push ghcr.io/bansikah22/k8s-pod-postmortem:latest
```

## Version Compatibility

| k8s-pod-postmortem | Kubernetes | Go |
|--------------------|------------|-----|
| v1.0.x | 1.20+ | 1.21+ |
| v0.1.x | 1.20+ | 1.21+ |

## Security Releases

For security fixes:

1. **Create security advisory** (private)
2. **Fix vulnerability** (private branch)
3. **Request CVE** (if applicable)
4. **Release security fix**
5. **Publish security advisory**

See [SECURITY.md](../SECURITY.md) for details.

## Changelog Format

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New features

### Changed
- Changes

### Fixed
- Bug fixes

## [1.0.0] - 2024-01-15

### Added
- Initial release
- Kubernetes pod diagnostics collection
- Failure classification engine
- Secret redaction
- GitHub Actions integration
```

## Release Announcements

After a successful release:

1. **GitHub Release Notes** - Auto-generated
2. **Documentation Update** - Update version references
3. **Social Media** - Announce on relevant channels
4. **Slack/Discord** - Notify community channels

## Support Policy

- **Latest version**: Full support
- **Previous minor version**: Bug fixes only
- **Older versions**: Security fixes only
- **End of life**: No support