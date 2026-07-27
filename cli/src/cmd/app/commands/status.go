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

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/registry"
	"github.com/spf13/cobra"
)

// minStatusWatchInterval is the smallest refresh interval accepted by --watch.
const minStatusWatchInterval = time.Second

var (
	statusWatch        bool
	statusInterval     time.Duration
	statusService      string
	statusDashboardURL bool
)

type statusReport struct {
	Running      bool                    `json:"running"`
	PID          int                     `json:"pid,omitempty"`
	DashboardURL string                  `json:"dashboardUrl,omitempty"`
	StartTime    time.Time               `json:"startTime,omitempty"`
	Uptime       string                  `json:"uptime,omitempty"`
	Services     []runstate.ServiceState `json:"services,omitempty"`
}

// NewStatusCommand creates the status command.
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether azd app run is active",
		Long: `Show whether azd app run is active, along with its PID, dashboard URL,
uptime, and running services.

Pass --watch to refresh the status on an interval until you press Ctrl+C, which
is handy for keeping an eye on services while they start. The global --json flag
prints a single snapshot and ignores --watch.

Examples:
  # One-time snapshot
  azd app status

  # Show one service
  azd app status --service api

  # Live view, refreshed every 2 seconds
  azd app status --watch

  # Live view, refreshed every 5 seconds
  azd app status --watch --interval 5s`,
		SilenceUsage: true,
		RunE:         runStatus,
	}
	cmd.Flags().BoolVar(&statusWatch, "watch", false, "Refresh the status on an interval until interrupted")
	cmd.Flags().DurationVar(&statusInterval, "interval", 2*time.Second, "Refresh interval for --watch (minimum 1s)")
	cmd.Flags().StringVarP(&statusService, "service", "s", "", "Filter to specific service(s), comma-separated")
	cmd.Flags().BoolVar(&statusDashboardURL, "dashboard-url", false, "Print only the dashboard URL for the active run")
	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	projectDir, err := findProjectDir()
	if err != nil {
		return err
	}
	requested, err := parseServiceList(statusService)
	if err != nil {
		return err
	}

	// --watch is an interactive text view. The global --json flag always wins and
	// prints a single snapshot instead.
	if statusWatch && !cliout.IsJSON() {
		if statusInterval < minStatusWatchInterval {
			return fmt.Errorf("--interval must be at least %s, got %s", minStatusWatchInterval, statusInterval)
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return watchStatus(ctx, os.Stdout, projectDir, statusInterval, requested)
	}

	report, err := buildStatusReport(projectDir)
	if err != nil {
		return err
	}
	if len(requested) > 0 {
		report, err = filterStatusReport(report, requested)
		if err != nil {
			return err
		}
	}

	if statusDashboardURL {
		url, err := dashboardURLFromStatusReport(report)
		if err != nil {
			return err
		}
		fmt.Println(url)
		return nil
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(report)
	}

	cliout.CommandHeader("status", "Show app status")
	printStatusReport(report)
	return nil
}

func dashboardURLFromStatusReport(report statusReport) (string, error) {
	if !report.Running {
		return "", fmt.Errorf("app is not running")
	}
	if report.DashboardURL == "" {
		return "", fmt.Errorf("dashboard URL is not available")
	}
	return report.DashboardURL, nil
}

