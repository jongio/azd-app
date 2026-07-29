# CLI Reference

Complete reference for all `azd app` commands and flags.

## 🔐 Azure Key Vault Integration

**All commands that start services or execute code automatically resolve Azure Key Vault references in environment variables.**

Supported formats:
- `@Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name)`
- `@Microsoft.KeyVault(VaultName=vault;SecretName=name)`
- `akvs://<guid>/<vault>/<secret>`

**→ [See Azure Key Vault Integration Guide](features/keyvault-integration.md)** for complete documentation, examples, and security best practices.

**Why it matters:**
- ✅ Store secrets securely in Azure Key Vault, not in code
- ✅ Works transparently - your code just uses `os.getenv()` / `process.env`
- ✅ Uses Azure authentication you already have (`az login`, Managed Identity)
- ✅ Graceful degradation - services start even if resolution fails (with warnings)

## Global Information

All commands automatically inherit azd environment context when run through `azd app <command>`. This includes Azure subscription information, resource groups, and environment-specific variables.

See [dev/azd-context-inheritance.md](dev/azd-context-inheritance.md) for details on accessing azd environment variables.

### Terminal Display

Progress bars automatically adapt to terminal width. Narrow terminals (<70 columns) use compact mode to prevent line wrapping.

### Global Flags

These flags are available for all commands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `default` | Output format (default, json) |
| `--debug` | | bool | `false` | Enable debug logging |
| `--structured-logs` | | bool | `false` | Enable structured JSON logging to stderr |
| `--cwd` | `-C` | string | `""` | Sets the current working directory |
| `--environment` | `-e` | string | `""` | The name of the environment to use |
| `--no-prompt` | | bool | `false` | Run without prompts, failing when a required value cannot be resolved |
| `--help` | `-h` | bool | `false` | Show help for the command |

**Examples:**
```bash
# Output in JSON format
azd app reqs --output json

# Enable debug logging
azd app run --debug

# Enable structured logs for log aggregation
azd app deps --structured-logs

# Run from a specific project directory
azd app run --cwd ./my-project

# Use a specific environment
azd app run --environment production
```

## Commands Overview

| Command | Description | Detailed Spec |
|---------|-------------|---------------|
| `init` | Initialize azure.yaml for local development by scanning your project | [→ Full Spec](commands/init.md) |
| `validate` | Validate azure.yaml with read-only checks before running services | [→ Full Spec](commands/validate.md) |
| `reqs` | Check and verify required tools and optionally auto-generate requirements | [→ Full Spec](commands/reqs.md) |
| `deps` | Install dependencies for detected projects | [→ Full Spec](commands/deps.md) |
| `outdated` | Report outdated dependencies across services | [→ Full Spec](commands/outdated.md) |
| `add` | Add a well-known container service to azure.yaml | [→ Full Spec](commands/add.md) |
| `config` | Show the effective resolved configuration for each service | [→ Full Spec](commands/config.md) |
| `run` | Run the development environment with service orchestration and lifecycle hooks | [→ Full Spec](commands/run.md) |
| `test` | Run tests for all services with coverage aggregation | [→ Full Spec](commands/test.md) |
| `start` | Start stopped services | [→ Full Spec](commands/start.md) |
| `stop` | Stop running services | [→ Full Spec](commands/stop.md) |
| `restart` | Restart services | [→ Full Spec](commands/restart.md) |
| `status` | Show whether `azd app run` is active | [→ Full Spec](commands/status.md) |
| `health` | Monitor health status of services (static or streaming mode) | [→ Full Spec](commands/health.md) |
| `logs` | View logs from running services | [→ Full Spec](commands/logs.md) |
| `info` | Show information about running services | [→ Full Spec](commands/info.md) |
| `graph` | Show the service dependency graph | [→ Full Spec](commands/graph.md) |
| `env` | Print the resolved environment for a service | [→ Full Spec](commands/env.md) |
| `proxy` | Route local requests to running services | [→ Full Spec](commands/proxy.md) |
| `cert` | Generate local HTTPS certificates | [→ Full Spec](commands/cert.md) |
| `clean` | Reclaim disk space from build artifacts and caches | [→ Full Spec](commands/clean.md) |
| `support-bundle` | Collect local diagnostics for support | [→ Full Spec](commands/support-bundle.md) |
| `remove` | Remove a service from azure.yaml | [→ Full Spec](commands/remove.md) |
| `hooks` | List the lifecycle hooks configured in azure.yaml | [→ Full Spec](commands/hooks.md) |
| `ports` | List the host ports each service binds and flag duplicate bindings | [→ Full Spec](commands/ports.md) |
| `open` | Open a running service URL in the browser | [→ Full Spec](commands/open.md) |
| `mcp` | Model Context Protocol server for AI assistant integration | [→ Full Spec](commands/mcp.md) |
| `notifications` | Manage process notifications for service state changes | [→ Full Spec](commands/notifications.md) |
| `version` | Show version information | [→ Full Spec](commands/version.md) |
| `listen` | Extension framework integration (hidden, used by azd internally) | [→ Full Spec](commands/listen.md) |

---

## `azd app init`

Scans your project to detect services, languages, frameworks, and infrastructure dependencies, then generates or enriches `azure.yaml` with all the settings needed for `azd app run`.

### Usage

```bash
azd app init [flags]
```

### Examples

```bash
# Initialize in current directory
azd app init

# Preview what would be generated (no file changes)
azd app init --dry-run

# Force overwrite of existing services section
azd app init --force
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dry-run` | | bool | `false` | Show what would be generated without writing files |
| `--force` | | bool | `false` | Overwrite existing services section in azure.yaml |

### Features

- ✅ Detects Node.js/TypeScript, Python, .NET, Go, Azure Functions, Logic Apps
- ✅ Identifies frameworks (Express, FastAPI, Next.js, Vite, etc.) and sets correct ports
- ✅ Infers run commands based on package manager and framework
- ✅ Detects infrastructure dependencies (postgres, redis, mongodb, etc.) → `uses` field
- ✅ Generates `reqs` section from detected languages
- ✅ Non-destructive enrichment of existing azure.yaml
- ✅ Deduplicates services when multiple detectors match the same directory
- ✅ JSON output for programmatic consumption

---

## `azd app validate`

Validates `azure.yaml` with read-only checks for service references, project paths, ports, service types, modes, and command readiness. Nothing is started and no files are written.

### Usage

```bash
azd app validate
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `default` | Output format: 'default' or 'json' (inherited from parent) |

### Checks

| Check ID | Severity | Description |
|----------|----------|-------------|
| `yaml.parse` | error | `azure.yaml` is not valid YAML |
| `schema.parse` | error | The parser rejected the configuration (reported only when no other error is found) |
| `services.empty` | warning | No services are defined |
| `service.name` | error | Service name contains unsupported characters |
| `project.missing` | error | `project` path does not exist |
| `project.not-directory` | error | `project` path is not a directory |
| `project.outside-root` | error | `project` path resolves outside the project root |
| `uses.unknown` | error | `uses` entry is not a defined service or resource |
| `port.invalid` | error | Host port is not between 1 and 65535 |
| `port.duplicate` | error | Two services request the same host port |
| `type.unsupported` | error | Service `type` is not http, tcp, process, or container |
| `mode.unsupported` | error | Service `mode` is not watch, build, daemon, or task |
| `command.missing` | warning | Process service has no command in `azure.yaml` |

### Examples

```bash
# Validate azure.yaml in the current project
azd app validate

