package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/registry"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"

	"github.com/spf13/cobra"
)

var (
	infoAll     bool
	infoProject string
)

// NewInfoCommand creates the info command.
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show information about running services",
		Long:  `Displays comprehensive information about all running services including URLs, status, health, and metadata`,
		RunE:  runInfo,
	}

	cmd.Flags().BoolVar(&infoAll, "all", false, "Show services from all projects on this machine")
	cmd.Flags().StringVar(&infoProject, "project", "", "Show services from a specific project directory")

	return cmd
}

// validateAndCleanServices checks if registered processes are still running and cleans up stale entries.
func validateAndCleanServices(reg *registry.ServiceRegistry) error {
	services := reg.ListAll()
	var servicesToRemove []string

	for _, svc := range services {
		// Primary check: is the port actually listening?
		// This is more reliable than PID checking due to PID reuse
		portListening := isPortReachable(svc.Port)
		pidExists := isProcessRunning(svc.PID)

		// If port is not listening, the service is effectively not running
		// even if a process with that PID exists (could be PID reuse)
		if !portListening {
			if pidExists {
				// PID exists but port isn't listening - likely PID reuse or crashed service
				// Mark as not running and remove
				servicesToRemove = append(servicesToRemove, svc.Name)
			} else {
				// Both PID and port are gone - definitely not running
				servicesToRemove = append(servicesToRemove, svc.Name)
			}
		} else {
			// Port is listening - service is running
			health := "healthy"
			if !pidExists {
				// Port is listening but PID changed - update health but keep running
				health = "unknown"
			}

			// Update the service status
			if svc.Health != health {
				if err := reg.UpdateStatus(svc.Name, "running", health); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to update service health: %v\n", err)
				}
			}
		}
	}

	// Remove stale services
	for _, serviceName := range servicesToRemove {
		if err := reg.Unregister(serviceName); err != nil {
			return fmt.Errorf("failed to unregister stale service %s: %w", serviceName, err)
		}
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is actually running.
// Works on both Windows and Unix systems.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, use tasklist command to check if process exists
		// #nosec G204 -- tasklist command with validated PID (integer), safe usage
		cmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		// If the process exists, tasklist will return a line with the PID
		return strings.Contains(string(output), strconv.Itoa(pid))
	} else {
		// On Unix systems, use os.FindProcess and send signal 0
		process, err := os.FindProcess(pid)
		if err != nil {
			return false
		}
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}
}

// isPortReachable checks if a port is reachable (simple health check).
func isPortReachable(port int) bool {
	// Check using netstat to see if port is listening
	if runtime.GOOS == "windows" {
		cmd := exec.Command("netstat", "-an")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		portStr := fmt.Sprintf(":%d ", port)
		// Check each line to ensure :PORT and LISTENING appear on the SAME line
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, portStr) && strings.Contains(line, "LISTENING") {
				return true
			}
		}
		return false
	} else {
		cmd := exec.Command("netstat", "-ln")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		portStr := fmt.Sprintf(":%d ", port)
		// Check each line to ensure the port appears with LISTEN state
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, portStr) && strings.Contains(line, "LISTEN") {
				return true
			}
		}
		return false
	}
}

// runInfo executes the info command.
func runInfo(cmd *cobra.Command, args []string) error {
	// Default: show services from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	projectDir := cwd
	if infoProject != "" {
		projectDir = infoProject
	}

	reg := registry.GetRegistry(projectDir)

	// Validate and clean up stale processes in real-time
	if err := validateAndCleanServices(reg); err != nil && !output.IsJSON() {
		output.Warning("Failed to validate service status: %v", err)
	}

	// Use shared serviceinfo package to get merged service data
	allServices, err := serviceinfo.GetServiceInfo(projectDir)
	if err != nil && !output.IsJSON() {
		output.Warning("Failed to get service info: %v", err)
	}

	// Get Azure environment values for environment variable display
	azureEnv := getAzureEnvironmentValues()

	// For JSON output
	if output.IsJSON() {
		return printInfoJSON(projectDir, allServices, azureEnv)
	}

	// Default output
	printInfoDefault(projectDir, allServices, azureEnv)
	return nil
} // printInfoJSON outputs service information in JSON format.
func printInfoJSON(projectDir string, services []*serviceinfo.ServiceInfo, azureEnv map[string]string) error {
	// Use serviceinfo.ServiceInfo directly - same schema as /api/services
	outputServices := make([]serviceinfo.ServiceInfo, 0, len(services))
	for _, svc := range services {
		// Add Azure-related environment variables if Azure info exists
		if svc.Azure != nil && azureEnv != nil {
			svc.EnvironmentVars = make(map[string]string)

			// Add the environment variables that were used to build the Azure info
			serviceName := strings.ToUpper(svc.Name)

			for envKey, envValue := range azureEnv {
				envKeyUpper := strings.ToUpper(envKey)

				// Include environment variables related to this service
				if strings.HasPrefix(envKeyUpper, serviceName+"_") ||
					strings.HasPrefix(envKeyUpper, "SERVICE_"+serviceName+"_") {
					svc.EnvironmentVars[envKey] = envValue
				}
			}
		}

		outputServices = append(outputServices, *svc) // Dereference pointer
	}

	return output.PrintJSON(map[string]interface{}{
		"project":  projectDir,
		"services": outputServices,
	})
}

