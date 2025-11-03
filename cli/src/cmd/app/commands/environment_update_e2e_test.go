//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jongio/azd-app/cli/src/internal/dashboard"
	"github.com/jongio/azd-app/cli/src/internal/serviceinfo"
)

// TestEnvironmentUpdateE2E tests the end-to-end flow:
// 1. Start dashboard with WebSocket client connected
// 2. Simulate azd provision completing (environment update event)
// 3. Verify dashboard receives updated service info via WebSocket
func TestEnvironmentUpdateE2E(t *testing.T) {
	// Skip in short mode as this is an integration test
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

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

	// Step 1: Start dashboard server
	srv := dashboard.GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start dashboard: %v", err)
	}
	defer srv.Stop()

	// Step 2: Connect WebSocket client
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer ws.Close()

	// Read initial message
	var initialMsg map[string]interface{}
	if err := ws.ReadJSON(&initialMsg); err != nil {
		t.Fatalf("failed to read initial message: %v", err)
	}

	// Verify initial state - no Azure info yet
	services := initialMsg["services"].([]interface{})
	for _, svc := range services {
		svcMap := svc.(map[string]interface{})
		if svcMap["name"] == "api" {
			if azure, ok := svcMap["azure"]; ok && azure != nil {
				azureMap := azure.(map[string]interface{})
				if azureMap["url"] != nil && azureMap["url"] != "" {
					t.Error("Azure URL should be empty before provision")
				}
			}
		}
	}

	// Step 3: Simulate azd provision completing - set environment variables
	os.Setenv("SERVICE_API_URL", "https://test-api.azurecontainerapps.io")
	os.Setenv("SERVICE_API_NAME", "test-api-ca")
	os.Setenv("SERVICE_WEB_URL", "https://test-web.azurecontainerapps.io")
	os.Setenv("SERVICE_WEB_NAME", "test-web-ca")
	defer func() {
		os.Unsetenv("SERVICE_API_URL")
		os.Unsetenv("SERVICE_API_NAME")
		os.Unsetenv("SERVICE_WEB_URL")
		os.Unsetenv("SERVICE_WEB_NAME")
	}()

	// Step 4: Simulate environment update event handler being called
	// Refresh the cache and broadcast directly since we can't set the path on the mock ProjectConfig
	serviceinfo.RefreshEnvironmentCache()

	// Call BroadcastServiceUpdate directly with the correct project directory
	if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
		t.Fatalf("BroadcastServiceUpdate failed: %v", err)
	}

	// Step 5: Wait for and verify WebSocket broadcast
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	var updateMsg map[string]interface{}
	if err := ws.ReadJSON(&updateMsg); err != nil {
		t.Fatalf("failed to read broadcast message: %v", err)
	}

	// Verify message structure
	if updateMsg["type"] != "services" {
		t.Errorf("message type = %v, want %q", updateMsg["type"], "services")
	}

	updatedServices := updateMsg["services"].([]interface{})

	// Verify services have Azure information
	foundAPI := false
	foundWeb := false

	for _, svc := range updatedServices {
		svcMap := svc.(map[string]interface{})
		name := svcMap["name"].(string)

		if name == "api" {
			foundAPI = true
			azure, ok := svcMap["azure"].(map[string]interface{})
			if !ok {
				t.Error("api service missing azure field")
				continue
			}

			if azure["url"] != "https://test-api.azurecontainerapps.io" {
				t.Errorf("api azure.url = %v, want %q", azure["url"], "https://test-api.azurecontainerapps.io")
			}
			if azure["resourceName"] != "test-api-ca" {
				t.Errorf("api azure.resourceName = %v, want %q", azure["resourceName"], "test-api-ca")
			}
		}

		if name == "web" {
			foundWeb = true
			azure, ok := svcMap["azure"].(map[string]interface{})
			if !ok {
				t.Error("web service missing azure field")
				continue
			}

			if azure["url"] != "https://test-web.azurecontainerapps.io" {
				t.Errorf("web azure.url = %v, want %q", azure["url"], "https://test-web.azurecontainerapps.io")
			}
			if azure["resourceName"] != "test-web-ca" {
				t.Errorf("web azure.resourceName = %v, want %q", azure["resourceName"], "test-web-ca")
			}
		}
	}

	if !foundAPI {
		t.Error("api service not found in updated services")
	}
	if !foundWeb {
		t.Error("web service not found in updated services")
	}
}

