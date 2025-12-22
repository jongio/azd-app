# Command Hooks Implementation Tasks

<!-- NEXT: 1 -->

## TODO: Phase 1 - Schema & Infrastructure

### 1. Update types.go with new hook fields
- **File**: `cli/src/internal/service/types.go`
- **Changes**:
  - Add 14 new hook fields to `Hooks` struct (pretest, posttest, predeps, postdeps, prestart, poststart, prestop, poststop, prerestart, postrestart, preadd, postadd, prereqs, postreqs)
  - Add getter methods for all 14 hooks following existing pattern
  - Ensure YAML tags are correct (`yaml:"pretest,omitempty"`, etc.)

### 2. Create shared hook helper functions
- **File**: `cli/src/cmd/app/commands/hooks_common.go` (new file)
- **Changes**:
  - Extract common hook execution logic from run.go
  - Create `loadAzureYamlForHooks()` helper
  - Create `executeCommandHook()` generic helper
  - Create `buildCommonHookEnvVars()` for shared env vars

## TODO: Phase 2 - High Priority Commands

### 3. Implement test command hooks
- **File**: `cli/src/cmd/app/commands/test.go`
- **Changes**:
  - Add hook execution before/after test run
  - Create `buildTestHookEnvVars()` for test-specific variables
  - Handle errors from pre-hook (stop execution)
  - Handle errors from post-hook (log warning)

### 4. Add test command hook tests
- **File**: `cli/src/cmd/app/commands/test_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePretestHook_Success
  - TestExecutePosttestHook_Success
  - TestExecutePretestHook_Failure
  - TestExecutePretestHook_NoHooks
  - TestBuildTestHookEnvironmentVariables

### 5. Implement deps command hooks
- **File**: `cli/src/cmd/app/commands/deps.go`
- **Changes**:
  - Add hook execution before/after dependency installation
  - Create `buildDepsHookEnvVars()` for deps-specific variables
  - Load azure.yaml if available (optional, deps works without it)

### 6. Add deps command hook tests
- **File**: `cli/src/cmd/app/commands/deps_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePredepsHook_Success
  - TestExecutePostdepsHook_Success
  - TestExecutePredepsHook_NoAzureYaml
  - TestBuildDepsHookEnvironmentVariables

### 7. Implement start command hooks
- **File**: `cli/src/cmd/app/commands/start.go`
- **Changes**:
  - Add hook execution before/after service start
  - Create `buildServiceHookEnvVars()` for service operation variables
  - Share env var builder with stop/restart

### 8. Add start command hook tests
- **File**: `cli/src/cmd/app/commands/start_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePrestartHook_Success
  - TestExecutePoststartHook_Success
  - TestBuildServiceHookEnvironmentVariables

### 9. Implement stop command hooks
- **File**: `cli/src/cmd/app/commands/stop.go`
- **Changes**:
  - Add hook execution before/after service stop
  - Use shared `buildServiceHookEnvVars()`

### 10. Add stop command hook tests
- **File**: `cli/src/cmd/app/commands/stop_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePrestopHook_Success
  - TestExecutePoststopHook_Success

### 11. Implement restart command hooks
- **File**: `cli/src/cmd/app/commands/restart.go`
- **Changes**:
  - Add hook execution before/after service restart
  - Use shared `buildServiceHookEnvVars()`

### 12. Add restart command hook tests
- **File**: `cli/src/cmd/app/commands/restart_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePrerestartHook_Success
  - TestExecutePostrestartHook_Success

## TODO: Phase 3 - Medium Priority Commands

### 13. Implement add command hooks
- **File**: `cli/src/cmd/app/commands/add.go`
- **Changes**:
  - Add hook execution before/after service addition
  - Create `buildAddHookEnvVars()` for add-specific variables

### 14. Add add command hook tests
- **File**: `cli/src/cmd/app/commands/add_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePreaddHook_Success
  - TestExecutePostaddHook_Success
  - TestBuildAddHookEnvironmentVariables

