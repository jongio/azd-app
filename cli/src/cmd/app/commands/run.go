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

	"github.com/jongio/azd-app/cli/src/internal/browser"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/notifications"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/yamlutil"

	"github.com/spf13/cobra"
)

const (
	runtimeModeAzd    = "azd"
	runtimeModeAspire = "aspire"
)

var (
	runServiceFilter string
	runEnvFile       string
	runVerbose       bool
	runDryRun        bool
	runRuntime       string
	runWeb           bool
)

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "run",
		Short:        "Run the development environment (services from azure.yaml, Aspire, pnpm, or docker compose)",
		Long:         `Automatically detects and runs services defined in azure.yaml, or falls back to: Aspire (AppHost.cs), pnpm dev/start scripts, or docker compose from package.json`,
		SilenceUsage: true, // Don't print usage on errors - it makes error messages hard to read
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithServices(cmd.Context(), cmd, args)
		},
	}

	// Add flags for service orchestration
	cmd.Flags().StringVarP(&runServiceFilter, "service", "s", "", "Run specific service(s) only (comma-separated)")
	cmd.Flags().StringVar(&runEnvFile, "env-file", "", "Load environment variables from .env file")
	cmd.Flags().BoolVarP(&runVerbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Show what would be run without starting services")
	cmd.Flags().StringVar(&runRuntime, "runtime", runtimeModeAzd, "Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run)")
	cmd.Flags().BoolVarP(&runWeb, "web", "w", false, "Open dashboard in browser")

	return cmd
}

// runWithServices runs services from azure.yaml.
func runWithServices(ctx context.Context, _ *cobra.Command, _ []string) error {
	output.CommandHeader("run", "Run the development environment")
	if err := validateRuntimeMode(runRuntime); err != nil {
		return err
	}

	// Execute dependencies first (reqs -> deps -> run)
	// The orchestrator automatically sets orchestrated mode for dependencies
	if err := cmdOrchestrator.Run("run"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}

	return runServicesFromAzureYaml(ctx, azureYamlPath, runRuntime)
}

// validateRuntimeMode validates the runtime mode parameter.
func validateRuntimeMode(mode string) error {
	if mode != runtimeModeAzd && mode != runtimeModeAspire {
		return fmt.Errorf("invalid --runtime value: %s (must be '%s' or '%s')", mode, runtimeModeAzd, runtimeModeAspire)
	}
	return nil
}

// findAzureYaml locates the azure.yaml file.
func findAzureYaml() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return "", fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath == "" {
		return "", fmt.Errorf("azure.yaml not found - create one with 'services' section to define your development environment")
	}

	return azureYamlPath, nil
}

// runServicesFromAzureYaml orchestrates services defined in azure.yaml.
func runServicesFromAzureYaml(ctx context.Context, azureYamlPath string, runtimeMode string) error {
	azureYamlDir := filepath.Dir(azureYamlPath)

	// Aspire mode: run AppHost directly
	if runtimeMode == runtimeModeAspire {
		return runAspireMode(ctx, azureYamlDir)
	}

	// AZD mode: orchestrate services individually
	return runAzdMode(ctx, azureYamlPath, azureYamlDir)
}

// runAzdMode runs services in azd mode with individual service orchestration.
func runAzdMode(ctx context.Context, azureYamlPath, azureYamlDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Parse azure.yaml
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Execute prerun hook before starting services
	if err := executePrerunHook(azureYaml, azureYamlDir); err != nil {
		return err
	}

	// Check if there are services defined
	if !service.HasServices(azureYaml) {
		return showNoServicesMessage()
	}

	// Filter and detect services
	services := filterServices(azureYaml)
	if len(services) == 0 {
		return fmt.Errorf("no services match filter: %s", runServiceFilter)
	}

	runtimes, err := detectServiceRuntimes(services, azureYamlDir, runtimeModeAzd)
	if err != nil {
		return err
	}

	// Dry-run mode: show what would be executed
	if runDryRun {
		return showDryRun(runtimes)
	}

	// Execute and monitor services
	return executeAndMonitorServices(runtimes, cwd, azureYaml, azureYamlDir)
}

// showNoServicesMessage displays a message when no services are defined.
func showNoServicesMessage() error {
	output.Info("No services defined in azure.yaml")
	output.Item("Add a 'services' section to azure.yaml to use service orchestration")
	output.Item("or remove azure.yaml to use auto-detection (Aspire, pnpm, docker-compose)")
	return nil
}

