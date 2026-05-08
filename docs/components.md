# Components

This document provides detailed information about the internal components of k8s-pod-postmortem.

## Component Overview

```mermaid
flowchart TB
    subgraph "Entry Point"
        MAIN[cmd/postmortem/main.go]
    end

    subgraph "Internal Packages"
        K8S[internal/kubernetes]
        ANALYSIS[internal/analysis]
        REDACT[internal/redact]
        REPORT[internal/reporting]
        GH[internal/github]
        TYPES[internal/types]
    end

    MAIN --> K8S
    MAIN --> ANALYSIS
    MAIN --> REDACT
    MAIN --> REPORT
    MAIN --> GH
    MAIN --> TYPES
```

## cmd/postmortem

The entry point of the application. This package orchestrates the entire diagnostic collection and reporting process.

### main.go

**Purpose**: Initialize and coordinate all components to collect diagnostics and generate reports.

**Responsibilities**:
- Parse command-line flags and configuration
- Initialize Kubernetes client
- Discover pod information (namespace and pod name)
- Coordinate diagnostic collection
- Trigger analysis
- Generate and output reports

**Key Functions**:
- `main()` - Entry point that orchestrates the workflow

## internal/types

Shared type definitions used across all packages.

### types.go

**Purpose**: Define all shared data structures and constants.

**Key Types**:

| Type | Description |
|------|-------------|
| `Diagnostics` | Container for all collected diagnostic information |
| `ContainerStats` | Container status and state information |
| `Event` | Simplified Kubernetes event representation |
| `Classification` | Failure classification result with recommendations |
| `Config` | Action configuration options |

**Failure Types**:

| Constant | Description |
|----------|-------------|
| `MemoryExhaustion` | OOMKilled containers |
| `NodePressure` | Evicted pods due to resource pressure |
| `CapacityShortage` | Failed scheduling |
| `ImageFailure` | Image pull failures |
| `ApplicationCrash` | Application crashes (CrashLoopBackOff) |
| `Preemption` | Spot/preemptible node interruption |
| `NodeFailure` | Node unavailable |
| `Timeout` | Deadline exceeded |
| `NetworkFailure` | CNI/network issues |
| `Unknown` | Unclassified failures |

## internal/kubernetes

Kubernetes API client for collecting diagnostic data.

### client.go

**Purpose**: Interact with the Kubernetes API to collect pod diagnostics.

**Key Functions**:

| Function | Description |
|----------|-------------|
| `NewClient()` | Create a new Kubernetes client using in-cluster config |
| `DiscoverPodInfo()` | Discover namespace and pod name automatically |
| `CollectDiagnostics()` | Gather all diagnostic information |
| `GetPod()` | Retrieve pod information |
| `GetEvents()` | Retrieve events for a pod |
| `GetPreviousLogs()` | Retrieve previous container logs |
| `GetNode()` | Retrieve node information (optional) |

**Authentication**:
- Uses in-cluster service account token
- Reads namespace from `/var/run/secrets/kubernetes.io/serviceaccount/namespace`
- Relies on RBAC permissions configured via Helm chart

**Data Collection Flow**:

```mermaid
sequenceDiagram
    participant Client
    participant K8sAPI as Kubernetes API
    
    Client->>K8sAPI: GetPod(namespace, podName)
    K8sAPI-->>Client: Pod details
    
    Client->>K8sAPI: GetEvents(namespace, podName)
    K8sAPI-->>Client: Event list
    
    loop For each container
        Client->>K8sAPI: GetPreviousLogs(namespace, pod, container)
        K8sAPI-->>Client: Container logs
    end
    
    opt If includeNodeInfo
        Client->>K8sAPI: GetNode(nodeName)
        K8sAPI-->>Client: Node details
    end
```

## internal/analysis

Failure classification and analysis engine.

### analyzer.go

**Purpose**: Analyze collected diagnostics and classify failures.

**Key Components**:

| Component | Description |
|-----------|-------------|
| `Analyzer` | Main analyzer struct with classification rules |
| `classificationRule` | Rule definition for matching and classifying failures |

**Classification Rules** (by priority):

| Priority | Reason | Failure Type | Confidence |
|----------|--------|--------------|------------|
| 100 | OOMKilled | MemoryExhaustion | 95% |
| 90 | Evicted | NodePressure | 95% |
| 85 | FailedScheduling | CapacityShortage | 90% |
| 80 | ImagePullBackOff/ErrImagePull | ImageFailure | 90% |
| 75 | CrashLoopBackOff | ApplicationCrash | 85% |
| 70 | Preempted | Preemption | 95% |
| 65 | NodeNotReady | NodeFailure | 90% |
| 60 | DeadlineExceeded | Timeout | 85% |
| 55 | NetworkPluginError | NetworkFailure | 80% |

**Key Functions**:

