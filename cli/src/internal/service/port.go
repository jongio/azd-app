package service

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/security"

	"gopkg.in/yaml.v3"
)

// DetectPort attempts to detect the port for a service using multiple strategies.
// Returns (port, isExplicit, error).
// isExplicit is true when the port comes from azure.yaml config - these ports are mandatory and cannot be changed.
// Priority: azure.yaml config > framework config files > environment variables > framework defaults > dynamic assignment.
func DetectPort(serviceName string, service Service, projectDir string, framework string, usedPorts map[int]bool) (int, bool, error) {
	// Priority 1: Explicit port in azure.yaml config (MANDATORY - never change these)
	if service.Config != nil {
		if portVal, exists := service.Config["port"]; exists {
			switch v := portVal.(type) {
			case int:
				return v, true, nil // isExplicit = true
			case float64:
				return int(v), true, nil // isExplicit = true
			case string:
				if port, err := strconv.Atoi(v); err == nil {
					return port, true, nil // isExplicit = true
				}
			}
		}
	}

	// Priority 2: Framework-specific configuration files
	if port, err := detectPortFromFrameworkConfig(projectDir, framework); err == nil && port > 0 {
		return port, false, nil // isExplicit = false
	}

	// Priority 3: Environment variables
	if port := detectPortFromEnv(serviceName); port > 0 {
		return port, false, nil // isExplicit = false
	}

	// Priority 4: Framework defaults
	if port := getFrameworkDefaultPort(framework, service.Language); port > 0 {
		// Check if port is already in use
		if !usedPorts[port] {
			return port, false, nil // isExplicit = false
		}
	}

	// Priority 5: Dynamic port assignment
	port, err := findAvailablePort(3000, usedPorts)
	return port, false, err // isExplicit = false
}

// detectPortFromFrameworkConfig reads framework-specific config files to find the port.
func detectPortFromFrameworkConfig(projectDir string, framework string) (int, error) {
	switch framework {
	case "Next.js", "React", "Vue", "Angular", "Express", "NestJS":
		return detectPortFromPackageJSON(projectDir)
	case "ASP.NET Core", "Aspire":
		return detectPortFromLaunchSettings(projectDir)
	case "Django":
		return detectPortFromDjangoSettings(projectDir)
	case "Spring Boot":
		return detectPortFromSpringConfig(projectDir)
	}

	return 0, fmt.Errorf("no port detection for framework: %s", framework)
}

// detectPortFromPackageJSON looks for port in npm scripts.
func detectPortFromPackageJSON(projectDir string) (int, error) {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	if err := security.ValidatePath(packageJSONPath); err != nil {
		return 0, err
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return 0, err
	}

	var packageJSON struct {
		Scripts map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return 0, err
	}

	// Look for port in dev or start scripts
	for _, scriptName := range []string{"dev", "start", "serve"} {
		if script, exists := packageJSON.Scripts[scriptName]; exists {
			if port := extractPortFromCommand(script); port > 0 {
				return port, nil
			}
		}
	}

	return 0, fmt.Errorf("no port found in package.json scripts")
}

