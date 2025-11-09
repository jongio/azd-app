package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/healthcheck"

	"github.com/spf13/cobra"
)

const (
	// minHealthInterval is the minimum allowed interval between health checks
	minHealthInterval = 1 * time.Second
	
	// minHealthTimeout is the minimum allowed timeout for health checks
	minHealthTimeout = 1 * time.Second
	
	// maxHealthTimeout is the maximum allowed timeout for health checks
	maxHealthTimeout = 60 * time.Second
	
	// defaultHealthInterval is the default interval for streaming mode
	defaultHealthInterval = 5 * time.Second
	
	// defaultHealthTimeout is the default timeout for health checks
	defaultHealthTimeout = 5 * time.Second
	
	// defaultHealthEndpoint is the default health check endpoint path
	defaultHealthEndpoint = "/health"
)

var (
	healthService  string
	healthStream   bool
	healthInterval time.Duration
	healthOutput   string
	healthEndpoint string
	healthTimeout  time.Duration
	healthAll      bool
	healthVerbose  bool
)

// NewHealthCommand creates the health command.
func NewHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Monitor health status of services",
		Long: `Check the health status of running services with support for point-in-time snapshots 
or real-time streaming. Automatically detects /health endpoints and falls back 
to port or process checks.`,
		RunE: runHealth,
	}

	cmd.Flags().StringVarP(&healthService, "service", "s", "", "Monitor specific service(s) only (comma-separated)")
	cmd.Flags().BoolVar(&healthStream, "stream", false, "Enable streaming mode for real-time updates")
	cmd.Flags().DurationVarP(&healthInterval, "interval", "i", defaultHealthInterval, "Interval between health checks in streaming mode")
	cmd.Flags().StringVarP(&healthOutput, "output", "o", "text", "Output format: 'text', 'json', 'table'")
	cmd.Flags().StringVar(&healthEndpoint, "endpoint", defaultHealthEndpoint, "Default health endpoint path to check")
	cmd.Flags().DurationVar(&healthTimeout, "timeout", defaultHealthTimeout, "Timeout for each health check")
	cmd.Flags().BoolVar(&healthAll, "all", false, "Show health for all projects on this machine")
	cmd.Flags().BoolVarP(&healthVerbose, "verbose", "v", false, "Show detailed health check information")

	return cmd
}

func runHealth(cmd *cobra.Command, args []string) error {
	// Validate flags
	if err := validateHealthFlags(); err != nil {
		return err
	}

	// Get current working directory for project context
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create health monitor
	monitor, err := healthcheck.NewHealthMonitor(healthcheck.MonitorConfig{
		ProjectDir:      projectDir,
		DefaultEndpoint: healthEndpoint,
		Timeout:         healthTimeout,
		Verbose:         healthVerbose,
	})
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}

	// Parse service filter
	serviceFilter := parseServiceFilter(healthService)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	setupSignalHandler(cancel)

	if healthStream {
		return runStreamingMode(ctx, monitor, serviceFilter)
	}

	return runStaticMode(ctx, monitor, serviceFilter)
}

// validateHealthFlags validates the health command flags
func validateHealthFlags() error {
	if healthInterval < minHealthInterval {
		return fmt.Errorf("interval must be at least %v", minHealthInterval)
	}
	if healthTimeout < minHealthTimeout || healthTimeout > maxHealthTimeout {
		return fmt.Errorf("timeout must be between %v and %v", minHealthTimeout, maxHealthTimeout)
	}
	if healthOutput != "text" && healthOutput != "json" && healthOutput != "table" {
		return fmt.Errorf("output must be 'text', 'json', or 'table'")
	}
	return nil
}

// parseServiceFilter parses the comma-separated service filter
func parseServiceFilter(serviceStr string) []string {
	if serviceStr == "" {
		return nil
	}
	
	services := strings.Split(serviceStr, ",")
	for i, s := range services {
		services[i] = strings.TrimSpace(s)
	}
	return services
}

// setupSignalHandler sets up signal handling for graceful shutdown
func setupSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()
}

