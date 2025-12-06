// Package portmanager provides port allocation, management, and process monitoring capabilities.
package portmanager

import (
	"net"
	"sync"
	"time"
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

	// staleThreshold defines how old an assignment must be to be considered stale.
	staleThreshold = 7 * 24 * time.Hour // 7 days
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

// ProcessInfo contains information about a process using a port.
type ProcessInfo struct {
	PID  int
	Name string
}