### 15. Implement reqs command hooks
- **File**: `cli/src/cmd/app/commands/reqs.go`
- **Changes**:
  - Add hook execution before/after requirements check
  - Create `buildReqsHookEnvVars()` for reqs-specific variables

### 16. Add reqs command hook tests
- **File**: `cli/src/cmd/app/commands/reqs_hooks_test.go` (new file)
- **Tests**:
  - TestExecutePrereqsHook_Success
  - TestExecutePostreqsHook_Success
  - TestBuildReqsHookEnvironmentVariables

## TODO: Phase 4 - Documentation

### 17. Update schema documentation
- **File**: `cli/docs/schema/azure.yaml.md`
- **Changes**:
  - Update `hooks` section to document all 14 new hooks
  - Add examples for each hook type
  - Update Hook Object section with use cases

### 18. Update test command documentation
- **File**: `cli/docs/commands/test.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document pretest and posttest hooks
  - Add examples (build before test, upload coverage)
  - Document environment variables

### 19. Update deps command documentation
- **File**: `cli/docs/commands/deps.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document predeps and postdeps hooks
  - Add examples (clean deps, security audit)
  - Document environment variables

### 20. Update start command documentation
- **File**: `cli/docs/commands/start.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document prestart and poststart hooks
  - Add examples (validate env, register services)
  - Document environment variables

### 21. Update stop command documentation
- **File**: `cli/docs/commands/stop.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document prestop and poststop hooks
  - Add examples (drain connections, archive logs)
  - Document environment variables

### 22. Update restart command documentation
- **File**: `cli/docs/commands/restart.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document prerestart and postrestart hooks
  - Add examples (backup state, verify health)
  - Document environment variables

### 23. Update add command documentation
- **File**: `cli/docs/commands/add.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document preadd and postadd hooks
  - Add examples (backup config, configure service)
  - Document environment variables

### 24. Update reqs command documentation
- **File**: `cli/docs/commands/reqs.md`
- **Changes**:
  - Add "Lifecycle Hooks" section
  - Document prereqs and postreqs hooks
  - Add examples (update package managers, configure tools)
  - Document environment variables

### 25. Update features/hooks.md
- **File**: `cli/docs/features/hooks.md`
- **Changes**:
  - Add section for each new hook type
  - Add comprehensive examples for all hooks
  - Add complete workflow examples
  - Document all environment variables

### 26. Update README if needed
- **File**: `cli/README.md` or root `README.md`
- **Changes**:
  - Update hooks documentation links if present
  - Add hooks to feature list if not already there

## TODO: Phase 5 - Integration Tests

### 27. Add test command integration tests
- **File**: `cli/src/cmd/app/commands/test_hooks_integration_test.go` (new file)
- **Tests**:
  - TestTestWithHooks_Integration
  - TestTestPreHookFailure
  - TestTestPostHookContinueOnError

### 28. Add deps command integration tests
- **File**: `cli/src/cmd/app/commands/deps_hooks_integration_test.go` (new file)
- **Tests**:
  - TestDepsWithHooks_Integration
  - TestDepsPreHookFailure

### 29. Add start/stop/restart integration tests
- **File**: `cli/src/cmd/app/commands/service_lifecycle_hooks_integration_test.go` (new file)
- **Tests**:
  - TestStartWithHooks_Integration
  - TestStopWithHooks_Integration
  - TestRestartWithHooks_Integration

### 30. Add add command integration tests
- **File**: `cli/src/cmd/app/commands/add_hooks_integration_test.go` (new file)
- **Tests**:
  - TestAddWithHooks_Integration
  - TestAddPreHookValidation

### 31. Add reqs command integration tests
- **File**: `cli/src/cmd/app/commands/reqs_hooks_integration_test.go` (new file)
- **Tests**:
  - TestReqsWithHooks_Integration
  - TestReqsPostHookConfiguration

## DONE

_(Completed tasks will be moved here)_
