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

// GoTestRunner runs tests for Go projects.
type GoTestRunner struct {
	projectDir string
	config     *ServiceTestConfig
}

// NewGoTestRunner creates a new Go test runner.
func NewGoTestRunner(projectDir string, config *ServiceTestConfig) *GoTestRunner {
	return &GoTestRunner{
		projectDir: projectDir,
		config:     config,
	}
}

// RunTests executes tests for the Go project.
func (r *GoTestRunner) RunTests(testType string, coverage bool) (*TestResult, error) {
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

// buildTestCommand builds the test command based on options.
func (r *GoTestRunner) buildTestCommand(testType string, coverage bool) (string, []string) {
	args := []string{"test", "-v"}

	// Handle nil config - skip explicit command checks
	if r.config != nil {
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
	}

	// Add test pattern filter for test type
	if testType != "all" {
		pattern := r.getTestPattern(testType)
		if pattern != "" {
			args = append(args, "-run", pattern)
		}
	}

	// Add coverage flag
	if coverage {
		args = append(args, "-cover")
		// Also generate coverage profile for detailed analysis
		args = append(args, "-coverprofile=coverage.out")
	}

	// Run all packages recursively
	args = append(args, "./...")

	return "go", args
}

// getTestPattern returns the test pattern for filtering by type.
func (r *GoTestRunner) getTestPattern(testType string) string {
	var pattern string

	switch testType {
	case "unit":
		if r.config != nil && r.config.Unit != nil && r.config.Unit.Pattern != "" {
			pattern = r.config.Unit.Pattern
		} else {
			// Default: match tests that don't contain Integration or E2E
			pattern = "^Test[^IE]|^Test$|^TestUnit"
		}
	case "integration":
		if r.config != nil && r.config.Integration != nil && r.config.Integration.Pattern != "" {
			pattern = r.config.Integration.Pattern
		} else {
			// Default: match tests with Integration in name
			pattern = "Integration"
		}
	case "e2e":
		if r.config != nil && r.config.E2E != nil && r.config.E2E.Pattern != "" {
			pattern = r.config.E2E.Pattern
		} else {
			// Default: match tests with E2E or EndToEnd in name
			pattern = "E2E|EndToEnd"
		}
	}

	return pattern
}

// parseCommand parses a command string into command and args.
func (r *GoTestRunner) parseCommand(cmdStr string) (string, []string) {
	parts := ParseCommandString(cmdStr)
	if len(parts) == 0 {
		return "go", []string{"test", "-v", "./..."}
	}
	if len(parts) == 1 {
		return parts[0], []string{}
	}
	return parts[0], parts[1:]
}

// parseTestOutput parses test output to extract results.
func (r *GoTestRunner) parseTestOutput(output string, result *TestResult) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse individual test results
		// Example: "--- PASS: TestAdd (0.00s)"
		// Example: "--- FAIL: TestSubtract (0.00s)"
		// Example: "--- SKIP: TestDivide (0.00s)"
		if strings.HasPrefix(line, "--- PASS:") {
			result.Passed++
			result.Total++
			r.parseTestDuration(line, result)
		} else if strings.HasPrefix(line, "--- FAIL:") {
			result.Failed++
			result.Total++
			r.parseTestDuration(line, result)
		} else if strings.HasPrefix(line, "--- SKIP:") {
			result.Skipped++
			result.Total++
		}

		// Parse package summary
		// Example: "ok      github.com/user/pkg    0.123s"
		// Example: "FAIL    github.com/user/pkg    0.456s"
		// Example: "?       github.com/user/pkg    [no test files]"
		if strings.HasPrefix(line, "ok ") || strings.HasPrefix(line, "FAIL\t") || strings.HasPrefix(line, "ok\t") {
			r.parsePackageSummary(line, result)
		}

		// Parse coverage
		// Example: "coverage: 85.7% of statements"
		if strings.Contains(line, "coverage:") && strings.Contains(line, "% of statements") {
			r.parseCoverage(line, result)
		}
	}

	// If we couldn't parse anything, check for basic indicators
	if result.Total == 0 {
		if strings.Contains(output, "PASS") && !strings.Contains(output, "FAIL") {
			result.Passed = 1
			result.Total = 1
		} else if strings.Contains(output, "FAIL") {
			result.Failed = 1
			result.Total = 1
		}
	}
}

