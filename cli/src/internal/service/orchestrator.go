// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jongio/azd-app/cli/src/internal/constants"
	"github.com/jongio/azd-app/cli/src/internal/portmanager"
	"github.com/jongio/azd-core/cliout"
	"github.com/jongio/azd-core/registry"
)

// OrchestrationResult contains the results of service orchestration.
type OrchestrationResult struct {
	Processes       map[string]*ServiceProcess
	Errors          map[string]error
	StartTime       time.Time
	ReadyTime       time.Time
	FunctionsParser *FunctionsOutputParser // Parser for Functions endpoints
}

// DefaultHealthWaitTimeout is the maximum time to wait for a service to become healthy.
const DefaultHealthWaitTimeout = 2 * time.Minute

// OrchestrateServices starts services in dependency order with parallel execution.
//
// This function orchestrates the startup of multiple services concurrently while ensuring
// proper environment variable inheritance, process isolation, and dependency ordering.
//
// Parameters:
//   - runtimes: Slice of ServiceRuntime definitions containing service metadata
//   - services: Map of service definitions from azure.yaml (for dependency information)
//   - envVars: Additional environment variables (e.g., from --env-file)
//   - logger: ServiceLogger for structured logging of orchestration events
//   - restartContainers: If true, restart containers even if already running; if false, reuse existing running containers
//
// Environment Inheritance:
// All services automatically inherit azd context from os.Environ() including:
//   - AZD_SERVER: gRPC server address for azd extension framework communication
//   - AZD_ACCESS_TOKEN: Authentication token for azd APIs
//   - AZURE_*: All Azure environment variables from azd env
//
// Dependency Ordering:
// Services are started in dependency order based on the 'uses' field in azure.yaml:
//   - Services with no dependencies start first (level 0)
//   - Services depending on level 0 start after those are healthy (level 1)
//   - And so on...
//   - Services within the same level start in parallel
//
// Returns:
//   - OrchestrationResult: Contains started processes, errors, and timing information
//   - error: Non-nil if any service fails to start; all services are stopped on error
//
// Process Isolation:
// Each service runs in a separate goroutine with panic recovery to prevent cascading failures.
func OrchestrateServices(ctx context.Context, runtimes []*ServiceRuntime, services map[string]Service, envVars map[string]string, logger *ServiceLogger, restartContainers bool) (*OrchestrationResult, error) {
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

	// Build dependency graph and get startup levels
	graph, err := BuildDependencyGraph(services, nil)
	if err != nil {
		return result, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	levels := TopologicalSort(graph)
	if len(levels) == 0 {
		// No services to start
		return result, nil
	}

	slog.Debug("starting service orchestration",
		slog.Int("service_count", len(runtimes)),
		slog.Int("dependency_levels", len(levels)))

	// Start services level by level
	projectDir, _ := os.Getwd()
	reg := registry.GetRegistry(projectDir)

	for levelIdx, levelServices := range levels {
		slog.Debug("starting dependency level",
			slog.Int("level", levelIdx),
			slog.Int("services", len(levelServices)))

		// Start all services in this level in parallel
		var mu sync.Mutex
		var wg sync.WaitGroup
		levelErrors := make(map[string]error)
		levelProcesses := make(map[string]*ServiceProcess)

		for _, serviceName := range levelServices {
			rt, exists := runtimeMap[serviceName]
			if !exists {
				// Service not in runtimes (might be filtered out)
				continue
			}

			wg.Add(1)
			go func(rt *ServiceRuntime) {
				defer wg.Done()

				process, startErr := startSingleService(ctx, rt, envVars, reg, logger, projectDir, restartContainers, functionsParser)

				mu.Lock()
				if startErr != nil {
					levelErrors[rt.Name] = startErr
					result.Errors[rt.Name] = startErr
				} else {
					levelProcesses[rt.Name] = process
					result.Processes[rt.Name] = process
				}
				mu.Unlock()
			}(rt)
		}

		// Wait for all services in this level to start with timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All services in level started
		case <-time.After(DefaultServiceStartTimeout):
			slog.Warn("service orchestration timeout at level",
				slog.Int("level", levelIdx),
				slog.Duration("timeout", DefaultServiceStartTimeout))
			StopAllServices(result.Processes)
			return result, fmt.Errorf("service orchestration timed out at level %d after %v", levelIdx, DefaultServiceStartTimeout)
		}

		// Check if any services failed to start in this level
		if len(levelErrors) > 0 {
			StopAllServices(result.Processes)
			for name, err := range levelErrors {
				return result, fmt.Errorf("failed to start service %s: %w", name, err)
			}
		}

		// Wait for all services in this level to become healthy before starting next level
		// (only if there are more levels to start)
		if levelIdx < len(levels)-1 {
			for serviceName, process := range levelProcesses {
				svc, svcExists := services[serviceName]
				if !svcExists {
					continue
				}

				if err := waitForServiceHealthy(serviceName, process, &svc, DefaultHealthWaitTimeout); err != nil {
					StopAllServices(result.Processes)
					return result, fmt.Errorf("service %s failed health check: %w", serviceName, err)
				}
			}

			slog.Debug("dependency level healthy, proceeding to next level",
				slog.Int("level", levelIdx))
		}
	}

	slog.Debug("service orchestration complete",
		slog.Int("started", len(result.Processes)),
		slog.Int("failed", len(result.Errors)))

	result.ReadyTime = time.Now()
	return result, nil
}

