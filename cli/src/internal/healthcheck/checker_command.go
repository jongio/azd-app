package healthcheck

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/docker"
)

func (c *HealthChecker) tryCustomHealthCheck(ctx context.Context, config *healthCheckConfig, svc serviceInfo) *httpHealthCheckResult {
	if len(config.Test) == 0 {
		return nil
	}

	test := config.Test[0]

	// Check if it's an HTTP URL (cross-platform approach)
	if strings.HasPrefix(test, "http://") || strings.HasPrefix(test, "https://") {
		return c.performHTTPCheck(ctx, test)
	}

	// Check for CMD or CMD-SHELL format
	if len(config.Test) > 1 {
		switch config.Test[0] {
		case "CMD":
			return c.performCommandCheck(ctx, config.Test[1:], svc)
		case "CMD-SHELL":
			return c.performShellCheck(ctx, config.Test[1], svc)
		case "NONE":
			return &httpHealthCheckResult{
				Endpoint: "none",
				Status:   HealthStatusHealthy,
			}
		}
	}

	// Single string that's not a URL - treat as shell command
	return c.performShellCheck(ctx, test, svc)
}

// performHTTPCheck performs a direct HTTP health check to a specific URL.

func (c *HealthChecker) performCommandCheck(ctx context.Context, args []string, svc serviceInfo) *httpHealthCheckResult {
	if len(args) == 0 {
		return nil
	}

	startTime := time.Now()
	result := &httpHealthCheckResult{
		Endpoint:     strings.Join(args, " "),
		ResponseTime: 0,
	}

	// For container services, execute inside the container
	if svc.Type == ServiceTypeContainer {
		containerName := fmt.Sprintf("azd-%s", svc.Name)
		client := docker.NewClient()

		exitCode, output, err := client.Exec(containerName, args)
		result.ResponseTime = time.Since(startTime)

		if err != nil {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("docker exec failed: %v", err)
		} else if exitCode != 0 {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("command exited with code %d: %s", exitCode, output)
		} else {
			result.Status = HealthStatusHealthy
		}
		return result
	}

	// For native services, execute on host
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	err := cmd.Run()
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

// performShellCheck executes a shell command for health check (CMD-SHELL format).
// For container services, the command is executed inside the container using docker exec sh -c.
func (c *HealthChecker) performShellCheck(ctx context.Context, command string, svc serviceInfo) *httpHealthCheckResult {
	startTime := time.Now()
	result := &httpHealthCheckResult{
		Endpoint:     command,
		ResponseTime: 0,
	}

	// For container services, execute inside the container
	if svc.Type == ServiceTypeContainer {
		containerName := fmt.Sprintf("azd-%s", svc.Name)
		client := docker.NewClient()

		exitCode, output, err := client.ExecShell(containerName, command)
		result.ResponseTime = time.Since(startTime)

		if err != nil {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("docker exec failed: %v", err)
		} else if exitCode != 0 {
			result.Status = HealthStatusUnhealthy
			result.Error = fmt.Sprintf("command exited with code %d: %s", exitCode, output)
		} else {
			result.Status = HealthStatusHealthy
		}
		return result
	}

	// For native services, execute on host
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	err := cmd.Run()
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = fmt.Sprintf("command failed: %v", err)
	} else {
		result.Status = HealthStatusHealthy
	}

	return result
}

// tryHTTPHealthCheck attempts HTTP health checks using smart endpoint discovery.
// Uses endpoint caching to avoid spamming multiple endpoints on every check.
// Discovery only happens on first check or when cached endpoint fails.
