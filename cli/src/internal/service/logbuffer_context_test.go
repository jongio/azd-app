package service

import (
	"testing"
	"time"
)

func TestLogBuffer_GetLogsWithContext(t *testing.T) {
	tests := []struct {
		name         string
		entries      []LogEntry
		level        LogLevel
		limit        int
		contextLines int
		since        time.Time
		wantCount    int
		wantFirst    string // First message (most recent)
	}{
		{
			name: "finds entries by error level",
			entries: []LogEntry{
				{Message: "info message", Level: LogLevelInfo, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "error message", Level: LogLevelError, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "another info", Level: LogLevelInfo, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelError,
			limit:        50,
			contextLines: 0,
			wantCount:    1,
			wantFirst:    "error message",
		},
		{
			name: "finds entries by warn level",
			entries: []LogEntry{
				{Message: "info message", Level: LogLevelInfo, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "warning message", Level: LogLevelWarn, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "error message", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelWarn,
			limit:        50,
			contextLines: 0,
			wantCount:    1,
			wantFirst:    "warning message",
		},
		{
			name: "finds entries by info level",
			entries: []LogEntry{
				{Message: "info message 1", Level: LogLevelInfo, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "warning message", Level: LogLevelWarn, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "info message 2", Level: LogLevelInfo, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelInfo,
			limit:        50,
			contextLines: 0,
			wantCount:    2,
			wantFirst:    "info message 2",
		},
		{
			name: "finds entries by debug level",
			entries: []LogEntry{
				{Message: "debug message", Level: LogLevelDebug, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "info message", Level: LogLevelInfo, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "another debug", Level: LogLevelDebug, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelDebug,
			limit:        50,
			contextLines: 0,
			wantCount:    2,
			wantFirst:    "another debug",
		},
		{
			name: "applies limit",
			entries: []LogEntry{
				{Message: "error 1", Level: LogLevelError, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "error 2", Level: LogLevelError, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "error 3", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelError,
			limit:        2,
			contextLines: 0,
			wantCount:    2,
			wantFirst:    "error 3", // Most recent first
		},
		{
			name: "returns entries in reverse chronological order",
			entries: []LogEntry{
				{Message: "old entry", Level: LogLevelError, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "new entry", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			level:        LogLevelError,
			limit:        50,
			contextLines: 0,
			wantCount:    2,
			wantFirst:    "new entry",
		},
		{
			name: "filters by since time",
			entries: []LogEntry{
				{Message: "old entry", Level: LogLevelError, Timestamp: time.Now().Add(-10 * time.Minute)},
				{Message: "recent entry", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Minute)},
			},
			level:        LogLevelError,
			limit:        50,
			contextLines: 0,
			since:        time.Now().Add(-5 * time.Minute),
			wantCount:    1,
			wantFirst:    "recent entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &LogBuffer{
				entries: tt.entries,
				maxSize: 1000,
			}

			entries := lb.GetLogsWithContext(tt.level, tt.limit, tt.contextLines, tt.since)

			if len(entries) != tt.wantCount {
				t.Errorf("got %d entries, want %d", len(entries), tt.wantCount)
			}

			if tt.wantCount > 0 && entries[0].Message != tt.wantFirst {
				t.Errorf("first entry message = %q, want %q", entries[0].Message, tt.wantFirst)
			}
		})
	}
}

func TestLogBuffer_GetLogsWithContext_Context(t *testing.T) {
	now := time.Now()
	entries := []LogEntry{
		{Message: "context before 2", Level: LogLevelInfo, Timestamp: now.Add(-5 * time.Second)},
		{Message: "context before 1", Level: LogLevelInfo, Timestamp: now.Add(-4 * time.Second)},
		{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-3 * time.Second)},
		{Message: "context after 1", Level: LogLevelInfo, Timestamp: now.Add(-2 * time.Second)},
		{Message: "context after 2", Level: LogLevelInfo, Timestamp: now.Add(-1 * time.Second)},
	}

	lb := &LogBuffer{
		entries: entries,
		maxSize: 1000,
	}

	t.Run("extracts context lines", func(t *testing.T) {
		results := lb.GetLogsWithContext(LogLevelError, 50, 2, time.Time{})

		if len(results) != 1 {
			t.Fatalf("got %d entries, want 1", len(results))
		}

		entry := results[0]

		// Check before context
		if len(entry.Context.Before) != 2 {
			t.Errorf("got %d before context lines, want 2", len(entry.Context.Before))
		}
		if len(entry.Context.Before) >= 2 {
			if entry.Context.Before[0] != "context before 2" {
				t.Errorf("before[0] = %q, want 'context before 2'", entry.Context.Before[0])
			}
			if entry.Context.Before[1] != "context before 1" {
				t.Errorf("before[1] = %q, want 'context before 1'", entry.Context.Before[1])
			}
		}

		// Check after context
		if len(entry.Context.After) != 2 {
			t.Errorf("got %d after context lines, want 2", len(entry.Context.After))
		}
		if len(entry.Context.After) >= 2 {
			if entry.Context.After[0] != "context after 1" {
				t.Errorf("after[0] = %q, want 'context after 1'", entry.Context.After[0])
			}
			if entry.Context.After[1] != "context after 2" {
				t.Errorf("after[1] = %q, want 'context after 2'", entry.Context.After[1])
			}
		}
	})

	t.Run("handles edge cases at start of buffer", func(t *testing.T) {
		edgeEntries := []LogEntry{
			{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-2 * time.Second)},
			{Message: "after", Level: LogLevelInfo, Timestamp: now.Add(-1 * time.Second)},
		}
		edgeLb := &LogBuffer{entries: edgeEntries, maxSize: 1000}

		results := edgeLb.GetLogsWithContext(LogLevelError, 50, 3, time.Time{})

		if len(results) != 1 {
			t.Fatalf("got %d entries, want 1", len(results))
		}

		// Should have no before context (entry is at start)
		if len(results[0].Context.Before) != 0 {
			t.Errorf("got %d before context lines, want 0", len(results[0].Context.Before))
		}

		// Should have 1 after context line
		if len(results[0].Context.After) != 1 {
			t.Errorf("got %d after context lines, want 1", len(results[0].Context.After))
		}
	})

	t.Run("handles edge cases at end of buffer", func(t *testing.T) {
		edgeEntries := []LogEntry{
			{Message: "before", Level: LogLevelInfo, Timestamp: now.Add(-2 * time.Second)},
			{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-1 * time.Second)},
		}
		edgeLb := &LogBuffer{entries: edgeEntries, maxSize: 1000}

		results := edgeLb.GetLogsWithContext(LogLevelError, 50, 3, time.Time{})

		if len(results) != 1 {
			t.Fatalf("got %d entries, want 1", len(results))
		}

		// Should have 1 before context line
		if len(results[0].Context.Before) != 1 {
			t.Errorf("got %d before context lines, want 1", len(results[0].Context.Before))
		}

		// Should have no after context (entry is at end)
		if len(results[0].Context.After) != 0 {
			t.Errorf("got %d after context lines, want 0", len(results[0].Context.After))
		}
	})

	t.Run("contextLines=0 returns no context", func(t *testing.T) {
		results := lb.GetLogsWithContext(LogLevelError, 50, 0, time.Time{})

		if len(results) != 1 {
			t.Fatalf("got %d entries, want 1", len(results))
		}

		if len(results[0].Context.Before) != 0 {
			t.Errorf("got %d before context lines, want 0", len(results[0].Context.Before))
		}
		if len(results[0].Context.After) != 0 {
			t.Errorf("got %d after context lines, want 0", len(results[0].Context.After))
		}
	})

	t.Run("clamps contextLines to max 10", func(t *testing.T) {
		// Create buffer with 15 entries before and after the error
		manyEntries := make([]LogEntry, 31)
		for i := 0; i < 15; i++ {
			manyEntries[i] = LogEntry{
				Message:   "before",
				Level:     LogLevelInfo,
				Timestamp: now.Add(time.Duration(-30+i) * time.Second),
			}
		}
		manyEntries[15] = LogEntry{
			Message:   "THE ERROR",
			Level:     LogLevelError,
			Timestamp: now.Add(-15 * time.Second),
		}
		for i := 16; i < 31; i++ {
			manyEntries[i] = LogEntry{
				Message:   "after",
				Level:     LogLevelInfo,
				Timestamp: now.Add(time.Duration(-30+i) * time.Second),
			}
		}

		manyLb := &LogBuffer{entries: manyEntries, maxSize: 1000}

		// Request 20 context lines, should be clamped to 10
		results := manyLb.GetLogsWithContext(LogLevelError, 50, 20, time.Time{})

		if len(results) != 1 {
			t.Fatalf("got %d entries, want 1", len(results))
		}

		if len(results[0].Context.Before) != 10 {
			t.Errorf("got %d before context lines, want 10 (clamped)", len(results[0].Context.Before))
		}
		if len(results[0].Context.After) != 10 {
			t.Errorf("got %d after context lines, want 10 (clamped)", len(results[0].Context.After))
		}
	})

	t.Run("handles consecutive errors", func(t *testing.T) {
		// Test that consecutive errors each get their own context
		entries := []LogEntry{
			{Message: "before 1", Level: LogLevelInfo, Timestamp: now.Add(-5 * time.Second)},
			{Message: "ERROR 1", Level: LogLevelError, Timestamp: now.Add(-4 * time.Second)},
			{Message: "ERROR 2", Level: LogLevelError, Timestamp: now.Add(-3 * time.Second)},
			{Message: "after 1", Level: LogLevelInfo, Timestamp: now.Add(-2 * time.Second)},
		}
		lb := &LogBuffer{entries: entries, maxSize: 1000}

		results := lb.GetLogsWithContext(LogLevelError, 50, 2, time.Time{})

		if len(results) != 2 {
			t.Fatalf("got %d entries, want 2", len(results))
		}

		// Most recent error first (ERROR 2)
		if results[0].Message != "ERROR 2" {
			t.Errorf("first result = %q, want 'ERROR 2'", results[0].Message)
		}

		// Second error (ERROR 1)
		if results[1].Message != "ERROR 1" {
			t.Errorf("second result = %q, want 'ERROR 1'", results[1].Message)
		}
	})
}

// TestLogBuffer_GetErrors tests the deprecated GetErrors method for backward compatibility.
func TestLogBuffer_GetErrors(t *testing.T) {
	tests := []struct {
		name          string
		entries       []LogEntry
		limit         int
		contextLines  int
		includeStderr bool
		since         time.Time
		wantCount     int
		wantFirst     string // First error message (most recent)
	}{
		{
			name: "finds error by level",
			entries: []LogEntry{
				{Message: "info message", Level: LogLevelInfo, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "error message", Level: LogLevelError, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "another info", Level: LogLevelInfo, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			limit:         50,
			contextLines:  0,
			includeStderr: false,
			wantCount:     1,
			wantFirst:     "error message",
		},
		{
			name: "finds stderr when includeStderr is true",
			entries: []LogEntry{
				{Message: "stdout message", Level: LogLevelInfo, IsStderr: false, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "stderr message", Level: LogLevelInfo, IsStderr: true, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			limit:         50,
			contextLines:  0,
			includeStderr: true,
			wantCount:     1,
			wantFirst:     "stderr message",
		},
		{
			name: "excludes stderr when includeStderr is false",
			entries: []LogEntry{
				{Message: "stdout message", Level: LogLevelInfo, IsStderr: false, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "stderr message", Level: LogLevelInfo, IsStderr: true, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			limit:         50,
			contextLines:  0,
			includeStderr: false,
			wantCount:     0,
		},
		{
			name: "applies limit",
			entries: []LogEntry{
				{Message: "error 1", Level: LogLevelError, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "error 2", Level: LogLevelError, Timestamp: time.Now().Add(-2 * time.Second)},
				{Message: "error 3", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			limit:         2,
			contextLines:  0,
			includeStderr: false,
			wantCount:     2,
			wantFirst:     "error 3", // Most recent first
		},
		{
			name: "returns errors in reverse chronological order",
			entries: []LogEntry{
				{Message: "old error", Level: LogLevelError, Timestamp: time.Now().Add(-3 * time.Second)},
				{Message: "new error", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Second)},
			},
			limit:         50,
			contextLines:  0,
			includeStderr: false,
			wantCount:     2,
			wantFirst:     "new error",
		},
		{
			name: "filters by since time",
			entries: []LogEntry{
				{Message: "old error", Level: LogLevelError, Timestamp: time.Now().Add(-10 * time.Minute)},
				{Message: "recent error", Level: LogLevelError, Timestamp: time.Now().Add(-1 * time.Minute)},
			},
			limit:         50,
			contextLines:  0,
			includeStderr: false,
			since:         time.Now().Add(-5 * time.Minute),
			wantCount:     1,
			wantFirst:     "recent error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := &LogBuffer{
				entries: tt.entries,
				maxSize: 1000,
			}

			errors := lb.GetErrors(tt.limit, tt.contextLines, tt.includeStderr, tt.since)

			if len(errors) != tt.wantCount {
				t.Errorf("got %d errors, want %d", len(errors), tt.wantCount)
			}

			if tt.wantCount > 0 && errors[0].Message != tt.wantFirst {
				t.Errorf("first error message = %q, want %q", errors[0].Message, tt.wantFirst)
			}
		})
	}
}

// TestLogBuffer_GetErrors_Context tests the deprecated GetErrors method context extraction.
func TestLogBuffer_GetErrors_Context(t *testing.T) {
	now := time.Now()
	entries := []LogEntry{
		{Message: "context before 2", Level: LogLevelInfo, Timestamp: now.Add(-5 * time.Second)},
		{Message: "context before 1", Level: LogLevelInfo, Timestamp: now.Add(-4 * time.Second)},
		{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-3 * time.Second)},
		{Message: "context after 1", Level: LogLevelInfo, Timestamp: now.Add(-2 * time.Second)},
		{Message: "context after 2", Level: LogLevelInfo, Timestamp: now.Add(-1 * time.Second)},
	}

	lb := &LogBuffer{
		entries: entries,
		maxSize: 1000,
	}

	t.Run("extracts context lines", func(t *testing.T) {
		errors := lb.GetErrors(50, 2, false, time.Time{})

		if len(errors) != 1 {
			t.Fatalf("got %d errors, want 1", len(errors))
		}

		err := errors[0]

		// Check before context
		if len(err.Context.Before) != 2 {
			t.Errorf("got %d before context lines, want 2", len(err.Context.Before))
		}
		if len(err.Context.Before) >= 2 {
			if err.Context.Before[0] != "context before 2" {
				t.Errorf("before[0] = %q, want 'context before 2'", err.Context.Before[0])
			}
			if err.Context.Before[1] != "context before 1" {
				t.Errorf("before[1] = %q, want 'context before 1'", err.Context.Before[1])
			}
		}

		// Check after context
		if len(err.Context.After) != 2 {
			t.Errorf("got %d after context lines, want 2", len(err.Context.After))
		}
		if len(err.Context.After) >= 2 {
			if err.Context.After[0] != "context after 1" {
				t.Errorf("after[0] = %q, want 'context after 1'", err.Context.After[0])
			}
			if err.Context.After[1] != "context after 2" {
				t.Errorf("after[1] = %q, want 'context after 2'", err.Context.After[1])
			}
		}
	})

	t.Run("handles edge cases at start of buffer", func(t *testing.T) {
		edgeEntries := []LogEntry{
			{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-2 * time.Second)},
			{Message: "after", Level: LogLevelInfo, Timestamp: now.Add(-1 * time.Second)},
		}
		edgeLb := &LogBuffer{entries: edgeEntries, maxSize: 1000}

		errors := edgeLb.GetErrors(50, 3, false, time.Time{})

		if len(errors) != 1 {
			t.Fatalf("got %d errors, want 1", len(errors))
		}

		// Should have no before context (error is at start)
		if len(errors[0].Context.Before) != 0 {
			t.Errorf("got %d before context lines, want 0", len(errors[0].Context.Before))
		}

		// Should have 1 after context line
		if len(errors[0].Context.After) != 1 {
			t.Errorf("got %d after context lines, want 1", len(errors[0].Context.After))
		}
	})

	t.Run("handles edge cases at end of buffer", func(t *testing.T) {
		edgeEntries := []LogEntry{
			{Message: "before", Level: LogLevelInfo, Timestamp: now.Add(-2 * time.Second)},
			{Message: "THE ERROR", Level: LogLevelError, Timestamp: now.Add(-1 * time.Second)},
		}
		edgeLb := &LogBuffer{entries: edgeEntries, maxSize: 1000}

		errors := edgeLb.GetErrors(50, 3, false, time.Time{})

		if len(errors) != 1 {
			t.Fatalf("got %d errors, want 1", len(errors))
		}

		// Should have 1 before context line
		if len(errors[0].Context.Before) != 1 {
			t.Errorf("got %d before context lines, want 1", len(errors[0].Context.Before))
		}

		// Should have no after context (error is at end)
		if len(errors[0].Context.After) != 0 {
			t.Errorf("got %d after context lines, want 0", len(errors[0].Context.After))
		}
	})

	t.Run("contextLines=0 returns no context", func(t *testing.T) {
		errors := lb.GetErrors(50, 0, false, time.Time{})

		if len(errors) != 1 {
			t.Fatalf("got %d errors, want 1", len(errors))
		}

		if len(errors[0].Context.Before) != 0 {
			t.Errorf("got %d before context lines, want 0", len(errors[0].Context.Before))
		}
		if len(errors[0].Context.After) != 0 {
			t.Errorf("got %d after context lines, want 0", len(errors[0].Context.After))
		}
	})

	t.Run("clamps contextLines to max 10", func(t *testing.T) {
		// Create buffer with 15 entries before and after the error
		manyEntries := make([]LogEntry, 31)
		for i := 0; i < 15; i++ {
			manyEntries[i] = LogEntry{
				Message:   "before",
				Level:     LogLevelInfo,
				Timestamp: now.Add(time.Duration(-30+i) * time.Second),
			}
		}
		manyEntries[15] = LogEntry{
			Message:   "THE ERROR",
			Level:     LogLevelError,
			Timestamp: now.Add(-15 * time.Second),
		}
		for i := 16; i < 31; i++ {
			manyEntries[i] = LogEntry{
				Message:   "after",
				Level:     LogLevelInfo,
				Timestamp: now.Add(time.Duration(-30+i) * time.Second),
			}
		}

		manyLb := &LogBuffer{entries: manyEntries, maxSize: 1000}

		// Request 20 context lines, should be clamped to 10
		errors := manyLb.GetErrors(50, 20, false, time.Time{})

		if len(errors) != 1 {
			t.Fatalf("got %d errors, want 1", len(errors))
		}

		if len(errors[0].Context.Before) != 10 {
			t.Errorf("got %d before context lines, want 10 (clamped)", len(errors[0].Context.Before))
		}
		if len(errors[0].Context.After) != 10 {
			t.Errorf("got %d after context lines, want 10 (clamped)", len(errors[0].Context.After))
		}
	})
}
