# `azd app test` Implementation Checklist

This checklist tracks the implementation of the polyglot test command with unified coverage and reporting.

## Phase 1: Core Infrastructure ✓

### Project Detection
- [ ] Create `src/internal/tester/tester.go`
  - [ ] `TestProject` struct definition
  - [ ] `FindTestProjects()` function
  - [ ] Node.js project detection
  - [ ] Python project detection
  - [ ] .NET project detection
  - [ ] Go project detection
  - [ ] Unit tests for detection logic

### Test Type Detection
- [ ] Create `src/internal/tester/detector.go`
  - [ ] `DetectNodeTests()` - parse package.json scripts
  - [ ] `DetectPythonTests()` - find test directories/markers
  - [ ] `DetectDotnetTests()` - find test projects with xunit/nunit
  - [ ] `DetectGoTests()` - find *_test.go files
  - [ ] Unit tests for each detector

### Security Validation
- [ ] Extend `src/internal/security/validation.go`
  - [ ] `ValidateTestScript()` - sanitize test script names
  - [ ] `ValidateTestPath()` - validate test file paths
  - [ ] Security tests for injection attacks

## Phase 2: Test Execution ✓

### Test Runners
- [ ] Create `src/internal/tester/node.go`
  - [ ] `RunNodeTests()` - execute npm/pnpm/yarn test
  - [ ] Add JUnit reporter flag
  - [ ] Add coverage flag
  - [ ] Handle custom script names

- [ ] Create `src/internal/tester/python.go`
  - [ ] `RunPythonTests()` - execute pytest
  - [ ] Support poetry/uv/pip execution
  - [ ] Add JUnit XML output flag
  - [ ] Add coverage flag
  - [ ] Support test markers and directories

- [ ] Create `src/internal/tester/dotnet.go`
  - [ ] `RunDotnetTests()` - execute dotnet test
  - [ ] Add category filters
  - [ ] Add JUnit logger flag
  - [ ] Add coverage collector flag

- [ ] Create `src/internal/tester/go.go`
  - [ ] `RunGoTests()` - execute go test
  - [ ] Add JSON output flag
  - [ ] Add coverage profile flag
  - [ ] Support -short and -run flags

### Test Result Collection
- [ ] Create `src/internal/tester/results.go`
  - [ ] `TestResult` struct
  - [ ] `TestResults` aggregation struct
  - [ ] `CollectTestResults()` function
  - [ ] Parse test output for pass/fail counts

## Phase 3: Coverage Aggregation ✓

### Coverage Structures
- [ ] Create `src/internal/coverage/coverage.go`
  - [ ] `CoverageReport` struct
  - [ ] `ProjectCoverage` struct
  - [ ] `FileCoverage` struct
  - [ ] `Cobertura` XML structs

### Format Converters
- [ ] Create `src/internal/coverage/converters.go`
  - [ ] `ConvertLcovToCobertura()` - Node.js LCOV → Cobertura
  - [ ] `ConvertJSONCoverageToCobertura()` - Jest JSON → Cobertura
  - [ ] `ConvertGoCoverageToCobertura()` - Go coverage → Cobertura
  - [ ] `parseLcovFile()` helper
  - [ ] `parseGoCoverageFile()` helper
  - [ ] Unit tests for each converter

### Coverage Merging
- [ ] Create `src/internal/coverage/merger.go`
  - [ ] `MergeCoberturaFiles()` - combine multiple XML files
  - [ ] `parseCoberturaFile()` - read Cobertura XML
  - [ ] `normalizePath()` - make paths workspace-relative
  - [ ] `calculateOverallLineRate()` - aggregate metrics
  - [ ] `calculateOverallBranchRate()` - aggregate branch coverage
  - [ ] Unit tests for merging logic

## Phase 4: Reporting ✓

### Coverage Reports
- [ ] Create `src/internal/coverage/reporter.go`
  - [ ] `GenerateReport()` - main entry point
  - [ ] `generateTerminalReport()` - colored terminal output
  - [ ] `generateHTMLReport()` - interactive HTML dashboard
  - [ ] `generateJSONReport()` - JSON export
  - [ ] `writeCoberturaXML()` - merged Cobertura output
  - [ ] Color coding helpers (green/yellow/red)
  - [ ] Coverage icon helpers (✅⚠️❌)

