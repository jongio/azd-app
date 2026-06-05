// Package commands provides the command-line interface for the azd-app CLI.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/installer"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/workspace"
	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
)

// DependencyInstaller handles installation of project dependencies.
type DependencyInstaller struct {
	searchRoot     string
	projects       DetectedProjects // Pre-filtered projects (optional)
	nodeProjects   []types.NodeProject
	pythonProjects []types.PythonProject
	dotnetProjects []types.DotnetProject
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

	nodeProjects := di.projects.Node
	pythonProjects := di.projects.Python
	dotnetProjects := di.projects.Dotnet
	if nodeProjects == nil {
		nodeProjects = di.nodeProjects
	}
	if pythonProjects == nil {
		pythonProjects = di.pythonProjects
	}
	if dotnetProjects == nil {
		dotnetProjects = di.dotnetProjects
	}

	if len(nodeProjects) > 0 {
		results = append(results, di.installNodeProjectList(nodeProjects)...)
	}
	if len(pythonProjects) > 0 {
		results = append(results, di.installPythonProjectList(pythonProjects)...)
	}
	if len(dotnetProjects) > 0 {
		results = append(results, di.installDotnetProjectList(dotnetProjects)...)
	}

	return results, nil
}

// installNodeProjectList installs dependencies for a list of Node.js projects.
func (di *DependencyInstaller) installNodeProjectList(nodeProjects []types.NodeProject) []InstallResult {
	results := make([]InstallResult, 0, len(nodeProjects))
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
	results := make([]InstallResult, 0, len(pythonProjects))
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
	results := make([]InstallResult, 0, len(dotnetProjects))
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

	if !cliout.IsJSON() {
		cliout.Step("📦", "Found %s Node.js project(s)", cliout.Count(len(nodeProjects)))
	}

	results := make([]InstallResult, 0, len(nodeProjects))
	for _, nodeProject := range nodeProjects {
		result := di.installProject("node", nodeProject.Dir, nodeProject.PackageManager, func() error {
			return installer.InstallNodeDependencies(nodeProject)
		})
		results = append(results, result)
	}

	if !cliout.IsJSON() {
		cliout.Newline()
	}

	return results, nil
}

// installPythonProjects installs dependencies for Python projects.
func (di *DependencyInstaller) installPythonProjects() ([]InstallResult, error) {
	pythonProjects, err := detector.FindPythonProjects(di.searchRoot)
	if err != nil || len(pythonProjects) == 0 {
		return nil, err
	}

	if !cliout.IsJSON() {
		cliout.Step("🐍", "Found %s Python project(s)", cliout.Count(len(pythonProjects)))
	}

	results := make([]InstallResult, 0, len(pythonProjects))
	for _, pyProject := range pythonProjects {
		result := di.installProject("python", pyProject.Dir, pyProject.PackageManager, func() error {
			return installer.SetupPythonVirtualEnv(pyProject)
		})
		results = append(results, result)
	}

	if !cliout.IsJSON() {
		cliout.Newline()
	}

	return results, nil
}

// installDotnetProjects installs dependencies for .NET projects.
func (di *DependencyInstaller) installDotnetProjects() ([]InstallResult, error) {
	dotnetProjects, err := detector.FindDotnetProjects(di.searchRoot)
	if err != nil || len(dotnetProjects) == 0 {
		return nil, err
	}

	if !cliout.IsJSON() {
		cliout.Step("🔷", "Found %s .NET project(s)", cliout.Count(len(dotnetProjects)))
	}

	results := make([]InstallResult, 0, len(dotnetProjects))
	for _, dotnetProject := range dotnetProjects {
		result := InstallResult{
			Type: "dotnet",
			Path: dotnetProject.Path,
		}
		if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
			if !cliout.IsJSON() {
				cliout.ItemWarning("Failed to restore %s: %v", dotnetProject.Path, err)
			}
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}

	if !cliout.IsJSON() {
		cliout.Newline()
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
	if !cliout.IsJSON() {
		relDir := dir
		if rel, err := filepath.Rel(di.searchRoot, dir); err == nil && rel != "." {
			relDir = rel
		}
		cliout.Item("Installing %s (%s)", relDir, manager)
	}

	if err := installFunc(); err != nil {
		if !cliout.IsJSON() {
			cliout.ItemWarning("Failed to install for %s: %v", dir, err)
		}
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}
	return result
}

// filterDetectedProjectsByService filters grouped projects to only include those matching the specified service names.
func filterDetectedProjectsByService(projects DetectedProjects, services []string, searchRoot string) DetectedProjects {
	// Build a set of service paths from azure.yaml
	servicePaths := make(map[string]bool)

	azureYamlPath, err := detector.FindAzureYaml(searchRoot)
	if err != nil || azureYamlPath == "" {
		// No azure.yaml found, can't filter by service
		return projects
	}

	azureYaml, err := parseAzureYaml(azureYamlPath)
	if err != nil {
		return projects
	}

	// Build map of service name to absolute path
	for name, svc := range azureYaml.Services {
		for _, filterName := range services {
			if name != filterName {
				continue
			}

			absPath, err := filepath.Abs(svc.Project)
			if err != nil {
				if !cliout.IsJSON() {
					cliout.Warning("Failed to resolve absolute path for service %s: %v", name, err)
				}
				continue
			}
			servicePaths[absPath] = true
			break
		}
	}

	// Filter Node.js projects
	var filteredNode []types.NodeProject
	for _, p := range projects.Node {
		absDir, _ := filepath.Abs(p.Dir)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredNode = append(filteredNode, p)
		}
	}

	// Filter Python projects
	var filteredPython []types.PythonProject
	for _, p := range projects.Python {
		absDir, _ := filepath.Abs(p.Dir)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredPython = append(filteredPython, p)
		}
	}

	// Filter .NET projects
	var filteredDotnet []types.DotnetProject
	for _, p := range projects.Dotnet {
		absPath, _ := filepath.Abs(p.Path)
		absDir := filepath.Dir(absPath)
		if servicePaths[absDir] || isSubdirectory(absDir, servicePaths) {
			filteredDotnet = append(filteredDotnet, p)
		}
	}

	return DetectedProjects{
		Node:   filteredNode,
		Python: filteredPython,
		Dotnet: filteredDotnet,
	}
}

