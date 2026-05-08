// Package redact provides sensitive data redaction functionality
package redact

import (
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

// Redactor handles sensitive data redaction
type Redactor struct {
	patterns []*redactionPattern
}

// redactionPattern defines a pattern to redact
type redactionPattern struct {
	name        string
	regex       *regexp.Regexp
	replacement string
}

// NewRedactor creates a new Redactor with default patterns
func NewRedactor() *Redactor {
	r := &Redactor{}
	r.initPatterns()
	return r
}

// initPatterns initializes the redaction patterns
func (r *Redactor) initPatterns() {
	// Common secret patterns
	patterns := []struct {
		name        string
		pattern     string
		replacement string
	}{
		// AWS credentials
		{
			name:        "AWS Access Key ID",
			pattern:     `(?i)(AWS_ACCESS_KEY_ID|aws_access_key_id)\s*[=:]\s*['"]?[A-Z0-9]{20}['"]?`,
			replacement: "[REDACTED_AWS_ACCESS_KEY_ID]",
		},
		{
			name:        "AWS Secret Access Key",
			pattern:     `(?i)(AWS_SECRET_ACCESS_KEY|aws_secret_access_key)\s*[=:]\s*['"]?[A-Za-z0-9/+=]{40}['"]?`,
			replacement: "[REDACTED_AWS_SECRET_ACCESS_KEY]",
		},
		{
			name:        "AWS Session Token",
			pattern:     `(?i)(AWS_SESSION_TOKEN|aws_session_token)\s*[=:]\s*['"]?[A-Za-z0-9/+=]+['"]?`,
			replacement: "[REDACTED_AWS_SESSION_TOKEN]",
		},
		// GitHub tokens
		{
			name:        "GitHub Token",
			pattern:     `(?i)(GITHUB_TOKEN|github_token|GH_TOKEN)\s*[=:]\s*['"]?[a-zA-Z0-9_]+['"]?`,
			replacement: "[REDACTED_GITHUB_TOKEN]",
		},
		{
			name:        "GitHub PAT",
			pattern:     `ghp_[a-zA-Z0-9]{36}`,
			replacement: "[REDACTED_GITHUB_PAT]",
		},
		{
			name:        "GitHub OAuth Token",
			pattern:     `gho_[a-zA-Z0-9]{36}`,
			replacement: "[REDACTED_GITHUB_OAUTH]",
		},
		{
			name:        "GitHub App Token",
			pattern:     `(ghu|ghs)_[a-zA-Z0-9]{36}`,
			replacement: "[REDACTED_GITHUB_APP_TOKEN]",
		},
		// Generic secrets
		{
			name:        "Authorization Header",
			pattern:     `(?i)(Authorization|authorization)\s*:\s*(Bearer|Basic)\s+[A-Za-z0-9\-._~+/]+=*`,
			replacement: "[REDACTED_AUTH_HEADER]",
		},
		{
			name:        "Bearer Token",
			pattern:     `(?i)(Bearer|bearer)\s+[A-Za-z0-9\-._~+/]+=*`,
			replacement: "[REDACTED_BEARER_TOKEN]",
		},
		// Private keys
		{
			name:        "Private Key",
			pattern:     `-----BEGIN (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----`,
			replacement: "[REDACTED_PRIVATE_KEY]",
		},
		// Database connection strings
		{
			name:        "Database URL",
			pattern:     `(?i)(postgres|mysql|mongodb|redis)://[^\s]+:[^\s]+@[^\s]+`,
			replacement: "[REDACTED_DATABASE_URL]",
		},
		// Generic password patterns
		{
			name:        "Password in connection string",
			pattern:     `(?i)(password|passwd|pwd)\s*[=:]\s*['"]?[^\s'"]+['"]?`,
			replacement: "[REDACTED_PASSWORD]",
		},
		// API Keys
		{
			name:        "API Key",
			pattern:     `(?i)(api[_-]?key|apikey)\s*[=:]\s*['"]?[a-zA-Z0-9_\-]+['"]?`,
			replacement: "[REDACTED_API_KEY]",
		},
		// Kubernetes secrets
		{
			name:        "Kubernetes Token",
			pattern:     `(?i)(token)\s*[=:]\s*['"]?[A-Za-z0-9\-._~+/]+=*['"]?`,
			replacement: "[REDACTED_TOKEN]",
		},
		// Base64 encoded secrets (common patterns)
		{
			name:        "Base64 Secret",
			pattern:     `(?i)(secret|password|key)\s*[=:]\s*['"]?[A-Za-z0-9\-._~+/]+=*['"]?`,
			replacement: "[REDACTED_SECRET]",
		},
		// Environment variable secrets
		{
			name:        "Secret Environment Variable",
			pattern:     `(?i)(SECRET|PASSWORD|TOKEN|KEY|CREDENTIAL)_[A-Z_]+\s*=\s*['"]?[^\s'"]+['"]?`,
			replacement: "[REDACTED_SECRET_ENV]",
		},
	}

	for _, p := range patterns {
		regex, err := regexp.Compile(p.pattern)
		if err != nil {
			// Skip invalid patterns
			continue
		}
		r.patterns = append(r.patterns, &redactionPattern{
			name:        p.name,
			regex:       regex,
			replacement: p.replacement,
		})
	}
}

// Redact applies redaction to diagnostics
func (r *Redactor) Redact(diagnostics *types.Diagnostics) *types.Diagnostics {
	// Create a copy to avoid modifying the original
	result := &types.Diagnostics{
		Namespace:      diagnostics.Namespace,
		PodName:        diagnostics.PodName,
		Pod:            diagnostics.Pod,
		Events:         make([]corev1.Event, len(diagnostics.Events)),
		PreviousLogs:   make(map[string]string),
		Node:           diagnostics.Node,
		ContainerStats: diagnostics.ContainerStats,
	}

	// Redact events
	for i, event := range diagnostics.Events {
		result.Events[i] = event
		result.Events[i].Message = r.redactString(event.Message)
	}

	// Redact logs
	for container, logs := range diagnostics.PreviousLogs {
		result.PreviousLogs[container] = r.redactString(logs)
	}

	return result
}

// redactString applies all redaction patterns to a string
func (r *Redactor) redactString(s string) string {
	result := s
	for _, pattern := range r.patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}
	return result
}

// RedactString is a convenience method for redacting arbitrary strings
func (r *Redactor) RedactString(s string) string {
	return r.redactString(s)
}

// AddPattern allows adding custom redaction patterns
func (r *Redactor) AddPattern(name, pattern, replacement string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.patterns = append(r.patterns, &redactionPattern{
		name:        name,
		regex:       regex,
		replacement: replacement,
	})
	return nil
}

// IsSecretKey checks if a key name suggests it contains sensitive data
func IsSecretKey(key string) bool {
	key = strings.ToLower(key)
	secretIndicators := []string{
		"password", "passwd", "pwd",
		"secret", "token", "key",
		"credential", "auth", "api_key",
		"private", "access_key", "session",
	}

	for _, indicator := range secretIndicators {
		if strings.Contains(key, indicator) {
			return true
		}
	}
	return false
}