// filterServices applies service filtering based on --service flag.
func filterServices(azureYaml *service.AzureYaml) map[string]service.Service {
	if runServiceFilter == "" {
		return azureYaml.Services
	}
	filterList := strings.Split(runServiceFilter, ",")
	return service.FilterServices(azureYaml, filterList)
}

// detectServiceRuntimes detects runtime information for all services.
func detectServiceRuntimes(services map[string]service.Service, azureYamlDir, runtimeMode string) ([]*service.ServiceRuntime, error) {
	usedPorts := make(map[int]bool)
	runtimes := make([]*service.ServiceRuntime, 0, len(services))

	// Find azure.yaml path for updates
	azureYamlPath := filepath.Join(azureYamlDir, "azure.yaml")

	for name, svc := range services {
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts, azureYamlDir, runtimeMode)
		if err != nil {
			return nil, fmt.Errorf("failed to detect runtime for service %s: %w", name, err)
		}
		usedPorts[runtime.Port] = true

		// If we auto-assigned a port and user wants to save it, update azure.yaml
		if runtime.ShouldUpdateAzureYaml {
			if err := yamlutil.UpdateServicePort(azureYamlPath, name, runtime.Port); err != nil {
				output.Warning("Failed to update azure.yaml for service %s: %v", name, err)
				output.Info("   Please manually add 'ports: [\"%d\"]' to service '%s' in azure.yaml", runtime.Port, name)
			} else {
				output.Success("Updated azure.yaml: Added ports: [\"%d\"] for service '%s'", runtime.Port, name)
			}
		}

		runtimes = append(runtimes, runtime)
	}

	return runtimes, nil
}

// executeAndMonitorServices starts services and monitors them until interrupted.
func executeAndMonitorServices(runtimes []*service.ServiceRuntime, cwd string, azureYaml *service.AzureYaml, azureYamlDir string) error {
	// Create logger
	logger := service.NewServiceLogger(runVerbose)
	logger.LogStartup(len(runtimes))

	// Load environment variables
	envVars, err := loadEnvironmentVariables()
	if err != nil {
		return err
	}

	// Orchestrate services
	result, err := service.OrchestrateServices(runtimes, envVars, logger)
	if err != nil {
		return fmt.Errorf("service orchestration failed: %w", err)
	}

	// Validate that all services are ready
	if err := service.ValidateOrchestration(result); err != nil {
		service.StopAllServices(result.Processes)
		return err
	}

	logger.LogReady()

	// Execute postrun hook after all services are ready
	if err := executePostrunHook(azureYaml, azureYamlDir); err != nil {
		output.Warning("Postrun hook failed but services are running: %v", err)
	}

	// Display Functions/Logic Apps endpoints if any were discovered
	if result.FunctionsParser != nil {
		// Give functions a moment to finish startup logging
		time.Sleep(2 * time.Second)

		for name, process := range result.Processes {
			if result.FunctionsParser.HasEndpoints(name) {
				result.FunctionsParser.DisplayEndpoints(name, process.Port)
			}
		}
	}

	// Start dashboard and wait for shutdown
	return monitorServicesUntilShutdown(result, cwd)
}

// loadEnvironmentVariables loads environment variables from --env-file if specified.
func loadEnvironmentVariables() (map[string]string, error) {
	if runEnvFile == "" {
		return make(map[string]string), nil
	}

	envVars, err := service.LoadDotEnv(runEnvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load env file: %w", err)
	}
	return envVars, nil
}

// monitorServicesUntilShutdown monitors all services with full process isolation.
//
// Process Isolation Design:
//   - Each service runs in an independent goroutine with panic recovery
//   - Service crashes/exits are logged but DON'T stop other services or the dashboard
//   - Only user signals (Ctrl+C/SIGTERM) trigger coordinated shutdown of all services
//   - Dashboard runs independently and survives individual service failures
//
// Lifecycle:
//  1. Start monitoring goroutines (one per service + dashboard)
//  2. Wait for user signal (Ctrl+C) or all services to naturally exit
//  3. On signal: initiate graceful shutdown with 10-second timeout
//  4. Stop all remaining services and dashboard
//
// This uses sync.WaitGroup (not errgroup) because we want all goroutines to complete
// independently rather than failing fast on first error.
func monitorServicesUntilShutdown(result *service.OrchestrationResult, cwd string) error {
	// Create context that cancels on SIGINT/SIGTERM only
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup
	dashboardServer := dashboard.GetServer(cwd)

	// Start notification manager for OS notifications on service issues
	notifMgr, err := notifications.NewNotificationManager(
		notifications.DefaultNotificationManagerConfig(cwd),
	)
	if err != nil {
		output.Warning("Notifications unavailable: %v", err)
	} else {
		notifMgr.Start()
		defer func() { _ = notifMgr.Stop() }()
		// Notifications enabled silently - no need to announce
	}

	// Start dashboard monitoring (passes notifMgr to set URL after dashboard starts)
	startDashboardMonitor(ctx, &wg, dashboardServer, notifMgr)

	// Start service process monitors
	startServiceMonitors(ctx, &wg, result.Processes, cwd)

	// Wait for signal (context cancellation) or all services to complete
	wg.Wait()

	// Perform cleanup shutdown
	return performGracefulShutdown(dashboardServer, result.Processes)
}

