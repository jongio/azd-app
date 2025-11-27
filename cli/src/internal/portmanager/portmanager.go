// Package portmanager provides port allocation, management, and process monitoring capabilities.
package portmanager

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/executor"
)

const (
	// Port scan limits
	// maxPortScanAttempts limits port scanning to prevent excessive delays.
	// 100 attempts in a 3000-65535 range gives ~0.15% coverage which is sufficient
	// for finding available ports while avoiding full range scans.
	maxPortScanAttempts = 100

	// Timeouts
	// killProcessTimeout allows processes time to shutdown before force-kill.
	// 5 seconds is generous for most processes to handle SIGKILL.
	killProcessTimeout = 5 * time.Second

	// processCleanupWait gives the OS time to release port resources after kill.
	// 1 second initial wait + retries accounts for TIME_WAIT state and process cleanup,
	// especially for Windows system processes that may take longer to release ports.
	processCleanupWait = 1 * time.Second

	// portCleanupRetries is the number of retry attempts when verifying port cleanup.
	// Each retry waits an additional 500ms, giving up to 3 seconds total for cleanup.
	portCleanupRetries = 4

	// portCleanupRetryWait is the additional wait time between retry attempts.
	portCleanupRetryWait = 500 * time.Millisecond

	// Cache limits
	// maxCacheSize prevents unbounded memory growth in long-running processes.
	// 50 projects × ~1KB each = ~50KB max overhead, which is negligible.
	// LRU eviction ensures active projects stay cached.
	maxCacheSize = 50

	// Environment variables for configuration
	envPortRangeStart = "AZD_PORT_RANGE_START"
	envPortRangeEnd   = "AZD_PORT_RANGE_END"
)

// PortAssignment represents a port assignment for a service.
type PortAssignment struct {
	ServiceName string    `json:"serviceName"`
	Port        int       `json:"port"`
	LastUsed    time.Time `json:"lastUsed"`
}

// PortReservation holds a port open to prevent TOCTOU race conditions.
// Call Release() just before your service binds to the port.
type PortReservation struct {
	Port     int
	listener net.Listener
	released bool
	mu       sync.Mutex
}

// Release closes the reservation listener, freeing the port for binding.
// This should be called immediately before your service binds to the port.
// Safe to call multiple times.
func (r *PortReservation) Release() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.released || r.listener == nil {
		return nil
	}

	r.released = true
	return r.listener.Close()
}

// PortManager manages port assignments for services.
type PortManager struct {
	mu          sync.RWMutex
	assignments map[string]*PortAssignment // key: serviceName
	filePath    string
	portRange   struct {
		start int
		end   int
	}
	// portChecker is a function that checks if a port is available
	// This can be overridden in tests to avoid network binding
	portChecker func(port int) bool
}

// cacheEntry holds a port manager with LRU tracking
type cacheEntry struct {
	manager  *PortManager
	lastUsed time.Time
}

var (
	managerCache   = make(map[string]*cacheEntry)
	managerCacheMu sync.RWMutex
)

