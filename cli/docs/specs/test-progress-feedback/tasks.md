# Test Command Progress Feedback - Implementation Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on
- DONE: Completed
- BLOCKED: Blocked by dependency

## Tasks

### Task 1: Add Service Validation Before Test Execution [Developer]
**Status:** DONE
**Description:** Add pre-test validation to check each service for testability before running.
**Files:**
- `internal/testing/validation.go` - Created with `ValidateService()`, language-specific validators
- `internal/testing/validation_test.go` - 32+ test cases

**Implementation:**
- `ServiceValidation` struct with Name, Language, Framework, TestFiles, CanTest, SkipReason
- Language validators: validateNodeService, validatePythonService, validateGoService, validateDotnetService
- Test file counting with `countTestFiles()` using `security.ValidatePath`
- Helper functions: fileExists, dirExists, readFileContains

### Task 2: Add Progress Display During Test Execution [Developer]
**Status:** DONE
**Description:** Integrate MultiProgress into test orchestrator for real-time feedback.
**Files:**
- `internal/testing/orchestrator.go` - Added ProgressCallback, ProgressEvent, progress emission
- `cmd/app/commands/test.go` - Added createProgressCallback, displayValidationSummary, showDryRunConfig

**Implementation:**
- ProgressEvent struct with Type, Service, Message, Data fields
- Event types: Validating, ValidationComplete, TestStarting, TestProgress, TestComplete, AllComplete
- SetProgressCallback(), emitProgress(), GetServices(), ValidateAllServices(), ExecuteTestsWithValidation()
- Dry-run now shows validation summary before config

### Task 3: Add Pre-Test Summary Output [Developer]
**Status:** DONE
**Description:** Display analysis summary before running tests.
**Files:**
- `cmd/app/commands/test.go` - displayValidationSummary() function

**Implementation:**
- Shows "Analyzing N services..." header
- Lists each service with detected framework and test file count
- Marks services that will be skipped with warning icon and reason
- Shows final count: "Found N testable services (M skipped)"
- Integrated into runTests() flow before test execution

### Task 4: Add Config Auto-Discovery Save Prompt [Developer]
**Status:** DONE
**Description:** Prompt users to save discovered test configuration.
**Files:**
- `internal/testing/config_writer.go` - Created GenerateTestConfigYAML, GetAutoDetectedServices, SaveTestConfigToAzureYaml
- `internal/testing/config_writer_test.go` - Unit tests for YAML generation, merging, edge cases
- `cmd/app/commands/test.go` - Added --save and --no-save flags, promptToSaveConfig()

**Implementation:**
- GenerateTestConfigYAML() produces clean YAML snippet
- GetAutoDetectedServices() filters services without existing config
- SaveTestConfigToAzureYaml() merges into existing file preserving content
- Interactive prompt shows config before asking to save
- Respects TTY/JSON mode, --save forces save, --no-save skips prompt

### Task 5: Add Smart Streaming Output [Developer]
**Status:** DONE
**Description:** Implement context-aware output mode selection with streaming as default.
**Files:**
- `internal/testing/output_mode.go` - Created OutputMode type, SelectOutputMode(), isCI(), IsTTY()
- `internal/testing/output_mode_test.go` - 13 test cases
- `cmd/app/commands/test.go` - Added --stream and --no-stream flags

**Implementation:**
- OutputMode constants: OutputModeStream, OutputModeStreamPrefixed, OutputModeProgress
- OutputModeOptions struct for configuration
- Smart selection: CI→stream, single→stream, parallel→progress, sequential→prefixed
- Mutual exclusion validation for --stream and --no-stream

### Task 6: Add Timeout and Stall Detection [Developer]
**Status:** DONE
**Description:** Add timeout protection and stall warnings.
**Files:**
- `internal/testing/types.go` - Added Timeout field to TestConfig
- `internal/testing/orchestrator.go` - Added executeWithTimeout(), DefaultTestTimeout
- `cmd/app/commands/test.go` - Added --timeout flag (default 10m)
- `internal/testing/orchestrator_test.go` - Timeout tests