// printInfoDefault outputs service information in default format.
func printInfoDefault(projectDir string, services []*serviceinfo.ServiceInfo, azureEnv map[string]string) {
	// Show project directory header
	output.Section("📦", fmt.Sprintf("Project: %s", projectDir))

	if len(services) == 0 {
		output.Info("No services defined in azure.yaml")
		output.Item("Run 'azd app reqs --generate' to create azure.yaml with service definitions")
		return
	}

	// Print services
	for i, svc := range services {
		if i > 0 {
			output.Divider()
		}

		// Get status and health from Local (with defaults if Local is nil)
		status := "unknown"
		health := "unknown"
		if svc.Local != nil {
			status = svc.Local.Status
			health = svc.Local.Health
		}

		statusIcon := getStatusIcon(status, health)
		output.Newline()
		output.Info("  %s %s", statusIcon, svc.Name)

		// Local development info
		if svc.Local != nil {
			if svc.Local.URL != "" {
				output.Label("  Local URL", svc.Local.URL)
			} else if svc.Local.Port > 0 {
				output.Label("  Local URL", fmt.Sprintf("http://localhost:%d (not running)", svc.Local.Port))
			}
		}

		// Azure URL and info
		if svc.Azure != nil {
			if svc.Azure.URL != "" {
				output.Label("  Azure URL", svc.Azure.URL)
			}
			if svc.Azure.ResourceName != "" {
				output.Label("  Azure Resource", svc.Azure.ResourceName)
			}
			if svc.Azure.ImageName != "" {
				output.Label("  Docker Image", svc.Azure.ImageName)
			}
		}

		// Service definition info
		if svc.Language != "" {
			output.Label("  Language", svc.Language)
		}
		if svc.Framework != "" {
			output.Label("  Framework", svc.Framework)
		}
		if svc.Project != "" {
			output.Label("  Project", svc.Project)
		}

		// Runtime info (only if service is running)
		if svc.Local != nil && svc.Local.Status == "running" {
			if svc.Local.Port > 0 {
				output.Label("  Port", fmt.Sprintf("%d", svc.Local.Port))
			}
			if svc.Local.PID > 0 {
				output.Label("  PID", fmt.Sprintf("%d", svc.Local.PID))
			}
			if svc.Local.StartTime != nil {
				output.Label("  Started", formatTime(*svc.Local.StartTime))
			}
			if svc.Local.LastChecked != nil {
				output.Label("  Checked", formatTime(*svc.Local.LastChecked))
			}
		}

		// Status and health (from Local)
		if svc.Local != nil {
			output.Label("  Status", formatStatus(svc.Local.Status))
			if svc.Local.Health != "unknown" {
				output.Label("  Health", formatHealth(svc.Local.Health))
			}
		}

		// Environment variables for this service (grouped by prefix)
		envVars := getServiceEnvironmentVars(svc.Name, azureEnv)
		if len(envVars) > 0 {
			output.Newline()
			output.Info("  Environment Variables:")
			for key, value := range envVars {
				output.Item("  %s = %s", key, value)
			}
		}
	}
	output.Newline()
}

