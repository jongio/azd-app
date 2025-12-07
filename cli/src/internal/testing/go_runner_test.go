package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGoTestRunner(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{
		Framework: "gotest",
	})

	if runner.projectDir != "/test/dir" {
		t.Errorf("Expected projectDir '/test/dir', got '%s'", runner.projectDir)
	}

	if runner.config.Framework != "gotest" {
		t.Errorf("Expected framework 'gotest', got '%s'", runner.config.Framework)
	}
}

func TestGoTestRunner_buildTestCommand_Default(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})

	command, args := runner.buildTestCommand("all", false)

	if command != "go" {
		t.Errorf("Expected command 'go', got '%s'", command)
	}

	// Check for -v flag
	hasVerbose := false
	for _, arg := range args {
		if arg == "-v" {
			hasVerbose = true
			break
		}
	}
	if !hasVerbose {
		t.Error("Expected '-v' flag in args")
	}

	// Check for ./...
	hasRecursive := false
	for _, arg := range args {
		if arg == "./..." {
			hasRecursive = true
			break
		}
	}
	if !hasRecursive {
		t.Error("Expected './...' in args")
	}
}

func TestGoTestRunner_buildTestCommand_WithCoverage(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})

	_, args := runner.buildTestCommand("all", true)

	hasCover := false
	hasCoverProfile := false
	for _, arg := range args {
		if arg == "-cover" {
			hasCover = true
		}
		if arg == "-coverprofile=coverage.out" {
			hasCoverProfile = true
		}
	}

	if !hasCover {
		t.Error("Expected '-cover' flag in args")
	}
	if !hasCoverProfile {
		t.Error("Expected '-coverprofile=coverage.out' flag in args")
	}
}

func TestGoTestRunner_buildTestCommand_WithTestType(t *testing.T) {
	tests := []struct {
		name        string
		testType    string
		wantPattern string
	}{
		{
			name:        "unit tests",
			testType:    "unit",
			wantPattern: "^Test[^IE]|^Test$|^TestUnit",
		},
		{
			name:        "integration tests",
			testType:    "integration",
			wantPattern: "Integration",
		},
		{
			name:        "e2e tests",
			testType:    "e2e",
			wantPattern: "E2E|EndToEnd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
			_, args := runner.buildTestCommand(tt.testType, false)

			hasRun := false
			hasPattern := false
			for i, arg := range args {
				if arg == "-run" && i+1 < len(args) {
					hasRun = true
					if args[i+1] == tt.wantPattern {
						hasPattern = true
					}
				}
			}

			if !hasRun {
				t.Error("Expected '-run' flag in args")
			}
			if !hasPattern {
				t.Errorf("Expected pattern '%s' after -run flag", tt.wantPattern)
			}
		})
	}
}

func TestGoTestRunner_buildTestCommand_WithCustomCommand(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{
		Unit: &TestTypeConfig{
			Command: "go test -v -short ./...",
		},
	})

	command, args := runner.buildTestCommand("unit", false)

	if command != "go" {
		t.Errorf("Expected command 'go', got '%s'", command)
	}

	expectedArgs := []string{"test", "-v", "-short", "./..."}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, expected := range expectedArgs {
		if i >= len(args) || args[i] != expected {
			t.Errorf("Expected arg[%d] = '%s', got '%s'", i, expected, args[i])
		}
	}
}

func TestGoTestRunner_buildTestCommand_WithCustomPattern(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{
		Unit: &TestTypeConfig{
			Pattern: "^TestMyUnit",
		},
	})

	_, args := runner.buildTestCommand("unit", false)

	hasPattern := false
	for i, arg := range args {
		if arg == "-run" && i+1 < len(args) && args[i+1] == "^TestMyUnit" {
			hasPattern = true
			break
		}
	}

	if !hasPattern {
		t.Error("Expected custom pattern '^TestMyUnit' after -run flag")
	}
}

