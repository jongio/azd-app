// Package runner provides execution capabilities for various project types including
// Aspire, Node.js, Python, .NET, and Azure Functions projects.
// It handles process management, environment configuration, and entry point detection.
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-core/cliout"
	types "github.com/jongio/azd-core/projecttype"
	"github.com/jongio/azd-core/security"
)

// RunAspire runs aspire run for an Aspire project.
func RunAspire(ctx context.Context, project types.AspireProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}

	cliout.Info("Starting Aspire project...")
	cliout.Item("Directory: %s", project.Dir)
	cliout.Item("Project: %s", project.ProjectFile)
	cliout.Newline()

	// Use dotnet run instead of aspire run to ensure environment variable propagation.
	// The aspire CLI internally calls dotnet run, but doesn't expose environment variable options.
	// By calling dotnet run directly, all environment variables (including AZD_SERVER,
	// AZD_ACCESS_TOKEN, and Azure environment values) are properly inherited.
	// See: https://github.com/dotnet/aspire/blob/main/src/Aspire.Cli/DotNet/DotNetCliRunner.cs
	args := []string{"run", "--project", project.ProjectFile}
	return executor.StartCommand(ctx, "dotnet", args, project.Dir)
}

// RunPnpmScript runs pnpm with the specified script.
func RunPnpmScript(ctx context.Context, script string) error {
	// Validate script name
	if err := security.SanitizeScriptName(script); err != nil {
		return fmt.Errorf("invalid script name: %w", err)
	}

	cliout.Info("Starting pnpm %s", script)
	cliout.Newline()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	return executor.StartCommand(ctx, "pnpm", []string{script}, cwd)
}

// RunDockerCompose runs a docker compose script from package.json.
func RunDockerCompose(ctx context.Context, scriptName, scriptCmd string) error {
	// Validate script name
	if err := security.SanitizeScriptName(scriptName); err != nil {
		return fmt.Errorf("invalid script name: %w", err)
	}

	cliout.Info("Starting docker compose via pnpm %s", scriptName)
	cliout.Item("Command: %s", scriptCmd)
	cliout.Newline()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	return executor.StartCommand(ctx, "pnpm", []string{scriptName}, cwd)
}

// RunNode runs a Node.js project with the detected package manager and script.
func RunNode(ctx context.Context, project types.NodeProject, script string) error {
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

	cliout.Info("Starting Node.js project with %s %s", project.PackageManager, script)
	cliout.Item("Directory: %s", project.Dir)
	cliout.Newline()

	return executor.StartCommand(ctx, project.PackageManager, []string{"run", script}, project.Dir)
}

// findPythonEntryPoint searches for common Python entry point files.
// It checks multiple locations in order of preference.
// Returns the relative path to the entry point file, or an error with helpful configuration guidance.
func findPythonEntryPoint(projectDir string) (string, error) {
	// Common entry point filenames in order of preference
	entryPoints := []string{
		"main.py",
		"app.py",
		"agent.py",
		"__main__.py",
		"run.py",
		"server.py",
	}

	// Common directories to search in order
	searchDirs := []string{
		"",          // Root directory
		"src",       // src/
		"src/app",   // src/app/
		"src/agent", // src/agent/
		"app",       // app/
		"agent",     // agent/
	}

	// Try each combination of directory and entry point
	for _, dir := range searchDirs {
		for _, entry := range entryPoints {
			var path string
			if dir == "" {
				path = filepath.Join(projectDir, entry)
			} else {
				path = filepath.Join(projectDir, dir, entry)
			}

			if _, err := os.Stat(path); err == nil {
				// Return relative path from project directory
				relPath, err := filepath.Rel(projectDir, path)
				if err != nil {
					relPath = path
				}
				return relPath, nil
			}
		}
	}

	// Provide helpful error message with configuration instructions
	return "", fmt.Errorf("no Python entry point found. Searched for: %v in directories: %v.\n\nTo fix this, you can:\n  1. Create one of the expected entry point files (e.g., main.py, app.py, agent.py)\n  2. OR specify a custom entry point in azure.yaml:\n     services:\n       yourservice:\n         language: python\n         project: ./path/to/service\n         entrypoint: path/to/your/entrypoint.py",
		entryPoints, searchDirs)
}

// RunPython runs a Python project with the detected package manager.
// If an entrypoint is specified in the PythonProject, it takes precedence over auto-detection.
func RunPython(ctx context.Context, project types.PythonProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}
	if err := security.ValidatePackageManager(project.PackageManager); err != nil {
		return fmt.Errorf("invalid package manager: %w", err)
	}

	// Determine entry point: use explicit entrypoint if provided, otherwise auto-detect
	var entryPoint string
	var err error
	if project.Entrypoint != "" {
		// Guard against leading-dash argv injection (CWE-88): an entrypoint like "-c" would
		// be parsed as a CLI flag by the Python interpreter rather than as a file path.
		if err := executor.RejectLeadingDash(project.Entrypoint); err != nil {
			return fmt.Errorf("invalid entrypoint: %w", err)
		}
		entryPoint = project.Entrypoint
		cliout.Info("Starting Python project with %s", project.PackageManager)
		cliout.Item("Directory: %s", project.Dir)
		cliout.Item("Entry point (from azure.yaml): %s", entryPoint)
		cliout.Newline()
	} else {
		entryPoint, err = findPythonEntryPoint(project.Dir)
		if err != nil {
			cliout.Error("Failed to find Python entry point")
			cliout.Newline()
			return err
		}
		cliout.Info("Starting Python project with %s", project.PackageManager)
		cliout.Item("Directory: %s", project.Dir)
		cliout.Item("Entry point (auto-detected): %s", entryPoint)
		cliout.Newline()
	}

	cmd, args, err := pythonRunCommand(project.PackageManager, project.Dir, entryPoint)
	if err != nil {
		return err
	}

	return executor.StartCommand(ctx, cmd, args, project.Dir)
}

