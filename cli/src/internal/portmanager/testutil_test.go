//go:build integration

package portmanager

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TestServer represents a spawned test server process.
type TestServer struct {
	Port    int
	PID     int
	Cmd     *exec.Cmd
	cancel  context.CancelFunc
	stopped bool
	mu      sync.Mutex
}

// TestServerOption configures test server spawning.
type TestServerOption func(*testServerConfig)

type testServerConfig struct {
	timeout time.Duration
}

// WithTimeout sets the maximum time to wait for server to start.
func WithTimeout(d time.Duration) TestServerOption {
	return func(c *testServerConfig) {
		c.timeout = d
	}
}

// SpawnPythonHTTPServer spawns a Python HTTP server on the specified port.
// Returns a TestServer with process info, or an error if spawning fails.
// The server binds to localhost only for security.
func SpawnPythonHTTPServer(port int, opts ...TestServerOption) (*TestServer, error) {
	cfg := &testServerConfig{
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Verify Python is available
	pythonCmd := getPythonCommand()
	if pythonCmd == "" {
		return nil, fmt.Errorf("python not found in PATH")
	}

	// Create context with timeout for startup
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)

	// Build the command to start Python HTTP server
	// Use -u for unbuffered output and bind to localhost only
	cmd := exec.CommandContext(ctx, pythonCmd, "-u", "-m", "http.server", strconv.Itoa(port), "--bind", "127.0.0.1")

	// Set process group for clean termination (Unix only)
	setProcessGroup(cmd)

	// Don't inherit stdin
	cmd.Stdin = nil

	// Capture stderr for diagnostics
	cmd.Stderr = os.Stderr

	// Start the server
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start Python HTTP server: %w", err)
	}

	server := &TestServer{
		Port:   port,
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		cancel: cancel,
	}

	// Wait for server to be ready (port is listening)
	if err := waitForPort(ctx, port, cfg.timeout); err != nil {
		server.Stop()
		return nil, fmt.Errorf("server failed to start listening on port %d: %w", port, err)
	}

	return server, nil
}

// SpawnPythonHTTPServerOnRandomPort spawns a Python HTTP server on an available port.
// Returns the TestServer with the assigned port.
// Uses retry logic to handle TOCTOU race conditions where another process may
// grab the port between finding it available and binding to it.
func SpawnPythonHTTPServerOnRandomPort(opts ...TestServerOption) (*TestServer, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		port, err := findAvailablePort()
		if err != nil {
			lastErr = fmt.Errorf("failed to find available port: %w", err)
			continue
		}

		server, err := SpawnPythonHTTPServer(port, opts...)
		if err == nil {
			return server, nil
		}

		// Port was grabbed by another process (TOCTOU race), retry
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}

	return nil, fmt.Errorf("failed to spawn server after %d attempts: %w", maxRetries, lastErr)
}

// Stop gracefully stops the test server.
func (s *TestServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil
	}
	s.stopped = true

	if s.cancel != nil {
		s.cancel()
	}

	if s.Cmd != nil && s.Cmd.Process != nil {
		// Try graceful termination first
		if runtime.GOOS == osWindows {
			// On Windows, just kill the process directly
			// Error ignored: process may have already exited
			_ = s.Cmd.Process.Kill()
		} else {
			// On Unix, send SIGTERM first for graceful shutdown
			// Error ignored: process may have already exited
			_ = s.Cmd.Process.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)
			// Force kill if still running
			// Error ignored: process may have already exited
			_ = s.Cmd.Process.Kill()
		}
		// Wait for process to exit (with timeout)
		done := make(chan error, 1)
		go func() {
			// Error ignored: we just need to wait for exit
			done <- s.Cmd.Wait()
		}()
		select {
		case <-done:
			// Process exited
		case <-time.After(2 * time.Second):
			// Force kill if still running after timeout
			// Error ignored: best effort cleanup
			_ = s.Cmd.Process.Kill()
		}
	}

	return nil
}

// IsRunning checks if the server process is still running.
func (s *TestServer) IsRunning() bool {
	if s.Cmd == nil || s.Cmd.Process == nil {
		return false
	}

	// Check if process exists
	if runtime.GOOS == osWindows {
		// On Windows, os.FindProcess always succeeds, so we need PowerShell check
		// Use timeout to prevent hanging in edge cases
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "powershell", "-Command", fmt.Sprintf("Get-Process -Id %d -ErrorAction SilentlyContinue", s.PID))
		if err := cmd.Run(); err != nil {
			return false
		}
		return true
	}

	// On Unix, check if process exists by sending signal 0
	process, err := os.FindProcess(s.PID)
	if err != nil {
		return false
	}
	err = process.Signal(syscallSignal0())
	return err == nil
}

