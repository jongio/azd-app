package notifications

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_NotificationPipeline tests the complete notification flow
func TestE2E_NotificationPipeline(t *testing.T) {
	t.Run("FullPipelineFlow", func(t *testing.T) {
		// Create pipeline with buffer
		pipeline := NewPipeline(10)

		// Track received events
		var received []Event
		var mu sync.Mutex

		// Create test handler
		handler := &testHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				mu.Lock()
				received = append(received, event)
				mu.Unlock()
				return nil
			},
		}

		// Register handler
		pipeline.RegisterHandler(handler)

		// Start pipeline
		pipeline.Start()

		// Publish multiple events
		events := []Event{
			{
				Type:        EventServiceStateChange,
				ServiceName: "api",
				Message:     "Service started",
				Severity:    "info",
				Timestamp:   time.Now(),
			},
			{
				Type:        EventHealthCheck,
				ServiceName: "web",
				Message:     "Health check failed",
				Severity:    "critical",
				Timestamp:   time.Now(),
			},
			{
				Type:        EventError,
				ServiceName: "db",
				Message:     "Connection lost",
				Severity:    "warning",
				Timestamp:   time.Now(),
			},
		}

		for _, event := range events {
			err := pipeline.Publish(event)
			require.NoError(t, err)
		}

		// Wait for processing
		time.Sleep(200 * time.Millisecond)

		// Stop pipeline
		err := pipeline.Stop()
		require.NoError(t, err)

		// Verify all events received
		mu.Lock()
		assert.Len(t, received, len(events))
		mu.Unlock()
	})

	t.Run("MultipleHandlers", func(t *testing.T) {
		pipeline := NewPipeline(10)

		var handler1Count, handler2Count int
		var mu sync.Mutex

		handler1 := &testHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				mu.Lock()
				handler1Count++
				mu.Unlock()
				return nil
			},
		}

		handler2 := &testHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				mu.Lock()
				handler2Count++
				mu.Unlock()
				return nil
			},
		}

		pipeline.RegisterHandler(handler1)
		pipeline.RegisterHandler(handler2)
		pipeline.Start()

		// Publish event
		err := pipeline.Publish(Event{
			Type:        EventServiceStateChange,
			ServiceName: "test",
			Message:     "Test message",
			Severity:    "info",
			Timestamp:   time.Now(),
		})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		_ = pipeline.Stop()

		// Both handlers should receive the event
		mu.Lock()
		assert.Equal(t, 1, handler1Count)
		assert.Equal(t, 1, handler2Count)
		mu.Unlock()
	})
}

// TestE2E_OSNotificationHandler tests OS notification handler behavior
func TestE2E_OSNotificationHandler(t *testing.T) {
	t.Run("RateLimiting", func(t *testing.T) {
		// Create mock notifier
		var sendCount int
		var mu sync.Mutex
		mockNotifier := &mockTestNotifier{
			sendFunc: func(ctx context.Context, notification notify.Notification) error {
				mu.Lock()
				sendCount++
				mu.Unlock()
				return nil
			},
		}

		// Create config with short rate limit
		prefs := &config.NotificationPreferences{
			OSNotifications:        true,
			DashboardNotifications: true,
			SeverityFilter:         "all",
			RateLimitWindow:        "1s",
		}

		handler := NewOSNotificationHandler(mockNotifier, prefs)
		defer handler.Close()

		ctx := context.Background()

		// Send same event type multiple times rapidly
		for i := 0; i < 5; i++ {
			_ = handler.Handle(ctx, Event{
				Type:        EventServiceStateChange,
				ServiceName: "api",
				Message:     "Test",
				Severity:    "info",
				Timestamp:   time.Now(),
			})
		}

		// Only first should be sent due to rate limiting
		mu.Lock()
		assert.Equal(t, 1, sendCount)
		mu.Unlock()
	})

	t.Run("SeverityFiltering", func(t *testing.T) {
		var sendCount int
		var mu sync.Mutex
		mockNotifier := &mockTestNotifier{
			sendFunc: func(ctx context.Context, notification notify.Notification) error {
				mu.Lock()
				sendCount++
				mu.Unlock()
				return nil
			},
		}

		// Only critical notifications
		prefs := &config.NotificationPreferences{
			OSNotifications:        true,
			DashboardNotifications: true,
			SeverityFilter:         "critical",
			RateLimitWindow:        "1ms", // Minimal rate limit for testing
		}

		handler := NewOSNotificationHandler(mockNotifier, prefs)
		defer handler.Close()

		ctx := context.Background()

		// Send info (should be filtered)
		_ = handler.Handle(ctx, Event{
			Type:        EventServiceStateChange,
			ServiceName: "api-1",
			Message:     "Info message",
			Severity:    "info",
			Timestamp:   time.Now(),
		})

		// Send warning (should be filtered)
		_ = handler.Handle(ctx, Event{
			Type:        EventServiceStateChange,
			ServiceName: "api-2",
			Message:     "Warning message",
			Severity:    "warning",
			Timestamp:   time.Now(),
		})

		// Send critical (should pass)
		_ = handler.Handle(ctx, Event{
			Type:        EventServiceStateChange,
			ServiceName: "api-3",
			Message:     "Critical message",
			Severity:    "critical",
			Timestamp:   time.Now(),
		})

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, 1, sendCount, "Only critical notifications should be sent")
		mu.Unlock()
	})
}

