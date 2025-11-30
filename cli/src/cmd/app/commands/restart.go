package commands

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/output"

	"github.com/spf13/cobra"
)

var (
	restartService string
	restartAll     bool
	restartYes     bool
)

// NewRestartCommand creates the restart command.
func NewRestartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart services",
		Long: `Restart one or more services.

This command stops and then starts services. It works on both running and
stopped services. Use --service to restart a specific service, or --all
to restart all services.

Services are stopped gracefully before being restarted. If a service
doesn't respond to graceful shutdown, it will be forcefully terminated.

Examples:
  # Restart a specific service
  azd app restart --service api

  # Restart multiple services
  azd app restart --service "api,web,worker"

  # Restart all services
  azd app restart --all

  # JSON output
  azd app restart --service api --output json`,
		SilenceUsage: true,
		RunE:         runRestart,
	}

	cmd.Flags().StringVarP(&restartService, "service", "s", "", "Service name(s) to restart (comma-separated)")
	cmd.Flags().BoolVar(&restartAll, "all", false, "Restart all services")
	cmd.Flags().BoolVarP(&restartYes, "yes", "y", false, "Skip confirmation prompt for --all")

	return cmd
}

func runRestart(cmd *cobra.Command, args []string) error {
	output.CommandHeader("restart", "Restart services")

	// Validate flags
	if restartService == "" && !restartAll {
		return fmt.Errorf("specify --service <name> or --all to restart services")
	}

	// Create controller
	ctrl, err := NewServiceController("")
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Set up context with signal handling
	ctx, _, cleanup := setupContextWithSignalHandling()
	defer cleanup()

	// Determine which services to restart
	var servicesToRestart []string
	if restartAll {
		servicesToRestart = ctrl.GetAllServices()
		if len(servicesToRestart) == 0 {
			printNoServicesRegistered()
			if output.IsJSON() {
				return output.PrintJSON(noServicesRegisteredResult())
			}
			return nil
		}
		if !confirmBulkOperation(len(servicesToRestart), "restart", restartYes) {
			output.Info("Operation cancelled")
			return nil
		}
	} else {
		servicesToRestart, err = parseServiceList(restartService)
		if err != nil {
			return err
		}
	}

	return executeServiceOperation(ctx, servicesToRestart, ctrl.RestartService, ctrl.BulkRestart, "restart")
}
