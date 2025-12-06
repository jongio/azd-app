// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/orchestrator"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/types"
	"github.com/jongio/azd-app/cli/src/internal/workspace"

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

// CleanDependenciesError represents an error during dependency cleaning with details.
type CleanDependenciesError struct {
	Count   int
	Details []string
}

// Error implements the error interface.
func (e *CleanDependenciesError) Error() string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("encountered %d error(s) while cleaning dependencies", e.Count)
	}
	if len(e.Details) == 1 {
		return fmt.Sprintf("failed to clean dependencies: %s", e.Details[0])
	}
	return fmt.Sprintf("encountered %d error(s) while cleaning dependencies:\n  - %s",
		e.Count, strings.Join(e.Details, "\n  - "))
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

	// Validate file permissions for security
	if err := security.ValidateFilePermissions(azureYamlPath); err != nil {
		return "", nil, fmt.Errorf("insecure file permissions on azure.yaml: %w", err)
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
	output.CommandHeader("reqs", "Check required prerequisites")
	// Load azure.yaml
	azureYamlPath, azureYaml, err := loadAzureYaml()
	if err != nil {
		return err
	}

	// If no reqs section exists, skip checks gracefully
	if len(azureYaml.Reqs) == 0 {
		if output.IsJSON() {
			return output.PrintJSON(ReqsResult{
				Satisfied: true,
				Reqs:      []ReqResult{},
			})
		}
		return nil
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
		output.Info("%s If you recently installed any missing tools, run 'azd app reqs --fix' to refresh PATH", output.IconBulb)
		return fmt.Errorf("requirement check failed")
	}

	output.Success("All reqs satisfied!")
	return nil
}

