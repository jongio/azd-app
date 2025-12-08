# Test Framework JSON Parsing - Implementation Tasks

## Status Key
- TODO: Not started
- IN PROGRESS: Currently being worked on  
- DONE: Completed

---

## Phase 1: Core Architecture [Developer]

### Task 1.1: Define Parser Interface and Types
**Status:** TODO
**Assignee:** Developer
**Description:** Create the parser interface and unified result types.
- Create `internal/testing/parser/types.go` with unified schema
- Create `internal/testing/parser/interface.go` with parser interface
- Create `internal/testing/parser/registry.go` for parser registration
- Add OutputConfig struct for framework-specific command configuration
- Unit tests for types and registry

### Task 1.2: Create Base Parser Implementation
**Status:** TODO
**Assignee:** Developer
**Description:** Implement base parser with common functionality.
- Create `internal/testing/parser/base.go` with shared parsing logic
- JSON validation and size limits
- Error handling and fallback support
- Duration normalization (ms to seconds)
- Unit tests for base parser

---

## Phase 2: Node.js Parsers [Developer]

### Task 2.1: Jest JSON Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement Jest JSON output parser.
- Create `internal/testing/parser/jest.go`
- Parse Jest JSON schema (numTotalTests, testResults, assertionResults)
- Extract test names, status, duration, error messages
- Configure `--json --outputFile` flags
- Unit tests with sample Jest JSON output

### Task 2.2: Vitest JSON Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement Vitest JSON output parser.
- Create `internal/testing/parser/vitest.go`
- Leverage Jest parser (Vitest is Jest-compatible)
- Configure `--reporter=json --outputFile` flags
- Verify compatibility with Vitest-specific fields
- Unit tests with sample Vitest JSON output

### Task 2.3: Mocha JSON Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement Mocha JSON output parser.
- Create `internal/testing/parser/mocha.go`
- Parse Mocha JSON schema (stats, tests, failures)
- Configure `--reporter json` flag
- Handle stdout-based JSON output
- Unit tests with sample Mocha JSON output

---

## Phase 3: Python Parsers [Developer]

### Task 3.1: pytest JSON Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement pytest-json-report parser.
- Create `internal/testing/parser/pytest.go`
- Parse pytest-json-report schema (summary, tests, duration)
- Auto-detect/suggest pytest-json-report plugin installation
- Configure `--json-report --json-report-file` flags
- Unit tests with sample pytest JSON output

### Task 3.2: pytest Plugin Auto-Installation
**Status:** TODO
**Assignee:** Developer
**Description:** Handle pytest-json-report plugin availability.
- Detect if pytest-json-report is installed
- Add to deps installation if missing (pyproject.toml or requirements.txt)
- Fallback to exit-code-only if plugin unavailable
- Integration test for plugin detection

---

## Phase 4: Go Parser [Developer]

### Task 4.1: go test JSON Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement go test -json NDJSON parser.
- Create `internal/testing/parser/gotest.go`
- Parse NDJSON stream (line-by-line JSON events)
- Track test lifecycle (run → pass/fail/skip)
- Aggregate results from multiple packages
- Handle benchmark output (Action: bench)
- Unit tests with sample go test -json output

---

## Phase 5: .NET Parser [Developer]

### Task 5.1: TRX XML Parser
**Status:** TODO
**Assignee:** Developer
**Description:** Implement .NET TRX format parser.
- Create `internal/testing/parser/trx.go`
- Parse TRX XML schema (UnitTestResult, outcome, duration)
- Extract error messages and stack traces
- Configure `--logger trx` flag
- Convert to unified TestResult schema
- Unit tests with sample TRX files

---

## Phase 6: Framework Test Projects [Developer]

### Task 6.1: Jest Test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create Jest test project for integration testing.
- Create `tests/projects/framework-tests/node-jest/`
- package.json with Jest configuration
- 3 passing tests, 1 skipped test
- Disabled failing test for error message testing
- Coverage configuration

