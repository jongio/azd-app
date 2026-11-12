// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
)

type dotnetRunnerMode int

const (
	dotnetRunnerVSTest dotnetRunnerMode = iota
	dotnetRunnerMTPNative
	dotnetRunnerMTPMSBuild
)

const (
	testFilterAll           = "all"
	dotnetCommand           = "dotnet"
	testCategoryUnit        = "Category=Unit"
	testCategoryIntegration = "Category=Integration"
	testCategoryE2E         = "Category=E2E"
)

// DotnetTestRunner runs tests for .NET projects.
type DotnetTestRunner struct {
	projectDir string
	config     *ServiceTestConfig
}

// NewDotnetTestRunner creates a new .NET test runner.
func NewDotnetTestRunner(projectDir string, config *ServiceTestConfig) *DotnetTestRunner {
	return &DotnetTestRunner{
		projectDir: projectDir,
		config:     config,
	}
}

// RunTests executes tests for the .NET project.
func (r *DotnetTestRunner) RunTests(ctx context.Context, testType string, coverage bool) (*TestResult, error) {
	result := &TestResult{
		TestType: testType,
		Success:  false,
	}

	// Build test command
	command, args := r.buildTestCommand(testType, coverage)

	// Execute the command
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
func (r *DotnetTestRunner) buildTestCommand(testType string, coverage bool) (string, []string) {
	args := []string{"test"}

	// Handle nil config - skip explicit command checks
	if r.config != nil {
		// Check if explicit command is configured
		switch testType {
		case testTypeUnit:
			if r.config.Unit != nil && r.config.Unit.Command != "" {
				return r.parseCommand(r.config.Unit.Command)
			}
		case testTypeIntegration:
			if r.config.Integration != nil && r.config.Integration.Command != "" {
				return r.parseCommand(r.config.Integration.Command)
			}
		case testTypeE2E:
			if r.config.E2E != nil && r.config.E2E.Command != "" {
				return r.parseCommand(r.config.E2E.Command)
			}
		}
	}

	// Find test project(s) and select the single target project
	testProjects := r.findTestProjects()
	selectedProject := ""
	if len(testProjects) > 0 {
		selectedProject = testProjects[0]
	}

	var filter string
	if testType != testFilterAll {
		filter = r.getTestFilter(testType)
	}

	switch r.detectRunnerMode(selectedProject) {
	case dotnetRunnerMTPNative:
		if selectedProject != "" {
			args = append(args, "--project", selectedProject)
		}
		if filter != "" {
			args = append(args, "--filter", filter)
		}
		if coverage {
			args = append(args, "--coverage", "--coverage-output-format", "cobertura")
		}
		return dotnetCommand, args

	case dotnetRunnerMTPMSBuild:
		if selectedProject != "" {
			args = append(args, selectedProject)
		}
		if filter != "" || coverage {
			args = append(args, "--")
			if filter != "" {
				args = append(args, "--filter", filter)
			}
			if coverage {
				args = append(args, "--coverage", "--coverage-output-format", "cobertura")
			}
		}
		return dotnetCommand, args

	default:
		if selectedProject != "" {
			args = append(args, selectedProject)
		}
		if filter != "" {
			args = append(args, "--filter", filter)
		}
		if coverage {
			args = append(args, "--collect", "XPlat Code Coverage")
		}
		args = append(args, "--logger", "console;verbosity=normal")
		return dotnetCommand, args
	}
}

type dotnetGlobalJSON struct {
	Test *struct {
		Runner string `json:"runner"`
	} `json:"test"`
}

// detectRunnerMode determines whether the selected test project runs via
// native .NET 10 Microsoft.Testing.Platform (opted in via the nearest global.json),
// MSBuild MTP compatibility mode (project-level TestingPlatformDotnetTestSupport /
// UseMicrosoftTestingPlatformRunner), or classic VSTest.
func (r *DotnetTestRunner) detectRunnerMode(selectedProject string) dotnetRunnerMode {
	searchStart := r.projectDir
	if selectedProject != "" {
		searchStart = filepath.Dir(selectedProject)
	}
	if globalJSONSelectsMTP(searchStart) {
		return dotnetRunnerMTPNative
	}

	if selectedProject != "" && projectEnablesMTP(selectedProject) {
		return dotnetRunnerMTPMSBuild
	}

	return dotnetRunnerVSTest
}

// globalJSONSelectsMTP walks upward to the filesystem root and inspects ONLY
// the nearest global.json file, matching .NET SDK global.json discovery rules.
func globalJSONSelectsMTP(startDir string) bool {
	if strings.TrimSpace(startDir) == "" {
		return false
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		dir = filepath.Clean(startDir)
	}

	for {
		candidate := filepath.Join(dir, "global.json")
		if data, readErr := os.ReadFile(candidate); readErr == nil {
			var parsed dotnetGlobalJSON
			if jsonErr := json.Unmarshal(data, &parsed); jsonErr == nil && parsed.Test != nil {
				return strings.EqualFold(strings.TrimSpace(parsed.Test.Runner), "Microsoft.Testing.Platform")
			}
			// Nearest global.json exists without test.runner=Microsoft.Testing.Platform;
			// .NET does not merge ancestor global.json files.
			return false
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

func projectEnablesMTP(projectPath string) bool {
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "<usemicrosofttestingplatformrunner>true") ||
		strings.Contains(content, "<testingplatformdotnettestsupport>true")
}

// getTestFilter returns the test filter expression for the test type.
func (r *DotnetTestRunner) getTestFilter(testType string) string {
	var filter string

	switch testType {
	case testTypeUnit:
		if r.config != nil && r.config.Unit != nil && r.config.Unit.Filter != "" {
			filter = r.config.Unit.Filter
		} else {
			// Default unit test filter
			filter = testCategoryUnit
		}
	case testTypeIntegration:
		if r.config != nil && r.config.Integration != nil && r.config.Integration.Filter != "" {
			filter = r.config.Integration.Filter
		} else {
			// Default integration test filter
			filter = testCategoryIntegration
		}
	case testTypeE2E:
		if r.config != nil && r.config.E2E != nil && r.config.E2E.Filter != "" {
			filter = r.config.E2E.Filter
		} else {
			// Default E2E test filter
			filter = testCategoryE2E
		}
	}

	return filter
}

// findTestProjects finds test project files in the directory.
func (r *DotnetTestRunner) findTestProjects() []string {
	var testProjects []string

	// Check if config specifies test projects
	if r.config != nil && r.config.Unit != nil && len(r.config.Unit.Projects) > 0 {
		// Resolve paths relative to project directory
		for _, proj := range r.config.Unit.Projects {
			if !filepath.IsAbs(proj) {
				proj = filepath.Join(r.projectDir, proj)
			}
			testProjects = append(testProjects, proj)
		}
		return testProjects
	}

	// Find .csproj or .fsproj files with "Test" in the name
	projects, err := detector.FindDotnetProjects(r.projectDir)
	if err != nil {
		return testProjects
	}

	for _, proj := range projects {
		fileName := filepath.Base(proj.Path)
		// Check if it's a test project (contains "Test" or "Tests")
		if strings.Contains(strings.ToLower(fileName), "test") {
			testProjects = append(testProjects, proj.Path)
		}
	}

	return testProjects
}

// parseCommand parses a command string into command and args.
func (r *DotnetTestRunner) parseCommand(cmdStr string) (string, []string) {
	parts := ParseCommandString(cmdStr)
	if len(parts) == 0 {
		return dotnetCommand, []string{"test"}
	}
	if len(parts) == 1 {
		return parts[0], []string{}
	}
	return parts[0], parts[1:]
}

// parseTestOutput parses test output to extract results.
func (r *DotnetTestRunner) parseTestOutput(output string, result *TestResult) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse summary line
		// Example: "Passed!  - Failed:     0, Passed:     5, Skipped:     0, Total:     5, Duration: 123 ms"
		// Example: "Failed!  - Failed:     1, Passed:     4, Skipped:     0, Total:     5, Duration: 123 ms"
		if strings.HasPrefix(line, "Passed!") || strings.HasPrefix(line, "Failed!") {
			r.parseDotnetSummary(line, result)
		}

		// Also parse "Total tests:" line format
		// Example: "Total tests: 5"
		if strings.HasPrefix(line, "Total tests:") {
			r.parseTotalTests(line, result)
		}

		// Parse individual result lines (VSTest and Microsoft.Testing.Platform formats)
		// Example: "Passed:    5" or "succeeded: 5"
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(line, "Passed:") {
			if num := extractNumber(strings.TrimPrefix(line, "Passed:")); num >= 0 {
				result.Passed = num
			}
		} else if strings.HasPrefix(lowerLine, "succeeded:") {
			if num := extractNumber(line[len("succeeded:"):]); num >= 0 {
				result.Passed = num
			}
		}
		if strings.HasPrefix(lowerLine, "failed:") {
			if num := extractNumber(line[len("failed:"):]); num >= 0 {
				result.Failed = num
			}
		}
		if strings.HasPrefix(lowerLine, "skipped:") {
			if num := extractNumber(line[len("skipped:"):]); num >= 0 {
				result.Skipped = num
			}
		}
		if strings.HasPrefix(lowerLine, "total:") {
			if num := extractNumber(line[len("total:"):]); num >= 0 {
				result.Total = num
			}
		}
	}

	// Calculate total if not already set
	if result.Total == 0 && (result.Passed > 0 || result.Failed > 0) {
		result.Total = result.Passed + result.Failed + result.Skipped
	}
}

// parseDotnetSummary parses the .NET test summary line.
func (r *DotnetTestRunner) parseDotnetSummary(line string, result *TestResult) {
	// Extract counts using regex
	failedMatch := regexp.MustCompile(`Failed:\s*(\d+)`).FindStringSubmatch(line)
	if len(failedMatch) > 1 {
		if num, err := strconv.Atoi(failedMatch[1]); err == nil {
			result.Failed = num
		}
	}

	passedMatch := regexp.MustCompile(`Passed:\s*(\d+)`).FindStringSubmatch(line)
	if len(passedMatch) > 1 {
		if num, err := strconv.Atoi(passedMatch[1]); err == nil {
			result.Passed = num
		}
	}

	skippedMatch := regexp.MustCompile(`Skipped:\s*(\d+)`).FindStringSubmatch(line)
	if len(skippedMatch) > 1 {
		if num, err := strconv.Atoi(skippedMatch[1]); err == nil {
			result.Skipped = num
		}
	}

	totalMatch := regexp.MustCompile(`Total:\s*(\d+)`).FindStringSubmatch(line)
	if len(totalMatch) > 1 {
		if num, err := strconv.Atoi(totalMatch[1]); err == nil {
			result.Total = num
		}
	}

	// Extract duration
	durationMatch := regexp.MustCompile(`Duration:\s*([\d.]+)\s*ms`).FindStringSubmatch(line)
	if len(durationMatch) > 1 {
		if duration, err := strconv.ParseFloat(durationMatch[1], 64); err == nil {
			result.Duration = duration / 1000.0 // Convert ms to seconds
		}
	}

	// Also check for seconds
	durationSecMatch := regexp.MustCompile(`Duration:\s*([\d.]+)\s*s`).FindStringSubmatch(line)
	if len(durationSecMatch) > 1 {
		if duration, err := strconv.ParseFloat(durationSecMatch[1], 64); err == nil {
			result.Duration = duration
		}
	}
}

// parseTotalTests parses the "Total tests:" line.
func (r *DotnetTestRunner) parseTotalTests(line string, result *TestResult) {
	// Example: "Total tests: 5"
	parts := strings.Split(line, ":")
	if len(parts) > 1 {
		if num := extractNumber(parts[1]); num >= 0 {
			result.Total = num
		}
	}
}

// HasTests checks if the project has test files.
func (r *DotnetTestRunner) HasTests() bool {
	// Check for test projects
	testProjects := r.findTestProjects()
	return len(testProjects) > 0
}
