package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/yamlutil"
)

// TestPortUserJourney_ExplicitPorts tests the full journey when user has explicit ports in azure.yaml
func TestPortUserJourney_ExplicitPorts(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml with explicit ports
	azureYaml := `name: test-app
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	// Create package.json that would suggest a different port
	packageJSON := `{
		"name": "api",
		"scripts": {
			"dev": "python -m uvicorn main:app --port 5000"
		}
	}`
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Detect port
	port, isExplicit, err := DetectPort("api", service, apiDir, "FastAPI", nil)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	// Verify explicit port takes precedence
	if port != 8080 {
		t.Errorf("Expected port 8080 from azure.yaml, got %d", port)
	}
	
	if !isExplicit {
		t.Error("Expected isExplicit=true for port from azure.yaml")
	}
	
	// Verify azure.yaml was NOT modified
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml: %v", err)
	}
	
	contentStr := string(content)
	if contentStr != azureYaml {
		t.Error("azure.yaml should not be modified when port is explicit")
	}
}

// TestPortUserJourney_FrameworkDefault tests the journey when using framework defaults
func TestPortUserJourney_FrameworkDefault(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml WITHOUT ports
	azureYaml := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Detect port (should use FastAPI default 8000)
	port, isExplicit, err := DetectPort("api", service, apiDir, "FastAPI", nil)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	// Verify framework default
	if port != 8000 {
		t.Errorf("Expected FastAPI default port 8000, got %d", port)
	}
	
	if isExplicit {
		t.Error("Expected isExplicit=false for framework default")
	}
	
	// Verify azure.yaml was NOT modified (framework default doesn't trigger update)
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml: %v", err)
	}
	
	contentStr := string(content)
	if contentStr != azureYaml {
		t.Error("azure.yaml should not be modified for framework default without conflict")
	}
}

// TestPortUserJourney_ConflictAutoAssign tests auto-assignment when port conflicts
func TestPortUserJourney_ConflictAutoAssign(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml WITHOUT ports
	azureYaml := `name: test-app
services:
  api:
    language: python
    project: ./api
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Simulate port 8000 being in use
	usedPorts := map[int]bool{8000: true}
	
	// Detect port (should auto-assign)
	port, isExplicit, err := DetectPort("api", service, apiDir, "FastAPI", usedPorts)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	// Verify auto-assigned port (not 8000)
	if port == 8000 {
		t.Error("Expected auto-assigned port, not the conflicting 8000")
	}
	
	if port < 1024 || port > 65535 {
		t.Errorf("Auto-assigned port %d is out of valid range", port)
	}
	
	if isExplicit {
		t.Error("Expected isExplicit=false for auto-assigned port")
	}
	
	// Update azure.yaml with the auto-assigned port
	if err := yamlutil.UpdateServicePort(azureYamlPath, "api", port); err != nil {
		t.Fatalf("Failed to update azure.yaml: %v", err)
	}
	
	// Verify azure.yaml was modified with ports array format
	content, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read azure.yaml: %v", err)
	}
	
	contentStr := string(content)
	
	// Should contain ports array
	if !contains(contentStr, "ports:") {
		t.Error("Expected azure.yaml to contain 'ports:' field")
	}
	
	// Should contain the auto-assigned port as string in array format
	if !contains(contentStr, `- "`) {
		t.Error("Expected ports array format with quoted string")
	}
}

