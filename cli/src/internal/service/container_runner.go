package service

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
)

const (
	// containerIDDisplayLength is the number of characters to display from container IDs in logs.
	// Docker typically uses the first 12 characters for display.
	containerIDDisplayLength = 12

	// containerStopGracePeriod is the timeout in seconds before forcefully stopping a container.
	containerStopGracePeriod = 5
)

// serviceNameRegex validates service names for container naming.
// Pattern: [a-zA-Z][a-zA-Z0-9_-]* (must start with letter)
var serviceNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// validateServiceNameForContainer ensures the service name is safe for use in container names.
func validateServiceNameForContainer(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("service name too long (max 64 characters)")
	}
	if !serviceNameRegex.MatchString(name) {
		return fmt.Errorf("invalid service name %q: must start with letter and contain only [a-zA-Z0-9_-]", name)
	}
	return nil
}

// StartContainerService starts a Docker container service and returns the process handle.
// Container services are identified by having an `image` field in azure.yaml.
//
// Parameters:
//   - runtime: ServiceRuntime containing service configuration
//   - projectDir: Project directory path
//   - restartContainers: If true, always restart containers even if already running.
//     If false, skip starting if container is already running and healthy.
func StartContainerService(runtime *ServiceRuntime, projectDir string, restartContainers bool) (*ServiceProcess, error) {
	// Validate service name before using it in container names
	if err := validateServiceNameForContainer(runtime.Name); err != nil {
		return nil, fmt.Errorf("invalid service name: %w", err)
	}

	// Get the container image from runtime.Command (set by detectContainerRuntime)
	image := runtime.Command
	if image == "" {
		return nil, fmt.Errorf("no image specified for container service %s", runtime.Name)
	}

	// Create Docker client
	client := docker.NewClient()

	// Check if Docker is available
	if !client.IsAvailable() {
		return nil, fmt.Errorf("docker is not available - please ensure Docker Desktop or Docker daemon is running")
	}

	slog.Debug("starting container service",
		slog.String("service", runtime.Name),
		slog.String("image", image),
		slog.Int("port", runtime.Port))

	// Pull image if needed (will be cached if already present)
	slog.Debug("pulling container image", slog.String("image", image))
	if err := client.Pull(image); err != nil {
		// Don't fail if pull fails - image might be cached locally
		slog.Warn("failed to pull image (continuing with cached version if available)",
			slog.String("image", image),
			slog.String("error", err.Error()))
	}

	// Check if container already exists
	containerName := fmt.Sprintf("azd-%s", runtime.Name)
	if !restartContainers {
		if container, err := client.InspectByName(containerName); err == nil && container != nil {
			// Container exists
			displayID := container.ID
			if len(displayID) > containerIDDisplayLength {
				displayID = displayID[:containerIDDisplayLength]
			}

			if client.IsRunning(container.ID) {
				// Already running - reuse it
				slog.Debug("container already running, reusing existing container",
					slog.String("service", runtime.Name),
					slog.String("container_name", containerName),
					slog.String("container_id", displayID))

				process := &ServiceProcess{
					Name:        runtime.Name,
					Runtime:     *runtime,
					Port:        runtime.Port,
					Ready:       true, // Container is already running
					Env:         runtime.Env,
					ContainerID: container.ID,
				}
				return process, nil
			}

			// Container exists but is stopped - start it instead of recreating
			slog.Debug("starting existing stopped container",
				slog.String("service", runtime.Name),
				slog.String("container_name", containerName),
				slog.String("container_id", displayID))

			if err := client.Start(container.ID); err != nil {
				slog.Warn("failed to start stopped container, will recreate",
					slog.String("error", err.Error()))
			} else {
				// Successfully started existing container
				process := &ServiceProcess{
					Name:        runtime.Name,
					Runtime:     *runtime,
					Port:        runtime.Port,
					Ready:       false, // Will be marked ready after health check
					Env:         runtime.Env,
					ContainerID: container.ID,
				}
				return process, nil
			}
		}
	}

	// Extract and remove internal auth vars before creating container config
	var shimPath, certsHostDir, extraHostsStr string
	if v, ok := runtime.Env["_AZD_AUTH_SHIM_PATH"]; ok {
		shimPath = v
		delete(runtime.Env, "_AZD_AUTH_SHIM_PATH")
	}
	if v, ok := runtime.Env["_AZD_AUTH_CERTS_HOST_DIR"]; ok {
		certsHostDir = v
		delete(runtime.Env, "_AZD_AUTH_CERTS_HOST_DIR")
	}
	if v, ok := runtime.Env["_AZD_AUTH_EXTRA_HOSTS"]; ok {
		extraHostsStr = v
		delete(runtime.Env, "_AZD_AUTH_EXTRA_HOSTS")
	}

	// Build container configuration (env vars are clean)
	config := docker.ContainerConfig{
		Name:        fmt.Sprintf("azd-%s", runtime.Name),
		Image:       image,
		Ports:       buildContainerPortMappings(runtime),
		Environment: runtime.Env,
	}

	// Add user-defined volumes from azure.yaml
	for _, vol := range runtime.ContainerVolumes {
		parsed, err := parseVolumeMount(vol, projectDir)
		if err != nil {
			slog.Warn("skipping invalid volume mount",
				slog.String("volume", vol),
				slog.String("error", err.Error()))
			continue
		}
		config.Volumes = append(config.Volumes, parsed)
	}

	// Add container command from azure.yaml
	if runtime.ContainerCommand != "" {
		// Split command using shell-style parsing (respects quotes)
		config.Command = splitCommand(runtime.ContainerCommand)
	}

	// Inject container auth volumes and extra hosts if enabled
	if shimPath != "" && certsHostDir != "" {
		config.Volumes = append(config.Volumes,
			docker.VolumeMount{Source: shimPath, Target: "/usr/local/bin/azd", ReadOnly: true},
			docker.VolumeMount{Source: certsHostDir, Target: "/run/secrets/azd-auth", ReadOnly: true},
		)

		// Add extra hosts if needed (e.g., host.docker.internal on native Linux)
		if extraHostsStr != "" {
			for _, h := range strings.Split(extraHostsStr, ",") {
				if h != "" {
					config.ExtraHosts = append(config.ExtraHosts, h)
				}
			}
		}

		slog.Info("container auth enabled",
			slog.String("service", runtime.Name),
			slog.String("shimPath", shimPath),
			slog.String("certsDir", certsHostDir))
	}

	// Run container
	containerID, err := client.Run(config)
	if err != nil {
		// If container already exists, try to remove and recreate
		if strings.Contains(err.Error(), "is already in use") {
			slog.Info("removing existing container", slog.String("name", config.Name))
			if stopErr := client.Stop(config.Name, containerStopGracePeriod); stopErr != nil {
				slog.Debug("failed to stop existing container", slog.String("error", stopErr.Error()))
			}
			if rmErr := client.Remove(config.Name); rmErr != nil {
				slog.Debug("failed to remove existing container", slog.String("error", rmErr.Error()))
			}
			// Try again
			containerID, err = client.Run(config)
			if err != nil {
				return nil, fmt.Errorf("failed to start container %s: %w", runtime.Name, err)
			}
		} else {
			return nil, fmt.Errorf("failed to start container %s: %w", runtime.Name, err)
		}
	}

	displayID := containerID
	if len(displayID) > containerIDDisplayLength {
		displayID = displayID[:containerIDDisplayLength]
	}
	slog.Debug("container started",
		slog.String("service", runtime.Name),
		slog.String("container_id", displayID),
		slog.Int("port", runtime.Port))

	// Create process handle for the container
	process := &ServiceProcess{
		Name:        runtime.Name,
		Runtime:     *runtime,
		Port:        runtime.Port,
		Ready:       false,
		Env:         runtime.Env,
		ContainerID: containerID,
	}

	return process, nil
}

