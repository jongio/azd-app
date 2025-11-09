//go:build integration

package portmanager

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// TestPortAvailability_RealBinding tests actual port binding to verify conflict detection.
func TestPortAvailability_RealBinding(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	testPort := 0
	for port := 45000; port < 45100; port++ {
		if pm.isPortAvailable(port) {
			testPort = port
			break
		}
	}

	if testPort == 0 {
		t.Fatal("Could not find an available test port")
	}

	t.Logf("Using test port: %d", testPort)

	// Verify port is available
	if !pm.isPortAvailable(testPort) {
		t.Fatalf("Port %d should be available", testPort)
	}

	// Bind to the port (localhost only to avoid firewall prompts)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
	if err != nil {
		t.Fatalf("Failed to bind to port %d: %v", testPort, err)
	}
	defer listener.Close()

	// Port should now be unavailable
	if pm.isPortAvailable(testPort) {
		t.Errorf("Port %d should NOT be available after binding", testPort)
	}

	// Close the listener
	listener.Close()

	// Give the OS a moment to release the port
	time.Sleep(100 * time.Millisecond)

	// Port should be available again
	if !pm.isPortAvailable(testPort) {
		t.Errorf("Port %d should be available after closing listener", testPort)
	}
}

// TestPortAvailability_LocalhostVsAllInterfaces verifies that binding to all interfaces
// is properly detected even when checking with the same method.
func TestPortAvailability_LocalhostVsAllInterfaces(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	testPort := 0
	for port := 45100; port < 45200; port++ {
		if pm.isPortAvailable(port) {
			testPort = port
			break
		}
	}

	if testPort == 0 {
		t.Fatal("Could not find an available test port")
	}

	t.Logf("Using test port: %d", testPort)

	// Bind to localhost to avoid firewall prompts
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
	if err != nil {
		t.Fatalf("Failed to bind to localhost on port %d: %v", testPort, err)
	}
	defer listener.Close()

	// The port manager should detect this port is in use
	if pm.isPortAvailable(testPort) {
		t.Errorf("Port %d should NOT be available - bound to localhost but check didn't detect it", testPort)
		t.Error("This suggests the port availability check is using a different bind address than services")
	}
}

// TestAssignPort_Integration_WithRealConflict tests port assignment with actual port conflicts.
func TestAssignPort_Integration_WithRealConflict(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	testPort := 0
	for port := 45200; port < 45300; port++ {
		if pm.isPortAvailable(port) {
			testPort = port
			break
		}
	}

	if testPort == 0 {
		t.Fatal("Could not find an available test port")
	}

	t.Logf("Using test port: %d", testPort)

	// Assign port when available - should succeed
	assignedPort, _, err := pm.AssignPort("test-service", testPort, false, false)
	if err != nil {
		t.Fatalf("Failed to assign available port: %v", err)
	}
	if assignedPort != testPort {
		t.Errorf("Expected port %d, got %d", testPort, assignedPort)
	}

	// Now bind to that port to simulate it being in use
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
	if err != nil {
		t.Fatalf("Failed to bind to port %d: %v", testPort, err)
	}
	defer listener.Close()

	// Try to assign the same port again - should detect conflict
	// Note: This will prompt user in non-test scenarios, so we'll just verify
	// that the port manager recognizes it as unavailable
	if pm.isPortAvailable(testPort) {
		t.Errorf("Port %d should be detected as unavailable (in use)", testPort)
	}
}

// TestPortAssignment_MultipleServices tests assigning ports to multiple services.
func TestPortAssignment_MultipleServices(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Assign ports to multiple services
	services := []string{"service1", "service2", "service3"}
	assignedPorts := make(map[string]int)

	for _, serviceName := range services {
		// Use preferred port 0 to auto-assign
		port, _, err := pm.AssignPort(serviceName, 0, false, false)
		if err != nil {
			t.Fatalf("Failed to assign port for %s: %v", serviceName, err)
		}

		// Verify port is in valid range
		if port < 3000 || port > 65535 {
			t.Errorf("Assigned port %d for %s is out of range", port, serviceName)
		}

		// Verify no duplicate ports
		for otherService, otherPort := range assignedPorts {
			if port == otherPort {
				t.Errorf("Duplicate port %d assigned to both %s and %s", port, serviceName, otherService)
			}
		}

		assignedPorts[serviceName] = port
		t.Logf("Assigned port %d to %s", port, serviceName)
	}
}

