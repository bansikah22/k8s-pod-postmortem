# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-05-12

### Added
- Initial stable release of k8s-pod-postmortem
- Kubernetes pod diagnostics collection for failed pods
- Failure classification engine supporting:
  - OOMKilled pods
  - Evicted pods
  - CrashLoopBackOff pods
  - ImagePullBackOff pods
  - FailedScheduling events
- Secret redaction for sensitive data (AWS keys, GitHub tokens, passwords, etc.)
- GitHub Actions integration with outputs for failure-type and failure-reason
- Helm chart for Kubernetes deployment
- Kubeconfig support for local testing with minikube/kind
- Comprehensive documentation suite:
  - Architecture overview
  - API reference
  - Deployment guide
  - Usage examples
  - Troubleshooting guide
- CI/CD pipeline with:
  - Go 1.23 support
  - Security scanning
  - SBOM generation
  - Multi-architecture Docker builds (amd64, arm64)
- Test coverage for analysis and redaction modules

### Security
- Implemented secret redaction to prevent credential leakage
- Updated Go to 1.23 for security patches
- Addressed security vulnerabilities in dependencies

[Unreleased]: https://github.com/bansikah22/k8s-pod-postmortem/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/bansikah22/k8s-pod-postmortem/releases/tag/v1.0.0