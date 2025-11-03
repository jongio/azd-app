package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogBuffer is a circular buffer for storing service logs with pub/sub support.
type LogBuffer struct {
	serviceName string
	entries     []LogEntry
	maxSize     int
	mu          sync.RWMutex
	subscribers map[chan LogEntry]bool
	subMu       sync.RWMutex
	filePath    string
	fileWriter  *bufio.Writer
	file        *os.File
	fileMu      sync.Mutex
}

// NewLogBuffer creates a new log buffer for a service.
func NewLogBuffer(serviceName string, maxSize int, enableFileLogging bool, projectDir string) (*LogBuffer, error) {
	lb := &LogBuffer{
		serviceName: serviceName,
		entries:     make([]LogEntry, 0, maxSize),
		maxSize:     maxSize,
		subscribers: make(map[chan LogEntry]bool),
	}

	// Setup file logging if enabled
	if enableFileLogging {
		logsDir := filepath.Join(projectDir, ".azure", "logs")
		if err := os.MkdirAll(logsDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create logs directory: %w", err)
		}

		lb.filePath = filepath.Join(logsDir, fmt.Sprintf("%s.log", serviceName))
		file, err := os.OpenFile(lb.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		lb.file = file
		lb.fileWriter = bufio.NewWriter(file)
	}

	return lb, nil
}

// Add appends a log entry to the buffer.
func (lb *LogBuffer) Add(entry LogEntry) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Add to circular buffer
	if len(lb.entries) >= lb.maxSize {
		// Remove oldest entry
		lb.entries = lb.entries[1:]
	}
	lb.entries = append(lb.entries, entry)

	// Write to file if enabled
	if lb.fileWriter != nil {
		lb.fileMu.Lock()
		lb.writeToFile(entry)
		lb.fileMu.Unlock()
	}

	// Broadcast to subscribers
	lb.broadcast(entry)
}

// writeToFile writes a log entry to the file (must be called with fileMu locked).
func (lb *LogBuffer) writeToFile(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	level := entry.Level.String()
	stream := "OUT"
	if entry.IsStderr {
		stream = "ERR"
	}

	line := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, stream, entry.Message)
	if _, err := lb.fileWriter.WriteString(line); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write log entry: %v\n", err)
	}
	if err := lb.fileWriter.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to flush log buffer: %v\n", err)
	}
}

// GetRecent returns the last N entries from the buffer.
func (lb *LogBuffer) GetRecent(n int) []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if n <= 0 || n > len(lb.entries) {
		n = len(lb.entries)
	}

	start := len(lb.entries) - n
	result := make([]LogEntry, n)
	copy(result, lb.entries[start:])
	return result
}

// GetSince returns entries since a specific time.
func (lb *LogBuffer) GetSince(since time.Time) []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]LogEntry, 0)
	for _, entry := range lb.entries {
		if entry.Timestamp.After(since) || entry.Timestamp.Equal(since) {
			result = append(result, entry)
		}
	}
	return result
}

// GetByLevel returns entries matching the specified log level.
func (lb *LogBuffer) GetByLevel(level LogLevel) []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]LogEntry, 0)
	for _, entry := range lb.entries {
		if entry.Level == level {
			result = append(result, entry)
		}
	}
	return result
}

// Subscribe creates a new subscription channel for live log streaming.
func (lb *LogBuffer) Subscribe() chan LogEntry {
	lb.subMu.Lock()
	defer lb.subMu.Unlock()

	ch := make(chan LogEntry, 100) // Buffered to prevent blocking
	lb.subscribers[ch] = true
	return ch
}

// Unsubscribe removes a subscription channel.
func (lb *LogBuffer) Unsubscribe(ch chan LogEntry) {
	lb.subMu.Lock()
	defer lb.subMu.Unlock()

	if _, exists := lb.subscribers[ch]; exists {
		delete(lb.subscribers, ch)
		close(ch)
	}
}

// broadcast sends a log entry to all subscribers.
func (lb *LogBuffer) broadcast(entry LogEntry) {
	lb.subMu.RLock()
	defer lb.subMu.RUnlock()

	for ch := range lb.subscribers {
		// Non-blocking send to prevent slow subscribers from blocking
		select {
		case ch <- entry:
		default:
			// Channel buffer full, skip this entry for this subscriber
		}
	}
}

// Clear empties the buffer.
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries = make([]LogEntry, 0, lb.maxSize)
}

// Close closes the log buffer and cleans up resources.
func (lb *LogBuffer) Close() error {
	// Close all subscriber channels
	lb.subMu.Lock()
	for ch := range lb.subscribers {
		close(ch)
		delete(lb.subscribers, ch)
	}
	lb.subMu.Unlock()

	// Close file if open
	if lb.file != nil {
		lb.fileMu.Lock()
		defer lb.fileMu.Unlock()

		if lb.fileWriter != nil {
			if err := lb.fileWriter.Flush(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to flush log buffer on close: %v\n", err)
			}
		}
		return lb.file.Close()
	}

	return nil
}

// inferLogLevel attempts to infer the log level from a log message.
func inferLogLevel(message string) LogLevel {
	lowerMsg := strings.ToLower(message)

	// Check for error indicators
	if strings.Contains(lowerMsg, "error") || strings.Contains(lowerMsg, "exception") ||
		strings.Contains(lowerMsg, "fatal") || strings.Contains(lowerMsg, "panic") {
		return LogLevelError
	}

	// Check for warning indicators
	if strings.Contains(lowerMsg, "warn") || strings.Contains(lowerMsg, "warning") {
		return LogLevelWarn
	}

	// Check for debug indicators
	if strings.Contains(lowerMsg, "debug") || strings.Contains(lowerMsg, "trace") {
		return LogLevelDebug
	}

	// Default to info
	return LogLevelInfo
}
