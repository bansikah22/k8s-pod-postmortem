// Package kubernetes provides Kubernetes client functionality for pod diagnostics
package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

// Client handles Kubernetes API interactions
type Client struct {
	clientset kubernetes.Interface
}

// NewClient creates a new Kubernetes client using in-cluster configuration
func NewClient() (*Client, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// getKubernetesConfig returns Kubernetes configuration, trying in-cluster first, then kubeconfig
func getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster configuration first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig for local development
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE") // Windows
	}
	if home == "" {
		return nil, fmt.Errorf("failed to get in-cluster config: %w, and no home directory found for kubeconfig", err)
	}

	kubeconfigPath := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to get in-cluster config: %w, and kubeconfig not found at %s", err, kubeconfigPath)
	}

	// Use KUBECONFIG environment variable if set
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfigPath = kubeconfigEnv
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// DiscoverPodInfo discovers namespace and pod name if not provided
func (c *Client) DiscoverPodInfo(_ context.Context, namespace, podName string) (string, string, error) {
	// Use provided values if both are set
	if namespace != "" && podName != "" {
		return namespace, podName, nil
	}

	// Discover namespace from service account
	if namespace == "" {
		ns, err := c.discoverNamespace()
		if err != nil {
			return "", "", fmt.Errorf("failed to discover namespace: %w", err)
		}
		namespace = ns
	}

	// Discover pod name from hostname
	if podName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return "", "", fmt.Errorf("failed to get hostname: %w", err)
		}
		podName = hostname
	}

	return namespace, podName, nil
}

// discoverNamespace reads the namespace from the service account namespace file
func (c *Client) discoverNamespace() (string, error) {
	namespaceFile := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	data, err := os.ReadFile(namespaceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read namespace file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// CollectDiagnostics gathers all diagnostic information for the pod
func (c *Client) CollectDiagnostics(ctx context.Context, namespace, podName string, logTailLines int, includeNodeInfo bool) (*types.Diagnostics, error) {
	diagnostics := &types.Diagnostics{
		Namespace:    namespace,
		PodName:      podName,
		PreviousLogs: make(map[string]string),
	}

	// Get pod information
	pod, err := c.GetPod(ctx, namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}
	diagnostics.Pod = pod

	// Extract container stats
	diagnostics.ContainerStats = extractContainerStats(pod)

	// Get events
	events, err := c.GetEvents(ctx, namespace, podName)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	diagnostics.Events = events

	// Get previous logs for each container
	for _, container := range pod.Spec.Containers {
		logs, err := c.GetPreviousLogs(ctx, namespace, podName, container.Name, int64(logTailLines))
		if err != nil {
			// Log error but continue - previous logs may not exist
			diagnostics.PreviousLogs[container.Name] = fmt.Sprintf("Error retrieving logs: %v", err)
		} else {
			diagnostics.PreviousLogs[container.Name] = logs
		}
	}

	// Get node information if requested
	if includeNodeInfo && pod.Spec.NodeName != "" {
		node, err := c.GetNode(ctx, pod.Spec.NodeName)
		if err != nil {
			return nil, fmt.Errorf("failed to get node: %w", err)
		}
		diagnostics.Node = node
	}

	return diagnostics, nil
}

// GetPod retrieves pod information
func (c *Client) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// GetEvents retrieves events for the pod
func (c *Client) GetEvents(ctx context.Context, namespace, podName string) ([]corev1.Event, error) {
	events, err := c.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", podName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	return events.Items, nil
}

// GetPreviousLogs retrieves logs from the previous container instance
func (c *Client) GetPreviousLogs(ctx context.Context, namespace, podName, containerName string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{
		Container:  containerName,
		Previous:   true,
		TailLines:  &tailLines,
		Timestamps: true,
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()

	data := make([]byte, 0, 4096)
	buf := make([]byte, 1024)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if err != io.EOF {
				return "", fmt.Errorf("error reading log stream: %w", err)
			}
			break
		}
	}

	return string(data), nil
}

// GetNode retrieves node information
func (c *Client) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	node, err := c.clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", name, err)
	}
	return node, nil
}

// extractContainerStats extracts container status information from pod
func extractContainerStats(pod *corev1.Pod) []types.ContainerStats {
	var stats []types.ContainerStats

	for _, cs := range pod.Status.ContainerStatuses {
		stat := types.ContainerStats{
			Name:         cs.Name,
			RestartCount: cs.RestartCount,
		}

		// Current state
		switch {
		case cs.State.Running != nil:
			stat.State = "Running"
		case cs.State.Waiting != nil:
			stat.State = "Waiting"
			stat.Reason = cs.State.Waiting.Reason
		case cs.State.Terminated != nil:
			stat.State = "Terminated"
			stat.Reason = cs.State.Terminated.Reason
			stat.ExitCode = cs.State.Terminated.ExitCode
		}

		// Last state
		switch {
		case cs.LastTerminationState.Running != nil:
			stat.LastState = "Running"
		case cs.LastTerminationState.Waiting != nil:
			stat.LastState = "Waiting"
		case cs.LastTerminationState.Terminated != nil:
			stat.LastState = "Terminated"
			// Use last termination info for better diagnostics
			if stat.State != "Terminated" {
				stat.Reason = cs.LastTerminationState.Terminated.Reason
				stat.ExitCode = cs.LastTerminationState.Terminated.ExitCode
			}
		}

		stats = append(stats, stat)
	}

	return stats
}
