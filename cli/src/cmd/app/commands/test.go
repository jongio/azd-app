package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/testing"
	"github.com/spf13/cobra"
)

// TestOptions holds the options for the test command.
// Using a struct instead of global variables for better testability and concurrency safety.
type TestOptions struct {
	Type            string
	Coverage        bool
	ServiceFilter   string
	Watch           bool
	UpdateSnapshots bool
	FailFast        bool
	Parallel        bool
	Threshold       int
	Verbose         bool
	DryRun          bool
	OutputFormat    string
	OutputDir       string
}

// NewTestCommand creates the test command.
func NewTestCommand() *cobra.Command {
	// Create options for this command invocation
	opts := &TestOptions{}

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests for all services with coverage aggregation",
		Long:  `Automatically detects and runs tests for Node.js (Jest/Vitest/Mocha), Python (pytest/unittest), and .NET (xUnit/NUnit/MSTest) projects with unified coverage reporting`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Try to get the output flag from parent or self
			var formatValue string
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			} else if flag := cmd.Flags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			}
			if formatValue != "" {
				return output.SetFormat(formatValue)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(opts)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&opts.Type, "type", "t", "all", "Test type to run: unit, integration, e2e, or all")
	cmd.Flags().BoolVarP(&opts.Coverage, "coverage", "c", false, "Generate code coverage reports")
	cmd.Flags().StringVarP(&opts.ServiceFilter, "service", "s", "", "Run tests for specific service(s) (comma-separated)")
	cmd.Flags().BoolVarP(&opts.Watch, "watch", "w", false, "Watch mode - re-run tests on file changes")
	cmd.Flags().BoolVarP(&opts.UpdateSnapshots, "update-snapshots", "u", false, "Update test snapshots")
	cmd.Flags().BoolVar(&opts.FailFast, "fail-fast", false, "Stop on first test failure")
	cmd.Flags().BoolVarP(&opts.Parallel, "parallel", "p", true, "Run tests for services in parallel")
	cmd.Flags().IntVar(&opts.Threshold, "threshold", 0, "Minimum coverage threshold (0-100)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose test output")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be tested without running tests")
	cmd.Flags().StringVar(&opts.OutputFormat, "output-format", "default", "Output format: default, json, junit, github")
	cmd.Flags().StringVar(&opts.OutputDir, "output-dir", "./test-results", "Directory for test reports and coverage")

	return cmd
}

// runTests executes tests for all services.
func runTests(opts *TestOptions) error {
	// Validate test type
	validTypes := map[string]bool{
		"unit":        true,
		"integration": true,
		"e2e":         true,
		"all":         true,
	}
	if !validTypes[opts.Type] {
		return fmt.Errorf("invalid test type: %s (must be unit, integration, e2e, or all)", opts.Type)
	}

	// Validate threshold
	if opts.Threshold < 0 || opts.Threshold > 100 {
		return fmt.Errorf("invalid coverage threshold: %d (must be between 0 and 100)", opts.Threshold)
	}

	// Validate output format
	validFormats := map[string]bool{
		"default": true,
		"json":    true,
		"junit":   true,
		"github":  true,
	}
	if !validFormats[opts.OutputFormat] {
		return fmt.Errorf("invalid output format: %s (must be default, json, junit, or github)", opts.OutputFormat)
	}

	// Execute dependencies first (reqs)
	if err := cmdOrchestrator.Run("test"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	// Find azure.yaml
	azureYamlPath, err := testing.FindAzureYaml()
	if err != nil {
		return fmt.Errorf("azure.yaml not found: %w", err)
	}

	if azureYamlPath == "" {
		return fmt.Errorf("azure.yaml not found - create one to define services for testing")
	}

	// Create test configuration
	config := &testing.TestConfig{
		Parallel:          opts.Parallel,
		FailFast:          opts.FailFast,
		CoverageThreshold: float64(opts.Threshold),
		OutputDir:         opts.OutputDir,
		Verbose:           opts.Verbose,
	}

	// Create orchestrator
	orchestrator := testing.NewTestOrchestrator(config)

	// Load services from azure.yaml
	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	if !output.IsJSON() {
		output.Step("🧪", "Running %s tests...", opts.Type)
		if opts.DryRun {
			output.Item("Dry run mode - showing configuration")
		}
	}

	// Dry run - just show configuration
	if opts.DryRun {
		if !output.IsJSON() {
			output.Step("📋", "Test configuration:")
			output.Item("Type: %s", opts.Type)
			output.Item("Coverage: %v", opts.Coverage)
			if opts.ServiceFilter != "" {
				output.Item("Services: %s", opts.ServiceFilter)
			}
			if opts.Threshold > 0 {
				output.Item("Coverage threshold: %d%%", opts.Threshold)
			}
			output.Item("Parallel: %v", opts.Parallel)
			output.Item("Output format: %s", opts.OutputFormat)
			output.Item("Output directory: %s", opts.OutputDir)
		}
		return nil
	}

	// Parse service filter
	var serviceFilter []string
	if opts.ServiceFilter != "" {
		serviceFilter = strings.Split(opts.ServiceFilter, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}

	// Watch mode
	if opts.Watch {
		return runWatchMode(orchestrator, opts.Type, serviceFilter)
	}

	// Execute tests
	result, err := orchestrator.ExecuteTests(opts.Type, serviceFilter)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Display results
	displayTestResults(result)

	// Check if tests passed
	if !result.Success {
		return fmt.Errorf("tests failed")
	}

	if opts.Coverage && opts.Threshold > 0 {
		if result.Coverage != nil && result.Coverage.Aggregate != nil {
			overall := result.Coverage.Aggregate.Lines.Percent
			if overall < float64(opts.Threshold) {
				return fmt.Errorf("coverage %.1f%% is below threshold of %d%%", overall, opts.Threshold)
			}
		}
	}

	return nil
}

// runWatchMode runs tests in watch mode
func runWatchMode(orchestrator *testing.TestOrchestrator, testType string, serviceFilter []string) error {
	// Get service paths to watch
	paths, err := orchestrator.GetServicePaths()
	if err != nil {
		return fmt.Errorf("failed to get service paths: %w", err)
	}

	// Create watcher
	watcher := testing.NewFileWatcher(paths)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Watch and run tests
	return watcher.Watch(ctx, func() error {
		result, err := orchestrator.ExecuteTests(testType, serviceFilter)
		if err != nil {
			// Don't fail in watch mode, just show error
			fmt.Printf("❌ Test execution failed: %v\n", err)
			return nil
		}

		displayTestResults(result)
		return nil
	})
}

// displayTestResults displays test results in the console.
func displayTestResults(result *testing.AggregateResult) {
	if output.IsJSON() {
		_ = output.PrintJSON(result)
		return
	}

	output.Section("📊", "Test Results")

	for _, svcResult := range result.Services {
		if svcResult.Success {
			output.Success("%s: %d passed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Total, svcResult.Duration)
		} else {
			output.Error("%s: %d passed, %d failed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Failed, svcResult.Total, svcResult.Duration)
			if svcResult.Error != "" {
				output.Item("Error: %s", svcResult.Error)
			}
		}
	}

	output.Section("━", "Summary")
	if result.Success {
		output.Success("All tests passed!")
	} else {
		output.Error("Tests failed")
	}
	output.Item("Total: %d passed, %d failed, %d skipped, %d total",
		result.Passed, result.Failed, result.Skipped, result.Total)
	output.Item("Duration: %.2fs", result.Duration)
}
