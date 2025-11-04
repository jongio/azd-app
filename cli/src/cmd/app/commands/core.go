package commands

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"

	"gopkg.in/yaml.v3"
)

// Global orchestrator instance shared across all commands.
var cmdOrchestrator *orchestrator.Orchestrator

// Global flag to disable cache (set by --no-cache flag)
var disableCache bool

// setDisableCache sets the cache disable flag
func setDisableCache(disable bool) {
	disableCache = disable
}

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

	// run depends on deps (which depends on reqs)
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "run",
		Dependencies: []string{"deps"},
		Execute:      executeRun,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register run command: %v\n", err)
		os.Exit(1)
	}
}

// executeReqs is the core logic for the reqs command.
func executeReqs() error {
	if !output.IsJSON() {
		output.Section("🔍", "Checking reqs...")
	}

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
		return fmt.Errorf("no azure.yaml found in current directory or parents - run 'azd app reqs --generate' to create one")
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

	if len(azureYaml.Reqs) == 0 {
		return fmt.Errorf("no reqs defined in azure.yaml - run 'azd app reqs --generate' to add them")
	}

	// Initialize cache manager (skip if caching is disabled)
	var cacheManager *cache.CacheManager
	if !disableCache {
		var err error
		cacheManager, err = cache.NewCacheManager()
		if err != nil {
			// If cache fails, proceed without caching (fallback)
			if !output.IsJSON() {
				output.Warning("Cache initialization failed, proceeding without cache: %v", err)
			}
			cacheManager = nil
		}
	} else if !output.IsJSON() {
		output.Info("Cache disabled, performing fresh reqs check...")
	}

	var results []ReqResult
	var allSatisfied bool

	// Try to get cached results first (only if cache is available and enabled)
	if cacheManager != nil {
		cachedResults, valid, cacheErr := cacheManager.GetCachedResults(azureYamlPath)
		if cacheErr != nil && !output.IsJSON() {
			output.Warning("Failed to read cache: %v", cacheErr)
		}
		if valid && cachedResults != nil {
			// Use cached results
			if !output.IsJSON() {
				output.Info("Using cached reqs check results...")
			}

			// Convert cached results to ReqResult format
			results = make([]ReqResult, len(cachedResults.Results))
			for i, cached := range cachedResults.Results {
				results[i] = ReqResult{
					ID:         cached.ID,
					Installed:  cached.Installed,
					Version:    cached.Version,
					Required:   cached.Required,
					Satisfied:  cached.Satisfied,
					Running:    cached.Running,
					CheckedRun: cached.CheckedRun,
					Message:    cached.Message,
				}
			}
			allSatisfied = cachedResults.AllPassed

			// Print cached results in non-JSON mode
			if !output.IsJSON() {
				for _, result := range results {
					printRequirementResult(result)
				}
			}
		} else {
			// Cache miss or invalid - perform fresh check
			results, allSatisfied = performReqsCheck(azureYaml.Reqs)

			// Save to cache
			cacheResults := make([]cache.CachedReqResult, len(results))
			for i, result := range results {
				cacheResults[i] = cache.CachedReqResult{
					ID:         result.ID,
					Installed:  result.Installed,
					Version:    result.Version,
					Required:   result.Required,
					Satisfied:  result.Satisfied,
					Running:    result.Running,
					CheckedRun: result.CheckedRun,
					Message:    result.Message,
				}
			}
			if saveErr := cacheManager.SaveResults(azureYamlPath, cacheResults, allSatisfied); saveErr != nil && !output.IsJSON() {
				output.Warning("Failed to save cache: %v", saveErr)
			}
		}
	} else {
		// No cache available - perform fresh check
		results, allSatisfied = performReqsCheck(azureYaml.Reqs)
	}

	// JSON output
	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"satisfied": allSatisfied,
			"reqs":      results,
		})
	}

	// Default output
	output.Newline()
	if !allSatisfied {
		return fmt.Errorf("requirement check failed")
	}

	output.Success("All reqs satisfied!")
	return nil
}

