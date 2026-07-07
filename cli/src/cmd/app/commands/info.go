package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

// minInfoWatchInterval is the smallest refresh interval accepted by --watch.
const minInfoWatchInterval = time.Second

var (
	infoAll      bool
	infoService  string
	infoWatch    bool
	infoInterval time.Duration
)

const (
	statusUnknown = "unknown"
	statusRunning = "running"
	statusStopped = "stopped"
	statusError   = "error"
)

// NewInfoCommand creates the info command.
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [service...]",
		Short: "Show information about running services",
		Long: `Displays comprehensive information about all running services including URLs, status, health, and metadata.

Pass one or more service names as arguments, or use --service (matching the
service-targeting flag used by health, logs, run, start, and restart), to show
only specific services.

Examples:
  # Show information about all services
  azd app info

  # Show information about specific services (positional)
  azd app info api web

  # Show information about a specific service via flag
  azd app info --service api

  # Show information about multiple services
  azd app info --service "api,web"

  # Live view, refreshed every 2 seconds (CPU and memory update each frame)
  azd app info --watch

  # Live view, refreshed every 5 seconds
  azd app info --watch --interval 5s`,
		SilenceUsage:      true,
		RunE:              runInfo,
		ValidArgsFunction: completeServiceArgs,
	}

	cmd.Flags().BoolVar(&infoAll, "all", false, "Show services from all projects on this machine")
	cmd.Flags().StringVarP(&infoService, "service", "s", "", "Show info for specific service(s) (comma-separated)")
	cmd.Flags().BoolVar(&infoWatch, "watch", false, "Refresh service info on an interval until interrupted")
	cmd.Flags().DurationVar(&infoInterval, "interval", 2*time.Second, "Refresh interval for --watch (minimum 1s)")

	return cmd
}

// runInfo executes the info command.
func runInfo(cmd *cobra.Command, args []string) error {
	// Get current working directory (may be set by --cwd flag)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve the requested services from positional args and/or --service.
	requested := append([]string{}, args...)
	if infoService != "" {
		parsed, perr := parseServiceList(infoService)
		if perr != nil {
			return perr
		}
		requested = append(requested, parsed...)
	}

	// --watch is an interactive text view. The global --json flag always wins and
	// prints a single snapshot instead.
	if infoWatch && !cliout.IsJSON() {
		if infoInterval < minInfoWatchInterval {
			return fmt.Errorf("--interval must be at least %s, got %s", minInfoWatchInterval, infoInterval)
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return watchInfo(ctx, os.Stdout, cwd, requested, infoInterval)
	}

	cliout.CommandHeader("info", "Show information about services")

	ctx := context.Background()

	// Try to get services from dashboard API first (live state)
	var allServices []*serviceinfo.ServiceInfo
	dashboardClient, err := dashboard.NewClient(ctx, cwd)
	if err == nil {
		// Dashboard is running, get live state from it
		allServices, err = dashboardClient.GetServices(ctx)
		if err != nil && !cliout.IsJSON() {
			cliout.Warning("Failed to get services from dashboard: %v", err)
			// Fall back to azure.yaml only
			allServices, err = serviceinfo.GetServiceInfo(cwd)
			if err != nil && !cliout.IsJSON() {
				cliout.Warning("Failed to get service info: %v", err)
			}
		}
	} else {
		// Dashboard not running - get service definitions from azure.yaml only
		// Note: Runtime state (running, ports, PIDs) will not be available
		allServices, err = serviceinfo.GetServiceInfo(cwd)
		if err != nil && !cliout.IsJSON() {
			cliout.Warning("Failed to get service info: %v", err)
		}
	}

	// Get Azure environment values for environment variable display
	azureEnv := getAzureEnvironmentValues()

	// Filter to the requested services.
	if len(requested) > 0 {
		allServices, err = filterServicesByName(allServices, requested)
		if err != nil {
			return err
		}
	}

	// For JSON output
	if cliout.IsJSON() {
		return printInfoJSON(cwd, allServices, azureEnv)
	}

	// Default output
	printInfoDefault(cwd, allServices, azureEnv)
	return nil
}

// gatherInfoServices returns the best available service information, preferring
// live state from the dashboard and falling back to azure.yaml definitions.
// Unlike the snapshot path it does not print warnings, so the watch loop stays
// clean frame to frame.
func gatherInfoServices(ctx context.Context, cwd string) []*serviceinfo.ServiceInfo {
	if client, err := dashboard.NewClient(ctx, cwd); err == nil {
		if services, err := client.GetServices(ctx); err == nil {
			return services
		}
	}
	services, _ := serviceinfo.GetServiceInfo(cwd)
	return services
}

// watchInfo renders service info to w immediately and then again on every tick
// until the context is canceled (for example by Ctrl+C).
func watchInfo(ctx context.Context, w io.Writer, cwd string, requested []string, interval time.Duration) error {
	render := func() error {
		services := gatherInfoServices(ctx, cwd)
		if len(requested) > 0 {
			filtered, err := filterServicesByName(services, requested)
			if err != nil {
				return err
			}
			services = filtered
		}
		renderInfoFrame(w, cwd, services, time.Now())
		return nil
	}

	if err := render(); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_, _ = fmt.Fprintln(w, "\nStopped watching.")
			return nil
		case <-ticker.C:
			if err := render(); err != nil {
				return err
			}
		}
	}
}

