package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

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

	if !output.IsJSON() {
		fmt.Printf("   📥 Installing: %s (%s)\n", project.Dir, project.PackageManager)
	}

	if err := executor.RunCommand(project.PackageManager, []string{"install"}, project.Dir); err != nil {
		return fmt.Errorf("failed to run %s install: %w", project.PackageManager, err)
	}

	if !output.IsJSON() {
		fmt.Printf("   ✓ Installed dependencies\n")
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
		fmt.Printf("   📥 Restoring: %s\n", project.Path)
	}

	dir := filepath.Dir(project.Path)
	if err := executor.RunCommand("dotnet", []string{"restore", project.Path}, dir); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	if !output.IsJSON() {
		fmt.Printf("   ✓ Restored packages\n")
	}
	return nil
}

// SetupPythonVirtualEnv creates a virtual environment and installs dependencies.
func SetupPythonVirtualEnv(project types.PythonProject) error {
	if !output.IsJSON() {
		fmt.Printf("   📦 %s (%s)\n", project.Dir, project.PackageManager)
	}

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
			fmt.Printf("   ⚠ uv not found, falling back to pip\n")
		}
		return setupWithPip(projectDir)
	}

	// uv automatically manages virtual environments
	// Just sync the project
	if !output.IsJSON() {
		fmt.Printf("   🔄 Syncing with uv...\n")
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
				fmt.Printf("   📥 Installing with uv pip...\n")
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
		fmt.Printf("   ✓ Environment ready (uv)\n")
	}
	return nil
}

// setupWithPoetry sets up a Python project using poetry.
func setupWithPoetry(projectDir string) error {
	// Check if poetry is installed
	if _, err := exec.LookPath("poetry"); err != nil {
		if !output.IsJSON() {
			fmt.Printf("   ⚠ poetry not found, falling back to pip\n")
		}
		return setupWithPip(projectDir)
	}

	// Check if virtual environment exists
	checkCmd := exec.Command("poetry", "env", "info", "--path")
	checkCmd.Dir = projectDir
	cmdOutput, err := checkCmd.CombinedOutput()

	if err == nil && len(cmdOutput) > 0 {
		if !output.IsJSON() {
			fmt.Printf("   ✓ Poetry environment exists\n")
		}
		return nil
	}

	if !output.IsJSON() {
		fmt.Printf("   📥 Installing dependencies with poetry...\n")
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
		fmt.Printf("   ✓ Dependencies installed (poetry)\n")
	}
	return nil
}

// setupWithPip sets up a Python project using pip and venv.
func setupWithPip(projectDir string) error {
	venvPath := filepath.Join(projectDir, ".venv")

	// Check if venv already exists
	if _, err := os.Stat(venvPath); err == nil {
		if !output.IsJSON() {
			fmt.Printf("   ✓ Virtual environment exists\n")
		}
		return nil
	}

	if !output.IsJSON() {
		fmt.Printf("   🔨 Creating virtual environment...\n")
	}

	// Create virtual environment
	cmd := exec.Command("python", "-m", "venv", ".venv")
	cmd.Dir = projectDir
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create venv: %v\n%s", err, cmdOutput)
	}

	if !output.IsJSON() {
		fmt.Printf("   ✓ Created .venv\n")
	}

	// Check if requirements.txt exists and install dependencies
	requirementsPath := filepath.Join(projectDir, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		if !output.IsJSON() {
			fmt.Printf("   📥 Installing dependencies...\n")
		}

		// Determine the pip path based on OS
		pipPath := filepath.Join(venvPath, "Scripts", "pip.exe")
		if _, err := os.Stat(pipPath); err != nil {
			// Try Unix-style path
			pipPath = filepath.Join(venvPath, "bin", "pip")
		}

		// Use safe executor for pip install
		if err := executor.RunCommand(pipPath, []string{"install", "-r", "requirements.txt"}, projectDir); err != nil {
			return fmt.Errorf("failed to install requirements: %w", err)
		}

		if !output.IsJSON() {
			fmt.Printf("   ✓ Dependencies installed (pip)\n")
		}
	}

	return nil
}
