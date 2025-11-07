package healthmonitor

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

// TestIntegration_HealthMonitorWithRegistry tests the integration between
// health monitor, registry, and observers.
func TestIntegration_HealthMonitorWithRegistry(t *testing.T) {
	tempDir := t.TempDir()

	// Get registry and health monitor
	reg := registry.GetRegistry(tempDir)
	monitor := GetMonitor(tempDir)

	// Create a test observer to track notifications
	notificationChan := make(chan *registry.ServiceRegistryEntry, 10)
	observer := &testObserver{notifications: notificationChan}

	// Subscribe observer to registry
	reg.Subscribe(observer)

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

	// Register a service in "starting" state
	entry := &registry.ServiceRegistryEntry{
		Name:   "test-service",
		Port:   port,
		Status: "starting",
		Health: "unknown",
	}
	if err := reg.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Clear any notifications from registration
	select {
	case <-notificationChan:
	default:
	}

	// Set shorter interval for testing
	monitor.interval = 500 * time.Millisecond

	// Start health monitor
	if err := monitor.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer monitor.Stop()

	// Wait for health monitor to detect the healthy service
	// It should update status from "starting" to "running" and health from "unknown" to "healthy"
	timeout := time.After(3 * time.Second)
	var notification *registry.ServiceRegistryEntry
	notificationReceived := false

	for !notificationReceived {
		select {
		case notification = <-notificationChan:
			if notification.Status == "running" && notification.Health == "healthy" {
				notificationReceived = true
			}
		case <-timeout:
			t.Fatal("Timed out waiting for health monitor to update service status")
		}
	}

	// Verify the notification has correct data
	if notification.Name != "test-service" {
		t.Errorf("Notification service name = %v, want test-service", notification.Name)
	}
	if notification.Port != port {
		t.Errorf("Notification port = %v, want %v", notification.Port, port)
	}

	// Verify registry was updated
	updatedEntry, exists := reg.GetService("test-service")
	if !exists {
		t.Fatal("Service not found in registry after update")
	}
	if updatedEntry.Status != "running" {
		t.Errorf("Registry status = %v, want running", updatedEntry.Status)
	}
	if updatedEntry.Health != "healthy" {
		t.Errorf("Registry health = %v, want healthy", updatedEntry.Health)
	}

	// Now simulate service crash by closing the server
	server.Close()
	listener.Close()
	time.Sleep(100 * time.Millisecond) // Give port time to be released

	// Wait for health monitor to detect the crashed service
	timeout = time.After(3 * time.Second)
	notificationReceived = false

	for !notificationReceived {
		select {
		case notification = <-notificationChan:
			if notification.Status == "error" && notification.Health == "unhealthy" {
				notificationReceived = true
			}
		case <-timeout:
			t.Fatal("Timed out waiting for health monitor to detect service crash")
		}
	}

	// Verify the crash was detected
	if notification.Name != "test-service" {
		t.Errorf("Crash notification service name = %v, want test-service", notification.Name)
	}
}

// testObserver is a test implementation of RegistryObserver.
type testObserver struct {
	notifications chan *registry.ServiceRegistryEntry
}

func (o *testObserver) OnServiceChanged(entry *registry.ServiceRegistryEntry) {
	o.notifications <- entry
}