// parseTestDuration extracts duration from test result line.
func (r *GoTestRunner) parseTestDuration(line string, result *TestResult) {
	// Example: "--- PASS: TestAdd (0.00s)"
	re := regexp.MustCompile(`\(([\d.]+)s\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Duration += duration
		}
	}
}

// parsePackageSummary parses package summary line for duration.
func (r *GoTestRunner) parsePackageSummary(line string, result *TestResult) {
	// Example: "ok      github.com/user/pkg    0.123s"
	re := regexp.MustCompile(`([\d.]+)s$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) > 1 {
		if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// Only use package duration if we don't have individual test durations
			if result.Duration == 0 {
				result.Duration = duration
			}
		}
	}
}

// parseCoverage extracts coverage percentage from output.
func (r *GoTestRunner) parseCoverage(line string, result *TestResult) {
	// Example: "coverage: 85.7% of statements"
	re := regexp.MustCompile(`coverage:\s*([\d.]+)%`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// Initialize coverage if nil
			if result.Coverage == nil {
				result.Coverage = &CoverageData{}
			}
			result.Coverage.Lines.Percent = percent
		}
	}
}

// HasTests checks if the project has test files.
func (r *GoTestRunner) HasTests() bool {
	// Check for go.mod
	goMod := filepath.Join(r.projectDir, "go.mod")
	if _, err := os.Stat(goMod); err != nil {
		return false
	}

	// Check for *_test.go files recursively
	hasTests := false
	_ = filepath.Walk(r.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "_test.go") {
			hasTests = true
			return filepath.SkipAll
		}
		return nil
	})

	return hasTests
}

// ParseCoverageProfile parses a Go coverage profile file.
func (r *GoTestRunner) ParseCoverageProfile(profilePath string) (*CoverageData, error) {
	// #nosec G304 -- profilePath is from test execution output, not user input
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read coverage profile: %w", err)
	}

	// Use a map to track files during parsing
	fileMap := make(map[string]*FileCoverage)

	coverage := &CoverageData{
		Files: make([]*FileCoverage, 0),
	}

	totalStatements := 0
	coveredStatements := 0

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		// Skip the mode line
		if i == 0 && strings.HasPrefix(line, "mode:") {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse coverage line: file:startLine.startCol,endLine.endCol numStatements count
		// Example: github.com/user/pkg/file.go:10.2,12.15 3 1
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Extract file path (before the colon with line info)
		colonIdx := strings.LastIndex(parts[0], ":")
		if colonIdx == -1 {
			continue
		}
		filePath := parts[0][:colonIdx]

		// Extract line range
		lineInfo := parts[0][colonIdx+1:]
		startLine := parseStartLine(lineInfo)

		numStatements, _ := strconv.Atoi(parts[1])
		count, _ := strconv.Atoi(parts[2])

		totalStatements += numStatements
		if count > 0 {
			coveredStatements += numStatements
		}

		// Track per-file coverage
		if _, ok := fileMap[filePath]; !ok {
			fileMap[filePath] = &FileCoverage{
				Path:     filePath,
				LineHits: make(map[int]int),
			}
		}
		fileMap[filePath].Lines.Total += numStatements
		if count > 0 {
			fileMap[filePath].Lines.Covered += numStatements
		}

		// Track line hits
		if startLine > 0 {
			fileMap[filePath].LineHits[startLine] += count
		}
	}

	// Calculate percentages
	coverage.Lines.Total = totalStatements
	coverage.Lines.Covered = coveredStatements
	if totalStatements > 0 {
		coverage.Lines.Percent = float64(coveredStatements) / float64(totalStatements) * 100
	}

	// Calculate per-file percentages and convert map to slice
	for _, fileCov := range fileMap {
		if fileCov.Lines.Total > 0 {
			fileCov.Lines.Percent = float64(fileCov.Lines.Covered) / float64(fileCov.Lines.Total) * 100
		}
		coverage.Files = append(coverage.Files, fileCov)
	}

	return coverage, nil
}

// parseStartLine extracts the start line number from line info like "10.2,12.15"
func parseStartLine(lineInfo string) int {
	// Format: startLine.startCol,endLine.endCol
	dotIdx := strings.Index(lineInfo, ".")
	if dotIdx == -1 {
		return 0
	}
	lineStr := lineInfo[:dotIdx]
	line, _ := strconv.Atoi(lineStr)
	return line
}
