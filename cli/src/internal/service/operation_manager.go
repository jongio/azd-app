// Package service provides runtime detection and service orchestration capabilities.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// OperationState represents the current state of a service operation.
type OperationState string

const (
	// OperationIdle indicates no operation is in progress.
	OperationIdle OperationState = "idle"
	// OperationStarting indicates a start operation is in progress.
	OperationStarting OperationState = "starting"
	// OperationStopping indicates a stop operation is in progress.
	OperationStopping OperationState = "stopping"
	// OperationRestarting indicates a restart operation is in progress.
	OperationRestarting OperationState = "restarting"
)

// OperationType represents the type of service operation.
type OperationType string

const (
	OpStart   OperationType = "start"
	OpStop    OperationType = "stop"
	OpRestart OperationType = "restart"
)

// ServiceOperationManager coordinates service operations with concurrency control.
// It prevents concurrent operations on the same service and provides operation tracking.
type ServiceOperationManager struct {
	mu             sync.RWMutex
	serviceStates  map[string]OperationState
	serviceMutexes map[string]*sync.Mutex
	timeout        time.Duration
}

var (
	operationManagerInstance *ServiceOperationManager
	operationManagerOnce     sync.Once
)

// DefaultOperationTimeout is the default timeout for service operations.
const DefaultOperationTimeout = 30 * time.Second

// GetOperationManager returns the singleton ServiceOperationManager instance.
func GetOperationManager() *ServiceOperationManager {
	operationManagerOnce.Do(func() {
		operationManagerInstance = &ServiceOperationManager{
			serviceStates:  make(map[string]OperationState),
			serviceMutexes: make(map[string]*sync.Mutex),
			timeout:        DefaultOperationTimeout,
		}
	})
	return operationManagerInstance
}

// getServiceMutex returns the mutex for a specific service, creating it if necessary.
func (m *ServiceOperationManager) getServiceMutex(serviceName string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()

	if mtx, exists := m.serviceMutexes[serviceName]; exists {
		return mtx
	}

	mtx := &sync.Mutex{}
	m.serviceMutexes[serviceName] = mtx
	m.serviceStates[serviceName] = OperationIdle
	return mtx
}

// GetOperationState returns the current operation state for a service.
func (m *ServiceOperationManager) GetOperationState(serviceName string) OperationState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if state, exists := m.serviceStates[serviceName]; exists {
		return state
	}
	return OperationIdle
}

// setOperationState updates the operation state for a service.
func (m *ServiceOperationManager) setOperationState(serviceName string, state OperationState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.serviceStates[serviceName] = state
}

// IsOperationInProgress checks if any operation is currently running for a service.
func (m *ServiceOperationManager) IsOperationInProgress(serviceName string) bool {
	return m.GetOperationState(serviceName) != OperationIdle
}

// operationTypeToState converts an OperationType to its corresponding OperationState.
func operationTypeToState(op OperationType) OperationState {
	switch op {
	case OpStart:
		return OperationStarting
	case OpStop:
		return OperationStopping
	case OpRestart:
		return OperationRestarting
	default:
		return OperationIdle
	}
}

// OperationResult contains the result of a service operation.
type OperationResult struct {
	ServiceName string
	Operation   OperationType
	Success     bool
	Error       error
	Duration    time.Duration
}