// startDashboardMonitor starts the dashboard server in a separate goroutine with panic recovery.
func startDashboardMonitor(ctx context.Context, wg *sync.WaitGroup, dashboardServer *dashboard.Server, notifMgr *notifications.NotificationManager) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				output.Error("Dashboard panic recovered: %v", r)
			}
		}()

		dashboardURL, err := dashboardServer.Start()
		if err != nil {
			output.Warning("Dashboard unavailable: %v", err)
			<-ctx.Done()
			return
		}

		// Set dashboard URL for clickable notifications
		if notifMgr != nil {
			notifMgr.SetDashboardURL(dashboardURL)
		}

		output.Plain("  Dashboard  %s", dashboardURL)
		output.Newline()

		// Launch browser after dashboard is ready (if enabled)
		browserLaunched := launchDashboardBrowser(dashboardURL)

		// Show compact hints on a single line
		if browserLaunched {
			output.Hint("Press Ctrl+C to stop")
		} else {
			output.Hint("Press Ctrl+C to stop", "--web to open browser")
		}

		// Block until context is cancelled
		<-ctx.Done()
	}()
}

// startServiceMonitors starts monitoring goroutines for all service processes.
func startServiceMonitors(ctx context.Context, wg *sync.WaitGroup, processes map[string]*service.ServiceProcess, projectDir string) {
	for name, process := range processes {
		if process.Process == nil {
			continue
		}
		wg.Add(1)
		go monitorServiceProcess(ctx, wg, name, process, projectDir)
	}
}

// performGracefulShutdown stops all services and dashboard with a timeout.
// Returns nil due to process isolation design - individual failures are logged but don't fail the command.
func performGracefulShutdown(dashboardServer *dashboard.Server, processes map[string]*service.ServiceProcess) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	output.Newline()
	output.Newline()
	output.Plain("Shutting down...")

	// Stop dashboard
	if stopErr := dashboardServer.Stop(); stopErr != nil {
		output.Warning("Failed to stop dashboard: %v", stopErr)
	}

	// Stop all services with graceful timeout
	if stopErr := shutdownAllServices(shutdownCtx, processes); stopErr != nil {
		output.Warning("Some services failed to stop cleanly: %v", stopErr)
	}

	output.Success("All services stopped")
	output.Newline()

	// Clean up port assignments on clean shutdown
	// Note: Port assignments are kept in the file for persistence across runs,
	// but we don't release them here to allow quick restarts with same ports.
	// Stale ports are cleaned up automatically after 7 days of inactivity.

	// Always return nil due to process isolation design:
	// Individual service crashes are logged but don't cause the run command to fail.
	// Only return errors for infrastructure issues (dashboard, shutdown timeout, etc.)
	return nil
}

