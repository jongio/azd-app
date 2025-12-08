# azd app test

## Overview

The `test` command provides a comprehensive testing framework that automatically detects and runs tests across all services in your application, supporting multiple languages, test types, and aggregated code coverage reporting.

## Purpose

- **Multi-Language Testing**: Run tests for Node.js, Python, Go, and .NET services
- **Test Type Separation**: Run unit, integration, and e2e tests independently or together
- **Auto-Detection**: Automatically detect test frameworks and configurations
- **Code Coverage**: Generate and aggregate coverage reports across all services
- **Parallel Execution**: Run tests for multiple services concurrently
- **Flexible Filtering**: Run tests for specific services or test types
- **CI/CD Integration**: Easy integration with continuous integration pipelines

## Command Usage

### Basic Commands

```bash
# Run all tests for all services
azd app test

# Run all tests with coverage
azd app test --coverage

# Run only unit tests
azd app test --type unit

# Run only integration tests
azd app test --type integration

# Run only e2e tests
azd app test --type e2e

# Run tests for specific service(s)
azd app test --service api,web

# Run unit tests with coverage for specific service
azd app test --type unit --coverage --service api

# Watch mode - re-run tests on file changes
azd app test --watch

# Update test snapshots
azd app test --update-snapshots

# Fail fast - stop on first failure
azd app test --fail-fast

# Run tests in parallel
azd app test --parallel

# Dry run - show what would be tested
azd app test --dry-run

# Verbose output
azd app test --verbose
```

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--type` | `-t` | string | `all` | Test type to run: `unit`, `integration`, `e2e`, or `all` |
| `--coverage` | `-c` | bool | `false` | Generate code coverage reports |
| `--service` | `-s` | string | `""` | Run tests for specific service(s) (comma-separated) |
| `--watch` | `-w` | bool | `false` | Watch mode - re-run tests on file changes |
| `--update-snapshots` | `-u` | bool | `false` | Update test snapshots (for snapshot testing) |
| `--fail-fast` | | bool | `false` | Stop on first test failure |
| `--parallel` | `-p` | bool | `true` | Run tests for services in parallel (default: true) |
| `--threshold` | | int | `0` | Minimum coverage threshold (0-100) - fail if below |
| `--verbose` | `-v` | bool | `false` | Enable verbose test output |
| `--dry-run` | | bool | `false` | Show what would be tested without running tests |
| `--output-format` | | string | `default` | Output format: `default`, `json`, `junit`, `github` |
| `--output-dir` | | string | `./test-results` | Directory for test reports and coverage |

## Execution Flow

### Overall Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app test                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Flags & Validate Configuration                        │
│  - Validate test type (unit/integration/e2e/all)             │
│  - Validate service filters                                  │
│  - Validate coverage threshold                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Run Prerequisites Check (reqs)                              │
│  - Automatically executed via orchestrator                   │
│  - Ensures test tools are installed                          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse azure.yaml                                            │
│  - Read services section                                     │
│  - Get test configurations                                   │
│  - Determine service types                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detect Test Frameworks (if not configured)                  │
│  - Scan for test files and configurations                    │
│  - Infer test commands                                       │
│  - Detect coverage tools                                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Tests (Parallel or Sequential)                      │
│  - Run tests for each service                                │
│  - Collect results and coverage                              │
│  - Stream output                                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Aggregate Results & Coverage                                │
│  - Combine test results from all services                    │
│  - Merge coverage reports                                    │
│  - Calculate aggregate metrics                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Generate Reports                                            │
│  - Output test results                                       │
│  - Generate coverage reports                                 │
│  - Create summary                                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check Thresholds & Exit                                     │
│  - Validate coverage threshold                               │
│  - Exit with appropriate code                                │
└─────────────────────────────────────────────────────────────┘
```

## Test Type Support

### Test Types

The command supports three test types, each serving a different purpose:

