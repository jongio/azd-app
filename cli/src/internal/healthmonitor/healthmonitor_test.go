package healthmonitor

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

func TestGetMonitor(t *testing.T) {
	tempDir := t.TempDir()

	monitor := GetMonitor(tempDir)
	if monitor == nil {
		t.Fatal("GetMonitor() returned nil")
	}

	// Getting again should return same instance
	monitor2 := GetMonitor(tempDir)
	if monitor != monitor2 {
		t.Error("GetMonitor() returned different instance for same directory")
	}
}

func TestStartStop(t *testing.T) {
	tempDir := t.TempDir()
	monitor := GetMonitor(tempDir)

	// Start should succeed
	if err := monitor.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Starting again should fail
	if err := monitor.Start(); err == nil {
		t.Error("Start() should fail when already running")
	}

	// Stop should succeed
	monitor.Stop()

	// Stopping again should be safe (no error)
	monitor.Stop()
}

func TestIsPortListening(t *testing.T) {
	// Start a test server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Port should be listening
	if !isPortListening(port) {
		t.Error("isPortListening() returned false for listening port")
	}

	// Close listener
	listener.Close()
	time.Sleep(100 * time.Millisecond) // Give time for port to be released

	// Port should not be listening
	if isPortListening(port) {
		t.Error("isPortListening() returned true for closed port")
	}
}

func TestCheckHTTPHealth(t *testing.T) {
	// Start a test HTTP server with health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	server := &http.Server{
		Addr:    "localhost:0",
		Handler: mux,
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check HTTP health
	healthy, checked := checkHTTPHealth(port)
	if !checked {
		t.Error("checkHTTPHealth() didn't find health endpoint")
	}
	if !healthy {
		t.Error("checkHTTPHealth() returned unhealthy for healthy endpoint")
	}
}

func TestCheckHTTPHealthUnhealthy(t *testing.T) {
	// Start a test HTTP server that returns 500
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Error")
	})

	server := &http.Server{
		Addr:    "localhost:0",
		Handler: mux,
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check HTTP health
	healthy, checked := checkHTTPHealth(port)
	if !checked {
		t.Error("checkHTTPHealth() didn't find health endpoint")
	}
	if healthy {
		t.Error("checkHTTPHealth() returned healthy for unhealthy endpoint")
	}
}

func TestCheckHTTPHealthNoEndpoint(t *testing.T) {
	// Start a test HTTP server without health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only respond to root path, not health endpoints
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "Not found")
		}
	})

	server := &http.Server{
		Addr:    "localhost:0",
		Handler: mux,
	}

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Check HTTP health
	healthy, checked := checkHTTPHealth(port)
	if checked {
		t.Error("checkHTTPHealth() found health endpoint when none exists")
	}
	if healthy {
		t.Error("checkHTTPHealth() returned healthy when no endpoint found")
	}
}

func TestCheckServiceWithPort(t *testing.T) {
	tempDir := t.TempDir()
	monitor := GetMonitor(tempDir)
	reg := registry.GetRegistry(tempDir)

	// Start a test server
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Register a service
	entry := &registry.ServiceRegistryEntry{
		Name:   "test-service",
		Port:   port,
		Status: "starting",
		Health: "unknown",
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Check service (port is listening, so should be running/healthy)
	status, health := monitor.checkService(entry)
	if status != "running" {
		t.Errorf("checkService() status = %v, want running", status)
	}
	if health != "healthy" {
		t.Errorf("checkService() health = %v, want healthy", health)
	}
}

func TestCheckServicePortNotListening(t *testing.T) {
	tempDir := t.TempDir()
	monitor := GetMonitor(tempDir)
	reg := registry.GetRegistry(tempDir)

	// Register a service with a port that's not listening
	entry := &registry.ServiceRegistryEntry{
		Name:   "test-service",
		Port:   59999, // Unlikely to be in use
		Status: "starting",
		Health: "unknown",
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Check service (port not listening, so should be starting/unknown)
	status, health := monitor.checkService(entry)
	if status != "starting" {
		t.Errorf("checkService() status = %v, want starting", status)
	}
	if health != "unknown" {
		t.Errorf("checkService() health = %v, want unknown", health)
	}
}
