// Package installer provides dependency installation capabilities for Node.js, Python, and .NET projects.
package installer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// InstallNodeDependencies installs dependencies using the detected package manager.
func InstallNodeDependencies(project types.NodeProject) error {
	return installNodeDependenciesWithWriter(project, nil)
}

// installNodeDependenciesWithWriter installs dependencies with optional writer for progress tracking.
func installNodeDependenciesWithWriter(project types.NodeProject, progressWriter io.Writer) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}

	if err := security.ValidatePackageManager(project.PackageManager); err != nil {
		return fmt.Errorf("invalid package manager: %w", err)
	}

	// Check if dependencies are already installed and up-to-date
	nodeModulesPath := filepath.Join(project.Dir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); err == nil {
		// node_modules exists, check if it's up-to-date
		if isDependenciesUpToDate(project.Dir, project.PackageManager) {
			if !output.IsJSON() && progressWriter == nil {
				output.ItemSuccess("Dependencies already up-to-date (skipping install)")
			}
			return nil
		}
	}

	// On Windows, package managers like npm, pnpm, yarn are .cmd files (batch scripts)
	// not binary executables. exec.Command() requires the shell to properly resolve
	// these .cmd files and handle path escaping, especially for deeply nested node_modules
	// that can exceed Windows' default 260-character path limit.
	//
	// Using cmd.exe /c ensures:
	// 1. Proper .cmd file resolution (npm.cmd, pnpm.cmd, yarn.cmd)
	// 2. Correct environment variable expansion
	// 3. Better handling of Windows path length issues
	var cmd *exec.Cmd
	var args []string

	// Add non-interactive flags to prevent prompts
	switch project.PackageManager {
	case "npm":
		args = []string{"install", "--no-audit", "--no-fund", "--prefer-offline"}
		// If this is a workspace root, use --workspaces flag to install all workspace packages
		if project.IsWorkspaceRoot {
			args = append(args, "--workspaces")
		}
	case "pnpm":
		args = []string{"install", "--prefer-offline"}
		// If this is a workspace root, use --recursive flag to install all workspace packages
		if project.IsWorkspaceRoot {
			args = append(args, "--recursive")
		}
	case "yarn":
		args = []string{"install", "--non-interactive", "--prefer-offline"}
	default:
		args = []string{"install"}
	}

	if runtime.GOOS == "windows" {
		// Use cmd.exe /c to properly invoke .cmd files
		cmdArgs := append([]string{"/c", project.PackageManager}, args...)
		cmd = exec.Command("cmd.exe", cmdArgs...)
	} else {
		cmd = exec.Command(project.PackageManager, args...)
	}

	cmd.Dir = project.Dir

	// Capture stderr for error reporting, even in progress mode
	var stderrBuf bytes.Buffer

	// Configure output based on mode
	if progressWriter != nil {
		// Parallel mode: send output to progress writer, but also capture stderr
		cmd.Stdout = progressWriter
		cmd.Stderr = io.MultiWriter(progressWriter, &stderrBuf)
	} else if output.IsJSON() {
		// JSON mode: suppress output but capture stderr for errors
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderrBuf
	} else {
		// Default mode: stream output directly but also capture stderr
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}
	// Don't set Stdin - we don't want interactive prompts
	cmd.Env = os.Environ()

	// Add NPM_CONFIG_PROGRESS for npm to ensure progress is shown
	if project.PackageManager == "npm" && progressWriter == nil && !output.IsJSON() {
		cmd.Env = append(cmd.Env, "NPM_CONFIG_PROGRESS=true", "NPM_CONFIG_LOGLEVEL=verbose")
	}

	// Run with retry logic for Windows file locking errors
	err := runWithRetry(cmd, &stderrBuf, 3)
	if err != nil {
		return formatNodeInstallError(project.PackageManager, project.Dir, cmd, err, stderrBuf.String())
	}

	if !output.IsJSON() && progressWriter == nil {
		output.ItemSuccess("Installed dependencies")
	}
	return nil
}

// RestoreDotnetProject runs dotnet restore on a project.
func RestoreDotnetProject(project types.DotnetProject) error {
	return restoreDotnetProjectWithWriter(project, nil)
}

