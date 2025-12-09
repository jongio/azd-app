// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// LogManager manages log buffers for all services in a project.
type LogManager struct {
	projectDir string
	buffers    map[string]*LogBuffer // key: serviceName
	logFilter  *LogFilter            // Optional log filter for all buffers
	mu         sync.RWMutex
}

var (
	logManagers   = make(map[string]*LogManager)
	logManagersMu sync.RWMutex
)

// GetLogManager returns the log manager for a project directory.
func GetLogManager(projectDir string) *LogManager {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			projectDir = "."
		} else {
			projectDir = cwd
		}
	}

	// Normalize path
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}

	logManagersMu.Lock()
	defer logManagersMu.Unlock()

	if lm, exists := logManagers[absPath]; exists {
		return lm
	}

	lm := &LogManager{
		projectDir: absPath,
		buffers:    make(map[string]*LogBuffer),
		logFilter:  loadLogFilterForProject(absPath),
	}
	logManagers[absPath] = lm

	return lm
}

// loadLogFilterForProject loads the log filter configuration from azure.yaml.
func loadLogFilterForProject(projectDir string) *LogFilter {
	azureYamlPath := filepath.Join(projectDir, "azure.yaml")
	azureYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		// No azure.yaml or parse error - use built-in filters only
		filter, _ := NewLogFilterWithBuiltins(nil)
		return filter
	}

	// Get filter config from azure.yaml
	filterConfig := azureYaml.Logs.GetFilters()
	var customPatterns []string
	includeBuiltins := true

	if filterConfig != nil {
		customPatterns = filterConfig.Exclude
		includeBuiltins = filterConfig.ShouldIncludeBuiltins()
	}

	var filter *LogFilter
	if includeBuiltins {
		filter, _ = NewLogFilterWithBuiltins(customPatterns)
	} else {
		filter, _ = NewLogFilter(customPatterns)
	}
	return filter
}

// CreateBuffer creates a log buffer for a service.
func (lm *LogManager) CreateBuffer(serviceName string, maxSize int, enableFileLogging bool) (*LogBuffer, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Return existing buffer if already created
	if buffer, exists := lm.buffers[serviceName]; exists {
		return buffer, nil
	}

	// Create new buffer with the log filter
	buffer, err := NewLogBufferWithFilter(serviceName, maxSize, enableFileLogging, lm.projectDir, lm.logFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to create log buffer for %s: %w", serviceName, err)
	}

	lm.buffers[serviceName] = buffer
	return buffer, nil
}

// GetBuffer retrieves a log buffer for a service.
func (lm *LogManager) GetBuffer(serviceName string) (*LogBuffer, bool) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	buffer, exists := lm.buffers[serviceName]
	return buffer, exists
}

// GetAllBuffers returns all log buffers.
func (lm *LogManager) GetAllBuffers() map[string]*LogBuffer {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]*LogBuffer, len(lm.buffers))
	for k, v := range lm.buffers {
		result[k] = v
	}
	return result
}

// GetAllLogs returns logs from all services, limited to N most recent entries per service.
func (lm *LogManager) GetAllLogs(n int) []LogEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var allLogs []LogEntry
	for _, buffer := range lm.buffers {
		logs := buffer.GetRecent(n)
		allLogs = append(allLogs, logs...)
	}

	// Sort by timestamp
	SortLogEntries(allLogs)

	return allLogs
}

// GetAllLogsSince returns logs from all services since a specific time.
func (lm *LogManager) GetAllLogsSince(since time.Time) []LogEntry {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var allLogs []LogEntry
	for _, buffer := range lm.buffers {
		logs := buffer.GetSince(since)
		allLogs = append(allLogs, logs...)
	}

	// Sort by timestamp
	SortLogEntries(allLogs)

	return allLogs
}

