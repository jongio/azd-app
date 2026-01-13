package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestParseAzureYaml_URLValidation tests URL validation during parsing
func TestParseAzureYaml_URLValidation(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		errContains string
	}{
		{
			name: "Exceeds max URL length",
			yaml: `name: test
services:
  web:
    host: containerapp
    local:
      customUrl: https://example.com/` + strings.Repeat("a", 3000),
			wantErr:     true,
			errContains: "exceeds maximum length",
		},
		{
			name: "File protocol injection",
			yaml: `name: test
services:
  web:
    host: containerapp
    url: file:///etc/passwd`,
			wantErr:     true,
			errContains: "must use http:// or https://",
		},
		{
			name: "JavaScript protocol injection",
			yaml: `name: test
services:
  web:
    host: containerapp
    local:
      customUrl: javascript:alert(document.cookie)`,
			wantErr:     true,
			errContains: "must use http:// or https://",
		},
		{
			name: "Data URI injection",
			yaml: `name: test
services:
  api:
    host: containerapp
    azure:
      customUrl: data:text/html,<script>alert(1)</script>`,
			wantErr:     true,
			errContains: "must use http:// or https://",
		},
		{
			name: "Gopher protocol",
			yaml: `name: test
services:
  api:
    host: containerapp
    azure:
      customUrl: gopher://evil.com`,
			wantErr:     true,
			errContains: "must use http:// or https://",
		},
		{
			name: "FTP protocol",
			yaml: `name: test
services:
  api:
    host: containerapp
    local:
      customUrl: ftp://files.example.com`,
			wantErr:     true,
			errContains: "must use http:// or https://",
		},
		{
			name: "Valid HTTPS with max length minus 1",
			yaml: `name: test
services:
  web:
    host: containerapp
    local:
      customUrl: https://example.com/` + strings.Repeat("a", 2027),
			wantErr: false,
		},
		{
			name: "Valid HTTP localhost",
			yaml: `name: test
services:
  web:
    host: local
    local:
      customUrl: http://localhost:8080`,
			wantErr: false,
		},
		{
			name: "Valid HTTPS with query params",
			yaml: `name: test
services:
  api:
    host: containerapp
    azure:
      customUrl: https://api.example.com/v1?key=value&foo=bar`,
			wantErr: false,
		},
		{
			name: "Valid HTTPS with fragment",
			yaml: `name: test
services:
  web:
    host: containerapp
    url: https://example.com/page#section`,
			wantErr: false,
		},
		{
			name: "Valid localhost IPv4",
			yaml: `name: test
services:
  web:
    host: local
    local:
      customUrl: http://127.0.0.1:3000`,
			wantErr: false,
		},
		{
			name: "Valid IPv6 localhost",
			yaml: `name: test
services:
  web:
    host: local
    local:
      customUrl: http://[::1]:8080`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(tt.yaml), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			_, err := service.ParseAzureYaml(tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error %v does not contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestURLPrecedenceScenarios tests various URL precedence scenarios
func TestURLPrecedenceScenarios(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		validate func(t *testing.T, az *service.AzureYaml)
	}{
		{
			name: "All four URL fields present",
			yaml: `name: test
services:
  web:
    host: containerapp
    url: https://deprecated.example.com
    local:
      customUrl: https://local.example.com
    azure:
      customUrl: https://azure-url.example.com
      customDomain: azure-domain.example.com`,
			validate: func(t *testing.T, az *service.AzureYaml) {
				web := az.Services["web"]
				if web.URL != "https://deprecated.example.com" {
					t.Errorf("URL mismatch")
				}
				if web.Local == nil || web.Local.CustomURL != "https://local.example.com" {
					t.Errorf("Local.CustomURL mismatch")
				}
				if web.Azure == nil || web.Azure.CustomURL != "https://azure-url.example.com" {
					t.Errorf("Azure.CustomURL mismatch")
				}
				if web.Azure.CustomDomain != "azure-domain.example.com" {
					t.Errorf("Azure.CustomDomain mismatch")
				}
			},
		},
		{
			name: "Only deprecated URL",
			yaml: `name: test
services:
  web:
    host: containerapp
    url: https://only-deprecated.example.com`,
			validate: func(t *testing.T, az *service.AzureYaml) {
				web := az.Services["web"]
				if web.URL != "https://only-deprecated.example.com" {
					t.Errorf("URL mismatch")
				}
				// Migration should happen
				if web.Azure == nil || web.Azure.CustomURL != "https://only-deprecated.example.com" {
					t.Errorf("Expected deprecated URL to be migrated to azure.customUrl")
				}
			},
		},
		{
			name: "Only local URL",
			yaml: `name: test
services:
  web:
    host: local
    local:
      customUrl: https://only-local.example.com`,
			validate: func(t *testing.T, az *service.AzureYaml) {
				web := az.Services["web"]
				if web.Local == nil || web.Local.CustomURL != "https://only-local.example.com" {
					t.Errorf("Local.CustomURL mismatch")
				}
				if web.Azure != nil && web.Azure.CustomURL != "" {
					t.Errorf("Azure.CustomURL should be empty")
				}
			},
		},
		{
			name: "Only azure URL and domain",
			yaml: `name: test
services:
  api:
    host: containerapp
    azure:
      customUrl: https://azure.example.com
      customDomain: www.example.com`,
			validate: func(t *testing.T, az *service.AzureYaml) {
				api := az.Services["api"]
				if api.Azure == nil {
					t.Fatal("Azure config should exist")
				}
				if api.Azure.CustomURL != "https://azure.example.com" {
					t.Errorf("Azure.CustomURL mismatch")
				}
				if api.Azure.CustomDomain != "www.example.com" {
					t.Errorf("Azure.CustomDomain mismatch")
				}
			},
		},
		{
			name: "Multiple services with different URL configs",
			yaml: `name: test
services:
  web:
    host: local
    local:
      customUrl: https://local-web.example.com
  api:
    host: containerapp
    azure:
      customUrl: https://api.example.com
  admin:
    host: containerapp
    url: https://admin.example.com`,
			validate: func(t *testing.T, az *service.AzureYaml) {
				if len(az.Services) != 3 {
					t.Fatalf("Expected 3 services, got %d", len(az.Services))
				}

				web := az.Services["web"]
				if web.Local == nil || web.Local.CustomURL != "https://local-web.example.com" {
					t.Errorf("web Local.CustomURL mismatch")
				}

				api := az.Services["api"]
				if api.Azure == nil || api.Azure.CustomURL != "https://api.example.com" {
					t.Errorf("api Azure.CustomURL mismatch")
				}

				admin := az.Services["admin"]
				if admin.URL != "https://admin.example.com" {
					t.Errorf("admin URL mismatch")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			azureYamlPath := filepath.Join(tmpDir, "azure.yaml")
			if err := os.WriteFile(azureYamlPath, []byte(tt.yaml), 0600); err != nil {
				t.Fatalf("Failed to create azure.yaml: %v", err)
			}

			azureYaml, err := service.ParseAzureYaml(tmpDir)
			if err != nil {
				t.Fatalf("ParseAzureYaml() failed: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, azureYaml)
			}
		})
	}
}

// TestSecurityValidationCoverage tests security validation is comprehensive
func TestSecurityValidationCoverage(t *testing.T) {
	securityTestCases := []struct {
		name        string
		url         string
		expectError bool
		reason      string
	}{
		{"file protocol", "file:///etc/passwd", true, "file protocol should be rejected"},
		{"javascript protocol", "javascript:alert(1)", true, "javascript protocol should be rejected"},
		{"data protocol", "data:text/html,<script>", true, "data protocol should be rejected"},
		{"vbscript protocol", "vbscript:msgbox(1)", true, "vbscript protocol should be rejected"},
		{"about protocol", "about:blank", true, "about protocol should be rejected"},
		{"mailto protocol", "mailto:test@example.com", true, "mailto protocol should be rejected"},
		{"tel protocol", "tel:+1234567890", true, "tel protocol should be rejected"},
		{"ssh protocol", "ssh://user@host", true, "ssh protocol should be rejected"},
		{"git protocol", "git://github.com/repo", true, "git protocol should be rejected"},
		{"ws protocol", "ws://example.com", true, "ws protocol should be rejected"},
		{"wss protocol", "wss://example.com", true, "wss protocol should be rejected"},
		{"ftp protocol", "ftp://files.example.com", true, "ftp protocol should be rejected"},
		{"sftp protocol", "sftp://files.example.com", true, "sftp protocol should be rejected"},
		{"http valid", "http://localhost:8080", false, "http should be allowed"},
		{"https valid", "https://example.com", false, "https should be allowed"},
	}

	for _, tc := range securityTestCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: tc.url,
				},
			}
			err := service.ValidateServiceConfig("test", svc)

			if tc.expectError && err == nil {
				t.Errorf("%s: expected error but got nil", tc.reason)
			} else if !tc.expectError && err != nil {
				t.Errorf("%s: expected no error but got: %v", tc.reason, err)
			}
		})
	}
}
