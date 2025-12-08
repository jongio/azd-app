# azd app test - Multi-Language Testing Framework

## Overview

The `azd app test` command provides unified test execution and coverage aggregation across multi-language applications. It auto-detects test frameworks, runs tests in parallel, and generates aggregated coverage reports suitable for CI/CD pipelines.

## Scope

- Test execution for Node.js, Python, .NET, and Go projects
- Automatic framework detection (Jest, Vitest, pytest, xUnit, go test, etc.)
- Support for unit, integration, and e2e test types
- Aggregated code coverage across all services
- Multiple output formats (console, JSON, JUnit, GitHub Actions)
- Watch mode for development
- Setup/teardown commands per test type

## Requirements

### Command Structure

- `azd app test` - Run all tests for all services
- Flags:
  - `--type, -t` - Test type: unit, integration, e2e, all (default: all)
  - `--coverage, -c` - Enable code coverage collection
  - `--service, -s` - Filter to specific service(s)
  - `--watch, -w` - Watch mode for development
  - `--threshold` - Minimum coverage percentage (0-100)
  - `--fail-fast` - Stop on first failure
  - `--parallel, -p` - Run services in parallel (default: true)
  - `--verbose, -v` - Verbose output
  - `--dry-run` - Show what would be tested
  - `--output-format` - Output format: default, json, junit, github
  - `--output-dir` - Directory for test reports

### Language Support

#### Node.js
- Frameworks: Jest, Vitest, Mocha, npm test (fallback)
- Coverage: Built-in (Jest/Vitest), c8, nyc
- Detection: package.json, config files (jest.config.js, vitest.config.ts)
- Package managers: npm, pnpm, yarn

#### Python
- Frameworks: pytest, unittest
- Coverage: pytest-cov, coverage.py
- Detection: pytest.ini, pyproject.toml, tests/ directory
- Package managers: uv, poetry, pip

#### .NET
- Frameworks: xUnit, NUnit, MSTest
- Coverage: coverlet
- Detection: *.csproj with test references, Tests.csproj naming
- Test filtering via `--filter Category=Unit`

#### Go
- Frameworks: go test (standard library)
- Coverage: go test -cover
- Detection: go.mod, *_test.go files
- Test filtering via -run flag with naming patterns

### Test Type Separation

- Tests should be organized by type (unit, integration, e2e)
- Each type can have:
  - Custom command
  - Markers/filters/patterns
  - Setup commands (run before tests)
  - Teardown commands (run after tests)
- Default detection:
  - Unit: tests marked/named `unit`, tests in `unit/` directories
  - Integration: tests marked/named `integration`
  - E2E: tests marked/named `e2e` or `end-to-end`

### Coverage Aggregation

- Collect coverage from each service after test execution
- Convert all formats to common format (Cobertura XML)
- Merge coverage data across services
- Generate reports:
  - JSON summary
  - HTML report
  - Cobertura XML (for CI integration)
- Validate against threshold
- Exit with error if threshold not met

### Output Formats

#### Default (Console)
- Progress indicator per service
- Summary of passed/failed/skipped tests
- Coverage percentages if enabled
- Colored output with symbols (✓, ✗)

#### JSON
- Machine-readable results
- Full test details including failures
- Coverage metrics per service and aggregate

#### JUnit XML
- Standard JUnit format for CI systems
- Test suites per service
- Failure messages and stack traces

#### GitHub Actions
- Annotations for failed tests
- Summary in workflow output
- Coverage comment on PR (when applicable)

### Configuration (azure.yaml)

```yaml
test:
  parallel: true
  coverageThreshold: 80
  outputDir: ./test-results

services:
  web:
    language: js
    project: ./src/web
    test:
      framework: vitest
      unit:
        command: pnpm test:unit
      integration:
        command: pnpm test:integration
        setup:
          - docker-compose up -d postgres
        teardown:
          - docker-compose down
      coverage:
        threshold: 85

  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        markers: [unit]
      integration:
        markers: [integration]
        setup:
          - docker-compose up -d
      coverage:
        source: src
        threshold: 90

  gateway:
    language: go
    project: ./src/gateway
    test:
      framework: gotest
      unit:
        pattern: "^Test[^Integration]"
      integration:
        pattern: "^TestIntegration"
      coverage:
        threshold: 80

  apphost:
    language: csharp
    project: ./src/AppHost
    test:
      framework: xunit
      unit:
        filter: "Category=Unit"
      integration:
        filter: "Category=Integration"
        projects:
          - ./tests/Integration.Tests.csproj
```

### Error Handling

- Continue on test failures unless --fail-fast
- Report partial results even with failures
- Clear error messages with context
- Distinguish between test failures and execution errors

### Performance

- Parallel execution across services (configurable)
- Efficient file watching for watch mode
- Caching of framework detection results
- Minimal overhead for command orchestration

### Security

- Validate paths stay within project boundaries
- No shell command injection (use exec.Command with args)
- Sanitize test output for display
- Respect .gitignore for coverage reports

## Success Criteria

- All four languages fully supported (Node.js, Python, .NET, Go)
- Auto-detection works for common framework configurations
- Coverage aggregation produces valid Cobertura XML
- Test coverage for `internal/testing/` reaches 80%
- Unit tests exist for all runners and orchestrator
- Integration tests with sample polyglot project
- Documentation complete with examples

## Out of Scope

- IDE integration (VS Code test explorer)
- Mutation testing
- Test impact analysis (affected tests detection)
- Visual regression testing
- Parallel test execution within a single service (framework-specific)
- Test sharding across machines
