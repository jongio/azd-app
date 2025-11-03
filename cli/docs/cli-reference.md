# CLI Reference

Complete reference for all `azd app` commands and flags.

## Global Information

All commands automatically inherit azd environment context when run through `azd app <command>`. This includes Azure subscription information, resource groups, and environment-specific variables.

See [azd-context.md](azd-context.md) for details on accessing azd environment variables.

## Commands Overview

| Command | Description |
|---------|-------------|
| `reqs` | Check and verify required tools and optionally auto-generate requirements |
| `deps` | Install dependencies for detected projects |
| `run` | Run the development environment with service orchestration |
| `logs` | View logs from running services |
| `info` | Show information about running services |
| `version` | Show version information |
| `listen` | Extension framework integration (hidden, used by azd internally) |

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
| `--gen` | | bool | `false` | Alias for --generate |
| `--dry-run` | | bool | `false` | Preview changes without modifying azure.yaml |
| `--no-cache` | | bool | `false` | Force fresh reqs check and bypass cached results |
| `--clear-cache` | | bool | `false` | Clear cached reqs results |

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
  - id: docker
    minVersion: "20.0.0"
    checkRunning: true
  - id: nodejs
    minVersion: "20.0.0"
  - id: python
    minVersion: "3.12.0"
```

See [reqs-command.md](dev/reqs-command.md) for detailed documentation.

---

## `azd app deps`

Automatically detects your project type and installs all dependencies.

### Usage

```bash
azd app deps
```

### Examples

```bash
# Install dependencies for all detected projects
azd app deps
```

### Flags

None. This command automatically detects and installs dependencies.

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
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Run specific service(s) only (comma-separated) |
| `--runtime` | | string | `azd` | Runtime mode: 'azd' (azd dashboard) or 'aspire' (native Aspire with dotnet run) |
| `--env-file` | | string | | Load environment variables from .env file |
| `--verbose` | `-v` | bool | `false` | Enable verbose logging |
| `--dry-run` | | bool | `false` | Show what would be run without starting services |

### Runtime Modes

#### azd (default)
- Uses azd's built-in dashboard
- Works with all project types
- Provides unified experience across languages
- Service orchestration and monitoring

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

### Service Configuration

Define services in `azure.yaml`:

```yaml
name: my-app
services:
  web:
    language: js
    host: containerapp
    project: ./src/web
  api:
    language: python
    host: containerapp
    project: ./src/api
```

### Dependencies

This command depends on `deps` and `reqs`, which will automatically run before starting services.

---

## `azd app logs`

View logs from running services with filtering and follow support.

### Usage

```bash
azd app logs [flags]
```

### Examples

```bash
# View logs from all services
azd app logs

# Follow logs in real-time
azd app logs --follow

# View logs for specific service(s)
azd app logs --service web,api

# Show last 50 lines
azd app logs --tail 50

# Show logs from last 5 minutes
azd app logs --since 5m

# Filter by log level
azd app logs --level error

# Output as JSON
azd app logs --format json

# Write logs to file
azd app logs --output debug.log

# Disable timestamps
azd app logs --timestamps=false

# Disable colored output
azd app logs --no-color
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--follow` | `-f` | bool | `false` | Follow log output (tail -f behavior) |
| `--service` | `-s` | string | | Filter by service name(s) (comma-separated) |
| `--tail` | `-n` | int | `100` | Number of lines to show from the end |
| `--since` | | string | | Show logs since duration (e.g., 5m, 1h) |
| `--timestamps` | | bool | `true` | Show timestamps with each log entry |
| `--no-color` | | bool | `false` | Disable colored output |
| `--level` | | string | `all` | Filter by log level (info, warn, error, debug, all) |
| `--format` | | string | `text` | Output format (text, json) |
| `--output` | | string | | Write logs to file instead of stdout |

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
azd app info --project /path/to/project
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | | bool | `false` | Show services from all projects on this machine |
| `--project` | | string | | Show services from a specific project directory |

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

---

## `azd app version`

Show version information for the azd app extension.

### Usage

```bash
azd app version
```

### Examples

```bash
# Display version
azd app version
```

### Output

Displays the current version of the extension:
```
azd app extension version 0.1.0
```

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

See [azd-context.md](azd-context.md) for complete details.

### Extension-Specific

- `AZAPP_VERBOSE`: Enable verbose logging (set by `--verbose`)
- `AZAPP_DRY_RUN`: Enable dry-run mode (set by `--dry-run`)

---

## Command Dependencies

Some commands automatically run prerequisite commands:

```
run → deps → reqs
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
```

### Debugging Issues

```bash
# Check requirements
azd app reqs --no-cache

# Preview what would run
azd app run --dry-run --verbose

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
