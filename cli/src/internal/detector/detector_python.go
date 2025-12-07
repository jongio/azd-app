package detector

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// FindPythonProjects searches for Python projects and detects their package manager.
//
// This function recursively searches for Python project indicators within the specified
// root directory and identifies the appropriate package manager for each project.
//
// Parameters:
//   - rootDir: Root directory to search (absolute or relative path)
//
// Returns:
//   - []types.PythonProject: Slice of detected Python projects with package manager info
//   - error: Non-nil if directory traversal fails
//
// Detection Strategy:
//   - Searches for requirements.txt, pyproject.toml, poetry.lock, or uv.lock
//   - Skips common directories: node_modules, .git, bin, obj, venv, .venv, __pycache__
//   - Does not traverse outside rootDir (prevents directory traversal)
//   - Package manager detection order: uv > poetry > pip
func FindPythonProjects(rootDir string) ([]types.PythonProject, error) {
	var pythonProjects []types.PythonProject
	seen := make(map[string]bool)

	// Clean the root directory path
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return pythonProjects, err
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		// Standard error handling: log and skip problematic paths
		// This prevents permission errors from terminating the search
		if err != nil {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil // Skip errors but continue walking
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
	info := DetectPythonPackageManagerWithSource(projectDir)
	return info.Name
}

// DetectPythonPackageManagerWithSource determines which package manager to use and returns detection source.
// Priority order: uv > poetry > pipenv > pip.
func DetectPythonPackageManagerWithSource(projectDir string) PackageManagerInfo {
	// Check for uv (uv.lock)
	if _, err := os.Stat(filepath.Join(projectDir, "uv.lock")); err == nil {
		return PackageManagerInfo{Name: "uv", Source: "uv.lock"}
	}

	// Check for poetry (poetry.lock)
	if _, err := os.Stat(filepath.Join(projectDir, "poetry.lock")); err == nil {
		return PackageManagerInfo{Name: "poetry", Source: "poetry.lock"}
	}

	// Check pyproject.toml for tool configuration
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	// Validate path before reading
	if err := security.ValidatePath(pyprojectPath); err == nil {
		// #nosec G304 -- Path validated by security.ValidatePath
		if data, err := os.ReadFile(pyprojectPath); err == nil {
			content := string(data)
			if strings.Contains(content, "[tool.poetry]") {
				return PackageManagerInfo{Name: "poetry", Source: "pyproject.toml"}
			}
			if strings.Contains(content, "[tool.uv]") {
				return PackageManagerInfo{Name: "uv", Source: "pyproject.toml"}
			}
		}
	}

	// Check for pipenv (Pipfile or Pipfile.lock)
	if _, err := os.Stat(filepath.Join(projectDir, "Pipfile")); err == nil {
		return PackageManagerInfo{Name: "pipenv", Source: "Pipfile"}
	}
	if _, err := os.Stat(filepath.Join(projectDir, "Pipfile.lock")); err == nil {
		return PackageManagerInfo{Name: "pipenv", Source: "Pipfile.lock"}
	}

	// Default to pip
	return PackageManagerInfo{Name: "pip", Source: "requirements.txt"}
}
