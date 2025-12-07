# azd app test - Implementation Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on
- DONE: Completed

## Tasks

### Task 1: Add Go Test Runner [Developer]
**Status:** DONE
**Description:** Implement GoTestRunner for Go projects with test execution and coverage collection.
- Create `internal/testing/go_runner.go`
- Support `go test` with `-v`, `-cover`, `-coverprofile` flags
- Parse go test output (PASS/FAIL/SKIP counts)
- Support test pattern filtering via `-run` flag
- Detect Go projects (go.mod, *_test.go files)
- Create `internal/testing/go_runner_test.go` with unit tests

### Task 2: Enhance Language Detection in Orchestrator [Developer]
**Status:** DONE
**Description:** Add Go language support to orchestrator and improve detection logic.
- Add "go" to language switch in `executeServiceTests`
- Add Go detection in `DetectTestConfig`
- Update `detectGoTestFramework` function
- Add Go to the list of supported languages in types

### Task 3: Improve Coverage Parsing [Developer]
**Status:** DONE
**Description:** Add parsers for each language's native coverage format.
- Parse Jest/Vitest coverage-final.json (Istanbul format)
- Parse pytest coverage.xml (Cobertura)
- Parse Go coverprofile format
- Parse .NET coverlet Cobertura XML
- Unified conversion to internal CoverageData type

### Task 4: Add JUnit Output Format [Developer]
**Status:** DONE
**Description:** Implement JUnit XML output for CI/CD integration.
- Create JUnit XML structure types
- Generate test suite per service
- Include failure messages and stack traces
- Write to output directory

### Task 5: Add GitHub Actions Output Format [Developer]
**Status:** DONE
**Description:** Implement GitHub Actions specific output.
- Create problem annotations for failed tests
- Output summary to GITHUB_STEP_SUMMARY
- Set output variables for coverage
- Support GitHub-specific error formatting

### Task 6: Create Polyglot Test Project [Developer]
**Status:** DONE
**Description:** Create a test project with all four languages for testing.
- Create `tests/projects/polyglot-test/` directory
- Add Node.js service with Vitest and tests
- Add Python service with pytest and tests
- Add Go service with go test and tests
- Add .NET service with xUnit and tests
- Create azure.yaml with test configuration
- Include unit, integration, and e2e test examples

### Task 7: Integration Tests for Test Command [Tester]
**Status:** DONE
**Description:** Write integration tests using the polyglot test project.
- Test running all tests
- Test running specific test types
- Test service filtering
- Test coverage collection
- Test threshold validation
- Test output formats

### Task 8: Unit Tests for Go Runner [Tester]
**Status:** DONE
**Description:** Comprehensive unit tests for GoTestRunner.
- Test command building
- Test output parsing (pass/fail/skip)
- Test coverage parsing
- Test pattern filtering
- Achieve 80%+ coverage

### Task 9: Unit Tests for Output Formats [Tester]
**Status:** DONE
**Description:** Unit tests for JUnit and GitHub Actions output.
- Test JUnit XML generation
- Test GitHub annotations
- Test file writing
- Test error handling

### Task 10: Improve Test Type Detection [Developer]
**Status:** DONE
**Description:** Better auto-detection of test types within services.
- Detect test directories (unit/, integration/, e2e/)
- Detect test file patterns by type
- Detect markers/attributes by type
- Fallback to "all" when type not determinable
**Implementation:**
- Created `detection.go` with `TestTypeDetector` struct
- Directory detection: unit/, integration/, e2e/ and nested variations
- File pattern detection: language-specific patterns (e.g., `_unit_test.go`, `test_unit*.py`)
- Marker detection: pytest marks, xUnit traits, Go build tags
- Added `SuggestTestTypeConfig()` for auto-configuration
- Integrated into `DetectTestConfig()` in orchestrator
- Added `GetAvailableTestTypes()` method to orchestrator
- Full test coverage in `detection_test.go`

