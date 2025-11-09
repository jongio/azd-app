package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/vscode"
	"github.com/jongio/azd-app/cli/src/internal/yamlutil"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
)

var (
	debugServiceFilter         string
	debugEnvFile               string
	debugVerbose               bool
	debugDryRun                bool
	debugRuntime               string
	debugWaitForDebugger       bool
	debugRegenerateDebugConfig bool
)

// NewDebugCommand creates the debug command.
func NewDebugCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Start services with debuggers enabled for F5 debugging in VS Code",
		Long: `Start your development environment with debuggers enabled for all services.

This command:
- Configures debug ports for each service based on language
- Generates VS Code launch.json and tasks.json configurations
- Starts services with debuggers enabled
- Shows debug connection information

Once services are running, press F5 in VS Code to attach debuggers to all services.

Supported languages:
- Node.js/TypeScript (Chrome DevTools Protocol)
- Python (debugpy)
- Go (Delve)
- .NET (CoreCLR)
- Java (JDWP)`,
		Example: `  # Start all services in debug mode
  azd app debug

  # Debug specific services only
  azd app debug --service web,api

  # Wait for debugger to attach before starting
  azd app debug --wait-for-debugger

  # Regenerate VS Code debug configurations
  azd app debug --regenerate-config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugMode(cmd, args)
		},
	}

	// Add flags for debug mode
	cmd.Flags().StringVarP(&debugServiceFilter, "service", "s", "", "Debug specific service(s) only (comma-separated)")
	cmd.Flags().StringVar(&debugEnvFile, "env-file", "", "Load environment variables from .env file")
	cmd.Flags().BoolVarP(&debugVerbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().BoolVar(&debugDryRun, "dry-run", false, "Show debug configuration without starting services")
	cmd.Flags().StringVar(&debugRuntime, "runtime", runtimeModeAzd, "Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run)")
	cmd.Flags().BoolVar(&debugWaitForDebugger, "wait-for-debugger", false, "Pause services until debugger attaches")
	cmd.Flags().BoolVar(&debugRegenerateDebugConfig, "regenerate-config", false, "Regenerate .vscode debug configurations")

	return cmd
}

// runDebugMode runs services in debug mode.
func runDebugMode(_ *cobra.Command, _ []string) error {
	if err := validateRuntimeMode(debugRuntime); err != nil {
		return err
	}

	// Execute dependencies first (reqs -> deps -> debug)
	if err := cmdOrchestrator.Run("run"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}

	return runServicesInDebugMode(azureYamlPath, debugRuntime)
}

// runServicesInDebugMode orchestrates services in debug mode.
func runServicesInDebugMode(azureYamlPath string, runtimeMode string) error {
	azureYamlDir := filepath.Dir(azureYamlPath)

	// Aspire mode: run AppHost directly (debug mode for Aspire uses standard run)
	if runtimeMode == runtimeModeAspire {
		output.Warning("Debug mode with Aspire runtime uses native Aspire debugging.")
		output.Info("Use Visual Studio or Rider for Aspire debugging.")
		return runAspireMode(azureYamlDir)
	}

	// AZD mode: orchestrate services individually with debug enabled
	return runAzdModeDebug(azureYamlPath, azureYamlDir)
}

// runAzdModeDebug runs services in azd mode with debug enabled.
func runAzdModeDebug(azureYamlPath, azureYamlDir string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Parse azure.yaml
	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	// Check if there are services defined
	if !service.HasServices(azureYaml) {
		return showNoServicesMessage()
	}

	// Filter and detect services
	services := filterDebugServices(azureYaml)
	if len(services) == 0 {
		return fmt.Errorf("no services match filter: %s", debugServiceFilter)
	}

	runtimes, err := detectServiceRuntimesDebug(services, azureYamlDir, runtimeModeAzd)
	if err != nil {
		return err
	}

	// Dry-run mode: show what would be executed
	if debugDryRun {
		return showDebugDryRun(runtimes)
	}

	// Execute and monitor services in debug mode
	return executeAndMonitorDebugServices(runtimes, cwd)
}

// filterDebugServices applies service filtering based on --service flag.
func filterDebugServices(azureYaml *service.AzureYaml) map[string]service.Service {
	if debugServiceFilter == "" {
		return azureYaml.Services
	}
	filterList := strings.Split(debugServiceFilter, ",")
	return service.FilterServices(azureYaml, filterList)
}

// detectServiceRuntimesDebug detects runtime information for all services with debug enabled.
func detectServiceRuntimesDebug(services map[string]service.Service, azureYamlDir, runtimeMode string) ([]*service.ServiceRuntime, error) {
	usedPorts := make(map[int]bool)
	runtimes := make([]*service.ServiceRuntime, 0, len(services))

	// Find azure.yaml path for updates
	azureYamlPath := filepath.Join(azureYamlDir, "azure.yaml")

	// Sort service names for deterministic ordering (fixes debug port race condition)
	serviceNames := make([]string, 0, len(services))
	for name := range services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// Count services per language for debug port assignment
	languageCounts := make(map[string]int)

	for _, name := range serviceNames {
		svc := services[name]
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

		// Always configure debug for this command
		normalizedLang := service.NormalizeLanguage(runtime.Language)
		languageIndex := languageCounts[normalizedLang]
		languageCounts[normalizedLang]++
		service.ConfigureDebug(runtime, true, debugWaitForDebugger, languageIndex)

		runtimes = append(runtimes, runtime)
	}

	return runtimes, nil
}

// executeAndMonitorDebugServices starts services in debug mode and monitors them until interrupted.
func executeAndMonitorDebugServices(runtimes []*service.ServiceRuntime, cwd string) error {
	// Generate VS Code debug configurations
	if err := generateDebugConfig(runtimes, cwd, debugRegenerateDebugConfig); err != nil {
		output.Warning("Failed to generate VS Code debug config: %v", err)
	}

	// Create logger
	logger := service.NewServiceLogger(debugVerbose)

	// Show debug mode message
	output.Info("🐛 Starting services in debug mode...")

	// Load environment variables
	envVars, err := loadDebugEnvironmentVariables()
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

	// Show debug information
	showDebugInfo(runtimes)

	// Start dashboard and wait for shutdown
	return monitorDebugServicesUntilShutdown(result, cwd)
}

// loadDebugEnvironmentVariables loads environment variables from --env-file if specified.
func loadDebugEnvironmentVariables() (map[string]string, error) {
	if debugEnvFile == "" {
		return make(map[string]string), nil
	}

	envVars, err := service.LoadDotEnv(debugEnvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load env file: %w", err)
	}
	return envVars, nil
}

// monitorDebugServicesUntilShutdown starts the dashboard and monitors services using errgroup.
func monitorDebugServicesUntilShutdown(result *service.OrchestrationResult, cwd string) error {
	// Create context that cancels on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	dashboardServer := dashboard.GetServer(cwd)

	// Goroutine 1: Dashboard server
	g.Go(func() error {
		dashboardURL, err := dashboardServer.Start()
		if err != nil {
			output.Warning("Dashboard unavailable: %v", err)
			<-ctx.Done()
			return nil
		}

		output.Newline()
		output.Info("📊 Dashboard: %s", output.URL(dashboardURL))
		output.Newline()
		output.Info("💡 Press Ctrl+C to stop all services")
		output.Newline()

		<-ctx.Done()
		return nil
	})

	// Goroutine 2+: One goroutine per service to wait for exit
	for name, process := range result.Processes {
		serviceName := name
		proc := process

		if proc.Process == nil {
			continue
		}

		g.Go(func() error {
			waitDone := make(chan error, 1)
			go func() {
				state, err := proc.Process.Wait()
				if err != nil {
					waitDone <- fmt.Errorf("service %s exited with error: %w", serviceName, err)
					return
				}
				if !state.Success() {
					exitCode := state.ExitCode()
					waitDone <- fmt.Errorf("service %s exited with code %d: %s", serviceName, exitCode, state.String())
					return
				}
				waitDone <- nil
			}()

			select {
			case err := <-waitDone:
				return err
			case <-ctx.Done():
				return nil
			}
		})
	}

	// Wait for first error, signal, or all services to exit
	err := g.Wait()

	// Perform cleanup shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	output.Newline()
	output.Newline()
	output.Warning("🛑 Shutting down services...")

	// Stop dashboard
	if stopErr := dashboardServer.Stop(); stopErr != nil {
		output.Warning("Failed to stop dashboard: %v", stopErr)
	}

	// Stop all services with graceful timeout
	if stopErr := shutdownAllServicesDebug(shutdownCtx, result.Processes); stopErr != nil {
		output.Warning("Some services failed to stop cleanly: %v", stopErr)
	}

	output.Success("All services stopped")
	output.Newline()

	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

// shutdownAllServicesDebug stops all services with graceful timeout.
func shutdownAllServicesDebug(ctx context.Context, processes map[string]*service.ServiceProcess) error {
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

			deadline, ok := ctx.Deadline()
			timeout := 5 * time.Second
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

// showDebugDryRun displays debug configuration without starting services.
func showDebugDryRun(runtimes []*service.ServiceRuntime) error {
	output.Section("🔍", "Debug mode dry-run: Showing debug configuration")

	for _, runtime := range runtimes {
		output.Newline()
		output.Info("%s", runtime.Name)
		output.Label("Language", runtime.Language)
		output.Label("Framework", runtime.Framework)
		output.Label("Port", fmt.Sprintf("%d", runtime.Port))
		output.Label("Directory", runtime.WorkingDir)
		output.Label("Command", fmt.Sprintf("%s %v", runtime.Command, runtime.Args))

		if runtime.Debug.Enabled {
			output.Label("Debug", fmt.Sprintf("enabled on port %d (%s)", runtime.Debug.Port, runtime.Debug.Protocol))
			if debugWaitForDebugger {
				output.Label("Wait Mode", "service will pause until debugger attaches")
			}
		}
	}

	return nil
}

// generateDebugConfig generates VS Code debug configurations.
func generateDebugConfig(runtimes []*service.ServiceRuntime, projectDir string, forceRegenerate bool) error {
	// Collect service debug info
	services := []vscode.ServiceDebugInfo{}
	for _, rt := range runtimes {
		if rt.Debug.Enabled {
			services = append(services, vscode.ServiceDebugInfo{
				Name:     rt.Name,
				Language: rt.Language,
				Port:     rt.Debug.Port,
			})
		}
	}

	if len(services) == 0 {
		return nil
	}

	// Check if this is the first time running in debug mode
	isFirstTime := forceRegenerate
	vscodeDir := filepath.Join(projectDir, ".vscode")
	launchPath := filepath.Join(vscodeDir, "launch.json")
	if _, err := os.Stat(launchPath); os.IsNotExist(err) {
		isFirstTime = true
	}

	// Generate config
	if _, err := vscode.EnsureDebugConfig(projectDir, services, forceRegenerate); err != nil {
		return err
	}

	// Show helpful message on first time
	if isFirstTime {
		output.Newline()
		output.Success("✅ Generated .vscode/launch.json and tasks.json")
		output.Newline()
	}

	return nil
}

// showDebugInfo displays debug information for all services.
func showDebugInfo(runtimes []*service.ServiceRuntime) {
	output.Newline()

	// Show debug ports
	hasDebugServices := false
	for _, rt := range runtimes {
		if rt.Debug.Enabled {
			hasDebugServices = true
			debugAddr := fmt.Sprintf("localhost:%d", rt.Debug.Port)
			output.ItemSuccess("   ✅ %s%-15s%s running (debug: %s)", output.Cyan, rt.Name, output.Reset, debugAddr)
		}
	}

	if hasDebugServices {
		output.Newline()
		output.Info("📖 To debug:")
		output.Item("   1. Press F5 in VS Code")
		output.Item("   2. Select \"🚀 Debug ALL Services\" to attach to all services")
		output.Item("   3. Or select individual service to debug")
		output.Newline()
	}
}
