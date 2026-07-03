package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/detector"
	"github.com/jongio/azd-app/cli/src/internal/runstate"
	"github.com/jongio/azd-core/cliout"

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
		Short: "Stop running services and tear down the app",
		Long: `Stop running services, execute lifecycle hooks, and tear down the app.

Without flags, stop sends a shutdown signal to the running 'azd app run'
process. This triggers graceful shutdown including prestop/poststop hooks,
port release, and process cleanup, identical to pressing Ctrl+C in the run
terminal.

Use --service to stop specific services while leaving the rest running, or
--all to stop every running service without tearing down the app. This
mirrors the start and restart commands.

Lifecycle hooks (whole-app teardown only):
  prestop  - runs before services are stopped (e.g., drain connections)
  poststop - runs after all services are stopped (e.g., cleanup temp files)

Examples:
  # Stop the running app (from any terminal in the project)
  azd app stop

  # Stop a specific service, leaving the rest running
  azd app stop --service api

  # Stop multiple services
  azd app stop --service "api,web,worker"

  # Stop all running services without tearing down the app
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

// runStop routes to service-scoped stop when --service or --all is set, and
// otherwise falls back to whole-app teardown for backward compatibility.
func runStop(cmd *cobra.Command, args []string) error {
	if stopService != "" || stopAll {
		return runStopServices()
	}
	return runStopApp()
}

// runStopServices stops specific services (or all running services) via the
// service controller without tearing down the app, mirroring start/restart.
func runStopServices() error {
	cliout.CommandHeader("stop", "Stop services")

	ctrl, err := NewServiceController("")
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	ctx, _, cleanup := setupContextWithSignalHandling()
	defer cleanup()

	var servicesToStop []string
	if stopAll {
		servicesToStop = ctrl.GetRunningServices()
		if len(servicesToStop) == 0 {
			handleNoServicesCase(ctrl, "running", "stop")
			return nil
		}
		if !confirmBulkOperation(len(servicesToStop), "stop", stopYes) {
			cliout.Info("Operation canceled")
			return nil
		}
	} else {
		servicesToStop, err = parseServiceList(stopService)
		if err != nil {
			return err
		}
	}

	return executeServiceOperation(ctx, servicesToStop, ctrl.StopService, ctrl.BulkStop, "stop")
}

// runStopApp tears down the whole app by signaling the running dashboard.
func runStopApp() error {
	cliout.CommandHeader("stop", "Stop running services")

	// Find project root
	projectDir, err := findProjectDir()
	if err != nil {
		return err
	}
	defer func() {
		_ = runstate.Remove(projectDir)
	}()

	// Discover the running dashboard port
	dashboardPort, err := discoverDashboardPort(projectDir)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://localhost:%d", dashboardPort)

	// Read the session token written by the running dashboard server.
	// The token is stored in a 0o600 file under ~/.azd/azd-app/{hash}/ and
	// removed when the server shuts down.
	token := dashboard.ReadTokenFile(projectDir)
	if token == "" {
		return fmt.Errorf("could not read session token — is 'azd app run' active in this project?")
	}

	// Send shutdown request
	if err := sendShutdownRequest(baseURL, token); err != nil {
		return err
	}

	cliout.Success("Shutdown signal sent — app is stopping")
	return nil
}

// findProjectDir locates the project root by finding azure.yaml.
func findProjectDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	azureYamlPath, err := detector.FindAzureYaml(cwd)
	if err != nil || azureYamlPath == "" {
		return "", fmt.Errorf("could not find azure.yaml — are you in a project directory?")
	}
	return filepath.Dir(azureYamlPath), nil
}

// discoverDashboardPort reads the dashboard port from the port file or azdconfig.
func discoverDashboardPort(projectDir string) (int, error) {
	// Primary: file-based discovery (works cross-process without gRPC dependency)
	if port := dashboard.ReadPortFile(projectDir); port > 0 {
		return port, nil
	}

	// Fallback: try azdconfig (may work if gRPC host is available)
	client, err := azdconfig.NewClient(context.Background())
	if err == nil {
		projectHash := azdconfig.ProjectHash(projectDir)
		if port, err := client.GetDashboardPort(projectHash); err == nil && port > 0 {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no running app found — is 'azd app run' active in this project?")
}

// sendShutdownRequest sends a POST to the dashboard's shutdown endpoint.
// It sets Sec-Fetch-Site: same-origin to satisfy the CWE-352 origin check on
// the server side. Go's net/http does not send this header automatically (it is
// injected by browser UAs); the CLI sets it explicitly so the server can apply
// the same allow-list logic to both browser and CLI callers.
func sendShutdownRequest(baseURL, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/shutdown", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Session-Token", token)
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send shutdown signal: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shutdown request failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
