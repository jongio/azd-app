// Package portmanager provides port allocation, management, and process monitoring capabilities.
package portmanager

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
)

// PortManager manages port assignments for services.
type PortManager struct {
	mu          sync.RWMutex
	assignments map[string]*PortAssignment // key: serviceName
	projectDir  string                     // absolute path to project directory
	projectHash string                     // hash of projectDir for config keys
	portRange   struct {
		start int
		end   int
	}
	// portChecker is a function that checks if a port is available
	// This can be overridden in tests to avoid network binding
	portChecker func(port int) bool
	// configClient is lazily initialized for azdconfig access
	configClient azdconfig.ConfigClient
}

// cacheEntry holds a port manager with LRU tracking
type cacheEntry struct {
	manager  *PortManager
	lastUsed time.Time
}

// preferenceAlwaysKillPortConflicts is the config key for the always-kill preference.
const preferenceAlwaysKillPortConflicts = "alwaysKillPortConflicts"

var (
	managerCache   = make(map[string]*cacheEntry)
	managerCacheMu sync.RWMutex

	// sharedInMemoryClient is used when gRPC is not available (e.g., during tests).
	// This ensures all port managers share the same in-memory storage for consistency.
	sharedInMemoryClient     azdconfig.ConfigClient
	sharedInMemoryClientOnce sync.Once

	// globalTestPortChecker is set by tests to bypass real port checking.
	// When set, all new port managers will use this instead of the default checker.
	globalTestPortChecker func(int) bool
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

	// Resolve symlinks to ensure consistent caching across different path representations.
	// This is critical on macOS where temp directories use symlinks (e.g., /var -> /private/var).
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
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

	manager := &PortManager{
		assignments: make(map[string]*PortAssignment),
		projectDir:  absPath,
		projectHash: azdconfig.ProjectHash(absPath),
	}

	// Configure port range from environment or use defaults
	manager.portRange.start = getPortRangeStart()
	manager.portRange.end = getPortRangeEnd()
	slog.Debug("port range configured", "start", manager.portRange.start, "end", manager.portRange.end)

	// Set port checker - use global test checker if set, otherwise default
	if globalTestPortChecker != nil {
		manager.portChecker = globalTestPortChecker
	} else {
		manager.portChecker = manager.defaultIsPortAvailable
	}

	// Load existing assignments from azdconfig
	if err := manager.load(); err != nil {
		slog.Warn("failed to load port assignments from config", "error", err)
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

// ClearCacheForTesting clears the port manager cache.
// This is only intended for use in tests to ensure clean state between test runs.
func ClearCacheForTesting() {
	managerCacheMu.Lock()
	defer managerCacheMu.Unlock()
	managerCache = make(map[string]*cacheEntry)
	// Also reset the shared in-memory client for clean test isolation
	sharedInMemoryClient = nil
	sharedInMemoryClientOnce = sync.Once{}
}

// SetTestModeForTesting enables test mode where all ports appear available.
// This prevents interactive prompts during tests when real ports are in use.
// Call with nil to restore normal behavior.
// Returns a cleanup function that should be deferred to restore normal mode.
func SetTestModeForTesting(portChecker func(int) bool) func() {
	managerCacheMu.Lock()
	defer managerCacheMu.Unlock()

	oldChecker := globalTestPortChecker
	globalTestPortChecker = portChecker

	// Return cleanup function
	return func() {
		managerCacheMu.Lock()
		defer managerCacheMu.Unlock()
		globalTestPortChecker = oldChecker
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
		return pm.assignExplicitPort(serviceName, preferredPort)
	}

	// FLEXIBLE PORT MODE: Port can be changed if needed, prompt user when conflicts detected
	return pm.assignFlexiblePort(serviceName, preferredPort)
}

// assignExplicitPort handles port assignment when the port is explicit (from azure.yaml).
// Must be called with pm.mu held. May temporarily release the lock for user input.
func (pm *PortManager) assignExplicitPort(serviceName string, port int) (int, bool, error) {
	// Validate port is in range
	if port < pm.portRange.start || port > pm.portRange.end {
		return 0, false, fmt.Errorf("explicit port %d for service '%s' is outside valid range %d-%d",
			port, serviceName, pm.portRange.start, pm.portRange.end)
	}

	// Check if port is available
	if pm.isPortAvailable(port) {
		return pm.saveAssignment(serviceName, port, false)
	}

	// Port is in use - handle conflict
	processInfo := getProcessInfoString(pm, port)
	return pm.handleConflictAndAssign(serviceName, port, processInfo, true)
}

// assignFlexiblePort handles port assignment when the port is flexible (can be changed).
// Must be called with pm.mu held. May temporarily release the lock for user input.
func (pm *PortManager) assignFlexiblePort(serviceName string, preferredPort int) (int, bool, error) {
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

		// Previously assigned port is now in use - handle conflict
		slog.Debug("assigned port is in use", "service", serviceName, "port", assignment.Port)
		processInfo := getProcessInfoString(pm, assignment.Port)
		return pm.handleConflictAndAssign(serviceName, assignment.Port, processInfo, false)
	}

	// Try preferred port first (if provided and in range)
	if preferredPort >= pm.portRange.start && preferredPort <= pm.portRange.end {
		slog.Debug("checking preferred port", "service", serviceName, "port", preferredPort)

		if pm.isPortAvailable(preferredPort) {
			slog.Debug("preferred port is available", "service", serviceName, "port", preferredPort)
			return pm.saveAssignment(serviceName, preferredPort, false)
		}

		// Preferred port unavailable - handle conflict
		slog.Debug("preferred port is in use", "service", serviceName, "port", preferredPort)
		processInfo := getProcessInfoString(pm, preferredPort)
		return pm.handleConflictAndAssign(serviceName, preferredPort, processInfo, false)
	}

	// Find an available port automatically
	return pm.autoAssignPort(serviceName)
}

// handleConflictAndAssign handles a port conflict by prompting the user and taking action.
// Must be called with pm.mu held. Temporarily releases the lock for user input.
func (pm *PortManager) handleConflictAndAssign(serviceName string, port int, processInfo string, isExplicit bool) (int, bool, error) {
	// Release mutex before blocking on user input to prevent deadlocks
	// WARNING: TOCTOU race - state may change during user input. We re-validate after.
	pm.mu.Unlock()
	action, err := handlePortConflict(pm, port, serviceName, processInfo, isExplicit)
	pm.mu.Lock()

	if err != nil {
		return 0, false, err
	}

	switch action {
	case ActionKill:
		return pm.killAndAssign(serviceName, port)

	case ActionReassign:
		return pm.reassignPort(serviceName, port, isExplicit)

	case ActionAlwaysKill:
		if err := pm.setAlwaysKillPreference(true); err != nil {
			slog.Warn("failed to save always-kill preference", "error", err)
		}
		printPreferenceSavedMessage()
		return pm.killAndAssign(serviceName, port)

	default: // ActionCancel
		return 0, false, fmt.Errorf("operation cancelled by user")
	}
}

// killAndAssign kills the process on the port and assigns it to the service.
// Must be called with pm.mu held.
func (pm *PortManager) killAndAssign(serviceName string, port int) (int, bool, error) {
	// Re-validate port state after re-acquiring lock (state may have changed during user input)
	if pm.isPortAvailable(port) {
		// Port became available while waiting for user input - use it directly
		result, _, err := pm.saveAssignment(serviceName, port, false)
		if err != nil {
			return 0, false, err
		}
		printPortAvailableMessage(serviceName, port)
		return result, false, nil
	}

	// Kill process
	if err := pm.killProcessOnPort(port); err != nil {
		fmt.Fprintf(os.Stderr, "\n⚠️  %v\n", err)
		printKillFailedTip()
		return 0, false, fmt.Errorf("failed to free port %d: %w", port, err)
	}

	// Verify port is now available with retries
	if !pm.verifyPortCleanup(port) {
		printPortStillInUseMessage(port)
		printKillFailedTip()
		return 0, false, fmt.Errorf("port %d is still in use after cleanup attempt", port)
	}

	result, _, err := pm.saveAssignment(serviceName, port, false)
	if err != nil {
		return 0, false, err
	}
	printPortFreedMessage(serviceName, port)
	return result, false, nil
}

// reassignPort finds an alternative port and assigns it to the service.
// Must be called with pm.mu held. May temporarily release the lock for user input.
func (pm *PortManager) reassignPort(serviceName string, originalPort int, isExplicit bool) (int, bool, error) {
	printFindingPortMessage(serviceName)

	port, err := pm.findAvailablePort()
	if err != nil {
		return 0, false, err
	}

	result, _, err := pm.saveAssignment(serviceName, port, false)
	if err != nil {
		return 0, false, err
	}

	printPortAssignedMessage(serviceName, port)

	// For explicit ports, offer to update azure.yaml
	if isExplicit {
		// Release mutex before blocking on user input
		pm.mu.Unlock()
		wantsUpdate := promptUpdateAzureYaml(port)
		pm.mu.Lock()

		if wantsUpdate {
			return result, true, nil // Signal caller to update azure.yaml
		}
	}

	return result, false, nil
}

// autoAssignPort finds and assigns an available port automatically.
// Must be called with pm.mu held.
func (pm *PortManager) autoAssignPort(serviceName string) (int, bool, error) {
	port, err := pm.findAvailablePort()
	if err != nil {
		return 0, false, err
	}

	result, _, err := pm.saveAssignment(serviceName, port, false)
	if err != nil {
		return 0, false, err
	}

	printAutoAssignedMessage(serviceName, port)
	return result, false, nil
}

// saveAssignment saves a port assignment and returns the port.
// Must be called with pm.mu held.
func (pm *PortManager) saveAssignment(serviceName string, port int, wasAutoAssigned bool) (int, bool, error) {
	pm.assignments[serviceName] = &PortAssignment{
		ServiceName: serviceName,
		Port:        port,
		LastUsed:    time.Now(),
	}
	if err := pm.save(); err != nil {
		return 0, false, fmt.Errorf("failed to save port assignment: %w", err)
	}
	return port, wasAutoAssigned, nil
}

// ReleasePort removes a port assignment.
func (pm *PortManager) ReleasePort(serviceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.assignments, serviceName)
	return pm.clearServicePort(serviceName)
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

// CleanStalePorts removes port assignments older than the stale threshold.
// Assignments are considered stale if they haven't been used in 7 days.
func (pm *PortManager) CleanStalePorts() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	threshold := time.Now().Add(-staleThreshold)

	// Check each assignment for staleness
	for name, assignment := range pm.assignments {
		if assignment.LastUsed.Before(threshold) {
			if err := pm.clearServicePort(name); err != nil {
				slog.Warn("failed to clear port for service", "service", name, "error", err)
			}
			delete(pm.assignments, name)
			slog.Debug("removed stale port assignment", "service", name, "port", assignment.Port, "lastUsed", assignment.LastUsed)
		}
	}

	return nil
}