| Type | Purpose | Typical Speed | Examples |
|------|---------|---------------|----------|
| `unit` | Test individual functions/classes in isolation | Fast (ms) | Pure functions, utilities, business logic |
| `integration` | Test interactions between components | Medium (s) | Database operations, API calls, service interactions |
| `e2e` | Test complete user workflows | Slow (min) | UI flows, full API scenarios, end-to-end journeys |

### Running Specific Test Types

```bash
# Unit tests only (fast feedback)
azd app test --type unit

# Integration tests (before deployment)
azd app test --type integration

# E2E tests (smoke tests in CI)
azd app test --type e2e

# All tests (comprehensive validation)
azd app test --type all  # or just: azd app test
```

## Language-Specific Support

### Node.js Testing

#### Supported Frameworks

- **Jest** (default, most common)
- **Vitest** (Vite ecosystem)
- **Mocha** + Chai
- **AVA**
- **Tap**

#### Auto-Detection

The command detects Jest/Vitest by checking:
1. `package.json` scripts: `test`, `test:unit`, `test:integration`, `test:e2e`
2. Configuration files: `jest.config.js`, `vitest.config.ts`
3. Test file patterns: `*.test.js`, `*.spec.js`, `__tests__/*`

#### Default Test Commands

```bash
# Unit tests
npm test -- --testPathPattern=unit
# or
npm run test:unit

# Integration tests
npm test -- --testPathPattern=integration
# or
npm run test:integration

# E2E tests
npm test -- --testPathPattern=e2e
# or
npm run test:e2e

# Coverage
npm test -- --coverage
```

#### Coverage Tools

- **Jest** built-in coverage (Istanbul/c8)
- **Vitest** built-in coverage (c8 or Istanbul)
- **nyc** (for Mocha/AVA)

#### Example Configuration

```yaml
# azure.yaml
services:
  web:
    language: js
    project: ./src/web
    test:
      framework: jest
      unit:
        command: npm run test:unit
        pattern: "**/*.test.js"
      integration:
        command: npm run test:integration
        pattern: "**/*.integration.test.js"
      e2e:
        command: npm run test:e2e
        pattern: "**/*.e2e.test.js"
      coverage:
        enabled: true
        tool: jest
        threshold: 80
        outputFormat: lcov
```

### Go Testing

#### Supported Frameworks

- **go test** (built-in, standard)

#### Auto-Detection

The command detects Go projects by checking:
1. `go.mod` file exists
2. Test file patterns: `*_test.go`
3. Test function patterns: `func Test*`, `func Benchmark*`

#### Default Test Commands

```bash
# All tests
go test ./...

# Unit tests (pattern-based)
go test ./... -run "^Test[^Integration]"

# Integration tests
go test ./... -run "TestIntegration"

# With coverage
go test ./... -cover -coverprofile=coverage.out

# Verbose output
go test ./... -v
```

#### Coverage Tools

- **go test -cover** (built-in coverage)
- **go tool cover** (HTML report generation)

#### Test Types via Patterns

Go uses regex patterns with the `-run` flag for test filtering:

| Type | Default Pattern | Description |
|------|-----------------|-------------|
| `unit` | `^Test[^Integration]` | Tests not containing "Integration" |
| `integration` | `TestIntegration` | Tests containing "Integration" |
| `e2e` | `TestE2E` | Tests containing "E2E" |

#### Example Configuration

```yaml
# azure.yaml
services:
  gateway:
    language: go
    project: ./src/gateway
    test:
      framework: gotest
      unit:
        pattern: "^Test[^Integration]"
      integration:
        pattern: "TestIntegration"
        setup:
          - docker-compose up -d
        teardown:
          - docker-compose down
      e2e:
        pattern: "TestE2E"
      coverage:
        enabled: true
        threshold: 80
```

### Python Testing

#### Supported Frameworks

- **pytest** (recommended, most popular)
- **unittest** (built-in)
- **nose2**

