package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetRegistry(t *testing.T) {
	tempDir := t.TempDir()

	registry := GetRegistry(tempDir)

	if registry == nil {
		t.Fatal("GetRegistry() returned nil")
	}

	// Verify .azure directory was created
	azureDir := filepath.Join(tempDir, ".azure")
	if _, err := os.Stat(azureDir); os.IsNotExist(err) {
		t.Errorf(".azure directory was not created")
	}

	// Get the same registry again - should return cached instance
	registry2 := GetRegistry(tempDir)
	if registry != registry2 {
		t.Errorf("GetRegistry() returned different instance for same directory")
	}
}

func TestGetRegistryEmptyDir(t *testing.T) {
	// Test with empty project dir (should use current directory)
	registry := GetRegistry("")

	if registry == nil {
		t.Fatal("GetRegistry(\"\") returned nil")
	}
}

func TestRegister(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		PID:        12345,
		Port:       8080,
		URL:        "http://localhost:8080",
		Language:   "go",
		Framework:  "gin",
		Status:     "ready",
		Health:     "healthy",
		StartTime:  time.Now(),
	}

	err := registry.Register(entry)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Verify service was registered
	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Errorf("GetService() service not found after Register()")
	}

	if svc.Name != "test-service" {
		t.Errorf("GetService() Name = %v, want test-service", svc.Name)
	}

	if svc.Port != 8080 {
		t.Errorf("GetService() Port = %v, want 8080", svc.Port)
	}

	// Verify LastChecked was set
	if svc.LastChecked.IsZero() {
		t.Errorf("Register() did not set LastChecked")
	}
}

func TestUnregister(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		Status:    "ready",
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Unregister it
	err := registry.Unregister("test-service")
	if err != nil {
		t.Fatalf("Unregister() error = %v, want nil", err)
	}

	// Verify service was removed
	_, exists := registry.GetService("test-service")
	if exists {
		t.Errorf("GetService() service still exists after Unregister()")
	}
}

func TestUpdateStatus(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		Status:    "starting",
		Health:    "unknown",
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Update status
	err := registry.UpdateStatus("test-service", "ready", "healthy")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v, want nil", err)
	}

	// Verify status was updated
	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Fatal("GetService() service not found")
	}

	if svc.Status != "ready" {
		t.Errorf("UpdateStatus() Status = %v, want ready", svc.Status)
	}

	if svc.Health != "healthy" {
		t.Errorf("UpdateStatus() Health = %v, want healthy", svc.Health)
	}

	// Verify LastChecked was updated
	if svc.LastChecked.IsZero() {
		t.Errorf("UpdateStatus() did not update LastChecked")
	}
}

func TestUpdateStatusNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	err := registry.UpdateStatus("nonexistent-service", "ready", "healthy")
	if err == nil {
		t.Errorf("UpdateStatus() for nonexistent service should fail")
	}
}

func TestGetService(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Get nonexistent service
	_, exists := registry.GetService("nonexistent")
	if exists {
		t.Errorf("GetService() found nonexistent service")
	}

	// Register and get service
	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Errorf("GetService() service not found")
	}

	if svc.Name != "test-service" {
		t.Errorf("GetService() Name = %v, want test-service", svc.Name)
	}
}

func TestListAll(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// List when empty
	services := registry.ListAll()
	if len(services) != 0 {
		t.Errorf("ListAll() length = %v, want 0", len(services))
	}

	// Register multiple services
	for i := 0; i < 3; i++ {
		entry := &ServiceRegistryEntry{
			Name:      string(rune('a'+i)) + "-service",
			Port:      8080 + i,
			StartTime: time.Now(),
		}
		if err := registry.Register(entry); err != nil {
			t.Fatalf("failed to register service %d: %v", i, err)
		}
	}

	// List all services
	services = registry.ListAll()
	if len(services) != 3 {
		t.Errorf("ListAll() length = %v, want 3", len(services))
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()

	// Create first registry and add a service
	registry1 := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		PID:        12345,
		Port:       8080,
		URL:        "http://localhost:8080",
		Language:   "go",
		Status:     "ready",
		Health:     "healthy",
		StartTime:  time.Now(),
	}

	err := registry1.Register(entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Clear the cache to force reload
	registryCacheMu.Lock()
	delete(registryCache, tempDir)
	registryCacheMu.Unlock()

	// Create new registry instance - should load from file
	registry2 := GetRegistry(tempDir)

	svc, exists := registry2.GetService("test-service")
	if !exists {
		t.Errorf("GetService() after reload: service not found")
	}

	if svc.Name != "test-service" {
		t.Errorf("GetService() after reload: Name = %v, want test-service", svc.Name)
	}

	if svc.Port != 8080 {
		t.Errorf("GetService() after reload: Port = %v, want 8080", svc.Port)
	}
}

func TestClear(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Register multiple services
	for i := 0; i < 3; i++ {
		entry := &ServiceRegistryEntry{
			Name:      string(rune('a'+i)) + "-service",
			Port:      8080 + i,
			StartTime: time.Now(),
		}
		if err := registry.Register(entry); err != nil {
			t.Fatalf("failed to register service %d: %v", i, err)
		}
	}

	// Verify services were registered
	if len(registry.ListAll()) != 3 {
		t.Fatalf("Expected 3 services before Clear()")
	}

	// Clear registry
	err := registry.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v, want nil", err)
	}

	// Verify all services were removed
	services := registry.ListAll()
	if len(services) != 0 {
		t.Errorf("ListAll() after Clear() length = %v, want 0", len(services))
	}
}