// DependencyInstaller handles installation of project dependencies.
type DependencyInstaller struct {
	searchRoot     string
	nodeProjects   []types.NodeProject   // Pre-filtered Node.js projects (optional)
	pythonProjects []types.PythonProject // Pre-filtered Python projects (optional)
	dotnetProjects []types.DotnetProject // Pre-filtered .NET projects (optional)
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
// Returns results for all attempted installations and any detection errors.
func (di *DependencyInstaller) InstallAll() ([]InstallResult, error) {
	var results []InstallResult
	var detectionErrors []error

	// Install Node.js dependencies
	nodeResults, err := di.installNodeProjects()
	if err != nil {
		detectionErrors = append(detectionErrors, fmt.Errorf("node detection: %w", err))
	}
	results = append(results, nodeResults...)

	// Install Python dependencies
	pythonResults, err := di.installPythonProjects()
	if err != nil {
		detectionErrors = append(detectionErrors, fmt.Errorf("python detection: %w", err))
	}
	results = append(results, pythonResults...)

	// Install .NET dependencies
	dotnetResults, err := di.installDotnetProjects()
	if err != nil {
		detectionErrors = append(detectionErrors, fmt.Errorf("dotnet detection: %w", err))
	}
	results = append(results, dotnetResults...)

	// Return combined detection errors if any occurred
	if len(detectionErrors) > 0 {
		errMsgs := make([]string, len(detectionErrors))
		for i, e := range detectionErrors {
			errMsgs[i] = e.Error()
		}
		return results, fmt.Errorf("detection errors: %s", strings.Join(errMsgs, "; "))
	}

	return results, nil
}

// InstallAllFiltered installs dependencies for pre-filtered projects.
// Use this when projects have already been detected and filtered (e.g., by service name).
func (di *DependencyInstaller) InstallAllFiltered() ([]InstallResult, error) {
	var results []InstallResult

	// Install Node.js dependencies from pre-filtered list
	if len(di.nodeProjects) > 0 {
		nodeResults := di.installNodeProjectList(di.nodeProjects)
		results = append(results, nodeResults...)
	}

	// Install Python dependencies from pre-filtered list
	if len(di.pythonProjects) > 0 {
		pythonResults := di.installPythonProjectList(di.pythonProjects)
		results = append(results, pythonResults...)
	}

	// Install .NET dependencies from pre-filtered list
	if len(di.dotnetProjects) > 0 {
		dotnetResults := di.installDotnetProjectList(di.dotnetProjects)
		results = append(results, dotnetResults...)
	}

	return results, nil
}

// installNodeProjectList installs dependencies for a list of Node.js projects.
func (di *DependencyInstaller) installNodeProjectList(nodeProjects []types.NodeProject) []InstallResult {
	var results []InstallResult
	for _, nodeProject := range nodeProjects {
		result := di.installProject("node", nodeProject.Dir, nodeProject.PackageManager, func() error {
			return installer.InstallNodeDependencies(nodeProject)
		})
		results = append(results, result)
	}
	return results
}

// installPythonProjectList installs dependencies for a list of Python projects.
func (di *DependencyInstaller) installPythonProjectList(pythonProjects []types.PythonProject) []InstallResult {
	var results []InstallResult
	for _, pyProject := range pythonProjects {
		result := di.installProject("python", pyProject.Dir, pyProject.PackageManager, func() error {
			return installer.SetupPythonVirtualEnv(pyProject)
		})
		results = append(results, result)
	}
	return results
}

// installDotnetProjectList installs dependencies for a list of .NET projects.
func (di *DependencyInstaller) installDotnetProjectList(dotnetProjects []types.DotnetProject) []InstallResult {
	var results []InstallResult
	for _, dotnetProject := range dotnetProjects {
		result := di.installProject("dotnet", filepath.Dir(dotnetProject.Path), "dotnet", func() error {
			return installer.RestoreDotnetProject(dotnetProject)
		})
		// For dotnet, we use Path instead of Dir in the result
		result.Path = dotnetProject.Path
		result.Dir = ""
		results = append(results, result)
	}
	return results
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
	// Get options set by the command
	opts := GetDepsOptions()

	// Create executor with production dependencies and execute
	executor := newDepsExecutor(opts)
	return executor.execute()
}

// showDryRunSummary displays what would be installed without actually installing.
func showDryRunSummary(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject, searchRoot string) error {
	if output.IsJSON() {
		// Build dry-run results
		var results []InstallResult
		for _, p := range nodeProjects {
			results = append(results, InstallResult{
				Type:    "node",
				Dir:     p.Dir,
				Manager: p.PackageManager,
				Success: true, // Would succeed (dry-run)
			})
		}
		for _, p := range pythonProjects {
			results = append(results, InstallResult{
				Type:    "python",
				Dir:     p.Dir,
				Manager: p.PackageManager,
				Success: true,
			})
		}
		for _, p := range dotnetProjects {
			results = append(results, InstallResult{
				Type:    "dotnet",
				Path:    p.Path,
				Success: true,
			})
		}
		return output.PrintJSON(DepsResult{
			Success:  true,
			Projects: results,
			Message:  "dry-run: no changes made",
		})
	}

	// Text output
	output.Section("📋", "Dry Run - Projects that would be installed")
	output.Newline()

	if len(nodeProjects) > 0 {
		output.Step("📦", "Node.js projects (%d)", len(nodeProjects))
		for _, p := range nodeProjects {
			relDir := p.Dir
			if rel, err := filepath.Rel(searchRoot, p.Dir); err == nil && rel != "." {
				relDir = rel
			}
			output.Item("%s (%s)", relDir, p.PackageManager)
		}
		output.Newline()
	}

	if len(pythonProjects) > 0 {
		output.Step("🐍", "Python projects (%d)", len(pythonProjects))
		for _, p := range pythonProjects {
			relDir := p.Dir
			if rel, err := filepath.Rel(searchRoot, p.Dir); err == nil && rel != "." {
				relDir = rel
			}
			output.Item("%s (%s)", relDir, p.PackageManager)
		}
		output.Newline()
	}

	if len(dotnetProjects) > 0 {
		output.Step("🔷", ".NET projects (%d)", len(dotnetProjects))
		for _, p := range dotnetProjects {
			relPath := p.Path
			if rel, err := filepath.Rel(searchRoot, p.Path); err == nil && rel != "." {
				relPath = rel
			}
			output.Item("%s", relPath)
		}
		output.Newline()
	}

	total := len(nodeProjects) + len(pythonProjects) + len(dotnetProjects)
	output.Info("Total: %d project(s) would be installed", total)
	output.Info("Run without --dry-run to install dependencies")

	return nil
}

// handleDepsError returns an error with JSON output if in JSON mode.
func handleDepsError(err error, message string) error {
	fullErr := fmt.Errorf("%s: %w", message, err)
	if output.IsJSON() {
		return output.PrintJSON(DepsResult{Error: fullErr.Error()})
	}
	return fullErr
}

// handleNoProjectsCase handles the case when no projects are detected.
func handleNoProjectsCase(searchRoot string, serviceFilter []string) error {
	// If user specified services but none matched, show a helpful message
	if len(serviceFilter) > 0 {
		msg := fmt.Sprintf("No projects found matching services: %v", serviceFilter)
		if output.IsJSON() {
			return output.PrintJSON(DepsResult{
				Success:  true,
				Projects: []InstallResult{},
				Message:  msg,
			})
		}
		output.Info("%s", msg)
		return nil
	}

	// Check if there are Logic Apps projects (which don't need dependency installation)
	functionApps, _ := detector.FindFunctionApps(searchRoot)
	hasLogicAppsOnly := false
	if len(functionApps) > 0 {
		hasLogicAppsOnly = true
		for _, app := range functionApps {
			if app.Variant != "logicapps" {
				hasLogicAppsOnly = false
				break
			}
		}
	}

	if output.IsJSON() {
		return output.PrintJSON(DepsResult{
			Success:  true,
			Projects: []InstallResult{},
			Message:  msgNoProjectsDetected,
		})
	}

	// Only show "No projects detected" if it's not a Logic Apps-only workspace
	if !hasLogicAppsOnly {
		output.Info("%s", msgNoProjectsDetected)
	}
	return nil
}

// filterProjectsByService filters projects to only include those matching the specified service names.
func filterProjectsByService(
	nodeProjects []types.NodeProject,
	pythonProjects []types.PythonProject,
	dotnetProjects []types.DotnetProject,
	services []string,
	searchRoot string,
) ([]types.NodeProject, []types.PythonProject, []types.DotnetProject) {
	// Build a set of service paths from azure.yaml
	servicePaths := make(map[string]bool)

	azureYamlPath, err := detector.FindAzureYaml(searchRoot)
	if err != nil || azureYamlPath == "" {
		// No azure.yaml found, can't filter by service
		return nodeProjects, pythonProjects, dotnetProjects
	}

	azureYaml, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		return nodeProjects, pythonProjects, dotnetProjects
	}

	azureYamlDir := filepath.Dir(azureYamlPath)

	// Build map of service name to absolute path
	for name, svc := range azureYaml.Services {
		// Check if this service is in the filter list
		for _, filterName := range services {
			if name == filterName {
				svcPath := filepath.Join(azureYamlDir, svc.Project)
				absPath, err := filepath.Abs(svcPath)
				if err != nil {
					// Log warning but continue processing other services
					if !output.IsJSON() {
						output.Warning("Failed to resolve absolute path for service %s: %v", name, err)
					}
					continue
				}
				servicePaths[absPath] = true
				break
			}
		}
	}

	// Filter Node.js projects
	var filteredNode []types.NodeProject
	for _, p := range nodeProjects {
		absDir, _ := filepath.Abs(p.Dir)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredNode = append(filteredNode, p)
		}
	}

	// Filter Python projects
	var filteredPython []types.PythonProject
	for _, p := range pythonProjects {
		absDir, _ := filepath.Abs(p.Dir)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredPython = append(filteredPython, p)
		}
	}

	// Filter .NET projects
	var filteredDotnet []types.DotnetProject
	for _, p := range dotnetProjects {
		absPath, _ := filepath.Abs(p.Path)
		absDir := filepath.Dir(absPath)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredDotnet = append(filteredDotnet, p)
		}
	}

	return filteredNode, filteredPython, filteredDotnet
}