# Validate as JSON for scripts and CI
azd app validate --output json
```

### Output

Findings are sorted by service, then by check ID. Warnings alone do not fail the command; any error-severity finding does.

```
azd app validate
──────────────────────────────

   [FAIL] web port.duplicate: host port 8080 is also used by service "api"
         fix: Assign a unique host port to one of the services.
   [WARN] worker command.missing: process service has no command in azure.yaml
         fix: Set command or make sure detection can infer how to run it.
```

JSON output emits the findings array (`[]` when there is nothing to report):

```json
[
  {
    "file": "/path/to/azure.yaml",
    "service": "web",
    "checkId": "port.duplicate",
    "severity": "error",
    "message": "host port 8080 is also used by service \"api\"",
    "hint": "Assign a unique host port to one of the services."
  }
]
```

**→ [See full validate command specification](commands/validate.md)** for the complete check list and exit codes.

---

## `azd app reqs`

Verifies that all required tools are installed and optionally checks if they are running. Can also auto-generate requirements from your project.

### Usage

```bash
azd app reqs [flags]
```

### Examples

```bash
# Check requirements defined in azure.yaml
azd app reqs

# Auto-generate requirements from your project
azd app reqs --generate

# Preview what would be generated without making changes
azd app reqs --generate --dry-run

# Force fresh check bypassing cache
azd app reqs --no-cache

# Clear cached requirement results
azd app reqs --clear-cache
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--generate` | `-g` | bool | `false` | Generate reqs from detected project dependencies |
| `--dry-run` | | bool | `false` | Preview changes without modifying azure.yaml |
| `--no-cache` | | bool | `false` | Force fresh reqs check and bypass cached results |
| `--clear-cache` | | bool | `false` | Clear cached reqs results |
| `--only-missing` | | bool | `false` | Show only requirements that are missing, too old, or not running |
| `--fix` | | bool | `false` | Attempt to fix PATH issues for missing tools |

### Features

- ✅ Checks if required tools are installed
- ✅ Validates minimum version requirements
- ✅ Verifies if services are running (e.g., Docker daemon)
- ✅ Auto-generates requirements from detected project dependencies
- ✅ Smart version normalization (Node: major only, Python: major.minor)
- ✅ Merges with existing requirements without duplicates
- ✅ Supports custom tool configurations

### Supported Tool Detection

- **Node.js**: Detects npm, pnpm, or yarn based on lock files
- **Python**: Detects pip, poetry, uv, or pipenv
- **.NET**: Detects dotnet SDK and Aspire workloads
- **Docker**: Detects from Dockerfile or docker-compose files
- **Git**: Detects from .git directory

### Configuration

Define requirements in `azure.yaml`:

```yaml
name: my-project
reqs:
  - name: docker
    minVersion: "20.0.0"
    checkRunning: true
  - name: nodejs
    minVersion: "20.0.0"
  - name: python
    minVersion: "3.12.0"
```

**→ [See full reqs command specification](commands/reqs.md)** for flows, diagrams, and detailed documentation.

---

## `azd app deps`

Automatically detects your project type and installs all dependencies.

### Usage

```bash
azd app deps [flags]
```

### Examples

```bash
# Install dependencies for all detected projects
azd app deps

# Show full installation output
azd app deps --verbose

# Clean reinstall (removes node_modules, .venv first)
azd app deps --clean

# Force fresh install (combines --clean and --no-cache)
azd app deps --force

# Or use run --force to reinstall deps before starting
azd app run --force
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | `false` | Show full installation output |
| `--clean` | | bool | `false` | Remove existing dependencies before installing (clears node_modules, .venv, etc.) |
| `--no-cache` | | bool | `false` | Force fresh dependency installation and bypass cached results |
| `--force` | `-f` | bool | `false` | Force clean reinstall (combines --clean and --no-cache) |
| `--dry-run` | | bool | `false` | Show what would be installed without actually installing |
| `--check` | | bool | `false` | Verify dependencies are installed without installing; exits non-zero if any are missing |
| `--service` | `-s` | strings | | Install dependencies only for specific services (can be specified multiple times) |

### Features

- 🔍 Detects Node.js, Python, and .NET projects
- 📦 Identifies package manager (npm/pnpm/yarn, uv/poetry/pip, dotnet)
- 🚀 Installs dependencies with the correct tool
- 🐍 Creates Python virtual environments automatically

### Supported Package Managers

- **Node.js**: npm, pnpm, yarn
- **Python**: uv, poetry, pip
- **.NET**: dotnet restore

### Dependencies

This command depends on `reqs` and will automatically run prerequisite checks before installing dependencies.

**→ [See full deps command specification](commands/deps.md)** for package manager detection flows and detailed documentation.

---

## `azd app outdated`

Check every service in `azure.yaml` for outdated dependencies and print one aggregated report.

The package manager is detected per service: npm, pnpm, or yarn for Node (based on the lockfile), pip for Python, `dotnet` for .NET, and `go` for Go. A service whose package manager is not installed is skipped with a warning rather than failing the run.

### Usage

```bash
azd app outdated [flags]
```

### Examples

```bash
# Report outdated dependencies for every service
azd app outdated

# Limit to one service
azd app outdated --service api

# Limit to selected package managers
azd app outdated --manager npm,pip

# Machine-readable output
azd app outdated --format json

# Fail with a non-zero exit code when anything is outdated, for CI gating
azd app outdated --exit-code

# Ignore packages you have pinned on purpose
azd app outdated --exit-code --ignore react,typescript
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | stringArray | | Limit to specific services (repeatable or comma-separated) |
| `--manager` | | stringArray | | Limit to package managers: npm, pnpm, yarn, pip, dotnet, or go |
| `--ignore` | | stringArray | | Package names to exclude from the report (repeatable or comma-separated) |
| `--format` | | string | `text` | Output format: `text` or `json` |
| `--exit-code` | | bool | `false` | Return a non-zero exit code when any dependency is outdated |

`--ignore` matches names case-insensitively, so a pinned package neither shows up in the report nor trips `--exit-code`.

**→ [See full outdated command specification](commands/outdated.md)** for per-manager behavior and detailed documentation.

---

## `azd app add`

Add a well-known container service to your azure.yaml configuration.

### Usage

```bash
azd app add [service] [flags]
```

### Examples

```bash
# List available services
azd app add --list

# Add Azurite storage emulator
azd app add azurite

# Add PostgreSQL and show connection string
azd app add postgres --show-connection

