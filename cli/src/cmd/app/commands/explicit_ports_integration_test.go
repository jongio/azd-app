//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/service"

	"gopkg.in/yaml.v3"
)

// TestExplicitPorts_FromAzureYaml tests end-to-end explicit port handling with real azure.yaml files.
func TestExplicitPorts_FromAzureYaml(t *testing.T) {
	// Use existing test azure.yaml
	testProject := filepath.Join("..", "..", "..", "tests", "projects", "azure")

	// Read azure.yaml
	azureYamlPath := filepath.Join(testProject, "azure.yaml")
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Skipf("Skipping: azure.yaml not found at %s", azureYamlPath)
	}

	var azureYaml struct {
		Services map[string]service.Service `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	// Test port detection for each service
	for serviceName, svc := range azureYaml.Services {
		port, isExplicit, err := service.DetectPort(serviceName, svc, testProject, "", nil)

		// Log results
		t.Logf("Service '%s': port=%d, isExplicit=%v, err=%v", serviceName, port, isExplicit, err)

		// If service has config.port, it should be explicit
		if svc.Config != nil {
			if portVal, exists := svc.Config["port"]; exists && portVal != nil {
				if !isExplicit {
					t.Errorf("Service '%s' has config.port but isExplicit=false", serviceName)
				}
			}
		}
	}
}

// TestExplicitPort_WithPortManager tests assigning explicit ports through port manager.
func TestExplicitPort_WithPortManager(t *testing.T) {
	tempDir := t.TempDir()
	pm := portmanager.GetPortManager(tempDir)

	tests := []struct {
		name        string
		serviceName string
		port        int
		isExplicit  bool
		shouldError bool
	}{
		{
			name:        "Explicit port in valid range",
			serviceName: "web",
			port:        3100,
			isExplicit:  true,
			shouldError: false,
		},
		{
			name:        "Explicit port out of range",
			serviceName: "api",
			port:        2000, // Below 3000
			isExplicit:  true,
			shouldError: true,
		},
		{
			name:        "Flexible port",
			serviceName: "worker",
			port:        3200,
			isExplicit:  false,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := pm.AssignPort(tt.serviceName, tt.port, tt.isExplicit, false)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none, port=%d", port)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if port == 0 {
					t.Error("Expected valid port, got 0")
				}
			}
		})
	}
}

// TestMultipleServices_MixedPorts tests assigning ports to multiple services with mixed explicit/flexible ports.
func TestMultipleServices_MixedPorts(t *testing.T) {
	tempDir := t.TempDir()
	pm := portmanager.GetPortManager(tempDir)

	services := []struct {
		name       string
		port       int
		isExplicit bool
	}{
		{"web", 3000, true},     // Explicit - MUST get 3000
		{"api", 8080, true},     // Explicit - MUST get 8080
		{"worker", 3100, false}, // Flexible - can get 3100 or alternative
		{"queue", 5000, false},  // Flexible - can get 5000 or alternative
	}

	assignedPorts := make(map[string]int)

	for _, svc := range services {
		port, err := pm.AssignPort(svc.name, svc.port, svc.isExplicit, false)
		if err != nil {
			t.Fatalf("Failed to assign port for %s: %v", svc.name, err)
		}
		assignedPorts[svc.name] = port

		// Explicit ports MUST match requested
		if svc.isExplicit && port != svc.port {
			t.Errorf("Service '%s' with explicit port %d got %d", svc.name, svc.port, port)
		}
	}

	// Log assignments
	for name, port := range assignedPorts {
		t.Logf("Service '%s' assigned port %d", name, port)
	}

	// Verify all assignments are in valid range
	for name, port := range assignedPorts {
		if port < 3000 || port > 9999 {
			t.Errorf("Service '%s' port %d is outside valid range 3000-9999", name, port)
		}
	}
}

// TestExplicitPort_Priority tests that explicit ports from azure.yaml override other sources.
func TestExplicitPort_Priority(t *testing.T) {
	tempDir := t.TempDir()

	// Create package.json with port 3333
	packageJSON := `{
		"scripts": {
			"dev": "next dev --port 3333"
		}
	}`

	packagePath := filepath.Join(tempDir, "package.json")
	if err := os.WriteFile(packagePath, []byte(packageJSON), 0600); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	tests := []struct {
		name             string
		service          service.Service
		expectedPort     int
		expectedExplicit bool
	}{
		{
			name: "Azure.yaml port overrides package.json",
			service: service.Service{
				Config: map[string]interface{}{
					"port": 4000,
				},
			},
			expectedPort:     4000,
			expectedExplicit: true,
		},
		{
			name: "Package.json port when no azure.yaml config",
			service: service.Service{
				Config: nil,
			},
			expectedPort:     3333,
			expectedExplicit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, isExplicit, err := service.DetectPort("test", tt.service, tempDir, "Next.js", nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}

			if isExplicit != tt.expectedExplicit {
				t.Errorf("Expected isExplicit=%v, got %v", tt.expectedExplicit, isExplicit)
			}
		})
	}
}