#### Auto-Detection

The command detects pytest by checking:
1. `pytest.ini`, `pyproject.toml` with `[tool.pytest]`
2. Test file patterns: `test_*.py`, `*_test.py`
3. Test directories: `tests/`, `test/`

#### Default Test Commands

```bash
# Unit tests
pytest tests/unit
# or
pytest -m unit

# Integration tests
pytest tests/integration
# or
pytest -m integration

# E2E tests
pytest tests/e2e
# or
pytest -m e2e

# Coverage
pytest --cov=src --cov-report=xml --cov-report=html
```

#### Coverage Tools

- **pytest-cov** (wrapper for coverage.py)
- **coverage.py** (standard Python coverage tool)

#### Example Configuration

```yaml
# azure.yaml
services:
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit -v
        markers: unit
      integration:
        command: pytest tests/integration -v
        markers: integration
      e2e:
        command: pytest tests/e2e -v
        markers: e2e
      coverage:
        enabled: true
        tool: pytest-cov
        threshold: 85
        source: src/api
        outputFormat: xml,html
```

### .NET Testing

#### Supported Frameworks

- **xUnit** (recommended for modern .NET)
- **NUnit**
- **MSTest**

#### Auto-Detection

The command detects test projects by checking:
1. Project file names: `*.Tests.csproj`, `*.Test.csproj`
2. Package references: `xunit`, `NUnit`, `MSTest`
3. Test directories: `tests/`, `Tests/`

#### Default Test Commands

```bash
# All tests
dotnet test

# Unit tests (with filter)
dotnet test --filter Category=Unit

# Integration tests
dotnet test --filter Category=Integration

# E2E tests
dotnet test --filter Category=E2E

# Coverage (using coverlet)
dotnet test /p:CollectCoverage=true /p:CoverletOutputFormat=cobertura
```

#### Coverage Tools

- **coverlet** (cross-platform, recommended)
- **dotCover** (JetBrains)
- **OpenCover** (legacy)

#### Example Configuration

```yaml
# azure.yaml
services:
  apphost:
    language: csharp
    project: ./src/AppHost
    test:
      framework: xunit
      unit:
        filter: "Category=Unit"
        projects:
          - ./src/AppHost.Tests/AppHost.Tests.csproj
      integration:
        filter: "Category=Integration"
        projects:
          - ./src/AppHost.IntegrationTests/AppHost.IntegrationTests.csproj
      e2e:
        filter: "Category=E2E"
        projects:
          - ./tests/E2E/E2E.Tests.csproj
      coverage:
        enabled: true
        tool: coverlet
        threshold: 80
        outputFormat: cobertura
        exclude:
          - "[*.Tests]*"
          - "[*]*.Migrations.*"
```

## Code Coverage

### Coverage Aggregation

When `--coverage` flag is used, the command:

1. **Collects** coverage from each service
2. **Converts** to a common format (Cobertura XML)
3. **Merges** coverage data across all services
4. **Generates** unified HTML report
5. **Calculates** aggregate metrics

### Coverage Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Run Tests with Coverage for Each Service                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
    ┌───────────────┬───────────────┬───────────────┬───────────────┐
    │               │               │               │               │
  Node.js        Python           Go             .NET
    │               │               │               │
    ↓               ↓               ↓               ↓
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌──────────┐
│  Jest   │   │ pytest- │   │ go test │   │ coverlet │
│ coverage│   │   cov   │   │ -cover  │   │          │
└─────────┘   └─────────┘   └─────────┘   └──────────┘
    │               │               │               │
    ↓               ↓               ↓               ↓
  lcov.info   coverage.xml  coverage.out   coverage.cobertura.xml
    │               │               │               │
    └───────────────┴───────────────┴───────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Convert All to Common Format (Cobertura XML)                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Merge Coverage Reports                                      │
