package service

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDockerComposeCompatibility validates that azure.yaml can parse
// environment variables in Docker Compose format and that services
// correctly receive the merged environment.
func TestDockerComposeCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a temporary project with azure.yaml
	tmpDir := t.TempDir()

	// Create azure.yaml with Docker Compose-style environment variables
	azureYamlContent := `name: compose-compat-test

services:
  # Map format - standard Docker Compose
  web:
    language: node
    project: ./web
    host: localhost
    ports:
      - "3000"
    environment:
      NODE_ENV: production
      PORT: "3000"
      API_URL: http://localhost:5000
      DEBUG: "false"
  
  # Array of strings - Docker Compose shorthand
  api:
    language: python
    project: ./api
    host: localhost
    ports:
      - "5000"
    environment:
      - FLASK_ENV=production
      - FLASK_APP=app.py
      - DATABASE_URL=postgresql://localhost:5432/db
      - REDIS_URL=redis://localhost:6379
  
  # Array of objects - legacy format with secret support
  worker:
    language: python
    project: ./worker
    host: localhost
    environment:
      - name: WORKER_THREADS
        value: "4"
      - name: QUEUE_URL
        value: redis://localhost:6379
      - name: API_KEY
        secret: super-secret-key
`

	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	// Parse azure.yaml
	azureYaml, err := ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Verify web service (map format)
	web, exists := azureYaml.Services["web"]
	if !exists {
		t.Fatal("web service not found")
	}

	webEnv := web.GetEnvironment()
	expectedWeb := map[string]string{
		"NODE_ENV": "production",
		"PORT":     "3000",
		"API_URL":  "http://localhost:5000",
		"DEBUG":    "false",
	}

	for key, expectedValue := range expectedWeb {
		if gotValue := webEnv[key]; gotValue != expectedValue {
			t.Errorf("web.%s = %q, want %q", key, gotValue, expectedValue)
		}
	}

	// Verify api service (array of strings format)
	api, exists := azureYaml.Services["api"]
	if !exists {
		t.Fatal("api service not found")
	}

	apiEnv := api.GetEnvironment()
	expectedAPI := map[string]string{
		"FLASK_ENV":    "production",
		"FLASK_APP":    "app.py",
		"DATABASE_URL": "postgresql://localhost:5432/db",
		"REDIS_URL":    "redis://localhost:6379",
	}

	for key, expectedValue := range expectedAPI {
		if gotValue := apiEnv[key]; gotValue != expectedValue {
			t.Errorf("api.%s = %q, want %q", key, gotValue, expectedValue)
		}
	}

	// Verify worker service (array of objects with secret)
	worker, exists := azureYaml.Services["worker"]
	if !exists {
		t.Fatal("worker service not found")
	}

	workerEnv := worker.GetEnvironment()
	expectedWorker := map[string]string{
		"WORKER_THREADS": "4",
		"QUEUE_URL":      "redis://localhost:6379",
		"API_KEY":        "super-secret-key", // Secret should be used
	}

	for key, expectedValue := range expectedWorker {
		if gotValue := workerEnv[key]; gotValue != expectedValue {
			t.Errorf("worker.%s = %q, want %q", key, gotValue, expectedValue)
		}
	}

	// Test environment resolution with Azure env and service URLs
	azureEnv := map[string]string{
		"AZURE_SUBSCRIPTION_ID": "test-sub-id",
		"AZURE_LOCATION":        "eastus",
	}

	serviceURLs := map[string]string{
		"SERVICE_URL_WEB": "http://localhost:3000",
		"SERVICE_URL_API": "http://localhost:5000",
	}

	// Resolve environment for web service
	resolvedEnv, err := ResolveEnvironment(web, azureEnv, "", serviceURLs)
	if err != nil {
		t.Fatalf("Failed to resolve environment: %v", err)
	}

	// Check that azure.yaml env vars are present
	if resolvedEnv["NODE_ENV"] != "production" {
		t.Errorf("Resolved NODE_ENV = %q, want %q", resolvedEnv["NODE_ENV"], "production")
	}

	// Check that Azure env vars are present
	if resolvedEnv["AZURE_SUBSCRIPTION_ID"] != "test-sub-id" {
		t.Errorf("Resolved AZURE_SUBSCRIPTION_ID = %q, want %q", resolvedEnv["AZURE_SUBSCRIPTION_ID"], "test-sub-id")
	}

	// Check that service URLs are present
	if resolvedEnv["SERVICE_URL_WEB"] != "http://localhost:3000" {
		t.Errorf("Resolved SERVICE_URL_WEB = %q, want %q", resolvedEnv["SERVICE_URL_WEB"], "http://localhost:3000")
	}

	// Azure.yaml env should override Azure env
	azureEnvWithConflict := map[string]string{
		"NODE_ENV": "development", // Should be overridden by azure.yaml
	}

	resolvedWithConflict, err := ResolveEnvironment(web, azureEnvWithConflict, "", serviceURLs)
	if err != nil {
		t.Fatalf("Failed to resolve environment with conflict: %v", err)
	}

	if resolvedWithConflict["NODE_ENV"] != "production" {
		t.Errorf("azure.yaml should override Azure env: got NODE_ENV = %q, want %q",
			resolvedWithConflict["NODE_ENV"], "production")
	}
}

