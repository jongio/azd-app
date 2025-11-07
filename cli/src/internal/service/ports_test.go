package service

import (
	"testing"
)

func TestParsePortSpec(t *testing.T) {
	tests := []struct {
		name          string
		spec          string
		isDocker      bool
		wantHost      int
		wantContainer int
		wantBindIP    string
		wantProtocol  string
	}{
		{
			name:          "single port - non-Docker",
			spec:          "8080",
			isDocker:      false,
			wantHost:      8080,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "single port - Docker (auto-assign host)",
			spec:          "8080",
			isDocker:      true,
			wantHost:      0, // 0 = auto-assign
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "host:container mapping",
			spec:          "3000:8080",
			isDocker:      false,
			wantHost:      3000,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "host:container mapping - Docker",
			spec:          "3000:8080",
			isDocker:      true,
			wantHost:      3000,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "ip:host:container mapping",
			spec:          "127.0.0.1:3000:8080",
			isDocker:      false,
			wantBindIP:    "127.0.0.1",
			wantHost:      3000,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "UDP protocol",
			spec:          "53:53/udp",
			isDocker:      false,
			wantHost:      53,
			wantContainer: 53,
			wantProtocol:  "udp",
		},
		{
			name:          "single port with UDP",
			spec:          "8080/udp",
			isDocker:      false,
			wantHost:      8080,
			wantContainer: 8080,
			wantProtocol:  "udp",
		},
		{
			name:          "whitespace trimmed",
			spec:          "  8080  ",
			isDocker:      false,
			wantHost:      8080,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "localhost binding",
			spec:          "localhost:3000:8080",
			isDocker:      false,
			wantBindIP:    "localhost",
			wantHost:      3000,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
		{
			name:          "IPv6 binding",
			spec:          "::1:3000:8080",
			isDocker:      true,
			wantBindIP:    "::1",
			wantHost:      3000,
			wantContainer: 8080,
			wantProtocol:  "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := ParsePortSpec(tt.spec, tt.isDocker)

			if mapping.HostPort != tt.wantHost {
				t.Errorf("HostPort = %d, want %d", mapping.HostPort, tt.wantHost)
			}
			if mapping.ContainerPort != tt.wantContainer {
				t.Errorf("ContainerPort = %d, want %d", mapping.ContainerPort, tt.wantContainer)
			}
			if mapping.BindIP != tt.wantBindIP {
				t.Errorf("BindIP = %q, want %q", mapping.BindIP, tt.wantBindIP)
			}
			if mapping.Protocol != tt.wantProtocol {
				t.Errorf("Protocol = %q, want %q", mapping.Protocol, tt.wantProtocol)
			}
		})
	}
}

func TestServiceGetPortMappings(t *testing.T) {
	tests := []struct {
		name           string
		service        Service
		wantMappings   int // number of mappings
		wantIsExplicit bool
		wantFirstHost  int
	}{
		{
			name: "single port - non-Docker",
			service: Service{
				Language: "node",
				Ports:    []string{"8080"},
			},
			wantMappings:   1,
			wantIsExplicit: true,
			wantFirstHost:  8080,
		},
		{
			name: "single port - Docker (auto-host)",
			service: Service{
				Language: "python",
				Docker:   &DockerConfig{Image: "python:3.9"},
				Ports:    []string{"8080"},
			},
			wantMappings:   1,
			wantIsExplicit: false, // host=0 means not explicit
			wantFirstHost:  0,
		},
		{
			name: "explicit mapping",
			service: Service{
				Language: "python",
				Docker:   &DockerConfig{Image: "python:3.9"},
				Ports:    []string{"3000:8080"},
			},
			wantMappings:   1,
			wantIsExplicit: true,
			wantFirstHost:  3000,
		},
		{
			name: "multiple ports",
			service: Service{
				Language: "go",
				Docker:   &DockerConfig{Image: "golang:1.21"},
				Ports:    []string{"3000:8080", "9229:9229"},
			},
			wantMappings:   2,
			wantIsExplicit: true,
			wantFirstHost:  3000,
		},
		{
			name: "no ports",
			service: Service{
				Language: "node",
				Ports:    []string{},
			},
			wantMappings:   0,
			wantIsExplicit: false,
			wantFirstHost:  0,
		},
		{
			name: "mixed explicit and auto",
			service: Service{
				Language: "node",
				Docker:   &DockerConfig{Image: "node:20"},
				Ports:    []string{"3000:8080", "9229"}, // first is explicit
			},
			wantMappings:   2,
			wantIsExplicit: true, // at least one is explicit
			wantFirstHost:  3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings, isExplicit := tt.service.GetPortMappings()

			if len(mappings) != tt.wantMappings {
				t.Errorf("GetPortMappings() returned %d mappings, want %d", len(mappings), tt.wantMappings)
			}

			if isExplicit != tt.wantIsExplicit {
				t.Errorf("isExplicit = %v, want %v", isExplicit, tt.wantIsExplicit)
			}

			if len(mappings) > 0 && mappings[0].HostPort != tt.wantFirstHost {
				t.Errorf("first mapping HostPort = %d, want %d", mappings[0].HostPort, tt.wantFirstHost)
			}
		})
	}
}

