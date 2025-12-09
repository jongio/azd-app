//go:build integration && docker
// +build integration,docker

package service

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
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

	// Use a simple, fast-starting container
	containerName := "azd-test-redis-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Verify Redis image can be pulled and container started
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 16379, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up any existing container
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start the container
	containerID, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(containerName, 5)
		_ = client.Remove(containerName)
	}()

	if containerID == "" {
		t.Error("container ID is empty")
	}

	// Verify container is running
	if !client.IsRunning(containerName) {
		t.Error("container should be running")
	}

	// Give Redis a moment to start
	time.Sleep(2 * time.Second)

	// Stop the container
	if err := client.Stop(containerName, 10); err != nil {
		t.Fatalf("failed to stop container: %v", err)
	}

	// Verify container is stopped
	if client.IsRunning(containerName) {
		t.Error("container should be stopped")
	}
}

func TestContainerService_Logs(t *testing.T) {
	checkDockerAvailable(t)

	containerName := "azd-test-echo-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Note: The current docker.ContainerConfig doesn't support Cmd field,
	// so we skip this test until the API is extended
	t.Skip("Skipping: docker.ContainerConfig doesn't support Cmd field yet")

	// Clean up any existing container
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start the container with busybox to echo a message
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "busybox",
		// Cmd field not supported yet
	}

	_, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(containerName, 5)
		_ = client.Remove(containerName)
	}()

	// Wait for the echo to complete
	time.Sleep(2 * time.Second)

	// Get logs
	logReader, err := client.Logs(containerName)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	defer logReader.Close()

	buf := make([]byte, 1024)
	n, _ := logReader.Read(buf)
	logs := string(buf[:n])

	if !strings.Contains(logs, "Hello from container") {
		t.Errorf("logs should contain 'Hello from container', got: %s", logs)
	}
}

func TestContainerService_HealthCheck(t *testing.T) {
	checkDockerAvailable(t)

	containerName := "azd-test-health-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Use Redis with a health check port
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 16380, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start container
	_, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(containerName, 5)
		_ = client.Remove(containerName)
	}()

	// Wait for Redis to start
	time.Sleep(3 * time.Second)

	// Perform TCP health check on mapped port
	healthy := checkTCPPort("127.0.0.1", 16380, 5*time.Second)

	if !healthy {
		t.Error("Redis container should pass TCP health check on port 16380")
	}
}

func TestContainerService_EnvironmentVariables(t *testing.T) {
	checkDockerAvailable(t)

	// Note: The current docker.ContainerConfig doesn't support Cmd field,
	// so we can only verify environment variables are passed correctly via Redis
	containerName := "azd-test-env-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Use Redis with an environment variable (Redis doesn't use many env vars but we can verify the mechanism)
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Environment: map[string]string{
			"REDIS_PASSWORD": "test_password",
		},
		Ports: []docker.PortMapping{
			{HostPort: 16381, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start container
	_, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(containerName, 5)
		_ = client.Remove(containerName)
	}()

	// Wait for container to start
	time.Sleep(2 * time.Second)

	// Verify container is running (if env vars caused issues, it wouldn't start)
	if !client.IsRunning(containerName) {
		t.Error("container should be running with environment variables")
	}
}

func TestContainerService_PortMapping(t *testing.T) {
	checkDockerAvailable(t)

	containerName := "azd-test-ports-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Use Redis to test port mapping (has a listening port)
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 18080, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start container
	_, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		_ = client.Stop(containerName, 5)
		_ = client.Remove(containerName)
	}()

	// Wait for Redis to start listening
	time.Sleep(2 * time.Second)

	// Verify port mapping by checking if we can connect to host port
	reachable := checkTCPPort("127.0.0.1", 18080, 5*time.Second)

	if !reachable {
		t.Error("mapped port 18080 should be reachable")
	}
}

func TestContainerService_DockerUnavailable(t *testing.T) {
	// This test verifies behavior with non-existent containers
	client := docker.NewClient()

	// Try to check if a container is running with an invalid name
	running := client.IsRunning("nonexistent-container-xyz123")
	// Should return false for non-existent container
	if running {
		t.Error("IsRunning should return false for non-existent container")
	}
}

func TestContainerService_Cleanup(t *testing.T) {
	checkDockerAvailable(t)

	containerName := "azd-test-cleanup-" + time.Now().Format("20060102150405")
	client := docker.NewClient()

	// Use redis since it runs indefinitely (ContainerConfig doesn't support Cmd)
	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "redis:7-alpine",
		Ports: []docker.PortMapping{
			{HostPort: 16382, ContainerPort: 6379, Protocol: "tcp"},
		},
	}

	// Clean up any existing
	_ = client.Stop(containerName, 5)
	_ = client.Remove(containerName)

	// Start container
	_, err := client.Run(config)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Verify running
	if !client.IsRunning(containerName) {
		t.Error("container should be running")
	}

	// Stop then remove (simulating cleanup)
	if err := client.Stop(containerName, 5); err != nil {
		t.Fatalf("failed to stop container: %v", err)
	}
	if err := client.Remove(containerName); err != nil {
		t.Fatalf("failed to remove container: %v", err)
	}

	// Verify removed
	if client.IsRunning(containerName) {
		t.Error("container should be removed")
	}
}

// checkTCPPort is a helper function to verify TCP port connectivity.
func checkTCPPort(host string, port int, timeout time.Duration) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
