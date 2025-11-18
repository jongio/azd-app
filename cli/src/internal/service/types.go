// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"io"
	"os"
	"strings"
	"time"
)

// AzureYaml represents the parsed azure.yaml file.
type AzureYaml struct {
	Name      string                 `yaml:"name"`
	Services  map[string]Service     `yaml:"services"`
	Resources map[string]Resource    `yaml:"resources"`
	Metadata  map[string]interface{} `yaml:"metadata,omitempty"`
	Hooks     *Hooks                 `yaml:"hooks,omitempty"`
}

// Service represents a service definition in azure.yaml.
type Service struct {
	Host        string        `yaml:"host"`
	Language    string        `yaml:"language,omitempty"`
	Project     string        `yaml:"project,omitempty"`
	Entrypoint  string        `yaml:"entrypoint,omitempty"` // Entry point file for Python/Node projects
	Image       string        `yaml:"image,omitempty"`
	Docker      *DockerConfig `yaml:"docker,omitempty"`
	Ports       []string      `yaml:"ports,omitempty"`       // Docker Compose style: ["8080"] or ["3000:8080"]
	Environment Environment   `yaml:"environment,omitempty"` // Docker Compose style: supports map, array of strings, or array of objects
	Uses        []string      `yaml:"uses,omitempty"`
}

// DockerConfig represents Docker build configuration.
type DockerConfig struct {
	Path        string   `yaml:"path,omitempty"`
	Context     string   `yaml:"context,omitempty"`
	Platform    string   `yaml:"platform,omitempty"`
	Registry    string   `yaml:"registry,omitempty"`
	Image       string   `yaml:"image,omitempty"`
	Tag         string   `yaml:"tag,omitempty"`
	BuildArgs   []string `yaml:"buildArgs,omitempty"`
	RemoteBuild bool     `yaml:"remoteBuild,omitempty"`
}

// EnvVar represents an environment variable.
// Supports Docker Compose-compatible formats:
//  1. Object format: {name: "KEY", value: "val"}
//  2. String format: "KEY=value"
//  3. Map format: KEY: value (handled by Environment type)
type EnvVar struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value,omitempty"`
	Secret string `yaml:"secret,omitempty"`
}

// Environment represents environment variables in Docker Compose-compatible formats.
// Supports three input formats:
//  1. Map format: {KEY: value, KEY2: value2}
//  2. Array of strings: ["KEY=value", "KEY2=value2"]
//  3. Array of objects: [{name: "KEY", value: "val"}]
type Environment map[string]string

// UnmarshalYAML implements custom YAML unmarshaling for Docker Compose compatibility.
func (e *Environment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if *e == nil {
		*e = make(Environment)
	}

	// Try map format first (most common): KEY: value
	var envMap map[string]string
	if err := unmarshal(&envMap); err == nil {
		for k, v := range envMap {
			(*e)[k] = v
		}
		return nil
	}

	// Try array format: ["KEY=value"] or [{name: "KEY", value: "val"}]
	var envArray []interface{}
	if err := unmarshal(&envArray); err != nil {
		return err
	}

	for _, item := range envArray {
		switch v := item.(type) {
		case string:
			// String format: "KEY=value"
			parts := strings.SplitN(v, "=", 2)
			if len(parts) == 2 {
				(*e)[parts[0]] = parts[1]
			} else if len(parts) == 1 {
				// KEY without value means empty string
				(*e)[parts[0]] = ""
			}
		case map[string]interface{}:
			// Object format: {name: "KEY", value: "val", secret: "secret"}
			name, hasName := v["name"].(string)
			if !hasName {
				continue
			}
			// Prefer secret over value if both are present
			if secret, hasSecret := v["secret"].(string); hasSecret {
				(*e)[name] = secret
			} else if value, hasValue := v["value"].(string); hasValue {
				(*e)[name] = value
			} else {
				(*e)[name] = ""
			}
		}
	}

	return nil
}

// Resource represents a resource definition in azure.yaml.
type Resource struct {
	Type     string   `yaml:"type"`
	Uses     []string `yaml:"uses,omitempty"`
	Existing bool     `yaml:"existing,omitempty"`
}

// GetEnvironment returns the environment variables for the service.
func (s *Service) GetEnvironment() map[string]string {
	if s.Environment == nil {
		return make(map[string]string)
	}
	return s.Environment
}

// ServiceRuntime contains the detected runtime information for a service.
type ServiceRuntime struct {
	Name                  string
	Language              string
	Framework             string
	PackageManager        string
	Command               string
	Args                  []string
	WorkingDir            string
	Port                  int
	Protocol              string
	Env                   map[string]string
	HealthCheck           HealthCheckConfig
	ShouldUpdateAzureYaml bool // True if user wants port added to azure.yaml
}