func TestGoTestRunner_parseTestOutput_Pass(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestSubtract
--- PASS: TestSubtract (0.01s)
PASS
ok      github.com/user/pkg    0.015s`

	runner.parseTestOutput(output, result)

	if result.Passed != 2 {
		t.Errorf("Expected 2 passed tests, got %d", result.Passed)
	}
	if result.Failed != 0 {
		t.Errorf("Expected 0 failed tests, got %d", result.Failed)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 total tests, got %d", result.Total)
	}
}

func TestGoTestRunner_parseTestOutput_Fail(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestSubtract
--- FAIL: TestSubtract (0.01s)
        main_test.go:15: Expected 5, got 4
FAIL
FAIL    github.com/user/pkg    0.015s`

	runner.parseTestOutput(output, result)

	if result.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", result.Passed)
	}
	if result.Failed != 1 {
		t.Errorf("Expected 1 failed test, got %d", result.Failed)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 total tests, got %d", result.Total)
	}
}

func TestGoTestRunner_parseTestOutput_Skip(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
=== RUN   TestSkipped
--- SKIP: TestSkipped (0.00s)
        main_test.go:20: Skipping test
PASS
ok      github.com/user/pkg    0.010s`

	runner.parseTestOutput(output, result)

	if result.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", result.Passed)
	}
	if result.Skipped != 1 {
		t.Errorf("Expected 1 skipped test, got %d", result.Skipped)
	}
	if result.Total != 2 {
		t.Errorf("Expected 2 total tests, got %d", result.Total)
	}
}

func TestGoTestRunner_parseTestOutput_Coverage(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
coverage: 85.7% of statements
ok      github.com/user/pkg    0.010s`

	runner.parseTestOutput(output, result)

	if result.Coverage == nil {
		t.Fatal("Expected coverage data, got nil")
	}

	expectedCoverage := 85.7
	if result.Coverage.Lines.Percent != expectedCoverage {
		t.Errorf("Expected coverage %.1f%%, got %.1f%%", expectedCoverage, result.Coverage.Lines.Percent)
	}
}

func TestGoTestRunner_parseTestOutput_NoTests(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `?       github.com/user/pkg    [no test files]`

	runner.parseTestOutput(output, result)

	if result.Total != 0 {
		t.Errorf("Expected 0 total tests, got %d", result.Total)
	}
}

func TestGoTestRunner_parseTestOutput_BasicPass(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `PASS`

	runner.parseTestOutput(output, result)

	if result.Total != 1 {
		t.Errorf("Expected 1 total test for basic PASS, got %d", result.Total)
	}
	if result.Passed != 1 {
		t.Errorf("Expected 1 passed test for basic PASS, got %d", result.Passed)
	}
}

func TestGoTestRunner_parseTestOutput_BasicFail(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})
	result := &TestResult{}

	output := `FAIL`

	runner.parseTestOutput(output, result)

	if result.Total != 1 {
		t.Errorf("Expected 1 total test for basic FAIL, got %d", result.Total)
	}
	if result.Failed != 1 {
		t.Errorf("Expected 1 failed test for basic FAIL, got %d", result.Failed)
	}
}

func TestGoTestRunner_HasTests(t *testing.T) {
	// Create a temp directory with go.mod and test file
	tempDir := t.TempDir()

	// Create go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a test file
	testFilePath := filepath.Join(tempDir, "main_test.go")
	err = os.WriteFile(testFilePath, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	runner := NewGoTestRunner(tempDir, &ServiceTestConfig{})

	if !runner.HasTests() {
		t.Error("Expected HasTests() to return true")
	}
}

func TestGoTestRunner_HasTests_NoGoMod(t *testing.T) {
	tempDir := t.TempDir()

	runner := NewGoTestRunner(tempDir, &ServiceTestConfig{})

	if runner.HasTests() {
		t.Error("Expected HasTests() to return false without go.mod")
	}
}

func TestGoTestRunner_HasTests_NoTestFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create go.mod but no test files
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	runner := NewGoTestRunner(tempDir, &ServiceTestConfig{})

	if runner.HasTests() {
		t.Error("Expected HasTests() to return false without test files")
	}
}

