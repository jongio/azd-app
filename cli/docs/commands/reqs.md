# azd app reqs

## Overview

The `reqs` command verifies that all required tools (prerequisites) are installed and meet minimum version requirements defined in `azure.yaml`. It can also auto-generate requirements from detected project dependencies.

## Purpose

- **Validate Prerequisites**: Ensure required tools are installed before running services
- **Version Compliance**: Verify tools meet minimum version requirements
- **Runtime Verification**: Check if tools are running (e.g., Docker daemon)
- **Auto-Generation**: Scan projects and auto-generate requirements configuration
- **Performance**: Cache results to speed up repeated checks

## Command Usage

```bash
azd app reqs [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--generate` | `-g` | bool | `false` | Generate reqs from detected project dependencies |
| `--dry-run` | | bool | `false` | Preview changes without modifying azure.yaml |
| `--no-cache` | | bool | `false` | Force fresh reqs check and bypass cached results |
| `--clear-cache` | | bool | `false` | Clear cached reqs results |
| `--fix` | | bool | `false` | Attempt to fix PATH issues for missing tools |

## Execution Flow

### Standard Check Mode

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app reqs                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check for Cached Results                                    │
│  Location: .azure/cache/reqs-results.json                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
            Cache Found?       No Cache or --no-cache
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌─────────────────────┐
            │ Use Cache   │   │  Perform Fresh      │
            │ Results     │   │  Requirement Check  │
            └─────────────┘   └─────────────────────┘
                    │                │
                    │                ↓
                    │         ┌──────────────────────┐
                    │         │ Parse azure.yaml     │
                    │         │ Extract reqs section │
                    │         └──────────────────────┘
                    │                │
                    │                ↓
                    │         ┌──────────────────────────┐
                    │         │ For Each Prerequisite:   │
                    │         │  1. Check if installed   │
                    │         │  2. Get version          │
                    │         │  3. Compare versions     │
                    │         │  4. Check if running     │
                    │         └──────────────────────────┘
                    │                │
                    │                ↓
                    │         ┌──────────────────────┐
                    │         │ Save Results to      │
                    │         │ Cache                │
                    │         └──────────────────────┘
                    │                │
                    └────────┬───────┘
                             ↓
                    ┌─────────────────┐
                    │ Display Results │
                    │  - Success/Fail │
                    │  - Versions     │
                    │  - Running?     │
                    └─────────────────┘
                             ↓
                    ┌─────────────────┐
                    │ Return Exit Code│
                    │  0 = Success    │
                    │  1 = Failure    │
                    └─────────────────┘
```

### Generate Mode Flow

```
┌─────────────────────────────────────────────────────────────┐
│              azd app reqs --generate                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detect Project Dependencies                                 │
│  - Scan for package.json, requirements.txt, *.csproj, etc.   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Determine Required Tools                                    │
│  - Node.js projects → node, npm/pnpm/yarn                    │
│  - Python projects → python, pip/poetry/uv                   │
│  - .NET projects → dotnet, aspire (if Aspire)                │
│  - Docker files → docker                                     │
│  - Git repo → git                                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check Installed Versions                                    │
│  - Execute: node --version, python --version, etc.           │
│  - Parse version output                                      │
│  - Normalize versions (major for Node, major.minor for Python)│
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Existing azure.yaml                                   │
│  - Read current reqs section (if exists)                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Merge Requirements                                          │
│  - Add new detected tools                                    │
│  - Skip duplicates                                           │
│  - Preserve existing requirements                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              --dry-run?            No
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌──────────────────┐
            │ Preview     │   │ Update azure.yaml│
            │ Changes     │   │ with new reqs    │
            └─────────────┘   └──────────────────┘
                    │                │
                    └────────┬───────┘
                             ↓
                    ┌─────────────────┐
                    │ Display Summary │
                    │  - Tools added  │
                    │  - Versions     │
                    └─────────────────┘
