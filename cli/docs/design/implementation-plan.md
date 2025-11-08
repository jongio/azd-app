# Implementation Plan: azd app test Command

## Summary

This document provides the implementation plan for adding comprehensive testing capabilities to the `azd app` extension, including the `azd app test` command with multi-language support, test type separation, and aggregated code coverage.

## Goals

1. Implement `azd app test` command that runs tests across all services
2. Support Node.js, Python, and .NET test frameworks
3. Enable test type separation (unit, integration, e2e)
4. Provide aggregated code coverage across all services
5. Integrate seamlessly with existing command orchestration
6. Support CI/CD workflows with multiple output formats

## Implementation Phases

### Phase 1: Core Infrastructure (Priority: High)

**Duration**: 1-2 weeks

#### 1.1 Type Definitions

**File**: `cli/src/internal/testing/types.go`

Define core data structures:
- `TestConfig` - Global test configuration
- `ServiceTestConfig` - Per-service test configuration
- `TestResult` - Test execution results
- `CoverageData` - Coverage metrics
- `AggregateResult` - Combined results from all services

**Acceptance Criteria**:
- All types properly documented
- Types support JSON marshaling for output
- Types integrate with existing `internal/types/types.go`

#### 1.2 Test Orchestrator

**File**: `cli/src/internal/testing/orchestrator.go`

Core orchestration logic:
- Parse `azure.yaml` for test configurations
- Detect test frameworks (if not explicitly configured)
- Manage parallel vs sequential execution
- Aggregate results from all services
- Handle errors and timeouts

**Key Functions**:
```go
func NewTestOrchestrator(config *TestConfig) *TestOrchestrator
func (o *TestOrchestrator) DetectTests(serviceDir string, language string) (*ServiceTestConfig, error)
func (o *TestOrchestrator) ExecuteTests(services []ServiceConfig, testType string) (*AggregateResult, error)
func (o *TestOrchestrator) RunParallel(services []ServiceConfig, testType string) ([]*TestResult, error)
```

**Acceptance Criteria**:
- Successfully parses azure.yaml test configurations
- Auto-detects test frameworks when not configured
- Executes tests in parallel and sequential modes
- Properly aggregates results from all services
- Unit test coverage > 80%

#### 1.3 Test Command

**File**: `cli/src/cmd/app/commands/test.go`

Command implementation:
- Define command with all flags
- Integrate with orchestrator
- Basic output formatting (default, JSON)
- Error handling

**Key Flags**:
- `--type` (unit/integration/e2e/all)
- `--coverage`
- `--service` (filter)
- `--parallel`
- `--fail-fast`
- `--threshold`

**Acceptance Criteria**:
- Command registered in main.go
- All flags working correctly
- Integration with orchestrator
- Basic output formats (default, JSON)
- Unit tests for command logic

### Phase 2: Language-Specific Runners (Priority: High)

**Duration**: 2-3 weeks

#### 2.1 Node.js Test Runner

**File**: `cli/src/internal/testing/node_runner.go`

Features:
- Detect framework (Jest, Vitest, Mocha)
- Execute tests with appropriate commands
- Parse test output
- Collect coverage (lcov, Istanbul)

**Framework Detection**:
1. Check for config files (`jest.config.js`, `vitest.config.ts`)
2. Check package.json dependencies
3. Default to npm test

**Acceptance Criteria**:
- Detects Jest, Vitest, Mocha correctly
- Executes tests with correct commands
- Parses test output to TestResult
- Collects coverage in lcov format
- Unit tests > 80%, integration tests with real projects

#### 2.2 Python Test Runner

**File**: `cli/src/internal/testing/python_runner.go`

Features:
- Detect framework (pytest, unittest)
- Manage virtual environment activation
- Execute tests with markers
- Collect coverage (pytest-cov)

**Framework Detection**:
1. Check for `pytest.ini`, `pyproject.toml`
2. Check for `tests/` directory structure
3. Default to pytest

**Acceptance Criteria**:
- Detects pytest, unittest correctly
- Activates virtual environment before testing
- Executes tests with markers (unit, integration, e2e)
- Collects coverage in Cobertura XML format
- Unit tests > 80%, integration tests with real venv

#### 2.3 .NET Test Runner

**File**: `cli/src/internal/testing/dotnet_runner.go`

Features:
- Detect framework (xUnit, NUnit, MSTest)
- Find test projects
- Execute tests with filters
- Collect coverage (coverlet)

**Framework Detection**:
1. Scan for `*.Tests.csproj` files
2. Check package references
3. Default to xUnit

**Acceptance Criteria**:
- Detects xUnit, NUnit, MSTest correctly
- Finds all test projects in directory
- Executes tests with category filters
- Collects coverage in Cobertura format
- Unit tests > 80%, integration tests with real .NET projects

### Phase 3: Coverage Aggregation (Priority: High)

**Duration**: 1-2 weeks

#### 3.1 Coverage Aggregator

**File**: `cli/src/internal/testing/coverage.go`

