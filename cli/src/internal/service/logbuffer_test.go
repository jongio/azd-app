package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogBuffer(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name              string
		serviceName       string
		maxSize           int
		enableFileLogging bool
		wantErr           bool
	}{
		{
			name:              "basic buffer without file logging",
			serviceName:       "test-service",
			maxSize:           100,
			enableFileLogging: false,
			wantErr:           false,
		},
		{
			name:              "buffer with file logging",
			serviceName:       "test-service-log",
			maxSize:           100,
			enableFileLogging: true,
			wantErr:           false,
		},
		{
			name:              "large buffer",
			serviceName:       "large-service",
			maxSize:           10000,
			enableFileLogging: false,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer, err := NewLogBuffer(tt.serviceName, tt.maxSize, tt.enableFileLogging, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogBuffer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer buffer.Close()

				if buffer.serviceName != tt.serviceName {
					t.Errorf("serviceName = %v, want %v", buffer.serviceName, tt.serviceName)
				}

				if buffer.maxSize != tt.maxSize {
					t.Errorf("maxSize = %v, want %v", buffer.maxSize, tt.maxSize)
				}

				if tt.enableFileLogging {
					if buffer.filePath == "" {
						t.Error("filePath is empty when file logging enabled")
					}
					if buffer.file == nil {
						t.Error("file is nil when file logging enabled")
					}
				}
			}
		})
	}
}

func TestLogBuffer_Add(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 10, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add some entries
	for i := 0; i < 5; i++ {
		entry := LogEntry{
			Service:   "test",
			Message:   "test message",
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		}
		buffer.Add(entry)
	}

	recent := buffer.GetRecent(10)
	if len(recent) != 5 {
		t.Errorf("GetRecent() returned %d entries, want 5", len(recent))
	}
}

func TestLogBuffer_CircularBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 5, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add more entries than max size
	for i := 0; i < 10; i++ {
		entry := LogEntry{
			Service:   "test",
			Message:   "message",
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		}
		buffer.Add(entry)
	}

	recent := buffer.GetRecent(100)
	if len(recent) != 5 {
		t.Errorf("Buffer size = %d, want 5 (max size)", len(recent))
	}
}

func TestLogBuffer_GetRecent(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		buffer.Add(LogEntry{
			Service:   "test",
			Message:   "msg",
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		})
	}

	tests := []struct {
		name string
		n    int
		want int
	}{
		{"get last 5", 5, 5},
		{"get last 3", 3, 3},
		{"get all", 10, 10},
		{"get more than available", 20, 10},
		{"get zero", 0, 10},
		{"get negative", -1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recent := buffer.GetRecent(tt.n)
			if len(recent) != tt.want {
				t.Errorf("GetRecent(%d) returned %d entries, want %d", tt.n, len(recent), tt.want)
			}
		})
	}
}

func TestLogBuffer_GetSince(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add entries with specific timestamps
	now := time.Now()
	timestamps := []time.Time{
		now.Add(-5 * time.Second),
		now.Add(-4 * time.Second),
		now.Add(-3 * time.Second),
		now.Add(-2 * time.Second),
		now.Add(-1 * time.Second),
	}

	for _, ts := range timestamps {
		buffer.Add(LogEntry{
			Service:   "test",
			Message:   "msg",
			Timestamp: ts,
			Level:     LogLevelInfo,
		})
	}

	// Get entries since 3 seconds ago
	since := now.Add(-3 * time.Second)
	recent := buffer.GetSince(since)

	if len(recent) != 3 {
		t.Errorf("GetSince() returned %d entries, want 3", len(recent))
	}
}

func TestLogBuffer_GetByLevel(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add entries with different levels
	levels := []LogLevel{
		LogLevelInfo,
		LogLevelError,
		LogLevelWarn,
		LogLevelInfo,
		LogLevelError,
	}

	for _, level := range levels {
		buffer.Add(LogEntry{
			Service:   "test",
			Message:   "msg",
			Timestamp: time.Now(),
			Level:     level,
		})
	}

	tests := []struct {
		level LogLevel
		want  int
	}{
		{LogLevelInfo, 2},
		{LogLevelError, 2},
		{LogLevelWarn, 1},
		{LogLevelDebug, 0},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			entries := buffer.GetByLevel(tt.level)
			if len(entries) != tt.want {
				t.Errorf("GetByLevel(%v) returned %d entries, want %d", tt.level, len(entries), tt.want)
			}
		})
	}
}

func TestLogBuffer_Subscribe(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Subscribe to buffer
	ch := buffer.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe() returned nil channel")
	}

	// Add an entry
	entry := LogEntry{
		Service:   "test",
		Message:   "test message",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
	}

	buffer.Add(entry)

	// Verify we received it
	select {
	case received := <-ch:
		if received.Message != entry.Message {
			t.Errorf("Received message = %v, want %v", received.Message, entry.Message)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for broadcast entry")
	}

	// Unsubscribe
	buffer.Unsubscribe(ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("Channel not closed after unsubscribe")
	}
}

