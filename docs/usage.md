# Usage

This document provides comprehensive usage instructions for k8s-pod-postmortem.

## Quick Start

### Basic Usage

Add the action to your workflow to automatically capture diagnostics when a job fails:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Build
        run: make build
      
      - name: Test
        run: make test
      
      - name: Post-mortem on failure
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
```

### With Cancelled Jobs

Capture diagnostics for both failed and cancelled jobs:

```yaml
- name: Post-mortem on failure or cancellation
  if: failure() || cancelled()
  uses: bansikah22/k8s-pod-postmortem@v1
```

## Configuration Options

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `log-tail-lines` | Number of log lines to tail from previous container | No | `200` |
| `include-node-info` | Include node information in diagnostics | No | `false` |
| `redact-secrets` | Redact sensitive data from output | No | `true` |
| `output-format` | Output format (currently only `markdown`) | No | `markdown` |
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

## Usage Examples

### Minimal Configuration

```yaml
jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Run tests
        run: npm test
      
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
```

### Full Configuration

```yaml
jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Run tests
        run: npm test
      
      - name: Post-mortem on failure
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
        with:
          log-tail-lines: '500'
          include-node-info: 'true'
          redact-secrets: 'true'
          output-format: 'markdown'
          namespace: ''
          pod-name: ''
          debug: 'false'
          timeout: '30s'
```

### Using Outputs

Access action outputs in subsequent steps:

```yaml
jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Run tests
        run: npm test
      
      - name: Post-mortem
        if: failure()
        id: postmortem
        uses: bansikah22/k8s-pod-postmortem@v1
      
      - name: Create issue on failure
        if: failure() && steps.postmortem.outputs.failure-type == 'MEMORY_EXHAUSTION'
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: 'OOMKilled: ${{ steps.postmortem.outputs.pod-name }}',
              body: 'Pod ${{ steps.postmortem.outputs.pod-name }} in namespace ${{ steps.postmortem.outputs.namespace }} was OOMKilled.',
              labels: ['bug', 'oom']
            })
```

### Conditional Notifications

Send notifications based on failure type:

```yaml
jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Run tests
        run: npm test
      
      - name: Post-mortem
        if: failure()
        id: postmortem
        uses: bansikah22/k8s-pod-postmortem@v1
      
      - name: Notify Slack on critical failure
        if: failure() && contains('CAPACITY_SHORTAGE,NODE_FAILURE', steps.postmortem.outputs.failure-type)
        uses: slackapi/slack-webhook@v1
        with:
          payload: |
            {
              "text": "Critical Kubernetes failure: ${{ steps.postmortem.outputs.failure-type }}",
              "blocks": [
                {
                  "type": "section",
                  "text": {
                    "type": "mrkdwn",
                    "text": "*Failure Type:* ${{ steps.postmortem.outputs.failure-type }}\n*Reason:* ${{ steps.postmortem.outputs.failure-reason }}"
                  }
                }
              ]
            }
```

### Multi-Container Pods

For pods with multiple containers, logs from all containers are collected:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    log-tail-lines: '100'  # Per container
```

### Specific Namespace

Target a specific namespace:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    namespace: 'production'
    pod-name: 'my-app-abc123-xyz'
```

### Debug Mode

Enable debug logging for troubleshooting:

```yaml
- name: Post-mortem with debug
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    debug: 'true'
```

## Workflow Patterns

### Matrix Builds

Use with matrix builds to capture diagnostics for each job:

```yaml
jobs:
  test:
    runs-on: [self-hosted, kubernetes]
    strategy:
      matrix:
        node: [16, 18, 20]
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node }}
      
      - name: Test
        run: npm test
      
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
        with:
          log-tail-lines: '300'
```

### Multi-Stage Pipeline

Capture diagnostics at each stage:

```yaml
jobs:
  lint:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      - run: npm run lint
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1

  test:
    runs-on: [self-hosted, kubernetes]
    needs: lint
    steps:
      - uses: actions/checkout@v4
      - run: npm test
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1

  build:
    runs-on: [self-hosted, kubernetes]
    needs: test
    steps:
      - uses: actions/checkout@v4
      - run: npm run build
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
```

### With Service Dependencies

For workflows with service containers:

```yaml
jobs:
  integration-test:
    runs-on: [self-hosted, kubernetes]
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5432:5432
      redis:
        image: redis:7
        ports:
          - 6379:6379
    steps:
      - uses: actions/checkout@v4
      
      - name: Run integration tests
        run: npm run test:integration
        env:
          DATABASE_URL: postgres://postgres:postgres@postgres:5432/test
          REDIS_URL: redis://redis:6379
      
      - name: Post-mortem
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
        with:
          include-node-info: 'true'
