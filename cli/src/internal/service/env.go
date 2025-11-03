package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"
)

// ResolveEnvironment merges environment variables from multiple sources.
// Priority: service-specific env > .env file > azure environment > OS environment.
func ResolveEnvironment(service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error) {
	env := make(map[string]string)

	// Start with OS environment
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}

	// Merge Azure environment variables (from azd context)
	for k, v := range azureEnv {
		env[k] = v
	}

	// Load and merge .env file if specified
	if dotEnvPath != "" {
		dotEnv, err := LoadDotEnv(dotEnvPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
		for k, v := range dotEnv {
			env[k] = v
		}
	}

	// Merge auto-generated service URLs
	for k, v := range serviceURLs {
		env[k] = v
	}

	// Merge service-specific environment variables from azure.yaml
	for _, envVar := range service.Env {
		value := envVar.Value
		if envVar.Secret != "" {
			value = envVar.Secret
		}

		// Perform variable substitution
		value = substituteEnvVars(value, env)
		env[envVar.Name] = value
	}

	return env, nil
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
	defer file.Close()

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
func MaskSecrets(service Service, env map[string]string) map[string]string {
	masked := make(map[string]string)

	// Create a set of secret variable names
	secrets := make(map[string]bool)
	for _, envVar := range service.Env {
		if envVar.Secret != "" {
			secrets[envVar.Name] = true
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
