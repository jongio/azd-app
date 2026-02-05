// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-core/keyvault"
	"github.com/jongio/azd-core/security"
)

// ResolveEnvironment merges environment variables from multiple sources and resolves Azure Key Vault references.
// Priority (highest to lowest): service-specific env > .env file > azure environment > OS environment.
// This function ensures that azd context variables (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)
// are preserved and passed to all child processes, as required by the azd extension framework.
// Key Vault references in any of these sources are automatically resolved to their actual secret values.
func ResolveEnvironment(ctx context.Context, service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error) {
	env := make(map[string]string)

	// Start with OS environment - this includes azd context variables when running as an azd extension:
	// - AZD_SERVER: gRPC server address for azd communication
	// - AZD_ACCESS_TOKEN: Authentication token for azd API
	// - AZURE_*: All Azure environment variables from azd env
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}

	// Merge Azure environment variables (from azd context) - these override OS env
	for k, v := range azureEnv {
		env[k] = v
	}

	// Load and merge .env file if specified - these override Azure env
	if dotEnvPath != "" {
		dotEnv, err := LoadDotEnv(dotEnvPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
		for k, v := range dotEnv {
			env[k] = v
		}
	}

	// Merge auto-generated service URLs - these override .env file
	for k, v := range serviceURLs {
		env[k] = v
	}

	// Merge service-specific environment variables from azure.yaml - highest priority
	serviceEnv := service.GetEnvironment()
	for name, value := range serviceEnv {
		// Perform variable substitution
		value = substituteEnvVars(value, env)
		env[name] = value
	}

	// Resolve Azure Key Vault references if present
	envSlice := envMapToSlice(env)
	if hasKeyVaultReferences(envSlice) {
		resolvedSlice, err := resolveKeyVaultReferences(ctx, envSlice)
		if err != nil {
			// Log warning but continue - graceful degradation
			fmt.Fprintf(os.Stderr, "Warning: Key Vault resolution encountered errors: %v\n", err)
		} else {
			// Convert back to map
			env = envSliceToMap(resolvedSlice)
		}
	}

	return env, nil
}

// resolveKeyVaultReferences resolves Key Vault references in environment variables.
// Returns the resolved variables or an error if resolution fails.
func resolveKeyVaultReferences(ctx context.Context, envVars []string) ([]string, error) {
	if len(envVars) == 0 {
		return envVars, nil
	}

	resolver, err := keyvault.NewKeyVaultResolver()
	if err != nil {
		// Log warning and return original values (graceful degradation)
		fmt.Fprintf(os.Stderr, "Warning: failed to create Key Vault resolver: %v\n", err)
		// Return original values, not an error - this ensures env vars without KV references still work
		return envVars, nil
	}
	if resolver == nil {
		// Defensive check - should never happen but prevents nil pointer dereference
		fmt.Fprintf(os.Stderr, "Warning: Key Vault resolver is nil, skipping resolution\n")
		return envVars, nil
	}

	resolvedVars, warnings, err := resolver.ResolveEnvironmentVariables(ctx, envVars, keyvault.ResolveEnvironmentOptions{
		StopOnError: false, // Graceful degradation by default
	})

	// Log any warnings
	for _, w := range warnings {
		// Check if debug mode is enabled before exposing variable names
		if debug := os.Getenv("AZD_DEBUG"); debug == "true" || debug == "1" {
			// In debug mode, include the variable key for troubleshooting
			if w.Key != "" {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve Key Vault reference for %s: %v\n", w.Key, w.Err)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", w.Err)
			}
		} else {
			// In normal mode, use generic message to avoid exposing sensitive variable names
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve Key Vault reference: %v\n", w.Err)
		}
	}

	if err != nil {
		// With StopOnError=false, this shouldn't happen, but handle it anyway
		return envVars, err
	}

	return resolvedVars, nil
}

// hasKeyVaultReferences checks if any environment variables contain Key Vault references.
func hasKeyVaultReferences(envVars []string) bool {
	if len(envVars) == 0 {
		return false
	}

	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 && keyvault.IsKeyVaultReference(parts[1]) {
			return true
		}
	}
	return false
}

// envMapToSlice converts an environment map to a slice of KEY=VALUE strings.
func envMapToSlice(env map[string]string) []string {
	if len(env) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// envSliceToMap converts a slice of KEY=VALUE strings to a map.
func envSliceToMap(envSlice []string) map[string]string {
	if len(envSlice) == 0 {
		return make(map[string]string)
	}

	result := make(map[string]string, len(envSlice))
	for _, envVar := range envSlice {
		// Skip empty lines
		if envVar == "" {
			continue
		}

		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			// Validate key doesn't contain invalid characters (security)
			// Environment variable names should only contain alphanumeric, underscore, and a limited set of safe chars
			// This prevents injection attacks via malformed env vars
			if key == "" || !isValidEnvVarName(key) {
				continue
			}
			// Also validate value doesn't contain null bytes (security)
			if strings.Contains(value, "\000") {
				continue
			}
			result[key] = value
		}
	}
	return result
}

