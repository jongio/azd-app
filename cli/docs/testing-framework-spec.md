# azd app test - Testing Framework Specification

## Quick Links

- **Command Reference**: [commands/test.md](commands/test.md) - Complete usage guide
- **Architecture**: [design/testing-framework.md](design/testing-framework.md) - Technical design
- **Implementation**: [design/implementation-plan.md](design/implementation-plan.md) - Roadmap
- **Configuration**: [schema/test-configuration.md](schema/test-configuration.md) - YAML schema
- **Design Overview**: [design/README.md](design/README.md) - Design docs index

## Overview

The `azd app test` command provides comprehensive testing capabilities for multi-language applications, with support for:

- **Multi-Language**: Node.js, Python, .NET
- **Test Types**: Unit, Integration, E2E (run independently or together)
- **Auto-Detection**: Smart framework detection with explicit configuration override
- **Coverage**: Aggregated code coverage across all services
- **CI/CD**: Multiple output formats (JSON, JUnit, GitHub Actions)
- **Parallel**: Fast test execution across services

## Basic Usage

```bash
# Run all tests
azd app test

# Run with coverage
azd app test --coverage --threshold 80

# Run specific test type
azd app test --type unit

# Run for specific service
azd app test --service api

# Watch mode (development)
azd app test --watch --type unit
```

## Example Configuration

```yaml
# azure.yaml
name: fullstack-app

test:
  coverageThreshold: 80
  parallel: true

services:
  # Node.js with Jest
  web:
    language: js
    project: ./src/web
    test:
      framework: jest
      unit:
        command: pnpm test:unit
      coverage:
        threshold: 85
  
  # Python with pytest
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      integration:
        markers: [integration]
        setup:
          - docker-compose up -d postgres
        teardown:
          - docker-compose down
      coverage:
        threshold: 90
  
  # .NET with xUnit
  apphost:
    language: csharp
    project: ./src/AppHost
    test:
      framework: xunit
      unit:
        filter: "Category=Unit"
      coverage:
        threshold: 80
```

## Supported Frameworks

| Language | Frameworks | Coverage Tools |
|----------|------------|----------------|
| **Node.js** | Jest, Vitest, Mocha, AVA, Tap | Jest (built-in), c8, nyc |
| **Python** | pytest, unittest, nose2 | pytest-cov, coverage.py |
| **.NET** | xUnit, NUnit, MSTest | coverlet, dotCover |

## Key Features

### 1. Test Type Separation

Run different test types independently for fast feedback:

```bash
# Fast unit tests during development
azd app test --type unit --watch

# Integration tests before commit
azd app test --type integration

# Full E2E suite in CI
azd app test --type e2e
```

### 2. Auto-Detection

Minimal configuration required - automatically detects:
- Test frameworks (Jest, pytest, xUnit, etc.)
- Test commands and patterns
- Coverage tools
- Test organization (unit/integration/e2e)

### 3. Code Coverage Aggregation

Unified coverage across all services:
- Collects coverage from each service
- Converts to common format (Cobertura XML)
- Merges coverage data
- Generates unified HTML reports
- Validates thresholds

Example output:
```
📊 Test Coverage Summary

Service: web (Node.js)
  Lines:    245/280   (87.5%)  ✓

Service: api (Python)  
  Lines:    512/580   (88.3%)  ✓

Aggregate Coverage
  Lines:    944/1070  (88.2%)  ✓

✓ Coverage threshold met (threshold: 80%)
```

### 4. CI/CD Integration

Multiple output formats for different CI systems:

```bash
# GitHub Actions
azd app test --output-format github

# JUnit XML for most CI systems
azd app test --output-format junit

# JSON for custom processing
azd app test --output-format json
```

### 5. Parallel Execution

Tests for different services run in parallel by default:

```
web (2.5s) ┐
api (3.2s) ├─ parallel → Total: 4.1s
app (4.1s) ┘

vs. sequential: 9.8s
```

## Architecture

```
┌─────────────────────────────────────┐
│         azd app test                │
│      (Command Entry Point)          │
└─────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│       Test Orchestrator             │
│ - Parse azure.yaml                  │
│ - Detect frameworks                 │
│ - Manage execution                  │
│ - Aggregate results                 │
└─────────────────────────────────────┘
                 ↓
    ┌────────────┼────────────┐
    ↓            ↓            ↓
┌────────┐  ┌─────────┐  ┌─────────┐
│ Node.js│  │ Python  │  │  .NET   │
│ Runner │  │ Runner  │  │ Runner  │
└────────┘  └─────────┘  └─────────┘
    │            │            │
    └────────────┼────────────┘
                 ↓
┌─────────────────────────────────────┐
│      Coverage Aggregator            │
│ - Collect from all services         │
│ - Convert formats                   │
│ - Merge coverage data               │
│ - Generate reports                  │
└─────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────┐
│       Report Generator              │
│ - Console output                    │
│ - JSON/JUnit/GitHub formats         │
│ - HTML coverage reports             │
└─────────────────────────────────────┘
```

## Implementation Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1**: Core Infrastructure | 1-2 weeks | Types, orchestrator, basic command |
| **Phase 2**: Language Runners | 2-3 weeks | Node.js, Python, .NET runners |
| **Phase 3**: Coverage | 1-2 weeks | Aggregation, reporting |
| **Phase 4**: Advanced Features | 1-2 weeks | Watch mode, setup/teardown |
| **Phase 5**: Testing & Docs | 1 week | Tests, final documentation |
| **Total** | **6-10 weeks** | Complete testing framework |

