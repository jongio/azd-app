# Azure.yaml Test Configuration Schema

This document describes the test configuration schema for `azure.yaml` to support the `azd app test` command.

## Overview

The test configuration allows you to specify how tests should be run for each service in your application. It supports auto-detection but also allows explicit configuration for full control.

## Schema Structure

### Global Test Configuration

```yaml
name: my-app

# Global test configuration (optional)
test:
  # Default coverage threshold for all services
  coverageThreshold: 80
  
  # Run tests in parallel by default
  parallel: true
  
  # Output directory for test results and coverage
  outputDir: ./test-results
  
  # Stop on first failure
  failFast: false
  
  # Verbose test output
  verbose: false

services:
  # ... service definitions
```

### Service Test Configuration

```yaml
services:
  service-name:
    language: js|python|csharp|dotnet
    project: ./path/to/service
    
    # Test configuration for this service
    test:
      # Test framework (optional, auto-detected if not specified)
      framework: jest|vitest|mocha|pytest|unittest|xunit|nunit|mstest
      
      # Unit tests configuration
      unit:
        # Command to run unit tests (optional)
        command: npm run test:unit
        
        # Test file pattern (optional)
        pattern: "**/*.test.js"
        
        # Test markers/tags (pytest)
        markers:
          - unit
        
        # Test filter (dotnet)
        filter: "Category=Unit"
        
        # Test projects (dotnet)
        projects:
          - ./tests/Unit.Tests/Unit.Tests.csproj
        
        # Setup commands (run before tests)
        setup:
          - docker-compose up -d postgres
        
        # Teardown commands (run after tests)
        teardown:
          - docker-compose down
      
      # Integration tests configuration
      integration:
        command: npm run test:integration
        pattern: "**/*.integration.test.js"
        markers:
          - integration
        filter: "Category=Integration"
        projects:
          - ./tests/Integration.Tests/Integration.Tests.csproj
        setup:
          - docker-compose up -d
        teardown:
          - docker-compose down
      
      # E2E tests configuration
      e2e:
        command: npm run test:e2e
        pattern: "**/*.e2e.test.js"
        markers:
          - e2e
        filter: "Category=E2E"
        projects:
          - ./tests/E2E.Tests/E2E.Tests.csproj
        setup:
          - azd app run --detach
        teardown:
          - pkill -f "azd app run"
      
      # Coverage configuration
      coverage:
        # Enable coverage collection
        enabled: true
        
        # Coverage tool (optional, auto-detected)
        tool: jest|c8|nyc|pytest-cov|coverage|coverlet
        
        # Minimum coverage threshold for this service
        threshold: 85
        
        # Coverage source directory (python)
        source: src/api
        
        # Coverage output format
        outputFormat: lcov|cobertura|xml|html|json
        
        # Files/patterns to exclude from coverage
        exclude:
          - "**/*.d.ts"
          - "**/node_modules/**"
          - "**/__mocks__/**"
          - "[*.Tests]*"
          - "[*]*.Migrations.*"
```

## Complete Examples

### Example 1: Node.js with Jest (Minimal)

```yaml
name: node-app

services:
  web:
    language: js
    project: ./src/web
    # Auto-detects Jest and runs npm test
```

### Example 2: Node.js with Jest (Explicit)

```yaml
name: node-app

services:
  web:
    language: js
    project: ./src/web
    test:
      framework: jest
      unit:
        command: pnpm test:unit
        pattern: "src/**/*.test.ts"
      integration:
        command: pnpm test:integration
        pattern: "src/**/*.integration.test.ts"
      e2e:
        command: pnpm test:e2e
        pattern: "e2e/**/*.spec.ts"
      coverage:
        enabled: true
        threshold: 85
        exclude:
          - "**/*.d.ts"
          - "**/node_modules/**"
```

### Example 3: Python with pytest

```yaml
name: python-app

services:
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit -v
        markers:
          - unit
      integration:
        command: pytest tests/integration -v
        markers:
          - integration
        setup:
          - docker-compose up -d postgres redis
        teardown:
          - docker-compose down
      e2e:
        command: pytest tests/e2e -v
        markers:
          - e2e
        setup:
          - azd app run --detach --service api
        teardown:
          - pkill -f "uvicorn"
      coverage:
        enabled: true
        tool: pytest-cov
        threshold: 90
        source: api
        outputFormat: xml,html
        exclude:
          - "*/tests/*"
          - "*/migrations/*"
```

