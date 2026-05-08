# Deployment

This document provides detailed instructions for deploying k8s-pod-postmortem in various environments.

## Prerequisites

- Kubernetes 1.20+
- Helm 3.x (for Helm deployment)
- kubectl configured with cluster access
- GitHub Actions self-hosted runners running in Kubernetes

## Deployment Methods

### Method 1: Helm Chart (Recommended)

The Helm chart is the recommended deployment method as it handles RBAC, ServiceAccount, and configuration automatically.

#### Installing the Helm Chart

```bash
# Add the Helm repository (if published)
helm repo add k8s-pod-postmortem https://bansikah22.github.io/k8s-pod-postmortem

# Update repository
helm repo update

# Install the chart
helm install k8s-pod-postmortem k8s-pod-postmortem/k8s-pod-postmortem \
  --namespace actions-runners \
  --create-namespace
```

#### Installing from Source

```bash
# Clone the repository
git clone https://github.com/bansikah22/k8s-pod-postmortem.git
cd k8s-pod-postmortem

# Install from local chart
helm install k8s-pod-postmortem ./charts/k8s-pod-postmortem \
  --namespace actions-runners \
  --create-namespace
```

#### Custom Values

Create a custom values file `values-custom.yaml`:

```yaml
# Custom values for k8s-pod-postmortem
rbac:
  create: true
  nameOverride: ""

serviceAccount:
  create: true
  name: "postmortem-sa"
  annotations:
    # Add annotations for external IAM roles (AWS IRSA, GKE Workload Identity, etc.)
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/PostMortemRole"

image:
  repository: ghcr.io/bansikah22/k8s-pod-postmortem
  tag: "v1.0.0"
  pullPolicy: IfNotPresent

# Security context
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true

# Resource limits
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

# Configuration
config:
  logTailLines: 200
  includeNodeInfo: false
  redactSecrets: true
  outputFormat: markdown
  timeout: 10s
```

Install with custom values:

```bash
helm install k8s-pod-postmortem ./charts/k8s-pod-postmortem \
  --namespace actions-runners \
  -f values-custom.yaml
```

### Method 2: Kubernetes Manifests

For environments without Helm, use raw Kubernetes manifests.

#### ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: k8s-pod-postmortem
  namespace: actions-runners
```

#### RBAC

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: k8s-pod-postmortem
  namespace: actions-runners
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
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: k8s-pod-postmortem
  namespace: actions-runners
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: k8s-pod-postmortem
subjects:
- kind: ServiceAccount
  name: k8s-pod-postmortem
  namespace: actions-runners
```

Apply manifests:

```bash
kubectl apply -f serviceaccount.yaml
kubectl apply -f rbac.yaml
```

### Method 3: GitHub Action (Automatic)

The action can be used directly in your workflow without pre-deployment. The necessary RBAC resources must exist in the cluster.

```yaml
# .github/workflows/ci.yaml
jobs:
  build:
    runs-on: [self-hosted, kubernetes]
    steps:
      - uses: actions/checkout@v4
      
      - name: Build
        run: make build
      
      - name: Post-mortem on failure
        if: failure()
        uses: bansikah22/k8s-pod-postmortem@v1
```

## Configuration Reference

### Helm Values

