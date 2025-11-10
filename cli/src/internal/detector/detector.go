package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

const (
	skipDirBin         = "bin"
	skipDirGit         = ".git"
	skipDirNodeModules = "node_modules"
	skipDirObj         = "obj"
)

// FindPythonProjects searches for Python projects and detects their package manager.
// Only searches within rootDir and does not traverse outside it.
func FindPythonProjects(rootDir string) ([]types.PythonProject, error) {
	var pythonProjects []types.PythonProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return pythonProjects, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
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

		// Skip common directories
		if info.IsDir() {
			name := info.Name()
			if name == skipDirNodeModules || name == skipDirBin || name == skipDirObj || name == skipDirGit ||
				name == "venv" || name == ".venv" || name == "__pycache__" || name == ".uv" {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			dir := filepath.Dir(path)

			// Skip if we've already found this directory
			if seen[dir] {
				return nil
			}

			// Look for Python project indicators
			if info.Name() == "requirements.txt" || info.Name() == "pyproject.toml" ||
				info.Name() == "poetry.lock" || info.Name() == "uv.lock" {
				packageManager := DetectPythonPackageManager(dir)
				pythonProjects = append(pythonProjects, types.PythonProject{
					Dir:            dir,
					PackageManager: packageManager,
				})
				seen[dir] = true
			}
		}

		return nil
	})

	return pythonProjects, err
}

// DetectPythonPackageManager determines which package manager to use.
// Priority order: uv > poetry > pip.
func DetectPythonPackageManager(projectDir string) string {
	// Check for uv (uv.lock)
	if _, err := os.Stat(filepath.Join(projectDir, "uv.lock")); err == nil {
		return "uv"
	}

	// Check for poetry (poetry.lock)
	if _, err := os.Stat(filepath.Join(projectDir, "poetry.lock")); err == nil {
		return "poetry"
	}

	// Check pyproject.toml for tool configuration
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	// Validate path before reading
	if err := security.ValidatePath(pyprojectPath); err == nil {
		// #nosec G304 -- Path validated by security.ValidatePath
		if data, err := os.ReadFile(pyprojectPath); err == nil {
			content := string(data)
			if strings.Contains(content, "[tool.poetry]") {
				return "poetry"
			}
			if strings.Contains(content, "[tool.uv]") {
				return "uv"
			}
		}
	}

	// Default to pip
	return "pip"
}

// FindNodeProjects searches for package.json files.
// Only searches within rootDir and does not traverse outside it.
func FindNodeProjects(rootDir string) ([]types.NodeProject, error) {
	var nodeProjects []types.NodeProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nodeProjects, err
	}

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

			packageManager := DetectNodePackageManagerWithBoundary(dir, rootDir)
			nodeProjects = append(nodeProjects, types.NodeProject{
				Dir:            dir,
				PackageManager: packageManager,
			})
			seen[dir] = true
		}

		return nil
	})

	return nodeProjects, err
}

// DetectNodePackageManager determines whether to use pnpm, yarn, or npm.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManager(projectDir string) string {
	// Use unbounded search (for backward compatibility with tests)
	return DetectNodePackageManagerWithBoundary(projectDir, "")
}