### Task 11: Watch Mode Improvements [Developer]
**Status:** DONE
**Description:** Enhance file watching for better developer experience.
- Debounce file change events
- Only re-run affected service tests
- Clear console between runs
- Show elapsed time since last run
- Support ctrl+C for clean exit
**Implementation:**
- Added debouncing with configurable `debounceDelay` (default 300ms)
- Added `WatchWithServiceFilter` to track affected services
- Added `getAffectedServices()` to map file changes to services
- Added `WithClearConsole(bool)` option to clear terminal
- Added `WithShowElapsedTime(bool)` option to show time since last run
- Signal handling for clean Ctrl+C exit with `syscall.SIGINT/SIGTERM`
- Added functional options: `WithDebounceDelay`, `WithServicePathMap`
- Added `SetPollInterval`, `AddIgnorePattern`, `SetServicePathMap` methods
- Full test coverage in `watcher_test.go`

### Task 12: Coverage Report Generation [Developer]
**Status:** DONE
**Description:** Generate comprehensive coverage reports.
- HTML report with source file linking
- Cobertura XML with full file details
- JSON summary report
- Per-service and aggregate views
- File-level and line-level coverage
**Implementation:**
- Enhanced `CoverageData` and `FileCoverage` types with `LineHits` map for line-level tracking
- JSON report: Added `CoverageJSONReport` with timestamp, threshold info, summary, per-service and per-file coverage
- Cobertura XML: Added `CoberturaLine` for line-level hits, source root support, full file details per service
- HTML reports:
  - Main index with summary grid, progress bars, threshold status, service table, file table
  - Per-service pages with file listings
  - Per-file pages with source code highlighting showing covered/uncovered lines
- Added `GenerateAllReports()` convenience method
- Added `SetSourceRoot()` for source file linking
- Added helper functions: `sanitizeFilename`, `getProgressClass`, `getThresholdClass`, `getThresholdMessage`
- Updated `go_runner.go` to parse line hits from coverage profile

### Task 13: Documentation Update [Developer]
**Status:** DONE
**Description:** Update documentation with Go support and examples.
- Update cli-reference.md
- Update testing-framework-spec.md
- Add Go examples to configuration docs
- Update README with test command info

### Task 14: Code Review and Security [SecOps]
**Status:** DONE
**Description:** Security review of test command implementation.
- Verify path validation
- Check for command injection vulnerabilities
- Review file write operations
- Validate external command execution
**Fixes Applied:**
- Fixed shell command injection in `runCommand` - now parses commands safely
- Added path traversal protection in `LoadServicesFromAzureYaml`
- Added HTML escaping in coverage report to prevent XSS
- Added tests for `parseCommandString` and path traversal protection

### Task 15: Clean Up Legacy Docs [Developer]
**Status:** DONE
**Description:** Remove or archive outdated documentation.
- Archive old testing-framework-spec.md ✓ Added note referencing new spec
- Update design/testing-framework.md ✓ Added note referencing new spec
- Consolidate documentation structure ✓ All docs now reference specs/azd-app-test/
- Remove duplicate content ✓ Kept design docs for architecture reference

## Coverage Targets

### internal/testing/ Coverage Goals:
- orchestrator.go: 80%+
- node_runner.go: 80%+
- python_runner.go: 80%+
- dotnet_runner.go: 80%+
- go_runner.go: 80%+
- coverage.go: 80%+
- watcher.go: 80%+
- types.go: 100% (data types)

### cmd/app/commands/ Coverage Goals:
- test.go: 70%+ (orchestrator function excluded)

## Dependencies

- Task 1 blocks Task 2, Task 6, Task 8
- Task 6 blocks Task 7
- Tasks 4, 5 can run in parallel after core implementation
- Task 14 should run after all code changes
- Task 15 should run last

## Timeline Estimate

| Phase | Tasks | Duration |
|-------|-------|----------|
| Go Support | 1, 2, 8 | 3 days |
| Output Formats | 4, 5, 9 | 2 days |
| Test Project | 6, 7 | 2 days |
| Improvements | 3, 10, 11, 12 | 3 days |
| Documentation | 13, 15 | 1 day |
| Security Review | 14 | 1 day |
| **Total** | | **12 days** |