// getConfigClient returns the azdconfig client, creating it lazily if needed.
// If no gRPC connection is available (e.g., during tests), falls back to a shared
// in-memory storage that persists across port manager instances.
func (pm *PortManager) getConfigClient() (azdconfig.ConfigClient, error) {
	if pm.configClient != nil {
		return pm.configClient, nil
	}

	client, err := azdconfig.NewClient(context.Background())
	if err != nil {
		// Fall back to shared in-memory client when gRPC is not available.
		// Using a shared client ensures port assignments are visible across
		// all port manager instances within the same process (important for tests).
		slog.Debug("gRPC connection not available, using shared in-memory port storage", "error", err)
		sharedInMemoryClientOnce.Do(func() {
			sharedInMemoryClient = azdconfig.NewInMemoryClient()
		})
		pm.configClient = sharedInMemoryClient
		return pm.configClient, nil
	}
	pm.configClient = client
	return client, nil
}

// SetConfigClient sets a custom config client for testing purposes.
func (pm *PortManager) SetConfigClient(client azdconfig.ConfigClient) {
	pm.configClient = client
}

// getAlwaysKillPreference returns true if the user has set the preference to always kill
// port conflicts without prompting.
func (pm *PortManager) getAlwaysKillPreference() bool {
	client, err := pm.getConfigClient()
	if err != nil {
		slog.Debug("failed to get config client for preference check", "error", err)
		return false
	}

	value, err := client.GetPreference(preferenceAlwaysKillPortConflicts)
	if err != nil {
		slog.Debug("failed to get always-kill preference", "error", err)
		return false
	}

	return value == "true"
}

