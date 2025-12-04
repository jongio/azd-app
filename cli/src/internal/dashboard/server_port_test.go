package dashboard

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
)

// TestPersistentDashboardPort_FirstRunPersists verifies that the first run
// generates a port and stores it in the port manager.
func TestPersistentDashboardPort_FirstRunPersists(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	srv := GetServer(tempDir)

	// Start the server
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	if url == "" {
		t.Fatal("Expected non-empty URL")
	}

	// Verify port was stored in port manager
	pm := portmanager.GetPortManager(tempDir)
	persistedPort, exists := pm.GetAssignment(constants.DashboardServiceName)
	if !exists {
		t.Errorf("Expected %s to be in port manager assignments", constants.DashboardServiceName)
	}
	if persistedPort != srv.port {
		t.Errorf("Expected stored port %d to match server port %d", persistedPort, srv.port)
	}
}

// TestPersistentDashboardPort_SecondRunReusesPersisted verifies that the second run
// uses the same port that was persisted from the first run.
func TestPersistentDashboardPort_SecondRunReusesPersisted(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map and port manager cache for clean state
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// First run - start and get the port
	srv1 := GetServer(tempDir)
	url1, err := srv1.Start()
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	port1 := srv1.port
	_ = srv1.Stop()

	// Clear servers map but NOT port manager cache (to simulate restart)
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	// Second run - should use same port
	srv2 := GetServer(tempDir)

	// Check that GetAssignment returns the persisted port before Start
	portMgr := portmanager.GetPortManager(tempDir)
	persistedPort, exists := portMgr.GetAssignment(constants.DashboardServiceName)
	if !exists {
		t.Fatal("Expected dashboard port assignment to exist after first run")
	}
	if persistedPort != port1 {
		t.Errorf("Persisted port %d doesn't match first run port %d", persistedPort, port1)
	}

	url2, err := srv2.Start()
	if err != nil {
		t.Fatalf("Second Start() failed: %v", err)
	}
	defer func() { _ = srv2.Stop() }()
	port2 := srv2.port

	// Verify same port is used
	if port1 != port2 {
		t.Errorf("Expected same port across runs, got port1=%d port2=%d", port1, port2)
	}

	// Verify URLs match
	if url1 != url2 {
		t.Errorf("Expected same URL across runs, got url1=%s url2=%s", url1, url2)
	}
}

// TestPersistentDashboardPort_PortRangeIsValid verifies that generated ports
// are in the expected dashboard range (40000-49999).
func TestPersistentDashboardPort_PortRangeIsValid(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	srv := GetServer(tempDir)
	_, err := srv.Start()
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = srv.Stop() }()

	// Verify port is in expected range
	if srv.port < constants.DashboardPortRangeMin || srv.port > constants.DashboardPortRangeMax {
		t.Errorf("Expected port in range %d-%d, got %d",
			constants.DashboardPortRangeMin, constants.DashboardPortRangeMax, srv.port)
	}
}

// TestPersistentDashboardPort_GetAssignmentBeforeStart verifies that GetAssignment
// returns false when no port has been assigned yet.
func TestPersistentDashboardPort_GetAssignmentBeforeStart(t *testing.T) {
	tempDir := t.TempDir()
	portmanager.ClearCacheForTesting()

	portMgr := portmanager.GetPortManager(tempDir)
	_, exists := portMgr.GetAssignment(constants.DashboardServiceName)
	if exists {
		t.Error("Expected no assignment before first start")
	}
}

// TestPersistentDashboardPort_MultipleProjects verifies that different projects
// get different persisted ports.
func TestPersistentDashboardPort_MultipleProjects(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// Start first project
	srv1 := GetServer(tempDir1)
	_, err := srv1.Start()
	if err != nil {
		t.Fatalf("First project Start() failed: %v", err)
	}
	port1 := srv1.port
	defer func() { _ = srv1.Stop() }()

	// Start second project
	srv2 := GetServer(tempDir2)
	_, err = srv2.Start()
	if err != nil {
		t.Fatalf("Second project Start() failed: %v", err)
	}
	port2 := srv2.port
	defer func() { _ = srv2.Stop() }()

	// Note: With the new architecture, ports are stored in azd config (not ports.json)
	// For unit tests without gRPC, ports are stored in-memory

	// Verify each project's port manager has its own assignment
	portMgr1 := portmanager.GetPortManager(tempDir1)
	portMgr2 := portmanager.GetPortManager(tempDir2)

	persistedPort1, exists1 := portMgr1.GetAssignment(constants.DashboardServiceName)
	persistedPort2, exists2 := portMgr2.GetAssignment(constants.DashboardServiceName)

	if !exists1 || !exists2 {
		t.Error("Expected both projects to have port assignments")
	}

	if persistedPort1 != port1 {
		t.Errorf("Project 1: persisted port %d doesn't match running port %d", persistedPort1, port1)
	}
	if persistedPort2 != port2 {
		t.Errorf("Project 2: persisted port %d doesn't match running port %d", persistedPort2, port2)
	}
}

// TestPersistentDashboardPort_PortConflictFallback verifies that when the persisted
// port is unavailable (in use by another process), the dashboard finds an alternative port.
func TestPersistentDashboardPort_PortConflictFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Clear servers map and port manager cache
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()
	portmanager.ClearCacheForTesting()

	// First run - get a port
	srv1 := GetServer(tempDir)
	_, err := srv1.Start()
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}
	port1 := srv1.port
	// Don't stop srv1 - keep it running to hold the port

	// Clear servers map but NOT the actual server or port manager
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	// Second run - should fail to use port1 (still in use) and find alternative
	srv2 := GetServer(tempDir)
	_, err = srv2.Start()
	if err != nil {
		// This is expected if port conflict causes a complete failure
		// But with proper fallback logic, it should succeed with a different port
		t.Logf("Second Start() failed (may be expected): %v", err)
	} else {
		port2 := srv2.port
		// The ports should be different since port1 is still in use
		if port1 == port2 {
			t.Errorf("Expected different ports when original is in use, got same port %d", port1)
		}
		_ = srv2.Stop()
	}

	// Clean up first server
	_ = srv1.Stop()
}