// restoreDotnetProjectWithWriter runs dotnet restore with optional progress writer.
func restoreDotnetProjectWithWriter(project types.DotnetProject, progressWriter io.Writer) error {
	// Validate path
	if err := security.ValidatePath(project.Path); err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	if !output.IsJSON() && progressWriter == nil {
		output.Item("Restoring: %s", project.Path)
	}

	// Run restore with streaming output
	dir := filepath.Dir(project.Path)
	cmd := exec.Command("dotnet", "restore", project.Path)
	cmd.Dir = dir

	// Capture stderr for error reporting
	var stderrBuf bytes.Buffer

	// Configure output
	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = io.MultiWriter(progressWriter, &stderrBuf)
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderrBuf
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}
	// Don't set Stdin - we don't want interactive prompts
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return formatDotnetRestoreError(project.Path, dir, cmd, err, stderrBuf.String())
	}

	if !output.IsJSON() && progressWriter == nil {
		output.ItemSuccess("Restored packages")
	}
	return nil
}

// SetupPythonVirtualEnv creates a virtual environment and installs dependencies.
func SetupPythonVirtualEnv(project types.PythonProject) error {
	return setupPythonVirtualEnvWithWriter(project, nil)
}

// setupPythonVirtualEnvWithWriter creates a virtual environment with optional progress writer.
func setupPythonVirtualEnvWithWriter(project types.PythonProject, progressWriter io.Writer) error {
	switch project.PackageManager {
	case "uv":
		return setupWithUv(project.Dir, progressWriter)
	case "poetry":
		return setupWithPoetry(project.Dir, progressWriter)
	case "pip":
		return setupWithPip(project.Dir, progressWriter)
	default:
		return fmt.Errorf("unknown package manager '%s' for Python project in %s", project.PackageManager, project.Dir)
	}
}

// setupWithUv sets up a Python project using uv.
func setupWithUv(projectDir string, progressWriter io.Writer) error {
	// Check if uv is installed
	if _, err := exec.LookPath("uv"); err != nil {
		if !output.IsJSON() && progressWriter == nil {
			output.ItemWarning("uv not found, falling back to pip")
		}
		return setupWithPip(projectDir, progressWriter)
	}

	// uv automatically manages virtual environments
	// Just sync the project
	if !output.IsJSON() && progressWriter == nil {
		output.Item("Installing dependencies into .venv (uv)...")
	}

	cmd := exec.Command("uv", "sync", "--no-progress")
	cmd.Dir = projectDir
	cmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

	var stderrBuf bytes.Buffer
	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = io.MultiWriter(progressWriter, &stderrBuf)
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderrBuf
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	if err := cmd.Run(); err != nil {
		// If uv sync fails, try uv pip install with explicit venv creation
		if _, statErr := os.Stat(filepath.Join(projectDir, "requirements.txt")); statErr == nil {
			// Create virtual environment first
			if !output.IsJSON() && progressWriter == nil {
				output.Item("Creating virtual environment at .venv (uv)...")
			}
			venvCmd := exec.Command("uv", "venv")
			venvCmd.Dir = projectDir
			venvCmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

			var venvStderrBuf bytes.Buffer
			if progressWriter != nil {
				venvCmd.Stdout = progressWriter
				venvCmd.Stderr = io.MultiWriter(progressWriter, &venvStderrBuf)
			} else if output.IsJSON() {
				venvCmd.Stdout = io.Discard
				venvCmd.Stderr = &venvStderrBuf
			} else {
				venvCmd.Stdout = os.Stdout
				venvCmd.Stderr = io.MultiWriter(os.Stderr, &venvStderrBuf)
			}

			if venvErr := venvCmd.Run(); venvErr != nil {
				return formatPythonInstallError("uv venv", projectDir, venvCmd, venvErr, venvStderrBuf.String())
			}

			// Install dependencies
			if !output.IsJSON() && progressWriter == nil {
				output.Item("Installing dependencies into .venv (uv pip)...")
			}
			installCmd := exec.Command("uv", "pip", "install", "-r", "requirements.txt", "--no-progress")
			installCmd.Dir = projectDir
			installCmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

			var installStderrBuf bytes.Buffer
			if progressWriter != nil {
				installCmd.Stdout = progressWriter
				installCmd.Stderr = io.MultiWriter(progressWriter, &installStderrBuf)
			} else if output.IsJSON() {
				installCmd.Stdout = io.Discard
				installCmd.Stderr = &installStderrBuf
			} else {
				installCmd.Stdout = os.Stdout
				installCmd.Stderr = io.MultiWriter(os.Stderr, &installStderrBuf)
			}

			if installErr := installCmd.Run(); installErr != nil {
				return formatPythonInstallError("uv pip install", projectDir, installCmd, installErr, installStderrBuf.String())
			}
		} else {
			return formatPythonInstallError("uv sync", projectDir, cmd, err, stderrBuf.String())
		}
	}

	if !output.IsJSON() && progressWriter == nil {
		output.ItemSuccess("Environment ready (uv)")
	}
	return nil
}

