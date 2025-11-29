package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	// Create controller
	ctrl, err := NewServiceController("")
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Determine which services to stop
	var servicesToStop []string
	if stopAll {
		servicesToStop = ctrl.GetRunningServices()
		// Confirm destructive operation
		if len(servicesToStop) > 0 && !stopYes && !output.IsJSON() {
			if !output.Confirm(fmt.Sprintf("Stop all %d running service(s)?", len(servicesToStop))) {
				output.Info("Operation cancelled")
				return nil
			}
		}
		if len(servicesToStop) == 0 {
			// Check if there are any services at all
			allServices := ctrl.GetAllServices()
			if len(allServices) == 0 {
				output.Info("No services are registered")
				output.Item("Run 'azd app run' first to start your development environment")
				if output.IsJSON() {
					return output.PrintJSON(BulkServiceControlResult{
						Success: false,
						Message: "No services registered. Run 'azd app run' first.",
						Results: []ServiceControlResult{},
					})
				}
				return nil
			}
			output.Info("No running services to stop (all services are already stopped)")
			if output.IsJSON() {
				return output.PrintJSON(BulkServiceControlResult{
					Success: true,
					Message: "No running services to stop",
					Results: []ServiceControlResult{},
				})
			}
			return nil
		}
	} else {
		servicesToStop = parseServiceList(stopService)
	}

	// Execute operation
	if len(servicesToStop) == 1 {
		result := ctrl.StopService(ctx, servicesToStop[0])
		if output.IsJSON() {
			return output.PrintJSON(result)
		}
		printResult(result, "stop")
		if !result.Success {
			return fmt.Errorf("failed to stop service: %s", result.Error)
		}
		return nil
	}

	// Bulk operation
	result := ctrl.BulkStop(ctx, servicesToStop)
	if output.IsJSON() {
		return output.PrintJSON(result)
	}
	printBulkResult(result, "stop")
	if !result.Success {
		return fmt.Errorf("failed to stop %d service(s)", result.FailureCount)
	}
	return nil
}