## Documentation Structure

```
docs/
├── commands/
│   └── test.md                     # Complete command reference (39KB)
│       ├── Usage and flags
│       ├── Language-specific support
│       ├── Coverage aggregation
│       ├── CI/CD integration
│       └── Examples and troubleshooting
│
├── design/
│   ├── README.md                   # Design docs overview (5KB)
│   ├── testing-framework.md        # Architecture (31KB)
│   │   ├── Component details
│   │   ├── Data structures
│   │   ├── Auto-detection logic
│   │   ├── Coverage conversion
│   │   └── Security & performance
│   └── implementation-plan.md      # Roadmap (16KB)
│       ├── 5 implementation phases
│       ├── Acceptance criteria
│       ├── Testing strategy
│       └── Timeline and risks
│
└── schema/
    └── test-configuration.md       # YAML schema (16KB)
        ├── Complete field reference
        ├── Configuration examples
        ├── Validation rules
        └── Migration guide
```

## Quick Start Examples

### Node.js Project (Minimal)

```yaml
# azure.yaml
services:
  web:
    language: js
    project: ./web
    # Auto-detects Jest from package.json
```

```bash
azd app test --coverage
```

### Python Project (Explicit)

```yaml
# azure.yaml
services:
  api:
    language: python
    project: ./api
    test:
      framework: pytest
      unit:
        markers: [unit]
      integration:
        markers: [integration]
        setup:
          - docker-compose up -d
        teardown:
          - docker-compose down
```

```bash
# Run only integration tests
azd app test --type integration --service api
```

### Full Stack Application

```yaml
# azure.yaml
name: my-app

test:
  coverageThreshold: 80
  parallel: true

services:
  frontend:
    language: js
    project: ./frontend
    test:
      framework: vitest
      
  backend:
    language: python
    project: ./backend
    test:
      framework: pytest
      
  api-gateway:
    language: csharp
    project: ./gateway
    test:
      framework: xunit
```

```bash
# Run all tests with coverage
azd app test --coverage

# Output:
# ✓ frontend (45 tests, 87.5% coverage)
# ✓ backend (67 tests, 88.3% coverage)
# ✓ api-gateway (28 tests, 89.0% coverage)
# 
# Aggregate: 140 tests, 88.2% coverage ✓
```

## CI/CD Examples

### GitHub Actions

```yaml
- name: Run tests with coverage
  run: azd app test --coverage --threshold 80 --output-format github

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./test-results/coverage/coverage.xml
```

### Azure Pipelines

```yaml
- script: azd app test --coverage --threshold 80 --output-format junit
  displayName: 'Run tests'

- task: PublishTestResults@2
  inputs:
    testResultsFormat: 'JUnit'
    testResultsFiles: '**/test-results/*.xml'
```

## Benefits

### For Developers

- ✅ **Fast Feedback**: Unit tests in < 5 seconds with watch mode
- ✅ **Clear Output**: Formatted, colored console output
- ✅ **Easy Debugging**: Verbose mode for detailed test info
- ✅ **Flexible**: Run specific tests or services

### For Teams

- ✅ **Consistent**: Same command across all services
- ✅ **Multi-Language**: Support for polyglot projects
- ✅ **Comprehensive**: Unit, integration, E2E in one tool
- ✅ **Coverage**: Unified coverage across entire app

### For CI/CD

- ✅ **Fast**: Parallel execution reduces build time
- ✅ **Reliable**: Threshold enforcement prevents regressions
- ✅ **Flexible**: Multiple output formats
- ✅ **Integrated**: Works with all major CI systems

## Design Principles

1. **Auto-Detection First**: Minimize configuration required
2. **Explicit Override**: Allow full control when needed
3. **Multi-Language Native**: First-class support for all languages
4. **CI/CD Ready**: Easy pipeline integration
5. **Fast Feedback**: Optimize for developer workflow
6. **Consistent UX**: Follow existing azd app patterns

## Next Steps

### For Users (When Implemented)

1. Update to latest version: `azd extension update jongio.azd.app`
2. Run tests: `azd app test`
3. Add coverage: `azd app test --coverage`
4. Configure in `azure.yaml` (optional)

### For Contributors

1. Review design documentation
2. Provide feedback on architecture
3. Contribute to implementation
4. Help with testing

## Related Commands

- **`azd app reqs`** - Check test tools are installed
- **`azd app deps`** - Install dependencies before testing
- **`azd app run`** - Run services for E2E tests

## Status

**Status**: 📝 Specification Complete, Implementation Planned

**Version**: 0.6.0 (planned)

**Documentation**: Complete (6 documents, ~120KB total)

## Feedback

Have feedback on this design? 

- Open an issue: [GitHub Issues](https://github.com/jongio/azd-app/issues)
- Start a discussion: [GitHub Discussions](https://github.com/jongio/azd-app/discussions)
- Review the PR: [Pull Request](https://github.com/jongio/azd-app/pulls)

---

**Last Updated**: 2025-11-08  
**Authors**: GitHub Copilot, jongio  
**License**: MIT