// setupWithPoetry sets up a Python project using poetry.
func setupWithPoetry(projectDir string, progressWriter io.Writer) error {
	// Check if poetry is installed
	if _, err := exec.LookPath("poetry"); err != nil {
		if !output.IsJSON() && progressWriter == nil {
			output.ItemWarning("poetry not found, falling back to pip")
		}
		return setupWithPip(projectDir, progressWriter)
	}

	// Check if virtual environment exists
	checkCmd := exec.Command("poetry", "env", "info", "--path")
	checkCmd.Dir = projectDir
	checkCmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)
	cmdOutput, err := checkCmd.CombinedOutput()

	if err == nil && len(cmdOutput) > 0 {
		if !output.IsJSON() && progressWriter == nil {
			venvPath := string(cmdOutput)
			output.ItemSuccess("Poetry environment exists at %s", venvPath)
		}
		return nil
	}

	if !output.IsJSON() && progressWriter == nil {
		output.Item("Installing dependencies into poetry venv...")
	}

	// Install dependencies (use --no-root to avoid installing the package itself)
	cmd := exec.Command("poetry", "install", "--no-root")
	cmd.Dir = projectDir
	cmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

	var stderrBuf bytes.Buffer
	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = io.MultiWriter(progressWriter, &stderrBuf)
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = &stderrBuf
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	}

	if err := cmd.Run(); err != nil {
		return formatPythonInstallError("poetry install", projectDir, cmd, err, stderrBuf.String())
	}

	if !output.IsJSON() && progressWriter == nil {
		output.ItemSuccess("Dependencies installed (poetry)")
	}
	return nil
}

// setupWithPip sets up a Python project using pip and venv.
func setupWithPip(projectDir string, progressWriter io.Writer) error {
	venvPath := filepath.Join(projectDir, ".venv")

	// Check if venv already exists, create if not
	if _, err := os.Stat(venvPath); err != nil {
		if !output.IsJSON() && progressWriter == nil {
			output.Item("Creating virtual environment at .venv...")
		}

		// Create virtual environment
		cmd := exec.Command("python", "-m", "venv", ".venv")
		cmd.Dir = projectDir
		cmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf
		cmd.Stdout = io.Discard

		if err := cmd.Run(); err != nil {
			return formatPythonInstallError("python -m venv", projectDir, cmd, err, stderrBuf.String())
		}

		if !output.IsJSON() && progressWriter == nil {
			output.ItemSuccess("Created .venv")
		}
	} else {
		if !output.IsJSON() && progressWriter == nil {
			output.ItemSuccess("Virtual environment exists")
		}
	}

	// Check if requirements.txt exists and install dependencies
	requirementsPath := filepath.Join(projectDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		if !output.IsJSON() && progressWriter == nil {
			output.Item("Installing dependencies into .venv (pip)...")
		}

		// Determine the pip path based on OS
		// Using the pip executable directly from the venv ensures packages
		// are installed into the correct virtual environment without needing
		// to activate it (activation is only needed for interactive shells)
		var pipPath string
		if runtime.GOOS == "windows" {
			pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")
		} else {
			pipPath = filepath.Join(venvPath, "bin", "pip")
		}

		// Run pip install with streaming output and optimizations
		pipCmd := exec.Command(pipPath, "install", "-r", "requirements.txt", "--disable-pip-version-check", "--prefer-binary")
		pipCmd.Dir = projectDir

		var stderrBuf bytes.Buffer
		if progressWriter != nil {
			pipCmd.Stdout = progressWriter
			pipCmd.Stderr = io.MultiWriter(progressWriter, &stderrBuf)
		} else if output.IsJSON() {
			pipCmd.Stdout = io.Discard
			pipCmd.Stderr = &stderrBuf
		} else {
			pipCmd.Stdout = os.Stdout
			pipCmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
		}
		// Don't set Stdin - we don't want interactive prompts
		pipCmd.Env = os.Environ()

		if err := pipCmd.Run(); err != nil {
			return formatPythonInstallError("pip install", projectDir, pipCmd, err, stderrBuf.String())
		}

		if !output.IsJSON() && progressWriter == nil {
			output.ItemSuccess("Dependencies installed (pip)")
		}
	}

	return nil
}

