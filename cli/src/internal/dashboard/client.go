package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// Client provides methods to query the dashboard API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new dashboard API client for the given project directory.
// Returns nil if the dashboard is not running for this project.
// It first tries azdconfig (gRPC), then falls back to reading ~/.azd/config.json directly.
func NewClient(ctx context.Context, projectDir string) (*Client, error) {
	projectHash := azdconfig.ProjectHash(projectDir)

	// Try azdconfig first (works when running as azd extension)
	configClient, err := azdconfig.NewClient(ctx)
	if err == nil {
		defer configClient.Close()
		port, err := configClient.GetDashboardPort(projectHash)
		if err == nil && port > 0 {
			return &Client{
				baseURL: fmt.Sprintf("http://localhost:%d", port),
				httpClient: &http.Client{
					Timeout: constants.DashboardAPITimeout,
				},
			}, nil
		}
	}

	// Fallback: read directly from ~/.azd/config.json (works when running standalone)
	port, err := readDashboardPortFromAzdConfig(projectHash)
	if err != nil {
		return nil, fmt.Errorf("dashboard not running for project: %w", err)
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
	projectHash := azdconfig.ProjectHash(projectDir)

	// Try azdconfig first (gRPC)
	configClient, err := azdconfig.NewClient(ctx)
	if err == nil {
		defer configClient.Close()
		port, err := configClient.GetDashboardPort(projectHash)
		if err == nil && port > 0 {
			return port
		}
	}

	// Fallback: read directly from ~/.azd/config.json
	port, _ := readDashboardPortFromAzdConfig(projectHash)
	return port
}

// azdConfigPath returns the path to the azd config file.
// This is a variable to allow test overrides.
var azdConfigPath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return fmt.Sprintf("%s/.azd/config.json", homeDir), nil
}

// azdConfig represents the structure of ~/.azd/config.json (partial).
type azdConfig struct {
	App *appConfig `json:"app"`
}

type appConfig struct {
	Projects map[string]*projectConfig `json:"projects"`
}

type projectConfig struct {
	DashboardPort int `json:"dashboardPort"`
}

// readDashboardPortFromAzdConfig reads the dashboard port directly from ~/.azd/config.json.
// This is used as a fallback when gRPC is not available (running from a separate terminal).
func readDashboardPortFromAzdConfig(projectHash string) (int, error) {
	configPath, err := azdConfigPath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read azd config: %w", err)
	}

	var cfg azdConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, fmt.Errorf("failed to parse azd config: %w", err)
	}

	if cfg.App == nil || cfg.App.Projects == nil {
		return 0, nil
	}

	proj, ok := cfg.App.Projects[projectHash]
	if !ok || proj == nil {
		return 0, nil
	}

	return proj.DashboardPort, nil
}

// GetWebSocketURL returns the WebSocket URL for the dashboard.
func (c *Client) GetWebSocketURL() string {
	return strings.Replace(c.baseURL, "http://", "ws://", 1)
}

// StreamLogs connects to the dashboard's log stream via WebSocket and sends log entries to the provided channel.
// The serviceName parameter filters logs to a specific service (empty string for all services).
// The function blocks until the context is cancelled or an error occurs.
// The caller is responsible for closing the logs channel after StreamLogs returns.
func (c *Client) StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error {
	// Build WebSocket URL
	wsURL := c.GetWebSocketURL() + "/api/logs/stream"
	if serviceName != "" {
		wsURL += "?service=" + serviceName
	}

	// Connect to WebSocket with timeout
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to log stream: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "client closing")

	// Read log entries from WebSocket
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var entry service.LogEntry
			if err := wsjson.Read(ctx, conn, &entry); err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// Check for normal closure
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					return nil
				}
				return fmt.Errorf("failed to read log entry: %w", err)
			}

			// Send to channel (non-blocking with timeout)
			select {
			case logs <- entry:
			case <-time.After(100 * time.Millisecond):
				// Drop if channel is full/slow
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