### Task 6.2: Vitest Test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create Vitest test project for integration testing.
- Create `tests/projects/framework-tests/node-vitest/`
- package.json and vitest.config.ts
- 3 passing tests, 1 skipped test
- Disabled failing test
- Coverage configuration with c8/istanbul

### Task 6.3: Mocha Test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create Mocha test project for integration testing.
- Create `tests/projects/framework-tests/node-mocha/`
- package.json with Mocha configuration
- 3 passing tests, 1 skipped test
- Disabled failing test
- NYC coverage configuration

### Task 6.4: pytest Test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create pytest test project for integration testing.
- Create `tests/projects/framework-tests/python-pytest/`
- pyproject.toml with pytest and pytest-json-report
- 3 passing tests, 1 skipped test (pytest.mark.skip)
- Disabled failing test (pytest.mark.xfail)
- pytest-cov configuration

### Task 6.5: xUnit Test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create xUnit test project for integration testing.
- Create `tests/projects/framework-tests/dotnet-xunit/`
- Tests.csproj with xUnit references
- 3 passing tests, 1 skipped test ([Skip])
- Disabled failing test
- Coverlet configuration

### Task 6.6: go test Project
**Status:** TODO
**Assignee:** Developer
**Description:** Create Go test project for integration testing.
- Create `tests/projects/framework-tests/go-gotest/`
- go.mod with testify (optional)
- 3 passing tests, 1 skipped test (t.Skip)
- Disabled failing test
- Coverage profile configuration

---

## Phase 7: Integration [Developer]

### Task 7.1: Update Node Runner for JSON
**Status:** TODO
**Assignee:** Developer
**Description:** Integrate JSON parsers into NodeTestRunner.
- Update `internal/testing/node_runner.go`
- Detect framework and select appropriate parser
- Configure JSON output flags
- Parse JSON file or stdout
- Fallback to text parsing on failure
- Integration tests

### Task 7.2: Update Python Runner for JSON
**Status:** TODO
**Assignee:** Developer
**Description:** Integrate JSON parsers into PythonTestRunner.
- Update `internal/testing/python_runner.go`
- Detect pytest-json-report availability
- Configure JSON output flags
- Parse JSON file
- Fallback handling
- Integration tests

### Task 7.3: Update Go Runner for JSON
**Status:** TODO
**Assignee:** Developer
**Description:** Integrate JSON parser into GoTestRunner.
- Update `internal/testing/go_runner.go`
- Add `-json` flag to test command
- Parse NDJSON stdout stream
- Integration tests

### Task 7.4: Update .NET Runner for JSON
**Status:** TODO
**Assignee:** Developer
**Description:** Integrate TRX parser into DotnetTestRunner.
- Update `internal/testing/dotnet_runner.go`
- Configure `--logger trx` flag
- Parse TRX file and convert to unified schema
- Integration tests

---

## Phase 8: Testing & Quality [Tester]

### Task 8.1: Parser Unit Tests
**Status:** TODO
**Assignee:** Tester
**Description:** Comprehensive unit tests for all parsers.
- Test each parser with valid JSON
- Test malformed JSON handling
- Test missing fields handling
- Test edge cases (empty results, very long test names)
- Achieve 90%+ coverage for parser package

### Task 8.2: Integration Tests with Framework Projects
**Status:** TODO
**Assignee:** Tester
**Description:** End-to-end tests using framework test projects.
- Test `azd app test` against each framework project
- Verify correct test counts
- Verify duration extraction
- Verify error message extraction
- Test coverage aggregation

### Task 8.3: Fallback Testing
**Status:** TODO
**Assignee:** Tester
**Description:** Test fallback behavior when JSON unavailable.
- Test with JSON disabled
- Test with malformed JSON
- Test with missing JSON file
- Verify graceful degradation
- Verify warning messages

---

## Phase 9: Security & Review [SecOps]

### Task 9.1: Security Review
**Status:** TODO
**Assignee:** SecOps
**Description:** Security review of parser implementation.
- JSON size limit validation
- No shell injection in command construction
- Temp file cleanup verification
- Error message sanitization
- Path traversal prevention in output files