```

### Fix Mode Flow

```
┌─────────────────────────────────────────────────────────────┐
│                 azd app reqs --fix                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Run Initial Requirements Check                              │
│  - Identify failed prerequisites                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
           All Satisfied?         Some Failed
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌──────────────────────┐
            │ Report      │   │ Refresh System PATH  │
            │ Success     │   │  - Windows: Read from│
            │             │   │    Machine + User env│
            └─────────────┘   │  - Unix: Current PATH│
                              └──────────────────────┘
                                       │
                                       ↓
                              ┌──────────────────────────┐
                              │ For Each Failed Tool:    │
                              │  1. Search in PATH       │
                              │  2. Search common dirs   │
                              │  3. Re-verify version    │
                              └──────────────────────────┘
                                       │
                                       ↓
                              ┌──────────────────────────┐
                              │ Re-check All Reqs        │
                              │  - After PATH refresh    │
                              └──────────────────────────┘
                                       │
                                       ↓
                              ┌──────────────────────────┐
                              │ Report Results:          │
                              │  - Fixed count           │
                              │  - Still missing         │
                              │  - Install suggestions   │
                              └──────────────────────────┘
```


## Prerequisite Checking Details

### Version Extraction Process

For each tool, the command:

1. **Executes Version Command**: Runs the tool-specific version command
2. **Parses Output**: Extracts version from command output
3. **Normalizes Format**: Applies tool-specific normalization

#### Tool Registry

The command uses a built-in registry for known tools:

| Tool | Command | Args | Version Field | Prefix |
|------|---------|------|---------------|--------|
| node | node | --version | 0 | v |
| npm | npm | --version | 0 | |
| pnpm | pnpm | --version | 0 | |
| yarn | yarn | --version | 0 | |
| python | python | --version | 1 | |
| pip | pip | --version | 1 | |
| poetry | poetry | --version | 2 | |
| uv | uv | --version | 0 | |
| pipenv | pipenv | --version | 0 | |
| dotnet | dotnet | --version | 0 | |
| aspire | aspire | --version | 0 | |
| docker | docker | --version | 2 | |
| git | git | --version | 2 | |
| go | go | version | 2 | |
| func | func | --version | 0 | |
| az | az | version | 0 | |
| azd | azd | version | 0 | |

**Tool Aliases**:

Some tools have alternative names that are automatically resolved:

| Alias | Resolves To |
|-------|-------------|
| nodejs | node |
| azure-cli | az |
| azure-functions-core-tools | func |

**Version Field Explanation**:
- `0`: Use entire output
- `1`: Use second field (split by whitespace)
- `2`: Use third field

**Example Outputs**:
```bash
# Node.js (field 0, strip "v" prefix)
$ node --version
v20.11.0
→ Extracted: 20.11.0

# Python (field 1)
$ python --version
Python 3.12.0
→ Extracted: 3.12.0

# Docker (field 2)
$ docker --version
Docker version 24.0.7, build ...
→ Extracted: 24.0.7
```

### Version Comparison

The command uses **semantic version comparison**:

```
┌──────────────────────────────────────────────┐
│  Parse versions into parts                   │
│  Example: "3.12.0" → [3, 12, 0]              │
└──────────────────────────────────────────────┘
                    ↓
┌──────────────────────────────────────────────┐
│  Compare each part left to right             │
│  - installed[i] > required[i] → PASS         │
│  - installed[i] < required[i] → FAIL         │
│  - installed[i] = required[i] → CONTINUE     │
└──────────────────────────────────────────────┘
                    ↓
┌──────────────────────────────────────────────┐
│  All parts equal or higher → SATISFIED       │
└──────────────────────────────────────────────┘
```

**Examples**:

| Installed | Required | Result | Reason |
|-----------|----------|--------|--------|
| 3.12.0 | 3.11.0 | ✅ PASS | 12 > 11 |
| 3.10.0 | 3.11.0 | ❌ FAIL | 10 < 11 |
| 3.11.5 | 3.11.0 | ✅ PASS | Equal major.minor, higher patch |
| 20.0.0 | 18.0.0 | ✅ PASS | 20 > 18 |

### Runtime Checking

For tools that require a running daemon (like Docker), the command can verify the service is active:

```yaml
reqs:
  - name: docker
    minVersion: "20.0.0"
    checkRunning: true
```

**Default Runtime Checks**:

| Tool | Check Command | Expected |
|------|---------------|----------|
| docker | `docker ps` | Exit code 0 |

**Custom Runtime Checks**:

```yaml
reqs:
  - name: postgres
    minVersion: "15.0.0"
    checkRunning: true
    runningCheckCommand: "pg_isready"
    runningCheckArgs: ["-h", "localhost"]
    runningCheckExitCode: 0