// isSubdirectory checks if path is a subdirectory of any path in the set.
// Uses filepath.Rel for cross-platform path comparison.
func isSubdirectory(path string, parentPaths map[string]bool) bool {
	// Normalize the path
	path = filepath.Clean(path)
	for parent := range parentPaths {
		parent = filepath.Clean(parent)
		// Skip if path equals parent (we want strict subdirectory)
		if path == parent {
			continue
		}
		// Use filepath.Rel to check if path is relative to parent
		rel, err := filepath.Rel(parent, path)
		if err != nil {
			continue
		}
		// If relative path doesn't start with "..", it's a subdirectory
		// Check for both ".." prefix and "." to prevent path traversal
		if !strings.HasPrefix(rel, "..") && rel != "." {
			return true
		}
	}
	return false
}

// parseAzureYaml parses the azure.yaml file.
func parseAzureYaml(azureYamlPath string) (*service.AzureYaml, error) {
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return nil, err
	}
	var azureYaml service.AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return nil, err
	}
	return &azureYaml, nil
}

// runParallelInstallation runs the parallel installer for non-JSON mode.
func runParallelInstallation(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject, verbose bool) error {
	parallelInstaller := installer.NewParallelInstaller()
	parallelInstaller.Verbose = verbose

	// Handle npm/yarn/pnpm workspace scenarios using workspace handler
	// When a workspace root exists, only install at the root level to avoid race conditions
	// on Windows where parallel npm installs compete for the same node_modules directory
	workspaceHandler := workspace.NewHandler()
	filteredNodeProjects := workspaceHandler.FilterNodeProjects(nodeProjects)

	for _, project := range filteredNodeProjects {
		parallelInstaller.AddNodeProject(project)
	}
	for _, project := range pythonProjects {
		parallelInstaller.AddPythonProject(project)
	}
	for _, project := range dotnetProjects {
		parallelInstaller.AddDotnetProject(project)
	}

	// Run all installations in parallel
	if err := parallelInstaller.Run(); err != nil {
		return err
	}

	// Check for failures
	if parallelInstaller.HasFailures() {
		failedProjects := parallelInstaller.FailedProjects()
		if len(failedProjects) > 0 {
			return fmt.Errorf("failed to install %d of %d projects: %v", len(failedProjects), parallelInstaller.TotalProjects(), failedProjects)
		}
		return fmt.Errorf("some installations failed")
	}

	return nil
}

