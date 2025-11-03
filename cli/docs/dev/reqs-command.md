# App Reqs Command

The `azd app reqs` command verifies that all required prerequisites defined in your `azure.yaml` file are installed and meet the minimum version requirements. It can also check if services are running (e.g., Docker daemon).

## Usage

```bash
# Navigate to your azd project directory
cd your-project

# Run the reqs command
azd app reqs
```

Or use the binary directly:

```bash
.\bin\App.exe reqs
```

## Prerequisites Format

Add a `reqs` section to your `azure.yaml` file:

```yaml
name: your-project
metadata:
  template: your-template@1.0.0

reqs:
  # JavaScript ecosystem
  - id: nodejs
    minVersion: "20.0.0"
    
  - id: pnpm
    minVersion: "9.0.0"
    
  # Python ecosystem
  - id: python
    minVersion: "3.12.0"
    
  # .NET ecosystem
  - id: dotnet
    minVersion: "9.0.0"
    
  # Azure tooling
  - id: azd
    minVersion: "1.5.0"
    
  - id: azure-cli
    minVersion: "2.60.0"
    
  # Docker with running check
  - id: docker
    minVersion: "20.0.0"
    checkRunning: true
```

## Runtime Checks (New Feature)

You can now verify that a dependency is not just installed, but also **running**. This is useful for services like Docker, databases, or other background processes.

### Built-in Running Checks

For known tools, the extension provides default running checks:

- **Docker**: Automatically runs `docker ps` to verify the Docker daemon is running

### Custom Running Checks

For custom services, you can define your own running checks:

```yaml
reqs:
  # Check if PostgreSQL is installed and running
  - id: postgresql
    minVersion: "14.0.0"
    command: psql
    args: ["--version"]
    checkRunning: true
    runningCheckCommand: pg_isready
    runningCheckArgs: ["-h", "localhost"]
    runningCheckExitCode: 0
    
  # Check if Redis is installed and running
  - id: redis
    minVersion: "6.0.0"
    command: redis-cli
    args: ["--version"]
    checkRunning: true
    runningCheckCommand: redis-cli
    runningCheckArgs: ["ping"]
    runningCheckExpected: "PONG"
    
  # Check if a custom service is running
  - id: my-service
    minVersion: "1.0.0"
    command: my-service
    args: ["--version"]
    checkRunning: true
    runningCheckCommand: my-service
    runningCheckArgs: ["status"]
    runningCheckExpected: "active"
    runningCheckExitCode: 0
```

### Running Check Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `checkRunning` | boolean | No | Whether to check if the tool is running |
| `runningCheckCommand` | string | No | Command to execute to check if running |
| `runningCheckArgs` | array | No | Arguments for the running check command |
| `runningCheckExpected` | string | No | Expected substring in the output (optional) |
| `runningCheckExitCode` | integer | No | Expected exit code (default: 0) |

**How it works:**
1. The version check runs first (if the tool isn't installed, no running check occurs)
2. If `checkRunning: true`, the running check executes
3. The check passes if:
   - Exit code matches `runningCheckExitCode` (default: 0), AND
   - If `runningCheckExpected` is set, the output contains that substring

## Supported Tools

The reqs command supports version checking for the following tools:

- `nodejs` / `node` - Node.js runtime
- `pnpm` - PNPM package manager
- `python` - Python runtime
- `dotnet` - .NET SDK
- `aspire` - .NET Aspire CLI
- `azd` - Azure Developer CLI
- `azure-cli` / `az` - Azure CLI
- `docker` - Docker (with built-in running check)
- Any tool that supports `--version` flag

## Custom Tool Configuration

For tools not in the registry, you can specify custom version check commands:

```yaml
reqs:
  - id: my-custom-tool
    minVersion: "2.1.0"
    command: my-tool
    args: ["version"]
    versionPrefix: "v"
    versionField: 1
```

### Custom Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Command to execute (overrides default) |
| `args` | array | Arguments to pass to the command |
| `versionPrefix` | string | Prefix to strip from version (e.g., "v" for "v1.2.3") |
| `versionField` | integer | Which field contains the version (0=whole output, 1=second field, etc.) |

## Output

The command will display:
- ✅ Green checkmark for tools that are installed and meet version requirements
- ✅ Green "RUNNING" indicator for tools that pass the running check
- ❌ Red X for tools that are missing or don't meet version requirements
- ❌ Red "NOT RUNNING" for tools that are installed but not running
- ⚠️ Warning for tools that are installed but version cannot be determined

Example output:

```
🔍 Checking requirements...

✅ nodejs: 22.19.0 (required: 20.0.0)
✅ pnpm: 10.20.0 (required: 9.0.0)
✅ python: 3.13.9 (required: 3.12.0)
✅ dotnet: 10.0.100 (required: 9.0.0)
✅ azd: 1.20.3 (required: 1.5.0)
✅ azure-cli: 2.67.0 (required: 2.60.0)
✅ docker: 24.0.5 (required: 20.0.0) - ✅ RUNNING

✅ All requirements are satisfied!
```

Example with failures:

```
🔍 Checking requirements...

✅ nodejs: 22.19.0 (required: 20.0.0)
❌ docker: 24.0.5 (required: 20.0.0) - ❌ NOT RUNNING
❌ postgresql: NOT INSTALLED (required: 14.0.0)

❌ Some requirements are missing or don't meet minimum version requirements
```

## Exit Codes

- `0` - All requirements are satisfied
- `1` - One or more requirements are missing, don't meet minimum version, or aren't running

## Testing

A test project is included in `tests/projects/azure/` with sample `azure.yaml` files containing various requirements. To test:

```bash
cd tests/projects/azure
azd app reqs
```

## Version Comparison

The command uses semantic versioning comparison:
- Compares major.minor.patch versions
- Installed version must be >= required version
- Example: 3.13.9 satisfies requirement for 3.12.0
