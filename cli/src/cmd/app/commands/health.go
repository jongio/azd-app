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
	healthService           string
	healthStream            bool
	healthInterval          time.Duration
	healthOutput            string
	healthEndpoint          string
	healthTimeout           time.Duration
	healthAll               bool
	healthVerbose           bool
	healthProfile           string
	healthLogLevel          string
	healthLogFormat         string
	healthEnableMetrics     bool
	healthMetricsPort       int
	healthCircuitBreaker    bool
	healthCircuitBreakCount int
	healthCircuitBreakTime  time.Duration
	healthRateLimit         int
	healthCacheTTL          time.Duration
	healthProfileSave       bool
)

// NewHealthCommand creates the health command.
func NewHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Monitor health status of services",
		Long: `Check the health status of running services with support for point-in-time snapshots 
or real-time streaming. Automatically detects /health endpoints and falls back 
to port or process checks.

Production Features:
  - Circuit breaker pattern to prevent cascading failures
  - Rate limiting per service to avoid overwhelming endpoints
  - Result caching to reduce redundant checks
  - Prometheus metrics exposition for observability
  - Structured logging (JSON, pretty, or text)
  - Environment-specific profiles (dev, prod, ci, staging)

Examples:
  # Quick check with defaults
  azd app health
  
  # Production mode with all features
  azd app health --profile production --metrics --circuit-breaker
  
  # Development mode with verbose logging
  azd app health --profile development --log-level debug --log-format pretty
  
  # Custom configuration
  azd app health --rate-limit 10 --cache-ttl 30s --timeout 10s
  
  # Stream with metrics
  azd app health --stream --interval 10s --metrics --metrics-port 9090`,
		SilenceUsage: true,
		RunE:         runHealth,
	}

	// Basic flags
	cmd.Flags().StringVarP(&healthService, "service", "s", "", "Monitor specific service(s) only (comma-separated)")
	cmd.Flags().BoolVar(&healthStream, "stream", false, "Enable streaming mode for real-time updates")
	cmd.Flags().DurationVarP(&healthInterval, "interval", "i", defaultHealthInterval, "Interval between health checks in streaming mode")
	cmd.Flags().StringVarP(&healthOutput, "output", "o", "text", "Output format: 'text', 'json', 'table'")
	cmd.Flags().StringVar(&healthEndpoint, "endpoint", defaultHealthEndpoint, "Default health endpoint path to check")
	cmd.Flags().DurationVar(&healthTimeout, "timeout", defaultHealthTimeout, "Timeout for each health check")
	cmd.Flags().BoolVar(&healthAll, "all", false, "Show health for all projects on this machine")
	cmd.Flags().BoolVarP(&healthVerbose, "verbose", "v", false, "Show detailed health check information")

	// Profile and logging flags
	cmd.Flags().StringVar(&healthProfile, "profile", "", "Health profile to use (development, production, ci, staging, or custom)")
	cmd.Flags().StringVar(&healthLogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	cmd.Flags().StringVar(&healthLogFormat, "log-format", "pretty", "Log format: json, pretty, text")
	cmd.Flags().BoolVar(&healthProfileSave, "save-profiles", false, "Save sample health profiles to .azd/health-profiles.yaml")

	// Metrics flags
	cmd.Flags().BoolVar(&healthEnableMetrics, "metrics", false, "Enable Prometheus metrics exposition")
	cmd.Flags().IntVar(&healthMetricsPort, "metrics-port", 9090, "Port for Prometheus metrics endpoint")

	// Circuit breaker flags
	cmd.Flags().BoolVar(&healthCircuitBreaker, "circuit-breaker", false, "Enable circuit breaker pattern")
	cmd.Flags().IntVar(&healthCircuitBreakCount, "circuit-break-count", 5, "Number of failures before opening circuit")
	cmd.Flags().DurationVar(&healthCircuitBreakTime, "circuit-break-timeout", 60*time.Second, "Circuit breaker timeout duration")

	// Rate limiting and caching flags
	cmd.Flags().IntVar(&healthRateLimit, "rate-limit", 0, "Max health checks per second per service (0 = unlimited)")
	cmd.Flags().DurationVar(&healthCacheTTL, "cache-ttl", 0, "Cache TTL for health results (0 = no caching)")

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

	// Handle profile saving
	if healthProfileSave {
		if err := healthcheck.SaveSampleProfiles(projectDir); err != nil {
			return fmt.Errorf("failed to save sample profiles: %w", err)
		}
		fmt.Println("Sample health profiles saved to .azd/health-profiles.yaml")
		fmt.Println("You can customize these profiles or create new ones.")
		return nil
	}

	// Load health profiles
	profiles, err := healthcheck.LoadHealthProfiles(projectDir)
	if err != nil && healthProfile != "" {
		// Only error if a specific profile was requested
		return fmt.Errorf("failed to load health profiles: %w", err)
	}

	// Start with default config
	config := healthcheck.MonitorConfig{
		ProjectDir:             projectDir,
		DefaultEndpoint:        healthEndpoint,
		Timeout:                healthTimeout,
		Verbose:                healthVerbose,
		LogLevel:               healthLogLevel,
		LogFormat:              healthLogFormat,
		EnableCircuitBreaker:   healthCircuitBreaker,
		CircuitBreakerFailures: healthCircuitBreakCount,
		CircuitBreakerTimeout:  healthCircuitBreakTime,
		RateLimit:              healthRateLimit,
		EnableMetrics:          healthEnableMetrics,
		MetricsPort:            healthMetricsPort,
		CacheTTL:               healthCacheTTL,
	}

	// Apply profile if specified
	if healthProfile != "" && profiles != nil {
		profile, err := profiles.GetProfile(healthProfile)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		// Apply profile settings (CLI flags take precedence)
		if !cmd.Flags().Changed("timeout") && profile.Timeout > 0 {
			config.Timeout = profile.Timeout
		}
		if !cmd.Flags().Changed("log-level") && profile.LogLevel != "" {
			config.LogLevel = profile.LogLevel
		}
		if !cmd.Flags().Changed("log-format") && profile.LogFormat != "" {
			config.LogFormat = profile.LogFormat
		}
		if !cmd.Flags().Changed("circuit-breaker") {
			config.EnableCircuitBreaker = profile.CircuitBreaker
		}
		if !cmd.Flags().Changed("circuit-break-count") && profile.CircuitBreakerFailures > 0 {
			config.CircuitBreakerFailures = profile.CircuitBreakerFailures
		}
		if !cmd.Flags().Changed("circuit-break-timeout") && profile.CircuitBreakerTimeout > 0 {
			config.CircuitBreakerTimeout = profile.CircuitBreakerTimeout
		}
		if !cmd.Flags().Changed("rate-limit") && profile.RateLimit > 0 {
			config.RateLimit = profile.RateLimit
		}
		if !cmd.Flags().Changed("metrics") {
			config.EnableMetrics = profile.Metrics
		}
		if !cmd.Flags().Changed("metrics-port") && profile.MetricsPort > 0 {
			config.MetricsPort = profile.MetricsPort
		}
		if !cmd.Flags().Changed("cache-ttl") && profile.CacheTTL > 0 {
			config.CacheTTL = profile.CacheTTL
		}

		fmt.Printf("Using health profile: %s\n", healthProfile)
	}

	// Create health monitor with enriched config
	monitor, err := healthcheck.NewHealthMonitor(config)
	if err != nil {
		return fmt.Errorf("failed to create health monitor: %w", err)
	}

	// Start metrics server if enabled
	if config.EnableMetrics {
		go func() {
			if err := healthcheck.ServeMetrics(config.MetricsPort); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Metrics server failed: %v\n", err)
			}
		}()
		fmt.Printf("Prometheus metrics available at http://localhost:%d/metrics\n", config.MetricsPort)
	}

	// Parse service filter
	serviceFilter := parseServiceFilter(healthService)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	setupSignalHandler(ctx, cancel)

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
	if healthStream && healthInterval <= healthTimeout {
		return fmt.Errorf("interval (%v) must be greater than timeout (%v) in streaming mode", healthInterval, healthTimeout)
	}
	if healthOutput != "text" && healthOutput != "json" && healthOutput != "table" {
		return fmt.Errorf("invalid output format: must be 'text', 'json', or 'table'")
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
// The goroutine will exit when either a signal is received or the context is cancelled
func setupSignalHandler(ctx context.Context, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			// Signal received - cancel context
			cancel()
		case <-ctx.Done():
			// Context cancelled (normal exit) - just clean up
		}
		// Clean up signal handling
		signal.Stop(sigChan)
		close(sigChan)
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
		return fmt.Errorf("%d service(s) unhealthy", report.Summary.Unhealthy)
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
