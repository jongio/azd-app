package detector

import (
	"os"
	"path/filepath"

	"github.com/jongio/azd-app/cli/src/internal/fileutil"
	"github.com/jongio/azd-app/cli/src/internal/security"
)

// Constants for directories to skip during project detection.
const (
	skipDirBin         = "bin"
	skipDirGit         = ".git"
	skipDirNodeModules = "node_modules"
	skipDirObj         = "obj"
)

// PackageManagerInfo contains the detected package manager and its detection source.
type PackageManagerInfo struct {
	Name   string // Package manager name (npm, yarn, pnpm, uv, poetry, pip)
	Source string // Source of detection (e.g., "package.json (packageManager field)", "pnpm-lock.yaml")
}

// Helper functions for discovery
// Note: Many file operations moved to internal/fileutil package

// fileExistsInDir is a convenience wrapper for fileutil.FileExists
func fileExistsInDir(dir string, filename string) bool {
	return fileutil.FileExists(dir, filename)
}

// containsTextInFile is a convenience wrapper for fileutil.ContainsText
func containsTextInFile(filePath string, text string) bool {
	return fileutil.ContainsText(filePath, text)
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