---

## Phase 10: Documentation [Developer]

### Task 10.1: Update CLI Reference
**Status:** TODO
**Assignee:** Developer
**Description:** Update documentation with JSON parsing details.
- Update cli-reference.md with supported frameworks
- Document framework detection logic
- Document fallback behavior
- Add troubleshooting section

### Task 10.2: Update azure.yaml Schema
**Status:** TODO
**Assignee:** Developer
**Description:** Add JSON configuration options to schema.
- Add `test.json.enabled` option
- Add `test.json.outputFile` option
- Add `test.json.plugin` option (for pytest)
- Update schema documentation

---

## Phase 11: Structured Log Output [Developer]

### Task 11.1: Define Test Event Schema
**Status:** TODO
**Assignee:** Developer
**Description:** Define structured log events for test command.
- `test:start` - Test execution beginning for a service
- `test:progress` - Incremental progress during test run
- `test:result` - Individual test case result
- `test:complete` - Service test run complete
- `test:summary` - Aggregate results across all services
- `test:coverage` - Coverage metrics
- Document event schema in spec

### Task 11.2: Implement Test Event Emitter
**Status:** TODO
**Assignee:** Developer
**Description:** Create event emitter for test structured logs.
- Create `internal/testing/events.go`
- Emit events when `--structured-logs` or `--output json` enabled
- Include timestamps, service names, framework info
- Support streaming output (NDJSON to stderr)
- Unit tests for event emission

### Task 11.3: Integrate Events into Test Runners
**Status:** TODO
**Assignee:** Developer
**Description:** Wire up event emission in test runners.
- Emit `test:start` before running tests
- Emit `test:progress` during long-running tests (if framework supports)
- Emit `test:complete` after parsing results
- Emit `test:summary` at end of orchestrator
- Integration tests for event flow

### Task 11.4: CI/CD Output Formats
**Status:** TODO
**Assignee:** Developer
**Description:** Specialized output for CI systems.
- GitHub Actions annotations for failures
- Azure Pipelines ##vso commands
- GitLab CI section markers
- Generic CI mode (exit codes + NDJSON)
- Documentation for CI integration

---

## Dependencies

```
Phase 1 (Architecture) 
    ↓
Phase 2-5 (Parsers) - can run in parallel
    ↓
Phase 6 (Test Projects) - can run in parallel with 2-5
    ↓
Phase 7 (Integration) - depends on 2-6
    ↓
Phase 8 (Testing) - depends on 7
    ↓
Phase 9-10 (Security, Docs) - depends on 8
    ↓
Phase 11 (Structured Logs) - depends on 7 (needs accurate parsed data)
```

## Timeline Estimate

| Phase | Duration | Parallel With |
|-------|----------|---------------|
| Phase 1: Architecture | 1 day | - |
| Phase 2: Node.js Parsers | 2 days | Phase 3-5 |
| Phase 3: Python Parser | 1 day | Phase 2, 4-5 |
| Phase 4: Go Parser | 1 day | Phase 2-3, 5 |
| Phase 5: .NET Parser | 1 day | Phase 2-4 |
| Phase 6: Test Projects | 2 days | Phase 2-5 |
| Phase 7: Integration | 2 days | - |
| Phase 8: Testing | 2 days | - |
| Phase 9: Security | 1 day | Phase 10-11 |
| Phase 10: Documentation | 1 day | Phase 9, 11 |
| Phase 11: Structured Logs | 2 days | Phase 9-10 |
| **Total (with parallelization)** | **~10 days** | |

## Success Metrics

- [ ] All Tier 1 frameworks have JSON parsers (Jest, Vitest, pytest, go test, xUnit)
- [ ] Test counts match expected values in all framework test projects
- [ ] Duration accuracy within 10ms
- [ ] Error messages preserved for failed tests
- [ ] 90%+ test coverage for parser package
- [ ] Graceful fallback when JSON unavailable
- [ ] No regressions in existing functionality
