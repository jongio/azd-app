package notifications

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/config"
	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/jongio/azd-app/cli/src/internal/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline(t *testing.T) {
	t.Run("PublishAndHandle", func(t *testing.T) {
		pipeline := NewPipeline(10)
		defer func() { _ = pipeline.Stop() }()

		var handled []Event
		var mu sync.Mutex
		handler := &mockHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				mu.Lock()
				handled = append(handled, event)
				mu.Unlock()
				return nil
			},
		}

		pipeline.RegisterHandler(handler)
		pipeline.Start()

		event := Event{
			Type:        EventServiceStateChange,
			ServiceName: "api",
			NewState:    &monitor.ServiceState{Status: "running"},
			Message:     "Service started",
			Severity:    "info",
			Timestamp:   time.Now(),
		}

		err := pipeline.Publish(event)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		require.Len(t, handled, 1)
		assert.Equal(t, "api", handled[0].ServiceName)
		mu.Unlock()
	})

	t.Run("MultipleHandlers", func(t *testing.T) {
		pipeline := NewPipeline(10)
		defer func() { _ = pipeline.Stop() }()

		var count1 atomic.Int32
		var count2 atomic.Int32

		pipeline.RegisterHandler(&mockHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				count1.Add(1)
				return nil
			},
		})

		pipeline.RegisterHandler(&mockHandler{
			handleFunc: func(ctx context.Context, event Event) error {
				count2.Add(1)
				return nil
			},
		})

		pipeline.Start()

		event := Event{
			Type:        EventHealthCheck,
			ServiceName: "web",
			Severity:    "info",
			Timestamp:   time.Now(),
		}

		_ = pipeline.Publish(event)
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, int32(1), count1.Load())
		assert.Equal(t, int32(1), count2.Load())
	})

	t.Run("BufferFull", func(t *testing.T) {
		pipeline := NewPipeline(2)
		defer func() { _ = pipeline.Stop() }()

		// Don't start processing to fill buffer
		event := Event{Type: EventError, Timestamp: time.Now()}

		require.NoError(t, pipeline.Publish(event))
		require.NoError(t, pipeline.Publish(event))

		err := pipeline.Publish(event)
		assert.Error(t, err)
	})
}

func TestOSNotificationHandler(t *testing.T) {
	t.Run("RateLimiting", func(t *testing.T) {
		notifier := &mockNotifier{available: true}
		prefs := config.DefaultNotificationPreferences()
		prefs.SeverityFilter = "all"
		prefs.RateLimitWindow = "1s"

		handler := NewOSNotificationHandler(notifier, prefs)

		event := Event{
			Type:        EventServiceStateChange,
			ServiceName: "api",
			Message:     "State changed",
			Severity:    "warning",
			Timestamp:   time.Now(),
		}

		ctx := context.Background()

		// First notification should go through
		err := handler.Handle(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, 1, notifier.sendCount)

		// Second notification should be rate limited
		err = handler.Handle(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, 1, notifier.sendCount)

		// After rate limit period, should go through
		time.Sleep(1 * time.Second)
		err = handler.Handle(ctx, event)
		require.NoError(t, err)
		assert.Equal(t, 2, notifier.sendCount)
	})
}

type mockHandler struct {
	handleFunc func(context.Context, Event) error
}

func (m *mockHandler) Handle(ctx context.Context, event Event) error {
	return m.handleFunc(ctx, event)
}

type mockNotifier struct {
	available bool
	sendCount int
}

func (m *mockNotifier) Send(ctx context.Context, notification notify.Notification) error {
	m.sendCount++
	return nil
}

func (m *mockNotifier) IsAvailable() bool {
	return m.available
}

func (m *mockNotifier) RequestPermission(ctx context.Context) error {
	return nil
}

func (m *mockNotifier) Close() error {
	return nil
}
