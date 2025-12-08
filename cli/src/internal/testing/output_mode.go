// Package testing provides test execution and coverage aggregation for multi-language projects.
package testing

import "os"

// OutputMode represents the output display mode for test results.
type OutputMode int

const (
	// OutputModeStream shows direct streaming output without prefixes.
	// Best for single service or when user wants raw output.
	OutputModeStream OutputMode = iota

	// OutputModeStreamPrefixed shows streaming output with [service] prefix.
	// Best for multiple services running sequentially to distinguish output.
	OutputModeStreamPrefixed

	// OutputModeProgress shows progress bars for test execution.
	// Best for multiple services running in parallel.
	OutputModeProgress
)

// String returns the string representation of the OutputMode.
func (m OutputMode) String() string {
	switch m {
	case OutputModeStream:
		return "stream"
	case OutputModeStreamPrefixed:
		return "stream-prefixed"
	case OutputModeProgress:
		return "progress"
	default:
		return "unknown"
	}
}

// OutputModeOptions contains options for selecting output mode.
type OutputModeOptions struct {
	// ForceStream forces streaming output mode (--stream flag).
	ForceStream bool
	// ForceProgress forces progress bar mode (--no-stream flag).
	ForceProgress bool
	// Parallel indicates if tests run in parallel.
	Parallel bool
}

// SelectOutputMode determines the best output mode based on options and environment.
// Selection logic:
//   - CI/non-TTY → always OutputModeStream (with prefix if multiple services)
//   - --stream flag → OutputModeStream
//   - --no-stream flag → OutputModeProgress
//   - Single service → OutputModeStream (no prefix)
//   - Multiple + sequential (parallel=false) → OutputModeStreamPrefixed
//   - Multiple + parallel → OutputModeProgress
func SelectOutputMode(opts OutputModeOptions, serviceCount int, isTTY bool) OutputMode {
	// CI or non-TTY environments: use streaming (best for log capture)
	if !isTTY || isCI() {
		if serviceCount > 1 {
			return OutputModeStreamPrefixed
		}
		return OutputModeStream
	}

	// Explicit --stream flag: use streaming
	if opts.ForceStream {
		return OutputModeStream
	}

	// Explicit --no-stream flag: use progress
	if opts.ForceProgress {
		return OutputModeProgress
	}

	// Single service: streaming without prefix
	if serviceCount <= 1 {
		return OutputModeStream
	}

	// Multiple services running sequentially: streaming with prefix
	if !opts.Parallel {
		return OutputModeStreamPrefixed
	}

	// Multiple services running in parallel: progress bars
	return OutputModeProgress
}

// isCI detects if running in a CI environment.
func isCI() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
		"TEAMCITY_VERSION",
		"TF_BUILD", // Azure Pipelines
		"BUILDKITE",
		"CODEBUILD_BUILD_ID", // AWS CodeBuild
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}

// IsTTY checks if stdout is a terminal (TTY).
// This is used to determine if we can use interactive output like progress bars.
func IsTTY() bool {
	// Check if stdout is a terminal
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
