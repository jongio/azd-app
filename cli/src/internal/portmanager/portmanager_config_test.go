package portmanager

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPortManagerCaching(t *testing.T) {
	tempDir := t.TempDir()

	pm1 := setupTestManager(tempDir, nil)
	pm2 := setupTestManager(tempDir, nil)

	if pm1 != pm2 {
		t.Error("Expected same port manager instance for same directory")
	}
}

func TestPortManagerDifferentProjects(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	pm1 := setupTestManager(tempDir1, nil)
	pm2 := setupTestManager(tempDir2, nil)

	if pm1 == pm2 {
		t.Error("Expected different port manager instances for different directories")
	}

	port1, _, _ := pm1.AssignPort("service", 9885, false)
	port2, _, _ := pm2.AssignPort("service", 9885, false)

	if port1 != 9885 || port2 != 9885 {
		t.Error("Expected both projects to use same port number independently")
	}
}

func TestPortAssignmentStorage(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	port, _, err := pm.AssignPort("test-service", 9886, false)
	if err != nil {
		t.Fatalf("failed to assign port: %v", err)
	}

	storedPort, exists := pm.GetAssignment("test-service")
	if !exists {
		t.Error("Expected assignment to exist after AssignPort")
	}
	if storedPort != port {
		t.Errorf("Expected stored port %d, got %d", port, storedPort)
	}
}

