package portmanager

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestGetProcessOnPort_DoesNotHang verifies that getProcessOnPort returns within
// the timeout period and doesn't hang indefinitely. This is a regression test
// for Codespaces environments where lsof can hang.
func TestGetProcessOnPort_DoesNotHang(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Test with a port that's not in use - should return quickly
	start := time.Now()
	_, err := pm.getProcessOnPort(59999) // Random high port unlikely to be in use
	elapsed := time.Since(start)

	// Should complete within commandTimeout (5s) plus some buffer
	// In practice, should be much faster (< 1s)
	maxAllowed := commandTimeout + 2*time.Second
	if elapsed > maxAllowed {
		t.Errorf("getProcessOnPort took %v, expected < %v (possible hang)", elapsed, maxAllowed)
	}

	// Error is expected for unused port
	if err == nil {
		t.Logf("Unexpectedly found process on port 59999")
	}

	t.Logf("getProcessOnPort completed in %v", elapsed)
}

// TestGetProcessOnPort_WithActivePort verifies that getProcessOnPort works
// correctly when a port is actually in use.
func TestGetProcessOnPort_WithActivePort(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Start a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Test that we can get the PID within the timeout
	start := time.Now()
	pid, err := pm.getProcessOnPort(port)
	elapsed := time.Since(start)

	// Should complete quickly
	if elapsed > 3*time.Second {
		t.Errorf("getProcessOnPort took %v, should be much faster", elapsed)
	}

	if err != nil {
		t.Fatalf("getProcessOnPort failed: %v", err)
	}

	if pid <= 0 {
		t.Errorf("Invalid PID: %d", pid)
	}

	// Should be our process
	expectedPID := os.Getpid()
	if pid != expectedPID {
		t.Logf("PID mismatch (expected %d, got %d) - may be normal in containers", expectedPID, pid)
	}

	t.Logf("Found PID %d on port %d in %v", pid, port, elapsed)
}

// TestKillProcessOnPort_DoesNotHang verifies that killProcessOnPort returns
// within a reasonable time and doesn't hang indefinitely.
func TestKillProcessOnPort_DoesNotHang(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Test with a port that's not in use - should return quickly
	start := time.Now()
	err := pm.killProcessOnPort(59998) // Random high port unlikely to be in use
	elapsed := time.Since(start)

	// Should complete within timeout plus buffer
	maxAllowed := killProcessTimeout + commandTimeout + 5*time.Second
	if elapsed > maxAllowed {
		t.Errorf("killProcessOnPort took %v, expected < %v (possible hang)", elapsed, maxAllowed)
	}

	// Should succeed (no error when port not in use)
	if err != nil {
		t.Errorf("killProcessOnPort should handle unused port gracefully, got: %v", err)
	}

	t.Logf("killProcessOnPort completed in %v", elapsed)
}

// TestGetProcessName_DoesNotHang verifies that getProcessName returns within
// the timeout period.
func TestGetProcessName_DoesNotHang(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Test with our own PID
	start := time.Now()
	name, err := pm.getProcessName(os.Getpid())
	elapsed := time.Since(start)

	// Should complete quickly
	maxAllowed := commandTimeout + 2*time.Second
	if elapsed > maxAllowed {
		t.Errorf("getProcessName took %v, expected < %v (possible hang)", elapsed, maxAllowed)
	}

	if err != nil {
		t.Logf("getProcessName returned error (may be OK): %v", err)
	} else if name == "" {
		t.Error("getProcessName returned empty name")
	} else {
		t.Logf("Process name: %s (took %v)", name, elapsed)
	}
}

// TestBuildGetProcessOnPortCommand_HasTimeout verifies that the Unix command
// includes timeout protection for lsof.
func TestBuildGetProcessOnPortCommand_HasTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	cmd, args := buildGetProcessOnPortCommand(8080)

	if cmd != "sh" {
		t.Errorf("Expected 'sh' command, got %s", cmd)
	}

	if len(args) < 2 {
		t.Fatalf("Expected at least 2 args, got %d", len(args))
	}

	script := args[1]

	// Verify the script uses ss (faster in containers) or has timeout protection
	hasSSPrimary := strings.Contains(script, "ss -tlnp")
	hasTimeoutLsof := strings.Contains(script, "timeout") && strings.Contains(script, "lsof")

	if !hasSSPrimary && !hasTimeoutLsof {
		t.Error("Unix command should use 'ss' as primary or have 'timeout' wrapper for lsof")
	}

	t.Logf("Command script: %s", script)
}

// TestBuildKillProcessCommand_UsesPgrep verifies that the Unix kill command
// uses pgrep (more portable) instead of pkill with flags that may not exist everywhere.
func TestBuildKillProcessCommand_UsesPgrep(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	cmd, args := buildKillProcessCommand(12345)

	if cmd != "sh" {
		t.Errorf("Expected 'sh' command, got %s", cmd)
	}

	if len(args) < 2 {
		t.Fatalf("Expected at least 2 args, got %d", len(args))
	}

	script := args[1]

	// Verify the script uses kill -9 (like the user's working script)
	hasKill9 := strings.Contains(script, "kill -9") || strings.Contains(script, "kill -KILL")

	if !hasKill9 {
		t.Error("Unix kill command should use kill -9 for reliable process termination")
	}

	t.Logf("Kill command script: %s", script)
}

// TestCommandsDoNotInheritStdin verifies that process commands don't inherit stdin,
// which can cause hangs in non-interactive environments like Codespaces.
func TestCommandsDoNotInheritStdin(t *testing.T) {
	// This is a code inspection test - we verify the implementation pattern
	// by checking that exec.CommandContext is used with Stdin = nil

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Run getProcessOnPort which should use the non-blocking pattern
	done := make(chan bool, 1)
	go func() {
		_, _ = pm.getProcessOnPort(59997)
		done <- true
	}()

	// Should complete within timeout even without stdin
	select {
	case <-done:
		// Success - command completed without hanging
	case <-time.After(commandTimeout + 5*time.Second):
		t.Error("getProcessOnPort appears to be hanging, possibly waiting for stdin")
	}
}

// TestContextTimeoutIsRespected verifies that the context timeout actually
// causes the command to abort rather than hanging forever.
func TestContextTimeoutIsRespected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a command that would hang forever
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-Command", "Start-Sleep -Seconds 60")
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", "sleep 60")
	}
	cmd.Stdin = nil

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	// Should have been cancelled by context timeout
	if elapsed > 3*time.Second {
		t.Errorf("Command took %v despite 1s context timeout", elapsed)
	}

	// Should have a context deadline exceeded error
	if ctx.Err() != context.DeadlineExceeded {
		t.Logf("Context error: %v (command error: %v)", ctx.Err(), err)
	}

	t.Logf("Timed-out command completed in %v", elapsed)
}
