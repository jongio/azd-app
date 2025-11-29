package service

import (
	"testing"
	"time"
)

func TestGetLogManager(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"with path", "/test/path"},
		{"empty path uses cwd", ""},
		{"relative path", "./test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := GetLogManager(tt.path)
			if lm == nil {
				t.Fatal("GetLogManager returned nil")
			}
			if lm.buffers == nil {
				t.Error("buffers map should be initialized")
			}

			// Getting the same path should return the same instance
			lm2 := GetLogManager(tt.path)
			if tt.path != "" && lm != lm2 {
				t.Error("GetLogManager should return same instance for same path")
			}
		})
	}
}

func TestLogManagerCreateBuffer(t *testing.T) {
	lm := GetLogManager("/test/logmanager")

	buffer, err := lm.CreateBuffer("test-service", 1000, false)
	if err != nil {
		t.Fatalf("CreateBuffer() error = %v", err)
	}
	if buffer == nil {
		t.Fatal("CreateBuffer returned nil buffer")
	}

	// Creating again should return existing buffer
	buffer2, err := lm.CreateBuffer("test-service", 1000, false)
	if err != nil {
		t.Fatalf("CreateBuffer() second call error = %v", err)
	}
	if buffer != buffer2 {
		t.Error("CreateBuffer should return existing buffer for same service")
	}
}

func TestLogManagerGetBuffer(t *testing.T) {
	lm := GetLogManager("/test/getbuffer")

	// Non-existent buffer
	_, exists := lm.GetBuffer("nonexistent")
	if exists {
		t.Error("GetBuffer should return false for nonexistent buffer")
	}

	// Create and get buffer
	buffer, _ := lm.CreateBuffer("api", 1000, false)
	gotBuffer, exists := lm.GetBuffer("api")
	if !exists {
		t.Error("GetBuffer should return true for existing buffer")
	}
	if gotBuffer != buffer {
		t.Error("GetBuffer should return the same buffer instance")
	}
}

func TestLogManagerGetAllBuffers(t *testing.T) {
	lm := GetLogManager("/test/getallbuffers")

	// Empty buffers
	buffers := lm.GetAllBuffers()
	if len(buffers) != 0 {
		t.Error("GetAllBuffers should return empty map initially")
	}

	// Create multiple buffers
	_, _ = lm.CreateBuffer("api", 1000, false)
	_, _ = lm.CreateBuffer("web", 1000, false)
	_, _ = lm.CreateBuffer("worker", 1000, false)

	buffers = lm.GetAllBuffers()
	if len(buffers) != 3 {
		t.Errorf("GetAllBuffers returned %d buffers, want 3", len(buffers))
	}

	// Verify all services are present
	expectedServices := []string{"api", "web", "worker"}
	for _, service := range expectedServices {
		if _, exists := buffers[service]; !exists {
			t.Errorf("GetAllBuffers missing service %q", service)
		}
	}
}

func TestLogManagerGetAllLogs(t *testing.T) {
	lm := GetLogManager("/test/getalllogs")

	// Create buffers and add logs
	buffer1, _ := lm.CreateBuffer("api", 1000, false)
	buffer2, _ := lm.CreateBuffer("web", 1000, false)

	buffer1.Add(LogEntry{Service: "api", Message: "api log 1", Timestamp: time.Now()})
	buffer1.Add(LogEntry{Service: "api", Message: "api log 2", Timestamp: time.Now()})
	buffer2.Add(LogEntry{Service: "web", Message: "web log 1", Timestamp: time.Now()})

	logs := lm.GetAllLogs(10)
	if len(logs) != 3 {
		t.Errorf("GetAllLogs returned %d logs, want 3", len(logs))
	}

	// Verify logs are from different services
	services := make(map[string]bool)
	for _, log := range logs {
		services[log.Service] = true
	}
	if len(services) != 2 {
		t.Errorf("GetAllLogs should include logs from 2 services, got %d", len(services))
	}
}

func TestLogManagerGetAllLogsSince(t *testing.T) {
	lm := GetLogManager("/test/getalllogssince")

	buffer, _ := lm.CreateBuffer("test", 1000, false)

	// Add logs at different times
	now := time.Now()
	buffer.Add(LogEntry{Service: "test", Message: "old log", Timestamp: now})
	time.Sleep(10 * time.Millisecond)
	since := time.Now()
	time.Sleep(10 * time.Millisecond)
	buffer.Add(LogEntry{Service: "test", Message: "new log 1", Timestamp: time.Now()})
	buffer.Add(LogEntry{Service: "test", Message: "new log 2", Timestamp: time.Now()})

	logs := lm.GetAllLogsSince(since)

	// Should only include logs after 'since' timestamp
	if len(logs) < 2 {
		t.Errorf("GetAllLogsSince returned %d logs, want at least 2", len(logs))
	}

	// Verify all logs are after 'since'
	for _, log := range logs {
		if log.Timestamp.Before(since) {
			t.Errorf("log timestamp %v is before since %v", log.Timestamp, since)
		}
	}

	// Verify we don't get the old log
	for _, log := range logs {
		if log.Message == "old log" {
			t.Error("GetAllLogsSince should not include logs before since timestamp")
		}
	}

	// Test with timestamp before all logs
	allLogs := lm.GetAllLogsSince(now.Add(-1 * time.Hour))
	if len(allLogs) < 3 {
		t.Errorf("GetAllLogsSince with old timestamp should return all logs, got %d", len(allLogs))
	}
}

