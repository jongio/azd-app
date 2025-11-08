// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// PythonTestRunner runs tests for Python projects.
type PythonTestRunner struct {
	projectDir     string
	config         *ServiceTestConfig
	packageManager string
}

// NewPythonTestRunner creates a new Python test runner.
func NewPythonTestRunner(projectDir string, config *ServiceTestConfig) *PythonTestRunner {
	// Detect package manager (uv, poetry, pip)
	packageManager := detectPythonPackageManager(projectDir)

	return &PythonTestRunner{
		projectDir:     projectDir,
		config:         config,
		packageManager: packageManager,
	}
}

// RunTests executes tests for the Python project.
func (r *PythonTestRunner) RunTests(testType string, coverage bool) (*TestResult, error) {
	result := &TestResult{
		TestType: testType,
		Success:  false,
	}

	// Build test command
	command, args := r.buildTestCommand(testType, coverage)

	// Execute the command
	ctx := context.Background()
	output, err := executor.RunCommandWithOutput(ctx, command, args, r.projectDir)
	
	// Parse the output to extract results
	r.parseTestOutput(string(output), result)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		// Don't return error if we got some results
		if result.Total > 0 {
			return result, nil
		}
		return result, fmt.Errorf("test execution failed: %w", err)
	}

	result.Success = result.Failed == 0
	return result, nil
}

// buildTestCommand builds the test command based on framework and options.
func (r *PythonTestRunner) buildTestCommand(testType string, coverage bool) (string, []string) {
	// Check if explicit command is configured
	switch testType {
	case "unit":
		if r.config.Unit != nil && r.config.Unit.Command != "" {
			return r.parseCommand(r.config.Unit.Command)
		}
	case "integration":
		if r.config.Integration != nil && r.config.Integration.Command != "" {
			return r.parseCommand(r.config.Integration.Command)
		}
	case "e2e":
		if r.config.E2E != nil && r.config.E2E.Command != "" {
			return r.parseCommand(r.config.E2E.Command)
		}
	}

	// Build default command based on framework and package manager
	switch r.config.Framework {
	case "pytest":
		return r.buildPytestCommand(testType, coverage)
	case "unittest":
		return r.buildUnittestCommand(testType, coverage)
	default:
		// Default to pytest
		return r.buildPytestCommand(testType, coverage)
	}
}

// buildPytestCommand builds a pytest command.
func (r *PythonTestRunner) buildPytestCommand(testType string, coverage bool) (string, []string) {
	var command string
	var args []string

	// Determine the command based on package manager
	switch r.packageManager {
	case "uv":
		command = "uv"
		args = []string{"run", "pytest"}
	case "poetry":
		command = "poetry"
		args = []string{"run", "pytest"}
	default:
		command = "pytest"
	}

	// Add markers for test type filtering
	if testType != "all" {
		// Check if config has markers
		var markers []string
		switch testType {
		case "unit":
			if r.config.Unit != nil && len(r.config.Unit.Markers) > 0 {
				markers = r.config.Unit.Markers
			} else {
				markers = []string{"unit"}
			}
		case "integration":
			if r.config.Integration != nil && len(r.config.Integration.Markers) > 0 {
				markers = r.config.Integration.Markers
			} else {
				markers = []string{"integration"}
			}
		case "e2e":
			if r.config.E2E != nil && len(r.config.E2E.Markers) > 0 {
				markers = r.config.E2E.Markers
			} else {
				markers = []string{"e2e"}
			}
		}

		// Add marker arguments
		for _, marker := range markers {
			args = append(args, "-m", marker)
		}
	}

	// Add coverage flag
	if coverage {
		args = append(args, "--cov")
		if r.config.Coverage != nil && r.config.Coverage.Source != "" {
			args = append(args, fmt.Sprintf("--cov=%s", r.config.Coverage.Source))
		}
	}

	// Add verbose flag for better output parsing
	args = append(args, "-v")

	return command, args
}

// buildUnittestCommand builds a unittest command.
func (r *PythonTestRunner) buildUnittestCommand(testType string, coverage bool) (string, []string) {
	var command string
	var args []string

	if coverage {
		// Use coverage.py with unittest
		switch r.packageManager {
		case "uv":
			command = "uv"
			args = []string{"run", "coverage", "run", "-m", "unittest", "discover"}
		case "poetry":
			command = "poetry"
			args = []string{"run", "coverage", "run", "-m", "unittest", "discover"}
		default:
			command = "coverage"
			args = []string{"run", "-m", "unittest", "discover"}
		}
	} else {
		// Run unittest directly
		switch r.packageManager {
		case "uv":
			command = "uv"
			args = []string{"run", "python", "-m", "unittest", "discover"}
		case "poetry":
			command = "poetry"
			args = []string{"run", "python", "-m", "unittest", "discover"}
		default:
			command = "python"
			args = []string{"-m", "unittest", "discover"}
		}
	}

	// Add test path if configured
	if testType != "all" {
		// Try to filter by directory
		testDir := fmt.Sprintf("tests/%s", testType)
		if _, err := os.Stat(filepath.Join(r.projectDir, testDir)); err == nil {
			args = append(args, "-s", testDir)
		}
	}

	return command, args
}

