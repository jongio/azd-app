// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import "time"

// Coverage threshold constants for UI display.
const (
	// CoverageThresholdHigh is the percentage above which coverage is considered high (green)
	CoverageThresholdHigh = 80.0
	// CoverageThresholdMedium is the percentage above which coverage is considered medium (yellow)
	CoverageThresholdMedium = 50.0
	// MinCoverageThreshold is the minimum valid coverage threshold (0%)
	MinCoverageThreshold = 0.0
	// MaxCoverageThreshold is the maximum valid coverage threshold (100%)
	MaxCoverageThreshold = 100.0
)

// File watcher timing constants.
const (
	// DefaultPollInterval is the default interval for file system polling
	DefaultPollInterval = 500 * time.Millisecond
	// DefaultDebounceDelay is the default delay for debouncing file change events
	DefaultDebounceDelay = 300 * time.Millisecond
)

// File permission constants.
const (
	// DirPermissions is the default permission mode for directories
	DirPermissions = 0o755
	// FilePermissions is the default permission mode for files
	FilePermissions = 0o644
)

// TestConfig represents the global test configuration.
type TestConfig struct {
	// Parallel indicates whether to run tests for services in parallel
	Parallel bool
	// FailFast indicates whether to stop on first test failure
	FailFast bool
	// CoverageThreshold is the minimum coverage percentage required (0-100)
	CoverageThreshold float64
	// OutputDir is the directory for test reports and coverage
	OutputDir string
	// Verbose enables verbose test output
	Verbose bool
}

// ServiceTestConfig represents test configuration for a service.
type ServiceTestConfig struct {
	// Framework is the test framework name (jest, pytest, xunit, etc.)
	Framework string `yaml:"framework" json:"framework"`
	// Unit test configuration
	Unit *TestTypeConfig `yaml:"unit" json:"unit"`
	// Integration test configuration
	Integration *TestTypeConfig `yaml:"integration" json:"integration"`
	// E2E test configuration
	E2E *TestTypeConfig `yaml:"e2e" json:"e2e"`
	// Coverage configuration
	Coverage *CoverageConfig `yaml:"coverage" json:"coverage"`
}

// TestTypeConfig represents configuration for a specific test type.
type TestTypeConfig struct {
	// Command is the command to run tests
	Command string `yaml:"command" json:"command"`
	// Pattern is the test file pattern (Node.js)
	Pattern string `yaml:"pattern" json:"pattern"`
	// Markers are pytest markers to filter tests (Python)
	Markers []string `yaml:"markers" json:"markers"`
	// Filter is the test filter expression (.NET)
	Filter string `yaml:"filter" json:"filter"`
	// Projects are test project paths (.NET)
	Projects []string `yaml:"projects" json:"projects"`
	// Setup commands to run before tests
	Setup []string `yaml:"setup" json:"setup"`
	// Teardown commands to run after tests
	Teardown []string `yaml:"teardown" json:"teardown"`
}

// CoverageConfig represents coverage configuration.
type CoverageConfig struct {
	// Enabled indicates whether to collect coverage
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Tool is the coverage tool name
	Tool string `yaml:"tool" json:"tool"`
	// Threshold is the minimum coverage percentage for this service
	Threshold float64 `yaml:"threshold" json:"threshold"`
	// Source is the source directory to measure coverage (Python)
	Source string `yaml:"source" json:"source"`
	// OutputFormat is the coverage output format
	OutputFormat string `yaml:"outputFormat" json:"outputFormat"`
	// Exclude are files/patterns to exclude from coverage
	Exclude []string `yaml:"exclude" json:"exclude"`
}

// TestResult represents the result of running tests for a service.
type TestResult struct {
	// Service name
	Service string
	// TestType is the type of test (unit, integration, e2e)
	TestType string
	// Passed is the number of passed tests
	Passed int
	// Failed is the number of failed tests
	Failed int
	// Skipped is the number of skipped tests
	Skipped int
	// Total is the total number of tests
	Total int
	// Duration is the test execution time in seconds
	Duration float64
	// Failures contains details of failed tests
	Failures []TestFailure
	// Coverage data (if coverage was enabled)
	Coverage *CoverageData
	// Success indicates whether all tests passed
	Success bool
	// Error message if test execution failed
	Error string
}

// TestFailure represents a single test failure.
type TestFailure struct {
	// Name is the test name
	Name string
	// Message is the failure message
	Message string
	// StackTrace is the failure stack trace
	StackTrace string
	// File is the file where the test failed
	File string
	// Line is the line number where the test failed
	Line int
}

// CoverageData represents coverage data for a service.
type CoverageData struct {
	// Lines coverage metric
	Lines CoverageMetric
	// Branches coverage metric
	Branches CoverageMetric
	// Functions coverage metric
	Functions CoverageMetric
	// Files contains per-file coverage data
	Files []*FileCoverage
}

// CoverageMetric represents a coverage metric.
type CoverageMetric struct {
	// Covered is the number of covered items
	Covered int
	// Total is the total number of items
	Total int
	// Percent is the coverage percentage
	Percent float64
}

// FileCoverage represents coverage for a single file.
type FileCoverage struct {
	// Path is the file path
	Path string
	// Lines coverage metric
	Lines CoverageMetric
	// Branches coverage metric
	Branches CoverageMetric
	// Functions coverage metric
	Functions CoverageMetric
	// CoveredLines are the line numbers that are covered
	CoveredLines []int
	// LineHits maps line numbers to the number of times they were executed
	LineHits map[int]int
}

// AggregateResult represents aggregated test results from all services.
type AggregateResult struct {
	// Services contains results for each service
	Services []*TestResult
	// Passed is the total number of passed tests
	Passed int
	// Failed is the total number of failed tests
	Failed int
	// Skipped is the total number of skipped tests
	Skipped int
	// Total is the total number of tests
	Total int
	// Duration is the total test execution time
	Duration float64
	// Coverage is the aggregated coverage data
	Coverage *AggregateCoverage
	// Success indicates whether all tests passed
	Success bool
	// Error message if test execution failed
	Error string
}

// AggregateCoverage represents aggregated coverage across all services.
type AggregateCoverage struct {
	// Services maps service name to coverage data
	Services map[string]*CoverageData
	// Aggregate is the combined coverage across all services
	Aggregate *CoverageData
	// Threshold is the required coverage threshold
	Threshold float64
	// Met indicates whether the threshold was met
	Met bool
}

// ParseCommandString parses a command string into command and args.
// Handles quoted strings to preserve arguments with spaces.
func ParseCommandString(cmdStr string) []string {
	var parts []string
	var current []rune
	inQuote := false
	quoteChar := rune(0)

	for _, c := range cmdStr {
		switch {
		case (c == '"' || c == '\'') && !inQuote:
			// Start of quoted section
			inQuote = true
			quoteChar = c
		case c == quoteChar && inQuote:
			// End of quoted section
			inQuote = false
			quoteChar = 0
		case c == ' ' && !inQuote:
			// Space outside quotes - end of argument
			if len(current) > 0 {
				parts = append(parts, string(current))
				current = current[:0]
			}
		default:
			current = append(current, c)
		}
	}

	// Add the last argument
	if len(current) > 0 {
		parts = append(parts, string(current))
	}

	return parts
}