// PortMapping represents a port mapping (Docker Compose style).
type PortMapping struct {
	HostPort      int    // Port on host machine (0 = auto-assign)
	ContainerPort int    // Port inside container (for Docker) or app port (for non-Docker)
	BindIP        string // IP to bind to (e.g., "127.0.0.1"), empty = all interfaces
	Protocol      string // "tcp" or "udp", defaults to "tcp"
}

// HealthCheckConfig defines how to check if a service is ready.
type HealthCheckConfig struct {
	Type     string        // "http", "port", "process", "log"
	Path     string        // For HTTP health checks (e.g., "/health")
	Port     int           // Port to check
	Timeout  time.Duration // How long to wait for service to be ready
	Interval time.Duration // How often to retry
	LogMatch string        // For log-based checks (e.g., "Server started")
}

// ServiceProcess represents a running service process.
type ServiceProcess struct {
	Name        string
	Runtime     ServiceRuntime
	PID         int
	Port        int
	URL         string
	Process     *os.Process
	Stdout      io.ReadCloser
	Stderr      io.ReadCloser
	StartTime   time.Time
	Ready       bool
	HealthCheck chan error
	Env         map[string]string
}

// DependencyGraph represents service dependencies.
type DependencyGraph struct {
	Nodes map[string]*DependencyNode
	Edges map[string][]string // service -> dependencies
}

// DependencyNode represents a node in the dependency graph.
type DependencyNode struct {
	Name         string
	Service      *Service
	IsResource   bool
	Dependencies []string
	Level        int // Topological level (0 = no deps, 1 = depends on level 0, etc.)
}

// OrchestrationOptions contains options for service orchestration.
type OrchestrationOptions struct {
	ServiceFilter []string          // Only run these services
	EnvFile       string            // Load env vars from this file
	Verbose       bool              // Show detailed logs
	DryRun        bool              // Don't start services, just show plan
	NoHealthCheck bool              // Skip health checks
	Timestamps    bool              // Add timestamps to logs
	WorkingDir    string            // Working directory for service detection
	AzureEnv      map[string]string // Azure environment variables from azd context
}

