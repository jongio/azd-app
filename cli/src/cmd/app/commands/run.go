package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/yamlutil"
	"github.com/oklog/run"

	"github.com/spf13/cobra"
)

const (
	runtimeModeAzd    = "azd"
	runtimeModeAspire = "aspire"
)

var (
	// ErrInterruptSignal is returned when the user interrupts the run command with SIGINT or SIGTERM
	errInterruptSignal = errors.New("received interrupt signal")

	runServiceFilter string
	runEnvFile       string
	runVerbose       bool
	runDryRun        bool
	runRuntime       string
)

// NewRunCommand creates the run command.
func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the development environment (services from azure.yaml, Aspire, pnpm, or docker compose)",
		Long:  `Automatically detects and runs services defined in azure.yaml, or falls back to: Aspire (AppHost.cs), pnpm dev/start scripts, or docker compose from package.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithServices(cmd, args)
		},
	}

	// Add flags for service orchestration
	cmd.Flags().StringVarP(&runServiceFilter, "service", "s", "", "Run specific service(s) only (comma-separated)")
	cmd.Flags().StringVar(&runEnvFile, "env-file", "", "Load environment variables from .env file")
	cmd.Flags().BoolVarP(&runVerbose, "verbose", "v", false, "Enable verbose logging")
	cmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Show what would be run without starting services")
	cmd.Flags().StringVar(&runRuntime, "runtime", runtimeModeAzd, "Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run)")

	return cmd
}

// runWithServices runs services from azure.yaml.
func runWithServices(_ *cobra.Command, _ []string) error {
	if err := validateRuntimeMode(runRuntime); err != nil {
		return err
	}

	// Execute dependencies first (reqs -> deps -> run)
	if err := cmdOrchestrator.Run("run"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	azureYamlPath, err := findAzureYaml()
	if err != nil {
		return err
	}

	return runServicesFromAzureYaml(azureYamlPath, runRuntime)
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
func runServicesFromAzureYaml(azureYamlPath string, runtimeMode string) error {
	azureYamlDir := filepath.Dir(azureYamlPath)

	// Aspire mode: run AppHost directly
	if runtimeMode == runtimeModeAspire {
		return runAspireMode(azureYamlDir)
	}

	// AZD mode: orchestrate services individually
	return runAzdMode(azureYamlPath, azureYamlDir)
}

// runAzdMode runs services in azd mode with individual service orchestration.
func runAzdMode(azureYamlPath, azureYamlDir string) error {
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
	return executeAndMonitorServices(runtimes, cwd)
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
func executeAndMonitorServices(runtimes []*service.ServiceRuntime, cwd string) error {
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

// monitorServicesUntilShutdown starts the dashboard and waits for shutdown signal.
// Uses oklog/run to coordinate the lifecycle of services and dashboard with graceful shutdown.
func monitorServicesUntilShutdown(result *service.OrchestrationResult, cwd string) error {
	var g run.Group

	// Actor 1: Dashboard server
	dashboardServer := dashboard.GetServer(cwd)
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(
			// Execute: Start and run the dashboard
			func() error {
				dashboardURL, err := dashboardServer.Start()
				if err != nil {
					output.Warning("Dashboard unavailable: %v", err)
					// Don't fail the entire run if dashboard fails
					// Just block until interrupted
					<-ctx.Done()
					return ctx.Err()
				}

				output.Newline()
				output.Info("📊 Dashboard: %s", output.URL(dashboardURL))
				output.Newline()
				output.Info("💡 Press Ctrl+C to stop all services")
				output.Newline()

				// Block until interrupted
				<-ctx.Done()
				return ctx.Err()
			},
			// Interrupt: Stop the dashboard
			func(error) {
				cancel()
				if err := dashboardServer.Stop(); err != nil {
					output.Warning("Failed to stop dashboard: %v", err)
				}
			},
		)
	}

	// Actor 2: Service process monitoring
	// Monitor all service processes and propagate errors if any exit unexpectedly
	{
		g.Add(
			// Execute: Wait for any service to exit
			func() error {
				return service.WaitForServices(result.Processes)
			},
			// Interrupt: Stop all services
			func(error) {
				output.Newline()
				output.Newline()
				output.Warning("🛑 Shutting down services...")
				service.StopAllServices(result.Processes)
				output.Success("All services stopped")
				output.Newline()
			},
		)
	}

	// Actor 3: Signal handler for graceful shutdown
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(
			// Execute: Wait for interrupt signal
			func() error {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
				defer signal.Stop(sigChan)
				select {
				case <-sigChan:
					return errInterruptSignal
				case <-ctx.Done():
					return ctx.Err()
				}
			},
			// Interrupt: Cancel the context
			func(error) {
				cancel()
			},
		)
	}

	// Run all actors - this blocks until one of them returns an error or is interrupted
	// All actors are stopped gracefully via their interrupt functions
	err := g.Run()

	// Clean up port assignments on clean shutdown
	// Note: Port assignments are kept in the file for persistence across runs,
	// but we don't release them here to allow quick restarts with same ports.
	// Stale ports are cleaned up automatically after 7 days of inactivity.

	// If the error is just from signal interrupt, that's expected
	if errors.Is(err, errInterruptSignal) {
		return nil
	}

	return err
}

// runAspireMode runs Aspire AppHost directly using dotnet run.
func runAspireMode(rootDir string) error {
	// Find Aspire AppHost project
	aspireProject, err := detector.FindAppHost(rootDir)
	if err != nil {
		return fmt.Errorf("failed to search for Aspire AppHost: %w", err)
	}

	if aspireProject == nil {
		return fmt.Errorf("no Aspire AppHost found - --runtime aspire requires an AppHost.cs or Program.cs file in a .csproj project")
	}

	output.Info("🚀 Running Aspire in native mode")
	output.Item("Directory: %s", aspireProject.Dir)
	output.Item("Project: %s", aspireProject.ProjectFile)
	output.Newline()
	output.Info("💡 Aspire dashboard will start automatically")
	output.Info("💡 All azd environment variables are available to your app")
	output.Newline()

	// Use executor to run dotnet with proper environment inheritance
	args := []string{"run", "--project", aspireProject.ProjectFile}

	output.Info("💡 Press Ctrl+C to stop")
	output.Newline()

	// Run dotnet and let it handle everything (inherits all azd env vars)
	return executor.StartCommand(context.Background(), "dotnet", args, aspireProject.Dir)
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
