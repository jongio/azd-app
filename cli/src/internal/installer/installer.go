package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/security"
	"github.com/jongio/azd-app/cli/src/internal/types"
)

// InstallNodeDependencies installs dependencies using the detected package manager.
func InstallNodeDependencies(project types.NodeProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}

	if err := security.ValidatePackageManager(project.PackageManager); err != nil {
		return fmt.Errorf("invalid package manager: %w", err)
	}

	if err := executor.RunCommand(project.PackageManager, []string{"install"}, project.Dir); err != nil {
		return fmt.Errorf("failed to run %s install: %w", project.PackageManager, err)
	}

	if !output.IsJSON() {
		output.ItemSuccess("Installed dependencies")
	}
	return nil
}

// RestoreDotnetProject runs dotnet restore on a project.
func RestoreDotnetProject(project types.DotnetProject) error {
	// Validate path
	if err := security.ValidatePath(project.Path); err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	if !output.IsJSON() {
		output.Item("Restoring: %s", project.Path)
	}

	dir := filepath.Dir(project.Path)
	if err := executor.RunCommand("dotnet", []string{"restore", project.Path}, dir); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	if !output.IsJSON() {
		output.ItemSuccess("Restored packages")
	}
	return nil
}

// SetupPythonVirtualEnv creates a virtual environment and installs dependencies.
func SetupPythonVirtualEnv(project types.PythonProject) error {
	switch project.PackageManager {
	case "uv":
		return setupWithUv(project.Dir)
	case "poetry":
		return setupWithPoetry(project.Dir)
	case "pip":
		return setupWithPip(project.Dir)
	default:
		return fmt.Errorf("unknown package manager: %s", project.PackageManager)
	}
}

// setupWithUv sets up a Python project using uv.
func setupWithUv(projectDir string) error {
	// Check if uv is installed
	if _, err := exec.LookPath("uv"); err != nil {
		if !output.IsJSON() {
			output.ItemWarning("uv not found, falling back to pip")
		}
		return setupWithPip(projectDir)
	}

	// uv automatically manages virtual environments
	// Just sync the project
	if !output.IsJSON() {
		output.Item("Syncing with uv...")
	}

	cmd := exec.Command("uv", "sync")
	cmd.Dir = projectDir

	if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		// If uv sync fails, try uv pip install
		if _, statErr := os.Stat(filepath.Join(projectDir, "requirements.txt")); statErr == nil {
			if !output.IsJSON() {
				output.Item("Installing with uv pip...")
			}
			installCmd := exec.Command("uv", "pip", "install", "-r", "requirements.txt")
			installCmd.Dir = projectDir

			if output.IsJSON() {
				installCmd.Stdout = io.Discard
				installCmd.Stderr = io.Discard
			} else {
				installCmd.Stdout = os.Stdout
				installCmd.Stderr = os.Stderr
			}

			if installErr := installCmd.Run(); installErr != nil {
				return fmt.Errorf("failed to install with uv: %v", installErr)
			}
		} else {
			return fmt.Errorf("uv sync failed: %v", err)
		}
	}

	if !output.IsJSON() {
		output.ItemSuccess("Environment ready (uv)")
	}
	return nil
}

// setupWithPoetry sets up a Python project using poetry.
func setupWithPoetry(projectDir string) error {
	// Check if poetry is installed
	if _, err := exec.LookPath("poetry"); err != nil {
		if !output.IsJSON() {
			output.ItemWarning("poetry not found, falling back to pip")
		}
		return setupWithPip(projectDir)
	}

	// Check if virtual environment exists
	checkCmd := exec.Command("poetry", "env", "info", "--path")
	checkCmd.Dir = projectDir
	cmdOutput, err := checkCmd.CombinedOutput()

	if err == nil && len(cmdOutput) > 0 {
		if !output.IsJSON() {
			output.ItemSuccess("Poetry environment exists")
		}
		return nil
	}

	if !output.IsJSON() {
		output.Item("Installing dependencies with poetry...")
	}

	// Install dependencies (use --no-root to avoid installing the package itself)
	cmd := exec.Command("poetry", "install", "--no-root")
	cmd.Dir = projectDir

	if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install with poetry: %v", err)
	}

	if !output.IsJSON() {
		output.ItemSuccess("Dependencies installed (poetry)")
	}
	return nil
}

// setupWithPip sets up a Python project using pip and venv.
func setupWithPip(projectDir string) error {
	venvPath := filepath.Join(projectDir, ".venv")

	// Check if venv already exists
	if _, err := os.Stat(venvPath); err == nil {
		if !output.IsJSON() {
			output.ItemSuccess("Virtual environment exists")
		}
		return nil
	}

	if !output.IsJSON() {
		output.Item("Creating virtual environment...")
	}

	// Create virtual environment
	cmd := exec.Command("python", "-m", "venv", ".venv")
	cmd.Dir = projectDir
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create venv: %v\n%s", err, cmdOutput)
	}

	if !output.IsJSON() {
		output.ItemSuccess("Created .venv")
	}

	// Check if requirements.txt exists and install dependencies
	requirementsPath := filepath.Join(projectDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		if !output.IsJSON() {
			output.Item("Installing dependencies...")
		}

		// Determine the pip path based on OS
		var pipPath string
		if runtime.GOOS == "windows" {
			pipPath = filepath.Join(venvPath, "Scripts", "pip.exe")
		} else {
			pipPath = filepath.Join(venvPath, "bin", "pip")
		}

		// Use safe executor for pip install
		if err := executor.RunCommand(pipPath, []string{"install", "-r", "requirements.txt"}, projectDir); err != nil {
			return fmt.Errorf("failed to install requirements: %w", err)
		}

		if !output.IsJSON() {
			output.ItemSuccess("Dependencies installed (pip)")
		}
	}

	return nil
}
