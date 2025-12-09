//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for the add command.
// Run with: go test -tags=integration ./...

func TestAddCommandIntegration_AddAzurite(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"azurite"})

	// Execute
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Read back the file
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Verify azurite was added
	if !strings.Contains(yamlContent, "azurite:") {
		t.Error("azure.yaml does not contain azurite service")
	}
	if !strings.Contains(yamlContent, "mcr.microsoft.com/azure-storage/azurite") {
		t.Error("azure.yaml does not contain azurite image")
	}
	if !strings.Contains(yamlContent, "10000:10000") {
		t.Error("azure.yaml does not contain blob port")
	}
	if !strings.Contains(yamlContent, "10001:10001") {
		t.Error("azure.yaml does not contain queue port")
	}
	if !strings.Contains(yamlContent, "10002:10002") {
		t.Error("azure.yaml does not contain table port")
	}
}

func TestAddCommandIntegration_AddCosmos(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"cosmos"})

	// Execute
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Read back the file
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Verify cosmos was added
	if !strings.Contains(yamlContent, "cosmos:") {
		t.Error("azure.yaml does not contain cosmos service")
	}
	if !strings.Contains(yamlContent, "mcr.microsoft.com/cosmosdb/linux/azure-cosmos-emulator") {
		t.Error("azure.yaml does not contain cosmos image")
	}
	if !strings.Contains(yamlContent, "8081:8081") {
		t.Error("azure.yaml does not contain cosmos HTTPS port")
	}
	// Check for environment variables
	if !strings.Contains(yamlContent, "AZURE_COSMOS_EMULATOR_PARTITION_COUNT") {
		t.Error("azure.yaml does not contain cosmos partition count env var")
	}
}

func TestAddCommandIntegration_AddRedis(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"redis"})

	// Execute
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Read back the file
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Verify redis was added with new services section
	if !strings.Contains(yamlContent, "services:") {
		t.Error("azure.yaml does not contain services section")
	}
	if !strings.Contains(yamlContent, "redis:") {
		t.Error("azure.yaml does not contain redis service")
	}
	if !strings.Contains(yamlContent, "redis:7-alpine") {
		t.Error("azure.yaml does not contain redis image")
	}
	if !strings.Contains(yamlContent, "6379:6379") {
		t.Error("azure.yaml does not contain redis port")
	}
}

func TestAddCommandIntegration_AddPostgres(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
services:
  web:
    language: javascript
    project: ./web
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"postgres"})

	// Execute
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// Read back the file
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Verify postgres was added
	if !strings.Contains(yamlContent, "postgres:") {
		t.Error("azure.yaml does not contain postgres service")
	}
	if !strings.Contains(yamlContent, "postgres:16-alpine") {
		t.Error("azure.yaml does not contain postgres image")
	}
	if !strings.Contains(yamlContent, "5432:5432") {
		t.Error("azure.yaml does not contain postgres port")
	}
	// Check for environment variables
	if !strings.Contains(yamlContent, "POSTGRES_USER") {
		t.Error("azure.yaml does not contain POSTGRES_USER env var")
	}
	if !strings.Contains(yamlContent, "POSTGRES_PASSWORD") {
		t.Error("azure.yaml does not contain POSTGRES_PASSWORD env var")
	}
}

func TestAddCommandIntegration_DuplicateService(t *testing.T) {
	tempDir := t.TempDir()

	// Create azure.yaml with existing redis
	content := `name: test-app
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command - should not error, but not duplicate
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"redis"})

	// Execute - should succeed (with warning message)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed unexpectedly: %v", err)
	}

	// Read back the file - should only have one redis service section
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Count occurrences of "  redis:" (service key with indent) - should be exactly 1
	// Note: We match with leading spaces to avoid counting "redis:7-alpine" image tag
	count := strings.Count(yamlContent, "  redis:")
	if count != 1 {
		t.Errorf("expected 1 occurrence of service 'redis:', got %d\nContent:\n%s", count, yamlContent)
	}
}

func TestAddCommandIntegration_UnknownService(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command with unknown service
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"unknown-service"})

	// Execute - should fail
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
	if !strings.Contains(err.Error(), "unknown service") {
		t.Errorf("expected error to contain 'unknown service', got: %v", err)
	}
}

func TestAddCommandIntegration_NoAzureYaml(t *testing.T) {
	tempDir := t.TempDir()

	// Change to temp directory (no azure.yaml)
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"redis"})

	// Execute - should fail
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no azure.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "azure.yaml") {
		t.Errorf("expected error to mention azure.yaml, got: %v", err)
	}
}

func TestAddCommandIntegration_ListServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml (needed for command context)
	content := `name: test-app
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Create and execute add command with --list
	cmd := NewAddCommand()
	cmd.SetArgs([]string{"--list"})

	// Execute - should succeed
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add --list command failed: %v", err)
	}
}

func TestAddCommandIntegration_MultipleServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Add multiple services sequentially
	services := []string{"azurite", "redis", "postgres"}
	for _, svc := range services {
		cmd := NewAddCommand()
		cmd.SetArgs([]string{svc})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add %s command failed: %v", svc, err)
		}
	}

	// Read back the file
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	yamlContent := string(data)

	// Verify all services were added
	for _, svc := range services {
		if !strings.Contains(yamlContent, svc+":") {
			t.Errorf("azure.yaml does not contain %s service", svc)
		}
	}

	// Verify original api service still exists
	if !strings.Contains(yamlContent, "api:") {
		t.Error("azure.yaml lost original api service")
	}
}
