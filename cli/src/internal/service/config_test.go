package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
	"github.com/jongio/azd-core/urlutil"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid HTTP URL",
			url:     "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "Valid HTTPS URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "Valid HTTPS URL with path",
			url:     "https://example.com/api/v1",
			wantErr: false,
		},
		{
			name:    "Valid HTTPS URL with port and path",
			url:     "https://example.com:8443/api",
			wantErr: false,
		},
		{
			name:    "Valid HTTP ngrok URL",
			url:     "https://abc123.ngrok.io",
			wantErr: false,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
			errMsg:  "url cannot be empty",
		},
		{
			name:    "URL without protocol",
			url:     "example.com",
			wantErr: true,
			errMsg:  "url must use http:// or https://",
		},
		{
			name:    "URL with FTP protocol",
			url:     "ftp://example.com",
			wantErr: true,
			errMsg:  "url must use http:// or https://, got: ftp",
		},
		{
			name:    "URL with only http://",
			url:     "http://",
			wantErr: true,
			errMsg:  "url missing host/domain",
		},
		{
			name:    "URL with only https://",
			url:     "https://",
			wantErr: true,
			errMsg:  "url missing host/domain",
		},
		{
			name:    "URL with whitespace",
			url:     " https://example.com ",
			wantErr: false, // Should be trimmed
		},
		{
			name:    "Minimal valid HTTP URL",
			url:     "http://a",
			wantErr: false,
		},
		{
			name:    "Minimal valid HTTPS URL",
			url:     "https://a",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := urlutil.Validate(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("urlutil.Validate() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("urlutil.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("urlutil.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateServiceConfig(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		url         string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "No URL configured",
			serviceName: "web",
			url:         "",
			wantErr:     false,
		},
		{
			name:        "Valid URL",
			serviceName: "web",
			url:         "https://example.com",
			wantErr:     false,
		},
		{
			name:        "Invalid URL",
			serviceName: "api",
			url:         "invalid-url",
			wantErr:     true,
			errMsg:      "invalid url for service 'api'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateServiceConfig(tt.serviceName, tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateServiceConfig() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateServiceConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateServiceConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestParseAzureYaml_WithURL(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errMsg      string
		validate    func(t *testing.T, yaml *service.AzureYaml)
	}{
		{
			name: "Valid url configuration",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    url: https://myapp.example.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				if yaml == nil {
					t.Fatal("Expected non-nil AzureYaml")
				}
				web, exists := yaml.Services["web"]
				if !exists {
					t.Fatal("Expected 'web' service to exist")
				}
				if web.URL == "" {
					t.Fatal("Expected URL to be non-empty")
				}
				if web.URL != "https://myapp.example.com" {
					t.Errorf("Expected url 'https://myapp.example.com', got %s", web.URL)
				}
			},
		},
		{
			name: "Multiple services with different urls",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    url: https://web.example.com
  api:
    project: ./src/api
    language: python
    host: containerapp
    url: https://api.example.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.URL != "https://web.example.com" {
					t.Errorf("Expected web url 'https://web.example.com', got %v", web.URL)
				}
				api := yaml.Services["api"]
				if api.URL != "https://api.example.com" {
					t.Errorf("Expected api url 'https://api.example.com', got %v", api.URL)
				}
			},
		},
		{
			name: "Service without url",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.URL != "" {
					t.Errorf("Expected URL to be empty, got %v", web.URL)
				}
			},
		},
		{
			name: "Invalid url - missing protocol",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    url: example.com
`,
			wantErr: true,
			errMsg:  "invalid url for service 'web'",
		},
		{
			name: "Invalid url - wrong protocol",
			yamlContent: `name: test-app
services:
  api:
    project: ./src/api
    language: python
    host: containerapp
    url: ftp://example.com
`,
			wantErr: true,
			errMsg:  "invalid url for service 'api'",
		},
		{
			name: "Service with url using HTTP (not HTTPS)",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    url: http://localhost:8080
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.URL != "http://localhost:8080" {
					t.Errorf("Expected url 'http://localhost:8080', got %v", web.URL)
				}
			},
		},
		{
			name: "Service without url field",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.URL != "" {
					t.Errorf("Expected empty url, got %s", web.URL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary azure.yaml
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(tt.yamlContent), 0600); err != nil {
				t.Fatalf("Failed to create test azure.yaml: %v", err)
			}

			// Parse the file
			azureYaml, err := service.ParseAzureYaml(tmpDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAzureYaml() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParseAzureYaml() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("ParseAzureYaml() unexpected error = %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, azureYaml)
				}
			}
		})
	}
}

func TestParseAzureYaml_BackwardCompatibility(t *testing.T) {
	// Test that existing azure.yaml files without config still work
	yamlContent := `name: legacy-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
  api:
    project: ./src/api
    language: python
    host: containerapp
resources:
  db:
    type: postgres.database
`

	tmpDir := t.TempDir()
	azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
	if err := os.WriteFile(azureYamlPath, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test azure.yaml: %v", err)
	}

	azureYaml, err := service.ParseAzureYaml(tmpDir)
	if err != nil {
		t.Fatalf("ParseAzureYaml() failed for legacy config: %v", err)
	}

	if len(azureYaml.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(azureYaml.Services))
	}

	for name, svc := range azureYaml.Services {
		if svc.URL != "" {
			t.Errorf("Service %s should have empty URL, got %v", name, svc.URL)
		}
	}
}