// isDependenciesUpToDate checks if node_modules is up-to-date with the lock file
func isDependenciesUpToDate(projectDir string, packageManager string) bool {
	nodeModulesPath := filepath.Join(projectDir, "node_modules")

	// Determine which lock file to check based on package manager
	var lockFile string
	var internalLockFile string
	switch packageManager {
	case "npm":
		lockFile = "package-lock.json"
		internalLockFile = filepath.Join("node_modules", ".package-lock.json")
	case "pnpm":
		lockFile = "pnpm-lock.yaml"
		// pnpm uses a virtual store, check .pnpm directory exists and is newer
		internalLockFile = filepath.Join("node_modules", ".pnpm")
	case "yarn":
		lockFile = "yarn.lock"
		// Yarn doesn't use an internal lock file in node_modules
		internalLockFile = ""
	default:
		return false
	}

	lockFilePath := filepath.Join(projectDir, lockFile)

	// Check if lock file exists
	lockFileInfo, err := os.Stat(lockFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			// Log unexpected errors but proceed conservatively
			fmt.Fprintf(os.Stderr, "Warning: Failed to check lock file %s: %v\n", lockFilePath, err)
		}
		return false
	}

	// Check if node_modules exists
	if _, err := os.Stat(nodeModulesPath); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: Failed to check node_modules: %v\n", err)
		}
		return false
	}

	// For npm and pnpm, check the internal lock file in node_modules
	if internalLockFile != "" {
		internalLockPath := filepath.Join(projectDir, internalLockFile)
		internalLockInfo, err := os.Stat(internalLockPath)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: Failed to check internal lock file: %v\n", err)
			}
			// Internal lock file doesn't exist or can't be accessed, needs install
			return false
		}

		// Compare timestamps - if internal lock is older than main lock, need to reinstall
		if internalLockInfo.ModTime().Before(lockFileInfo.ModTime()) {
			return false
		}
	}

	// Dependencies appear to be up-to-date
	return true
}

// formatNodeInstallError creates a detailed error message for node package manager failures
func formatNodeInstallError(packageManager, projectDir string, cmd *exec.Cmd, cmdErr error, stderr string) error {
	var exitCode int
	if exitErr, ok := cmdErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else {
		// If not an ExitError, check if the error message contains exit status
		errMsg := cmdErr.Error()
		if strings.Contains(errMsg, "exit status") {
			// Try to extract exit code from error message
			var code int
			if _, err := fmt.Sscanf(errMsg, "exit status %d", &code); err == nil {
				exitCode = code
			}
		}
	}

	// Build base error message
	errMsg := fmt.Sprintf("failed to run %s install", packageManager)

	// Add exit code context
	switch exitCode {
	case 1:
		errMsg += " (command failed with errors)"
	case 127:
		errMsg += fmt.Sprintf(" (%s not found - please install %s)", packageManager, packageManager)
	case 254:
		errMsg += fmt.Sprintf(" (%s command failed - check if %s is installed and in PATH)", packageManager, packageManager)
	default:
		if exitCode != 0 {
			errMsg += fmt.Sprintf(" (exit code %d)", exitCode)
		}
	}

	// Extract meaningful error details from stderr
	errorDetails := extractErrorDetails(stderr, packageManager)
	if errorDetails != "" {
		errMsg += ": " + errorDetails
	}

	// Add installation suggestions based on exit code and error pattern
	suggestion := getSuggestion(packageManager, exitCode, stderr)
	if suggestion != "" {
		errMsg += "\n   Suggestion: " + suggestion
	}

	return fmt.Errorf("%s\n   Directory: %s\n   Command: %s", errMsg, projectDir, formatCommand(cmd))
}

