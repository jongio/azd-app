// Package azdconfig provides access to azd's configuration services via gRPC.
// This package wraps the azdext UserConfig and Environment services to store
// and retrieve configuration for the azd app extension.
//
// For unit testing, use NewInMemoryClient() which provides an in-memory
// implementation that doesn't require a gRPC connection.
//
// # Why this is not azdext.ConfigHelper
//
// azdext.ConfigHelper wraps the same two gRPC services, but it cannot stand in
// for this package:
//
//   - It exposes no wrapper for the UserConfig GetSection RPC, which
//     GetAllServicePorts needs to read a whole subtree in one call.
//   - Its setters are SetUserJSON and SetEnvJSON, which json.Marshal the value.
//     This package writes raw bytes, so switching would change the stored
//     representation of every existing key and orphan values already on disk.
//   - It is a concrete struct over *azdext.AzdClient with no interface, so it
//     cannot replace the ConfigClient seam that dashboard.Server and
//     portmanager.PortManager depend on, nor InMemoryClient, which is both the
//     test fake and the production fallback when azd's gRPC server is absent.
//   - ProjectHash determines on-disk paths under ~/.azd/azd-app/<hash>/, so it
//     has to keep producing the same value regardless of config plumbing.
//
// Its path validator also permits dots, so it would not have prevented the
// nested-key bug that encodeSegment now guards against.
package azdconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

// ConfigClient defines the interface for configuration operations.
// This interface is implemented by both the gRPC client (Client) and the
// in-memory client (InMemoryClient) for testing.
//
// Project-scoped settings use a projectHash parameter which is a unique
// identifier derived from the project directory path. Use ProjectHash()
// to generate this value.
//
// Methods return (zero-value, nil) when a key is not found, not an error.
// This allows callers to distinguish between "not set" and "error reading".
type ConfigClient interface {
	// GetDashboardPort retrieves the dashboard port for a project. Returns 0 if not set.
	GetDashboardPort(projectHash string) (int, error)
	// SetDashboardPort stores the dashboard port for a project.
	SetDashboardPort(projectHash string, port int) error
	// ClearDashboardPort removes the dashboard port for a project.
	ClearDashboardPort(projectHash string) error
	// GetServicePort retrieves the assigned port for a service. Returns 0 if not set.
	GetServicePort(projectHash, serviceName string) (int, error)
	// SetServicePort stores the assigned port for a service.
	SetServicePort(projectHash, serviceName string, port int) error
	// ClearServicePort removes the assigned port for a service.
	ClearServicePort(projectHash, serviceName string) error
	// GetAllServicePorts retrieves all port assignments for a project.
	GetAllServicePorts(projectHash string) (map[string]int, error)
	// GetPreference retrieves a user preference value as a string. Returns "" if not set.
	GetPreference(key string) (string, error)
	// SetPreference stores a user preference value.
	SetPreference(key, value string) error
	// GetPreferenceSection retrieves a user preference section as JSON bytes. Returns nil if not set.
	GetPreferenceSection(key string) ([]byte, error)
	// SetPreferenceSection stores a user preference section from JSON bytes.
	SetPreferenceSection(key string, value []byte) error
	// ClearPreference removes a user preference.
	ClearPreference(key string) error
	// Close releases any resources held by the client (e.g., gRPC connection).
	Close()
}

// Client wraps the azd gRPC client for configuration operations.
type Client struct {
	azdClient *azdext.AzdClient
	ctx       context.Context
}

// Ensure Client implements ConfigClient
var _ ConfigClient = (*Client)(nil)

// NewClient creates a new configuration client.
// This should be called within a command context where azd has started the gRPC server.
func NewClient(ctx context.Context) (*Client, error) {
	azdClient, err := azdext.NewAzdClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create azd client: %w", err)
	}

	return &Client{
		azdClient: azdClient,
		ctx:       azdext.WithAccessToken(ctx),
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() {
	if c.azdClient != nil {
		c.azdClient.Close()
	}
}

// ProjectHash returns a unique hash for a project directory path.
// This is used as a key in the config to avoid path characters in keys.
func ProjectHash(projectDir string) string {
	// Normalize the path
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}
	absPath = filepath.Clean(absPath)
	absPath = strings.ToLower(absPath) // Case-insensitive on Windows

	// Create a short hash
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
}

