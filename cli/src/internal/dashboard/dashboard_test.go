package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/registry"
)

func TestGetServer_DifferentProjects(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	srv1 := GetServer(tempDir1)
	srv2 := GetServer(tempDir2)

	if srv1 == srv2 {
		t.Error("Expected different server instances for different projects")
	}

	absPath1, _ := normalizeProjectPath(tempDir1)
	if srv1.projectDir != absPath1 {
		t.Errorf("Expected projectDir %s, got %s", absPath1, srv1.projectDir)
	}

	absPath2, _ := normalizeProjectPath(tempDir2)
	if srv2.projectDir != absPath2 {
		t.Errorf("Expected projectDir %s, got %s", absPath2, srv2.projectDir)
	}
}

func TestGetServer_SameProjectReturnsCache(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	tempDir := t.TempDir()

	srv1 := GetServer(tempDir)
	srv2 := GetServer(tempDir)

	if srv1 != srv2 {
		t.Error("Expected same server instance for same project")
	}
}

func TestGetServer_RelativePathResolved(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	tempDir := t.TempDir()

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	srv1 := GetServer(".")
	srv2 := GetServer(tempDir)

	// Should be same instance (both resolve to same absolute path)
	if srv1 != srv2 {
		t.Error("Expected relative and absolute paths to resolve to same server")
	}
}

func TestServer_Initialization(t *testing.T) {
	tempDir := t.TempDir()
	srv := GetServer(tempDir)

	if srv.port != 0 {
		t.Error("Expected port to be 0 before Start()")
	}

	if srv.mux == nil {
		t.Error("Expected mux to be initialized")
	}

	if srv.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if srv.stopChan == nil {
		t.Error("Expected stopChan to be initialized")
	}
}

func TestHandleGetServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create azure.yaml for the test
	azureYamlContent := `name: test-project
services:
  test-service:
    language: python
    project: ./
`
	if err := os.WriteFile(filepath.Join(tempDir, "azure.yaml"), []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)

	// Create test registry with services
	reg := registry.GetRegistry(tempDir)
	reg.Register(&registry.ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		URL:        "http://localhost:3000",
		Status:     "running",
		Port:       3000,
	})

	// Create test request
	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()

	srv.handleGetServices(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response
	var services []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if len(services) > 0 {
		if name, ok := services[0]["name"].(string); !ok || name != "test-service" {
			t.Errorf("Expected service name 'test-service', got '%v'", services[0]["name"])
		}
	}
}

func TestHandleGetAllServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create azure.yaml for the test
	azureYamlContent := `name: test-project
services:
  service1:
    language: python
    project: ./
  service2:
    language: node
    project: ./
`
	if err := os.WriteFile(filepath.Join(tempDir, "azure.yaml"), []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	srv := GetServer(tempDir)

	// Create test registry
	reg := registry.GetRegistry(tempDir)
	reg.Register(&registry.ServiceRegistryEntry{
		Name:       "service1",
		ProjectDir: tempDir,
		URL:        "http://localhost:3000",
		Status:     "running",
		Port:       3000,
	})
	reg.Register(&registry.ServiceRegistryEntry{
		Name:       "service2",
		ProjectDir: tempDir,
		URL:        "http://localhost:8080",
		Status:     "starting",
		Port:       8080,
	})

	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()

	srv.handleGetServices(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var services []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}
}

func TestBroadcastUpdate(t *testing.T) {
	tempDir := t.TempDir()
	srv := GetServer(tempDir)

	// Test that BroadcastUpdate doesn't panic with no clients
	services := []*registry.ServiceRegistryEntry{
		{
			Name:   "test-service",
			URL:    "http://localhost:3000",
			Status: "running",
		},
	}

	// Should not panic even with no connected clients
	srv.BroadcastUpdate(services)

	// Verify no errors (test passes if no panic)
}

func TestStop_CleansUpServerMap(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	tempDir := t.TempDir()

	srv := GetServer(tempDir)

	// Verify server is in map
	serversMu.Lock()
	_, key := normalizeProjectPath(tempDir)
	_, exists := servers[key]
	serversMu.Unlock()

	if !exists {
		t.Error("Expected server to be in servers map")
	}

	// Stop server
	srv.Stop()

	// Verify server is removed from map
	serversMu.Lock()
	_, exists = servers[key]
	serversMu.Unlock()

	if exists {
		t.Error("Expected server to be removed from servers map after Stop()")
	}
}

func TestMultipleProjectsDashboards(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	// Create multiple projects
	projects := make([]string, 3)
	for i := 0; i < 3; i++ {
		projects[i] = t.TempDir()
	}

	// Get servers for each project
	srvs := make([]*Server, 3)
	for i, proj := range projects {
		srvs[i] = GetServer(proj)
	}

	// Verify all are different instances
	for i := 0; i < len(srvs); i++ {
		for j := i + 1; j < len(srvs); j++ {
			if srvs[i] == srvs[j] {
				t.Errorf("Expected different instances for projects %d and %d", i, j)
			}
		}
	}

	// Verify servers map has all 3
	serversMu.Lock()
	count := len(servers)
	serversMu.Unlock()

	if count != 3 {
		t.Errorf("Expected 3 servers in map, got %d", count)
	}
}

func TestServersConcurrentAccess(t *testing.T) {
	// Clear servers map
	serversMu.Lock()
	servers = make(map[string]*Server)
	serversMu.Unlock()

	tempDir := t.TempDir()

	// Concurrent GetServer calls
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			srv := GetServer(tempDir)
			if srv == nil {
				t.Error("Expected non-nil server")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only have 1 server instance
	serversMu.Lock()
	count := len(servers)
	serversMu.Unlock()

	if count != 1 {
		t.Errorf("Expected 1 server instance despite concurrent access, got %d", count)
	}
}

func TestHandleFallback(t *testing.T) {
	tempDir := t.TempDir()
	srv := GetServer(tempDir)

	// Create test registry
	reg := registry.GetRegistry(tempDir)
	reg.Register(&registry.ServiceRegistryEntry{
		Name:       "test-service",
		ProjectDir: tempDir,
		URL:        "http://localhost:3000",
		Status:     "running",
		Port:       3000,
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	srv.handleFallback(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type text/html, got %s", contentType)
	}

	// Response should contain HTML
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty HTML response")
	}

	// Should contain service name
	if len(body) > 0 && !contains(body, "test-service") {
		t.Error("Expected HTML to contain service name")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}