// buildContainerPortMappings converts ServiceRuntime port to Docker port mappings.
func buildContainerPortMappings(runtime *ServiceRuntime) []docker.PortMapping {
	var mappings []docker.PortMapping

	// If runtime has a port, map it
	if runtime.Port > 0 {
		mappings = append(mappings, docker.PortMapping{
			HostPort:      runtime.Port,
			ContainerPort: runtime.Port, // Assume same port for now
			Protocol:      "tcp",
		})
	}

	// TODO(#1001): Parse additional ports from runtime if needed
	// Currently only maps the primary port from runtime.Port. Need to support multiple port mappings
	// for services that expose additional ports (e.g., debug ports, metrics endpoints).

	return mappings
}

// parseVolumeMount parses a Docker Compose-style volume string (e.g., "./src:/app:ro")
// into a docker.VolumeMount. Relative source paths are resolved against projectDir.
// Handles Windows absolute paths (e.g., "C:\data:/app").
func parseVolumeMount(volume string, projectDir string) (docker.VolumeMount, error) {
	parts := strings.Split(volume, ":")

	// On Windows, absolute paths like C:\foo:/app split into ["C", "\foo", "/app"].
	// Detect drive letter and rejoin.
	if len(parts) >= 3 && len(parts[0]) == 1 && strings.ContainsAny(parts[0], "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		// Rejoin drive letter: "C" + ":" + "\foo" => "C:\foo"
		parts = append([]string{parts[0] + ":" + parts[1]}, parts[2:]...)
	}

	if len(parts) < 2 {
		return docker.VolumeMount{}, fmt.Errorf("invalid volume format %q: expected host:container[:ro]", volume)
	}

	source := parts[0]
	target := parts[1]
	readOnly := len(parts) >= 3 && parts[2] == "ro"

	// Resolve relative source paths against the project directory
	if !filepath.IsAbs(source) {
		source = filepath.Join(projectDir, source)
	}

	return docker.VolumeMount{
		Source:   source,
		Target:   target,
		ReadOnly: readOnly,
	}, nil
}