// GetPortManager returns a cached port manager instance for the given project directory.
// If projectDir is empty, uses the current working directory.
//
// Thread-safety: This function is safe for concurrent use across goroutines.
// The returned PortManager uses internal locking for most operations.
//
// Caching: Port managers are cached per absolute path with an LRU policy (max 50 entries).
// Multiple calls with the same projectDir (after normalization) return the same instance.
// The cache is per-process; different azd processes do not share cached instances.
//
// Note: The cache helps with performance in long-running processes but does not provide
// cross-process synchronization. File-based persistence handles multi-process coordination.
func GetPortManager(projectDir string) *PortManager {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			slog.Warn("failed to get working directory, using current directory", "error", err)
			projectDir = "."
		} else {
			projectDir = cwd
		}
	}

	// Normalize path - fail fast if path cannot be resolved
	absPath, err := filepath.Abs(projectDir)
	if err != nil {
		slog.Error("failed to resolve absolute path", "path", projectDir, "error", err)
		// Fall back to using the provided path, but log the issue
		absPath = projectDir
	}

	slog.Debug("getting port manager", "path", projectDir, "normalized", absPath)

	managerCacheMu.Lock()
	defer managerCacheMu.Unlock()

	// Check cache and update last used time
	if entry, exists := managerCache[absPath]; exists {
		entry.lastUsed = time.Now()
		slog.Debug("returning cached port manager", "path", absPath)
		return entry.manager
	}

	// Evict oldest entry if cache is full
	if len(managerCache) >= maxCacheSize {
		evictOldestCacheEntry()
	}

	slog.Debug("creating new port manager", "path", absPath)

	portsDir := filepath.Join(absPath, ".azure")
	portsFile := filepath.Join(portsDir, "ports.json")

	manager := &PortManager{
		assignments: make(map[string]*PortAssignment),
		filePath:    portsFile,
	}

	// Configure port range from environment or use defaults
	manager.portRange.start = getPortRangeStart()
	manager.portRange.end = getPortRangeEnd()
	slog.Debug("port range configured", "start", manager.portRange.start, "end", manager.portRange.end)

	// Set default port checker (can be overridden in tests)
	manager.portChecker = manager.defaultIsPortAvailable

	// Ensure directory exists with strict permissions matching the file (0700)
	if err := os.MkdirAll(portsDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create ports directory: %v\n", err)
	}

	// Load existing assignments
	if err := manager.load(); err != nil && !os.IsNotExist(err) {
		slog.Warn("failed to load port assignments", "error", err, "path", portsFile)
	}

	managerCache[absPath] = &cacheEntry{
		manager:  manager,
		lastUsed: time.Now(),
	}
	return manager
}

// evictOldestCacheEntry removes the least recently used entry from the cache.
// Must be called with managerCacheMu held.
func evictOldestCacheEntry() {
	var oldestPath string
	var oldestTime time.Time

	for path, entry := range managerCache {
		if oldestPath == "" || entry.lastUsed.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.lastUsed
		}
	}

	if oldestPath != "" {
		slog.Debug("evicting port manager from cache", "path", oldestPath, "lastUsed", oldestTime)
		delete(managerCache, oldestPath)
	}
}

// getPortRangeStart returns the configured port range start or default.
func getPortRangeStart() int {
	if val := os.Getenv(envPortRangeStart); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port <= 65535 {
			slog.Info("using custom port range start", "port", port)
			return port
		}
		slog.Warn("invalid port range start, using default", "value", val)
	}
	return 3000 // Default: avoid well-known and registered ports
}

// getPortRangeEnd returns the configured port range end or default.
func getPortRangeEnd() int {
	if val := os.Getenv(envPortRangeEnd); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 && port <= 65535 {
			slog.Info("using custom port range end", "port", port)
			return port
		}
		slog.Warn("invalid port range end, using default", "value", val)
	}
	return 65535 // Default: maximum valid port
}

