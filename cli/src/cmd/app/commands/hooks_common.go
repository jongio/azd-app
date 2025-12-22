package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/executor"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// loadAzureYamlForHooks attempts to load azure.yaml for hook execution.
// Returns nil, nil if azure.yaml is not found (hooks are optional).
// Returns nil, error only if azure.yaml exists but fails to parse.
func loadAzureYamlForHooks() (*service.AzureYaml, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil {
		// No azure.yaml found - this is ok, hooks are optional
		return nil, nil
	}

	azureYaml, err := service.ParseAzureYaml(azureYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse azure.yaml: %w", err)
	}

	return azureYaml, nil
}

// executeCommandHook executes a lifecycle hook for a specific command.
// This is a generic helper that all commands can use.
func executeCommandHook(azureYaml *service.AzureYaml, hook *service.Hook, hookName string, envVars []string) error {
	if azureYaml == nil || azureYaml.Hooks == nil || hook == nil {
		return nil // No hook configured
	}

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Convert service.Hook to executor.Hook
	convertedHook := convertHook(hook)
	config := executor.ResolveHookConfig(convertedHook)
	if config == nil {
		return nil
	}

	// Combine common env vars with command-specific env vars
	commonEnvVars := buildCommonHookEnvVars(azureYaml, workingDir)
	config.Env = append(commonEnvVars, envVars...)

	return executor.ExecuteHook(context.Background(), hookName, *config, workingDir)
}

// buildCommonHookEnvVars builds environment variables common to all hooks.
// Command-specific hooks should call this and append their own variables.
func buildCommonHookEnvVars(azureYaml *service.AzureYaml, workingDir string) []string {
	envVars := []string{
		fmt.Sprintf("%s=%s", executor.EnvProjectDir, workingDir),
	}

	if azureYaml != nil {
		envVars = append(envVars, fmt.Sprintf("%s=%s", executor.EnvProjectName, azureYaml.Name))

		// Add count of services for context
		if azureYaml.Services != nil {
			envVars = append(envVars, fmt.Sprintf("%s=%d", executor.EnvServiceCount, len(azureYaml.Services)))
		}
	}

	return envVars
}

// buildKeyValueEnvVar creates a single key=value environment variable string.
// This is a helper to ensure consistent formatting across all hook env vars.
func buildKeyValueEnvVar(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// buildBoolEnvVar creates a boolean environment variable string.
func buildBoolEnvVar(key string, value bool) string {
	return buildKeyValueEnvVar(key, fmt.Sprintf("%t", value))
}

// buildIntEnvVar creates an integer environment variable string.
func buildIntEnvVar(key string, value int) string {
	return buildKeyValueEnvVar(key, fmt.Sprintf("%d", value))
}

// buildStringListEnvVar creates an environment variable from a string slice (comma-separated).
func buildStringListEnvVar(key string, values []string) string {
	return buildKeyValueEnvVar(key, strings.Join(values, ","))
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
