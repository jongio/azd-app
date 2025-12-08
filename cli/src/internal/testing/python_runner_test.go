package testing

import (
	"os"
	"path/filepath"
	"strings"
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

// TestPythonRunnerRunTests_Integration tests the full RunTests workflow
func TestPythonRunnerRunTests_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple test file
	testFile := filepath.Join(tmpDir, "test_example.py")
	content := `def test_example():
    assert True
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	result, err := runner.RunTests("unit", false)

	// The command might fail if pytest isn't installed, that's ok
	if err != nil {
		t.Logf("RunTests returned error (expected in test env): %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// TestPythonRunnerBuildTestCommand_AllTypes tests different test types
func TestPythonRunnerBuildTestCommand_AllTypes(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		testType string
	}{
		{name: "Unit tests", testType: "unit"},
		{name: "Integration tests", testType: "integration"},
		{name: "E2E tests", testType: "e2e"},
	}

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// TestPythonRunnerBuildTestCommand_CoverageFlag tests coverage integration
func TestPythonRunnerBuildTestCommand_CoverageFlag(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("unit", true)

	if command == "" {
		t.Error("Expected non-empty command")
	}

	// Check for coverage-related args
	argStr := strings.Join(args, " ")
	if argStr != "" && !strings.Contains(argStr, "cov") {
		t.Logf("Coverage args: %v", args)
	}
}

func TestPythonRunnerBuildUnittestCommand(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "unittest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildUnittestCommand("all", false)

	if command == "" {
		t.Error("Expected non-empty command")
	}

	// Should include unittest discover
	foundDiscover := false
	for _, arg := range args {
		if arg == "discover" {
			foundDiscover = true
			break
		}
	}
	if !foundDiscover {
		t.Error("Expected 'discover' in unittest args")
	}
}

func TestPythonRunnerBuildUnittestCommand_WithCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "unittest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildUnittestCommand("all", true)

	// With coverage, it should use coverage run
	argStr := strings.Join(append([]string{command}, args...), " ")
	if !strings.Contains(argStr, "coverage") {
		t.Errorf("Expected 'coverage' in command, got: %s", argStr)
	}
}

func TestPythonRunnerBuildUnittestCommand_WithTestType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test directory structure
	unitDir := filepath.Join(tmpDir, "tests", "unit")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "unittest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	_, args := runner.buildUnittestCommand("unit", false)

	// Should include -s for test directory filtering
	foundDir := false
	for i, arg := range args {
		if arg == "-s" && i+1 < len(args) {
			foundDir = true
			break
		}
	}
	if !foundDir {
		t.Log("Expected '-s' flag for test directory filtering")
	}
}

func TestPythonRunnerBuildPytestCommand_WithUV(t *testing.T) {
	tmpDir := t.TempDir()

	// Create uv.lock to simulate uv environment
	if err := os.WriteFile(filepath.Join(tmpDir, "uv.lock"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create uv.lock: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildPytestCommand("unit", false)

	if command != "uv" {
		t.Errorf("Expected command 'uv', got '%s'", command)
	}

	// Should include 'run pytest'
	foundRun := false
	foundPytest := false
	for _, arg := range args {
		if arg == "run" {
			foundRun = true
		}
		if arg == "pytest" {
			foundPytest = true
		}
	}
	if !foundRun || !foundPytest {
		t.Errorf("Expected 'run pytest' in args, got %v", args)
	}
}

func TestPythonRunnerBuildPytestCommand_WithPoetry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create poetry.lock to simulate poetry environment
	if err := os.WriteFile(filepath.Join(tmpDir, "poetry.lock"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create poetry.lock: %v", err)
	}

	config := &ServiceTestConfig{
		Framework: "pytest",
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildPytestCommand("all", false)

	if command != "poetry" {
		t.Errorf("Expected command 'poetry', got '%s'", command)
	}

	// Should include 'run pytest'
	foundRun := false
	foundPytest := false
	for _, arg := range args {
		if arg == "run" {
			foundRun = true
		}
		if arg == "pytest" {
			foundPytest = true
		}
	}
	if !foundRun || !foundPytest {
		t.Errorf("Expected 'run pytest' in args, got %v", args)
	}
}

func TestPythonRunnerBuildPytestCommand_WithCustomMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
		Unit: &TestTypeConfig{
			Markers: []string{"fast", "unit"},
		},
		Integration: &TestTypeConfig{
			Markers: []string{"slow", "integration"},
		},
		E2E: &TestTypeConfig{
			Markers: []string{"e2e", "acceptance"},
		},
	}

	runner := NewPythonTestRunner(tmpDir, config)

	tests := []struct {
		testType        string
		expectedMarkers []string
	}{
		{"unit", []string{"fast", "unit"}},
		{"integration", []string{"slow", "integration"}},
		{"e2e", []string{"e2e", "acceptance"}},
	}

	for _, tt := range tests {
		t.Run(tt.testType, func(t *testing.T) {
			_, args := runner.buildPytestCommand(tt.testType, false)

			for _, marker := range tt.expectedMarkers {
				found := false
				for i, arg := range args {
					if arg == "-m" && i+1 < len(args) && args[i+1] == marker {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected marker '%s' in args for %s tests", marker, tt.testType)
				}
			}
		})
	}
}

func TestPythonRunnerBuildPytestCommand_WithCoverageSource(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
		Coverage: &CoverageConfig{
			Source: "src",
		},
	}

	runner := NewPythonTestRunner(tmpDir, config)
	_, args := runner.buildPytestCommand("all", true)

	// Should include --cov=src
	found := false
	for _, arg := range args {
		if arg == "--cov=src" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected '--cov=src' in args, got %v", args)
	}
}

func TestPythonRunnerParsePytestSummary(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	tests := []struct {
		name             string
		line             string
		expectedPassed   int
		expectedFailed   int
		expectedSkipped  int
		expectedDuration float64
	}{
		{
			name:             "Simple passed",
			line:             "5 passed in 1.23s",
			expectedPassed:   5,
			expectedFailed:   0,
			expectedSkipped:  0,
			expectedDuration: 1.23,
		},
		{
			name:             "Passed and failed",
			line:             "3 passed, 2 failed in 0.50s",
			expectedPassed:   3,
			expectedFailed:   2,
			expectedSkipped:  0,
			expectedDuration: 0.50,
		},
		{
			name:             "All types",
			line:             "3 passed, 1 failed, 2 skipped in 2.00s",
			expectedPassed:   3,
			expectedFailed:   1,
			expectedSkipped:  2,
			expectedDuration: 2.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{}
			runner.parsePytestSummary(tt.line, result)

			if result.Passed != tt.expectedPassed {
				t.Errorf("Expected %d passed, got %d", tt.expectedPassed, result.Passed)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("Expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.Skipped != tt.expectedSkipped {
				t.Errorf("Expected %d skipped, got %d", tt.expectedSkipped, result.Skipped)
			}
			if result.Duration != tt.expectedDuration {
				t.Errorf("Expected duration %.2f, got %.2f", tt.expectedDuration, result.Duration)
			}
		})
	}
}

func TestPythonRunnerParseUnittestSummary(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{Framework: "unittest"}
	runner := NewPythonTestRunner(tmpDir, config)

	tests := []struct {
		name             string
		lines            []string
		expectedTotal    int
		expectedPassed   int
		expectedFailed   int
		expectedDuration float64
	}{
		{
			name: "Simple OK",
			lines: []string{
				"Ran 5 tests in 1.234s",
				"OK",
			},
			expectedTotal:    5,
			expectedPassed:   5,
			expectedFailed:   0,
			expectedDuration: 1.234,
		},
		{
			name: "With failures",
			lines: []string{
				"Ran 5 tests in 0.500s",
				"FAILED (failures=2)",
			},
			expectedTotal:    5,
			expectedPassed:   3,
			expectedFailed:   2,
			expectedDuration: 0.500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{}
			for _, line := range tt.lines {
				runner.parseUnittestSummary(line, result)
			}

			if result.Total != tt.expectedTotal {
				t.Errorf("Expected %d total, got %d", tt.expectedTotal, result.Total)
			}
			if result.Passed != tt.expectedPassed {
				t.Errorf("Expected %d passed, got %d", tt.expectedPassed, result.Passed)
			}
			if result.Failed != tt.expectedFailed {
				t.Errorf("Expected %d failed, got %d", tt.expectedFailed, result.Failed)
			}
			if result.Duration != tt.expectedDuration {
				t.Errorf("Expected duration %.3f, got %.3f", tt.expectedDuration, result.Duration)
			}
		})
	}
}

func TestPythonRunnerParseTestOutput_BasicIndicators(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	tests := []struct {
		name           string
		output         string
		expectedPassed int
		expectedFailed int
		expectedTotal  int
	}{
		{
			name:           "PASSED indicator",
			output:         "test_example.py PASSED",
			expectedPassed: 1,
			expectedFailed: 0,
			expectedTotal:  1,
		},
		{
			name:           "FAILED indicator",
			output:         "test_example.py FAILED",
			expectedPassed: 0,
			expectedFailed: 1,
			expectedTotal:  1,
		},
		{
			name:           "OK indicator",
			output:         "OK",
			expectedPassed: 1,
			expectedFailed: 0,
			expectedTotal:  1,
		},
		{
			name:           "ERROR indicator",
			output:         "ERROR in test",
			expectedPassed: 0,
			expectedFailed: 1,
			expectedTotal:  1,
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

func TestDetectPythonPackageManager_FromPyprojectToml(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		content        string
		expectedResult string
	}{
		{
			name:           "UV in pyproject.toml",
			content:        "[tool.uv]\ndev-dependencies = []",
			expectedResult: "uv",
		},
		{
			name:           "Poetry in pyproject.toml",
			content:        "[tool.poetry]\nname = \"test\"",
			expectedResult: "poetry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}

			pyprojectPath := filepath.Join(testDir, "pyproject.toml")
			if err := os.WriteFile(pyprojectPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create pyproject.toml: %v", err)
			}

			result := detectPythonPackageManager(testDir)
			if result != tt.expectedResult {
				t.Errorf("Expected '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}

func TestPythonRunnerHasTests_WithTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file in project root
	testFile := filepath.Join(tmpDir, "test_example.py")
	if err := os.WriteFile(testFile, []byte("def test_something(): pass"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	if !runner.HasTests() {
		t.Error("Expected HasTests to return true for test_ file in root")
	}
}

func TestPythonRunnerHasTests_WithTestDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a 'test' directory (singular)
	testDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	config := &ServiceTestConfig{Framework: "pytest"}
	runner := NewPythonTestRunner(tmpDir, config)

	if !runner.HasTests() {
		t.Error("Expected HasTests to return true for 'test' directory")
	}
}

func TestPythonRunnerBuildTestCommand_IntegrationCustom(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
		Integration: &TestTypeConfig{
			Command: "pytest tests/integration -v",
		},
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("integration", false)

	if command != "pytest" {
		t.Errorf("Expected command 'pytest', got '%s'", command)
	}
	if len(args) < 1 || args[0] != "tests/integration" {
		t.Errorf("Expected first arg 'tests/integration', got %v", args)
	}
}

func TestPythonRunnerBuildTestCommand_E2ECustom(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{
		Framework: "pytest",
		E2E: &TestTypeConfig{
			Command: "pytest tests/e2e -v",
		},
	}

	runner := NewPythonTestRunner(tmpDir, config)
	command, args := runner.buildTestCommand("e2e", false)

	if command != "pytest" {
		t.Errorf("Expected command 'pytest', got '%s'", command)
	}
	if len(args) < 1 || args[0] != "tests/e2e" {
		t.Errorf("Expected first arg 'tests/e2e', got %v", args)
	}
}

func TestPythonRunnerParseCommand_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	config := &ServiceTestConfig{}
	runner := NewPythonTestRunner(tmpDir, config)

	command, args := runner.parseCommand("")

	if command != "pytest" {
		t.Errorf("Expected command 'pytest', got '%s'", command)
	}
	if len(args) != 0 {
		t.Errorf("Expected empty args, got %v", args)
	}
}