### Example 4: .NET with xUnit

```yaml
name: dotnet-app

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
          - ./tests/Integration/Integration.Tests.csproj
        setup:
          - docker-compose up -d sqlserver
        teardown:
          - docker-compose down
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

### Example 5: Full Stack Application

```yaml
name: fullstack-app

# Global test settings
test:
  coverageThreshold: 80
  parallel: true
  outputDir: ./test-results
  verbose: false

services:
  # Frontend (React + Vitest)
  web:
    language: js
    project: ./src/web
    test:
      framework: vitest
      unit:
        command: pnpm test:unit
      integration:
        command: pnpm test:integration
      e2e:
        command: pnpm test:e2e
        setup:
          - pnpm preview --detach
      coverage:
        enabled: true
        threshold: 85
        exclude:
          - "**/*.d.ts"
          - "**/node_modules/**"
  
  # Backend API (Python + FastAPI + pytest)
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        markers:
          - unit
      integration:
        markers:
          - integration
        setup:
          - docker-compose up -d postgres redis
        teardown:
          - docker-compose down
      e2e:
        markers:
          - e2e
      coverage:
        enabled: true
        threshold: 90
        source: api
        outputFormat: xml,html
  
  # .NET Aspire AppHost
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
      coverage:
        enabled: true
        threshold: 80