// detectPortFromLaunchSettings reads .NET launchSettings.json.
func detectPortFromLaunchSettings(projectDir string) (int, error) {
	launchSettingsPath := filepath.Join(projectDir, "Properties", "launchSettings.json")
	if err := security.ValidatePath(launchSettingsPath); err != nil {
		return 0, err
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(launchSettingsPath)
	if err != nil {
		return 0, err
	}

	var launchSettings struct {
		Profiles map[string]struct {
			ApplicationURL string `json:"applicationUrl"`
		} `json:"profiles"`
	}

	if err := json.Unmarshal(data, &launchSettings); err != nil {
		return 0, err
	}

	// Look for HTTP URL in profiles
	for _, profile := range launchSettings.Profiles {
		if profile.ApplicationURL != "" {
			// Parse URLs like "http://localhost:5000;https://localhost:5001"
			urls := strings.Split(profile.ApplicationURL, ";")
			for _, url := range urls {
				url = strings.TrimSpace(url)
				if strings.HasPrefix(url, "http://") {
					if port := extractPortFromURL(url); port > 0 {
						return port, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("no port found in launchSettings.json")
}

// detectPortFromDjangoSettings reads Django settings.py for PORT.
func detectPortFromDjangoSettings(projectDir string) (int, error) {
	// Django typically uses default port 8000, but check settings.py
	settingsPath := filepath.Join(projectDir, "settings.py")
	if err := security.ValidatePath(settingsPath); err != nil {
		// Try common Django structure
		settingsPath = filepath.Join(projectDir, filepath.Base(projectDir), "settings.py")
		if err := security.ValidatePath(settingsPath); err != nil {
			return 0, err
		}
	}

	// #nosec G304 -- Path validated by security.ValidatePath
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return 0, err
	}

	content := string(data)
	portRegex := regexp.MustCompile(`PORT\s*=\s*(\d+)`)
	if matches := portRegex.FindStringSubmatch(content); len(matches) > 1 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no PORT in Django settings")
}

// detectPortFromSpringConfig reads Spring Boot application.properties or application.yml.
func detectPortFromSpringConfig(projectDir string) (int, error) {
	// Try application.properties
	propsPath := filepath.Join(projectDir, "src", "main", "resources", "application.properties")
	if err := security.ValidatePath(propsPath); err == nil {
		// #nosec G304 -- Path validated by security.ValidatePath
		if data, err := os.ReadFile(propsPath); err == nil {
			content := string(data)
			portRegex := regexp.MustCompile(`server\.port\s*=\s*(\d+)`)
			if matches := portRegex.FindStringSubmatch(content); len(matches) > 1 {
				if port, err := strconv.Atoi(matches[1]); err == nil {
					return port, nil
				}
			}
		}
	}

	// Try application.yml
	ymlPath := filepath.Join(projectDir, "src", "main", "resources", "application.yml")
	if err := security.ValidatePath(ymlPath); err == nil {
		// #nosec G304 -- Path validated by security.ValidatePath
		if data, err := os.ReadFile(ymlPath); err == nil {
			var config struct {
				Server struct {
					Port int `yaml:"port"`
				} `yaml:"server"`
			}
			if err := yaml.Unmarshal(data, &config); err == nil && config.Server.Port > 0 {
				return config.Server.Port, nil
			}
		}
	}

	return 0, fmt.Errorf("no server.port in Spring Boot config")
}

// detectPortFromEnv checks environment variables for port configuration.
func detectPortFromEnv(serviceName string) int {
	// Check service-specific env var
	servicePortVar := fmt.Sprintf("%s_PORT", strings.ToUpper(serviceName))
	if portStr := os.Getenv(servicePortVar); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			return port
		}
	}

	// Check common port env vars
	for _, envVar := range []string{"PORT", "HTTP_PORT", "WEB_PORT", "SERVICE_PORT"} {
		if portStr := os.Getenv(envVar); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
				return port
			}
		}
	}

	return 0
}

// getFrameworkDefaultPort returns the default port for a framework or language.
func getFrameworkDefaultPort(framework string, language string) int {
	// Check framework-specific defaults first
	frameworkDefaults := map[string]int{
		"Next.js":      3000,
		"React":        5173,
		"Vue":          5173,
		"Angular":      4200,
		"Express":      3000,
		"NestJS":       3000,
		"Svelte":       5173,
		"Astro":        4321,
		"Remix":        3000,
		"Nuxt":         3000,
		"Django":       8000,
		"FastAPI":      8000,
		"Flask":        5000,
		"Streamlit":    8501,
		"Gradio":       7860,
		"ASP.NET Core": 5000,
		"Aspire":       15888,
		"Blazor":       5000,
		"Spring Boot":  8080,
		"Quarkus":      8080,
		"Micronaut":    8080,
	}

	if port, exists := frameworkDefaults[framework]; exists {
		return port
	}

	// Fall back to language defaults
	langLower := strings.ToLower(language)
	if port, exists := DefaultPorts[langLower]; exists {
		return port
	}

	return 0
}

// extractPortFromCommand extracts port number from a command string.
// Handles patterns like: --port 3000, --port=3000, -p 3000, -p=3000.
func extractPortFromCommand(cmd string) int {
	portRegex := regexp.MustCompile(`(?:--port[=\s]|:)(\d+)`)
	if matches := portRegex.FindStringSubmatch(cmd); len(matches) > 1 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			return port
		}
	}
	return 0
}

// extractPortFromURL extracts port from URL string.
func extractPortFromURL(url string) int {
	portRegex := regexp.MustCompile(`:(\d+)`)
	if matches := portRegex.FindStringSubmatch(url); len(matches) > 1 {
		if port, err := strconv.Atoi(matches[1]); err == nil {
			return port
		}
	}
	return 0
}

// findAvailablePort finds an available port starting from startPort.
func findAvailablePort(startPort int, usedPorts map[int]bool) (int, error) {
	for port := startPort; port < 65535; port++ {
		if usedPorts[port] {
			continue
		}

		// Try to bind to the port to check if it's available
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			if closeErr := listener.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close listener: %v\n", closeErr)
			}
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found")
}

// IsPortAvailable checks if a port is available.
func IsPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	if closeErr := listener.Close(); closeErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close listener: %v\n", closeErr)
	}
	return true
}
