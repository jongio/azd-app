// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/cache"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/types"
	"github.com/jongio/azd-core/security"

	"gopkg.in/yaml.v3"
)

// createCacheManager creates a cache manager with fallback to disabled cache on error.
func createCacheManager(enabled bool) *cache.CacheManager {
	cacheManager, err := cache.NewCacheManagerWithOptions(cache.CacheOptions{
		Enabled: enabled,
		TTL:     0, // Use default TTL
	})
	if err != nil {
		// If cache fails to initialize, proceed without caching (fallback)
		if !cliout.IsJSON() {
			cliout.Warning("Cache initialization failed, proceeding without cache: %v", err)
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
		if errors.Is(err, security.ErrInsecureFilePermissions) {
			// If running in a container environment, print a warning and continue.
			if security.IsContainerEnvironment() {
				fmt.Fprintf(os.Stderr, "Warning: azure.yaml has world-writable permissions (0666). This is common in container environments but consider fixing with: chmod 644 %s\n", azureYamlPath)
			} else {
				return "", nil, fmt.Errorf("insecure file permissions on azure.yaml: %w", err)
			}
		} else {
			return "", nil, fmt.Errorf("insecure file permissions on azure.yaml: %w", err)
		}
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

// handleDepsError returns an error with JSON output if in JSON mode.
func handleDepsError(err error, message string) error {
	fullErr := fmt.Errorf("%s: %w", message, err)
	if cliout.IsJSON() {
		return cliout.PrintJSON(DepsResult{Error: fullErr.Error()})
	}
	return fullErr
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
		if !cliout.IsJSON() {
			cliout.Warning("Failed to read cache: %v", err)
		}
		// In JSON mode, error is still visible in debug/log output but doesn't affect user output
	}

	if !valid || cachedResults == nil {
		return nil, false, false // Cache miss
	}

	// Cache hit
	if !cliout.IsJSON() {
		cliout.Info("Using cached reqs check results...")
	}

	// Convert cached results
	results := convertCachedResults(cachedResults.Results)

	// Print cached results
	if !cliout.IsJSON() {
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

	if err := cacheManager.SaveResults(azureYamlPath, cacheResults, allSatisfied); err != nil && !cliout.IsJSON() {
		cliout.Warning("Failed to save cache: %v", err)
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
		cliout.ItemError("%s: NOT INSTALLED (required: %s)", result.Name, result.Required)
		return
	}

	if result.Version == "" {
		cliout.ItemWarning("%s: INSTALLED (version unknown, required: %s)", result.Name, result.Required)
	} else if !result.Satisfied && !result.CheckedRun {
		cliout.ItemError("%s: %s (required: %s)", result.Name, result.Version, result.Required)
		return
	} else {
		cliout.ItemSuccess("%s: %s (required: %s)", result.Name, result.Version, result.Required)
	}

	// Check running status if applicable
	if result.CheckedRun {
		rf.printRunningStatus(result.Running)
	}
}

// printRunningStatus prints the running status indicator.
func (rf *ResultFormatter) printRunningStatus(isRunning bool) {
	if isRunning {
		cliout.Item("- %s✓%s RUNNING", cliout.Green, cliout.Reset)
	} else {
		cliout.Item("- %s✗%s NOT RUNNING", cliout.Red, cliout.Reset)
	}
}

// PrintAll formats and prints all requirement results.
func (rf *ResultFormatter) PrintAll(results []ReqResult) {
	for _, result := range results {
		rf.Print(result)
	}
}

// cleanDependencies removes existing dependency directories for all detected projects.
func cleanDependencies(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject) error {
	if !cliout.IsJSON() {
		cliout.Newline()
		cliout.Section("🧹", "Cleaning Dependencies")
		cliout.Newline()
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

	if !cliout.IsJSON() && len(errors) == 0 {
		cliout.Newline()
		cliout.Success("Dependencies cleaned successfully")
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

	if !cliout.IsJSON() {
		cliout.Item("Removing %s", path)
	}
	if err := os.RemoveAll(path); err != nil {
		if !cliout.IsJSON() {
			cliout.ItemError("Failed: %v", err)
		}
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}
	if !cliout.IsJSON() {
		cliout.ItemSuccess("Removed successfully")
	}
	return nil
}

// readFileSecure reads a file with security validation.
// #nosec G304 -- Called after security validation
func readFileSecure(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// unmarshalYaml unmarshals YAML data into a struct.
func unmarshalYaml(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
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

// parseAzureYaml parses the azure.yaml file.
func parseAzureYaml(azureYamlPath string) (*service.AzureYaml, error) {
	data, err := readFileSecure(azureYamlPath)
	if err != nil {
		return nil, err
	}
	var azureYaml service.AzureYaml
	if err := unmarshalYaml(data, &azureYaml); err != nil {
		return nil, err
	}
	return &azureYaml, nil
}

// showDryRunSummary displays what would be installed without actually installing.
func showDryRunSummary(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject, searchRoot string) error {
	if cliout.IsJSON() {
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
		return cliout.PrintJSON(DepsResult{
			Success:  true,
			Projects: results,
			Message:  "dry-run: no changes made",
		})
	}

	// Text output
	cliout.Section("📋", "Dry Run - Projects that would be installed")
	cliout.Newline()

	if len(nodeProjects) > 0 {
		cliout.Step("📦", "Node.js projects (%d)", len(nodeProjects))
		for _, p := range nodeProjects {
			relDir := p.Dir
			if rel, err := filepath.Rel(searchRoot, p.Dir); err == nil && rel != "." {
				relDir = rel
			}
			cliout.Item("%s (%s)", relDir, p.PackageManager)
		}
		cliout.Newline()
	}

	if len(pythonProjects) > 0 {
		cliout.Step("🐍", "Python projects (%d)", len(pythonProjects))
		for _, p := range pythonProjects {
			relDir := p.Dir
			if rel, err := filepath.Rel(searchRoot, p.Dir); err == nil && rel != "." {
				relDir = rel
			}
			cliout.Item("%s (%s)", relDir, p.PackageManager)
		}
		cliout.Newline()
	}

	if len(dotnetProjects) > 0 {
		cliout.Step("🔷", ".NET projects (%d)", len(dotnetProjects))
		for _, p := range dotnetProjects {
			relPath := p.Path
			if rel, err := filepath.Rel(searchRoot, p.Path); err == nil && rel != "." {
				relPath = rel
			}
			cliout.Item("%s", relPath)
		}
		cliout.Newline()
	}

	total := len(nodeProjects) + len(pythonProjects) + len(dotnetProjects)
	cliout.Info("Total: %d project(s) would be installed", total)
	cliout.Info("Run without --dry-run to install dependencies")

	return nil
}

// handleNoProjectsCase handles the case when no projects are detected.
func handleNoProjectsCase(searchRoot string, serviceFilter []string) error {
	// If user specified services but none matched, show a helpful message
	if len(serviceFilter) > 0 {
		msg := fmt.Sprintf("No projects found matching services: %v", serviceFilter)
		if cliout.IsJSON() {
			return cliout.PrintJSON(DepsResult{
				Success:  true,
				Projects: []InstallResult{},
				Message:  msg,
			})
		}
		cliout.Info("%s", msg)
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

	if cliout.IsJSON() {
		return cliout.PrintJSON(DepsResult{
			Success:  true,
			Projects: []InstallResult{},
			Message:  msgNoProjectsDetected,
		})
	}

	// Only show "No projects detected" if it's not a Logic Apps-only workspace
	if !hasLogicAppsOnly {
		cliout.Info("%s", msgNoProjectsDetected)
	}
	return nil
}
