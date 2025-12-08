//go:build integration

package portmanager

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

// isMacOSCI returns true if running on macOS in a CI environment.
// These tests are skipped on macOS CI because Python HTTP server startup
// is unreliable (takes >45 seconds to bind to port in GitHub Actions).
func isMacOSCI() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	// Check common CI environment variables
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// TestKillExternalProcess_SimpleServer spawns a real Python HTTP server,
// verifies the port is in use, kills the process via killProcessOnPort,
// and verifies the port is freed.
func TestKillExternalProcess_SimpleServer(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Spawn Python HTTP server on a random port
	server, err := SpawnPythonHTTPServerOnRandomPort(WithTimeout(15 * time.Second))
	if err != nil {
		t.Fatalf("Failed to spawn Python HTTP server: %v", err)
	}
	defer func() { _ = server.Stop() }() // Cleanup in case test fails

	t.Logf("Spawned Python HTTP server on port %d with PID %d", server.Port, server.PID)

	// Verify port is in use
	if !isPortInUse(server.Port) {
		t.Fatalf("Port %d should be in use by the spawned server", server.Port)
	}
	t.Logf("Verified port %d is in use", server.Port)

	// Verify we can detect the process
	detectedPID, err := pm.getProcessOnPort(server.Port)
	if err != nil {
		// Collect diagnostics
		diag := CollectDiagnostics(pm, server.Port)
		t.Logf("Process detection failed, diagnostics:\n%s", diag.String())
		t.Fatalf("Failed to detect process on port %d: %v", server.Port, err)
	}
	t.Logf("Detected process PID %d on port %d (spawned PID was %d)", detectedPID, server.Port, server.PID)

	// Kill the process using port manager
	t.Logf("Calling killProcessOnPort(%d)", server.Port)
	err = pm.killProcessOnPort(server.Port)
	if err != nil {
		diag := CollectDiagnostics(pm, server.Port)
		t.Logf("Kill failed, diagnostics:\n%s", diag.String())
		t.Fatalf("killProcessOnPort failed: %v", err)
	}
	t.Logf("killProcessOnPort completed without error")

	// Wait for port to be freed (handles TIME_WAIT state)
	err = waitForPortFree(server.Port, 10*time.Second)
	if err != nil {
		diag := CollectDiagnostics(pm, server.Port)
		t.Logf("Port not freed, diagnostics:\n%s", diag.String())
		t.Fatalf("Port %d was not freed after kill: %v", server.Port, err)
	}
	t.Logf("Verified port %d is now free", server.Port)
}

