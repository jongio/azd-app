# MCP Error Logs Tasks

## Completed Tasks

### 1. Add GetErrors method to LogBuffer
- [x] **DONE** - Developer
- Add `GetErrors(limit int, contextLines int, includeStderr bool, since time.Time) []ErrorEntry` to LogBuffer
- Create `ErrorEntry` struct with context fields (before/after lines)
- Implement context window extraction with deduplication
- Handle edge cases (errors at start/end of buffer)

### 2. Add GetAllErrors method to LogManager
- [x] **DONE** - Developer
- Add `GetAllErrors(serviceName string, limit int, contextLines int, includeStderr bool, since time.Time) []ErrorEntry`
- Support filtering by service name (empty = all services)
- Support time-based filtering
- Aggregate and sort errors across services

### 3. Create get_service_errors MCP tool
- [x] **DONE** - Developer
- Add `newGetServiceErrorsTool()` in mcp_tools.go
- Calls existing `azd app logs` command with time filtering
- Parses JSON output and identifies errors
- Extracts context lines around each error
- Returns structured JSON response

### 4. Add unit tests
- [x] **DONE** - Developer
- Test error filtering by level and stderr
- Test context extraction (before/after lines)
- Test edge cases (error at start/end of buffer)

### 5. Update documentation
- [x] **DONE** - Developer
- Add get_service_errors to MCP command docs
- Document parameters and response format
- Update web docs (tools.astro, index.astro, ai-debugging.astro, prompts.astro)
- Fix tool count: 12 MCP tools total

---

## Refactor Tasks (Design Improvement)

### 6. Add --context flag to logs command
- [x] **DONE** - Developer
- Add `--context N` flag to `azd app logs` command
- Requires `--level` to be set (error, warn, debug, info - not "all")
- Context range: 0-10 lines (clamped)
- Works with `--format json` for structured output
- Update command help/examples

### 7. Generalize GetErrors to GetLogsWithContext
- [x] **DONE** - Developer
- Rename `LogBuffer.GetErrors` → `GetLogsWithContext(level LogLevel, ...)`
- Support any log level, not just errors
- Update `LogManager.GetAllErrors` → `GetAllLogsWithContext`
- Keep backward compat or update all callers

### 8. Simplify MCP get_service_errors tool
- [x] **DONE** - Developer
- Remove duplicate context extraction logic from mcp_tools.go
- Call CLI with `--level error --context N --format json`
- Parse and return JSON directly (no re-processing)
- Lines ~311-381 in mcp_tools.go to be simplified

### 9. Add tests for --context flag
- [x] **DONE** - Developer
- Test `--context` with `--level error`
- Test `--context` with `--level warn`
- Test `--context` without `--level` (should error)
- Test context clamping (0-10)
- Test JSON output format with context

### 10. Update CLI documentation
- [x] **DONE** - Developer
- Add `--context` to logs command docs (cli-reference.md)
- Add examples: `azd app logs --level error --context 3`
- Update docs/commands/logs.md
