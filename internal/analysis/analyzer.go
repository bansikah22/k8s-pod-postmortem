// Package analysis provides failure classification and analysis functionality
package analysis

import (
	"strings"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

// Analyzer performs failure classification
type Analyzer struct {
	rules []classificationRule
}

// classificationRule defines a rule for classifying failures
type classificationRule struct {
	matchFunc func(*types.Diagnostics) bool
	classify  func(*types.Diagnostics) types.Classification
	priority  int
}

// NewAnalyzer creates a new Analyzer with default rules
func NewAnalyzer() *Analyzer {
	a := &Analyzer{}
	a.initRules()
	return a
}

// initRules initializes the classification rules
func (a *Analyzer) initRules() {
	a.rules = []classificationRule{
		{
			priority: 100,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasContainerReason(d, "OOMKilled")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.MemoryExhaustion,
					Reason:     "OOMKilled",
					Message:    "Container was terminated due to out-of-memory condition",
					Confidence: 0.95,
					Recommendations: []string{
						"Increase memory limits for the container",
						"Review application memory usage patterns",
						"Consider implementing vertical pod autoscaling",
						"Add memory profiling to identify leaks",
					},
				}
			},
		},
		{
			priority: 90,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasContainerReason(d, "Evicted")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.NodePressure,
					Reason:     "Evicted",
					Message:    "Pod was evicted due to node resource pressure",
					Confidence: 0.95,
					Recommendations: []string{
						"Check node resource availability",
						"Review pod resource requests and limits",
						"Consider adding node autoscaling",
						"Review pod priority classes",
					},
				}
			},
		},
		{
			priority: 85,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasEventReason(d, "FailedScheduling")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.CapacityShortage,
					Reason:     "FailedScheduling",
					Message:    "Pod could not be scheduled due to resource constraints",
					Confidence: 0.90,
					Recommendations: []string{
						"Review resource requests and limits",
						"Check node capacity and availability",
						"Consider adding more nodes",
						"Review pod anti-affinity rules",
						"Check for node taints and tolerations",
					},
				}
			},
		},
		{
			priority: 80,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasContainerReason(d, "ImagePullBackOff") || hasContainerReason(d, "ErrImagePull")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.ImageFailure,
					Reason:     "ImagePullBackOff",
					Message:    "Container image could not be pulled",
					Confidence: 0.90,
					Recommendations: []string{
						"Verify image name and tag are correct",
						"Check image registry credentials",
						"Ensure image exists in the registry",
						"Check network connectivity to registry",
						"Verify image pull secrets are configured",
					},
				}
			},
		},
		{
			priority: 75,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasContainerReason(d, "CrashLoopBackOff")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.ApplicationCrash,
					Reason:     "CrashLoopBackOff",
					Message:    "Container is repeatedly crashing",
					Confidence: 0.90,
					Recommendations: []string{
						"Review container logs for application errors",
						"Check application health check configuration",
						"Verify application startup sequence",
						"Review resource limits (CPU/memory)",
						"Check for missing configuration or secrets",
					},
				}
			},
		},
		{
			priority: 70,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasEventReason(d, "Preempted")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.Preemption,
					Reason:     "Preempted",
					Message:    "Pod was preempted by higher priority workload",
					Confidence: 0.95,
					Recommendations: []string{
						"Review pod priority classes",
						"Consider using spot instance tolerations",
						"Implement graceful shutdown handling",
						"Review cluster capacity planning",
					},
				}
			},
		},
		{
			priority: 65,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasEventReason(d, "NodeNotReady") || hasEventReason(d, "NodeLost")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.NodeFailure,
					Reason:     "NodeNotReady",
					Message:    "Node hosting the pod became unavailable",
					Confidence: 0.90,
					Recommendations: []string{
						"Check node health status",
						"Review node conditions",
						"Check for hardware issues",
						"Review cloud provider node status",
						"Consider pod disruption budgets",
					},
				}
			},
		},
		{
			priority: 60,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasEventReason(d, "DeadlineExceeded")
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.Timeout,
					Reason:     "DeadlineExceeded",
					Message:    "Pod startup or operation exceeded deadline",
					Confidence: 0.85,
					Recommendations: []string{
						"Increase activeDeadlineSeconds if applicable",
						"Review application startup time",
						"Check for slow network dependencies",
						"Consider optimizing container startup",
					},
				}
			},
		},
		{
			priority: 55,
			matchFunc: func(d *types.Diagnostics) bool {
				return hasNetworkFailure(d)
			},
			classify: func(_ *types.Diagnostics) types.Classification {
				return types.Classification{
					Type:       types.NetworkFailure,
					Reason:     "NetworkPluginError",
					Message:    "Container network interface failed",
					Confidence: 0.80,
					Recommendations: []string{
						"Check CNI plugin status",
						"Review network policies",
						"Verify pod network configuration",
						"Check for IP address exhaustion",
					},
				}
			},
		},
	}
}

// Analyze performs failure classification on diagnostics
func (a *Analyzer) Analyze(diagnostics *types.Diagnostics) types.Classification {
	// Sort rules by priority (already sorted in initRules)
	for _, rule := range a.rules {
		if rule.matchFunc(diagnostics) {
			return rule.classify(diagnostics)
		}
	}

	// Default classification
	return types.Classification{
		Type:       types.Unknown,
		Reason:     "Unknown",
		Message:    "Unable to determine root cause from available diagnostics",
		Confidence: 0.0,
		Recommendations: []string{
			"Review pod events and logs manually",
			"Check application-specific error handling",
			"Review recent changes to the deployment",
		},
	}
}

// hasContainerReason checks if any container has the specified reason
func hasContainerReason(d *types.Diagnostics, reason string) bool {
	for _, stat := range d.ContainerStats {
		if strings.EqualFold(stat.Reason, reason) {
			return true
		}
	}
	return false
}

// hasEventReason checks if any event has the specified reason
func hasEventReason(d *types.Diagnostics, reason string) bool {
	for _, event := range d.Events {
		if strings.EqualFold(event.Reason, reason) {
			return true
		}
	}
	return false
}

// hasNetworkFailure checks for network-related failures
func hasNetworkFailure(d *types.Diagnostics) bool {
	networkReasons := []string{
		"NetworkPluginError",
		"NetworkNotReady",
		"CNIError",
	}

	for _, event := range d.Events {
		for _, reason := range networkReasons {
			if strings.EqualFold(event.Reason, reason) {
				return true
			}
		}
	}

	// Check for network-related messages
	for _, event := range d.Events {
		msg := strings.ToLower(event.Message)
		if strings.Contains(msg, "network") ||
			strings.Contains(msg, "cni") ||
			strings.Contains(msg, "ip address") {
			return true
		}
	}

	return false
}
