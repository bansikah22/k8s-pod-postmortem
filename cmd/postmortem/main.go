// Package main is the entry point for the k8s-pod-postmortem action
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bansikah22/k8s-pod-postmortem/internal/analysis"
	"github.com/bansikah22/k8s-pod-postmortem/internal/github"
	"github.com/bansikah22/k8s-pod-postmortem/internal/kubernetes"
	"github.com/bansikah22/k8s-pod-postmortem/internal/redact"
	"github.com/bansikah22/k8s-pod-postmortem/internal/reporting"
	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

func main() {
	ctx, cancel := signalAwareContext(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("Action failed", "error", err)
		os.Exit(1)
	}
}

// signalAwareContext creates a context that is cancelled on SIGINT/SIGTERM
func signalAwareContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func run(ctx context.Context) error {
	cfg := parseFlags()

	setupLogger(cfg.Debug)

	slog.Info("Starting k8s-pod-postmortem action",
		"namespace", cfg.Namespace,
		"pod", cfg.PodName,
		"log_tail_lines", cfg.LogTailLines,
		"include_node_info", cfg.IncludeNodeInfo,
		"redact_secrets", cfg.RedactSecrets,
	)

	// Create timeout context
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer timeoutCancel()

	// Initialize Kubernetes client
	k8sClient, err := kubernetes.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Discover namespace and pod if not provided
	namespace, podName, err := k8sClient.DiscoverPodInfo(timeoutCtx, cfg.Namespace, cfg.PodName)
	if err != nil {
		return fmt.Errorf("failed to discover pod info: %w", err)
	}

	slog.Info("Pod identified", "namespace", namespace, "pod", podName)

	// Collect diagnostics
	diagnostics, err := k8sClient.CollectDiagnostics(timeoutCtx, namespace, podName, cfg.LogTailLines, cfg.IncludeNodeInfo)
	if err != nil {
		return fmt.Errorf("failed to collect diagnostics: %w", err)
	}

	// Redact sensitive data
	if cfg.RedactSecrets {
		redactor := redact.NewRedactor()
		diagnostics = redactor.Redact(diagnostics)
	}

	// Analyze failure
	analyzer := analysis.NewAnalyzer()
	classification := analyzer.Analyze(diagnostics)

	slog.Info("Failure classified", "type", classification.Type, "reason", classification.Reason)

	// Generate report
	reporter := reporting.NewReporter()
	report := reporter.GenerateMarkdown(diagnostics, classification)

	// Write to GitHub Actions summary
	ghClient := github.NewClient()
	if err := ghClient.WriteSummary(report); err != nil {
		return fmt.Errorf("failed to write GitHub summary: %w", err)
	}

	slog.Info("Post-mortem report published successfully")

	return nil
}

func parseFlags() *types.Config {
	cfg := &types.Config{}

	flag.IntVar(&cfg.LogTailLines, "log-tail-lines", 200, "Number of log lines to tail from previous container")
	flag.BoolVar(&cfg.IncludeNodeInfo, "include-node-info", false, "Include node information in diagnostics")
	flag.BoolVar(&cfg.RedactSecrets, "redact-secrets", true, "Redact sensitive data from output")
	flag.StringVar(&cfg.OutputFormat, "output-format", "markdown", "Output format (markdown)")
	flag.StringVar(&cfg.Namespace, "namespace", "", "Kubernetes namespace (auto-detected if empty)")
	flag.StringVar(&cfg.PodName, "pod-name", "", "Pod name (auto-detected if empty)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging")
	flag.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Action timeout")

	flag.Parse()

	return cfg
}

func setupLogger(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, opts))
	slog.SetDefault(logger)
}
