package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

const errEmptyContainerID = "container ID cannot be empty"

// ExecClient implements the Client interface using the docker CLI.
type ExecClient struct{}

// NewClient creates a new Docker client that uses the docker CLI.
func NewClient() *ExecClient {
	return &ExecClient{}
}

// IsAvailable checks if Docker is installed and running.
func (c *ExecClient) IsAvailable() bool {
	cmd := exec.CommandContext(context.Background(), "docker", "info")
	err := cmd.Run()
	return err == nil
}

// Pull downloads an image if not present locally.
func (c *ExecClient) Pull(image string) error {
	if err := ValidateImageName(image); err != nil {
		return err
	}

	cmd := exec.CommandContext(context.Background(), "docker", "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("failed to pull image %q: %s", image, stderrStr)
		}
		return fmt.Errorf("failed to pull image %q: %w", image, err)
	}

	return nil
}

// ImageExists reports whether an image is present in the local Docker cache.
// Returns false if the image is absent or docker cannot be reached.
func (c *ExecClient) ImageExists(image string) bool {
	if err := ValidateImageName(image); err != nil {
		return false
	}
	cmd := exec.CommandContext(context.Background(), "docker", "image", "inspect", image)
	return cmd.Run() == nil
}

// NetworkExists reports whether a user-defined Docker network exists.
func (c *ExecClient) NetworkExists(name string) (bool, error) {
	if err := ValidateNetworkName(name); err != nil {
		return false, err
	}
	// ValidateNetworkName permits "" (for struct validation); these network
	// methods require an actual name.
	if name == "" {
		return false, fmt.Errorf("network name cannot be empty")
	}
	cmd := exec.CommandContext(context.Background(), "docker", "network", "inspect", name)
	if err := cmd.Run(); err != nil {
		// A non-zero exit means the network does not exist; that is not an error.
		return false, nil
	}
	return true, nil
}

// EnsureNetwork creates a user-defined bridge network if it does not already
// exist. It is idempotent and safe to call concurrently: a race where two
// callers both create the network surfaces as an "already exists" error from
// docker, which is treated as success.
func (c *ExecClient) EnsureNetwork(name string) error {
	if err := ValidateNetworkName(name); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	cmd := exec.CommandContext(context.Background(), "docker", "network", "create", "--driver", "bridge", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		// Tolerate the race / re-run case where the network already exists.
		if isAlreadyExistsError(stderrStr) {
			return nil
		}
		if stderrStr != "" {
			return fmt.Errorf("failed to create network %q: %s: %w", name, stderrStr, err)
		}
		return fmt.Errorf("failed to create network %q: %w", name, err)
	}
	return nil
}

// RemoveNetwork removes a user-defined Docker network. A missing network is not
// an error. Removal fails if containers are still attached; callers should treat
// that as non-fatal.
func (c *ExecClient) RemoveNetwork(name string) error {
	if err := ValidateNetworkName(name); err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("network name cannot be empty")
	}

	cmd := exec.CommandContext(context.Background(), "docker", "network", "rm", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if isNetworkNotFoundError(stderrStr) {
			return nil
		}
		if stderrStr != "" {
			return fmt.Errorf("failed to remove network %q: %s: %w", name, stderrStr, err)
		}
		return fmt.Errorf("failed to remove network %q: %w", name, err)
	}
	return nil
}

// ConnectNetwork attaches an existing container to a network with optional DNS
// aliases. It is idempotent: if the container is already connected, it returns
// nil. Used to (re)attach reused containers so sibling DNS keeps working after a
// fast restart.
func (c *ExecClient) ConnectNetwork(network, container string, aliases []string) error {
	if err := ValidateNetworkName(network); err != nil {
		return err
	}
	if err := ValidateContainerName(container); err != nil {
		return err
	}
	if network == "" || container == "" {
		return fmt.Errorf("network and container names cannot be empty")
	}

	args := []string{"network", "connect"}
	for _, alias := range aliases {
		args = append(args, "--alias", alias)
	}
	args = append(args, network, container)

	cmd := exec.CommandContext(context.Background(), "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		// Already attached, treat as success (idempotent).
		if isAlreadyConnectedError(stderrStr) {
			return nil
		}
		if stderrStr != "" {
			return fmt.Errorf("failed to connect container %q to network %q: %s: %w", container, network, stderrStr, err)
		}
		return fmt.Errorf("failed to connect container %q to network %q: %w", container, network, err)
	}
	return nil
}

// isAlreadyExistsError reports whether docker stderr indicates the resource
// (e.g. a network) already exists. Used to make creation idempotent.
func isAlreadyExistsError(stderr string) bool {
	return strings.Contains(stderr, "already exists")
}

// isNetworkNotFoundError reports whether docker stderr indicates the network was
// not found. Used to make removal tolerant of a missing network.
func isNetworkNotFoundError(stderr string) bool {
	return strings.Contains(stderr, "not found") || strings.Contains(stderr, "No such network")
}

