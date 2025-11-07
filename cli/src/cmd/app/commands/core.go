// Package commands provides the command-line interface for the azd-app CLI.
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

// ExecutionContext holds runtime configuration for command execution.
type ExecutionContext struct {
	CacheEnabled bool
}

// ReqsResult represents the JSON output structure for reqs command.
type ReqsResult struct {
	Satisfied bool        `json:"satisfied"`
	Reqs      []ReqResult `json:"reqs"`
}

// DepsResult represents the JSON output structure for deps command.
type DepsResult struct {
	Success  bool            `json:"success"`
	Projects []InstallResult `json:"projects"`
	Message  string          `json:"message,omitempty"`
	Error    string          `json:"error,omitempty"`
}

const (
	// msgNoProjectsDetected is used when no projects are found for dependency installation.
	msgNoProjectsDetected = "No projects detected"
)

// Global execution context (temporary until proper context passing is implemented)
var execContext = &ExecutionContext{
	CacheEnabled: true, // Default: cache is enabled
}

// SetCacheEnabled configures whether caching should be enabled.
func SetCacheEnabled(enabled bool) {
	execContext.CacheEnabled = enabled
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
		// Log error but don't exit - let the app handle it gracefully
		fmt.Fprintf(os.Stderr, "Warning: Failed to register reqs command: %v\n", err)
	}

	// deps depends on reqs
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute:      executeDeps,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register deps command: %v\n", err)
	}

	// run depends on deps (which depends on reqs)
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "run",
		Dependencies: []string{"deps"},
		Execute:      executeRun,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to register run command: %v\n", err)
	}
}

// createCacheManager creates a cache manager with fallback to disabled cache on error.
func createCacheManager(enabled bool) *cache.CacheManager {
	cacheManager, err := cache.NewCacheManagerWithOptions(cache.CacheOptions{
		Enabled: enabled,
		TTL:     0, // Use default TTL
	})
	if err != nil {
		// If cache fails to initialize, proceed without caching (fallback)
		if !output.IsJSON() {
			output.Warning("Cache initialization failed, proceeding without cache: %v", err)
		}
		// Return disabled cache manager (won't fail)
		cacheManager, _ = cache.NewCacheManagerWithOptions(cache.CacheOptions{Enabled: false})
	}
	return cacheManager
}

// loadAzureYaml loads and validates the azure.yaml file.
func loadAzureYaml() (string, *AzureYaml, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find azure.yaml in current or parent directories
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return "", nil, fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath == "" {
		return "", nil, fmt.Errorf("no azure.yaml found in current directory or parents - run 'azd app reqs --generate' to create one")
	}

	// Validate path to azure.yaml
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return "", nil, fmt.Errorf("invalid path: %w", err)
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return "", nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	return azureYamlPath, &azureYaml, nil
}