// parseCommand parses a command string into command and args.
func (r *PythonTestRunner) parseCommand(cmdStr string) (string, []string) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "pytest", []string{}
	}
	if len(parts) == 1 {
		return parts[0], []string{}
	}
	return parts[0], parts[1:]
}

// parseTestOutput parses test output to extract results.
func (r *PythonTestRunner) parseTestOutput(output string, result *TestResult) {
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse pytest summary line
		// Example: "5 passed in 1.23s"
		// Example: "4 passed, 1 failed in 1.23s"
		if r.config.Framework == "pytest" || r.config.Framework == "" {
			r.parsePytestSummary(line, result)
		}

		// Parse unittest summary
		// Example: "Ran 5 tests in 1.234s"
		if r.config.Framework == "unittest" {
			r.parseUnittestSummary(line, result)
		}
	}

	// If we couldn't parse anything, check for basic indicators
	if result.Total == 0 {
		if strings.Contains(output, "PASSED") || strings.Contains(output, "OK") {
			result.Passed = 1
			result.Total = 1
		} else if strings.Contains(output, "FAILED") || strings.Contains(output, "ERROR") {
			result.Failed = 1
			result.Total = 1
		}
	}
}

// parsePytestSummary parses pytest summary line.
func (r *PythonTestRunner) parsePytestSummary(line string, result *TestResult) {
	// Pattern: "5 passed in 1.23s"
	// Pattern: "4 passed, 1 failed in 1.23s"
	// Pattern: "3 passed, 1 failed, 1 skipped in 1.23s"

	// Extract duration first
	if strings.Contains(line, " in ") && strings.Contains(line, "s") {
		durationMatch := regexp.MustCompile(`in ([\d.]+)s`).FindStringSubmatch(line)
		if len(durationMatch) > 1 {
			if duration, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
				result.Duration = duration
			}
		}
	}

	// Parse test counts
	if strings.Contains(line, "passed") {
		if num := extractNumber(strings.Split(line, "passed")[0]); num >= 0 {
			result.Passed = num
		}
	}

	if strings.Contains(line, "failed") {
		if num := extractNumber(strings.Split(line, "failed")[0]); num >= 0 {
			result.Failed = num
		}
	}

	if strings.Contains(line, "skipped") {
		if num := extractNumber(strings.Split(line, "skipped")[0]); num >= 0 {
			result.Skipped = num
		}
	}

	// Calculate total
	if result.Passed > 0 || result.Failed > 0 {
		result.Total = result.Passed + result.Failed + result.Skipped
	}
}

// parseUnittestSummary parses unittest summary line.
func (r *PythonTestRunner) parseUnittestSummary(line string, result *TestResult) {
	// Pattern: "Ran 5 tests in 1.234s"
	if strings.HasPrefix(line, "Ran ") && strings.Contains(line, "test") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if num, err := strconv.Atoi(parts[1]); err == nil {
				result.Total = num
			}
		}

		// Extract duration
		if strings.Contains(line, " in ") {
			durationMatch := regexp.MustCompile(`in ([\d.]+)s`).FindStringSubmatch(line)
			if len(durationMatch) > 1 {
				if duration, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
					result.Duration = duration
				}
			}
		}
	}

	// Check for OK or FAILED
	if line == "OK" {
		if result.Total > 0 {
			result.Passed = result.Total
		}
	} else if strings.Contains(line, "FAILED") {
		// Pattern: "FAILED (failures=1)"
		failureMatch := regexp.MustCompile(`failures=(\d+)`).FindStringSubmatch(line)
		if len(failureMatch) > 1 {
			if num, err := strconv.Atoi(failureMatch[1]); err == nil {
				result.Failed = num
				result.Passed = result.Total - result.Failed
			}
		}
	}
}

// detectPythonPackageManager detects the Python package manager.
func detectPythonPackageManager(dir string) string {
	// Check for uv
	if _, err := os.Stat(filepath.Join(dir, "uv.lock")); err == nil {
		return "uv"
	}

	// Check for poetry
	if _, err := os.Stat(filepath.Join(dir, "poetry.lock")); err == nil {
		return "poetry"
	}

	// Check for pyproject.toml with uv or poetry
	pyprojectPath := filepath.Join(dir, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		content := string(data)
		if strings.Contains(content, "[tool.uv]") {
			return "uv"
		}
		if strings.Contains(content, "[tool.poetry]") {
			return "poetry"
		}
	}

	// Default to pip
	return "pip"
}

// HasTests checks if the project has test files.
func (r *PythonTestRunner) HasTests() bool {
	// Check for test directories
	testDirs := []string{"tests", "test"}
	for _, dir := range testDirs {
		testPath := filepath.Join(r.projectDir, dir)
		if info, err := os.Stat(testPath); err == nil && info.IsDir() {
			return true
		}
	}

	// Check for test files in project root
	files, err := os.ReadDir(r.projectDir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if !file.IsDir() {
			name := file.Name()
			if strings.HasPrefix(name, "test_") && strings.HasSuffix(name, ".py") {
				return true
			}
		}
	}

	return false
}