// TestEnvironmentUpdateE2E_MultipleClients verifies that all connected clients
// receive the environment update broadcast
func TestEnvironmentUpdateE2E_MultipleClients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup project
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

	// Start dashboard
	srv := dashboard.GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start dashboard: %v", err)
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

	// Set environment variables
	os.Setenv("SERVICE_API_URL", "https://updated-api.azurecontainerapps.io")
	defer os.Unsetenv("SERVICE_API_URL")

	// Refresh environment cache
	serviceinfo.RefreshEnvironmentCache()

	// Trigger broadcast
	if err := srv.BroadcastServiceUpdate(tempDir); err != nil {
		t.Fatalf("BroadcastServiceUpdate failed: %v", err)
	}

	// Verify all clients received the update
	for i, ws := range clients {
		ws.SetReadDeadline(time.Now().Add(3 * time.Second))
		var updateMsg map[string]interface{}
		if err := ws.ReadJSON(&updateMsg); err != nil {
			t.Errorf("client %d failed to receive update: %v", i, err)
			continue
		}

		if updateMsg["type"] != "services" {
			t.Errorf("client %d: message type = %v, want %q", i, updateMsg["type"], "services")
		}

		// Verify Azure URL is present
		services := updateMsg["services"].([]interface{})
		found := false
		for _, svc := range services {
			svcMap := svc.(map[string]interface{})
			if svcMap["name"] == "api" {
				found = true
				if azure, ok := svcMap["azure"].(map[string]interface{}); ok {
					if azure["url"] != "https://updated-api.azurecontainerapps.io" {
						t.Errorf("client %d: api azure.url = %v, want %q",
							i, azure["url"], "https://updated-api.azurecontainerapps.io")
					}
				}
			}
		}

		if !found {
			t.Errorf("client %d: api service not found in update", i)
		}
	}
}

// TestEnvironmentCache_ThreadSafety tests that concurrent environment updates
// and cache reads don't cause race conditions
func TestEnvironmentCache_ThreadSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test environment
	os.Setenv("TEST_VAR_1", "value1")
	os.Setenv("TEST_VAR_2", "value2")
	defer func() {
		os.Unsetenv("TEST_VAR_1")
		os.Unsetenv("TEST_VAR_2")
	}()

	// Start dashboard
	tempDir := t.TempDir()
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte("name: test\nservices:\n  api:\n    language: python\n"), 0600); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	srv := dashboard.GetServer(tempDir)
	url, err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start dashboard: %v", err)
	}
	defer srv.Stop()

	// Connect clients
	numClients := 5
	clients := make([]*websocket.Conn, numClients)
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/api/ws"

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		defer ws.Close()
		clients[i] = ws

		// Drain initial message
		var msg map[string]interface{}
		ws.ReadJSON(&msg)
	}

	// Perform concurrent environment updates and broadcasts
	done := make(chan bool)
	numIterations := 20

	// Goroutine 1: Update environment and refresh cache
	go func() {
		for i := 0; i < numIterations; i++ {
			os.Setenv("CONCURRENT_VAR", "iteration_"+string(rune(i)))
			serviceinfo.RefreshEnvironmentCache()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Trigger broadcasts
	go func() {
		for i := 0; i < numIterations; i++ {
			srv.BroadcastServiceUpdate(tempDir)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify all clients can still receive messages (no deadlocks)
	os.Setenv("FINAL_VAR", "final_value")
	serviceinfo.RefreshEnvironmentCache()
	srv.BroadcastServiceUpdate(tempDir)

	for i, ws := range clients {
		ws.SetReadDeadline(time.Now().Add(2 * time.Second))
		var msg map[string]interface{}
		// Read all pending messages
		for {
			if err := ws.ReadJSON(&msg); err != nil {
				break
			}
		}
		// Should be able to read without timeout or panic
		if msg["type"] != "services" {
			t.Errorf("client %d: final message type = %v, want %q", i, msg["type"], "services")
		}
	}
}
