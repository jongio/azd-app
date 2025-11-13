// Package installer handles dependency installation for various package managers.
package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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
	case "pnpm":
		args = []string{"install", "--prefer-offline"}
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

	// Configure output based on mode
	if progressWriter != nil {
		// Parallel mode: send output to progress writer
		cmd.Stdout = progressWriter
		cmd.Stderr = progressWriter
	} else if output.IsJSON() {
		// JSON mode: suppress output
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		// Default mode: stream output directly
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// Don't set Stdin - we don't want interactive prompts
	cmd.Env = os.Environ()

	// Add NPM_CONFIG_PROGRESS for npm to ensure progress is shown
	if project.PackageManager == "npm" && progressWriter == nil && !output.IsJSON() {
		cmd.Env = append(cmd.Env, "NPM_CONFIG_PROGRESS=true", "NPM_CONFIG_LOGLEVEL=verbose")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s install: %w", project.PackageManager, err)
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

	// Configure output
	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = progressWriter
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// Don't set Stdin - we don't want interactive prompts
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
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
		return fmt.Errorf("unknown package manager: %s", project.PackageManager)
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

	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = progressWriter
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
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

			if progressWriter != nil {
				venvCmd.Stdout = progressWriter
				venvCmd.Stderr = progressWriter
			} else if output.IsJSON() {
				venvCmd.Stdout = io.Discard
				venvCmd.Stderr = io.Discard
			} else {
				venvCmd.Stdout = os.Stdout
				venvCmd.Stderr = os.Stderr
			}

			if venvErr := venvCmd.Run(); venvErr != nil {
				return fmt.Errorf("failed to create venv with uv: %w", venvErr)
			}

			// Install dependencies
			if !output.IsJSON() && progressWriter == nil {
				output.Item("Installing dependencies into .venv (uv pip)...")
			}
			installCmd := exec.Command("uv", "pip", "install", "-r", "requirements.txt", "--no-progress")
			installCmd.Dir = projectDir
			installCmd.Env = os.Environ() // Inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)

			if progressWriter != nil {
				installCmd.Stdout = progressWriter
				installCmd.Stderr = progressWriter
			} else if output.IsJSON() {
				installCmd.Stdout = io.Discard
				installCmd.Stderr = io.Discard
			} else {
				installCmd.Stdout = os.Stdout
				installCmd.Stderr = os.Stderr
			}

			if installErr := installCmd.Run(); installErr != nil {
				return fmt.Errorf("failed to install with uv: %w", installErr)
			}
		} else {
			return fmt.Errorf("uv sync failed: %w", err)
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

	if progressWriter != nil {
		cmd.Stdout = progressWriter
		cmd.Stderr = progressWriter
	} else if output.IsJSON() {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install with poetry: %w", err)
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
		cmdOutput, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create venv: %w\n%s", err, cmdOutput)
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

		if progressWriter != nil {
			pipCmd.Stdout = progressWriter
			pipCmd.Stderr = progressWriter
		} else if output.IsJSON() {
			pipCmd.Stdout = io.Discard
			pipCmd.Stderr = io.Discard
		} else {
			pipCmd.Stdout = os.Stdout
			pipCmd.Stderr = os.Stderr
		}
		// Don't set Stdin - we don't want interactive prompts
		pipCmd.Env = os.Environ()

		if err := pipCmd.Run(); err != nil {
			return fmt.Errorf("failed to install requirements: %w", err)
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
