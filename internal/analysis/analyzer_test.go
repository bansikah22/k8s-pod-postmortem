package analysis

import (
	"testing"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
	corev1 "k8s.io/api/core/v1"
)

func TestAnalyzer_Analyze_OOMKilled(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		ContainerStats: []types.ContainerStats{
			{
				Name:   "runner",
				State:  "Terminated",
				Reason: "OOMKilled",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.MemoryExhaustion {
		t.Errorf("expected MemoryExhaustion, got %s", classification.Type)
	}
	if classification.Reason != "OOMKilled" {
		t.Errorf("expected OOMKilled, got %s", classification.Reason)
	}
	if classification.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", classification.Confidence)
	}
}

func TestAnalyzer_Analyze_Evicted(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		ContainerStats: []types.ContainerStats{
			{
				Name:   "runner",
				State:  "Terminated",
				Reason: "Evicted",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.NodePressure {
		t.Errorf("expected NodePressure, got %s", classification.Type)
	}
}

func TestAnalyzer_Analyze_CrashLoopBackOff(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		ContainerStats: []types.ContainerStats{
			{
				Name:   "runner",
				State:  "Waiting",
				Reason: "CrashLoopBackOff",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.ApplicationCrash {
		t.Errorf("expected ApplicationCrash, got %s", classification.Type)
	}
}

func TestAnalyzer_Analyze_ImagePullBackOff(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		ContainerStats: []types.ContainerStats{
			{
				Name:   "runner",
				State:  "Waiting",
				Reason: "ImagePullBackOff",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.ImageFailure {
		t.Errorf("expected ImageFailure, got %s", classification.Type)
	}
}

func TestAnalyzer_Analyze_FailedScheduling(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		Events: []corev1.Event{
			{
				Reason: "FailedScheduling",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.CapacityShortage {
		t.Errorf("expected CapacityShortage, got %s", classification.Type)
	}
}

func TestAnalyzer_Analyze_Unknown(t *testing.T) {
	analyzer := NewAnalyzer()
	diagnostics := &types.Diagnostics{
		ContainerStats: []types.ContainerStats{
			{
				Name:  "runner",
				State: "Running",
			},
		},
	}

	classification := analyzer.Analyze(diagnostics)

	if classification.Type != types.Unknown {
		t.Errorf("expected Unknown, got %s", classification.Type)
	}
	if classification.Confidence != 0.0 {
		t.Errorf("expected confidence 0.0, got %f", classification.Confidence)
	}
}

func TestHasContainerReason(t *testing.T) {
	tests := []struct {
		name     string
		stats    []types.ContainerStats
		reason   string
		expected bool
	}{
		{
			name: "found reason",
			stats: []types.ContainerStats{
				{Reason: "OOMKilled"},
			},
			reason:   "OOMKilled",
			expected: true,
		},
		{
			name: "not found reason",
			stats: []types.ContainerStats{
				{Reason: "CrashLoopBackOff"},
			},
			reason:   "OOMKilled",
			expected: false,
		},
		{
			name:     "empty stats",
			stats:    []types.ContainerStats{},
			reason:   "OOMKilled",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &types.Diagnostics{ContainerStats: tt.stats}
			result := hasContainerReason(d, tt.reason)
			if result != tt.expected {
				t.Errorf("hasContainerReason() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasEventReason(t *testing.T) {
	tests := []struct {
		name     string
		events   []corev1.Event
		reason   string
		expected bool
	}{
		{
			name: "found event reason",
			events: []corev1.Event{
				{Reason: "FailedScheduling"},
			},
			reason:   "FailedScheduling",
			expected: true,
		},
		{
			name: "not found event reason",
			events: []corev1.Event{
				{Reason: "Started"},
			},
			reason:   "FailedScheduling",
			expected: false,
		},
		{
			name:     "empty events",
			events:   []corev1.Event{},
			reason:   "FailedScheduling",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &types.Diagnostics{Events: tt.events}
			result := hasEventReason(d, tt.reason)
			if result != tt.expected {
				t.Errorf("hasEventReason() = %v, want %v", result, tt.expected)
			}
		})
	}
}