func TestGoTestRunner_ParseCoverageProfile(t *testing.T) {
	tempDir := t.TempDir()
	profilePath := filepath.Join(tempDir, "coverage.out")

	// Create a sample coverage profile
	coverageData := `mode: set
github.com/user/pkg/main.go:10.2,12.15 3 1
github.com/user/pkg/main.go:14.2,18.15 5 0
github.com/user/pkg/util.go:5.2,10.15 6 1
`
	err := os.WriteFile(profilePath, []byte(coverageData), 0644)
	if err != nil {
		t.Fatalf("Failed to create coverage profile: %v", err)
	}

	runner := NewGoTestRunner(tempDir, &ServiceTestConfig{})
	coverage, err := runner.ParseCoverageProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse coverage profile: %v", err)
	}

	// Total statements: 3 + 5 + 6 = 14
	// Covered statements: 3 + 0 + 6 = 9
	// Coverage: 9/14 = 64.28%

	expectedTotal := 14
	expectedCovered := 9

	if coverage.Lines.Total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, coverage.Lines.Total)
	}
	if coverage.Lines.Covered != expectedCovered {
		t.Errorf("Expected covered %d, got %d", expectedCovered, coverage.Lines.Covered)
	}

	// Check files
	if len(coverage.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(coverage.Files))
	}
}

func TestGoTestRunner_parseCommand(t *testing.T) {
	runner := NewGoTestRunner("/test/dir", &ServiceTestConfig{})

	tests := []struct {
		name       string
		input      string
		wantCmd    string
		wantArgLen int
	}{
		{
			name:       "empty command",
			input:      "",
			wantCmd:    "go",
			wantArgLen: 3, // test, -v, ./...
		},
		{
			name:       "single command",
			input:      "go",
			wantCmd:    "go",
			wantArgLen: 0,
		},
		{
			name:       "full command",
			input:      "go test -v -short",
			wantCmd:    "go",
			wantArgLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, args := runner.parseCommand(tt.input)
			if cmd != tt.wantCmd {
				t.Errorf("Expected command '%s', got '%s'", tt.wantCmd, cmd)
			}
			if len(args) != tt.wantArgLen {
				t.Errorf("Expected %d args, got %d", tt.wantArgLen, len(args))
			}
		})
	}
}

func TestGoTestRunner_getTestPattern(t *testing.T) {
	tests := []struct {
		name        string
		testType    string
		config      *ServiceTestConfig
		wantPattern string
	}{
		{
			name:        "unit default",
			testType:    "unit",
			config:      &ServiceTestConfig{},
			wantPattern: "^Test[^IE]|^Test$|^TestUnit",
		},
		{
			name:     "unit custom",
			testType: "unit",
			config: &ServiceTestConfig{
				Unit: &TestTypeConfig{Pattern: "^TestCustomUnit"},
			},
			wantPattern: "^TestCustomUnit",
		},
		{
			name:        "integration default",
			testType:    "integration",
			config:      &ServiceTestConfig{},
			wantPattern: "Integration",
		},
		{
			name:        "e2e default",
			testType:    "e2e",
			config:      &ServiceTestConfig{},
			wantPattern: "E2E|EndToEnd",
		},
		{
			name:        "unknown type",
			testType:    "unknown",
			config:      &ServiceTestConfig{},
			wantPattern: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewGoTestRunner("/test/dir", tt.config)
			pattern := runner.getTestPattern(tt.testType)
			if pattern != tt.wantPattern {
				t.Errorf("Expected pattern '%s', got '%s'", tt.wantPattern, pattern)
			}
		})
	}
}
