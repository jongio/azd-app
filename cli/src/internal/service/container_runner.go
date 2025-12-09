package service

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
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
//                        If false, skip starting if container is already running and healthy.
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
		return nil, fmt.Errorf("Docker is not available - please ensure Docker Desktop or Docker daemon is running")
	}

	slog.Info("starting container service",
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

	// Check if container already exists and is running
	containerName := fmt.Sprintf("azd-%s", runtime.Name)
	if !restartContainers {
		if container, err := client.InspectByName(containerName); err == nil && container != nil {
			// Container exists
			if client.IsRunning(container.ID) {
				slog.Info("container already running, reusing existing container",
					slog.String("service", runtime.Name),
					slog.String("container_name", containerName),
					slog.String("container_id", container.ID[:min(12, len(container.ID))]))
				
				// Return existing container process
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
		}
	}

	// Build container configuration
	config := docker.ContainerConfig{
		Name:        fmt.Sprintf("azd-%s", runtime.Name),
		Image:       image,
		Ports:       buildContainerPortMappings(runtime),
		Environment: runtime.Env,
	}

	// Run container
	containerID, err := client.Run(config)
	if err != nil {
		// If container already exists, try to remove and recreate
		if strings.Contains(err.Error(), "is already in use") {
			slog.Info("removing existing container", slog.String("name", config.Name))
			if stopErr := client.Stop(config.Name, 5); stopErr != nil {
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

	slog.Info("container started",
		slog.String("service", runtime.Name),
		slog.String("container_id", containerID[:12]),
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

	// TODO: Parse additional ports from runtime if needed

	return mappings
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

	slog.Info("stopping container service",
		slog.String("service", process.Name),
		slog.String("container_id", containerID[:min(12, len(containerID))]))

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

	slog.Info("container stopped",
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
		logReader.Close()
		return fmt.Errorf("failed to create log buffer: %w", err)
	}

	// Start goroutine to collect logs
	go collectContainerLogs(logReader, process.Name, buffer)

	return nil
}

// collectContainerLogs reads from a container log stream and adds entries to the buffer.
func collectContainerLogs(reader io.ReadCloser, serviceName string, buffer *LogBuffer) {
	defer reader.Close()

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
