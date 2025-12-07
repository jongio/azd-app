# Structured Logging Enhancement

## Overview

Enhance the existing logging infrastructure to provide consistent, structured logging across the entire codebase. This enables machine-parseable output for CI/CD pipelines, better debugging of multi-service scenarios, and integration with observability tools.

## Problem Analysis

### Current State
1. **Inconsistent logging approaches:**
   - `internal/service/` uses `slog.Debug/Info/Warn/Error` correctly
   - `internal/testing/` uses `fmt.Printf` for all output
   - `internal/output/` has its own styling for console output
   - Some files mix approaches within the same package

2. **Missing capabilities:**
   - No component/category tagging for filtering
   - No correlation IDs for tracing operations across services
   - No structured context for test operations
   - Hard to parse logs in CI/CD environments

3. **User experience gaps:**
   - Watch mode output uses emojis but isn't machine-parseable
   - Test results can't be filtered by service/test type
   - Debugging multi-service issues is difficult

### Files Using fmt.Printf Instead of Structured Logging
- `internal/testing/orchestrator.go` - 2 instances
- `internal/testing/watcher.go` - 13 instances  
- `internal/testing/reporter.go` - 4 instances
- `internal/onboarding/notifications.go` - 20+ instances (user prompts, OK)
- `internal/output/output.go` - Many instances (styled output, OK)

## Holistic Benefits of Structured Logging

### 1. The `logs` Command Already Supports It

The `azd app logs` command already has `--format json` for structured output:
```bash
azd app logs --format json --follow
```

This outputs each log entry as a JSON object with service, timestamp, level, and message. Extending structured logging throughout the codebase means **all operations** can produce machine-parseable output, not just service logs.

### 2. CI/CD Pipeline Integration

With `--structured-logs`, pipelines can:
- Parse test results programmatically
- Filter by component (test, health, service)
- Extract metrics (coverage %, test counts, durations)
- Integrate with log aggregation services (Datadog, Splunk, ELK)

```yaml
# GitHub Actions example
- name: Run tests with structured output
  run: azd app test --structured-logs 2> test-logs.json
  
- name: Parse test results
  run: |
    jq 'select(.component=="test" and .event=="test_completed")' test-logs.json
```

### 3. Command-Specific Benefits

| Command | Structured Logging Benefit |
|---------|---------------------------|
| `run` | Service start/stop events with PIDs, ports, timing |
| `test` | Test execution events, coverage data, failures with context |
| `health` | Health check results as JSON events per service |
| `logs` | Already supports JSON - becomes consistent with other commands |
| `deps` | Dependency installation progress and outcomes |
| `reqs` | Requirement validation results per check |

### 4. Debugging Multi-Service Scenarios

When running 5+ services with `azd app run`, structured logging enables:
- Filtering logs by service: `jq 'select(.service=="api")'`
- Finding errors across services: `jq 'select(.level=="ERROR")'`
- Correlating events by timestamp
- Identifying which service caused a cascade failure

### 5. Observability Integration

Structured JSON logs integrate directly with:
- **Prometheus**: Extract metrics from log events
- **Grafana Loki**: Query logs by component/service labels
- **Azure Monitor**: Ingest via Log Analytics
- **Local development**: Pipe to `jq` for filtering

## Requirements

### 1. Log Categories (Components)
Define standard component names for filtering:
- `service` - Service orchestration and lifecycle
- `test` - Test execution and coverage
- `config` - Configuration loading
- `health` - Health checks
- `port` - Port management
- `detect` - Language/framework detection
- `watch` - File watching
- `coverage` - Coverage aggregation
- `logs` - Log streaming operations

### 2. Structured Context Fields
Standard fields for all log entries:
- `component` - Log category (required)
- `service` - Service name when applicable
- `operation` - Current operation (start, stop, test, etc.)
- `duration` - Duration for completed operations
- `error` - Error details when applicable
- `event` - Machine-parseable event type

### 3. Log Levels
- `Debug` - Detailed diagnostic info (requires --debug flag)
- `Info` - Normal operation progress
- `Warn` - Non-fatal issues that need attention
- `Error` - Failures that stop an operation

