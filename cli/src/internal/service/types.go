// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Service type constants define how a service is accessed (protocol level).
const (
	// ServiceTypeHTTP indicates a service that serves HTTP/HTTPS traffic.
	// Health checks use HTTP endpoint probing. This is the default for services with ports.
	ServiceTypeHTTP = "http"

	// ServiceTypeTCP indicates a service that accepts raw TCP connections (databases, gRPC).
	// Health checks use TCP port connectivity.
	ServiceTypeTCP = "tcp"

	// ServiceTypeProcess indicates a service with no network endpoint.
	// Health checks verify the process is running. This is the default for services without ports.
	ServiceTypeProcess = "process"

	// ServiceTypeContainer indicates a Docker container service.
	// Health checks use TCP port connectivity by default.
	// Container services are started via Docker rather than native processes.
	ServiceTypeContainer = "container"
)

// Service mode constants define the lifecycle behavior of process-type services.
const (
	// ServiceModeWatch indicates a continuous process that watches for file changes.
	// Examples: tsc --watch, nodemon, air (Go), dotnet watch.
	// Health: process alive + optional pattern match in stdout.
	// Status display: "Watching" when healthy.
	ServiceModeWatch = "watch"

	// ServiceModeBuild indicates a one-time build process that exits on completion.
	// Examples: tsc, go build, dotnet build, npm run build.
	// Health: exit code 0 = success, non-zero = failure.
	// Status display: "Building" → "Built" or "Failed".
	ServiceModeBuild = "build"

	// ServiceModeDaemon indicates a long-running background process.
	// Examples: MCP servers, queue workers, file processors.
	// Health: process alive.
	// Status display: "Running" when healthy.
	ServiceModeDaemon = "daemon"

	// ServiceModeTask indicates a one-time task run on demand.
	// Examples: database migrations, seed scripts.
	// Health: exit code 0 = success, non-zero = failure.
	// Status display: "Running" → "Complete" or "Failed".
	ServiceModeTask = "task"
)

// AzureYaml represents the parsed azure.yaml file.
type AzureYaml struct {
	Name      string              `yaml:"name"`
	Services  map[string]Service  `yaml:"services"`
	Resources map[string]Resource `yaml:"resources"`
	Metadata  map[string]any      `yaml:"metadata,omitempty"`
	Hooks     *Hooks              `yaml:"hooks,omitempty"`
	Dashboard *DashboardConfig    `yaml:"dashboard,omitempty"`
	Logs      *LogsConfig         `yaml:"logs,omitempty"` // Project-level logging configuration
}

// DashboardConfig represents dashboard configuration in azure.yaml.
type DashboardConfig struct {
	Browser string `yaml:"browser,omitempty"` // Browser target: default, system, none
}

// Service represents a service definition in azure.yaml.
type Service struct {
	Host               string             `yaml:"host"`
	Language           string             `yaml:"language,omitempty"`
	Project            string             `yaml:"project,omitempty"`
	Command            string             `yaml:"command,omitempty"`    // Full command to run (e.g., "uvicorn main:app --reload"). Primary way to override.
	Entrypoint         string             `yaml:"entrypoint,omitempty"` // Advanced: executable only, use with command for args. Rarely needed.
	Image              string             `yaml:"image,omitempty"`
	Docker             *DockerConfig      `yaml:"docker,omitempty"`
	Ports              []string           `yaml:"ports,omitempty"`       // Docker Compose style: ["8080"] or ["3000:8080"]
	Environment        Environment        `yaml:"environment,omitempty"` // Docker Compose style: supports map, array of strings, or array of objects
	Uses               []string           `yaml:"uses,omitempty"`
	Logs               *LogsConfig        `yaml:"logs,omitempty"`        // Service-level logging configuration
	Healthcheck        *HealthcheckConfig `yaml:"healthcheck,omitempty"` // Docker Compose-compatible health check configuration
	HealthcheckEnabled *bool              `yaml:"-"`                     // Internal flag: nil = use default, false = explicitly disabled, true = explicitly enabled
	Type               string             `yaml:"type,omitempty"`        // Service type: "http", "tcp", "process". Default: "http" if ports defined, "process" otherwise.
	Mode               string             `yaml:"mode,omitempty"`        // Run mode (for type=process): "watch", "build", "daemon", "task". Default: "daemon".
}

