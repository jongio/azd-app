package authclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				ServerURL: "http://localhost:8080",
				Secret:    "test-secret",
				Timeout:   30 * time.Second,
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "empty server URL",
			config: &Config{
				ServerURL: "",
				Secret:    "test-secret",
			},
			expectError: true,
		},
		{
			name: "empty secret",
			config: &Config{
				ServerURL: "http://localhost:8080",
				Secret:    "",
			},
			expectError: true,
		},
		{
			name: "invalid server URL",
			config: &Config{
				ServerURL: "://invalid-url",
				Secret:    "test-secret",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClient_GetToken(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/token" {
			t.Errorf("expected path /token, got %s", r.URL.Path)
		}

		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-secret" {
			t.Errorf("expected Authorization header 'Bearer test-secret', got '%s'", authHeader)
		}

		// Return token response
		response := TokenResponse{
			AccessToken: "test-jwt-token",
			TokenType:   "Bearer",
			ExpiresIn:   900,
			Scope:       r.URL.Query().Get("scope"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Get token
	token, err := client.GetToken("https://management.azure.com/.default")
	if err != nil {
		t.Errorf("failed to get token: %v", err)
	}

	if token != "test-jwt-token" {
		t.Errorf("expected token 'test-jwt-token', got '%s'", token)
	}
}

func TestClient_GetToken_DefaultScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		if scope != "https://management.azure.com/.default" {
			t.Errorf("expected default scope, got %s", scope)
		}

		response := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   900,
			Scope:       scope,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Get token without specifying scope
	_, err = client.GetToken("")
	if err != nil {
		t.Errorf("failed to get token: %v", err)
	}
}

func TestClient_GetToken_Caching(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		
		response := TokenResponse{
			AccessToken: "cached-token",
			TokenType:   "Bearer",
			ExpiresIn:   900,
			Scope:       r.URL.Query().Get("scope"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// First request
	token1, err := client.GetToken("https://management.azure.com/.default")
	if err != nil {
		t.Errorf("failed to get token: %v", err)
	}

	// Second request (should be cached)
	token2, err := client.GetToken("https://management.azure.com/.default")
	if err != nil {
		t.Errorf("failed to get token: %v", err)
	}

	if token1 != token2 {
		t.Error("expected same token from cache")
	}

	if requestCount != 1 {
		t.Errorf("expected 1 request to server, got %d", requestCount)
	}
}

func TestClient_GetToken_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	config.MaxRetries = 0 // Disable retries for faster test
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetToken("https://management.azure.com/.default")
	if err == nil {
		t.Error("expected error when server returns 500")
	}
}

func TestClient_GetToken_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	config.MaxRetries = 0
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.GetToken("https://management.azure.com/.default")
	if err == nil {
		t.Error("expected error when server returns 401")
	}
}

func TestClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.HealthCheck()
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
}

func TestClient_HealthCheck_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.HealthCheck()
	if err == nil {
		t.Error("expected error for unhealthy server")
	}
}

func TestClient_ClearCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   900,
			Scope:       r.URL.Query().Get("scope"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := DefaultConfig(server.URL, "test-secret")
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Get token to populate cache
	_, err = client.GetToken("https://management.azure.com/.default")
	if err != nil {
		t.Errorf("failed to get token: %v", err)
	}

	// Clear cache
	client.ClearCache()

	// Verify cache is empty
	client.mu.RLock()
	cacheSize := len(client.tokenCache)
	client.mu.RUnlock()

	if cacheSize != 0 {
		t.Errorf("expected cache to be empty, got %d entries", cacheSize)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("http://localhost:8080", "test-secret")

	if config.ServerURL != "http://localhost:8080" {
		t.Errorf("expected server URL http://localhost:8080, got %s", config.ServerURL)
	}

	if config.Secret != "test-secret" {
		t.Errorf("expected secret test-secret, got %s", config.Secret)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}

	if config.RetryBackoff != 1*time.Second {
		t.Errorf("expected retry backoff 1s, got %v", config.RetryBackoff)
	}
}
