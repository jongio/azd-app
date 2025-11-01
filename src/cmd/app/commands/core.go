package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"app/src/internal/detector"
	"app/src/internal/installer"
	"app/src/internal/orchestrator"
	"app/src/internal/runner"
	"app/src/internal/security"
	"app/src/internal/types"

	"gopkg.in/yaml.v3"
)

// Global orchestrator instance shared across all commands.
var cmdOrchestrator *orchestrator.Orchestrator

// init initializes the command orchestrator and registers all commands.
func init() {
	cmdOrchestrator = orchestrator.NewOrchestrator()

	// Register commands with their dependencies
	// reqs has no dependencies
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:    "reqs",
		Execute: executeReqs,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register reqs command: %v\n", err)
		os.Exit(1)
	}

	// deps depends on reqs
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "deps",
		Dependencies: []string{"reqs"},
		Execute:      executeDeps,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register deps command: %v\n", err)
		os.Exit(1)
	}

	// run depends on deps (which transitively depends on reqs)
	if err := cmdOrchestrator.Register(&orchestrator.Command{
		Name:         "run",
		Dependencies: []string{"deps"},
		Execute:      executeRun,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register run command: %v\n", err)
		os.Exit(1)
	}
}

// executeReqs is the core logic for the reqs command.
func executeReqs() error {
	fmt.Println("🔍 Checking requirements...")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find azure.yaml in current or parent directories
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		return fmt.Errorf("error searching for azure.yaml: %w", err)
	}

	if azureYamlPath == "" {
		fmt.Println("ℹ️  No azure.yaml found - skipping requirement check")
		fmt.Println()
		return nil
	}

	// Validate path to azure.yaml
	if err := security.ValidatePath(azureYamlPath); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// #nosec G304 -- Path validated by security.ValidatePath above
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		return fmt.Errorf("failed to read azure.yaml: %w", err)
	}

	var azureYaml AzureYaml
	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		return fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	if len(azureYaml.Requirements) == 0 {
		fmt.Println("ℹ️  No requirements defined in azure.yaml")
		fmt.Println()
		return nil
	}

	allSatisfied := true
	for _, prereq := range azureYaml.Requirements {
		passed := checkPrerequisite(prereq)
		if !passed {
			allSatisfied = false
		}
	}

	fmt.Println()
	if !allSatisfied {
		return fmt.Errorf("requirement check failed")
	}

	fmt.Println("✅ All requirements satisfied!")
	return nil
}

// executeDeps is the core logic for the deps command.
func executeDeps() error {
	fmt.Println("🔍 Installing dependencies...")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	hasProjects := false

	// Step 1: Find and install Node.js projects
	nodeProjects, err := detector.FindNodeProjects(cwd)
	if err == nil && len(nodeProjects) > 0 {
		hasProjects = true
		fmt.Printf("📦 Found %d Node.js project(s)\n", len(nodeProjects))
		for _, nodeProject := range nodeProjects {
			if err := installer.InstallNodeDependencies(nodeProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to install for %s: %v\n", nodeProject.Dir, err)
			}
		}
		fmt.Println()
	}

	// Step 2: Find and install Python projects
	pythonProjects, err := detector.FindPythonProjects(cwd)
	if err == nil && len(pythonProjects) > 0 {
		hasProjects = true
		fmt.Printf("🐍 Found %d Python project(s)\n", len(pythonProjects))
		for _, pyProject := range pythonProjects {
			if err := installer.SetupPythonVirtualEnv(pyProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to setup environment for %s: %v\n", pyProject.Dir, err)
			}
		}
		fmt.Println()
	}

	// Step 3: Find and install .NET projects
	dotnetProjects, err := detector.FindDotnetProjects(cwd)
	if err == nil && len(dotnetProjects) > 0 {
		hasProjects = true
		fmt.Printf("🔷 Found %d .NET project(s)\n", len(dotnetProjects))
		for _, dotnetProject := range dotnetProjects {
			if err := installer.RestoreDotnetProject(dotnetProject); err != nil {
				fmt.Printf("   ⚠️  Warning: Failed to restore %s: %v\n", dotnetProject.Path, err)
			}
		}
		fmt.Println()
	}

	if !hasProjects {
		fmt.Println("ℹ️  No projects detected - skipping dependency installation")
		fmt.Println()
		return nil
	}

	fmt.Println("✅ Dependencies installed successfully!")
	return nil
}