func runStaticMode(ctx context.Context, monitor *healthcheck.HealthMonitor, serviceFilter []string) error {
	// Perform single health check
	report, err := monitor.Check(ctx, serviceFilter)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// Format and display output
	if err := displayHealthReport(report); err != nil {
		return err
	}

	// Return exit code based on health status
	if report.Summary.Unhealthy > 0 {
		return fmt.Errorf("") // Exit with code 1 but no message
	}

	return nil
}

func runStreamingMode(ctx context.Context, monitor *healthcheck.HealthMonitor, serviceFilter []string) error {
	// Check if output is to a TTY - using simple check
	isTTY := isatty()

	if isTTY {
		// Interactive mode - clear screen and show live updates
		fmt.Print("\033[2J") // Clear screen
		displayStreamHeader()
	}

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	checkCount := 0
	var prevReport *healthcheck.HealthReport

	// Perform initial check immediately
	if err := performStreamCheck(ctx, monitor, serviceFilter, &checkCount, &prevReport, isTTY); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown
			if isTTY {
				displayStreamFooter(checkCount)
			}
			return nil

		case <-ticker.C:
			if err := performStreamCheck(ctx, monitor, serviceFilter, &checkCount, &prevReport, isTTY); err != nil {
				if ctx.Err() != nil {
					return nil // Context cancelled, normal shutdown
				}
				return err
			}
		}
	}
}

func performStreamCheck(ctx context.Context, monitor *healthcheck.HealthMonitor, serviceFilter []string, checkCount *int, prevReport **healthcheck.HealthReport, isTTY bool) error {
	report, err := monitor.Check(ctx, serviceFilter)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	*checkCount++

	if isTTY {
		// Clear screen and redraw
		fmt.Print("\033[H") // Move cursor to home
		displayStreamHeader()
		displayStreamStatus(report, *checkCount)

		// Show changes if we have a previous report
		if *prevReport != nil {
			displayStreamChanges(*prevReport, report)
		}
	} else {
		// Non-TTY mode - output JSON lines
		data, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		fmt.Println(string(data))
	}

	*prevReport = report
	return nil
}

func displayHealthReport(report *healthcheck.HealthReport) error {
	switch healthOutput {
	case "json":
		return displayJSONReport(report)
	case "table":
		return displayTableReport(report)
	default: // text
		return displayTextReport(report)
	}
}

