// Package reporting provides markdown report generation functionality
package reporting

import (
	"fmt"
	"strings"
	"time"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

// Reporter generates markdown reports for post-mortem diagnostics
type Reporter struct{}

// NewReporter creates a new Reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

// GenerateMarkdown generates a markdown report from diagnostics and classification
func (r *Reporter) GenerateMarkdown(diagnostics *types.Diagnostics, classification types.Classification) string {
	var sb strings.Builder

	r.writeHeader(&sb)
	r.writeSummary(&sb, diagnostics, classification)
	r.writeContainerStatus(&sb, diagnostics)
	r.writeEventsTimeline(&sb, diagnostics)
	r.writePreviousLogs(&sb, diagnostics)
	r.writeRecommendations(&sb, classification)
	r.writeFooter(&sb)

	return sb.String()
}

func (r *Reporter) writeHeader(sb *strings.Builder) {
	sb.WriteString("# Kubernetes Pod Post-Mortem\n\n")
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("---\n\n")
}

func (r *Reporter) writeSummary(sb *strings.Builder, diagnostics *types.Diagnostics, classification types.Classification) {
	sb.WriteString("## Failure Classification\n\n")
	sb.WriteString(fmt.Sprintf("**Type:** `%s`\n\n", classification.Type))
	sb.WriteString(fmt.Sprintf("**Reason:** %s\n\n", classification.Reason))
	sb.WriteString(fmt.Sprintf("**Message:** %s\n\n", classification.Message))
	sb.WriteString(fmt.Sprintf("**Confidence:** %.0f%%\n\n", classification.Confidence*100))
	sb.WriteString("---\n\n")

	sb.WriteString("## Pod Information\n\n")
	sb.WriteString(fmt.Sprintf("**Namespace:** `%s`\n\n", diagnostics.Namespace))
	sb.WriteString(fmt.Sprintf("**Pod:** `%s`\n\n", diagnostics.PodName))

	if diagnostics.Pod != nil {
		sb.WriteString(fmt.Sprintf("**Node:** `%s`\n\n", diagnostics.Pod.Spec.NodeName))
		sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", diagnostics.Pod.CreationTimestamp.Format(time.RFC3339)))
	}
	sb.WriteString("---\n\n")
}

func (r *Reporter) writeContainerStatus(sb *strings.Builder, diagnostics *types.Diagnostics) {
	sb.WriteString("## Container Status\n\n")

	if len(diagnostics.ContainerStats) == 0 {
		sb.WriteString("*No container status information available*\n\n")
		sb.WriteString("---\n\n")
		return
	}

	sb.WriteString("| Container | State | Reason | Exit Code | Restarts |\n")
	sb.WriteString("|-----------|-------|--------|-----------|----------|\n")

	for _, stat := range diagnostics.ContainerStats {
		exitCode := "-"
		if stat.ExitCode != 0 {
			exitCode = fmt.Sprintf("%d", stat.ExitCode)
		}
		reason := stat.Reason
		if reason == "" {
			reason = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
			stat.Name, stat.State, reason, exitCode, stat.RestartCount))
	}
	sb.WriteString("\n---\n\n")
}

func (r *Reporter) writeEventsTimeline(sb *strings.Builder, diagnostics *types.Diagnostics) {
	sb.WriteString("## Events Timeline\n\n")

	if len(diagnostics.Events) == 0 {
		sb.WriteString("*No events found*\n\n")
		sb.WriteString("---\n\n")
		return
	}

	// Sort events by time (most recent first)
	events := diagnostics.Events
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		eventType := "Normal"
		if event.Type == "Warning" || event.Type == "Error" {
			eventType = fmt.Sprintf("**%s**", event.Type)
		}

		timestamp := "unknown"
		if !event.LastTimestamp.IsZero() {
			timestamp = event.LastTimestamp.Format(time.RFC3339)
		}
		sb.WriteString(fmt.Sprintf("- %s `%s` at %s\n",
			eventType, event.Reason, timestamp))
		if event.Message != "" {
			sb.WriteString(fmt.Sprintf("  > %s\n", event.Message))
		}
	}
	sb.WriteString("\n---\n\n")
}

func (r *Reporter) writePreviousLogs(sb *strings.Builder, diagnostics *types.Diagnostics) {
	sb.WriteString("## Previous Container Logs\n\n")

	if len(diagnostics.PreviousLogs) == 0 {
		sb.WriteString("*No previous logs available*\n\n")
		sb.WriteString("---\n\n")
		return
	}

	for container, logs := range diagnostics.PreviousLogs {
		sb.WriteString(fmt.Sprintf("### Container: `%s`\n\n", container))

		if logs == "" {
			sb.WriteString("*No logs captured*\n\n")
			continue
		}

		// Truncate logs if too long
		maxLogLines := 50
		lines := strings.Split(logs, "\n")
		if len(lines) > maxLogLines {
			sb.WriteString(fmt.Sprintf("*Showing last %d lines (truncated)*\n\n", maxLogLines))
			lines = lines[len(lines)-maxLogLines:]
		}

		sb.WriteString("```\n")
		for _, line := range lines {
			sb.WriteString(line + "\n")
		}
		sb.WriteString("```\n\n")
	}
	sb.WriteString("---\n\n")
}

func (r *Reporter) writeRecommendations(sb *strings.Builder, classification types.Classification) {
	sb.WriteString("## Recommendations\n\n")

	if len(classification.Recommendations) == 0 {
		sb.WriteString("*No specific recommendations available*\n\n")
		sb.WriteString("---\n\n")
		return
	}

	for i, rec := range classification.Recommendations {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
	}
	sb.WriteString("\n---\n\n")
}

func (r *Reporter) writeFooter(sb *strings.Builder) {
	sb.WriteString("## Additional Resources\n\n")
	sb.WriteString("- [Kubernetes Troubleshooting Guide](https://kubernetes.io/docs/tasks/debug/)\n")
	sb.WriteString("- [Pod Lifecycle Documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)\n")
	sb.WriteString("- [Container Exit Codes Reference](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-exit-codes)\n\n")
	sb.WriteString("---\n\n")
	sb.WriteString("*This report was generated by k8s-pod-postmortem GitHub Action*\n")
}
