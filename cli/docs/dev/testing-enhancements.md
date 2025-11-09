# Test Documentation for Enhanced Features

This document describes the comprehensive test suite for the new features and enhancements implemented in PR #40.

## Test Files

### 1. `graceful_shutdown_test.go`
Tests for graceful shutdown functionality with timeout control.

#### Test Coverage

**Unit Tests (fast, run with `-short`)**
- `TestStopServiceGraceful_NilProcess` - Validates error handling for nil process
- `TestStopServiceGraceful_NilServiceProcess` - Validates error handling for nil ServiceProcess

**Integration Tests (requires actual processes)**
- `TestStopServiceGraceful_Success` - Verifies graceful shutdown completes within timeout
- `TestStopServiceGraceful_ForcedKillAfterTimeout` - Verifies forced kill after timeout expires
- `TestStopServiceGraceful_AlreadyExited` - Handles already-exited processes correctly
- `TestStopServiceGraceful_MultipleTimeouts` - Tests different timeout durations (100ms, 500ms, 1s, 3s)
- `TestStopService_UsesGracefulDefault` - Confirms StopService uses 5s default timeout
- `TestStopAllServices_GracefulShutdown` - Verifies concurrent shutdown of multiple services

#### Key Behaviors Tested
- SIGINT sent first, waits for graceful shutdown
- Force kill (SIGKILL) if process doesn't exit within timeout
- Timeout values are respected
- Error handling for invalid inputs
- Concurrent shutdown of multiple services

### 2. `backoff_test.go`
Tests for exponential backoff implementation in health checks.

#### Test Coverage

**Unit Tests (fast, run with `-short`)**
- `TestWaitForPort_BackoffTimeout` - Validates backoff timeout behavior
- `TestPerformHealthCheck_BackoffTimeout` - Tests health check with backoff timeout
- `TestBackoffConfiguration_InitialInterval` - Verifies 100ms initial interval
- `TestBackoffConfiguration_MaxInterval` - Confirms 2s max interval cap
- `TestBackoffConfiguration_Multiplier` - Validates 2.0 exponential multiplier

**Integration Tests (requires network/servers)**
- `TestWaitForPort_ExponentialBackoff` - Tests backoff with delayed server start
- `TestWaitForPort_ImmediateSuccess` - Verifies quick completion when port ready
- `TestPerformHealthCheck_ExponentialBackoff` - Tests health check backoff behavior
- `TestPerformHealthCheck_HTTPBackoff` - Validates HTTP health check retries with backoff

#### Key Behaviors Tested
- Initial interval: 100ms (WaitForPort), config.Interval (PerformHealthCheck)
- Max interval: 2s (WaitForPort), 5s (PerformHealthCheck)
- Multiplier: 2.0 (exponential growth)
- Timeouts are respected
- Efficient retry patterns reduce CPU usage
- HTTP health checks retry with backoff

### 3. `run_test.go` (additions)
Tests for service monitoring and lifecycle management in the run command.

#### Test Coverage

**Unit Tests (fast, run with `-short`)**
- `TestRunCommandFlags` - Validates command-line flag parsing
- `TestRunCommandRuntimeValidation` - Tests runtime mode validation (azd/aspire)
- `TestRunCommandFlagDefaults` - Verifies default flag values

**Integration Tests (requires actual processes)**
- `TestMonitorServicesUntilShutdown_StartupTimeout` - Tests service startup and quick interrupt
- `TestMonitorServicesUntilShutdown_SignalHandling` - Verifies SIGINT/SIGTERM handling
- `TestMonitorServicesUntilShutdown_MultipleServices` - Tests monitoring multiple services
- `TestShutdownAllServices_WithContext` - Validates context-based shutdown coordination
- `TestShutdownAllServices_ContextTimeout` - Tests shutdown respects context deadline
- `TestProcessExit_CausesMonitoringToStop` - Verifies monitoring stops when process exits
- `TestMonitorServices_RunsIndefinitely` - Confirms services run without automatic timeout

#### Key Behaviors Tested
- Services run indefinitely until signaled (no artificial startup timeout)
- Context cancellation propagates correctly
- Signal handling (SIGINT/SIGTERM) triggers coordinated shutdown
- Multiple services monitored concurrently with errgroup
- Graceful shutdown with configurable timeout (default 10s)
- Process exit detection triggers shutdown of all services

## Running Tests

### Run All Short Tests (Fast)
```bash
go test ./src/internal/service ./src/cmd/app/commands -short
```

### Run Tests for Specific Features

**Graceful Shutdown:**
```bash
go test ./src/internal/service -run "TestStopServiceGraceful"
```

**Exponential Backoff:**
```bash
go test ./src/internal/service -run "Backoff"
```

**Startup Timeout & Monitoring:**
```bash
go test ./src/cmd/app/commands -run "Startup|Shutdown|Monitor"
```

### Run All Tests (Including Integration Tests)
```bash
go test ./src/internal/service ./src/cmd/app/commands -v -timeout 5m
```

## Test Results

All tests pass successfully:

- **Unit tests**: Complete in < 10s
- **Integration tests**: Complete in < 2m
- **Test coverage**: Graceful shutdown, exponential backoff, startup timeout, context cancellation, signal handling

## Structured Logging Validation

The integration tests also validate structured logging output:

```
INFO stopping service service=test-graceful pid=53032 port=8090 timeout=5s
WARN graceful shutdown timeout, forcing kill service=test-graceful timeout=500ms
INFO service stopped (forced after timeout) service=test-graceful
```

## Test Design Principles

1. **Fast unit tests** - Run with `-short` flag, no actual processes
2. **Comprehensive integration tests** - Use real processes, test actual behavior
3. **Timeout awareness** - All tests have reasonable timeouts to prevent hangs
4. **Platform compatibility** - Tests account for Windows/Unix differences
5. **Cleanup handling** - Proper teardown of processes and log buffers
6. **Observable behavior** - Tests verify timing, errors, and state changes

## Known Platform Differences

- **Windows**: `os.Interrupt` not supported, falls back to `Kill()`
- **Unix**: Full SIGINT support for graceful shutdown
- Tests handle both scenarios gracefully

## Future Enhancements

Potential test improvements for future PRs:

1. Mock clock for deterministic timing tests
2. Process crash simulation
3. Network failure simulation for health checks
4. Stress tests with many concurrent services
5. Memory leak detection in long-running monitors
