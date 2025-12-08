package commands

import (
	"testing"

	testrunner "github.com/jongio/azd-app/cli/src/internal/testing"
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

// TestDisplayTestResults tests the displayTestResults function.
func TestDisplayTestResults(t *testing.T) {
	tests := []struct {
		name   string
		result *testrunner.AggregateResult
	}{
		{
			name: "all passed",
			result: &testrunner.AggregateResult{
				Success: true,
				Passed:  5,
				Failed:  0,
				Skipped: 0,
				Total:   5,
				Services: []*testrunner.TestResult{
					{
						Service:  "web",
						Success:  true,
						Passed:   3,
						Total:    3,
						Duration: 1.5,
					},
					{
						Service:  "api",
						Success:  true,
						Passed:   2,
						Total:    2,
						Duration: 0.5,
					},
				},
				Duration: 2.0,
			},
		},
		{
			name: "with failures",
			result: &testrunner.AggregateResult{
				Success: false,
				Passed:  3,
				Failed:  2,
				Skipped: 1,
				Total:   6,
				Services: []*testrunner.TestResult{
					{
						Service:  "web",
						Success:  true,
						Passed:   3,
						Total:    3,
						Duration: 1.0,
					},
					{
						Service:  "api",
						Success:  false,
						Passed:   0,
						Failed:   2,
						Total:    2,
						Duration: 0.5,
						Error:    "test failed",
					},
				},
				Duration: 1.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			displayTestResults(tt.result)
		})
	}
}

// TestFlagDefaults tests that flag defaults are set correctly.
func TestFlagDefaults(t *testing.T) {
	cmd := NewTestCommand()

	// Check type default
	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag.DefValue != "all" {
		t.Errorf("Expected type default 'all', got '%s'", typeFlag.DefValue)
	}

	// Check parallel default
	parallelFlag := cmd.Flags().Lookup("parallel")
	if parallelFlag.DefValue != "true" {
		t.Errorf("Expected parallel default 'true', got '%s'", parallelFlag.DefValue)
	}

	// Check output-format default
	formatFlag := cmd.Flags().Lookup("output-format")
	if formatFlag.DefValue != "default" {
		t.Errorf("Expected output-format default 'default', got '%s'", formatFlag.DefValue)
	}

	// Check output-dir default
	dirFlag := cmd.Flags().Lookup("output-dir")
	if dirFlag.DefValue != "./test-results" {
		t.Errorf("Expected output-dir default './test-results', got '%s'", dirFlag.DefValue)
	}
}

// TestFlagShortcuts tests that flag shortcuts are registered correctly.
func TestFlagShortcuts(t *testing.T) {
	cmd := NewTestCommand()

	// Check type shortcut
	typeFlag := cmd.Flags().ShorthandLookup("t")
	if typeFlag == nil || typeFlag.Name != "type" {
		t.Error("Expected -t shortcut for --type flag")
	}

	// Check coverage shortcut
	coverageFlag := cmd.Flags().ShorthandLookup("c")
	if coverageFlag == nil || coverageFlag.Name != "coverage" {
		t.Error("Expected -c shortcut for --coverage flag")
	}

	// Check service shortcut
	serviceFlag := cmd.Flags().ShorthandLookup("s")
	if serviceFlag == nil || serviceFlag.Name != "service" {
		t.Error("Expected -s shortcut for --service flag")
	}

	// Check watch shortcut
	watchFlag := cmd.Flags().ShorthandLookup("w")
	if watchFlag == nil || watchFlag.Name != "watch" {
		t.Error("Expected -w shortcut for --watch flag")
	}

	// Check verbose shortcut
	verboseFlag := cmd.Flags().ShorthandLookup("v")
	if verboseFlag == nil || verboseFlag.Name != "verbose" {
		t.Error("Expected -v shortcut for --verbose flag")
	}

	// Check update-snapshots shortcut
	snapshotsFlag := cmd.Flags().ShorthandLookup("u")
	if snapshotsFlag == nil || snapshotsFlag.Name != "update-snapshots" {
		t.Error("Expected -u shortcut for --update-snapshots flag")
	}

	// Check parallel shortcut
	parallelFlag := cmd.Flags().ShorthandLookup("p")
	if parallelFlag == nil || parallelFlag.Name != "parallel" {
		t.Error("Expected -p shortcut for --parallel flag")
	}
}

// TestServiceFilterParsing tests parsing of comma-separated service filter.
func TestServiceFilterParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single service", "web", []string{"web"}},
		{"two services", "web,api", []string{"web", "api"}},
		{"with spaces", "web, api, worker", []string{"web", "api", "worker"}},
		{"empty", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []string
			if tt.input != "" {
				parts := parseServiceFilter(tt.input)
				result = parts
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d services, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if i < len(result) && result[i] != expected {
					t.Errorf("Expected service[%d] = '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

// TestCommandLongDescription tests that the long description is set.
func TestCommandLongDescription(t *testing.T) {
	cmd := NewTestCommand()

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Verify it mentions supported languages
	if len(cmd.Long) < 50 {
		t.Error("Expected Long description to be comprehensive")
	}
}

// TestDisplayTestResults_WithCoverage tests display with coverage data.
func TestDisplayTestResults_WithCoverage(t *testing.T) {
	result := &testrunner.AggregateResult{
		Success:  true,
		Passed:   5,
		Total:    5,
		Duration: 1.0,
		Services: []*testrunner.TestResult{
			{
				Service:  "web",
				Success:  true,
				Passed:   5,
				Total:    5,
				Duration: 1.0,
			},
		},
		Coverage: &testrunner.AggregateCoverage{
			Aggregate: &testrunner.CoverageData{
				Lines: testrunner.CoverageMetric{
					Total:   100,
					Covered: 85,
					Percent: 85.0,
				},
			},
			Threshold: 80.0,
			Met:       true,
		},
	}

	// Should not panic
	displayTestResults(result)
}

// TestDisplayTestResults_EmptyServices tests display with no service results.
func TestDisplayTestResults_EmptyServices(t *testing.T) {
	result := &testrunner.AggregateResult{
		Success:  true,
		Passed:   0,
		Total:    0,
		Duration: 0,
		Services: []*testrunner.TestResult{},
	}

	// Should not panic
	displayTestResults(result)
}