// isAlreadyConnectedError reports whether docker stderr indicates the container
// is already attached to the network. Used to make connect idempotent.
func isAlreadyConnectedError(stderr string) bool {
	return strings.Contains(stderr, "already exists in network") ||
		strings.Contains(stderr, "is already connected")
}

// Run creates and starts a container with the given configuration.
func (c *ExecClient) Run(config ContainerConfig) (string, error) {
	if err := config.Validate(); err != nil {
		return "", err
	}

	args := buildRunArgs(config)
	cmd := exec.CommandContext(context.Background(), "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			// Check for common errors
			if strings.Contains(stderrStr, "is already in use") {
				return "", fmt.Errorf("container name %q is already in use: %w", config.Name, err)
			}
			return "", fmt.Errorf("failed to run container: %s: %w", stderrStr, err)
		}
		return "", fmt.Errorf("failed to run container: %w", err)
	}

	containerID := strings.TrimSpace(stdout.String())
	if containerID == "" {
		return "", fmt.Errorf("docker run returned empty container ID")
	}

	return containerID, nil
}

// buildRunArgs constructs the arguments for docker run.
func buildRunArgs(config ContainerConfig) []string {
	args := []string{"run", "-d"}

	// Add container name
	if config.Name != "" {
		args = append(args, "--name", config.Name)
	}

	// Attach to a user-defined network so sibling containers can resolve this
	// one by its aliases (container-to-container DNS, Docker Compose parity).
	if config.Network != "" {
		args = append(args, "--network", config.Network)
		for _, alias := range config.NetworkAliases {
			args = append(args, "--network-alias", alias)
		}
	}

	// Enforce "never" at run time so docker does not implicitly pull an absent
	// image. "always" is handled by an explicit Pull in the runner; "missing"
	// and "" rely on docker's default (pull only when absent).
	if config.PullPolicy == PullNever {
		args = append(args, "--pull", "never")
	}

	// Add port mappings
	for _, port := range config.Ports {
		args = append(args, "-p", formatPortMapping(port))
	}

	// Add environment variables
	for key, value := range config.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add volume mounts (named volumes and bind mounts; bind host paths are
	// resolved to absolute by the caller before reaching here).
	for _, vol := range config.Volumes {
		args = append(args, "-v", vol)
	}

	// Add image
	args = append(args, config.Image)

	// Command tokens follow the image and override the image's default CMD.
	args = append(args, config.Command...)

	return args
}

// formatPortMapping formats a port mapping for the docker CLI.
// Supports optional bind IP (e.g., "127.0.0.1:8080:80/tcp").
func formatPortMapping(port PortMapping) string {
	protocol := port.GetProtocol()

	if port.HostPort == 0 {
		// Auto-assign host port
		if port.BindIP != "" {
			return fmt.Sprintf("%s::%d/%s", port.BindIP, port.ContainerPort, protocol)
		}
		return fmt.Sprintf("%d/%s", port.ContainerPort, protocol)
	}

	if port.BindIP != "" {
		return fmt.Sprintf("%s:%d:%d/%s", port.BindIP, port.HostPort, port.ContainerPort, protocol)
	}
	return fmt.Sprintf("%d:%d/%s", port.HostPort, port.ContainerPort, protocol)
}

// Stop stops a running container with the specified timeout.
func (c *ExecClient) Stop(containerID string, timeoutSeconds int) error {
	if containerID == "" {
		return errors.New(errEmptyContainerID)
	}

	args := []string{"stop", "-t", fmt.Sprintf("%d", timeoutSeconds), containerID}
	cmd := exec.CommandContext(context.Background(), "docker", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("failed to stop container %q: %s: %w", containerID, stderrStr, err)
		}
		return fmt.Errorf("failed to stop container %q: %w", containerID, err)
	}

	return nil
}

// Start starts an existing stopped container.
func (c *ExecClient) Start(containerID string) error {
	if containerID == "" {
		return errors.New(errEmptyContainerID)
	}

	cmd := exec.CommandContext(context.Background(), "docker", "start", containerID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("failed to start container %q: %s: %w", containerID, stderrStr, err)
		}
		return fmt.Errorf("failed to start container %q: %w", containerID, err)
	}

	return nil
}

// Remove removes a container.
func (c *ExecClient) Remove(containerID string) error {
	if containerID == "" {
		return errors.New(errEmptyContainerID)
	}

	cmd := exec.CommandContext(context.Background(), "docker", "rm", containerID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return fmt.Errorf("failed to remove container %q: %s: %w", containerID, stderrStr, err)
		}
		return fmt.Errorf("failed to remove container %q: %w", containerID, err)
	}

	return nil
}

