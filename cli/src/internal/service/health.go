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
func PerformHealthCheck(process *ServiceProcess) error {
	config := process.Runtime.HealthCheck

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
		case "port":
			err = PortHealthCheck(process.Port)
		case "process":
			err = ProcessHealthCheck(process)
		default:
			// Default to HTTP health check
			err = HTTPHealthCheck(process.Port, config.Path)
		}

		if err == nil {
			// Health check succeeded
			process.Ready = true
		}

		return err
	}

	return backoff.Retry(operation, b)
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
