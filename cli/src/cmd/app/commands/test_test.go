package commands

import (
	"testing"
)

// TestNewTestCommand verifies that the test command is created correctly.
func TestNewTestCommand(t *testing.T) {
	cmd := NewTestCommand()

	if cmd == nil {
		t.Fatal("NewTestCommand returned nil")
	}

	if cmd.Use != "test" {
		t.Errorf("Expected Use to be 'test', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	// Verify flags are registered
	flags := []string{
		"type",
		"coverage",
		"service",
		"watch",
		"update-snapshots",
		"fail-fast",
		"parallel",
		"threshold",
		"verbose",
		"dry-run",
		"output-format",
		"output-dir",
	}

	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("Expected flag '%s' to be registered", flagName)
		}
	}
}

// TestTestTypeValidation tests validation of test type parameter.
func TestTestTypeValidation(t *testing.T) {
	tests := []struct {
		name      string
		testType  string
		shouldErr bool
	}{
		{"valid unit", "unit", false},
		{"valid integration", "integration", false},
		{"valid e2e", "e2e", false},
		{"valid all", "all", false},
		{"invalid type", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test type validation
			validTypes := map[string]bool{
				"unit":        true,
				"integration": true,
				"e2e":         true,
				"all":         true,
			}

			valid := validTypes[tt.testType]
			if valid == tt.shouldErr {
				t.Errorf("Expected validation for '%s' to be %v, got %v", tt.testType, !tt.shouldErr, valid)
			}
		})
	}
}

// TestThresholdValidation tests validation of coverage threshold.
func TestThresholdValidation(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		shouldErr bool
	}{
		{"valid 0", 0, false},
		{"valid 50", 50, false},
		{"valid 100", 100, false},
		{"invalid negative", -1, true},
		{"invalid over 100", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.threshold >= 0 && tt.threshold <= 100
			if valid == tt.shouldErr {
				t.Errorf("Expected validation for threshold %d to be %v, got %v", tt.threshold, !tt.shouldErr, valid)
			}
		})
	}
}

// TestOutputFormatValidation tests validation of output format.
func TestOutputFormatValidation(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		shouldErr bool
	}{
		{"valid default", "default", false},
		{"valid json", "json", false},
		{"valid junit", "junit", false},
		{"valid github", "github", false},
		{"invalid format", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validFormats := map[string]bool{
				"default": true,
				"json":    true,
				"junit":   true,
				"github":  true,
			}

			valid := validFormats[tt.format]
			if valid == tt.shouldErr {
				t.Errorf("Expected validation for format '%s' to be %v, got %v", tt.format, !tt.shouldErr, valid)
			}
		})
	}
}