// LogEntry represents a log entry from a service.
type LogEntry struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Level     LogLevel  `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	IsStderr  bool      `json:"isStderr"`
}

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
	LogLevelDebug
)

// String returns the string representation of a log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// FrameworkDefaults contains default configuration for known frameworks.
type FrameworkDefaults struct {
	Name           string
	Language       string
	DetectFiles    []string          // Files that indicate this framework
	DetectContent  map[string]string // File content patterns (file -> pattern)
	DefaultPort    int
	DevCommand     string
	DevArgs        []string
	HealthEndpoint string
	HealthLogMatch string
}

// Common framework defaults
var (
	// Node.js/TypeScript Frameworks
	FrameworkNextJS = FrameworkDefaults{
		Name:           "Next.js",
		Language:       "TypeScript",
		DetectFiles:    []string{"next.config.js", "next.config.ts", "next.config.mjs"},
		DetectContent:  map[string]string{"package.json": "\"next\""},
		DefaultPort:    3000,
		DevCommand:     "run",
		DevArgs:        []string{"dev"},
		HealthEndpoint: "/",
		HealthLogMatch: "ready on",
	}

	FrameworkReact = FrameworkDefaults{
		Name:           "React",
		Language:       "TypeScript",
		DetectFiles:    []string{"vite.config.ts", "vite.config.js"},
		DetectContent:  map[string]string{"package.json": "\"react\""},
		DefaultPort:    5173,
		DevCommand:     "run",
		DevArgs:        []string{"dev"},
		HealthEndpoint: "/",
	}

	FrameworkAngular = FrameworkDefaults{
		Name:           "Angular",
		Language:       "TypeScript",
		DetectFiles:    []string{"angular.json"},
		DefaultPort:    4200,
		DevCommand:     "ng",
		DevArgs:        []string{"serve"},
		HealthEndpoint: "/",
	}

	// Python Frameworks
	FrameworkDjango = FrameworkDefaults{
		Name:           "Django",
		Language:       "Python",
		DetectFiles:    []string{"manage.py"},
		DetectContent:  map[string]string{"manage.py": "django"},
		DefaultPort:    8000,
		DevCommand:     "python",
		DevArgs:        []string{"manage.py", "runserver"},
		HealthEndpoint: "/",
		HealthLogMatch: "Starting development server",
	}

	FrameworkFastAPI = FrameworkDefaults{
		Name:           "FastAPI",
		Language:       "Python",
		DetectContent:  map[string]string{"main.py": "FastAPI", "app.py": "FastAPI"},
		DefaultPort:    8000,
		DevCommand:     "uvicorn",
		HealthEndpoint: "/health",
	}

	FrameworkFlask = FrameworkDefaults{
		Name:           "Flask",
		Language:       "Python",
		DetectContent:  map[string]string{"app.py": "Flask", "main.py": "Flask"},
		DefaultPort:    5000,
		DevCommand:     "flask",
		DevArgs:        []string{"run"},
		HealthEndpoint: "/",
	}

	// .NET Frameworks
	FrameworkAspire = FrameworkDefaults{
		Name:           "Aspire",
		Language:       ".NET",
		DetectFiles:    []string{"AppHost.cs"},
		DefaultPort:    15888, // Aspire dashboard port
		DevCommand:     "dotnet",
		DevArgs:        []string{"run"},
		HealthEndpoint: "/",
		HealthLogMatch: "Now listening on",
	}

	FrameworkASPNET = FrameworkDefaults{
		Name:           "ASP.NET Core",
		Language:       ".NET",
		DefaultPort:    5000,
		DevCommand:     "dotnet",
		DevArgs:        []string{"run"},
		HealthEndpoint: "/",
		HealthLogMatch: "Now listening on",
	}

	// Java Frameworks
	FrameworkSpringBoot = FrameworkDefaults{
		Name:           "Spring Boot",
		Language:       "Java",
		DetectFiles:    []string{"pom.xml", "build.gradle"},
		DetectContent:  map[string]string{"pom.xml": "spring-boot", "build.gradle": "spring-boot"},
		DefaultPort:    8080,
		DevCommand:     "mvn",
		DevArgs:        []string{"spring-boot:run"},
		HealthEndpoint: "/actuator/health",
		HealthLogMatch: "Started",
	}
)

// DefaultPorts maps languages to their conventional default ports.
var DefaultPorts = map[string]int{
	"node":       3000,
	"nodejs":     3000,
	"javascript": 3000,
	"js":         3000,
	"typescript": 3000,
	"ts":         3000,
	"python":     8000,
	"py":         8000,
	"dotnet":     5000,
	"csharp":     5000,
	"java":       8080,
	"go":         8080,
	"rust":       8000,
	"php":        8000,
}

// GetPortMappings returns all port mappings for a service.
// Returns (mappings, isExplicit) where isExplicit indicates if any port was explicitly configured.
func (s *Service) GetPortMappings() ([]PortMapping, bool) {
	if len(s.Ports) == 0 {
		return nil, false
	}

	mappings := make([]PortMapping, 0, len(s.Ports))
	hasExplicitPort := false

	// Determine if this is a Docker/containerized service
	isDocker := s.Docker != nil

	for _, portSpec := range s.Ports {
		mapping := ParsePortSpec(portSpec, isDocker)
		if mapping.HostPort > 0 {
			hasExplicitPort = true
		}
		mappings = append(mappings, mapping)
	}

	return mappings, hasExplicitPort
}

// GetPrimaryPort returns the first (primary) port mapping.
//
// Returns:
//   - hostPort: The port on the host machine (0 = auto-assign for Docker)
//   - containerPort: The port inside the container or app
//   - isExplicit: True if a host port was explicitly specified (not auto-assigned)
//
// Examples:
//
//	// Non-Docker service with single port
//	service := Service{Ports: ["8080"]}
//	host, container, isExplicit := service.GetPrimaryPort()
//	// Result: host=8080, container=8080, isExplicit=true
//
//	// Docker service with explicit mapping
//	service := Service{Docker: &DockerConfig{}, Ports: ["3000:8080"]}
//	host, container, isExplicit := service.GetPrimaryPort()
//	// Result: host=3000, container=8080, isExplicit=true
//
//	// Docker service with container-only port (auto-assign host)
//	service := Service{Docker: &DockerConfig{}, Ports: ["8080"]}
//	host, container, isExplicit := service.GetPrimaryPort()
//	// Result: host=0, container=8080, isExplicit=false
func (s *Service) GetPrimaryPort() (int, int, bool) {
	mappings, isExplicit := s.GetPortMappings()
	if len(mappings) == 0 {
		return 0, 0, false
	}

	return mappings[0].HostPort, mappings[0].ContainerPort, isExplicit
}

// Hooks represents lifecycle hooks for run command.
type Hooks struct {
	Prerun  *Hook `yaml:"prerun,omitempty"`
	Postrun *Hook `yaml:"postrun,omitempty"`
}

// Hook represents a lifecycle hook configuration.
type Hook struct {
	Run             string        `yaml:"run"`                       // Script or command to execute
	Shell           string        `yaml:"shell,omitempty"`           // Shell to use (sh, bash, pwsh, etc.)
	ContinueOnError bool          `yaml:"continueOnError,omitempty"` // Continue if hook fails
	Interactive     bool          `yaml:"interactive,omitempty"`     // Requires user interaction
	Windows         *PlatformHook `yaml:"windows,omitempty"`         // Windows-specific override
	Posix           *PlatformHook `yaml:"posix,omitempty"`           // POSIX (Linux/macOS)-specific override
}

// PlatformHook represents platform-specific hook configuration.
type PlatformHook struct {
	Run             string `yaml:"run"`
	Shell           string `yaml:"shell,omitempty"`
	ContinueOnError *bool  `yaml:"continueOnError,omitempty"` // Pointer to allow override to false
	Interactive     *bool  `yaml:"interactive,omitempty"`     // Pointer to allow override to false
}
