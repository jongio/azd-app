package portmanager

import (
	"sync"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
)

// mockPortChecker returns a port checker that simulates port availability without network binding.
// This avoids Windows Firewall prompts during testing.
func mockPortChecker(unavailablePorts map[int]bool) func(int) bool {
	mu := sync.Mutex{}
	return func(port int) bool {
		mu.Lock()
		defer mu.Unlock()
		return !unavailablePorts[port]
	}
}

// setupTestManager creates a PortManager with a mocked port checker and in-memory config client for testing.
func setupTestManager(tempDir string, unavailablePorts map[int]bool) *PortManager {
	pm := GetPortManager(tempDir)
	if unavailablePorts == nil {
		unavailablePorts = make(map[int]bool)
	}
	pm.portChecker = mockPortChecker(unavailablePorts)
	// Use in-memory config client to avoid needing azd gRPC connection
	pm.SetConfigClient(azdconfig.NewInMemoryClient())
	return pm
}

func TestAssignPort_Explicit_Available(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 9876, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 9876 {
		t.Errorf("Expected port 9876, got %d", port)
	}

	port, exists := pm.GetAssignment("test-service")
	if !exists {
		t.Fatal("Expected assignment to exist")
	}

	if port != 9876 {
		t.Errorf("Expected saved port 9876, got %d", port)
	}
}

func TestAssignPort_Explicit_OutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	_, _, err := pm.AssignPort("test-service", 100, true)
	if err == nil {
		t.Fatal("Expected error for port outside range, got nil")
	}

	expectedErr := "explicit port 100 for service 'test-service' is outside valid range 3000-65535"
	if err.Error() != expectedErr {
		t.Errorf("Expected error: %s, got: %v", expectedErr, err)
	}
}

func TestAssignPort_Flexible_Available(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 9877, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 9877 {
		t.Errorf("Expected port 9877, got %d", port)
	}
}

func TestAssignPort_Flexible_FindsAlternative(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port1, _, err := pm.AssignPort("service1", 9878, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port1 != 9878 {
		t.Errorf("Expected port 9878, got %d", port1)
	}

	port2, _, err := pm.AssignPort("service2", 9878, false)
	if err != nil {
		t.Fatalf("Expected no error for flexible port, got: %v", err)
	}

	if port2 < 3000 || port2 > 9999 {
		t.Errorf("Expected port in range 3000-9999, got %d", port2)
	}
}

func TestAssignPort_Persistence(t *testing.T) {
	tempDir := t.TempDir()

	pm := setupTestManager(tempDir, nil)
	port1, _, err := pm.AssignPort("test-service", 9879, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	port2, exists := pm.GetAssignment("test-service")
	if !exists {
		t.Fatal("Expected assignment to exist in same manager instance")
	}

	if port2 != port1 {
		t.Errorf("Expected stored port %d, got %d", port1, port2)
	}
}

func TestAssignPort_SameServiceTwice(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port1, _, err := pm.AssignPort("test-service", 9880, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	port2, _, err := pm.AssignPort("test-service", 8888, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port1 != port2 {
		t.Errorf("Expected same port on reassignment, got %d and %d", port1, port2)
	}
}

func TestGetAssignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	_, exists := pm.GetAssignment("nonexistent")
	if exists {
		t.Error("Expected assignment to not exist")
	}

	expectedPort := 9882
	if _, _, err := pm.AssignPort("test-service", expectedPort, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	port, exists := pm.GetAssignment("test-service")
	if !exists {
		t.Fatal("Expected assignment to exist")
	}

	if port != expectedPort {
		t.Errorf("Expected port %d, got %d", expectedPort, port)
	}
}

func TestCleanStaleAssignments(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	pm.mu.Lock()
	pm.assignments["stale-service"] = &PortAssignment{
		ServiceName: "stale-service",
		Port:        9883,
		LastUsed:    time.Now().Add(-25 * time.Hour),
	}
	pm.mu.Unlock()

	if _, _, err := pm.AssignPort("active-service", 9884, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	if err := pm.CleanStalePorts(); err != nil {
		t.Fatalf("CleanStalePorts failed: %v", err)
	}

	if _, exists := pm.GetAssignment("active-service"); !exists {
		t.Error("Expected active assignment to remain")
	}
}

func TestIsPortAvailable(t *testing.T) {
	tempDir := t.TempDir()
	unavailable := map[int]bool{8080: true, 9090: true}
	pm := setupTestManager(tempDir, unavailable)

	if pm.isPortAvailable(8080) {
		t.Error("Port 8080 should NOT be available")
	}

	if !pm.isPortAvailable(3000) {
		t.Error("Port 3000 should be available")
	}
}

func TestFindAvailablePort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, err := pm.findAvailablePort()
	if err != nil {
		t.Fatalf("Expected to find available port, got error: %v", err)
	}

	if port < 3000 || port > 65535 {
		t.Errorf("Expected port in valid range, got %d", port)
	}

	t.Logf("Found available port: %d", port)
}

func TestMultipleServicesAssignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	services := map[string]int{
		"frontend": 9887,
		"backend":  9888,
		"api":      9889,
		"worker":   9890,
	}

	for name, preferredPort := range services {
		port, _, err := pm.AssignPort(name, preferredPort, false)
		if err != nil {
			t.Fatalf("Failed to assign port for %s: %v", name, err)
		}

		if port != preferredPort {
			t.Errorf("Service %s: expected port %d, got %d", name, preferredPort, port)
		}
	}

	for name, expectedPort := range services {
		port, exists := pm.GetAssignment(name)
		if !exists {
			t.Errorf("Expected assignment for %s to exist", name)
			continue
		}

		if port != expectedPort {
			t.Errorf("Service %s: expected port %d, got %d", name, expectedPort, port)
		}
	}
}

func TestAssignPort_HighPortNumber(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 65535, true)
	if err != nil {
		t.Fatalf("Expected no error for port 65535, got: %v", err)
	}

	if port != 65535 {
		t.Errorf("Expected port 65535, got %d", port)
	}
}

func TestAssignPort_LowValidPort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 3000, true)
	if err != nil {
		t.Fatalf("Expected no error for port 3000, got: %v", err)
	}

	if port != 3000 {
		t.Errorf("Expected port 3000, got %d", port)
	}
}