**Implementation:**
- DefaultTestTimeout = 10 minutes
- executeWithTimeout() wraps test execution with context timeout
- Clear error: "test execution timed out after X"
- Flag accepts duration strings (5m, 30s, 1h)

### Task 7: Unit Tests for Progress Feedback [Tester]
**Status:** DONE
**Depends on:** Task 2
**Description:** Test progress display functionality.
**Files:**
- `internal/testing/validation_test.go` - 32+ test cases for validation
- `internal/testing/output_mode_test.go` - 13 test cases for output mode selection
- `internal/testing/config_writer_test.go` - Config writer tests
- `internal/testing/orchestrator_test.go` - Progress and timeout tests

**Coverage:** Testing package at ~78% coverage

### Task 8: Integration Tests [Tester]
**Status:** DONE
**Depends on:** Tasks 1-6
**Description:** End-to-end tests for new functionality.
**Files:**
- `internal/testing/discovery_test.go` - New file with 10 discovery-specific tests
- `tests/projects/discovery-test/` - New test project for manual and automated testing

**Implementation:**
- TestDiscovery_LoadDiscoveryProject - loads 6 services
- TestDiscovery_ValidateServices - validates testability per service
- TestDiscovery_FrameworkDetection - detects vitest, jest, pytest, gotest
- TestDiscovery_TestFileCount - counts test files
- TestDiscovery_SkipReason - verifies skip reasons for non-testable services
- TestDiscovery_NestedTestFiles - tests deeply nested __tests__ folders
- TestDiscovery_ConfigAutoDetection - identifies auto-detected services
- TestDiscovery_GenerateYAML - tests YAML generation
- TestDiscovery_OutputModeSelection - tests smart output mode selection

### Task 10: Create Discovery Test Project [Developer]
**Status:** DONE
**Description:** Create a test project without any test config for manual testing.
**Files:**
- `tests/projects/discovery-test/azure.yaml` - 6 services, no test config
- `tests/projects/discovery-test/web/` - vitest project
- `tests/projects/discovery-test/api/` - jest project
- `tests/projects/discovery-test/backend/` - pytest project
- `tests/projects/discovery-test/gateway/` - go test project
- `tests/projects/discovery-test/config/` - NO tests (should be skipped)
- `tests/projects/discovery-test/nested/` - tests in deeply nested __tests__ folder

**Test Scenarios:**
- Standard framework detection (vitest, jest, pytest, go test)
- Service without tests (should skip with reason)
- Nested test file detection (__tests__ folder)
- Auto-discovery config generation

### Task 9: Update Documentation [Developer]
**Status:** DONE
**Depends on:** Tasks 1-6
**Description:** Update CLI reference and spec documentation.
**Files:**
- `docs/cli-reference.md` - Added new flags, smart output modes section, troubleshooting tip

**Implementation:**
- Added --stream, --no-stream, --timeout, --save, --no-save to flags table
- Added "Smart Output Modes" section documenting auto-selection behavior
- Added troubleshooting tip about using --stream if tests appear to hang

## Dependencies

```
Task 1 (Validation) ─┬─► Task 2 (Progress) ─┬─► Task 7 (Unit Tests)
                     │                       │
                     └─► Task 3 (Summary)    └─► Task 8 (Integration)
                     │
Task 4 (Config Save) ─┴─► Task 5 (Streaming) ─┴─► Task 9 (Docs)
                     │
Task 6 (Timeout) ────┘
```

## Coverage Targets

- `validation.go`: 85%+
- `config_writer.go`: 80%+
- `orchestrator.go` (new code): 80%+

## Timeline Estimate

| Phase | Tasks | Duration |
|-------|-------|----------|
| Validation | 1, 3 | 1 day |
| Progress Display | 2 | 1 day |
| Config Save | 4 | 1 day |
| Streaming + Timeout | 5, 6 | 1 day |
| Testing | 7, 8 | 1 day |
| Documentation | 9 | 0.5 day |
| **Total** | | **5.5 days** |
