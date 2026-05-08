# Troubleshooting

This document provides solutions to common issues when using k8s-pod-postmortem.

## Common Issues

### Permission Denied Errors

#### Symptoms

```
Error: failed to get pod: pods "runner-abc123" is forbidden: User "system:serviceaccount:actions-runners:default" cannot get resource "pods" in API group "" in the namespace "actions-runners"
```

#### Cause

The service account does not have the required RBAC permissions.

#### Solution

Ensure the RBAC resources are created:

```bash
# Check if Role exists
kubectl get role k8s-pod-postmortem -n actions-runners

# Check if RoleBinding exists
kubectl get rolebinding k8s-pod-postmortem -n actions-runners

# Check if ServiceAccount exists
kubectl get serviceaccount k8s-pod-postmortem -n actions-runners
```

If missing, install the Helm chart or apply the RBAC manifests:

```bash
# Using Helm
helm install k8s-pod-postmortem ./charts/k8s-pod-postmortem --namespace actions-runners

# Or apply manifests directly
kubectl apply -f charts/k8s-pod-postmortem/templates/rbac.yaml -n actions-runners
kubectl apply -f charts/k8s-pod-postmortem/templates/serviceaccount.yaml -n actions-runners
```

---

### Pod Not Found

#### Symptoms

```
Error: failed to get pod: pods "runner-abc123" not found
```

#### Cause

The pod name or namespace is incorrect, or the pod has already been deleted.

#### Solution

1. **Verify namespace auto-detection:**

```bash
# Check the namespace file
kubectl exec -n actions-runners runner-abc123 -- cat /var/run/secrets/kubernetes.io/serviceaccount/namespace
```

2. **Verify pod exists:**

```bash
kubectl get pods -n actions-runners
```

3. **Specify namespace and pod name explicitly:**

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    namespace: 'actions-runners'
    pod-name: 'runner-abc123'
```

---

### No Previous Logs Available

#### Symptoms

```
Error retrieving logs: container "runner" in pod "runner-abc123" is waiting to start
```

Or:

```
Previous logs: *No previous logs available*
```

#### Cause

The container has not restarted yet, so there are no previous logs to retrieve.

#### Solution

This is expected behavior for first-time failures. The action will still collect:
- Pod status
- Container states
- Events
- Current state information

If you need logs from the current container, consider:

1. **Increasing log tail lines:**

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    log-tail-lines: '500'
```

2. **Checking container restart count:**

The report shows restart count. If it's 0, there are no previous logs.

---

### Timeout Errors

#### Symptoms

```
Error: context deadline exceeded
```

Or:

```
Error: failed to get events: the server was unable to return a response in time
```

#### Cause

The Kubernetes API is slow to respond or the timeout is too short.

#### Solution

Increase the timeout:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    timeout: '30s'
```

---

### Service Account Token Issues

#### Symptoms

```
Error: failed to get in-cluster config: open /var/run/secrets/kubernetes.io/serviceaccount/token: no such file or directory
```

#### Cause

The service account token is not mounted in the container.

#### Solution

1. **Verify the runner pod has a service account:**

```bash
kubectl get pod runner-abc123 -n actions-runners -o jsonpath='{.spec.serviceAccountName}'
```

2. **Check if token is mounted:**

```bash
kubectl exec -n actions-runners runner-abc123 -- ls /var/run/secrets/kubernetes.io/serviceaccount/
```

3. **Ensure automountServiceAccountToken is not disabled:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: runner
spec:
  automountServiceAccountToken: true  # Must be true or not set
  serviceAccountName: runner-sa
```

---

### Namespace Discovery Failure

#### Symptoms

```
Error: failed to discover namespace: open /var/run/secrets/kubernetes.io/serviceaccount/namespace: no such file or directory
```

#### Cause

The namespace file is not available in the container.

#### Solution

Specify the namespace explicitly:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    namespace: 'actions-runners'
```

---

### Image Pull Failures

#### Symptoms

```
Error: failed to pull image "ghcr.io/bansikah22/k8s-pod-postmortem:v1": image pull failed
```

#### Cause

The container image cannot be pulled due to network issues or authentication.

#### Solution

1. **Check image exists:**

```bash
docker pull ghcr.io/bansikah22/k8s-pod-postmortem:v1
```

2. **Use a specific version:**

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1.0.0
```

3. **Configure image pull secrets:**

```yaml
# In values.yaml
imagePullSecrets:
  - name: ghcr-credentials
```

---

### Empty Report

#### Symptoms

The report shows minimal or no information.

#### Cause

The pod may have just started, or there are no events/logs to collect.

#### Solution

