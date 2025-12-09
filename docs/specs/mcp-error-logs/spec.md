# MCP Error Logs Enhancement

## Problem Statement

The current `get_service_logs` MCP tool returns all logs, which is inefficient for the primary debugging use case where Copilot needs to quickly identify and diagnose errors. Returning full logs wastes tokens and makes it harder for AI to focus on the actual problem.

## Goals

1. Provide an optimized method for retrieving service errors with context
2. Minimize token usage while maximizing debugging value
3. Include contextual information around errors for better diagnosis
4. Single source of truth for context extraction logic (no duplication)

## Design Decision

**Approach: CLI `--context` flag + simplified MCP tool**

Add `--context N` flag to the `logs` command that works with `--level` filtering. The MCP tool then simply calls the CLI with appropriate flags.

### Why this approach:
- **No code duplication** - Context extraction logic lives only in CLI
- **Generalized** - `--context` works with any level (error, warn, debug)
- **CLI users benefit** - Not just MCP, humans can use `--context` too
- **Testable** - Single code path to test and maintain

### Current Problem (to be fixed)
The MCP `get_service_errors` tool currently duplicates context extraction logic:
1. Calls `azd app logs --format json`
2. Parses JSON output in MCP tool
3. Re-implements context extraction (same logic as `LogBuffer.GetErrors`)

This creates two code paths that could diverge.

## CLI Enhancement

### New `--context` flag for `logs` command

```bash
# Errors with 3 lines context
azd app logs --level error --context 3

# Warnings with 5 lines context  
azd app logs --level warn --context 5

# Debug logs with context
azd app logs --level debug --context 2
```

**Rules**:
- `--context` requires `--level` to be set (not `all`)
- Context range: 0-10 lines (clamped)
- Works with `--format json` for structured output

### Generalize GetErrors → GetLogsWithContext

Rename/generalize `LogBuffer.GetErrors()` to work with any log level:

```go
// Before (error-only)
func (lb *LogBuffer) GetErrors(limit, contextLines int, includeStderr bool, since time.Time) []ErrorEntry

// After (any level)
func (lb *LogBuffer) GetLogsWithContext(level LogLevel, limit, contextLines int, since time.Time) []LogEntryWithContext
```

## MCP Tool Specification

### `get_service_errors`

**Description**: Get error logs from services with surrounding context for debugging.

**Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| projectDir | string | No | cwd | Project directory path |
| serviceName | string | No | all | Filter to specific service |
| since | string | No | 10m | Time window (e.g., "5m", "1h") |
| tail | number | No | 500 | Log lines to retrieve |
| contextLines | number | No | 3 | Lines before/after errors (0-10) |

**Implementation** (simplified):
```go
// Just call CLI with flags - no parsing/extraction needed
cmd := exec.Command("azd", "app", "logs", 
    "--level", "error",
    "--context", fmt.Sprintf("%d", contextLines),
    "--format", "json",
    "--since", since,
    "--tail", fmt.Sprintf("%d", tail))
```

**Returns**: JSON with summary and errors array including context.

## Success Criteria

- [x] `--context N` flag added to `logs` command
- [x] Works with any `--level` (error, warn, debug, info)
- [x] `GetErrors` generalized to `GetLogsWithContext`
- [x] MCP tool simplified to just call CLI with flags
- [x] Duplicate context extraction removed from MCP tool
- [x] All existing tests pass + new tests for `--context`
