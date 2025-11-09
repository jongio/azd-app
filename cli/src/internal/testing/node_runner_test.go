package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewNodeTestRunner(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "jest",
	}

	runner := NewNodeTestRunner(tmpDir, config)
	if runner == nil {
		t.Fatal("Expected runner to be created")
	}
	if runner.projectDir != tmpDir {
		t.Error("Project dir not set correctly")
	}
	if runner.config != config {
		t.Error("Config not set correctly")
	}
	if runner.packageManager == "" {
		t.Error("Package manager should be set")
	}
}

func TestNodeRunnerBuildTestCommand_CustomCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "jest",
		Unit: &TestTypeConfig{
			Command: "npm run test:unit",
		},
	}

	runner := NewNodeTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", false)

	if command != "npm" {
		t.Errorf("Expected command 'npm', got '%s'", command)
	}
	if len(args) < 2 || args[0] != "run" || args[1] != "test:unit" {
		t.Errorf("Expected args 'run test:unit', got %v", args)
	}
}

func TestNodeRunnerBuildTestCommand_Jest(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "jest",
	}

	runner := NewNodeTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", false)

	// Should use package manager
	if command == "" {
		t.Error("Command should not be empty")
	}

	// Should include test
	found := false
	for _, arg := range args {
		if arg == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'test' in args")
	}
}

func TestNodeRunnerBuildTestCommand_WithCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "jest",
	}

	runner := NewNodeTestRunner(tmpDir, config)
	_, args := runner.buildTestCommand("all", true)

	// Should include coverage flag
	found := false
	for _, arg := range args {
		if arg == "--coverage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected '--coverage' in args")
	}
}

func TestNodeRunnerParseCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewNodeTestRunner(tmpDir, config)

	tests := []struct {
		name            string
		command         string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:            "Simple command",
			command:         "npm test",
			expectedCommand: "npm",
			expectedArgs:    []string{"test"},
		},
		{
			name:            "Command with multiple args",
			command:         "npm run test:unit --coverage",
			expectedCommand: "npm",
			expectedArgs:    []string{"run", "test:unit", "--coverage"},
		},
		{
			name:            "Single word command",
			command:         "jest",
			expectedCommand: "jest",
			expectedArgs:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := runner.parseCommand(tt.command)
			if cmd != tt.expectedCommand {
				t.Errorf("Expected command '%s', got '%s'", tt.expectedCommand, cmd)
			}
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
			}
			for i, arg := range args {
				if i < len(tt.expectedArgs) && arg != tt.expectedArgs[i] {
					t.Errorf("Expected arg[%d] '%s', got '%s'", i, tt.expectedArgs[i], arg)
				}
			}
		})
	}
}

func TestNodeRunnerParseTestOutput(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewNodeTestRunner(tmpDir, config)

	tests := []struct {
		name           string
		output         string
		expectedPassed int
		expectedFailed int
		expectedTotal  int
	}{
		{
			name: "Jest output",
			output: `PASS  src/utils.test.js
  ✓ test 1 (5 ms)
  ✓ test 2 (3 ms)

Tests:       2 passed, 2 total
Time:        1.234 s`,
			expectedPassed: 2,
			expectedFailed: 0,
			expectedTotal:  2,
		},
		{
			name: "Jest output with failures",
			output: `FAIL  src/utils.test.js
  ✓ test 1 (5 ms)
  ✕ test 2 (3 ms)

Tests:       1 passed, 1 failed, 2 total
Time:        1.234 s`,
			expectedPassed: 1,
			expectedFailed: 1,
			expectedTotal:  2,
		},
		{
			name: "Vitest output",
			output: `✓ src/utils.test.ts (2)
   ✓ test 1
   ✓ test 2

Tests:  2 passed, 2 total
Time:   1.23s`,
			expectedPassed: 2,
			expectedFailed: 0,
			expectedTotal:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{}
			runner.parseTestOutput(tt.output, result)

			if result.Passed != tt.expectedPassed {
				t.Errorf("Expected %d passed, got %d", tt.expectedPassed, result.Passed)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("Expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.Total != tt.expectedTotal {
				t.Errorf("Expected %d total, got %d", tt.expectedTotal, result.Total)
			}
		})
	}
}

func TestNodeRunnerHasTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a package.json with test script
	packageJSON := `{
		"scripts": {
			"test": "jest"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	config := &ServiceTestConfig{Framework: "jest"}
	runner := NewNodeTestRunner(tmpDir, config)

	if !runner.HasTests() {
		t.Error("Expected HasTests to return true")
	}
}

func TestNodeRunnerHasTests_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ServiceTestConfig{Framework: "jest"}
	runner := NewNodeTestRunner(tmpDir, config)

	if runner.HasTests() {
		t.Error("Expected HasTests to return false for directory without tests")
	}
}

// TestNodeRunnerRunTests_Integration tests the full RunTests workflow
func TestNodeRunnerRunTests_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple package.json
	packageJSON := filepath.Join(tmpDir, "package.json")
	content := `{
"name": "test-project",
"scripts": {
"test": "echo 'Tests: 5 passed, 5 total'"
}
}`
	if err := os.WriteFile(packageJSON, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "jest",
	}

	runner := NewNodeTestRunner(tmpDir, config)
	result, err := runner.RunTests("unit", false)

	// The command should execute (even if it's just echo)
	if err != nil {
		// It's ok if it fails due to npm not being available
		t.Logf("RunTests returned error (expected in test env): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// TestNodeRunnerBuildTestCommand_Coverage tests coverage flag
func TestNodeRunnerBuildTestCommand_Coverage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "jest",
	}

	runner := NewNodeTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", true)

	if command != "npm" {
		t.Errorf("Expected command 'npm', got '%s'", command)
	}

	// Check that coverage flag is present
	found := false
	for _, arg := range args {
		if arg == "--coverage" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected --coverage flag in args")
	}
}

// TestNodeRunnerBuildTestCommand_AllTypes tests different test types
func TestNodeRunnerBuildTestCommand_AllTypes(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		testType string
		config   *ServiceTestConfig
	}{
		{
			name:     "Unit tests",
			testType: "unit",
			config:   &ServiceTestConfig{Framework: "jest"},
		},
		{
			name:     "Integration tests",
			testType: "integration",
			config:   &ServiceTestConfig{Framework: "jest"},
		},
		{
			name:     "E2E tests",
			testType: "e2e",
			config:   &ServiceTestConfig{Framework: "jest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewNodeTestRunner(tmpDir, tt.config)
			command, args := runner.buildTestCommand(tt.testType, false)

			if command == "" {
				t.Error("Expected non-empty command")
			}
			if len(args) == 0 {
				t.Error("Expected non-empty args")
			}
		})
	}
}