// AssignPort assigns or retrieves a port for a service.
//
// Parameters:
//   - serviceName: The unique name of the service (must not be empty)
//   - preferredPort: The desired port number. For explicit ports, must be 1-65535.
//   - isExplicit: If true, the port came from azure.yaml config and MUST be used (never changed).
//     If the explicit port is unavailable, the user will be prompted to either kill the
//     existing process or choose a different port.
//
// Returns:
//   - port: The assigned port number (guaranteed to be in the valid range 3000-65535)
//   - wasAutoAssigned: True if the user was prompted and chose to auto-assign a different port.
//     This signals that azure.yaml should be updated with the new port.
//   - error: Non-nil if the assignment failed (validation error, user cancelled, no ports available)
//
// Port range: 3000-65535
//   - Minimum 3000: Avoids well-known ports (0-1023) and registered ports (1024-2999)
//     which often require admin privileges and conflict with system services.
//   - Maximum 65535: Standard TCP/IP limit for port numbers.
//
// Behavior:
//   - EXPLICIT MODE (isExplicit=true): Port is mandatory. If unavailable, prompts user.
//   - FLEXIBLE MODE (isExplicit=false): Port is preferred but can be changed automatically.
//     If unavailable, finds an alternative port without user interaction.
//
// The assigned port is persisted to .azure/ports.json for consistency across runs.
//
// Thread-safety:
// This function uses internal locking but TEMPORARILY RELEASES THE LOCK when prompting
// for user input to prevent deadlocks. During this window, other goroutines can modify
// the port assignments. DO NOT call this function concurrently for the same serviceName.
// Concurrent calls for different services are safe.
//
// TOCTOU Race Condition:
// There is a Time-Of-Check-Time-Of-Use race between checking port availability and the
// caller binding to it. Another process could bind to the port in the interim. Callers
// MUST handle port binding failures gracefully and may retry by calling AssignPort again.
func (pm *PortManager) AssignPort(serviceName string, preferredPort int, isExplicit bool) (int, bool, error) {
	// Validate inputs
	if serviceName == "" {
		return 0, false, fmt.Errorf("serviceName cannot be empty")
	}
	if isExplicit && (preferredPort <= 0 || preferredPort > 65535) {
		return 0, false, fmt.Errorf("explicit port must be between 1-65535, got %d", preferredPort)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// EXPLICIT PORT MODE: Port from azure.yaml - MUST be used, prompt if in use
	if isExplicit {
		// Validate port is in range
		if preferredPort < pm.portRange.start || preferredPort > pm.portRange.end {
			return 0, false, fmt.Errorf("explicit port %d for service '%s' is outside valid range %d-%d",
				preferredPort, serviceName, pm.portRange.start, pm.portRange.end)
		}

		// Check if port is available
		if pm.isPortAvailable(preferredPort) {
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			return preferredPort, false, nil
		}

		// Port is in use - prompt user with options
		// Try to get process info to help user
		processInfo := ""
		if info, err := pm.getProcessInfoOnPort(preferredPort); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" by %s (PID %d)", info.Name, info.PID)
			} else {
				processInfo = fmt.Sprintf(" by PID %d", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' requires port %d (configured in azure.yaml)\n", serviceName, preferredPort)
		fmt.Fprintf(os.Stderr, "This port is currently in use%s.\n\n", processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", preferredPort)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		// Release mutex before blocking on user input to prevent deadlocks
		pm.mu.Unlock()
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		pm.mu.Lock()

		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(preferredPort); err != nil {
				return 0, false, fmt.Errorf("failed to free port %d: %w", preferredPort, err)
			}

			// Verify port is now available with retries
			if !pm.verifyPortCleanup(preferredPort) {
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", preferredPort)
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			return preferredPort, false, nil

		case "2":
			// Assign random port
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)
			port, err := pm.findAvailablePort()
			if err != nil {
				return 0, false, err
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        port,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\n✓ Assigned port %d to service '%s'\n", port, serviceName)
			fmt.Fprintf(os.Stderr, "\n⚠️  IMPORTANT: Update your application code to use port %d\n", port)
			fmt.Fprintf(os.Stderr, "Would you like to update azure.yaml to use port %d for future runs? (y/N): ", port)

			// Release mutex before blocking on user input
			pm.mu.Unlock()
			updateResponse, err := reader.ReadString('\n')
			pm.mu.Lock()
			if err == nil {
				updateResponse = strings.TrimSpace(strings.ToLower(updateResponse))
				if updateResponse == "y" || updateResponse == "yes" {
					return port, true, nil // Signal caller to update azure.yaml
				}
			}

			return port, false, nil

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// FLEXIBLE PORT MODE: Port can be changed if needed, prompt user when conflicts detected

	// Check if we already have an assignment
	if assignment, exists := pm.assignments[serviceName]; exists {
		assignment.LastUsed = time.Now()

		slog.Debug("checking assigned port", "service", serviceName, "port", assignment.Port)

		// Check if assigned port is available
		if pm.isPortAvailable(assignment.Port) {
			slog.Debug("assigned port is available", "service", serviceName, "port", assignment.Port)
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			return assignment.Port, false, nil
		}

		// Previously assigned port is now in use - prompt user
		slog.Debug("assigned port is in use", "service", serviceName, "port", assignment.Port)

		// Try to get process info to help user
		processInfo := ""
		assignedPort := assignment.Port
		if info, err := pm.getProcessInfoOnPort(assignedPort); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" (PID %d: %s)", info.PID, info.Name)
			} else {
				processInfo = fmt.Sprintf(" (PID %d)", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' port %d is already in use%s\n", serviceName, assignedPort, processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", assignedPort)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		// Release mutex before blocking on user input to prevent deadlocks
		pm.mu.Unlock()
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		pm.mu.Lock()

		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(assignedPort); err != nil {
				fmt.Fprintf(os.Stderr, "\n⚠️  %v\n", err)
				fmt.Fprintf(os.Stderr, "\nTip: Choose option 2 to find a different available port\n\n")
				return 0, false, fmt.Errorf("failed to free port %d: %w", assignedPort, err)
			}

			// Verify port is now available with retries
			if !pm.verifyPortCleanup(assignedPort) {
				fmt.Fprintf(os.Stderr, "\n⚠️  Port %d is still in use after cleanup\n", assignedPort)
				fmt.Fprintf(os.Stderr, "Tip: Choose option 2 to find a different available port\n\n")
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", assignedPort)
			}

			// Keep the same port assignment
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Port %d freed and ready for service '%s'\n\n", assignedPort, serviceName)
			return assignedPort, false, nil

		case "2":
			// Find alternative port
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)
			port, err := pm.findAvailablePort()
			if err != nil {
				return 0, false, err
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        port,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✓ Assigned port %d to service '%s'\n\n", port, serviceName)
			return port, false, nil

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// Try preferred port first (if provided)
	if preferredPort >= pm.portRange.start && preferredPort <= pm.portRange.end {
		slog.Debug("checking preferred port", "service", serviceName, "port", preferredPort)

		if pm.isPortAvailable(preferredPort) {
			slog.Debug("preferred port is available", "service", serviceName, "port", preferredPort)
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			return preferredPort, false, nil
		}

		// Preferred port unavailable - prompt user
		slog.Debug("preferred port is in use", "service", serviceName, "port", preferredPort)

		// Try to get process info to help user
		processInfo := ""
		if info, err := pm.getProcessInfoOnPort(preferredPort); err == nil {
			if info.Name != "" {
				processInfo = fmt.Sprintf(" (PID %d: %s)", info.PID, info.Name)
			} else {
				processInfo = fmt.Sprintf(" (PID %d)", info.PID)
			}
		}

		fmt.Fprintf(os.Stderr, "\n⚠️  Service '%s' preferred port %d is already in use%s\n", serviceName, preferredPort, processInfo)
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  1) Kill the process using port %d\n", preferredPort)
		fmt.Fprintf(os.Stderr, "  2) Assign a different port automatically\n")
		fmt.Fprintf(os.Stderr, "  3) Cancel\n\n")
		fmt.Fprintf(os.Stderr, "Choose (1/2/3): ")

		// Release mutex before blocking on user input to prevent deadlocks
		pm.mu.Unlock()
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		pm.mu.Lock()

		if err != nil {
			return 0, false, fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(response)
		switch response {
		case "1":
			// Kill process
			if err := pm.killProcessOnPort(preferredPort); err != nil {
				fmt.Fprintf(os.Stderr, "\n⚠️  %v\n", err)
				fmt.Fprintf(os.Stderr, "\nTip: Choose option 2 to use a different port automatically\n\n")
				return 0, false, fmt.Errorf("failed to free port %d: %w", preferredPort, err)
			}

			// Verify port is now available with retries
			if !pm.verifyPortCleanup(preferredPort) {
				fmt.Fprintf(os.Stderr, "\n⚠️  Port %d is still in use after cleanup\n", preferredPort)
				fmt.Fprintf(os.Stderr, "Tip: Choose option 2 to use a different port automatically\n\n")
				return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", preferredPort)
			}

			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			if err := pm.save(); err != nil {
				return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Port %d freed and assigned to service '%s'\n\n", preferredPort, serviceName)
			return preferredPort, false, nil

		case "2":
			// Find alternative port (fall through to auto-assign below)
			fmt.Fprintf(os.Stderr, "\nFinding available port for '%s'...\n", serviceName)

		default:
			return 0, false, fmt.Errorf("operation cancelled by user")
		}
	}

	// Find an available port automatically
	port, err := pm.findAvailablePort()
	if err != nil {
		return 0, false, err
	}

	pm.assignments[serviceName] = &PortAssignment{
		ServiceName: serviceName,
		Port:        port,
		LastUsed:    time.Now(),
	}
	if err := pm.save(); err != nil {
		return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
	}

	// Notify user about auto-assigned port
	fmt.Fprintf(os.Stderr, "✓ Auto-assigned port %d to service '%s'\n\n", port, serviceName)

	return port, false, nil
}

// ReleasePort removes a port assignment.
func (pm *PortManager) ReleasePort(serviceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.assignments, serviceName)
	return pm.save()
}

// GetAssignment returns the port assignment for a service.
func (pm *PortManager) GetAssignment(serviceName string) (int, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if assignment, exists := pm.assignments[serviceName]; exists {
		return assignment.Port, true
	}
	return 0, false
}

// CleanStalePorts removes assignments for ports that haven't been used in over 7 days.
func (pm *PortManager) CleanStalePorts() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for name, assignment := range pm.assignments {
		if assignment.LastUsed.Before(cutoff) {
			delete(pm.assignments, name)
		}
	}
	if err := pm.save(); err != nil {
		return fmt.Errorf("failed to save port assignments after cleanup: %w", err)
	}
	return nil
}

// IsPortAvailable checks if a port is available for binding.
// This is a public wrapper around the internal port checking logic.
func (pm *PortManager) IsPortAvailable(port int) bool {
	return pm.isPortAvailable(port)
}

// ReservePort attempts to reserve a port by binding to it.
// This eliminates TOCTOU race conditions by holding the port open until
// the caller is ready to bind their service.
//
// Usage:
//
//	reservation, err := pm.ReservePort(8080)
//	if err != nil {
//	    // Port not available, try another
//	}
//	defer reservation.Release() // Always release, even on error paths
//
//	// Immediately before starting service:
//	reservation.Release()
//	service.Start() // Must bind quickly after release
//
// Returns:
//   - *PortReservation: Holds the port open. Call Release() before binding.
//   - error: Non-nil if port cannot be reserved
func (pm *PortManager) ReservePort(port int) (*PortReservation, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("port %d is not available: %w", port, err)
	}

	return &PortReservation{
		Port:     port,
		listener: listener,
		released: false,
	}, nil
}

// FindAndReservePort finds an available port and reserves it atomically.
// This combines port finding and reservation to eliminate TOCTOU races.
//
// Returns:
//   - *PortReservation: Holds the port open. Call Release() before binding.
//   - error: Non-nil if no port can be reserved after max attempts
func (pm *PortManager) FindAndReservePort(serviceName string, preferredPort int) (*PortReservation, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Build map of assigned ports to avoid duplicates
	assignedPorts := make(map[int]bool)
	for _, assignment := range pm.assignments {
		assignedPorts[assignment.Port] = true
	}

	// Try preferred port first
	if preferredPort >= pm.portRange.start && preferredPort <= pm.portRange.end && !assignedPorts[preferredPort] {
		if reservation, err := pm.ReservePort(preferredPort); err == nil {
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        preferredPort,
				LastUsed:    time.Now(),
			}
			_ = pm.save()
			return reservation, nil
		}
	}

	// Calculate port range size
	rangeSize := pm.portRange.end - pm.portRange.start + 1
	if rangeSize <= 0 {
		return nil, fmt.Errorf("invalid port range: %d-%d", pm.portRange.start, pm.portRange.end)
	}

	// Randomize starting point
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		nBig = big.NewInt(0)
	}
	startOffset := int(nBig.Int64())

	// Try to find and reserve a port
	for attempt := 0; attempt < maxPortScanAttempts && attempt < rangeSize; attempt++ {
		port := pm.portRange.start + ((startOffset + attempt) % rangeSize)

		if assignedPorts[port] {
			continue
		}

		if reservation, err := pm.ReservePort(port); err == nil {
			pm.assignments[serviceName] = &PortAssignment{
				ServiceName: serviceName,
				Port:        port,
				LastUsed:    time.Now(),
			}
			_ = pm.save()
			return reservation, nil
		}
	}

	return nil, fmt.Errorf("no available ports found after %d attempts", maxPortScanAttempts)
}

// isPortAvailable checks if a port is available by attempting to bind to it.
//
// IMPORTANT - TOCTOU Race Condition:
// This check is inherently racy. Between checking and using the port, another process
// can bind to it. Callers should:
// 1. Call this to find a candidate port
// 2. Attempt to bind to the port immediately
// 3. Handle bind failures gracefully (possibly by calling AssignPort again)
//
// This is a fundamental limitation of port allocation and cannot be fully eliminated
// without holding the port open, which would prevent the caller from using it.
func (pm *PortManager) isPortAvailable(port int) bool {
	if pm.portChecker != nil {
		return pm.portChecker(port)
	}
	return pm.defaultIsPortAvailable(port)
}

// defaultIsPortAvailable is the default implementation that actually binds to check port availability.
func (pm *PortManager) defaultIsPortAvailable(port int) bool {
	// Bind to localhost to avoid Windows Firewall prompts
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Debug("port bind test failed", "port", port, "error", err)
		return false
	}
	if err := listener.Close(); err != nil {
		// This is a benign error - port availability was confirmed
		slog.Debug("failed to close listener during availability check", "port", port, "error", err)
	}
	slog.Debug("port is available", "port", port)
	return true
}

