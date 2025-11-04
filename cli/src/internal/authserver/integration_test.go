package authserver

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestAuthServerIntegration tests the full server lifecycle.
func TestAuthServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create server config
	config := DefaultConfig()
	config.Port = 18080 // Use a non-standard port to avoid conflicts
	config.SharedSecret = "test-secret-integration"
	config.BindAddress = "127.0.0.1" // Bind to localhost only

	// Create server
	server, err := NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Verify server is running
	if !server.IsRunning() {
		t.Fatal("Server should be running")
	}

	// Test health endpoint (no auth required)
	healthURL := "http://127.0.0.1:18080/health"
	resp, err := http.Get(healthURL)
	if err != nil {
		t.Errorf("Failed to call health endpoint: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 from health endpoint, got %d", resp.StatusCode)
		}
	}

	// Test unauthorized access to token endpoint
	tokenURL := "http://127.0.0.1:18080/token"
	resp, err = http.Get(tokenURL)
	if err != nil {
		t.Errorf("Failed to call token endpoint: %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 from token endpoint without auth, got %d", resp.StatusCode)
		}
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Verify server is stopped
	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

// TestAuthServerWithMockCredentials tests server behavior when Azure credentials are unavailable.
// This is expected in test environments without Azure authentication.
func TestAuthServerWithoutAzureCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create server config
	config := DefaultConfig()
	config.Port = 18081
	config.SharedSecret = "test-secret"
	config.BindAddress = "127.0.0.1"

	// Attempt to create server - this may fail if Azure credentials are not available
	// which is expected in CI/CD environments
	server, err := NewServer(config)
	
	// If server creation fails due to credentials, that's expected and ok
	if err != nil {
		if server != nil {
			t.Error("Server should be nil when creation fails")
		}
		// This is expected - server requires Azure credentials
		t.Logf("Server creation failed as expected (no Azure credentials): %v", err)
		return
	}

	// If we got a server, clean it up
	if server != nil {
		server.Start()
		time.Sleep(100 * time.Millisecond)
		server.Stop()
	}
}

// TestRateLimiterIntegration tests rate limiting with actual requests.
func TestRateLimiterIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.Port = 18082
	config.SharedSecret = "test-secret-rate-limit"
	config.BindAddress = "127.0.0.1"
	config.RateLimitRequests = 3 // Low limit for testing

	server, err := NewServer(config)
	if err != nil {
		t.Skipf("Skipping rate limiter test (Azure credentials not available): %v", err)
		return
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(200 * time.Millisecond)

	// Make requests up to the limit
	client := &http.Client{Timeout: 5 * time.Second}
	tokenURL := "http://127.0.0.1:18082/token?scope=https://management.azure.com/.default"

	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", tokenURL, nil)
		req.Header.Set("Authorization", "Bearer test-secret-rate-limit")

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		} else if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusInternalServerError {
			// OK or error from Azure (expected without real credentials)
			successCount++
		}
	}

	// We should see some rate limiting
	if rateLimitedCount == 0 {
		t.Error("Expected to see rate limiting, but got none")
	}

	t.Logf("Successful requests: %d, Rate limited: %d", successCount, rateLimitedCount)
}

// TestServerURLGeneration tests GetURL method.
func TestServerURLGeneration(t *testing.T) {
	tests := []struct {
		name        string
		port        int
		enableTLS   bool
		bindAddress string
		expectedURL string
	}{
		{
			name:        "HTTP localhost",
			port:        8080,
			enableTLS:   false,
			bindAddress: "127.0.0.1",
			expectedURL: "http://127.0.0.1:8080",
		},
		{
			name:        "HTTPS any interface",
			port:        8443,
			enableTLS:   true,
			bindAddress: "0.0.0.0",
			expectedURL: "https://0.0.0.0:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Port:        tt.port,
				EnableTLS:   tt.enableTLS,
				BindAddress: tt.bindAddress,
				SharedSecret: "test",
				TokenExpiry: 15 * time.Minute,
				RateLimitRequests: 10,
			}

			// Skip creating actual server if TLS (would need certs)
			if tt.enableTLS {
				config.CertFile = "/tmp/cert.pem"
				config.KeyFile = "/tmp/key.pem"
			}

			server := &Server{config: config}
			url := server.GetURL()

			if url != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, url)
			}
		})
	}
}

// TestJSONOutput tests JSON response format.
func TestJSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := DefaultConfig()
	config.Port = 18083
	config.SharedSecret = "test-secret-json"
	config.BindAddress = "127.0.0.1"

	server, err := NewServer(config)
	if err != nil {
		t.Skipf("Skipping JSON test (Azure credentials not available): %v", err)
		return
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(200 * time.Millisecond)

	// Test health endpoint JSON
	healthURL := "http://127.0.0.1:18083/health"
	resp, err := http.Get(healthURL)
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}
	defer resp.Body.Close()

	var healthResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
	}

	if status, ok := healthResponse["status"].(string); !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", healthResponse["status"])
	}
}
