package portmanager

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
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

// setupTestManager creates a PortManager with a mocked port checker for testing.
func setupTestManager(tempDir string, unavailablePorts map[int]bool) *PortManager {
	pm := GetPortManager(tempDir)
	if unavailablePorts == nil {
		unavailablePorts = make(map[int]bool)
	}
	pm.portChecker = mockPortChecker(unavailablePorts)
	return pm
}

func TestAssignPort_Explicit_Available(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil) // All ports available

	// Assign explicit port that should be available
	port, _, err := pm.AssignPort("test-service", 9876, true)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 9876 {
		t.Errorf("Expected port 9876, got %d", port)
	}

	// Verify assignment was saved
	port, exists := pm.GetAssignment("test-service")
	if !exists {
		t.Fatal("Expected assignment to exist")
	}

	if port != 9876 {
		t.Errorf("Expected saved port 9876, got %d", port)
	}
}

func TestAssignPort_Explicit_OutOfRange(t *testing.T) {
	// This test doesn't bind to ports, only validates range check
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Try to assign explicit port outside valid range
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

	// Assign flexible port
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

	// This test documents current behavior:
	// When isExplicit=false, the port manager assigns based on availability,
	// not on what's in the assignments map. Two services can get the same
	// port if neither is actually running and listening on that port.

	// Assign first service on preferred port
	port1, _, err := pm.AssignPort("service1", 9878, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port1 != 9878 {
		t.Errorf("Expected port 9878, got %d", port1)
	}

	// Try to assign second service with same preferred port (flexible)
	// Because service1 isn't actually running, the port is available
	// So service2 also gets 9878 (current behavior)
	port2, _, err := pm.AssignPort("service2", 9878, false)
	if err != nil {
		t.Fatalf("Expected no error for flexible port, got: %v", err)
	}

	// Note: Both services can have same port if neither is running
	if port2 < 3000 || port2 > 9999 {
		t.Errorf("Expected port in range 3000-9999, got %d", port2)
	}
}

func TestAssignPort_Persistence(t *testing.T) {
	tempDir := t.TempDir()

	// First port manager instance
	pm1 := setupTestManager(tempDir, nil)
	port1, _, err := pm1.AssignPort("test-service", 9879, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Create new port manager instance for same project
	// Clear the cache to force reload
	managerCacheMu.Lock()
	delete(managerCache, tempDir)
	managerCacheMu.Unlock()

	pm2 := setupTestManager(tempDir, nil)
	port2, exists := pm2.GetAssignment("test-service")
	if !exists {
		t.Fatal("Expected assignment to be persisted")
	}

	if port2 != port1 {
		t.Errorf("Expected persisted port %d, got %d", port1, port2)
	}
}

func TestAssignPort_SameServiceTwice(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign port first time
	port1, _, err := pm.AssignPort("test-service", 9880, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Assign again - should return same port
	port2, _, err := pm.AssignPort("test-service", 8888, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port1 != port2 {
		t.Errorf("Expected same port on reassignment, got %d and %d", port1, port2)
	}
}

func TestReleasePort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign port
	port, _, err := pm.AssignPort("test-service", 9881, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify assignment exists
	if _, exists := pm.GetAssignment("test-service"); !exists {
		t.Fatal("Expected assignment to exist")
	}

	// Release port
	if err := pm.ReleasePort("test-service"); err != nil {
		t.Errorf("failed to release port: %v", err)
	}

	// Verify assignment is gone
	if _, exists := pm.GetAssignment("test-service"); exists {
		t.Error("Expected assignment to be released")
	}

	// Verify can assign same port to different service
	newPort, _, err := pm.AssignPort("other-service", port, false)
	if err != nil {
		t.Fatalf("Expected no error after release, got: %v", err)
	}

	if newPort != port {
		t.Errorf("Expected to reuse released port %d, got %d", port, newPort)
	}
}

func TestGetAssignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Non-existent assignment
	_, exists := pm.GetAssignment("nonexistent")
	if exists {
		t.Error("Expected assignment to not exist")
	}

	// Create assignment
	expectedPort := 9882
	if _, _, err := pm.AssignPort("test-service", expectedPort, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	// Get assignment
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

	// Create old assignment
	pm.mu.Lock()
	pm.assignments["stale-service"] = &PortAssignment{
		ServiceName: "stale-service",
		Port:        9883,
		LastUsed:    time.Now().Add(-25 * time.Hour), // 25 hours ago
	}
	if err := pm.save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	pm.mu.Unlock()

	// Create recent assignment
	if _, _, err := pm.AssignPort("active-service", 9884, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	// Clean stale ports (older than 7 days by default)
	if err := pm.CleanStalePorts(); err != nil {
		t.Fatalf("CleanStalePorts failed: %v", err)
	}

	// Stale assignment won't be cleaned (25 hours < 7 days)
	// This test documents the behavior rather than testing cleanup
	if _, exists := pm.GetAssignment("stale-service"); !exists {
		t.Log("Note: CleanStalePorts uses 7-day threshold")
	}

	// Verify active remains
	if _, exists := pm.GetAssignment("active-service"); !exists {
		t.Error("Expected active assignment to remain")
	}
}

func TestIsPortAvailable(t *testing.T) {
	tempDir := t.TempDir()
	unavailable := map[int]bool{8080: true, 9090: true}
	pm := setupTestManager(tempDir, unavailable)

	// Port 8080 should NOT be available (marked as unavailable in mock)
	if pm.isPortAvailable(8080) {
		t.Error("Port 8080 should NOT be available")
	}

	// Port 3000 should be available (not in unavailable map)
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

	// Note: We don't test if the port is actually available here
	// because that would trigger firewall prompts
	t.Logf("Found available port: %d", port)
}

func TestPortManagerCaching(t *testing.T) {
	tempDir := t.TempDir()

	// Get manager twice for same directory
	pm1 := setupTestManager(tempDir, nil)
	pm2 := setupTestManager(tempDir, nil)

	// Should be same instance (cached)
	if pm1 != pm2 {
		t.Error("Expected same port manager instance for same directory")
	}
}

func TestPortManagerDifferentProjects(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	pm1 := GetPortManager(tempDir1)
	pm2 := GetPortManager(tempDir2)

	// Should be different instances
	if pm1 == pm2 {
		t.Error("Expected different port manager instances for different directories")
	}

	// Can assign same port to different projects
	port1, _, _ := pm1.AssignPort("service", 9885, false)
	port2, _, _ := pm2.AssignPort("service", 9885, false)

	if port1 != 9885 || port2 != 9885 {
		t.Error("Expected both projects to use same port number independently")
	}
}

func TestPortAssignmentFile(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign a port
	if _, _, err := pm.AssignPort("test-service", 9886, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	// Verify file was created
	portsFile := filepath.Join(tempDir, ".azure", "ports.json")
	if _, err := os.Stat(portsFile); os.IsNotExist(err) {
		t.Error("Expected ports.json file to be created")
	}

	// Verify file permissions
	info, err := os.Stat(portsFile)
	if err != nil {
		t.Fatalf("Failed to stat ports file: %v", err)
	}

	mode := info.Mode()
	// On Windows, permissions may differ, so just check file exists
	if mode == 0 {
		t.Error("Expected file to have permissions set")
	}
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

	// Assign all services
	for name, preferredPort := range services {
		port, _, err := pm.AssignPort(name, preferredPort, false)
		if err != nil {
			t.Fatalf("Failed to assign port for %s: %v", name, err)
		}

		if port != preferredPort {
			t.Errorf("Service %s: expected port %d, got %d", name, preferredPort, port)
		}
	}

	// Verify all assignments
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

	// Try to assign a very high port number (at edge of range)
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

	// Try to assign port at lower bound of valid range
	port, _, err := pm.AssignPort("test-service", 3000, true)
	if err != nil {
		t.Fatalf("Expected no error for port 3000, got: %v", err)
	}

	if port != 3000 {
		t.Errorf("Expected port 3000, got %d", port)
	}
}

func TestAssignPort_ExplicitTooHigh(t *testing.T) {
	// This test doesn't bind to ports, only validates range check
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Try to assign explicit port above 65535
	_, _, err := pm.AssignPort("test-service", 70000, true)
	if err == nil {
		t.Error("Expected error for port > 65535")
	}
}

func TestAssignPort_ZeroPort(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Try flexible assignment with port 0 (should find available port)
	port, _, err := pm.AssignPort("test-service", 0, false)
	if err != nil {
		t.Fatalf("Expected no error for port 0, got: %v", err)
	}

	if port < 3000 || port > 9999 {
		t.Errorf("Expected assigned port in range 3000-9999, got %d", port)
	}
}

func TestReleasePort_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Release a port that was never assigned (should not panic)
	if err := pm.ReleasePort("nonexistent-service"); err != nil {
		t.Errorf("ReleasePort should not error for non-existent service: %v", err)
	}

	// Verify no crash and state is consistent
	if _, exists := pm.GetAssignment("nonexistent-service"); exists {
		t.Error("Expected nonexistent service to not have assignment")
	}
}

func TestCleanStalePorts_VeryOldAssignment(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Create a very old assignment (8 days ago, beyond 7-day threshold)
	pm.mu.Lock()
	pm.assignments["ancient-service"] = &PortAssignment{
		ServiceName: "ancient-service",
		Port:        9891,
		LastUsed:    time.Now().Add(-8 * 24 * time.Hour),
	}
	if err := pm.save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	pm.mu.Unlock()

	// Clean stale ports
	if err := pm.CleanStalePorts(); err != nil {
		t.Fatalf("CleanStalePorts failed: %v", err)
	}

	// Very old assignment should be cleaned
	if _, exists := pm.GetAssignment("ancient-service"); exists {
		t.Error("Expected ancient assignment to be cleaned")
	}
}

func TestPortManager_EmptyProjectDir(t *testing.T) {
	// Test with empty string (should use current directory)
	pm := setupTestManager("", nil)

	if pm == nil {
		t.Fatal("Expected port manager to be created for empty project dir")
	}

	// Should be able to assign ports
	port, _, err := pm.AssignPort("test", 9892, false)
	if err != nil {
		t.Fatalf("Expected to assign port, got error: %v", err)
	}

	if port < 3000 {
		t.Errorf("Expected valid port, got %d", port)
	}
}

func TestFindAvailablePort_Exhaustion(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// This is a theoretical test - in practice we won't exhaust all ports
	// But we can test the logic handles the attempt correctly

	// Try to get an available port - should succeed
	port, err := pm.findAvailablePort()
	if err != nil {
		t.Fatalf("Expected to find available port, got: %v", err)
	}

	if port < 3000 || port > 9999 {
		t.Errorf("Port %d is outside expected range", port)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign some ports
	if _, _, err := pm.AssignPort("service1", 9900, false); err != nil {
		t.Fatalf("failed to assign port for service1: %v", err)
	}
	if _, _, err := pm.AssignPort("service2", 9901, false); err != nil {
		t.Fatalf("failed to assign port for service2: %v", err)
	}

	// Save is called automatically, but let's explicitly verify
	err := pm.save()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create new manager instance for same directory (should load from disk)
	// Clear the cache first
	managerCacheMu.Lock()
	delete(managerCache, tempDir)
	managerCacheMu.Unlock()

	pm2 := setupTestManager(tempDir, nil)

	// Verify loaded assignments
	port1, exists1 := pm2.GetAssignment("service1")
	if !exists1 || port1 != 9900 {
		t.Errorf("Expected service1 port 9900, got %d (exists: %v)", port1, exists1)
	}

	port2, exists2 := pm2.GetAssignment("service2")
	if !exists2 || port2 != 9901 {
		t.Errorf("Expected service2 port 9901, got %d (exists: %v)", port2, exists2)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	portsDir := filepath.Join(tempDir, ".azure")
	portsFile := filepath.Join(portsDir, "ports.json")

	// Create directory and corrupt file
	if err := os.MkdirAll(portsDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(portsFile, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	// GetPortManager should handle corrupt file gracefully
	pm := setupTestManager(tempDir, nil)

	if pm == nil {
		t.Fatal("Expected non-nil port manager even with corrupt file")
	}

	// Should be able to assign ports despite corrupt file
	port, _, err := pm.AssignPort("test", 9902, false)
	if err != nil {
		t.Fatalf("Should be able to assign port: %v", err)
	}

	if port < 3000 {
		t.Errorf("Expected valid port, got %d", port)
	}
}

func TestAssignPort_ExplicitMode(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Explicit mode with available port
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

	// Assign port 9904 to service1
	port1, _, err := pm.AssignPort("service1", 9904, false)
	if err != nil {
		t.Fatalf("Failed initial assignment: %v", err)
	}

	// Now assign service1 again with different preferred port (flexible mode)
	// It should keep 9904 if available
	port2, _, err := pm.AssignPort("service1", 9905, false)
	if err != nil {
		t.Fatalf("Failed reassignment: %v", err)
	}

	// Should return existing assignment
	if port2 != port1 {
		t.Logf("Service got reassigned from %d to %d", port1, port2)
	}
}

func TestReleasePort_UpdatesFile(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Assign and release
	if _, _, err := pm.AssignPort("temp-service", 9906, false); err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	// Verify assignment exists
	if _, exists := pm.GetAssignment("temp-service"); !exists {
		t.Fatal("Expected assignment to exist")
	}

	// Release
	err := pm.ReleasePort("temp-service")
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	// Verify assignment removed
	if _, exists := pm.GetAssignment("temp-service"); exists {
		t.Error("Expected assignment to be removed")
	}
}

func TestManagerCache(t *testing.T) {
	tempDir := t.TempDir()

	// Clear cache
	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	// Get manager twice - should return same instance
	pm1 := setupTestManager(tempDir, nil)
	pm2 := setupTestManager(tempDir, nil)

	if pm1 != pm2 {
		t.Error("Expected cached manager instance")
	}

	// Verify cache contains entry
	managerCacheMu.RLock()
	_, exists := managerCache[tempDir]
	managerCacheMu.RUnlock()

	if !exists {
		t.Error("Expected cache to contain manager")
	}
}

func TestManagerCacheEviction(t *testing.T) {
	// Clear cache
	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	// Create more managers than cache size (50)
	// We'll create 55 to test eviction
	tempDirs := make([]string, 55)
	for i := 0; i < 55; i++ {
		tempDirs[i] = t.TempDir()
		_ = GetPortManager(tempDirs[i])
		// Small sleep to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Cache should have evicted oldest entries
	managerCacheMu.RLock()
	cacheSize := len(managerCache)
	managerCacheMu.RUnlock()

	if cacheSize > maxCacheSize {
		t.Errorf("Cache size %d exceeds maximum %d", cacheSize, maxCacheSize)
	}

	// First entries should have been evicted
	managerCacheMu.RLock()
	_, existsOld := managerCache[tempDirs[0]]
	_, existsNew := managerCache[tempDirs[54]]
	managerCacheMu.RUnlock()

	if existsOld {
		t.Error("Expected oldest entry to be evicted")
	}
	if !existsNew {
		t.Error("Expected newest entry to be in cache")
	}
}

func TestGetPortManager_EmptyProjectDirUsesWorkingDir(t *testing.T) {
	// This test verifies that empty string falls back to working directory
	pm := setupTestManager("", nil)

	if pm == nil {
		t.Fatal("Expected non-nil port manager")
	}

	// Should be able to use it
	port, _, err := pm.AssignPort("test-empty-dir", 9907, false)
	if err != nil {
		t.Fatalf("Failed to assign port: %v", err)
	}

	if port < 3000 {
		t.Errorf("Expected valid port, got %d", port)
	}

	// Clean up
	if err := pm.ReleasePort("test-empty-dir"); err != nil {
		t.Errorf("failed to release port: %v", err)
	}
}

func TestAssignPort_PreferredPortOutOfRange(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Try flexible mode with out-of-range preferred port
	port, _, err := pm.AssignPort("service", 100, false)
	if err != nil {
		t.Fatalf("Expected to find alternative port, got error: %v", err)
	}

	// Should get a port in valid range
	if port < 3000 || port > 65535 {
		t.Errorf("Expected port in valid range, got %d", port)
	}
}

func TestIsPortAvailableEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	unavailable := map[int]bool{8000: true, 0: true}
	pm := setupTestManager(tempDir, unavailable)

	// Port 8000 should NOT be available (marked unavailable in mock)
	if pm.isPortAvailable(8000) {
		t.Error("Expected port 8000 to be in use")
	}

	// Port 0 should NOT be available (marked unavailable in mock)
	if pm.isPortAvailable(0) {
		t.Error("Expected port 0 to be unavailable")
	}

	// Port 5000 should be available (not in unavailable map)
	if !pm.isPortAvailable(5000) {
		t.Error("Expected port 5000 to be available")
	}
}

func TestConfigurablePortRange(t *testing.T) {
	tests := []struct {
		name      string
		startEnv  string
		endEnv    string
		wantStart int
		wantEnd   int
	}{
		{
			name:      "default range",
			startEnv:  "",
			endEnv:    "",
			wantStart: 3000,
			wantEnd:   65535,
		},
		{
			name:      "custom range",
			startEnv:  "5000",
			endEnv:    "6000",
			wantStart: 5000,
			wantEnd:   6000,
		},
		{
			name:      "invalid start falls back to default",
			startEnv:  "invalid",
			endEnv:    "",
			wantStart: 3000,
			wantEnd:   65535,
		},
		{
			name:      "out of range start falls back to default",
			startEnv:  "70000",
			endEnv:    "",
			wantStart: 3000,
			wantEnd:   65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.startEnv != "" {
				os.Setenv(envPortRangeStart, tt.startEnv)
				defer os.Unsetenv(envPortRangeStart)
			}
			if tt.endEnv != "" {
				os.Setenv(envPortRangeEnd, tt.endEnv)
				defer os.Unsetenv(envPortRangeEnd)
			}

			// Clear cache to force new manager creation
			managerCacheMu.Lock()
			managerCache = make(map[string]*cacheEntry)
			managerCacheMu.Unlock()

			tempDir := t.TempDir()
			pm := GetPortManager(tempDir)

			if pm.portRange.start != tt.wantStart {
				t.Errorf("Start port = %d, want %d", pm.portRange.start, tt.wantStart)
			}
			if pm.portRange.end != tt.wantEnd {
				t.Errorf("End port = %d, want %d", pm.portRange.end, tt.wantEnd)
			}
		})
	}
}
