package commands

import (
	"fmt"

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
	ctx, _, cleanup := setupContextWithSignalHandling()
	defer cleanup()

	// Determine which services to stop
	var servicesToStop []string
	if stopAll {
		servicesToStop = ctrl.GetRunningServices()
		if len(servicesToStop) > 0 && !confirmBulkOperation(len(servicesToStop), "stop", stopYes) {
			output.Info("Operation cancelled")
			return nil
		}
		if len(servicesToStop) == 0 {
			if handleNoServicesCase(ctrl, "running", "stop") {
				return nil
			}
		}
	} else {
		servicesToStop, err = parseServiceList(stopService)
		if err != nil {
			return err
		}
	}

	return executeServiceOperation(ctx, servicesToStop, ctrl.StopService, ctrl.BulkStop, "stop")
}