# JSON output
azd app add redis --output json
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--list` | | bool | `false` | List all available services |
| `--dry-run` | | bool | `false` | Show the azure.yaml service block without modifying the file |
| `--show-connection` | | bool | `false` | Show connection string after adding |

### Available Services

- `azurite` - Azure Storage emulator (Blob, Queue, Table)
- `cosmos` - Azure Cosmos DB emulator
- `redis` - Redis in-memory cache
- `postgres` - PostgreSQL database

**→ [See full add command specification](commands/add.md)** for examples and configuration details.

---

## `azd app run`

Starts your development environment based on project type with support for multi-service orchestration.

### Usage

```bash
azd app run [flags]
```

### Examples

```bash
# Run with default azd dashboard
azd app run

# Run specific services only
azd app run --service web,api

# Use native Aspire dashboard (for .NET Aspire projects)
azd app run --runtime aspire

# Preview what would run without starting
azd app run --dry-run

# Enable verbose logging
azd app run --verbose

# Load environment variables from custom file
azd app run --env-file .env.local

# Combine multiple flags
azd app run -s web -v --runtime aspire

# Force clean dependency reinstall before running
azd app run --force
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Run specific service(s) only (comma-separated) |
| `--except` | | string | | Run every service except the named one(s) (comma-separated); cannot be combined with `--service` |
| `--scale` | | stringArray | | Run multiple instances of a service, for example `--scale worker=3` (repeatable, comma-separated) |
| `--detach` | | bool | `false` | Run the app in the background and return to the shell |
| `--runtime` | | string | `azd` | Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run) |
| `--env-file` | | string | | Load environment variables from .env file |
| `--env` | | stringArray | | Set an environment variable inline as KEY=VALUE (repeatable, overrides --env-file) |
| `--no-deps` | | bool | `false` | Skip reqs and dependency installation before starting services |
| `--verbose` | `-v` | bool | `false` | Enable verbose logging |
| `--dry-run` | | bool | `false` | Show what would be run without starting services |
| `--no-timing` | | bool | `false` | Hide the per-service startup timing summary shown after services are ready |
| `--restart-containers` | | bool | `false` | Restart containers even if they are already running |
| `--force` | `-f` | bool | `false` | Force clean dependency reinstall (passes --force to deps) |
| `--trust` | | bool | `false` | Trust this workspace for code execution and remember the decision |
| `--skip-secret-scan` | | bool | `false` | Skip the advisory scan for hardcoded secrets in tracked config |
| `--skip-exposure-check` | | bool | `false` | Skip the warning shown when a service binds to all network interfaces |
| `--web` | `-w` | bool | `false` | Open dashboard in browser |

### Runtime Modes

#### azd (default)
- Uses azd's built-in dashboard
- Works with all project types
- Provides unified experience across languages
- Service orchestration and monitoring
- Log source switcher for local/Azure logs

#### aspire
- Uses native .NET Aspire dashboard via `dotnet run`
- Only for .NET Aspire projects with AppHost.cs
- Provides full Aspire tooling integration
- Access to Aspire-specific features

### Supported Project Types

- **azure.yaml services**: Multi-service orchestration with defined services
- **.NET Aspire**: Projects with AppHost.cs
- **Node.js**: pnpm dev/start scripts
- **Docker Compose**: Container orchestration
- **Logic Apps Standard**: Azure Logic Apps workflows (see [Azure Functions + Logic Apps](features/azure-functions.md))

### Service Configuration

Define services in `azure.yaml`:

```yaml
name: my-app
services:
  web:
    language: js
    host: local
    project: ./src/web
  api:
    language: python
    host: local
    project: ./src/api
```

### Dependencies

This command depends on `deps` and `reqs`, which will automatically run before starting services.

### Hooks

The `run` command supports lifecycle hooks that execute before and after services start:

- **prerun**: Executes before starting any services (e.g., database migrations, setup tasks)
- **postrun**: Executes after all services are ready (e.g., notifications, opening browsers)

**→ [See Hooks Documentation](hooks.md)** for complete hook configuration and examples.

**→ [See full run command specification](commands/run.md)** for orchestration flows, runtime modes, and detailed documentation.

---

## `azd app test`

Run tests for all services in your application with support for different test types and aggregated code coverage.

### Usage

```bash
azd app test [flags]
```

### Examples

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

# Fail fast - stop on first failure
azd app test --fail-fast

# Run tests in parallel
azd app test --parallel

# Set coverage threshold
azd app test --coverage --threshold 80

# Dry run - show what would be tested
azd app test --dry-run
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--type` | `-t` | string | `all` | Test type to run: `unit`, `integration`, `e2e`, or `all` |
| `--coverage` | `-c` | bool | `false` | Generate code coverage reports |
| `--service` | `-s` | string | `""` | Run tests for specific service(s) (comma-separated) |
| `--changed` | | bool | `false` | Only test services with files changed since `--changed-base` |
| `--changed-base` | | string | | Git ref to compare against for `--changed` (for example `HEAD` or `origin/main`) |
| `--watch` | `-w` | bool | `false` | Watch mode - re-run tests on file changes |
| `--update-snapshots` | `-u` | bool | `false` | Update test snapshots |
| `--fail-fast` | | bool | `false` | Stop on first test failure |
| `--parallel` | `-p` | bool | `true` | Run tests for services in parallel |
| `--threshold` | | int | `0` | Minimum coverage threshold (0-100) |
| `--verbose` | `-v` | bool | `false` | Enable verbose test output |
| `--dry-run` | | bool | `false` | Show what would be tested without running tests |
| `--output-format` | | string | `default` | Output format: `default`, `json`, `junit`, `github` |
| `--output-dir` | | string | `./test-results` | Directory for test reports and coverage |
| `--stream` | | bool | `false` | Force streaming output even in parallel mode |
| `--no-stream` | | bool | `false` | Force progress bar mode (suppress streaming) |
| `--timeout` | | duration | `10m` | Per-service timeout for test execution |
| `--save` | | bool | `false` | Save auto-detected test config to azure.yaml |
| `--no-save` | | bool | `false` | Don't prompt to save auto-detected test config |

### Smart Output Modes

The test command automatically selects the best output mode based on context:

| Scenario | Output Mode | Description |
|----------|-------------|-------------|
| Single service | Streaming | Output streams directly to terminal |
| Multiple services + `--parallel` | Progress bars | Shows progress bars for each service |
| Multiple services + sequential | Prefixed streaming | Streams with `[service]` prefix |
| CI/non-TTY environment | Streaming | Always streams for log compatibility |

Use `--stream` to force streaming output even when running multiple services in parallel. Use `--no-stream` to force progress bar mode.

> **💡 Troubleshooting Tip:** If tests appear to hang with no output, try using `--stream` to see real-time test output. This is especially useful when debugging long-running tests or investigating test failures.

### Test Types

| Type | Purpose | Speed | Examples |
|------|---------|-------|----------|
| `unit` | Test individual functions/classes | Fast | Pure functions, business logic |
| `integration` | Test component interactions | Medium | Database ops, API calls |
| `e2e` | Test complete workflows | Slow | UI flows, full scenarios |

### Supported Frameworks

**Node.js**: Jest, Vitest, Mocha, AVA, Tap  
**Python**: pytest, unittest, nose2  
**Go**: go test (built-in)  
**.NET**: xUnit, NUnit, MSTest

### Configuration

Define test configuration in `azure.yaml`:

```yaml
services:
  api:
    language: python
    project: ./src/api
    test:
      framework: pytest
      unit:
        command: pytest tests/unit -v
      integration:
        command: pytest tests/integration -v
        setup:
          - docker-compose up -d postgres
        teardown:
          - docker-compose down
      coverage:
        enabled: true
        threshold: 90
  
  gateway:
    language: go
    project: ./src/gateway
    test:
      framework: gotest
      unit:
        pattern: "^Test[^Integration]"
      integration:
        pattern: "TestIntegration"
      coverage:
        threshold: 80