### Test Result Reports
- [ ] Create `src/internal/tester/junit.go`
  - [ ] JUnit XML structures
  - [ ] `ParseJUnitXML()` - read JUnit files
  - [ ] `MergeJUnitXML()` - combine multiple files
  - [ ] `ConvertToJUnitXML()` - format conversion

- [ ] Create `src/internal/tester/converters_test.go`
  - [ ] `ConvertJestToJUnit()` - Jest JSON → JUnit
  - [ ] `ConvertPytestToJUnit()` - validation only (pytest has JUnit)
  - [ ] `ConvertTRXToJUnit()` - .NET TRX → JUnit
  - [ ] `ConvertGoTestToJUnit()` - Go JSON → JUnit

### Unified Reporter
- [ ] Create `src/internal/tester/reporter.go`
  - [ ] `GenerateUnifiedReport()` - tests + coverage
  - [ ] `generateTerminalUnifiedReport()` - combined output
  - [ ] `generateHTMLUnifiedReport()` - combined HTML
  - [ ] `generateJSONUnifiedReport()` - combined JSON
  - [ ] `generateJUnitReport()` - test results only

## Phase 5: Command Implementation ✓

### Command Structure
- [ ] Create `src/cmd/app/commands/test.go`
  - [ ] `newTestCommand()` - Cobra command definition
  - [ ] `TestOptions` struct with all flags
  - [ ] `executeTest()` - main execution logic
  - [ ] `runTestsParallel()` - concurrent execution
  - [ ] `runTestsSequential()` - serial execution
  - [ ] `displayTestSummary()` - final output
  - [ ] Error handling and exit codes

### Flag Definitions
- [ ] Add flags to test command
  - [ ] `--type` / `-t` - test type selection
  - [ ] `--project` / `-p` - specific project
  - [ ] `--parallel` - parallel execution (default true)
  - [ ] `--fail-fast` / `-f` - stop on first failure
  - [ ] `--coverage` / `-c` - collect coverage
  - [ ] `--coverage-format` - output format
  - [ ] `--coverage-threshold` - minimum coverage %
  - [ ] `--coverage-files` - per-file breakdown
  - [ ] `--report-format` - test results format
  - [ ] `--watch` / `-w` - watch mode
  - [ ] `--verbose` / `-v` - verbose output
  - [ ] `--filter` - test name filter
  - [ ] `--output-dir` - report directory

### Orchestrator Integration
- [ ] Update `src/cmd/app/commands/core.go`
  - [ ] Register test command with orchestrator
  - [ ] Set dependency: test → deps → reqs
  - [ ] Update init() function

## Phase 6: Configuration Support ✓

### Custom Configuration
- [ ] Create `src/internal/tester/config.go`
  - [ ] `TestConfig` struct for azd-test.yaml
  - [ ] `LoadTestConfig()` - read YAML config
  - [ ] `validateConfig()` - validate configuration
  - [ ] Support for custom command mappings

### Workspace Configuration
- [ ] Create `src/internal/tester/workspace.go`
  - [ ] `WorkspaceConfig` struct
  - [ ] `LoadWorkspaceConfig()` - read workspace config
  - [ ] Support for project-specific configs
  - [ ] Global settings (parallel, fail-fast, thresholds)

## Phase 7: Test Fixtures ✓

### Node.js Fixtures
- [ ] Create `tests/projects/with-tests/node-jest/`
  - [ ] package.json with test scripts
  - [ ] jest.config.js
  - [ ] src/math.js (code to test)
  - [ ] __tests__/unit/math.test.js
  - [ ] __tests__/e2e/api.test.js

- [ ] Create `tests/projects/with-tests/node-vitest/`
  - [ ] package.json with Vitest
  - [ ] vitest.config.js
  - [ ] src/ and tests/ directories