// TestPortUserJourney_DockerContainerOnly tests Docker service with container-only port
func TestPortUserJourney_DockerContainerOnly(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml with Docker service using container-only port
	azureYaml := `name: test-app
services:
  api:
    language: python
    project: ./api
    docker:
      image: python:3.9
    ports:
      - "8080"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Get port mapping
	hostPort, containerPort, _ := service.GetPrimaryPort()
	
	// Verify Docker behavior: container port explicit, host port auto-assigned
	if containerPort != 8080 {
		t.Errorf("Expected container port 8080, got %d", containerPort)
	}
	
	if hostPort != 0 {
		t.Errorf("Expected host port 0 (auto-assign) for Docker, got %d", hostPort)
	}
	
	// In real scenario, HostPort would be assigned by Docker
	// For URL generation, we'd use the actual assigned port
}

// TestPortUserJourney_HostContainerMapping tests explicit host:container mapping
func TestPortUserJourney_HostContainerMapping(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml with host:container mapping
	azureYaml := `name: test-app
services:
  api:
    language: python
    project: ./api
    docker:
      image: python:3.9
    ports:
      - "3000:8080"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Detect port
	port, isExplicit, err := DetectPort("api", service, apiDir, "FastAPI", nil)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	// Verify host port is returned for URL generation
	if port != 3000 {
		t.Errorf("Expected host port 3000 for URL, got %d", port)
	}
	
	if !isExplicit {
		t.Error("Expected isExplicit=true for explicit mapping")
	}
	
	// Get full mapping
	hostPort, containerPort, _ := service.GetPrimaryPort()
	
	if hostPort != 3000 {
		t.Errorf("Expected host port 3000, got %d", hostPort)
	}
	
	if containerPort != 8080 {
		t.Errorf("Expected container port 8080, got %d", containerPort)
	}
}

// TestPortUserJourney_MultiplePorts tests service with multiple ports
func TestPortUserJourney_MultiplePorts(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml with multiple ports
	azureYaml := `name: test-app
services:
  api:
    language: go
    project: ./api
    ports:
      - "8080"
      - "9090"
      - "9091"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create api dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["api"]
	
	// Detect port (should return primary/first port)
	port, isExplicit, err := DetectPort("api", service, apiDir, "Go", nil)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	// Verify first port is returned (primary)
	if port != 8080 {
		t.Errorf("Expected primary port 8080, got %d", port)
	}
	
	if !isExplicit {
		t.Error("Expected isExplicit=true for explicit ports")
	}
	
	// Get all mappings
	mappings, _ := service.GetPortMappings()
	
	if len(mappings) != 3 {
		t.Errorf("Expected 3 port mappings, got %d", len(mappings))
	}
	
	expectedPorts := []int{8080, 9090, 9091}
	for i, mapping := range mappings {
		if mapping.HostPort != expectedPorts[i] {
			t.Errorf("Port %d: expected %d, got %d", i, expectedPorts[i], mapping.HostPort)
		}
		if mapping.ContainerPort != expectedPorts[i] {
			t.Errorf("Port %d: expected container port %d, got %d", i, expectedPorts[i], mapping.ContainerPort)
		}
	}
	
	// Verify primary port
	primaryHost, _, _ := service.GetPrimaryPort()
	if primaryHost != 8080 {
		t.Errorf("Expected primary port 8080, got %d", primaryHost)
	}
}

// TestPortUserJourney_IPBinding tests IP-specific port binding
func TestPortUserJourney_IPBinding(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create azure.yaml with IP binding
	azureYaml := `name: test-app
services:
  db:
    language: other
    project: ./db
    ports:
      - "127.0.0.1:5432:5432"
`
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(azureYaml), 0600); err != nil {
		t.Fatalf("Failed to create azure.yaml: %v", err)
	}
	
	dbDir := filepath.Join(tmpDir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("Failed to create db dir: %v", err)
	}
	
	// Parse azure.yaml
	parsedYaml, err := ParseAzureYaml(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}
	
	service := parsedYaml.Services["db"]
	
	// Get port mappings to check IP binding
	mappings, _ := service.GetPortMappings()
	if len(mappings) == 0 {
		t.Fatal("Expected at least one port mapping")
	}
	mapping := mappings[0]
	
	// Verify IP binding
	if mapping.BindIP != "127.0.0.1" {
		t.Errorf("Expected BindIP 127.0.0.1, got %q", mapping.BindIP)
	}
	
	if mapping.HostPort != 5432 {
		t.Errorf("Expected host port 5432, got %d", mapping.HostPort)
	}
	
	if mapping.ContainerPort != 5432 {
		t.Errorf("Expected container port 5432, got %d", mapping.ContainerPort)
	}
	
	// Detect port should return host port
	port, isExplicit, err := DetectPort("db", service, dbDir, "", nil)
	if err != nil {
		t.Fatalf("DetectPort failed: %v", err)
	}
	
	if port != 5432 {
		t.Errorf("Expected port 5432, got %d", port)
	}
	
	if !isExplicit {
		t.Error("Expected isExplicit=true for explicit IP binding")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)+1 && containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
