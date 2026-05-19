package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/browser"
	"github.com/jongio/azd-core/cliout"
)

// executePrerunHook executes the prerun hook if configured.
func executePrerunHook(azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPrerun(), "prerun", workingDir)
}

// executePostrunHook executes the postrun hook if configured.
func executePostrunHook(azureYaml *service.AzureYaml, workingDir string) error {
	return executeHook(azureYaml, azureYaml.Hooks, azureYaml.Hooks.GetPostrun(), "postrun", workingDir)
}

// executeHook executes a lifecycle hook with the given name and configuration.
// This is a common helper function to avoid duplication between prerun and postrun hooks.
func executeHook(azureYaml *service.AzureYaml, hooks *service.Hooks, hook *service.Hook, hookName, workingDir string) error {
	if hooks == nil || hook == nil {
		return nil // No hook configured
	}

	// Convert service.Hook to executor.Hook
	convertedHook := convertHook(hook)
	config := executor.ResolveHookConfig(convertedHook)
	if config == nil {
		return nil
	}

	// Build environment variables for the hook
	// Following azd pattern: pass project directory and any other context
	hookEnvVars := buildHookEnvironmentVariables(azureYaml, workingDir)
	config.Env = hookEnvVars

	return executor.ExecuteHook(context.Background(), hookName, *config, workingDir)
}

// buildHookEnvironmentVariables builds environment variables to pass to hooks
// Following the pattern from azure/azure-dev
func buildHookEnvironmentVariables(azureYaml *service.AzureYaml, workingDir string) []string {
	envVars := []string{
		fmt.Sprintf("%s=%s", executor.EnvProjectDir, workingDir),
		fmt.Sprintf("%s=%s", executor.EnvProjectName, azureYaml.Name),
	}

	// Add count of services for context
	if azureYaml.Services != nil {
		envVars = append(envVars, fmt.Sprintf("%s=%d", executor.EnvServiceCount, len(azureYaml.Services)))
	}

	return envVars
}

// convertHook converts service.Hook to executor.Hook to avoid circular imports.
func convertHook(h *service.Hook) *executor.Hook {
	if h == nil {
		return nil
	}
	return executor.NewHook(
		h.Run,
		h.Shell,
		h.ContinueOnError,
		h.Interactive,
		convertPlatformHook(h.Windows),
		convertPlatformHook(h.Posix),
	)
}

// convertPlatformHook converts service.PlatformHook to executor.PlatformHook.
func convertPlatformHook(ph *service.PlatformHook) *executor.PlatformHook {
	if ph == nil {
		return nil
	}
	return executor.NewPlatformHook(
		ph.Run,
		ph.Shell,
		ph.ContinueOnError,
		ph.Interactive,
	)
}

// resolveBrowserTarget determines which browser target to use.
// Browser is OFF by default. Only opens if --web flag is specified.
func resolveBrowserTarget(_ *service.AzureYaml) browser.Target {
	if runWeb {
		return browser.TargetSystem
	}
	return browser.TargetNone
}

// launchDashboardBrowser launches the dashboard in the configured browser.
// Returns true if browser was launched, false if not (e.g., target is none).
func launchDashboardBrowser(dashboardURL string) bool {
	// Parse azure.yaml to get project config for browser preference
	azureYamlPath, err := findAzureYaml()
	var azureYaml *service.AzureYaml
	if err == nil {
		azureYaml, _ = service.ParseAzureYaml(azureYamlPath)
	}

	// Resolve browser target using priority system
	target := resolveBrowserTarget(azureYaml)

	// If target is none, don't launch
	if target == browser.TargetNone {
		return false
	}

	// Display launch message
	targetName := browser.GetTargetDisplayName(target)
	cliout.Plain("  Opening in %s...", targetName)

	// Launch browser (non-blocking)
	if err := browser.Launch(browser.LaunchOptions{
		URL:     dashboardURL,
		Target:  target,
		Timeout: 5 * time.Second,
	}); err != nil {
		cliout.Warning("Could not open browser: %v", err)
		cliout.Info("Dashboard available at: %s", dashboardURL)
	}
	return true
}
