package commands

import (
	"fmt"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/service"

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

	// Load azure.yaml for hooks (optional)
	azureYaml, err := loadAzureYamlForHooks()
	if err != nil {
		return fmt.Errorf("failed to load azure.yaml: %w", err)
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

	// Execute prestart hook if configured
	if azureYaml != nil {
		if err := executePrestartHook(azureYaml, servicesToStart); err != nil {
			return fmt.Errorf("prestart hook failed: %w", err)
		}
	}

	// Defer poststart hook execution
	defer func() {
		if azureYaml != nil {
			if postErr := executePoststartHook(azureYaml, servicesToStart); postErr != nil {
				output.Warning("Post-start hook failed: %v", postErr)
			}
		}
	}()

	return executeServiceOperation(ctx, servicesToStart, ctrl.StartService, ctrl.BulkStart, "start")
}

// executePrestartHook executes the prestart hook if configured.
func executePrestartHook(azureYaml *service.AzureYaml, services []string) error {
	envVars := buildServiceHookEnvVars("start", services)
	return executeCommandHook(azureYaml, azureYaml.Hooks.GetPrestart(), "prestart", envVars)
}

// executePoststartHook executes the poststart hook if configured.
func executePoststartHook(azureYaml *service.AzureYaml, services []string) error {
	envVars := buildServiceHookEnvVars("start", services)
	return executeCommandHook(azureYaml, azureYaml.Hooks.GetPoststart(), "poststart", envVars)
}

// buildServiceHookEnvVars builds service operation-specific environment variables for hooks.
// Shared by start, stop, and restart commands.
func buildServiceHookEnvVars(operation string, services []string) []string {
	return []string{
		buildKeyValueEnvVar("AZD_APP_OPERATION", operation),
		buildStringListEnvVar("AZD_APP_SERVICES", services),
	}
}