│  - Combine all service coverage                              │
│  - Calculate line coverage                                   │
│  - Calculate branch coverage                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Generate Unified Reports                                    │
│  - HTML report (interactive)                                 │
│  - JSON summary                                              │
│  - Console summary                                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check Coverage Threshold                                    │
│  - Compare against --threshold flag                          │
│  - Fail if below threshold                                   │
└─────────────────────────────────────────────────────────────┘
```

### Coverage Output

#### Console Summary

```
📊 Test Coverage Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Service: web (Node.js)
  Lines:    245/280   (87.5%)  ✓
  Branches: 89/102    (87.3%)  ✓
  Functions: 67/75    (89.3%)  ✓

Service: api (Python)
  Lines:    512/580   (88.3%)  ✓
  Branches: 145/168   (86.3%)  ✓
  Functions: 98/110   (89.1%)  ✓

Service: apphost (.NET)
  Lines:    187/210   (89.0%)  ✓
  Branches: 56/64     (87.5%)  ✓
  Methods:  42/48     (87.5%)  ✓

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Aggregate Coverage
  Lines:    944/1070  (88.2%)  ✓
  Branches: 290/334   (86.8%)  ✓
  Functions: 207/233  (88.8%)  ✓

✓ Coverage threshold met (threshold: 80%)

Reports:
  HTML:  ./test-results/coverage/index.html
  JSON:  ./test-results/coverage/coverage.json
  XML:   ./test-results/coverage/coverage.xml
```

#### HTML Report Structure

```
test-results/
├── coverage/
│   ├── index.html          # Main coverage dashboard
│   ├── web/                # Per-service coverage
│   │   ├── index.html
│   │   └── src/
│   ├── api/
│   │   ├── index.html
│   │   └── src/
│   ├── apphost/
│   │   ├── index.html
│   │   └── src/
│   ├── coverage.json       # Machine-readable summary
│   └── coverage.xml        # Cobertura XML (for CI tools)
└── test-results/
    ├── web-results.xml     # JUnit format
    ├── api-results.xml
    └── apphost-results.xml
```

### Coverage Threshold

```bash
# Fail if coverage below 80%
azd app test --coverage --threshold 80

# Per-service thresholds in azure.yaml
services:
  api:
    test:
      coverage:
        threshold: 90  # Stricter for critical service

  web:
    test:
      coverage:
        threshold: 75  # More lenient for UI
```

## Configuration in azure.yaml

### Complete Example

```yaml
name: fullstack-app

# Global test configuration
test:
  # Global coverage threshold
  coverageThreshold: 80
  
  # Default parallel execution
  parallel: true
  
  # Global output directory
  outputDir: ./test-results
  
  # Fail fast on first error
  failFast: false

services:
  web:
    language: js
    project: ./src/web
    test:
      # Auto-detected if not specified
      framework: jest
      
      # Unit tests
      unit:
        command: pnpm test:unit
        pattern: "src/**/*.test.ts"
        
      # Integration tests
      integration:
        command: pnpm test:integration
        pattern: "src/**/*.integration.test.ts"
        
      # E2E tests
      e2e:
        command: pnpm test:e2e
        pattern: "e2e/**/*.spec.ts"
        
      # Coverage configuration
      coverage:
        enabled: true
        tool: jest
        threshold: 85
        outputFormat: lcov
        exclude:
          - "**/*.d.ts"
          - "**/node_modules/**"
          - "**/__mocks__/**"
  
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      
      unit:
        command: pytest tests/unit -v
        markers: unit
        
      integration:
        command: pytest tests/integration -v
        markers: integration
        setup:
          # Run before integration tests
          - docker-compose up -d postgres
        teardown:
          # Run after integration tests
          - docker-compose down
        
      e2e:
        command: pytest tests/e2e -v
        markers: e2e
        
      coverage:
        enabled: true
        tool: pytest-cov
        threshold: 90
        source: api
        outputFormat: xml,html
        exclude:
          - "*/tests/*"
          - "*/migrations/*"
  
  apphost:
    language: csharp
    project: ./src/AppHost
    test:
      framework: xunit
      
      unit:
        filter: "Category=Unit"
        projects:
          - ./src/AppHost.Tests/AppHost.Tests.csproj
          
      integration:
        filter: "Category=Integration"
        projects:
          - ./tests/Integration/Integration.Tests.csproj
          
      e2e:
        filter: "Category=E2E"
        projects:
          - ./tests/E2E/E2E.Tests.csproj
          
      coverage:
        enabled: true
        tool: coverlet
        threshold: 80
        outputFormat: cobertura
        exclude:
          - "[*.Tests]*"
          - "[*]*.Migrations.*"

  gateway:
    language: go
    project: ./src/gateway
    test:
      framework: gotest
      
      unit:
        pattern: "^Test[^Integration]"
        
      integration:
        pattern: "TestIntegration"
        setup:
          - docker-compose up -d redis
        teardown:
          - docker-compose down
          
      coverage:
        enabled: true
        threshold: 80
