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

var (
	testType            string
	testCoverage        bool
	testServiceFilter   string
	testWatch           bool
	testUpdateSnapshots bool
	testFailFast        bool
	testParallel        bool
	testThreshold       int
	testVerbose         bool
	testDryRun          bool
	testOutputFormat    string
	testOutputDir       string
)

// NewTestCommand creates the test command.
func NewTestCommand() *cobra.Command {
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
			return runTests(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&testType, "type", "t", "all", "Test type to run: unit, integration, e2e, or all")
	cmd.Flags().BoolVarP(&testCoverage, "coverage", "c", false, "Generate code coverage reports")
	cmd.Flags().StringVarP(&testServiceFilter, "service", "s", "", "Run tests for specific service(s) (comma-separated)")
	cmd.Flags().BoolVarP(&testWatch, "watch", "w", false, "Watch mode - re-run tests on file changes")
	cmd.Flags().BoolVarP(&testUpdateSnapshots, "update-snapshots", "u", false, "Update test snapshots")
	cmd.Flags().BoolVar(&testFailFast, "fail-fast", false, "Stop on first test failure")
	cmd.Flags().BoolVarP(&testParallel, "parallel", "p", true, "Run tests for services in parallel")
	cmd.Flags().IntVar(&testThreshold, "threshold", 0, "Minimum coverage threshold (0-100)")
	cmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Enable verbose test output")
	cmd.Flags().BoolVar(&testDryRun, "dry-run", false, "Show what would be tested without running tests")
	cmd.Flags().StringVar(&testOutputFormat, "output-format", "default", "Output format: default, json, junit, github")
	cmd.Flags().StringVar(&testOutputDir, "output-dir", "./test-results", "Directory for test reports and coverage")

	return cmd
}

// runTests executes tests for all services.
func runTests(_ *cobra.Command, _ []string) error {
	// Validate test type
	validTypes := map[string]bool{
		"unit":        true,
		"integration": true,
		"e2e":         true,
		"all":         true,
	}
	if !validTypes[testType] {
		return fmt.Errorf("invalid test type: %s (must be unit, integration, e2e, or all)", testType)
	}

	// Validate threshold
	if testThreshold < 0 || testThreshold > 100 {
		return fmt.Errorf("invalid coverage threshold: %d (must be between 0 and 100)", testThreshold)
	}

	// Validate output format
	validFormats := map[string]bool{
		"default": true,
		"json":    true,
		"junit":   true,
		"github":  true,
	}
	if !validFormats[testOutputFormat] {
		return fmt.Errorf("invalid output format: %s (must be default, json, junit, or github)", testOutputFormat)
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
		Parallel:          testParallel,
		FailFast:          testFailFast,
		CoverageThreshold: float64(testThreshold),
		OutputDir:         testOutputDir,
		Verbose:           testVerbose,
	}

	// Create orchestrator
	orchestrator := testing.NewTestOrchestrator(config)

	// Load services from azure.yaml
	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	if !output.IsJSON() {
		output.Step("🧪", "Running %s tests...", testType)
		if testDryRun {
			output.Item("Dry run mode - showing configuration")
		}
	}

	// Dry run - just show configuration
	if testDryRun {
		if !output.IsJSON() {
			output.Step("📋", "Test configuration:")
			output.Item("Type: %s", testType)
			output.Item("Coverage: %v", testCoverage)
			if testServiceFilter != "" {
				output.Item("Services: %s", testServiceFilter)
			}
			if testThreshold > 0 {
				output.Item("Coverage threshold: %d%%", testThreshold)
			}
			output.Item("Parallel: %v", testParallel)
			output.Item("Output format: %s", testOutputFormat)
			output.Item("Output directory: %s", testOutputDir)
		}
		return nil
	}

	// Parse service filter
	var serviceFilter []string
	if testServiceFilter != "" {
		serviceFilter = strings.Split(testServiceFilter, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}

	// Watch mode
	if testWatch {
		return runWatchMode(orchestrator, testType, serviceFilter)
	}

	// Execute tests
	result, err := orchestrator.ExecuteTests(testType, serviceFilter)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Display results
	displayTestResults(result)

	// Check if tests passed
	if !result.Success {
		return fmt.Errorf("tests failed")
	}

	if testCoverage && testThreshold > 0 {
		if result.Coverage != nil {
			overall := result.Coverage.Aggregate.Lines.Percent
			if overall < float64(testThreshold) {
				return fmt.Errorf("coverage %.1f%% is below threshold of %d%%", overall, testThreshold)
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
		// TODO: Output JSON format
		return
	}

	output.Section("📊", "Test Results")

	for _, svcResult := range result.Services {
		if svcResult.Success {
			output.Success("✓ %s: %d passed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Total, svcResult.Duration)
		} else {
			output.Error("✗ %s: %d passed, %d failed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Failed, svcResult.Total, svcResult.Duration)
			if svcResult.Error != "" {
				output.Item("Error: %s", svcResult.Error)
			}
		}
	}

	output.Section("━", "Summary")
	if result.Success {
		output.Success("✓ All tests passed!")
	} else {
		output.Error("✗ Tests failed")
	}
	output.Item("Total: %d passed, %d failed, %d skipped, %d total",
		result.Passed, result.Failed, result.Skipped, result.Total)
	output.Item("Duration: %.2fs", result.Duration)
}