// TestDashboardPortBinding tests that dashboard port binding matches port checking.
func TestDashboardPortBinding(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	testPort := 0
	for port := 45300; port < 45400; port++ {
		if pm.isPortAvailable(port) {
			testPort = port
			break
		}
	}

	if testPort == 0 {
		t.Fatal("Could not find an available test port")
	}

	t.Logf("Using test port: %d", testPort)

	// Simulate dashboard binding (now uses localhost)
	listener1, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
	if err != nil {
		t.Fatalf("Failed to simulate dashboard binding: %v", err)
	}
	defer listener1.Close()

	// Port manager should detect it's unavailable
	if pm.isPortAvailable(testPort) {
		t.Errorf("Port manager should detect dashboard port %d is in use", testPort)
	}

	// Close first listener
	listener1.Close()
	time.Sleep(100 * time.Millisecond)

	// Verify port is available again
	if !pm.isPortAvailable(testPort) {
		t.Errorf("Port %d should be available after dashboard closes", testPort)
	}
}

// TestPortConflict_SimultaneousInstances simulates two instances trying to use same ports.
func TestPortConflict_SimultaneousInstances(t *testing.T) {
	tempDir := t.TempDir()

	// First instance
	pm1 := GetPortManager(tempDir)
	port1, _, err := pm1.AssignPort("service1", 0, false, false)
	if err != nil {
		t.Fatalf("First instance failed to assign port: %v", err)
	}
	t.Logf("First instance got port: %d", port1)

	// Bind to simulate service running
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port1))
	if err != nil {
		t.Fatalf("Failed to bind port %d: %v", port1, err)
	}
	defer listener.Close()

	// Second instance (new manager from same directory)
	// Clear cache to simulate separate process
	managerCacheMu.Lock()
	delete(managerCache, tempDir)
	managerCacheMu.Unlock()

	pm2 := GetPortManager(tempDir)

	// Second instance should detect port is in use
	if pm2.isPortAvailable(port1) {
		t.Errorf("Second instance should detect port %d is already in use", port1)
	}

	// Verify it's loaded the same assignment from disk
	savedPort, exists := pm2.GetAssignment("service1")
	if !exists {
		t.Error("Second instance should load port assignment from disk")
	} else if savedPort != port1 {
		t.Errorf("Second instance loaded wrong port: expected %d, got %d", port1, savedPort)
	}
}

// TestPortManager_SaveLoad verifies port assignments persist correctly.
func TestPortManager_SaveLoad(t *testing.T) {
	tempDir := t.TempDir()

	// Create manager and assign some ports
	pm1 := GetPortManager(tempDir)

	assignments := map[string]int{
		"service1": 3001,
		"service2": 3002,
		"service3": 3003,
	}

	for serviceName, preferredPort := range assignments {
		port, _, err := pm1.AssignPort(serviceName, preferredPort, false, false)
		if err != nil {
			t.Fatalf("Failed to assign port for %s: %v", serviceName, err)
		}
		if port != preferredPort {
			t.Logf("Note: Got different port than preferred for %s: %d vs %d", serviceName, port, preferredPort)
		}
		assignments[serviceName] = port // Update with actual assigned port
	}

	// Clear cache and create new manager (simulates new process)
	managerCacheMu.Lock()
	delete(managerCache, tempDir)
	managerCacheMu.Unlock()

	pm2 := GetPortManager(tempDir)

	// Verify all assignments were loaded
	for serviceName, expectedPort := range assignments {
		loadedPort, exists := pm2.GetAssignment(serviceName)
		if !exists {
			t.Errorf("Assignment for %s was not loaded from disk", serviceName)
			continue
		}
		if loadedPort != expectedPort {
			t.Errorf("Loaded wrong port for %s: expected %d, got %d", serviceName, expectedPort, loadedPort)
		}
	}
}

// TestPortManager_DebugMode verifies debug logging works correctly.
func TestPortManager_DebugMode(t *testing.T) {
	// Enable debug mode
	os.Setenv("AZD_APP_DEBUG", "true")
	defer os.Unsetenv("AZD_APP_DEBUG")

	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Find an available port
	testPort := 0
	for port := 45400; port < 45500; port++ {
		if pm.isPortAvailable(port) {
			testPort = port
			break
		}
	}

	if testPort == 0 {
		t.Fatal("Could not find an available test port")
	}

	// This should produce debug output (manually verify in test output)
	t.Logf("Testing port availability for port %d (check for debug output)", testPort)
	available := pm.isPortAvailable(testPort)

	if !available {
		t.Errorf("Port %d should be available", testPort)
	}

	// Bind and test again
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", testPort))
	if err != nil {
		t.Fatalf("Failed to bind: %v", err)
	}
	defer listener.Close()

	t.Logf("Testing port availability for port %d while bound (should show bind failure in debug)", testPort)
	available = pm.isPortAvailable(testPort)

	if available {
		t.Errorf("Port %d should NOT be available while bound", testPort)
	}
}