| Value | Description | Default |
|-------|-------------|---------|
| `rbac.create` | Create RBAC resources | `true` |
| `rbac.nameOverride` | Override RBAC resource names | `""` |
| `serviceAccount.create` | Create ServiceAccount | `true` |
| `serviceAccount.name` | ServiceAccount name | `""` (auto-generated) |
| `serviceAccount.annotations` | ServiceAccount annotations | `{}` |
| `image.repository` | Container image repository | `ghcr.io/bansikah22/k8s-pod-postmortem` |
| `image.tag` | Container image tag | `""` (chart appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext` | Pod security context | See values.yaml |
| `securityContext` | Container security context | See values.yaml |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Pod tolerations | `[]` |
| `affinity` | Pod affinity | `{}` |
| `config.logTailLines` | Number of log lines to collect | `200` |
| `config.includeNodeInfo` | Include node information | `false` |
| `config.redactSecrets` | Enable secret redaction | `true` |
| `config.outputFormat` | Output format | `markdown` |
| `config.timeout` | API timeout | `10s` |

### Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `namespace` | Target namespace | No | Auto-discovered |
| `pod-name` | Target pod name | No | Auto-discovered |
| `log-tail-lines` | Number of log lines | No | `200` |
| `include-node-info` | Include node information | No | `false` |
| `redact-secrets` | Enable secret redaction | No | `true` |
| `output-format` | Output format | No | `markdown` |
| `timeout` | API timeout duration | No | `10s` |

## Environment-Specific Deployments

### AWS EKS

For EKS with IAM Roles for Service Accounts (IRSA):

```yaml
# values-eks.yaml
serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/PostMortemRole"

# Optional: Use IRSA for additional permissions
# Note: The action only needs Kubernetes API access, not AWS API access
```

### Google GKE

For GKE with Workload Identity:

```yaml
# values-gke.yaml
serviceAccount:
  create: true
  annotations:
    iam.gke.io/gcp-service-account: "postmortem@project-id.iam.gserviceaccount.com"
```

### Azure AKS

For AKS with Azure AD Workload Identity:

```yaml
# values-aks.yaml
serviceAccount:
  create: true
  annotations:
    azure.workload.identity/client-id: "client-id"
    azure.workload.identity/tenant-id: "tenant-id"
```

### OpenShift

For OpenShift with SCC:

```yaml
# values-openshift.yaml
podSecurityContext:
  runAsNonRoot: true
  # OpenShift assigns random UIDs
  # runAsUser: 1000
  # runAsGroup: 1000

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true

# Add SCC if needed
# oc adm policy add-scc-to-user nonroot -z k8s-pod-postmortem -n actions-runners
```

## Deployment Verification

### Verify RBAC

```bash
# Check Role
kubectl get role k8s-pod-postmortem -n actions-runners -o yaml

# Check RoleBinding
kubectl get rolebinding k8s-pod-postmortem -n actions-runners -o yaml

# Check ServiceAccount
kubectl get serviceaccount k8s-pod-postmortem -n actions-runners -o yaml
```

### Test Permissions

```bash
# Test if ServiceAccount can get pods
kubectl auth can-i get pods -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem

# Test if ServiceAccount can get logs
kubectl auth can-i get pods/log -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem

# Test if ServiceAccount can get events
kubectl auth can-i get events -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem
```

### Verify with Test Pod

```bash
# Create a test pod
kubectl run test-pod --image=busybox --restart=Never -n actions-runners -- sleep 60

# Run the action manually
kubectl run postmortem-test \
  --image=ghcr.io/bansikah22/k8s-pod-postmortem:v1.0.0 \
  --serviceaccount=k8s-pod-postmortem \
  -n actions-runners \
  --restart=Never \
  -- ./postmortem --namespace=actions-runners --pod-name=test-pod

# Check logs
kubectl logs postmortem-test -n actions-runners
```

## Upgrading

### Helm Upgrade

```bash
# Update repository
helm repo update

# Upgrade release
helm upgrade k8s-pod-postmortem k8s-pod-postmortem/k8s-pod-postmortem \
  --namespace actions-runners \
  --values values-custom.yaml
```

### Rollback

```bash
# List releases
helm history k8s-pod-postmortem -n actions-runners

# Rollback to previous version
helm rollback k8s-pod-postmortem -n actions-runners

# Rollback to specific revision
helm rollback k8s-pod-postmortem 2 -n actions-runners
```

## Uninstallation

### Helm Uninstall

```bash
helm uninstall k8s-pod-postmortem -n actions-runners
```

### Manual Cleanup

```bash
# Delete RoleBinding
kubectl delete rolebinding k8s-pod-postmortem -n actions-runners

# Delete Role
kubectl delete role k8s-pod-postmortem -n actions-runners

# Delete ServiceAccount
kubectl delete serviceaccount k8s-pod-postmortem -n actions-runners
```

## High Availability Considerations

k8s-pod-postmortem runs as an action step, not as a long-running service. However, consider:

### Runner Availability

- Use multiple runner replicas for high availability
- Configure runner autoscaling for burst workloads
- Use pod disruption budgets for runner deployments

### Resource Planning

```yaml
# Runner deployment with postmortem support
apiVersion: apps/v1
kind: Deployment
metadata:
  name: actions-runner
  namespace: actions-runners
spec:
  replicas: 3
  template:
    spec:
      serviceAccountName: k8s-pod-postmortem
      containers:
      - name: runner
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

## Monitoring and Observability

### Metrics

The action doesn't expose metrics directly, but you can monitor:

- Runner pod status
- Action execution time via GitHub Actions logs
- Kubernetes events for the runner namespace

### Logging

View action logs in GitHub Actions UI:

1. Navigate to the workflow run
2. Click on the failed job
3. Expand the "Post-mortem on failure" step
4. View the generated report

### Debugging

Enable debug output:

```yaml
- name: Post-mortem on failure
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  env:
    ACTIONS_STEP_DEBUG: true
    ACTIONS_RUNNER_DEBUG: true
```

## Security Hardening

### Network Policies

Restrict network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: k8s-pod-postmortem
  namespace: actions-runners
spec:
  podSelector:
    matchLabels:
      app: actions-runner
  policyTypes:
  - Egress
  egress:
  # Allow Kubernetes API access
  - to:
    - ipBlock:
        cidr: 10.0.0.0/8
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 6443
  # Allow GitHub API access
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
    ports:
    - protocol: TCP
      port: 443
```

### Pod Security Standards

Ensure compliance with Pod Security Standards:

```yaml
# Enforce restricted policy
apiVersion: v1
kind: Namespace
metadata:
  name: actions-runners
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
```

## Troubleshooting

See [Troubleshooting Guide](./troubleshooting.md) for common issues and solutions.