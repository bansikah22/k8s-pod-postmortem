# API Reference

This document provides detailed API reference for k8s-pod-postmortem internal packages.

## Table of Contents

- [types](#types)
- [kubernetes](#kubernetes)
- [analysis](#analysis)
- [redact](#redact)
- [reporting](#reporting)
- [github](#github)

---

## types

Package types provides shared types for the k8s-pod-postmortem action.

### Types

#### Diagnostics

```go
type Diagnostics struct {
    Namespace      string
    PodName        string
    Pod            *corev1.Pod
    Events         []corev1.Event
    PreviousLogs   map[string]string
    Node           *corev1.Node
    ContainerStats []ContainerStats
}
```

Diagnostics contains all collected diagnostic information.

| Field | Type | Description |
|-------|------|-------------|
| `Namespace` | `string` | Kubernetes namespace |
| `PodName` | `string` | Pod name |
| `Pod` | `*corev1.Pod` | Pod object from Kubernetes API |
| `Events` | `[]corev1.Event` | List of events related to the pod |
| `PreviousLogs` | `map[string]string` | Previous container logs by container name |
| `Node` | `*corev1.Node` | Node information (optional) |
| `ContainerStats` | `[]ContainerStats` | Container status information |

---

#### ContainerStats

```go
type ContainerStats struct {
    Name         string
    State        string
    Reason       string
    ExitCode     int32
    RestartCount int32
    LastState    string
}
```

ContainerStats holds container status information.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Container name |
| `State` | `string` | Current state (Running, Waiting, Terminated) |
| `Reason` | `string` | State reason (e.g., OOMKilled, CrashLoopBackOff) |
| `ExitCode` | `int32` | Container exit code |
| `RestartCount` | `int32` | Number of restarts |
| `LastState` | `string` | Last termination state |

---

#### Event

```go
type Event struct {
    Type    string
    Reason  string
    Message string
    Source  string
    Count   int32
    Time    time.Time
}
```

Event represents a simplified Kubernetes event.

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Event type (Normal, Warning) |
| `Reason` | `string` | Event reason |
| `Message` | `string` | Event message |
| `Source` | `string` | Event source |
| `Count` | `int32` | Number of occurrences |
| `Time` | `time.Time` | Event timestamp |

---

#### Classification

```go
type Classification struct {
    Type            FailureType
    Reason          string
    Message         string
    Recommendations []string
    Confidence      float64
}
```

Classification represents the result of failure analysis.

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `FailureType` | Classification type |
| `Reason` | `string` | Failure reason |
| `Message` | `string` | Human-readable message |
| `Recommendations` | `[]string` | List of recommendations |
| `Confidence` | `float64` | Confidence score (0-1) |

---

#### FailureType

```go
type FailureType string

const (
    MemoryExhaustion  FailureType = "MEMORY_EXHAUSTION"
    NodePressure      FailureType = "NODE_PRESSURE"
    CapacityShortage  FailureType = "CAPACITY_SHORTAGE"
    ImageFailure      FailureType = "IMAGE_FAILURE"
    ApplicationCrash  FailureType = "APPLICATION_CRASH"
    Preemption        FailureType = "PREEMPTION"
    NodeFailure       FailureType = "NODE_FAILURE"
    Timeout           FailureType = "TIMEOUT"
    NetworkFailure    FailureType = "NETWORK_FAILURE"
    Unknown           FailureType = "UNKNOWN"
)
```

FailureType represents the classification of a failure.

---

#### Config

```go
type Config struct {
    LogTailLines   int
    IncludeNodeInfo bool
    RedactSecrets   bool
    OutputFormat    string
    Namespace       string
    PodName         string
    Debug           bool
    Timeout         time.Duration
}
```

Config holds the action configuration.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `LogTailLines` | `int` | `200` | Number of log lines to collect |
| `IncludeNodeInfo` | `bool` | `false` | Include node information |
| `RedactSecrets` | `bool` | `true` | Enable secret redaction |
| `OutputFormat` | `string` | `"markdown"` | Output format |
| `Namespace` | `string` | `""` | Target namespace (auto-discovered if empty) |
| `PodName` | `string` | `""` | Target pod (auto-discovered if empty) |
| `Debug` | `bool` | `false` | Enable debug logging |
| `Timeout` | `time.Duration` | `10s` | API timeout |

---

## kubernetes

Package kubernetes provides Kubernetes client functionality for pod diagnostics.

### Client

```go
type Client struct {
    clientset kubernetes.Interface
}
```

Client handles Kubernetes API interactions.

### Functions

#### NewClient

```go
func NewClient() (*Client, error)
```

NewClient creates a new Kubernetes client using in-cluster configuration.

**Returns:**
- `*Client` - Kubernetes client
- `error` - Error if client creation fails

**Example:**

```go
client, err := kubernetes.NewClient()
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
```

---

#### DiscoverPodInfo

```go
func (c *Client) DiscoverPodInfo(ctx context.Context, namespace, podName string) (string, string, error)
```

DiscoverPodInfo discovers namespace and pod name if not provided.

**Parameters:**
- `ctx` - Context for cancellation
- `namespace` - Namespace (empty for auto-discovery)
- `podName` - Pod name (empty for auto-discovery)

**Returns:**
- `string` - Namespace
- `string` - Pod name
- `error` - Error if discovery fails

**Example:**

```go
namespace, podName, err := client.DiscoverPodInfo(ctx, "", "")
if err != nil {
    log.Fatalf("Failed to discover pod info: %v", err)
}
```

---

#### CollectDiagnostics

```go
func (c *Client) CollectDiagnostics(ctx context.Context, namespace, podName string, logTailLines int, includeNodeInfo bool) (*types.Diagnostics, error)
```

CollectDiagnostics gathers all diagnostic information for the pod.

**Parameters:**
- `ctx` - Context for cancellation
- `namespace` - Target namespace
- `podName` - Target pod name
- `logTailLines` - Number of log lines to collect
- `includeNodeInfo` - Whether to include node information

**Returns:**
- `*types.Diagnostics` - Collected diagnostics
- `error` - Error if collection fails

**Example:**

```go
diagnostics, err := client.CollectDiagnostics(ctx, "default", "my-pod", 200, false)
if err != nil {
    log.Fatalf("Failed to collect diagnostics: %v", err)
}
```

---

#### GetPod

```go
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
```

GetPod retrieves pod information from Kubernetes API.

**Parameters:**
- `ctx` - Context for cancellation
- `namespace` - Pod namespace
- `name` - Pod name

**Returns:**
- `*corev1.Pod` - Pod object
- `error` - Error if retrieval fails

---

#### GetEvents

```go
func (c *Client) GetEvents(ctx context.Context, namespace, podName string) ([]corev1.Event, error)
```

GetEvents retrieves events for the pod.

**Parameters:**
- `ctx` - Context for cancellation
- `namespace` - Pod namespace
- `podName` - Pod name

**Returns:**
- `[]corev1.Event` - List of events
- `error` - Error if retrieval fails

---

#### GetPreviousLogs

```go
func (c *Client) GetPreviousLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64) (string, error)
```

GetPreviousLogs retrieves logs from the previous container instance.

**Parameters:**
- `ctx` - Context for cancellation
- `namespace` - Pod namespace
- `podName` - Pod name
- `containerName` - Container name
- `tailLines` - Number of lines to retrieve

**Returns:**
- `string` - Container logs
- `error` - Error if retrieval fails

---

#### GetNode

```go
func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error)
```

GetNode retrieves node information.

**Parameters:**
- `ctx` - Context for cancellation
- `name` - Node name

**Returns:**
- `*corev1.Node` - Node object
- `error` - Error if retrieval fails

---

## analysis

Package analysis provides failure classification and analysis functionality.

### Analyzer

```go
type Analyzer struct {
    rules []classificationRule
}
```

Analyzer performs failure classification.

### Functions

#### NewAnalyzer

```go
func NewAnalyzer() *Analyzer
```

NewAnalyzer creates a new Analyzer with default rules.

**Returns:**
- `*Analyzer` - Analyzer instance

**Example:**

```go
analyzer := analysis.NewAnalyzer()
```

---

#### Analyze

```go
func (a *Analyzer) Analyze(diagnostics *types.Diagnostics) types.Classification
```

Analyze classifies the failure based on diagnostics.

**Parameters:**
- `diagnostics` - Collected diagnostics

**Returns:**
- `types.Classification` - Failure classification

**Example:**

```go
classification := analyzer.Analyze(diagnostics)
fmt.Printf("Failure type: %s\n", classification.Type)
fmt.Printf("Confidence: %.0f%%\n", classification.Confidence * 100)
```

---

### Classification Rules

The analyzer uses priority-based rules to classify failures:

| Priority | Reason | Type | Confidence |
|----------|--------|------|------------|
| 100 | OOMKilled | MemoryExhaustion | 95% |
| 90 | Evicted | NodePressure | 95% |
| 85 | FailedScheduling | CapacityShortage | 90% |
| 80 | ImagePullBackOff/ErrImagePull | ImageFailure | 90% |
| 75 | CrashLoopBackOff | ApplicationCrash | 85% |
| 70 | Preempted | Preemption | 95% |
| 65 | NodeNotReady | NodeFailure | 90% |
| 60 | DeadlineExceeded | Timeout | 85% |
| 55 | NetworkPluginError | NetworkFailure | 80% |

---

## redact

Package redact provides sensitive data redaction functionality.

### Redactor

```go
type Redactor struct {
    patterns []*redactionPattern
}
```

Redactor handles sensitive data redaction.

### Functions

#### NewRedactor

```go
func NewRedactor() *Redactor
```

NewRedactor creates a new Redactor with default patterns.

**Returns:**
- `*Redactor` - Redactor instance

**Example:**

```go
redactor := redact.NewRedactor()
```

---

#### Redact

```go
func (r *Redactor) Redact(text string) string
```

Redact removes sensitive information from text.

**Parameters:**
- `text` - Text to redact

**Returns:**
- `string` - Redacted text

**Example:**

```go
redactor := redact.NewRedactor()
logs := "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE"
clean := redactor.Redact(logs)
// Output: AWS_ACCESS_KEY_ID=[REDACTED_AWS_ACCESS_KEY_ID]
```

---

#### RedactDiagnostics

```go
func (r *Redactor) RedactDiagnostics(diagnostics *types.Diagnostics) *types.Diagnostics
```

RedactDiagnostics redacts sensitive data from diagnostics.

**Parameters:**
- `diagnostics` - Diagnostics to redact

**Returns:**
- `*types.Diagnostics` - Redacted diagnostics

---

### Redaction Patterns

Default patterns include:

| Pattern | Regex | Replacement |
|---------|-------|-------------|
| AWS Access Key ID | `AWS_ACCESS_KEY_ID=...` | `[REDACTED_AWS_ACCESS_KEY_ID]` |
| AWS Secret Key | `AWS_SECRET_ACCESS_KEY=...` | `[REDACTED_AWS_SECRET_ACCESS_KEY]` |
| GitHub PAT | `ghp_[a-zA-Z0-9]{36}` | `[REDACTED_GITHUB_PAT]` |
| Bearer Token | `Bearer ...` | `[REDACTED_BEARER_TOKEN]` |
| Private Key | `-----BEGIN PRIVATE KEY-----` | `[REDACTED_PRIVATE_KEY]` |
| Database URL | `(postgres\|mysql\|mongodb)://...` | `[REDACTED_DATABASE_URL]` |
| Password | `password=...` | `[REDACTED]` |

---

## reporting

Package reporting provides markdown report generation functionality.

### Reporter

```go
type Reporter struct {}
```

Reporter generates markdown reports for post-mortem diagnostics.

### Functions

#### NewReporter

```go
func NewReporter() *Reporter
```

NewReporter creates a new Reporter.

**Returns:**
- `*Reporter` - Reporter instance

---

#### GenerateMarkdown

```go
func (r *Reporter) GenerateMarkdown(diagnostics *types.Diagnostics, classification types.Classification) string
```

GenerateMarkdown generates a markdown report from diagnostics and classification.

**Parameters:**
- `diagnostics` - Collected diagnostics
- `classification` - Failure classification

**Returns:**
- `string` - Markdown report

**Example:**

```go
reporter := reporting.NewReporter()
report := reporter.GenerateMarkdown(diagnostics, classification)
fmt.Println(report)
```

---

### Report Structure

The generated report includes:

1. **Header** - Title and timestamp
2. **Failure Classification** - Type, reason, message, confidence
3. **Pod Information** - Namespace, pod name, node, creation time
4. **Container Status** - Table of container states
5. **Events Timeline** - Chronological event list
6. **Previous Logs** - Container logs from previous instance
7. **Recommendations** - Actionable recommendations
8. **Footer** - Generated by notice

---

## github

Package github provides GitHub Actions integration functionality.

### Client

```go
type Client struct {
    outputFile  string
    summaryFile string
    workspace   string
}
```

Client handles GitHub Actions integration.

### Functions

#### NewClient

```go
func NewClient() *Client
```

NewClient creates a GitHub client from environment variables.

**Returns:**
- `*Client` - GitHub client

**Example:**

```go
client := github.NewClient()
```

---

#### WriteSummary

```go
func (c *Client) WriteSummary(report string) error
```

WriteSummary writes content to the GitHub Actions step summary.

**Parameters:**
- `report` - Markdown content to write

**Returns:**
- `error` - Error if write fails

**Example:**

```go
client := github.NewClient()
err := client.WriteSummary(report)
if err != nil {
    log.Fatalf("Failed to write summary: %v", err)
}
```

---

#### SetOutput

```go
func (c *Client) SetOutput(name, value string) error
```

SetOutput sets a GitHub Actions output variable.

**Parameters:**
- `name` - Output variable name
- `value` - Output value

**Returns:**
- `error` - Error if write fails

**Example:**

```go
client := github.NewClient()
err := client.SetOutput("failure-type", "MEMORY_EXHAUSTION")
if err != nil {
    log.Fatalf("Failed to set output: %v", err)
}
```

---

#### GetWorkspace

```go
func (c *Client) GetWorkspace() string
```

GetWorkspace returns the GitHub workspace path.

**Returns:**
- `string` - Workspace path

---

#### IsRunningInGitHubActions

```go
func (c *Client) IsRunningInGitHubActions() bool
```

IsRunningInGitHubActions returns true if running in GitHub Actions.

**Returns:**
- `bool` - True if in GitHub Actions

---

#### GetEnv

```go
func (c *Client) GetEnv(name string) string
```

GetEnv returns a GitHub Actions environment variable.

**Parameters:**
- `name` - Environment variable name

**Returns:**
- `string` - Environment variable value

---

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_STEP_SUMMARY` | Path to step summary file |
| `GITHUB_OUTPUT` | Path to output file |
| `GITHUB_WORKSPACE` | Workspace directory path |

---

## Error Handling

All packages return errors as the last return value. Always handle errors appropriately:

```go
client, err := kubernetes.NewClient()
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}

diagnostics, err := client.CollectDiagnostics(ctx, namespace, podName, 200, false)
if err != nil {
    log.Fatalf("Failed to collect diagnostics: %v", err)
}
```

## Context Usage

All Kubernetes API calls support context for cancellation and timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

diagnostics, err := client.CollectDiagnostics(ctx, namespace, podName, 200, false)
```

## Thread Safety

- `Client` instances are safe for concurrent use
- `Analyzer` instances are safe for concurrent use
- `Redactor` instances are safe for concurrent use
- `Reporter` instances are safe for concurrent use
- `GitHub Client` instances are NOT safe for concurrent use (create per goroutine)