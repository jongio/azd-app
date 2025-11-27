// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"fmt"
	"log/slog"
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
	Processes       map[string]*ServiceProcess
	Errors          map[string]error
	StartTime       time.Time
	ReadyTime       time.Time
	FunctionsParser *FunctionsOutputParser // Parser for Functions endpoints
}

// OrchestrateServices starts services in dependency order with parallel execution.
//
// This function orchestrates the startup of multiple services concurrently while ensuring
// proper environment variable inheritance and process isolation.
//
// Parameters:
//   - runtimes: Slice of ServiceRuntime definitions containing service metadata
//   - envVars: Additional environment variables (e.g., from --env-file)
//   - logger: ServiceLogger for structured logging of orchestration events
//
// Environment Inheritance:
// All services automatically inherit azd context from os.Environ() including:
//   - AZD_SERVER: gRPC server address for azd extension framework communication
//   - AZD_ACCESS_TOKEN: Authentication token for azd APIs
//   - AZURE_*: All Azure environment variables from azd env
//
// Returns:
//   - OrchestrationResult: Contains started processes, errors, and timing information
//   - error: Non-nil if any service fails to start; all services are stopped on error
//
// Process Isolation:
// Each service runs in a separate goroutine with panic recovery to prevent cascading failures.
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

	// Create Functions output parser
	functionsParser := NewFunctionsOutputParser(false)
	result.FunctionsParser = functionsParser

	// Start all services in parallel
	projectDir, _ := os.Getwd()
	reg := registry.GetRegistry(projectDir)

	slog.Debug("starting service orchestration",
		slog.Int("service_count", len(runtimes)))

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
			envKey := EnvServiceURLPrefix + serviceNameUpper + EnvServiceURLSuffix
			if url, exists := envVars[envKey]; exists {
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

			// Inject FUNCTIONS_WORKER_RUNTIME for Logic Apps if missing
			// This prevents func CLI from prompting interactively
			serviceEnv = InjectFunctionsWorkerRuntime(serviceEnv, rt)

			// Reserve port to prevent TOCTOU race condition
			// This holds the port open until we're ready to start the service
			portMgr := portmanager.GetPortManager(projectDir)
			reservation, portErr := portMgr.ReservePort(rt.Port)
			if portErr != nil {
				mu.Lock()
				err := fmt.Errorf("port %d is no longer available (taken by another process): %w", rt.Port, portErr)
				startErrors[rt.Name] = err
				result.Errors[rt.Name] = err
				mu.Unlock()
				if err := reg.UpdateStatus(rt.Name, "error", "unknown"); err != nil {
					logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", err))
				}
				logger.LogService(rt.Name, fmt.Sprintf("❌ Port %d conflict detected", rt.Port))
				return
			}

			// Release port reservation immediately before starting service
			// The service must bind quickly after this to avoid a new race
			if err := reservation.Release(); err != nil {
				slog.Debug("failed to release port reservation", "port", rt.Port, "error", err)
			}

			// Start service
			process, err := StartService(rt, serviceEnv, projectDir, functionsParser)
			if err != nil {
				mu.Lock()
				startErrors[rt.Name] = err
				result.Errors[rt.Name] = err
				mu.Unlock()
				slog.Error("failed to start service",
					slog.String("service", rt.Name),
					slog.Int("port", rt.Port),
					slog.String("error", err.Error()))
				if err := reg.UpdateStatus(rt.Name, "error", "unknown"); err != nil {
					logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", err))
				}
				logger.LogService(rt.Name, fmt.Sprintf("Failed to start: %v", err))
				return
			}

			pid := 0
			if process.Process != nil {
				pid = process.Process.Pid
			}
			slog.Debug("service started",
				slog.String("service", rt.Name),
				slog.Int("port", rt.Port),
				slog.Int("pid", pid),
				slog.String("language", rt.Language),
				slog.String("framework", rt.Framework))

			// Update registry with PID
			if entry, exists := reg.GetService(rt.Name); exists {
				if process.Process != nil {
					entry.PID = process.Process.Pid
				}
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

	// Wait for all services to finish starting with timeout
	// Use context with timeout to prevent indefinite blocking
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout (5 minutes default)
	select {
	case <-done:
		// All services started successfully
	case <-time.After(DefaultServiceStartTimeout):
		slog.Warn("service orchestration timeout",
			slog.Duration("timeout", DefaultServiceStartTimeout),
			slog.Int("started", len(result.Processes)))
		StopAllServices(result.Processes)
		return result, fmt.Errorf("service orchestration timed out after %v", DefaultServiceStartTimeout)
	}

	slog.Debug("service orchestration complete",
		slog.Int("started", len(result.Processes)),
		slog.Int("failed", len(startErrors)))

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

// StopAllServices stops all running services with graceful shutdown.
//
// This function coordinates the shutdown of multiple services concurrently,
// ensuring each service has an opportunity for graceful termination before
// forceful termination.
//
// Parameters:
//   - processes: Map of service names to ServiceProcess instances
//
// Behavior:
//   - Each service is stopped in parallel via goroutine
//   - Graceful shutdown timeout: 5 seconds per service
//   - Services are unregistered from the service registry
//   - Errors are logged but don't prevent other services from stopping
//   - Function blocks until all services have stopped
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

			// Stop service with graceful timeout
			if err := StopServiceGraceful(proc, DefaultStopTimeout); err != nil {
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

// ValidateOrchestration validates that all services started successfully and are ready.
//
// This function checks the orchestration result to ensure:
//   - No errors occurred during service startup
//   - All services transitioned to ready state
//
// Parameters:
//   - result: OrchestrationResult from OrchestrateServices
//
// Returns:
//   - nil if all services are ready
//   - error describing the validation failure
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