// pythonRunCommand resolves the interpreter command and arguments for a Python
// project. uv and poetry manage their own environments, so the entry point is
// handed to their `run python` subcommand; pip projects are executed with a
// resolved interpreter path instead.
func pythonRunCommand(packageManager, projectDir, entryPoint string) (cmd string, args []string, err error) {
	switch packageManager {
	case "uv", "poetry":
		return packageManager, []string{"run", "python", entryPoint}, nil
	case "pip":
		return pipInterpreter(projectDir), []string{entryPoint}, nil
	default:
		return "", nil, fmt.Errorf("unsupported package manager: %s", packageManager)
	}
}

// pipInterpreter returns the interpreter to use for a pip project, preferring a
// project-local virtualenv (".venv", then "venv") before falling back to the
// system interpreter on PATH.
func pipInterpreter(projectDir string) string {
	for _, venvDir := range []string{".venv", "venv"} {
		candidate := venvPythonPath(projectDir, venvDir)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "python"
}

// venvPythonPath returns the platform-specific interpreter path inside a
// virtualenv directory. Windows venvs use Scripts/python.exe; everything else
// uses bin/python.
func venvPythonPath(projectDir, venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectDir, venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(projectDir, venvDir, "bin", "python")
}

// RunDotnet runs a .NET project with 'dotnet run'.
func RunDotnet(ctx context.Context, project types.DotnetProject) error {
	// Validate inputs
	if err := security.ValidatePath(project.Path); err != nil {
		return fmt.Errorf("invalid project path: %w", err)
	}

	cliout.Info("Starting .NET project...")
	cliout.Item("Project: %s", project.Path)
	cliout.Newline()

	args, dir, err := dotnetRunArgs(project.Path)
	if err != nil {
		return err
	}

	return executor.StartCommand(ctx, "dotnet", args, dir)
}

// dotnetRunArgs builds the `dotnet run` argument list and working directory for
// a project path. A .sln cannot be passed via --project, so solutions are run
// from their own directory; project files are passed explicitly and therefore
// run from the caller's working directory.
func dotnetRunArgs(projectPath string) (args []string, dir string, err error) {
	if filepath.Ext(projectPath) == ".sln" {
		return []string{"run"}, filepath.Dir(projectPath), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return []string{"run", "--project", projectPath}, cwd, nil
}

// RunFunctionApp runs an Azure Functions project (any variant) with Azure Functions Core Tools.
// This is the unified runner for all Azure Functions variants including Logic Apps Standard.
func RunFunctionApp(ctx context.Context, project types.FunctionAppProject, port int) error {
	// Validate inputs
	if err := security.ValidatePath(project.Dir); err != nil {
		return fmt.Errorf("invalid project directory: %w", err)
	}

	// Validate required files exist
	hostJSONPath := filepath.Join(project.Dir, "host.json")
	if _, err := os.Stat(hostJSONPath); os.IsNotExist(err) {
		return fmt.Errorf("azure Functions project missing host.json: run 'func init' to initialize the project")
	}

	// Variant-specific validation
	if project.Variant == "logicapps" {
		workflowsPath := filepath.Join(project.Dir, "workflows")
		if info, err := os.Stat(workflowsPath); err != nil || !info.IsDir() {
			return fmt.Errorf("logic Apps project missing workflows/ directory")
		}
	}

	// Get display name for variant
	variantDisplayName := getVariantDisplayName(project.Variant)

	cliout.Info("Starting %s project...", variantDisplayName)
	cliout.Item("Directory: %s", project.Dir)
	cliout.Item("Language: %s", project.Language)
	cliout.Item("Port: %d", port)
	cliout.Newline()

	// Run Azure Functions Core Tools
	args := []string{"start", "--port", fmt.Sprintf("%d", port)}
	return executor.StartCommand(ctx, "func", args, project.Dir)
}

// getVariantDisplayName returns a user-friendly display name for a Functions variant.
func getVariantDisplayName(variant string) string {
	switch variant {
	case "logicapps":
		return "Logic Apps Standard"
	case "nodejs":
		return "Node.js Functions"
	case "python":
		return "Python Functions"
	case "dotnet":
		return ".NET Functions"
	case "java":
		return "Java Functions"
	default:
		return "Azure Functions"
	}
}
