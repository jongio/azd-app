package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPythonTestRunner(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
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

func TestPythonRunnerBuildTestCommand_CustomCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
		Unit: &TestTypeConfig{
			Command: "pytest tests/unit",
		},
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", false)

	if command != "pytest" {
		t.Errorf("Expected command 'pytest', got '%s'", command)
	}
	if len(args) < 1 || args[0] != "tests/unit" {
		t.Errorf("Expected args 'tests/unit', got %v", args)
	}
}

func TestPythonRunnerBuildPytestCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildPytestCommand("unit", false)

	if command == "" {
		t.Error("Command should not be empty")
	}

	// Should include -m marker for test type
	foundMarker := false
	for i, arg := range args {
		if arg == "-m" && i+1 < len(args) && args[i+1] == "unit" {
			foundMarker = true
			break
		}
	}
	if !foundMarker {
		t.Error("Expected '-m unit' in args")
	}
}

func TestPythonRunnerBuildPytestCommand_WithCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	_, args := runner.buildPytestCommand("all", true)

	// Should include coverage flag
	found := false
	for _, arg := range args {
		if arg == "--cov" || arg == "--cov=." {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected coverage flag in args")
	}
}

func TestPythonRunnerParseCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewPythonTestRunner(tmpDir, config)

	tests := []struct {
		name            string
		command         string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:            "Simple command",
			command:         "pytest tests",
			expectedCommand: "pytest",
			expectedArgs:    []string{"tests"},
		},
		{
			name:            "Command with multiple args",
			command:         "pytest tests/unit -v --cov",
			expectedCommand: "pytest",
			expectedArgs:    []string{"tests/unit", "-v", "--cov"},
		},
		{
			name:            "Single word command",
			command:         "pytest",
			expectedCommand: "pytest",
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

func TestPythonRunnerParseTestOutput(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		framework      string
		output         string
		expectedPassed int
		expectedFailed int
		expectedTotal  int
	}{
		{
			name:      "Pytest output",
			framework: "pytest",
			output: `collected 5 items

tests/test_utils.py .....

====== 5 passed in 0.12s ======`,
			expectedPassed: 5,
			expectedFailed: 0,
			expectedTotal:  5,
		},
		{
			name:      "Pytest output with failures",
			framework: "pytest",
			output: `collected 5 items

tests/test_utils.py ..F.F

====== 3 passed, 2 failed in 0.12s ======`,
			expectedPassed: 3,
			expectedFailed: 2,
			expectedTotal:  5,
		},
		{
			name:      "Unittest output",
			framework: "unittest",
			output: `....
----------------------------------------------------------------------
Ran 4 tests in 0.001s

OK`,
			expectedPassed: 4,
			expectedFailed: 0,
			expectedTotal:  4,
		},
		{
			name:      "Unittest output with failures",
			framework: "unittest",
			output: `..F.
----------------------------------------------------------------------
Ran 4 tests in 0.001s

FAILED (failures=1)`,
			expectedPassed: 3,
			expectedFailed: 1,
			expectedTotal:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServiceTestConfig{Framework: tt.framework}
			runner := NewPythonTestRunner(tmpDir, config)
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

func TestDetectPythonPackageManager(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		setupFunc      func(string) error
		expectedResult string
	}{
		{
			name: "Detect uv",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "uv.lock"), []byte(""), 0644)
			},
			expectedResult: "uv",
		},
		{
			name: "Detect poetry",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte(""), 0644)
			},
			expectedResult: "poetry",
		},
		{
			name: "Default to pip",
			setupFunc: func(dir string) error {
				return nil
			},
			expectedResult: "pip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			result := detectPythonPackageManager(testDir)
			if result != tt.expectedResult {
				t.Errorf("Expected '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}

func TestPythonRunnerHasTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tests directory
	testsDir := filepath.Join(tmpDir, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatalf("Failed to create tests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "test_example.py"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	if !runner.HasTests() {
		t.Error("Expected HasTests to return true")
	}
}

func TestPythonRunnerHasTests_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	if runner.HasTests() {
		t.Error("Expected HasTests to return false for directory without tests")
	}
}
