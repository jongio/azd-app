// Package azure provides Azure cloud integration for log streaming and resource discovery.
package azure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// Stream buffer size constants
const (
	// defaultScanBufferCap is the initial buffer capacity for log line scanning
	defaultScanBufferCap = 64 * 1024 // 64KB
	// maxScanBufferSize is the maximum buffer size for log line scanning
	maxScanBufferSize = 1024 * 1024 // 1MB
	// defaultReconnectInterval is the default delay between reconnection attempts
	defaultReconnectInterval = 5 * time.Second
	// streamerChannelBuffer is the buffer size for the log entry channel
	streamerChannelBuffer = 1000
)

// RealtimeLogStreamer defines the interface for real-time log streaming from Azure services.
// Implementations provide low-latency log streaming using service-specific APIs.
type RealtimeLogStreamer interface {
	// Start begins streaming logs. Logs are sent to the provided channel.
	// The method blocks until the context is cancelled or an error occurs.
	// Returns nil on graceful shutdown (context cancelled), error otherwise.
	Start(ctx context.Context, logs chan<- LogEntry) error

	// Stop gracefully stops the streamer.
	Stop() error

	// ServiceName returns the name of the service being streamed.
	ServiceName() string

	// ResourceType returns the Azure resource type.
	ResourceType() ResourceType

	// IsConnected returns true if the streamer is actively connected.
	IsConnected() bool
}

// StreamerConfig contains configuration for creating real-time streamers.
type StreamerConfig struct {
	// SubscriptionID is the Azure subscription ID.
	SubscriptionID string

	// ResourceGroup is the Azure resource group name.
	ResourceGroup string

	// ResourceName is the name of the Azure resource.
	ResourceName string

	// ServiceName is the logical service name (may differ from resource name).
	ServiceName string

	// Credential is the Azure credential for authentication.
	Credential azcore.TokenCredential

	// ReconnectInterval is the delay between reconnection attempts.
	// Defaults to 5 seconds if not specified.
	ReconnectInterval time.Duration

	// MaxReconnectAttempts is the maximum number of reconnection attempts.
	// 0 means unlimited reconnections.
	MaxReconnectAttempts int
}

// baseStreamer provides common functionality for all streamers.
type baseStreamer struct {
	config      StreamerConfig
	connected   bool
	connectedMu sync.RWMutex
	stopCh      chan struct{}
	stopOnce    sync.Once
	httpClient  *http.Client
}

func newBaseStreamer(config StreamerConfig) *baseStreamer {
	if config.ReconnectInterval == 0 {
		config.ReconnectInterval = defaultReconnectInterval
	}

	return &baseStreamer{
		config:     config,
		stopCh:     make(chan struct{}),
		httpClient: &http.Client{Timeout: 0}, // No timeout for streaming connections
	}
}

func (b *baseStreamer) setConnected(connected bool) {
	b.connectedMu.Lock()
	defer b.connectedMu.Unlock()
	b.connected = connected
}

func (b *baseStreamer) IsConnected() bool {
	b.connectedMu.RLock()
	defer b.connectedMu.RUnlock()
	return b.connected
}

func (b *baseStreamer) ServiceName() string {
	return b.config.ServiceName
}

func (b *baseStreamer) Stop() error {
	b.stopOnce.Do(func() {
		close(b.stopCh)
	})
	b.setConnected(false)
	return nil
}

func (b *baseStreamer) getToken(ctx context.Context, scope string) (string, error) {
	token, err := b.config.Credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token.Token, nil
}

// ContainerAppStreamer streams logs from Azure Container Apps using the log stream API.
type ContainerAppStreamer struct {
	*baseStreamer
}

// NewContainerAppStreamer creates a new Container Apps log streamer.
func NewContainerAppStreamer(config StreamerConfig) *ContainerAppStreamer {
	return &ContainerAppStreamer{
		baseStreamer: newBaseStreamer(config),
	}
}

// ResourceType returns the Container App resource type.
func (c *ContainerAppStreamer) ResourceType() ResourceType {
	return ResourceTypeContainerApp
}