// getAzureEndpoints extracts Azure endpoint URLs from environment variables.
func getAzureEndpoints() map[string]string {
	endpoints := make(map[string]string)

	for _, env := range os.Environ() {
		// Look for SERVICE_{name}_ENDPOINT_URL or SERVICE_{name}_URL
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		if strings.HasPrefix(key, "SERVICE_") && (strings.HasSuffix(key, "_ENDPOINT_URL") || strings.HasSuffix(key, "_URL")) {
			// Extract service name
			serviceName := strings.TrimPrefix(key, "SERVICE_")
			serviceName = strings.TrimSuffix(serviceName, "_ENDPOINT_URL")
			serviceName = strings.TrimSuffix(serviceName, "_URL")
			serviceName = strings.ToLower(serviceName)

			endpoints[serviceName] = value
		}
	}

	return endpoints
}

// getServiceEnvVars returns all SERVICE_{name}_ environment variables for a service.
func getServiceEnvVars(serviceName string) map[string]string {
	envVars := make(map[string]string)
	prefix := "SERVICE_" + strings.ToUpper(serviceName) + "_"

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		if strings.HasPrefix(key, prefix) {
			envVars[key] = value
		}
	}

	return envVars
}

// getServiceEnvironmentVars returns environment variables for a specific service,
// filtering and organizing them by relevant prefixes.
func getServiceEnvironmentVars(serviceName string, azureEnv map[string]string) map[string]string {
	envVars := make(map[string]string)
	serviceNameUpper := strings.ToUpper(serviceName)

	// Patterns to match (in priority order):
	// 1. SERVICE_{SERVICENAME}_* (highest priority - service-specific)
	// 2. AZURE_{SERVICENAME}_* (Azure-specific for this service)

	for key, value := range azureEnv {
		keyUpper := strings.ToUpper(key)

		// Match SERVICE_{SERVICENAME}_*
		if strings.HasPrefix(keyUpper, "SERVICE_"+serviceNameUpper+"_") {
			envVars[key] = value
			continue
		}

		// Match AZURE_{SERVICENAME}_*
		if strings.HasPrefix(keyUpper, "AZURE_"+serviceNameUpper+"_") {
			envVars[key] = value
			continue
		}
	}

	return envVars
}

// formatStatus returns a colored status string.
func formatStatus(status string) string {
	switch status {
	case "ready":
		return colorGreen + status + colorReset
	case "starting":
		return colorYellow + status + colorReset
	case "error":
		return colorRed + status + colorReset
	case "stopped":
		return colorGray + status + colorReset
	default:
		return status
	}
}

// formatHealth returns a colored health string.
func formatHealth(health string) string {
	switch health {
	case "healthy":
		return colorGreen + health + colorReset
	case "unhealthy":
		return colorRed + health + colorReset
	case "unknown":
		return colorYellow + health + colorReset
	default:
		return health
	}
}

// formatTime formats a time.Time for display.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return colorGray + "N/A" + colorReset
	}

	now := time.Now()
	duration := now.Sub(t)

	// Show relative time for recent events
	if duration < time.Minute {
		return fmt.Sprintf("%s ago", formatDuration(duration))
	} else if duration < time.Hour {
		return fmt.Sprintf("%s ago", formatDuration(duration))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%s ago", formatDuration(duration))
	}

	// Show absolute time for older events
	return t.Format("2006-01-02 15:04:05")
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// getStatusIcon returns a colored icon based on status and health.
func getStatusIcon(status, health string) string {
	if status == "ready" && health == "healthy" {
		return colorGreen + "✓" + colorReset
	}
	if status == "starting" {
		return colorYellow + "○" + colorReset
	}
	if status == "error" || health == "unhealthy" {
		return colorRed + "✗" + colorReset
	}
	if status == "stopped" {
		return colorGray + "●" + colorReset
	}
	return colorYellow + "?" + colorReset
}

// getCurrentDir returns the current working directory.
func getCurrentDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// ANSI color constants
const (
	colorGreen  = "\033[92m"
	colorYellow = "\033[93m"
	colorRed    = "\033[91m"
	colorGray   = "\033[90m"
	colorReset  = "\033[0m"
)

// getAzureEnvironmentValues gets environment values from azd env get-values or current environment.
func getAzureEnvironmentValues() map[string]string {
	allEnvVars := make(map[string]string)

	// First, try to get values from azd env get-values
	cmd := exec.Command("azd", "env", "get-values", "--output", "json")
	output, err := cmd.Output()
	if err == nil {
		var envVars map[string]string
		if err := json.Unmarshal(output, &envVars); err == nil {
			// Add all environment variables from azd
			for key, value := range envVars {
				allEnvVars[key] = value
			}
		}
	}

	// Fallback: Check current process environment for all variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]
		allEnvVars[key] = value
	}

	return allEnvVars
}