// executeReqs is the core logic for the reqs command.
func executeReqs() error {
	if !output.IsJSON() {
		output.Section("🔍", "Checking reqs...")
	}

	// Load azure.yaml
	azureYamlPath, azureYaml, err := loadAzureYaml()
	if err != nil {
		return err
	}

	if len(azureYaml.Reqs) == 0 {
		return fmt.Errorf("no reqs defined in azure.yaml - run 'azd app reqs --generate' to add them")
	}

	// Initialize cache manager
	cacheManager := createCacheManager(execContext.CacheEnabled)

	// Check requirements (with caching)
	results, allSatisfied := checkRequirementsWithCache(azureYaml.Reqs, azureYamlPath, cacheManager)

	// JSON output
	if output.IsJSON() {
		return output.PrintJSON(ReqsResult{
			Satisfied: allSatisfied,
			Reqs:      results,
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

// DependencyInstaller handles installation of project dependencies.
type DependencyInstaller struct {
	searchRoot string
}

// NewDependencyInstaller creates a new dependency installer.
func NewDependencyInstaller(searchRoot string) *DependencyInstaller {
	return &DependencyInstaller{
		searchRoot: searchRoot,
	}
}

// InstallResult represents the result of installing dependencies for a project.
type InstallResult struct {
	Type    string `json:"type"`
	Dir     string `json:"dir,omitempty"`
	Path    string `json:"path,omitempty"`
	Manager string `json:"manager,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// InstallAll installs dependencies for all detected project types.
func (di *DependencyInstaller) InstallAll() ([]InstallResult, error) {
	var results []InstallResult
	hasProjects := false

	// Install Node.js dependencies
	nodeResults, err := di.installNodeProjects()
	if err == nil && len(nodeResults) > 0 {
		hasProjects = true
		results = append(results, nodeResults...)
	}

	// Install Python dependencies
	pythonResults, err := di.installPythonProjects()
	if err == nil && len(pythonResults) > 0 {
		hasProjects = true
		results = append(results, pythonResults...)
	}

	// Install .NET dependencies
	dotnetResults, err := di.installDotnetProjects()
	if err == nil && len(dotnetResults) > 0 {
		hasProjects = true
		results = append(results, dotnetResults...)
	}

	if !hasProjects {
		return results, nil
	}

	return results, nil
}

// installNodeProjects installs dependencies for Node.js projects.
func (di *DependencyInstaller) installNodeProjects() ([]InstallResult, error) {
	nodeProjects, err := detector.FindNodeProjects(di.searchRoot)
	if err != nil || len(nodeProjects) == 0 {
		return nil, err
	}

	if !output.IsJSON() {
		output.Step("📦", "Found %s Node.js project(s)", output.Count(len(nodeProjects)))
	}

	var results []InstallResult
	for _, nodeProject := range nodeProjects {
		result := di.installProject("node", nodeProject.Dir, nodeProject.PackageManager, func() error {
			return installer.InstallNodeDependencies(nodeProject)
		})
		results = append(results, result)
	}

	if !output.IsJSON() {
		output.Newline()
	}

	return results, nil
}

// installPythonProjects installs dependencies for Python projects.
func (di *DependencyInstaller) installPythonProjects() ([]InstallResult, error) {
	pythonProjects, err := detector.FindPythonProjects(di.searchRoot)
	if err != nil || len(pythonProjects) == 0 {
		return nil, err
	}

	if !output.IsJSON() {
		output.Step("🐍", "Found %s Python project(s)", output.Count(len(pythonProjects)))
	}

	var results []InstallResult
	for _, pyProject := range pythonProjects {
		result := di.installProject("python", pyProject.Dir, pyProject.PackageManager, func() error {
			return installer.SetupPythonVirtualEnv(pyProject)
		})
		results = append(results, result)
	}

	if !output.IsJSON() {
		output.Newline()
	}

	return results, nil
}

// installDotnetProjects installs dependencies for .NET projects.
func (di *DependencyInstaller) installDotnetProjects() ([]InstallResult, error) {
	dotnetProjects, err := detector.FindDotnetProjects(di.searchRoot)
	if err != nil || len(dotnetProjects) == 0 {
		return nil, err
	}

	if !output.IsJSON() {
		output.Step("🔷", "Found %s .NET project(s)", output.Count(len(dotnetProjects)))
	}

	var results []InstallResult
	for _, dotnetProject := range dotnetProjects {
		result := InstallResult{
			Type: "dotnet",
			Path: dotnetProject.Path,
		}
		if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
			if !output.IsJSON() {
				output.ItemWarning("Failed to restore %s: %v", dotnetProject.Path, err)
			}
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}

	if !output.IsJSON() {
		output.Newline()
	}

	return results, nil
}

// installProject installs dependencies for a single project.
func (di *DependencyInstaller) installProject(projectType, dir, manager string, installFunc func() error) InstallResult {
	result := InstallResult{
		Type:    projectType,
		Dir:     dir,
		Manager: manager,
	}

	// Show which project we're installing
	if !output.IsJSON() {
		relDir := dir
		if rel, err := filepath.Rel(di.searchRoot, dir); err == nil && rel != "." {
			relDir = rel
		}
		output.Item("Installing %s (%s)", relDir, manager)
	}

	if err := installFunc(); err != nil {
		if !output.IsJSON() {
			output.ItemWarning("Failed to install for %s: %v", dir, err)
		}
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}
	return result
}

// executeDeps is the core logic for the deps command.
func executeDeps() error {
	if !output.IsJSON() {
		output.Newline()
		output.Section("🔍", "Installing dependencies")
	}

	// Determine search root
	searchRoot, err := getSearchRoot()
	if err != nil {
		if output.IsJSON() {
			return output.PrintJSON(DepsResult{Error: err.Error()})
		}
		return err
	}

	// Install dependencies
	installer := NewDependencyInstaller(searchRoot)
	results, err := installer.InstallAll()
	if err != nil {
		return err
	}

	// Handle no projects case
	if len(results) == 0 {
		if output.IsJSON() {
			return output.PrintJSON(DepsResult{
				Success:  true,
				Projects: []InstallResult{},
				Message:  msgNoProjectsDetected,
			})
		}
		output.Info(msgNoProjectsDetected + " - skipping dependency installation")
		return nil
	}

	// Output results
	if output.IsJSON() {
		allSuccess := checkAllSuccess(results)
		return output.PrintJSON(DepsResult{
			Success:  allSuccess,
			Projects: results,
		})
	}

	output.Success("Dependencies installed successfully!")
	return nil
}

// getSearchRoot determines the search root for finding projects.
func getSearchRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return "", fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath != "" {
		return filepath.Dir(azureYamlPath), nil
	}
	return cwd, nil
}

// checkAllSuccess checks if all install results succeeded.
func checkAllSuccess(results []InstallResult) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
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

// checkRequirementsWithCache checks requirements with cache support.
func checkRequirementsWithCache(reqs []Prerequisite, azureYamlPath string, cacheManager *cache.CacheManager) ([]ReqResult, bool) {
	// Try cache first if enabled
	if cacheManager.IsEnabled() {
		if results, allSatisfied, ok := tryGetCachedResults(azureYamlPath, cacheManager); ok {
			return results, allSatisfied
		}
	}

	// Perform fresh check
	results, allSatisfied := performReqsCheck(reqs)

	// Print fresh results in non-JSON mode
	if !output.IsJSON() {
		formatter := NewResultFormatter()
		formatter.PrintAll(results)
	}

	// Save to cache if enabled
	if cacheManager.IsEnabled() {
		saveToCache(azureYamlPath, results, allSatisfied, cacheManager)
	}

	return results, allSatisfied
}

// tryGetCachedResults attempts to retrieve and use cached results.
func tryGetCachedResults(azureYamlPath string, cacheManager *cache.CacheManager) ([]ReqResult, bool, bool) {
	cachedResults, valid, err := cacheManager.GetCachedResults(azureYamlPath)
	if err != nil && !output.IsJSON() {
		output.Warning("Failed to read cache: %v", err)
	}

	if !valid || cachedResults == nil {
		return nil, false, false // Cache miss
	}

	// Cache hit
	if !output.IsJSON() {
		output.Info("Using cached reqs check results...")
	}

	// Convert cached results
	results := convertCachedResults(cachedResults.Results)

	// Print cached results
	if !output.IsJSON() {
		formatter := NewResultFormatter()
		formatter.PrintAll(results)
	}

	return results, cachedResults.AllPassed, true
}

// convertCachedResults converts cached results to ReqResult format.
func convertCachedResults(cached []cache.CachedReqResult) []ReqResult {
	results := make([]ReqResult, len(cached))
	for i, c := range cached {
		results[i] = ReqResult{
			Name:       c.Name,
			Installed:  c.Installed,
			Version:    c.Version,
			Required:   c.Required,
			Satisfied:  c.Satisfied,
			Running:    c.Running,
			CheckedRun: c.CheckedRun,
			Message:    c.Message,
		}
	}
	return results
}

// saveToCache saves results to cache with error handling.
func saveToCache(azureYamlPath string, results []ReqResult, allSatisfied bool, cacheManager *cache.CacheManager) {
	cacheResults := make([]cache.CachedReqResult, len(results))
	for i, result := range results {
		cacheResults[i] = cache.CachedReqResult{
			Name:       result.Name,
			Installed:  result.Installed,
			Version:    result.Version,
			Required:   result.Required,
			Satisfied:  result.Satisfied,
			Running:    result.Running,
			CheckedRun: result.CheckedRun,
			Message:    result.Message,
		}
	}

	if err := cacheManager.SaveResults(azureYamlPath, cacheResults, allSatisfied); err != nil && !output.IsJSON() {
		output.Warning("Failed to save cache: %v", err)
	}
}

// performReqsCheck performs fresh reqs checking.
func performReqsCheck(reqs []Prerequisite) ([]ReqResult, bool) {
	checker := NewPrerequisiteChecker()
	results := make([]ReqResult, 0, len(reqs))
	allSatisfied := true

	for _, prereq := range reqs {
		result := checker.Check(prereq)
		results = append(results, result)
		if !result.Satisfied {
			allSatisfied = false
		}
	}

	return results, allSatisfied
}

// ResultFormatter handles formatting of requirement check results.
type ResultFormatter struct{}

// NewResultFormatter creates a new result formatter.
func NewResultFormatter() *ResultFormatter {
	return &ResultFormatter{}
}

// Print formats and prints a single requirement result.
func (rf *ResultFormatter) Print(result ReqResult) {
	if !result.Installed {
		output.ItemError("%s: NOT INSTALLED (required: %s)", result.Name, result.Required)
		return
	}

	if result.Version == "" {
		output.ItemWarning("%s: INSTALLED (version unknown, required: %s)", result.Name, result.Required)
	} else if !result.Satisfied && !result.CheckedRun {
		output.ItemError("%s: %s (required: %s)", result.Name, result.Version, result.Required)
		return
	} else {
		output.ItemSuccess("%s: %s (required: %s)", result.Name, result.Version, result.Required)
	}

	// Check running status if applicable
	if result.CheckedRun {
		rf.printRunningStatus(result.Running)
	}
}

// printRunningStatus prints the running status indicator.
func (rf *ResultFormatter) printRunningStatus(isRunning bool) {
	if isRunning {
		output.Item("- %s✓%s RUNNING", output.Green, output.Reset)
	} else {
		output.Item("- %s✗%s NOT RUNNING", output.Red, output.Reset)
	}
}

// PrintAll formats and prints all requirement results.
func (rf *ResultFormatter) PrintAll(results []ReqResult) {
	for _, result := range results {
		rf.Print(result)
	}
}
