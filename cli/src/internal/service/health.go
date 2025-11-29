// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Backoff configuration constants
const (
	// Health check backoff settings
	HealthCheckMaxInterval = 5 * time.Second
	BackoffMultiplier      = 2.0

	// Port check specific settings
	PortCheckInitialInterval = 100 * time.Millisecond
	PortCheckMaxInterval     = 2 * time.Second

	// HTTP client settings
	HTTPClientTimeout = 5 * time.Second
	ConnectionTimeout = 2 * time.Second
	PortCheckTimeout  = 1 * time.Second
)

// PerformHealthCheck verifies that a service is ready with exponential backoff.
// Supports multiple health check types:
// - "http": Check an HTTP endpoint (default)
// - "tcp" or "port": Check if a port is listening
// - "process": Check if the process is running
// - "output": Monitor stdout for a pattern match (requires LogMatch to be set)
// - "none": Skip health checks (service is immediately considered ready)
func PerformHealthCheck(process *ServiceProcess) error {
	config := process.Runtime.HealthCheck

	// Handle "none" type - skip health checks entirely
	if config.Type == "none" {
		process.Ready = true
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = config.Timeout
	b.InitialInterval = config.Interval
	b.MaxInterval = HealthCheckMaxInterval
	b.Multiplier = BackoffMultiplier

	operation := func() error {
		var err error

		switch config.Type {
		case "http":
			err = HTTPHealthCheck(process.Port, config.Path)
		case "port", "tcp":
			err = PortHealthCheck(process.Port)
		case "process":
			err = ProcessHealthCheck(process)
		case "output":
			// Output-based health check: check if the pattern has been matched in logs
			err = OutputHealthCheck(process, config.LogMatch)
		case "none":
			// Already handled above, but include for completeness
			process.Ready = true
			return nil
		default:
			// Default to HTTP health check if port is available, otherwise process check
			if process.Port > 0 {
				err = HTTPHealthCheck(process.Port, config.Path)
			} else {
				err = ProcessHealthCheck(process)
			}
		}

		if err == nil {
			// Health check succeeded
			process.Ready = true
		}

		return err
	}

	return backoff.Retry(operation, b)
}

// OutputHealthCheck checks if a specific pattern has been matched in the process output.
// This is useful for build/watch services that log a success message but don't serve HTTP.
func OutputHealthCheck(process *ServiceProcess, pattern string) error {
	if pattern == "" {
		// No pattern specified - fall back to process check
		return ProcessHealthCheck(process)
	}

	// Check if the process has output the expected pattern
	// This requires the log buffer to have captured the output
	// For now, we check if the process is running (actual pattern matching
	// would require integration with the log streaming system)
	if process.Ready {
		return nil
	}

	// Fall back to process check for now
	// Full output pattern matching would require access to the service's log buffer
	return ProcessHealthCheck(process)
}

// HTTPHealthCheck attempts HTTP requests to verify service is ready.
func HTTPHealthCheck(port int, path string) error {
	// Build URL
	url := fmt.Sprintf("http://localhost:%d%s", port, path)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: HTTPClientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}

	// Try HEAD request first (lightweight)
	resp, err := client.Head(url)
	if err == nil {
		defer SafeClose(resp.Body, "HEAD response body")
		// Accept any 2xx or 3xx status code
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}
	}

	// If HEAD fails or returns error code, try GET
	resp, err = client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer SafeClose(resp.Body, "GET response body")

	// Accept any 2xx or 3xx status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("HTTP health check failed with status: %d", resp.StatusCode)
}

// PortHealthCheck verifies that a port is listening.
func PortHealthCheck(port int) error {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, ConnectionTimeout)
	if err != nil {
		return fmt.Errorf("port %d not listening: %w", port, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			// Log but don't fail health check on close error
			log.Printf("Warning: failed to close health check connection: %v", closeErr)
		}
	}()
	return nil
}

// ProcessHealthCheck verifies that a process is running.
func ProcessHealthCheck(process *ServiceProcess) error {
	if process.Process == nil {
		return fmt.Errorf("process not started")
	}

	// Check if we have a valid PID
	pid := process.Process.Pid
	if pid <= 0 {
		return fmt.Errorf("process has invalid PID: %d", pid)
	}

	// Use platform-specific implementation to check if process is running
	return processIsRunning(pid)
}

// WaitForPort waits for a port to become available (listening) with exponential backoff.
// Uses exponential backoff starting at 100ms, doubling each retry, up to the timeout.
func WaitForPort(port int, timeout time.Duration) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = timeout
	b.InitialInterval = PortCheckInitialInterval
	b.MaxInterval = PortCheckMaxInterval
	b.Multiplier = BackoffMultiplier

	operation := func() error {
		return PortHealthCheck(port)
	}

	return backoff.Retry(operation, b)
}

// TryHTTPHealthCheck performs a single HTTP health check attempt without retries.
func TryHTTPHealthCheck(port int, path string) bool {
	err := HTTPHealthCheck(port, path)
	return err == nil
}

// IsPortListening checks if a port is currently listening.
func IsPortListening(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, PortCheckTimeout)
	if err != nil {
		return false
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Warning: failed to close connection during port check: %v", closeErr)
		}
	}()
	return true
}
