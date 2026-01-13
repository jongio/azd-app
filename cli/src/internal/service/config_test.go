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
		service     *service.Service
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "No URL configured",
			serviceName: "web",
			service:     &service.Service{},
			wantErr:     false,
		},
		{
			name:        "Valid deprecated root-level URL",
			serviceName: "web",
			service:     &service.Service{URL: "https://example.com"},
			wantErr:     false,
		},
		{
			name:        "Invalid deprecated root-level URL",
			serviceName: "api",
			service:     &service.Service{URL: "invalid-url"},
			wantErr:     true,
			errMsg:      "invalid url for service 'api'",
		},
		{
			name:        "Valid local.customUrl",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "https://myapp.ngrok.io",
				},
			},
			wantErr: false,
		},
		{
			name:        "Invalid local.customUrl",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "not-a-url",
				},
			},
			wantErr: true,
			errMsg:  "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Valid azure.customUrl",
			serviceName: "api",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomURL: "https://api.example.com",
				},
			},
			wantErr: false,
		},
		{
			name:        "Invalid azure.customUrl",
			serviceName: "api",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomURL: "ftp://invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid azure.customUrl for service 'api'",
		},
		{
			name:        "Valid azure.customDomain",
			serviceName: "web",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomDomain: "www.mycompany.com",
				},
			},
			wantErr: false,
		},
		{
			name:        "Invalid azure.customDomain with protocol",
			serviceName: "web",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomDomain: "https://www.example.com",
				},
			},
			wantErr: true,
			errMsg:  "invalid azure.customDomain for service 'web'",
		},
		{
			name:        "Multiple valid URLs",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "https://localhost.local",
				},
				Azure: &service.AzureServiceConfig{
					CustomURL:    "https://api.example.com",
					CustomDomain: "www.example.com",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateServiceConfig(tt.serviceName, tt.service)
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
			name: "Valid deprecated root-level url (backward compat)",
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
				if web.URL != "https://myapp.example.com" {
					t.Errorf("Expected deprecated URL 'https://myapp.example.com', got %s", web.URL)
				}
				// Check backward compat migration
				if web.Azure == nil || web.Azure.CustomURL != "https://myapp.example.com" {
					t.Errorf("Expected azure.customUrl to be migrated from deprecated url")
				}
			},
		},
		{
			name: "Valid local.customUrl",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: local
    local:
      customUrl: https://myapp.ngrok.io
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.Local == nil {
					t.Fatal("Expected local config to exist")
				}
				if web.Local.CustomURL != "https://myapp.ngrok.io" {
					t.Errorf("Expected local.customUrl 'https://myapp.ngrok.io', got %s", web.Local.CustomURL)
				}
			},
		},
		{
			name: "Valid azure.customUrl",
			yamlContent: `name: test-app
services:
  api:
    project: ./src/api
    language: python
    host: containerapp
    azure:
      customUrl: https://api.mycompany.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				api := yaml.Services["api"]
				if api.Azure == nil {
					t.Fatal("Expected azure config to exist")
				}
				if api.Azure.CustomURL != "https://api.mycompany.com" {
					t.Errorf("Expected azure.customUrl 'https://api.mycompany.com', got %s", api.Azure.CustomURL)
				}
			},
		},
		{
			name: "Valid azure.customDomain with user source",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    azure:
      customDomain: www.mycompany.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.Azure == nil {
					t.Fatal("Expected azure config to exist")
				}
				if web.Azure.CustomDomain != "www.mycompany.com" {
					t.Errorf("Expected azure.customDomain 'www.mycompany.com', got %s", web.Azure.CustomDomain)
				}
				if web.Azure.CustomDomainSource != "user" {
					t.Errorf("Expected customDomainSource 'user', got %s", web.Azure.CustomDomainSource)
				}
			},
		},
		{
			name: "Multiple URL fields",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    local:
      customUrl: https://localhost.local
    azure:
      customUrl: https://api.example.com
      customDomain: www.example.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.Local == nil {
					t.Fatal("Expected local config")
				}
				if web.Local.CustomURL != "https://localhost.local" {
					t.Errorf("Expected local.customUrl 'https://localhost.local', got %s", web.Local.CustomURL)
				}
				if web.Azure == nil {
					t.Fatal("Expected azure config")
				}
				if web.Azure.CustomURL != "https://api.example.com" {
					t.Errorf("Expected azure.customUrl 'https://api.example.com', got %s", web.Azure.CustomURL)
				}
				if web.Azure.CustomDomain != "www.example.com" {
					t.Errorf("Expected azure.customDomain 'www.example.com', got %s", web.Azure.CustomDomain)
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
    azure:
      customUrl: https://api.example.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.URL != "https://web.example.com" {
					t.Errorf("Expected web url 'https://web.example.com', got %v", web.URL)
				}
				api := yaml.Services["api"]
				if api.Azure == nil || api.Azure.CustomURL != "https://api.example.com" {
					t.Errorf("Expected api azure.customUrl 'https://api.example.com', got %v", api.Azure.CustomURL)
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
				if web.Local != nil && web.Local.CustomURL != "" {
					t.Errorf("Expected local.customUrl to be empty, got %v", web.Local.CustomURL)
				}
				if web.Azure != nil && web.Azure.CustomURL != "" {
					t.Errorf("Expected azure.customUrl to be empty, got %v", web.Azure.CustomURL)
				}
			},
		},
		{
			name: "Invalid local.customUrl - missing protocol",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: local
    local:
      customUrl: example.com
`,
			wantErr: true,
			errMsg:  "invalid local.customUrl for service 'web'",
		},
		{
			name: "Invalid azure.customUrl - wrong protocol",
			yamlContent: `name: test-app
services:
  api:
    project: ./src/api
    language: python
    host: containerapp
    azure:
      customUrl: ftp://example.com
`,
			wantErr: true,
			errMsg:  "invalid azure.customUrl for service 'api'",
		},
		{
			name: "Invalid azure.customDomain",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    azure:
      customDomain: not-a-url
`,
			wantErr: true,
			errMsg:  "invalid azure.customDomain for service 'web'",
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
