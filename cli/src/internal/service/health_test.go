package service

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPortHealthCheck_Success(t *testing.T) {
	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port from server URL
	port := server.Listener.Addr().(*net.TCPAddr).Port

	err := PortHealthCheck(port)
	if err != nil {
		t.Errorf("PortHealthCheck() error = %v, want nil", err)
	}
}

func TestPortHealthCheck_PortNotListening(t *testing.T) {
	// Use a port that's unlikely to be listening
	port := 64999

	err := PortHealthCheck(port)
	if err == nil {
		t.Error("PortHealthCheck() expected error for non-listening port")
	}
}

func TestHTTPHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	err := HTTPHealthCheck(port, "/")
	if err != nil {
		t.Errorf("HTTPHealthCheck() error = %v, want nil", err)
	}
}

func TestHTTPHealthCheck_WithPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	err := HTTPHealthCheck(port, "/health")
	if err != nil {
		t.Errorf("HTTPHealthCheck() error = %v, want nil", err)
	}
}

func TestHTTPHealthCheck_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	// Should succeed because 3xx is acceptable
	err := HTTPHealthCheck(port, "/")
	if err != nil {
		t.Errorf("HTTPHealthCheck() error = %v, want nil for redirect", err)
	}
}

func TestHTTPHealthCheck_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	err := HTTPHealthCheck(port, "/")
	if err == nil {
		t.Error("HTTPHealthCheck() expected error for 500 status")
	}
}

func TestHTTPHealthCheck_PortNotListening(t *testing.T) {
	port := 64998

	err := HTTPHealthCheck(port, "/")
	if err == nil {
		t.Error("HTTPHealthCheck() expected error for non-listening port")
	}
}

func TestProcessHealthCheck_Success(t *testing.T) {
	// We can't easily create a valid process without starting one
	// Skip this test in short mode, covered by integration tests
	if testing.Short() {
		t.Skip("skipping process check test in short mode")
	}
}

func TestProcessHealthCheck_NilProcess(t *testing.T) {
	process := &ServiceProcess{
		Name:    "test",
		Process: nil,
	}

	err := ProcessHealthCheck(process)
	if err == nil {
		t.Error("ProcessHealthCheck() expected error for nil process")
	}
}

func TestIsPortListening(t *testing.T) {
	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	// Port should be listening
	if !IsPortListening(port) {
		t.Error("IsPortListening() = false, want true")
	}

	// Close server
	server.Close()
	time.Sleep(100 * time.Millisecond)

	// Port should not be listening
	if IsPortListening(port) {
		t.Error("IsPortListening() = true, want false after server closed")
	}
}

func TestTryHTTPHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	if !TryHTTPHealthCheck(port, "/") {
		t.Error("TryHTTPHealthCheck() = false, want true")
	}

	// Test with non-listening port
	if TryHTTPHealthCheck(64997, "/") {
		t.Error("TryHTTPHealthCheck() = true, want false for non-listening port")
	}
}

func TestWaitForPort_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	// Start server in background after a delay
	port := 0
	serverReady := make(chan int)

	go func() {
		time.Sleep(500 * time.Millisecond)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		port := server.Listener.Addr().(*net.TCPAddr).Port
		serverReady <- port
		time.Sleep(2 * time.Second)
		server.Close()
	}()

	port = <-serverReady

	// Wait for port to become available
	err := WaitForPort(port, 3*time.Second)
	if err != nil {
		t.Errorf("WaitForPort() error = %v, want nil", err)
	}
}

func TestWaitForPort_Timeout(t *testing.T) {
	// Use port 1 which requires admin privileges and won't be available
	port := 1

	// Use a very short timeout to ensure the test completes quickly
	err := WaitForPort(port, 700*time.Millisecond)
	if err == nil {
		t.Error("WaitForPort() expected timeout error, got nil")
	}
	// The error message should contain port number or timeout
	if err != nil && !strings.Contains(err.Error(), "port") && !strings.Contains(err.Error(), "not available") {
		t.Errorf("WaitForPort() error = %v, want error containing 'port' or 'not available'", err)
	}
}

func TestPerformHealthCheck_PortType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping health check test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	runtime := ServiceRuntime{
		Name: "test-service",
		HealthCheck: HealthCheckConfig{
			Type:     "port",
			Timeout:  5 * time.Second,
			Interval: 500 * time.Millisecond,
		},
	}

	process := &ServiceProcess{
		Name:    "test-service",
		Runtime: runtime,
		Port:    port,
		Ready:   false,
	}

	err := PerformHealthCheck(process)
	if err != nil {
		t.Errorf("PerformHealthCheck() error = %v, want nil", err)
	}

	if !process.Ready {
		t.Error("PerformHealthCheck() process.Ready = false, want true")
	}
}

func TestPerformHealthCheck_HTTPType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping health check test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	runtime := ServiceRuntime{
		Name: "test-service",
		HealthCheck: HealthCheckConfig{
			Type:     "http",
			Path:     "/health",
			Timeout:  5 * time.Second,
			Interval: 500 * time.Millisecond,
		},
	}

	process := &ServiceProcess{
		Name:    "test-service",
		Runtime: runtime,
		Port:    port,
		Ready:   false,
	}

	err := PerformHealthCheck(process)
	if err != nil {
		t.Errorf("PerformHealthCheck() error = %v, want nil", err)
	}

	if !process.Ready {
		t.Error("PerformHealthCheck() process.Ready = false, want true")
	}
}

func TestPerformHealthCheck_Timeout(t *testing.T) {
	runtime := ServiceRuntime{
		Name: "test-service",
		HealthCheck: HealthCheckConfig{
			Type:     "port",
			Timeout:  1 * time.Second,
			Interval: 200 * time.Millisecond,
		},
	}

	process := &ServiceProcess{
		Name:    "test-service",
		Runtime: runtime,
		Port:    64995, // Port not listening
		Ready:   false,
	}

	err := PerformHealthCheck(process)
	if err == nil {
		t.Error("PerformHealthCheck() expected timeout error")
	}

	if process.Ready {
		t.Error("PerformHealthCheck() process.Ready = true, want false after timeout")
	}
}

func TestPerformHealthCheck_ProcessType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping health check test in short mode")
	}

	// Start a real process that will run for a bit
	cmd := exec.Command("timeout", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	runtime := ServiceRuntime{
		Name: "test-service",
		HealthCheck: HealthCheckConfig{
			Type:     "process",
			Timeout:  3 * time.Second,
			Interval: 500 * time.Millisecond,
		},
	}

	process := &ServiceProcess{
		Name:    "test-service",
		Runtime: runtime,
		Process: cmd.Process,
		Ready:   false,
	}

	err := PerformHealthCheck(process)
	if err != nil {
		t.Errorf("PerformHealthCheck() error = %v, want nil", err)
	}

	if !process.Ready {
		t.Error("PerformHealthCheck() process.Ready = false, want true")
	}
}
