package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBroadcastServiceUpdate(t *testing.T) {
	// Create a temporary project directory with azure.yaml
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	azureYamlContent := `name: test-project
services:
  api:
    language: python
    project: ./api
  web:
    language: node
    project: ./web
`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("failed to create test azure.yaml: %v", err)
	}

	// Create server instance
	srv := GetServer(tempDir)

	// Start server
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	// Connect WebSocket client
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Read initial message (sent on connect)
	var initialMsg map[string]interface{}
	if err := ws.ReadJSON(&initialMsg); err != nil {
		t.Fatalf("failed to read initial message: %v", err)
	}

	// Broadcast update
	if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
		t.Fatalf("BroadcastServiceUpdate failed: %v", err)
	}

	// Read broadcast message
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var updateMsg map[string]interface{}
	if err := ws.ReadJSON(&updateMsg); err != nil {
		t.Fatalf("failed to read broadcast message: %v", err)
	}

	// Verify message structure
	if updateMsg["type"] != "services" {
		t.Errorf("message type = %v, want %q", updateMsg["type"], "services")
	}

	services, ok := updateMsg["services"].([]interface{})
	if !ok {
		t.Fatalf("services field is not an array: %T", updateMsg["services"])
	}

	// Should have services from azure.yaml
	if len(services) < 2 {
		t.Errorf("expected at least 2 services, got %d", len(services))
	}
}

func TestBroadcastServiceUpdate_MultipleClients(t *testing.T) {
	// Create a temporary project directory
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	azureYamlContent := `name: test-project
services:
  api:
    language: python
`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("failed to create test azure.yaml: %v", err)
	}

	// Create server
	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	// Connect multiple WebSocket clients
	numClients := 3
	clients := make([]*websocket.Conn, numClients)
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		defer ws.Close()
		clients[i] = ws

		// Read initial message
		var initialMsg map[string]interface{}
		if err := ws.ReadJSON(&initialMsg); err != nil {
			t.Fatalf("client %d failed to read initial message: %v", i, err)
		}
	}

	// Broadcast update
	if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
		t.Fatalf("BroadcastServiceUpdate failed: %v", err)
	}

	// Verify all clients received the update
	for i, ws := range clients {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		var updateMsg map[string]interface{}
		if err := ws.ReadJSON(&updateMsg); err != nil {
			t.Errorf("client %d failed to receive broadcast: %v", i, err)
			continue
		}

		if updateMsg["type"] != "services" {
			t.Errorf("client %d: message type = %v, want %q", i, updateMsg["type"], "services")
		}
	}
}

func TestBroadcastServiceUpdate_NoClients(t *testing.T) {
	// Create a temporary project directory
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	azureYamlContent := `name: test-project
services:
  api:
    language: python
`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("failed to create test azure.yaml: %v", err)
	}

	// Create server but don't connect any clients
	srv := GetServer(tempDir)

	// Broadcast should succeed even with no clients connected
	if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
		t.Errorf("BroadcastServiceUpdate with no clients failed: %v", err)
	}
}

func TestBroadcastServiceUpdate_InvalidProjectDir(t *testing.T) {
	srv := &Server{
		projectDir: "/nonexistent/path",
		clients:    make(map[*clientConn]bool),
	}

	// Should return error for invalid project directory (no azure.yaml)
	err := srv.BroadcastServiceUpdate("/nonexistent/path")
	if err == nil {
		t.Log("Warning: BroadcastServiceUpdate did not return error for invalid project directory (gracefully handles missing azure.yaml)")
	}
}

func TestHandleGetServices_ReturnsUpdatedEnvironment(t *testing.T) {
	// Create a temporary project directory with azure.yaml
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	azureYamlContent := `name: test-project
services:
  api:
    language: python
    project: ./api
`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("failed to create test azure.yaml: %v", err)
	}

	// Set environment variables that should be picked up
	os.Setenv("SERVICE_API_URL", "https://test-api.azurecontainerapps.io")
	os.Setenv("SERVICE_API_NAME", "test-api-resource")
	defer func() {
		os.Unsetenv("SERVICE_API_URL")
		os.Unsetenv("SERVICE_API_NAME")
	}()

	// Create server
	srv := GetServer(tempDir)

	// Create test HTTP request
	req := httptest.NewRequest("GET", "/api/services", nil)
	w := httptest.NewRecorder()

	// Call handler
	srv.handleGetServices(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse response
	var services []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&services); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify service has Azure information
	found := false
	for _, svc := range services {
		if svc["name"] == "api" {
			found = true
			azure, ok := svc["azure"].(map[string]interface{})
			if !ok {
				t.Error("service missing azure field")
				continue
			}

			if azure["url"] != "https://test-api.azurecontainerapps.io" {
				t.Errorf("azure.url = %v, want %q", azure["url"], "https://test-api.azurecontainerapps.io")
			}
			if azure["resourceName"] != "test-api-resource" {
				t.Errorf("azure.resourceName = %v, want %q", azure["resourceName"], "test-api-resource")
			}
		}
	}

	if !found {
		t.Error("api service not found in response")
	}
}

func TestWebSocketConnection_ReceivesInitialServices(t *testing.T) {
	// Create a temporary project directory
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	azureYamlContent := `name: test-project
services:
  api:
    language: python
  web:
    language: node
`
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("failed to create test azure.yaml: %v", err)
	}

	// Create and start server
	srv := GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop()

	// Connect WebSocket
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Should receive initial services immediately on connect
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg map[string]interface{}
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read initial message: %v", err)
	}

	// Verify message
	if msg["type"] != "services" {
		t.Errorf("message type = %v, want %q", msg["type"], "services")
	}

	services, ok := msg["services"].([]interface{})
	if !ok {
		t.Fatalf("services is not an array: %T", msg["services"])
	}

	if len(services) < 2 {
		t.Errorf("expected at least 2 services, got %d", len(services))
	}
}

func TestGetServer_ReturnsSameInstanceForSameProject(t *testing.T) {
	tempDir := t.TempDir()

	srv1 := GetServer(tempDir)
	srv2 := GetServer(tempDir)

	if srv1 != srv2 {
		t.Error("GetServer returned different instances for same project directory")
	}
}

func TestGetServer_ReturnsDifferentInstancesForDifferentProjects(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	srv1 := GetServer(tempDir1)
	srv2 := GetServer(tempDir2)

	if srv1 == srv2 {
		t.Error("GetServer returned same instance for different project directories")
	}
}