1. **Enable debug mode:**

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    debug: 'true'
```

2. **Check pod status manually:**

```bash
kubectl describe pod runner-abc123 -n actions-runners
kubectl logs runner-abc123 -n actions-runners --previous
kubectl get events -n actions-runners --field-selector involvedObject.name=runner-abc123
```

---

### RBAC Role Not Found

#### Symptoms

```
Error: roles.rbac.authorization.k8s.io "k8s-pod-postmortem" not found
```

#### Cause

The RBAC resources were not created or were deleted.

#### Solution

Re-create the RBAC resources:

```bash
# Using Helm
helm upgrade --install k8s-pod-postmortem ./charts/k8s-pod-postmortem --namespace actions-runners

# Or apply directly
kubectl apply -f - <<EOF
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
EOF
```

---

## Debugging

### Enable Debug Logging

Enable debug mode for verbose output:

```yaml
- name: Post-mortem with debug
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    debug: 'true'
```

### Manual Testing

Test the action manually:

```bash
# Run in a test pod
kubectl run test-postmortem \
  --image=ghcr.io/bansikah22/k8s-pod-postmortem:v1 \
  --serviceaccount=k8s-pod-postmortem \
  -n actions-runners \
  --restart=Never \
  -- ./postmortem --namespace=actions-runners --pod-name=test-pod --debug

# Check logs
kubectl logs test-postmortem -n actions-runners
```

### Check Kubernetes API Access

Verify the service account can access required resources:

```bash
# Test pod access
kubectl auth can-i get pods -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem

# Test logs access
kubectl auth can-i get pods/log -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem

# Test events access
kubectl auth can-i get events -n actions-runners --as=system:serviceaccount:actions-runners:k8s-pod-postmortem
```

### Check Runner Pod Configuration

Verify the runner pod configuration:

```bash
# Get pod details
kubectl get pod runner-abc123 -n actions-runners -o yaml

# Check service account
kubectl get pod runner-abc123 -n actions-runners -o jsonpath='{.spec.serviceAccountName}'

# Check mounted volumes
kubectl get pod runner-abc123 -n actions-runners -o jsonpath='{.spec.volumes}'
```

---

## Known Limitations

### 1. Namespace Scoping

The action can only access pods in the same namespace as the runner. If you need cross-namespace access:

- Deploy runners in each namespace
- Or use a cluster-wide Role (not recommended for security)

### 2. Ephemeral Pods

If the runner pod is deleted before the action runs, diagnostics cannot be collected.

**Workaround:** Configure runner pod termination grace period:

```yaml
spec:
  terminationGracePeriodSeconds: 300  # 5 minutes
```

### 3. Multi-Container Pods

Logs are collected from all containers, but only previous instance logs are available.

**Workaround:** Increase log tail lines for more context:

```yaml
with:
  log-tail-lines: '500'
```

### 4. Large Log Volumes

Very large logs may be truncated.

**Workaround:** Adjust log tail lines based on your needs.

---

## Frequently Asked Questions

### Q: Can I use this with GitHub-hosted runners?

**A:** No, this action requires self-hosted Kubernetes runners because it needs direct access to the Kubernetes API.

### Q: Can I collect diagnostics from other namespaces?

**A:** No, the action is limited to the namespace where the runner is deployed. This is a security feature.

### Q: How do I add custom redaction patterns?

**A:** Currently, custom patterns are not supported. File an issue if you need this feature.

### Q: Why is my report empty?

**A:** Check:
1. Pod exists and has events
2. Container has restarted (for previous logs)
3. RBAC permissions are correct
4. Enable debug mode for more information

### Q: Can I use this outside of GitHub Actions?

**A:** Yes, you can run the binary directly:

```bash
./postmortem --namespace=default --pod-name=my-pod --output=markdown
```

### Q: How do I get node information?

**A:** Enable node info collection:

```yaml
- name: Post-mortem
  if: failure()
  uses: bansikah22/k8s-pod-postmortem@v1
  with:
    include-node-info: 'true'
```

Note: This requires additional RBAC permissions for nodes.

---

## Getting Help

### Check Logs

1. **GitHub Actions logs:** View the step output in the workflow run
2. **Runner logs:** Check runner pod logs
3. **Kubernetes events:** `kubectl get events -n actions-runners`

### File an Issue

If you encounter a bug, please file an issue with:

1. GitHub Actions workflow YAML
2. Error message
3. Runner pod description
4. Kubernetes version
5. Steps to reproduce

### Community Support

- GitHub Issues: https://github.com/bansikah22/k8s-pod-postmortem/issues
- Discussions: https://github.com/bansikah22/k8s-pod-postmortem/discussions

---

## Error Reference

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to get in-cluster config` | Not running in Kubernetes | Use self-hosted Kubernetes runners |
| `pods "X" is forbidden` | RBAC permissions missing | Create Role and RoleBinding |
| `pods "X" not found` | Pod doesn't exist | Check namespace and pod name |
| `context deadline exceeded` | API timeout | Increase timeout setting |
| `no such file or directory` | Service account not mounted | Check automountServiceAccountToken |
| `container is waiting to start` | Container not running | Check container status |
| `image pull failed` | Image not accessible | Check image exists and credentials |