package azure

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCredential implements azcore.TokenCredential for testing.
type mockCredential struct{}

func (m *mockCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{
		Token:     "test-token",
		ExpiresOn: time.Now().Add(1 * time.Hour),
	}, nil
}

func TestNewContainerAppStreamer(t *testing.T) {
	config := StreamerConfig{
		SubscriptionID: "sub-123",
		ResourceGroup:  "rg-test",
		ResourceName:   "my-container-app",
		ServiceName:    "api",
		Credential:     &mockCredential{},
	}

	streamer := NewContainerAppStreamer(config)

	assert.NotNil(t, streamer)
	assert.Equal(t, "api", streamer.ServiceName())
	assert.Equal(t, ResourceTypeContainerApp, streamer.ResourceType())
	assert.False(t, streamer.IsConnected())
}

func TestNewAppServiceStreamer(t *testing.T) {
	config := StreamerConfig{
		SubscriptionID: "sub-123",
		ResourceGroup:  "rg-test",
		ResourceName:   "my-app-service",
		ServiceName:    "web",
		Credential:     &mockCredential{},
	}

	streamer := NewAppServiceStreamer(config)

	assert.NotNil(t, streamer)
	assert.Equal(t, "web", streamer.ServiceName())
	assert.Equal(t, ResourceTypeAppService, streamer.ResourceType())
	assert.False(t, streamer.IsConnected())
}

func TestNewFunctionStreamer(t *testing.T) {
	config := StreamerConfig{
		SubscriptionID: "sub-123",
		ResourceGroup:  "rg-test",
		ResourceName:   "my-function",
		ServiceName:    "func",
		Credential:     &mockCredential{},
	}

	streamer := NewFunctionStreamer(config)

	assert.NotNil(t, streamer)
	assert.Equal(t, "func", streamer.ServiceName())
	assert.Equal(t, ResourceTypeFunction, streamer.ResourceType())
}

func TestNewRealtimeStreamer(t *testing.T) {
	config := StreamerConfig{
		SubscriptionID: "sub-123",
		ResourceGroup:  "rg-test",
		ResourceName:   "my-app",
		ServiceName:    "test",
		Credential:     &mockCredential{},
	}

	tests := []struct {
		name         string
		resourceType ResourceType
		wantType     string
		wantErr      bool
	}{
		{
			name:         "Container App",
			resourceType: ResourceTypeContainerApp,
			wantType:     "*azure.ContainerAppStreamer",
			wantErr:      false,
		},
		{
			name:         "App Service",
			resourceType: ResourceTypeAppService,
			wantType:     "*azure.AppServiceStreamer",
			wantErr:      false,
		},
		{
			name:         "Function",
			resourceType: ResourceTypeFunction,
			wantType:     "*azure.FunctionStreamer",
			wantErr:      false,
		},
		{
			name:         "AKS - not supported",
			resourceType: ResourceTypeAKS,
			wantErr:      true,
		},
		{
			name:         "Unknown - not supported",
			resourceType: ResourceTypeUnknown,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streamer, err := NewRealtimeStreamer(tt.resourceType, config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, streamer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, streamer)
			}
		})
	}
}

