package docker

import (
	"testing"
)

func TestPortMappingGetProtocol(t *testing.T) {
	tests := []struct {
		name     string
		mapping  PortMapping
		expected string
	}{
		{
			name:     "empty protocol defaults to tcp",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80},
			expected: "tcp",
		},
		{
			name:     "explicit tcp protocol",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			expected: "tcp",
		},
		{
			name:     "explicit udp protocol",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"},
			expected: "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mapping.GetProtocol()
			if got != tt.expected {
				t.Errorf("GetProtocol() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatPortMapping(t *testing.T) {
	tests := []struct {
		name     string
		mapping  PortMapping
		expected string
	}{
		{
			name:     "basic port mapping with host port",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80},
			expected: "8080:80/tcp",
		},
		{
			name:     "auto-assign host port",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 80},
			expected: "80/tcp",
		},
		{
			name:     "udp protocol",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"},
			expected: "53:53/udp",
		},
		{
			name:     "auto-assign with udp",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 53, Protocol: "udp"},
			expected: "53/udp",
		},
		{
			name:     "different host and container ports",
			mapping:  PortMapping{HostPort: 10000, ContainerPort: 10001},
			expected: "10000:10001/tcp",
		},
		{
			name:     "bind to localhost with explicit ports",
			mapping:  PortMapping{HostPort: 8080, ContainerPort: 80, BindIP: "127.0.0.1"},
			expected: "127.0.0.1:8080:80/tcp",
		},
		{
			name:     "bind to localhost with auto-assign",
			mapping:  PortMapping{HostPort: 0, ContainerPort: 80, BindIP: "127.0.0.1"},
			expected: "127.0.0.1::80/tcp",
		},
		{
			name:     "bind to specific IP with udp",
			mapping:  PortMapping{HostPort: 53, ContainerPort: 53, BindIP: "10.0.0.1", Protocol: "udp"},
			expected: "10.0.0.1:53:53/udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPortMapping(tt.mapping)
			if got != tt.expected {
				t.Errorf("formatPortMapping() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildRunArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   ContainerConfig
		expected []string
	}{
		{
			name: "basic config with image only",
			config: ContainerConfig{
				Image: "nginx",
			},
			expected: []string{"run", "-d", "nginx"},
		},
		{
			name: "config with name",
			config: ContainerConfig{
				Name:  "my-container",
				Image: "nginx",
			},
			expected: []string{"run", "-d", "--name", "my-container", "nginx"},
		},
		{
			name: "config with single port",
			config: ContainerConfig{
				Image: "nginx",
				Ports: []PortMapping{
					{HostPort: 8080, ContainerPort: 80},
				},
			},
			expected: []string{"run", "-d", "-p", "8080:80/tcp", "nginx"},
		},
		{
			name: "config with multiple ports",
			config: ContainerConfig{
				Image: "azurite",
				Ports: []PortMapping{
					{HostPort: 10000, ContainerPort: 10000},
					{HostPort: 10001, ContainerPort: 10001},
					{HostPort: 10002, ContainerPort: 10002},
				},
			},
			expected: []string{"run", "-d", "-p", "10000:10000/tcp", "-p", "10001:10001/tcp", "-p", "10002:10002/tcp", "azurite"},
		},
		{
			name: "config with all options",
			config: ContainerConfig{
				Name:  "test-container",
				Image: "mcr.microsoft.com/azure-storage/azurite",
				Ports: []PortMapping{
					{HostPort: 10000, ContainerPort: 10000},
				},
				Environment: map[string]string{
					"DEBUG": "true",
				},
			},
			// Note: environment variables are added from a map, so order is not guaranteed
			// We'll check for presence instead of exact order in a separate test
			expected: nil, // Will check manually
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRunArgs(tt.config)

			if tt.expected == nil {
				// For the "all options" test, check for presence of key elements
				assertContains(t, got, "run")
				assertContains(t, got, "-d")
				assertContains(t, got, "--name")
				assertContains(t, got, "test-container")
				assertContains(t, got, "-p")
				assertContains(t, got, "10000:10000/tcp")
				assertContains(t, got, "-e")
				assertContains(t, got, "DEBUG=true")
				assertContains(t, got, "mcr.microsoft.com/azure-storage/azurite")

				// Image should be last
				if got[len(got)-1] != "mcr.microsoft.com/azure-storage/azurite" {
					t.Errorf("Image should be the last argument, got: %v", got)
				}
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("buildRunArgs() returned %d args, want %d. Got: %v", len(got), len(tt.expected), got)
				return
			}

			for i, arg := range tt.expected {
				if got[i] != arg {
					t.Errorf("buildRunArgs()[%d] = %q, want %q", i, got[i], arg)
				}
			}
		})
	}
}

func TestBuildRunArgsEnvironmentVariables(t *testing.T) {
	config := ContainerConfig{
		Image: "test",
		Environment: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value with spaces",
		},
	}

	got := buildRunArgs(config)

	// Count environment flags
	envCount := 0
	for i, arg := range got {
		if arg == "-e" && i+1 < len(got) {
			envCount++
		}
	}

	if envCount != 3 {
		t.Errorf("Expected 3 environment variables, got %d. Args: %v", envCount, got)
	}

	// Check that all env vars are present
	assertContains(t, got, "VAR1=value1")
	assertContains(t, got, "VAR2=value2")
	assertContains(t, got, "VAR3=value with spaces")
}

func TestContainerConfigValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty image returns error", func(t *testing.T) {
		_, err := client.Run(ContainerConfig{})
		if err == nil {
			t.Error("Expected error for empty image, got nil")
		}
	})
}

func TestStopValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		err := client.Stop("", 10)
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestRemoveValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		err := client.Remove("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestLogsValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		_, err := client.Logs("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestInspectValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns error", func(t *testing.T) {
		_, err := client.Inspect("")
		if err == nil {
			t.Error("Expected error for empty container ID, got nil")
		}
	})
}

func TestIsRunningValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty container ID returns false", func(t *testing.T) {
		if client.IsRunning("") {
			t.Error("Expected false for empty container ID")
		}
	})
}

func TestPullValidation(t *testing.T) {
	client := NewClient()

	t.Run("empty image returns error", func(t *testing.T) {
		err := client.Pull("")
		if err == nil {
			t.Error("Expected error for empty image, got nil")
		}
	})
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Error("NewClient() returned nil")
	}
}

func TestValidateImageName(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		wantErr bool
	}{
		{"valid simple image", "nginx", false},
		{"valid image with tag", "nginx:latest", false},
		{"valid image with registry", "docker.io/library/nginx", false},
		{"valid mcr image", "mcr.microsoft.com/azure-storage/azurite", false},
		{"valid image with tag and registry", "mcr.microsoft.com/azure-storage/azurite:latest", false},
		{"valid image with digest", "nginx@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"empty image", "", true},
		{"image with spaces", "nginx latest", true},
		{"image with special chars", "nginx;rm -rf /", true},
		{"image too long", string(make([]byte, 300)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageName(tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageName(%q) error = %v, wantErr %v", tt.image, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		wantErr       bool
	}{
		{"valid name", "my-container", false},
		{"valid name with underscore", "my_container", false},
		{"valid name with dot", "my.container", false},
		{"empty name (valid)", "", false},
		{"starts with number", "1container", false},
		{"starts with hyphen", "-container", true},
		{"contains spaces", "my container", true},
		{"too long", string(make([]byte, 200)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerName(tt.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContainerName(%q) error = %v, wantErr %v", tt.containerName, err, tt.wantErr)
			}
		})
	}
}

func TestPortMappingValidate(t *testing.T) {
	tests := []struct {
		name    string
		mapping PortMapping
		wantErr bool
	}{
		{"valid mapping", PortMapping{HostPort: 8080, ContainerPort: 80}, false},
		{"auto-assign host port", PortMapping{HostPort: 0, ContainerPort: 80}, false},
		{"invalid host port", PortMapping{HostPort: 70000, ContainerPort: 80}, true},
		{"invalid container port", PortMapping{HostPort: 8080, ContainerPort: 0}, true},
		{"container port too high", PortMapping{HostPort: 8080, ContainerPort: 70000}, true},
		{"invalid protocol", PortMapping{HostPort: 8080, ContainerPort: 80, Protocol: "http"}, true},
		{"valid udp", PortMapping{HostPort: 53, ContainerPort: 53, Protocol: "udp"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mapping.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PortMapping.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  ContainerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  ContainerConfig{Image: "nginx", Name: "my-nginx"},
			wantErr: false,
		},
		{
			name:    "valid config without name",
			config:  ContainerConfig{Image: "nginx"},
			wantErr: false,
		},
		{
			name:    "invalid image",
			config:  ContainerConfig{Image: ""},
			wantErr: true,
		},
		{
			name:    "invalid container name",
			config:  ContainerConfig{Image: "nginx", Name: "-invalid"},
			wantErr: true,
		},
		{
			name: "invalid port mapping",
			config: ContainerConfig{
				Image: "nginx",
				Ports: []PortMapping{{HostPort: 8080, ContainerPort: 0}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ContainerConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// assertContains checks if a slice contains a specific string.
func assertContains(t *testing.T, slice []string, want string) {
	t.Helper()
	for _, s := range slice {
		if s == want {
			return
		}
	}
	t.Errorf("Slice does not contain %q. Got: %v", want, slice)
}