func TestServiceGetPrimaryPort(t *testing.T) {
	tests := []struct {
		name           string
		service        Service
		wantHost       int
		wantContainer  int
		wantIsExplicit bool
	}{
		{
			name: "single port",
			service: Service{
				Language: "node",
				Ports:    []string{"8080"},
			},
			wantHost:       8080,
			wantContainer:  8080,
			wantIsExplicit: true,
		},
		{
			name: "host:container mapping",
			service: Service{
				Language: "python",
				Docker:   &DockerConfig{Image: "python:3.9"},
				Ports:    []string{"3000:8080"},
			},
			wantHost:       3000,
			wantContainer:  8080,
			wantIsExplicit: true,
		},
		{
			name: "multiple ports - returns first",
			service: Service{
				Language: "go",
				Docker:   &DockerConfig{Image: "golang:1.21"},
				Ports:    []string{"3000:8080", "9229:9229"},
			},
			wantHost:       3000,
			wantContainer:  8080,
			wantIsExplicit: true,
		},
		{
			name: "no ports",
			service: Service{
				Language: "node",
				Ports:    []string{},
			},
			wantHost:       0,
			wantContainer:  0,
			wantIsExplicit: false,
		},
		{
			name: "Docker auto-assign",
			service: Service{
				Language: "python",
				Docker:   &DockerConfig{Image: "python:3.9"},
				Ports:    []string{"8080"}, // Container only
			},
			wantHost:       0, // Auto-assign
			wantContainer:  8080,
			wantIsExplicit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, container, isExplicit := tt.service.GetPrimaryPort()

			if host != tt.wantHost {
				t.Errorf("host = %d, want %d", host, tt.wantHost)
			}
			if container != tt.wantContainer {
				t.Errorf("container = %d, want %d", container, tt.wantContainer)
			}
			if isExplicit != tt.wantIsExplicit {
				t.Errorf("isExplicit = %v, want %v", isExplicit, tt.wantIsExplicit)
			}
		})
	}
}

func TestDetectPortWithPortsArray(t *testing.T) {
	tests := []struct {
		name           string
		service        Service
		wantPort       int
		wantIsExplicit bool
		wantError      bool
	}{
		{
			name: "explicit port from ports array",
			service: Service{
				Language: "node",
				Ports:    []string{"3000"},
			},
			wantPort:       3000,
			wantIsExplicit: true,
			wantError:      false,
		},
		{
			name: "explicit host:container mapping",
			service: Service{
				Language: "docker",
				Ports:    []string{"3000:8080"},
			},
			wantPort:       3000,
			wantIsExplicit: true,
			wantError:      false,
		},
		{
			name: "no ports - falls back to auto-assign",
			service: Service{
				Language: "node",
			},
			wantIsExplicit: false,
			wantError:      false,
			// Port will be auto-assigned, we don't check exact value
		},
		{
			name: "Docker container-only port",
			service: Service{
				Language: "python",
				Docker:   &DockerConfig{Image: "python:3.9"},
				Ports:    []string{"8080"}, // No host port specified
			},
			wantPort:       0, // Host port is auto-assign (0)
			wantIsExplicit: false,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usedPorts := make(map[int]bool)
			port, isExplicit, err := DetectPort("test-service", tt.service, ".", "", usedPorts)

			if (err != nil) != tt.wantError {
				t.Errorf("DetectPort() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if isExplicit != tt.wantIsExplicit {
				t.Errorf("isExplicit = %v, want %v", isExplicit, tt.wantIsExplicit)
			}

			// Only check port value if it's explicit
			if tt.wantIsExplicit && port != tt.wantPort {
				t.Errorf("port = %d, want %d", port, tt.wantPort)
			}
		})
	}
}
