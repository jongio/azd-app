package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/registry"
	"github.com/spf13/cobra"
)

type statusReport struct {
	Running      bool                    `json:"running"`
	PID          int                     `json:"pid,omitempty"`
	DashboardURL string                  `json:"dashboardUrl,omitempty"`
	StartTime    time.Time               `json:"startTime,omitempty"`
	Services     []runstate.ServiceState `json:"services,omitempty"`
}

// NewStatusCommand creates the status command.
func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show whether azd app run is active",
		SilenceUsage: true,
		RunE:         runStatus,
	}
}

func runStatus(_ *cobra.Command, _ []string) error {
	cliout.CommandHeader("status", "Show app status")

	projectDir, err := findProjectDir()
	if err != nil {
		return err
	}

	report, err := buildStatusReport(projectDir)
	if err != nil {
		return err
	}

	if cliout.IsJSON() {
		return cliout.PrintJSON(report)
	}

	printStatusReport(report)
	return nil
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

	return statusReport{
		Running:      true,
		PID:          st.PID,
		DashboardURL: dashboardURL,
		StartTime:    st.StartTime,
		Services:     statusServices(projectDir, st.Services),
	}, nil
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
