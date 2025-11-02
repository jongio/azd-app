package service

import (
	"fmt"
	"os"
	"sync"
	"time"

	"app/src/internal/registry"
)

// OrchestrationResult contains the results of service orchestration.
type OrchestrationResult struct {
	Processes map[string]*ServiceProcess
	Errors    map[string]error
	StartTime time.Time
	ReadyTime time.Time
}

// OrchestrateServices starts services in dependency order with parallel execution.
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

			// Register service in starting state
			reg.Register(&registry.ServiceRegistryEntry{
				Name:       rt.Name,
				ProjectDir: projectDir,
				Port:       rt.Port,
				URL:        fmt.Sprintf("http://localhost:%d", rt.Port),
				Language:   rt.Language,
				Framework:  rt.Framework,
				Status:     "starting",
				Health:     "unknown",
				StartTime:  time.Now(),
			})

			// Resolve environment variables for this service
			serviceEnv := make(map[string]string)
			for k, v := range envVars {
				serviceEnv[k] = v
			}
			// Merge runtime-specific env
			for k, v := range rt.Env {
				serviceEnv[k] = v
			}

			// Start service
			process, err := StartService(rt, serviceEnv)
			if err != nil {
				mu.Lock()
				startErrors[rt.Name] = err
				result.Errors[rt.Name] = err
				mu.Unlock()
				reg.UpdateStatus(rt.Name, "error", "unknown")
				logger.LogService(rt.Name, fmt.Sprintf("Failed to start: %v", err))
				return
			}

			// Update registry with PID
			if entry, exists := reg.GetService(rt.Name); exists {
				entry.PID = process.Process.Pid
				reg.Register(entry)
			}

			mu.Lock()
			result.Processes[rt.Name] = process
			mu.Unlock()

			// Log service URL immediately
			fmt.Printf("Started %s: %s\n", rt.Name, fmt.Sprintf("http://localhost:%d", process.Port))
			reg.UpdateStatus(rt.Name, "running", "healthy")
			process.Ready = true

			// Keep reading output in background to prevent process from blocking
			go func() {
				ReadServiceOutput(process.Stdout, make(chan string))
			}()
			go func() {
				ReadServiceOutput(process.Stderr, make(chan string))
			}()
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
			reg.UpdateStatus(serviceName, "stopping", "unknown")

			if err := StopService(proc); err != nil {
				// Log error but continue stopping other services
				fmt.Printf("Error stopping service %s: %v\n", serviceName, err)
			}

			// Unregister from registry
			reg.Unregister(serviceName)
		}(name, process)
	}

	wg.Wait()
}

// WaitForServices waits for all services to exit.
func WaitForServices(processes map[string]*ServiceProcess) error {
	// Wait for any service to exit
	for name, process := range processes {
		if process.Process != nil {
			state, err := process.Process.Wait()
			if err != nil {
				return fmt.Errorf("service %s exited with error: %w", name, err)
			}
			if !state.Success() {
				return fmt.Errorf("service %s exited with non-zero status: %s", name, state.String())
			}
		}
	}

	return nil
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