// watchStatus renders the status to w immediately and then again on every tick
// until the context is canceled (for example by Ctrl+C).
func watchStatus(ctx context.Context, w io.Writer, projectDir string, interval time.Duration, requested []string) error {
	render := func() error {
		report, err := buildStatusReport(projectDir)
		if err != nil {
			return err
		}
		if len(requested) > 0 {
			report, err = filterStatusReport(report, requested)
			if err != nil {
				return err
			}
		}
		renderStatusReport(w, report, time.Now())
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

// renderStatusReport clears the screen and writes a single status frame with a
// refresh timestamp header. It is used by the --watch loop.
func renderStatusReport(w io.Writer, report statusReport, refreshedAt time.Time) {
	// Clear the screen so each refresh replaces the previous frame.
	_, _ = fmt.Fprint(w, "\033[H\033[2J")
	_, _ = fmt.Fprintf(w, "azd app status (refreshed %s, press Ctrl+C to stop)\n\n", refreshedAt.Format("15:04:05"))
	for _, line := range statusTextLines(report) {
		_, _ = fmt.Fprintln(w, line)
	}
}

func buildStatusReport(projectDir string) (statusReport, error) {
	st, exists, err := runstate.Read(projectDir)
	if err != nil {
		return statusReport{}, fmt.Errorf("read run state: %w", err)
	}

	if !exists {
		return statusReport{Running: false}, nil
	}

	if !runstate.IsRunning(st) {
		_ = runstate.Remove(projectDir)
		return statusReport{Running: false}, nil
	}

	dashboardURL := st.DashboardURL
	if port := dashboard.ReadPortFile(projectDir); port > 0 {
		dashboardURL = fmt.Sprintf("http://localhost:%d", port)
	}

	report := statusReport{
		Running:      true,
		PID:          st.PID,
		DashboardURL: dashboardURL,
		StartTime:    st.StartTime,
		Services:     statusServices(projectDir, st.Services),
	}
	if !st.StartTime.IsZero() {
		report.Uptime = formatUptime(time.Since(st.StartTime))
	}
	return report, nil
}

func filterStatusReport(report statusReport, requested []string) (statusReport, error) {
	if len(requested) == 0 || !report.Running {
		return report, nil
	}

	available := make([]string, 0, len(report.Services))
	byName := make(map[string]runstate.ServiceState, len(report.Services))
	for _, svc := range report.Services {
		available = append(available, svc.Name)
		byName[svc.Name] = svc
	}
	sort.Strings(available)

	filtered := make([]runstate.ServiceState, 0, len(requested))
	for _, name := range requested {
		resolved, err := resolveServiceName(name, available)
		if err != nil {
			return statusReport{}, err
		}
		filtered = append(filtered, byName[resolved])
	}

	report.Services = filtered
	return report, nil
}

func statusServices(projectDir string, fallback []runstate.ServiceState) []runstate.ServiceState {
	entries := registry.GetRegistry(projectDir).ListAll()
	services := make([]runstate.ServiceState, 0, len(entries))
	for _, entry := range entries {
		if entry.Status != constants.StatusRunning && entry.Status != constants.StatusReady {
			continue
		}

		serviceState := runstate.ServiceState{
			Name: entry.Name,
			URL:  entry.URL,
			Port: entry.Port,
		}
		if serviceState.URL == "" && serviceState.Port > 0 {
			serviceState.URL = fmt.Sprintf("http://localhost:%d", serviceState.Port)
		}
		services = append(services, serviceState)
	}

	if len(services) == 0 {
		services = append(services, fallback...)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

func printStatusReport(report statusReport) {
	lines := statusTextLines(report)
	if len(lines) == 0 {
		return
	}

	if report.Running {
		cliout.Success("%s", lines[0])
	} else {
		cliout.Info("%s", lines[0])
	}

	for _, line := range lines[1:] {
		cliout.Plain("%s", line)
	}
}

func statusTextLines(report statusReport) []string {
	if !report.Running {
		return []string{"App is not running"}
	}

	lines := []string{
		"App is running",
		fmt.Sprintf("PID: %d", report.PID),
	}

	if report.DashboardURL != "" {
		lines = append(lines, fmt.Sprintf("Dashboard: %s", report.DashboardURL))
	}
	if !report.StartTime.IsZero() {
		lines = append(lines, fmt.Sprintf("Started: %s", report.StartTime.Format(time.RFC3339)))
	}
	if report.Uptime != "" {
		lines = append(lines, fmt.Sprintf("Uptime: %s", report.Uptime))
	}

	if len(report.Services) == 0 {
		lines = append(lines, "Services: none")
		return lines
	}

	lines = append(lines, "Services:")
	for _, svc := range report.Services {
		lines = append(lines, fmt.Sprintf("  - %s", formatStatusService(svc)))
	}

	return lines
}

// formatUptime renders a duration as a compact human-readable string such as
// "45s", "3m12s", "1h04m", or "2d03h". Negative durations are clamped to zero.
// The largest two units are shown so the value stays short at any scale.
func formatUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int64(d / time.Second)
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd%02dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	case minutes > 0:
		return fmt.Sprintf("%dm%02ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

func formatStatusService(svc runstate.ServiceState) string {
	var endpoint string
	if svc.URL != "" {
		endpoint = svc.URL
	} else if svc.Port > 0 {
		endpoint = fmt.Sprintf("http://localhost:%d", svc.Port)
	}

	switch {
	case endpoint != "" && svc.Port > 0 && !strings.Contains(endpoint, fmt.Sprintf(":%d", svc.Port)):
		return fmt.Sprintf("%s: %s (port %d)", svc.Name, endpoint, svc.Port)
	case endpoint != "":
		return fmt.Sprintf("%s: %s", svc.Name, endpoint)
	case svc.Port > 0:
		return fmt.Sprintf("%s: port %d", svc.Name, svc.Port)
	default:
		return svc.Name
	}
}