// runJSONInstallation runs installation in JSON mode with sequential output.
func runJSONInstallation(searchRoot string, nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject) error {
	depInstaller := NewDependencyInstaller(searchRoot)
	depInstaller.nodeProjects = nodeProjects
	depInstaller.pythonProjects = pythonProjects
	depInstaller.dotnetProjects = dotnetProjects

	results, err := depInstaller.InstallAllFiltered()
	if err != nil {
		return err
	}

	allSuccess := checkAllSuccess(results)
	return output.PrintJSON(DepsResult{
		Success:  allSuccess,
		Projects: results,
	})
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
	// The actual run logic is handled by the run command's RunE function
	// This is just a marker to ensure the dependency chain is executed
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

	// Perform fresh check (output is shown inline during checks)
	results, allSatisfied := performReqsCheck(reqs)

	// Save to cache if enabled
	if cacheManager.IsEnabled() {
		saveToCache(azureYamlPath, results, allSatisfied, cacheManager)
	}

	return results, allSatisfied
}

// tryGetCachedResults attempts to retrieve and use cached results.
func tryGetCachedResults(azureYamlPath string, cacheManager *cache.CacheManager) ([]ReqResult, bool, bool) {
	cachedResults, valid, err := cacheManager.GetCachedResults(azureYamlPath)
	if err != nil {
		// Log cache read errors in both JSON and non-JSON modes for visibility
		if !output.IsJSON() {
			output.Warning("Failed to read cache: %v", err)
		}
		// In JSON mode, error is still visible in debug/log output but doesn't affect user output
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

// detectAllProjects detects all project types in the given directory.
// This is a convenience wrapper for testing and backward compatibility.
func detectAllProjects(searchRoot string) ([]types.NodeProject, []types.PythonProject, []types.DotnetProject, error) {
	nodeProjects, err := detector.FindNodeProjects(searchRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to detect Node.js projects: %w", err)
	}

	pythonProjects, err := detector.FindPythonProjects(searchRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to detect Python projects: %w", err)
	}

	dotnetProjects, err := detector.FindDotnetProjects(searchRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to detect .NET projects: %w", err)
	}

	return nodeProjects, pythonProjects, dotnetProjects, nil
}

// cleanDependencies removes existing dependency directories for all detected projects.
// cleanDirectory removes a directory if it exists and logs the operation.
// Returns an error if removal fails.
func cleanDirectory(path string) error {
	if _, err := os.Stat(path); err != nil {
		return nil // Directory doesn't exist, nothing to clean
	}

	// Validate that we're only cleaning expected dependency directories
	// to prevent accidental deletion of important files
	dirName := filepath.Base(path)
	validDirs := map[string]bool{
		"node_modules":  true,
		".venv":         true,
		"obj":           true,
		"bin":           true,
		"__pycache__":   true,
		".pytest_cache": true,
	}

	if !validDirs[dirName] {
		return fmt.Errorf("refusing to clean unexpected directory: %s (only dependency directories are allowed)", path)
	}

	if !output.IsJSON() {
		output.Item("Removing %s", path)
	}
	if err := os.RemoveAll(path); err != nil {
		if !output.IsJSON() {
			output.ItemError("Failed: %v", err)
		}
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}
	if !output.IsJSON() {
		output.ItemSuccess("Removed successfully")
	}
	return nil
}

func cleanDependencies(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject) error {
	if !output.IsJSON() {
		output.Newline()
		output.Section("🧹", "Cleaning Dependencies")
		output.Newline()
	}

	var errors []error

	// Clean Node.js projects
	for _, project := range nodeProjects {
		nodeModulesPath := filepath.Join(project.Dir, "node_modules")
		if err := cleanDirectory(nodeModulesPath); err != nil {
			errors = append(errors, err)
		}
	}

	// Clean Python projects
	for _, project := range pythonProjects {
		venvPath := filepath.Join(project.Dir, ".venv")
		if err := cleanDirectory(venvPath); err != nil {
			errors = append(errors, err)
		}
	}

	// Clean .NET projects (obj and bin directories)
	for _, project := range dotnetProjects {
		projectDir := filepath.Dir(project.Path)
		for _, dir := range []string{"obj", "bin"} {
			dirPath := filepath.Join(projectDir, dir)
			if err := cleanDirectory(dirPath); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if !output.IsJSON() && len(errors) == 0 {
		output.Newline()
		output.Success("Dependencies cleaned successfully")
	}

	if len(errors) > 0 {
		// Build detailed error message with all failures
		var errorDetails []string
		for _, err := range errors {
			errorDetails = append(errorDetails, err.Error())
		}
		return &CleanDependenciesError{
			Count:   len(errors),
			Details: errorDetails,
		}
	}

	return nil
}
