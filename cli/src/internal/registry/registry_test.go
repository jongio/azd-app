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

// TestObserverPattern tests the observer pattern for registry changes.
func TestObserverPattern(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Create a test observer
	notificationChan := make(chan *ServiceRegistryEntry, 10)
	observer := &testObserver{notifications: notificationChan}

	// Subscribe observer
	registry.Subscribe(observer)
	defer registry.Unsubscribe(observer)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		PID:        12345,
		Port:       8080,
		Status:     "starting",
		Health:     "unknown",
	}
	if err := registry.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Update status (should trigger notification)
	if err := registry.UpdateStatus("test-service", "running", "healthy"); err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Wait for notification
	select {
	case notification := <-notificationChan:
		if notification.Name != "test-service" {
			t.Errorf("Notification service name = %v, want test-service", notification.Name)
		}
		if notification.Status != "running" {
			t.Errorf("Notification status = %v, want running", notification.Status)
		}
		if notification.Health != "healthy" {
			t.Errorf("Notification health = %v, want healthy", notification.Health)
		}
	case <-time.After(1 * time.Second):
		t.Error("Observer was not notified within timeout")
	}
}

// TestObserverNoNotificationWhenNoChange tests that observers are not notified when status doesn't change.
func TestObserverNoNotificationWhenNoChange(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Create a test observer
	notificationChan := make(chan *ServiceRegistryEntry, 10)
	observer := &testObserver{notifications: notificationChan}

	// Subscribe observer
	registry.Subscribe(observer)
	defer registry.Unsubscribe(observer)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		Status:     "running",
		Health:     "healthy",
	}
	if err := registry.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Update with same status (should NOT trigger notification)
	if err := registry.UpdateStatus("test-service", "running", "healthy"); err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Verify no notification was sent
	select {
	case <-notificationChan:
		t.Error("Observer was notified when status didn't change")
	case <-time.After(100 * time.Millisecond):
		// Expected - no notification
	}
}

// TestMultipleObservers tests that multiple observers receive notifications.
func TestMultipleObservers(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Create multiple test observers
	chan1 := make(chan *ServiceRegistryEntry, 10)
	chan2 := make(chan *ServiceRegistryEntry, 10)
	observer1 := &testObserver{notifications: chan1}
	observer2 := &testObserver{notifications: chan2}

	// Subscribe both observers
	registry.Subscribe(observer1)
	defer registry.Unsubscribe(observer1)
	registry.Subscribe(observer2)
	defer registry.Unsubscribe(observer2)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:   "test-service",
		Status: "starting",
		Health: "unknown",
	}
	if err := registry.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Update status
	if err := registry.UpdateStatus("test-service", "running", "healthy"); err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Both observers should receive notification
	timeout := time.After(1 * time.Second)
	receivedCount := 0

	for receivedCount < 2 {
		select {
		case <-chan1:
			receivedCount++
		case <-chan2:
			receivedCount++
		case <-timeout:
			t.Errorf("Only %d/2 observers were notified", receivedCount)
			return
		}
	}
}

// TestUnsubscribe tests that observers can be removed from the registry.
func TestUnsubscribe(t *testing.T) {
	tempDir := t.TempDir()
	registry := GetRegistry(tempDir)

	// Create test observers
	chan1 := make(chan *ServiceRegistryEntry, 10)
	chan2 := make(chan *ServiceRegistryEntry, 10)
	observer1 := &testObserver{notifications: chan1}
	observer2 := &testObserver{notifications: chan2}

	// Subscribe both observers
	registry.Subscribe(observer1)
	registry.Subscribe(observer2)

	// Register a service
	entry := &ServiceRegistryEntry{
		Name:   "test-service",
		Status: "starting",
		Health: "unknown",
	}
	if err := registry.Register(entry); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Update status (both should receive notification)
	if err := registry.UpdateStatus("test-service", "running", "healthy"); err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Verify both received notification
	timeout := time.After(1 * time.Second)
	receivedCount := 0
	for receivedCount < 2 {
		select {
		case <-chan1:
			receivedCount++
		case <-chan2:
			receivedCount++
		case <-timeout:
			t.Fatalf("Expected 2 notifications, got %d", receivedCount)
		}
	}

	// Unsubscribe observer1
	registry.Unsubscribe(observer1)

	// Clear channels
	select {
	case <-chan1:
	default:
	}
	select {
	case <-chan2:
	default:
	}

	// Update status again (only observer2 should receive notification)
	if err := registry.UpdateStatus("test-service", "running", "unhealthy"); err != nil {
		t.Fatalf("UpdateStatus() failed: %v", err)
	}

	// Wait and check notifications
	timeout = time.After(1 * time.Second)
	receivedCount = 0
	for receivedCount < 1 {
		select {
		case <-chan1:
			t.Error("Observer1 received notification after unsubscribe")
			return
		case <-chan2:
			receivedCount++
		case <-timeout:
			t.Fatal("Observer2 did not receive notification")
		}
	}

	// Ensure observer1 didn't receive anything
	select {
	case <-chan1:
		t.Error("Observer1 received notification after unsubscribe")
	case <-time.After(200 * time.Millisecond):
		// Expected - observer1 should not receive notification
	}
	
	// Clean up observer2
	registry.Unsubscribe(observer2)
}

// testObserver is a test implementation of RegistryObserver.
type testObserver struct {
	notifications chan *ServiceRegistryEntry
}

func (o *testObserver) OnServiceChanged(entry *ServiceRegistryEntry) {
	o.notifications <- entry
}

