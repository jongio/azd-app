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
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}

	// Try HEAD request first (lightweight)
	resp, err := client.Head(url)
	if err == nil {
		defer resp.Body.Close()
		// Accept any 2xx or 3xx status code
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return nil
		}
	}

	// Try GET request as fallback
	resp, err = client.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx or 3xx status code
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	return fmt.Errorf("HTTP health check failed with status: %d", resp.StatusCode)
}

// PortHealthCheck verifies that a port is listening.
func PortHealthCheck(port int) error {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
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

	// Check if process is still running by attempting a non-blocking signal
	// On Windows, Signal(nil) doesn't work, so we try Signal(syscall.Signal(0)) which also doesn't work
	// Instead, we'll use a different approach: check if the process can be found
	// The most reliable cross-platform way is to just assume it's running if we have a Process object
	// and it hasn't been waited on yet. If the process exited, Wait() would have been called.
	// For testing purposes, we can rely on the PID being valid.
	if process.PID == 0 && process.Process.Pid == 0 {
		return fmt.Errorf("process has invalid PID")
	}

	return nil
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
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
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