// startSingleService starts a single service and returns the process.
// This is extracted from the original OrchestrateServices to be reused for level-based startup.
func startSingleService(ctx context.Context, rt *ServiceRuntime, envVars map[string]string, reg *registry.ServiceRegistry, logger *ServiceLogger, projectDir string, restartContainers bool, functionsParser *FunctionsOutputParser) (*ServiceProcess, error) {
	// Extract Azure URL from environment variables if available
	azureURL := ""
	serviceNameUpper := strings.ToUpper(rt.Name)
	envKey := EnvServiceURLPrefix + serviceNameUpper + EnvServiceURLSuffix
	if url, exists := envVars[envKey]; exists {
		azureURL = url
	}

	// Register service in starting state
	// Only set URL if port is assigned (port > 0)
	serviceURL := ""
	if rt.Port > 0 {
		serviceURL = fmt.Sprintf("http://localhost:%d", rt.Port)
	}
	if err := reg.Register(&registry.ServiceRegistryEntry{
		Name:       rt.Name,
		ProjectDir: projectDir,
		Port:       rt.Port,
		URL:        serviceURL,
		AzureURL:   azureURL,
		Language:   rt.Language,
		Framework:  rt.Framework,
		Status:     constants.StatusStarting,
		StartTime:  time.Now(),
		Type:       rt.Type,
		Mode:       rt.Mode,
	}); err != nil {
		logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to register service: %v", err))
	}

	// Resolve environment variables for this service using the centralized ResolveEnvironment function.
	// This handles:
	// - OS environment inheritance (including azd context: AZD_SERVER, AZD_ACCESS_TOKEN, AZURE_*)
	// - Custom environment variables from --env-file
	// - Service-specific environment from azure.yaml
	// - Azure Key Vault reference resolution (automatic)
	// - Variable substitution
	//
	// We need to create a dummy Service to pass runtime.Env to ResolveEnvironment
	dummyService := Service{
		Environment: rt.Env,
	}

	// Create context for Key Vault resolution (with reasonable timeout)
	resolveCtx, resolveCancel := context.WithTimeout(ctx, 30*time.Second)
	defer resolveCancel()

	serviceEnv, resolveErr := ResolveEnvironment(resolveCtx, dummyService, make(map[string]string), "", envVars)
	if resolveErr != nil {
		slog.Warn("environment resolution warning",
			slog.String("service", rt.Name),
			slog.String("error", resolveErr.Error()))
		// Continue with degraded environment - warnings already logged by ResolveEnvironment
	}

	// Inject FUNCTIONS_WORKER_RUNTIME for Logic Apps if missing
	// This prevents func CLI from prompting interactively
	serviceEnv = InjectFunctionsWorkerRuntime(serviceEnv, rt)

	// Ensure runtime-assigned port and service name propagate to the process
	if serviceEnv == nil {
		serviceEnv = map[string]string{}
	}
	if rt.Port > 0 {
		portValue := strconv.Itoa(rt.Port)
		serviceEnv["PORT"] = portValue
		serviceEnv["AZD_PORT"] = portValue
	}
	serviceEnv["SERVICE_NAME"] = rt.Name

	// For container services, skip port reservation - the container may already
	// be running on that port, and StartContainerService handles reuse logic.
	// For native services, reserve port to prevent TOCTOU race condition.
	if rt.Type != ServiceTypeContainer {
		portMgr := portmanager.GetPortManager(projectDir)
		reservation, portErr := portMgr.ReservePort(rt.Port)
		if portErr != nil {
			err := fmt.Errorf("port %d is no longer available (taken by another process): %w", rt.Port, portErr)
			if regErr := reg.UpdateStatus(rt.Name, constants.StatusError); regErr != nil {
				logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", regErr))
			}
			logger.LogService(rt.Name, fmt.Sprintf("❌ Port %d conflict detected", rt.Port))
			return nil, err
		}

		// Release port reservation immediately before starting service
		// The service must bind quickly after this to avoid a new race
		if err := reservation.Release(); err != nil {
			slog.Debug("failed to release port reservation", "port", rt.Port, "error", err)
		}
	}

	// Start service - use container runner for container services
	var process *ServiceProcess
	var err error
	if rt.Type == ServiceTypeContainer {
		process, err = StartContainerService(rt, projectDir, restartContainers)
		if err == nil {
			// Start container log collection
			if logErr := StartContainerLogCollection(process, projectDir); logErr != nil {
				slog.Warn("failed to start container log collection",
					slog.String("service", rt.Name),
					slog.String("error", logErr.Error()))
			}
		}
	} else {
		process, err = StartService(rt, serviceEnv, projectDir, functionsParser)
	}
	if err != nil {
		slog.Error("failed to start service",
			slog.String("service", rt.Name),
			slog.Int("port", rt.Port),
			slog.String("error", err.Error()))
		if regErr := reg.UpdateStatus(rt.Name, constants.StatusError); regErr != nil {
			logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", regErr))
		}
		logger.LogService(rt.Name, fmt.Sprintf("Failed to start: %v", err))
		return nil, err
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
		if regErr := reg.Register(entry); regErr != nil {
			logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update registry with PID: %v", regErr))
		}
	}

	// Update status to running
	// Health will be determined dynamically by health checks
	if regErr := reg.UpdateStatus(rt.Name, constants.StatusRunning); regErr != nil {
		logger.LogService(rt.Name, fmt.Sprintf("Warning: failed to update status: %v", regErr))
	}
	process.Ready = true

	return process, nil
}