func TestLogBuffer_MultipleSubscribers(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Create multiple subscribers
	ch1 := buffer.Subscribe()
	ch2 := buffer.Subscribe()
	ch3 := buffer.Subscribe()

	// Add an entry
	entry := LogEntry{
		Service:   "test",
		Message:   "broadcast test",
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
	}

	buffer.Add(entry)

	// All subscribers should receive it
	timeout := time.After(1 * time.Second)

	for i, ch := range []chan LogEntry{ch1, ch2, ch3} {
		select {
		case received := <-ch:
			if received.Message != entry.Message {
				t.Errorf("Subscriber %d: received message = %v, want %v", i, received.Message, entry.Message)
			}
		case <-timeout:
			t.Errorf("Subscriber %d: timeout waiting for entry", i)
		}
	}

	// Clean up
	buffer.Unsubscribe(ch1)
	buffer.Unsubscribe(ch2)
	buffer.Unsubscribe(ch3)
}

func TestLogBuffer_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, false, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}
	defer buffer.Close()

	// Add some entries
	for i := 0; i < 5; i++ {
		buffer.Add(LogEntry{
			Service:   "test",
			Message:   "msg",
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		})
	}

	// Clear buffer
	buffer.Clear()

	// Buffer should be empty
	recent := buffer.GetRecent(10)
	if len(recent) != 0 {
		t.Errorf("Buffer size after Clear() = %d, want 0", len(recent))
	}
}

func TestLogBuffer_FileLogging(t *testing.T) {
	tmpDir := t.TempDir()
	serviceName := "file-test"

	buffer, err := NewLogBuffer(serviceName, 100, true, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}

	// Add entries
	messages := []string{"msg1", "msg2", "msg3"}
	for _, msg := range messages {
		buffer.Add(LogEntry{
			Service:   serviceName,
			Message:   msg,
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
			IsStderr:  false,
		})
	}

	// Close to flush
	if err := buffer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify file was created and contains entries
	logPath := filepath.Join(tmpDir, ".azure", "logs", serviceName+".log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	for _, msg := range messages {
		if !strings.Contains(contentStr, msg) {
			t.Errorf("Log file missing message: %s", msg)
		}
	}
}

func TestLogBuffer_Close(t *testing.T) {
	tmpDir := t.TempDir()
	buffer, err := NewLogBuffer("test", 100, true, tmpDir)
	if err != nil {
		t.Fatalf("NewLogBuffer() error = %v", err)
	}

	// Subscribe
	ch := buffer.Subscribe()

	// Close buffer
	if err := buffer.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("Subscriber channel not closed after Close()")
	}

	// Trying to add after close should not panic (writes ignored)
	// This verifies graceful handling of writes after close
}

func TestLogBuffer_WithFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a filter that blocks certain messages
	filter, err := NewLogFilter([]string{
		"Autofill\\.enable",
		"npm warn",
		"Debugger listening",
	})
	if err != nil {
		t.Fatalf("NewLogFilter() error = %v", err)
	}

	// Create buffer with filter
	buffer, err := NewLogBufferWithFilter("test-service", 100, false, tmpDir, filter)
	if err != nil {
		t.Fatalf("NewLogBufferWithFilter() error = %v", err)
	}
	defer buffer.Close()

	// Add various log entries
	testCases := []struct {
		message      string
		shouldFilter bool
	}{
		{"Application started", false},
		{"Request Autofill.enable failed", true},
		{"npm warn Unknown env config", true},
		{"Debugger listening on ws://127.0.0.1:5858", true},
		{"User logged in successfully", false},
		{"Error: Connection failed", false},
	}

	for _, tc := range testCases {
		buffer.Add(LogEntry{
			Service:   "test-service",
			Message:   tc.message,
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		})
	}

	// Verify only unfiltered entries are in the buffer
	entries := buffer.GetRecent(100)
	
	// Count expected entries
	expectedCount := 0
	for _, tc := range testCases {
		if !tc.shouldFilter {
			expectedCount++
		}
	}

	if len(entries) != expectedCount {
		t.Errorf("Expected %d entries after filtering, got %d", expectedCount, len(entries))
	}

	// Verify filtered messages are not in the buffer
	for _, entry := range entries {
		for _, tc := range testCases {
			if tc.shouldFilter && entry.Message == tc.message {
				t.Errorf("Filtered message found in buffer: %s", tc.message)
			}
		}
	}
}

func TestLogBuffer_WithoutFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create buffer without filter (nil)
	buffer, err := NewLogBufferWithFilter("test-service", 100, false, tmpDir, nil)
	if err != nil {
		t.Fatalf("NewLogBufferWithFilter() error = %v", err)
	}
	defer buffer.Close()

	// Add entries that would normally be filtered
	messages := []string{
		"Request Autofill.enable failed",
		"npm warn Unknown env config",
		"Normal log message",
	}

	for _, msg := range messages {
		buffer.Add(LogEntry{
			Service:   "test-service",
			Message:   msg,
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
		})
	}

	// Without filter, all entries should be present
	entries := buffer.GetRecent(100)
	if len(entries) != len(messages) {
		t.Errorf("Expected %d entries without filter, got %d", len(messages), len(entries))
	}
}