// formatDotnetRestoreError creates a detailed error message for dotnet restore failures
func formatDotnetRestoreError(projectPath, dir string, cmd *exec.Cmd, cmdErr error, stderr string) error {
	var exitCode int
	if exitErr, ok := cmdErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else {
		// If not an ExitError, try to extract from error message
		errMsg := cmdErr.Error()
		if strings.Contains(errMsg, "exit status") {
			var code int
			if _, err := fmt.Sscanf(errMsg, "exit status %d", &code); err == nil {
				exitCode = code
			}
		}
	}

	errMsg := "failed to restore .NET project"
	if exitCode == 127 {
		errMsg += " (dotnet not found - please install .NET SDK)"
	} else if exitCode != 0 {
		errMsg += fmt.Sprintf(" (exit code %d)", exitCode)
	}

	errorDetails := extractErrorDetails(stderr, "dotnet")
	if errorDetails != "" {
		errMsg += ": " + errorDetails
	}

	return fmt.Errorf("%s\n   Project: %s\n   Directory: %s\n   Command: %s", errMsg, projectPath, dir, formatCommand(cmd))
}

// extractErrorDetails extracts the most relevant error lines from stderr
func extractErrorDetails(stderr, tool string) string {
	if stderr == "" {
		return ""
	}

	// Limit stderr to prevent memory issues with extremely verbose output
	const maxStderrLen = 10000 // 10KB should be enough for error context
	if len(stderr) > maxStderrLen {
		stderr = stderr[:maxStderrLen] + "... (truncated)"
	}

	lines := strings.Split(stderr, "\n")
	var errorLines []string

	// Look for common error patterns
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip noise lines
		if strings.HasPrefix(line, "Progress:") ||
			strings.HasPrefix(line, "Downloading") ||
			strings.HasPrefix(line, "Building") {
			continue
		}

		// Capture error indicators
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") ||
			strings.Contains(lowerLine, "failed") ||
			strings.Contains(lowerLine, "enoent") ||
			strings.Contains(lowerLine, "permission denied") ||
			strings.Contains(lowerLine, "cannot find") ||
			strings.Contains(lowerLine, "command not found") {
			errorLines = append(errorLines, line)
			if len(errorLines) >= 3 {
				break // Limit to first 3 error lines
			}
		}
	}

	if len(errorLines) > 0 {
		result := strings.Join(errorLines, "; ")
		// Truncate if combined error lines are too long
		const maxErrorLen = 500
		if len(result) > maxErrorLen {
			return result[:maxErrorLen] + "..."
		}
		return result
	}

	// If no specific error patterns found, return last few non-empty lines
	var lastLines []string
	for i := len(lines) - 1; i >= 0 && len(lastLines) < 2; i-- {
		if trimmed := strings.TrimSpace(lines[i]); trimmed != "" {
			lastLines = append([]string{trimmed}, lastLines...)
		}
	}

	if len(lastLines) > 0 {
		result := strings.Join(lastLines, "; ")
		const maxErrorLen = 500
		if len(result) > maxErrorLen {
			return result[:maxErrorLen] + "..."
		}
		return result
	}

	return ""
}

// getSuggestion provides helpful suggestions based on error patterns
func getSuggestion(packageManager string, exitCode int, stderr string) string {
	lowerStderr := strings.ToLower(stderr)

	// Check for "command not found" or exit code 127/254
	if exitCode == 127 || exitCode == 254 || strings.Contains(lowerStderr, "command not found") {
		switch packageManager {
		case "pnpm":
			return "Install pnpm with: npm install -g pnpm"
		case "yarn":
			return "Install yarn with: npm install -g yarn"
		case "npm":
			return "Install Node.js and npm from: https://nodejs.org"
		}
	}

	// Permission errors
	if strings.Contains(lowerStderr, "permission denied") || strings.Contains(lowerStderr, "eacces") {
		return "Try running with appropriate permissions or check file/directory ownership"
	}

	// Network errors
	if strings.Contains(lowerStderr, "enotfound") || strings.Contains(lowerStderr, "network") {
		return "Check your network connection and proxy settings"
	}

	// Disk space
	if strings.Contains(lowerStderr, "enospc") || strings.Contains(lowerStderr, "no space") {
		return "Free up disk space and try again"
	}

	// Lock file issues
	if strings.Contains(lowerStderr, "lock") && strings.Contains(lowerStderr, "conflict") {
		return "Delete the lock file and node_modules, then try again"
	}

	return ""
}

// formatCommand formats a command for display
func formatCommand(cmd *exec.Cmd) string {
	if cmd == nil {
		return "(unknown command)"
	}
	if len(cmd.Args) == 0 {
		return cmd.Path
	}
	if len(cmd.Args) == 1 {
		return cmd.Args[0]
	}
	return cmd.Args[0] + " " + strings.Join(cmd.Args[1:], " ")
}