```

### Minimal Configuration (Auto-Detection)

If you don't specify test configuration, the command auto-detects:

```yaml
name: simple-app

services:
  web:
    language: js
    project: ./web
    # Auto-detects: Jest, looks for package.json scripts
  
  api:
    language: python
    project: ./api
    # Auto-detects: pytest, looks for pytest.ini or tests/ dir
```

## Auto-Detection Rules

### Node.js

1. **Check package.json scripts** in order:
   - `test:unit`, `test:integration`, `test:e2e`
   - `test` (fallback)

2. **Check for test files**:
   - `*.test.{js,ts,jsx,tsx}`
   - `*.spec.{js,ts,jsx,tsx}`
   - `__tests__/*.{js,ts,jsx,tsx}`

3. **Detect framework**:
   - `jest.config.*` → Jest
   - `vitest.config.*` → Vitest
   - `.mocharc.*` → Mocha
   - `package.json` dependencies

### Python

1. **Check for pytest**:
   - `pytest.ini`
   - `pyproject.toml` with `[tool.pytest]`
   - `tests/` or `test/` directory

2. **Check for unittest**:
   - `test_*.py` files
   - No pytest configuration

3. **Detect test markers/organization**:
   - `tests/unit/`
   - `tests/integration/`
   - `tests/e2e/`
   - Or pytest markers in code

### .NET

1. **Find test projects**:
   - `*.Tests.csproj`
   - `*.Test.csproj`
   - Projects in `tests/` or `Tests/` directory

2. **Detect framework**:
   - Package reference to `xunit`
   - Package reference to `NUnit`
   - Package reference to `MSTest`

3. **Detect test categories**:
   - `[Trait("Category", "Unit")]`
   - `[Category("Integration")]`
   - `[TestCategory("E2E")]`

## Output Formats

### Default (Human-Readable)

```
🧪 Running tests for 3 services...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Service: web (Node.js)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running unit tests...
  ✓ UserService.create (12 ms)
  ✓ UserService.update (8 ms)
  ✓ UserService.delete (5 ms)

  Tests:    45 passed, 45 total
  Time:     2.456 s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Service: api (Python)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running unit tests...
  ✓ test_user_creation (0.034s)
  ✓ test_user_validation (0.028s)
  
  Tests:    67 passed, 67 total
  Time:     3.234 s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Services:  3 tested
Tests:     187 passed, 0 failed, 187 total
Time:      8.923 s

✓ All tests passed!
```

### JSON Output

```json
{
  "success": true,
  "summary": {
    "services": 3,
    "totalTests": 187,
    "passed": 187,
    "failed": 0,
    "skipped": 0,
    "duration": 8.923
  },
  "services": [
    {
      "name": "web",
      "type": "node",
      "framework": "jest",
      "results": {
        "unit": {
          "passed": 45,
          "failed": 0,
          "skipped": 0,
          "duration": 2.456
        }
      },
      "coverage": {
        "lines": 87.5,
        "branches": 87.3,
        "functions": 89.3
      }
    }
  ],
  "coverage": {
    "aggregate": {
      "lines": 88.2,
      "branches": 86.8,
      "functions": 88.8
    },
    "threshold": 80,
    "met": true
  }
}
```

### JUnit XML Output

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="azd-app-tests" tests="187" failures="0" errors="0" time="8.923">
  <testsuite name="web" tests="45" failures="0" errors="0" time="2.456">
    <testcase name="UserService.create" classname="web.unit" time="0.012"/>
    <testcase name="UserService.update" classname="web.unit" time="0.008"/>
    <!-- ... -->
  </testsuite>
  <testsuite name="api" tests="67" failures="0" errors="0" time="3.234">
    <testcase name="test_user_creation" classname="api.unit" time="0.034"/>
    <!-- ... -->
  </testsuite>
</testsuites>
```

### GitHub Actions Format

Automatically sets GitHub Actions outputs and annotations:

```bash
# In GitHub Actions workflow
- name: Run tests
  run: azd app test --coverage --output-format github

# Creates outputs:
# - test-success: true/false
# - coverage: 88.2
# - threshold-met: true/false
# - failed-tests: 0

# Creates annotations for failures:
# ::error file=src/api/user.py,line=45::AssertionError: Expected 200, got 404
```

## Watch Mode

```bash
# Watch mode - re-run tests on file changes
azd app test --watch

# Watch with coverage
azd app test --watch --coverage

# Watch specific service
azd app test --watch --service api
```

Watch mode monitors file changes and automatically re-runs tests:

```
🔍 Watching for file changes...

Changed: src/api/user.py
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Running tests for: api
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  ✓ test_user_creation (0.034s)
  ✓ test_user_validation (0.028s)
  
  Tests:    67 passed, 67 total
  Time:     1.234 s

✓ Tests passed

🔍 Watching for file changes... (press Ctrl+C to exit)
```

## Parallel Execution

By default, tests for different services run in parallel:

```bash
# Parallel (default, faster)
azd app test --parallel

# Sequential (safer, easier to debug)
azd app test --parallel=false
```

Parallel execution:

```
┌─────────────────────────────────────────────────────────────┐
│  Start All Service Tests Concurrently                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┼───────────────────┐
        │                   │                   │
    web (2.5s)          api (3.2s)        apphost (4.1s)
        │                   │                   │
        └───────────────────┼───────────────────┘
                            ↓
                    Wait for all to complete
                            ↓
                    Total time: 4.1s
```

Sequential execution:

```
web (2.5s) → api (3.2s) → apphost (4.1s)
Total time: 9.8s
```

## Error Handling

### Failed Tests

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Service: api (Python)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running unit tests...
  ✓ test_user_creation (0.034s)
  ✗ test_user_validation (0.028s)
  
  FAILED tests/unit/test_user.py::test_user_validation
  
  AssertionError: Expected 200, got 404
  
  File "tests/unit/test_user.py", line 45
    assert response.status_code == 200
  
  Tests:    66 passed, 1 failed, 67 total
  Time:     3.234 s

✗ Tests failed
```

### Coverage Below Threshold

```
📊 Test Coverage Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Aggregate Coverage
  Lines:    720/1070  (67.3%)  ✗
  Branches: 210/334   (62.9%)  ✗

✗ Coverage threshold not met (threshold: 80%, actual: 67.3%)

Low coverage files:
  src/api/payments.py    (45.2%)  ← needs attention
  src/web/checkout.ts    (52.1%)  ← needs attention
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All tests passed, coverage threshold met (if specified) |
| 1 | One or more tests failed |
| 2 | Coverage below threshold |
| 3 | Test execution error (framework not found, invalid config) |
| 4 | Prerequisites not met (test tools not installed) |

## Command Dependency Chain

The `test` command is part of the orchestrated command chain:

```
┌─────────────────────────────────────────────────────────────┐
│                      User runs:                              │
│                   azd app test                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Orchestrator Executes Dependencies:                         │
│                                                              │
│  1. reqs (check prerequisites)                               │
│     └─ Ensure test tools are installed                       │
│                                                              │
│  2. test (this command)                                      │
│     └─ Run tests                                             │
└─────────────────────────────────────────────────────────────┘
```

Note: `test` does NOT automatically run `deps`. Tests should be run on code with dependencies already installed.

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install azd
        run: curl -fsSL https://aka.ms/install-azd.sh | bash
      
      - name: Install azd app extension
        run: |
          azd config set alpha.extension.enabled on
          azd extension install jongio.azd.app
      
      - name: Install dependencies
        run: azd app deps
      
      - name: Run tests with coverage
        run: azd app test --coverage --threshold 80 --output-format github
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./test-results/coverage/coverage.xml
```

### Azure Pipelines Example

```yaml
trigger:
  - main

pool:
  vmImage: 'ubuntu-latest'

steps:
  - script: curl -fsSL https://aka.ms/install-azd.sh | bash
    displayName: 'Install azd'
  
  - script: |
      azd config set alpha.extension.enabled on
      azd extension install jongio.azd.app
    displayName: 'Install azd app extension'
  
  - script: azd app deps
    displayName: 'Install dependencies'
  
  - script: azd app test --coverage --threshold 80 --output-format junit
    displayName: 'Run tests'
  
  - task: PublishTestResults@2
    inputs:
      testResultsFormat: 'JUnit'
      testResultsFiles: '**/test-results/*.xml'
    displayName: 'Publish test results'
  
  - task: PublishCodeCoverageResults@1
    inputs:
      codeCoverageTool: 'Cobertura'
      summaryFileLocation: '**/coverage.xml'
    displayName: 'Publish coverage'
