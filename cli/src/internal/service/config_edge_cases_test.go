package service_test

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestNilPointerSafety tests nil pointer handling across the service package
func TestNilPointerSafety(t *testing.T) {
	t.Run("ValidateServiceConfig with nil service", func(t *testing.T) {
		err := service.ValidateServiceConfig("test", nil)
		if err != nil {
			t.Errorf("Expected nil error for nil service, got %v", err)
		}
	})

	t.Run("ValidateServiceConfig with nil Local", func(t *testing.T) {
		svc := &service.Service{
			Local: nil,
		}
		err := service.ValidateServiceConfig("test", svc)
		if err != nil {
			t.Errorf("Expected nil error for nil Local, got %v", err)
		}
	})

	t.Run("ValidateServiceConfig with nil Azure", func(t *testing.T) {
		svc := &service.Service{
			Azure: nil,
		}
		err := service.ValidateServiceConfig("test", svc)
		if err != nil {
			t.Errorf("Expected nil error for nil Azure, got %v", err)
		}
	})

	t.Run("ValidateServiceConfig with empty strings", func(t *testing.T) {
		svc := &service.Service{
			URL: "",
			Local: &service.LocalServiceConfig{
				CustomURL: "",
			},
			Azure: &service.AzureServiceConfig{
				CustomURL:    "",
				CustomDomain: "",
			},
		}
		err := service.ValidateServiceConfig("test", svc)
		if err != nil {
			t.Errorf("Expected nil error for empty strings, got %v", err)
		}
	})
}

// TestServiceDefaults tests default values and initialization
func TestServiceDefaults(t *testing.T) {
	t.Run("Empty service has expected zero values", func(t *testing.T) {
		svc := &service.Service{}
		if svc.Host != "" {
			t.Errorf("Expected empty Host, got %v", svc.Host)
		}
		if svc.Language != "" {
			t.Errorf("Expected empty Language, got %v", svc.Language)
		}
		if svc.Local != nil {
			t.Errorf("Expected nil Local, got %v", svc.Local)
		}
		if svc.Azure != nil {
			t.Errorf("Expected nil Azure, got %v", svc.Azure)
		}
	})
}

// TestURLPrecedence tests URL precedence across different configuration fields
func TestURLPrecedence(t *testing.T) {
	t.Run("All four URLs set", func(t *testing.T) {
		svc := &service.Service{
			URL: "https://deprecated.example.com",
			Local: &service.LocalServiceConfig{
				CustomURL: "https://local.example.com",
			},
			Azure: &service.AzureServiceConfig{
				CustomURL:    "https://azure-url.example.com",
				CustomDomain: "azure-domain.example.com",
			},
		}

		// Validate all URLs are valid
		err := service.ValidateServiceConfig("test", svc)
		if err != nil {
			t.Errorf("Expected no error when all URLs valid, got %v", err)
		}

		// Verify all fields retained their values
		if svc.URL != "https://deprecated.example.com" {
			t.Errorf("URL changed unexpectedly")
		}
		if svc.Local.CustomURL != "https://local.example.com" {
			t.Errorf("Local.CustomURL changed unexpectedly")
		}
		if svc.Azure.CustomURL != "https://azure-url.example.com" {
			t.Errorf("Azure.CustomURL changed unexpectedly")
		}
		if svc.Azure.CustomDomain != "azure-domain.example.com" {
			t.Errorf("Azure.CustomDomain changed unexpectedly")
		}
	})
}

// TestValidationErrorMessages tests that error messages are clear and helpful
func TestValidationErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		service     *service.Service
		wantErrMsg  string
	}{
		{
			name:        "Invalid URL includes service name",
			serviceName: "my-service",
			service: &service.Service{
				URL: "not-valid",
			},
			wantErrMsg: "my-service",
		},
		{
			name:        "Invalid local URL includes field name",
			serviceName: "api",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "bad-url",
				},
			},
			wantErrMsg: "local.customUrl",
		},
		{
			name:        "Invalid azure URL includes field name",
			serviceName: "web",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomURL: "invalid",
				},
			},
			wantErrMsg: "azure.customUrl",
		},
		{
			name:        "Invalid azure domain includes field name",
			serviceName: "web",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomDomain: "https://not-just-domain.com",
				},
			},
			wantErrMsg: "azure.customDomain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateServiceConfig(tt.serviceName, tt.service)
			if err == nil {
				t.Fatal("Expected error but got nil")
			}
			if tt.wantErrMsg != "" {
				errMsg := err.Error()
				if len(errMsg) == 0 || !strings.Contains(errMsg, tt.wantErrMsg) {
					t.Errorf("Error message = %q, want containing %q", errMsg, tt.wantErrMsg)
				}
			}
		})
	}
}

// TestEdgeCaseURLs tests edge cases for URL validation
func TestEdgeCaseURLs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "URL with query parameters",
			url:     "https://example.com?param=value",
			wantErr: false,
		},
		{
			name:    "URL with fragment",
			url:     "https://example.com#section",
			wantErr: false,
		},
		{
			name:    "URL with username and password",
			url:     "https://user:pass@example.com",
			wantErr: false,
		},
		{
			name:    "URL with unusual but valid port",
			url:     "https://example.com:65535",
			wantErr: false,
		},
		{
			name:    "URL with subdomain",
			url:     "https://api.v2.example.com",
			wantErr: false,
		},
		{
			name:    "URL with hyphenated domain",
			url:     "https://my-api-server.example.com",
			wantErr: false,
		},
		{
			name:    "Localhost with port",
			url:     "http://localhost:3000",
			wantErr: false,
		},
		{
			name:    "127.0.0.1 with port",
			url:     "http://127.0.0.1:8080",
			wantErr: false,
		},
		{
			name:    "IPv6 localhost",
			url:     "http://[::1]:8080",
			wantErr: false,
		},
		{
			name:    "IPv6 address",
			url:     "http://[2001:db8::1]:8080",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: tt.url,
				},
			}
			err := service.ValidateServiceConfig("test", svc)
			if tt.wantErr && err == nil {
				t.Error("Expected error but got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got %v", err)
			}
		})
	}
}