// renderInfoFrame clears the screen and writes a single info frame with a
// refresh timestamp header. It is used by the --watch loop.
func renderInfoFrame(w io.Writer, projectDir string, services []*serviceinfo.ServiceInfo, refreshedAt time.Time) {
	// Clear the screen so each refresh replaces the previous frame.
	_, _ = fmt.Fprint(w, "\033[H\033[2J")
	_, _ = fmt.Fprintf(w, "azd app info (refreshed %s, press Ctrl+C to stop)\n", refreshedAt.Format("15:04:05"))
	_, _ = fmt.Fprintf(w, "Project: %s\n\n", projectDir)
	for _, line := range infoWatchLines(services) {
		_, _ = fmt.Fprintln(w, line)
	}
}

// infoWatchLines builds the plain-text body of a watch frame, one service per
// block, highlighting the live runtime values (status, port, PID, CPU, memory).
func infoWatchLines(services []*serviceinfo.ServiceInfo) []string {
	if len(services) == 0 {
		return []string{"No services defined in azure.yaml"}
	}

	lines := make([]string, 0, len(services)*2)
	for _, svc := range services {
		status := statusUnknown
		health := statusUnknown
		if svc.Local != nil {
			if svc.Local.Status != "" {
				status = svc.Local.Status
			}
			if svc.Local.Health != "" {
				health = svc.Local.Health
			}
		}
		lines = append(lines, fmt.Sprintf("%s  [%s/%s]", svc.Name, status, health))

		if svc.Local != nil && svc.Local.Status == statusRunning {
			detail := make([]string, 0, 4)
			if svc.Local.Port > 0 {
				detail = append(detail, fmt.Sprintf("port %d", svc.Local.Port))
			}
			if svc.Local.PID > 0 {
				detail = append(detail, fmt.Sprintf("pid %d", svc.Local.PID))
			}
			if svc.Local.CPUPercent > 0 {
				detail = append(detail, fmt.Sprintf("cpu %s", formatCPUPercent(svc.Local.CPUPercent)))
			}
			if svc.Local.MemoryBytes > 0 {
				detail = append(detail, fmt.Sprintf("mem %s", formatMemoryBytes(svc.Local.MemoryBytes)))
			}
			if len(detail) > 0 {
				lines = append(lines, "  "+strings.Join(detail, "  "))
			}
			if svc.Local.URL != "" {
				lines = append(lines, "  "+svc.Local.URL)
			}
		}
	}
	return lines
}

// filterServicesByName returns only the services whose names are in the
// requested list. Each requested name is validated against the available
// services, returning a helpful "did you mean" error when a name is unknown.
func filterServicesByName(services []*serviceinfo.ServiceInfo, requested []string) ([]*serviceinfo.ServiceInfo, error) {
	available := make([]string, 0, len(services))
	byName := make(map[string]*serviceinfo.ServiceInfo, len(services))
	for _, svc := range services {
		available = append(available, svc.Name)
		byName[svc.Name] = svc
	}
	sort.Strings(available)

	filtered := make([]*serviceinfo.ServiceInfo, 0, len(requested))
	for _, name := range requested {
		resolved, err := resolveServiceName(name, available)
		if err != nil {
			return nil, err
		}
		filtered = append(filtered, byName[resolved])
	}
	return filtered, nil
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
					svc.EnvironmentVars[envKey] = redactSecretValue(envKey, envValue)
				}
			}
		}

		outputServices = append(outputServices, *svc) // Dereference pointer
	}

	return cliout.PrintJSON(map[string]any{
		"project":  projectDir,
		"services": outputServices,
	})
}

