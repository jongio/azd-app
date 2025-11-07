package yamlutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateServicePort(t *testing.T) {
	tests := []struct {
		name            string
		initialYaml     string
		serviceName     string
		port            int
		expectedContain string
		wantErr         bool
	}{
		{
			name: "add port to service without port",
			initialYaml: `name: test-app
services:
  api:
    language: python
    project: ./api
  web:
    language: js
    project: ./web
`,
			serviceName: "api",
			port:        5000,
			expectedContain: `  api:
    ports:
      - "5000"
    language: python`,
		},
		{
			name: "update existing port",
			initialYaml: `name: test-app
services:
  api:
    ports:
      - "3000"
    language: python
    project: ./api
`,
			serviceName:     "api",
			port:            8080,
			expectedContain: `- "8080"`,
		},
		{
			name: "add port to service with comments",
			initialYaml: `name: test-app
services:
  # Main API service
  api:
    language: python  # Python FastAPI
    project: ./api
`,
			serviceName: "api",
			port:        5000,
			expectedContain: `  api:
    ports:
      - "5000"
    language: python`,
		},
		{
			name: "service not found",
			initialYaml: `name: test-app
services:
  api:
    language: python
`,
			serviceName: "web",
			port:        3000,
			wantErr:     true,
		},
		{
			name: "no services section",
			initialYaml: `name: test-app
reqs:
  - name: node
`,
			serviceName: "api",
			port:        3000,
			wantErr:     true,
		},
		{
			name: "update inline array format",
			initialYaml: `name: test-app
services:
  api:
    ports: ["3000", "8080"]
    language: python
`,
			serviceName:     "api",
			port:            5000,
			expectedContain: `ports: ["5000"]`,
		},
		{
			name: "update inline single port",
			initialYaml: `name: test-app
services:
  api:
    ports: ["3000"]
    language: python
`,
			serviceName:     "api",
			port:            8080,
			expectedContain: `ports: ["8080"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

			if err := os.WriteFile(azureYamlPath, []byte(tt.initialYaml), 0600); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Update port
			err := UpdateServicePort(azureYamlPath, tt.serviceName, tt.port)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Read result
			result, err := os.ReadFile(azureYamlPath)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			resultStr := string(result)

			// Check expected content
			if !strings.Contains(resultStr, tt.expectedContain) {
				t.Errorf("Expected result to contain:\n%s\n\nGot:\n%s", tt.expectedContain, resultStr)
			}

			// Verify comments are preserved
			if strings.Contains(tt.initialYaml, "#") {
				if !strings.Contains(resultStr, "#") {
					t.Error("Comments were not preserved")
				}
			}

			// Verify name is preserved
			if !strings.Contains(resultStr, "name: test-app") {
				t.Error("File name was not preserved")
			}
		})
	}
}

func TestUpdateServicePort_PreservesFormatting(t *testing.T) {
	initialYaml := `name: fullstack-app

# Service definitions
services:
  # Backend API
  api:
    language: python
    project: ./backend
    env:
      - name: DATABASE_URL
        value: postgres://localhost

  # Frontend web app
  web:
    language: js
    project: ./frontend

# System requirements
reqs:
  - name: node
    minVersion: "18.0.0"
`

	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")

	if err := os.WriteFile(azureYamlPath, []byte(initialYaml), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add port to api service
	if err := UpdateServicePort(azureYamlPath, "api", 5000); err != nil {
		t.Fatalf("Failed to update port: %v", err)
	}

	result, err := os.ReadFile(azureYamlPath)
	if err != nil {
		t.Fatalf("Failed to read result: %v", err)
	}

	resultStr := string(result)

	// Verify all comments are preserved
	expectedComments := []string{
		"# Service definitions",
		"# Backend API",
		"# Frontend web app",
		"# System requirements",
	}

	for _, comment := range expectedComments {
		if !strings.Contains(resultStr, comment) {
			t.Errorf("Comment not preserved: %s", comment)
		}
	}

	// Verify reqs section is preserved
	if !strings.Contains(resultStr, "reqs:") {
		t.Error("reqs section was not preserved")
	}

	// Verify port was added
	if !strings.Contains(resultStr, `ports:`) || !strings.Contains(resultStr, `- "5000"`) {
		t.Error("ports array was not added correctly")
	}

	// Verify env section is preserved
	if !strings.Contains(resultStr, "DATABASE_URL") {
		t.Error("env section was not preserved")
	}
}