### Python Fixtures
- [ ] Create `tests/projects/with-tests/python-pytest/`
  - [ ] pyproject.toml with pytest config
  - [ ] pytest.ini
  - [ ] src/calculator.py
  - [ ] tests/unit/test_calculator.py
  - [ ] tests/e2e/test_api.py

### .NET Fixtures
- [ ] Create `tests/projects/with-tests/dotnet-xunit/`
  - [ ] src/MyApp/MyApp.csproj
  - [ ] src/MyApp/Services/UserService.cs
  - [ ] tests/MyApp.Tests/MyApp.Tests.csproj
  - [ ] tests/MyApp.Tests/UserServiceTests.cs

### Go Fixtures
- [ ] Create `tests/projects/with-tests/go-testing/`
  - [ ] go.mod
  - [ ] calculator.go
  - [ ] calculator_test.go
  - [ ] integration_test.go

### Polyglot Fixture
- [ ] Create `tests/projects/with-tests/polyglot-workspace/`
  - [ ] frontend/ (Node.js)
  - [ ] backend/ (Python)
  - [ ] services/ (. NET)
  - [ ] shared/ (Go)

## Phase 8: Integration Tests ✓

### Test Coverage
- [ ] Create `src/cmd/app/commands/test_integration_test.go`
  - [ ] `TestTestCommand_NodeJestProject`
  - [ ] `TestTestCommand_PythonPytestProject`
  - [ ] `TestTestCommand_DotnetXunitProject`
  - [ ] `TestTestCommand_GoTestingProject`
  - [ ] `TestTestCommand_PolyglotWorkspace`
  - [ ] `TestCoverageAggregation`
  - [ ] `TestParallelExecution`
  - [ ] `TestFailFast`
  - [ ] `TestCoverageThreshold`

### Unit Tests
- [ ] Test detector functions
- [ ] Test converter functions
- [ ] Test merger logic
- [ ] Test reporter output
- [ ] Test security validation
- [ ] Achieve 80%+ coverage

## Phase 9: Documentation ✓

### Technical Documentation
- [ ] Complete `docs/test-command-spec.md`
  - [x] Overview and design principles
  - [x] Command signature and flags
  - [x] Test detection strategy
  - [x] Execution flow
  - [x] Coverage aggregation
  - [x] Reporting formats
  - [x] Local testing strategy
  - [x] User guide sections

- [ ] Create `docs/test-command-quickstart.md`
  - [x] 5-minute setup
  - [x] Common commands
  - [x] Ecosystem-specific setup
  - [x] Troubleshooting
  - [x] CI/CD examples

### User Documentation
- [ ] Update `README.md`
  - [ ] Add test command to command list
  - [ ] Add quick example
  - [ ] Link to detailed docs

- [ ] Update `CONTRIBUTING.md`
  - [ ] Add guidelines for test fixtures
  - [ ] Add testing requirements

- [ ] Update `docs/add-command-guide.md`
  - [ ] Add test command as example

## Phase 10: Local Testing ✓

### Testing Scripts
- [ ] Create `scripts/test-locally.ps1`
  - [ ] Build and install extension
  - [ ] Run test on each fixture project
  - [ ] Validate output and exit codes
  - [ ] Check coverage file generation

### Manual Testing
- [ ] Test Node.js jest project
- [ ] Test Node.js vitest project
- [ ] Test Python pytest project
- [ ] Test .NET xunit project
- [ ] Test Go testing project
- [ ] Test polyglot workspace
- [ ] Test parallel execution
- [ ] Test fail-fast mode
- [ ] Test coverage aggregation
- [ ] Test all report formats

### Edge Cases
- [ ] No tests found
- [ ] Missing test runners
- [ ] Invalid coverage files
- [ ] Mixed pass/fail results
- [ ] Coverage below threshold
- [ ] Custom configuration files

## Phase 11: CI/CD Integration ✓

### GitHub Actions
- [ ] Create `.github/workflows/test-command.yml`
  - [ ] Run tests on all fixture projects
  - [ ] Validate coverage aggregation
  - [ ] Upload coverage to Codecov
  - [ ] Check exit codes