// printInfoDefault outputs service information in default format.
func printInfoDefault(projectDir string, services []*serviceinfo.ServiceInfo, azureEnv map[string]string) {
	// Show project directory header
	cliout.Section("📦", fmt.Sprintf("Project: %s", projectDir))

	if len(services) == 0 {
		cliout.Info("No services defined in azure.yaml")
		cliout.Item("Run 'azd app reqs --generate' to create azure.yaml with service definitions")
		return
	}

	// Print services
	for i, svc := range services {
		if i > 0 {
			cliout.Divider()
		}

		// Get status and health from Local (with defaults if Local is nil)
		status := statusUnknown
		health := statusUnknown
		if svc.Local != nil {
			status = svc.Local.Status
			health = svc.Local.Health
		}

		statusIcon := getInfoStatusIcon(status, health)
		cliout.Newline()
		cliout.Info("  %s %s", statusIcon, svc.Name)

		// Local development info
		if svc.Local != nil {
			if svc.Local.URL != "" {
				cliout.Label("  Local URL", svc.Local.URL)
			} else if svc.Local.Port > 0 {
				cliout.Label("  Local URL", fmt.Sprintf("http://localhost:%d (not running)", svc.Local.Port))
			}

			// Show custom local URL if configured
			if svc.Local.CustomURL != "" {
				cliout.Label("  Custom Local URL", svc.Local.CustomURL)
			}
		}

		// Azure URL and info
		if svc.Azure != nil {
			// Show auto-discovered Azure URL
			if svc.Azure.URL != "" {
				cliout.Label("  Azure URL", svc.Azure.URL)
			}

			// Show custom Azure URL if configured
			if svc.Azure.CustomURL != "" {
				cliout.Label("  Custom Azure URL", svc.Azure.CustomURL)
			}

			// Show custom domain if configured
			if svc.Azure.CustomDomain != "" {
				label := "  Custom Domain"
				if svc.Azure.CustomDomainSource != "" {
					label = fmt.Sprintf("  Custom Domain (%s)", svc.Azure.CustomDomainSource)
				}
				cliout.Label(label, svc.Azure.CustomDomain)
			}

			if svc.Azure.ImageName != "" {
				cliout.Label("  Docker Image", svc.Azure.ImageName)
			}
		}

		// Service definition info
		if svc.Language != "" {
			cliout.Label("  Language", svc.Language)
		}
		if svc.Framework != "" {
			cliout.Label("  Framework", svc.Framework)
		}
		if svc.Project != "" {
			cliout.Label("  Project", svc.Project)
		}

		// Runtime info (only if service is running)
		if svc.Local != nil && svc.Local.Status == statusRunning {
			if svc.Local.Port > 0 {
				cliout.Label("  Port", fmt.Sprintf("%d", svc.Local.Port))
			}
			if svc.Local.PID > 0 {
				cliout.Label("  PID", fmt.Sprintf("%d", svc.Local.PID))
			}
			if svc.Local.StartTime != nil {
				cliout.Label("  Started", formatTime(*svc.Local.StartTime))
			}
			if svc.Local.LastChecked != nil {
				cliout.Label("  Checked", formatTime(*svc.Local.LastChecked))
			}
			if svc.Local.CPUPercent > 0 {
				cliout.Label("  CPU", formatCPUPercent(svc.Local.CPUPercent))
			}
			if svc.Local.MemoryBytes > 0 {
				cliout.Label("  Memory", formatMemoryBytes(svc.Local.MemoryBytes))
			}
		}

		// Status and health (from Local)
		if svc.Local != nil {
			cliout.Label("  Status", formatStatus(svc.Local.Status))
			if svc.Local.Health != statusUnknown {
				cliout.Label("  Health", formatHealth(svc.Local.Health))
			}
		}

		// Environment variables for this service (grouped by prefix)
		envVars := getServiceEnvironmentVars(svc.Name, azureEnv)
		if len(envVars) > 0 {
			cliout.Newline()
			cliout.Info("  Environment Variables:")
			for key, value := range envVars {
				cliout.Item("  %s = %s", key, redactSecretValue(key, value))
			}
		}
	}
	cliout.Newline()
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
	case statusRunning:
		return colorGreen + status + colorReset
	case "starting":
		return colorYellow + status + colorReset
	case statusError:
		return colorRed + status + colorReset
	case statusStopped, "not-running":
		return colorGray + status + colorReset
	case statusUnknown:
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
	case statusUnknown:
		return colorYellow + health + colorReset
	default:
		return health
	}
}

// formatCPUPercent formats CPU usage as a percentage of a single core.
func formatCPUPercent(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// formatMemoryBytes formats a byte count using binary units (KiB/MiB/GiB).
func formatMemoryBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
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
	if status == statusRunning && health == "healthy" {
		return colorGreen + "✓" + colorReset
	}
	// Running but unhealthy - red X
	if status == statusRunning && health == "unhealthy" {
		return colorRed + "✗" + colorReset
	}
	// Starting - yellow circle
	if status == "starting" {
		return colorYellow + "○" + colorReset
	}
	// Error status - red X
	if status == statusError {
		return colorRed + "✗" + colorReset
	}
	// Stopped or not-running - gray dot
	if status == statusStopped || status == "not-running" {
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

// getAzureEnvironmentValues gets environment values from the process.
// When running as an azd extension, all environment variables are injected by azd.
func getAzureEnvironmentValues() map[string]string {
	envVars := make(map[string]string)
	for _, line := range os.Environ() {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}
	return envVars
}