// verifyPortCleanup verifies that a port is available after cleanup, with retries.
// This is necessary on Windows where port release can take longer, especially for system processes.
func (pm *PortManager) verifyPortCleanup(port int) bool {
	for attempt := 0; attempt < portCleanupRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying port cleanup verification", "port", port, "attempt", attempt+1)
			time.Sleep(portCleanupRetryWait)
		}

		if pm.isPortAvailable(port) {
			if attempt > 0 {
				slog.Debug("port became available after retry", "port", port, "attempts", attempt+1)
			}
			return true
		}
	}

	slog.Debug("port still in use after all retry attempts", "port", port, "attempts", portCleanupRetries)
	return false
}

// findAvailablePort finds an available port in the port range.
// Uses cryptographically secure randomized starting point with bounded attempts to:
// 1. Reduce collision probability when multiple services start simultaneously
// 2. Avoid exhaustive scanning of the entire port range
// 3. Prevent predictable port allocation patterns
func (pm *PortManager) findAvailablePort() (int, error) {
	// Build map of assigned ports to avoid duplicates
	assignedPorts := make(map[int]bool)
	for _, assignment := range pm.assignments {
		assignedPorts[assignment.Port] = true
	}

	// Calculate port range size
	rangeSize := pm.portRange.end - pm.portRange.start + 1
	if rangeSize <= 0 {
		return 0, fmt.Errorf("invalid port range: %d-%d", pm.portRange.start, pm.portRange.end)
	}

	// Randomize starting point using crypto/rand for security
	// This prevents predictable port allocation patterns that could be exploited
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		// Fallback to sequential search from start if crypto/rand fails
		slog.Warn("failed to generate secure random offset, using sequential search", "error", err)
		nBig = big.NewInt(0)
	}
	startOffset := int(nBig.Int64())

	// Try maxPortScanAttempts ports starting from random position
	for attempt := 0; attempt < maxPortScanAttempts && attempt < rangeSize; attempt++ {
		// Wrap around the range using modulo arithmetic
		port := pm.portRange.start + ((startOffset + attempt) % rangeSize)

		if assignedPorts[port] {
			continue
		}
		if pm.isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports found after %d attempts in range %d-%d", maxPortScanAttempts, pm.portRange.start, pm.portRange.end)
}

