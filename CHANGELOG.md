# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **`azd app init` Command** — Project initialization and azure.yaml generation
  - Scans project structure to detect services (Node.js, Python, .NET, Go, Azure Functions)
  - Detects frameworks (Express, FastAPI, Vite, Next.js, Gin, Django, Flask, etc.)
  - Auto-detects infrastructure dependencies (`uses` field) by scanning package files for database/cache/queue client libraries
  - Generates complete azure.yaml with ports, commands, and startup ordering
  - Supports `--dry-run` mode to preview without writing files
  - Supports `--force` to overwrite existing services section
  - Service deduplication (Functions wins over language-based detection for same directory)
  - Detected dependencies: PostgreSQL, MySQL, MongoDB, Cosmos DB, Redis, RabbitMQ, Kafka, Azure Service Bus, Azure Storage

- **Custom URL Configuration** (`url` property in azure.yaml)
  - Configure custom access URLs for services (custom domains, reverse proxies, CDN, tunnels, API gateways)
  - Dashboard displays custom URLs with purple visual indicator and "Custom URL" badge
  - Console output shows both "Deployment URL" and "Access URL" when configured
  - Click service links in dashboard navigates to custom URL instead of deployment URL
  - Tooltip shows default URL when custom URL is configured
  - JSON schema validation for HTTP/HTTPS URLs
  - Example project: `cli/tests/projects/url-demo/`
  - Comprehensive CORS configuration guide: `docs/guides/cors-with-alternate-urls.md`
  - Multi-language CORS examples (Node.js, Python, .NET, ASP.NET Core)

- **Trust prompt defaults to Yes** with `-y` shorthand for non-interactive confirmation
- **azd-app-onboard Copilot skill** with Vally evals and CI integration

### Fixed
- Dashboard streaming RPC disconnects caused by WriteTimeout
- Auto-resolve port conflicts with `--force` and in non-interactive mode
- Security remediation — 29 findings across 7 workstreams

### Changed
- Updated dependencies

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

## [0.5.0] - 2025-09-15

### Added
- Live dashboard with service monitoring
- Real-time log streaming
- Azure environment integration
- Python entry point auto-detection

## [0.4.0] - 2025-07-20

### Added
- Service orchestration from azure.yaml
- Multi-language dependency installation
- Prerequisite checking with caching

---

For more details, see the [full documentation](./cli/docs/).
