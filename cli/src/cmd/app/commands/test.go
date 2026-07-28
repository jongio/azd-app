package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/testing"
	"github.com/jongio/azd-core/cliout"
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
	Environment     string
	Changed         bool
	ChangedBase     string
}

// NewTestCommand creates the test command.
func NewTestCommand() *cobra.Command {
	// Create options for this command invocation
	opts := &TestOptions{}

	commandOrchestrator := newCommandOrchestrator()

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests for all services with coverage aggregation",
		Long:  "Automatically detects and runs tests for Node.js (Jest/Vitest/Mocha), Python (pytest/unittest), and .NET (xUnit/NUnit/MSTest) projects with unified coverage reporting.\n\nUse --changed to only test services with files changed since --changed-base (default HEAD). It looks at staged, unstaged, and untracked files, maps each one to the service whose project directory contains it, and runs only those services. Combine it with --service to intersect the two filters.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Try to get the output flag from parent or self
			var formatValue string
			if flag := cmd.InheritedFlags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			} else if flag := cmd.Flags().Lookup("output"); flag != nil {
				formatValue = flag.Value.String()
			}
			if formatValue != "" {
				return cliout.SetFormat(formatValue)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(commandOrchestrator, opts)
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
	cmd.Flags().StringVarP(&opts.Environment, "environment", "e", "", "Target azd environment name (loads vars from .azure/<env>/.env)")
	cmd.Flags().BoolVar(&opts.Changed, "changed", false, "Only test services with files changed since --changed-base")
	cmd.Flags().StringVar(&opts.ChangedBase, "changed-base", "HEAD", "Git ref to compare against for --changed (e.g. HEAD, origin/main)")

	registerServiceFlagCompletion(cmd, "service")

	return cmd
}