```

## Best Practices

### 1. Organize Tests by Type

```
project/
├── tests/
│   ├── unit/           # Fast, isolated tests
│   ├── integration/    # Tests with dependencies
│   └── e2e/           # Full workflow tests
```

### 2. Use Test Markers/Categories

```python
# Python (pytest)
@pytest.mark.unit
def test_calculate():
    assert calculate(2, 2) == 4

@pytest.mark.integration
def test_database_query():
    result = db.query("SELECT * FROM users")
    assert len(result) > 0
```

```typescript
// Node.js (Jest)
describe('UserService', () => {
  describe('unit', () => {
    test('creates user', () => {
      expect(createUser()).toBeDefined();
    });
  });
});
```

```csharp
// .NET (xUnit)
[Trait("Category", "Unit")]
public void CreateUser_ValidData_ReturnsUser() {
    // Test
}

[Trait("Category", "Integration")]
public void SaveUser_ValidUser_SavesToDatabase() {
    // Test
}
```

### 3. Fast Feedback Loop

```bash
# Development workflow
azd app test --type unit --watch --service api

# Pre-commit
azd app test --type unit

# Pre-push
azd app test --type unit --type integration

# CI pipeline
azd app test --coverage --threshold 80
```

### 4. Coverage Thresholds

```yaml
# Stricter for critical services
services:
  payment-api:
    test:
      coverage:
        threshold: 95  # High-risk code

  admin-ui:
    test:
      coverage:
        threshold: 70  # UI code, harder to test
