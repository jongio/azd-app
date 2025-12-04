package registry

import (
	"testing"
	"time"
)

func TestGetRegistry(t *testing.T) {
	tempDir := t.TempDir()

	// Clear cache to ensure fresh instance
	registryCacheMu.Lock()
	delete(registryCache, tempDir)
	registryCacheMu.Unlock()

	registry := GetRegistry(tempDir)

	if registry == nil {
		t.Fatal("GetRegistry() returned nil")
	}

	// Registry is now in-memory only, so no .azure directory check needed

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
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Update status
	err := registry.UpdateStatus("test-service", "ready")
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

	// Verify LastChecked was updated
	if svc.LastChecked.IsZero() {
		t.Errorf("UpdateStatus() did not update LastChecked")
	}
}

func TestUpdateStatusNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	err := registry.UpdateStatus("nonexistent-service", "ready")
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
	// Registry is now in-memory only - test that clearing cache gives fresh instance
	tempDir := t.TempDir()

	// Clear cache to ensure fresh instance
	registryCacheMu.Lock()
	delete(registryCache, tempDir)
	registryCacheMu.Unlock()

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
		StartTime:  time.Now(),
	}

	err := registry1.Register(entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify service is in current registry
	svc, exists := registry1.GetService("test-service")
	if !exists {
		t.Errorf("GetService() in same instance: service not found")
	}
	if svc.Port != 8080 {
		t.Errorf("GetService(): Port = %v, want 8080", svc.Port)
	}

	// Clear the cache to simulate process restart
	registryCacheMu.Lock()
	delete(registryCache, tempDir)
	registryCacheMu.Unlock()

	// Create new registry instance - should start empty (in-memory only)
	registry2 := GetRegistry(tempDir)

	_, exists = registry2.GetService("test-service")
	if exists {
		t.Errorf("GetService() after cache clear: expected service to NOT exist (in-memory registry)")
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
	// Registry is now in-memory only - this test verifies the in-memory behavior
	tempDir := t.TempDir()

	// Clear cache
	registryCacheMu.Lock()
	delete(registryCache, tempDir)
	registryCacheMu.Unlock()

	registry := GetRegistry(tempDir)

	entry := &ServiceRegistryEntry{
		Name:      "test-service",
		Port:      8080,
		StartTime: time.Now(),
	}

	if err := registry.Register(entry); err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	// Verify service is in registry
	svc, exists := registry.GetService("test-service")
	if !exists {
		t.Errorf("Service not found in registry after Register()")
	}
	if svc.Port != 8080 {
		t.Errorf("Port = %v, want 8080", svc.Port)
	}

	// Note: No file persistence - state lives only in memory and azd config
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
