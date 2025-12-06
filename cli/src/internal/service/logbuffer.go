// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// MaxLogFileSize is the maximum size of a log file before rotation (1MB)
	MaxLogFileSize = 1 * 1024 * 1024
	// MaxLogFileBackups is the number of backup log files to keep
	MaxLogFileBackups = 2
)

// LogBuffer is a circular buffer for storing service logs with pub/sub support.
type LogBuffer struct {
	serviceName     string
	entries         []LogEntry
	maxSize         int
	mu              sync.RWMutex
	subscribers     map[chan LogEntry]bool
	subMu           sync.RWMutex
	filePath        string
	fileWriter      *bufio.Writer
	file            *os.File
	fileMu          sync.Mutex
	logFilter       *LogFilter // Optional filter for noisy log messages
	currentFileSize int64      // Track current file size for rotation
}

// NewLogBuffer creates a new log buffer for a service.
func NewLogBuffer(serviceName string, maxSize int, enableFileLogging bool, projectDir string) (*LogBuffer, error) {
	return NewLogBufferWithFilter(serviceName, maxSize, enableFileLogging, projectDir, nil)
}

// NewLogBufferWithFilter creates a new log buffer with optional log filtering.
func NewLogBufferWithFilter(serviceName string, maxSize int, enableFileLogging bool, projectDir string, filter *LogFilter) (*LogBuffer, error) {
	lb := &LogBuffer{
		serviceName: serviceName,
		entries:     make([]LogEntry, 0, maxSize),
		maxSize:     maxSize,
		subscribers: make(map[chan LogEntry]bool),
		logFilter:   filter,
	}

	// Setup file logging if enabled
	if enableFileLogging {
		logsDir := filepath.Join(projectDir, ".azure", "logs")
		// Use 0700 for directory permissions to match file privacy intent (0600)
		// This ensures only the owner can access log files
		if err := os.MkdirAll(logsDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create logs directory: %w", err)
		}

		lb.filePath = filepath.Join(logsDir, fmt.Sprintf("%s.log", serviceName))
		file, err := os.OpenFile(lb.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		// Get initial file size for rotation tracking
		if stat, err := file.Stat(); err == nil {
			lb.currentFileSize = stat.Size()
		}

		lb.file = file
		lb.fileWriter = bufio.NewWriter(file)
	}

	return lb, nil
}

// Add appends a log entry to the buffer.
// If a log filter is configured, noisy messages are filtered out.
func (lb *LogBuffer) Add(entry LogEntry) {
	// Apply log filter if configured
	if lb.logFilter != nil && lb.logFilter.ShouldFilter(entry.Message) {
		return // Skip noisy log entry
	}

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
	// Check if rotation is needed
	if lb.currentFileSize >= MaxLogFileSize {
		lb.rotateLogFile()
	}

	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	level := entry.Level.String()
	stream := "OUT"
	if entry.IsStderr {
		stream = "ERR"
	}

	line := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, stream, entry.Message)
	n, err := lb.fileWriter.WriteString(line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write log entry: %v\n", err)
		return
	}
	lb.currentFileSize += int64(n)

	if err := lb.fileWriter.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to flush log buffer: %v\n", err)
	}
}

// rotateLogFile rotates the current log file (must be called with fileMu locked).
func (lb *LogBuffer) rotateLogFile() {
	// Flush and close current file
	if lb.fileWriter != nil {
		lb.fileWriter.Flush()
	}
	if lb.file != nil {
		lb.file.Close()
	}

	// Rotate existing backup files (delete oldest, shift others)
	for i := MaxLogFileBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", lb.filePath, i)
		newPath := fmt.Sprintf("%s.%d", lb.filePath, i+1)
		_ = os.Rename(oldPath, newPath) // Ignore errors - file may not exist
	}

	// Delete the oldest backup if it exceeds MaxLogFileBackups
	oldestBackup := fmt.Sprintf("%s.%d", lb.filePath, MaxLogFileBackups+1)
	os.Remove(oldestBackup) // Ignore errors

	// Rename current file to .1
	if err := os.Rename(lb.filePath, lb.filePath+".1"); err != nil {
		slog.Debug("failed to rotate log file", "error", err)
	}

	// Open new file
	file, err := os.OpenFile(lb.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create new log file after rotation: %v\n", err)
		return
	}

	lb.file = file
	lb.fileWriter = bufio.NewWriter(file)
	lb.currentFileSize = 0
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

// ContainsPattern checks if any log entry matches the given regex pattern.
// Returns true if the pattern is found in any log message.
func (lb *LogBuffer) ContainsPattern(pattern string) bool {
	if pattern == "" {
		return false
	}

	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Use simple string matching for common patterns (faster)
	// Fall back to regex for complex patterns
	for _, entry := range lb.entries {
		if strings.Contains(entry.Message, pattern) {
			return true
		}
	}

	return false
}

// ContainsPatternRegex checks if any log entry matches the given regex pattern.
// Returns true if the pattern is found in any log message.
func (lb *LogBuffer) ContainsPatternRegex(pattern string) (bool, error) {
	if pattern == "" {
		return false, nil
	}

	lb.mu.RLock()
	defer lb.mu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	for _, entry := range lb.entries {
		if re.MatchString(entry.Message) {
			return true, nil
		}
	}

	return false, nil
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
// Uses non-blocking sends with timeout to prevent deadlocks.
func (lb *LogBuffer) broadcast(entry LogEntry) {
	lb.subMu.RLock()
	defer lb.subMu.RUnlock()

	for ch := range lb.subscribers {
		// Non-blocking send with timeout to prevent slow subscribers from blocking
		// If subscriber can't keep up, we drop the message rather than blocking
		// Use a goroutine with recover to handle closed channel panics safely
		func(c chan LogEntry) {
			defer func() {
				if r := recover(); r != nil {
					slog.Debug("recovered from panic during log broadcast", "error", r)
				}
			}()
			select {
			case c <- entry:
				// Successfully sent
			case <-time.After(DefaultLogSubscriberTimeout):
				// Subscriber too slow, drop message
				slog.Debug("dropped log entry for slow subscriber", "service", entry.Service)
			default:
				// Channel buffer full, skip this entry for this subscriber
			}
		}(ch)
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
	// Close all subscriber channels - take write lock to prevent new subscribers
	lb.subMu.Lock()
	subscribers := make(map[chan LogEntry]bool)
	for ch := range lb.subscribers {
		subscribers[ch] = true
		delete(lb.subscribers, ch)
	}
	lb.subMu.Unlock()

	// Close channels outside of lock to prevent deadlock
	for ch := range subscribers {
		close(ch)
	}

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

	// Check for patterns that should always be INFO (overrides error/warning detection)
	// These are success messages that contain words like "error" but aren't actually errors
	for _, pattern := range infoOverridePatterns {
		if strings.Contains(lowerMsg, pattern) {
			return LogLevelInfo
		}
	}

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

// infoOverridePatterns contains patterns that should always be classified as INFO,
// even if they contain words like "error" or "warning".
// These are typically success messages from build tools.
var infoOverridePatterns = []string{
	// TypeScript compiler success messages
	"found 0 errors",
	"0 error(s)",
	"0 errors",
	// Build success patterns
	"build succeeded",
	"compilation succeeded",
	"compiled successfully",
	// Test success patterns
	"0 failed",
	"all tests passed",
}
