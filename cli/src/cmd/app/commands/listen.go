package commands

import (
	"context"
	"log"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
	"github.com/jongio/azd-app/cli/src/internal/servicetarget"
	"github.com/spf13/cobra"
)

// NewListenCommand creates the listen command using the azdext SDK helper.
// It registers a local service target provider and post-provision event handler.
func NewListenCommand() *cobra.Command {
	return azdext.NewListenCommand(func(host *azdext.ExtensionHost) {
		client := host.Client()
		host.
			WithServiceTarget("local", func() azdext.ServiceTargetProvider {
				return servicetarget.NewLocalServiceTargetProvider(client)
			}).
			WithServiceEventHandler("postprovision", handlePostProvision, &azdext.ServiceEventOptions{})
	})
}

// handlePostProvision is called after azd provision completes for each service.
// This handler refreshes the dashboard to show the latest environment values.
func handlePostProvision(ctx context.Context, args *azdext.ServiceEventArgs) error {
	log.Printf("[azd-app] Post-provision event received for service: %s", args.Service.GetName())

	// The environment variables are now updated in the process by azd
	// We just need to trigger a refresh of the cached environment and broadcast to dashboards

	// Get the project directory
	projectDir := args.Project.GetPath()

	// Refresh the environment cache from the current process environment
	// (azd has already updated os.Environ() by the time this handler is called)
	serviceinfo.RefreshEnvironmentCache()
	log.Printf("[azd-app] Refreshed environment cache from updated process environment")

	// Broadcast updated service info to all connected dashboard clients
	srv := dashboard.GetServer(projectDir)
	if srv != nil {
		if err := srv.BroadcastServiceUpdate(projectDir); err != nil {
			log.Printf("[azd-app] Warning: Failed to broadcast service update: %v", err)
		} else {
			log.Printf("[azd-app] Successfully broadcasted environment update to dashboard clients")
		}
	}

	return nil
}