// runTests executes tests for all services.
func runTests(commandOrchestrator *orchestrator.Orchestrator, opts *TestOptions) error {
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
		return errors.New("--stream and --no-stream are mutually exclusive")
	}
	if opts.Save && opts.NoSave {
		return errors.New("--save and --no-save are mutually exclusive")
	}

	// Execute dependencies first (reqs)
	if err := commandOrchestrator.Run("test"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	// Find azure.yaml
	azureYamlPath, err := testing.FindAzureYaml()
	if err != nil {
		return fmt.Errorf("azure.yaml not found: %w", err)
	}

	if azureYamlPath == "" {
		return errors.New("azure.yaml not found - create one to define services for testing")
	}

	// Load azd environment variables if --environment is specified
	if opts.Environment != "" {
		if err := loadAzdEnvironment(azureYamlPath, opts.Environment); err != nil {
			return fmt.Errorf("failed to load environment %q: %w", opts.Environment, err)
		}
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

	// --changed narrows the run to services whose files changed since the base
	// ref, intersected with any explicit --service filter.
	if opts.Changed {
		affected, err := changedServiceFilter(orchestrator.GetServices(), serviceFilter, opts.ChangedBase)
		if err != nil {
			return err
		}
		if len(affected) == 0 {
			if cliout.IsJSON() {
				return cliout.PrintJSON(map[string]any{"affected": []string{}, "tested": false})
			}
			cliout.Info("No services affected by changes since %s. Nothing to test.", opts.ChangedBase)
			return nil
		}
		serviceFilter = affected
		if !cliout.IsJSON() {
			cliout.Info("Testing services changed since %s: %s", opts.ChangedBase, strings.Join(affected, ", "))
		}
	}

	// Dry run - just show configuration and validation
	if opts.DryRun {
		return runTestDryRun(orchestrator, opts, serviceFilter)
	}

	// Set up progress callback for interactive output
	if !cliout.IsJSON() {
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
	if !cliout.IsJSON() {
		displayValidationSummary(validations)
	}

	// Check for auto-detected services and prompt to save config
	if !opts.NoSave {
		autoDetected := testing.GetAutoDetectedServices(validations, services)
		if len(autoDetected) > 0 {
			if err := promptSaveTestConfig(opts, azureYamlPath, validations, services, autoDetected); err != nil {
				// Log warning but don't fail the command
				cliout.Warning("Failed to save test config: %v", err)
			}
		}
	}

	// Display results
	displayTestResults(result)

	// Check if tests passed
	if !result.Success {
		return errors.New("tests failed")
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
	if !cliout.IsJSON() {
		cliout.Step("📋", "Test configuration:")
		cliout.Item("Type: %s", opts.Type)
		cliout.Item("Coverage: %v", opts.Coverage)
		if opts.ServiceFilter != "" {
			cliout.Item("Services: %s", opts.ServiceFilter)
		}
		if opts.Threshold > 0 {
			cliout.Item("Coverage threshold: %d%%", opts.Threshold)
		}
		cliout.Item("Parallel: %v", opts.Parallel)
		cliout.Item("Output format: %s", opts.OutputFormat)
		cliout.Item("Output directory: %s", opts.OutputDir)
		cliout.Item("Timeout: %s", opts.Timeout)
		if opts.Stream {
			cliout.Item("Output mode: streaming (forced)")
		} else if opts.NoStream {
			cliout.Item("Output mode: progress bars (forced)")
		} else {
			cliout.Item("Output mode: auto")
		}
		if opts.Environment != "" {
			cliout.Item("Environment: %s", opts.Environment)
		}
		cliout.Newline()
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

	validations := testing.ValidateServicesForType(services, opts.Type)
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
			cliout.Step("🔍", "Analyzing services...")

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
			cliout.Step("🧪", "Running tests...")
			cliout.Item("▸ %s (%s) - Running...", event.Service, framework)

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
	if cliout.IsJSON() || len(validations) == 0 {
		return
	}

	testable := testing.GetTestableServices(validations)
	skipped := testing.GetSkippedServices(validations)

	cliout.Newline()

	// Show each service's validation status
	for _, v := range validations {
		if !v.CanTest {
			cliout.ItemWarning("%s: %s (skipping)", v.Name, v.SkipReason)
			continue
		}
		// An explicit command wins over the configured framework, so report the
		// command that will actually run rather than claiming the framework was
		// detected.
		if v.Command != "" {
			cliout.ItemSuccess("%s: custom command (%s)", v.Name, v.Command)
			continue
		}

		testFilesInfo := ""
		if v.TestFiles > 0 {
			testFilesInfo = fmt.Sprintf(" (%d test files)", v.TestFiles)
		}
		cliout.ItemSuccess("%s: %s detected%s", v.Name, v.Framework, testFilesInfo)
	}

	cliout.Newline()
	if len(skipped) > 0 {
		cliout.Info("Found %d testable services (%d skipped)", len(testable), len(skipped))
	} else {
		cliout.Info("Found %d testable services", len(testable))
	}
	cliout.Newline()
}

// promptSaveTestConfig prompts the user to save auto-detected test config to azure.yaml.
func promptSaveTestConfig(opts *TestOptions, azureYamlPath string, validations []testing.ServiceValidation, services []testing.ServiceInfo, autoDetected []testing.ServiceValidation) error {
	// If --save flag is set, save without prompting
	if opts.Save {
		if err := testing.SaveTestConfigToAzureYaml(azureYamlPath, validations, services); err != nil {
			return err
		}
		if !cliout.IsJSON() {
			cliout.Success("Test configuration saved to azure.yaml")
		}
		return nil
	}

	// If not running in TTY (non-interactive), skip prompting
	if !testing.IsTTY() || cliout.IsJSON() {
		return nil
	}

	// Display the discovered config
	cliout.Newline()
	cliout.Section("💾", "Auto-detected test configuration")
	for _, v := range autoDetected {
		cliout.Item("%s: %s", v.Name, v.Framework)
	}
	cliout.Newline()

	// Prompt to save
	if cliout.Confirm("Save test configuration to azure.yaml?") {
		if err := testing.SaveTestConfigToAzureYaml(azureYamlPath, validations, services); err != nil {
			return err
		}
		cliout.Success("Test configuration saved to azure.yaml")
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
	if cliout.IsJSON() {
		_ = cliout.PrintJSON(result)
		return
	}

	cliout.Section("📊", "Test Results")

	for _, svcResult := range result.Services {
		if svcResult.Success {
			cliout.Success("%s: %d passed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Total, svcResult.Duration)
		} else {
			cliout.Error("%s: %d passed, %d failed, %d total (%.2fs)",
				svcResult.Service, svcResult.Passed, svcResult.Failed, svcResult.Total, svcResult.Duration)
			if svcResult.Error != "" {
				cliout.Item("Error: %s", svcResult.Error)
			}
		}
	}

	cliout.Section("━", "Summary")
	if result.Success {
		cliout.Success("All tests passed!")
	} else {
		cliout.Error("Tests failed")
	}
	cliout.Item("Total: %d passed, %d failed, %d skipped, %d total",
		result.Passed, result.Failed, result.Skipped, result.Total)
	cliout.Item("Duration: %.2fs", result.Duration)
}

// loadAzdEnvironment loads environment variables from an azd environment's .env file
// and sets them in the current process so child processes (test runners) inherit them.
// The env file is located at .azure/<envName>/.env relative to the project root.
func loadAzdEnvironment(azureYamlPath string, envName string) error {
	// Validate envName to prevent path traversal
	if strings.ContainsAny(envName, `/\`) || strings.Contains(envName, "..") {
		return fmt.Errorf("invalid environment name %q: must not contain path separators or '..'", envName)
	}

	projectDir := filepath.Dir(azureYamlPath)
	envFilePath := filepath.Join(projectDir, ".azure", envName, ".env")

	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("environment %q not found (expected file: %s)", envName, envFilePath)
	}

	envVars, err := service.LoadDotEnv(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to read environment file: %w", err)
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %q: %w", key, err)
		}
	}

	cliout.Info("Loaded %d environment variables from %q", len(envVars), envName)
	return nil
}