// filterProjectsByService preserves the legacy test-facing signature while delegating to grouped project filtering.
func filterProjectsByService(nodeProjects []types.NodeProject, pythonProjects []types.PythonProject, dotnetProjects []types.DotnetProject, services []string, searchRoot string) ([]types.NodeProject, []types.PythonProject, []types.DotnetProject) {
	filtered := filterDetectedProjectsByService(DetectedProjects{
		Node:   nodeProjects,
		Python: pythonProjects,
		Dotnet: dotnetProjects,
	}, services, searchRoot)

	return filtered.Node, filtered.Python, filtered.Dotnet
}

// detectProjectsFromAzureYaml reads azure.yaml and detects project types directly from
// service project paths, without walking the entire directory tree.
// Returns an error if no azure.yaml is found or no services are defined.
func detectProjectsFromAzureYaml(searchRoot string) ([]types.NodeProject, []types.PythonProject, []types.DotnetProject, error) {
	azureYamlPath, err := detector.FindAzureYaml(searchRoot)
	if err != nil || azureYamlPath == "" {
		return nil, nil, nil, fmt.Errorf("azure.yaml not found - create one with a 'services' section to define your development environment")
	}

	azureYaml, err := service.ParseAzureYaml(filepath.Dir(azureYamlPath))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if !service.HasServices(azureYaml) {
		return nil, nil, nil, fmt.Errorf("no services defined in azure.yaml - add a 'services' section to define your development environment")
	}

	// Resolve the project root to an absolute path for containment checks.
	// EvalSymlinks is required on platforms like macOS where temp dirs use
	// symlinks (e.g., /var → /private/var); ParseAzureYaml returns resolved
	// paths, so the root must also be resolved for Rel to work correctly.
	absSearchRoot, err := filepath.Abs(searchRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve project root: %w", err)
	}
	if resolved, symlinkErr := filepath.EvalSymlinks(absSearchRoot); symlinkErr == nil {
		absSearchRoot = resolved
	}

	var nodeProjects []types.NodeProject
	var pythonProjects []types.PythonProject
	var dotnetProjects []types.DotnetProject

	for _, svc := range azureYaml.Services {
		projectDir := svc.Project
		if projectDir == "" {
			continue
		}

		// Validate the project path stays within the project root (prevent path traversal)
		absProjectDir, err := filepath.Abs(projectDir)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve service project path %q: %w", projectDir, err)
		}
		rel, err := filepath.Rel(absSearchRoot, absProjectDir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, nil, nil, fmt.Errorf("service project path %q resolves outside the project root - check the 'project' path in azure.yaml", projectDir)
		}

		// Verify the project directory exists
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return nil, nil, nil, fmt.Errorf("service project directory %q does not exist - check the 'project' path in azure.yaml", projectDir)
		}

		// Check for Node.js project (package.json)
		if _, err := os.Stat(filepath.Join(projectDir, "package.json")); err == nil {
			pm := detector.DetectNodePackageManager(projectDir)
			isWorkspaceRoot := detector.HasNpmWorkspaces(projectDir)
			nodeProjects = append(nodeProjects, types.NodeProject{
				Dir:             projectDir,
				PackageManager:  pm,
				IsWorkspaceRoot: isWorkspaceRoot,
			})
			continue
		}

		// Check for Python project
		pythonFiles := []string{"requirements.txt", "pyproject.toml", "poetry.lock", "uv.lock", "Pipfile"}
		isPython := false
		for _, f := range pythonFiles {
			if _, err := os.Stat(filepath.Join(projectDir, f)); err == nil {
				isPython = true
				break
			}
		}
		if isPython {
			pm := detector.DetectPythonPackageManager(projectDir)
			pythonProjects = append(pythonProjects, types.PythonProject{
				Dir:            projectDir,
				PackageManager: pm,
			})
			continue
		}

		// Check for .NET project (*.csproj or *.sln in the directory)
		entries, err := os.ReadDir(projectDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				ext := filepath.Ext(entry.Name())
				if ext == ".csproj" || ext == ".sln" {
					dotnetProjects = append(dotnetProjects, types.DotnetProject{
						Path: filepath.Join(projectDir, entry.Name()),
					})
					break
				}
			}
		}
	}

	// Link workspace children to their workspace roots.
	// When services are defined in azure.yaml, the workspace root may not be
	// listed as a service itself. Detect workspace roots by walking up from each
	// project directory, and add implicit workspace root projects so that the
	// workspace handler can correctly deduplicate installs.
	nodeProjects = linkWorkspaceChildren(nodeProjects, absSearchRoot)

	return nodeProjects, pythonProjects, dotnetProjects, nil
}