// setAlwaysKillPreference sets the preference to always kill port conflicts without prompting.
func (pm *PortManager) setAlwaysKillPreference(alwaysKill bool) error {
	client, err := pm.getConfigClient()
	if err != nil {
		return fmt.Errorf("failed to get config client: %w", err)
	}

	value := "false"
	if alwaysKill {
		value = "true"
	}

	if err := client.SetPreference(preferenceAlwaysKillPortConflicts, value); err != nil {
		return fmt.Errorf("failed to set always-kill preference: %w", err)
	}

	slog.Info("updated always-kill port conflicts preference", "value", alwaysKill)
	return nil
}

// load reads port assignments from azd's UserConfig service.
func (pm *PortManager) load() error {
	client, err := pm.getConfigClient()
	if err != nil {
		// If we can't connect to azd, start with empty assignments
		slog.Debug("could not connect to azdconfig, starting with empty assignments", "error", err)
		return nil
	}

	ports, err := client.GetAllServicePorts(pm.projectHash)
	if err != nil {
		// gRPC operation failed - switch to shared in-memory client.
		// This can happen when gRPC connection was established but the server
		// is not running or not responding properly.
		slog.Debug("gRPC operation failed, switching to shared in-memory port storage", "error", err)
		sharedInMemoryClientOnce.Do(func() {
			sharedInMemoryClient = azdconfig.NewInMemoryClient()
		})
		pm.configClient = sharedInMemoryClient
		// Try loading again from the in-memory client (will be empty on first access)
		ports, _ = pm.configClient.GetAllServicePorts(pm.projectHash)
	}

	// Convert map[string]int to map[string]*PortAssignment
	for serviceName, port := range ports {
		pm.assignments[serviceName] = &PortAssignment{
			ServiceName: serviceName,
			Port:        port,
			LastUsed:    time.Now(), // We don't persist LastUsed anymore
		}
	}

	slog.Debug("loaded port assignments from config", "count", len(pm.assignments))
	return nil
}

// save writes port assignments to azd's UserConfig service.
func (pm *PortManager) save() error {
	client, err := pm.getConfigClient()
	if err != nil {
		return fmt.Errorf("failed to get config client: %w", err)
	}

	// Save each port assignment individually for efficient updates
	for serviceName, assignment := range pm.assignments {
		if err := client.SetServicePort(pm.projectHash, serviceName, assignment.Port); err != nil {
			return fmt.Errorf("failed to save port for service %s: %w", serviceName, err)
		}
	}

	slog.Debug("saved port assignments to config", "count", len(pm.assignments))
	return nil
}

// clearServicePort removes a service port from the config.
func (pm *PortManager) clearServicePort(serviceName string) error {
	client, err := pm.getConfigClient()
	if err != nil {
		return fmt.Errorf("failed to get config client: %w", err)
	}

	if err := client.ClearServicePort(pm.projectHash, serviceName); err != nil {
		return fmt.Errorf("failed to clear port for service %s: %w", serviceName, err)
	}

	slog.Debug("cleared service port from config", "service", serviceName)
	return nil
}