| Function | Description |
|----------|-------------|
| `NewAnalyzer()` | Create analyzer with default rules |
| `Analyze()` | Classify failure based on diagnostics |
| `initRules()` | Initialize classification rules |

**Analysis Flow**:

```mermaid
flowchart TD
    A[Diagnostics] --> B[Analyzer.Analyze]
    B --> C{Check Container Reasons}
    C -->|OOMKilled| D[MemoryExhaustion]
    C -->|Evicted| E[NodePressure]
    C -->|ImagePullBackOff| F[ImageFailure]
    C -->|CrashLoopBackOff| G[ApplicationCrash]
    C -->|None| H{Check Event Reasons}
    H -->|FailedScheduling| I[CapacityShortage]
    H -->|Preempted| J[Preemption]
    H -->|NodeNotReady| K[NodeFailure]
    H -->|None| L[Unknown]
    
    D --> M[Generate Recommendations]
    E --> M
    F --> M
    G --> M
    I --> M
    J --> M
    K --> M
    L --> M
```

## internal/redact

Secret redaction for security compliance.

### redactor.go

**Purpose**: Remove sensitive information from logs and output.

**Key Features**:
- Pattern-based secret detection
- Configurable redaction rules
- Support for common secret formats

**Redacted Patterns**:

| Pattern Type | Example |
|--------------|---------|
| AWS Access Keys | `AKIA...` |
| AWS Secret Keys | Base64 encoded secrets |
| Private Keys | `-----BEGIN PRIVATE KEY-----` |
| API Tokens | Bearer tokens |
| Passwords | URL-encoded passwords |
| Connection Strings | Database URLs with credentials |

**Key Functions**:

| Function | Description |
|----------|-------------|
| `NewRedactor()` | Create redactor with default patterns |
| `Redact()` | Remove secrets from text |
| `AddPattern()` | Add custom redaction pattern |

## internal/reporting

Report generation for GitHub Actions Job Summary.

### reporter.go

**Purpose**: Generate formatted markdown reports for GitHub Actions.

**Key Functions**:

| Function | Description |
|----------|-------------|
| `NewReporter()` | Create a new reporter instance |
| `Generate()` | Generate complete markdown report |
| `WriteToSummary()` | Write report to GITHUB_STEP_SUMMARY file |

**Report Structure**:

```markdown
## 🔍 Pod Post-Mortem Report

### Summary
- **Pod**: <name>
- **Namespace**: <namespace>
- **Classification**: <type>
- **Confidence**: <percentage>

### Container Status
| Container | State | Reason | Exit Code |
|----------|-------|--------|-----------|
| ... | ... | ... | ... |

### Events Timeline
- [timestamp] Event: message

### Previous Logs
```
<container logs>
```

### Recommendations
1. <recommendation>
2. <recommendation>
```

## internal/github

GitHub Actions integration.

### client.go

**Purpose**: Handle GitHub Actions specific functionality.

**Key Features**:
- Write to GitHub Step Summary
- Set output variables
- Environment variable access

**Key Functions**:

| Function | Description |
|----------|-------------|
| `NewClient()` | Create GitHub client from environment |
| `WriteSummary()` | Write content to step summary |
| `SetOutput()` | Set workflow output variable |

**Environment Variables Used**:

| Variable | Description |
|----------|-------------|
| `GITHUB_STEP_SUMMARY` | Path to step summary file |
| `GITHUB_OUTPUT` | Path to output file |
| `GITHUB_WORKFLOW` | Workflow name |
| `GITHUB_RUN_ID` | Run identifier |

## Component Interaction

```mermaid
sequenceDiagram
    participant Main
    participant K8sClient
    participant Analyzer
    participant Redactor
    participant Reporter
    participant GitHub
    
    Main->>K8sClient: CollectDiagnostics()
    K8sClient-->>Main: Diagnostics
    
    Main->>Analyzer: Analyze(Diagnostics)
    Analyzer-->>Main: Classification
    
    Main->>Redactor: Redact(Logs)
    Redactor-->>Main: Sanitized Logs
    
    Main->>Reporter: Generate(Diagnostics, Classification)
    Reporter-->>Main: Markdown Report
    
    Main->>GitHub: WriteSummary(Report)
```

## Configuration

All components can be configured via the `types.Config` struct:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `LogTailLines` | int | 100 | Number of log lines to collect |
| `IncludeNodeInfo` | bool | false | Include node information |
| `RedactSecrets` | bool | true | Enable secret redaction |
| `OutputFormat` | string | "markdown" | Output format |
| `Namespace` | string | "" | Target namespace (auto-discovered if empty) |
| `PodName` | string | "" | Target pod (auto-discovered if empty) |
| `Debug` | bool | false | Enable debug logging |
| `Timeout` | Duration | 30s | API timeout |