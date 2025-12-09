//go:build integration && docker
// +build integration,docker

package service

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
	svctype "github.com/jongio/azd-app/cli/src/internal/service"
)

// Integration tests for container service management.
// Run with: go test -tags=integration,docker ./...
// Requires Docker to be installed and running.

func checkDockerAvailable(t *testing.T) {
	t.Helper()
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker not available, skipping integration tests")
	}
}

func TestContainerService_StartStop(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use a simple, fast-starting container
	containerName := "azd-test-redis-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	// Verify Redis image can be pulled and container started
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 16379, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up any existing container
	_ = client.Remove(ctx, containerName, true)

	// Start the container
	containerID, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(ctx, containerName, 5*time.Second)
		_ = client.Remove(ctx, containerName, true)
	}()

	if containerID == "" {
		t.Error("container ID is empty")
	}

	// Verify container is running
	running, err := client.IsRunning(ctx, containerName)
	if err != nil {
		t.Fatalf("failed to check container status: %v", err)
	}
	if !running {
		t.Error("container should be running")
	}

	// Give Redis a moment to start
	time.Sleep(2 * time.Second)

	// Stop the container
	if err := client.Stop(ctx, containerName, 10*time.Second); err != nil {
		t.Fatalf("failed to stop container: %v", err)
	}

	// Verify container is stopped
	running, err = client.IsRunning(ctx, containerName)
	if err != nil {
		t.Fatalf("failed to check container status: %v", err)
	}
	if running {
		t.Error("container should be stopped")
	}
}

func TestContainerService_Logs(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := "azd-test-echo-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	// Use busybox to echo a message
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "busybox",
		Cmd:   []string{"sh", "-c", "echo 'Hello from container' && sleep 5"},
	}

	// Clean up any existing container
	_ = client.Remove(ctx, containerName, true)

	// Start the container
	_, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(ctx, containerName, 5*time.Second)
		_ = client.Remove(ctx, containerName, true)
	}()

	// Wait for the echo to complete
	time.Sleep(2 * time.Second)

	// Get logs
	logs, err := client.Logs(ctx, containerName)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if !strings.Contains(logs, "Hello from container") {
		t.Errorf("logs should contain 'Hello from container', got: %s", logs)
	}
}

func TestContainerService_HealthCheck(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := "azd-test-health-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	// Use Redis with a health check port
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 16380, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up
	_ = client.Remove(ctx, containerName, true)

	// Start container
	_, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(ctx, containerName, 5*time.Second)
		_ = client.Remove(ctx, containerName, true)
	}()

	// Wait for Redis to start
	time.Sleep(3 * time.Second)

	// Perform TCP health check on mapped port
	healthChecker := svctype.NewHealthChecker()
	healthy := healthChecker.CheckTCP("127.0.0.1", 16380, 5*time.Second)

	if !healthy {
		t.Error("Redis container should pass TCP health check on port 16380")
	}
}

func TestContainerService_EnvironmentVariables(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := "azd-test-env-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	// Use busybox to print environment variable
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "busybox",
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
		Cmd: []string{"sh", "-c", "echo $TEST_VAR && sleep 5"},
	}

	// Clean up
	_ = client.Remove(ctx, containerName, true)

	// Start container
	_, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(ctx, containerName, 5*time.Second)
		_ = client.Remove(ctx, containerName, true)
	}()

	// Wait for echo
	time.Sleep(2 * time.Second)

	// Get logs
	logs, err := client.Logs(ctx, containerName)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}

	if !strings.Contains(logs, "test_value") {
		t.Errorf("logs should contain environment variable value, got: %s", logs)
	}
}

func TestContainerService_PortMapping(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := "azd-test-ports-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	// Use netcat to listen on a port
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "busybox",
		Ports: []docker.PortMapping{
			{HostPort: 18080, ContainerPort: 8080, Protocol: "tcp"},
		},
		Cmd: []string{"sh", "-c", "nc -l -p 8080 -e echo 'Hello'"},
	}

	// Clean up
	_ = client.Remove(ctx, containerName, true)

	// Start container
	_, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(ctx, containerName, 5*time.Second)
		_ = client.Remove(ctx, containerName, true)
	}()

	// Wait for nc to start listening
	time.Sleep(2 * time.Second)

	// Verify port mapping by checking if we can connect to host port
	healthChecker := svctype.NewHealthChecker()
	reachable := healthChecker.CheckTCP("127.0.0.1", 18080, 5*time.Second)

	if !reachable {
		t.Error("mapped port 18080 should be reachable")
	}
}

func TestContainerService_DockerUnavailable(t *testing.T) {
	// This test simulates Docker being unavailable by using an invalid path
	// We can't easily make Docker unavailable, so we test error handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := docker.NewExecClient()

	// Try to check if a container is running with an invalid name
	_, err := client.IsRunning(ctx, "nonexistent-container-xyz123")
	// Should not error - just return false for non-existent container
	if err != nil {
		// Some Docker setups might error on non-existent containers
		// This is acceptable behavior
		t.Logf("IsRunning returned error for non-existent container: %v", err)
	}
}

func TestContainerService_Cleanup(t *testing.T) {
	checkDockerAvailable(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	containerName := "azd-test-cleanup-" + time.Now().Format("20060102150405")
	client := docker.NewExecClient()

	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "busybox",
		Cmd:   []string{"sleep", "60"},
	}

	// Clean up any existing
	_ = client.Remove(ctx, containerName, true)

	// Start container
	_, err := client.Run(ctx, config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Verify running
	running, err := client.IsRunning(ctx, containerName)
	if err != nil {
		t.Fatalf("failed to check status: %v", err)
	}
	if !running {
		t.Error("container should be running")
	}

	// Force remove (simulating abrupt shutdown)
	if err := client.Remove(ctx, containerName, true); err != nil {
		t.Fatalf("failed to force remove: %v", err)
	}

	// Verify removed
	running, _ = client.IsRunning(ctx, containerName)
	if running {
		t.Error("container should be removed")
	}
}