// Logs returns a reader for the container's stdout/stderr stream.
// Uses concurrent readers to properly multiplex stdout and stderr since
// io.MultiReader reads sequentially (would block on stdout, never read stderr).
func (c *ExecClient) Logs(containerID string) (io.ReadCloser, error) {
	if containerID == "" {
		return nil, errors.New(errEmptyContainerID)
	}

	cmd := exec.CommandContext(context.Background(), "docker", "logs", "-f", containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("failed to start docker logs: %w", err)
	}

	// Create pipe to combine stdout and stderr with line-level atomicity.
	// Each goroutine reads complete lines and writes them as a single call to
	// the PipeWriter, preventing mid-line interleaving when both streams are
	// active simultaneously.
	pr, pw := io.Pipe()

	// Copy both streams concurrently with line-aware writes
	go func() {
		var wg sync.WaitGroup
		wg.Add(2)

		// scanLines reads complete lines from src and writes each as an atomic
		// call to pw. A shared mutex is not needed because PipeWriter.Write is
		// already serialized, and each Write call carries a full line.
		scanLines := func(src io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(src)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Bytes()
				buf := make([]byte, len(line)+1)
				copy(buf, line)
				buf[len(line)] = '\n'
				_, _ = pw.Write(buf)
			}
		}

		go scanLines(stdout)
		go scanLines(stderr)

		// Wait for both to complete, then close the writer
		wg.Wait()
		_ = pw.Close()
	}()

	return &combinedReadCloser{
		reader: pr,
		cmd:    cmd,
		pipe:   pw,
	}, nil
}

// combinedReadCloser combines multiple readers and manages the underlying command.
type combinedReadCloser struct {
	reader io.Reader
	cmd    *exec.Cmd
	pipe   *io.PipeWriter
}

func (c *combinedReadCloser) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

// Close shuts down the combined reader by closing the pipe and terminating the underlying command.
func (c *combinedReadCloser) Close() error {
	// Close the pipe writer to unblock any pending reads
	if c.pipe != nil {
		_ = c.pipe.Close()
	}
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
}

// dockerInspectResult represents the JSON output from docker inspect.
type dockerInspectResult struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status  string `json:"Status"`
		Running bool   `json:"Running"`
	} `json:"State"`
	Config struct {
		Image string `json:"Image"`
	} `json:"Config"`
}

// Inspect returns detailed information about a container.
func (c *ExecClient) Inspect(containerID string) (*Container, error) {
	if containerID == "" {
		return nil, errors.New(errEmptyContainerID)
	}

	cmd := exec.CommandContext(context.Background(), "docker", "inspect", containerID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "No such object") || strings.Contains(stderrStr, "no such") {
			return nil, fmt.Errorf("container %q not found", containerID)
		}
		if stderrStr != "" {
			return nil, fmt.Errorf("failed to inspect container %q: %s", containerID, stderrStr)
		}
		return nil, fmt.Errorf("failed to inspect container %q: %w", containerID, err)
	}

	var results []dockerInspectResult
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("container %q not found", containerID)
	}

	result := results[0]
	return &Container{
		ID:     result.ID,
		Name:   strings.TrimPrefix(result.Name, "/"),
		Image:  result.Config.Image,
		Status: result.State.Status,
	}, nil
}

// IsRunning checks if a container is currently running.
func (c *ExecClient) IsRunning(containerID string) bool {
	if containerID == "" {
		return false
	}

	cmd := exec.CommandContext(context.Background(), "docker", "inspect", "-f", "{{.State.Running}}", containerID)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false
	}

	return strings.TrimSpace(stdout.String()) == "true"
}

// InspectByName finds and inspects a container by name.
// Returns the container if found, nil error if not found.
func (c *ExecClient) InspectByName(containerName string) (*Container, error) {
	if containerName == "" {
		return nil, fmt.Errorf("container name cannot be empty")
	}

	// Try to inspect by name (docker inspect accepts names as well as IDs)
	return c.Inspect(containerName)
}

// Exec runs a command inside a running container and returns the exit code.
// Returns 0 if the command succeeds, non-zero on failure.
func (c *ExecClient) Exec(containerName string, command []string) (int, string, error) {
	if containerName == "" {
		return -1, "", fmt.Errorf("container name cannot be empty")
	}
	if len(command) == 0 {
		return -1, "", fmt.Errorf("command cannot be empty")
	}

	args := append([]string{"exec", containerName}, command...)
	cmd := exec.CommandContext(context.Background(), "docker", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}

	if err != nil {
		// Try to get exit code from error
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), output, nil
		}
		return -1, output, fmt.Errorf("failed to exec in container %q: %w", containerName, err)
	}

	return 0, output, nil
}

// ExecShell runs a shell command inside a running container.
// Uses sh -c for Unix-like execution inside the container.
func (c *ExecClient) ExecShell(containerName string, shellCommand string) (int, string, error) {
	return c.Exec(containerName, []string{"sh", "-c", shellCommand})
}

// Ensure ExecClient implements Client interface.
var _ Client = (*ExecClient)(nil)