Features:
- Collect coverage from all services
- Convert formats to Cobertura XML
- Merge coverage data
- Calculate aggregate metrics
- Check thresholds

**Format Conversion**:
- LCOV (Jest) → Cobertura XML
- Istanbul JSON (Vitest) → Cobertura XML
- Cobertura XML (pytest, coverlet) → already in format

**Acceptance Criteria**:
- Converts LCOV to Cobertura correctly
- Converts Istanbul JSON to Cobertura correctly
- Merges multiple Cobertura files correctly
- Calculates accurate aggregate metrics
- Validates coverage thresholds
- Unit tests > 80%

#### 3.2 Report Generator

**File**: `cli/src/internal/testing/reporter.go`

Features:
- Generate console output (default)
- Generate JSON output
- Generate JUnit XML output
- Generate HTML coverage reports
- Set GitHub Actions outputs

**Output Formats**:
1. **Default**: Human-readable console
2. **JSON**: Machine-readable results
3. **JUnit**: CI/CD integration
4. **GitHub**: GitHub Actions annotations

**Acceptance Criteria**:
- Generates clear, formatted console output
- Generates valid JSON with all metrics
- Generates valid JUnit XML
- Creates HTML coverage reports
- Sets GitHub Actions outputs correctly
- Unit tests > 80%

### Phase 4: Advanced Features (Priority: Medium)

**Duration**: 1-2 weeks

#### 4.1 Watch Mode

Features:
- Monitor file system for changes
- Re-run tests automatically
- Smart test selection (only affected tests)

**Implementation**:
- Use `fsnotify` or similar library
- Debounce file changes
- Filter relevant file changes

**Acceptance Criteria**:
- Detects file changes correctly
- Re-runs tests automatically
- Provides clear feedback in watch mode
- Can be interrupted gracefully (Ctrl+C)

#### 4.2 Setup/Teardown Commands

Features:
- Execute setup commands before tests
- Execute teardown commands after tests
- Handle failures gracefully

**Configuration**:
```yaml
test:
  integration:
    setup:
      - docker-compose up -d
    teardown:
      - docker-compose down
```

**Acceptance Criteria**:
- Executes setup commands in order
- Executes teardown even on test failure
- Logs setup/teardown output
- Handles command failures gracefully

#### 4.3 Output Directory Management

Features:
- Create output directory structure
- Organize reports by service
- Clean old reports

**Structure**:
```
test-results/
├── coverage/
│   ├── index.html
│   ├── web/
│   ├── api/
│   └── coverage.xml
└── results/
    ├── web-results.xml
    └── api-results.xml
```

**Acceptance Criteria**:
- Creates consistent directory structure
- Organizes reports by service
- Provides clear paths in output

### Phase 5: Testing & Documentation (Priority: High)

**Duration**: 1 week

#### 5.1 Unit Tests

**Coverage Goal**: > 80% for all components

Test coverage for:
- Test orchestrator
- All language runners
- Coverage aggregator
- Report generator
- Command logic

**Acceptance Criteria**:
- All components have unit tests
- Coverage > 80% overall
- Tests follow existing patterns
- Mock external dependencies

#### 5.2 Integration Tests

**Test Projects**:
Create test fixtures in `tests/projects/`:
- `tests/projects/node/test-jest-project/`
- `tests/projects/python/test-pytest-project/`
- `tests/projects/dotnet/test-xunit-project/`

**Test Scenarios**:
- Auto-detection of frameworks
- Test execution with real frameworks
- Coverage collection and merging
- Multi-service test execution

**Acceptance Criteria**:
- Integration tests with real test frameworks
- Tests run in CI pipeline
- Tests validate end-to-end functionality

#### 5.3 Documentation

**Documents to Create/Update**:
- ✅ `docs/commands/test.md` - Complete command reference
- ✅ `docs/design/testing-framework.md` - Architecture and design
- ✅ `docs/schema/test-configuration.md` - YAML configuration schema
- ✅ `docs/cli-reference.md` - Add test command section
- `README.md` - Add test command to feature list

**Acceptance Criteria**:
- All documentation complete and accurate
- Examples are tested and working
- Troubleshooting section covers common issues
- Migration guide for existing projects

## Integration with Existing Code

### Command Registration

**File**: `cli/src/cmd/app/main.go`

Add test command to root command:
```go
rootCmd.AddCommand(
    commands.NewReqsCommand(),
    commands.NewRunCommand(),
    commands.NewDepsCommand(),
    commands.NewTestCommand(),  // NEW
    commands.NewLogsCommand(),
    commands.NewInfoCommand(),
    commands.NewVersionCommand(),
    commands.NewListenCommand(),
)
```

### Orchestrator Integration

The test command should integrate with the existing orchestrator pattern:

```go
// In commands/test.go
func (cmd *testCommand) RunE(cmd *cobra.Command, args []string) error {
    // Execute dependencies first (reqs)
    if err := cmdOrchestrator.Run("test"); err != nil {
        return fmt.Errorf("failed to execute command dependencies: %w", err)
    }
    
    // Run tests
    return runTests(...)
}
```

