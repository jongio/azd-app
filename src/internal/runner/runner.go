package runner

import (
	"fmt"
	"os"

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
	fmt.Println()

	// Run aspire run and let it handle all output
	return executor.StartCommand("aspire", []string{"run"}, project.Dir)
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
