package notifications

import (
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/monitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationManager(t *testing.T) {
	t.Run("NewNotificationManager", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)
		require.NotNil(t, nm)
		assert.NotNil(t, nm.stateMonitor)
		assert.NotNil(t, nm.pipeline)
		assert.NotNil(t, nm.registry)
	})

	t.Run("StartStop", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		nm.Start()
		assert.True(t, nm.started)

		err = nm.Stop()
		require.NoError(t, err)
		assert.False(t, nm.started)
	})

	t.Run("DoubleStart", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		nm.Start()
		nm.Start() // Should be safe to call multiple times
		assert.True(t, nm.started)

		err = nm.Stop()
		require.NoError(t, err)
	})

	t.Run("DoubleStop", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		nm.Start()
		err = nm.Stop()
		require.NoError(t, err)

		err = nm.Stop() // Should be safe to call multiple times
		require.NoError(t, err)
	})

	t.Run("HandleStateTransition", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		nm.Start()
		defer func() { _ = nm.Stop() }()

		// Simulate a state transition
		transition := monitor.StateTransition{
			ServiceName: "test-service",
			FromState: &monitor.ServiceState{
				Name:   "test-service",
				Status: "running",
				Health: "healthy",
			},
			ToState: &monitor.ServiceState{
				Name:   "test-service",
				Status: "error",
				Health: "unhealthy",
			},
			Severity:    monitor.SeverityCritical,
			Description: "Service crashed",
			Timestamp:   time.Now(),
		}

		// Call handleStateTransition directly
		nm.handleStateTransition(transition)

		// Give time for async processing
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("GetHistory", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		history := nm.GetHistory()
		assert.NotNil(t, history)
		assert.Empty(t, history) // Initially empty
	})

	t.Run("IsNotificationsEnabled", func(t *testing.T) {
		cfg := DefaultNotificationManagerConfig(t.TempDir())
		nm, err := NewNotificationManager(cfg)
		require.NoError(t, err)

		// On Windows with PowerShell available, notifications should be enabled
		// On other systems or CI, might be disabled
		// Just verify the method doesn't panic
		_ = nm.IsNotificationsEnabled()
	})
}

func TestDefaultNotificationManagerConfig(t *testing.T) {
	projectDir := "/test/project"
	cfg := DefaultNotificationManagerConfig(projectDir)

	assert.Equal(t, projectDir, cfg.ProjectDir)
	assert.Equal(t, 5*time.Second, cfg.MonitorInterval)
	assert.Equal(t, 100, cfg.BufferSize)
}
