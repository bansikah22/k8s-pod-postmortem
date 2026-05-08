package redact

import (
	"testing"

	"github.com/bansikah22/k8s-pod-postmortem/internal/types"
)

func TestRedactor_RedactString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "redacts AWS access key",
			input:    "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			contains: "[REDACTED_AWS_ACCESS_KEY_ID]",
		},
		{
			name:     "redacts AWS secret key",
			input:    "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			contains: "[REDACTED_AWS_SECRET_ACCESS_KEY]",
		},
		{
			name:     "redacts GitHub token",
			input:    "GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED_GITHUB_TOKEN]",
		},
		{
			name:     "redacts password",
			input:    "password=supersecret123",
			contains: "[REDACTED_PASSWORD]",
		},
		{
			name:     "redacts authorization header",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains: "[REDACTED_AUTH_HEADER]",
		},
		{
			name:     "redacts private key",
			input:    "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB\n-----END RSA PRIVATE KEY-----",
			contains: "[REDACTED_PRIVATE_KEY]",
		},
	}

	r := NewRedactor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.RedactString(tt.input)
			if !contains(result, tt.contains) {
				t.Errorf("RedactString() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

func TestRedactor_Redact(t *testing.T) {
	r := NewRedactor()

	diagnostics := &types.Diagnostics{
		Namespace: "default",
		PodName:   "test-pod",
		PreviousLogs: map[string]string{
			"container": "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\npassword=secret123",
		},
	}

	result := r.Redact(diagnostics)

	if result.Namespace != diagnostics.Namespace {
		t.Errorf("Namespace mismatch")
	}
	if result.PodName != diagnostics.PodName {
		t.Errorf("PodName mismatch")
	}

	logResult := result.PreviousLogs["container"]
	if contains(logResult, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS access key not redacted")
	}
	if contains(logResult, "secret123") {
		t.Errorf("Password not redacted")
	}
}

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"PASSWORD", true},
		{"api_key", true},
		{"secret_token", true},
		{"credential", true},
		{"username", false},
		{"hostname", false},
		{"port", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsSecretKey(tt.key)
			if result != tt.expected {
				t.Errorf("IsSecretKey(%s) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