// getPythonCommand returns the Python command available on the system.
func getPythonCommand() string {
	// Try python3 first (preferred on Unix), then python
	commands := []string{"python3", "python"}
	if runtime.GOOS == osWindows {
		// On Windows, python is more common than python3
		commands = []string{"python", "python3", "py"}
	}

	for _, cmd := range commands {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path
		}
	}
	return ""
}

// IsPythonAvailable checks if Python is available for tests.
func IsPythonAvailable() bool {
	return getPythonCommand() != ""
}

// findAvailablePort finds an available port by attempting to bind.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Give the OS a moment to release the port
	time.Sleep(50 * time.Millisecond)

	return port, nil
}

// waitForPort waits for a port to become available (listening).
func waitForPort(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d to be listening", port)
}

// waitForPortFree waits for a port to become available (not listening).
// Handles TCP TIME_WAIT state with retry logic.
func waitForPortFree(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if canBindToPort(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d to be freed", port)
}

// canBindToPort attempts to bind to a port to verify it's available.
func canBindToPort(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// isPortInUse checks if a port is currently in use.
func isPortInUse(port int) bool {
	return !canBindToPort(port)
}

// setProcessGroup sets process group for clean termination (platform-specific).
// Implemented in platform-specific files.
func setProcessGroup(cmd *exec.Cmd) {
	// Default implementation does nothing
	// Platform-specific implementations in testutil_unix_test.go and testutil_windows_test.go
	setProcessGroupImpl(cmd)
}

// syscallSignal0 returns signal 0 for process existence check.
// On Windows, this is not used (we use a different method).
func syscallSignal0() os.Signal {
	return syscallSignal0Impl()
}

// DiagnosticInfo collects diagnostic information for test failures.
type DiagnosticInfo struct {
	Port            int
	ProcessDetected bool
	ProcessPID      int
	ProcessName     string
	PortInUse       bool
	CanBind         bool
	NetstatOutput   string
	Timestamp       time.Time
}

// CollectDiagnostics gathers diagnostic information about a port.
func CollectDiagnostics(pm *PortManager, port int) DiagnosticInfo {
	info := DiagnosticInfo{
		Port:      port,
		Timestamp: time.Now(),
	}

	// Check if we can detect a process on the port
	if pid, err := pm.getProcessOnPort(port); err == nil {
		info.ProcessDetected = true
		info.ProcessPID = pid
		if name, err := pm.getProcessName(pid); err == nil {
			info.ProcessName = name
		}
	}

	// Check port status
	info.PortInUse = isPortInUse(port)
	info.CanBind = canBindToPort(port)

	// Get netstat/ss output for additional diagnostics
	info.NetstatOutput = getNetstatOutput(port)

	return info
}

// getNetstatOutput returns netstat/ss output for a specific port.
func getNetstatOutput(port int) string {
	var cmd *exec.Cmd

	if runtime.GOOS == osWindows {
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("netstat -ano | Select-String ':%d '", port))
	} else {
		// Try ss first, then netstat
		cmd = exec.Command("sh", "-c", fmt.Sprintf("ss -tlnp 2>/dev/null | grep ':%d ' || netstat -tlnp 2>/dev/null | grep ':%d '", port, port))
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return strings.TrimSpace(string(output))
}

// String returns a formatted diagnostic string.
func (d DiagnosticInfo) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Port Diagnostics for port %d at %s:\n", d.Port, d.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("  Process Detected: %v\n", d.ProcessDetected))
	if d.ProcessDetected {
		sb.WriteString(fmt.Sprintf("  Process PID: %d\n", d.ProcessPID))
		sb.WriteString(fmt.Sprintf("  Process Name: %s\n", d.ProcessName))
	}
	sb.WriteString(fmt.Sprintf("  Port In Use: %v\n", d.PortInUse))
	sb.WriteString(fmt.Sprintf("  Can Bind: %v\n", d.CanBind))
	if d.NetstatOutput != "" {
		sb.WriteString(fmt.Sprintf("  Netstat Output:\n    %s\n", strings.ReplaceAll(d.NetstatOutput, "\n", "\n    ")))
	}
	return sb.String()
}