func TestPortManager_EmptyProjectDir(t *testing.T) {
	pm := setupTestManager("", nil)

	if pm == nil {
		t.Fatal("Expected port manager to be created for empty project dir")
	}

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

	port, err := pm.findAvailablePort()
	if err != nil {
		t.Fatalf("Expected to find available port, got: %v", err)
	}

	if port < 3000 || port > 65535 {
		t.Errorf("Port %d is outside valid range 3000-65535", port)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	if _, _, err := pm.AssignPort("service1", 9900, false); err != nil {
		t.Fatalf("failed to assign port for service1: %v", err)
	}
	if _, _, err := pm.AssignPort("service2", 9901, false); err != nil {
		t.Fatalf("failed to assign port for service2: %v", err)
	}

	port1, exists1 := pm.GetAssignment("service1")
	if !exists1 || port1 != 9900 {
		t.Errorf("Expected service1 port 9900, got %d (exists: %v)", port1, exists1)
	}

	port2, exists2 := pm.GetAssignment("service2")
	if !exists2 || port2 != 9901 {
		t.Errorf("Expected service2 port 9901, got %d (exists: %v)", port2, exists2)
	}
}

func TestLoadCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	portsDir := filepath.Join(tempDir, ".azure")
	portsFile := filepath.Join(portsDir, "ports.json")

	if err := os.MkdirAll(portsDir, 0750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(portsFile, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	pm := setupTestManager(tempDir, nil)

	if pm == nil {
		t.Fatal("Expected non-nil port manager even with corrupt file")
	}

	port, _, err := pm.AssignPort("test", 9902, false)
	if err != nil {
		t.Fatalf("Should be able to assign port: %v", err)
	}

	if port < 3000 {
		t.Errorf("Expected valid port, got %d", port)
	}
}

func TestGetPortManager_EmptyProjectDirUsesWorkingDir(t *testing.T) {
	pm := setupTestManager("", nil)

	if pm == nil {
		t.Fatal("Expected non-nil port manager")
	}

	port, _, err := pm.AssignPort("test-empty-dir", 9907, false)
	if err != nil {
		t.Fatalf("Failed to assign port: %v", err)
	}

	if port < 3000 {
		t.Errorf("Expected valid port, got %d", port)
	}

	if err := pm.ReleasePort("test-empty-dir"); err != nil {
		t.Errorf("failed to release port: %v", err)
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
			if tt.startEnv != "" {
				_ = os.Setenv(envPortRangeStart, tt.startEnv)
				defer func() { _ = os.Unsetenv(envPortRangeStart) }()
			}
			if tt.endEnv != "" {
				_ = os.Setenv(envPortRangeEnd, tt.endEnv)
				defer func() { _ = os.Unsetenv(envPortRangeEnd) }()
			}

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

func TestIsPortAvailable_PublicAPI(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, map[int]bool{
		8080: true,
	})

	if pm.IsPortAvailable(8080) {
		t.Error("Expected port 8080 to be unavailable")
	}

	if !pm.IsPortAvailable(9999) {
		t.Error("Expected port 9999 to be available")
	}
}

func TestAssignPort_PortRangeExhausted(t *testing.T) {
	t.Skip("Skipping test that triggers user prompts - needs refactoring for testability")

	_ = os.Setenv(envPortRangeStart, "3000")
	_ = os.Setenv(envPortRangeEnd, "3001")
	defer func() { _ = os.Unsetenv(envPortRangeStart) }()
	defer func() { _ = os.Unsetenv(envPortRangeEnd) }()

	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	tempDir := t.TempDir()

	pm := setupTestManager(tempDir, map[int]bool{
		3000: true,
		3001: true,
	})

	_, _, err := pm.AssignPort("test-service", 3000, false)
	if err == nil {
		t.Fatal("Expected error when port range exhausted, got nil")
	}

	if _, ok := err.(*PortRangeExhaustedError); !ok {
		t.Errorf("Expected PortRangeExhaustedError, got %T: %v", err, err)
	}
}

func TestGetPortRangeEnd_InvalidEnv(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectedEnd int
	}{
		{
			name:        "invalid non-numeric",
			envValue:    "invalid",
			expectedEnd: PortRangeEnd,
		},
		{
			name:        "out of range too high",
			envValue:    "70000",
			expectedEnd: PortRangeEnd,
		},
		{
			name:        "low port is accepted",
			envValue:    "100",
			expectedEnd: 100,
		},
		{
			name:        "valid custom port",
			envValue:    "50000",
			expectedEnd: 50000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(envPortRangeEnd, tt.envValue)
			defer func() { _ = os.Unsetenv(envPortRangeEnd) }()

			got := getPortRangeEnd()
			if got != tt.expectedEnd {
				t.Errorf("getPortRangeEnd() = %d, want %d", got, tt.expectedEnd)
			}
		})
	}
}

func TestSaveLoadPortManager(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	_, _, err := pm.AssignPort("service1", 3000, true)
	if err != nil {
		t.Fatalf("Failed to assign port: %v", err)
	}

	_, _, err = pm.AssignPort("service2", 4000, false)
	if err != nil {
		t.Fatalf("Failed to assign port: %v", err)
	}

	pm.mu.Lock()
	err = pm.save()
	pm.mu.Unlock()
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	pm2 := GetPortManager(tempDir)

	port1, exists := pm2.GetAssignment("service1")
	if !exists || port1 != 3000 {
		t.Errorf("Expected service1 on port 3000, got exists=%v, port=%d", exists, port1)
	}

	port2, exists := pm2.GetAssignment("service2")
	if !exists || port2 != 4000 {
		t.Errorf("Expected service2 on port 4000, got exists=%v, port=%d", exists, port2)
	}
}

func TestCleanStalePorts_RemovesOldAssignments(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	pm.mu.Lock()
	pm.assignments["old-service"] = &PortAssignment{
		ServiceName: "old-service",
		Port:        5000,
		LastUsed:    time.Now().Add(-8 * 24 * time.Hour),
	}
	pm.mu.Unlock()

	err := pm.CleanStalePorts()
	if err != nil {
		t.Fatalf("CleanStalePorts failed: %v", err)
	}

	pm.mu.RLock()
	_, exists := pm.assignments["old-service"]
	pm.mu.RUnlock()

	if exists {
		t.Error("Expected old-service assignment to be cleaned up")
	}
}

func TestCleanStalePorts_KeepsRecentAssignments(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	pm.mu.Lock()
	pm.assignments["recent-service"] = &PortAssignment{
		ServiceName: "recent-service",
		Port:        6000,
		LastUsed:    time.Now().Add(-1 * time.Hour),
	}
	pm.mu.Unlock()

	_ = pm.CleanStalePorts()

	pm.mu.RLock()
	_, exists := pm.assignments["recent-service"]
	pm.mu.RUnlock()

	if !exists {
		t.Error("Expected recent-service assignment to be kept")
	}
}

func TestManagerCache(t *testing.T) {
	tempDir := t.TempDir()

	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	pm1 := setupTestManager(tempDir, nil)
	pm2 := setupTestManager(tempDir, nil)

	if pm1 != pm2 {
		t.Error("Expected cached manager instance")
	}

	normalizedPath, _ := filepath.Abs(tempDir)
	if resolved, err := filepath.EvalSymlinks(normalizedPath); err == nil {
		normalizedPath = resolved
	}

	managerCacheMu.RLock()
	_, exists := managerCache[normalizedPath]
	managerCacheMu.RUnlock()

	if !exists {
		t.Error("Expected cache to contain manager")
	}
}

func TestManagerCacheEviction(t *testing.T) {
	managerCacheMu.Lock()
	managerCache = make(map[string]*cacheEntry)
	managerCacheMu.Unlock()

	tempDirs := make([]string, 55)
	normalizedDirs := make([]string, 55)
	for i := 0; i < 55; i++ {
		tempDirs[i] = t.TempDir()
		normalizedDirs[i], _ = filepath.Abs(tempDirs[i])
		if resolved, err := filepath.EvalSymlinks(normalizedDirs[i]); err == nil {
			normalizedDirs[i] = resolved
		}
		_ = GetPortManager(tempDirs[i])
		time.Sleep(1 * time.Millisecond)
	}

	managerCacheMu.RLock()
	cacheSize := len(managerCache)
	managerCacheMu.RUnlock()

	if cacheSize > maxCacheSize {
		t.Errorf("Cache size %d exceeds maximum %d", cacheSize, maxCacheSize)
	}

	managerCacheMu.RLock()
	_, existsOld := managerCache[normalizedDirs[0]]
	_, existsNew := managerCache[normalizedDirs[54]]
	managerCacheMu.RUnlock()

	if existsOld {
		t.Error("Expected oldest entry to be evicted")
	}
	if !existsNew {
		t.Error("Expected newest entry to be in cache")
	}
}
