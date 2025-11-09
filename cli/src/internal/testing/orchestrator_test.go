package testing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTestOrchestrator(t *testing.T) {
	config := &TestConfig{
		Parallel: true,
		Verbose:  false,
	}

	orchestrator := NewTestOrchestrator(config)
	if orchestrator == nil {
		t.Fatal("Expected orchestrator to be created")
	}
	if orchestrator.config != config {
		t.Error("Config not set correctly")
	}
	if len(orchestrator.services) != 0 {
		t.Error("Services should be empty initially")
	}
}

func TestLoadServicesFromAzureYaml(t *testing.T) {
	// Create a temporary azure.yaml file
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services:
  web:
    language: js
    project: ./src/web
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("LoadServicesFromAzureYaml failed: %v", err)
	}

	if len(orchestrator.services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(orchestrator.services))
	}

	// Check first service
	found := false
	for _, svc := range orchestrator.services {
		if svc.Name == "web" {
			found = true
			if svc.Language != "js" {
				t.Errorf("Expected language 'js', got '%s'", svc.Language)
			}
		}
	}
	if !found {
		t.Error("Service 'web' not found")
	}
}

func TestLoadServicesFromAzureYaml_NoServices(t *testing.T) {
	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	yamlContent := `name: test-app
services: {}
`

	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml(azureYamlPath)
	if err == nil {
		t.Error("Expected error for no services")
	}
}

func TestLoadServicesFromAzureYaml_InvalidPath(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	err := orchestrator.LoadServicesFromAzureYaml("/non/existent/path")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestDetectTestConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service directory with package.json
	serviceDir := filepath.Join(tmpDir, "web")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		t.Fatalf("Failed to create service dir: %v", err)
	}

	packageJSON := `{
		"name": "web",
		"scripts": {
			"test": "jest"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(serviceDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	service := ServiceInfo{
		Name:     "web",
		Language: "js",
		Dir:      serviceDir,
		Config:   nil,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig.Framework != "jest" {
		t.Errorf("Expected framework 'jest', got '%s'", testConfig.Framework)
	}
}

func TestDetectTestConfig_ExistingConfig(t *testing.T) {
	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)

	existingConfig := &ServiceTestConfig{
		Framework: "custom",
	}

	service := ServiceInfo{
		Name:     "web",
		Language: "js",
		Dir:      "/tmp",
		Config:   existingConfig,
	}

	testConfig, err := orchestrator.DetectTestConfig(service)
	if err != nil {
		t.Fatalf("DetectTestConfig failed: %v", err)
	}

	if testConfig != existingConfig {
		t.Error("Should return existing config")
	}
}

func TestGetServicePaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service directories
	webDir := filepath.Join(tmpDir, "web")
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatalf("Failed to create web dir: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}

	config := &TestConfig{}
	orchestrator := NewTestOrchestrator(config)
	orchestrator.services = []ServiceInfo{
		{Name: "web", Dir: webDir},
		{Name: "api", Dir: apiDir},
	}

	paths, err := orchestrator.GetServicePaths()
	if err != nil {
		t.Fatalf("GetServicePaths failed: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	// Check paths are included
	foundWeb := false
	foundAPI := false
	for _, path := range paths {
		if path == webDir {
			foundWeb = true
		}
		if path == apiDir {
			foundAPI = true
		}
	}

	if !foundWeb || !foundAPI {
		t.Error("Expected both service paths to be returned")
	}
}