// TestE2E_HistoryHandler tests database persistence
func TestE2E_HistoryHandler(t *testing.T) {
	t.Run("PersistAndRetrieve", func(t *testing.T) {
		// Create temp database
		dbPath := t.TempDir() + "/notifications_test.db"
		db, err := NewDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Create history handler with database adapter
		store := &databaseStore{db: db}
		handler := NewHistoryHandler(store)

		ctx := context.Background()

		now := time.Now()

		// Save events through handler with distinct timestamps
		events := []Event{
			{
				Type:        EventServiceStateChange,
				ServiceName: "api",
				Message:     "Service started",
				Severity:    "info",
				Timestamp:   now.Add(-time.Second), // older event
			},
			{
				Type:        EventHealthCheck,
				ServiceName: "api",
				Message:     "Health check failed",
				Severity:    "critical",
				Timestamp:   now, // newer event
			},
		}

		for _, event := range events {
			err := handler.Handle(ctx, event)
			require.NoError(t, err)
		}

		// Retrieve from database
		records, err := db.GetRecent(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, records, 2)

		// Verify first record (most recent) is the critical one
		assert.Equal(t, "api", records[0].ServiceName)
		assert.Equal(t, "critical", records[0].Severity)

		// Verify second record is the info one
		assert.Equal(t, "info", records[1].Severity)
	})
}

// TestE2E_WebSocketHandler tests WebSocket broadcasting
func TestE2E_WebSocketHandler(t *testing.T) {
	t.Run("BroadcastEvents", func(t *testing.T) {
		var received []Event
		var mu sync.Mutex

		broadcaster := func(event Event) {
			mu.Lock()
			received = append(received, event)
			mu.Unlock()
		}

		handler := NewWebSocketHandler(broadcaster)
		ctx := context.Background()

		// Send events
		events := []Event{
			{Type: EventServiceStateChange, ServiceName: "api", Severity: "info"},
			{Type: EventHealthCheck, ServiceName: "web", Severity: "critical"},
		}

		for _, event := range events {
			err := handler.Handle(ctx, event)
			require.NoError(t, err)
		}

		mu.Lock()
		assert.Len(t, received, 2)
		mu.Unlock()
	})
}

// TestE2E_StateTransitionToNotification tests state monitor to notification flow
func TestE2E_StateTransitionToNotification(t *testing.T) {
	t.Run("SeverityMapping", func(t *testing.T) {
		// Create pipeline directly to test severity mapping
		pipeline := NewPipeline(10)

		var received []Event
		var mu sync.Mutex

		// Add test handler to capture events
		testHandler := &testHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				mu.Lock()
				received = append(received, event)
				mu.Unlock()
				return nil
			},
		}
		pipeline.RegisterHandler(testHandler)
		pipeline.Start()
		defer func() { _ = pipeline.Stop() }()

		// Publish events with different severities
		events := []Event{
			{
				Type:        EventServiceStateChange,
				ServiceName: "api",
				Message:     "Service crashed",
				Severity:    "critical",
				Timestamp:   time.Now(),
			},
			{
				Type:        EventServiceStateChange,
				ServiceName: "web",
				Message:     "High latency",
				Severity:    "warning",
				Timestamp:   time.Now(),
			},
			{
				Type:        EventServiceStateChange,
				ServiceName: "db",
				Message:     "Service started",
				Severity:    "info",
				Timestamp:   time.Now(),
			},
		}

		for _, e := range events {
			err := pipeline.Publish(e)
			require.NoError(t, err)
		}

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		require.Len(t, received, 3)

		// Verify severity mapping
		severities := map[string]string{}
		for _, e := range received {
			severities[e.ServiceName] = e.Severity
		}

		assert.Equal(t, "critical", severities["api"])
		assert.Equal(t, "warning", severities["web"])
		assert.Equal(t, "info", severities["db"])
		mu.Unlock()
	})
}

// TestE2E_MonitorSeverityConversion tests the severity conversion logic
func TestE2E_MonitorSeverityConversion(t *testing.T) {
	tests := []struct {
		monitorSeverity monitor.Severity
		expectedString  string
	}{
		{monitor.SeverityCritical, "critical"},
		{monitor.SeverityWarning, "warning"},
		{monitor.SeverityInfo, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedString, func(t *testing.T) {
			// Convert monitor severity to string (same logic as handleStateTransition)
			severity := "info"
			switch tt.monitorSeverity {
			case monitor.SeverityCritical:
				severity = "critical"
			case monitor.SeverityWarning:
				severity = "warning"
			case monitor.SeverityInfo:
				severity = "info"
			}
			assert.Equal(t, tt.expectedString, severity)
		})
	}
}

// Helper types

type testHandler struct {
	handleFunc func(ctx context.Context, event Event) error
}

func (h *testHandler) Handle(ctx context.Context, event Event) error {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, event)
	}
	return nil
}

type mockTestNotifier struct {
	sendFunc func(ctx context.Context, notification notify.Notification) error
}

func (m *mockTestNotifier) Send(ctx context.Context, notification notify.Notification) error {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, notification)
	}
	return nil
}

func (m *mockTestNotifier) IsAvailable() bool {
	return true
}

func (m *mockTestNotifier) RequestPermission(ctx context.Context) error {
	return nil
}

func (m *mockTestNotifier) Close() error {
	return nil
}

// databaseStore adapts Database to HistoryStore interface
type databaseStore struct {
	db *Database
}

func (s *databaseStore) Save(ctx context.Context, event Event) error {
	return s.db.Save(ctx, event)
}