```

**→ [See full test command specification](commands/test.md)** for detailed documentation, auto-detection rules, and coverage aggregation.

**→ [See test configuration schema](schema/test-configuration.md)** for complete YAML configuration reference.

---

## `azd app start`

Start stopped services.

### Usage

```bash
azd app start [flags]
```

### Examples

```bash
# Start a specific service
azd app start --service api

# Start multiple services
azd app start --service "api,web,worker"

# Start all stopped services
azd app start --all
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Service name(s) to start (comma-separated) |
| `--all` | | bool | `false` | Start all stopped services |

### Description

Start one or more stopped services that were previously running. This command operates on the service registry maintained by `azd app run`. If no services are registered, use `azd app run` to start your development environment first.

**→ [See full start command specification](commands/start.md)** for complete documentation.

---

## `azd app stop`

Stop running services, execute lifecycle hooks, and tear down the app.

### Usage

```bash
azd app stop [flags]
```

### Examples

```bash
# Stop the running app (from any terminal in the project)
azd app stop

# Stop specific services
azd app stop --service api,web

# Stop every running service without a confirmation prompt
azd app stop --all --yes
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | | string | | Service name(s) to stop (comma-separated) |
| `--all` | | bool | `false` | Stop all running services |
| `--yes` | | bool | `false` | Skip confirmation prompt for `--all` |

### Description

Sends a shutdown signal to the running `azd app run` process. This triggers graceful shutdown including prestop/poststop hooks, port release, and process cleanup, identical to pressing Ctrl+C in the run terminal. With no flags it stops the whole app; use `--service` to stop individual services.

**→ [See full stop command specification](commands/stop.md)** for complete documentation.

---

## `azd app restart`

Restart services.

### Usage

```bash
azd app restart [flags]
```

### Examples

```bash
# Restart a specific service
azd app restart --service api

# Restart multiple services
azd app restart --service "api,web,worker"

# Restart all services
azd app restart --all
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Service name(s) to restart (comma-separated) |
| `--all` | | bool | `false` | Restart all services |
| `--yes` | `-y` | bool | `false` | Skip confirmation prompt for `--all` |

### Description

Restart one or more services. This command stops and then starts services. It works on both running and stopped services. Services are stopped gracefully before being restarted.

**→ [See full restart command specification](commands/restart.md)** for complete documentation.

---

## `azd app status`

Report whether `azd app run` is currently active for this project, and print a one-line summary per service.

The command reads the run state file that `azd app run` maintains, so it works from any terminal and does not attach to the running session. When nothing is running it prints a short "not running" message and exits 0, which makes it safe to call from scripts and shell prompts.

### Usage

```bash
azd app status [flags]
```

### Examples

```bash
# One-shot status for every service
azd app status

# Status for a single service
azd app status --service api

# Live view that refreshes on an interval
azd app status --watch

# Refresh every five seconds instead of the default two
azd app status --watch --interval 5s

# Print only the dashboard URL, useful for scripting
azd app status --dashboard-url
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Show status for a single service |
| `--watch` | | bool | `false` | Continuously refresh the status view |
| `--interval` | | string | `2s` | Refresh interval when `--watch` is set (Go duration, for example `500ms` or `5s`) |
| `--dashboard-url` | | bool | `false` | Print only the dashboard URL and exit |
| `--exit-code` | | bool | `false` | Return a non-zero exit code when the app is not running (ignored with `--watch`) |

**→ [See full status command specification](commands/status.md)** for run-state details and detailed documentation.

---

## `azd app health`

Monitor the health status of running services with production-grade reliability and observability features.

**⭐ NEW: Production Features**
- Circuit breaker pattern to prevent cascading failures
- Rate limiting per service to avoid overwhelming endpoints
- Result caching to reduce redundant checks
- Prometheus metrics exposition for observability
- Structured logging (JSON, pretty, or text)
- Environment-specific profiles (dev, prod, ci, staging)

See [health-production-features.md](health-production-features.md) for comprehensive documentation.

### Usage

```bash
azd app health [flags]
```

### Examples

**Basic Usage:**
```bash
# Quick health check of all services
azd app health

# Check health of specific service(s)
azd app health --service web,api

# Stream health updates in real-time
azd app health --stream --interval 10s

# Output as JSON for automation
azd app health --output json
```

**Production Features:**
```bash
# Use production profile (circuit breaker + metrics + caching)
azd app health --profile production --stream

# Development mode with verbose logging
azd app health --profile development --log-level debug

# Custom production config
azd app health \
  --circuit-breaker \
  --rate-limit 10 \
  --cache-ttl 5s \
  --metrics \
  --log-format json

# Generate sample profiles
azd app health --save-profiles
```

### Flags

**Basic Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Monitor specific service(s) only (comma-separated) |
| `--stream` | | bool | `false` | Enable streaming mode for real-time updates |
| `--interval` | `-i` | duration | `5s` | Interval between health checks in streaming mode |
| `--output` | `-o` | string | `text` | Output format: 'text', 'json', 'table' |
| `--endpoint` | | string | `/health` | Default health endpoint path to check |
| `--timeout` | | duration | `5s` | Timeout for each health check |
| `--all` | | bool | `false` | Show health for all projects on this machine |
| `--fail-on-degraded` | | bool | `false` | Return a non-zero exit code when any service is degraded |
| `--summary-only` | | bool | `false` | Print only the aggregate health summary for non-JSON output |
| `--verbose` | `-v` | bool | `false` | Show detailed health check information |

**Production Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--profile` | string | | Health profile: development, production, ci, staging, or custom |
| `--log-level` | string | `info` | Log level: debug, info, warn, error |
| `--log-format` | string | `pretty` | Log format: json, pretty, text |
| `--save-profiles` | bool | `false` | Save sample health profiles to .azd/health-profiles.yaml |
| `--metrics` | bool | `false` | Enable Prometheus metrics exposition |
| `--metrics-port` | int | `9090` | Port for Prometheus metrics endpoint |
| `--circuit-breaker` | bool | `false` | Enable circuit breaker pattern |
| `--circuit-break-count` | int | `5` | Number of failures before opening circuit |
| `--circuit-break-timeout` | duration | `60s` | Circuit breaker timeout duration |
| `--rate-limit` | int | `0` | Max health checks per second per service (0 = unlimited) |
| `--cache-ttl` | duration | `0` | Cache TTL for health results (0 = no caching) |

