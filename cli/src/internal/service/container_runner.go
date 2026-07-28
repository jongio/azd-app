package service

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/detector"
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

	// Get the container image from runtime.Image (set by detectContainerRuntime)
	image := runtime.Image
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

	// Ensure a per-project network exists so container services can resolve each
	// other by service name. Derive the name from the project ROOT (the azure.yaml
	// directory) so it is identical whether we were passed the project root
	// (restart/dashboard paths) or a working subdirectory (run path uses os.Getwd).
	// Best-effort: if the network can't be created, fall back to no shared network
	// (single-container projects are unaffected).
	networkDir := projectNetworkDir(projectDir)
	networkName := DeriveNetworkName(networkDir)
	networkAliases := []string{runtime.Name}
	if err := client.EnsureNetwork(networkName); err != nil {
		slog.Warn("failed to create container network; containers will not share a network",
			slog.String("network", networkName),
			slog.String("error", err.Error()))
		networkName = ""
	}

	// Pull the image according to the configured pull policy.
	if shouldPullImage(client, image, runtime.PullPolicy) {
		slog.Debug("pulling container image", slog.String("image", image))
		if err := client.Pull(image); err != nil {
			if runtime.PullPolicy == docker.PullAlways {
				return nil, fmt.Errorf("failed to pull image %q (pull_policy=always): %w", image, err)
			}
			// Don't fail otherwise - image might be cached locally
			slog.Warn("failed to pull image (continuing with cached version if available)",
				slog.String("image", image),
				slog.String("error", err.Error()))
		}
	}
	if runtime.PullPolicy == docker.PullNever && !client.ImageExists(image) {
		return nil, fmt.Errorf("image %q is not present locally and pull_policy=never", image)
	}

	// Derive a project-scoped container name so containers from different
	// projects never collide. Uses the same hash as the network name.
	projectRoot := projectNetworkDir(projectDir)
	absRoot, _ := filepath.Abs(projectRoot)
	absRoot = filepath.Clean(absRoot)
	sum := sha256.Sum256([]byte(absRoot))
	projHash := hex.EncodeToString(sum[:])[:8]
	containerName := fmt.Sprintf("azd-%s-%s", runtime.Name, projHash)

	// reconnectToNetwork idempotently attaches a reused container to the project
	// network so sibling DNS keeps working after a fast restart.
	reconnectToNetwork := func() {
		if networkName == "" {
			return
		}
		if connErr := client.ConnectNetwork(networkName, containerName, networkAliases); connErr != nil {
			slog.Debug("failed to reconnect reused container to network",
				slog.String("service", runtime.Name),
				slog.String("error", connErr.Error()))
		}
	}

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

				// Reconnect to the project network so sibling DNS keeps working
				// after a fast restart.
				reconnectToNetwork()

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
				// Reconnect to the project network (idempotent if already connected).
				reconnectToNetwork()
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

	// Build container configuration
	config := docker.ContainerConfig{
		Name:           containerName,
		Image:          image,
		Ports:          buildContainerPortMappings(runtime),
		Environment:    runtime.Env,
		Command:        runtime.Args,
		Volumes:        runtime.Volumes,
		Network:        networkName,
		NetworkAliases: networkAliases,
		PullPolicy:     runtime.PullPolicy,
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

// buildContainerPortMappings converts a ServiceRuntime's ports to Docker port
// mappings. When the runtime has explicit multi-port mappings (from the service
// `ports` list) every port is published. Otherwise it falls back to the single
// primary port for backward compatibility.
func buildContainerPortMappings(runtime *ServiceRuntime) []docker.PortMapping {
	if len(runtime.Ports) > 0 {
		mappings := make([]docker.PortMapping, 0, len(runtime.Ports))
		for _, p := range runtime.Ports {
			mappings = append(mappings, docker.PortMapping{
				HostPort:      p.HostPort,
				ContainerPort: p.ContainerPort,
				BindIP:        p.BindIP,
				Protocol:      protoOrDefault(p.Protocol),
			})
		}
		return mappings
	}

	// Fallback: single primary port (services configured without an explicit
	// ports list, e.g. an auto-assigned primary).
	var mappings []docker.PortMapping
	if runtime.Port > 0 {
		mappings = append(mappings, docker.PortMapping{
			HostPort:      runtime.Port,
			ContainerPort: runtime.Port,
			Protocol:      "tcp",
		})
	}
	return mappings
}

// protoOrDefault returns the given protocol or "tcp" when empty.
func protoOrDefault(p string) string {
	if p == "" {
		return "tcp"
	}
	return p
}

// projectNetworkDir normalizes a working directory to the project root (the
// directory containing azure.yaml) so the derived per-project network name is
// identical across the run path (which may pass a working subdirectory) and the
// restart/dashboard paths (which pass the project root). Falls back to the given
// directory when no azure.yaml is found.
func projectNetworkDir(projectDir string) string {
	if azureYamlPath, err := detector.FindAzureYaml(projectDir); err == nil && azureYamlPath != "" {
		return filepath.Dir(azureYamlPath)
	}
	return projectDir
}

// imageExistenceChecker is the minimal Docker capability shouldPullImage needs.
type imageExistenceChecker interface {
	ImageExists(image string) bool
}

// shouldPullImage decides whether to pull the image before running, based on the
// pull policy:
//   - "never":  never pull.
//   - "missing": pull only when the image is absent locally.
//   - "always": always pull.
//   - "" (default): best-effort pull (preserves prior behavior).
func shouldPullImage(client imageExistenceChecker, image, policy string) bool {
	switch policy {
	case docker.PullNever:
		return false
	case docker.PullMissing:
		return !client.ImageExists(image)
	case docker.PullAlways:
		return true
	default:
		return true
	}
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
			Level:     inferLogLevel(scanner.Text(), false),
		}
		buffer.Add(entry)
	}
	if err := scanner.Err(); err != nil {
		slog.Debug("error reading container logs", slog.String("service", serviceName), slog.Any("error", err))
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
