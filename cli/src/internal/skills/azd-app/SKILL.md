---
name: azd-app
description: |
  Local multi-service development orchestrator for Azure Developer CLI. Runs, tests, monitors,
  and manages all project services with one command. USE FOR: run local services, install
  dependencies, run tests, check health, view logs, start stop restart services, MCP server,
  manage service lifecycle, multi-language development. DO NOT USE FOR: Azure deployments
  (use azd deploy), infrastructure provisioning (use azd provision), CI/CD pipelines.
---

# azd-app Extension

azd-app is an Azure Developer CLI (`azd`) extension that automatically detects, configures,
and orchestrates multi-service development environments. It supports Node.js, Python, .NET,
Java, Go, Rust, PHP, and Docker services in a single project.

## When to Use

- Running all project services locally with a single command
- Installing dependencies across multiple languages
- Running tests with unified coverage reporting
- Checking service health endpoints
- Viewing aggregated logs from all services
- Managing service lifecycle (start/stop/restart)
- Exposing project capabilities via MCP server

## Commands Reference

### `azd app run`
Run all detected services concurrently with dependency installation and port management.

| Flag | Description |
|------|-------------|
| `--service, -s` | Run only the specified service |
| `--web` | Open browser to web service after startup |
| `--env-file` | Load environment variables from a file |
| `--verbose` | Show detailed output |
| `--dry-run` | Show what would run without executing |
| `--restart-containers` | Force restart Docker containers |
| `--force` | Skip confirmation prompts |

### `azd app deps`
Install dependencies for all services (npm install, pip install, dotnet restore, etc.).

| Flag | Description |
|------|-------------|
| `--service, -s` | Install deps for a specific service only |
| `--verbose` | Show detailed output |

### `azd app reqs`
Check system requirements (runtimes, tools) for all detected services.

| Flag | Description |
|------|-------------|
| `--fix` | Attempt to install missing requirements |

### `azd app test`
Run tests for all services with unified coverage aggregation.

| Flag | Description |
|------|-------------|
| `--service, -s` | Test only the specified service |
| `--coverage` | Enable coverage collection |
| `--threshold` | Minimum coverage percentage to pass |
| `--watch` | Watch mode for continuous testing |
| `--fail-fast` | Stop on first failure |
| `--parallel` | Run service tests in parallel |
| `--update-snapshots` | Update test snapshots |
| `--verbose` | Show detailed test output |
| `--dry-run` | Show what would run without executing |
| `--timeout` | Maximum time for test execution |
| `--stream / --no-stream` | Stream test output in real time |

### `azd app health`
Check health endpoints for running services.

| Flag | Description |
|------|-------------|
| `--service, -s` | Check health of a specific service |
| `--stream` | Continuously monitor health |
| `--interval` | Interval between health checks (default: 5s) |
| `--timeout` | HTTP timeout per check (default: 5s) |
| `--endpoint` | Custom health endpoint path (default: /health) |
| `--all` | Check all services, not just running ones |
| `--verbose` | Show detailed health info |

### `azd app info`
Show project information: detected services, languages, ports, and configuration.

### `azd app logs`
View aggregated logs from all running services.

| Flag | Description |
|------|-------------|
| `--service, -s` | Show logs for a specific service |
| `--follow, -f` | Follow log output |
| `--tail` | Number of recent lines to show |

### `azd app start`
Start a previously stopped service.

| Flag | Description |
|------|-------------|
| `--service, -s` | Service to start |

### `azd app stop`
Stop a running service.

| Flag | Description |
|------|-------------|
| `--service, -s` | Service to stop |

### `azd app restart`
Restart a running service.

| Flag | Description |
|------|-------------|
| `--service, -s` | Service to restart |

### `azd app add`
Add a new service to the project configuration.

### `azd app mcp serve`
Start the MCP (Model Context Protocol) server exposing project tools for AI assistants.

### `azd app version`
Print the azd-app extension version.

## Environment Variables

azd-app supports three formats for environment variable values in `azure.yaml`:

1. **Literal values**: `MY_VAR: "hello"`
2. **Environment variable references**: `MY_VAR: ${OTHER_VAR}`
3. **Key Vault references**: `MY_VAR: keyvault://my-vault/my-secret`

Environment variables are resolved at service startup and injected into each service process.

## Hooks

azd-app supports lifecycle hooks defined in `azure.yaml`:

- **prerun** — runs before the service starts
- **postrun** — runs after the service stops

Hooks can be defined per-service or at the project level.

## Supported Languages

| Language | Dependency Install | Test Runner | Runtime |
|----------|-------------------|-------------|---------|
| Node.js | npm/yarn/pnpm install | Jest, Vitest, Mocha | node |
| Python | pip install / venv | pytest, unittest | python |
| .NET | dotnet restore | xUnit, NUnit, MSTest | dotnet run |
| Java | maven/gradle | JUnit, TestNG | mvn/gradle |
| Go | go mod download | go test | go run |
| Rust | cargo build | cargo test | cargo run |
| PHP | composer install | PHPUnit | php |
| Docker | docker build | — | docker compose |

## MCP Tools

When running `azd app mcp serve`, the following tools are available:

| Tool | Description |
|------|-------------|
| `get_services` | List all detected services with status |
| `get_service_logs` | Retrieve logs for a specific service |
| `get_service_errors` | Get error-level logs for a service |
| `get_project_info` | Get project metadata and configuration |
| `run_services` | Start all or specific services |
| `stop_services` | Stop running services |
| `start_service` | Start a specific stopped service |
| `restart_service` | Restart a specific service |
| `install_dependencies` | Install deps for all or specific services |
| `check_requirements` | Check system requirements |
| `get_environment_variables` | List resolved environment variables |
| `set_environment_variable` | Set an environment variable |

## Common Workflows

```bash
# Run all services
azd app run

# Run a single service
azd app run --service api

# Run and open browser
azd app run --web

# Install all dependencies
azd app deps

# Run tests with coverage
azd app test --coverage --threshold 80

# Watch tests
azd app test --watch --service api

# Check health continuously
azd app health --stream --interval 10s

# View live logs
azd app logs --follow

# Start MCP server for AI assistants
azd app mcp serve
```
