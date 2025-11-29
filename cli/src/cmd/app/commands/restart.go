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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Determine which services to restart
	var servicesToRestart []string
	if restartAll {
		servicesToRestart = ctrl.GetAllServices()
		// Confirm destructive operation
		if len(servicesToRestart) > 0 && !restartYes && !output.IsJSON() {
			if !output.Confirm(fmt.Sprintf("Restart all %d service(s)?", len(servicesToRestart))) {
				output.Info("Operation cancelled")
				return nil
			}
		}
		if len(servicesToRestart) == 0 {
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
	} else {
		servicesToRestart = parseServiceList(restartService)
	}

	// Execute operation
	if len(servicesToRestart) == 1 {
		result := ctrl.RestartService(ctx, servicesToRestart[0])
		if output.IsJSON() {
			return output.PrintJSON(result)
		}
		printResult(result, "restart")
		if !result.Success {
			return fmt.Errorf("failed to restart service: %s", result.Error)
		}
		return nil
	}

	// Bulk operation
	result := ctrl.BulkRestart(ctx, servicesToRestart)
	if output.IsJSON() {
		return output.PrintJSON(result)
	}
	printBulkResult(result, "restart")
	if !result.Success {
		return fmt.Errorf("failed to restart %d service(s)", result.FailureCount)
	}
	return nil
}
