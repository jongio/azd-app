// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/output"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-app/cli/src/internal/registry"
)

// OrchestrationResult contains the results of service orchestration.
type OrchestrationResult struct {
	Processes map[string]*ServiceProcess
	Errors    map[string]error
	StartTime time.Time
	ReadyTime time.Time
}

// OrchestrateServices starts services in dependency order with parallel execution.
// The envVars parameter contains additional environment variables (e.g., from --env-file).
// All services automatically inherit azd context from os.Environ() including:
// - AZD_SERVER: gRPC server address for azd extension framework communication
// - AZD_ACCESS_TOKEN: Authentication token for azd APIs
// - AZURE_*: All Azure environment variables from azd env
func OrchestrateServices(runtimes []*ServiceRuntime, envVars map[string]string, logger *ServiceLogger) (*OrchestrationResult, error) {
	result := &OrchestrationResult{
		Processes: make(map[string]*ServiceProcess),
		Errors:    make(map[string]error),
		StartTime: time.Now(),
	}

	// Create a map of service name to runtime for quick lookup
	runtimeMap := make(map[string]*ServiceRuntime)
	for _, rt := range runtimes {
		runtimeMap[rt.Name] = rt
	}

	// Start all services in parallel
	projectDir, _ := os.Getwd()
	reg := registry.GetRegistry(projectDir)

	var mu sync.Mutex
	var wg sync.WaitGroup
	startErrors := make(map[string]error)

	for _, runtime := range runtimes {
		wg.Add(1)
		go func(rt *ServiceRuntime) {
			defer wg.Done()

			// Extract Azure URL from environment variables if available
			azureURL := ""
			serviceNameUpper := strings.ToUpper(rt.Name)
			if url, exists := envVars["SERVICE_"+serviceNameUpper+"_URL"]; exists {
				azureURL = url
			}

			// Register service in starting state
			if err := reg.Register(&registry.ServiceRegistryEntry{
				Name:       rt.Name,
				ProjectDir: projectDir,
				Port:       rt.Port,
				URL:        fmt.Sprintf("http://localhost:%d", rt.Port),
				AzureURL:   azureURL,
				Language:   rt.Language,
				Framework:  rt.Framework,
				Status:     "starting",
				Health:     "unknown",
				StartTime:  time.Now(),
			}); err != nil {
				logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to register service: %v", err))
			}

			// Resolve environment variables for this service
			// Start with os.Environ() to inherit azd context (AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)
			serviceEnv := make(map[string]string)
			azureVarCount := 0
			for _, e := range os.Environ() {
				pair := strings.SplitN(e, "=", 2)
				if len(pair) == 2 {
					serviceEnv[pair[0]] = pair[1]
					// Count AZURE_* and AZD_* variables for debugging
					if strings.HasPrefix(pair[0], "AZURE_") || strings.HasPrefix(pair[0], "AZD_") {
						azureVarCount++
					}
				}
			}
			// Merge custom environment variables from --env-file
			for k, v := range envVars {
				serviceEnv[k] = v
			}
			// Merge runtime-specific env (highest priority)
			for k, v := range rt.Env {
				serviceEnv[k] = v
			}

			// Final port availability check before starting service
			// This catches race conditions where port became unavailable between detection and start
			portMgr := portmanager.GetPortManager(projectDir)
			if !portMgr.IsPortAvailable(rt.Port) {
				mu.Lock()
				err := fmt.Errorf("port %d is no longer available (taken by another process)", rt.Port)
				startErrors[rt.Name] = err
				result.Errors[rt.Name] = err
				mu.Unlock()
				if err := reg.UpdateStatus(rt.Name, "error", "unknown"); err != nil {
					logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", err))
				}
				logger.LogService(rt.Name, fmt.Sprintf("❌ Port %d conflict detected", rt.Port))
				return
			}

			// Start service
			process, err := StartService(rt, serviceEnv, projectDir)
			if err != nil {
				mu.Lock()
				startErrors[rt.Name] = err
				result.Errors[rt.Name] = err
				mu.Unlock()
				if err := reg.UpdateStatus(rt.Name, "error", "unknown"); err != nil {
					logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", err))
				}
				logger.LogService(rt.Name, fmt.Sprintf("Failed to start: %v", err))
				return
			}

			// Update registry with PID
			if entry, exists := reg.GetService(rt.Name); exists {
				entry.PID = process.Process.Pid
				if err := reg.Register(entry); err != nil {
					logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update registry with PID: %v", err))
				}
			}

			mu.Lock()
			result.Processes[rt.Name] = process
			mu.Unlock()

			// Log service URL immediately with modern formatting
			url := fmt.Sprintf("http://localhost:%d", process.Port)
			output.ItemSuccess("%s%-15s%s → %s", output.Cyan, rt.Name, output.Reset, url)

			if err := reg.UpdateStatus(rt.Name, "running", "healthy"); err != nil {
				logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", err))
			}
			process.Ready = true

			// Note: Log collection is already handled by StartLogCollection in StartService
			// which sets up goroutines to read from stdout/stderr and populate the log buffer
		}(runtime)
	}

	// Wait for all services to finish starting
	wg.Wait()

	// Check if any services failed to start
	if len(startErrors) > 0 {
		StopAllServices(result.Processes)
		// Return the first error encountered
		for name, err := range startErrors {
			return result, fmt.Errorf("failed to start service %s: %w", name, err)
		}
	}

	result.ReadyTime = time.Now()
	return result, nil
}