// executeRun is the core logic for the run command.
func executeRun() error {
	fmt.Println("🚀 Starting development environment...")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Strategy 1: Check if current directory is itself a runnable project
	// This allows running from within any project directory

	// Check for Aspire project (AppHost.cs) in current directory first
	aspireProject, err := detector.FindAppHost(cwd)
	if err == nil && aspireProject != nil {
		fmt.Println("✨ Found Aspire project:", aspireProject.Dir)
		return runner.RunAspire(*aspireProject)
	}

	// Check for Python project in current directory
	if isPythonProject(cwd) {
		packageManager := detector.DetectPythonPackageManager(cwd)
		pythonProject := types.PythonProject{
			Dir:            cwd,
			PackageManager: packageManager,
		}
		fmt.Printf("✨ Found Python project (%s)\n", packageManager)
		return runner.RunPython(pythonProject)
	}

	// Check for Node.js project in current directory
	if isNodeProject(cwd) {
		packageManager := detector.DetectNodePackageManager(cwd)
		script := detector.DetectPnpmScript(cwd)
		if script == "" {
			return fmt.Errorf("no dev or start script found in package.json")
		}
		nodeProject := types.NodeProject{
			Dir:            cwd,
			PackageManager: packageManager,
		}
		fmt.Printf("✨ Found Node.js project (%s)\n", packageManager)
		return runner.RunNode(nodeProject, script)
	}

	// Check for .NET project in current directory
	if isDotnetProject(cwd) {
		// Find the .csproj or .sln in current directory
		dotnetProjects, err := detector.FindDotnetProjects(cwd)
		if err == nil && len(dotnetProjects) > 0 {
			fmt.Println("✨ Found .NET project")
			return runner.RunDotnet(dotnetProjects[0])
		}
	}

	// Strategy 2: Look for docker compose in package.json
	if detector.HasPackageJson(cwd) {
		if detector.HasDockerComposeScript(cwd) {
			scriptName := detector.FindDockerComposeScript(cwd)
			if scriptName != "" {
				fmt.Println("✨ Found docker compose script in package.json")
				return runner.RunDockerCompose(scriptName, "docker compose up")
			}
		}
	}

	// Strategy 3: Fall back to scanning for projects (old behavior)
	// This handles cases where user runs from a parent directory

	hasProjects := false

	// Check for Node.js projects
	nodeProjects, err := detector.FindNodeProjects(cwd)
	if err == nil && len(nodeProjects) > 0 {
		hasProjects = true
		fmt.Printf("📦 Found %d Node.js project(s)\n", len(nodeProjects))
		for _, nodeProject := range nodeProjects {
			// Try to run dev or start script
			script := detector.DetectPnpmScript(nodeProject.Dir)
			if script != "" {
				fmt.Printf("   ✨ Starting %s with script: %s\n", nodeProject.Dir, script)
				return runner.RunNode(nodeProject, script)
			}
		}
	}

	// Check for Python projects
	pythonProjects, err := detector.FindPythonProjects(cwd)
	if err == nil && len(pythonProjects) > 0 {
		hasProjects = true
		fmt.Printf("🐍 Found %d Python project(s)\n", len(pythonProjects))
		for _, pyProject := range pythonProjects {
			fmt.Printf("   • %s (%s)\n", pyProject.Dir, pyProject.PackageManager)
		}
		fmt.Println()
		fmt.Println("ℹ️  Python projects detected")
		fmt.Println("   Tip: Run 'azd app run' from within the Python project directory")
		fmt.Println("   Or add an Aspire AppHost for orchestrated startup")
	}

	// Check for .NET projects
	dotnetProjects, err := detector.FindDotnetProjects(cwd)
	if err == nil && len(dotnetProjects) > 0 {
		hasProjects = true
		fmt.Printf("🔷 Found %d .NET project(s)\n", len(dotnetProjects))
		for _, dotnetProject := range dotnetProjects {
			fmt.Printf("   • %s\n", dotnetProject.Path)
		}
		fmt.Println()
		fmt.Println("ℹ️  .NET projects detected")
		fmt.Println("   Tip: Run 'azd app run' from within the .NET project directory")
		fmt.Println("   Or add an Aspire AppHost for orchestrated startup")
	}

	if !hasProjects {
		fmt.Println("❌ No development environment detected!")
		fmt.Println()
		fmt.Println("Supported environments:")
		fmt.Println("  • .NET Aspire (AppHost.cs) - recommended for multi-project solutions")
		fmt.Println("  • Docker Compose (in package.json)")
		fmt.Println("  • Node.js with dev/start scripts")
		fmt.Println("  • Python projects (main.py, app.py)")
		fmt.Println("  • .NET projects (.csproj, .sln)")
		return fmt.Errorf("no development environment found")
	}

	return fmt.Errorf("no runnable project configuration found in current directory")
}

// Helper functions to check if a directory contains specific project types

func isPythonProject(dir string) bool {
	// Check for Python project indicators
	indicators := []string{"requirements.txt", "pyproject.toml", "poetry.lock", "uv.lock"}
	for _, indicator := range indicators {
		if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
			return true
		}
	}
	return false
}

func isNodeProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil
}

func isDotnetProject(dir string) bool {
	// Check for .csproj or .sln files in current directory
	matches, _ := filepath.Glob(filepath.Join(dir, "*.csproj"))
	if len(matches) > 0 {
		return true
	}
	matches, _ = filepath.Glob(filepath.Join(dir, "*.sln"))
	return len(matches) > 0
}