```

**Runtime Check Flow**:

```
┌─────────────────────────────────────────┐
│ Check if tool is installed & version OK │
└─────────────────────────────────────────┘
                    ↓
            ┌───────┴────────┐
            │                │
      checkRunning?          No
            │                │
           Yes               ↓
            │         ┌─────────────┐
            ↓         │   DONE      │
┌────────────────────┐└─────────────┘
│ Execute Runtime    │
│ Check Command      │
└────────────────────┘
            ↓
┌────────────────────────────┐
│ Verify Exit Code & Output  │
│  - Check expected exit code│
│  - Search for expected text│
└────────────────────────────┘
            ↓
    ┌───────┴────────┐
    │                │
  Success          Fail
    │                │
    ↓                ↓
┌─────────┐   ┌──────────┐
│RUNNING  │   │NOT       │
│✓        │   │RUNNING ✗ │
└─────────┘   └──────────┘
```

### Podman Support

When Podman is installed and aliased to Docker (common in rootless container setups), the `docker` command returns Podman's multi-line version format instead of Docker's single-line format.

**Behavior:**
- Podman is detected by checking for "Podman Engine" in the output
- **Version comparison is skipped** because Docker and Podman use incompatible version schemes (Docker: `20.x`, `28.x` vs Podman: `4.x`, `5.x`)
- The requirement is marked satisfied if Podman is installed
- If `checkRunning: true`, the runtime check still executes

**Example Output:**
```
✓ docker: 5.7.0 via Podman (version check skipped)
  ✓ RUNNING
```

**JSON Output:**
```json
{
  "name": "docker",
  "installed": true,
  "version": "5.7.0",
  "required": "20.0.0",
  "satisfied": true,
  "isPodman": true,
  "message": "Podman detected (version check skipped)"
}
```

This ensures that users with Podman configured as a Docker replacement can use `azd app reqs` without false failures due to version mismatches.

## Caching Mechanism

### Cache Location

```
project-root/
  .azure/
    cache/
      reqs-results.json
```

### Cache Structure

```json
{
  "version": "1.0",
  "timestamp": "2024-11-04T10:30:00Z",
  "results": [
    {
      "name": "node",
      "installed": true,
      "version": "20.11.0",
      "required": "18.0.0",
      "satisfied": true,
      "running": false,
      "checkedRunning": false,
      "message": "Satisfied"
    },
    {
      "name": "docker",
      "installed": true,
      "version": "24.0.7",
      "required": "20.0.0",
      "satisfied": true,
      "running": true,
      "checkedRunning": true,
      "message": "Running"
    }
  ]
}
```

### Cache Invalidation

The cache is considered valid if:
- Cache file exists
- Cache file is readable
- `--no-cache` flag is NOT used

The cache is **automatically cleared** when:
- `--clear-cache` flag is used
- `azure.yaml` reqs section is modified (future enhancement)

### Cache Benefits

- **Speed**: Skips tool execution on repeated runs
- **Consistency**: Shows same results unless explicitly refreshed
- **Offline**: Can show previous results without network/tool access

## Auto-Generation Details

### Detection Logic

```
┌────────────────────────────────────────────┐
│  Scan Project for Dependency Files         │
└────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────┐
│  Node.js Detection                         │
│  - package.json → node required            │
│  - package-lock.json → npm                 │
│  - pnpm-lock.yaml → pnpm                   │
│  - yarn.lock → yarn                        │
└────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────┐
│  Python Detection                          │
│  - requirements.txt → python + pip         │
│  - pyproject.toml → check tool.poetry/uv   │
│  - Pipfile → pipenv                        │
└────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────┐
│  .NET Detection                            │
│  - *.csproj → dotnet                       │
│  - Check for Aspire.Hosting → aspire       │
└────────────────────────────────────────────┘
                    ↓
