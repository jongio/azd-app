package notifications

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabase(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	defer os.Remove(dbPath)

	db, err := NewDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	t.Run("SaveAndRetrieve", func(t *testing.T) {
		event := Event{
			Type:        EventServiceStateChange,
			ServiceName: "api",
			Message:     "Service started",
			Severity:    "info",
			Timestamp:   time.Now(),
			Metadata:    map[string]interface{}{"port": 8080},
		}

		err := db.Save(ctx, event)
		require.NoError(t, err)

		records, err := db.GetRecent(ctx, 10)
		require.NoError(t, err)
		require.Len(t, records, 1)

		assert.Equal(t, "api", records[0].ServiceName)
		assert.Equal(t, "Service started", records[0].Message)
		assert.False(t, records[0].Read)
	})

	t.Run("GetByService", func(t *testing.T) {
		events := []Event{
			{Type: EventHealthCheck, ServiceName: "web", Message: "Health OK", Severity: "info", Timestamp: time.Now()},
			{Type: EventError, ServiceName: "api", Message: "Error occurred", Severity: "critical", Timestamp: time.Now()},
			{Type: EventHealthCheck, ServiceName: "web", Message: "Health OK 2", Severity: "info", Timestamp: time.Now()},
		}

		for _, e := range events {
			require.NoError(t, db.Save(ctx, e))
		}

		records, err := db.GetByService(ctx, "web", 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(records), 2)

		for _, r := range records {
			assert.Equal(t, "web", r.ServiceName)
		}
	})

	t.Run("MarkAsRead", func(t *testing.T) {
		event := Event{
			Type:        EventDeploymentComplete,
			ServiceName: "deploy",
			Message:     "Deployment successful",
			Severity:    "info",
			Timestamp:   time.Now(),
		}

		require.NoError(t, db.Save(ctx, event))

		records, err := db.GetRecent(ctx, 1)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.False(t, records[0].Read)

		err = db.MarkAsRead(ctx, records[0].ID)
		require.NoError(t, err)

		records, err = db.GetRecent(ctx, 1)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.True(t, records[0].Read)
	})

	t.Run("GetUnread", func(t *testing.T) {
		// Save a few notifications
		for i := 0; i < 3; i++ {
			event := Event{
				Type:        EventHealthCheck,
				ServiceName: "test",
				Message:     "Test message",
				Severity:    "info",
				Timestamp:   time.Now(),
			}
			require.NoError(t, db.Save(ctx, event))
		}

		unread, err := db.GetUnread(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(unread), 3)

		// Mark all as read
		require.NoError(t, db.MarkAllAsRead(ctx))

		unread, err = db.GetUnread(ctx)
		require.NoError(t, err)
		assert.Empty(t, unread)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats, err := db.GetStats(ctx)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, stats["total"], 0)
		assert.GreaterOrEqual(t, stats["unread"], 0)
		assert.GreaterOrEqual(t, stats["critical"], 0)
	})

	t.Run("ClearOld", func(t *testing.T) {
		// Save an old notification
		oldEvent := Event{
			Type:        EventHealthCheck,
			ServiceName: "old",
			Message:     "Old notification",
			Severity:    "info",
			Timestamp:   time.Now().Add(-48 * time.Hour),
		}
		require.NoError(t, db.Save(ctx, oldEvent))

		err := db.ClearOld(ctx, 24*time.Hour)
		require.NoError(t, err)

		// Verify old notification was deleted
		records, err := db.GetByService(ctx, "old", 10)
		require.NoError(t, err)
		assert.Empty(t, records)
	})
}
