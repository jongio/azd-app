// Package registry provides functionality for managing running service registrations.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

	// Handle empty file gracefully
	if len(data) == 0 {
		r.services = make(map[string]*ServiceRegistryEntry)
		return nil
	}

	// Unmarshal into a temporary map to preserve existing services on parse error
	var loaded map[string]*ServiceRegistryEntry
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}

	if loaded != nil {
		r.services = loaded
	} else {
		r.services = make(map[string]*ServiceRegistryEntry)
	}

	return nil
}

// Clear removes all entries from the registry.
func (r *ServiceRegistry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*ServiceRegistryEntry)
	return r.save()
}
