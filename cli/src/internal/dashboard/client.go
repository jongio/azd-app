package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// Client provides methods to query the dashboard API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new dashboard API client for the given project directory.
// Returns nil if the dashboard is not running for this project.
func NewClient(ctx context.Context, projectDir string) (*Client, error) {
	configClient, err := azdconfig.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create config client: %w", err)
	}
	defer configClient.Close()

	projectHash := azdconfig.ProjectHash(projectDir)
	port, err := configClient.GetDashboardPort(projectHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard port: %w", err)
	}

	if port == 0 {
		return nil, fmt.Errorf("dashboard not running for project")
	}

	return &Client{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		httpClient: &http.Client{
			Timeout: constants.DashboardAPITimeout,
		},
	}, nil
}

// NewClientWithPort creates a new dashboard API client for a known port.
func NewClientWithPort(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		httpClient: &http.Client{
			Timeout: constants.DashboardAPITimeout,
		},
	}
}

// Ping checks if the dashboard is running and responsive.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/ping", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dashboard returned status %d", resp.StatusCode)
	}

	return nil
}

// GetServices returns the list of services from the dashboard.
func (c *Client) GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/services", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dashboard returned status %d: %s", resp.StatusCode, string(body))
	}

	var services []*serviceinfo.ServiceInfo
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to decode services: %w", err)
	}

	return services, nil
}

// StopService requests the dashboard to stop a specific service.
func (c *Client) StopService(ctx context.Context, serviceName string) error {
	url := fmt.Sprintf("%s/api/services/%s/stop", c.baseURL, serviceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop service: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StopAllServices requests the dashboard to stop all services.
func (c *Client) StopAllServices(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/services/stop", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop services: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// IsDashboardRunning checks if a dashboard is running for the given project.
func IsDashboardRunning(ctx context.Context, projectDir string) bool {
	client, err := NewClient(ctx, projectDir)
	if err != nil {
		return false
	}

	return client.Ping(ctx) == nil
}

// GetDashboardPort returns the dashboard port for a project, or 0 if not running.
func GetDashboardPort(ctx context.Context, projectDir string) int {
	configClient, err := azdconfig.NewClient(ctx)
	if err != nil {
		return 0
	}
	defer configClient.Close()

	projectHash := azdconfig.ProjectHash(projectDir)
	port, err := configClient.GetDashboardPort(projectHash)
	if err != nil {
		return 0
	}

	return port
}
