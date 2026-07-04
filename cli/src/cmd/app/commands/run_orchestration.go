package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/notifications"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/supervisor"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/registry"
)

// executeAndMonitorServices starts services and monitors them until interrupted.
func executeAndMonitorServices(ctx context.Context, runtimes []*service.ServiceRuntime, cwd string, azureYaml *service.AzureYaml, azureYamlDir string) error {
	// Create logger
	logger := service.NewServiceLogger(runVerbose)
	logger.LogStartup(len(runtimes))

	// Load environment variables
	envVars, err := loadEnvironmentVariables()
	if err != nil {
		return err
	}

	// Orchestrate services with dependency ordering
	result, err := service.OrchestrateServices(ctx, runtimes, azureYaml.Services, envVars, logger, runRestartContainers)
	if err != nil {
		return fmt.Errorf("service orchestration failed: %w", err)
	}

	// Validate that all services are ready
	if err := service.ValidateOrchestration(result); err != nil {
		service.StopAllServices(result.Processes)
		return err
	}

	// Display service URLs (local + custom + Azure endpoints/domains)
	serviceSummaries := buildServiceSummaries(cwd, azureYaml, result.Processes)
	logger.LogSummary(serviceSummaries)

	logger.LogReady()

	// Record per-service startup timing and surface regressions vs prior runs.
	recordStartupTimings(cwd, result)

	// Execute postrun hook after all services are ready
	if err := executePostrunHook(ctx, azureYaml, azureYamlDir); err != nil {
		cliout.Warning("Postrun hook failed but services are running: %v", err)
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
	return monitorServicesUntilShutdown(result, cwd, azureYaml, azureYamlDir)
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
func monitorServicesUntilShutdown(result *service.OrchestrationResult, cwd string, azureYaml *service.AzureYaml, azureYamlDir string) error {
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
		cliout.Warning("Notifications unavailable: %v", err)
	} else {
		notifMgr.Start()
		configureAutoRestartSupervisor(ctx, notifMgr, result.Processes, cwd)
		defer func() { _ = notifMgr.Stop() }()
	}

	// Start dashboard monitoring (passes notifMgr to set URL after dashboard starts)
	startDashboardMonitor(ctx, &wg, dashboardServer, notifMgr)

	// Start service process monitors
	startServiceMonitors(ctx, &wg, result.Processes, cwd)

	if azureYaml != nil {
		writeRunState(cwd, result, dashboardServer)
	}

	// Also cancel on remote shutdown request (from `azd app stop` in another terminal)
	go func() {
		select {
		case <-dashboardServer.ShutdownChan():
			cancel()
		case <-ctx.Done():
		}
	}()

	// Wait for signal (context cancellation) or all services to complete
	wg.Wait()

	// Perform cleanup shutdown with hooks
	return performGracefulShutdown(cwd, dashboardServer, result.Processes, azureYaml, azureYamlDir)
}

// startDashboardMonitor starts the dashboard server in a separate goroutine with panic recovery.
func startDashboardMonitor(ctx context.Context, wg *sync.WaitGroup, dashboardServer *dashboard.Server, notifMgr *notifications.NotificationManager) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				cliout.Error("Dashboard panic recovered: %v", r)
			}
		}()

		dashboardURL, err := dashboardServer.Start()
		if err != nil {
			cliout.Warning("Dashboard unavailable: %v", err)
			<-ctx.Done()
			return
		}

		// Set dashboard URL for clickable notifications
		if notifMgr != nil {
			notifMgr.SetDashboardURL(dashboardURL)
		}

		cliout.Plain("  Dashboard  %s", dashboardURL)
		cliout.Newline()

		// Launch browser after dashboard is ready (if enabled)
		browserLaunched := launchDashboardBrowser(dashboardURL)

		// Show compact hints on a single line
		if browserLaunched {
			cliout.Hint("Press Ctrl+C to stop")
		} else {
			cliout.Hint("Press Ctrl+C to stop", "--web to open browser")
		}

		// Block until context is canceled
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