### Features

**Basic Features:**
- ✅ **HTTP Health Checks**: Automatically detect and use `/health` endpoints
- ✅ **Port Checks**: Fall back to TCP port checks for non-HTTP services
- ✅ **Process Checks**: Verify process is running as last resort
- ✅ **Streaming Mode**: Real-time continuous monitoring with configurable intervals
- ✅ **Static Mode**: Point-in-time health snapshot
- ✅ **Smart Detection**: Try common health paths (/health, /healthz, /ready, /alive)
- ✅ **Multiple Formats**: Text, JSON, or table output

**Production Features (NEW):**
- 🔥 **Circuit Breaker**: Prevents cascading failures with automatic recovery
- 🚦 **Rate Limiting**: Per-service token bucket rate limiter
- ⚡ **Result Caching**: TTL-based caching to reduce load
- 📊 **Prometheus Metrics**: 6 metrics for full observability
- 📝 **Structured Logging**: JSON/pretty/text with configurable levels
- 🎯 **Health Profiles**: Environment-specific configurations

### Health Check Strategy

The command uses a cascading strategy:

1. **HTTP Health Endpoint** (Preferred)
   - Check explicit `healthCheck.endpoint` in azure.yaml
   - Try common paths: `/health`, `/healthz`, `/ready`, `/alive`, `/ping`
   - Accept 2xx and 3xx status codes as healthy

2. **TCP Port Check** (Fallback)
   - Verify service is listening on configured port
   - Useful for databases, non-HTTP services

3. **Process Check** (Last Resort)
   - Verify process is still running
   - Least reliable, only confirms existence

### Health Status Values

| Status | Meaning | Criteria |
|--------|---------|----------|
| `healthy` | Service fully operational | HTTP 2xx/3xx, port listening, or process running |
| `degraded` | Service running with issues | HTTP returns degraded status |
| `unhealthy` | Service not functioning | HTTP 4xx/5xx, port not listening, process dead |
| `starting` | Service initializing | Recently started, not yet ready |
| `unknown` | Cannot determine health | No health check available or check error |

### Configuration

Define health checks in `azure.yaml`:

```yaml
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
    healthCheck:
      type: http              # http, port, process
      endpoint: /api/health   # HTTP endpoint path
      timeout: 5s             # Timeout for each check
      interval: 10s           # Interval for streaming mode
      headers:                # Optional HTTP headers
        Authorization: Bearer token
```

### Output Formats

#### Text (default)
```
Health Check (2024-11-08 10:30:00)
=====================================

✓ web                          healthy      (http)
  Response Time: 45ms

✓ api                          healthy      (http)
  Response Time: 23ms

Summary: 2 healthy, 0 degraded, 0 unhealthy
Overall Status: HEALTHY
```

#### JSON
```json
{
  "timestamp": "2024-11-08T10:30:00Z",
  "services": [
    {
      "serviceName": "web",
      "status": "healthy",
      "checkType": "http",
      "responseTime": 45
    }
  ],
  "summary": {
    "total": 1,
    "healthy": 1,
    "overall": "healthy"
  }
}
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | All services healthy |
| `1` | One or more services unhealthy or degraded |
| `2` | Error performing health checks |
| `130` | Interrupted (Ctrl+C in streaming mode) |

**→ [See full health command specification](commands/health.md)** for health check strategies, streaming mode details, and comprehensive documentation.

---

## `azd app logs`

View logs from running services with filtering and follow support.

### Usage

```bash
azd app logs [flags]
```

### Prerequisites by Source

| Source | Requires `azd app run`? | Notes |
|--------|-------------------------|-------|
| `local` (default) | Yes | Services must be running locally |
| `azure` | **No** | Queries Azure Log Analytics directly |
| `all` | Yes | Local component requires services |

### Examples

```bash
# Local logs (requires azd app run)
azd app run                       # Start services first
azd app logs                      # View logs from all services
azd app logs --follow             # Follow logs in real-time

# Azure logs (standalone - no azd app run required)
azd app logs --source azure               # View logs from Azure
azd app logs --source azure -f            # Stream Azure logs (polls every 30s)
azd app logs --source azure --since 1h    # Logs from last hour

# View logs for specific service(s)
azd app logs --service web,api

# Show last 50 lines
azd app logs --tail 50

# Show logs from last 5 minutes
azd app logs --since 5m

# Filter by log level
azd app logs --level error

# Show errors with 3 lines of context before and after
azd app logs --level error --context 3

# Output as JSON
azd app logs --format json

# Write logs to file
azd app logs --file debug.log

# Disable timestamps
azd app logs --timestamps=false

# Disable colored output
azd app logs --no-color
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--follow` | `-f` | bool | `false` | Follow log output (tail -f behavior) |
| `--source` | | string | `local` | Log source: `local`, `azure`, or `all` |
| `--service` | `-s` | string | | Filter by service name(s) (comma-separated) |
| `--tail` | `-n` | int | `100` | Number of lines to show from the end |
| `--since` | | string | | Show logs since duration (e.g., 5m, 1h) |
| `--timestamps` | | bool | `true` | Show timestamps with each log entry |
| `--no-timestamps` | | bool | `false` | Hide timestamps in text log output |
| `--no-color` | | bool | `false` | Disable colored output |
| `--level` | | string | `all` | Filter by log level (info, warn, error, debug, all) |
| `--min-level` | | string | | Show entries at this severity or higher (`debug` < `info` < `warn` < `error`); cannot be combined with an explicit `--level` or `--context` |
| `--summary` | | bool | `false` | Show counts by service and level instead of raw log entries |
| `--context` | | int | `0` | Number of context lines before/after matching entries (0-10, requires --level) |
| `--format` | | string | `text` | Output format (text, json) |
| `--file` | | string | | Write logs to file instead of stdout |
| `--exclude` | | string | | Regex patterns to exclude (comma-separated) |
| `--grep` | | string | | Only show log lines matching this regex (applied after `--exclude`) |
| `--alerts` | | bool | `false` | Raise alerts for log lines matching built-in patterns (panic, unhandled exception, fatal) |
| `--redact` | | bool | `false` | Redact secret-shaped values before printing logs |
| `--no-builtins` | | bool | `false` | Disable built-in filter patterns |

### Log Sources

| Source | Description | Streaming |
|--------|-------------|-----------|
| `local` | Logs from locally running services (default) | Real-time via process stdout/stderr |
| `azure` | Logs from Azure Log Analytics | Polls every 30s (1-5 minute ingestion delay) |
| `all` | Both local and Azure logs merged | Mixed |

### Log Levels

- `all`: Show all log levels (default)
- `info`: Information messages
- `warn`: Warning messages
- `error`: Error messages only
- `debug`: Debug messages (most verbose)

### Output Formats

#### text (default)
Human-readable format with optional colors and timestamps:
```
2024-01-15 10:30:45 [web] INFO Starting server on port 3000
2024-01-15 10:30:46 [api] INFO Connected to database
```

#### json
Machine-readable JSON format:
```json
{"timestamp":"2024-01-15T10:30:45Z","service":"web","level":"info","message":"Starting server on port 3000"}
```

### Dashboard Log Viewer

The dashboard provides a visual log viewer with additional features:

- **Log Source Switcher**: Toggle between local and Azure logs
  - Local (💻): Logs from locally running services
  - Azure (☁️): Logs from Azure-deployed services via Log Analytics
  - Keyboard shortcut: `Ctrl+Shift+M` to toggle modes
- **Grid/Unified View**: Switch between per-service panes or unified log stream
- **Real-time Search**: Filter logs as you type
- **Export**: Download logs in multiple formats

**→ [See full logs command specification](commands/logs.md)** for log streaming flows, filtering mechanisms, and detailed documentation.

---

## `azd app info`

Show comprehensive information about running services.

### Usage

```bash
azd app info [flags]
```

### Examples

```bash
# Show info for services in current project
azd app info

