package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Determine which services to start
	var servicesToStart []string
	if startAll {
		servicesToStart = ctrl.GetStoppedServices()
		if len(servicesToStart) == 0 {
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
			output.Info("No stopped services to start (all services are already running)")
			if output.IsJSON() {
				return output.PrintJSON(BulkServiceControlResult{
					Success: true,
					Message: "No stopped services to start",
					Results: []ServiceControlResult{},
				})
			}
			return nil
		}
	} else {
		servicesToStart = parseServiceList(startService)
	}

	// Execute operation
	if len(servicesToStart) == 1 {
		result := ctrl.StartService(ctx, servicesToStart[0])
		if output.IsJSON() {
			return output.PrintJSON(result)
		}
		printResult(result, "start")
		if !result.Success {
			return fmt.Errorf("failed to start service: %s", result.Error)
		}
		return nil
	}

	// Bulk operation
	result := ctrl.BulkStart(ctx, servicesToStart)
	if output.IsJSON() {
		return output.PrintJSON(result)
	}
	printBulkResult(result, "start")
	if !result.Success {
		return fmt.Errorf("failed to start %d service(s)", result.FailureCount)
	}
	return nil
}

// parseServiceList splits a comma-separated service list and trims whitespace.
func parseServiceList(services string) []string {
	if services == "" {
		return nil
	}
	parts := strings.Split(services, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
