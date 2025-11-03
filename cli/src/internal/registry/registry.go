package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ServiceRegistryEntry represents a running service in the registry.
type ServiceRegistryEntry struct {
	Name        string    `json:"name"`
	ProjectDir  string    `json:"projectDir"`
	PID         int       `json:"pid"`
	Port        int       `json:"port"`
	URL         string    `json:"url"`
	AzureURL    string    `json:"azureUrl,omitempty"`
	Language    string    `json:"language"`
	Framework   string    `json:"framework"`
	Status      string    `json:"status"` // "starting", "ready", "stopping", "stopped", "error"
	Health      string    `json:"health"` // "healthy", "unhealthy", "unknown"
	StartTime   time.Time `json:"startTime"`
	LastChecked time.Time `json:"lastChecked"`
	Error       string    `json:"error,omitempty"`
}

// ServiceRegistry manages the registry of running services for a project.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*ServiceRegistryEntry // key: serviceName
	filePath string
}

var (
	registryCache   = make(map[string]*ServiceRegistry)
	registryCacheMu sync.RWMutex
)

// GetRegistry returns the service registry instance for the given project directory.
// If projectDir is empty, uses current working directory.
func GetRegistry(projectDir string) *ServiceRegistry {
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

	registryCacheMu.Lock()
	defer registryCacheMu.Unlock()

	if reg, exists := registryCache[absPath]; exists {
		return reg
	}

	registryDir := filepath.Join(absPath, ".azure")
	registryFile := filepath.Join(registryDir, "services.json")

	registry := &ServiceRegistry{
		services: make(map[string]*ServiceRegistryEntry),
		filePath: registryFile,
	}

	// Ensure directory exists
	if err := os.MkdirAll(registryDir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create registry directory: %v\n", err)
	}

	// Load existing registry
	if err := registry.load(); err != nil {
		// Ignore load errors on first run
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load service registry: %v\n", err)
		}
	}

	// Don't clean stale entries immediately on load - let services manage their own lifecycle
	// This prevents removing recently started services that haven't had their LastChecked updated yet

	registryCache[absPath] = registry
	return registry
}

// Register adds a service to the registry.
func (r *ServiceRegistry) Register(entry *ServiceRegistryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[entry.Name] = entry
	entry.LastChecked = time.Now()

	err := r.save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save registry for %s: %v\n", entry.Name, err)
	}
	return err
}

// Unregister removes a service from the registry.
func (r *ServiceRegistry) Unregister(serviceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.services, serviceName)

	return r.save()
}

// UpdateStatus updates the status of a service.
func (r *ServiceRegistry) UpdateStatus(serviceName, status, health string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, exists := r.services[serviceName]; exists {
		svc.Status = status
		svc.Health = health
		svc.LastChecked = time.Now()
		return r.save()
	}
	return fmt.Errorf("service not found: %s", serviceName)
}

// GetService retrieves a service entry.
func (r *ServiceRegistry) GetService(serviceName string) (*ServiceRegistryEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[serviceName]
	return entry, exists
}

// ListAll returns all registered services.
func (r *ServiceRegistry) ListAll() []*ServiceRegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ServiceRegistryEntry, 0, len(r.services))
	for _, entry := range r.services {
		result = append(result, entry)
	}
	return result
}

// save persists the registry to disk.
func (r *ServiceRegistry) save() error {
	data, err := json.MarshalIndent(r.services, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// load reads the registry from disk.
func (r *ServiceRegistry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	services := make(map[string]*ServiceRegistryEntry)
	if err := json.Unmarshal(data, &services); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	r.services = services
	return nil
}

// cleanStale removes entries for processes that are no longer running.
//
//nolint:unused // Kept for future use - will be used for automatic cleanup
func (r *ServiceRegistry) cleanStale() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// On Windows, process checking is unreliable, so we use a timeout-based approach
	// Remove entries that haven't been checked in over 1 hour
	timeout := time.Hour
	now := time.Now()

	for key, entry := range r.services {
		// If last checked is zero or very old, remove it
		if entry.LastChecked.IsZero() || now.Sub(entry.LastChecked) > timeout {
			delete(r.services, key)
			continue
		}

		// On non-Windows systems, we can do actual process checking
		if runtime.GOOS != "windows" && entry.PID > 0 {
			if !isProcessRunning(entry.PID) {
				delete(r.services, key)
			}
		}
	}

	// Save after cleanup
	_ = r.save()
}

// isProcessRunning checks if a process with the given PID is running.
// This only works reliably on Unix systems.
//
//nolint:unused // Kept for future use - will be used by cleanStale
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, Signal(0) is a standard way to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Clear removes all entries from the registry.
func (r *ServiceRegistry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*ServiceRegistryEntry)
	return r.save()
}
