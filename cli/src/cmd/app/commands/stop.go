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
	"github.com/jongio/azd-core/cliout"

	"github.com/spf13/cobra"
)

// NewStopCommand creates the stop command.
func NewStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running services and tear down the app",
		Long: `Stop running services, execute lifecycle hooks, and tear down the app.

Sends a shutdown signal to the running 'azd app run' process. This triggers
graceful shutdown including prestop/poststop hooks, port release, and
process cleanup — identical to pressing Ctrl+C in the run terminal.

Lifecycle hooks:
  prestop  - runs before services are stopped (e.g., drain connections)
  poststop - runs after all services are stopped (e.g., cleanup temp files)

Examples:
  # Stop the running app (from any terminal in the project)
  azd app stop`,
		SilenceUsage: true,
		RunE:         runStop,
	}

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	cliout.CommandHeader("stop", "Stop running services")

	// Find project root
	projectDir, err := findProjectDir()
	if err != nil {
		return err
	}

	// Discover the running dashboard port
	dashboardPort, err := discoverDashboardPort(projectDir)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://localhost:%d", dashboardPort)

	// Get session token for authentication
	token, err := fetchSessionToken(baseURL)
	if err != nil {
		return fmt.Errorf("failed to authenticate with running app: %w", err)
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

// fetchSessionToken retrieves the session token from the running dashboard.
func fetchSessionToken(baseURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/session-token", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not connect to running app — it may have already stopped")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response from session-token endpoint (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// sendShutdownRequest sends a POST to the dashboard's shutdown endpoint.
func sendShutdownRequest(baseURL, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/shutdown", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Session-Token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send shutdown signal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("shutdown request failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