func configureAutoRestartSupervisor(
	ctx context.Context,
	notifMgr *notifications.NotificationManager,
	processes map[string]*service.ServiceProcess,
	projectDir string,
) {
	if notifMgr == nil || len(processes) == 0 {
		return
	}

	policies := make(map[string]service.RestartPolicy, len(processes))
	for name, process := range processes {
		if process == nil {
			continue
		}
		policy := strings.ToLower(strings.TrimSpace(process.Runtime.Restart.Policy))
		if policy != service.RestartPolicyOnFailure && policy != service.RestartPolicyAlways {
			continue
		}
		policies[name] = process.Runtime.Restart
	}

	if len(policies) == 0 {
		return
	}

	controller, err := NewServiceController(projectDir)
	if err != nil {
		cliout.Warning("Auto-restart unavailable: %v", err)
		return
	}

	restartSupervisor := supervisor.New(ctx, policies, func(serviceName string) error {
		restartResult := controller.RestartService(ctx, serviceName)
		if restartResult == nil {
			return fmt.Errorf("failed to restart service '%s': no result returned", serviceName)
		}

		if !restartResult.Success {
			if restartResult.Error != "" {
				return fmt.Errorf("failed to restart service '%s': %s", serviceName, restartResult.Error)
			}
			return fmt.Errorf("failed to restart service '%s': %s", serviceName, restartResult.Message)
		}

		return nil
	})

	notifMgr.AddStateListener(restartSupervisor.OnStateTransition)
}

// performGracefulShutdown stops all services and dashboard with a timeout.
// Runs prestop/poststop hooks around service shutdown.
// Returns nil due to process isolation design - individual failures are logged but don't fail the command.
func performGracefulShutdown(projectDir string, dashboardServer *dashboard.Server, processes map[string]*service.ServiceProcess, azureYaml *service.AzureYaml, azureYamlDir string) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	cliout.Newline()
	cliout.Newline()
	cliout.Plain("Shutting down...")

	// Execute prestop hook
	if azureYaml != nil {
		if hookErr := executePrestopHook(shutdownCtx, azureYaml, azureYamlDir); hookErr != nil {
			cliout.Warning("Prestop hook failed: %v", hookErr)
		}
	}

	// Stop dashboard
	if stopErr := dashboardServer.Stop(); stopErr != nil {
		cliout.Warning("Failed to stop dashboard: %v", stopErr)
	}

	// Stop all services with graceful timeout
	if stopErr := shutdownAllServices(shutdownCtx, processes); stopErr != nil {
		cliout.Warning("Some services failed to stop cleanly: %v", stopErr)
	}

	if removeErr := runstate.Remove(projectDir); removeErr != nil {
		cliout.Warning("Failed to clear run state: %v", removeErr)
	}

	cliout.Success("All services stopped")

	// Execute poststop hook
	if azureYaml != nil {
		if hookErr := executePoststopHook(shutdownCtx, azureYaml, azureYamlDir); hookErr != nil {
			cliout.Warning("Poststop hook failed: %v", hookErr)
		}
	}

	cliout.Newline()

	// Clean up port assignments on clean shutdown
	// Note: Port assignments are kept in the file for persistence across runs,
	// but we don't release them here to allow quick restarts with same ports.
	// Stale ports are cleaned up automatically after 7 days of inactivity.

	// Always return nil due to process isolation design:
	// Individual service crashes are logged but don't cause the run command to fail.
	// Only return errors for infrastructure issues (dashboard, shutdown timeout, etc.)
	return nil
}

func writeRunState(projectDir string, result *service.OrchestrationResult, dashboardServer *dashboard.Server) {
	st := runstate.RunState{
		PID:       os.Getpid(),
		Services:  buildRunStateServices(result.Processes),
		StartTime: result.StartTime,
	}

	if st.StartTime.IsZero() {
		st.StartTime = time.Now()
	}

	st.DashboardURL = waitForDashboardURL(projectDir, dashboardServer, 10*time.Second)
	if err := runstate.Write(projectDir, st); err != nil {
		cliout.Warning("Failed to write run state: %v", err)
	}
}

