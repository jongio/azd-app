// Package service provides runtime detection and service orchestration capabilities.
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

// ParsePortSpec parses a Docker Compose style port specification.
//
// Supported formats:
//   - "8080"                    - Single port (behavior depends on isDocker)
//   - "3000:8080"               - Host port 3000 maps to container port 8080
//   - "127.0.0.1:3000:8080"    - Bind to specific IP, host port 3000, container port 8080
//   - "8080/udp"                - Single port with UDP protocol
//   - "3000:8080/tcp"           - Port mapping with explicit TCP protocol
//   - "[::1]:3000:8080"         - IPv6 address binding (brackets required)
//   - "::1:3000:8080"           - IPv6 address binding (alternative format)
//
// Parameters:
//   - spec: The port specification string
//   - isDocker: If true, single ports like "8080" mean container-only (host auto-assigned).
//     If false, single ports mean both host and container use the same port.
//
// Returns:
//   - PortMapping with HostPort, ContainerPort, BindIP, and Protocol fields.
//     HostPort of 0 means auto-assign (only in Docker mode with single port spec).
//
// Examples:
//
//	// Non-Docker service: "8080" means listen on port 8080
//	mapping := ParsePortSpec("8080", false)
//	// Result: HostPort=8080, ContainerPort=8080
//
//	// Docker service: "8080" means expose container port 8080, auto-assign host port
//	mapping := ParsePortSpec("8080", true)
//	// Result: HostPort=0 (auto), ContainerPort=8080
//
//	// Explicit mapping: host 3000 -> container 8080
//	mapping := ParsePortSpec("3000:8080", true)
//	// Result: HostPort=3000, ContainerPort=8080
//
//	// Bind to localhost only
//	mapping := ParsePortSpec("127.0.0.1:3000:8080", false)
//	// Result: BindIP="127.0.0.1", HostPort=3000, ContainerPort=8080
func ParsePortSpec(spec string, isDocker bool) PortMapping {
	spec = strings.TrimSpace(spec)

	// Handle protocol suffix (e.g., "8080/udp")
	protocol := "tcp"
	if strings.Contains(spec, "/") {
		parts := strings.Split(spec, "/")
		spec = parts[0]
		if len(parts) > 1 {
			protocol = strings.ToLower(parts[1])
		}
	}

	// Handle IPv6 addresses in brackets [::1]:3000:8080
	var bindIP string
	if strings.HasPrefix(spec, "[") {
		closeBracket := strings.Index(spec, "]")
		if closeBracket > 0 {
			bindIP = spec[1:closeBracket] // Extract IPv6 without brackets
			spec = spec[closeBracket+1:]  // Keep the rest ":3000:8080"
			spec = strings.TrimPrefix(spec, ":")
		}
	}

	// Split by colons to handle different formats
	parts := strings.Split(spec, ":")

	// If we extracted an IPv6 address, we're in ip:host:container format
	if bindIP != "" {
		if len(parts) == 2 {
			host, _ := strconv.Atoi(parts[0])
			container, _ := strconv.Atoi(parts[1])
			return PortMapping{
				BindIP:        bindIP,
				HostPort:      host,
				ContainerPort: container,
				Protocol:      protocol,
			}
		}
	}

	// Handle IPv6 without brackets (e.g., "::1:3000:8080")
	// If we have more than 3 parts and the first parts contain only hex digits, colons, or are empty,
	// it's likely an IPv6 address
	if len(parts) > 3 {
		// Try to parse as IPv6:host:container
		// Last two parts should be port numbers
		lastIdx := len(parts) - 1
		secondLastIdx := len(parts) - 2

		hostPort, hostErr := strconv.Atoi(parts[secondLastIdx])
		containerPort, containerErr := strconv.Atoi(parts[lastIdx])

		if hostErr == nil && containerErr == nil {
			// Everything before the last two parts is the IP
			ipPart := strings.Join(parts[:secondLastIdx], ":")
			return PortMapping{
				BindIP:        ipPart,
				HostPort:      hostPort,
				ContainerPort: containerPort,
				Protocol:      protocol,
			}
		}
	}

	switch len(parts) {
	case 1:
		// "8080" - single port
		port, _ := strconv.Atoi(parts[0])

		if isDocker {
			// Docker: container port only, host auto-assigned (Docker Compose behavior)
			return PortMapping{
				HostPort:      0, // 0 = auto-assign
				ContainerPort: port,
				Protocol:      protocol,
			}
		}

		// Non-Docker: same port for both host and app
		return PortMapping{
			HostPort:      port,
			ContainerPort: port,
			Protocol:      protocol,
		}

	case 2:
		// "3000:8080" - host:container
		host, _ := strconv.Atoi(parts[0])
		container, _ := strconv.Atoi(parts[1])
		return PortMapping{
			HostPort:      host,
			ContainerPort: container,
			Protocol:      protocol,
		}

	case 3:
		// "127.0.0.1:3000:8080" - ip:host:container
		host, _ := strconv.Atoi(parts[1])
		container, _ := strconv.Atoi(parts[2])
		return PortMapping{
			BindIP:        parts[0],
			HostPort:      host,
			ContainerPort: container,
			Protocol:      protocol,
		}
	}

	// Invalid format - return empty
	return PortMapping{Protocol: protocol}
}

// DetectPort attempts to detect the port for a service using multiple strategies.
//
// Port Detection Priority (highest to lowest):
//  1. Explicit ports in azure.yaml (service.Ports field) - MANDATORY, never changed
//  2. Framework-specific config files (package.json, launchSettings.json, etc.)
//  3. Environment variables (SERVICE_NAME_PORT, PORT, HTTP_PORT, etc.)
//  4. Framework default ports (Next.js=3000, Django=8000, Spring Boot=8080, etc.)
//  5. Dynamic port assignment (finds first available port starting from 3000)
//
// Parameters:
//   - serviceName: Name of the service (used for service-specific env vars)
//   - service: Service configuration from azure.yaml
//   - projectDir: Absolute path to the service's project directory
//   - framework: Detected framework name (e.g., "Next.js", "Django")
//   - usedPorts: Map of ports already assigned to other services
//
// Returns:
//   - port: The detected port number
//   - isExplicit: True if port came from azure.yaml (priority 1), false otherwise.
//     Explicit ports are mandatory and trigger user prompts if unavailable.
//   - error: Non-nil if port detection completely failed
//
// Examples:
//
//	// Service with explicit port in azure.yaml
//	service := Service{Ports: ["3000"]}
//	port, isExplicit, _ := DetectPort("web", service, "/app/web", "Next.js", nil)
//	// Result: port=3000, isExplicit=true
//
//	// Service with no explicit port, uses framework default
//	service := Service{}
//	port, isExplicit, _ := DetectPort("api", service, "/app/api", "Django", nil)
//	// Result: port=8000 (Django default), isExplicit=false
//
//	// Service with framework default already in use
//	usedPorts := map[int]bool{8000: true}
//	port, isExplicit, _ := DetectPort("api", service, "/app/api", "Django", usedPorts)
//	// Result: port=3000 (next available), isExplicit=false
func DetectPort(serviceName string, service Service, projectDir string, framework string, usedPorts map[int]bool) (int, bool, error) {
	// Priority 1: Explicit ports in azure.yaml (MANDATORY - never change these)
	if hostPort, _, isExplicit := service.GetPrimaryPort(); isExplicit {
		return hostPort, true, nil // isExplicit = true
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
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
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
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	if closeErr := listener.Close(); closeErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to close listener: %v\n", closeErr)
	}
	return true
}
