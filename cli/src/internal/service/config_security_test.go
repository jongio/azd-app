package service_test

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/service"
)

// TestValidateServiceConfig_Security tests security-related validation
func TestValidateServiceConfig_Security(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		service     *service.Service
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "URL exceeds max length",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "https://example.com/" + strings.Repeat("a", 2048),
				},
			},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:        "URL with file protocol should fail",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "file:///etc/passwd",
				},
			},
			wantErr: true,
			errMsg:  "must use http:// or https://",
		},
		{
			name:        "URL with javascript protocol should fail",
			serviceName: "web",
			service: &service.Service{
				Azure: &service.AzureServiceConfig{
					CustomURL: "javascript:alert(1)",
				},
			},
			wantErr: true,
			errMsg:  "must use http:// or https://",
		},
		{
			name:        "URL with data protocol should fail",
			serviceName: "web",
			service: &service.Service{
				URL: "data:text/html,<script>alert(1)</script>",
			},
			wantErr: true,
			errMsg:  "must use http:// or https://",
		},
		{
			name:        "URL at exactly max length should pass",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					// 2048 chars total
					CustomURL: "https://example.com/" + strings.Repeat("a", 2028),
				},
			},
			wantErr: false,
		},
		{
			name:        "URL one char over max length should fail",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "https://example.com/" + strings.Repeat("a", 2029),
				},
			},
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:        "Valid localhost URL",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "http://localhost:8080",
				},
			},
			wantErr: false,
		},
		{
			name:        "Valid 127.0.0.1 URL",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "http://127.0.0.1:3000",
				},
			},
			wantErr: false,
		},
		{
			name:        "Valid IPv6 localhost URL",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "http://[::1]:8080",
				},
			},
			wantErr: false,
		},
		{
			name:        "Empty service (nil)",
			serviceName: "web",
			service:     nil,
			wantErr:     false,
		},
		{
			name:        "Service with nil Local",
			serviceName: "web",
			service: &service.Service{
				Local: nil,
			},
			wantErr: false,
		},
		{
			name:        "Service with empty Local.CustomURL",
			serviceName: "web",
			service: &service.Service{
				Local: &service.LocalServiceConfig{
					CustomURL: "",
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

// TestValidateServiceConfig_AllURLFields tests validation when all URL fields are set
func TestValidateServiceConfig_AllURLFields(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		service     *service.Service
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "All valid URLs",
			serviceName: "web",
			service: &service.Service{
				URL: "https://deprecated.example.com",
				Local: &service.LocalServiceConfig{
					CustomURL: "https://local.example.com",
				},
				Azure: &service.AzureServiceConfig{
					CustomURL:    "https://azure.example.com",
					CustomDomain: "www.example.com",
				},
			},
			wantErr: false,
		},
		{
			name:        "Invalid deprecated URL with valid others",
			serviceName: "web",
			service: &service.Service{
				URL: "not-a-url",
				Local: &service.LocalServiceConfig{
					CustomURL: "https://local.example.com",
				},
			},
			wantErr: true,
			errMsg:  "invalid url for service 'web'",
		},
		{
			name:        "Valid deprecated URL with invalid local",
			serviceName: "web",
			service: &service.Service{
				URL: "https://deprecated.example.com",
				Local: &service.LocalServiceConfig{
					CustomURL: "ftp://invalid",
				},
			},
			wantErr: true,
			errMsg:  "invalid local.customUrl for service 'web'",
		},
		{
			name:        "Valid local and deprecated URLs with invalid azure",
			serviceName: "web",
			service: &service.Service{
				URL: "https://deprecated.example.com",
				Local: &service.LocalServiceConfig{
					CustomURL: "https://local.example.com",
				},
				Azure: &service.AzureServiceConfig{
					CustomURL: "javascript:alert(1)",
				},
			},
			wantErr: true,
			errMsg:  "must use http:// or https://",
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
