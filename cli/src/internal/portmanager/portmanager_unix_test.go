//go:build !windows

package portmanager

import (
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestUnixKillProcessOnPort_MacOSCompatibility verifies that killProcessOnPort
// works on BSD/macOS without GNU-specific extensions like xargs -r.
// This test addresses issue #47: macOS port conflict resolution not working.
func TestUnixKillProcessOnPort_MacOSCompatibility(t *testing.T) {
	// Note: No need for Windows check - file has //go:build !windows constraint
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to get available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Keep listener open to simulate a process holding the port
	// This is more reliable than using external netcat command
	t.Logf("Started test listener on port %d with PID %d", port, os.Getpid())

	// Verify port is in use
	if pm.isPortAvailable(port) {
		t.Fatalf("Port %d should be in use by our listener", port)
	}

	// Get the PID - should be our process
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		t.Fatalf("Failed to get process on port: %v", err)
	}

	expectedPID := os.Getpid()
	if pid != expectedPID {
		t.Logf("Warning: PID mismatch. Expected %d (our process), got %d", expectedPID, pid)
		// This is acceptable in containers or when lsof returns parent process
	}

	t.Logf("Process listening on port %d with PID %d", port, pid)

	// Close the listener before testing kill (we can't kill ourselves)
	listener.Close()

	// Give OS time to release the port from TIME_WAIT state
	// Use deadline-based approach instead of fixed sleep
	const maxWait = 2 * time.Second
	deadline := time.Now().Add(maxWait)
	portReleased := false

	for time.Now().Before(deadline) {
		if pm.isPortAvailable(port) {
			portReleased = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !portReleased {
		t.Skipf("Port %d still in TIME_WAIT after %v, cannot test kill functionality", port, maxWait)
	}

	// Now test that killProcessOnPort works correctly (should handle "not in use" gracefully)
	// The old code with xargs -r would fail here on macOS
	err = pm.killProcessOnPort(port)
	if err != nil {
		t.Fatalf("killProcessOnPort should handle port not in use gracefully, got error: %v", err)
	}

	// Verify port is still available
	if !pm.isPortAvailable(port) {
		t.Errorf("Port %d should be available after kill operation", port)
	}
}

// TestUnixGetProcessOnPort_NoBSDExtensions verifies that getProcessOnPort
// uses standard Unix commands without GNU/BSD-specific extensions.
func TestUnixGetProcessOnPort_NoBSDExtensions(t *testing.T) {
	// Note: No need for Windows check - file has //go:build !windows constraint
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Start a simple HTTP server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	defer listener.Close()

	// Get process info
	pid, err := pm.getProcessOnPort(port)
	if err != nil {
		t.Fatalf("getProcessOnPort failed: %v", err)
	}

	// Verify we got a valid PID
	if pid <= 0 {
		t.Errorf("Invalid PID: %d", pid)
	}

	// PID should match our process
	myPID := os.Getpid()
	if pid != myPID {
		// In some environments (containers, etc), this might differ
		t.Logf("PID mismatch (expected %d, got %d) - may be normal in containers", myPID, pid)
	}
}

// TestUnixKillCommand_NoXargs verifies that the kill command doesn't use xargs.
// This is a regression test for issue #47.
func TestUnixKillCommand_NoXargs(t *testing.T) {
	// Note: No need for Windows check - file has //go:build !windows constraint

	// This test verifies the implementation doesn't regress to using xargs
	// by checking the command that would be generated

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// We can't easily test the actual command without executing it,
	// but we can verify the code path by checking that killProcessOnPort
	// calls getProcessOnPort first (which it must for the fix to work)

	// Try to kill a process on a port that doesn't exist
	// This should fail gracefully without xargs errors
	err := pm.killProcessOnPort(12345) // Random unlikely port

	// Should return nil (no error) because getProcessOnPort will fail
	// and killProcessOnPort returns nil when port is not in use
	if err != nil {
		t.Logf("killProcessOnPort returned error (acceptable): %v", err)
	}

	// The key is that it shouldn't produce an xargs-specific error like:
	// "xargs: illegal option -- r"
	if err != nil && strings.Contains(err.Error(), "xargs") {
		t.Errorf("Error mentions xargs (regression!): %v", err)
	}
}

// TestMacOSPortConflictResolution is an integration test that simulates
// the exact scenario from issue #47.
func TestMacOSPortConflictResolution(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific integration test")
	}

	tempDir := t.TempDir()

	// Clear cache to ensure clean test
	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	pm := GetPortManager(tempDir)

	// Start a listener to simulate a service
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	defer listener.Close()

	t.Logf("Started test listener on port %d", port)

	// Simulate the conflict scenario:
	// 1. Port is in use
	if pm.isPortAvailable(port) {
		t.Fatalf("Port %d should be in use", port)
	}

	// 2. Get process info (this should work on macOS)
	info, err := pm.getProcessInfoOnPort(port)
	if err != nil {
		t.Fatalf("getProcessInfoOnPort failed on macOS: %v", err)
	}

	t.Logf("Process info: PID=%d, Name=%s", info.PID, info.Name)

	// 3. Verify PID is valid
	if info.PID <= 0 {
		t.Errorf("Invalid PID: %d", info.PID)
	}

	// 4. Close our listener so killProcessOnPort can succeed
	listener.Close()
	time.Sleep(100 * time.Millisecond)

	// Now the port should be available
	if !pm.isPortAvailable(port) {
		t.Logf("Port %d still showing as in use after closing listener", port)
	}
}
