package commands

import (
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
	"github.com/jongio/azd-app/cli/src/internal/vscode"

	"github.com/spf13/cobra"
)

const (
	runtimeModeAzd    = "azd"
	runtimeModeAspire = "aspire"
)

var (
	runServiceFilter         string
	runEnvFile               string
	runVerbose               bool
	runDryRun                bool
	runRuntime               string
	runDebug                 bool
	runWaitForDebugger       bool
	runRegenerateDebugConfig bool
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
	cmd.Flags().BoolVar(&runDebug, "debug", false, "Start services with debuggers enabled")
	cmd.Flags().BoolVar(&runWaitForDebugger, "wait-for-debugger", false, "Pause services until debugger attaches")
	cmd.Flags().BoolVar(&runRegenerateDebugConfig, "regenerate-debug-config", false, "Regenerate .vscode debug configurations")

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

	// Count services per language for debug port assignment
	languageCounts := make(map[string]int)

	for name, svc := range services {
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts, azureYamlDir, runtimeMode)
		if err != nil {
			return nil, fmt.Errorf("failed to detect runtime for service %s: %w", name, err)
		}
		usedPorts[runtime.Port] = true

		// Configure debug if --debug flag is set
		if runDebug {
			normalizedLang := service.NormalizeLanguage(runtime.Language)
			languageIndex := languageCounts[normalizedLang]
			languageCounts[normalizedLang]++
			service.ConfigureDebug(runtime, runDebug, runWaitForDebugger, languageIndex)
		}

		runtimes = append(runtimes, runtime)
	}

	return runtimes, nil
}

// executeAndMonitorServices starts services and monitors them until interrupted.
func executeAndMonitorServices(runtimes []*service.ServiceRuntime, cwd string) error {
	// Generate VS Code debug configurations if in debug mode
	if runDebug {
		if err := generateDebugConfig(runtimes, cwd); err != nil {
			output.Warning("Failed to generate VS Code debug config: %v", err)
		}
	}

	// Create logger
	logger := service.NewServiceLogger(runVerbose)

	// Show debug mode message if enabled
	if runDebug {
		output.Info("🐛 Starting services in debug mode...")
	} else {
		logger.LogStartup(len(runtimes))
	}

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

	// Show debug information if in debug mode
	if runDebug {
		showDebugInfo(runtimes)
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

// monitorServicesUntilShutdown starts the dashboard and waits for shutdown signal.
func monitorServicesUntilShutdown(result *service.OrchestrationResult, cwd string) error {
	dashboardServer := startDashboard(cwd)

	output.Info("💡 Press Ctrl+C to stop all services")
	output.Newline()

	waitForShutdownSignal()

	return shutdownServices(result, dashboardServer)
}

// startDashboard starts the azd dashboard server.
func startDashboard(cwd string) *dashboard.Server {
	dashboardServer := dashboard.GetServer(cwd)
	dashboardURL, err := dashboardServer.Start()
	if err != nil {
		output.Warning("Dashboard unavailable: %v", err)
		return nil
	}

	output.Newline()
	output.Info("📊 Dashboard: %s", output.URL(dashboardURL))
	output.Newline()
	return dashboardServer
}

// waitForShutdownSignal blocks until SIGINT or SIGTERM is received.
func waitForShutdownSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
}

// shutdownServices stops all services and the dashboard.
func shutdownServices(result *service.OrchestrationResult, dashboardServer *dashboard.Server) error {
	output.Newline()
	output.Newline()
	output.Warning("🛑 Shutting down services...")

	if dashboardServer != nil {
		if err := dashboardServer.Stop(); err != nil {
			output.Warning("Failed to stop dashboard: %v", err)
		}
	}

	service.StopAllServices(result.Processes)
	output.Success("All services stopped")
	output.Newline()

	return nil
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
	return executor.StartCommand("dotnet", args, aspireProject.Dir)
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

		// Show debug info if enabled
		if runtime.Debug.Enabled {
			output.Label("Debug", fmt.Sprintf("enabled on port %d (%s)", runtime.Debug.Port, runtime.Debug.Protocol))
		}
	}

	return nil
}

// generateDebugConfig generates VS Code debug configurations.
func generateDebugConfig(runtimes []*service.ServiceRuntime, projectDir string) error {
	// Check if first time or forced regeneration
	isFirstTime := runRegenerateDebugConfig

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
		return nil // No debug services
	}

	// Check if this is the first time running in debug mode
	vscodeDir := filepath.Join(projectDir, ".vscode")
	launchPath := filepath.Join(vscodeDir, "launch.json")
	if _, err := os.Stat(launchPath); os.IsNotExist(err) {
		isFirstTime = true
	}

	// Generate config
	if err := vscode.EnsureDebugConfig(projectDir, services, runRegenerateDebugConfig); err != nil {
		return err
	}

	// Show helpful message on first time
	if isFirstTime {
		output.Newline()
		output.Success("🛠️  Generated .vscode/launch.json and tasks.json")
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