// executeDeps is the core logic for the deps command.
func executeDeps() error {
	if !output.IsJSON() {
		output.Newline()
		output.Section("🔍", "Installing dependencies")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		if output.IsJSON() {
			return output.PrintJSON(map[string]interface{}{
				"error": err.Error(),
			})
		}
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	hasProjects := false
	var results []map[string]interface{}

	// Step 1: Find and install Node.js projects
	//nolint:dupl // Similar code pattern repeated for each project type for clarity
	nodeProjects, err := detector.FindNodeProjects(cwd)
	if err == nil && len(nodeProjects) > 0 {
		hasProjects = true
		if !output.IsJSON() {
			output.Step("📦", "Found %s Node.js project(s)", output.Count(len(nodeProjects)))
		}
		for _, nodeProject := range nodeProjects {
			result := map[string]interface{}{
				"type":    "node",
				"dir":     nodeProject.Dir,
				"manager": nodeProject.PackageManager,
			}
			if err := installer.InstallNodeDependencies(nodeProject); err != nil {
				if !output.IsJSON() {
					output.ItemWarning("Failed to install for %s: %v", nodeProject.Dir, err)
				}
				result["success"] = false
				result["error"] = err.Error()
			} else {
				result["success"] = true
			}
			results = append(results, result)
		}
		if !output.IsJSON() {
			output.Newline()
		}
	}

	// Step 2: Find and install Python projects
	//nolint:dupl // Similar code pattern repeated for each project type for clarity
	pythonProjects, err := detector.FindPythonProjects(cwd)
	if err == nil && len(pythonProjects) > 0 {
		hasProjects = true
		if !output.IsJSON() {
			output.Step("🐍", "Found %s Python project(s)", output.Count(len(pythonProjects)))
		}
		for _, pyProject := range pythonProjects {
			result := map[string]interface{}{
				"type":    "python",
				"dir":     pyProject.Dir,
				"manager": pyProject.PackageManager,
			}
			if err := installer.SetupPythonVirtualEnv(pyProject); err != nil {
				if !output.IsJSON() {
					output.ItemWarning("Failed to setup environment for %s: %v", pyProject.Dir, err)
				}
				result["success"] = false
				result["error"] = err.Error()
			} else {
				result["success"] = true
			}
			results = append(results, result)
		}
		if !output.IsJSON() {
			output.Newline()
		}
	}

	// Step 3: Find and install .NET projects
	dotnetProjects, err := detector.FindDotnetProjects(cwd)
	if err == nil && len(dotnetProjects) > 0 {
		hasProjects = true
		if !output.IsJSON() {
			output.Step("🔷", "Found %s .NET project(s)", output.Count(len(dotnetProjects)))
		}
		for _, dotnetProject := range dotnetProjects {
			result := map[string]interface{}{
				"type": "dotnet",
				"path": dotnetProject.Path,
			}
			if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
				if !output.IsJSON() {
					output.ItemWarning("Failed to restore %s: %v", dotnetProject.Path, err)
				}
				result["success"] = false
				result["error"] = err.Error()
			} else {
				result["success"] = true
			}
			results = append(results, result)
		}
		if !output.IsJSON() {
			output.Newline()
		}
	}

	if !hasProjects {
		if output.IsJSON() {
			return output.PrintJSON(map[string]interface{}{
				"success":  true,
				"projects": []interface{}{},
				"message":  "No projects detected",
			})
		}
		output.Info("No projects detected - skipping dependency installation")
		return nil
	}

	if output.IsJSON() {
		// Check if any project failed
		allSuccess := true
		for _, result := range results {
			if success, ok := result["success"].(bool); ok && !success {
				allSuccess = false
				break
			}
		}
		return output.PrintJSON(map[string]interface{}{
			"success":  allSuccess,
			"projects": results,
		})
	}

	output.Success("Dependencies installed successfully!")
	return nil
}

// executeRun is the function executed by the orchestrator for the run command.
// This ensures deps (and transitively reqs) are run before starting services.
func executeRun() error {
	if !output.IsJSON() {
		output.Section("🚀", "Starting services (reqs and deps already checked)...")
	}
	// The actual run logic is handled by the run command's RunE function
	// This is just a marker to ensure the dependency chain is executed
	return nil
}

// Deprecated: Legacy function kept for reference
var _ = _runAzureYamlServices

// runAzureYamlServices runs services defined in azure.yaml using service orchestration.
// This is called from executeDeps to handle azure.yaml services in the orchestrator context.
func _runAzureYamlServices(azureYaml *service.AzureYaml, azureYamlPath string) error {
	// Import the runServicesFromAzureYaml logic by calling it directly
	// We can't easily reuse the function from run.go due to package isolation,
	// so we'll implement a simple version that calls the service orchestrator

	// Get directory containing azure.yaml
	azureYamlDir := filepath.Dir(azureYamlPath)

	output.Section("🚀", "Starting development environment")

	// Filter services if needed (for now, run all services)
	services := azureYaml.Services

	// Track used ports to avoid conflicts
	usedPorts := make(map[int]bool)

	// Detect runtime for each service
	runtimes := make([]*service.ServiceRuntime, 0, len(services))
	for name, svc := range services {
		// Use "azd" mode by default for background service tracking
		runtime, err := service.DetectServiceRuntime(name, svc, usedPorts, azureYamlDir, "azd")
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
		output.Warning("Failed to start dashboard: %v", err)
	} else {
		output.Newline()
		output.Info("📊 Dashboard: %s", output.URL(dashboardURL))
		output.Newline()
	}

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

	return nil
}

// performReqsCheck performs fresh reqs checking.
func performReqsCheck(reqs []Prerequisite) ([]ReqResult, bool) {
	results := make([]ReqResult, 0, len(reqs))
	allSatisfied := true

	for _, prereq := range reqs {
		result := checkPrerequisiteWithResult(prereq)
		results = append(results, result)
		if !result.Satisfied {
			allSatisfied = false
		}
	}

	return results, allSatisfied
}

// printRequirementResult prints a single requirement result in human-readable format.
func printRequirementResult(result ReqResult) {
	if !result.Installed {
		output.ItemError("%s: NOT INSTALLED (required: %s)", result.ID, result.Required)
		return
	}

	if result.Version == "" {
		output.ItemWarning("%s: INSTALLED (version unknown, required: %s)", result.ID, result.Required)
	} else if !result.Satisfied && !result.CheckedRun {
		output.ItemError("%s: %s (required: %s)", result.ID, result.Version, result.Required)
		return
	} else {
		output.ItemSuccess("%s: %s (required: %s)", result.ID, result.Version, result.Required)
	}

	// Check running status if applicable
	if result.CheckedRun {
		if result.Running {
			output.Item("- %s✓%s RUNNING", output.Green, output.Reset)
		} else {
			output.Item("- %s✗%s NOT RUNNING", output.Red, output.Reset)
		}
	}
}
