package commands

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
)

var (
	startService string
	startAll     bool
)

// NewStartCommand creates the start command.
func NewStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start stopped services",
		Long: `Start one or more stopped services that were previously running.

This command starts services that are currently in a stopped or error state.
Use --service to start a specific service, or --all to start all stopped services.

The start command requires a running dashboard instance. If no services are
running, use 'azd app run' to start your development environment first.

Examples:
  # Start a specific service
  azd app start --service api

  # Start multiple services
  azd app start --service "api,web,worker"

  # Start all stopped services
  azd app start --all

  # JSON output
  azd app start --service api --output json`,
		SilenceUsage: true,
		RunE:         runStart,
	}

	cmd.Flags().StringVarP(&startService, "service", "s", "", "Service name(s) to start (comma-separated)")
	cmd.Flags().BoolVar(&startAll, "all", false, "Start all stopped services")

	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	output.CommandHeader("start", "Start stopped services")

	// Validate flags
	if startService == "" && !startAll {
		return fmt.Errorf("specify --service <name> or --all to start services")
	}

	// Create controller
	ctrl, err := NewServiceController("")
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Set up context with signal handling
	ctx, _, cleanup := setupContextWithSignalHandling()
	defer cleanup()

	// Determine which services to start
	var servicesToStart []string
	if startAll {
		servicesToStart = ctrl.GetStoppedServices()
		if len(servicesToStart) == 0 {
			if handleNoServicesCase(ctrl, "stopped", "start") {
				return nil
			}
		}
	} else {
		servicesToStart, err = parseServiceList(startService)
		if err != nil {
			return err
		}
	}

	return executeServiceOperation(ctx, servicesToStart, ctrl.StartService, ctrl.BulkStart, "start")
}
