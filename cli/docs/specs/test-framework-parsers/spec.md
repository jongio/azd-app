# Test Framework JSON Parsing System

## Overview

Replace text-based test output parsing with structured JSON parsing for reliable, accurate test result extraction across all supported languages and frameworks.

## Problem Statement

Current text-based parsing is fragile:
- ANSI color codes interfere with regex matching
- Output format varies between framework versions
- International/locale-specific output breaks parsing
- Edge cases (long test names, special characters) cause failures

## Solution

Use JSON output from test frameworks wherever available, with a pluggable parser architecture for extensibility.

## Scope

### In Scope
- JSON output configuration for all supported frameworks
- Unified parser interface for framework-agnostic result handling
- Test projects for each supported framework
- Coverage JSON parsing improvements

### Out of Scope
- Adding new language support
- IDE integrations
- Custom reporter development for frameworks lacking JSON

## Supported Frameworks

### Node.js
| Framework | JSON Support | CLI Flag | Notes |
|-----------|-------------|----------|-------|
| Jest | Native | `--json --outputFile=results.json` | Most common, well-documented |
| Vitest | Native | `--reporter=json --outputFile=results.json` | Jest-compatible schema |
| Mocha | Native | `--reporter json` | Simpler schema |
| Playwright | Native | `--reporter=json` | E2E focused |

### Python
| Framework | JSON Support | Method | Notes |
|-----------|-------------|--------|-------|
| pytest | Plugin | `pytest-json-report` | Most common Python framework |
| unittest | None | Use pytest runner | Fallback to exit code |
| Robot Framework | Native (7.0+) | `--output output.json` | Automation focused |

### .NET
| Framework | JSON Support | Method | Notes |
|-----------|-------------|--------|-------|
| xUnit | TRX/XML | `--logger trx` | Convert TRX to JSON |
| NUnit | XML | `--result results.xml` | Convert to JSON |
| MSTest | TRX | `--logger trx` | Same as xUnit |

### Go
| Framework | JSON Support | CLI Flag | Notes |
|-----------|-------------|----------|-------|
| go test | Native | `-json` | NDJSON (newline-delimited) |
| Ginkgo | Native | `--json-report=results.json` | BDD framework |

## Architecture

### Parser Interface

```
TestResultParser interface:
  - Parse(jsonData []byte) -> TestResult
  - SupportsFramework(framework string) -> bool
  - GetOutputConfig() -> OutputConfig

OutputConfig:
  - Command: string        // e.g., "npm"
  - Args: []string         // e.g., ["test", "--", "--json"]
  - OutputFile: string     // e.g., "jest-results.json" (if file-based)
  - UsesStdout: bool       // true if JSON comes from stdout
```

### Parser Registry

```
ParserRegistry:
  - Register(parser TestResultParser)
  - GetParser(framework string) -> TestResultParser
  - ListSupported() -> []string
```

### Unified Result Schema

```
TestResult:
  Success: bool
  Total: int
  Passed: int
  Failed: int
  Skipped: int
  Duration: float64 (seconds)
  Suites: []TestSuite
  Error: string

TestSuite:
  Name: string
  Tests: []TestCase
  Duration: float64

TestCase:
  Name: string
  Status: "passed" | "failed" | "skipped"
  Duration: float64
  Error: TestError (optional)

TestError:
  Message: string
  Stack: string
  Expected: string (optional)
  Actual: string (optional)
```

## Framework-Specific Details

### Jest/Vitest JSON Schema

```json
{
  "numTotalTests": 10,
  "numPassedTests": 9,
  "numFailedTests": 1,
  "numPendingTests": 0,
  "success": false,
  "testResults": [{
    "name": "/path/to/test.js",
    "status": "failed",
    "assertionResults": [{
      "title": "test name",
      "status": "passed|failed|pending",
      "duration": 123,
      "failureMessages": ["error..."]
    }]
  }]
}
```

### Go test -json Schema (NDJSON)

```json
{"Time":"...","Action":"run","Package":"pkg","Test":"TestName"}
{"Time":"...","Action":"output","Package":"pkg","Test":"TestName","Output":"..."}
{"Time":"...","Action":"pass","Package":"pkg","Test":"TestName","Elapsed":0.001}
```

Actions: `start`, `run`, `pause`, `cont`, `pass`, `bench`, `fail`, `skip`, `output`

### pytest-json-report Schema

