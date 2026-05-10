# k8s-pod-postmortem

Kubernetes CI/CD Failure Post-Mortem Action - Automatically capture, preserve, analyze, and surface pod-level failure diagnostics for ephemeral GitHub Actions runners.

## Overview

When GitHub Actions jobs run on self-hosted Kubernetes runners, runner pods are typically ephemeral and terminate immediately after failure. This action automatically collects failure context and publishes a structured post-mortem report directly into the GitHub Actions Job Summary UI.

## Features

- Automatic pod diagnostics collection on CI/CD failures
- Preserves ephemeral runtime evidence
- Publishes readable post-mortem summaries in GitHub
- Eliminates manual kubectl investigation for common failures
- Uses secure short-lived authentication mechanisms
- Follows Kubernetes and cloud security best practices

## Supported Failure Types

| Failure Type | Description |
|--------------|-------------|
| OOMKilled | Container exceeded memory limit |
| Evicted | Node under memory/disk pressure |
| Preempted | Spot/preemptible node interruption |
| CrashLoopBackOff | Application repeatedly crashing |
| ImagePullBackOff | Registry/image authentication issues |
| FailedScheduling | Resource constraints or taints |
| NodeNotReady | Worker node unavailable |
| DeadlineExceeded | Pod startup timeout |
| NetworkFailure | CNI/network plugin issues |

## Prerequisites

- Kubernetes cluster with self-hosted GitHub Actions runners
- RBAC permissions for the action (see RBAC section)
- Go 1.22+ (for building from source)

## Installation

### As GitHub Action

```yaml
- name: Kubernetes Post-Mortem
  if: failure() || cancelled()
  uses: bansikah22/k8s-pod-postmortem@v0.1.0
  with:
    log-tail-lines: '200'
    include-node-info: 'true'
    redact-secrets: 'true'
```

### Using Helm

```bash
helm install k8s-pod-postmortem ./charts/k8s-pod-postmortem
```

## Configuration

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `log-tail-lines` | Number of log lines to tail from previous container | No | `200` |
| `include-node-info` | Include node information in diagnostics | No | `false` |
| `redact-secrets` | Redact sensitive data from output | No | `true` |
| `output-format` | Output format (markdown) | No | `markdown` |
| `namespace` | Kubernetes namespace (auto-detected if empty) | No | `""` |
| `pod-name` | Pod name (auto-detected if empty) | No | `""` |
| `debug` | Enable debug logging | No | `false` |
| `timeout` | Action timeout in seconds | No | `10s` |

### Action Outputs

| Output | Description |
|--------|-------------|
| `failure-type` | Classification of the failure type |
| `failure-reason` | Reason for the failure |
| `namespace` | Kubernetes namespace of the pod |
| `pod-name` | Name of the pod |

## RBAC Requirements

The action requires the following RBAC permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-pod-postmortem
rules:
- apiGroups: [""]
  resources:
    - pods
    - pods/log
    - events
  verbs:
    - get
    - list
    - watch
```

Optional (for node information):

```yaml
- apiGroups: [""]
  resources:
    - nodes
  verbs:
    - get
```

## Security

### Authentication

The action uses in-cluster authentication via the service account token mounted in the pod. No static credentials or kubeconfigs are required.

### Secret Redaction

The action automatically redacts sensitive data from logs and output:

- AWS credentials (Access Key ID, Secret Access Key, Session Token)
- GitHub tokens (PAT, OAuth tokens, App tokens)
- Authorization headers
- Private keys
- Database connection strings
- Passwords and API keys
- Kubernetes tokens

### Least Privilege

The action follows the principle of least privilege with namespace-scoped RBAC.

## Development

### Prerequisites

- Go 1.22+
- Docker (for container builds)
- Make
- golangci-lint (for linting)

### Installing golangci-lint

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Windows
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/bansikah22/k8s-pod-postmortem.git
cd k8s-pod-postmortem

# Download dependencies
go mod download

# Build binary
make build

# Run linter
make lint

# Run unit tests
make test

# Run with coverage
make coverage
```

### Docker Build

```bash
make docker-build
```

### Running Locally

The binary can be run locally for testing outside of Kubernetes:

```bash
# Build first
make build

# Run with flags (simulates running in a pod)
./bin/postmortem \
  --namespace=default \
  --pod-name=my-pod \
  --log-tail-lines=200 \
  --include-node-info=false \
  --redact-secrets=true \
  --debug
```

**Note**: When running locally without Kubernetes cluster access, the action will fail to connect to the Kubernetes API. For local testing, you can:

1. Use a kubeconfig file pointing to a test cluster
2. Use kind or minikube for local Kubernetes
3. Run only the unit tests

### Testing Without Kubernetes Cluster

#### Unit Tests Only

Run unit tests that don't require cluster access:

```bash
# Run all unit tests
go test -v ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Using kind (Kubernetes in Docker)

For local integration testing:

```bash
# Install kind
go install sigs.k8s.io/kind@latest

# Create a cluster
kind create cluster --name test-cluster

# Set kubeconfig
export KUBECONFIG="$(kind get kubeconfig-path --name test-cluster)"

# Run integration tests
make test

# Delete cluster when done
kind delete cluster --name test-cluster
```

#### Using minikube

```bash
# Install minikube (follow instructions at https://minikube.sigs.k8s.io/)

# Start cluster
minikube start

# Run tests
make test

# Stop cluster
minikube stop
```

#### Mock Testing

The project includes mock implementations for testing without a real cluster. The unit tests in `internal/analysis/analyzer_test.go` and `internal/redact/redactor_test.go` demonstrate testing without Kubernetes dependencies.

### Testing Specific Components

```bash
# Test analyzer package
go test -v ./internal/analysis/...

# Test redactor package
go test -v ./internal/redact/...

# Test reporting package
go test -v ./internal/reporting/...

# Test with verbose output
go test -v -run TestAnalyzer_Analyze ./internal/analysis/...
```

### Local Development Workflow

```bash
# 1. Make code changes

# 2. Run linter
make lint

# 3. Run unit tests
make test

# 4. Build binary
make build

# 5. Test with kind cluster (optional)
kind create cluster
make test
kind delete cluster
```

## Example Output

```markdown
# Kubernetes Pod Post-Mortem

## Failure Classification
MEMORY_EXHAUSTION

## Pod Information
**Namespace:** `github-runners`
**Pod:** `runner-x7gh2`
**Node:** `worker-node-1`

## Container Status

| Container | State | Reason | Exit Code | Restarts |
|-----------|-------|--------|-----------|----------|
| runner | Terminated | OOMKilled | 137 | 3 |

## Events Timeline

- Warning OOMKilling
- Container terminated
- Pod restarted

## Previous Container Logs

java.lang.OutOfMemoryError: Java heap space

## Recommendations

1. Increase memory limits for the container
2. Review application memory usage patterns
3. Consider implementing vertical pod autoscaling
4. Add memory profiling to identify leaks
```

## Cloud Provider Authentication

### AWS EKS

Use IRSA (IAM Roles for Service Accounts) or GitHub OIDC federation.

### GKE

Use Workload Identity Federation.

### AKS

Use Azure Federated Identity Credentials.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linter
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

## Support

For issues and feature requests, please use the [GitHub Issues](https://github.com/bansikah22/k8s-pod-postmortem/issues) page.