### 4. Output Modes
- **Default (console):** Human-friendly colored output with timestamps
- **Structured (--structured-logs):** JSON to stderr for CI/CD parsing
- **Debug (--debug):** Verbose text output to stderr

### 5. Event Types for Machine Parsing

Define standard event types that tools can filter on:
```
test_started, test_completed, test_failed
coverage_collected, coverage_aggregated
service_started, service_stopped, service_error
health_check, health_changed
file_changed, watch_triggered
```

## Design

### Logger Interface Enhancement

```go
// Logger wraps slog with component context
type Logger struct {
    component string
    slogger   *slog.Logger
}

// NewLogger creates a logger for a component
func NewLogger(component string) *Logger

// WithService returns a logger with service context
func (l *Logger) WithService(name string) *Logger

// WithOperation returns a logger with operation context
func (l *Logger) WithOperation(op string) *Logger

// Test-specific logging
func (l *Logger) TestStarted(service, testFile string)
func (l *Logger) TestCompleted(service string, passed, failed, skipped int, duration float64)
func (l *Logger) CoverageCollected(service string, percent float64)
```

### Usage Example

```go
// In orchestrator.go
log := logging.NewLogger("test")

log.Info("starting test run",
    "event", "test_run_started",
    "services", len(services),
    "testType", testType,
)

// Per-service logging
svcLog := log.WithService(service.Name)
svcLog.TestStarted(service.Name, testFile)

// On completion
svcLog.TestCompleted(service.Name, result.Passed, result.Failed, result.Skipped, result.Duration)
```

### JSON Output (Structured Mode)

```json
{"time":"2025-01-15T10:30:00Z","level":"INFO","msg":"starting test run","component":"test","event":"test_run_started","services":4,"testType":"unit"}
{"time":"2025-01-15T10:30:01Z","level":"INFO","msg":"test started","component":"test","event":"test_started","service":"api","file":"api.test.ts"}
{"time":"2025-01-15T10:30:05Z","level":"INFO","msg":"test completed","component":"test","event":"test_completed","service":"api","passed":15,"failed":0,"skipped":0,"duration_sec":4.2}
{"time":"2025-01-15T10:30:06Z","level":"INFO","msg":"coverage collected","component":"test","event":"coverage_collected","service":"api","coverage_pct":87.5}
```

## Implementation Tasks

### Task 1: Enhance Logger Package [Developer] - DONE
- Add `NewLogger(component)` function
- Add `WithService`, `WithOperation`, `WithFields` context methods
- Add test-specific convenience methods (TestStarted, TestCompleted, CoverageCollected)
- Add `IsStructured()` function to check logging mode
- Ensure backwards compatibility with existing `logging.Debug/Info/Warn/Error`

### Task 2: Migrate Testing Package [Developer]
- Replace `fmt.Printf` with structured logging in `orchestrator.go`
- Replace `fmt.Printf` with structured logging in `watcher.go`  
- Replace `fmt.Printf` with structured logging in `reporter.go`
- Keep emoji/styled output for console mode only (check `!logging.IsStructured()`)

### Task 3: Add Test Logging Tests [Tester]
- Unit tests for logger component context
- Unit tests for JSON output format
- Integration test for test command logging

### Task 4: Documentation [Developer]
- Update cli-reference.md with --debug and --structured-logs flags
- Add examples of filtering logs by component

## Success Criteria

1. All `fmt.Printf` in testing package replaced with structured logging
2. `--debug` flag shows detailed operation logs
3. `--structured-logs` flag outputs parseable JSON to stderr
4. Logs filterable by component in CI systems
5. All existing tests pass
6. Preflight checks pass

## Migration Strategy

1. Keep existing console output appearance unchanged
2. Add structured logging calls alongside (or replacing) fmt calls
3. Use `logging.IsDebugEnabled()` for verbose-only output
4. Use `logging.IsStructured()` to skip emoji output in structured mode
5. Test output goes to stdout, logs go to stderr

## Future Enhancements (Out of Scope)

1. **Correlation IDs**: Add request/operation IDs for tracing across services
2. **Log Rotation**: Automatic log file rotation and cleanup
3. **Remote Logging**: Direct integration with Azure Monitor or other services
4. **Log Search**: Built-in command to search historical logs

