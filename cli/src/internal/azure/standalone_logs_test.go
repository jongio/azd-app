package azure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendToEnvFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".azure", "test-env", ".env")

	// Test 1: Create new file with variable
	err := appendToEnvFile(envPath, "TEST_VAR", "test_value")
	if err != nil {
		t.Fatalf("appendToEnvFile() failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	expectedContent := "TEST_VAR=test_value"
	if !strings.Contains(string(content), expectedContent) {
		t.Errorf("Expected content to contain %q, got %q", expectedContent, string(content))
	}

	// Test 2: Update existing variable
	err = appendToEnvFile(envPath, "TEST_VAR", "new_value")
	if err != nil {
		t.Fatalf("appendToEnvFile() update failed: %v", err)
	}

	content, err = os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	if !strings.Contains(string(content), "TEST_VAR=new_value") {
		t.Errorf("Expected updated value, got %q", string(content))
	}

	// Verify old value is not present
	if strings.Contains(string(content), "test_value") {
		t.Errorf("Old value should be replaced, got %q", string(content))
	}

	// Test 3: Add another variable
	err = appendToEnvFile(envPath, "ANOTHER_VAR", "another_value")
	if err != nil {
		t.Fatalf("appendToEnvFile() append failed: %v", err)
	}

	content, err = os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read env file: %v", err)
	}

	if !strings.Contains(string(content), "TEST_VAR=new_value") {
		t.Errorf("Expected first variable to remain, got %q", string(content))
	}

	if !strings.Contains(string(content), "ANOTHER_VAR=another_value") {
		t.Errorf("Expected second variable, got %q", string(content))
	}
}

func TestDiscoverAndStoreWorkspaceID_AlreadySet(t *testing.T) {
	// Save original env var
	originalGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	defer func() {
		if originalGUID != "" {
			os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", originalGUID)
		} else {
			os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
		}
	}()

	// Set env var
	testGUID := "test-workspace-id"
	os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", testGUID)

	// Create temp directory
	tmpDir := t.TempDir()

	// Call discovery
	guid, wasDiscovered, err := DiscoverAndStoreWorkspaceID(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("DiscoverAndStoreWorkspaceID() failed: %v", err)
	}

	// Should return existing GUID without discovering
	if wasDiscovered {
		t.Error("Expected wasDiscovered=false when GUID already set")
	}

	if guid != testGUID {
		t.Errorf("Expected GUID %q, got %q", testGUID, guid)
	}
}

func TestDiscoverAndStoreWorkspaceID_NoResourceGroup(t *testing.T) {
	// Save and clear env vars
	originalGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	originalRG := os.Getenv("AZURE_RESOURCE_GROUP")
	defer func() {
		if originalGUID != "" {
			os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", originalGUID)
		} else {
			os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
		}
		if originalRG != "" {
			os.Setenv("AZURE_RESOURCE_GROUP", originalRG)
		} else {
			os.Unsetenv("AZURE_RESOURCE_GROUP")
		}
	}()

	os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	os.Unsetenv("AZURE_RESOURCE_GROUP")

	// Create temp directory
	tmpDir := t.TempDir()

	// Call discovery
	_, _, err := DiscoverAndStoreWorkspaceID(context.Background(), tmpDir)
	if err == nil {
		t.Error("Expected error when AZURE_RESOURCE_GROUP not set")
	}

	if !strings.Contains(err.Error(), "AZURE_RESOURCE_GROUP") {
		t.Errorf("Expected error about AZURE_RESOURCE_GROUP, got: %v", err)
	}
}

func TestGetWorkspaceIDFromEnv_FromEnvVar(t *testing.T) {
	// Save original env var
	originalGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	defer func() {
		if originalGUID != "" {
			os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", originalGUID)
		} else {
			os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
		}
	}()

	// Set env var
	testGUID := "test-workspace-id-from-env"
	os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", testGUID)

	// Call function
	guid := GetWorkspaceIDFromEnv("")
	if guid != testGUID {
		t.Errorf("Expected GUID %q, got %q", testGUID, guid)
	}
}

func TestGetWorkspaceIDFromEnv_FromFile(t *testing.T) {
	// Save and clear env var
	originalGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	defer func() {
		if originalGUID != "" {
			os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", originalGUID)
		} else {
			os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
		}
	}()
	os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")

	// Create temp directory with .azure/test-env/.env file
	tmpDir := t.TempDir()

	// Create .azure/.env with env name
	defaultEnvFile := filepath.Join(tmpDir, ".azure", ".env")
	if err := os.MkdirAll(filepath.Dir(defaultEnvFile), 0755); err != nil {
		t.Fatalf("Failed to create .azure directory: %v", err)
	}
	if err := os.WriteFile(defaultEnvFile, []byte("AZURE_ENV_NAME=test-env\n"), 0644); err != nil {
		t.Fatalf("Failed to create .azure/.env: %v", err)
	}

	// Create environment .env file with workspace ID
	envFile := filepath.Join(tmpDir, ".azure", "test-env", ".env")
	if err := os.MkdirAll(filepath.Dir(envFile), 0755); err != nil {
		t.Fatalf("Failed to create env directory: %v", err)
	}

	testGUID := "test-workspace-id-from-file"
	envContent := "AZURE_RESOURCE_GROUP=test-rg\nAZURE_LOG_ANALYTICS_WORKSPACE_GUID=" + testGUID + "\n"
	if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to create env file: %v", err)
	}

	// Call function
	guid := GetWorkspaceIDFromEnv(tmpDir)
	if guid != testGUID {
		t.Errorf("Expected GUID %q from file, got %q", testGUID, guid)
	}
}

func TestGetWorkspaceIDFromEnv_NotFound(t *testing.T) {
	// Save and clear env var
	originalGUID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
	defer func() {
		if originalGUID != "" {
			os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID", originalGUID)
		} else {
			os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")
		}
	}()
	os.Unsetenv("AZURE_LOG_ANALYTICS_WORKSPACE_GUID")

	// Create temp directory without .env file
	tmpDir := t.TempDir()

	// Call function
	guid := GetWorkspaceIDFromEnv(tmpDir)
	if guid != "" {
		t.Errorf("Expected empty GUID, got %q", guid)
	}
}

func TestMapServiceNames_UsesAzureNameMapping(t *testing.T) {
	entries := []LogEntry{{
		Service:       "api-azure",
		ContainerName: "api-azure",
	}}
	services := []ServiceInfo{{
		Name:      "api",
		AzureName: "api-azure",
	}}

	result := mapServiceNames(entries, services)
	if result[0].Service != "api" {
		t.Fatalf("expected logical service name 'api', got %q", result[0].Service)
	}
}

func TestMapServiceNames_PrefersContainerNameWhenServiceEmpty(t *testing.T) {
	entries := []LogEntry{{
		Service:       "",
		ContainerName: "web-backend",
		InstanceID:    "rev-123",
	}}
	services := []ServiceInfo{{
		Name:      "web",
		AzureName: "web-backend",
	}}

	result := mapServiceNames(entries, services)
	if result[0].Service != "web" {
		t.Fatalf("expected logical service name 'web', got %q", result[0].Service)
	}
}
