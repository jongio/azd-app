package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ExecClient implements the Client interface using the docker CLI.
type ExecClient struct{}

// NewClient creates a new Docker client that uses the docker CLI.
func NewClient() *ExecClient {
	return &ExecClient{}
}

// IsAvailable checks if Docker is installed and running.
func (c *ExecClient) IsAvailable() bool {
	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	return err == nil
}

// Pull downloads an image if not present locally.
func (c *ExecClient) Pull(image string) error {
	if err := ValidateImageName(image); err != nil {
		return err
	}

	cmd := exec.Command("docker", "pull", image)
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

// Run creates and starts a container with the given configuration.
func (c *ExecClient) Run(config ContainerConfig) (string, error) {
	if err := config.Validate(); err != nil {
		return "", err
	}

	args := buildRunArgs(config)
	cmd := exec.Command("docker", args...)

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

	// Add port mappings
	for _, port := range config.Ports {
		args = append(args, "-p", formatPortMapping(port))
	}

	// Add environment variables
	for key, value := range config.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add image
	args = append(args, config.Image)

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
		return fmt.Errorf("container ID cannot be empty")
	}

	args := []string{"stop", "-t", fmt.Sprintf("%d", timeoutSeconds), containerID}
	cmd := exec.Command("docker", args...)

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

// Remove removes a container.
func (c *ExecClient) Remove(containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	cmd := exec.Command("docker", "rm", containerID)
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
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	cmd := exec.Command("docker", "logs", "-f", containerID)
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

	// Create pipe to combine stdout and stderr concurrently
	// io.MultiReader reads sequentially which blocks on stdout in follow mode
	pr, pw := io.Pipe()

	// Copy both streams concurrently to the pipe writer
	go func() {
		var wg sync.WaitGroup
		wg.Add(2)

		// Copy stdout
		go func() {
			defer wg.Done()
			_, _ = io.Copy(pw, stdout)
		}()

		// Copy stderr
		go func() {
			defer wg.Done()
			_, _ = io.Copy(pw, stderr)
		}()

		// Wait for both to complete, then close the writer
		wg.Wait()
		pw.Close()
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

func (c *combinedReadCloser) Close() error {
	// Close the pipe writer to unblock any pending reads
	if c.pipe != nil {
		c.pipe.Close()
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
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	cmd := exec.Command("docker", "inspect", containerID)
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

	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerID)
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
	cmd := exec.Command("docker", args...)

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