┌────────────────────────────────────────────┐
│  Infrastructure Detection                  │
│  - Dockerfile → docker                     │
│  - docker-compose.yml → docker             │
│  - .git/ → git                             │
└────────────────────────────────────────────┘
```

### Version Normalization

Different tools have different version normalization strategies:

| Tool Type | Strategy | Example Input | Output |
|-----------|----------|---------------|--------|
| Node.js | Major only | v20.11.0 | 20.0.0 |
| Python | Major.Minor | 3.12.5 | 3.12.0 |
| .NET | As-is | 8.0.100 | 8.0.100 |
| Docker | As-is | 24.0.7 | 24.0.7 |

**Rationale**:
- **Node.js**: Major version breaks are significant; minor versions are compatible
- **Python**: Minor versions matter (3.11 vs 3.12 has breaking changes)
- **.NET**: Full version needed for SDK features
- **Docker**: Full version for debugging support

### Merge Strategy

When generating requirements:

1. **Read existing** `reqs` section from `azure.yaml`
2. **Detect new** requirements from project
3. **Merge** by ID:
   - If ID exists → **keep existing** (preserve manual edits)
   - If ID new → **add detected**
4. **Sort** by ID alphabetically
5. **Write** back to `azure.yaml`

```
Existing azure.yaml:           Detected:              Result:
reqs:                          - docker: 24.0.7       reqs:
  - name: node                 - node: 20.11.0          - name: docker
    minVersion: "18.0.0"       - python: 3.12.0           minVersion: "24.0.7"
                                                        - name: node
                                                          minVersion: "18.0.0"  (preserved)
                                                        - name: python
                                                          minVersion: "3.12.0"
```

## Configuration

### azure.yaml Structure

```yaml
name: my-project

reqs:
  # Basic requirement
  - name: node
    minVersion: "18.0.0"
  
  # Azure Functions Core Tools
  - name: func
    minVersion: "4.0.0"
  
  # With runtime check
  - name: docker
    minVersion: "20.0.0"
    checkRunning: true
  
  # Custom tool configuration
  - name: postgres
    minVersion: "15.0.0"
    command: "psql"
    args: ["--version"]
    versionField: 2
    checkRunning: true
    runningCheckCommand: "pg_isready"
    runningCheckArgs: ["-h", "localhost"]
    runningCheckExitCode: 0
```

### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Unique tool identifier |
| `minVersion` | string | ✅ | Minimum version (semantic) |
| `command` | string | ❌ | Override command to execute |
| `args` | []string | ❌ | Override command arguments |
| `versionPrefix` | string | ❌ | Prefix to strip (e.g., "v") |
| `versionField` | int | ❌ | Which field contains version |
| `checkRunning` | bool | ❌ | Check if tool is running |
| `runningCheckCommand` | string | ❌ | Command to check running status |
| `runningCheckArgs` | []string | ❌ | Arguments for running check |
| `runningCheckExpected` | string | ❌ | Expected substring in output |
| `runningCheckExitCode` | int | ❌ | Expected exit code (default: 0) |
| `installUrl` | string | ❌ | URL to installation page (shown on failure) |

## Output Formats

### Text Output (Default)

```
✓ Checking prerequisites

✓ node: 20.11.0 (required: 18.0.0)
✓ docker: 24.0.7 (required: 20.0.0)
  - ✓ RUNNING
✗ python: NOT INSTALLED (required: 3.11.0)
   Install: https://www.python.org/downloads/

✗ Some prerequisites are not satisfied
```

### Install URLs

When a requirement check fails (not installed or version too old), the command displays an install URL to help users quickly find installation instructions.

**Built-in Install URLs**:

The following tools have built-in install URLs:

| Tool | Install URL |
|------|-------------|
| node | https://nodejs.org/ |
| npm | https://nodejs.org/ |
| pnpm | https://pnpm.io/installation |
| yarn | https://yarnpkg.com/getting-started/install |
| python | https://www.python.org/downloads/ |
| pip | https://www.python.org/downloads/ |
| poetry | https://python-poetry.org/docs/#installation |
| uv | https://docs.astral.sh/uv/getting-started/installation/ |
| pipenv | https://pipenv.pypa.io/en/latest/installation.html |
| dotnet | https://dotnet.microsoft.com/download |
| aspire | https://learn.microsoft.com/dotnet/aspire/fundamentals/setup-tooling |
| docker | https://www.docker.com/products/docker-desktop |
| git | https://git-scm.com/downloads |
| go | https://go.dev/dl/ |
| azd | https://aka.ms/install-azd |
| az | https://aka.ms/installazurecli |
| func | https://learn.microsoft.com/azure/azure-functions/functions-run-local |
| java | https://adoptium.net/ |
| mvn | https://maven.apache.org/install.html |
| gradle | https://gradle.org/install/ |

**Custom Install URLs**:

For custom tools or to override the built-in URL, specify `installUrl` in your azure.yaml:

```yaml
reqs:
  - name: mytool
    minVersion: "1.0.0"
    command: "mytool"
    args: ["--version"]
    installUrl: "https://example.com/mytool/install"