// ExecuteOperation executes a service operation with proper locking and state management.
// The operationFunc is the actual operation to perform (start, stop, restart).
// Returns an error if the operation cannot be started (e.g., another operation is in progress).
func (m *ServiceOperationManager) ExecuteOperation(
	ctx context.Context,
	serviceName string,
	operation OperationType,
	operationFunc func(ctx context.Context) error,
) *OperationResult {
	startTime := time.Now()
	result := &OperationResult{
		ServiceName: serviceName,
		Operation:   operation,
		Success:     false,
	}

	// Get service-specific mutex
	mtx := m.getServiceMutex(serviceName)

	// Try to acquire the lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use a channel to implement lock with timeout
	// The done channel signals the goroutine to release the lock if timeout fired
	lockAcquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		mtx.Lock()
		select {
		case <-done:
			// Timeout already fired, release the lock immediately
			mtx.Unlock()
			return
		case lockAcquired <- struct{}{}:
			// Successfully signaled that lock was acquired
			// Caller will handle unlock via defer
		}
	}()

	select {
	case <-lockAcquired:
		// Lock acquired, proceed
		defer mtx.Unlock()
	case <-lockCtx.Done():
		close(done) // Signal goroutine to release lock if it acquires it later
		result.Error = fmt.Errorf("timeout waiting for service lock (another operation may be in progress)")
		result.Duration = time.Since(startTime)
		slog.Warn("operation lock timeout",
			slog.String("service", serviceName),
			slog.String("operation", string(operation)))
		return result
	}

	// Check if operation is already in progress
	currentState := m.GetOperationState(serviceName)
	if currentState != OperationIdle {
		result.Error = fmt.Errorf("operation '%s' already in progress for service '%s'", currentState, serviceName)
		result.Duration = time.Since(startTime)
		return result
	}

	// Set operation state based on operation type
	m.setOperationState(serviceName, operationTypeToState(operation))

	// Ensure state is reset when done
	defer m.setOperationState(serviceName, OperationIdle)

	slog.Info("starting service operation",
		slog.String("service", serviceName),
		slog.String("operation", string(operation)))

	// Create timeout context for the operation
	opCtx, opCancel := context.WithTimeout(ctx, m.timeout)
	defer opCancel()

	// Execute the operation
	if err := operationFunc(opCtx); err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		slog.Error("service operation failed",
			slog.String("service", serviceName),
			slog.String("operation", string(operation)),
			slog.String("error", err.Error()),
			slog.Duration("duration", result.Duration))
		return result
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	slog.Info("service operation completed",
		slog.String("service", serviceName),
		slog.String("operation", string(operation)),
		slog.Duration("duration", result.Duration))

	return result
}

// BulkOperationResult contains the results of a bulk operation across multiple services.
type BulkOperationResult struct {
	Operation     OperationType
	Results       []*OperationResult
	SuccessCount  int
	FailureCount  int
	TotalDuration time.Duration
}

// ExecuteBulkOperation executes an operation on multiple services concurrently.
// The operationFunc is called for each service with the service name and context.
func (m *ServiceOperationManager) ExecuteBulkOperation(
	ctx context.Context,
	serviceNames []string,
	operation OperationType,
	operationFuncFactory func(serviceName string) func(ctx context.Context) error,
) *BulkOperationResult {
	startTime := time.Now()
	result := &BulkOperationResult{
		Operation: operation,
		Results:   make([]*OperationResult, 0, len(serviceNames)),
	}

	if len(serviceNames) == 0 {
		result.TotalDuration = time.Since(startTime)
		return result
	}

	slog.Info("starting bulk service operation",
		slog.String("operation", string(operation)),
		slog.Int("count", len(serviceNames)))

	var wg sync.WaitGroup
	resultsChan := make(chan *OperationResult, len(serviceNames))

	for _, serviceName := range serviceNames {
		wg.Add(1)
		go func(svcName string) {
			defer wg.Done()
			opFunc := operationFuncFactory(svcName)
			opResult := m.ExecuteOperation(ctx, svcName, operation, opFunc)
			resultsChan <- opResult
		}(serviceName)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for opResult := range resultsChan {
		result.Results = append(result.Results, opResult)
		if opResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}
	}

	result.TotalDuration = time.Since(startTime)
	slog.Info("bulk service operation completed",
		slog.String("operation", string(operation)),
		slog.Int("success", result.SuccessCount),
		slog.Int("failure", result.FailureCount),
		slog.Duration("duration", result.TotalDuration))

	return result
}

// ClearServiceState removes the state tracking for a service.
// This should be called when a service is unregistered.
func (m *ServiceOperationManager) ClearServiceState(serviceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.serviceStates, serviceName)
	delete(m.serviceMutexes, serviceName)
}

// SetTimeout sets the operation timeout.
func (m *ServiceOperationManager) SetTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = timeout
}

// GetTimeout returns the current operation timeout.
func (m *ServiceOperationManager) GetTimeout() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timeout
}