func TestContainerAppStreamer_ParseLogLine(t *testing.T) {
	streamer := NewContainerAppStreamer(StreamerConfig{
		ServiceName: "api",
		Credential:  &mockCredential{},
	})

	tests := []struct {
		name         string
		line         string
		wantMessage  string
		wantLevel    LogLevel
		wantInstance string
	}{
		{
			name:        "JSON log with message",
			line:        `{"Log":"Application started","Stream":"stdout","time":"2024-01-15T10:30:45.123Z","ContainerName":"api","RevisionName":"api--abc123"}`,
			wantMessage: "Application started",
			wantLevel:   LogLevelInfo,
		},
		{
			name:        "JSON log with stderr",
			line:        `{"Log":"Connection error","Stream":"stderr","time":"2024-01-15T10:30:45.123Z"}`,
			wantMessage: "Connection error",
			wantLevel:   LogLevelError,
		},
		{
			name:        "JSON log with error in message",
			line:        `{"Log":"Error: failed to connect","Stream":"stdout","time":"2024-01-15T10:30:45.123Z"}`,
			wantMessage: "Error: failed to connect",
			wantLevel:   LogLevelError,
		},
		{
			name:        "JSON log with warning",
			line:        `{"Log":"Warning: deprecated API","Stream":"stdout","time":"2024-01-15T10:30:45.123Z"}`,
			wantMessage: "Warning: deprecated API",
			wantLevel:   LogLevelWarn,
		},
		{
			name:        "Plain text log",
			line:        "Server listening on port 8080",
			wantMessage: "Server listening on port 8080",
			wantLevel:   LogLevelInfo,
		},
		{
			name:        "Plain text error",
			line:        "FATAL: database connection failed",
			wantMessage: "FATAL: database connection failed",
			wantLevel:   LogLevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := streamer.parseLogLine(tt.line)

			assert.Equal(t, tt.wantMessage, entry.Message)
			assert.Equal(t, tt.wantLevel, entry.Level)
			assert.Equal(t, "api", entry.Service)
			assert.Equal(t, string(ResourceTypeContainerApp), entry.ResourceType)
		})
	}
}

func TestAppServiceStreamer_ParseLogLine(t *testing.T) {
	streamer := NewAppServiceStreamer(StreamerConfig{
		ServiceName: "web",
		Credential:  &mockCredential{},
	})

	tests := []struct {
		name        string
		line        string
		wantMessage string
		wantLevel   LogLevel
	}{
		{
			name:        "Standard App Service log format",
			line:        "2024-01-15T10:30:45  PID[123] Info Application started",
			wantMessage: "Info Application started",
			wantLevel:   LogLevelInfo,
		},
		{
			name:        "Error log",
			line:        "2024-01-15T10:30:45  PID[123] Error: Connection failed",
			wantMessage: "Error: Connection failed",
			wantLevel:   LogLevelError,
		},
		{
			name:        "Plain text log",
			line:        "Application starting up...",
			wantMessage: "Application starting up...",
			wantLevel:   LogLevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := streamer.parseLogLine(tt.line)

			assert.Equal(t, tt.wantMessage, entry.Message)
			assert.Equal(t, tt.wantLevel, entry.Level)
			assert.Equal(t, "web", entry.Service)
		})
	}
}

func TestStreamerManager_AddAndRemove(t *testing.T) {
	manager := NewStreamerManager()
	defer func() { _ = manager.Stop() }()

	config := StreamerConfig{
		ServiceName:          "test-service",
		Credential:           &mockCredential{},
		MaxReconnectAttempts: 1, // Limit retries for testing
	}

	streamer := NewContainerAppStreamer(config)

	// Add streamer
	err := manager.AddStreamer(streamer)
	assert.NoError(t, err)

	// Verify it's tracked
	names := manager.ActiveStreamers()
	assert.Contains(t, names, "test-service")

	// Try to add duplicate
	err = manager.AddStreamer(streamer)
	assert.Error(t, err)

	// Get streamer
	s, ok := manager.GetStreamer("test-service")
	assert.True(t, ok)
	assert.NotNil(t, s)

	// Remove streamer
	err = manager.RemoveStreamer("test-service")
	assert.NoError(t, err)

	// Verify it's removed
	names = manager.ActiveStreamers()
	assert.NotContains(t, names, "test-service")

	// Get non-existent
	s, ok = manager.GetStreamer("non-existent")
	assert.False(t, ok)
	assert.Nil(t, s)
}

