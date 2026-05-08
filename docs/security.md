# Security

This document outlines the security model, best practices, and considerations for k8s-pod-postmortem.

## Security Architecture

```mermaid
flowchart TB
    subgraph "Authentication"
        A[Service Account Token]
        B[In-Cluster Config]
        C[Kubernetes API]
    end

    subgraph "Authorization - RBAC"
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

## Authentication

### Service Account Authentication

k8s-pod-postmortem uses Kubernetes service account authentication with the following flow:

```mermaid
sequenceDiagram
    participant Pod as Runner Pod
    participant SA as Service Account
    participant K8s as Kubernetes API
    
    Pod->>SA: Mount service account token
    SA-->>Pod: Token at /var/run/secrets/kubernetes.io/serviceaccount/token
    Pod->>K8s: API request with Bearer token
    K8s-->>Pod: Authorized response
```

### In-Cluster Configuration

The application uses in-cluster configuration which automatically:

1. Reads the service account token from `/var/run/secrets/kubernetes.io/serviceaccount/token`
2. Reads the CA certificate from `/var/run/secrets/kubernetes.io/serviceaccount/ca.crt`
3. Discovers the namespace from `/var/run/secrets/kubernetes.io/serviceaccount/namespace`

**Benefits**:
- No static credentials stored in code or configuration
- Automatic token rotation by Kubernetes
- Short-lived tokens with automatic refresh

## Authorization (RBAC)

### Least Privilege Principle

k8s-pod-postmortem follows the principle of least privilege. The default RBAC configuration grants only the minimum permissions required:

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

### Permission Breakdown

| Resource | Verbs | Purpose |
|----------|-------|---------|
| `pods` | get, list, watch | Retrieve pod information and status |
| `pods/log` | get | Collect previous container logs |
| `events` | get, list | Retrieve pod-related events |
| `nodes` | get | Optional: Collect node information (disabled by default) |

### Namespace Scoping

The Role and RoleBinding are namespace-scoped, meaning:

- **No cluster-wide access**: The action can only access resources in its own namespace
- **No secrets access**: Cannot read Kubernetes secrets
- **No write access**: Read-only permissions only

### RBAC Configuration

```mermaid
flowchart LR
    subgraph "Namespace: runner-namespace"
        A[ServiceAccount] --> B[RoleBinding]
        B --> C[Role]
        C --> D[pods - get/list/watch]
        C --> E[pods/log - get]
        C --> F[events - get/list]
    end
```

## Secret Redaction

### Overview

k8s-pod-postmortem includes a built-in secret redaction engine that automatically removes sensitive information from logs and output.

### Redacted Patterns

The following patterns are automatically redacted:

#### Cloud Provider Credentials

| Pattern | Example | Replacement |
|---------|---------|-------------|
| AWS Access Key ID | `AWS_ACCESS_KEY_ID=AKIA...` | `[REDACTED_AWS_ACCESS_KEY_ID]` |
| AWS Secret Access Key | `AWS_SECRET_ACCESS_KEY=...` | `[REDACTED_AWS_SECRET_ACCESS_KEY]` |
| AWS Session Token | `AWS_SESSION_TOKEN=...` | `[REDACTED_AWS_SESSION_TOKEN]` |

#### GitHub Tokens

| Pattern | Example | Replacement |
|---------|---------|-------------|
| GitHub Token | `GITHUB_TOKEN=...` | `[REDACTED_GITHUB_TOKEN]` |
| GitHub PAT | `ghp_xxxx...` | `[REDACTED_GITHUB_PAT]` |
| GitHub OAuth | `gho_xxxx...` | `[REDACTED_GITHUB_OAUTH]` |
| GitHub App Token | `ghs_xxxx...` | `[REDACTED_GITHUB_APP_TOKEN]` |

#### Authentication Headers

| Pattern | Example | Replacement |
|---------|---------|-------------|
| Authorization Header | `Authorization: Bearer ...` | `[REDACTED_AUTH_HEADER]` |
| Bearer Token | `Bearer eyJ...` | `[REDACTED_BEARER_TOKEN]` |

#### Private Keys

| Pattern | Replacement |
|---------|-------------|
| RSA/DSA/EC/OpenSSH Private Keys | `[REDACTED_PRIVATE_KEY]` |

#### Connection Strings

| Pattern | Example | Replacement |
|---------|---------|-------------|
| Database URLs | `postgres://user:pass@host` | `[REDACTED_DATABASE_URL]` |
| Redis URLs | `redis://user:pass@host` | `[REDACTED_DATABASE_URL]` |
| MongoDB URLs | `mongodb://user:pass@host` | `[REDACTED_DATABASE_URL]` |

#### Generic Patterns

| Pattern | Example | Replacement |
|---------|---------|-------------|
| Password in config | `password=secret123` | `password=[REDACTED]` |
| API Key | `api_key=abcd1234` | `api_key=[REDACTED]` |
| Secret in URL | `https://user:pass@host` | `https://[REDACTED]:[REDACTED]@host` |

### Redaction Implementation