### Extension Manifest

**File**: `cli/extension.yaml`

Update version and add test example:
```yaml
version: 0.6.0  # Bump version
examples:
  - name: test
    description: Run tests for all services
    usage: azd app test --coverage
```

## Testing Strategy

### Unit Tests

**Goal**: > 80% coverage

**Approach**:
- Test each component in isolation
- Mock external dependencies
- Follow existing test patterns in codebase
- Use table-driven tests for multiple scenarios

**Example**:
```go
func TestDetectNodeFramework(t *testing.T) {
    tests := []struct {
        name     string
        files    map[string]string
        expected string
    }{
        {
            name: "detects jest from config",
            files: map[string]string{
                "jest.config.js": "module.exports = {}",
            },
            expected: "jest",
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

**Goal**: Validate with real frameworks

**Approach**:
- Create test fixture projects
- Run actual test frameworks
- Verify output parsing
- Validate coverage collection

**Test Projects**:
```
tests/projects/
├── node/
│   └── test-jest-project/
│       ├── package.json
│       ├── jest.config.js
│       └── src/
│           ├── math.js
│           └── __tests__/
│               └── math.test.js
├── python/
│   └── test-pytest-project/
│       ├── pyproject.toml
│       └── tests/
│           ├── unit/
│           │   └── test_math.py
│           └── integration/
│               └── test_db.py
└── dotnet/
    └── test-xunit-project/
        ├── App/
        │   └── Math.cs
        └── App.Tests/
            └── MathTests.cs
```

### E2E Tests

**Goal**: Validate complete workflows

**Scenarios**:
1. Run all tests with coverage
2. Run specific test types
3. Run tests for specific services
4. Parallel execution
5. Coverage threshold validation

## Dependencies

### Go Dependencies

No new dependencies expected - use existing:
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- Standard library for everything else

### Test Dependencies

For integration tests:
- Node.js, npm/pnpm (already in devcontainer)
- Python, pytest (already in devcontainer)
- .NET SDK (already in devcontainer)

## Risks and Mitigations

### Risk 1: Framework Detection Complexity

**Risk**: Auto-detection may fail for edge cases

**Mitigation**:
- Provide explicit configuration option
- Clear error messages when detection fails
- Comprehensive documentation

### Risk 2: Coverage Format Conversion

**Risk**: Converting between coverage formats may lose information

**Mitigation**:
- Use well-tested libraries for conversion
- Validate conversion with real projects
- Support native formats as fallback

### Risk 3: Performance with Large Projects

**Risk**: Running tests for many services may be slow

**Mitigation**:
- Implement parallel execution by default
- Provide service filtering
- Add watch mode for incremental testing

### Risk 4: CI/CD Integration Complexity

**Risk**: Different CI systems need different output formats

**Mitigation**:
- Support multiple output formats (JUnit, JSON, GitHub)
- Provide clear examples for each CI system
- Follow established standards

## Success Criteria

### Functionality

- ✅ Command runs tests for all supported languages
- ✅ Test types (unit, integration, e2e) work independently
- ✅ Coverage is aggregated across all services
- ✅ Multiple output formats work correctly
- ✅ Parallel execution reduces overall test time

### Code Quality

- ✅ Unit test coverage > 80%
- ✅ Integration tests validate real-world usage
- ✅ Code follows existing patterns and conventions
- ✅ No security vulnerabilities introduced
- ✅ Linter passes without errors

### Documentation

- ✅ Complete command reference
- ✅ Architecture and design documentation
- ✅ Configuration schema documentation
- ✅ Examples for all supported languages
- ✅ Troubleshooting guide

### User Experience

- ✅ Clear, actionable error messages
- ✅ Helpful output formatting
- ✅ Fast feedback for common use cases
- ✅ Easy migration from existing test workflows

## Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: Core Infrastructure | 1-2 weeks | Types, orchestrator, basic command |
| Phase 2: Language Runners | 2-3 weeks | Node.js, Python, .NET runners |
| Phase 3: Coverage | 1-2 weeks | Aggregation, reporting |
| Phase 4: Advanced Features | 1-2 weeks | Watch mode, setup/teardown |
| Phase 5: Testing & Docs | 1 week | Tests, documentation |
| **Total** | **6-10 weeks** | Complete testing framework |

## Next Steps

1. ✅ **Create design documentation** (This document and related docs)
2. **Get approval** from project maintainer
3. **Implement Phase 1** - Core infrastructure
4. **Implement Phase 2** - Language runners
5. **Implement Phase 3** - Coverage aggregation
6. **Implement Phase 4** - Advanced features
7. **Complete Phase 5** - Testing and documentation
8. **Release** as part of next version (v0.6.0)

## Related Documents

- [Test Command Specification](commands/test.md)
- [Testing Framework Design](design/testing-framework.md)
- [Test Configuration Schema](schema/test-configuration.md)
- [CLI Reference](cli-reference.md)
