# Refactoring Tasks

## Progress: 3/10 tasks complete

---

## Task 1: Remove deprecated RunLogicApp function

**Agent**: Developer
**Status**: DONE

**Description**:
Remove the deprecated `RunLogicApp` function from `cli/src/internal/runner/runner.go` and update any tests that reference it.

**Files**:
- `cli/src/internal/runner/runner.go` (remove RunLogicApp)
- `cli/src/internal/runner/runner_test.go` (update test)

**Acceptance Criteria**:
- Function removed
- Test updated to use RunFunctionApp directly
- All tests pass

---

## Task 2: Remove deprecated StopService wrapper

**Agent**: Developer
**Status**: DONE

**Description**:
Update all callers of `StopService` to use `StopServiceGraceful` directly, then remove the deprecated wrapper.

**Files**:
- `cli/src/internal/service/executor.go`
- `cli/src/internal/service/executor_test.go`
- `cli/src/internal/service/graceful_shutdown_test.go`

**Acceptance Criteria**:
- All callers use StopServiceGraceful with explicit timeout
- Deprecated wrapper removed
- All tests pass

---

## Task 3: Split healthcheck/monitor.go (1518 lines)

**Agent**: Developer
**Status**: DONE

**Description**:
Split the oversized monitor.go file into focused modules:
- `monitor.go` - HealthMonitor struct, NewHealthMonitor, CheckHealth methods
- `checker.go` - HealthChecker struct, HTTP/TCP/Process check implementations
- `types.go` - HealthStatus, HealthCheckResult, HealthReport, etc.

**Files**:
- `cli/src/internal/healthcheck/monitor.go` (split)

**Acceptance Criteria**:
- Each new file under 200 lines
- All exports remain accessible
- All existing tests pass
- No functionality changes

---

## Task 4: Split portmanager/portmanager.go (1174 lines)

**Agent**: Developer
**Status**: TODO

**Description**:
Split into focused modules:
- `portmanager.go` - PortManager struct, GetPortManager, core methods
- `allocation.go` - Port allocation, scanning, reservation logic
- `process.go` - Process monitoring, killing, cleanup

**Files**:
- `cli/src/internal/portmanager/portmanager.go` (split)

**Acceptance Criteria**:
- Each new file under 200 lines
- All exports remain accessible
- All existing tests pass

---

## Task 5: Split commands/mcp.go (1178 lines)

**Agent**: Developer
**Status**: TODO

**Description**:
Split into focused modules:
- `mcp.go` - NewMCPCommand, runMCPServer, server setup
- `mcp_tools.go` - All tool implementations (get_services, run_services, etc.)
- `mcp_resources.go` - Resource implementations (azure.yaml, service config)

**Files**:
- `cli/src/cmd/app/commands/mcp.go` (split)

**Acceptance Criteria**:
- Each new file under 200 lines
- All exports remain accessible
- All existing tests pass

---

## Task 6: Split commands/logs.go (1001 lines)

**Agent**: Developer
**Status**: TODO

**Description**:
Split into focused modules:
- `logs.go` - NewLogsCommand, logsExecutor, command setup
- `logs_streaming.go` - Log streaming logic, channel handling
- `logs_formatting.go` - Output formatting, filtering, file writing

**Files**:
- `cli/src/cmd/app/commands/logs.go` (split)

**Acceptance Criteria**:
- Each new file under 200 lines
- All exports remain accessible
- All existing tests pass

---

## Task 7: Split commands/core.go (1095 lines)

**Agent**: Developer
**Status**: TODO

**Description**:
Split into focused modules:
- `core.go` - Shared orchestrator, execution context, common types
- `reqs_core.go` - executeReqs and requirement checking logic
- `deps_core.go` - executeDeps and dependency installation logic

**Files**:
- `cli/src/cmd/app/commands/core.go` (split)

**Acceptance Criteria**:
- Each new file under 200 lines
- All exports remain accessible
- All existing tests pass

---

## Task 8: Extract shared copy-button script (Web)

**Agent**: Developer
**Status**: TODO

**Description**:
Extract the duplicated copy-button script from CLI reference pages into a shared component.

**Duplicated in**:
- restart.astro, run.astro, mcp.astro, stop.astro, start.astro, version.astro, health.astro, logs.astro

**Acceptance Criteria**:
- Create `web/src/components/CopyButton.astro` with shared script
- Update all CLI reference pages to use the component
- Copy functionality works identically

---

## Task 9: Address TODO in notifications.go

**Agent**: Developer
**Status**: TODO

**Description**:
Address the two TODO items in `cli/src/internal/config/notifications.go`:
1. Line 242: Add validation for serviceName format
2. Line 324: Consider caching parsed time values

**Files**:
- `cli/src/internal/config/notifications.go`

**Acceptance Criteria**:
- serviceName validation implemented (non-empty, valid characters)
- Time parsing caching implemented if beneficial
- All tests pass

---

## Task 10: Run full test suite validation

**Agent**: Tester
**Status**: TODO

**Description**:
After all refactoring tasks complete, run the full test suite to verify no regressions.

**Commands**:
```bash
cd cli && mage test
cd cli/dashboard && pnpm test
cd web && pnpm test
```

**Acceptance Criteria**:
- All Go tests pass
- All dashboard tests pass
- All web tests pass
- Coverage does not decrease