```

**Output with Custom Install URL**:
```
✗ mytool: NOT INSTALLED (required: 1.0.0)
   Install: https://example.com/mytool/install
```

### JSON Output (`--output json`)

```json
{
  "satisfied": false,
  "reqs": [
    {
      "name": "node",
      "installed": true,
      "version": "20.11.0",
      "required": "18.0.0",
      "satisfied": true,
      "message": "Satisfied",
      "installUrl": "https://nodejs.org/"
    },
    {
      "name": "docker",
      "installed": true,
      "version": "24.0.7",
      "required": "20.0.0",
      "satisfied": true,
      "running": true,
      "checkedRunning": true,
      "message": "Running",
      "installUrl": "https://www.docker.com/products/docker-desktop"
    },
    {
      "name": "python",
      "installed": false,
      "required": "3.11.0",
      "satisfied": false,
      "message": "Not installed",
      "installUrl": "https://www.python.org/downloads/"
    }
  ]
}
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | All prerequisites satisfied |
| 1 | Failure | One or more prerequisites not satisfied |

## Common Use Cases

### 1. First-Time Project Setup

```bash
# Check what's needed
azd app reqs

# Install missing tools manually, then verify
azd app reqs --no-cache
```

### 2. Auto-Generate Configuration

```bash
# Preview what would be generated
azd app reqs --generate --dry-run

# Generate and save to azure.yaml
azd app reqs --generate
```

### 3. CI/CD Pipeline

```bash
# Force fresh check in CI
azd app reqs --no-cache --output json

# Exit code indicates pass/fail
```

### 4. Cache Management

```bash
# Clear stale cache
azd app reqs --clear-cache

# Then run fresh check
azd app reqs --no-cache
```

### 5. Fixing PATH Issues

```bash
# When tools are installed but not detected
azd app reqs --fix

# The fix command will:
# 1. Refresh environment PATH from system settings
# 2. Search for missing tools in common locations
# 3. Re-verify requirements after PATH update
# 4. Provide installation instructions for truly missing tools
```

## Integration with Other Commands

The `reqs` command is the **foundation** of the command dependency chain:

```
reqs
  ↓
deps (automatically runs reqs first)
  ↓
run (automatically runs deps → reqs)
```

This ensures prerequisites are always validated before:
- Installing dependencies
- Running services

## Troubleshooting

### Issue: "Tool not found" but it's installed

**Cause**: Tool not in PATH or PATH needs refresh

**Solution 1: Use --fix flag**:
```bash
# Automatically refresh PATH and search for tools
azd app reqs --fix
```

**Important: PATH Refresh Behavior**

**Windows**: The `--fix` command refreshes PATH from the Windows registry, but changes only affect the current `azd app` process and its child processes. If you've recently installed a tool and added it to your system PATH, you may need to:
1. Close all terminal windows
2. Open a new terminal
3. Run `azd app reqs --fix` again

This limitation exists because Windows processes inherit environment variables at startup and cannot reload system-wide PATH changes dynamically.

**Unix/macOS**: The `--fix` command uses the current session's PATH. If you've modified shell profile files (`.bashrc`, `.zshrc`, etc.), you must restart your terminal for changes to take effect.

**Solution 2: Manual PATH configuration**:
```bash
# Check PATH
echo $PATH

# Add tool to PATH or use custom command:
# azure.yaml:
reqs:
  - name: node
    command: "/usr/local/bin/node"
    args: ["--version"]
```

### Issue: Version check always fails

**Cause**: Incorrect version field parsing

**Solution**:
```bash
# Manually check version output
node --version  # Output: v20.11.0

# Configure field correctly:
reqs:
  - name: node
    versionField: 0      # Use full output
    versionPrefix: "v"   # Strip 'v' prefix
```

### Issue: Cache shows outdated results

**Solution**:
```bash
# Clear cache and re-check
azd app reqs --clear-cache
azd app reqs --no-cache
```

### Issue: Docker shows installed but "not running"

**Cause**: Docker daemon not started

**Solution**:
```bash
# Start Docker Desktop or daemon
# Then verify:
azd app reqs --no-cache
```

## Best Practices

1. **Version Specifications**: Use realistic minimum versions based on features you need
2. **Running Checks**: Only enable `checkRunning` for daemons that must be active
3. **Custom Tools**: Document custom tool configurations in comments
4. **CI/CD**: Always use `--no-cache` in automated pipelines
5. **Generation**: Review generated requirements before committing
6. **Cache Clearing**: Clear cache after tool updates