// configPath builds a config path for a project setting.
func projectConfigPath(projectHash, key string) string {
	return fmt.Sprintf("app.projects.%s.%s", projectHash, key)
}

// preferencePath builds a config path for a user preference.
func preferencePath(key string) string {
	return fmt.Sprintf("app.preferences.%s", key)
}

// encodeSegment makes an arbitrary name safe to use as a single segment of a
// dot-separated config path.
//
// Config paths address a nested JSON tree, so a service named "foo.bar" written
// verbatim produces {"foo":{"bar":port}} rather than a single "foo.bar" key.
// GetAllServicePorts reads that subtree back and its value type switch only
// accepts numbers and strings, so a nested object is skipped and the port
// silently disappears. Service names come from azure.yaml and nothing on the
// port-assignment path validates them.
//
// Percent-encoding every byte outside [A-Za-z0-9_-] keeps the segment flat and
// reversible. Names that already consist only of those characters encode to
// themselves, so keys written by earlier versions still resolve unchanged; only
// names that were already broken get a new, working key.
func encodeSegment(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// decodeSegment reverses encodeSegment. A malformed escape is returned as-is
// rather than dropped, so a hand-edited config still round-trips to something
// the user can recognize.
func decodeSegment(name string) string {
	if !strings.Contains(name, "%") {
		return name
	}
	var b strings.Builder
	b.Grow(len(name))
	for i := 0; i < len(name); i++ {
		if name[i] != '%' || i+2 >= len(name) {
			b.WriteByte(name[i])
			continue
		}
		hi, okHi := unhex(name[i+1])
		lo, okLo := unhex(name[i+2])
		if !okHi || !okLo {
			b.WriteByte(name[i])
			continue
		}
		b.WriteByte(hi<<4 | lo)
		i += 2
	}
	return b.String()
}

func unhex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}

// servicePortPath builds the config path holding one service's assigned port.
func servicePortPath(projectHash, serviceName string) string {
	return projectConfigPath(projectHash, "ports."+encodeSegment(serviceName))
}

