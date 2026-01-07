package service

import (
	"strings"
	"testing"

	"github.com/jongio/azd-app/cli/src/internal/docker"
)

func TestValidateServiceNameForContainer(t *testing.T) {
	tests := []struct {
		name    string
		svcName string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple name",
			svcName: "api",
			wantErr: false,
		},
		{
			name:    "valid name with hyphens",
			svcName: "api-service",
			wantErr: false,
		},
		{
			name:    "valid name with underscores",
			svcName: "api_service",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			svcName: "api2",
			wantErr: false,
		},
		{
			name:    "valid complex name",
			svcName: "MyAPI_Service-2",
			wantErr: false,
		},
		{
			name:    "empty name",
			svcName: "",
			wantErr: true,
			errMsg:  "service name cannot be empty",
		},
		{
			name:    "name too long",
			svcName: strings.Repeat("a", 65),
			wantErr: true,
			errMsg:  "service name too long",
		},
		{
			name:    "name starts with number",
			svcName: "2api",
			wantErr: true,
			errMsg:  "must start with letter",
		},
		{
			name:    "name starts with hyphen",
			svcName: "-api",
			wantErr: true,
			errMsg:  "must start with letter",
		},
		{
			name:    "name starts with underscore",
			svcName: "_api",
			wantErr: true,
			errMsg:  "must start with letter",
		},
		{
			name:    "name contains invalid characters",
			svcName: "api@service",
			wantErr: true,
			errMsg:  "must start with letter",
		},
		{
			name:    "name contains spaces",
			svcName: "api service",
			wantErr: true,
			errMsg:  "must start with letter",
		},
		{
			name:    "name contains dots",
			svcName: "api.service",
			wantErr: true,
			errMsg:  "must start with letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceNameForContainer(tt.svcName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateServiceNameForContainer() expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateServiceNameForContainer() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateServiceNameForContainer() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestBuildContainerPortMappings(t *testing.T) {
	tests := []struct {
		name     string
		runtime  *ServiceRuntime
		expected []docker.PortMapping
	}{
		{
			name: "runtime with port",
			runtime: &ServiceRuntime{
				Name: "test",
				Port: 8080,
			},
			expected: []docker.PortMapping{
				{
					HostPort:      8080,
					ContainerPort: 8080,
					Protocol:      "tcp",
				},
			},
		},
		{
			name: "runtime with no port",
			runtime: &ServiceRuntime{
				Name: "test",
				Port: 0,
			},
			expected: []docker.PortMapping{},
		},
		{
			name: "runtime with different port",
			runtime: &ServiceRuntime{
				Name: "test",
				Port: 3000,
			},
			expected: []docker.PortMapping{
				{
					HostPort:      3000,
					ContainerPort: 3000,
					Protocol:      "tcp",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildContainerPortMappings(tt.runtime)

			if len(result) != len(tt.expected) {
				t.Errorf("buildContainerPortMappings() returned %d mappings, want %d", len(result), len(tt.expected))
				return
			}

			for i, mapping := range result {
				expected := tt.expected[i]
				if mapping.HostPort != expected.HostPort {
					t.Errorf("mapping[%d].HostPort = %v, want %v", i, mapping.HostPort, expected.HostPort)
				}
				if mapping.ContainerPort != expected.ContainerPort {
					t.Errorf("mapping[%d].ContainerPort = %v, want %v", i, mapping.ContainerPort, expected.ContainerPort)
				}
				if mapping.Protocol != expected.Protocol {
					t.Errorf("mapping[%d].Protocol = %v, want %v", i, mapping.Protocol, expected.Protocol)
				}
			}
		})
	}
}
