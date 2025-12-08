package detector

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// FindNodeProjects searches for package.json files.
// Only searches within rootDir and does not traverse outside it.
// Detects npm/yarn/pnpm workspace configurations and marks workspace relationships.
func FindNodeProjects(rootDir string) ([]types.NodeProject, error) {
	var nodeProjects []types.NodeProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nodeProjects, err
	}

	// Track workspace root directories
	workspaceRoots := make(map[string]bool)

	// First pass: find all package.json files and identify workspace roots
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Ensure we don't traverse outside rootDir
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, absPath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			name := info.Name()
			// Skip common directories
			if name == skipDirNodeModules || name == skipDirGit || name == skipDirBin || name == skipDirObj {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() && info.Name() == "package.json" {
			dir := filepath.Dir(path)

			if seen[dir] {
				return nil
			}

			// Skip Logic Apps projects (they have package.json for npm scripts but aren't Node.js projects)
			if isLogicAppsDirectory(dir) {
				return nil
			}

			packageManager := DetectNodePackageManagerWithBoundary(dir, rootDir)
			isWorkspaceRoot := HasNpmWorkspaces(dir)

			nodeProjects = append(nodeProjects, types.NodeProject{
				Dir:             dir,
				PackageManager:  packageManager,
				IsWorkspaceRoot: isWorkspaceRoot,
			})
			seen[dir] = true

			// Track workspace roots
			if isWorkspaceRoot {
				workspaceRoots[dir] = true
			}
		}

		return nil
	})

	// Second pass: identify workspace children and link them to their workspace root
	for i := range nodeProjects {
		if !nodeProjects[i].IsWorkspaceRoot {
			// Check if this project is within a workspace root
			projectDir := nodeProjects[i].Dir
			for workspaceRoot := range workspaceRoots {
				// Check if projectDir is within workspaceRoot
				relPath, err := filepath.Rel(workspaceRoot, projectDir)
				if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
					// This project is a child of the workspace
					nodeProjects[i].WorkspaceRoot = workspaceRoot
					break
				}
			}
		}
	}

	return nodeProjects, err
}

// DetectNodePackageManager determines whether to use pnpm, yarn, or npm.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManager(projectDir string) string {
	// Use unbounded search (for backward compatibility with tests)
	return DetectNodePackageManagerWithBoundary(projectDir, "")
}

// DetectNodePackageManagerWithSource returns both the package manager and detection source.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManagerWithSource(projectDir string) PackageManagerInfo {
	return DetectNodePackageManagerWithBoundaryAndSource(projectDir, projectDir)
}

// DetectNodePackageManagerWithBoundary determines package manager by checking only the project directory.
// Does not search up the directory tree to avoid interference from parent workspace configurations.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManagerWithBoundary(projectDir string, boundaryDir string) string {
	info := DetectNodePackageManagerWithBoundaryAndSource(projectDir, boundaryDir)
	return info.Name
}

// DetectNodePackageManagerWithBoundaryAndSource determines package manager and source by checking only the project directory.
// Does not search up the directory tree to avoid interference from parent workspace configurations.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManagerWithBoundaryAndSource(projectDir string, boundaryDir string) PackageManagerInfo {
	// Clean the paths to absolute
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		absDir = projectDir
	}

	// First, check for packageManager field in package.json (highest priority)
	if pkgMgr := GetPackageManagerFromPackageJSON(absDir); pkgMgr != "" {
		return PackageManagerInfo{
			Name:   pkgMgr,
			Source: "package.json (packageManager field)",
		}
	}

	// Fall back to lock file detection
	// Priority: pnpm-lock.yaml > pnpm-workspace.yaml > yarn.lock > package-lock.json > npm (default)
	if _, err := os.Stat(filepath.Join(absDir, "pnpm-lock.yaml")); err == nil {
		return PackageManagerInfo{Name: "pnpm", Source: "pnpm-lock.yaml"}
	}
	if _, err := os.Stat(filepath.Join(absDir, "pnpm-workspace.yaml")); err == nil {
		return PackageManagerInfo{Name: "pnpm", Source: "pnpm-workspace.yaml"}
	}
	if _, err := os.Stat(filepath.Join(absDir, "yarn.lock")); err == nil {
		return PackageManagerInfo{Name: "yarn", Source: "yarn.lock"}
	}
	if _, err := os.Stat(filepath.Join(absDir, "package-lock.json")); err == nil {
		return PackageManagerInfo{Name: "npm", Source: "package-lock.json"}
	}

	// Default to npm if no lock files found
	return PackageManagerInfo{Name: "npm", Source: "package.json"}
}

