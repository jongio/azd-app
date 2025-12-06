package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"

	"github.com/spf13/cobra"
)

var (
	infoAll bool
)

// NewInfoCommand creates the info command.
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "info",
		Short:        "Show information about running services",
		Long:         `Displays comprehensive information about all running services including URLs, status, health, and metadata`,
		SilenceUsage: true,
		RunE:         runInfo,
	}

	cmd.Flags().BoolVar(&infoAll, "all", false, "Show services from all projects on this machine")

	return cmd
}

// runInfo executes the info command.
func runInfo(cmd *cobra.Command, args []string) error {
	output.CommandHeader("info", "Show information about services")
	// Get current working directory (may be set by --cwd flag)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	ctx := context.Background()

	// Try to get services from dashboard API first (live state)
	var allServices []*serviceinfo.ServiceInfo
	dashboardClient, err := dashboard.NewClient(ctx, cwd)
	if err == nil {
		// Dashboard is running, get live state from it
		allServices, err = dashboardClient.GetServices(ctx)
		if err != nil && !output.IsJSON() {
			output.Warning("Failed to get services from dashboard: %v", err)
			// Fall back to azure.yaml only
			allServices, err = serviceinfo.GetServiceInfo(cwd)
			if err != nil && !output.IsJSON() {
				output.Warning("Failed to get service info: %v", err)
			}
		}
	} else {
		// Dashboard not running - get service definitions from azure.yaml only
		// Note: Runtime state (running, ports, PIDs) will not be available
		allServices, err = serviceinfo.GetServiceInfo(cwd)
		if err != nil && !output.IsJSON() {
			output.Warning("Failed to get service info: %v", err)
		}
	}

	// Get Azure environment values for environment variable display
	azureEnv := getAzureEnvironmentValues(ctx)

	// For JSON output
	if output.IsJSON() {
		return printInfoJSON(cwd, allServices, azureEnv)
	}

	// Default output
	printInfoDefault(cwd, allServices, azureEnv)
	return nil
}

// printInfoJSON outputs service information in JSON format.
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

		statusIcon := getInfoStatusIcon(status, health)
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
// Valid statuses: "running", "starting", "error", "stopped", "not-running", "unknown"
func formatStatus(status string) string {
	switch status {
	case "running":
		return colorGreen + status + colorReset
	case "starting":
		return colorYellow + status + colorReset
	case "error":
		return colorRed + status + colorReset
	case "stopped", "not-running":
		return colorGray + status + colorReset
	case "unknown":
		return colorYellow + status + colorReset
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

	// Show relative time for recent events (within 24 hours)
	if duration < 24*time.Hour {
		return fmt.Sprintf("%s ago", formatInfoDuration(duration))
	}

	// Show absolute time for older events
	return t.Format("2006-01-02 15:04:05")
}

// formatDuration formats a duration in a human-readable way.
func formatInfoDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// getInfoStatusIcon returns a colored icon based on status and health.
// Valid statuses: "running", "starting", "error", "stopped", "not-running", "unknown"
func getInfoStatusIcon(status, health string) string {
	// Running and healthy - green check
	if status == "running" && health == "healthy" {
		return colorGreen + "✓" + colorReset
	}
	// Running but unhealthy - red X
	if status == "running" && health == "unhealthy" {
		return colorRed + "✗" + colorReset
	}
	// Starting - yellow circle
	if status == "starting" {
		return colorYellow + "○" + colorReset
	}
	// Error status - red X
	if status == "error" {
		return colorRed + "✗" + colorReset
	}
	// Stopped or not-running - gray dot
	if status == "stopped" || status == "not-running" {
		return colorGray + "●" + colorReset
	}
	// Unknown or any other status - yellow question mark
	return colorYellow + "?" + colorReset
}

// ANSI color constants
const (
	colorGreen  = "\033[92m"
	colorYellow = "\033[93m"
	colorRed    = "\033[91m"
	colorGray   = "\033[90m"
	colorReset  = "\033[0m"
)

// getAzureEnvironmentValues gets environment values from azd env get-values with timeout.
// Returns all environment variables defined in the azd environment.
// Accepts a context to support cancellation and timeout.
func getAzureEnvironmentValues(ctx context.Context) map[string]string {
	allEnvVars := make(map[string]string)

	// Set a reasonable timeout for the command (5 seconds)
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create command with context for timeout support
	cmd := exec.CommandContext(cmdCtx, "azd", "env", "get-values", "--output", "json")

	cmdOutput, err := cmd.Output()
	if err != nil {
		// Log error but don't fail - environment values are optional
		// This can happen if azd is not installed, not logged in, or no environment is active
		if !output.IsJSON() {
			// Only log in non-JSON mode to avoid polluting JSON output
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				// Command failed with stderr
				output.Warning("Failed to get Azure environment values: %s", string(exitErr.Stderr))
			} else {
				// Command not found or other error
				output.Warning("Failed to get Azure environment values: %v", err)
			}
		}
		return allEnvVars
	}

	var envVars map[string]string
	if err := json.Unmarshal(cmdOutput, &envVars); err != nil {
		if !output.IsJSON() {
			output.Warning("Failed to parse Azure environment values: %v", err)
		}
		return allEnvVars
	}

	// Add all environment variables from azd
	for key, value := range envVars {
		allEnvVars[key] = value
	}

	return allEnvVars
}
