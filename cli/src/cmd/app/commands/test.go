package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
	Stream          bool
	NoStream        bool
	Timeout         time.Duration
	Save            bool
	NoSave          bool
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
	cmd.Flags().BoolVar(&opts.Stream, "stream", false, "Force streaming output (direct test output)")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "Force progress bar mode instead of streaming")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Per-service test timeout (e.g., 5m, 30s, 1h)")
	cmd.Flags().BoolVar(&opts.Save, "save", false, "Save auto-detected test config to azure.yaml without prompting")
	cmd.Flags().BoolVar(&opts.NoSave, "no-save", false, "Don't prompt to save auto-detected test config")

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

	// Validate mutually exclusive flags
	if opts.Stream && opts.NoStream {
		return fmt.Errorf("--stream and --no-stream are mutually exclusive")
	}
	if opts.Save && opts.NoSave {
		return fmt.Errorf("--save and --no-save are mutually exclusive")
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
		Timeout:           opts.Timeout,
	}

	// Create orchestrator
	orchestrator := testing.NewTestOrchestrator(config)

	// Load services from azure.yaml
	if err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath); err != nil {
		return fmt.Errorf("failed to load services: %w", err)
	}

	// Parse service filter
	var serviceFilter []string
	if opts.ServiceFilter != "" {
		serviceFilter = strings.Split(opts.ServiceFilter, ",")
		for i := range serviceFilter {
			serviceFilter[i] = strings.TrimSpace(serviceFilter[i])
		}
	}

	// Dry run - just show configuration and validation
	if opts.DryRun {
		return runTestDryRun(orchestrator, opts, serviceFilter)
	}

	// Set up progress callback for interactive output
	if !output.IsJSON() {
		orchestrator.SetProgressCallback(createProgressCallback())
	}

	// Watch mode
	if opts.Watch {
		return runWatchMode(orchestrator, opts.Type, serviceFilter)
	}

	// Execute tests with validation
	result, validations, err := orchestrator.ExecuteTestsWithValidation(opts.Type, serviceFilter)
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Get services for config save checking
	services := orchestrator.GetServices()

	// Display validation summary if not JSON
	if !output.IsJSON() {
		displayValidationSummary(validations)
	}

	// Check for auto-detected services and prompt to save config
	if !opts.NoSave {
		autoDetected := testing.GetAutoDetectedServices(validations, services)
		if len(autoDetected) > 0 {
			if err := promptSaveTestConfig(opts, azureYamlPath, validations, services, autoDetected); err != nil {
				// Log warning but don't fail the command
				output.Warning("Failed to save test config: %v", err)
			}
		}
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

// runTestDryRun shows configuration and validation without running tests.
func runTestDryRun(orchestrator *testing.TestOrchestrator, opts *TestOptions, serviceFilter []string) error {
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
		output.Item("Timeout: %s", opts.Timeout)
		if opts.Stream {
			output.Item("Output mode: streaming (forced)")
		} else if opts.NoStream {
			output.Item("Output mode: progress bars (forced)")
		} else {
			output.Item("Output mode: auto")
		}
		output.Newline()
	}

	// Validate services
	services := orchestrator.GetServices()
	if len(serviceFilter) > 0 {
		filtered := make([]testing.ServiceInfo, 0)
		filterMap := make(map[string]bool)
		for _, name := range serviceFilter {
			filterMap[strings.TrimSpace(name)] = true
		}
		for _, svc := range services {
			if filterMap[svc.Name] {
				filtered = append(filtered, svc)
			}
		}
		services = filtered
	}

	validations := testing.ValidateServices(services)
	displayValidationSummary(validations)

	return nil
}

// createProgressCallback creates a callback function for progress updates.
func createProgressCallback() testing.ProgressCallback {
	var mu sync.Mutex
	currentService := ""

	return func(event testing.ProgressEvent) {
		mu.Lock()
		defer mu.Unlock()

		switch event.Type {
		case testing.ProgressEventValidationStart:
			output.Step("🔍", "Analyzing services...")

		case testing.ProgressEventServiceValidated:
			// Validation details are shown in displayValidationSummary

		case testing.ProgressEventValidationComplete:
			// Summary shown separately

		case testing.ProgressEventTestStart:
			currentService = event.Service
			framework := event.Framework
			if framework == "" {
				framework = "tests"
			}
			output.Step("🧪", "Running tests...")
			output.Item("▸ %s (%s) - Running...", event.Service, framework)

		case testing.ProgressEventTestComplete:
			// Test completed - result will be shown in displayTestResults
			_ = currentService // Used for tracking state

		case testing.ProgressEventServiceSkipped:
			// Skipped services are shown in displayValidationSummary
		}
	}
}

// displayValidationSummary displays the validation results summary.
func displayValidationSummary(validations []testing.ServiceValidation) {
	if output.IsJSON() || len(validations) == 0 {
		return
	}

	testable := testing.GetTestableServices(validations)
	skipped := testing.GetSkippedServices(validations)

	output.Newline()

	// Show each service's validation status
	for _, v := range validations {
		if v.CanTest {
			testFilesInfo := ""
			if v.TestFiles > 0 {
				testFilesInfo = fmt.Sprintf(" (%d test files)", v.TestFiles)
			}
			output.ItemSuccess("%s: %s detected%s", v.Name, v.Framework, testFilesInfo)
		} else {
			output.ItemWarning("%s: %s (skipping)", v.Name, v.SkipReason)
		}
	}

	output.Newline()
	if len(skipped) > 0 {
		output.Info("Found %d testable services (%d skipped)", len(testable), len(skipped))
	} else {
		output.Info("Found %d testable services", len(testable))
	}
	output.Newline()
}

// promptSaveTestConfig prompts the user to save auto-detected test config to azure.yaml.
func promptSaveTestConfig(opts *TestOptions, azureYamlPath string, validations []testing.ServiceValidation, services []testing.ServiceInfo, autoDetected []testing.ServiceValidation) error {
	// If --save flag is set, save without prompting
	if opts.Save {
		if err := testing.SaveTestConfigToAzureYaml(azureYamlPath, validations, services); err != nil {
			return err
		}
		if !output.IsJSON() {
			output.Success("Test configuration saved to azure.yaml")
		}
		return nil
	}

	// If not running in TTY (non-interactive), skip prompting
	if !testing.IsTTY() || output.IsJSON() {
		return nil
	}

	// Display the discovered config
	output.Newline()
	output.Section("💾", "Auto-detected test configuration")
	for _, v := range autoDetected {
		output.Item("%s: %s", v.Name, v.Framework)
	}
	output.Newline()

	// Prompt to save
	if output.Confirm("Save test configuration to azure.yaml?") {
		if err := testing.SaveTestConfigToAzureYaml(azureYamlPath, validations, services); err != nil {
			return err
		}
		output.Success("Test configuration saved to azure.yaml")
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