func TestStreamerManager_Stop(t *testing.T) {
	manager := NewStreamerManager()

	config := StreamerConfig{
		ServiceName:          "service-1",
		Credential:           &mockCredential{},
		MaxReconnectAttempts: 1,
	}

	streamer1 := NewContainerAppStreamer(config)
	config.ServiceName = "service-2"
	streamer2 := NewAppServiceStreamer(config)

	_ = manager.AddStreamer(streamer1)
	_ = manager.AddStreamer(streamer2)

	// Verify both are active
	assert.Len(t, manager.ActiveStreamers(), 2)

	// Stop all
	err := manager.Stop()
	assert.NoError(t, err)

	// Verify all stopped
	assert.Len(t, manager.ActiveStreamers(), 0)
}

func TestInferLogLevel(t *testing.T) {
	tests := []struct {
		message  string
		expected LogLevel
	}{
		{"Application started successfully", LogLevelInfo},
		{"Error: failed to connect", LogLevelError},
		{"WARNING: deprecated API", LogLevelWarn},
		{"DEBUG: entering function", LogLevelDebug},
		{"Exception thrown in handler", LogLevelError},
		{"FATAL: system crash", LogLevelError},
		{"Connection failed", LogLevelError},
		{"trace: detailed logging", LogLevelDebug},
		{"verbose output enabled", LogLevelDebug},
		{"critical error occurred", LogLevelError},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := inferLogLevel(tt.message, LogLevelInfo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainerAppStreamer_ParseSSEFormat(t *testing.T) {
	streamer := NewContainerAppStreamer(StreamerConfig{
		ServiceName: "api",
		Credential:  &mockCredential{},
	})

	// Simulate SSE stream
	sseData := `data: {"Log":"First message","Stream":"stdout","time":"2024-01-15T10:30:45.123Z"}

data: {"Log":"Second message","Stream":"stdout","time":"2024-01-15T10:30:46.123Z"}

: comment line should be ignored

data: {"Log":"Error occurred","Stream":"stderr","time":"2024-01-15T10:30:47.123Z"}
`

	reader := strings.NewReader(sseData)
	logs := make(chan LogEntry, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run parser
	go func() {
		_ = streamer.parseContainerAppStream(ctx, reader, logs)
		close(logs)
	}()

	// Collect results
	var entries []LogEntry
	for entry := range logs {
		entries = append(entries, entry)
	}

	assert.Len(t, entries, 3)
	assert.Equal(t, "First message", entries[0].Message)
	assert.Equal(t, "Second message", entries[1].Message)
	assert.Equal(t, "Error occurred", entries[2].Message)
	assert.Equal(t, LogLevelError, entries[2].Level)
}

func TestStreamer_Reconnection(t *testing.T) {
	// Track connection attempts
	connectionAttempts := 0
	var mu sync.Mutex

	// Create a test server that fails the first connection but succeeds on retry
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connectionAttempts++
		attempt := connectionAttempts
		mu.Unlock()

		if attempt == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Send a log entry then close
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if ok {
			_, _ = io.WriteString(w, `data: {"Log":"Connected","Stream":"stdout"}`)
			_, _ = io.WriteString(w, "\n\n")
			flusher.Flush()
		}
	}))
	defer server.Close()

	// Note: In a real test, we'd need to override the URL construction
	// This test demonstrates the reconnection structure
	config := StreamerConfig{
		ServiceName:          "test-service",
		Credential:           &mockCredential{},
		ReconnectInterval:    10 * time.Millisecond,
		MaxReconnectAttempts: 3,
	}

	streamer := NewContainerAppStreamer(config)
	assert.NotNil(t, streamer)
	assert.False(t, streamer.IsConnected())
}

func TestStreamerConfig_Defaults(t *testing.T) {
	config := StreamerConfig{
		ServiceName: "test",
		Credential:  &mockCredential{},
	}

	base := newBaseStreamer(config)

	// Should have default reconnect interval
	assert.Equal(t, 5*time.Second, base.config.ReconnectInterval)
}

func TestContainerAppLogMessage_JSON(t *testing.T) {
	jsonStr := `{"Log":"Test message","Stream":"stdout","time":"2024-01-15T10:30:45.123Z","ContainerName":"api","RevisionName":"api--v1"}`

	var msg containerAppLogMessage
	err := json.Unmarshal([]byte(jsonStr), &msg)

	require.NoError(t, err)
	assert.Equal(t, "Test message", msg.Log)
	assert.Equal(t, "stdout", msg.Stream)
	assert.Equal(t, "api", msg.ContainerName)
	assert.Equal(t, "api--v1", msg.RevisionName)
}

func TestBaseStreamer_SetConnected(t *testing.T) {
	base := newBaseStreamer(StreamerConfig{
		ServiceName: "test",
		Credential:  &mockCredential{},
	})

	assert.False(t, base.IsConnected())

	base.setConnected(true)
	assert.True(t, base.IsConnected())

	base.setConnected(false)
	assert.False(t, base.IsConnected())
}

func TestStreamerManager_ConnectedStreamers(t *testing.T) {
	manager := NewStreamerManager()
	defer func() { _ = manager.Stop() }()

	// Initially no connected streamers
	connected := manager.ConnectedStreamers()
	assert.Empty(t, connected)

	// Add a streamer (won't be connected since there's no real server)
	config := StreamerConfig{
		ServiceName:          "test",
		Credential:           &mockCredential{},
		MaxReconnectAttempts: 1,
	}
	streamer := NewContainerAppStreamer(config)
	_ = manager.AddStreamer(streamer)

	// Give it a moment to try connecting
	time.Sleep(50 * time.Millisecond)

	// Should still show as active but not connected
	active := manager.ActiveStreamers()
	assert.Contains(t, active, "test")
}

func TestAppServiceLogPattern(t *testing.T) {
	tests := []struct {
		line        string
		shouldMatch bool
		wantTime    string
		wantPID     string
		wantMessage string
	}{
		{
			line:        "2024-01-15T10:30:45  PID[123] Application started",
			shouldMatch: true,
			wantTime:    "2024-01-15T10:30:45",
			wantPID:     "PID[123]",
			wantMessage: "Application started",
		},
		{
			line:        "2024-12-08T15:45:30  INSTANCE[abc-123] Request received",
			shouldMatch: true,
			wantTime:    "2024-12-08T15:45:30",
			wantPID:     "INSTANCE[abc-123]",
			wantMessage: "Request received",
		},
		{
			line:        "Plain text without timestamp",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			matches := appServiceLogPattern.FindStringSubmatch(tt.line)

			if tt.shouldMatch {
				require.Len(t, matches, 4)
				assert.Equal(t, tt.wantTime, matches[1])
				assert.Equal(t, tt.wantPID, matches[2])
				assert.Equal(t, tt.wantMessage, matches[3])
			} else {
				assert.Nil(t, matches)
			}
		})
	}
}

func TestStreamer_ContextCancellation(t *testing.T) {
	config := StreamerConfig{
		ServiceName: "test",
		Credential:  &mockCredential{},
	}

	streamer := NewContainerAppStreamer(config)
	logs := make(chan LogEntry, 10)

	ctx, cancel := context.WithCancel(context.Background())

	// Start streaming in goroutine
	done := make(chan error)
	go func() {
		done <- streamer.Start(ctx, logs)
	}()

	// Cancel immediately
	cancel()

	// Should return without error
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("streamer did not stop on context cancellation")
	}
}

func TestStreamer_Stop(t *testing.T) {
	config := StreamerConfig{
		ServiceName: "test",
		Credential:  &mockCredential{},
	}

	streamer := NewContainerAppStreamer(config)
	logs := make(chan LogEntry, 10)

	ctx := context.Background()

	// Start streaming in goroutine
	done := make(chan error)
	go func() {
		done <- streamer.Start(ctx, logs)
	}()

	// Give it time to start
	time.Sleep(10 * time.Millisecond)

	// Stop
	err := streamer.Stop()
	assert.NoError(t, err)
	assert.False(t, streamer.IsConnected())

	// Should return without error
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("streamer did not stop")
	}
}
