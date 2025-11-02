package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"app/src/internal/registry"

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
	services := reg.ListAll()

	if len(services) == 0 {
		fmt.Println("No services are running in this project")
		fmt.Println("Run 'azd app run' to start services")
		return nil
	}

	// Get Azure environment variables for endpoint mapping
	azdEnv := getAzureEndpoints()

	// Show project directory header
	fmt.Printf("\n%s╔═══════════════════════════════════════════════════════════════════════╗%s\n", colorBold, colorReset)
	fmt.Printf("%s║ Project: %-60s║%s\n", colorBold, projectDir, colorReset)
	fmt.Printf("%s╚═══════════════════════════════════════════════════════════════════════╝%s\n", colorBold, colorReset)

	// Print services
	for i, svc := range services {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 75))
		}

		statusIcon := getStatusIcon(svc.Status, svc.Health)
		fmt.Printf("\n  %s %s%s%s\n", statusIcon, colorBold, svc.Name, colorReset)
		fmt.Printf("  URL:       %s\n", svc.URL)

		// Show Azure endpoint if available
		if azureEndpoint, exists := azdEnv[svc.Name]; exists {
			fmt.Printf("  Azure URL: %s\n", azureEndpoint)
		}

		fmt.Printf("  Language:  %s\n", svc.Language)
		fmt.Printf("  Framework: %s\n", svc.Framework)
		fmt.Printf("  Port:      %d\n", svc.Port)
		fmt.Printf("  Status:    %s\n", formatStatus(svc.Status))
		fmt.Printf("  Health:    %s\n", formatHealth(svc.Health))
		fmt.Printf("  PID:       %d\n", svc.PID)
		fmt.Printf("  Started:   %s\n", formatTime(svc.StartTime))
		fmt.Printf("  Checked:   %s\n", formatTime(svc.LastChecked))

		// Show all SERVICE_{name}_ environment variables
		serviceEnvVars := getServiceEnvVars(svc.Name)
		if len(serviceEnvVars) > 0 {
			fmt.Printf("\n  %sEnvironment Variables:%s\n", colorBold, colorReset)
			for key, value := range serviceEnvVars {
				fmt.Printf("    %s=%s\n", key, value)
			}
		}

		if svc.Error != "" {
			fmt.Printf("\n  %sError:     %s%s\n", colorRed, svc.Error, colorReset)
		}
	}
	fmt.Println()

	return nil
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
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)