// formatPythonInstallError creates a detailed error message for Python installer failures
func formatPythonInstallError(tool, projectDir string, cmd *exec.Cmd, cmdErr error, stderr string) error {
	var exitCode int
	if exitErr, ok := cmdErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else {
		// If not an ExitError, try to extract from error message
		errMsg := cmdErr.Error()
		if strings.Contains(errMsg, "exit status") {
			var code int
			if _, err := fmt.Sscanf(errMsg, "exit status %d", &code); err == nil {
				exitCode = code
			}
		}
	}

	errMsg := fmt.Sprintf("failed to run %s", tool)

	// Add exit code context
	if exitCode == 127 {
		errMsg += fmt.Sprintf(" (%s not found - please install %s)", tool, tool)
	} else if exitCode != 0 {
		errMsg += fmt.Sprintf(" (exit code %d)", exitCode)
	}

	// Extract error details
	errorDetails := extractErrorDetails(stderr, tool)
	if errorDetails != "" {
		errMsg += ": " + errorDetails
	}

	// Add Python-specific suggestions
	suggestion := getPythonSuggestion(tool, exitCode, stderr)
	if suggestion != "" {
		errMsg += "\n   Suggestion: " + suggestion
	}

	return fmt.Errorf("%s\n   Directory: %s\n   Command: %s", errMsg, projectDir, formatCommand(cmd))
}

// getPythonSuggestion provides helpful suggestions for Python tool failures
func getPythonSuggestion(tool string, exitCode int, stderr string) string {
	lowerStderr := strings.ToLower(stderr)

	// Check for tool not found
	if exitCode == 127 || strings.Contains(lowerStderr, "command not found") {
		switch tool {
		case "uv":
			return "Install uv with: pip install uv"
		case "poetry":
			return "Install poetry with: pip install poetry"
		case "python":
			return "Install Python from: https://www.python.org/downloads/"
		}
	}

	// Permission errors
	if strings.Contains(lowerStderr, "permission denied") || strings.Contains(lowerStderr, "eacces") {
		return "Try running with appropriate permissions or check file/directory ownership"
	}

	// Network errors
	if strings.Contains(lowerStderr, "could not find") || strings.Contains(lowerStderr, "connection") {
		return "Check your network connection and PyPI access"
	}

	// Virtual environment issues
	if strings.Contains(lowerStderr, "venv") || strings.Contains(lowerStderr, "virtualenv") {
		return "Try deleting the .venv directory and running again"
	}

	return ""
}

// runWithRetry executes a command with retry logic for Windows file locking errors.
// This is a safety net for race conditions in npm workspaces on Windows where
// concurrent npm processes may compete for the same files.
func runWithRetry(cmd *exec.Cmd, stderrBuf *bytes.Buffer, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Run the command
		err := cmd.Run()

		// If successful, return
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if this is a file locking error that we should retry
		stderr := stderrBuf.String()
		if isFileLockingError(stderr) && attempt < maxRetries {
			// Calculate exponential backoff delay
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			if !output.IsJSON() {
				output.ItemWarning("File locking error detected, retrying in %v... (attempt %d/%d)", delay, attempt, maxRetries)
			}
			time.Sleep(delay)

			// Reset stderr buffer for next attempt
			stderrBuf.Reset()

			// Recreate the command for the next attempt (exec.Cmd can only be run once)
			newCmd := exec.Command(cmd.Path, cmd.Args[1:]...)
			newCmd.Dir = cmd.Dir
			newCmd.Env = cmd.Env
			newCmd.Stdout = cmd.Stdout
			newCmd.Stderr = io.MultiWriter(cmd.Stderr, stderrBuf)
			newCmd.Stdin = cmd.Stdin
			cmd = newCmd
			continue
		}

		// Not a file locking error or max retries reached
		return err
	}

	return lastErr
}

// isFileLockingError checks if the error message indicates a Windows file locking issue.
// Common errors include EBUSY (file busy) and ENOTEMPTY (directory not empty).
func isFileLockingError(stderr string) bool {
	lowerStderr := strings.ToLower(stderr)
	return strings.Contains(lowerStderr, "ebusy") ||
		strings.Contains(lowerStderr, "enotempty") ||
		strings.Contains(lowerStderr, "eperm") && runtime.GOOS == "windows"
}