# Show services from all projects on this machine
azd app info --all

# Show services from specific project directory
azd app info --cwd /path/to/project
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | | bool | `false` | Show services from all projects on this machine |
| `--service` | | string | | Show info for specific service(s) (comma-separated) |
| `--names` | | bool | `false` | Print service names only, one per line |
| `--watch` | | bool | `false` | Refresh service info on an interval until interrupted |
| `--interval` | | string | | Refresh interval for `--watch` (minimum `1s`) |
| `--cwd` | `-C` | string | | Sets the current working directory |

### Output

Displays comprehensive information including:
- Service names and types
- Running status
- URLs (HTTP/HTTPS endpoints)
- Health status
- Metadata (ports, PIDs, start time)

Example output:
```
Services in current project:

web
  Status: Running
  URL: http://localhost:3000
  Health: Healthy
  Type: Node.js
  PID: 12345

api
  Status: Running
  URL: http://localhost:5000
  Health: Healthy
  Type: Python
  PID: 12346
```

**→ [See full info command specification](commands/info.md)** for service registry details and detailed documentation.

---

## `azd app graph`

Show services, resources, dependency edges, and startup levels from `azure.yaml`.

`text`, `json`, and `markdown` print to stdout. `mermaid`, `dot`, and `d2` emit a diagram you can drop into a README or an architecture doc. Combine any format with `--output-file` to write the result to a file instead of stdout.

`--focus <service>` narrows the graph to one service, everything it depends on, and everything that depends on it. `--services-only` omits resource nodes and shows only service-to-service edges, which is useful for diagrams that need the app shape without managed resources.

### Usage

```bash
azd app graph [flags]
```

### Examples

```bash
# Human-readable text (default)
azd app graph

# Mermaid flowchart written to a file
azd app graph --output mermaid --output-file docs/services.mmd

# Graphviz DOT to stdout
azd app graph --output dot

# D2 diagram written to a file
azd app graph --output d2 --output-file docs/services.d2

# Markdown tables for docs or issue comments
azd app graph --output markdown

# Just the api service and its connected nodes
azd app graph --focus api

# Service-only Mermaid diagram
azd app graph --services-only --output mermaid
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `text` | Output format: `text`, `json`, `markdown`, `mermaid`, `dot`, or `d2` |
| `--output-file` | | string | | Write output to this file instead of stdout |
| `--focus` | | string | | Limit the graph to a service, its dependencies, and its dependents |
| `--services-only` | | bool | `false` | Show only services and service-to-service edges |

**→ [See full graph command specification](commands/graph.md)** for format details and detailed documentation.

---

## `azd app env`

Print the effective environment a service receives when it runs.

The output merges the process environment, the azd environment values, and the service-specific variables from `azure.yaml`, the same way `azd app run` resolves them. Pass a service name to print its environment, or run without a name to list the available services.

Secret-shaped values are masked by default. Use `--no-mask` to print raw values, for example when piping the output into another command.

`--all` prints the resolved environment for every service in one run. The `dotenv`, `shell`, and `powershell` formats group each service under a `# <service>` header; the `json` format emits an object keyed by service name.

### Usage

```bash
azd app env [service] [flags]
```

### Examples

```bash
# Resolved environment for the api service (KEY=value lines)
azd app env api

# Shell export statements
azd app env api --format shell

# PowerShell $env: assignments
azd app env api --format powershell | iex

# JSON object (also selected by the global --json flag)
azd app env api --format json

# Raw values, no masking
azd app env api --no-mask

# Explain where each effective value came from
azd app env api --explain

# Compare the resolved environment of two services
azd app env --diff api web

# List variable names without values
azd app env api --keys

# Write the resolved environment to api/.env
azd app env api --write

# Write one .env file per service into the build/env folder
azd app env --all --write --out build/env
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | | bool | `false` | Print the resolved environment for every service |
| `--format` | | string | `dotenv` | Output format: `dotenv`, `shell`, `powershell`, or `json` |
| `--keys` | | bool | `false` | Print variable names only |
| `--prefix` | | stringArray | | Only include variables whose names start with the given prefix (repeatable) |
| `--no-mask` | | bool | `false` | Print raw values instead of masking secret-shaped values |
| `--explain` | | bool | `false` | Show the source of each effective value and any sources it overrode |
| `--diff` | | bool | `false` | Compare the resolved environment of two services (pass two service names) |
| `--env-file` | | string | | Path to a `.env` file to merge, matching `azd app run` |
| `--write` | | bool | `false` | Write the resolved environment to a `.env` file instead of printing it |
| `--out` | | string | | Destination folder for `--write` files (writes `<service>.env`); defaults to each service directory |

**→ [See full env command specification](commands/env.md)** for resolution order and detailed documentation.

---

## `azd app proxy`

Start a local reverse proxy for running services.

Each running service with a local port gets a path route:

```text
/<service>/... -> http://localhost:<port>/...
```

The proxy strips the `/<service>` prefix before forwarding, so `/api/users` forwards to `/users` on the `api` service.

### Usage

```bash
azd app proxy [flags]
```

### Examples

```bash
# Start proxy on the default port
azd app proxy

# Start proxy on a custom port
azd app proxy --port 9090
```

Example route table:

```text
Proxy listening on http://localhost:8080
/api/ -> http://localhost:5001
/web/ -> http://localhost:3000
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--port` | | int | `8080` | Port for the proxy listener |

**→ [See full proxy command specification](commands/proxy.md)** for routing behavior and detailed documentation.

---

## `azd app cert`

Generate local HTTPS certificates for development.

This command creates a local certificate authority and a TLS server certificate under `~/.azd/app/certs`. By default it includes `localhost` and `127.0.0.1`.

Run it again to reuse existing valid certificates. Use `--force` to regenerate the server certificate and key.

### Usage

```bash
azd app cert [flags]
```

### Examples

```bash
# Generate certs for localhost and 127.0.0.1
azd app cert

# Add extra hosts (repeat --host as needed)
azd app cert --host api.local.test --host auth.local.test

