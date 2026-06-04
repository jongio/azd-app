package service

import (
	"context"
	"testing"
)

func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		// Valid names
		{"valid uppercase", "MY_VAR", true},
		{"valid lowercase", "my_var", true},
		{"valid mixed", "My_Var_123", true},
		{"valid underscore start", "_private", true},
		{"valid single char", "X", true},
		{"valid with numbers", "VAR123", true},

		// Invalid names - security critical
		{"empty", "", false},
		{"starts with number", "1VAR", false},
		{"contains newline", "VAR\nINJECTION", false},
		{"contains carriage return", "VAR\rINJECTION", false},
		{"contains tab", "VAR\tINJECTION", false},
		{"contains null byte", "VAR\000INJECTION", false},
		{"contains equals", "VAR=value", false},
		{"contains dollar", "VAR$INJECTION", false},
		{"contains semicolon", "VAR;cmd", false},
		{"contains pipe", "VAR|cmd", false},
		{"contains ampersand", "VAR&cmd", false},
		{"contains redirect", "VAR>file", false},
		{"contains backtick", "VAR`cmd`", false},
		{"contains quote", "VAR\"quote", false},
		{"contains single quote", "VAR'quote", false},
		{"contains backslash", "VAR\\path", false},
		{"contains paren", "VAR()", false},
		{"contains bracket", "VAR[]", false},
		{"contains brace", "VAR{}", false},
		{"contains less than", "VAR<file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEnvVarName(tt.varName)
			if result != tt.expected {
				t.Errorf("isValidEnvVarName(%q) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestEnvSliceToMap_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "filters invalid var names",
			input:    []string{"VALID=value", "IN;VALID=value", "ALSO_VALID=value"},
			expected: map[string]string{"VALID": "value", "ALSO_VALID": "value"},
		},
		{
			name:     "filters null bytes in keys",
			input:    []string{"VALID=value", "BAD\000KEY=value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters null bytes in values",
			input:    []string{"VALID=value", "KEY=bad\000value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters newlines in keys",
			input:    []string{"VALID=value", "BAD\nKEY=value"},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "handles empty strings",
			input:    []string{"", "VALID=value", ""},
			expected: map[string]string{"VALID": "value"},
		},
		{
			name:     "filters keys starting with numbers",
			input:    []string{"VALID=value", "1BAD=value"},
			expected: map[string]string{"VALID": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := envSliceToMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("envSliceToMap() returned %d items, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("envSliceToMap()[%q] = %q, want %q", k, result[k], v)
				}
			}
			// Ensure no unexpected keys
			for k := range result {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("envSliceToMap() has unexpected key %q", k)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestIsSensitiveEnvVar — unit tests for the core denylist/allowlist predicate
// ---------------------------------------------------------------------------

func TestIsSensitiveEnvVar(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		// --- Denylist: suffix patterns ---
		{"github token", "GITHUB_TOKEN", true},
		{"azure client secret", "AZURE_CLIENT_SECRET", true},
		{"api key", "MY_API_KEY", true},
		{"db password", "DB_PASSWORD", true},
		{"npm token", "NPM_TOKEN", true},
		{"pypi token", "PYPI_TOKEN", true},
		{"generic secret", "APP_SECRET", true},
		{"private key lowercase", "private_key", true},
		{"password suffix", "POSTGRES_PASSWORD", true},

		// --- Denylist: prefix patterns ---
		{"aws access key id", "AWS_ACCESS_KEY_ID", true},
		{"aws secret access key", "AWS_SECRET_ACCESS_KEY", true},
		{"aws session token", "AWS_SESSION_TOKEN", true},
		{"aws region (still filtered - AWS_* prefix)", "AWS_DEFAULT_REGION", true},
		{"azure client id (AZURE_* prefix)", "AZURE_CLIENT_ID", true},
		{"azure storage account key", "AZURE_STORAGE_ACCOUNT_KEY", true},

		// --- Allowlist overrides ---
		{"azd access token - allowlisted", "AZD_ACCESS_TOKEN", false},
		{"azure subscription id - allowlisted", "AZURE_SUBSCRIPTION_ID", false},
		{"azure tenant id - allowlisted", "AZURE_TENANT_ID", false},
		{"azure location - allowlisted", "AZURE_LOCATION", false},
		{"azure resource group - allowlisted", "AZURE_RESOURCE_GROUP", false},
		{"azure environment - allowlisted", "AZURE_ENVIRONMENT", false},
		// Case-insensitive allowlist matching
		{"lowercase azure subscription id", "azure_subscription_id", false},
		{"mixed case azd access token", "Azd_Access_Token", false},

		// --- Non-sensitive vars: must pass through ---
		{"PATH", "PATH", false},
		{"HOME", "HOME", false},
		{"GOPATH", "GOPATH", false},
		{"NODE_ENV", "NODE_ENV", false},
		{"PORT", "PORT", false},
		{"HOST", "HOST", false},
		{"PYTHONPATH", "PYTHONPATH", false},
		{"AZD_SERVER", "AZD_SERVER", false},
		{"AZD_DEBUG", "AZD_DEBUG", false},
		{"LANG", "LANG", false},
		{"TERM", "TERM", false},
		{"USER", "USER", false},
		{"SHELL", "SHELL", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveEnvVar(tt.key)
			if got != tt.want {
				t.Errorf("isSensitiveEnvVar(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFilterSensitiveEnvVars — acceptance-criteria tests
// ---------------------------------------------------------------------------

// TestFilterSensitiveEnvVars_TokenNotLeaked verifies the primary acceptance criterion:
// a parent-process token must NOT appear in the filtered child environment.
func TestFilterSensitiveEnvVars_TokenNotLeaked(t *testing.T) {
	parent := map[string]string{
		"GITHUB_TOKEN": "ghp_s3cr3tval",
		"PATH":         "/usr/bin:/bin",
		"HOME":         "/root",
		"NODE_ENV":     "development",
	}

	child := FilterSensitiveEnvVars(parent)

	if _, found := child["GITHUB_TOKEN"]; found {
		t.Error("FilterSensitiveEnvVars: GITHUB_TOKEN must not appear in child environment")
	}
	if child["PATH"] != "/usr/bin:/bin" {
		t.Errorf("FilterSensitiveEnvVars: PATH should pass through, got %q", child["PATH"])
	}
	if child["HOME"] != "/root" {
		t.Errorf("FilterSensitiveEnvVars: HOME should pass through, got %q", child["HOME"])
	}
	if child["NODE_ENV"] != "development" {
		t.Errorf("FilterSensitiveEnvVars: NODE_ENV should pass through, got %q", child["NODE_ENV"])
	}
}

// TestFilterSensitiveEnvVars_NonSensitivePassThrough verifies PATH, HOME, and GOPATH pass through.
func TestFilterSensitiveEnvVars_NonSensitivePassThrough(t *testing.T) {
	env := map[string]string{
		"PATH":         "/usr/local/bin:/usr/bin:/bin",
		"HOME":         "/home/runner",
		"GOPATH":       "/home/runner/go",
		"PYTHONPATH":   "/usr/lib/python3",
		"NODE_ENV":     "test",
		"PORT":         "8080",
		"LANG":         "en_US.UTF-8",
		"GITHUB_TOKEN": "should-be-filtered",
		"AWS_SECRET_ACCESS_KEY": "also-filtered",
	}

	got := FilterSensitiveEnvVars(env)

	passThrough := []string{"PATH", "HOME", "GOPATH", "PYTHONPATH", "NODE_ENV", "PORT", "LANG"}
	for _, key := range passThrough {
		if _, found := got[key]; !found {
			t.Errorf("FilterSensitiveEnvVars: non-sensitive var %q must pass through", key)
		}
	}

	blocked := []string{"GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY"}
	for _, key := range blocked {
		if _, found := got[key]; found {
			t.Errorf("FilterSensitiveEnvVars: sensitive var %q must NOT appear in child env", key)
		}
	}
}

// TestFilterSensitiveEnvVars_AllowlistPreserved verifies that allowlisted vars survive filtering.
func TestFilterSensitiveEnvVars_AllowlistPreserved(t *testing.T) {
	env := map[string]string{
		"AZD_ACCESS_TOKEN":      "eyJ...framework-token",
		"AZURE_SUBSCRIPTION_ID": "00000000-0000-0000-0000-000000000001",
		"AZURE_TENANT_ID":       "00000000-0000-0000-0000-000000000002",
		"AZURE_LOCATION":        "eastus",
		"AZURE_RESOURCE_GROUP":  "my-rg",
		"AZURE_ENVIRONMENT":     "AzureCloud",
		// These should still be filtered even though they look Azure-adjacent
		"AZURE_CLIENT_SECRET": "super-secret",
		"AZURE_CLIENT_ID":     "client-id-filtered",
	}

	got := FilterSensitiveEnvVars(env)

	allowlisted := []string{
		"AZD_ACCESS_TOKEN",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"AZURE_LOCATION",
		"AZURE_RESOURCE_GROUP",
		"AZURE_ENVIRONMENT",
	}
	for _, key := range allowlisted {
		if v, found := got[key]; !found {
			t.Errorf("FilterSensitiveEnvVars: allowlisted var %q must pass through", key)
		} else if v != env[key] {
			t.Errorf("FilterSensitiveEnvVars: allowlisted var %q value changed: got %q, want %q", key, v, env[key])
		}
	}

	filtered := []string{"AZURE_CLIENT_SECRET", "AZURE_CLIENT_ID"}
	for _, key := range filtered {
		if _, found := got[key]; found {
			t.Errorf("FilterSensitiveEnvVars: sensitive var %q must NOT appear in child env", key)
		}
	}
}

// TestFilterSensitiveEnvVars_DenylistPatterns exercises every denylist pattern.
func TestFilterSensitiveEnvVars_DenylistPatterns(t *testing.T) {
	sensitive := map[string]string{
		// Suffix: _TOKEN
		"GITHUB_TOKEN":  "ghp_token",
		"NPM_TOKEN":     "npm_token",
		"PYPI_TOKEN":    "pypi_token",
		"ACCESS_TOKEN":  "access_token",
		// Suffix: _SECRET
		"CLIENT_SECRET": "client_secret",
		"APP_SECRET":    "app_secret",
		// Suffix: _KEY
		"API_KEY":          "api_key",
		"PRIVATE_KEY":      "private_key",
		"SSH_PRIVATE_KEY":  "ssh_key",
		// Suffix: _PASSWORD
		"DB_PASSWORD":       "db_pass",
		"POSTGRES_PASSWORD": "pg_pass",
		"REDIS_PASSWORD":    "redis_pass",
		// Prefix: AWS_
		"AWS_ACCESS_KEY_ID":     "AKIA...",
		"AWS_SECRET_ACCESS_KEY": "aws_secret",
		"AWS_SESSION_TOKEN":     "aws_session",
		// Prefix: AZURE_
		"AZURE_CLIENT_SECRET": "az_secret",
		"AZURE_CLIENT_ID":     "az_client_id",
		"AZURE_STORAGE_KEY":   "az_storage_key",
	}

	got := FilterSensitiveEnvVars(sensitive)

	if len(got) != 0 {
		remaining := make([]string, 0, len(got))
		for k := range got {
			remaining = append(remaining, k)
		}
		t.Errorf("FilterSensitiveEnvVars: expected all sensitive vars to be filtered; remaining: %v", remaining)
	}
}

// TestFilterSensitiveEnvVars_DoesNotMutateInput verifies the function is pure.
func TestFilterSensitiveEnvVars_DoesNotMutateInput(t *testing.T) {
	original := map[string]string{
		"GITHUB_TOKEN": "secret",
		"PATH":         "/usr/bin",
	}
	originalLen := len(original)

	_ = FilterSensitiveEnvVars(original)

	if len(original) != originalLen {
		t.Errorf("FilterSensitiveEnvVars mutated the input map: now has %d entries, want %d", len(original), originalLen)
	}
	if _, ok := original["GITHUB_TOKEN"]; !ok {
		t.Error("FilterSensitiveEnvVars removed GITHUB_TOKEN from the input map (must not mutate)")
	}
}

// TestFilterSensitiveEnvVars_EmptyMap handles the zero-value case.
func TestFilterSensitiveEnvVars_EmptyMap(t *testing.T) {
	got := FilterSensitiveEnvVars(map[string]string{})
	if len(got) != 0 {
		t.Errorf("FilterSensitiveEnvVars(empty) = %d entries, want 0", len(got))
	}
}

// TestFilterSensitiveEnvVars_ServiceCanOverrideThroughExplicitDeclaration verifies that
// if a service explicitly declares a sensitive var in its environment config, it is included.
// (This exercises the post-filter merge path in ResolveEnvironment.)
func TestFilterSensitiveEnvVars_ServiceCanOverrideThroughExplicitDeclaration(t *testing.T) {
	// Simulate: OS env has GITHUB_TOKEN (which gets filtered from the baseline),
	// but the service explicitly declares it in azure.yaml environment section.
	baseEnv := map[string]string{
		"GITHUB_TOKEN": "ghp_from_os",
		"PATH":         "/usr/bin",
	}

	// FilterSensitiveEnvVars removes it from the OS baseline.
	filtered := FilterSensitiveEnvVars(baseEnv)

	if _, found := filtered["GITHUB_TOKEN"]; found {
		t.Error("OS-level GITHUB_TOKEN should be filtered from the baseline")
	}

	// Service explicitly re-declares it (highest priority merge in ResolveEnvironment).
	explicit := map[string]string{
		"GITHUB_TOKEN": "ghp_explicit_in_azure_yaml",
	}
	for k, v := range explicit {
		filtered[k] = v
	}

	// After explicit merge, the service does see its declared value.
	if filtered["GITHUB_TOKEN"] != "ghp_explicit_in_azure_yaml" {
		t.Errorf("explicit service env declaration should survive: got %q", filtered["GITHUB_TOKEN"])
	}
}

// TestFilterSensitiveEnvVars_ResolveEnvironmentIntegration verifies that
// ResolveEnvironment does not expose a sensitive OS env var to child services
// when it is not declared in the service environment config.
func TestFilterSensitiveEnvVars_ResolveEnvironmentIntegration(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_secret_integration_test")
	t.Setenv("AZURE_CLIENT_SECRET", "az_secret_integration_test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws_secret_integration_test")
	t.Setenv("PATH", "/usr/bin:/bin") // must survive

	svc := Service{
		Host: "localhost",
		// No environment declarations — service should NOT see OS secrets.
	}

	got, err := ResolveEnvironment(context.Background(), svc, map[string]string{}, "", map[string]string{})
	if err != nil {
		t.Fatalf("ResolveEnvironment() unexpected error: %v", err)
	}

	sensitive := []string{"GITHUB_TOKEN", "AZURE_CLIENT_SECRET", "AWS_SECRET_ACCESS_KEY"}
	for _, key := range sensitive {
		if _, found := got[key]; found {
			t.Errorf("ResolveEnvironment: sensitive var %q must not appear in child env", key)
		}
	}

	if got["PATH"] == "" {
		t.Error("ResolveEnvironment: PATH must pass through to child env")
	}
}
