package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Database manages notification history persistence
type Database struct {
	db   *sql.DB
	mu   sync.RWMutex
	path string
}

// NotificationRecord represents a persisted notification
type NotificationRecord struct {
	ID           int64                  `json:"id"`
	Type         string                 `json:"type"`
	ServiceName  string                 `json:"serviceName"`
	Message      string                 `json:"message"`
	Severity     string                 `json:"severity"`
	Timestamp    time.Time              `json:"timestamp"`
	Read         bool                   `json:"read"`
	Acknowledged bool                   `json:"acknowledged"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewDatabase creates a new notification database
func NewDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{
		db:   db,
		path: path,
	}

	if err := d.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

// initialize creates database schema
func (d *Database) initialize() error {
	schema := `
		CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			service_name TEXT NOT NULL,
			message TEXT NOT NULL,
			severity TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			read INTEGER DEFAULT 0,
			acknowledged INTEGER DEFAULT 0,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_service_name ON notifications(service_name);
		CREATE INDEX IF NOT EXISTS idx_timestamp ON notifications(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_read ON notifications(read);
		CREATE INDEX IF NOT EXISTS idx_severity ON notifications(severity);
	`

	_, err := d.db.Exec(schema)
	return err
}

// Save stores a notification event
func (d *Database) Save(ctx context.Context, event Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO notifications (type, service_name, message, severity, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.ExecContext(ctx, query,
		event.Type,
		event.ServiceName,
		event.Message,
		event.Severity,
		event.Timestamp,
		string(metadata),
	)

	return err
}

// GetRecent retrieves recent notifications
func (d *Database) GetRecent(ctx context.Context, limit int) ([]NotificationRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, type, service_name, message, severity, timestamp, read, acknowledged, metadata
		FROM notifications
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanRecords(rows)
}

// GetByService retrieves notifications for a specific service
func (d *Database) GetByService(ctx context.Context, serviceName string, limit int) ([]NotificationRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, type, service_name, message, severity, timestamp, read, acknowledged, metadata
		FROM notifications
		WHERE service_name = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.db.QueryContext(ctx, query, serviceName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanRecords(rows)
}

// GetUnread retrieves unread notifications
func (d *Database) GetUnread(ctx context.Context) ([]NotificationRecord, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, type, service_name, message, severity, timestamp, read, acknowledged, metadata
		FROM notifications
		WHERE read = 0
		ORDER BY timestamp DESC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.scanRecords(rows)
}

// MarkAsRead marks a notification as read
func (d *Database) MarkAsRead(ctx context.Context, id int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE notifications SET read = 1 WHERE id = ?`
	_, err := d.db.ExecContext(ctx, query, id)
	return err
}

// MarkAllAsRead marks all notifications as read
func (d *Database) MarkAllAsRead(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `UPDATE notifications SET read = 1 WHERE read = 0`
	_, err := d.db.ExecContext(ctx, query)
	return err
}

// ClearAll deletes all notifications
func (d *Database) ClearAll(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM notifications`
	_, err := d.db.ExecContext(ctx, query)
	return err
}

// ClearOld deletes notifications older than the specified duration
func (d *Database) ClearOld(ctx context.Context, olderThan time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM notifications WHERE timestamp < ?`
	_, err := d.db.ExecContext(ctx, query, cutoff)
	return err
}

// GetStats returns notification statistics
func (d *Database) GetStats(ctx context.Context) (map[string]int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[string]int)

	// Total count
	var total int
	err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications`).Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	// Unread count
	var unread int
	err = d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE read = 0`).Scan(&unread)
	if err != nil {
		return nil, err
	}
	stats["unread"] = unread

	// Critical count
	var critical int
	err = d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE severity = 'critical'`).Scan(&critical)
	if err != nil {
		return nil, err
	}
	stats["critical"] = critical

	return stats, nil
}

// scanRecords converts SQL rows to NotificationRecord slice
func (d *Database) scanRecords(rows *sql.Rows) ([]NotificationRecord, error) {
	records := make([]NotificationRecord, 0)

	for rows.Next() {
		var r NotificationRecord
		var metadataJSON sql.NullString
		var readInt, ackInt int

		err := rows.Scan(
			&r.ID,
			&r.Type,
			&r.ServiceName,
			&r.Message,
			&r.Severity,
			&r.Timestamp,
			&readInt,
			&ackInt,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		r.Read = readInt == 1
		r.Acknowledged = ackInt == 1

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &r.Metadata); err != nil {
				return nil, err
			}
		}

		records = append(records, r)
	}

	return records, rows.Err()
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db.Close()
}
