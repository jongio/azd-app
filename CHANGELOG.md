# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2025-11-08

### Added
- **Multi-language testing framework** (`azd app test` command)
  - Automatic framework detection for Node.js (Jest, Vitest, Mocha), Python (pytest, unittest), and .NET (xUnit, NUnit, MSTest)
  - Test type separation (unit, integration, e2e) with filtering support
  - Multi-service code coverage aggregation
  - Coverage threshold enforcement
  - Multiple report formats (JSON, Cobertura XML, HTML)
  - Watch mode for continuous testing during development
  - Setup/teardown command execution support
  - Comprehensive test output parsing for all supported frameworks
  - Parallel and sequential test execution modes
  - Service filtering for targeted testing

### Technical Details
- New package: `cli/src/internal/testing/`
  - `types.go` - Core type definitions for test configuration and results
  - `orchestrator.go` - Test orchestration across multiple services
  - `node_runner.go` - Node.js test execution (Jest, Vitest, Mocha)
  - `python_runner.go` - Python test execution (pytest, unittest)
  - `dotnet_runner.go` - .NET test execution (xUnit, NUnit, MSTest)
  - `coverage.go` - Coverage aggregation and report generation
  - `watcher.go` - File watching for test re-runs
- New command: `cli/src/cmd/app/commands/test.go`
- Comprehensive documentation:
  - `cli/docs/commands/test.md` - Complete command reference
  - `cli/docs/design/testing-framework.md` - Architecture design
  - `cli/docs/design/implementation-plan.md` - Implementation roadmap
  - `cli/docs/schema/test-configuration.md` - YAML configuration schema

### Command Flags
- `--type` - Test type to run (unit/integration/e2e/all)
- `--coverage` - Generate code coverage reports
- `--threshold` - Minimum coverage threshold (0-100)
- `--service` - Run tests for specific service(s)
- `--parallel` - Run tests in parallel (default: true)
- `--watch` - Watch mode for continuous testing
- `--fail-fast` - Stop on first test failure
- `--verbose` - Enable verbose test output
- `--dry-run` - Show configuration without running tests
- `--output-format` - Output format (default/json/junit/github)
- `--output-dir` - Directory for test reports

### Examples
```bash
# Run all tests
azd app test

# Run with coverage and threshold
azd app test --coverage --threshold 80

# Run specific test type
azd app test --type unit

# Run in watch mode
azd app test --watch --type unit

# Run for specific service
azd app test --service api --coverage
```

## [0.5.0] - Previous Release

### Added
- Live dashboard with service monitoring
- Real-time log streaming
- Azure environment integration
- Python entry point auto-detection

## [0.4.0] - Previous Release

### Added
- Service orchestration from azure.yaml
- Multi-language dependency installation
- Prerequisite checking with caching

---

For more details, see the [full documentation](./cli/docs/).