# Regenerate the server certificate
azd app cert --force
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--host` | | stringArray | | Additional host to include in certificate SANs (repeatable) |
| `--force` | | bool | `false` | Regenerate server certificate and key |

**→ [See full cert command specification](commands/cert.md)** for trust-store setup and detailed documentation.

---

## `azd app clean`

Remove build output and cache directories for the services defined in `azure.yaml`.

By default `clean` removes build artifacts and caches (`dist`, `build`, `bin`, `obj`, `__pycache__`, `.pytest_cache`, and similar). Dependency directories such as `node_modules` and `.venv` are left in place unless you pass `--deps`.

Only directories inside a detected service directory are ever removed, and only when their name matches a known artifact. Paths outside the project are never touched.

### Usage

```bash
azd app clean [flags]
```

### Examples

```bash
# Show what would be removed and how much space it frees
azd app clean --dry-run

# Remove build artifacts across all services
azd app clean

# Also remove dependency directories
azd app clean --deps

# Only remove artifacts untouched for at least 24 hours
azd app clean --older-than 24h

# Limit to one service
azd app clean --service api
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | stringArray | | Limit to specific services (can be specified multiple times) |
| `--deps` | | bool | `false` | Also remove dependency directories (`node_modules`, `.venv`) |
| `--older-than` | | string | `0s` | Only remove artifacts older than this duration (for example, `24h`) |
| `--dry-run` | | bool | `false` | List what would be removed and the reclaimable size without deleting |

**→ [See full clean command specification](commands/clean.md)** for the artifact match list and detailed documentation.

---

## `azd app support-bundle`

Collect sanitized project, service, health, and log diagnostics into a local folder for issue reports.

Use `--dry-run` to preview the output folder and file list without writing files. Pass `--zip` to also create a shareable zip archive.

### Usage

```bash
azd app support-bundle [flags]
```

### Examples

```bash
# Preview the bundle plan without writing files
azd app support-bundle --dry-run

# Write a bundle to the default folder
azd app support-bundle

# Write a bundle and zip it for sharing
azd app support-bundle --zip

# Limit logs and health to selected services
azd app support-bundle --service api,web

# Include more log history per service
azd app support-bundle --tail 1000
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | | Output folder path |
| `--service` | `-s` | string | | Include logs and health for specific service(s), comma-separated |
| `--tail` | | int | `200` | Recent log lines per service to include |
| `--zip` | | bool | `false` | Create a zip archive after writing the support bundle |
| `--dry-run` | | bool | `false` | Show the bundle plan without writing files |

**→ [See full support-bundle command specification](commands/support-bundle.md)** for the redaction rules and detailed documentation.

---

## `azd app config`

Show the configuration azd app resolved from azure.yaml for each service.

### Usage

```bash
azd app config [service] [flags]
```

### Examples

```bash
# Configuration for every service
azd app config

# Configuration for a single service
azd app config api

# JSON object keyed by service name
azd app config --output json
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `default` | Output format (default, json) |

### Behavior

- Prints the host, effective service type (marked `explicit` or `inferred`), language, project path, run command, image, ports, and dependencies (uses) for each service.
- Lists which optional blocks are configured on a service: `docker`, `healthcheck`, `restart`, `resources`, `logs`, `local`, `azure`.
- Pass a service name to limit output to that service. An unknown name returns an error listing the available services.
- `--output json` emits an object keyed by service name.

**→ [See full config command specification](commands/config.md)** for details.

---

## `azd app remove`

Remove a service from the `services` section of `azure.yaml`.

This is the inverse of `azd app add`. It deletes only the named service entry. Other services and settings stay semantically unchanged, though yaml formatting may be normalized. Use it to undo an add or to drop a service you no longer run.

### Usage

```bash
azd app remove <service>
```

### Examples

```bash
# Remove the redis service
azd app remove redis

# JSON output
azd app remove redis --output json
```

### Behavior

- Removing a service that is not present fails and lists the current service names.
- Only the named service entry is deleted. Remaining services and settings are preserved.
- Supports the global `--output json` flag for scripting.

**→ [See full remove command specification](commands/remove.md)** for the yaml rewrite rules and detailed documentation.

---

## `azd app hooks`

List the project-level lifecycle hooks configured in `azure.yaml`.

Shows each configured hook with the command it runs, the shell it uses, and any per-platform override for Windows or POSIX. Use it to confirm what will run around a `run` or `stop` without opening the file.

### Usage

```bash
azd app hooks
```

### Examples

```bash
# List configured hooks
azd app hooks

# JSON array of hooks
azd app hooks --output json
```

### Lifecycle hooks

| Hook | When it runs |
|------|--------------|
| `prerun` | before services start |
| `postrun` | after all services are ready |
| `prestop` | before services are stopped |
| `poststop` | after services are stopped |

### Notes

- Hooks are listed in lifecycle order. Hooks that are not configured are omitted.
- A hook with a Windows or POSIX override shows the override command and shell on its own line.
- When no hooks are configured the command prints a short message and exits zero.

**→ [See full hooks command specification](commands/hooks.md)** for the override resolution rules and detailed documentation.

---

## `azd app open`

Resolve a service URL from `azure.yaml` or the running app state and open it in the default browser.

Use `--path` to append a route such as `/health`. Use `--print` to write the URL to stdout without launching a browser, which is useful in scripts and over SSH.

### Usage

```bash
azd app open <service> [flags]
```

### Examples

```bash
# Open the api service in the browser
azd app open api

# Open a specific route
azd app open api --path /health

# Print the URL instead of opening a browser
azd app open api --print
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--path` | | string | | Path to append to the service URL |
| `--print` | | bool | `false` | Print the URL without opening a browser |

**→ [See full open command specification](commands/open.md)** for URL resolution order and detailed documentation.

---

## `azd app ports`

Reads `azure.yaml` and lists the host port each service binds. An explicit host port is shown as its number; a port left for the tool to assign is shown as `auto`. When two bindings claim the same explicit host port the command reports the conflict and exits non-zero, so it works as a preflight check before `azd app run`.

### Usage

```bash
azd app ports [flags]
```

### Examples

```bash
# List host ports for every service
azd app ports

# JSON object keyed by service name
azd app ports --output json
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `text` | Output format: `text` or `json` |

### Notes

- Each binding is printed as `host -> container/protocol`. Ports without an explicit host binding show `auto`.
- A host port claimed by more than one binding (across services or within one service) is marked `(conflict)` and listed in a warning.
- The command exits non-zero when any conflict is found, so it can gate a run in scripts and CI.

---

## `azd app mcp`

Model Context Protocol (MCP) server for AI assistant integration. Enables AI assistants like Claude Desktop and GitHub Copilot to interact with your azd app projects.

### Usage

```bash
azd app mcp serve
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `serve` | Start the MCP server for AI assistant integration |

### Examples

```bash
# Start the MCP server (typically called by AI assistants)
azd app mcp serve

# Test the server manually
azd app mcp serve
# Then send MCP protocol messages via stdin
```

### Tools Provided

The MCP server exposes 10 tools:

