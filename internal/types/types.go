// Package types provides shared types for the k8s-pod-postmortem action
package types

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Diagnostics contains all collected diagnostic information
type Diagnostics struct {
	Namespace      string
	PodName        string
	Pod            *corev1.Pod
	Events         []corev1.Event
	PreviousLogs   map[string]string
	Node           *corev1.Node
	ContainerStats []ContainerStats
}

// ContainerStats holds container status information
type ContainerStats struct {
	Name         string
	State        string
	Reason       string
	ExitCode     int32
	RestartCount int32
	LastState    string
}

// Event represents a simplified Kubernetes event
type Event struct {
	Type    string
	Reason  string
	Message string
	Source  string
	Count   int32
	Time    time.Time
}

// Classification represents the result of failure analysis
type Classification struct {
	Type            FailureType
	Reason          string
	Message         string
	Recommendations []string
	Confidence      float64
}

// FailureType represents the classification of a failure
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

// Config holds the action configuration
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