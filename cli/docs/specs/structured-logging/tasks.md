# Structured Logging - Implementation Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on
- DONE: Completed

## Tasks

### Task 1: Enhance Logger Package [Developer]
**Status:** DONE
**Description:** Add component-based logging with context propagation.
- Add `NewLogger(component string)` factory function
- Add `WithService(name string)` context method
- Add `WithOperation(op string)` context method
- Add `WithFields(args ...any)` for arbitrary context
- Add test-specific helpers: `TestStarted`, `TestCompleted`, `CoverageCollected`
- Add `IsStructured()` function to check logging mode
- Ensure backwards compatibility with existing functions

### Task 2: Migrate Testing Orchestrator [Developer]
**Status:** DONE
**Description:** Replace fmt.Printf calls in orchestrator.go with structured logging.
- Replace warning logs with logging.Warn
- Replace progress logs with logging.Info/Debug
- Add component="test" to all logs
- Add service context for per-service operations

### Task 3: Migrate Testing Watcher [Developer]
**Status:** DONE
**Description:** Replace fmt.Printf calls in watcher.go with structured logging.
- Replace file change notifications with structured logs
- Keep emoji output for console mode (check !logging.IsStructured())
- Add watch-specific log events
- Log elapsed time and affected services

### Task 4: Migrate Testing Reporter [Developer]
**Status:** DONE
**Description:** Replace fmt.Printf calls in reporter.go with structured logging.
- Keep GitHub Actions annotations as-is (special format required)
- Replace warning logs with logging.Warn
- Add structured coverage reporting logs

### Task 5: Add Unit Tests [Tester]
**Status:** DONE
**Description:** Test the enhanced logging infrastructure.
- Test NewLogger creates correct component context
- Test WithService/WithOperation add correct fields
- Test JSON output format when structured=true
- Test text output format when structured=false
- Test backwards compatibility with existing functions
- Note: Existing tests in logging and testing packages pass

### Task 6: Documentation Update [Developer]
**Status:** DONE
**Description:** Document logging conventions and flags.
- --debug and --structured-logs flags already documented in main.go
- Spec serves as documentation for logging conventions
- Examples of filtering logs by component in spec.md

## Dependencies

- Task 1 blocks Tasks 2, 3, 4
- Task 5 can run after Task 1
- Task 6 runs last

## Timeline Estimate

| Phase | Tasks | Duration |
|-------|-------|----------|
| Infrastructure | 1, 5 | 1 day |
| Migration | 2, 3, 4 | 1 day |
| Documentation | 6 | 0.5 day |
| **Total** | | **2.5 days** |
