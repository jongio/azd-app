package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
)

var (
	stopService string
	stopAll     bool
	stopYes     bool
)

// NewStopCommand creates the stop command.
func NewStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running services",
		Long: `Stop one or more running services gracefully.

This command stops services that are currently running.
Use --service to stop a specific service, or --all to stop all running services.

Services are stopped gracefully with a timeout. If a service doesn't respond
to graceful shutdown, it will be forcefully terminated.

Examples:
  # Stop a specific service
  azd app stop --service api

  # Stop multiple services
  azd app stop --service "api,web,worker"

  # Stop all running services
  azd app stop --all

  # JSON output
  azd app stop --service api --output json`,
		SilenceUsage: true,
		RunE:         runStop,
	}

	cmd.Flags().StringVarP(&stopService, "service", "s", "", "Service name(s) to stop (comma-separated)")
	cmd.Flags().BoolVar(&stopAll, "all", false, "Stop all running services")
	cmd.Flags().BoolVarP(&stopYes, "yes", "y", false, "Skip confirmation prompt for --all")

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	output.CommandHeader("stop", "Stop running services")

	// Validate flags
	if stopService == "" && !stopAll {
		return fmt.Errorf("specify --service <name> or --all to stop services")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	ctx := context.Background()

	// Try to get dashboard client
	dashboardClient, err := dashboard.NewClient(ctx, cwd)
	if err != nil {
		output.Info("No services are running (dashboard not active)")
		return nil
	}

	// Check if dashboard is actually responding
	if err := dashboardClient.Ping(ctx); err != nil {
		output.Info("No services are running (dashboard not responding)")
		return nil
	}

	// Stop services via dashboard API
	if stopAll {
		// Get service list first to show confirmation
		services, err := dashboardClient.GetServices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get services: %w", err)
		}

		var runningServices []string
		for _, svc := range services {
			if svc.Local != nil && (svc.Local.Status == "running" || svc.Local.Status == "ready") {
				runningServices = append(runningServices, svc.Name)
			}
		}

		if len(runningServices) == 0 {
			output.Info("No services are currently running")
			return nil
		}

		if !confirmBulkOperation(len(runningServices), "stop", stopYes) {
			output.Info("Operation cancelled")
			return nil
		}

		if err := dashboardClient.StopAllServices(ctx); err != nil {
			return fmt.Errorf("failed to stop services: %w", err)
		}

		output.Success("Stopped %d service(s)", len(runningServices))
	} else {
		// Stop specific service(s)
		serviceNames := strings.Split(stopService, ",")
		for i, name := range serviceNames {
			serviceNames[i] = strings.TrimSpace(name)
		}

		var stopped, failed int
		for _, name := range serviceNames {
			if err := dashboardClient.StopService(ctx, name); err != nil {
				output.Warning("Failed to stop '%s': %v", name, err)
				failed++
			} else {
				output.Success("Stopped '%s'", name)
				stopped++
			}
		}

		if failed > 0 {
			return fmt.Errorf("failed to stop %d service(s)", failed)
		}
	}

	return nil
}
