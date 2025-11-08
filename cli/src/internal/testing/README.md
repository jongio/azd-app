# Testing Package

This package provides test execution and coverage aggregation for multi-language projects.

## Implementation Status

### Phase 1: Core Infrastructure (COMPLETE ✅)

**Completed:**
- ✅ Type definitions (`types.go`)
  - `TestConfig` - Global test configuration
  - `ServiceTestConfig` - Per-service test configuration
  - `TestResult` - Test execution results
  - `CoverageData` - Coverage metrics
  - `AggregateResult` - Combined results
  - `AggregateCoverage` - Aggregated coverage

- ✅ Test command (`commands/test.go`)
  - All flags defined and working
  - Input validation (test type, threshold, output format)
  - Integration with command orchestrator
  - Dry-run mode
  - Full test execution workflow

- ✅ Test orchestrator (`orchestrator.go`)
  - Loads services from azure.yaml
  - Framework auto-detection (Node.js, Python, .NET)
  - Test execution management
  - Result aggregation
  - Service filtering

- ✅ Node.js test runner (`node_runner.go`)
  - Framework detection (Jest, Vitest, Mocha)
  - Test command generation
  - Test execution
  - Output parsing
  - Result extraction

- ✅ Unit tests
  - Type definitions tested
  - Command structure tested
  - Validation logic tested

### Phase 2: Language Runners (COMPLETE ✅)

**Completed:**
- ✅ Node.js test runner with Jest/Vitest/Mocha support
- ✅ Python test runner with pytest/unittest support
- ✅ Framework auto-detection for both languages
- ✅ Test output parsing for all frameworks

**Remaining:**
- ⏳ .NET test runner (xUnit/NUnit/MSTest)

### Phases 3-5: Not Yet Started

- ⏳ Coverage aggregation and format conversion
- ⏳ Advanced features (watch mode, setup/teardown)
- ⏳ Final documentation and integration tests

See [implementation plan](../../docs/design/implementation-plan.md) for details.

## Current Functionality

The test command is functional for Node.js and Python projects. You can:

```bash
# View help and all available flags
azd app test --help

# Run tests for all services (requires azure.yaml)
azd app test

# Run specific test type
azd app test --type unit

# Run tests for specific service
azd app test --service web

# Dry-run to see what would be tested
azd app test --dry-run --type unit --coverage --threshold 80

# Validate parameters
azd app test --type unit        # ✓ Valid
azd app test --type invalid     # ✗ Error: invalid test type
azd app test --threshold 150    # ✗ Error: threshold must be 0-100
```

## Usage Example

Create an `azure.yaml` with your services:

```yaml
name: my-app
reqs:
  - id: node
    minVersion: "18.0.0"
  - id: python
    minVersion: "3.9.0"
services:
  web:
    language: js
    project: ./web
  api:
    language: python
    project: ./api
```

Ensure your projects have test scripts:

**Node.js** (`package.json`):
```json
{
  "scripts": {
    "test": "jest"
  }
}
```

**Python** (tests directory with pytest):
```
api/
  tests/
    test_api.py
  pyproject.toml
```

Then run:

```bash
azd app test
```

## Architecture

```
TestOrchestrator ✅
     ↓
  ┌──┴──┬──────┬────────┐
  │     │      │        │
Node  Python  .NET   Coverage
Runner Runner Runner  Aggregator
  ✅     ✅      ⏳       ⏳
```

## Framework Detection

### Node.js ✅
- Checks for `jest.config.*`, `vitest.config.*`, `.mocharc.*`
- Falls back to checking `package.json` dependencies
- Defaults to `npm test`

### Python ✅
- Checks for `pytest.ini`, `pyproject.toml`, `setup.cfg`
- Detects package manager (uv, poetry, pip)
- Supports pytest markers for test type filtering
- Falls back to unittest if pytest not detected

### .NET ⏳
- Will scan for `*.Tests.csproj` files
- Check package references for xUnit/NUnit/MSTest

## Contributing

When adding functionality:
1. Update type definitions as needed
2. Add unit tests for all new functions
3. Update this README with implementation status
4. Follow existing code patterns

See [implementation plan](../../docs/design/implementation-plan.md) for the complete roadmap.