| Category | Tool | Description |
|----------|------|-------------|
| Observability | `get_services` | Get comprehensive information about all running services |
| Observability | `get_service_logs` | Retrieve logs with filtering by service, level, time |
| Observability | `get_project_info` | Get project metadata from azure.yaml |
| Operations | `run_services` | Start development services |
| Operations | `stop_services` | Get guidance on stopping services |
| Operations | `restart_service` | Get guidance on restarting a service |
| Operations | `install_dependencies` | Install dependencies for all projects |
| Operations | `check_requirements` | Check if prerequisites are installed |
| Configuration | `get_environment_variables` | Get configured environment variables |
| Configuration | `set_environment_variable` | Get guidance on setting environment variables |

### Resources Provided

| URI | Name | Description |
|-----|------|-------------|
| `azure://project/azure.yaml` | azure.yaml | Project configuration file |
| `azure://project/services/configs` | service-configs | Service configurations |

### Integration

**Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "azd-app": {
      "command": "azd",
      "args": ["app", "mcp", "serve"]
    }
  }
}
```

**VS Code** (`.vscode/settings.json`):
```json
{
  "mcp": {
    "servers": {
      "Azure Developer CLI - App Extension": {
        "command": "azd",
        "args": ["app", "mcp", "serve"]
      }
    }
  }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROJECT_DIR` | Project directory for operations | `.` (current directory) |

**→ [See full mcp command specification](commands/mcp.md)** for tool parameters, technical details, and comprehensive documentation.

---

## `azd app version`

Show version information for the azd app extension.

### Usage

```bash
azd app version [flags]
```

### Examples

```bash
# Display version
azd app version

# Print only the version number, for scripting
azd app version --quiet
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--quiet` | | bool | `false` | Only print version number |

### Output

Displays the current version of the extension:
```
azd app extension version 0.5.1
```

**→ [See full version command specification](commands/version.md)** for version format details.

---

## `azd app completion`

Generate shell autocompletion scripts for `azd app`.

### Usage

```bash
azd app completion [command]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `bash` | Generate the autocompletion script for bash |
| `fish` | Generate the autocompletion script for fish |
| `powershell` | Generate the autocompletion script for PowerShell |
| `zsh` | Generate the autocompletion script for zsh |

### Examples

```bash
# Bash (add to ~/.bashrc)
azd app completion bash > ~/.azd-app-completion.bash

# Zsh (add to ~/.zshrc)
azd app completion zsh > ~/.azd-app-completion.zsh

# PowerShell (current session)
azd app completion powershell | Out-String | Invoke-Expression
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--help` | `-h` | bool | `false` | Show help for completion |

**→ [See full completion command specification](commands/completion.md)** for shell-specific install instructions.

---

## `azd app notifications`

View and manage notifications for service state changes and events.

### Usage

```bash
azd app notifications [command]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List notification history |
| `mark-read` | Mark notification(s) as read |
| `clear` | Clear notification history |
| `stats` | Show notification statistics |
| `test` | Send a test notification |
| `enable` | Enable or disable OS notifications |

### Flags

| Subcommand | Flag | Type | Default | Description |
|------------|------|------|---------|-------------|
| `list` | `--unread` | bool | `false` | Show only unread notifications |
| `list` | `--service` | string | | Filter by service name |
| `list` | `--limit` | int | `50` | Maximum number of notifications to show |
| `mark-read` | `--all` | bool | `false` | Mark all notifications as read |
| `clear` | `--older-than` | string | | Clear notifications older than duration (for example, `24h` or `7d`) |
| `clear` | `--yes` | bool | `false` | Clear all notification history without prompting |
| `enable` | `--disable` | bool | `false` | Disable OS notifications instead of enabling |

### Examples

```bash
# View all recent notifications
azd app notifications list

# View unread notifications only
azd app notifications list --unread

# View notifications for a specific service
azd app notifications list --service api

# Mark all notifications as read
azd app notifications mark-read --all

# Clear notifications older than 7 days
azd app notifications clear --older-than 168h

# Show notification statistics
azd app notifications stats

# Send a test notification
azd app notifications test

# Enable OS notifications
azd app notifications enable

# Disable OS notifications
azd app notifications enable --disable
```

**→ [See full notifications command specification](commands/notifications.md)** for complete subcommand documentation.

---

## `azd app listen`

Start the extension server (internal, required by the azd extension framework).

This command is invoked by `azd` to communicate with the extension over JSON-RPC on stdio.
It is hidden from `azd app --help` and is not intended to be run directly.

### Usage

```bash
azd app listen
```

**→ [See full listen command specification](commands/listen.md)** for integration notes.

---

## Exit Codes

All commands follow standard exit code conventions:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Misuse of command (invalid arguments) |

---

## Environment Variables

### Inherited from azd

When running through `azd app <command>`, these variables are automatically available:

- `AZURE_SUBSCRIPTION_ID`: Current Azure subscription
- `AZURE_RESOURCE_GROUP_NAME`: Target resource group
- `AZURE_ENV_NAME`: Environment name
- `AZURE_LOCATION`: Azure region
- `AZD_SERVER`: gRPC server address for azd communication
- `AZD_ACCESS_TOKEN`: Authentication token for azd API

See [dev/azd-context-inheritance.md](dev/azd-context-inheritance.md) for complete details.

### Extension-Specific

- `AZAPP_VERBOSE`: Enable verbose logging (set by `--verbose`)
- `AZAPP_DRY_RUN`: Enable dry-run mode (set by `--dry-run`)

---

## Command Dependencies

Some commands automatically run prerequisite commands:

```
run → deps → reqs
health → (no dependencies)
logs → (no dependencies)
info → (no dependencies)
reqs → (no dependencies)
deps → reqs
version → (no dependencies)
```

This ensures the environment is properly configured before execution. For example, `azd app run` will automatically:
1. Check prerequisites (`reqs`)
2. Install dependencies (`deps`)
3. Start services (`run`)

See [command-dependency-chain.md](dev/command-dependency-chain.md) for implementation details.

---

## Common Workflows

### First Time Setup

```bash
# Check prerequisites
azd app reqs

# Install dependencies
azd app deps

# Run development environment
azd app run
```

### Daily Development

```bash
# Start services
azd app run

# View logs in another terminal
azd app logs --follow

# Check service status
azd app info

# Monitor health in real-time
azd app health --stream
```

### Debugging Issues

```bash
# Check requirements
azd app reqs --no-cache

# Preview what would run
azd app run --dry-run --verbose

# Check health status
azd app health --verbose

# View error logs
azd app logs --level error

# Follow logs for specific service
azd app logs -f -s api
```

### Working with Aspire Projects

```bash
# Use native Aspire dashboard
azd app run --runtime aspire

# Run specific Aspire services
azd app run --runtime aspire -s web,api

# View Aspire service logs
azd app logs -f -s web
```

---

## Getting Help

For any command, use the `--help` flag:

```bash
azd app --help
azd app run --help
azd app logs --help
```

## Additional Resources

- [Azure Developer CLI Documentation](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
- [Extension Framework](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Project Repository](https://github.com/jongio/azd-app)
