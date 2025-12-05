package portmanager

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"time"
)

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
// If the service already has a persisted assignment with the preferred port,
// it will attempt to reuse that port first for consistency across runs.
//
// Returns:
//   - *PortReservation: Holds the port open. Call Release() before binding.
//   - error: Non-nil if no port can be reserved after max attempts
func (pm *PortManager) FindAndReservePort(serviceName string, preferredPort int) (*PortReservation, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Build map of assigned ports to avoid duplicates, excluding this service's own assignment
	// This allows a service to reuse its own persisted port for consistency across runs
	assignedPorts := make(map[int]bool)
	for name, assignment := range pm.assignments {
		if name != serviceName {
			assignedPorts[assignment.Port] = true
		}
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
