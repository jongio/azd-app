package portmanager

import (
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/azdconfig"
)

func TestGetAlwaysKillPreference_Default(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// By default, always-kill should be false
	result := pm.getAlwaysKillPreference()
	if result != false {
		t.Errorf("getAlwaysKillPreference() = %v, want false by default", result)
	}
}

func TestGetAlwaysKillPreference_SessionFlag(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Set session flag
	pm.sessionAlwaysKill = true

	result := pm.getAlwaysKillPreference()
	if result != true {
		t.Errorf("getAlwaysKillPreference() = %v, want true when session flag set", result)
	}

	// Reset and verify it returns to false
	pm.sessionAlwaysKill = false
	result = pm.getAlwaysKillPreference()
	if result != false {
		t.Errorf("getAlwaysKillPreference() = %v, want false after resetting session flag", result)
	}
}

func TestSetAlwaysKillPreference(t *testing.T) {
	tempDir := t.TempDir()
	pm := setupTestManager(tempDir, nil)

	// Set to true
	err := pm.setAlwaysKillPreference(true)
	if err != nil {
		t.Fatalf("setAlwaysKillPreference(true) error = %v", err)
	}

	// Verify it was set
	result := pm.getAlwaysKillPreference()
	if result != true {
		t.Errorf("getAlwaysKillPreference() = %v, want true after setting preference", result)
	}

	// Set to false
	err = pm.setAlwaysKillPreference(false)
	if err != nil {
		t.Fatalf("setAlwaysKillPreference(false) error = %v", err)
	}

	// Verify it was set
	result = pm.getAlwaysKillPreference()
	if result != false {
		t.Errorf("getAlwaysKillPreference() = %v, want false after setting preference", result)
	}
}

func TestSetAlwaysKillPreference_Persistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create shared in-memory client
	sharedClient := azdconfig.NewInMemoryClient()

	// First manager
	pm := GetPortManager(tempDir)
	pm.SetConfigClient(sharedClient)

	// Set preference to true
	err := pm.setAlwaysKillPreference(true)
	if err != nil {
		t.Fatalf("setAlwaysKillPreference(true) error = %v", err)
	}

	// Get new manager instance for same project using same config client
	pm2 := GetPortManager(tempDir)
	pm2.SetConfigClient(sharedClient)

	// Verify persistence - the preference should still be true
	result := pm2.getAlwaysKillPreference()
	if result != true {
		t.Errorf("getAlwaysKillPreference() = %v, want true from persisted preference", result)
	}
}

func TestClearCacheForTesting(t *testing.T) {
	tempDir := t.TempDir()

	// Create first manager
	pm1 := setupTestManager(tempDir, nil)

	// Assign a port
	port, _, err := pm1.AssignPort("test-service", 9999, false)
	if err != nil {
		t.Fatalf("AssignPort() error = %v", err)
	}
	if port != 9999 {
		t.Errorf("AssignPort() = %v, want 9999", port)
	}

	// Clear cache
	ClearCacheForTesting()

	// Create new manager for same directory
	pm2 := GetPortManager(tempDir)
	pm2.SetConfigClient(azdconfig.NewInMemoryClient())

	// They should be different instances after cache clear
	if pm1 == pm2 {
		t.Error("Expected different instances after ClearCacheForTesting")
	}
}

func TestSetTestModeForTesting(t *testing.T) {
	// Create a mock port checker
	unavailablePorts := make(map[int]bool)
	mockChecker := mockPortChecker(unavailablePorts)

	// Enable test mode with mock checker
	cleanup := SetTestModeForTesting(mockChecker)
	defer cleanup() // Restore original state

	// Create a manager
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// In test mode, config client should be in-memory
	// We can verify this by checking that operations work without azd connection
	pm.SetConfigClient(azdconfig.NewInMemoryClient())

	port, _, err := pm.AssignPort("test", 8888, false)
	if err != nil {
		t.Errorf("Expected port assignment to work in test mode, got error: %v", err)
	}
	if port != 8888 {
		t.Errorf("AssignPort() = %v, want 8888", port)
	}
}

func TestGetAlwaysKillPreference_WithNilClient(t *testing.T) {
	tempDir := t.TempDir()
	pm := GetPortManager(tempDir)

	// Don't set a config client - it will try to create one
	// This test ensures the function handles errors gracefully
	pm.configClient = nil

	// Should return false and not panic when config client fails
	result := pm.getAlwaysKillPreference()
	if result != false {
		t.Errorf("getAlwaysKillPreference() = %v, want false when config client unavailable", result)
	}
}
