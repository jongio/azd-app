package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetOperationManager(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil

	mgr1 := GetOperationManager()
	if mgr1 == nil {
		t.Fatal("GetOperationManager() returned nil")
	}

	mgr2 := GetOperationManager()
	if mgr1 != mgr2 {
		t.Error("GetOperationManager() should return the same instance (singleton)")
	}
}

func TestOperationState(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-service"

	// Initial state should be idle
	state := mgr.GetOperationState(serviceName)
	if state != OperationIdle {
		t.Errorf("GetOperationState() = %v, want %v", state, OperationIdle)
	}

	// Should not be in progress
	if mgr.IsOperationInProgress(serviceName) {
		t.Error("IsOperationInProgress() = true, want false for new service")
	}
}

func TestExecuteOperation_Success(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-service-success"
	ctx := context.Background()

	executed := false
	result := mgr.ExecuteOperation(ctx, serviceName, OpStart, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if !executed {
		t.Error("Operation function was not executed")
	}
	if !result.Success {
		t.Errorf("ExecuteOperation() Success = false, want true")
	}
	if result.Error != nil {
		t.Errorf("ExecuteOperation() Error = %v, want nil", result.Error)
	}
	if result.ServiceName != serviceName {
		t.Errorf("ExecuteOperation() ServiceName = %v, want %v", result.ServiceName, serviceName)
	}
	if result.Operation != OpStart {
		t.Errorf("ExecuteOperation() Operation = %v, want %v", result.Operation, OpStart)
	}
	if result.Duration < 0 {
		t.Error("ExecuteOperation() Duration should be non-negative")
	}
}

func TestExecuteOperation_Failure(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-service-fail"
	ctx := context.Background()
	expectedErr := errors.New("operation failed")

	result := mgr.ExecuteOperation(ctx, serviceName, OpStop, func(ctx context.Context) error {
		return expectedErr
	})

	if result.Success {
		t.Error("ExecuteOperation() Success = true, want false")
	}
	if result.Error == nil {
		t.Error("ExecuteOperation() Error = nil, want error")
	}
	if result.Error.Error() != expectedErr.Error() {
		t.Errorf("ExecuteOperation() Error = %v, want %v", result.Error, expectedErr)
	}
}

func TestExecuteOperation_ConcurrentPrevention(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-service-concurrent"
	ctx := context.Background()

	// Start a long-running operation
	started := make(chan struct{})
	finish := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mgr.ExecuteOperation(ctx, serviceName, OpStart, func(ctx context.Context) error {
			close(started)
			<-finish
			return nil
		})
	}()

	// Wait for first operation to start
	<-started

	// Try to start another operation on the same service
	result := mgr.ExecuteOperation(ctx, serviceName, OpStop, func(ctx context.Context) error {
		return nil
	})

	// The second operation should fail because another is in progress
	if result.Success {
		t.Error("Second operation should fail when another is in progress")
	}
	if result.Error == nil {
		t.Error("Second operation should have an error")
	}

	// Allow first operation to finish
	close(finish)
	wg.Wait()

	// Now a new operation should succeed
	result = mgr.ExecuteOperation(ctx, serviceName, OpStop, func(ctx context.Context) error {
		return nil
	})
	if !result.Success {
		t.Errorf("Operation after completion should succeed, got error: %v", result.Error)
	}
}