// monitorServiceProcess monitors a single service process for exit or cancellation.
// This function runs in its own goroutine with panic recovery to ensure one service
// crash doesn't affect others (process isolation).
func monitorServiceProcess(ctx context.Context, wg *sync.WaitGroup, serviceName string, proc *service.ServiceProcess, projectDir string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			output.Error("Service monitor panic recovered for %s: %v", serviceName, r)
		}
	}()

	// Wait for either process exit or context cancellation
	// Use buffered channel to prevent goroutine leak
	type exitResult struct {
		exitCode int
		err      error
	}
	waitDone := make(chan exitResult, 1)
	go func() {
		state, err := proc.Process.Wait()
		if err != nil {
			waitDone <- exitResult{exitCode: -1, err: fmt.Errorf("service %s exited with error: %w", serviceName, err)}
			return
		}
		exitCode := state.ExitCode()
		if !state.Success() {
			waitDone <- exitResult{exitCode: exitCode, err: fmt.Errorf("service %s exited with code %d: %s", serviceName, exitCode, state.String())}
			return
		}
		waitDone <- exitResult{exitCode: 0, err: nil}
	}()

	select {
	case result := <-waitDone:
		// Service exited - record exit info in registry
		reg := registry.GetRegistry(projectDir)
		endTime := time.Now()

		// Always record exit code and end time for build/task mode tracking
		if regErr := reg.UpdateExitInfo(serviceName, result.exitCode, endTime); regErr != nil {
			output.Warning("Failed to update exit info for %s: %v", serviceName, regErr)
		}

		// Get service mode from registry to determine appropriate status
		entry, _ := reg.GetService(serviceName)
		mode := ""
		if entry != nil {
			mode = entry.Mode
		}

		if result.err != nil {
			// Update registry to trigger OS notification via state monitor
			if regErr := reg.UpdateStatus(serviceName, "error"); regErr != nil {
				output.Warning("Failed to update registry for %s: %v", serviceName, regErr)
			}

			// Show mode-appropriate error message
			switch mode {
			case service.ServiceModeBuild:
				output.Error("✗ Build failed: %s (exit code %d)", serviceName, result.exitCode)
			case service.ServiceModeTask:
				output.Error("✗ Task failed: %s (exit code %d)", serviceName, result.exitCode)
			default:
				output.Error("⚠️  %v", result.err)
				output.Warning("Service %s stopped. Other services continue running.", serviceName)
				output.Info("Press Ctrl+C to stop all services")
			}
		} else {
			// Update registry for clean exit
			// Use mode-appropriate status
			var status string
			switch mode {
			case service.ServiceModeBuild:
				status = "built"
				// Don't print message - build completion is expected, status visible in dashboard
			case service.ServiceModeTask:
				status = "completed"
				// Don't print message - task completion is expected, status visible in dashboard
			default:
				status = "stopped"
				output.Info("Service %s exited cleanly", serviceName)
			}
			if regErr := reg.UpdateStatus(serviceName, status); regErr != nil {
				output.Warning("Failed to update registry for %s: %v", serviceName, regErr)
			}
		}
		// Intentionally don't cancel context - other services should continue
	case <-ctx.Done():
		// Context cancelled by signal - proceed to graceful shutdown
		return
	}
}

// shutdownAllServices stops all services with graceful timeout.
// Runs all shutdowns in parallel goroutines and waits for all to complete.
// Returns aggregated errors from any services that failed to stop cleanly.
func shutdownAllServices(ctx context.Context, processes map[string]*service.ServiceProcess) error {
	var shutdownErrors []error
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, process := range processes {
		wg.Add(1)
		go func(serviceName string, proc *service.ServiceProcess) {
			defer wg.Done()

			if proc.Process == nil {
				return
			}

			// Determine timeout from context
			deadline, ok := ctx.Deadline()
			timeout := service.DefaultStopTimeout
			if ok {
				timeout = time.Until(deadline)
				if timeout < time.Second {
					timeout = time.Second
				}
			}

			if err := service.StopServiceGraceful(proc, timeout); err != nil {
				mu.Lock()
				shutdownErrors = append(shutdownErrors, fmt.Errorf("%s: %w", serviceName, err))
				mu.Unlock()
			}
		}(name, process)
	}

	wg.Wait()

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("failed to stop %d service(s): %w", len(shutdownErrors), errors.Join(shutdownErrors...))
	}
	return nil
}

// runAspireMode runs Aspire AppHost directly using dotnet run.
func runAspireMode(ctx context.Context, rootDir string) error {
	// Find Aspire AppHost project
	aspireProject, err := detector.FindAppHost(rootDir)
	if err != nil {
		return fmt.Errorf("failed to search for Aspire AppHost: %w", err)
	}

	if aspireProject == nil {
		return fmt.Errorf("no Aspire AppHost found - --runtime aspire requires an AppHost.cs or Program.cs file in a .csproj project")
	}

	output.Plain("Running Aspire in native mode")
	output.Item("Directory: %s", aspireProject.Dir)
	output.Item("Project: %s", aspireProject.ProjectFile)
	output.Newline()
	output.Plain("Aspire dashboard will start automatically")
	output.Newline()

	// Use executor to run dotnet with proper environment inheritance
	args := []string{"run", "--project", aspireProject.ProjectFile}

	output.Hint("Press Ctrl+C to stop")
	output.Newline()

	// Run dotnet and let it handle everything (inherits all azd env vars)
	return executor.StartCommand(ctx, "dotnet", args, aspireProject.Dir)
}

