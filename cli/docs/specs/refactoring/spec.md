# Refactoring Specification

## Overview

This document identifies refactoring opportunities across the azd-app codebase based on code analysis. The goal is to improve maintainability, reduce duplication, and remove deprecated/dead code while keeping all functionality intact.

## Priority Areas

### 1. Large Files (>200 Lines)

Files exceeding the 200-line guideline that should be split:

| File | Lines | Proposed Action |
|------|-------|-----------------|
| `cli/src/internal/healthcheck/monitor.go` | 1518 | Split into monitor.go, checker.go, config.go |
| `cli/src/internal/portmanager/portmanager.go` | 1174 | Split into manager.go, allocation.go, cache.go |
| `cli/src/cmd/app/commands/mcp.go` | 1178 | Split into mcp.go, mcp_tools.go, mcp_resources.go |
| `cli/src/cmd/app/commands/core.go` | 1095 | Split into core.go, deps.go, reqs.go |
| `cli/src/cmd/app/commands/logs.go` | 1001 | Split into logs.go, logs_streaming.go, logs_formatting.go |
| `cli/src/internal/service/detector.go` | 816 | Split into detector.go, detector_runtime.go |
| `cli/src/internal/detector/detector.go` | 796 | Split into detector.go, detector_node.go, detector_python.go, detector_dotnet.go |
| `cli/src/internal/dashboard/server.go` | 733 | Split into server.go, handlers.go |
| `cli/src/internal/installer/installer.go` | 708 | Split into installer.go, installer_node.go, installer_python.go |
| `web/src/pages/reference/changelog/index.astro` | 1154 | Auto-generated - consider paginated component |

### 2. Deprecated Code to Remove

| Location | Symbol | Reason |
|----------|--------|--------|
| `internal/runner/runner.go:314` | `RunLogicApp` | Deprecated - use RunFunctionApp |
| `internal/service/executor.go:99` | `StopService` | Deprecated - use StopServiceGraceful |
| `internal/output/progress.go:529` | `Stop()` | Deprecated - use Complete/Fail |
| `internal/detector/detector.go:592` | `FindFunctionApps` (old) | Deprecated - use updated version |

### 3. Code Duplication

**Copy Button Script (Web)**
Identical script duplicated across all CLI reference .astro pages:
- `web/src/pages/reference/cli/restart.astro`
- `web/src/pages/reference/cli/run.astro`
- `web/src/pages/reference/cli/mcp.astro`
- `web/src/pages/reference/cli/stop.astro`
- `web/src/pages/reference/cli/start.astro`
- `web/src/pages/reference/cli/version.astro`
- `web/src/pages/reference/cli/health.astro`
- `web/src/pages/reference/cli/logs.astro`

**Action**: Extract to shared component or script include.

### 4. TODO Items to Address

| File | Line | Description |
|------|------|-------------|
| `internal/config/notifications.go` | 242 | Add validation for serviceName format |
| `internal/config/notifications.go` | 324 | Consider caching parsed time values |

## Out of Scope

- Test files (even if >200 lines) - comprehensiveness is acceptable
- Generated files that get rebuilt
- Third-party dependencies

## Success Criteria

1. All source files under 200 lines (excluding tests)
2. No deprecated functions remain (or have clear migration path)
3. Duplicated code extracted into shared modules
4. All existing tests continue to pass
5. No functionality changes - pure refactoring
