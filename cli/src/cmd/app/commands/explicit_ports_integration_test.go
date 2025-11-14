//go:build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/service"

	"gopkg.in/yaml.v3"
)

// TestExplicitPortsFromAzureYaml tests end-to-end explicit port handling with azure.yaml files.
// This test uses a real test project file to verify integration.
func TestExplicitPortsFromAzureYaml(t *testing.T) {
	// Use existing test azure.yaml - this is an integration test
	testProject := filepath.Join("..", "..", "..", "tests", "projects", "azure")

	// Read azure.yaml
	azureYamlPath := filepath.Join(testProject, "azure.yaml")
	data, err := os.ReadFile(azureYamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Skipping: test azure.yaml not found at %s", azureYamlPath)
		}
		t.Fatalf("Failed to read azure.yaml: %v", err)
	}

	var azureYaml struct {
		Services map[string]service.Service `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &azureYaml); err != nil {
		t.Fatalf("Failed to parse azure.yaml: %v", err)
	}

	if len(azureYaml.Services) == 0 {
		t.Skip("No services defined in azure.yaml")
	}

	// Test port detection for each service
	for serviceName, svc := range azureYaml.Services {
		t.Run(serviceName, func(t *testing.T) {
			port, isExplicit, err := service.DetectPort(serviceName, svc, testProject, "", nil)

			t.Logf("Service %q: port=%d, isExplicit=%v, err=%v", serviceName, port, isExplicit, err)

			// If service has Ports configured, it should be detected as explicit
			if hostPort, _, hasExplicit := svc.GetPrimaryPort(); hasExplicit && hostPort > 0 {
				if !isExplicit {
					t.Errorf("Service %q has ports configured but isExplicit=false", serviceName)
				}
				if port != hostPort {
					t.Errorf("Service %q detected port %d, want %d", serviceName, port, hostPort)
				}
			}
		})
	}
}

// TestExplicitPortWithPortManager tests assigning explicit ports through port manager.
func TestExplicitPortWithPortManager(t *testing.T) {
	tempDir := t.TempDir()
	pm := portmanager.GetPortManager(tempDir)

	tests := []struct {
		name         string
		serviceName  string
		port         int
		isExplicit   bool
		wantErr      bool
		errSubstring string
	}{
		{
			name:        "explicit port in valid range",
			serviceName: "web",
			port:        3100,
			isExplicit:  true,
			wantErr:     false,
		},
		{
			name:         "explicit port below minimum",
			serviceName:  "api",
			port:         2000,
			isExplicit:   true,
			wantErr:      true,
			errSubstring: "outside valid range",
		},
		{
			name:        "flexible port assigns successfully",
			serviceName: "worker",
			port:        3200,
			isExplicit:  false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, _, err := pm.AssignPort(tt.serviceName, tt.port, tt.isExplicit)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("AssignPort() expected error, got port=%d", port)
				}
				if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("AssignPort() error = %v, want substring %q", err, tt.errSubstring)
				}
				return
			}

			if err != nil {
				t.Fatalf("AssignPort() unexpected error: %v", err)
			}

			if port <= 0 {
				t.Errorf("AssignPort() returned invalid port %d", port)
			}

			// Verify assignment was persisted
			if gotPort, exists := pm.GetAssignment(tt.serviceName); !exists || gotPort != port {
				t.Errorf("GetAssignment() = %d, %v; want %d, true", gotPort, exists, port)
			}

			// Cleanup
			if err := pm.ReleasePort(tt.serviceName); err != nil {
				t.Logf("Warning: failed to release port: %v", err)
			}
		})
	}
}

// TestMultipleServicesMixedPorts tests assigning ports to multiple services with mixed explicit/flexible ports.
func TestMultipleServicesMixedPorts(t *testing.T) {
	tempDir := t.TempDir()
	pm := portmanager.GetPortManager(tempDir)

	services := []struct {
		name       string
		port       int
		isExplicit bool
	}{
		{name: "web", port: 3000, isExplicit: true},     // Explicit - MUST get 3000
		{name: "api", port: 8080, isExplicit: true},     // Explicit - MUST get 8080
		{name: "worker", port: 3100, isExplicit: false}, // Flexible - can get 3100 or alternative
		{name: "queue", port: 5000, isExplicit: false},  // Flexible - can get 5000 or alternative
	}

	assignedPorts := make(map[string]int)
	// Cleanup at the end
	defer func() {
		for name := range assignedPorts {
			if err := pm.ReleasePort(name); err != nil {
				t.Logf("Warning: failed to release port for %s: %v", name, err)
			}
		}
	}()

	for _, svc := range services {
		port, _, err := pm.AssignPort(svc.name, svc.port, svc.isExplicit)
		if err != nil {
			t.Fatalf("AssignPort(%s) failed: %v", svc.name, err)
		}
		assignedPorts[svc.name] = port

		// Explicit ports MUST match requested
		if svc.isExplicit && port != svc.port {
			t.Errorf("Service %q with explicit port %d got %d", svc.name, svc.port, port)
		}

		t.Logf("Service %q assigned port %d (explicit=%v)", svc.name, port, svc.isExplicit)
	}

	// Verify all assignments are in valid range
	for name, port := range assignedPorts {
		if port < 3000 || port > 65535 {
			t.Errorf("Service %q port %d is outside valid range 3000-65535", name, port)
		}
	}
}

// TestExplicitPortPriority tests that explicit ports from azure.yaml override other sources.
func TestExplicitPortPriority(t *testing.T) {
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
		name         string
		service      service.Service
		wantPort     int
		wantExplicit bool
	}{
		{
			name: "azure.yaml port overrides package.json",
			service: service.Service{
				Ports: []string{"4000"},
			},
			wantPort:     4000,
			wantExplicit: true,
		},
		{
			name: "package.json port when no azure.yaml port",
			service: service.Service{
				Ports: nil,
			},
			wantPort:     3333,
			wantExplicit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPort, gotExplicit, err := service.DetectPort("test", tt.service, tempDir, "Next.js", nil)
			if err != nil {
				t.Fatalf("DetectPort() unexpected error: %v", err)
			}

			if gotPort != tt.wantPort {
				t.Errorf("DetectPort() port = %d, want %d", gotPort, tt.wantPort)
			}

			if gotExplicit != tt.wantExplicit {
				t.Errorf("DetectPort() isExplicit = %v, want %v", gotExplicit, tt.wantExplicit)
			}
		})
	}
}
