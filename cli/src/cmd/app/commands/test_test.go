package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	testrunner "github.com/jongio/azd-app/cli/src/internal/testing"
	"github.com/jongio/azd-core/cliout"
	"github.com/spf13/cobra"
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
		"changed",
		"changed-base",
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

// TestEnvFlagNotShadowed verifies the test command does not define its own
// --environment flag. azd reserves --environment/-e globally, and a local copy
// shadowed it so azd never saw the value the user passed. The command now reads
// the inherited persistent flag instead.
func TestEnvFlagNotShadowed(t *testing.T) {
	cmd := NewTestCommand()

	if flag := cmd.Flags().Lookup("environment"); flag != nil {
		t.Fatal("test command must not define a local --environment flag; it shadows the azd global flag")
	}
	if flag := cmd.Flags().ShorthandLookup("e"); flag != nil {
		t.Fatalf("test command must not define a local -e shorthand, found %q", flag.Name)
	}

	root := &cobra.Command{Use: "app"}
	root.PersistentFlags().StringP("environment", "e", "", "azd environment name")
	root.AddCommand(cmd)

	if err := cmd.ParseFlags([]string{"-e", "staging"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	inherited := cmd.Flags().Lookup("environment")
	if inherited == nil {
		t.Fatal("expected --environment to resolve from the inherited persistent flag")
	}
	if got := inherited.Value.String(); got != "staging" {
		t.Errorf("inherited environment = %q, want %q", got, "staging")
	}
}

// TestEnvFlagPropagatesToOptions pins the second half of the fix: reading the
// inherited flag is only useful if the value reaches opts.Environment. Without
// this, deleting the copy in RunE would leave TestEnvFlagNotShadowed green
// while `azd app test -e staging` silently ran against the default environment.
func TestEnvFlagPropagatesToOptions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"long form", []string{"--environment", "staging"}, "staging"},
		{"shorthand", []string{"-e", "prod"}, "prod"},
		{"omitted", nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var captured *TestOptions
			original := runTestsFn
			runTestsFn = func(_ *orchestrator.Orchestrator, opts *TestOptions) error {
				captured = opts
				return nil
			}
			t.Cleanup(func() { runTestsFn = original })

			root := &cobra.Command{Use: "app"}
			root.PersistentFlags().StringP("environment", "e", "", "azd environment name")
			cmd := NewTestCommand()
			root.AddCommand(cmd)

			root.SetArgs(append([]string{"test"}, tc.args...))
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			if err := root.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}

			if captured == nil {
				t.Fatal("RunE did not reach the test runner")
			}
			if captured.Environment != tc.want {
				t.Errorf("opts.Environment = %q, want %q", captured.Environment, tc.want)
			}
		})
	}
}

