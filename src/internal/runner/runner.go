package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"app/src/internal/executor"
	"app/src/internal/security"
	"app/src/internal/types"
)

// RunAspire runs aspire run for an Aspire project.
func RunAspire(project types.AspireProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}

	fmt.Println("🚀 Starting Aspire project...")
	fmt.Println("📁 Directory:", project.Dir)
	fmt.Println("📝 Project:", project.ProjectFile)
	fmt.Println()

	// Use dotnet run instead of aspire run to ensure environment variable propagation.
	// The aspire CLI internally calls dotnet run, but doesn't expose environment variable options.
	// By calling dotnet run directly, all environment variables (including AZD_SERVER,
	// AZD_ACCESS_TOKEN, and Azure environment values) are properly inherited.
	// See: https://github.com/dotnet/aspire/blob/main/src/Aspire.Cli/DotNet/DotNetCliRunner.cs
	args := []string{"run", "--project", project.ProjectFile}
	return executor.StartCommand("dotnet", args, project.Dir)
}

// RunPnpmScript runs pnpm with the specified script.
func RunPnpmScript(script string) error {
	// Validate script name
	if err := security.SanitizeScriptName(script); err != nil {
		return fmt.Errorf("invalid script name: %w", err)
	}

	fmt.Println("🚀 Starting pnpm", script)
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	return executor.StartCommand("pnpm", []string{script}, cwd)
}

// RunDockerCompose runs a docker compose script from package.json.
func RunDockerCompose(scriptName, scriptCmd string) error {
	// Validate script name
	if err := security.SanitizeScriptName(scriptName); err != nil {
		return fmt.Errorf("invalid script name: %w", err)
	}

	fmt.Println("🚀 Starting docker compose via pnpm", scriptName)
	fmt.Println("   Command:", scriptCmd)
	fmt.Println()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	return executor.StartCommand("pnpm", []string{scriptName}, cwd)
}

// RunNode runs a Node.js project with the detected package manager and script.
func RunNode(project types.NodeProject, script string) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}
	if err := security.SanitizeScriptName(script); err != nil {
		return fmt.Errorf("invalid script name: %w", err)
	}
	if err := security.ValidatePackageManager(project.PackageManager); err != nil {
		return fmt.Errorf("invalid package manager: %w", err)
	}

	fmt.Printf("🚀 Starting Node.js project with %s %s\n", project.PackageManager, script)
	fmt.Println("📁 Directory:", project.Dir)
	fmt.Println()

	return executor.StartCommand(project.PackageManager, []string{"run", script}, project.Dir)
}

// RunPython runs a Python project with the detected package manager.
// For projects with a dev script, it runs that. Otherwise, it looks for common entry points.
func RunPython(project types.PythonProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}
	if err := security.ValidatePackageManager(project.PackageManager); err != nil {
		return fmt.Errorf("invalid package manager: %w", err)
	}

	fmt.Printf("🚀 Starting Python project with %s\n", project.PackageManager)
	fmt.Println("📁 Directory:", project.Dir)
	fmt.Println()

	// Different package managers have different run commands
	var cmd string
	var args []string

	switch project.PackageManager {
	case "uv":
		// uv run <script or module>
		// Try common entry points: main.py, app.py, src/main.py
		args = []string{"run", "python"}
		if _, err := os.Stat(fmt.Sprintf("%s/main.py", project.Dir)); err == nil {
			args = append(args, "main.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/app.py", project.Dir)); err == nil {
			args = append(args, "app.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/src/main.py", project.Dir)); err == nil {
			args = append(args, "src/main.py")
		} else {
			return fmt.Errorf("no entry point found (main.py, app.py, or src/main.py)")
		}
		cmd = "uv"

	case "poetry":
		// poetry run python <script>
		args = []string{"run", "python"}
		if _, err := os.Stat(fmt.Sprintf("%s/main.py", project.Dir)); err == nil {
			args = append(args, "main.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/app.py", project.Dir)); err == nil {
			args = append(args, "app.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/src/main.py", project.Dir)); err == nil {
			args = append(args, "src/main.py")
		} else {
			return fmt.Errorf("no entry point found (main.py, app.py, or src/main.py)")
		}
		cmd = "poetry"

	case "pip":
		// Activate venv and run python
		// For now, just run python directly from venv if it exists
		args = []string{}
		if _, err := os.Stat(fmt.Sprintf("%s/main.py", project.Dir)); err == nil {
			args = append(args, "main.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/app.py", project.Dir)); err == nil {
			args = append(args, "app.py")
		} else if _, err := os.Stat(fmt.Sprintf("%s/src/main.py", project.Dir)); err == nil {
			args = append(args, "src/main.py")
		} else {
			return fmt.Errorf("no entry point found (main.py, app.py, or src/main.py)")
		}
		// Check for venv
		venvPython := fmt.Sprintf("%s/.venv/Scripts/python.exe", project.Dir)
		if _, err := os.Stat(venvPython); err == nil {
			cmd = venvPython
		} else {
			venvPython = fmt.Sprintf("%s/venv/Scripts/python.exe", project.Dir)
			if _, err := os.Stat(venvPython); err == nil {
				cmd = venvPython
			} else {
				// Fall back to system python
				cmd = "python"
			}
		}

	default:
		return fmt.Errorf("unsupported package manager: %s", project.PackageManager)
	}

	return executor.StartCommand(cmd, args, project.Dir)
}

// RunDotnet runs a .NET project with 'dotnet run'.
func RunDotnet(project types.DotnetProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Path); err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	fmt.Println("🚀 Starting .NET project...")
	fmt.Println("📁 Project:", project.Path)
	fmt.Println()

	// For .sln files, we need to run from the directory
	// For .csproj files, we can pass the path directly
	args := []string{"run"}
	dir := ""

	if filepath.Ext(project.Path) == ".sln" {
		dir = filepath.Dir(project.Path)
	} else {
		args = append(args, "--project", project.Path)
		dir, _ = os.Getwd()
	}

	return executor.StartCommand("dotnet", args, dir)
}