## Related Commands

- [`azd app deps`](./deps.md) - Install dependencies (depends on reqs)
- [`azd app run`](./run.md) - Run services (depends on deps → reqs)

## Examples

### Example 1: Basic Check

```bash
$ azd app reqs

✓ Checking prerequisites

✓ node: 20.11.0 (required: 18.0.0)
✓ npm: 10.2.4 (required: 9.0.0)
✓ docker: 24.0.7 (required: 20.0.0)
  - ✓ RUNNING

✓ All prerequisites satisfied
```

### Example 2: Generate from Project

```bash
$ azd app reqs --generate --dry-run

Detecting project dependencies...

Detected tools:
  - node: 20.11.0
  - pnpm: 8.15.0
  - python: 3.12.0
  - pip: 24.0
  - docker: 24.0.7
  - git: 2.43.0

Dry-run mode: azure.yaml would be updated with:

reqs:
  - name: docker
    minVersion: "24.0.7"
  - name: git
    minVersion: "2.43.0"
  - name: node
    minVersion: "20.0.0"
  - name: pip
    minVersion: "24.0.0"
  - name: pnpm
    minVersion: "8.15.0"
  - name: python
    minVersion: "3.12.0"
```

### Example 3: Custom Tool

```yaml
reqs:
  - name: postgresql
    command: "psql"
    args: ["--version"]
    versionField: 2
    versionPrefix: ""
    minVersion: "15.0.0"
    checkRunning: true
    runningCheckCommand: "pg_isready"
    runningCheckArgs: ["-h", "localhost", "-p", "5432"]
```

```bash
$ azd app reqs

✓ Checking prerequisites

✓ postgresql: 15.4.0 (required: 15.0.0)
  - ✓ RUNNING
```

### Example 4: Fix PATH Issues

```bash
# Initial check shows tools missing
$ azd app reqs

✓ Checking prerequisites

✗ node: NOT INSTALLED (required: 20.0.0)
✗ docker: NOT INSTALLED (required: 20.10.0)
✓ python: 3.12.0 (required: 3.11.0)

✗ Some prerequisites are not satisfied

# Run fix to resolve PATH issues
$ azd app reqs --fix

🔧 Attempting to fix requirement issues...
   ✗ node: NOT INSTALLED (required: 20.0.0)
   ✗ docker: NOT INSTALLED (required: 20.10.0)

🔄 Refreshing environment PATH...
   ✓ PATH refreshed successfully

🔍 Searching for node...
   ✓ Found: C:\Program Files\nodejs\node.exe
   ✓ Version verified successfully

🔍 Searching for docker...
   ✓ Found: C:\Program Files\Docker\Docker\resources\bin\docker.exe
   ✓ Version verified successfully

📋 Re-checking requirements...
   ✓ node: 24.11.0 (required: 20.0.0)
   ✓ docker: 28.5.1 (required: 20.10.0)
   ✓ python: 3.12.0 (required: 3.11.0)

✓ Fixed 2 of 2 issues!

✓ All requirements now satisfied!

# JSON output mode
$ azd app reqs --fix --output json
{
  "success": true,
  "fixed": 2,
  "total": 2,
  "allSatisfied": true,
  "fixes": [
    {
      "name": "node",
      "fixed": true,
      "found": true,
      "path": "C:\\Program Files\\nodejs\\node.exe",
      "message": "Found and verified: C:\\Program Files\\nodejs\\node.exe",
      "satisfied": true
    },
    {
      "name": "docker",
      "fixed": true,
      "found": true,
      "path": "C:\\Program Files\\Docker\\Docker\\resources\\bin\\docker.exe",
      "message": "Found and verified: C:\\Program Files\\Docker\\Docker\\resources\\bin\\docker.exe",
      "satisfied": true
    }
  ],
  "results": [
    {
      "name": "node",
      "installed": true,
      "version": "24.11.0",
      "required": "20.0.0",
      "satisfied": true,
      "message": "Satisfied"
    },
    {
      "name": "docker",
      "installed": true,
      "version": "28.5.1",
      "required": "20.10.0",
      "satisfied": true,
      "message": "Satisfied"
    },
    {
      "name": "python",
      "installed": true,
      "version": "3.12.0",
      "required": "3.11.0",
      "satisfied": true,
      "message": "Satisfied"
    }
  ]
}
```