// DetectNodePackageManagerWithBoundary determines package manager by checking only the project directory.
// Does not search up the directory tree to avoid interference from parent workspace configurations.
// Priority: packageManager field in package.json > lock files > npm (default).
func DetectNodePackageManagerWithBoundary(projectDir string, boundaryDir string) string {
	// Clean the paths to absolute
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		absDir = projectDir
	}

	// First, check for packageManager field in package.json (highest priority)
	if pkgMgr := getPackageManagerFromPackageJson(absDir); pkgMgr != "" {
		return pkgMgr
	}

	// Fall back to lock file detection
	// Priority: pnpm-lock.yaml > pnpm-workspace.yaml > yarn.lock > package-lock.json > npm (default)
	if _, err := os.Stat(filepath.Join(absDir, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(absDir, "pnpm-workspace.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(absDir, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(absDir, "package-lock.json")); err == nil {
		return "npm"
	}

	// Default to npm if no lock files found
	return "npm"
}

// getPackageManagerFromPackageJson reads package.json and extracts the packageManager field.
// The packageManager field format is: "name@version" (e.g., "pnpm@8.15.0", "yarn@4.1.0", "npm@10.5.0")
// Returns the package manager name (without version) if found, empty string otherwise.
func getPackageManagerFromPackageJson(projectDir string) string {
	packageJsonPath := filepath.Join(projectDir, "package.json")

	// Validate path before reading
	if err := security.ValidatePath(packageJsonPath); err != nil {
		return ""
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		PackageManager string `json:"packageManager"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
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

// FindDotnetProjects searches for .csproj and .sln files.
// Only searches within rootDir and does not traverse outside it.
func FindDotnetProjects(rootDir string) ([]types.DotnetProject, error) {
	var dotnetProjects []types.DotnetProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return dotnetProjects, err
	}

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
			if name == skipDirNodeModules || name == skipDirGit || name == skipDirBin || name == skipDirObj {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			ext := filepath.Ext(info.Name())
			if ext == ".csproj" || ext == ".sln" {
				// For .csproj, use the directory; for .sln, use the file itself
				if ext == ".sln" {
					if !seen[path] {
						dotnetProjects = append(dotnetProjects, types.DotnetProject{
							Path: path,
						})
						seen[path] = true
					}
				} else {
					dir := filepath.Dir(path)
					if !seen[dir] {
						dotnetProjects = append(dotnetProjects, types.DotnetProject{
							Path: path,
						})
						seen[dir] = true
					}
				}
			}
		}

		return nil
	})

	return dotnetProjects, err
}

// FindAppHost searches for AppHost.cs recursively.
// Only searches within rootDir and does not traverse outside it.
func FindAppHost(rootDir string) (*types.AspireProject, error) {
	var aspireProject *types.AspireProject

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
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

		// Skip common directories
		if info.IsDir() {
			name := info.Name()
			if name == skipDirNodeModules || name == skipDirBin || name == skipDirObj || name == skipDirGit {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() && (info.Name() == "AppHost.cs" || info.Name() == "Program.cs") {
			// Check if it's in a project directory (has .csproj)
			dir := filepath.Dir(path)
			matches, err := filepath.Glob(filepath.Join(dir, "*.csproj"))
			if err != nil {
				return nil // Skip on error
			}
			if len(matches) > 0 {
				aspireProject = &types.AspireProject{
					Dir:         dir,
					ProjectFile: matches[0],
				}
				return filepath.SkipAll // Found it, stop walking
			}
		}

		return nil
	})

	return aspireProject, err
}

// HasPackageJson checks if package.json exists in a directory.
func HasPackageJson(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

// DetectPnpmScript checks for dev or start scripts in package.json.
// Returns the script name ("dev" or "start") if found, empty string otherwise.
func DetectPnpmScript(dir string) string {
	packageJsonPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJsonPath); err != nil {
		return ""
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJsonPath)
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
	packageJsonPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJsonPath); err != nil {
		return false
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJsonPath)
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
	packageJsonPath := filepath.Join(dir, "package.json")
	// Validate path before reading
	if err := security.ValidatePath(packageJsonPath); err != nil {
		return ""
	}
	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJsonPath)
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

// FindAzureYaml searches for azure.yaml in the current directory and parent directories.
// It stops searching when it reaches the filesystem root or finds a .git directory.
// Returns the absolute path to azure.yaml if found, or empty string if not found.
func FindAzureYaml(startDir string) (string, error) {
	// Clean the path to absolute
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	currentDir := absDir
	for {
		// Check for azure.yaml in current directory
		azureYamlPath := filepath.Join(currentDir, "azure.yaml")
		if err := security.ValidatePath(azureYamlPath); err == nil {
			if _, err := os.Stat(azureYamlPath); err == nil {
				return azureYamlPath, nil
			}
		}

		// Stop if we hit a .git directory (repository root)
		if _, err := os.Stat(filepath.Join(currentDir, ".git")); err == nil {
			break
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}

	// Not found
	return "", nil
}
