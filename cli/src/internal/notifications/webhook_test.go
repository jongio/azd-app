package notifications

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func webhookTestPrefs(url string) *config.NotificationPreferences {
	prefs := config.DefaultNotificationPreferences()
	prefs.WebhookEnabled = true
	prefs.WebhookURL = url
	prefs.SeverityFilter = "all"
	prefs.RateLimitWindow = "5m"
	return prefs
}

func webhookTestEvent() Event {
	return Event{
		Type:        EventServiceStateChange,
		ServiceName: "api",
		Severity:    "warning",
		Message:     "service became unhealthy",
		Timestamp:   time.Now(),
		OldState:    &monitor.ServiceState{Status: "running", Health: "healthy", PID: 42, Port: 8080},
		NewState:    &monitor.ServiceState{Status: "running", Health: "unhealthy", PID: 42, Port: 8080},
	}
}

func TestNewWebhookHandlerDisabled(t *testing.T) {
	// Not enabled
	prefs := config.DefaultNotificationPreferences()
	assert.Nil(t, NewWebhookHandler(prefs), "handler should be nil when webhook is not enabled")

	// Enabled but no URL
	prefs2 := config.DefaultNotificationPreferences()
	prefs2.WebhookEnabled = true
	assert.Nil(t, NewWebhookHandler(prefs2), "handler should be nil when webhook URL is empty")

	// Nil config
	assert.Nil(t, NewWebhookHandler(nil), "handler should be nil for nil config")
}

func TestWebhookHandlerPostsPayload(t *testing.T) {
	var received atomic.Int64
	var body []byte
	var contentType, customHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		contentType = r.Header.Get("Content-Type")
		customHeader = r.Header.Get("X-Token")
		b, _ := io.ReadAll(r.Body)
		body = b
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	prefs := webhookTestPrefs(server.URL)
	prefs.WebhookHeaders = map[string]string{"X-Token": "secret123"}

	h := NewWebhookHandler(prefs)
	require.NotNil(t, h)

	event := webhookTestEvent()
	require.NoError(t, h.Handle(context.Background(), event))

	require.Equal(t, int64(1), received.Load())
	assert.Equal(t, "application/json", contentType)
	assert.Equal(t, "secret123", customHeader)

	var payload WebhookEventPayload
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, string(EventServiceStateChange), payload.Type)
	assert.Equal(t, "api", payload.Service)
	assert.Equal(t, "warning", payload.Severity)
	assert.Equal(t, "service became unhealthy", payload.Message)
	assert.False(t, payload.Timestamp.IsZero())
	require.NotNil(t, payload.OldState)
	require.NotNil(t, payload.NewState)
	assert.Equal(t, "healthy", payload.OldState.Health)
	assert.Equal(t, "unhealthy", payload.NewState.Health)
}

func TestWebhookHandlerSeverityFilter(t *testing.T) {
	var received atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	prefs := webhookTestPrefs(server.URL)
	prefs.SeverityFilter = "critical" // only critical qualifies

	h := NewWebhookHandler(prefs)
	require.NotNil(t, h)

	event := webhookTestEvent() // severity "warning"
	require.NoError(t, h.Handle(context.Background(), event))

	assert.Equal(t, int64(0), received.Load(), "warning event should be filtered out by critical-only filter")
}

func TestWebhookHandlerRateLimit(t *testing.T) {
	var received atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	prefs := webhookTestPrefs(server.URL)
	prefs.RateLimitWindow = "1h" // effectively dedupe within the test

	h := NewWebhookHandler(prefs)
	require.NotNil(t, h)

	event := webhookTestEvent()
	require.NoError(t, h.Handle(context.Background(), event))
	require.NoError(t, h.Handle(context.Background(), event))

	assert.Equal(t, int64(1), received.Load(), "duplicate event within the window should be sent once")
}

func TestWebhookHandlerFailSoft(t *testing.T) {
	// Server that returns an error status.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prefs := webhookTestPrefs(server.URL)
	h := NewWebhookHandler(prefs)
	require.NotNil(t, h)

	// Should return an error but not panic.
	assert.NotPanics(t, func() {
		err := h.Handle(context.Background(), webhookTestEvent())
		assert.Error(t, err)
	})
}

func TestWebhookHandlerUnreachableDoesNotPanic(t *testing.T) {
	prefs := webhookTestPrefs("http://127.0.0.1:0") // invalid port, connection fails fast
	h := NewWebhookHandler(prefs)
	require.NotNil(t, h)

	assert.NotPanics(t, func() {
		err := h.Handle(context.Background(), webhookTestEvent())
		assert.Error(t, err)
	})
}

func TestWebhookConfigValidation(t *testing.T) {
	// Enabled with a valid URL passes.
	prefs := config.DefaultNotificationPreferences()
	prefs.WebhookEnabled = true
	prefs.WebhookURL = "https://example.com/hook"
	require.NoError(t, prefs.Validate())

	// Enabled with empty URL fails.
	prefs.WebhookURL = ""
	require.Error(t, prefs.Validate())

	// Enabled with a non-http scheme fails.
	prefs.WebhookURL = "ftp://example.com/hook"
	require.Error(t, prefs.Validate())

	// Disabled with empty URL is fine.
	prefs.WebhookEnabled = false
	prefs.WebhookURL = ""
	require.NoError(t, prefs.Validate())
}
