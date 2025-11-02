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
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"

	"github.com/spf13/cobra"
)

var (
	runServiceFilter string
	runEnvFile       string
	runVerbose       bool
	runDryRun        bool
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

	return cmd
}

// runWithServices runs services from azure.yaml.
func runWithServices(cmd *cobra.Command, args []string) error {
	// Execute dependencies first (reqs -> deps -> run)
	if err := cmdOrchestrator.Run("run"); err != nil {
		return fmt.Errorf("failed to execute command dependencies: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try to find azure.yaml
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	// Require azure.yaml
	if azureYamlPath == "" {
		return fmt.Errorf("azure.yaml not found - create one with 'services' section to define your development environment")
	}

	return runServicesFromAzureYaml(azureYamlPath)
}

// runServicesFromAzureYaml orchestrates services defined in azure.yaml.
func runServicesFromAzureYaml(azureYamlPath string) error {
	// Get directory containing azure.yaml
	azureYamlDir := filepath.Dir(azureYamlPath)

	// Get current working directory for dashboard
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
		output.Info("No services defined in azure.yaml")
		output.Item("Add a 'services' section to azure.yaml to use service orchestration")
		output.Item("or remove azure.yaml to use auto-detection (Aspire, pnpm, docker-compose)")
		return nil
	}

	// Filter services if --service flag is provided
	services := azureYaml.Services
	if runServiceFilter != "" {
		// Split comma-separated list
		filterList := strings.Split(runServiceFilter, ",")
		services = service.FilterServices(azureYaml, filterList)
		if len(services) == 0 {
			return fmt.Errorf("no services match filter: %s", runServiceFilter)
		}
	}

	// Track used ports to avoid conflicts
	usedPorts := make(map[int]bool)

	// Detect runtime for each service
	runtimes := make([]*service.ServiceRuntime, 0, len(services))
	for name, svc := range services {
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts, azureYamlDir)
		if err != nil {
			return fmt.Errorf("failed to detect runtime for service %s: %w", name, err)
		}
		usedPorts[runtime.Port] = true
		runtimes = append(runtimes, runtime)
	}

	// Dry-run mode: show what would be executed
	if runDryRun {
		return showDryRun(runtimes)
	}

	// Create logger
	logger := service.NewServiceLogger(runVerbose)
	logger.LogStartup(len(runtimes))

	// Load environment variables
	envVars := make(map[string]string)
	if runEnvFile != "" {
		loadedEnv, err := service.LoadDotEnv(runEnvFile)
		if err != nil {
			return fmt.Errorf("failed to load env file: %w", err)
		}
		envVars = loadedEnv
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

	// Show ready message
	logger.LogReady()

	// Start dashboard server
	dashboardServer := dashboard.GetServer(cwd)
	dashboardURL, err := dashboardServer.Start()
	if err != nil {
		output.Warning("Dashboard unavailable: %v", err)
	} else {
		output.Newline()
		output.Info("📊 Dashboard: %s", output.URL(dashboardURL))
		output.Newline()
	}

	output.Info("💡 Press Ctrl+C to stop all services")
	output.Newline()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan

	output.Newline()
	output.Newline()
	output.Warning("🛑 Shutting down services...")

	// Stop dashboard
	if err := dashboardServer.Stop(); err != nil {
		output.Warning("Failed to stop dashboard: %v", err)
	}

	service.StopAllServices(result.Processes)
	output.Success("All services stopped")
	output.Newline()

	return nil
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