func TestLogManagerGetServiceNames(t *testing.T) {
	lm := GetLogManager("/test/getservicenames")

	// Empty case
	names := lm.GetServiceNames()
	if len(names) != 0 {
		t.Error("GetServiceNames should return empty slice initially")
	}

	// Create buffers
	_, _ = lm.CreateBuffer("api", 1000, false)
	_, _ = lm.CreateBuffer("web", 1000, false)
	_, _ = lm.CreateBuffer("db", 1000, false)

	names = lm.GetServiceNames()
	if len(names) != 3 {
		t.Errorf("GetServiceNames returned %d names, want 3", len(names))
	}

	// Verify names are sorted (alphabetically)
	// The actual sort order depends on the implementation
	// Just verify we have the right names
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	expectedNames := []string{"api", "db", "web"}
	for _, expected := range expectedNames {
		if !nameMap[expected] {
			t.Errorf("GetServiceNames missing expected name: %s", expected)
		}
	}
}

func TestLogManagerRemoveBuffer(t *testing.T) {
	lm := GetLogManager("/test/removebuffer")

	// Create buffer
	buffer, _ := lm.CreateBuffer("temp-service", 1000, false)
	if buffer == nil {
		t.Fatal("failed to create buffer")
	}

	// Verify it exists
	_, exists := lm.GetBuffer("temp-service")
	if !exists {
		t.Error("buffer should exist after creation")
	}

	// Remove buffer
	err := lm.RemoveBuffer("temp-service")
	if err != nil {
		t.Fatalf("Failed to remove buffer: %v", err)
	}

	// Verify it's gone
	_, exists = lm.GetBuffer("temp-service")
	if exists {
		t.Error("buffer should not exist after removal")
	}

	// Removing non-existent buffer should not panic
	_ = lm.RemoveBuffer("nonexistent")
}

func TestLogManagerClear(t *testing.T) {
	lm := GetLogManager("/test/clearmanager")

	// Create multiple buffers
	_, _ = lm.CreateBuffer("api", 1000, false)
	_, _ = lm.CreateBuffer("web", 1000, false)

	buffers := lm.GetAllBuffers()
	if len(buffers) != 2 {
		t.Errorf("expected 2 buffers before clear, got %d", len(buffers))
	}

	// Clear all
	err := lm.Clear()
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	buffers = lm.GetAllBuffers()
	if len(buffers) != 0 {
		t.Errorf("expected 0 buffers after clear, got %d", len(buffers))
	}
}

func TestLogManagerClearBuffer(t *testing.T) {
	lm := GetLogManager("/test/clearbuffer")

	// Create buffer and add logs
	buffer, _ := lm.CreateBuffer("api", 1000, false)
	buffer.Add(LogEntry{Service: "api", Message: "log 1", Timestamp: time.Now()})
	buffer.Add(LogEntry{Service: "api", Message: "log 2", Timestamp: time.Now()})

	logs := buffer.GetRecent(10)
	if len(logs) != 2 {
		t.Errorf("expected 2 logs before clear, got %d", len(logs))
	}

	// Clear specific buffer
	err := lm.ClearBuffer("api")
	if err != nil {
		t.Fatalf("Failed to clear buffer: %v", err)
	}

	logs = buffer.GetRecent(10)
	if len(logs) != 0 {
		t.Errorf("expected 0 logs after clear, got %d", len(logs))
	}

	// Clearing non-existent buffer should not panic
	_ = lm.ClearBuffer("nonexistent")
}

func TestSortLogEntries(t *testing.T) {
	now := time.Now()
	entries := []LogEntry{
		{Timestamp: now.Add(2 * time.Second), Message: "third"},
		{Timestamp: now, Message: "first"},
		{Timestamp: now.Add(1 * time.Second), Message: "second"},
	}

	SortLogEntries(entries)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Message != "first" {
		t.Errorf("entries[0] message = %q, want %q", entries[0].Message, "first")
	}
	if entries[1].Message != "second" {
		t.Errorf("entries[1] message = %q, want %q", entries[1].Message, "second")
	}
	if entries[2].Message != "third" {
		t.Errorf("entries[2] message = %q, want %q", entries[2].Message, "third")
	}

	// Verify timestamps are in ascending order
	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.Before(entries[i-1].Timestamp) {
			t.Errorf("entries not sorted: entry %d timestamp is before entry %d", i, i-1)
		}
	}
}

func TestLogManagerConcurrency(t *testing.T) {
	lm := GetLogManager("/test/concurrency")

	// Test concurrent buffer creation
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			_, err := lm.CreateBuffer("concurrent-test", 1000, false)
			if err != nil {
				t.Errorf("concurrent CreateBuffer failed: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only have one buffer
	buffers := lm.GetAllBuffers()
	if len(buffers) != 1 {
		t.Errorf("concurrent CreateBuffer should create only 1 buffer, got %d", len(buffers))
	}
}
