// Package github provides GitHub Actions integration functionality
package github

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// GitHub environment variables
	githubOutputFile    = "GITHUB_OUTPUT"
	githubStepSummaryFile = "GITHUB_STEP_SUMMARY"
	githubWorkspace     = "GITHUB_WORKSPACE"
)

// Client handles GitHub Actions integration
type Client struct {
	outputFile    string
	summaryFile   string
	workspace     string
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{
		outputFile:  os.Getenv(githubOutputFile),
		summaryFile: os.Getenv(githubStepSummaryFile),
		workspace:   os.Getenv(githubWorkspace),
	}
}

// WriteSummary writes the post-mortem report to GitHub Actions job summary
func (c *Client) WriteSummary(report string) error {
	if c.summaryFile == "" {
		// Not running in GitHub Actions, print to stdout
		fmt.Println(report)
		return nil
	}

	// Append to the step summary file
	file, err := os.OpenFile(c.summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open step summary file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(report); err != nil {
		return fmt.Errorf("failed to write step summary: %w", err)
	}

	return nil
}

// SetOutput sets a GitHub Actions output variable
func (c *Client) SetOutput(name, value string) error {
	if c.outputFile == "" {
		// Not running in GitHub Actions, skip
		return nil
	}

	file, err := os.OpenFile(c.outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// Handle multiline values
	if strings.Contains(value, "\n") {
		// Use heredoc syntax for multiline values
		delimiter := "EOF"
		for strings.Contains(value, delimiter) {
			delimiter += "X"
		}
		_, err = fmt.Fprintf(file, "%s<<%s\n%s\n%s\n", name, delimiter, value, delimiter)
	} else {
		_, err = fmt.Fprintf(file, "%s=%s\n", name, value)
	}

	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// GetWorkspace returns the GitHub workspace path
func (c *Client) GetWorkspace() string {
	return c.workspace
}

// IsRunningInGitHubActions returns true if running in GitHub Actions
func (c *Client) IsRunningInGitHubActions() bool {
	return c.summaryFile != ""
}

// GetEnv returns a GitHub Actions environment variable
func (c *Client) GetEnv(name string) string {
	return os.Getenv(name)
}

// GetRequiredEnv returns a required GitHub Actions environment variable or an error
func (c *Client) GetRequiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

// ResolvePath resolves a path relative to the GitHub workspace
func (c *Client) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if c.workspace == "" {
		return path
	}
	return filepath.Join(c.workspace, path)
}