// serviceRaw is used to handle both boolean and object healthcheck values.
// It duplicates all fields from Service except Healthcheck to avoid infinite recursion.
type serviceRaw struct {
	Host        string        `yaml:"host"`
	Language    string        `yaml:"language,omitempty"`
	Project     string        `yaml:"project,omitempty"`
	Entrypoint  string        `yaml:"entrypoint,omitempty"`
	Command     string        `yaml:"command,omitempty"`
	Image       string        `yaml:"image,omitempty"`
	Docker      *DockerConfig `yaml:"docker,omitempty"`
	Ports       []string      `yaml:"ports,omitempty"`
	Environment Environment   `yaml:"environment,omitempty"`
	Uses        []string      `yaml:"uses,omitempty"`
	Logs        *LogsConfig   `yaml:"logs,omitempty"`
	Healthcheck any           `yaml:"healthcheck,omitempty"`
	Type        string        `yaml:"type,omitempty"`
	Mode        string        `yaml:"mode,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling to handle healthcheck: false.
func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw serviceRaw
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Copy all fields from the raw struct
	s.Host = raw.Host
	s.Language = raw.Language
	s.Project = raw.Project
	s.Entrypoint = raw.Entrypoint
	s.Command = raw.Command
	s.Image = raw.Image
	s.Docker = raw.Docker
	s.Ports = raw.Ports
	s.Environment = raw.Environment
	s.Uses = raw.Uses
	s.Logs = raw.Logs
	s.Type = raw.Type
	s.Mode = raw.Mode

	// Handle healthcheck field
	switch v := raw.Healthcheck.(type) {
	case bool:
		// healthcheck: false or healthcheck: true
		s.HealthcheckEnabled = &v
		if !v {
			// Create a HealthcheckConfig with Disable: true to match the behavior
			s.Healthcheck = &HealthcheckConfig{Disable: true}
		}
	case map[string]any:
		// healthcheck: { ... } - use standard unmarshaling
		// Need to re-unmarshal just the healthcheck field
		var hc HealthcheckConfig
		// Convert map back to YAML bytes and unmarshal
		data, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &hc); err != nil {
			return err
		}
		s.Healthcheck = &hc
	case nil:
		// No healthcheck specified
		s.Healthcheck = nil
	}

	return nil
}

// IsHealthcheckDisabled returns true if health checks should be skipped for this service.
// This can be triggered by:
// - healthcheck: false (boolean)
// - healthcheck.disable: true
// - healthcheck.type: "none"
// - healthcheck.test: ["NONE"]
func (s *Service) IsHealthcheckDisabled() bool {
	// Check if explicitly set to false
	if s.HealthcheckEnabled != nil && !*s.HealthcheckEnabled {
		return true
	}

	// Delegate to HealthcheckConfig.IsDisabled()
	return s.Healthcheck.IsDisabled()
}

// NeedsPort returns true if this service needs a port assigned.
// Services must explicitly define ports in azure.yaml to have a port assigned.
// Services without ports (e.g., build/watch services like tsc --watch) will use
// process-based health checks instead of HTTP health checks.
func (s *Service) NeedsPort() bool {
	// Only services with explicitly configured ports need a port assigned.
	// Services without ports will use process-based health checks.
	return len(s.Ports) > 0
}

// GetServiceType returns the service type, inferring from configuration if not explicitly set.
// Returns: "http" (default if ports defined), "tcp", or "process" (default if no ports).
func (s *Service) GetServiceType() string {
	// If explicitly set, use that
	if s.Type != "" {
		return s.Type
	}

	// Infer from configuration
	if s.NeedsPort() {
		return ServiceTypeHTTP
	}

	return ServiceTypeProcess
}

// GetServiceMode returns the run mode for process-type services.
// Returns: "watch", "build", "daemon" (default), or "task".
// For non-process types, returns empty string.
func (s *Service) GetServiceMode() string {
	// Only applicable for process type
	if s.GetServiceType() != ServiceTypeProcess {
		return ""
	}

	// If explicitly set, use that
	if s.Mode != "" {
		return s.Mode
	}

	// Default to daemon for process services
	return ServiceModeDaemon
}

// IsProcessService returns true if this is a process-type service (no network endpoint).
func (s *Service) IsProcessService() bool {
	return s.GetServiceType() == ServiceTypeProcess
}

// IsContainerService returns true if this service should run as a Docker container.
// A service is a container service when it has an `image` field (direct image reference)
// or a `docker.image` field (Docker config with image).
// Container services are launched via Docker rather than running as native processes.
func (s *Service) IsContainerService() bool {
	// Direct image reference
	if s.Image != "" {
		return true
	}
	// Docker config with image
	if s.Docker != nil && s.Docker.Image != "" {
		return true
	}
	return false
}

// GetContainerImage returns the Docker image for a container service.
// Returns empty string if not a container service.
func (s *Service) GetContainerImage() string {
	if s.Image != "" {
		return s.Image
	}
	if s.Docker != nil && s.Docker.Image != "" {
		return s.Docker.Image
	}
	return ""
}

// IsWatchMode returns true if this is a process service in watch mode.
func (s *Service) IsWatchMode() bool {
	return s.GetServiceType() == ServiceTypeProcess && s.GetServiceMode() == ServiceModeWatch
}

// IsBuildMode returns true if this is a process service in build mode.
func (s *Service) IsBuildMode() bool {
	return s.GetServiceType() == ServiceTypeProcess && s.GetServiceMode() == ServiceModeBuild
}

// HealthcheckConfig represents Docker Compose-compatible health check configuration.
// Supports cross-platform HTTP URL checks or shell command checks.
// Can also be set to false (via HealthcheckDisabled) to skip health checks entirely.
type HealthcheckConfig struct {
	// Test is the health check command or URL.
	// For cross-platform compatibility, use HTTP URL string (e.g., "http://localhost:8080/health").
	// Can also be shell command string or array (CMD or CMD-SHELL format).
	// Examples:
	//   - "http://localhost:8080/health" (cross-platform HTTP check)
	//   - ["CMD", "curl", "-f", "http://localhost/health"]
	//   - ["CMD-SHELL", "curl -f http://localhost/health || exit 1"]
	//   - ["NONE"] (disable health check)
	Test any `yaml:"test,omitempty"`

	// Type specifies the health check method: "http", "tcp", "process", "output", or "none".
	// - "http": Check an HTTP endpoint (default)
	// - "tcp": Check if a port is listening
	// - "process": Check if the process is running
	// - "output": Monitor stdout for a pattern match
	// - "none": Disable health checks (service is always considered healthy)
	Type string `yaml:"type,omitempty"`

	// Path is the HTTP path for health checks (when type=http).
	// Defaults to "/health".
	Path string `yaml:"path,omitempty"`

	// Pattern is a regex pattern to match in stdout (when type=output).
	// Service is considered healthy when this pattern is matched.
	// Examples: "Found 0 errors", "Server started", "Listening on port"
	Pattern string `yaml:"pattern,omitempty"`

	// Interval is the time between health checks (e.g., "30s", "1m").
	Interval string `yaml:"interval,omitempty"`

	// Timeout is the maximum time for health check to complete (e.g., "30s", "1m").
	Timeout string `yaml:"timeout,omitempty"`

	// Retries is the number of consecutive failures before marking unhealthy.
	Retries int `yaml:"retries,omitempty"`

	// StartPeriod is the grace period for container initialization (e.g., "0s", "40s").
	StartPeriod string `yaml:"start_period,omitempty"`

	// StartInterval is the time between health checks during start period (e.g., "5s").
	StartInterval string `yaml:"start_interval,omitempty"`

	// Disable set to true disables the healthcheck entirely.
	// This is equivalent to test: ["NONE"] or type: "none".
	Disable bool `yaml:"disable,omitempty"`
}

// IsDisabled returns true if health checks should be skipped for this service.
// This can be triggered by:
// - disable: true
// - type: "none"
// - test: ["NONE"]
func (h *HealthcheckConfig) IsDisabled() bool {
	if h == nil {
		return false
	}

	// Check explicit disable flag
	if h.Disable {
		return true
	}

	// Check type: "none"
	if h.Type == "none" {
		return true
	}

	// Check test: ["NONE"]
	switch t := h.Test.(type) {
	case []any:
		if len(t) > 0 {
			if str, ok := t[0].(string); ok && str == "NONE" {
				return true
			}
		}
	case []string:
		if len(t) > 0 && t[0] == "NONE" {
			return true
		}
	}

	return false
}

// GetType returns the health check type, with "http" as the default.
func (h *HealthcheckConfig) GetType() string {
	if h == nil {
		return "http"
	}
	if h.Type != "" {
		return h.Type
	}
	return "http"
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
	var envArray []any
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
		case map[string]any:
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
	ShouldUpdateAzureYaml bool   // True if user wants port added to azure.yaml
	Type                  string // Service type: "http", "tcp", "process"
	Mode                  string // Run mode (for type=process): "watch", "build", "daemon", "task"
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
	ContainerID string // Container ID for container services (Type=container)
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

// LogContext contains log lines before and after a log entry for debugging context.
type LogContext struct {
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

// LogEntryWithContext represents a log entry with surrounding context.
// Used when filtering logs by level and including context lines for debugging.
type LogEntryWithContext struct {
	Service   string     `json:"service"`
	Message   string     `json:"message"`
	Level     LogLevel   `json:"level"`
	Timestamp time.Time  `json:"timestamp"`
	IsStderr  bool       `json:"isStderr"`
	Context   LogContext `json:"context,omitempty"`
	Count     int        `json:"count,omitempty"` // For deduplication: how many times this entry occurred
}

// ErrorEntry is deprecated: use LogEntryWithContext instead.
// Deprecated: This type alias exists for backward compatibility.
type ErrorEntry = LogEntryWithContext

// ErrorContext is deprecated: use LogContext instead.
// Deprecated: This type alias exists for backward compatibility.
type ErrorContext = LogContext

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

// GetPrerun safely retrieves the prerun hook, returning nil if not configured.
func (h *Hooks) GetPrerun() *Hook {
	if h == nil {
		return nil
	}
	return h.Prerun
}

// GetPostrun safely retrieves the postrun hook, returning nil if not configured.
func (h *Hooks) GetPostrun() *Hook {
	if h == nil {
		return nil
	}
	return h.Postrun
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
