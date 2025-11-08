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

// RegistryObserver is an interface for observing registry changes.
//
// Concurrency/Threading Model:
//   - OnServiceChanged will be called asynchronously from a separate goroutine.
//   - Implementations must be thread-safe.
//   - Observers must not call registry methods synchronously from OnServiceChanged to avoid potential deadlocks.
//   - The entry parameter is a copy, not the original.
type RegistryObserver interface {
	OnServiceChanged(entry *ServiceRegistryEntry)
}

// ServiceRegistry manages the registry of running services for a project.
type ServiceRegistry struct {
	mu         sync.RWMutex
	services   map[string]*ServiceRegistryEntry // key: serviceName
	filePath   string
	observers  []RegistryObserver
	observerMu sync.RWMutex
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
		services:  make(map[string]*ServiceRegistryEntry),
		filePath:  registryFile,
		observers: make([]RegistryObserver, 0),
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
		oldStatus, oldHealth := svc.Status, svc.Health
		svc.Status = status
		svc.Health = health
		svc.LastChecked = time.Now()

		if err := r.save(); err != nil {
			return err
		}

		// Notify observers only if status actually changed
		if oldStatus != status || oldHealth != health {
			// Create a copy to avoid sharing mutable state with observers
			entryCopy := *svc
			// NOTE: notifyObservers is called while holding the mu write lock.
			// This is safe because notifyObservers only copies the observer list,
			// and dispatches notifications asynchronously in separate goroutines.
			// IMPORTANT: Observers must not call registry methods synchronously
			// from OnServiceChanged to avoid deadlocks.
			r.notifyObservers(&entryCopy)
		}
		return nil
	}
	return fmt.Errorf("service not found: %s", serviceName)
}

// UpdateStatusWithError updates the status, health, and error message of a service.
func (r *ServiceRegistry) UpdateStatusWithError(serviceName, status, health, errorMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, exists := r.services[serviceName]; exists {
		oldStatus, oldHealth, oldError := svc.Status, svc.Health, svc.Error
		svc.Status = status
		svc.Health = health
		svc.Error = errorMsg
		svc.LastChecked = time.Now()

		if err := r.save(); err != nil {
			return err
		}

		// Notify observers if status, health, or error message changed
		if oldStatus != status || oldHealth != health || oldError != errorMsg {
			// Create a copy to avoid sharing mutable state with observers
			entryCopy := *svc
			r.notifyObservers(&entryCopy)
		}
		return nil
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

// Subscribe adds an observer to the registry. Duplicate subscriptions are prevented.
func (r *ServiceRegistry) Subscribe(observer RegistryObserver) {
	r.observerMu.Lock()
	defer r.observerMu.Unlock()
	
	// Prevent duplicate subscriptions
	for _, obs := range r.observers {
		if obs == observer {
			return
		}
	}
	r.observers = append(r.observers, observer)
}

// Unsubscribe removes an observer from the registry.
// Returns true if the observer was found and removed, false otherwise.
// Note: The same observer reference must be used for both Subscribe and Unsubscribe operations.
// Interface comparison checks both type and underlying value (memory address for pointers).
func (r *ServiceRegistry) Unsubscribe(observer RegistryObserver) bool {
	r.observerMu.Lock()
	defer r.observerMu.Unlock()

	removed := false
	newObservers := make([]RegistryObserver, 0, len(r.observers))
	for _, obs := range r.observers {
		if obs == observer {
			removed = true
			continue
		}
		newObservers = append(newObservers, obs)
	}
	r.observers = newObservers
	return removed
}

// notifyObservers notifies all observers of a service change.
func (r *ServiceRegistry) notifyObservers(entry *ServiceRegistryEntry) {
	r.observerMu.RLock()
	observers := make([]RegistryObserver, len(r.observers))
	copy(observers, r.observers)
	r.observerMu.RUnlock()

	// Notify each observer in a separate goroutine to avoid blocking
	// Passing the loop variable as a parameter ensures correct capture in each iteration
	// (required for Go < 1.22; optional for Go 1.22+)
	for _, obs := range observers {
		go func(observer RegistryObserver) {
			defer func() {
				if rec := recover(); rec != nil {
					fmt.Fprintf(
						os.Stderr,
						"Observer panic: %v (observer type: %T, service: %s)\n",
						rec, observer, entry.Name,
					)
				}
			}()
			observer.OnServiceChanged(entry)
		}(obs)
	}
}

// Clear removes all entries from the registry.
func (r *ServiceRegistry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*ServiceRegistryEntry)
	return r.save()
}
