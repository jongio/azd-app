// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/types"
	"github.com/jongio/azd-app/cli/src/internal/workspace"
)

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