```

## Field Descriptions

### Global Test Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `test.coverageThreshold` | number | 0 | Minimum coverage percentage (0-100) for all services |
| `test.parallel` | boolean | true | Run tests for multiple services in parallel |
| `test.outputDir` | string | `./test-results` | Directory for test results and coverage reports |
| `test.failFast` | boolean | false | Stop execution on first test failure |
| `test.verbose` | boolean | false | Enable verbose test output |

### Service Test Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `test.framework` | string | No | Test framework name (auto-detected if omitted) |
| `test.unit` | object | No | Unit test configuration |
| `test.integration` | object | No | Integration test configuration |
| `test.e2e` | object | No | End-to-end test configuration |
| `test.coverage` | object | No | Coverage configuration |

### Test Type Configuration Fields

#### Common Fields (all languages)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | No | Command to run tests (auto-generated if omitted) |
| `setup` | array | No | Commands to run before tests |
| `teardown` | array | No | Commands to run after tests |

#### Node.js Specific Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | No | Test file pattern (e.g., `**/*.test.js`) |

#### Python Specific Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `markers` | array | No | Pytest markers to filter tests (e.g., `[unit, slow]`) |

#### .NET Specific Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `filter` | string | No | Test filter expression (e.g., `Category=Unit`) |
| `projects` | array | No | List of test project paths |

### Coverage Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | false | Enable coverage collection |
| `tool` | string | auto | Coverage tool name (auto-detected if omitted) |
| `threshold` | number | 0 | Minimum coverage percentage for this service |
| `source` | string | - | Source directory to measure coverage (Python) |
| `outputFormat` | string | auto | Output format: `lcov`, `cobertura`, `xml`, `html`, `json` |
| `exclude` | array | [] | Files/patterns to exclude from coverage |

## Auto-Detection Behavior

If test configuration is omitted or incomplete, `azd app test` auto-detects:

### Node.js Auto-Detection

1. **Framework**: Checks for `jest.config.*`, `vitest.config.*`, package.json dependencies
2. **Unit tests**: Runs `npm test` or checks for `test:unit` script
3. **Integration tests**: Checks for `test:integration` script
4. **E2E tests**: Checks for `test:e2e` script
5. **Coverage**: Uses framework's built-in coverage (Jest/Vitest)

### Python Auto-Detection

1. **Framework**: Checks for `pytest.ini`, `pyproject.toml` with pytest config
2. **Unit tests**: Runs `pytest tests/unit` or `pytest -m unit`
3. **Integration tests**: Runs `pytest tests/integration` or `pytest -m integration`
4. **E2E tests**: Runs `pytest tests/e2e` or `pytest -m e2e`
5. **Coverage**: Uses `pytest-cov` or `coverage.py`

### .NET Auto-Detection

1. **Framework**: Checks package references in `*.csproj` files
2. **Unit tests**: Finds `*.Tests.csproj` files, runs `dotnet test --filter Category=Unit`
3. **Integration tests**: Runs `dotnet test --filter Category=Integration`
4. **E2E tests**: Runs `dotnet test --filter Category=E2E`
5. **Coverage**: Uses `coverlet` with Cobertura format

## Validation Rules

The following validation rules are applied to the configuration:

1. **Coverage Threshold**:
   - Must be between 0 and 100
   - Service threshold overrides global threshold

2. **Framework**:
   - Must be a supported framework for the language
   - Node.js: `jest`, `vitest`, `mocha`, `ava`, `tap`
   - Python: `pytest`, `unittest`, `nose2`
   - .NET: `xunit`, `nunit`, `mstest`

3. **Test Type**:
   - At least one test type (unit/integration/e2e) should be configured or detectable
   - If explicit command is provided, it must be a valid shell command

4. **Setup/Teardown**:
   - Commands must be valid shell commands
   - Executed in the service project directory
   - Teardown runs even if tests fail

5. **Projects** (.NET only):
   - Paths must be relative to service directory
   - Files must exist and be valid `.csproj` or `.fsproj` files

## Migration Guide

### From Manual Testing to azd app test

#### Before (manual)

```bash
# Terminal 1 - Start database
cd src/api
docker-compose up postgres

# Terminal 2 - Run unit tests
cd src/api
pytest tests/unit

# Terminal 3 - Run integration tests
cd src/api
pytest tests/integration

# Terminal 4 - Run web tests
cd src/web
npm test
```

#### After (automated)

```yaml
# azure.yaml
services:
  api:
    language: python
    project: ./src/api
    test:
      integration:
        setup:
          - docker-compose up -d postgres
        teardown:
          - docker-compose down
  
  web:
    language: js
    project: ./src/web
```

```bash
# One command for everything
azd app test
```

### From package.json scripts

#### Before

```json
{
  "scripts": {
    "test": "jest",
    "test:unit": "jest --testPathPattern=unit",
    "test:integration": "jest --testPathPattern=integration",
    "test:e2e": "playwright test"
  }
}
```

#### After

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
      integration:
        command: npm run test:integration
      e2e:
        command: npm run test:e2e
```

Now you can run specific test types:
```bash
azd app test --type unit
azd app test --type integration
azd app test --type e2e
```

## Best Practices

### 1. Use Setup/Teardown for Dependencies

```yaml
services:
  api:
    test:
      integration:
        setup:
          - docker-compose up -d postgres redis
        teardown:
          - docker-compose down
```

### 2. Set Appropriate Coverage Thresholds

```yaml
# Global default
test:
  coverageThreshold: 80

services:
  # Critical service - higher threshold
  payment-api:
    test:
      coverage:
        threshold: 95
  
  # UI - more lenient
  admin-ui:
    test:
      coverage:
        threshold: 70
```

### 3. Organize Tests by Type

```
project/
├── tests/
│   ├── unit/          # Fast, isolated tests
│   ├── integration/   # Tests with dependencies
│   └── e2e/          # Full workflow tests
```

### 4. Use Markers/Categories

```python
# Python
@pytest.mark.unit
@pytest.mark.slow
def test_complex_calculation():
    pass

@pytest.mark.integration
@pytest.mark.database
def test_database_query():
    pass
```

```csharp
// .NET
[Trait("Category", "Unit")]
[Trait("Category", "Fast")]
public void Test() { }

[Trait("Category", "Integration")]
[Trait("Category", "Database")]
public void TestDatabase() { }
```

### 5. Exclude Generated Code from Coverage

```yaml
services:
  api:
    test:
      coverage:
        exclude:
          - "*/migrations/*"
          - "*/generated/*"
          - "*/__pycache__/*"
```

## Troubleshooting

### Configuration Not Found

**Issue**: `azd app test` says "no test configuration found"

**Solution**: Add explicit configuration or ensure auto-detection files exist:
- Node.js: `package.json` with test script
- Python: `pytest.ini` or `tests/` directory
- .NET: `*.Tests.csproj` files

### Tests Not Running

**Issue**: Tests are not executed

**Solution**: Check command is valid:
```yaml
test:
  unit:
    command: npm run test:unit  # Make sure this script exists in package.json
```

### Coverage Not Generated

**Issue**: No coverage report generated

**Solution**: Ensure coverage tool is installed:
```yaml
# Node.js
devDependencies:
  jest: "^29.0.0"

# Python
dependencies:
  pytest-cov: "^4.0.0"

# .NET
<PackageReference Include="coverlet.msbuild" Version="6.0.0" />
```

## See Also

- [Test Command Documentation](../commands/test.md)
- [Testing Framework Design](../design/testing-framework.md)
- [CLI Reference](../cli-reference.md)