// TestLoadAzdEnvironment tests loading azd environment variables from .azure/<env>/.env.
func TestLoadAzdEnvironment(t *testing.T) {
	t.Run("loads environment variables", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create azure.yaml
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create .azure/staging/.env
		envDir := filepath.Join(tmpDir, ".azure", "staging")
		if err := os.MkdirAll(envDir, 0o755); err != nil {
			t.Fatal(err)
		}
		envContent := "API_URL=https://staging.example.com\nDB_HOST=staging-db.internal\n"
		if err := os.WriteFile(filepath.Join(envDir, ".env"), []byte(envContent), 0o644); err != nil {
			t.Fatal(err)
		}

		// Clear env vars that might be set
		_ = os.Unsetenv("API_URL")
		_ = os.Unsetenv("DB_HOST")

		err := loadAzdEnvironment(azureYamlPath, "staging")
		if err != nil {
			t.Fatalf("loadAzdEnvironment() error = %v", err)
		}

		// Verify env vars are set
		if got := os.Getenv("API_URL"); got != "https://staging.example.com" {
			t.Errorf("API_URL = %q, want %q", got, "https://staging.example.com")
		}
		if got := os.Getenv("DB_HOST"); got != "staging-db.internal" {
			t.Errorf("DB_HOST = %q, want %q", got, "staging-db.internal")
		}

		// Cleanup
		_ = os.Unsetenv("API_URL")
		_ = os.Unsetenv("DB_HOST")
	})

	t.Run("returns error for nonexistent environment", func(t *testing.T) {
		tmpDir := t.TempDir()
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		err := loadAzdEnvironment(azureYamlPath, "nonexistent")
		if err == nil {
			t.Fatal("Expected error for nonexistent environment")
		}

		if !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("Error should mention environment name, got: %v", err)
		}
	})

	t.Run("rejects path traversal in env name", func(t *testing.T) {
		tmpDir := t.TempDir()
		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		traversalNames := []string{"../../etc", "foo/bar", `foo\bar`, ".."}
		for _, name := range traversalNames {
			err := loadAzdEnvironment(azureYamlPath, name)
			if err == nil {
				t.Errorf("Expected error for env name %q, got nil", name)
			}
			if err != nil && !strings.Contains(err.Error(), "invalid environment name") {
				t.Errorf("Expected 'invalid environment name' error for %q, got: %v", name, err)
			}
		}
	})

	t.Run("handles empty env file", func(t *testing.T) {
		tmpDir := t.TempDir()

		azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
		if err := os.WriteFile(azureYamlPath, []byte("name: test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		envDir := filepath.Join(tmpDir, ".azure", "empty-env")
		if err := os.MkdirAll(envDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(envDir, ".env"), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}

		err := loadAzdEnvironment(azureYamlPath, "empty-env")
		if err != nil {
			t.Fatalf("loadAzdEnvironment() error = %v", err)
		}
	})
}

// Issue #557: an explicit test.<type>.command must be reported as the command
// that will actually run. Reporting the framework instead told users their
// configured command was being ignored, which is what made the execution bug
// look like it was still present after it had been fixed.
func TestDisplayValidationSummary_ExplicitCommandReportedOverFramework(t *testing.T) {
	out, err := captureStdout(t, func() error {
		displayValidationSummary([]testrunner.ServiceValidation{
			{Name: "api", CanTest: true, Framework: "pytest", Command: "uv run pytest -q"},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}

	if !strings.Contains(out, "uv run pytest -q") {
		t.Errorf("summary must name the command that will run, got:\n%s", out)
	}
	if strings.Contains(out, "pytest detected") {
		t.Errorf("summary must not claim framework detection when a command is set, got:\n%s", out)
	}
}

// A service with no explicit command still reports its detected framework.
func TestDisplayValidationSummary_FrameworkReportedWhenNoCommand(t *testing.T) {
	out, err := captureStdout(t, func() error {
		displayValidationSummary([]testrunner.ServiceValidation{
			{Name: "web", CanTest: true, Framework: "vitest", TestFiles: 3},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}

	if !strings.Contains(out, "vitest detected") {
		t.Errorf("expected framework detection line, got:\n%s", out)
	}
	if !strings.Contains(out, "3 test files") {
		t.Errorf("expected test file count, got:\n%s", out)
	}
}

// Skipped services report why they were skipped and are excluded from the count.
func TestDisplayValidationSummary_ReportsSkippedServices(t *testing.T) {
	out, err := captureStdout(t, func() error {
		displayValidationSummary([]testrunner.ServiceValidation{
			{Name: "api", CanTest: true, Command: "go test ./..."},
			{Name: "docs", CanTest: false, SkipReason: "no test framework detected"},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}

	if !strings.Contains(out, "no test framework detected") {
		t.Errorf("expected skip reason, got:\n%s", out)
	}
	if !strings.Contains(out, "1 testable services (1 skipped)") {
		t.Errorf("expected testable/skipped counts, got:\n%s", out)
	}
}

// JSON output is a machine contract, so the human summary must stay out of it.
func TestDisplayValidationSummary_SilentInJSONMode(t *testing.T) {
	if err := cliout.SetFormat("json"); err != nil {
		t.Fatalf("SetFormat: %v", err)
	}
	t.Cleanup(func() { _ = cliout.SetFormat("default") })

	out, err := captureStdout(t, func() error {
		displayValidationSummary([]testrunner.ServiceValidation{
			{Name: "api", CanTest: true, Command: "go test ./..."},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}

	if strings.TrimSpace(out) != "" {
		t.Errorf("JSON mode must emit no human summary, got:\n%s", out)
	}
}