```go
// Example of how redaction works
type Redactor struct {
    patterns []*redactionPattern
}

type redactionPattern struct {
    name        string
    regex       *regexp.Regexp
    replacement string
}

func (r *Redactor) Redact(text string) string {
    for _, pattern := range r.patterns {
        text = pattern.regex.ReplaceAllString(text, pattern.replacement)
    }
    return text
}
```

### Enabling/Disabling Redaction

Redaction is enabled by default. To disable:

```yaml
# In workflow.yaml
- uses: bansikah22/k8s-pod-postmortem@v1
  with:
    redact-secrets: 'false'
```

## Data Flow Security

```mermaid
flowchart TD
    subgraph "Data Collection"
        A[Kubernetes API] --> B[Raw Diagnostics]
        B --> C[Raw Logs]
    end

    subgraph "Processing"
        C --> D[Secret Redaction]
        D --> E[Sanitized Data]
    end

    subgraph "Output"
        E --> F[GitHub Step Summary]
        E --> G[Output Variables]
    end

    style D fill:#90EE90
    style E fill:#90EE90
```

## Security Best Practices

### 1. Use Dedicated Service Accounts

Create a dedicated service account for the post-mortem action:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: postmortem-sa
  namespace: actions-runners
```

### 2. Limit Namespace Access

Only grant permissions in the namespace where runners operate:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: postmortem-rb
  namespace: actions-runners  # Restrict to runner namespace
subjects:
- kind: ServiceAccount
  name: postmortem-sa
  namespace: actions-runners
roleRef:
  kind: Role
  name: postmortem-role
  apiGroup: rbac.authorization.k8s.io
```

### 3. Enable Secret Redaction

Always keep secret redaction enabled in production:

```yaml
- uses: bansikah22/k8s-pod-postmortem@v1
  with:
    redact-secrets: 'true'  # Default, but explicit is better
```

### 4. Review Logs Before Sharing

Even with redaction, review the generated reports before sharing externally:

- Verify no sensitive information slipped through
- Check for patterns specific to your organization
- Consider adding custom redaction patterns

### 5. Use Short-Lived Tokens

Kubernetes automatically rotates service account tokens. Ensure your cluster is configured with:

- Token rotation enabled (default in Kubernetes 1.21+)
- Short token validity periods
- Bound service account tokens (Kubernetes 1.22+)

### 6. Audit Access

Enable Kubernetes audit logging to track access:

```yaml
# kube-apiserver configuration
--audit-log-path=/var/log/kubernetes/audit.log
--audit-log-maxage=30
--audit-log-maxbackup=10
--audit-log-maxsize=100
```

## Threat Model

### What We Protect Against

| Threat | Mitigation |
|--------|------------|
| Credential exposure in logs | Secret redaction patterns |
| Unauthorized API access | Namespace-scoped RBAC |
| Privilege escalation | Least privilege permissions |
| Data exfiltration | No write permissions, read-only access |

### What We Don't Protect Against

| Threat | Reason |
|--------|--------|
| Compromised runner environment | Outside the scope of this action |
| Malicious workflow modifications | Use branch protection and required reviews |
| Network interception | Use encrypted Kubernetes API communication |
| Insider threats | Implement proper access controls and audit logs |

## Security Checklist

Before deploying k8s-pod-postmortem:

- [ ] Review RBAC permissions are minimal
- [ ] Verify namespace scoping is correct
- [ ] Enable secret redaction
- [ ] Use dedicated service account
- [ ] Review custom redaction patterns needed
- [ ] Enable Kubernetes audit logging
- [ ] Test with non-sensitive workloads first
- [ ] Review generated reports for sensitive data

## Custom Redaction Patterns

To add custom redaction patterns for organization-specific secrets:

### Example: Custom API Key Pattern

```yaml
# In your workflow
- uses: bansikah22/k8s-pod-postmortem@v1
  with:
    custom-redaction-patterns: |
      MY_API_KEY=[A-Za-z0-9]{32}
      INTERNAL_TOKEN=token_[A-Za-z0-9]+
```

### Pattern Format

Patterns use Go regular expression syntax:

| Element | Meaning |
|---------|---------|
| `(?i)` | Case-insensitive matching |
| `\s*` | Zero or more whitespace |
| `[A-Za-z0-9]+` | One or more alphanumeric characters |
| `['"]?` | Optional quote character |

## Compliance Considerations

### GDPR

- Secret redaction helps prevent PII exposure
- Review logs for any remaining PII
- Implement data retention policies for reports

### SOC 2

- Least privilege access aligns with access control requirements
- Audit logging provides traceability
- Regular security reviews recommended

### HIPAA

- Additional redaction patterns may be needed for PHI
- Review all output before storage
- Implement appropriate access controls

## Reporting Security Issues

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public issue
2. Email security details to the maintainers
3. Allow time for investigation and fix
4. Coordinate disclosure timing

## Security Updates

Security updates are released as:

- Patch versions for critical fixes
- Minor versions for security enhancements
- Documented in CHANGELOG.md
- Announced in GitHub Security Advisories