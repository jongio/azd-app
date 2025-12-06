// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

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
	Framework string
	// Unit test configuration
	Unit *TestTypeConfig
	// Integration test configuration
	Integration *TestTypeConfig
	// E2E test configuration
	E2E *TestTypeConfig
	// Coverage configuration
	Coverage *CoverageConfig
}

// TestTypeConfig represents configuration for a specific test type.
type TestTypeConfig struct {
	// Command is the command to run tests
	Command string
	// Pattern is the test file pattern (Node.js)
	Pattern string
	// Markers are pytest markers to filter tests (Python)
	Markers []string
	// Filter is the test filter expression (.NET)
	Filter string
	// Projects are test project paths (.NET)
	Projects []string
	// Setup commands to run before tests
	Setup []string
	// Teardown commands to run after tests
	Teardown []string
}

// CoverageConfig represents coverage configuration.
type CoverageConfig struct {
	// Enabled indicates whether to collect coverage
	Enabled bool
	// Tool is the coverage tool name
	Tool string
	// Threshold is the minimum coverage percentage for this service
	Threshold float64
	// Source is the source directory to measure coverage (Python)
	Source string
	// OutputFormat is the coverage output format
	OutputFormat string
	// Exclude are files/patterns to exclude from coverage
	Exclude []string
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
	Files map[string]*FileCoverage
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