// ProcessInfo contains information about a process using a port.
type ProcessInfo struct {
	PID  int
	Name string
}

// getProcessInfoOnPort retrieves the PID and name of the process listening on the specified port.
func (pm *PortManager) getProcessInfoOnPort(port int) (*ProcessInfo, error) {
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		return nil, err
	}

	name, _ := pm.getProcessName(pid) // Ignore error, we'll use PID only if name lookup fails
	return &ProcessInfo{PID: pid, Name: name}, nil
}

// getProcessOnPort retrieves the PID of the process listening on the specified port.
func (pm *PortManager) getProcessOnPort(port int) (int, error) {
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port number: %d (must be 1-65535)", port)
	}

	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use netstat to find PID
		cmd = "powershell"
		psScript := fmt.Sprintf(`
			$connections = netstat -ano | Select-String ":%d " | Select-String "LISTENING"
			foreach ($line in $connections) {
				$parts = $line -split '\s+' | Where-Object { $_ }
				$procId = $parts[-1]
				if ($procId -match '^\d+$') {
					Write-Output $procId
					exit 0
				}
			}
		`, port)
		args = []string{"-Command", psScript}
	} else {
		// Unix: use lsof to find PID
		cmd = "sh"
		args = []string{"-c", fmt.Sprintf("lsof -ti:%d | head -n 1", port)}
	}

	// Execute command to get PID
	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), port is validated int
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get process on port %d: %w", port, err)
	}

	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, fmt.Errorf("no process found on port %d", port)
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID '%s': %w", pidStr, err)
	}

	return pid, nil
}