```json
{
  "summary": {
    "total": 10,
    "passed": 9,
    "failed": 1,
    "skipped": 0
  },
  "duration": 1.234,
  "tests": [{
    "nodeid": "test_file.py::test_name",
    "outcome": "passed|failed|skipped",
    "duration": 0.001,
    "call": {
      "longrepr": "error message..."
    }
  }]
}
```

### .NET TRX to JSON Conversion

TRX files are XML. Parse and convert to unified schema:
- `//UnitTestResult/@outcome` -> status
- `//UnitTestResult/@duration` -> duration
- `//ErrorInfo/Message` -> error message
- `//ErrorInfo/StackTrace` -> stack trace

## Configuration (azure.yaml)

```yaml
services:
  api:
    language: js
    test:
      framework: jest
      # Optional: override auto-detection
      json:
        enabled: true
        outputFile: ./test-results/jest.json

  backend:
    language: python
    test:
      framework: pytest
      json:
        plugin: pytest-json-report  # Will be auto-installed if missing
        outputFile: ./test-results/pytest.json

  gateway:
    language: go
    test:
      framework: gotest
      # go test -json writes to stdout, no file needed
```

## Fallback Strategy

When JSON parsing fails or isn't available:
1. Try JSON output first
2. Fall back to text parsing with ANSI stripping
3. Last resort: use exit code only (0=pass, non-zero=fail)

Log warnings when using fallbacks to encourage framework upgrades.

## Test Projects

Create test projects in `tests/projects/framework-tests/`:

```
framework-tests/
  node-jest/
    package.json
    src/
    __tests__/
  node-vitest/
    package.json
    vitest.config.ts
    src/
    tests/
  node-mocha/
    package.json
    test/
  python-pytest/
    pyproject.toml
    src/
    tests/
  dotnet-xunit/
    Tests.csproj
    UnitTests.cs
  dotnet-nunit/
    Tests.csproj
    UnitTests.cs
  go-gotest/
    go.mod
    main.go
    main_test.go
  go-ginkgo/
    go.mod
    suite_test.go
```

Each project should have:
- 3+ passing tests
- 1 intentionally failing test (disabled by default)
- 1 skipped test
- Unit and integration test examples
- Coverage configuration

## Success Criteria

1. JSON parsing works for all Tier 1 frameworks (Jest, Vitest, pytest, go test, xUnit)
2. Accurate test counts (passed/failed/skipped/total)
3. Duration extracted correctly
4. Error messages preserved for failed tests
5. 90%+ test coverage for parser code
6. Integration tests with all framework test projects
7. Graceful fallback when JSON unavailable
8. Structured log events emitted for CI/CD integration

## Structured Log Output

When `--structured-logs` or `--output json` is enabled, emit NDJSON events to stderr:

### Event Types

```json
{"event": "test:start", "service": "api", "framework": "jest", "type": "unit", "timestamp": "2025-01-15T10:30:00Z"}
{"event": "test:progress", "service": "api", "current": 5, "total": 10, "timestamp": "..."}
{"event": "test:result", "service": "api", "test": "should return 200", "status": "passed", "duration": 0.023}
{"event": "test:result", "service": "api", "test": "should validate input", "status": "failed", "error": "expected 400", "duration": 0.015}
{"event": "test:complete", "service": "api", "passed": 9, "failed": 1, "skipped": 0, "duration": 2.34}
{"event": "test:coverage", "service": "api", "lines": 85.2, "branches": 72.1, "functions": 90.0}
{"event": "test:summary", "passed": 25, "failed": 1, "skipped": 2, "duration": 5.67, "coverage": 82.5}
```

### CI-Specific Output

| CI System | Output Format |
|-----------|---------------|
| GitHub Actions | `::error file=...::` annotations |
| Azure Pipelines | `##vso[task.logissue]` commands |
| GitLab CI | Section markers, artifacts |
| Generic | NDJSON + exit codes |

## Migration Path

1. Implement JSON parsers alongside existing text parsers
2. Default to JSON when framework supports it
3. Keep text parsers as fallback
4. Deprecate text-only parsing in future release

## Security Considerations

- Validate JSON before parsing (size limits, schema validation)
- Sanitize error messages before display
- Temp files for JSON output cleaned up after parsing
- No shell injection in command construction

## Performance

- JSON parsing is faster than regex text parsing
- Single file read vs. streaming stdout parsing
- Parallel parsing for multi-service projects

## Dependencies

- No new external dependencies for core parsers
- pytest-json-report: optional Python plugin (auto-installable)
- TRX parser: standard XML parsing (stdlib)