### Azure Pipelines
- [ ] Create example pipeline YAML
  - [ ] Run tests with coverage
  - [ ] Publish test results
  - [ ] Publish coverage results

## Phase 12: Performance Optimization ✓

### Parallel Execution
- [ ] Implement goroutine-based parallel test execution
- [ ] Add context for cancellation (fail-fast)
- [ ] Optimize coverage file I/O
- [ ] Cache parsed coverage data

### Memory Optimization
- [ ] Stream large coverage files
- [ ] Limit buffered test output
- [ ] Clean up temp files after aggregation

## Phase 13: Polish & Release ✓

### Error Messages
- [ ] Helpful error messages with suggestions
- [ ] Link to documentation in errors
- [ ] Color-coded error output

### Logging
- [ ] Progress indicators for long operations
- [ ] Verbose mode with detailed logs
- [ ] Debug mode for troubleshooting

### Examples
- [ ] Add example projects to repository
- [ ] Create video/GIF demos
- [ ] Blog post with walkthrough

### Release Prep
- [ ] Update CHANGELOG.md
- [ ] Version bump
- [ ] Release notes
- [ ] Migration guide for existing users

## Acceptance Criteria

### Functional Requirements
- [ ] ✅ Detects Node.js, Python, .NET, and Go test projects
- [ ] ✅ Runs unit, e2e, and integration tests by type
- [ ] ✅ Collects coverage from all languages
- [ ] ✅ Converts all formats to Cobertura XML
- [ ] ✅ Merges coverage into unified report
- [ ] ✅ Generates terminal, HTML, JSON, JUnit reports
- [ ] ✅ Supports parallel and sequential execution
- [ ] ✅ Implements fail-fast mode
- [ ] ✅ Enforces coverage thresholds
- [ ] ✅ Supports custom configuration

### Non-Functional Requirements
- [ ] ✅ 80%+ test coverage
- [ ] ✅ All integration tests pass
- [ ] ✅ Performance: <5s overhead for report generation
- [ ] ✅ Memory: <100MB for coverage aggregation
- [ ] ✅ Documentation complete and accurate
- [ ] ✅ Examples work out-of-box
- [ ] ✅ CI/CD integration validated

### User Experience
- [ ] ✅ Works with zero config for standard projects
- [ ] ✅ Clear error messages with next steps
- [ ] ✅ Beautiful terminal output
- [ ] ✅ Interactive HTML reports
- [ ] ✅ Fast parallel execution
- [ ] ✅ Helpful documentation

## Progress Tracking

- **Phase 1-4**: Core Infrastructure ⬜ 0%
- **Phase 5-6**: Command & Config ⬜ 0%
- **Phase 7-8**: Testing ⬜ 0%
- **Phase 9**: Documentation ✅ 100%
- **Phase 10-11**: Validation ⬜ 0%
- **Phase 12-13**: Polish ⬜ 0%

**Overall Progress**: 📊 ~8% (Documentation phase complete)

---

## Notes for Implementers

1. **Start Small**: Begin with Node.js support, then add other languages
2. **Test Early**: Create test fixtures before implementing features
3. **Incremental**: Each phase should be testable independently
4. **Reference Existing**: Follow patterns from deps/run/reqs commands
5. **Security First**: Always validate user inputs (paths, scripts)
6. **Performance**: Profile parallel execution with many projects
7. **Documentation**: Update docs as you implement, not after

## Quick Start for Development

```bash
# 1. Create test fixtures
mkdir -p tests/projects/with-tests/node-jest
# ... add package.json, tests, etc.

# 2. Implement detection
# Edit src/internal/tester/tester.go

# 3. Write unit tests
# Edit src/internal/tester/tester_test.go

# 4. Test locally
go test ./src/internal/tester -v

# 5. Implement command
# Edit src/cmd/app/commands/test.go

# 6. Build and test
.\build.ps1
.\install-local.ps1
cd tests/projects/with-tests/node-jest
azd app test

# 7. Iterate!
```

Good luck! 🚀