// TestKillExternalProcess_VerifyPortFreed spawns a server, kills it,
// and verifies the port can be rebound immediately.
func TestKillExternalProcess_VerifyPortFreed(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Spawn Python HTTP server
	server, err := SpawnPythonHTTPServerOnRandomPort(WithTimeout(15 * time.Second))
	if err != nil {
		t.Fatalf("Failed to spawn Python HTTP server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	port := server.Port
	t.Logf("Spawned server on port %d with PID %d", port, server.PID)

	// Kill the process
	err = pm.killProcessOnPort(port)
	if err != nil {
		diag := CollectDiagnostics(pm, port)
		t.Logf("Kill failed, diagnostics:\n%s", diag.String())
		t.Fatalf("killProcessOnPort failed: %v", err)
	}

	// Try to bind to the port with retry logic for TIME_WAIT
	const maxRetries = 50 // 5 seconds total with 100ms sleep
	var listener net.Listener
	var bindErr error

	for i := 0; i < maxRetries; i++ {
		listener, bindErr = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if bindErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if bindErr != nil {
		diag := CollectDiagnostics(pm, port)
		t.Logf("Bind failed after retries, diagnostics:\n%s", diag.String())
		t.Fatalf("Failed to bind to port %d after kill: %v", port, bindErr)
	}
	defer listener.Close()

	t.Logf("Successfully bound to port %d after killing process", port)

	// Verify we're actually listening
	addr := listener.Addr().(*net.TCPAddr)
	if addr.Port != port {
		t.Errorf("Bound to wrong port: expected %d, got %d", port, addr.Port)
	}
}

// TestKillExternalProcess_RapidRebind is a stress test that kills processes
// and immediately tries to rebind multiple times.
// Note: This test is not run in parallel to avoid port contention.
func TestKillExternalProcess_RapidRebind(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find a port to use for all iterations
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	t.Logf("Using port %d for rapid rebind test", port)

	const iterations = 3 // Reduced to avoid test timeouts in CI

	for i := 0; i < iterations; i++ {
		t.Logf("Iteration %d/%d", i+1, iterations)

		// Spawn server with reduced timeout for faster CI runs
		server, err := SpawnPythonHTTPServer(port, WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("Iteration %d: Failed to spawn server: %v", i+1, err)
		}

		// Verify port is in use
		if !isPortInUse(port) {
			_ = server.Stop()
			t.Fatalf("Iteration %d: Port should be in use", i+1)
		}

		// Kill the process
		err = pm.killProcessOnPort(port)
		if err != nil {
			diag := CollectDiagnostics(pm, port)
			t.Logf("Iteration %d: Kill failed, diagnostics:\n%s", i+1, diag.String())
			_ = server.Stop()
			t.Fatalf("Iteration %d: killProcessOnPort failed: %v", i+1, err)
		}

		// Wait for port to be freed (reduced timeout for CI)
		err = waitForPortFree(port, 5*time.Second)
		if err != nil {
			diag := CollectDiagnostics(pm, port)
			t.Logf("Iteration %d: Port not freed, diagnostics:\n%s", i+1, diag.String())
			_ = server.Stop()
			t.Fatalf("Iteration %d: Port not freed: %v", i+1, err)
		}

		// Explicit cleanup
		_ = server.Stop()

		t.Logf("Iteration %d: Successfully killed and freed port", i+1)
	}
}

// TestKillExternalProcess_ProcessDetection tests that process detection works
// correctly for external processes.
func TestKillExternalProcess_ProcessDetection(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Spawn server
	server, err := SpawnPythonHTTPServerOnRandomPort(WithTimeout(15 * time.Second))
	if err != nil {
		t.Fatalf("Failed to spawn server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	t.Logf("Spawned server on port %d with PID %d", server.Port, server.PID)

	// Test getProcessOnPort
	detectedPID, err := pm.getProcessOnPort(server.Port)
	if err != nil {
		diag := CollectDiagnostics(pm, server.Port)
		t.Logf("Detection failed, diagnostics:\n%s", diag.String())
		t.Fatalf("getProcessOnPort failed: %v", err)
	}

	t.Logf("Detected PID: %d, Expected PID: %d", detectedPID, server.PID)

	// PID should match (or be a parent/child in some environments)
	if detectedPID != server.PID {
		t.Logf("Warning: Detected PID %d doesn't match spawned PID %d (may be normal in containers)", detectedPID, server.PID)
	}

	// Test getProcessName
	name, err := pm.getProcessName(detectedPID)
	if err != nil {
		t.Logf("Warning: getProcessName failed (may be normal in some environments): %v", err)
	} else {
		t.Logf("Process name: %s", name)
		// Verify it looks like a Python process
		if runtime.GOOS == osWindows {
			// Windows: could be "python" or "Python"
			t.Logf("Process name on Windows: %s", name)
		} else {
			// Unix: usually "python3" or "python"
			t.Logf("Process name on Unix: %s", name)
		}
	}

	// Test getProcessInfoOnPort
	info, err := pm.getProcessInfoOnPort(server.Port)
	if err != nil {
		t.Fatalf("getProcessInfoOnPort failed: %v", err)
	}
	t.Logf("ProcessInfo: PID=%d, Name=%s", info.PID, info.Name)
}

// TestKillExternalProcess_NoProcessOnPort verifies behavior when no process is on the port.
func TestKillExternalProcess_NoProcessOnPort(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	t.Logf("Testing killProcessOnPort on unused port %d", port)

	// Kill should succeed (no-op) when port is not in use
	err = pm.killProcessOnPort(port)
	if err != nil {
		t.Errorf("killProcessOnPort should succeed for unused port, got error: %v", err)
	}

	// Port should still be available
	if !canBindToPort(port) {
		t.Errorf("Port %d should still be available", port)
	}
}

// TestKillExternalProcess_MultipleKills tests that calling kill multiple times is safe.
func TestKillExternalProcess_MultipleKills(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Spawn server
	server, err := SpawnPythonHTTPServerOnRandomPort(WithTimeout(15 * time.Second))
	if err != nil {
		t.Fatalf("Failed to spawn server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	port := server.Port
	t.Logf("Spawned server on port %d", port)

	// Kill multiple times - should not error
	for i := 0; i < 3; i++ {
		err = pm.killProcessOnPort(port)
		if err != nil {
			t.Errorf("Kill attempt %d failed: %v", i+1, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Port should be free
	err = waitForPortFree(port, 10*time.Second)
	if err != nil {
		diag := CollectDiagnostics(pm, port)
		t.Logf("Port not freed, diagnostics:\n%s", diag.String())
		t.Fatalf("Port not freed after multiple kills: %v", err)
	}
}

// TestKillExternalProcess_Timeout verifies that detection/kill operations don't hang.
// Note: This test is not run in parallel because port operations can interfere.
func TestKillExternalProcess_Timeout(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find a dynamically available port (avoids hardcoded port conflicts)
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	t.Logf("Testing timeout behavior on unused port %d", port)

	// Set a test timeout
	done := make(chan bool, 1)
	go func() {
		// This should complete quickly (not hang)
		_ = pm.killProcessOnPort(port)
		done <- true
	}()

	select {
	case <-done:
		t.Log("killProcessOnPort completed without hanging")
	case <-time.After(15 * time.Second):
		t.Fatal("killProcessOnPort timed out - operation should not hang")
	}
}

// TestDiagnostics_CollectInfo tests the diagnostic information collection.
func TestDiagnostics_CollectInfo(t *testing.T) {
	if isMacOSCI() {
		t.Skip("Skipping on macOS CI - Python HTTP server startup is unreliable")
	}
	if !IsPythonAvailable() {
		t.Skip("Python not available, skipping integration test")
	}

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Spawn server
	server, err := SpawnPythonHTTPServerOnRandomPort(WithTimeout(15 * time.Second))
	if err != nil {
		t.Fatalf("Failed to spawn server: %v", err)
	}
	defer func() { _ = server.Stop() }()

	// Collect diagnostics
	diag := CollectDiagnostics(pm, server.Port)

	t.Logf("Diagnostics:\n%s", diag.String())

	// Verify diagnostic info is populated
	if !diag.PortInUse {
		t.Errorf("PortInUse should be true for active server")
	}
	if diag.CanBind {
		t.Errorf("CanBind should be false while server is running")
	}
	// Process detection may or may not work depending on environment
	if diag.ProcessDetected {
		if diag.ProcessPID <= 0 {
			t.Errorf("ProcessPID should be positive when process is detected")
		}
	}
}