// linkWorkspaceChildren finds workspace roots for Node.js projects and links
// children to their roots. If a workspace root is not already in the projects
// list, it is added automatically. This ensures FilterNodeProjects can correctly
// skip workspace children and install only at the root.
func linkWorkspaceChildren(projects []types.NodeProject, searchRoot string) []types.NodeProject {
	if len(projects) == 0 {
		return projects
	}

	// Collect existing workspace roots, mapping absolute path to original Dir value.
	// This ensures children's WorkspaceRoot matches the root's Dir in FilterNodeProjects.
	workspaceRootDirs := make(map[string]string) // abs path → project.Dir
	for _, p := range projects {
		if p.IsWorkspaceRoot {
			absDir, err := filepath.Abs(p.Dir)
			if err != nil {
				continue
			}
			workspaceRootDirs[absDir] = p.Dir
		}
	}

	// For each non-root project, walk up to find its workspace root
	discoveredRoots := make(map[string]bool)
	for i := range projects {
		if projects[i].IsWorkspaceRoot {
			continue
		}

		absDir, err := filepath.Abs(projects[i].Dir)
		if err != nil {
			continue
		}

		// Walk up from project dir looking for a workspace root
		root := findWorkspaceRootUpward(absDir, searchRoot)
		if root == "" {
			continue
		}

		if origDir, exists := workspaceRootDirs[root]; exists {
			// Root already in project list; use its original Dir for path consistency
			projects[i].WorkspaceRoot = origDir
		} else {
			// Root not in project list; use absolute path (will match added root's Dir)
			projects[i].WorkspaceRoot = root
			discoveredRoots[root] = true
		}
	}

	// Add discovered workspace roots that aren't already in the list
	for root := range discoveredRoots {
		pm := detector.DetectNodePackageManager(root)
		projects = append([]types.NodeProject{{
			Dir:             root,
			PackageManager:  pm,
			IsWorkspaceRoot: true,
		}}, projects...)
	}

	return projects
}

// findWorkspaceRootUpward walks up from startDir toward boundaryDir looking for
// a directory that contains pnpm-workspace.yaml or a package.json with workspaces.
// Returns the absolute path of the workspace root, or empty string if not found.
func findWorkspaceRootUpward(startDir, boundaryDir string) string {
	dir := startDir
	for {
		// Don't search above the boundary (project root)
		rel, err := filepath.Rel(boundaryDir, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return ""
		}

		if detector.HasNpmWorkspaces(dir) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return ""
		}
		dir = parent
	}
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
func runParallelInstallation(projects DetectedProjects, verbose bool) error {
	parallelInstaller := installer.NewParallelInstaller()
	parallelInstaller.Verbose = verbose

	// Handle npm/yarn/pnpm workspace scenarios using workspace handler
	// When a workspace root exists, only install at the root level to avoid race conditions
	// on Windows where parallel npm installs compete for the same node_modules directory
	workspaceHandler := workspace.NewHandler()
	filteredNodeProjects := workspaceHandler.FilterNodeProjects(projects.Node)

	for _, project := range filteredNodeProjects {
		parallelInstaller.AddNodeProject(project)
	}
	for _, project := range projects.Python {
		parallelInstaller.AddPythonProject(project)
	}
	for _, project := range projects.Dotnet {
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

// runJSONInstallation runs installation in JSON mode with sequential cliout.
func runJSONInstallation(searchRoot string, projects DetectedProjects) error {
	depInstaller := NewDependencyInstaller(searchRoot)
	depInstaller.projects = projects

	results, err := depInstaller.InstallAllFiltered()
	if err != nil {
		return err
	}

	allSuccess := checkAllSuccess(results)
	return cliout.PrintJSON(DepsResult{
		Success:  allSuccess,
		Projects: results,
	})
}