// Start begins streaming logs from the Container App.
// Uses the Container Apps log stream API endpoint.
func (c *ContainerAppStreamer) Start(ctx context.Context, logs chan<- LogEntry) error {
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stopCh:
			return nil
		default:
		}

		err := c.streamLogs(ctx, logs)
		if err != nil {
			c.setConnected(false)
			slog.Debug("container app log stream error", "service", c.config.ServiceName, "error", err)

			// Check if we should retry
			if c.config.MaxReconnectAttempts > 0 {
				attempts++
				if attempts >= c.config.MaxReconnectAttempts {
					return fmt.Errorf("max reconnection attempts reached: %w", err)
				}
			}

			// Wait before reconnecting
			select {
			case <-ctx.Done():
				return nil
			case <-c.stopCh:
				return nil
			case <-time.After(c.config.ReconnectInterval):
				slog.Debug("reconnecting to container app log stream", "service", c.config.ServiceName, "attempt", attempts)
			}
		}
	}
}

func (c *ContainerAppStreamer) streamLogs(ctx context.Context, logs chan<- LogEntry) error {
	// Get token for ARM
	token, err := c.getToken(ctx, "https://management.azure.com/.default")
	if err != nil {
		return err
	}

	// Build the log stream URL
	// POST /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.App/containerApps/{app}/getLogStream?api-version=2023-05-01
	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.App/containerApps/%s/getLogStream?api-version=2023-05-01",
		c.config.SubscriptionID,
		c.config.ResourceGroup,
		c.config.ResourceName,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to log stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("log stream returned status %d: %s", resp.StatusCode, string(body))
	}

	c.setConnected(true)
	slog.Debug("connected to container app log stream", "service", c.config.ServiceName)

	return c.parseContainerAppStream(ctx, resp.Body, logs)
}

// containerAppLogMessage represents a log message from Container Apps log stream.
type containerAppLogMessage struct {
	Log           string `json:"Log"`
	Stream        string `json:"Stream"`
	Time          string `json:"time"`
	ContainerName string `json:"ContainerName,omitempty"`
	RevisionName  string `json:"RevisionName,omitempty"`
}

