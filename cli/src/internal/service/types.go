package service

import (
	"io"
	"os"
	"time"
)

// AzureYaml represents the parsed azure.yaml file.
type AzureYaml struct {
	Name      string                 `yaml:"name"`
	Services  map[string]Service     `yaml:"services"`
	Resources map[string]Resource    `yaml:"resources"`
	Metadata  map[string]interface{} `yaml:"metadata,omitempty"`
}

// Service represents a service definition in azure.yaml.
type Service struct {
	Host       string                 `yaml:"host"`
	Language   string                 `yaml:"language,omitempty"`
	Project    string                 `yaml:"project,omitempty"`
	Entrypoint string                 `yaml:"entrypoint,omitempty"` // Entry point file for Python/Node projects
	Image      string                 `yaml:"image,omitempty"`
	Docker     *DockerConfig          `yaml:"docker,omitempty"`
	Config     map[string]interface{} `yaml:"config,omitempty"`
	Env        []EnvVar               `yaml:"env,omitempty"`
	Uses       []string               `yaml:"uses,omitempty"`
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
type EnvVar struct {
	Name   string `yaml:"name"`
	Value  string `yaml:"value,omitempty"`
	Secret string `yaml:"secret,omitempty"`
}

// Resource represents a resource definition in azure.yaml.
type Resource struct {
	Type     string   `yaml:"type"`
	Uses     []string `yaml:"uses,omitempty"`
	Existing bool     `yaml:"existing,omitempty"`
}

// ServiceRuntime contains the detected runtime information for a service.
type ServiceRuntime struct {
	Name           string
	Language       string
	Framework      string
	PackageManager string
	Command        string
	Args           []string
	WorkingDir     string
	Port           int
	Protocol       string
	Env            map[string]string
	HealthCheck    HealthCheckConfig
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
