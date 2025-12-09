// Package pathutil provides utilities for managing system PATH environment variable.
package pathutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// RefreshPATH refreshes the current process's PATH environment variable
// by reading from the system and user environment variables.
// Returns the new PATH value and any error encountered.
func RefreshPATH() (string, error) {
	if runtime.GOOS == "windows" {
		return refreshWindowsPATH()
	}
	return refreshUnixPATH()
}

// refreshWindowsPATH refreshes PATH on Windows by reading from registry.
func refreshWindowsPATH() (string, error) {
	// Get Machine PATH with security flags
	machinePath, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command",
		"[Environment]::GetEnvironmentVariable('PATH', 'Machine')").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get machine PATH: %w", err)
	}

	// Get User PATH with security flags
	userPath, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command",
		"[Environment]::GetEnvironmentVariable('PATH', 'User')").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get user PATH: %w", err)
	}

	// Combine and clean up
	machinePathStr := strings.TrimSpace(string(machinePath))
	userPathStr := strings.TrimSpace(string(userPath))

	var newPath string
	if machinePathStr != "" && userPathStr != "" {
		newPath = machinePathStr + ";" + userPathStr
	} else if machinePathStr != "" {
		newPath = machinePathStr
	} else {
		newPath = userPathStr
	}

	// Set the new PATH
	if err := os.Setenv("PATH", newPath); err != nil {
		return "", fmt.Errorf("failed to set PATH: %w", err)
	}

	return newPath, nil
}

// refreshUnixPATH refreshes PATH on Unix-like systems.
// Note: This doesn't actually source shell profiles because Go processes can't
// easily inherit sourced environment variables. Instead, we just return the current PATH.
// The user should restart their shell for permanent changes.
func refreshUnixPATH() (string, error) {
	// On Unix systems, we can't easily refresh the PATH from shell profiles
	// because that requires sourcing files in a shell context.
	// The best we can do is return the current PATH.
	currentPath := os.Getenv("PATH")
	return currentPath, nil
}

// FindToolInPath searches for a tool executable in the system PATH.
// Returns the full path to the executable if found, empty string otherwise.
func FindToolInPath(toolName string) string {
	// Add .exe extension on Windows if not present
	searchName := toolName
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(toolName), ".exe") {
		searchName = toolName + ".exe"
	}

	// Try to find the executable
	path, err := exec.LookPath(searchName)
	if err != nil {
		return ""
	}

	return path
}

// SearchToolInSystemPath searches for a tool in common system directories.
// This is useful for finding tools that are installed but not in the current PATH.
// Returns the full path to the executable if found, empty string otherwise.
func SearchToolInSystemPath(toolName string) string {
	// Add .exe extension on Windows
	exeName := toolName
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(toolName), ".exe") {
		exeName = toolName + ".exe"
	}

	// Define common search paths based on OS
	var searchPaths []string
	if runtime.GOOS == "windows" {
		searchPaths = []string{
			"C:\\Program Files\\nodejs",
			"C:\\Program Files\\Docker\\Docker\\resources\\bin",
			"C:\\Program Files\\Git\\cmd",
			"C:\\Program Files\\Python312",
			"C:\\Program Files\\Python311",
			"C:\\Program Files\\Python310",
			"C:\\Program Files\\dotnet",
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python"),
			filepath.Join(os.Getenv("APPDATA"), "npm"),
			filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "npm"),
			filepath.Join(os.Getenv("USERPROFILE"), "go", "bin"), // Go tools installed via 'go install'
		}
	} else {
		homeDir, _ := os.UserHomeDir()
		searchPaths = []string{
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
			"/opt/homebrew/bin",
			"/usr/local/opt",
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, ".cargo", "bin"),
			filepath.Join(homeDir, "go", "bin"),
		}
	}

	// Search in each path
	for _, dir := range searchPaths {
		fullPath := filepath.Join(dir, exeName)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

// GetInstallSuggestion returns a suggestion for how to install a missing tool.
func GetInstallSuggestion(toolName string) string {
	suggestions := map[string]string{
		"node":   "Install from https://nodejs.org/",
		"pnpm":   "Install from https://pnpm.io/installation",
		"npm":    "Install Node.js from https://nodejs.org/",
		"yarn":   "Install from https://yarnpkg.com/getting-started/install",
		"python": "Install from https://www.python.org/downloads/",
		"pip":    "Install Python from https://www.python.org/downloads/",
		"poetry": "Install from https://python-poetry.org/docs/#installation",
		"uv":     "Install from https://docs.astral.sh/uv/getting-started/installation/",
		"pipenv": "Install from https://pipenv.pypa.io/en/latest/installation.html",
		"docker": "Install Docker Desktop from https://www.docker.com/products/docker-desktop",
		"git":    "Install from https://git-scm.com/downloads",
		"go":     "Install from https://go.dev/dl/",
		"dotnet": "Install from https://dotnet.microsoft.com/download",
		"aspire": "Install from https://learn.microsoft.com/dotnet/aspire/fundamentals/setup-tooling",
		"azd":    "Install from https://aka.ms/install-azd",
		"az":     "Install from https://aka.ms/installazurecli",
		"air":    "Install from https://github.com/air-verse/air#installation",
		"func":   "Install from https://learn.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools",
		"java":   "Install from https://adoptium.net/",
		"mvn":    "Install from https://maven.apache.org/install.html",
		"gradle": "Install from https://gradle.org/install/",
		"gh":     "Install from https://cli.github.com/",
	}

	if suggestion, ok := suggestions[toolName]; ok {
		return suggestion
	}
	return fmt.Sprintf("Please install %s manually", toolName)
}
