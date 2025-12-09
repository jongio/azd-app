// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/jongio/azd-app/cli/src/internal/azure"
)

// LogMode represents the source of logs.
type LogMode string

const (
	// LogModeLocal indicates logs from locally running services.
	LogModeLocal LogMode = "local"
	// LogModeAzure indicates logs from Azure-deployed services.
	LogModeAzure LogMode = "azure"
)

// AzureLogsConfig represents Azure log streaming configuration.
type AzureLogsConfig struct {
	Enabled         bool              `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	WorkspaceID     string            `yaml:"workspace,omitempty" json:"workspace,omitempty"`
	PollingInterval time.Duration     `yaml:"pollingInterval,omitempty" json:"pollingInterval,omitempty"`
	DefaultTimespan time.Duration     `yaml:"defaultTimespan,omitempty" json:"defaultTimespan,omitempty"`
	Queries         map[string]string `yaml:"queries,omitempty" json:"queries,omitempty"`   // Custom KQL queries per service
	RealtimeEnabled bool              `yaml:"realtime,omitempty" json:"realtime,omitempty"` // Enable real-time streaming APIs
}

// DefaultAzurePollingInterval is the default interval for polling Azure logs.
const DefaultAzurePollingInterval = 30 * time.Second

// DefaultAzureTimespan is the default timespan for Azure log queries.
const DefaultAzureTimespan = 1 * time.Hour

// AzureLogBuffer manages Azure log streaming with mode switching support.
type AzureLogBuffer struct {
	config          *AzureLogsConfig
	mode            LogMode
	modeMu          sync.RWMutex
	logClient       *azure.LogAnalyticsClient
	resourceDiscov  *azure.ResourceDiscovery
	buffers         map[string]*LogBuffer // Service name -> buffer
	buffersMu       sync.RWMutex
	subscribers     map[chan LogEntry]bool
	subMu           sync.RWMutex
	pollCancel      context.CancelFunc
	pollCtx         context.Context
	lastQueryTime   map[string]time.Time
	lastQueryMu     sync.RWMutex
	projectDir      string
	subscriptionID  string
	resourceGroup   string
	environmentName string
	credential      azcore.TokenCredential
	streamerManager *azure.StreamerManager
	realtimeMode    bool
	realtimeMu      sync.RWMutex
}

// NewAzureLogBuffer creates a new Azure log buffer manager.
func NewAzureLogBuffer(config *AzureLogsConfig, projectDir string) *AzureLogBuffer {
	if config == nil {
		config = &AzureLogsConfig{}
	}

	// Apply defaults
	if config.PollingInterval == 0 {
		config.PollingInterval = DefaultAzurePollingInterval
	}
	if config.DefaultTimespan == 0 {
		config.DefaultTimespan = DefaultAzureTimespan
	}

	return &AzureLogBuffer{
		config:        config,
		mode:          LogModeLocal,
		buffers:       make(map[string]*LogBuffer),
		subscribers:   make(map[chan LogEntry]bool),
		lastQueryTime: make(map[string]time.Time),
		projectDir:    projectDir,
	}
}

// SetMode switches the log source mode.
func (a *AzureLogBuffer) SetMode(mode LogMode) error {
	a.modeMu.Lock()
	defer a.modeMu.Unlock()

	if a.mode == mode {
		return nil // No change needed
	}

	oldMode := a.mode
	a.mode = mode

	slog.Info("switching log mode", "from", oldMode, "to", mode)

	// Stop Azure polling when switching to local
	if mode == LogModeLocal && a.pollCancel != nil {
		a.pollCancel()
		a.pollCancel = nil
	}

	// Start Azure polling when switching to Azure
	if mode == LogModeAzure && a.config.Enabled {
		return a.startAzurePolling()
	}

	return nil
}

// GetMode returns the current log source mode.
func (a *AzureLogBuffer) GetMode() LogMode {
	a.modeMu.RLock()
	defer a.modeMu.RUnlock()
	return a.mode
}

// Initialize sets up Azure credentials and resource discovery.
func (a *AzureLogBuffer) Initialize(ctx context.Context, subscriptionID, resourceGroup, environmentName string) error {
	a.subscriptionID = subscriptionID
	a.resourceGroup = resourceGroup
	a.environmentName = environmentName

	// Create credential
	cred, err := azure.NewAzureCredential()
	if err != nil {
		return err
	}
	a.credential = cred

	// Create Log Analytics client
	if a.config.WorkspaceID != "" {
		client, err := azure.NewLogAnalyticsClient(cred, a.config.WorkspaceID)
		if err != nil {
			return err
		}
		a.logClient = client
	}

	// Create resource discovery
	discovery := azure.NewResourceDiscovery(cred, a.projectDir)
	a.resourceDiscov = discovery

	// Initialize streamer manager for real-time streaming
	a.streamerManager = azure.NewStreamerManager()

	return nil
}

// startAzurePolling begins polling Azure for logs.
func (a *AzureLogBuffer) startAzurePolling() error {
	// Cancel any existing polling
	if a.pollCancel != nil {
		a.pollCancel()
	}

	a.pollCtx, a.pollCancel = context.WithCancel(context.Background())

	// Check if real-time mode is enabled and should be used
	a.realtimeMu.RLock()
	useRealtime := a.realtimeMode && a.config.RealtimeEnabled
	a.realtimeMu.RUnlock()

	if useRealtime {
		// Start real-time streaming
		if err := a.startRealtimeStreaming(); err != nil {
			slog.Warn("failed to start real-time streaming, falling back to polling", "error", err)
			// Fall back to polling
			go a.pollAzureLogs()
		} else {
			// Start goroutine to process real-time logs
			go a.processRealtimeLogs()
			return nil
		}
	} else if a.logClient != nil {
		// Use standard polling
		go a.pollAzureLogs()
	}

	return nil
}

// pollAzureLogs continuously polls Azure for new logs.
func (a *AzureLogBuffer) pollAzureLogs() {
	ticker := time.NewTicker(a.config.PollingInterval)
	defer ticker.Stop()

	// Do initial poll immediately
	a.fetchAzureLogs()

	for {
		select {
		case <-a.pollCtx.Done():
			slog.Debug("azure log polling stopped")
			return
		case <-ticker.C:
			a.fetchAzureLogs()
		}
	}
}

// fetchAzureLogs fetches logs from Azure for all discovered services.
func (a *AzureLogBuffer) fetchAzureLogs() {
	if a.resourceDiscov == nil || a.logClient == nil {
		return
	}

	// Discover resources
	result, err := a.resourceDiscov.Discover(a.pollCtx)
	if err != nil {
		slog.Debug("failed to discover azure resources", "error", err)
		return
	}

	for _, resource := range result.Resources {
		a.fetchServiceLogs(*resource)
	}
}

// fetchServiceLogs fetches logs for a specific service from Azure.
func (a *AzureLogBuffer) fetchServiceLogs(resource azure.AzureResource) {
	// Get last query time for incremental fetches
	a.lastQueryMu.RLock()
	lastTime, hasLastTime := a.lastQueryTime[resource.ServiceName]
	a.lastQueryMu.RUnlock()

	var azureEntries []azure.LogEntry
	var err error

	// Get custom query if configured
	customQuery := ""
	if a.config.Queries != nil {
		customQuery = a.config.Queries[resource.ServiceName]
	}

	if hasLastTime {
		// Incremental query since last fetch
		azureEntries, err = a.logClient.QueryLogsSince(a.pollCtx, resource.ServiceName, resource.ResourceType, lastTime, customQuery)
	} else {
		// Initial query with default timespan
		azureEntries, err = a.logClient.QueryLogs(a.pollCtx, resource.ServiceName, resource.ResourceType, a.config.DefaultTimespan, customQuery)
	}

	if err != nil {
		slog.Debug("failed to query azure logs", "service", resource.ServiceName, "error", err)
		return
	}

	// Update last query time
	a.lastQueryMu.Lock()
	a.lastQueryTime[resource.ServiceName] = time.Now()
	a.lastQueryMu.Unlock()

	// Convert and add entries to buffer
	for _, azEntry := range azureEntries {
		entry := convertAzureLogEntry(azEntry)
		a.addEntry(entry)
	}

	slog.Debug("fetched azure logs", "service", resource.ServiceName, "count", len(azureEntries))
}

// startRealtimeStreaming initializes and starts real-time log streamers for discovered resources.
func (a *AzureLogBuffer) startRealtimeStreaming() error {
	if a.resourceDiscov == nil || a.credential == nil {
		return fmt.Errorf("resource discovery or credentials not initialized")
	}

	// Discover resources
	result, err := a.resourceDiscov.Discover(a.pollCtx)
	if err != nil {
		return fmt.Errorf("failed to discover resources: %w", err)
	}

	streamersStarted := 0
	var lastErr error

	for _, resource := range result.Resources {
		// Check if real-time streaming is supported for this resource type
		if !isRealtimeSupported(resource.ResourceType) {
			slog.Debug("real-time streaming not supported", "service", resource.ServiceName, "resourceType", resource.ResourceType)
			continue
		}

		config := azure.StreamerConfig{
			SubscriptionID:       a.subscriptionID,
			ResourceGroup:        a.resourceGroup,
			ResourceName:         resource.Name,
			ServiceName:          resource.ServiceName,
			Credential:           a.credential,
			ReconnectInterval:    5 * time.Second,
			MaxReconnectAttempts: 0, // Unlimited retries
		}

		streamer, err := azure.NewRealtimeStreamer(resource.ResourceType, config)
		if err != nil {
			slog.Debug("failed to create streamer", "service", resource.ServiceName, "error", err)
			lastErr = err
			continue
		}

		if err := a.streamerManager.AddStreamer(streamer); err != nil {
			slog.Debug("failed to add streamer", "service", resource.ServiceName, "error", err)
			lastErr = err
			continue
		}

		streamersStarted++
		slog.Info("started real-time log streaming", "service", resource.ServiceName, "resourceType", resource.ResourceType)
	}

	if streamersStarted == 0 && lastErr != nil {
		return fmt.Errorf("failed to start any streamers: %w", lastErr)
	}

	slog.Info("real-time streaming initialized", "streamersCount", streamersStarted)
	return nil
}

// processRealtimeLogs reads logs from the streamer manager and adds them to buffers.
func (a *AzureLogBuffer) processRealtimeLogs() {
	if a.streamerManager == nil {
		return
	}

	logs := a.streamerManager.Logs()

	for {
		select {
		case <-a.pollCtx.Done():
			return
		case logEntry, ok := <-logs:
			if !ok {
				return
			}
			// Convert azure.LogEntry to service.LogEntry
			entry := convertRealtimeLogEntry(logEntry)
			a.addEntry(entry)
		}
	}
}

// convertRealtimeLogEntry converts an azure.LogEntry from real-time streaming to a service.LogEntry.
func convertRealtimeLogEntry(azEntry azure.LogEntry) LogEntry {
	return LogEntry{
		Service:   azEntry.Service,
		Message:   azEntry.Message,
		Level:     convertAzureLogLevel(azEntry.Level),
		Timestamp: azEntry.Timestamp,
		Source:    LogSourceAzure,
		AzureMetadata: &AzureLogMetadata{
			ResourceType:  azEntry.ResourceType,
			ContainerName: azEntry.ContainerName,
			InstanceID:    azEntry.InstanceID,
		},
	}
}

// isRealtimeSupported returns true if real-time streaming is supported for the given resource type.
func isRealtimeSupported(resourceType azure.ResourceType) bool {
	switch resourceType {
	case azure.ResourceTypeContainerApp, azure.ResourceTypeAppService, azure.ResourceTypeFunction:
		return true
	default:
		return false
	}
}

// SetRealtimeMode enables or disables real-time streaming mode.
// When enabled, the buffer will attempt to use service-specific streaming APIs
// for lower latency. Falls back to polling on stream failures.
func (a *AzureLogBuffer) SetRealtimeMode(enabled bool) error {
	a.realtimeMu.Lock()
	wasEnabled := a.realtimeMode
	a.realtimeMode = enabled
	a.realtimeMu.Unlock()

	// Get mode while holding neither lock to avoid potential deadlock
	a.modeMu.RLock()
	currentMode := a.mode
	a.modeMu.RUnlock()

	// If mode changed and we're in Azure mode, restart streaming/polling
	if wasEnabled != enabled && currentMode == LogModeAzure {
		slog.Info("switching real-time mode", "enabled", enabled)

		// Stop existing streamers if disabling
		if !enabled && a.streamerManager != nil {
			if err := a.streamerManager.Stop(); err != nil {
				slog.Debug("error stopping streamer manager", "error", err)
			}
			a.streamerManager = azure.NewStreamerManager()
		}

		// Restart with new mode
		return a.startAzurePolling()
	}

	return nil
}

// GetRealtimeMode returns whether real-time streaming is currently enabled.
func (a *AzureLogBuffer) GetRealtimeMode() bool {
	a.realtimeMu.RLock()
	defer a.realtimeMu.RUnlock()
	return a.realtimeMode
}

// GetRealtimeStatus returns the current status of real-time streaming.
func (a *AzureLogBuffer) GetRealtimeStatus() RealtimeStatus {
	a.realtimeMu.RLock()
	enabled := a.realtimeMode
	a.realtimeMu.RUnlock()

	status := RealtimeStatus{
		Enabled:   enabled,
		Supported: a.config.RealtimeEnabled,
	}

	if a.streamerManager != nil {
		status.ActiveStreamers = a.streamerManager.ActiveStreamers()
		status.ConnectedStreamers = a.streamerManager.ConnectedStreamers()
	}

	return status
}

// RealtimeStatus represents the current status of real-time log streaming.
type RealtimeStatus struct {
	Enabled            bool     `json:"enabled"`
	Supported          bool     `json:"supported"`
	ActiveStreamers    []string `json:"activeStreamers"`
	ConnectedStreamers []string `json:"connectedStreamers"`
}

// convertAzureLogEntry converts an Azure LogEntry to a service LogEntry.
func convertAzureLogEntry(azEntry azure.LogEntry) LogEntry {
	return LogEntry{
		Service:   azEntry.Service,
		Message:   azEntry.Message,
		Level:     convertAzureLogLevel(azEntry.Level),
		Timestamp: azEntry.Timestamp,
		Source:    LogSourceAzure,
		AzureMetadata: &AzureLogMetadata{
			ResourceType:  azEntry.ResourceType,
			ContainerName: azEntry.ContainerName,
			InstanceID:    azEntry.InstanceID,
		},
	}
}

// convertAzureLogLevel converts an azure.LogLevel to a service.LogLevel.
func convertAzureLogLevel(level azure.LogLevel) LogLevel {
	switch level {
	case azure.LogLevelError:
		return LogLevelError
	case azure.LogLevelWarn:
		return LogLevelWarn
	case azure.LogLevelDebug:
		return LogLevelDebug
	default:
		return LogLevelInfo
	}
}

// addEntry adds a log entry to the appropriate buffer and broadcasts it.
func (a *AzureLogBuffer) addEntry(entry LogEntry) {
	// Get or create buffer for service
	a.buffersMu.Lock()
	buffer, exists := a.buffers[entry.Service]
	if !exists {
		var err error
		buffer, err = NewLogBuffer(entry.Service, DefaultMaxLogLines, false, a.projectDir)
		if err != nil {
			a.buffersMu.Unlock()
			slog.Debug("failed to create buffer for azure logs", "service", entry.Service, "error", err)
			return
		}
		a.buffers[entry.Service] = buffer
	}
	a.buffersMu.Unlock()

	// Add to buffer
	buffer.Add(entry)

	// Broadcast to subscribers
	a.broadcast(entry)
}

// Subscribe creates a subscription channel for live Azure log streaming.
func (a *AzureLogBuffer) Subscribe() chan LogEntry {
	a.subMu.Lock()
	defer a.subMu.Unlock()

	ch := make(chan LogEntry, 100)
	a.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a subscription channel.
func (a *AzureLogBuffer) Unsubscribe(ch chan LogEntry) {
	a.subMu.Lock()
	defer a.subMu.Unlock()

	if _, exists := a.subscribers[ch]; exists {
		delete(a.subscribers, ch)
		close(ch)
	}
}

// broadcast sends a log entry to all Azure log subscribers.
func (a *AzureLogBuffer) broadcast(entry LogEntry) {
	a.subMu.RLock()
	defer a.subMu.RUnlock()

	for ch := range a.subscribers {
		func(c chan LogEntry) {
			defer func() {
				if r := recover(); r != nil {
					slog.Debug("recovered from panic during azure log broadcast", "error", r)
				}
			}()
			select {
			case c <- entry:
				// Successfully sent
			case <-time.After(DefaultLogSubscriberTimeout):
				// Subscriber too slow, drop message
				slog.Debug("dropped azure log entry for slow subscriber", "service", entry.Service)
			default:
				// Channel buffer full, skip
			}
		}(ch)
	}
}

// GetRecentLogs returns recent logs for a service from Azure.
func (a *AzureLogBuffer) GetRecentLogs(serviceName string, count int) []LogEntry {
	a.buffersMu.RLock()
	buffer, exists := a.buffers[serviceName]
	a.buffersMu.RUnlock()

	if !exists {
		return nil
	}

	return buffer.GetRecent(count)
}

// GetAllRecentLogs returns recent logs for all services.
// Caps total entries to prevent unbounded memory growth.
func (a *AzureLogBuffer) GetAllRecentLogs(count int) []LogEntry {
	a.buffersMu.RLock()
	defer a.buffersMu.RUnlock()

	// Cap total entries to 3x requested count to prevent memory exhaustion
	maxTotal := count * 3
	if maxTotal < 100 {
		maxTotal = 100
	}

	var all []LogEntry
	for _, buffer := range a.buffers {
		if len(all) >= maxTotal {
			break
		}
		entries := buffer.GetRecent(count)
		all = append(all, entries...)
	}

	// Final cap
	if len(all) > maxTotal {
		all = all[:maxTotal]
	}

	return all
}

// GetAzureStatus returns the current Azure connection status.
func (a *AzureLogBuffer) GetAzureStatus() AzureStatus {
	a.modeMu.RLock()
	mode := a.mode
	a.modeMu.RUnlock()

	status := AzureStatus{
		Mode:      mode,
		Connected: a.logClient != nil,
		Enabled:   a.config.Enabled,
	}

	if a.resourceDiscov != nil {
		result, _ := a.resourceDiscov.Discover(context.Background())
		if result != nil {
			status.ResourceCount = len(result.Resources)
		}
	}

	return status
}

// AzureStatus represents the current Azure log streaming status.
type AzureStatus struct {
	Mode          LogMode `json:"mode"`
	Connected     bool    `json:"connected"`
	Enabled       bool    `json:"enabled"`
	ResourceCount int     `json:"resourceCount"`
	LastError     string  `json:"lastError,omitempty"`
}

// Close stops polling and cleans up resources.
func (a *AzureLogBuffer) Close() error {
	// Stop polling
	if a.pollCancel != nil {
		a.pollCancel()
	}

	// Stop real-time streamers
	if a.streamerManager != nil {
		if err := a.streamerManager.Stop(); err != nil {
			slog.Debug("error stopping streamer manager", "error", err)
		}
	}

	// Close all subscribers
	a.subMu.Lock()
	for ch := range a.subscribers {
		close(ch)
		delete(a.subscribers, ch)
	}
	a.subMu.Unlock()

	// Close all buffers
	a.buffersMu.Lock()
	for _, buffer := range a.buffers {
		buffer.Close()
	}
	a.buffers = make(map[string]*LogBuffer)
	a.buffersMu.Unlock()

	return nil
}