// GetPackageManagerFromPackageJSON reads package.json and extracts the packageManager field.
// The packageManager field format is: "name@version" (e.g., "pnpm@8.15.0", "yarn@4.1.0", "npm@10.5.0")
// Returns the package manager name (without version) if found, empty string otherwise.
func GetPackageManagerFromPackageJSON(projectDir string) string {
	packageJSONPath := filepath.Join(projectDir, "package.json")

	// Validate path before reading
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return ""
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		PackageManager string `json:"packageManager"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		// Log invalid JSON for debugging purposes, including full project directory path
		slog.Debug("failed to parse package.json", "project", projectDir, "error", "invalid JSON format")
		return ""
	}

	// The packageManager field is in the format "name@version"
	// We need to extract just the name part
	if pkg.PackageManager == "" {
		return ""
	}

	// Split by '@' to extract the package manager name from "name@version" format
	// (e.g., "npm@8.19.2" -> "npm")
	parts := strings.Split(pkg.PackageManager, "@")

	// The package manager name is the first part
	pkgMgrName := parts[0]

	// Validate it's a supported package manager
	switch pkgMgrName {
	case "npm", "yarn", "pnpm":
		return pkgMgrName
	default:
		// Unsupported package manager, fall back to lock file detection
		return ""
	}
}

// HasPackageJson checks if package.json exists in a directory.
func HasPackageJson(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

// HasNpmWorkspaces checks if package.json defines npm/yarn/pnpm workspaces.
// Returns true if the workspaces field is present and not empty, or if pnpm-workspace.yaml exists.
func HasNpmWorkspaces(dir string) bool {
	// Check for pnpm-workspace.yaml first (pnpm-specific workspace configuration)
	pnpmWorkspacePath := filepath.Join(dir, "pnpm-workspace.yaml")
	if err := security.ValidatePath(pnpmWorkspacePath); err == nil {
		if _, err := os.Stat(pnpmWorkspacePath); err == nil {
			return true
		}
	}

	packageJSONPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return false
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	// Check if workspaces field exists and is not empty
	if pkg.Workspaces == nil {
		return false
	}

	// workspaces can be either an array or an object with packages field
	switch v := pkg.Workspaces.(type) {
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		if packages, ok := v["packages"].([]interface{}); ok {
			return len(packages) > 0
		}
		return false
	default:
		return false
	}
}

// DetectPnpmScript checks for dev or start scripts in package.json.
// Returns the script name ("dev" or "start") if found, empty string otherwise.
func DetectPnpmScript(dir string) string {
	packageJSONPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return ""
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// Check for dev script first, then start
	if _, ok := pkg.Scripts["dev"]; ok {
		return "dev"
	}
	if _, ok := pkg.Scripts["start"]; ok {
		return "start"
	}

	return ""
}

// HasDockerComposeScript checks if package.json has docker compose command.
func HasDockerComposeScript(dir string) bool {
	packageJSONPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return false
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	// Look for any script containing "docker compose up" or "docker-compose up"
	for _, scriptCmd := range pkg.Scripts {
		if strings.Contains(scriptCmd, "docker compose up") || strings.Contains(scriptCmd, "docker-compose up") {
			return true
		}
	}

	return false
}

// FindDockerComposeScript finds the script name containing docker compose.
func FindDockerComposeScript(dir string) string {
	packageJSONPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return ""
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	// Find the script with docker compose
	for scriptName, scriptCmd := range pkg.Scripts {
		if strings.Contains(scriptCmd, "docker compose up") || strings.Contains(scriptCmd, "docker-compose up") {
			return scriptName
		}
	}

	return ""
}