// showDryRun displays what would be executed without starting services.
func showDryRun(runtimes []*service.ServiceRuntime) error {
	output.Section("🔍", "Dry-run mode: Showing execution plan")

	for _, runtime := range runtimes {
		output.Newline()
		output.Info("%s", runtime.Name)
		output.Label("Language", runtime.Language)
		output.Label("Framework", runtime.Framework)
		output.Label("Port", fmt.Sprintf("%d", runtime.Port))
		output.Label("Directory", runtime.WorkingDir)
		output.Label("Command", fmt.Sprintf("%s %v", runtime.Command, runtime.Args))
	}

	return nil
}

// executePrerunHook executes the prerun hook if configured.
func executePrerunHook(azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPrerun(), "prerun", workingDir)
}

// executePostrunHook executes the postrun hook if configured.
func executePostrunHook(azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPostrun(), "postrun", workingDir)
}

// executeHook executes a lifecycle hook with the given name and configuration.
// This is a common helper function to avoid duplication between prerun and postrun hooks.
func executeHook(azureYaml *service.AzureYaml, hooks *service.Hooks, hook *service.Hook, hookName, workingDir string) error {
	if hooks == nil || hook == nil {
		return nil // No hook configured
	}

	// Convert service.Hook to executor.Hook
	convertedHook := convertHook(hook)
	config := executor.ResolveHookConfig(convertedHook)
	if config == nil {
		return nil
	}

	// Build environment variables for the hook
	// Following azd pattern: pass project directory and any other context
	hookEnvVars := buildHookEnvironmentVariables(azureYaml, workingDir)
	config.Env = hookEnvVars

	return executor.ExecuteHook(context.Background(), hookName, *config, workingDir)
}

// buildHookEnvironmentVariables builds environment variables to pass to hooks
// Following the pattern from azure/azure-dev
func buildHookEnvironmentVariables(azureYaml *service.AzureYaml, workingDir string) []string {
	envVars := []string{
		fmt.Sprintf("%s=%s", executor.EnvProjectDir, workingDir),
		fmt.Sprintf("%s=%s", executor.EnvProjectName, azureYaml.Name),
	}

	// Add count of services for context
	if azureYaml.Services != nil {
		envVars = append(envVars, fmt.Sprintf("%s=%d", executor.EnvServiceCount, len(azureYaml.Services)))
	}

	return envVars
}

// convertHook converts service.Hook to executor.Hook to avoid circular imports.
func convertHook(h *service.Hook) *executor.Hook {
	if h == nil {
		return nil
	}
	return executor.NewHook(
		h.Run,
		h.Shell,
		h.ContinueOnError,
		h.Interactive,
		convertPlatformHook(h.Windows),
		convertPlatformHook(h.Posix),
	)
}

// convertPlatformHook converts service.PlatformHook to executor.PlatformHook.
func convertPlatformHook(ph *service.PlatformHook) *executor.PlatformHook {
	if ph == nil {
		return nil
	}
	return executor.NewPlatformHook(
		ph.Run,
		ph.Shell,
		ph.ContinueOnError,
		ph.Interactive,
	)
}

// resolveBrowserTarget determines which browser target to use.
// Browser is OFF by default. Only opens if --web flag is specified.
func resolveBrowserTarget(_ *service.AzureYaml) browser.Target {
	if runWeb {
		return browser.TargetSystem
	}
	return browser.TargetNone
}

// launchDashboardBrowser launches the dashboard in the configured browser.
// Returns true if browser was launched, false if not (e.g., target is none).
func launchDashboardBrowser(dashboardURL string) bool {
	// Parse azure.yaml to get project config for browser preference
	azureYamlPath, err := findAzureYaml()
	var azureYaml *service.AzureYaml
	if err == nil {
		azureYaml, _ = service.ParseAzureYaml(azureYamlPath)
	}

	// Resolve browser target using priority system
	target := resolveBrowserTarget(azureYaml)

	// If target is none, don't launch
	if target == browser.TargetNone {
		return false
	}

	// Display launch message
	targetName := browser.GetTargetDisplayName(target)
	output.Plain("  Opening in %s...", targetName)

	// Launch browser (non-blocking)
	if err := browser.Launch(browser.LaunchOptions{
		URL:     dashboardURL,
		Target:  target,
		Timeout: 5 * time.Second,
	}); err != nil {
		output.Warning("Could not open browser: %v", err)
		output.Info("Dashboard available at: %s", dashboardURL)
	}
	return true
}