```

## Security Considerations

### Trust Model

The `test` command executes commands defined in `azure.yaml` with your user permissions. When you run `azd app test`, you implicitly trust:

- **Custom test commands** (defined via `command:` in test config)
- **Setup/teardown scripts** (run before/after tests)
- **Test framework commands** (pytest, jest, go test, dotnet test)

**This follows the same trust model as:**
- npm scripts (package.json)
- Makefile targets
- docker-compose.yml commands
- Azure Developer CLI (azd) hooks

### Security Guidance

1. **Review azure.yaml before running**: Especially in cloned/downloaded projects
2. **Inspect custom test commands**: Check what setup/teardown scripts do
3. **Treat azure.yaml like code**: Test commands can execute arbitrary code
4. **Use version control**: Track changes to test configurations

### What the Test Command Can Execute

- **Custom commands**: Any command specified via `command:` in test config
- **Setup commands**: Pre-test scripts (e.g., `docker-compose up -d`)
- **Teardown commands**: Post-test cleanup scripts
- **Framework CLIs**: pytest, jest, go test, dotnet test with any arguments

### Recommended Practices

- **Audit third-party templates**: Review azure.yaml test config before running
- **Isolate test environments**: Use containers for integration tests
- **Don't run as root/admin**: Use least privilege principle
- **Review setup scripts**: Especially `docker-compose` or database commands

## Troubleshooting

### Issue: Tests not found

**Cause**: Auto-detection failed, no test configuration

**Solution**:
1. Check test file naming (must match patterns)
2. Add explicit configuration in `azure.yaml`
3. Run with `--verbose` to see detection logic

### Issue: Coverage reports not generated

**Cause**: Coverage tool not installed

**Solution**:
```bash
# Node.js
npm install --save-dev jest @types/jest