```

## Generated Report

The action generates a markdown report in the GitHub Actions Job Summary:

### Report Structure

```markdown
## 🔍 Pod Post-Mortem Report

### Summary
- **Pod**: runner-abc123-xyz
- **Namespace**: actions-runners
- **Classification**: MEMORY_EXHAUSTION
- **Confidence**: 95%

### Container Status
| Container | State | Reason | Exit Code | Restarts |
|-----------|-------|--------|-----------|----------|
| runner | Terminated | OOMKilled | 137 | 3 |

### Events Timeline
| Time | Type | Reason | Message |
|------|------|--------|---------|
| 2024-01-15T10:30:00Z | Warning | BackOff | Container failed to start |

### Previous Logs
```
[Container logs from previous instance]
```

### Recommendations
1. Increase memory limits for the container
2. Review application memory usage patterns
3. Consider implementing vertical pod autoscaling
4. Add memory profiling to identify leaks
```

### Accessing the Report

1. Navigate to your GitHub repository
2. Click on "Actions" tab
3. Select the failed workflow run
4. Expand the failed job
5. Click on "Post-mortem" step
6. View the report in the step summary

## Failure Classifications

The action classifies failures into the following types:

| Classification | Reason | Description |
|----------------|--------|-------------|
| `MEMORY_EXHAUSTION` | OOMKilled | Container exceeded memory limit |
| `NODE_PRESSURE` | Evicted | Pod evicted due to node resource pressure |
| `CAPACITY_SHORTAGE` | FailedScheduling | Pod could not be scheduled |
| `IMAGE_FAILURE` | ImagePullBackOff | Container image pull failed |
| `APPLICATION_CRASH` | CrashLoopBackOff | Application repeatedly crashing |
| `PREEMPTION` | Preempted | Spot/preemptible node interruption |
| `NODE_FAILURE` | NodeNotReady | Worker node unavailable |
| `TIMEOUT` | DeadlineExceeded | Pod startup timeout |
| `NETWORK_FAILURE` | NetworkPluginError | CNI/network issues |
| `UNKNOWN` | - | Unclassified failure |

## Best Practices

### 1. Use Auto-Detection

Let the action auto-detect namespace and pod name:

```yaml
# Good - Auto-detection
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1

# Avoid - Hardcoded values
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    namespace: 'actions-runners'
    pod-name: 'runner-abc123'  # Pod name changes each run
```

### 2. Enable Secret Redaction

Always keep secret redaction enabled:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    redact-secrets: 'true'  # Default, but explicit is better
```

### 3. Adjust Log Lines

Adjust log lines based on your needs:

```yaml
# For quick diagnosis
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    log-tail-lines: '100'

# For detailed analysis
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    log-tail-lines: '1000'
```

### 4. Use Outputs for Automation

Leverage outputs for automated responses:

```yaml
- name: Post-mortem
  if: failure()
  id: postmortem
  uses: bansikah22/k8s-pod-postmortem@v1

- name: Auto-remediate known issues
  if: failure() && steps.postmortem.outputs.failure-type == 'CAPACITY_SHORTAGE'
  run: |
    # Trigger cluster autoscaler or notify ops team
    echo "Capacity issue detected, scaling cluster..."
```

### 5. Combine with Other Actions

Use with other diagnostic actions:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1

- name: Upload diagnostics
  if: failure()
  uses: actions/upload-artifact@v4
  with:
    name: diagnostics
    path: diagnostics/
```

## Limitations

### Runner Requirements

- Must run on self-hosted Kubernetes runners
- Runner pod must have appropriate RBAC permissions
- Service account must be configured with the action

### Access Scope

- Can only access pods in the same namespace as the runner
- Cannot access cluster-level resources
- Limited to read-only operations

### Log Availability

- Previous logs only available if container restarted
- Logs may be truncated based on `log-tail-lines` setting
- Some ephemeral containers may not have logs

## Troubleshooting

For common issues and solutions, see the [Troubleshooting Guide](./troubleshooting.md).