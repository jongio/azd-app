package commands

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"app/src/internal/dashboard"
	"app/src/internal/detector"
	"app/src/internal/installer"
	"app/src/internal/orchestrator"
	"app/src/internal/security"
	"app/src/internal/service"

	"gopkg.in/yaml.v3"
)

// Global orchestrator instance shared across all commands.
var cmdOrchestrator *orchestrator.Orchestrator

// init initializes the command orchestrator and registers all commands.
func init() {
	cmdOrchestrator = orchestrator.NewOrchestrator()

	// Register commands with their dependencies
	// reqs has no dependencies
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:    "reqs",
		Execute: executeReqs,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register reqs command: %v\n", err)
		os.Exit(1)
	}

	// deps depends on reqs
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute:      executeDeps,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register deps command: %v\n", err)
		os.Exit(1)
	}

	// Note: 'run' command is now standalone in run.go and doesn't use the orchestrator
}

// executeReqs is the core logic for the reqs command.
func executeReqs() error {
	fmt.Println("🔍 Checking requirements...")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find azure.yaml in current or parent directories
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath == "" {
		fmt.Println("ℹ️  No azure.yaml found - skipping requirement check")
		fmt.Println()
		return nil
	}

	// Validate path to azure.yaml
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if len(azureYaml.Requirements) == 0 {
		fmt.Println("ℹ️  No requirements defined in azure.yaml")
		fmt.Println()
		return nil
	}

	allSatisfied := true
	for _, prereq := range azureYaml.Requirements {
		passed := checkPrerequisite(prereq)
		if !passed {
			allSatisfied = false
		}
	}

	fmt.Println()
	if !allSatisfied {
		return fmt.Errorf("requirement check failed")
	}

	fmt.Println("✅ All requirements satisfied!")
	return nil
}

// executeDeps is the core logic for the deps command.
func executeDeps() error {
	fmt.Println("🔍 Installing dependencies...")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	hasProjects := false

	// Step 1: Find and install Node.js projects
	nodeProjects, err := detector.FindNodeProjects(cwd)
	if err == nil && len(nodeProjects) > 0 {
		hasProjects = true
		fmt.Printf("📦 Found %d Node.js project(s)\n", len(nodeProjects))
		for _, nodeProject := range nodeProjects {
			if err := installer.InstallNodeDependencies(nodeProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to install for %s: %v\n", nodeProject.Dir, err)
			}
		}
		fmt.Println()
	}

	// Step 2: Find and install Python projects
	pythonProjects, err := detector.FindPythonProjects(cwd)
	if err == nil && len(pythonProjects) > 0 {
		hasProjects = true
		fmt.Printf("🐍 Found %d Python project(s)\n", len(pythonProjects))
		for _, pyProject := range pythonProjects {
			if err := installer.SetupPythonVirtualEnv(pyProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to setup environment for %s: %v\n", pyProject.Dir, err)
			}
		}
		fmt.Println()
	}

	// Step 3: Find and install .NET projects
	dotnetProjects, err := detector.FindDotnetProjects(cwd)
	if err == nil && len(dotnetProjects) > 0 {
		hasProjects = true
		fmt.Printf("🔷 Found %d .NET project(s)\n", len(dotnetProjects))
		for _, dotnetProject := range dotnetProjects {
			if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to restore %s: %v\n", dotnetProject.Path, err)
			}
		}
		fmt.Println()
	}

	if !hasProjects {
		fmt.Println("ℹ️  No projects detected - skipping dependency installation")
		fmt.Println()
		return nil
	}

	fmt.Println("✅ Dependencies installed successfully!")
	return nil
}

// runAzureYamlServices runs services defined in azure.yaml using service orchestration.
// This is called from executeDeps to handle azure.yaml services in the orchestrator context.
func runAzureYamlServices(azureYaml *service.AzureYaml, azureYamlPath string) error {
	// Import the runServicesFromAzureYaml logic by calling it directly
	// We can't easily reuse the function from run.go due to package isolation,
	// so we'll implement a simple version that calls the service orchestrator
	
	fmt.Println("🚀 Starting development environment...")
	fmt.Println()
	
	// Filter services if needed (for now, run all services)
	services := azureYaml.Services

	// Track used ports to avoid conflicts
	usedPorts := make(map[int]bool)

	// Detect runtime for each service
	runtimes := make([]*service.ServiceRuntime, 0, len(services))
	for name, svc := range services {
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts)
		if err != nil {
			return fmt.Errorf("failed to detect runtime for service %s: %w", name, err)
		}
		usedPorts[runtime.Port] = true
		runtimes = append(runtimes, runtime)
	}

	// Create logger
	logger := service.NewServiceLogger(false)
	logger.LogStartup(len(runtimes))

	// Orchestrate services (using empty env vars)
	envVars := make(map[string]string)
	result, err := service.OrchestrateServices(runtimes, envVars, logger)
	if err != nil {
		return fmt.Errorf("service orchestration failed: %w", err)
	}

	// Validate that all services are ready
	if err := service.ValidateOrchestration(result); err != nil {
		service.StopAllServices(result.Processes)
		return fmt.Errorf("service validation failed: %w", err)
	}

	// Get service URLs and log summary
	urls := service.GetServiceURLs(result.Processes)
	logger.LogSummary(urls)

	// Start dashboard server (simplified version)
	cwd, _ := os.Getwd()
	dashboardServer := dashboard.GetServer(cwd)
	dashboardURL, err := dashboardServer.Start()
	if err != nil {
		fmt.Printf("Warning: Failed to start dashboard: %v\n", err)
	} else {
		fmt.Printf("\n📊 Dashboard: %s\n", dashboardURL)
		fmt.Println()
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan

	fmt.Println("\n\n🛑 Shutting down services...")
	
	// Stop dashboard
	if err := dashboardServer.Stop(); err != nil {
		fmt.Printf("Warning: Failed to stop dashboard: %v\n", err)
	}
	
	service.StopAllServices(result.Processes)
	fmt.Println("✅ All services stopped")

	return nil
}
