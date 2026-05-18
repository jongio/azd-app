package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

func readLogsFromFile(projectDir, serviceName string, tail int, sinceTime time.Time) ([]service.LogEntry, error) {
	logsDir := filepath.Join(projectDir, ".azure", "logs")
	baseLogFile := filepath.Join(logsDir, serviceName+".log")

	var allEntries []service.LogEntry

	// Read from rotated files first (oldest to newest: .log.2, .log.1, .log)
	logFiles := []string{
		baseLogFile + ".2",
		baseLogFile + ".1",
		baseLogFile,
	}

	for _, logFile := range logFiles {
		entries, err := readSingleLogFile(logFile, serviceName, sinceTime)
		if err != nil {
			continue // File may not exist (rotated files are optional)
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no log files found for service %s", serviceName)
	}

	// Apply tail limit
	if tail > 0 && len(allEntries) > tail {
		allEntries = allEntries[len(allEntries)-tail:]
	}

	return allEntries, nil
}

// readSingleLogFile reads log entries from a single log file.
func readSingleLogFile(logFile, serviceName string, sinceTime time.Time) ([]service.LogEntry, error) {
	cleanLogFile := filepath.Clean(logFile)
	file, err := os.Open(cleanLogFile)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var entries []service.LogEntry
	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle long log lines (stack traces, JSON dumps)
	scanner.Buffer(make([]byte, scannerInitialBufferSize), maxLogLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		entry, err := parseLogLine(line, serviceName)
		if err != nil {
			continue // Skip unparseable lines
		}

		// Apply since filter
		if !sinceTime.IsZero() && entry.Timestamp.Before(sinceTime) {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

// parseLogLine parses a log line from the file format:
// [2006-01-02 15:04:05.000] [LEVEL] [STREAM] message
func parseLogLine(line, serviceName string) (service.LogEntry, error) {
	entry := service.LogEntry{
		Service: serviceName,
	}

	// Parse timestamp: [2006-01-02 15:04:05.000]
	if len(line) < 25 || line[0] != '[' {
		return entry, fmt.Errorf("invalid log line format")
	}

	endTimestamp := strings.Index(line[1:], "]")
	if endTimestamp == -1 {
		return entry, fmt.Errorf("missing timestamp end bracket")
	}

	timestampStr := line[1 : endTimestamp+1]
	timestamp, err := time.Parse("2006-01-02 15:04:05.000", timestampStr)
	if err != nil {
		return entry, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	entry.Timestamp = timestamp

	// Parse remaining: [LEVEL] [STREAM] message
	remaining := line[endTimestamp+3:] // Skip "] "

	// Parse level: [LEVEL]
	if len(remaining) < 3 || remaining[0] != '[' {
		entry.Message = remaining
		entry.Level = service.LogLevelInfo
		return entry, nil
	}

	endLevel := strings.Index(remaining[1:], "]")
	if endLevel == -1 {
		entry.Message = remaining
		entry.Level = service.LogLevelInfo
		return entry, nil
	}

	levelStr := remaining[1 : endLevel+1]
	entry.Level = parseLogLevelFromString(levelStr)
	remaining = remaining[endLevel+3:] // Skip "] "

	// Parse stream: [STREAM]
	if len(remaining) >= 3 && remaining[0] == '[' {
		endStream := strings.Index(remaining[1:], "]")
		if endStream != -1 {
			streamStr := remaining[1 : endStream+1]
			entry.IsStderr = streamStr == "ERR"
			remaining = remaining[endStream+3:] // Skip "] "
		}
	}

	entry.Message = remaining
	return entry, nil
}

// parseLogLevelFromString parses a log level from a string.
func parseLogLevelFromString(level string) service.LogLevel {
	switch strings.ToUpper(level) {
	case "INFO":
		return service.LogLevelInfo
	case "WARN", "WARNING":
		return service.LogLevelWarn
	case "ERROR":
		return service.LogLevelError
	case "DEBUG":
		return service.LogLevelDebug
	default:
		return service.LogLevelInfo
	}
}

// ANSI color constants for log output formatting.
// Other colors (colorGray, colorRed, colorYellow, colorReset) are in info.go.

// displayLogsText displays logs in text format.
// Uses io.Writer interface for better testability and flexibility.