func displayJSONReport(report *healthcheck.HealthReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func displayTextReport(report *healthcheck.HealthReport) error {
	fmt.Printf("Health Check (%s)\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println("=====================================")
	fmt.Println()

	for _, result := range report.Services {
		icon := getStatusIcon(result.Status)
		fmt.Printf("%s %-25s %-12s (%s)\n", icon, result.ServiceName, result.Status, result.CheckType)

		if result.Endpoint != "" {
			fmt.Printf("  Endpoint: %s\n", result.Endpoint)
		}
		if result.ResponseTime > 0 {
			fmt.Printf("  Response Time: %dms\n", result.ResponseTime.Milliseconds())
		}
		if result.StatusCode > 0 {
			fmt.Printf("  Status Code: %d\n", result.StatusCode)
		}
		if result.Port > 0 {
			fmt.Printf("  Port: %d\n", result.Port)
		}
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}

		// Show details if verbose or if there are details
		if healthVerbose && result.Details != nil {
			fmt.Println("  Details:")
			for k, v := range result.Details {
				fmt.Printf("    - %s: %v\n", k, v)
			}
		}

		if result.Uptime > 0 {
			fmt.Printf("  Uptime: %s\n", formatDuration(result.Uptime))
		}

		fmt.Println()
	}

	fmt.Println("─────────────────────────────────────────────")
	fmt.Println()
	fmt.Printf("Summary: %d healthy, %d degraded, %d unhealthy\n",
		report.Summary.Healthy, report.Summary.Degraded, report.Summary.Unhealthy)
	fmt.Printf("Overall Status: %s\n", strings.ToUpper(string(report.Summary.Overall)))

	return nil
}

func displayTableReport(report *healthcheck.HealthReport) error {
	// Header
	fmt.Println("┌──────────────┬───────────┬───────────┬──────────────────────────────────┬──────────┐")
	fmt.Println("│ SERVICE      │ STATUS    │ TYPE      │ ENDPOINT/PORT                    │ RESPONSE │")
	fmt.Println("├──────────────┼───────────┼───────────┼──────────────────────────────────┼──────────┤")

	// Services
	for _, result := range report.Services {
		endpoint := result.Endpoint
		if endpoint == "" && result.Port > 0 {
			endpoint = fmt.Sprintf("localhost:%d", result.Port)
		}
		if endpoint == "" {
			endpoint = "-"
		}

		response := "-"
		if result.ResponseTime > 0 {
			response = fmt.Sprintf("%dms", result.ResponseTime.Milliseconds())
		} else if result.Error != "" {
			response = "error"
		}

		fmt.Printf("│ %-12s │ %-9s │ %-9s │ %-32s │ %-8s │\n",
			truncate(result.ServiceName, 12),
			truncate(string(result.Status), 9),
			truncate(string(result.CheckType), 9),
			truncate(endpoint, 32),
			response,
		)
	}

	// Footer
	fmt.Println("└──────────────┴───────────┴───────────┴──────────────────────────────────┴──────────┘")

	return nil
}

func displayStreamHeader() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║              Real-Time Health Monitoring                  ║")
	fmt.Printf("║  Started: %-20s    Interval: %-4s       ║\n",
		time.Now().Format("2006-01-02 15:04:05"),
		healthInterval.String())
	fmt.Println("║  Press Ctrl+C to stop                                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func displayStreamStatus(report *healthcheck.HealthReport, checkCount int) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Last Update: %-20s Checks: %-4d         │\n",
		report.Timestamp.Format("15:04:05"),
		checkCount)
	fmt.Println("├─────────────────────────────────────────────────────────────┤")

	for _, result := range report.Services {
		icon := getStatusIcon(result.Status)
		responseTime := "-"
		if result.ResponseTime > 0 {
			responseTime = fmt.Sprintf("%3dms", result.ResponseTime.Milliseconds())
		}

		uptime := "-"
		if result.Uptime > 0 {
			uptime = formatDuration(result.Uptime)
		}

		fmt.Printf("│ %s %-12s %-10s %6s  Up: %-10s      │\n",
			icon,
			truncate(result.ServiceName, 12),
			truncate(string(result.Status), 10),
			responseTime,
			truncate(uptime, 10),
		)
	}

	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

func displayStreamChanges(prev, curr *healthcheck.HealthReport) {
	changes := detectChanges(prev, curr)
	if len(changes) > 0 {
		fmt.Println("\nRecent Changes:")
		for _, change := range changes {
			fmt.Printf("  %s - %s: %s → %s\n",
				change.Timestamp.Format("15:04:05"),
				change.ServiceName,
				change.OldStatus,
				change.NewStatus,
			)
		}
	}
}

func displayStreamFooter(checkCount int) {
	fmt.Println("\n🛑 Stopping health monitoring...")
	fmt.Printf("Total checks performed: %d\n", checkCount)
}

type statusChange struct {
	ServiceName string
	OldStatus   healthcheck.HealthStatus
	NewStatus   healthcheck.HealthStatus
	Timestamp   time.Time
}

func detectChanges(prev, curr *healthcheck.HealthReport) []statusChange {
	var changes []statusChange

	prevMap := make(map[string]healthcheck.HealthStatus)
	for _, svc := range prev.Services {
		prevMap[svc.ServiceName] = svc.Status
	}

	for _, svc := range curr.Services {
		if prevStatus, exists := prevMap[svc.ServiceName]; exists {
			if prevStatus != svc.Status {
				changes = append(changes, statusChange{
					ServiceName: svc.ServiceName,
					OldStatus:   prevStatus,
					NewStatus:   svc.Status,
					Timestamp:   curr.Timestamp,
				})
			}
		}
	}

	return changes
}

func getStatusIcon(status healthcheck.HealthStatus) string {
	switch status {
	case healthcheck.HealthStatusHealthy:
		return "✓"
	case healthcheck.HealthStatusDegraded:
		return "⚠"
	case healthcheck.HealthStatusUnhealthy:
		return "✗"
	case healthcheck.HealthStatusStarting:
		return "○"
	default:
		return "?"
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func isatty() bool {
	// Simple check for TTY
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