// splitCommand splits a shell command string into individual arguments.
// Handles quoted strings (single and double quotes).
func splitCommand(command string) []string {
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for _, r := range command {
		switch {
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case r == ' ' && !inSingleQuote && !inDoubleQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// StopContainerService stops a Docker container service.
func StopContainerService(process *ServiceProcess, timeout time.Duration) error {
	if process == nil {
		return fmt.Errorf("process is nil")
	}

	// Get container ID from process
	containerID := process.ContainerID
	if containerID == "" {
		return fmt.Errorf("no container ID for service %s", process.Name)
	}

	client := docker.NewClient()

	displayID := containerID
	if len(displayID) > containerIDDisplayLength {
		displayID = displayID[:containerIDDisplayLength]
	}
	slog.Debug("stopping container service",
		slog.String("service", process.Name),
		slog.String("container_id", displayID))

	// Stop container with timeout
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 10
	}

	if err := client.Stop(containerID, timeoutSeconds); err != nil {
		slog.Warn("failed to stop container gracefully",
			slog.String("service", process.Name),
			slog.String("error", err.Error()))
	}

	// Remove container
	if err := client.Remove(containerID); err != nil {
		slog.Warn("failed to remove container",
			slog.String("service", process.Name),
			slog.String("error", err.Error()))
	}

	slog.Debug("container stopped",
		slog.String("service", process.Name))

	return nil
}

// StartContainerLogCollection starts collecting logs from a container.
func StartContainerLogCollection(process *ServiceProcess, projectDir string) error {
	containerID := process.ContainerID
	if containerID == "" {
		return fmt.Errorf("no container ID for service %s", process.Name)
	}

	client := docker.NewClient()

	// Get log stream from container
	logReader, err := client.Logs(containerID)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}

	// Get or create log manager for this project
	logManager := GetLogManager(projectDir)

	// Create log buffer for this service
	buffer, err := logManager.CreateBuffer(process.Name, 1000, true)
	if err != nil {
		_ = logReader.Close()
		return fmt.Errorf("failed to create log buffer: %w", err)
	}

	// Start goroutine to collect logs
	go collectContainerLogs(logReader, process.Name, buffer)

	return nil
}

// collectContainerLogs reads from a container log stream and adds entries to the buffer.
func collectContainerLogs(reader io.ReadCloser, serviceName string, buffer *LogBuffer) {
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		entry := LogEntry{
			Service:   serviceName,
			Message:   scanner.Text(),
			Timestamp: time.Now(),
			IsStderr:  false, // Docker logs combine stdout/stderr
			Level:     inferLogLevel(scanner.Text()),
		}
		buffer.Add(entry)
	}
}

// IsContainerRunning checks if a container service is still running.
func IsContainerRunning(process *ServiceProcess) bool {
	containerID := process.ContainerID
	if containerID == "" {
		return false
	}

	client := docker.NewClient()
	return client.IsRunning(containerID)
}
