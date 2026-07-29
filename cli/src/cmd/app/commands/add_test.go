package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/wellknown"
)

func TestServiceExistsInYaml(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()

	// Create azure.yaml with existing service
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
  azurite:
    host: containerapp
    image: mcr.microsoft.com/azure-storage/azurite:latest
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	tests := []struct {
		name     string
		service  string
		expected bool
	}{
		{"existing service api", "api", true},
		{"existing service azurite", "azurite", true},
		{"non-existing service redis", "redis", false},
		{"non-existing service cosmos", "cosmos", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := serviceExistsInYaml(azureYamlPath, tt.service)
			if err != nil {
				t.Fatalf("serviceExistsInYaml() error: %v", err)
			}
			if exists != tt.expected {
				t.Errorf("serviceExistsInYaml(%q) = %v, want %v", tt.service, exists, tt.expected)
			}
		})
	}
}

func TestServiceExistsInYamlNoServices(t *testing.T) {
	tempDir := t.TempDir()

	// Create azure.yaml without services section
	content := `name: test-app
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	exists, err := serviceExistsInYaml(azureYamlPath, "azurite")
	if err != nil {
		t.Fatalf("serviceExistsInYaml() error: %v", err)
	}
	if exists {
		t.Error("serviceExistsInYaml() = true for non-existent services section, want false")
	}
}

func TestAddServiceToYaml(t *testing.T) {
	tempDir := t.TempDir()

	// Create initial azure.yaml
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Get redis definition
	def := wellknown.Get("redis")
	if def == nil {
		t.Fatal("redis service not found in wellknown registry")
	}

	// Add redis service
	if err := addServiceToYaml(azureYamlPath, "redis", def); err != nil {
		t.Fatalf("addServiceToYaml() error: %v", err)
	}

	// Verify service was added
	exists, err := serviceExistsInYaml(azureYamlPath, "redis")
	if err != nil {
		t.Fatalf("serviceExistsInYaml() error: %v", err)
	}
	if !exists {
		t.Error("redis service was not added to azure.yaml")
	}

	// Read file content and verify structure
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	content = string(data)

	// Check redis image is present
	if !strings.Contains(content, "redis:7-alpine") {
		t.Error("azure.yaml does not contain redis:7-alpine image")
	}

	// Check ports are present
	if !strings.Contains(content, "6379:6379") {
		t.Error("azure.yaml does not contain redis port 6379:6379")
	}
}

func TestAddServiceToYamlCreatesServicesSection(t *testing.T) {
	tempDir := t.TempDir()

	// Create azure.yaml without services section
	content := `name: test-app
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	// Get azurite definition
	def := wellknown.Get("azurite")
	if def == nil {
		t.Fatal("azurite service not found in wellknown registry")
	}

	// Add azurite service
	if err := addServiceToYaml(azureYamlPath, "azurite", def); err != nil {
		t.Fatalf("addServiceToYaml() error: %v", err)
	}

	// Verify service was added
	exists, err := serviceExistsInYaml(azureYamlPath, "azurite")
	if err != nil {
		t.Fatalf("serviceExistsInYaml() error: %v", err)
	}
	if !exists {
		t.Error("azurite service was not added to azure.yaml")
	}

	// Read file content and verify services section was created
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	content = string(data)

	if !strings.Contains(content, "services:") {
		t.Error("azure.yaml does not contain services section")
	}
}

func TestBuildServiceNode(t *testing.T) {
	def := wellknown.Get("postgres")
	if def == nil {
		t.Fatal("postgres service not found in wellknown registry")
	}

	node := buildServiceNode(def)

	if node == nil {
		t.Fatal("buildServiceNode() returned nil")
	}

	// Check that node is a mapping
	if node.Kind != 4 { // yaml.MappingNode = 4
		t.Errorf("node.Kind = %d, want 4 (MappingNode)", node.Kind)
	}

	// Check that image, ports, and environment are in the content
	hasImage := false
	hasPorts := false
	hasEnv := false

	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value
		switch key {
		case "image":
			hasImage = true
		case "ports":
			hasPorts = true
		case "environment":
			hasEnv = true
		}
	}

	if !hasImage {
		t.Error("buildServiceNode() did not include image")
	}
	if !hasPorts {
		t.Error("buildServiceNode() did not include ports")
	}
	if !hasEnv {
		t.Error("buildServiceNode() did not include environment (postgres has env vars)")
	}
}

func TestBuildServicePreview(t *testing.T) {
	def := wellknown.Get("redis")
	if def == nil {
		t.Fatal("redis service not found in wellknown registry")
	}

	preview, err := buildServicePreview("redis", def)
	if err != nil {
		t.Fatalf("buildServicePreview() error: %v", err)
	}

	for _, want := range []string{"services:", "redis:", "image: redis:7-alpine", "6379:6379"} {
		if !strings.Contains(preview, want) {
			t.Errorf("preview missing %q:\n%s", want, preview)
		}
	}
}

func TestRunAddDryRunDoesNotModifyAzureYaml(t *testing.T) {
	tempDir := t.TempDir()
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("failed to restore cwd: %v", chdirErr)
		}
	})
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cmd := NewAddCommand()
	cmd.SetArgs([]string{"redis", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add --dry-run failed: %v", err)
	}

	after, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	if string(after) != content {
		t.Fatalf("dry-run modified azure.yaml:\n%s", string(after))
	}
}

func TestAddServiceToYamlUsesTwoSpaceIndentAndAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	content := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tempDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create azure.yaml: %v", err)
	}

	def := wellknown.Get("redis")
	if def == nil {
		t.Fatal("redis service not found in wellknown registry")
	}
	if err := addServiceToYaml(azureYamlPath, "redis", def); err != nil {
		t.Fatalf("addServiceToYaml() error: %v", err)
	}
	if _, err := os.Stat(azureYamlPath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file cleanup error = %v, want not exist", err)
	}

	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("failed to read azure.yaml: %v", err)
	}
	got := string(data)
	if strings.Contains(got, "    api:") || strings.Contains(got, "      language:") {
		t.Fatalf("azure.yaml was written with four-space mapping indentation:\n%s", got)
	}
	for _, want := range []string{"  api:", "    language: python", "  redis:", "    image: redis:7-alpine"} {
		if !strings.Contains(got, want) {
			t.Fatalf("azure.yaml missing %q after add:\n%s", want, got)
		}
	}
}