// waitForServiceHealthy waits for a service to become healthy before proceeding.
// This is used to ensure dependencies are healthy before starting dependent services.
func waitForServiceHealthy(name string, process *ServiceProcess, svc *Service, timeout time.Duration) error {
	// If health check is disabled, return immediately
	if svc.IsHealthcheckDisabled() {
		slog.Debug("health check disabled for service, skipping",
			slog.String("service", name))
		return nil
	}

	// Use existing PerformHealthCheck which handles all health check types
	// with exponential backoff
	originalTimeout := process.Runtime.HealthCheck.Timeout
	if timeout > 0 {
		process.Runtime.HealthCheck.Timeout = timeout
	}

	err := PerformHealthCheck(process)

	// Restore original timeout
	process.Runtime.HealthCheck.Timeout = originalTimeout

	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	slog.Debug("service is healthy",
		slog.String("service", name))

	return nil
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

			// Skip nil processes
			if proc == nil {
				return
			}

			// Update status to stopping
			if err := reg.UpdateStatus(serviceName, constants.StatusStopping); err != nil {
				cliout.Error("Warning: failed to update status for %s: %v", serviceName, err)
			}

			// Stop service - use container runner for container services
			var stopErr error
			if proc.Runtime.Type == ServiceTypeContainer {
				stopErr = StopContainerService(proc, DefaultStopTimeout)
			} else {
				stopErr = StopServiceGraceful(proc, DefaultStopTimeout)
			}
			if stopErr != nil {
				// Log error but continue stopping other services
				cliout.Error("Error stopping service %s: %v", serviceName, stopErr)
			}

			// Unregister from registry
			if err := reg.Unregister(serviceName); err != nil {
				cliout.Error("Warning: failed to unregister service %s: %v", serviceName, err)
			}
		}(name, process)
	}

	wg.Wait()
}

// GetServiceURLs generates URLs for all running services.
// Only includes services that have an assigned port (port > 0).
func GetServiceURLs(processes map[string]*ServiceProcess) map[string]string {
	urls := make(map[string]string)

	for name, process := range processes {
		// Only include services with assigned ports (port > 0)
		if process.Ready && process.Port > 0 {
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
