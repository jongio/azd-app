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
			errMsg:  "url must use http:// or https://",
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
			service:     &service.Service{Host: "containerapp"},
			wantErr:     false,
		},
		{
			name:        "Valid local customUrl - HTTPS",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "https://abc.ngrok-free.app"}},
			wantErr:     false,
		},
		{
			name:        "Valid local customUrl - HTTP localhost",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "http://localhost:8080"}},
			wantErr:     false,
		},
		{
			name:        "Valid local customUrl - HTTP with domain (allowed for local)",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "http://example.com"}},
			wantErr:     false,
		},
		{
			name:        "Valid azure customUrl - HTTPS",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomURL: "https://api.example.com"}},
			wantErr:     false,
		},
		{
			name:        "Valid azure customUrl - HTTP localhost",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomURL: "http://localhost:3000"}},
			wantErr:     false,
		},
		{
			name:        "Valid azure customDomain - HTTPS",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomDomain: "https://myapp.example.com"}},
			wantErr:     false,
		},
		{
			name:        "Invalid local customUrl - no protocol",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "invalid-url"}},
			wantErr:     true,
			errMsg:      "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Invalid local customUrl - wrong protocol",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "ftp://example.com"}},
			wantErr:     true,
			errMsg:      "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Invalid local customUrl - protocol injection",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "javascript:alert(1)"}},
			wantErr:     true,
			errMsg:      "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Invalid local customUrl - no host",
			serviceName: "web",
			service:     &service.Service{Host: "containerapp", Local: &service.LocalConfig{CustomURL: "http://"}},
			wantErr:     true,
			errMsg:      "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Invalid azure customUrl - HTTP with non-localhost",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomURL: "http://example.com"}},
			wantErr:     true,
			errMsg:      "invalid azure.customUrl for service 'api'",
		},
		{
			name:        "Invalid azure customUrl - wrong protocol",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomURL: "ftp://example.com"}},
			wantErr:     true,
			errMsg:      "invalid azure.customUrl for service 'api'",
		},
		{
			name:        "Invalid azure customDomain - HTTP with non-localhost",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomDomain: "http://example.com"}},
			wantErr:     true,
			errMsg:      "invalid azure.customDomain for service 'api'",
		},
		{
			name:        "Invalid azure customDomain - wrong protocol",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomDomain: "ftp://example.com"}},
			wantErr:     true,
			errMsg:      "invalid azure.customDomain for service 'api'",
		},
		{
			name:        "Invalid azure customDomain - protocol injection",
			serviceName: "api",
			service:     &service.Service{Host: "containerapp", Azure: &service.AzureConfig{CustomDomain: "javascript:alert(1)"}},
			wantErr:     true,
			errMsg:      "invalid azure.customDomain for service 'api'",
		},
		{
			name:        "Multiple URLs - all valid",
			serviceName: "web",
			service: &service.Service{
				Host:  "containerapp",
				Local: &service.LocalConfig{CustomURL: "https://abc.ngrok.io"},
				Azure: &service.AzureConfig{CustomURL: "https://cdn.example.com", CustomDomain: "https://myapp.example.com"},
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
			name: "Valid local.customUrl configuration",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    local:
      customUrl: https://abc.ngrok-free.app
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
				if web.Local == nil {
					t.Fatal("Expected Local config to exist")
				}
				if web.Local.CustomURL != "https://abc.ngrok-free.app" {
					t.Errorf("Expected local.customUrl 'https://abc.ngrok-free.app', got %s", web.Local.CustomURL)
				}
			},
		},
		{
			name: "Valid azure.customUrl configuration",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    azure:
      customUrl: https://myapp.example.com
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
				if web.Azure == nil {
					t.Fatal("Expected Azure config to exist")
				}
				if web.Azure.CustomURL != "https://myapp.example.com" {
					t.Errorf("Expected azure.customUrl 'https://myapp.example.com', got %s", web.Azure.CustomURL)
				}
			},
		},
		{
			name: "Multiple services with different customUrls",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    local:
      customUrl: https://web.ngrok.io
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
				if web.Local == nil || web.Local.CustomURL != "https://web.ngrok.io" {
					t.Errorf("Expected web local.customUrl 'https://web.ngrok.io'")
				}
				api := yaml.Services["api"]
				if api.Azure == nil || api.Azure.CustomURL != "https://api.example.com" {
					t.Errorf("Expected api azure.customUrl 'https://api.example.com'")
				}
			},
		},
		{
			name: "Service without customUrl",
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
				if web.Local != nil && web.Local.CustomURL != "" {
					t.Errorf("Expected local.customUrl to be empty, got %v", web.Local.CustomURL)
				}
				if web.Azure != nil && web.Azure.CustomURL != "" {
					t.Errorf("Expected azure.customUrl to be empty, got %v", web.Azure.CustomURL)
				}
			},
		},
		{
			name: "Legacy url field should error",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    url: https://myapp.example.com
`,
			wantErr: true,
			errMsg:  "legacy 'url' field is no longer supported",
		},
		{
			name: "Invalid local.customUrl - missing protocol",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
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
			name: "Service with customUrl using HTTP (not HTTPS)",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    local:
      customUrl: http://localhost:8080
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.Local == nil || web.Local.CustomURL != "http://localhost:8080" {
					t.Errorf("Expected local.customUrl 'http://localhost:8080'")
				}
			},
		},
		{
			name: "Service with azure.customDomain",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    azure:
      customDomain: https://myapp.example.com
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				web := yaml.Services["web"]
				if web.Azure == nil || web.Azure.CustomDomain != "https://myapp.example.com" {
					t.Errorf("Expected azure.customDomain 'https://myapp.example.com'")
				}
			},
		},
		{
			name: "Invalid azure.customUrl - HTTP non-localhost",
			yamlContent: `name: test-app
services:
  api:
    project: ./src/api
    language: python
    host: containerapp
    azure:
      customUrl: http://api.example.com
`,
			wantErr: true,
			errMsg:  "invalid azure.customUrl for service 'api'",
		},
		{
			name: "Invalid azure.customDomain - HTTP non-localhost",
			yamlContent: `name: test-app
services:
  web:
    project: ./src/web
    language: js
    host: containerapp
    azure:
      customDomain: http://myapp.example.com
`,
			wantErr: true,
			errMsg:  "invalid azure.customDomain for service 'web'",
		},
		{
			name: "Valid azure.customUrl - HTTP localhost",
			yamlContent: `name: test-app
services:
  api:
    project: ./src/api
    language: python
    host: containerapp
    azure:
      customUrl: http://localhost:3000
`,
			wantErr: false,
			validate: func(t *testing.T, yaml *service.AzureYaml) {
				api := yaml.Services["api"]
				if api.Azure == nil || api.Azure.CustomURL != "http://localhost:3000" {
					t.Errorf("Expected azure.customUrl 'http://localhost:3000'")
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