// getProcessName retrieves the process name for a given PID.
func (pm *PortManager) getProcessName(pid int) (string, error) {
	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use tasklist to get process name
		cmd = "powershell"
		psScript := fmt.Sprintf(`
			$proc = Get-Process -Id %d -ErrorAction SilentlyContinue
			if ($proc) {
				Write-Output $proc.ProcessName
			}
		`, pid)
		args = []string{"-Command", psScript}
	} else {
		// Unix: use ps to get process name
		cmd = "sh"
		args = []string{"-c", fmt.Sprintf("ps -p %d -o comm=", pid)}
	}

	// Execute command to get process name
	// #nosec G204 -- cmd is either "powershell" or "sh" (hard-coded), pid is validated int
	output, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process name for PID %d: %w", pid, err)
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return "", fmt.Errorf("no process name found for PID %d", pid)
	}

	return name, nil
}

// killProcessOnPort kills any process listening on the specified port.
func (pm *PortManager) killProcessOnPort(port int) error {
	// Get the PID first so we can provide feedback
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		// Port might not be in use anymore
		return nil
	}

	// Log without exposing too much system info to prevent information disclosure
	slog.Info("terminating process on port", "port", port, "pid", pid)

	var cmd []string
	var args []string

	if runtime.GOOS == "windows" {
		// Windows: use taskkill
		cmd = []string{"powershell", "-Command"}
		psScript := fmt.Sprintf("Stop-Process -Id %d -Force -ErrorAction SilentlyContinue", pid)
		args = append(cmd, psScript)
	} else {
		// Unix: use kill
		cmd = []string{"sh", "-c"}
		shScript := fmt.Sprintf("kill -9 %d 2>/dev/null || true", pid)
		args = append(cmd, shScript)
	}

	// Execute the kill command
	ctx, cancel := context.WithTimeout(context.Background(), killProcessTimeout)
	defer cancel()
	// #nosec G204 -- Command injection safe: cmd is hard-coded ("powershell" or "sh"),
	// and PID is validated integer from strconv.Atoi in getProcessOnPort (no user input)
	if err := executor.RunCommand(ctx, cmd[0], args[1:], "."); err != nil {
		// Log error but don't fail - process might have already exited
		slog.Debug("kill command completed with error", "pid", pid, "error", err)
	}

	// Wait a moment for process to die
	time.Sleep(processCleanupWait)

	// Verify the process was actually killed
	// This is critical for protected/system processes that cannot be terminated
	if stillRunningPid, err := pm.getProcessOnPort(port); err == nil && stillRunningPid == pid {
		processName, _ := pm.getProcessName(pid)
		slog.Warn("process could not be terminated - likely a protected system process",
			"port", port, "pid", pid, "name", processName)
		return fmt.Errorf("process %d (%s) could not be terminated - it may be a protected system process or require administrator privileges",
			pid, processName)
	}

	return nil
}

// load reads port assignments from disk.
func (pm *PortManager) load() error {
	data, err := os.ReadFile(pm.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &pm.assignments)
}

// save writes port assignments to disk using atomic write (temp file + rename).
// This prevents file corruption when multiple processes write concurrently.
func (pm *PortManager) save() error {
	// Use MarshalIndent for human-readable output - the file is small (<1KB typically)
	// and saved infrequently, so the performance impact is negligible
	data, err := json.MarshalIndent(pm.assignments, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port assignments: %w", err)
	}

	// Atomic write: write to temp file, then rename
	// This ensures the file is never in a partially-written state
	tmpFile := pm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename is atomic on POSIX systems (and Windows with proper flags)
	if err := os.Rename(tmpFile, pm.filePath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
