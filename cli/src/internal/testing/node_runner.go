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

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
)

// NodeTestRunner runs tests for Node.js projects.
type NodeTestRunner struct {
	projectDir     string
	config         *ServiceTestConfig
	packageManager string
}

// NewNodeTestRunner creates a new Node.js test runner.
func NewNodeTestRunner(projectDir string, config *ServiceTestConfig) *NodeTestRunner {
	// Detect package manager
	packageManager := detector.DetectNodePackageManagerWithBoundary(projectDir, projectDir)
	if packageManager == "" {
		packageManager = "npm"
	}

	return &NodeTestRunner{
		projectDir:     projectDir,
		config:         config,
		packageManager: packageManager,
	}
}

// RunTests executes tests for the Node.js project.
func (r *NodeTestRunner) RunTests(testType string, coverage bool) (*TestResult, error) {
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
func (r *NodeTestRunner) buildTestCommand(testType string, coverage bool) (string, []string) {
	var args []string

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

	// Build default command based on framework
	switch r.config.Framework {
	case "jest":
		args = []string{"test"}
		if testType != "all" {
			args = append(args, "--", fmt.Sprintf("--testPathPattern=%s", testType))
		}
		if coverage {
			args = append(args, "--coverage")
		}

	case "vitest":
		args = []string{"test", "--run"}
		if testType != "all" {
			args = append(args, fmt.Sprintf("--testNamePattern=%s", testType))
		}
		if coverage {
			args = append(args, "--coverage")
		}

	case "mocha":
		args = []string{"test"}
		// Mocha typically uses different test files for different types
		if testType != "all" {
			args = append(args, fmt.Sprintf("test/%s/**/*.test.js", testType))
		}

	default:
		// Default to npm test
		args = []string{"test"}
	}

	return r.packageManager, args
}

// parseCommand parses a command string into command and args.
func (r *NodeTestRunner) parseCommand(cmdStr string) (string, []string) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return r.packageManager, []string{"test"}
	}
	if len(parts) == 1 {
		return parts[0], []string{}
	}
	return parts[0], parts[1:]
}

// parseTestOutput parses test output to extract results.
func (r *NodeTestRunner) parseTestOutput(output string, result *TestResult) {
	// Try to parse Jest/Vitest output format
	// Example: "Tests:  5 passed, 5 total"
	// Example: "Tests:  4 passed, 1 failed, 5 total"

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Jest/Vitest summary line
		if strings.HasPrefix(line, "Tests:") {
			r.parseJestSummary(line, result)
		}

		// Mocha summary
		if strings.Contains(line, "passing") {
			r.parseMochaSummary(line, result)
		}

		// Extract duration
		if strings.Contains(line, "Time:") {
			r.parseDuration(line, result)
		}
	}

	// If we couldn't parse anything, set defaults
	if result.Total == 0 {
		// Check for simple success/failure indicators
		if strings.Contains(output, "PASS") || strings.Contains(output, "✓") {
			result.Passed = 1
			result.Total = 1
		} else if strings.Contains(output, "FAIL") || strings.Contains(output, "✗") {
			result.Failed = 1
			result.Total = 1
		}
	}
}

// parseJestSummary parses Jest/Vitest summary line.
func (r *NodeTestRunner) parseJestSummary(line string, result *TestResult) {
	// Remove "Tests:" prefix
	line = strings.TrimPrefix(line, "Tests:")
	line = strings.TrimSpace(line)

	// Parse patterns like "5 passed, 5 total" or "4 passed, 1 failed, 5 total"
	parts := strings.Split(line, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "passed") {
			if num := extractNumber(part); num >= 0 {
				result.Passed = num
			}
		} else if strings.Contains(part, "failed") {
			if num := extractNumber(part); num >= 0 {
				result.Failed = num
			}
		} else if strings.Contains(part, "skipped") {
			if num := extractNumber(part); num >= 0 {
				result.Skipped = num
			}
		} else if strings.Contains(part, "total") {
			if num := extractNumber(part); num >= 0 {
				result.Total = num
			}
		}
	}
}

// parseMochaSummary parses Mocha summary line.
func (r *NodeTestRunner) parseMochaSummary(line string, result *TestResult) {
	// Example: "5 passing (10ms)"
	// Example: "4 passing (10ms), 1 failing"

	if strings.Contains(line, "passing") {
		if num := extractNumber(strings.Split(line, "passing")[0]); num >= 0 {
			result.Passed = num
		}
	}

	if strings.Contains(line, "failing") {
		if num := extractNumber(strings.Split(line, "failing")[0]); num >= 0 {
			result.Failed = num
		}
	}

	result.Total = result.Passed + result.Failed
}

// parseDuration parses duration from test output.
func (r *NodeTestRunner) parseDuration(line string, result *TestResult) {
	// Example: "Time: 2.456 s"
	re := regexp.MustCompile(`Time:\s*([\d.]+)\s*s`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Duration = duration
		}
	}
}

// extractNumber extracts the first number from a string.
func extractNumber(s string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(s)
	if match == "" {
		return -1
	}
	num, err := strconv.Atoi(match)
	if err != nil {
		return -1
	}
	return num
}

// HasTests checks if the project has test files.
func (r *NodeTestRunner) HasTests() bool {
	// Check for common test file patterns
	patterns := []string{
		"**/*.test.js",
		"**/*.test.ts",
		"**/*.spec.js",
		"**/*.spec.ts",
		"test/**/*.js",
		"tests/**/*.js",
		"__tests__/**/*.js",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(r.projectDir, pattern))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	// Check if package.json has test script
	packageJSON := filepath.Join(r.projectDir, "package.json")
	if data, err := os.ReadFile(packageJSON); err == nil {
		return strings.Contains(string(data), `"test"`)
	}

	return false
}
