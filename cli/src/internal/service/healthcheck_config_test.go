// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHealthcheckConfig_IsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *HealthcheckConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config",
			config:   &HealthcheckConfig{},
			expected: false,
		},
		{
			name:     "disable true",
			config:   &HealthcheckConfig{Disable: true},
			expected: true,
		},
		{
			name:     "type none",
			config:   &HealthcheckConfig{Type: "none"},
			expected: true,
		},
		{
			name:     "test NONE array",
			config:   &HealthcheckConfig{Test: []interface{}{"NONE"}},
			expected: true,
		},
		{
			name:     "test NONE string array",
			config:   &HealthcheckConfig{Test: []string{"NONE"}},
			expected: true,
		},
		{
			name:     "test http URL",
			config:   &HealthcheckConfig{Test: "http://localhost:8080/health"},
			expected: false,
		},
		{
			name:     "type http",
			config:   &HealthcheckConfig{Type: "http"},
			expected: false,
		},
		{
			name:     "type process",
			config:   &HealthcheckConfig{Type: "process"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsDisabled()
			if result != tt.expected {
				t.Errorf("IsDisabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHealthcheckConfig_GetType(t *testing.T) {
	tests := []struct {
		name     string
		config   *HealthcheckConfig
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "http",
		},
		{
			name:     "empty config",
			config:   &HealthcheckConfig{},
			expected: "http",
		},
		{
			name:     "type http",
			config:   &HealthcheckConfig{Type: "http"},
			expected: "http",
		},
		{
			name:     "type process",
			config:   &HealthcheckConfig{Type: "process"},
			expected: "process",
		},
		{
			name:     "type none",
			config:   &HealthcheckConfig{Type: "none"},
			expected: "none",
		},
		{
			name:     "type output",
			config:   &HealthcheckConfig{Type: "output", Pattern: "Found 0 errors"},
			expected: "output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetType()
			if result != tt.expected {
				t.Errorf("GetType() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestService_IsHealthcheckDisabled(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected bool
	}{
		{
			name:     "no healthcheck",
			service:  Service{},
			expected: false,
		},
		{
			name: "healthcheck disable true",
			service: Service{
				Healthcheck: &HealthcheckConfig{Disable: true},
			},
			expected: true,
		},
		{
			name: "healthcheck type none",
			service: Service{
				Healthcheck: &HealthcheckConfig{Type: "none"},
			},
			expected: true,
		},
		{
			name: "healthcheck type http",
			service: Service{
				Healthcheck: &HealthcheckConfig{Type: "http", Path: "/health"},
			},
			expected: false,
		},
		{
			name: "healthcheck enabled explicitly disabled",
			service: Service{
				HealthcheckEnabled: boolPtr(false),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.service.IsHealthcheckDisabled()
			if result != tt.expected {
				t.Errorf("IsHealthcheckDisabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestService_NeedsPort(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected bool
	}{
		{
			name:     "no ports no healthcheck",
			service:  Service{},
			expected: false, // No ports defined = no port needed (uses process health check)
		},
		{
			name: "explicit ports",
			service: Service{
				Ports: []string{"3000"},
			},
			expected: true,
		},
		{
			name: "healthcheck disabled no ports",
			service: Service{
				Healthcheck: &HealthcheckConfig{Disable: true},
			},
			expected: false, // No ports defined = no port needed
		},
		{
			name: "healthcheck disabled with ports",
			service: Service{
				Ports:       []string{"3000"},
				Healthcheck: &HealthcheckConfig{Disable: true},
			},
			expected: true, // Explicit ports take precedence
		},
		{
			name: "healthcheck type none",
			service: Service{
				Healthcheck: &HealthcheckConfig{Type: "none"},
			},
			expected: false, // No ports defined = no port needed
		},
		{
			name: "healthcheck type process",
			service: Service{
				Healthcheck: &HealthcheckConfig{Type: "process"},
			},
			expected: false, // No ports defined = no port needed (uses process health check)
		},
		{
			name: "healthcheck type http with ports",
			service: Service{
				Ports:       []string{"8080"},
				Healthcheck: &HealthcheckConfig{Type: "http"},
			},
			expected: true, // Explicit ports = port needed
		},
		{
			name: "multiple ports",
			service: Service{
				Ports: []string{"3000", "3001"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.service.NeedsPort()
			if result != tt.expected {
				t.Errorf("NeedsPort() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestService_UnmarshalYAML_HealthcheckBoolean(t *testing.T) {
	tests := []struct {
		name                      string
		yamlContent               string
		expectHealthcheckDisabled bool
	}{
		{
			name: "healthcheck false",
			yamlContent: `
project: ./api
healthcheck: false
`,
			expectHealthcheckDisabled: true,
		},
		{
			name: "healthcheck true",
			yamlContent: `
project: ./api
healthcheck: true
`,
			expectHealthcheckDisabled: false,
		},
		{
			name: "healthcheck object with disable",
			yamlContent: `
project: ./api
healthcheck:
  disable: true
`,
			expectHealthcheckDisabled: true,
		},
		{
			name: "healthcheck object with type none",
			yamlContent: `
project: ./api
healthcheck:
  type: none
`,
			expectHealthcheckDisabled: true,
		},
		{
			name: "healthcheck object with type http",
			yamlContent: `
project: ./api
healthcheck:
  type: http
  path: /health
`,
			expectHealthcheckDisabled: false,
		},
		{
			name: "no healthcheck",
			yamlContent: `
project: ./api
`,
			expectHealthcheckDisabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var service Service
			err := yaml.Unmarshal([]byte(tt.yamlContent), &service)
			if err != nil {
				t.Fatalf("Failed to unmarshal YAML: %v", err)
			}

			result := service.IsHealthcheckDisabled()
			if result != tt.expectHealthcheckDisabled {
				t.Errorf("IsHealthcheckDisabled() = %v, expected %v", result, tt.expectHealthcheckDisabled)
			}
		})
	}
}

func TestPerformHealthCheck_NoneType(t *testing.T) {
	process := &ServiceProcess{
		Runtime: ServiceRuntime{
			HealthCheck: HealthCheckConfig{
				Type: "none",
			},
		},
	}

	err := PerformHealthCheck(process)
	if err != nil {
		t.Errorf("PerformHealthCheck() returned error for type 'none': %v", err)
	}

	if !process.Ready {
		t.Error("PerformHealthCheck() did not set process.Ready to true for type 'none'")
	}
}