func TestExecuteOperation_DifferentServices(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	ctx := context.Background()

	// Operations on different services should run concurrently
	started1 := make(chan struct{})
	started2 := make(chan struct{})
	finish := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		mgr.ExecuteOperation(ctx, "service-1", OpStart, func(ctx context.Context) error {
			close(started1)
			<-finish
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		mgr.ExecuteOperation(ctx, "service-2", OpStart, func(ctx context.Context) error {
			close(started2)
			<-finish
			return nil
		})
	}()

	// Both should start (no blocking on each other)
	select {
	case <-started1:
	case <-time.After(time.Second):
		t.Fatal("Service 1 operation did not start")
	}

	select {
	case <-started2:
	case <-time.After(time.Second):
		t.Fatal("Service 2 operation did not start")
	}

	close(finish)
	wg.Wait()
}

func TestExecuteBulkOperation(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	ctx := context.Background()
	services := []string{"svc-1", "svc-2", "svc-3"}

	var executedCount int32

	result := mgr.ExecuteBulkOperation(ctx, services, OpStart, func(serviceName string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			atomic.AddInt32(&executedCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			return nil
		}
	})

	if int(executedCount) != len(services) {
		t.Errorf("Expected %d operations, got %d", len(services), executedCount)
	}
	if result.SuccessCount != len(services) {
		t.Errorf("SuccessCount = %d, want %d", result.SuccessCount, len(services))
	}
	if result.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", result.FailureCount)
	}
	if len(result.Results) != len(services) {
		t.Errorf("Results count = %d, want %d", len(result.Results), len(services))
	}
}

func TestExecuteBulkOperation_PartialFailure(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	ctx := context.Background()
	services := []string{"ok-1", "fail-1", "ok-2"}

	result := mgr.ExecuteBulkOperation(ctx, services, OpStop, func(serviceName string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			if serviceName == "fail-1" {
				return errors.New("intentional failure")
			}
			return nil
		}
	})

	if result.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", result.SuccessCount)
	}
	if result.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", result.FailureCount)
	}
}

func TestExecuteBulkOperation_Empty(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	ctx := context.Background()

	result := mgr.ExecuteBulkOperation(ctx, []string{}, OpRestart, func(serviceName string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			return nil
		}
	})

	if result.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", result.SuccessCount)
	}
	if result.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", result.FailureCount)
	}
	if len(result.Results) != 0 {
		t.Errorf("Results count = %d, want 0", len(result.Results))
	}
}

func TestClearServiceState(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-clear"
	ctx := context.Background()

	// Execute an operation to create state
	mgr.ExecuteOperation(ctx, serviceName, OpStart, func(ctx context.Context) error {
		return nil
	})

	// Clear state
	mgr.ClearServiceState(serviceName)

	// State should be idle (default for unknown service)
	state := mgr.GetOperationState(serviceName)
	if state != OperationIdle {
		t.Errorf("GetOperationState() = %v, want %v after clear", state, OperationIdle)
	}
}

func TestSetTimeout(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	newTimeout := 60 * time.Second
	mgr.SetTimeout(newTimeout)

	if mgr.GetTimeout() != newTimeout {
		t.Errorf("GetTimeout() = %v, want %v", mgr.GetTimeout(), newTimeout)
	}
}

func TestOperationStateTransitions(t *testing.T) {
	// Reset singleton for testing
	operationManagerOnce = sync.Once{}
	operationManagerInstance = nil
	mgr := GetOperationManager()

	serviceName := "test-transitions"

	testCases := []struct {
		operation     OperationType
		expectedState OperationState
	}{
		{OpStart, OperationStarting},
		{OpStop, OperationStopping},
		{OpRestart, OperationRestarting},
	}

	for _, tc := range testCases {
		t.Run(string(tc.operation), func(t *testing.T) {
			// Reset for each test case
			operationManagerOnce = sync.Once{}
			operationManagerInstance = nil
			mgr = GetOperationManager()

			stateObserved := make(chan OperationState, 1)
			ctx := context.Background()

			mgr.ExecuteOperation(ctx, serviceName, tc.operation, func(ctx context.Context) error {
				// Capture state during operation
				stateObserved <- mgr.GetOperationState(serviceName)
				return nil
			})

			select {
			case state := <-stateObserved:
				if state != tc.expectedState {
					t.Errorf("During %s, state = %v, want %v", tc.operation, state, tc.expectedState)
				}
			case <-time.After(time.Second):
				t.Fatal("Timeout waiting for state observation")
			}

			// After completion, state should be idle
			finalState := mgr.GetOperationState(serviceName)
			if finalState != OperationIdle {
				t.Errorf("After %s completion, state = %v, want %v", tc.operation, finalState, OperationIdle)
			}
		})
	}
}
