# Refactoring Specification

## Overview

This document identifies refactoring opportunities across the azd-app codebase based on code analysis. The goal is to improve maintainability, reduce duplication, and remove deprecated/dead code while keeping all functionality intact.

## Phase 1 Status: COMPLETE ✅

The first phase of refactoring has been completed. See tasks.md for details.

## Phase 2 Status: COMPLETE ✅

Tasks 11-16 completed. Tasks 13-14 deferred to Phase 3.

## Phase 3: Current Refactoring Targets

### 1. Large Files (>200 Lines) - Updated

Files exceeding the 200-line guideline that should be split:

| File | Lines | Priority | Proposed Action |
|------|-------|----------|-----------------|
| `cli/src/cmd/app/commands/core.go` | 954 | HIGH | Split into core.go, core_deps.go, core_helpers.go |
| `cli/src/cmd/app/commands/logs.go` | 882 | HIGH | Split into logs.go, logs_streaming.go |
| `cli/src/internal/portmanager/portmanager.go` | 826 | HIGH | Extract prompt logic to portmanager_prompts.go |
| `cli/src/internal/dashboard/server.go` | 758 | MEDIUM | Split into server.go, handlers.go |
| `cli/src/internal/detector/detector.go` | 744 | MEDIUM | Split by language (deferred from Phase 2) |
| `cli/src/cmd/app/commands/reqs.go` | 720 | MEDIUM | Split into reqs.go, reqs_install.go |
| `cli/src/internal/testing/coverage.go` | 709 | MEDIUM | Split into coverage.go, coverage_reports.go |
| `cli/src/internal/installer/installer.go` | 708 | MEDIUM | Split into installer.go, installer_runners.go |
| `cli/src/cmd/app/commands/generate.go` | 689 | MEDIUM | Split into generate.go, generate_templates.go |
| `cli/src/internal/healthcheck/checker.go` | 688 | MEDIUM | Split into checker.go, checker_http.go |
| `cli/src/internal/service/types.go` | 686 | LOW | Split into types.go, types_config.go |
| `cli/src/cmd/app/commands/mcp_tools.go` | 663 | LOW | Split into mcp_tools.go, mcp_tools_services.go |
| `cli/src/cmd/app/commands/run.go` | 639 | LOW | Extract helpers |

### 2. Deprecated Code to Address

| Location | Symbol | Status | Action |
|----------|--------|--------|--------|
| `internal/output/progress.go:529` | `Stop()` | Active | Mark deprecated, add migration guide |
| `commands/deps.go:179` | `runCommand` | Deprecated | Remove if unused or migrate callers |
| `commands/logs.go:964` | `buildLogFilter` | Deprecated | Remove wrapper, use internal directly |
| `commands/reqs.go:432` | `checkPrerequisitesSync` | Deprecated | Remove if no callers |

### 3. Code Duplication to Address

The `portmanager.go` `AssignPort` function has 3 nearly identical port conflict handling blocks (~150 lines each):
1. Explicit port conflict (lines 255-405)
2. Previously assigned port conflict (lines 440-580)  
3. Preferred port conflict (lines 595-750)

**Proposed refactor**: Extract common logic to:
- `handlePortConflict(port int, serviceName string, isExplicit bool) (action PortConflictAction, err error)`
- `PortConflictAction` enum: `ActionKill`, `ActionReassign`, `ActionCancel`, `ActionAlwaysKill`

## Out of Scope

- Test files (even if >200 lines) - comprehensiveness is acceptable
- Generated files that get rebuilt
- Third-party dependencies
- magefile.go (build tooling, different standards)

## Success Criteria

1. All HIGH priority files split to under 400 lines (interim goal)
2. MEDIUM priority files split to under 500 lines
3. Deprecated functions removed or marked with clear migration path
4. Duplicated port conflict handling consolidated
5. All existing tests continue to pass
6. No functionality changes - pure refactoring