# Python
pip install pytest pytest-cov

# .NET
dotnet add package coverlet.msbuild
```

### Issue: Parallel execution causing failures

**Cause**: Tests have shared state or resource conflicts

**Solution**:
```bash
# Run sequentially
azd app test --parallel=false

# Or fix tests to be isolated
```

## Related Commands

- [`azd app reqs`](./reqs.md) - Check prerequisites (test tools)
- [`azd app deps`](./deps.md) - Install dependencies (before testing)
- [`azd app run`](./run.md) - Run services (for e2e tests)

## Future Enhancements

### Planned Features

1. **Test Sharding** - Distribute tests across multiple machines
2. **Mutation Testing** - Validate test quality
3. **Visual Regression Testing** - Screenshot comparison
4. **Performance Testing Integration** - Load/stress tests
5. **Test Impact Analysis** - Run only affected tests
6. **Flaky Test Detection** - Identify unstable tests

## Examples

### Example 1: Unit Tests Only

```bash
$ azd app test --type unit

🧪 Running unit tests for 3 services...

✓ web     (45 tests, 2.5s)
✓ api     (67 tests, 3.2s)
✓ apphost (28 tests, 1.8s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Tests:     140 passed, 140 total
Time:      7.5 s

✓ All tests passed!
```

### Example 2: Full Test Suite with Coverage

```bash
$ azd app test --coverage --threshold 80

🧪 Running all tests with coverage for 3 services...

✓ web     (unit: 45, integration: 23, e2e: 12 | 80 total, 12.3s)
✓ api     (unit: 67, integration: 34, e2e: 18 | 119 total, 18.7s)
✓ apphost (unit: 28, integration: 15, e2e: 8 | 51 total, 9.2s)

📊 Coverage: 88.2% (threshold: 80%) ✓

✓ All tests passed, coverage threshold met!

Reports: ./test-results/coverage/index.html
```

### Example 3: Specific Service Integration Tests

```bash
$ azd app test --type integration --service api --verbose

🧪 Running integration tests for service: api

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Service: api (Python)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running setup: docker-compose up -d postgres
✓ Setup complete

Running: pytest tests/integration -v -m integration

tests/integration/test_user_api.py::test_create_user PASSED
tests/integration/test_user_api.py::test_update_user PASSED
tests/integration/test_auth.py::test_login PASSED
...

Tests:    34 passed, 34 total
Time:     5.234 s

Running teardown: docker-compose down
✓ Teardown complete

✓ Integration tests passed
```
