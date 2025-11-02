package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogManager manages log buffers for all services in a project.
type LogManager struct {
	projectDir string
	buffers    map[string]*LogBuffer // key: serviceName
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
	}
	logManagers[absPath] = lm

	return lm
}

// CreateBuffer creates a log buffer for a service.
func (lm *LogManager) CreateBuffer(serviceName string, maxSize int, enableFileLogging bool) (*LogBuffer, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Return existing buffer if already created
	if buffer, exists := lm.buffers[serviceName]; exists {
		return buffer, nil
	}

	// Create new buffer
	buffer, err := NewLogBuffer(serviceName, maxSize, enableFileLogging, lm.projectDir)
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
	sortLogEntries(allLogs)

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
	sortLogEntries(allLogs)

	return allLogs
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

	if len(errs) > 0 {
		return fmt.Errorf("errors while clearing buffers: %v", errs)
	}
	return nil
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

// sortLogEntries sorts log entries by timestamp (ascending).
func sortLogEntries(entries []LogEntry) {
	// Simple bubble sort - fine for reasonable log sizes
	// For larger datasets, consider using sort.Slice
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Timestamp.After(entries[j].Timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
