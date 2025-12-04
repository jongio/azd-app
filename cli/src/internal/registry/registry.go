// Package registry provides functionality for managing running service registrations.
// NOTE: This package now uses in-memory storage only. No files are persisted.
// Service state is transient and only valid while the dashboard/orchestrator is running.
package registry

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"
)

// ServiceRegistryEntry represents a running service in the registry.
// NOTE: Health status is NOT stored here - it is computed dynamically via health checks.
// This prevents stale cached health data from causing issues with the dashboard.
type ServiceRegistryEntry struct {
	Name        string    `json:"name"`
	ProjectDir  string    `json:"projectDir"`
	PID         int       `json:"pid"`
	Port        int       `json:"port"`
	URL         string    `json:"url"`
	AzureURL    string    `json:"azureUrl,omitempty"`
	Language    string    `json:"language"`
	Framework   string    `json:"framework"`
	Status      string    `json:"status"` // "starting", "ready", "stopping", "stopped", "error", "building", "built", "completed", "failed", "watching"
	StartTime   time.Time `json:"startTime"`
	LastChecked time.Time `json:"lastChecked"`
	Error       string    `json:"error,omitempty"`
	Type        string    `json:"type,omitempty"`     // "http", "tcp", "process"
	Mode        string    `json:"mode,omitempty"`     // "watch", "build", "daemon", "task" (for type=process)
	ExitCode    *int      `json:"exitCode,omitempty"` // Exit code for completed build/task mode services (nil = still running)
	EndTime     time.Time `json:"endTime,omitempty"`  // When the process exited (for build/task modes)
}

// ServiceRegistry manages the registry of running services for a project.
// NOTE: This is an in-memory only registry. No files are persisted.
type ServiceRegistry struct {
	mu         sync.RWMutex
	services   map[string]*ServiceRegistryEntry // key: serviceName
	projectDir string
}

var (
	registryCache   = make(map[string]*ServiceRegistry)
	registryCacheMu sync.RWMutex
)

// GetRegistry returns the service registry instance for the given project directory.
// If projectDir is empty, uses current working directory.
// NOTE: Registry is in-memory only. No files are persisted or loaded.
func GetRegistry(projectDir string) *ServiceRegistry {
	if projectDir == "" {
		projectDir = "."
	}

	// Normalize path
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		absPath = projectDir
	}

	registryCacheMu.Lock()
	defer registryCacheMu.Unlock()

	if reg, exists := registryCache[absPath]; exists {
		return reg
	}

	registry := &ServiceRegistry{
		services:   make(map[string]*ServiceRegistryEntry),
		projectDir: absPath,
	}

	slog.Debug("created new in-memory registry", "projectDir", absPath)

	registryCache[absPath] = registry
	return registry
}

// Register adds a service to the registry.
// NOTE: In-memory only, no file persistence.
func (r *ServiceRegistry) Register(entry *ServiceRegistryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[entry.Name] = entry
	entry.LastChecked = time.Now()

	slog.Debug("registered service", "name", entry.Name, "status", entry.Status)
	return nil
}

// Unregister removes a service from the registry.
// NOTE: In-memory only, no file persistence.
func (r *ServiceRegistry) Unregister(serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.services, serviceName)

	slog.Debug("unregistered service", "name", serviceName)
	return nil
}

// UpdateStatus updates the lifecycle status of a service.
// NOTE: Health is not stored in registry - it is computed dynamically via health checks.
func (r *ServiceRegistry) UpdateStatus(serviceName, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, exists := r.services[serviceName]; exists {
		svc.Status = status
		svc.LastChecked = time.Now()
		slog.Debug("updated service status", "name", serviceName, "status", status)
		return nil
	}
	return fmt.Errorf("service not found: %s", serviceName)
}

// UpdateExitInfo updates the exit code and end time for a completed service.
// This is used for build/task mode services that complete and exit.
func (r *ServiceRegistry) UpdateExitInfo(serviceName string, exitCode int, endTime time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, exists := r.services[serviceName]; exists {
		svc.ExitCode = &exitCode
		svc.EndTime = endTime
		svc.LastChecked = time.Now()
		slog.Debug("updated service exit info", "name", serviceName, "exitCode", exitCode)
		return nil
	}
	return fmt.Errorf("service not found: %s", serviceName)
}

// GetService retrieves a service entry.
// Returns a copy of the entry to prevent race conditions on concurrent access.
func (r *ServiceRegistry) GetService(serviceName string) (*ServiceRegistryEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[serviceName]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent data races when caller modifies the entry
	copy := *entry
	return &copy, true
}

// ListAll returns all registered services.
// Returns copies of the entries to prevent race conditions on concurrent access.
func (r *ServiceRegistry) ListAll() []*ServiceRegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ServiceRegistryEntry, 0, len(r.services))
	for _, entry := range r.services {
		// Return copies to prevent data races
		copy := *entry
		result = append(result, &copy)
	}
	return result
}

// Clear removes all entries from the registry.
// NOTE: In-memory only, no file persistence.
func (r *ServiceRegistry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*ServiceRegistryEntry)
	slog.Debug("cleared registry")
	return nil
}