func (c *ContainerAppStreamer) parseContainerAppStream(ctx context.Context, reader io.Reader, logs chan<- LogEntry) error {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially large log lines
	buf := make([]byte, 0, defaultScanBufferCap)
	scanner.Buffer(buf, maxScanBufferSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stopCh:
			return nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Handle SSE format: "data: {...}"
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimPrefix(line, "data:")
			line = strings.TrimSpace(line)
		}

		// Skip SSE comments and empty data
		if strings.HasPrefix(line, ":") || line == "" {
			continue
		}

		entry := c.parseLogLine(line)
		if entry.Message != "" {
			select {
			case logs <- entry:
			case <-ctx.Done():
				return nil
			case <-c.stopCh:
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

func (c *ContainerAppStreamer) parseLogLine(line string) LogEntry {
	entry := LogEntry{
		Service:      c.config.ServiceName,
		ResourceType: string(ResourceTypeContainerApp),
		Timestamp:    time.Now(),
		Level:        LogLevelInfo,
	}

	// Try to parse as JSON
	var msg containerAppLogMessage
	if err := json.Unmarshal([]byte(line), &msg); err == nil {
		entry.Message = msg.Log
		entry.ContainerName = msg.ContainerName
		entry.InstanceID = msg.RevisionName

		// Parse timestamp if available
		if msg.Time != "" {
			if t, err := time.Parse(time.RFC3339Nano, msg.Time); err == nil {
				entry.Timestamp = t
			}
		}

		// Determine level from stream type
		if strings.EqualFold(msg.Stream, "stderr") {
			entry.Level = LogLevelError
		}
	} else {
		// Plain text log
		entry.Message = line
	}

	// Infer level from content
	entry.Level = inferLogLevel(entry.Message, entry.Level)

	return entry
}

// AppServiceStreamer streams logs from Azure App Service using the Kudu logstream endpoint.
type AppServiceStreamer struct {
	*baseStreamer
}

// NewAppServiceStreamer creates a new App Service log streamer.
func NewAppServiceStreamer(config StreamerConfig) *AppServiceStreamer {
	return &AppServiceStreamer{
		baseStreamer: newBaseStreamer(config),
	}
}

// ResourceType returns the App Service resource type.
func (a *AppServiceStreamer) ResourceType() ResourceType {
	return ResourceTypeAppService
}

// Start begins streaming logs from the App Service.
// Uses the Kudu logstream API endpoint.
func (a *AppServiceStreamer) Start(ctx context.Context, logs chan<- LogEntry) error {
	attempts := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-a.stopCh:
			return nil
		default:
		}

		err := a.streamLogs(ctx, logs)
		if err != nil {
			a.setConnected(false)
			slog.Debug("app service log stream error", "service", a.config.ServiceName, "error", err)

			// Check if we should retry
			if a.config.MaxReconnectAttempts > 0 {
				attempts++
				if attempts >= a.config.MaxReconnectAttempts {
					return fmt.Errorf("max reconnection attempts reached: %w", err)
				}
			}

			// Wait before reconnecting
			select {
			case <-ctx.Done():
				return nil
			case <-a.stopCh:
				return nil
			case <-time.After(a.config.ReconnectInterval):
				slog.Debug("reconnecting to app service log stream", "service", a.config.ServiceName, "attempt", attempts)
			}
		}
	}
}

func (a *AppServiceStreamer) streamLogs(ctx context.Context, logs chan<- LogEntry) error {
	// Get token for App Service
	token, err := a.getToken(ctx, "https://management.azure.com/.default")
	if err != nil {
		return err
	}

	// Build the Kudu log stream URL
	// GET https://{app}.scm.azurewebsites.net/api/logstream
	url := fmt.Sprintf("https://%s.scm.azurewebsites.net/api/logstream", a.config.ResourceName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to log stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("log stream returned status %d: %s", resp.StatusCode, string(body))
	}

	a.setConnected(true)
	slog.Debug("connected to app service log stream", "service", a.config.ServiceName)

	return a.parseAppServiceStream(ctx, resp.Body, logs)
}

// appServiceLogPattern matches App Service log format: "2024-01-15T10:30:45  PID[123] Category message"
var appServiceLogPattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\s+(\S+)\s+(.*)$`)

func (a *AppServiceStreamer) parseAppServiceStream(ctx context.Context, reader io.Reader, logs chan<- LogEntry) error {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for potentially large log lines
	buf := make([]byte, 0, defaultScanBufferCap)
	scanner.Buffer(buf, maxScanBufferSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		case <-a.stopCh:
			return nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := a.parseLogLine(line)
		if entry.Message != "" {
			select {
			case logs <- entry:
			case <-ctx.Done():
				return nil
			case <-a.stopCh:
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

func (a *AppServiceStreamer) parseLogLine(line string) LogEntry {
	entry := LogEntry{
		Service:      a.config.ServiceName,
		ResourceType: string(ResourceTypeAppService),
		Timestamp:    time.Now(),
		Level:        LogLevelInfo,
	}

	// Try to parse App Service log format
	matches := appServiceLogPattern.FindStringSubmatch(line)
	if len(matches) == 4 {
		// Parse timestamp
		if t, err := time.Parse("2006-01-02T15:04:05", matches[1]); err == nil {
			entry.Timestamp = t
		}
		entry.InstanceID = matches[2] // PID or instance identifier
		entry.Message = matches[3]
	} else {
		// Plain text log
		entry.Message = line
	}

	// Infer level from content
	entry.Level = inferLogLevel(entry.Message, entry.Level)

	return entry
}

// FunctionStreamer streams logs from Azure Functions using the log stream API.
// It uses the same Kudu-based streaming as App Service.
type FunctionStreamer struct {
	*AppServiceStreamer
}

// NewFunctionStreamer creates a new Azure Functions log streamer.
func NewFunctionStreamer(config StreamerConfig) *FunctionStreamer {
	return &FunctionStreamer{
		AppServiceStreamer: NewAppServiceStreamer(config),
	}
}

// ResourceType returns the Function resource type.
func (f *FunctionStreamer) ResourceType() ResourceType {
	return ResourceTypeFunction
}

// inferLogLevel determines log level from message content.
func inferLogLevel(message string, defaultLevel LogLevel) LogLevel {
	msgLower := strings.ToLower(message)

	// Error indicators
	if strings.Contains(msgLower, "error") ||
		strings.Contains(msgLower, "exception") ||
		strings.Contains(msgLower, "failed") ||
		strings.Contains(msgLower, "fatal") ||
		strings.Contains(msgLower, "critical") {
		return LogLevelError
	}

	// Warning indicators
	if strings.Contains(msgLower, "warning") ||
		strings.Contains(msgLower, "warn") {
		return LogLevelWarn
	}

	// Debug indicators
	if strings.Contains(msgLower, "debug") ||
		strings.Contains(msgLower, "trace") ||
		strings.Contains(msgLower, "verbose") {
		return LogLevelDebug
	}

	return defaultLevel
}

// NewRealtimeStreamer creates the appropriate streamer based on resource type.
func NewRealtimeStreamer(resourceType ResourceType, config StreamerConfig) (RealtimeLogStreamer, error) {
	switch resourceType {
	case ResourceTypeContainerApp:
		return NewContainerAppStreamer(config), nil
	case ResourceTypeAppService:
		return NewAppServiceStreamer(config), nil
	case ResourceTypeFunction:
		return NewFunctionStreamer(config), nil
	default:
		return nil, fmt.Errorf("real-time streaming not supported for resource type: %s", resourceType)
	}
}

// StreamerManager manages multiple real-time log streamers.
type StreamerManager struct {
	streamers map[string]RealtimeLogStreamer
	mu        sync.RWMutex
	logs      chan LogEntry
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewStreamerManager creates a new streamer manager.
func NewStreamerManager() *StreamerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamerManager{
		streamers: make(map[string]RealtimeLogStreamer),
		logs:      make(chan LogEntry, streamerChannelBuffer),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddStreamer adds and starts a streamer for a service.
func (m *StreamerManager) AddStreamer(streamer RealtimeLogStreamer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	serviceName := streamer.ServiceName()
	if _, exists := m.streamers[serviceName]; exists {
		return fmt.Errorf("streamer already exists for service: %s", serviceName)
	}

	m.streamers[serviceName] = streamer

	// Start streaming in a goroutine
	go func() {
		if err := streamer.Start(m.ctx, m.logs); err != nil {
			slog.Error("streamer error", "service", serviceName, "error", err)
		}
	}()

	return nil
}

// RemoveStreamer stops and removes a streamer.
func (m *StreamerManager) RemoveStreamer(serviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	streamer, exists := m.streamers[serviceName]
	if !exists {
		return nil
	}

	if err := streamer.Stop(); err != nil {
		return err
	}

	delete(m.streamers, serviceName)
	return nil
}

// GetStreamer returns the streamer for a service.
func (m *StreamerManager) GetStreamer(serviceName string) (RealtimeLogStreamer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streamers[serviceName]
	return s, ok
}

// Logs returns the channel for receiving log entries from all streamers.
func (m *StreamerManager) Logs() <-chan LogEntry {
	return m.logs
}

// Stop stops all streamers and cleans up resources.
func (m *StreamerManager) Stop() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, streamer := range m.streamers {
		if err := streamer.Stop(); err != nil {
			lastErr = err
			slog.Error("failed to stop streamer", "service", name, "error", err)
		}
	}

	m.streamers = make(map[string]RealtimeLogStreamer)
	return lastErr
}

// ActiveStreamers returns the names of all active streamers.
func (m *StreamerManager) ActiveStreamers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.streamers))
	for name := range m.streamers {
		names = append(names, name)
	}
	return names
}

// ConnectedStreamers returns the names of streamers that are currently connected.
func (m *StreamerManager) ConnectedStreamers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0)
	for name, streamer := range m.streamers {
		if streamer.IsConnected() {
			names = append(names, name)
		}
	}
	return names
}