// GetAllLogsWithContext returns log entries matching the specified level from all services with context.
// Parameters:
//   - serviceName: filter to specific service (empty = all services)
//   - level: log level to filter by (LogLevelError, LogLevelWarn, etc.)
//   - limit: maximum total entries to return (0 = default 50)
//   - contextLines: lines before/after each entry (0-10, default 3)
//   - since: only entries after this time (zero = no filter)
//
// Returns entries sorted by timestamp (most recent first).
func (lm *LogManager) GetAllLogsWithContext(serviceName string, level LogLevel, limit int, contextLines int, since time.Time) []LogEntryWithContext {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Apply defaults
	if limit <= 0 {
		limit = 50
	}
	// Note: contextLines is clamped in LogBuffer.GetLogsWithContext
	// Negative values will be clamped to 0 there

	var allEntries []LogEntryWithContext

	for name, buffer := range lm.buffers {
		// Filter by service name if specified
		if serviceName != "" && name != serviceName {
			continue
		}

		entries := buffer.GetLogsWithContext(level, 0, contextLines, since) // Get all from buffer, limit later
		allEntries = append(allEntries, entries...)
	}

	// Sort by timestamp (most recent first)
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp.After(allEntries[j].Timestamp)
	})

	// Apply limit
	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	return allEntries
}

// GetAllErrors is deprecated: use GetAllLogsWithContext instead.
// Deprecated: This method exists for backward compatibility.
// Parameters:
//   - serviceName: filter to specific service (empty = all services)
//   - limit: maximum total errors to return (0 = default 50)
//   - contextLines: lines before/after each error (0-10, default 3)
//   - includeStderr: treat all stderr as errors (default true)
//   - since: only errors after this time (zero = no filter)
//
// Returns errors sorted by timestamp (most recent first).
func (lm *LogManager) GetAllErrors(serviceName string, limit int, contextLines int, includeStderr bool, since time.Time) []LogEntryWithContext {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Apply defaults
	if limit <= 0 {
		limit = 50
	}
	// Note: contextLines is clamped in LogBuffer.GetErrors
	// Negative values will be clamped to 0 there

	var allErrors []LogEntryWithContext

	for name, buffer := range lm.buffers {
		// Filter by service name if specified
		if serviceName != "" && name != serviceName {
			continue
		}

		errors := buffer.GetErrors(0, contextLines, includeStderr, since) // Get all from buffer, limit later
		allErrors = append(allErrors, errors...)
	}

	// Sort by timestamp (most recent first)
	sort.Slice(allErrors, func(i, j int) bool {
		return allErrors[i].Timestamp.After(allErrors[j].Timestamp)
	})

	// Apply limit
	if len(allErrors) > limit {
		allErrors = allErrors[:limit]
	}

	return allErrors
}

// GetServiceNames returns the names of all services with log buffers.
func (lm *LogManager) GetServiceNames() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	names := make([]string, 0, len(lm.buffers))
	for name := range lm.buffers {
		names = append(names, name)
	}
	return names
}

// RemoveBuffer removes a log buffer for a service.
func (lm *LogManager) RemoveBuffer(serviceName string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	buffer, exists := lm.buffers[serviceName]
	if !exists {
		return fmt.Errorf("no log buffer found for service: %s", serviceName)
	}

	// Close the buffer and clean up resources
	if err := buffer.Close(); err != nil {
		return fmt.Errorf("failed to close log buffer for %s: %w", serviceName, err)
	}

	delete(lm.buffers, serviceName)
	return nil
}

// Clear removes all log buffers.
func (lm *LogManager) Clear() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	var errs []error
	for name, buffer := range lm.buffers {
		if err := buffer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close buffer for %s: %w", name, err))
		}
	}

	lm.buffers = make(map[string]*LogBuffer)

	return errors.Join(errs...)
}

// ClearBuffer clears the entries in a specific service's buffer without removing it.
func (lm *LogManager) ClearBuffer(serviceName string) error {
	lm.mu.RLock()
	buffer, exists := lm.buffers[serviceName]
	lm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no log buffer found for service: %s", serviceName)
	}

	buffer.Clear()
	return nil
}

// SortLogEntries sorts log entries by timestamp (ascending).
func SortLogEntries(entries []LogEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
}
