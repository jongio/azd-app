package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// PerformHealthCheck verifies that a service is ready.
func PerformHealthCheck(process *ServiceProcess) error {
	config := process.Runtime.HealthCheck

	startTime := time.Now()
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	timeout := time.After(config.Timeout)

	for {
		select {
		case <-timeout:
			elapsed := time.Since(startTime)
			return fmt.Errorf("health check timed out after %v", elapsed.Round(time.Second))

		case <-ticker.C:
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
				return nil
			}

			// Health check failed, will retry after interval
		}
	}
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
	if err := conn.Close(); err != nil {
		// Log but don't fail health check on close error
		log.Printf("Warning: failed to close health check connection: %v", err)
	}
	return nil
}

// ProcessHealthCheck verifies that a process is running.
func ProcessHealthCheck(process *ServiceProcess) error {
	if process.Process == nil {
		return fmt.Errorf("process not started")
	}

	// Check if process is still running
	// On Unix, sending signal 0 checks existence without affecting the process
	// On Windows, this will return an error if process doesn't exist
	err := process.Process.Signal(nil)
	if err != nil {
		return fmt.Errorf("process not running: %w", err)
	}

	return nil
}

// WaitForPort waits for a port to become available (listening).
func WaitForPort(port int, timeout time.Duration) error {
	startTime := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			elapsed := time.Since(startTime)
			return fmt.Errorf("port %d not available after %v", port, elapsed.Round(time.Second))

		case <-ticker.C:
			if err := PortHealthCheck(port); err == nil {
				return nil
			}
		}
	}
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
	if err := conn.Close(); err != nil {
		log.Printf("Warning: failed to close connection during port check: %v", err)
	}
	return true
}
