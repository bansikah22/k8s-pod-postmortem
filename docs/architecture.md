# Architecture

## High-Level Architecture

```mermaid
flowchart TB
    subgraph "GitHub Actions Workflow"
        A[Job Running in Ephemeral Runner Pod]
        B{Job Status}
        A --> B
    end

    subgraph "k8s-pod-postmortem Action"
        C[1. Identify Current Pod]
        D[2. Authenticate via Service Account]
        E[3. Query Kubernetes API]
        F[4. Gather Events & Logs]
        G[5. Analyze Failure Causes]
        H[6. Generate Markdown Report]
        
        C --> D --> E --> F --> G --> H
    end

    subgraph "GitHub Job Summary"
        I[Root Cause Report]
        J[Events Timeline]
        K[Container Exit Reasons]
        L[Previous Logs]
        M[Recommendations]
    end

    B -->|failure() or cancelled()| C
    H --> I
    H --> J
    H --> K
    H --> L
    H --> M
```

## Component Architecture

```mermaid
flowchart LR
    subgraph "cmd/postmortem"
        A[main.go]
    end

    subgraph "internal/"
        B[kubernetes/client.go]
        C[analysis/analyzer.go]
        D[redact/redactor.go]
        E[reporting/reporter.go]
        F[github/client.go]
        G[types/types.go]
    end

    subgraph "External Systems"
        H[Kubernetes API]
        I[GitHub Actions Summary]
    end

    A --> B
    A --> C
    A --> D
    A --> E
    A --> F
    A --> G

    B --> H
    F --> I
```

## Data Flow

```mermaid
sequenceDiagram
    participant GA as GitHub Actions
    participant PM as postmortem binary
    participant K8s as Kubernetes API
    participant GH as GitHub Summary

    GA->>PM: Execute on failure()
    PM->>PM: Parse flags & config
    PM->>PM: Discover namespace & pod
    PM->>K8s: Get pod information
    K8s-->>PM: Pod details
    PM->>K8s: Get events
    K8s-->>PM: Events list
    PM->>K8s: Get previous logs
    K8s-->>PM: Container logs
    PM->>PM: Redact secrets
    PM->>PM: Analyze failure
    PM->>PM: Generate report
    PM->>GH: Write to GITHUB_STEP_SUMMARY
    GH-->>PM: Success
    PM-->>GA: Exit
```

## Internal Package Dependencies

```mermaid
flowchart TD
    subgraph "Entry Point"
        MAIN[cmd/postmortem/main.go]
    end

    subgraph "Core Packages"
        K8S[internal/kubernetes]
        ANALYSIS[internal/analysis]
        REDACT[internal/redact]
        REPORT[internal/reporting]
        GH[internal/github]
        TYPES[internal/types]
    end

    subgraph "External Dependencies"
        K8S_API[k8s.io/client-go]
        CORE_API[k8s.io/api/core/v1]
    end

    MAIN --> K8S
    MAIN --> ANALYSIS
    MAIN --> REDACT
    MAIN --> REPORT
    MAIN --> GH
    MAIN --> TYPES

    K8S --> TYPES
    K8S --> K8S_API
    K8S --> CORE_API

    ANALYSIS --> TYPES
    ANALYSIS --> CORE_API

    REDACT --> TYPES
    REDACT --> CORE_API

    REPORT --> TYPES

    GH --> TYPES
```

## Failure Classification Flow

```mermaid
flowchart TD
    A[Pod Diagnostics] --> B{Container Reason?}
    B -->|OOMKilled| C[MEMORY_EXHAUSTION]
    B -->|Evicted| D[NODE_PRESSURE]
    B -->|CrashLoopBackOff| E[APPLICATION_CRASH]
    B -->|ImagePullBackOff| F[IMAGE_FAILURE]
    B -->|None| G{Event Reason?}

    G -->|FailedScheduling| H[CAPACITY_SHORTAGE]
    G -->|Preempted| I[PREEMPTION]
    G -->|NodeNotReady| J[NODE_FAILURE]
    G -->|DeadlineExceeded| K[TIMEOUT]
    G -->|NetworkPluginError| L[NETWORK_FAILURE]
    G -->|None| M[UNKNOWN]

    C --> N[Generate Recommendations]
    D --> N
    E --> N
    F --> N
    H --> N
    I --> N
    J --> N
    K --> N
    L --> N
    M --> N
```

## Security Architecture

```mermaid
flowchart TD
    subgraph "Authentication"
        A[Service Account Token]
        B[In-Cluster Config]
        C[Kubernetes API]
    end

    subgraph "Authorization"
        D[Role: pods, pods/log, events]
        E[RoleBinding]
        F[ServiceAccount]
    end

    subgraph "Data Protection"
        G[Secret Redaction]
        H[Pattern Matching]
        I[Sanitized Output]
    end

    A --> B --> C
    D --> E --> F
    G --> H --> I
```

## CI/CD Pipeline Architecture

```mermaid
flowchart LR
    subgraph "Development"
        A[Code Changes]
        B[Unit Tests]
        C[Lint]
    end

    subgraph "Build"
        D[Build Binary]
        E[Build Docker Image]
        F[Security Scan]
    end

    subgraph "Release"
        G[Sign Image]
        H[Generate SBOM]
        I[GitHub Release]
    end

    A --> B --> C --> D --> E --> F --> G --> H --> I
```