// StopAllServices stops all running services.
func StopAllServices(processes map[string]*ServiceProcess) {
	var wg sync.WaitGroup
	projectDir, _ := os.Getwd()
	reg := registry.GetRegistry(projectDir)

	for name, process := range processes {
		wg.Add(1)
		go func(serviceName string, proc *ServiceProcess) {
			defer wg.Done()

			// Update status to stopping
			if err := reg.UpdateStatus(serviceName, "stopping", "unknown"); err != nil {
				output.Error("Warning: failed to update status for %s: %v", serviceName, err)
			}

			if err := StopService(proc); err != nil {
				// Log error but continue stopping other services
				output.Error("Error stopping service %s: %v", serviceName, err)
			}

			// Unregister from registry
			if err := reg.Unregister(serviceName); err != nil {
				output.Error("Warning: failed to unregister service %s: %v", serviceName, err)
			}
		}(name, process)
	}

	wg.Wait()
}

// WaitForServices waits for the first service to exit.
// Returns immediately when any service process exits, which triggers shutdown of all services.
func WaitForServices(processes map[string]*ServiceProcess) error {
	// Count valid processes with running Process objects
	validProcessCount := 0
	for _, process := range processes {
		if process.Process != nil {
			validProcessCount++
		}
	}

	// If no valid processes, return immediately
	if validProcessCount == 0 {
		return nil
	}

	// Create error channel to collect exit status from any service
	errCh := make(chan error, 1)

	// Start a goroutine for each service to wait for it to exit
	for name, process := range processes {
		if process.Process == nil {
			continue
		}

		go func(serviceName string, proc *ServiceProcess) {
			state, err := proc.Process.Wait()
			if err != nil {
				errCh <- fmt.Errorf("service %s exited with error: %w", serviceName, err)
				return
			}
			if !state.Success() {
				errCh <- fmt.Errorf("service %s exited with non-zero status: %s", serviceName, state.String())
				return
			}
			// Service exited cleanly
			errCh <- nil
		}(name, process)
	}

	// Return when the first process exits (either with error or success)
	return <-errCh
}

// GetServiceURLs generates URLs for all running services.
func GetServiceURLs(processes map[string]*ServiceProcess) map[string]string {
	urls := make(map[string]string)

	for name, process := range processes {
		if process.Ready {
			urls[name] = fmt.Sprintf("http://localhost:%d", process.Port)
		}
	}

	return urls
}

// ValidateOrchestration validates that all services are ready.
func ValidateOrchestration(result *OrchestrationResult) error {
	if len(result.Errors) > 0 {
		return fmt.Errorf("orchestration failed with %d errors", len(result.Errors))
	}

	for name, process := range result.Processes {
		if !process.Ready {
			return fmt.Errorf("service %s is not ready", name)
		}
	}

	return nil
}