func TestRegistryPersistence(t *testing.T) {
	tempDir := t.TempDir()
	registryFile := filepath.Join(tempDir, ".azure", "services.json")

	registry := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(registryFile); os.IsNotExist(err) {
		t.Errorf("Registry file was not created")
	}

	// Read file and verify contents
	data, err := os.ReadFile(registryFile)
	if err != nil {
		t.Fatalf("Failed to read registry file: %v", err)
	}

	var services map[string]*ServiceRegistryEntry
	if err := json.Unmarshal(data, &services); err != nil {
		t.Fatalf("Failed to unmarshal registry file: %v", err)
	}

	if _, exists := services["test-service"]; !exists {
		t.Errorf("Service not found in persisted registry file")
	}
}

func TestRegisterWithAzureURL(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		URL:       "http://localhost:8080",
		AzureURL:  "https://test-service.azurewebsites.net",
		StartTime: time.Now(),
	}

	err := registry.Register(entry)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Fatal("GetService() service not found")
	}

	if svc.AzureURL != "https://test-service.azurewebsites.net" {
		t.Errorf("GetService() AzureURL = %v, want https://test-service.azurewebsites.net", svc.AzureURL)
	}
}

func TestRegisterWithError(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		Status:    "error",
		Health:    "unhealthy",
		Error:     "failed to start",
		StartTime: time.Now(),
	}

	err := registry.Register(entry)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Fatal("GetService() service not found")
	}

	if svc.Error != "failed to start" {
		t.Errorf("GetService() Error = %v, want 'failed to start'", svc.Error)
	}

	if svc.Status != "error" {
		t.Errorf("GetService() Status = %v, want error", svc.Status)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	done := make(chan bool)

	// Concurrent registers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			entry := &ServiceRegistryEntry{
				Name:      string(rune('a'+idx)) + "-service",
				Port:      8080 + idx,
				StartTime: time.Now(),
			}
			if err := registry.Register(entry); err != nil {
				t.Errorf("failed to register service %d: %v", idx, err)
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all were registered
	services := registry.ListAll()
	if len(services) != 10 {
		t.Errorf("ListAll() length = %v, want 10", len(services))
	}
}

// TestCleanStale tests the cleanStale function
func TestCleanStale(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Add a service with old LastChecked time
	oldEntry := &ServiceRegistryEntry{
		Name:        "old-service",
		Port:        9000,
		StartTime:   time.Now().Add(-2 * time.Hour),
		LastChecked: time.Now(), // Register will set this to now
	}
	if err := registry.Register(oldEntry); err != nil {
		t.Fatalf("failed to register old service: %v", err)
	}

	// Manually update LastChecked to an old time after registration
	registry.mu.Lock()
	if svc, exists := registry.services["old-service"]; exists {
		svc.LastChecked = time.Now().Add(-2 * time.Hour)
	}
	registry.mu.Unlock()

	// Add a recent service
	recentEntry := &ServiceRegistryEntry{
		Name:        "recent-service",
		Port:        9001,
		StartTime:   time.Now(),
		LastChecked: time.Now(),
	}
	if err := registry.Register(recentEntry); err != nil {
		t.Fatalf("failed to register recent service: %v", err)
	}

	// Call cleanStale
	registry.cleanStale()

	// Verify old entry was removed
	_, exists := registry.GetService("old-service")
	if exists {
		t.Error("Expected old service to be removed")
	}

	// Verify recent entry remains
	_, exists = registry.GetService("recent-service")
	if !exists {
		t.Error("Recent service should still exist")
	}
}

// TestIsProcessRunning tests the isProcessRunning function
func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	currentPID := os.Getpid()
	if !isProcessRunning(currentPID) {
		t.Error("Current process should be reported as running")
	}

	// Test with invalid PID (should not be running)
	invalidPID := 999999
	if isProcessRunning(invalidPID) {
		t.Log("PID 999999 is running (unexpected but possible)")
	}
}