// TestDockerComposeSpecialCharacters tests that special characters
// in environment variable values are handled correctly.
func TestDockerComposeSpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	azureYamlContent := `name: special-chars-test

services:
  db:
    language: node
    host: localhost
    environment:
      # Connection strings with special characters
      DATABASE_URL: "postgresql://user:p@ssw0rd@localhost:5432/db"
      REDIS_URL: "redis://:my!secret@localhost:6379/0"
      # API keys with special characters
      API_KEY: "abc123!@#$%^&*()"
      # Equals in value
      CONNECTION_STRING: "Server=localhost;User=admin;Password=pass=123"
`

	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	azureYaml, err := ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	db := azureYaml.Services["db"]
	dbEnv := db.GetEnvironment()

	// Verify special characters are preserved
	expected := map[string]string{
		"DATABASE_URL":      "postgresql://user:p@ssw0rd@localhost:5432/db",
		"REDIS_URL":         "redis://:my!secret@localhost:6379/0",
		"API_KEY":           "abc123!@#$%^&*()",
		"CONNECTION_STRING": "Server=localhost;User=admin;Password=pass=123",
	}

	for key, expectedValue := range expected {
		if gotValue := dbEnv[key]; gotValue != expectedValue {
			t.Errorf("%s = %q, want %q", key, gotValue, expectedValue)
		}
	}
}

// TestDockerComposeArrayFormatEdgeCases tests edge cases in array format.
func TestDockerComposeArrayFormatEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	azureYamlContent := `name: edge-cases-test

services:
  test:
    language: node
    host: localhost
    environment:
      # Key without value
      - EMPTY_VAR
      # Multiple equals signs
      - COMPLEX=key=value=something
      # Spaces around equals (should work)
      - SPACED = value with spaces
      # Empty value
      - EXPLICIT_EMPTY=
`

	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYamlContent), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}

	azureYaml, err := ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	test := azureYaml.Services["test"]
	testEnv := test.GetEnvironment()

	// Key without value should be empty string
	if val, exists := testEnv["EMPTY_VAR"]; !exists {
		t.Error("EMPTY_VAR should exist")
	} else if val != "" {
		t.Errorf("EMPTY_VAR = %q, want empty string", val)
	}

	// Multiple equals should preserve everything after first =
	if testEnv["COMPLEX"] != "key=value=something" {
		t.Errorf("COMPLEX = %q, want %q", testEnv["COMPLEX"], "key=value=something")
	}

	// Explicit empty value
	if val, exists := testEnv["EXPLICIT_EMPTY"]; !exists {
		t.Error("EXPLICIT_EMPTY should exist")
	} else if val != "" {
		t.Errorf("EXPLICIT_EMPTY = %q, want empty string", val)
	}
}
