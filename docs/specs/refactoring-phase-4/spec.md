# Refactoring Phase 4 - Specification

## Overview

Comprehensive refactoring pass to address code duplication, large files, magic numbers, and dead code across the azd-app codebase.

## Priority Framework

- **CRITICAL**: Files >1000 lines or blocking maintainability
- **HIGH**: Files >500 lines or significant duplication
- **MEDIUM**: Files >300 lines or moderate duplication
- **LOW**: Code hygiene, cleanup, minor improvements

## Scope

### Backend (Go)
- `cli/src/internal/dashboard/` - Azure logs handlers and server files
- `cli/src/internal/detector/` - Already refactored, validation only
- `cli/src/internal/installer/` - Error formatting duplication
- Test files - Split oversized test suites

### Frontend (React/TypeScript)
- `cli/dashboard/src/components/ConsoleView.tsx` (1,337 lines)
- `cli/dashboard/src/components/LogsPane.tsx` (1,317 lines)
- `cli/dashboard/src/lib/service-utils.ts` (872 lines)
- Extract hooks and utilities

### Web (Astro/TypeScript)
- `web/scripts/` - Split large script files
- `web/src/pages/` - Convert to content collections where appropriate
- `web/src/components/` - Split large components

### Constants Extraction
- Go timeouts and limits
- TypeScript UI/log/WebSocket constants
- Error code strings

## Goals

1. **Reduce file sizes**: Target ≤200 lines per file (max 300 for complex files)
2. **Eliminate duplication**: Extract common patterns to utilities
3. **Replace magic numbers**: Create constant files
4. **Remove dead code**: Clean up commented code and unused functions
5. **Improve testability**: Smaller files are easier to test

## Success Criteria

- No files >500 lines (except intentional exceptions)
- 30% reduction in code duplication
- All magic numbers documented as constants
- 100% test pass rate maintained
- No functional regressions

## Out of Scope

- Idiomatic Go patterns (e.g., `if err != nil`)
- Architecture changes
- Performance optimization (unless identified during refactor)
- New features

## Risks

- Breaking changes if module boundaries incorrect
- Test failures if refactoring introduces bugs
- Merge conflicts with active branches

## Mitigation

- Run full test suite after each major split
- Incremental commits per file split
- Preserve git history with clear commit messages
- Test locally before committing