func buildRunStateServices(processes map[string]*service.ServiceProcess) []runstate.ServiceState {
	names := make([]string, 0, len(processes))
	for name := range processes {
		names = append(names, name)
	}
	sort.Strings(names)

	services := make([]runstate.ServiceState, 0, len(names))
	for _, name := range names {
		process := processes[name]
		if process == nil {
			continue
		}

		serviceState := runstate.ServiceState{
			Name: name,
			Port: process.Port,
			URL:  process.URL,
		}
		if serviceState.URL == "" && serviceState.Port > 0 {
			serviceState.URL = fmt.Sprintf("http://localhost:%d", serviceState.Port)
		}

		services = append(services, serviceState)
	}

	return services
}

func waitForDashboardURL(projectDir string, dashboardServer *dashboard.Server, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if dashboardServer != nil {
			if url := dashboardServer.GetURL(); url != "" {
				return url
			}
		}

		if port := dashboard.ReadPortFile(projectDir); port > 0 {
			return fmt.Sprintf("http://localhost:%d", port)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return ""
}

// executePrestopHook executes the prestop hook if configured.
func executePrestopHook(ctx context.Context, azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(ctx, azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPrestop(), "prestop", workingDir)
}

// executePoststopHook executes the poststop hook if configured.
func executePoststopHook(ctx context.Context, azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(ctx, azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPoststop(), "poststop", workingDir)
}

// monitorServiceProcess monitors a single service process for exit or cancellation.
// This function runs in its own goroutine with panic recovery to ensure one service
// crash doesn't affect others (process isolation).
func monitorServiceProcess(ctx context.Context, wg *sync.WaitGroup, serviceName string, proc *service.ServiceProcess, projectDir string) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			cliout.Error("Service monitor panic recovered for %s: %v", serviceName, r)
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
			cliout.Warning("Failed to update exit info for %s: %v", serviceName, regErr)
		}

		// Get service mode from registry to determine appropriate status
		entry, _ := reg.GetService(serviceName)
		mode := ""
		if entry != nil {
			mode = entry.Mode
		}

		if result.err != nil {
			// Update registry to trigger OS notification via state monitor
			if regErr := updateRegistryWithRetry(reg, serviceName, "error"); regErr != nil {
				cliout.Error("Failed to update registry for %s after retries: %v", serviceName, regErr)
			}

			// Show mode-appropriate error message
			switch mode {
			case service.ServiceModeBuild:
				cliout.Error("Build failed: %s (exit code %d)", serviceName, result.exitCode)
			case service.ServiceModeTask:
				cliout.Error("Task failed: %s (exit code %d)", serviceName, result.exitCode)
			default:
				cliout.Error("⚠️  %v", result.err)
				cliout.Warning("Service %s stopped. Other services continue running.", serviceName)
				cliout.Info("Press Ctrl+C to stop all services")
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
				cliout.Info("Service %s exited cleanly", serviceName)
			}

			if regErr := updateRegistryWithRetry(reg, serviceName, status); regErr != nil {
				cliout.Warning("Failed to update registry for %s after retries: %v", serviceName, regErr)
			}
		}
		// Intentionally don't cancel context - other services should continue
	case <-ctx.Done():
		// Context canceled by signal - proceed to graceful shutdown
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

	cliout.Plain("Running Aspire in native mode")
	cliout.Item("Directory: %s", aspireProject.Dir)
	cliout.Item("Project: %s", aspireProject.ProjectFile)
	cliout.Newline()
	cliout.Plain("Aspire dashboard will start automatically")
	cliout.Newline()

	// Use executor to run dotnet with proper environment inheritance
	args := []string{"run", "--project", aspireProject.ProjectFile}

	cliout.Hint("Press Ctrl+C to stop")
	cliout.Newline()

	// Run dotnet and let it handle everything (inherits all azd env vars)
	return executor.StartCommand(ctx, "dotnet", args, aspireProject.Dir)
}

// updateRegistryWithRetry updates the service registry with retry and exponential backoff.
func updateRegistryWithRetry(reg *registry.ServiceRegistry, serviceName, status string) error {
	const maxRetries = 3
	retryDelay := 100 * time.Millisecond
	var err error
	for i := 0; i < maxRetries; i++ {
		err = reg.UpdateStatus(serviceName, status)
		if err == nil {
			return nil
		}
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2
		}
	}
	return err
}
