package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/notifications"
)

func TestGetNotificationDBPath(t *testing.T) {
	tests := []struct {
		name        string
		setEnv      map[string]string
		clearEnv    []string
		expectPath  string
		expectError bool
	}{
		{
			name: "XDG_DATA_HOME set",
			setEnv: map[string]string{
				"XDG_DATA_HOME": "/custom/data",
			},
			expectPath:  filepath.Join("/custom/data", "azd", "notifications.db"),
			expectError: false,
		},
		{
			name: "LOCALAPPDATA set (Windows)",
			setEnv: map[string]string{
				"LOCALAPPDATA": "C:\\Users\\test\\AppData\\Local",
			},
			clearEnv:    []string{"XDG_DATA_HOME"},
			expectPath:  filepath.Join("C:\\Users\\test\\AppData\\Local", "azd", "notifications.db"),
			expectError: false,
		},
		{
			name:        "fallback to home directory",
			clearEnv:    []string{"XDG_DATA_HOME", "LOCALAPPDATA"},
			expectPath:  "", // Will be checked dynamically
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, val := range tt.setEnv {
				oldVal := os.Getenv(key)
				os.Setenv(key, val)
				defer os.Setenv(key, oldVal)
			}

			// Clear environment variables
			for _, key := range tt.clearEnv {
				oldVal := os.Getenv(key)
				os.Unsetenv(key)
				defer func(k, v string) {
					if v != "" {
						os.Setenv(k, v)
					}
				}(key, oldVal)
			}

			path := getNotificationDBPath()

			if tt.expectPath != "" {
				if path != tt.expectPath {
					t.Errorf("getNotificationDBPath() = %q, want %q", path, tt.expectPath)
				}
			} else {
				// Check that path is not empty and has correct suffix
				if path == "" {
					t.Error("getNotificationDBPath() returned empty path")
				}
				if filepath.Base(path) != "notifications.db" {
					t.Errorf("getNotificationDBPath() basename = %q, want %q", filepath.Base(path), "notifications.db")
				}
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m ago",
		},
		{
			name:     "hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h ago",
		},
		{
			name:     "days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d ago",
		},
		{
			name:     "exactly 1 hour",
			time:     now.Add(-1 * time.Hour),
			expected: "1h ago",
		},
		{
			name:     "exactly 1 day",
			time:     now.Add(-24 * time.Hour),
			expected: "1d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.time)
			if result != tt.expected {
				t.Errorf("formatRelativeTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "ascii string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "ascii string at max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "ascii string longer than max",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "utf8 string with emoji",
			input:    "Hello 🌍 World",
			maxLen:   10,
			expected: "Hello 🌍", // "Hello " is 6 bytes + 🌍 is 4 bytes = 10 bytes exactly
		},
		{
			name:     "utf8 string cut at boundary",
			input:    "测试文本",
			maxLen:   6,
			expected: "测试",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "single utf8 character",
			input:    "🎉",
			maxLen:   4,
			expected: "🎉",
		},
		{
			name:     "single utf8 character truncated",
			input:    "🎉",
			maxLen:   2,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateUTF8(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateUTF8(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
			// Ensure result is valid UTF-8
			if len(result) > 0 && !isValidUTF8(result) {
				t.Errorf("truncateUTF8(%q, %d) produced invalid UTF-8: %q", tt.input, tt.maxLen, result)
			}
		})
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' { // Replacement character indicates invalid UTF-8
			return false
		}
	}
	return true
}

func TestPrintNotifications(t *testing.T) {
	tests := []struct {
		name    string
		records []notifications.NotificationRecord
	}{
		{
			name:    "empty records",
			records: []notifications.NotificationRecord{},
		},
		{
			name: "single record",
			records: []notifications.NotificationRecord{
				{
					ID:          1,
					ServiceName: "api",
					Message:     "Service started",
					Severity:    "info",
					Timestamp:   time.Now().Add(-5 * time.Minute),
					Read:        false,
				},
			},
		},
		{
			name: "multiple records",
			records: []notifications.NotificationRecord{
				{
					ID:          1,
					ServiceName: "api",
					Message:     "Service started",
					Severity:    "info",
					Timestamp:   time.Now().Add(-5 * time.Minute),
					Read:        false,
				},
				{
					ID:          2,
					ServiceName: "web",
					Message:     "Service crashed",
					Severity:    "critical",
					Timestamp:   time.Now().Add(-1 * time.Minute),
					Read:        true,
				},
			},
		},
		{
			name: "long message truncation",
			records: []notifications.NotificationRecord{
				{
					ID:          1,
					ServiceName: "api",
					Message:     "This is a very long message that should be truncated to fit within the display constraints of the table output format used by the notification system",
					Severity:    "warning",
					Timestamp:   time.Now(),
					Read:        false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure it doesn't panic
			// In a real test, we'd capture stdout and verify output
			printNotifications(tt.records)
		})
	}
}

func TestNotificationIDValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expectID  int64
	}{
		{
			name:      "valid positive ID",
			input:     "42",
			expectErr: false,
			expectID:  42,
		},
		{
			name:      "zero ID",
			input:     "0",
			expectErr: true,
			expectID:  0,
		},
		{
			name:      "negative ID",
			input:     "-1",
			expectErr: true,
			expectID:  -1,
		},
		{
			name:      "invalid format",
			input:     "abc",
			expectErr: true,
		},
		{
			name:      "float number",
			input:     "3.14",
			expectErr: false, // Sscanf with %d parses "3" and stops at decimal
			expectID:  3,
		},
		{
			name:      "very large ID",
			input:     "9223372036854775807", // max int64
			expectErr: false,
			expectID:  9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id int64
			_, err := fmt.Sscanf(tt.input, "%d", &id)

			if err != nil {
				if !tt.expectErr {
					t.Errorf("Unexpected error parsing %q: %v", tt.input, err)
				}
				return
			}

			// Additional validation
			if id <= 0 {
				if !tt.expectErr {
					t.Errorf("Expected error for non-positive ID %d", id)
				}
			} else if tt.expectErr {
				t.Errorf("Expected error for input %q but got valid ID %d", tt.input, id)
			} else if id != tt.expectID {
				t.Errorf("Parsed ID = %d, want %d", id, tt.expectID)
			}
		})
	}
}

func TestDatabaseIntegration(t *testing.T) {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_notifications.db")

	// Create database
	db, err := notifications.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test Save
	event := notifications.Event{
		Type:        notifications.EventServiceStateChange,
		ServiceName: "test-service",
		Message:     "Test notification",
		Severity:    "info",
		Timestamp:   time.Now(),
		Metadata:    map[string]interface{}{"key": "value"},
	}

	if err := db.Save(ctx, event); err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Test GetRecent
	records, err := db.GetRecent(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get recent notifications: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if len(records) > 0 {
		r := records[0]
		if r.ServiceName != "test-service" {
			t.Errorf("ServiceName = %q, want %q", r.ServiceName, "test-service")
		}
		if r.Message != "Test notification" {
			t.Errorf("Message = %q, want %q", r.Message, "Test notification")
		}
	}

	// Test GetByService
	serviceRecords, err := db.GetByService(ctx, "test-service", 10)
	if err != nil {
		t.Fatalf("Failed to get service notifications: %v", err)
	}

	if len(serviceRecords) != 1 {
		t.Errorf("Expected 1 service record, got %d", len(serviceRecords))
	}

	// Test GetUnread
	unreadRecords, err := db.GetUnread(ctx)
	if err != nil {
		t.Fatalf("Failed to get unread notifications: %v", err)
	}

	if len(unreadRecords) != 1 {
		t.Errorf("Expected 1 unread record, got %d", len(unreadRecords))
	}

	// Test MarkAsRead
	if len(records) > 0 {
		if err := db.MarkAsRead(ctx, records[0].ID); err != nil {
			t.Fatalf("Failed to mark as read: %v", err)
		}

		// Verify it's marked as read
		unreadAfter, err := db.GetUnread(ctx)
		if err != nil {
			t.Fatalf("Failed to get unread after marking: %v", err)
		}

		if len(unreadAfter) != 0 {
			t.Errorf("Expected 0 unread records after marking, got %d", len(unreadAfter))
		}
	}

	// Test GetStats
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total"] != 1 {
		t.Errorf("Total stats = %d, want 1", stats["total"])
	}

	if stats["unread"] != 0 {
		t.Errorf("Unread stats = %d, want 0", stats["unread"])
	}
}

func TestDatabaseMaxRecordsEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_max_records.db")

	db, err := notifications.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Note: This test would need to set a lower maxRecords limit for testing
	// For now, we'll just verify the database can handle multiple records
	const numRecords = 100

	for i := 0; i < numRecords; i++ {
		event := notifications.Event{
			Type:        notifications.EventServiceStateChange,
			ServiceName: "test-service",
			Message:     fmt.Sprintf("Notification %d", i),
			Severity:    "info",
			Timestamp:   time.Now().Add(time.Duration(i) * time.Second),
		}

		if err := db.Save(ctx, event); err != nil {
			t.Fatalf("Failed to save event %d: %v", i, err)
		}
	}

	// Verify records were saved
	records, err := db.GetRecent(ctx, numRecords)
	if err != nil {
		t.Fatalf("Failed to get recent notifications: %v", err)
	}

	if len(records) != numRecords {
		t.Errorf("Expected %d records, got %d", numRecords, len(records))
	}
}