func TestAssignPort_ExplicitTooHigh(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	_, _, err := pm.AssignPort("test-service", 70000, true)
	if err == nil {
		t.Error("Expected error for port > 65535")
	}
}

func TestAssignPort_ZeroPort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 0, false)
	if err != nil {
		t.Fatalf("Expected no error for port 0, got: %v", err)
	}

	if port < 3000 || port > 65535 {
		t.Errorf("Expected assigned port in valid range 3000-65535, got %d", port)
	}
}

func TestAssignPort_ExplicitMode(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("explicit-service", 9903, true)
	if err != nil {
		t.Fatalf("Failed to assign explicit port: %v", err)
	}

	if port != 9903 {
		t.Errorf("Expected exact port 9903, got %d", port)
	}
}

func TestAssignPort_FlexibleReassignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port1, _, err := pm.AssignPort("service1", 9904, false)
	if err != nil {
		t.Fatalf("Failed initial assignment: %v", err)
	}

	port2, _, err := pm.AssignPort("service1", 9905, false)
	if err != nil {
		t.Fatalf("Failed reassignment: %v", err)
	}

	if port2 != port1 {
		t.Logf("Service got reassigned from %d to %d", port1, port2)
	}
}

func TestAssignPort_PreferredPortOutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("service", 100, false)
	if err != nil {
		t.Fatalf("Expected to find alternative port, got error: %v", err)
	}

	if port < 3000 || port > 65535 {
		t.Errorf("Expected port in valid range, got %d", port)
	}
}

func TestIsPortAvailableEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	unavailable := map[int]bool{8000: true, 0: true}
	pm := setupTestManager(tempDir, unavailable)

	if pm.isPortAvailable(8000) {
		t.Error("Expected port 8000 to be in use")
	}

	if pm.isPortAvailable(0) {
		t.Error("Expected port 0 to be unavailable")
	}

	if !pm.isPortAvailable(5000) {
		t.Error("Expected port 5000 to be available")
	}
}