// InjectFunctionsWorkerRuntime adds FUNCTIONS_WORKER_RUNTIME and other required settings
// for Logic Apps and Azure Functions. This prevents func CLI from prompting interactively.
// Also injects azd env values for Logic Apps connection configuration.
func InjectFunctionsWorkerRuntime(env map[string]string, runtime *ServiceRuntime) map[string]string {
	// Only inject for Logic Apps Standard and Azure Functions
	isFunctions := strings.Contains(runtime.Framework, "Logic Apps") ||
		strings.Contains(runtime.Framework, "Functions")

	if !isFunctions {
		return env
	}

	// Check if there's a local.settings.json to read settings from
	localSettingsPath := filepath.Join(runtime.WorkingDir, "local.settings.json")
	if settings := loadLocalSettings(localSettingsPath); settings != nil {
		// Inject missing settings from local.settings.json
		for key, value := range settings {
			if _, exists := env[key]; !exists {
				env[key] = value
			}
		}
	}

	// Inject FUNCTIONS_WORKER_RUNTIME if still missing
	if _, exists := env["FUNCTIONS_WORKER_RUNTIME"]; !exists {
		// Default based on framework type
		if strings.Contains(runtime.Framework, "Logic Apps") {
			env["FUNCTIONS_WORKER_RUNTIME"] = "node"
		}
	}

	// Inject AzureWebJobsStorage if missing (required for local dev)
	if _, exists := env["AzureWebJobsStorage"]; !exists {
		env["AzureWebJobsStorage"] = "UseDevelopmentStorage=true"
	}

	return env
}

// loadLocalSettings reads all Values from local.settings.json.
func loadLocalSettings(path string) map[string]string {
	// #nosec G304 -- Path is constructed from validated runtime.WorkingDir
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var config struct {
		Values map[string]string `json:"Values"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	return config.Values
}

// GenerateServiceURLs creates auto-generated environment variables for service URLs.
func GenerateServiceURLs(processes map[string]*ServiceProcess) map[string]string {
	urls := make(map[string]string)

	for name, process := range processes {
		if process == nil || !process.Ready {
			continue
		}

		serviceName := strings.ToUpper(name)
		serviceName = strings.ReplaceAll(serviceName, "-", "_")

		// SERVICE_URL_{NAME}
		urls[fmt.Sprintf("SERVICE_URL_%s", serviceName)] = process.URL

		// SERVICE_PORT_{NAME}
		urls[fmt.Sprintf("SERVICE_PORT_%s", serviceName)] = fmt.Sprintf("%d", process.Port)

		// SERVICE_HOST_{NAME}
		urls[fmt.Sprintf("SERVICE_HOST_%s", serviceName)] = "localhost"
	}

	return urls
}

// LoadDotEnv loads environment variables from a .env file.
func LoadDotEnv(path string) (map[string]string, error) {
	if err := security.ValidatePath(path); err != nil {
		return nil, fmt.Errorf("invalid .env file path: %w", err)
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer func() { _ = file.Close() }()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return env, nil
}

// isValidEnvVarName checks if an environment variable name is valid.
// Valid names contain only alphanumeric characters, underscores, and limited safe characters.
// This prevents injection attacks via malformed environment variable names.
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	// Check for control characters and other dangerous characters
	if strings.ContainsAny(name, "\n\r\t\000=$;|&<>(){}[]`\"'\\") {
		return false
	}
	// Must start with letter or underscore (POSIX requirement)
	if len(name) > 0 {
		first := name[0]
		if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') && first != '_' {
			return false
		}
	}
	// Remaining characters must be alphanumeric or underscore
	for _, ch := range name {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}

// substituteEnvVars performs variable substitution in a string.
// Supports ${VAR} and $VAR syntax.
func substituteEnvVars(value string, env map[string]string) string {
	// Replace ${VAR} syntax
	result := os.Expand(value, func(key string) string {
		if val, exists := env[key]; exists {
			return val
		}
		return os.Getenv(key)
	})

	return result
}

// MaskSecrets masks secret values in environment variables for display.
// Note: With the new Docker Compose-compatible format, secrets are handled inline
// and we don't track which variables are secrets separately. This function is
// kept for backward compatibility but may need updating if secret tracking is needed.
func MaskSecrets(service Service, env map[string]string) map[string]string {
	masked := make(map[string]string)

	// Create a set of secret variable names from common patterns
	secrets := make(map[string]bool)
	for key := range env {
		keyUpper := strings.ToUpper(key)
		// Mask common secret patterns
		if strings.Contains(keyUpper, "SECRET") ||
			strings.Contains(keyUpper, "PASSWORD") ||
			strings.Contains(keyUpper, "TOKEN") ||
			strings.Contains(keyUpper, "KEY") && !strings.Contains(keyUpper, "PUBLIC") {
			secrets[key] = true
		}
	}

	// Mask secrets
	for k, v := range env {
		if secrets[k] {
			masked[k] = "***"
		} else {
			masked[k] = v
		}
	}

	return masked
}

// LoadEnvFileIfExists loads a .env file if it exists, otherwise returns empty map.
func LoadEnvFileIfExists(projectDir string, filename string) (map[string]string, error) {
	envPath := filepath.Join(projectDir, filename)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	return LoadDotEnv(envPath)
}