// GetDashboardPort retrieves the dashboard port for a project.
// Returns 0 if not set.
func (c *Client) GetDashboardPort(projectHash string) (int, error) {
	path := projectConfigPath(projectHash, "dashboardPort")
	resp, err := c.azdClient.UserConfig().GetString(c.ctx, &azdext.GetUserConfigStringRequest{
		Path: path,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get dashboard port: %w", err)
	}
	if !resp.Found || resp.Value == "" {
		return 0, nil
	}
	port, err := strconv.Atoi(resp.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid dashboard port value: %w", err)
	}
	return port, nil
}

// SetDashboardPort stores the dashboard port for a project.
func (c *Client) SetDashboardPort(projectHash string, port int) error {
	path := projectConfigPath(projectHash, "dashboardPort")
	_, err := c.azdClient.UserConfig().Set(c.ctx, &azdext.SetUserConfigRequest{
		Path:  path,
		Value: []byte(strconv.Itoa(port)),
	})
	if err != nil {
		return fmt.Errorf("failed to set dashboard port: %w", err)
	}
	return nil
}

// ClearDashboardPort removes the dashboard port for a project.
func (c *Client) ClearDashboardPort(projectHash string) error {
	path := projectConfigPath(projectHash, "dashboardPort")
	_, err := c.azdClient.UserConfig().Unset(c.ctx, &azdext.UnsetUserConfigRequest{
		Path: path,
	})
	if err != nil {
		return fmt.Errorf("failed to clear dashboard port: %w", err)
	}
	return nil
}

// GetServicePort retrieves the assigned port for a service.
// Returns 0 if not set.
func (c *Client) GetServicePort(projectHash, serviceName string) (int, error) {
	path := servicePortPath(projectHash, serviceName)
	resp, err := c.azdClient.UserConfig().GetString(c.ctx, &azdext.GetUserConfigStringRequest{
		Path: path,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get service port: %w", err)
	}
	if !resp.Found || resp.Value == "" {
		return 0, nil
	}
	port, err := strconv.Atoi(resp.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid service port value: %w", err)
	}
	return port, nil
}

// SetServicePort stores the assigned port for a service.
func (c *Client) SetServicePort(projectHash, serviceName string, port int) error {
	path := servicePortPath(projectHash, serviceName)
	_, err := c.azdClient.UserConfig().Set(c.ctx, &azdext.SetUserConfigRequest{
		Path:  path,
		Value: []byte(strconv.Itoa(port)),
	})
	if err != nil {
		return fmt.Errorf("failed to set service port: %w", err)
	}
	return nil
}

// ClearServicePort removes the assigned port for a service.
func (c *Client) ClearServicePort(projectHash, serviceName string) error {
	path := servicePortPath(projectHash, serviceName)
	_, err := c.azdClient.UserConfig().Unset(c.ctx, &azdext.UnsetUserConfigRequest{
		Path: path,
	})
	if err != nil {
		return fmt.Errorf("failed to clear service port: %w", err)
	}
	return nil
}

// GetAllServicePorts retrieves all port assignments for a project.
func (c *Client) GetAllServicePorts(projectHash string) (map[string]int, error) {
	path := projectConfigPath(projectHash, "ports")
	resp, err := c.azdClient.UserConfig().GetSection(c.ctx, &azdext.GetUserConfigSectionRequest{
		Path: path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get service ports: %w", err)
	}
	if !resp.Found || len(resp.Section) == 0 {
		return make(map[string]int), nil
	}

	var ports map[string]any
	if err := json.Unmarshal(resp.Section, &ports); err != nil {
		return nil, fmt.Errorf("failed to parse service ports: %w", err)
	}

	result := make(map[string]int)
	for name, value := range ports {
		switch v := value.(type) {
		case float64:
			result[decodeSegment(name)] = int(v)
		case string:
			if port, err := strconv.Atoi(v); err == nil {
				result[decodeSegment(name)] = port
			}
		}
	}
	return result, nil
}

// GetPreference retrieves a user preference value as a string.
func (c *Client) GetPreference(key string) (string, error) {
	path := preferencePath(key)
	resp, err := c.azdClient.UserConfig().GetString(c.ctx, &azdext.GetUserConfigStringRequest{
		Path: path,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get preference: %w", err)
	}
	if !resp.Found {
		return "", nil
	}
	return resp.Value, nil
}

// SetPreference stores a user preference value.
func (c *Client) SetPreference(key, value string) error {
	path := preferencePath(key)
	_, err := c.azdClient.UserConfig().Set(c.ctx, &azdext.SetUserConfigRequest{
		Path:  path,
		Value: []byte(value),
	})
	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}
	return nil
}

// GetPreferenceSection retrieves a user preference section as JSON bytes.
func (c *Client) GetPreferenceSection(key string) ([]byte, error) {
	path := preferencePath(key)
	resp, err := c.azdClient.UserConfig().GetSection(c.ctx, &azdext.GetUserConfigSectionRequest{
		Path: path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get preference section: %w", err)
	}
	if !resp.Found {
		return nil, nil
	}
	return resp.Section, nil
}

// SetPreferenceSection stores a user preference section from JSON bytes.
func (c *Client) SetPreferenceSection(key string, value []byte) error {
	path := preferencePath(key)
	_, err := c.azdClient.UserConfig().Set(c.ctx, &azdext.SetUserConfigRequest{
		Path:  path,
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("failed to set preference section: %w", err)
	}
	return nil
}

// ClearPreference removes a user preference.
func (c *Client) ClearPreference(key string) error {
	path := preferencePath(key)
	_, err := c.azdClient.UserConfig().Unset(c.ctx, &azdext.UnsetUserConfigRequest{
		Path: path,
	})
	if err != nil {
		return fmt.Errorf("failed to clear preference: %w", err)
	}
	return nil
}

// InMemoryClient provides an in-memory implementation of ConfigClient for testing.
// This allows unit tests to run without requiring a gRPC connection to azd.
type InMemoryClient struct {
	mu   sync.RWMutex
	data map[string]any
}

// Ensure InMemoryClient implements ConfigClient
var _ ConfigClient = (*InMemoryClient)(nil)

// NewInMemoryClient creates a new in-memory configuration client for testing.
func NewInMemoryClient() *InMemoryClient {
	return &InMemoryClient{
		data: make(map[string]any),
	}
}

// Close is a no-op for the in-memory client.
func (c *InMemoryClient) Close() {}

// GetDashboardPort retrieves the dashboard port for a project.
func (c *InMemoryClient) GetDashboardPort(projectHash string) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := projectConfigPath(projectHash, "dashboardPort")
	if v, ok := c.data[path]; ok {
		if port, ok := v.(int); ok {
			return port, nil
		}
	}
	return 0, nil
}

// SetDashboardPort stores the dashboard port for a project.
func (c *InMemoryClient) SetDashboardPort(projectHash string, port int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := projectConfigPath(projectHash, "dashboardPort")
	c.data[path] = port
	return nil
}

// ClearDashboardPort removes the dashboard port for a project.
func (c *InMemoryClient) ClearDashboardPort(projectHash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := projectConfigPath(projectHash, "dashboardPort")
	delete(c.data, path)
	return nil
}

// GetServicePort retrieves the assigned port for a service.
func (c *InMemoryClient) GetServicePort(projectHash, serviceName string) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := servicePortPath(projectHash, serviceName)
	if v, ok := c.data[path]; ok {
		if port, ok := v.(int); ok {
			return port, nil
		}
	}
	return 0, nil
}

// SetServicePort stores the assigned port for a service.
func (c *InMemoryClient) SetServicePort(projectHash, serviceName string, port int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := servicePortPath(projectHash, serviceName)
	c.data[path] = port
	return nil
}

// ClearServicePort removes the assigned port for a service.
func (c *InMemoryClient) ClearServicePort(projectHash, serviceName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := servicePortPath(projectHash, serviceName)
	delete(c.data, path)
	return nil
}

// GetAllServicePorts retrieves all port assignments for a project.
func (c *InMemoryClient) GetAllServicePorts(projectHash string) (map[string]int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	prefix := projectConfigPath(projectHash, "ports.")
	result := make(map[string]int)
	for k, v := range c.data {
		if strings.HasPrefix(k, prefix) {
			serviceName := decodeSegment(strings.TrimPrefix(k, prefix))
			if port, ok := v.(int); ok {
				result[serviceName] = port
			}
		}
	}
	return result, nil
}

// GetPreference retrieves a user preference value as a string.
func (c *InMemoryClient) GetPreference(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := preferencePath(key)
	if v, ok := c.data[path]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
	}
	return "", nil
}

// SetPreference stores a user preference value.
func (c *InMemoryClient) SetPreference(key, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := preferencePath(key)
	c.data[path] = value
	return nil
}

// GetPreferenceSection retrieves a user preference section as JSON bytes.
func (c *InMemoryClient) GetPreferenceSection(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := preferencePath(key)
	if v, ok := c.data[path]; ok {
		if b, ok := v.([]byte); ok {
			return b, nil
		}
	}
	return nil, nil
}

// SetPreferenceSection stores a user preference section from JSON bytes.
func (c *InMemoryClient) SetPreferenceSection(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := preferencePath(key)
	c.data[path] = value
	return nil
}

// ClearPreference removes a user preference.
func (c *InMemoryClient) ClearPreference(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := preferencePath(key)
	delete(c.data, path)
	return nil
}
