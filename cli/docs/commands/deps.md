# azd app deps

## Overview

The `deps` command automatically detects project types and installs all dependencies using the appropriate package manager for each detected project (Node.js, Python, .NET).

## Purpose

- **Auto-Detection**: Automatically identify project types in your workspace
- **Package Manager Selection**: Choose the correct package manager based on lock files
- **Multi-Project Support**: Handle multiple projects with different languages
- **Dependency Installation**: Install all required dependencies
- **Virtual Environment Setup**: Create Python virtual environments automatically
- **Prerequisite Validation**: Ensure required tools are installed before proceeding

## Command Usage

```bash
azd app deps
```

### Flags

This command has no additional flags. It automatically detects and installs dependencies for all projects.

## Execution Flow

### Overall Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app deps                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Run Prerequisites Check (reqs)                              │
│  - Automatically executed via orchestrator                   │
│  - Ensures required tools are installed                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse azure.yaml                                            │
│  - Read services section                                     │
│  - Get project paths and languages                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  For Each Service:                                           │
│  1. Determine project type (Node.js/Python/.NET)             │
│  2. Detect package manager                                   │
│  3. Install dependencies                                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Report Results                                              │
│  - Success/failure for each project                          │
│  - Summary of installed dependencies                         │
└─────────────────────────────────────────────────────────────┘
```

### Detailed Service Detection Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Read Service from azure.yaml                                │
│  - service.project (directory path)                          │
│  - service.language (js/python/dotnet/csharp)                │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              language=js        language=python
                    │                │
                    ↓                ↓
        ┌────────────────┐   ┌────────────────┐
        │ Node.js Flow   │   │ Python Flow    │
        └────────────────┘   └────────────────┘
                    │                │
                    └────────┬───────┘
                             │
                    language=dotnet/csharp
                             │
                             ↓
                   ┌────────────────┐
                   │ .NET Flow      │
                   └────────────────┘
```

## Node.js Dependency Installation

### Package Manager Detection

```
┌─────────────────────────────────────────────────────────────┐
│  Check for Lock Files in Service Directory                   │
│  (with boundary - don't search parent directories)           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detection Priority:                                         │
│  1. pnpm-lock.yaml → pnpm                                    │
│  2. yarn.lock → yarn                                         │
│  3. package-lock.json → npm                                  │
│  4. package.json → npm (default)                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
                   pnpm             npm
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌──────────────────┐
        │ pnpm install     │ │ npm install      │
        └──────────────────┘ └──────────────────┘
                    │                │
                    └────────┬───────┘
                             │
                           yarn
                             │
                             ↓
                   ┌──────────────────┐
                   │ yarn install     │
                   └──────────────────┘
```

### Installation Process

```
┌─────────────────────────────────────────────────────────────┐
│  Validate package.json exists                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Package Manager Install Command                     │
│  - pnpm install                                              │
│  - npm install                                               │
│  - yarn install                                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Monitor Installation Progress                               │
│  - Stream output to console                                  │
│  - Capture errors                                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Verify Installation Success                                 │
│  - Check exit code                                           │
│  - Verify node_modules exists                                │
└─────────────────────────────────────────────────────────────┘
```

**Example Output**:
```
📦 Found Node.js service: web
Installing: ./src/web (pnpm)

pnpm install
Packages: +245
Progress: resolved 245, reused 203, downloaded 42, added 245, done
✓ Dependencies installed successfully
```

## Python Dependency Installation

### Package Manager Detection

```
┌─────────────────────────────────────────────────────────────┐
│  Check for Configuration Files in Service Directory          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detection Priority:                                         │
│  1. pyproject.toml + tool.uv → uv                            │
│  2. pyproject.toml + tool.poetry → poetry                    │
│  3. Pipfile → pipenv                                         │
│  4. requirements.txt → pip (default)                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────────┐
                    │                    │
                   uv                 poetry
                    │                    │
                    ↓                    ↓
        ┌──────────────────┐   ┌──────────────────┐
        │ uv Flow          │   │ poetry Flow      │
        └──────────────────┘   └──────────────────┘
                    │                    │
                    └────────┬───────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                  pipenv             pip
                    │                 │
                    ↓                 ↓
        ┌──────────────────┐ ┌──────────────────┐
        │ pipenv Flow      │ │ pip Flow         │
        └──────────────────┘ └──────────────────┘
```

### Virtual Environment Setup

```
┌─────────────────────────────────────────────────────────────┐
│  Check if Virtual Environment Exists                         │
│  - .venv/ or venv/ directory                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
                 Exists           Not Exists
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌──────────────────┐
            │ Use Existing│   │ Create New       │
            └─────────────┘   │  python -m venv  │
                    │         │  .venv           │
                    │         └──────────────────┘
                    │                │
                    └────────┬───────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│  Activate Virtual Environment                                │
│  - Windows: .venv\Scripts\activate                           │
│  - Unix: source .venv/bin/activate                           │
└─────────────────────────────────────────────────────────────┘
```

### Installation by Package Manager

#### UV Installation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  uv venv (create/verify virtual environment)                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  uv pip install -r requirements.txt                          │
│  (or uv sync if pyproject.toml with dependencies)            │
└─────────────────────────────────────────────────────────────┘
```

#### Poetry Installation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  poetry install                                              │
│  - Reads pyproject.toml                                      │
│  - Manages virtualenv automatically                          │
│  - Installs locked dependencies from poetry.lock             │
└─────────────────────────────────────────────────────────────┘
```

#### Pipenv Installation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  pipenv install                                              │
│  - Reads Pipfile                                             │
│  - Creates virtualenv automatically                          │
│  - Installs locked dependencies from Pipfile.lock            │
└─────────────────────────────────────────────────────────────┘
```

#### Pip Installation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Create/verify virtual environment                           │
│  python -m venv .venv                                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Activate virtual environment                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Install dependencies                                        │
│  pip install -r requirements.txt                             │
└─────────────────────────────────────────────────────────────┘
```

**Example Output**:
```
🐍 Found Python service: api
./src/api (uv)

Creating virtual environment...
uv venv
Using Python 3.12.0 interpreter at: /usr/bin/python3.12
Creating virtualenv at: .venv

Installing dependencies...
uv pip install -r requirements.txt
Resolved 45 packages in 234ms
Downloaded 45 packages in 1.23s
Installed 45 packages in 567ms
✓ Dependencies installed successfully
```

## .NET Dependency Installation

### Project Detection

```
┌─────────────────────────────────────────────────────────────┐
│  Scan Service Directory for .NET Projects                    │
│  - Find all *.csproj files                                   │
│  - Find all *.fsproj files                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  For Each Project File:                                      │
│  - Get project path                                          │
│  - Prepare for restore                                       │
└─────────────────────────────────────────────────────────────┘
```

### Restore Process

```
┌─────────────────────────────────────────────────────────────┐
│  dotnet restore <project-file>                               │
│  - Restore NuGet packages                                    │
│  - Resolve project dependencies                              │
│  - Download missing packages                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Verify Restoration Success                                  │
│  - Check exit code                                           │
│  - Verify obj/ directory created                             │
│  - Verify project.assets.json exists                         │
└─────────────────────────────────────────────────────────────┘
```

**Example Output**:
```
🔷 Found .NET service: apphost
./src/apphost

Restoring: ./src/apphost/AppHost.csproj
  Determining projects to restore...
  Restored ./src/apphost/AppHost.csproj (in 2.3 sec).
✓ Dependencies restored successfully
```

## Command Dependency Chain

The `deps` command is part of the orchestrated command chain:

```
┌─────────────────────────────────────────────────────────────┐
│                      User runs:                              │
│                   azd app deps                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Orchestrator Executes Dependencies:                         │
│                                                              │
│  1. reqs (check prerequisites)                               │
│     └─ Exit if prerequisites not met                         │
│                                                              │
│  2. deps (this command)                                      │
│     └─ Install dependencies                                  │
└─────────────────────────────────────────────────────────────┘
```

This ensures that:
- Required tools are installed before trying to use them
- Users don't need to manually run `reqs` first
- The dependency chain is automatic and transparent

## Multi-Service Handling

When an `azure.yaml` defines multiple services:

```yaml
name: my-app
services:
  web:
    language: js
    project: ./src/web
  api:
    language: python
    project: ./src/api
  apphost:
    language: csharp
    project: ./src/apphost
```

The installation process:

```
┌─────────────────────────────────────────────────────────────┐
│  Process services sequentially                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Service: web (Node.js)                                      │
│  1. Detect: pnpm                                             │
│  2. Execute: pnpm install                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Service: api (Python)                                       │
│  1. Detect: uv                                               │
│  2. Create: virtual environment                              │
│  3. Execute: uv pip install -r requirements.txt              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Service: apphost (.NET)                                     │
│  1. Find: AppHost.csproj                                     │
│  2. Execute: dotnet restore AppHost.csproj                   │
└─────────────────────────────────────────────────────────────┘
```

## Error Handling

### Error Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Execute Installation Command                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              Exit Code = 0      Exit Code ≠ 0
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌─────────────────┐
            │ SUCCESS     │   │ FAILURE         │
            │ Continue    │   │ Stop & Report   │
            └─────────────┘   └─────────────────┘
```

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| Tool not found | Package manager not installed | Run `azd app reqs` to check |
| Lock file mismatch | Different PM used previously | Delete lock file and node_modules |
| Network timeout | Package registry unreachable | Check network, retry |
| Permission denied | Insufficient permissions | Run with appropriate permissions |
| Disk full | No space for packages | Free up disk space |

**Example Error Output**:
```
📦 Found Node.js service: web
Installing: ./src/web (pnpm)

✗ Failed to install dependencies
Error: pnpm: command not found

Run 'azd app reqs' to check prerequisites
```

## Configuration via azure.yaml

### Service Configuration

```yaml
name: my-app

services:
  # Node.js service
  frontend:
    language: js
    host: containerapp
    project: ./src/frontend
    # Deps will auto-detect pnpm/npm/yarn
  
  # Python service
  backend:
    language: python
    host: containerapp
    project: ./src/backend
    # Deps will auto-detect uv/poetry/pip
  
  # .NET service
  api:
    language: csharp
    host: containerapp
    project: ./src/api
    # Deps will run dotnet restore
```

### Language Values

Supported `language` values:

| Value | Meaning |
|-------|---------|
| `js` | JavaScript/TypeScript (Node.js) |
| `python` | Python |
| `csharp` | C# (.NET) |
| `dotnet` | .NET (any language) |
| `fsharp` | F# (.NET) |

## Output Formats

### Text Output (Default)

```
✓ Prerequisites check passed

📦 Installing dependencies for 3 services

📦 Found Node.js service: web
Installing: ./src/web (pnpm)
✓ Dependencies installed successfully

🐍 Found Python service: api
./src/api (uv)
✓ Dependencies installed successfully

🔷 Found .NET service: apphost
./src/apphost
✓ Dependencies restored successfully

✓ All dependencies installed successfully
```

### JSON Output (`--output json`)

```json
{
  "success": true,
  "projects": [
    {
      "type": "node",
      "dir": "./src/web",
      "manager": "pnpm",
      "success": true
    },
    {
      "type": "python",
      "dir": "./src/api",
      "manager": "uv",
      "success": true
    },
    {
      "type": "dotnet",
      "dir": "./src/apphost",
      "success": true
    }
  ]
}
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | All dependencies installed |
| 1 | Failure | One or more installations failed |

## Best Practices

1. **Lock Files**: Commit lock files to version control for reproducible builds
2. **Virtual Environments**: Don't commit `.venv/` or `node_modules/` directories
3. **Cache**: Use CI caching for faster builds (cache `node_modules/`, `.venv/`)
4. **Prerequisites**: Ensure `azd app reqs` passes before running deps
5. **Clean Install**: Delete dependency directories for clean reinstall

## Performance Considerations

### Parallel Installation

Currently, services are processed **sequentially**. This is safe but slower for multi-service projects.

Future enhancement:
```
Sequential (current):     Parallel (future):
web     (10s)             web     (10s)
api     (15s)             api     (15s)  ← overlap
apphost (5s)              apphost (5s)   ← overlap
Total: 30s                Total: ~15s
```

### Caching Strategies

For faster dependency installation:

| Package Manager | Cache Location | CI Cache |
|-----------------|----------------|----------|
| pnpm | `~/.pnpm-store` | Cache this directory |
| npm | `~/.npm` | Cache this directory |
| yarn | `~/.yarn/cache` | Cache this directory |
| pip | `~/.cache/pip` | Cache this directory |
| uv | `~/.cache/uv` | Cache this directory |
| poetry | `~/.cache/pypoetry` | Cache this directory |
| NuGet | `~/.nuget/packages` | Cache this directory |

## Troubleshooting

### Issue: "No projects found"

**Cause**: `azure.yaml` has no services defined

**Solution**:
```yaml
# Add services to azure.yaml
services:
  web:
    language: js
    project: ./src/web
```

### Issue: Wrong package manager detected

**Cause**: Multiple lock files exist

**Solution**:
```bash
# Clean up conflicting lock files
cd src/web
rm package-lock.json  # If using pnpm
# Keep only one lock file type
```

### Issue: Python virtual environment issues

**Cause**: Corrupted or incompatible venv

**Solution**:
```bash
# Delete and recreate
rm -rf .venv
azd app deps  # Will create fresh venv
```

### Issue: .NET restore fails

**Cause**: NuGet sources unreachable

**Solution**:
```bash
# Check NuGet sources
dotnet nuget list source

# Clear NuGet cache
dotnet nuget locals all --clear
```

## Common Use Cases

### 1. First-Time Setup

```bash
# Clone repository
git clone https://github.com/myorg/myapp
cd myapp

# Install dependencies
azd app deps
```

### 2. After Pulling Changes

```bash
# Pull latest code
git pull origin main

# Update dependencies
azd app deps
```

### 3. Clean Reinstall

```bash
# Node.js
rm -rf node_modules package-lock.json
azd app deps

# Python
rm -rf .venv
azd app deps

# .NET
dotnet clean
azd app deps
```

### 4. CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Install dependencies
  run: azd app deps
  env:
    CI: true
```

## Integration with Other Commands

### Dependency Graph

```
azd app run
     ↓
azd app deps  ← You are here
     ↓
azd app reqs
```

When you run:
- `azd app deps` → Automatically runs `reqs` first
- `azd app run` → Automatically runs `deps` (which runs `reqs`)

### Manual vs Automatic

You can run `deps` manually or let other commands run it automatically:

```bash
# Manual
azd app reqs
azd app deps
azd app run

# Automatic (recommended)
azd app run  # Runs reqs → deps → run
```

## Related Commands

- [`azd app reqs`](./reqs.md) - Check prerequisites (runs before deps)
- [`azd app run`](./run.md) - Run services (runs deps automatically)

## Examples

### Example 1: Full Stack App

```yaml
# azure.yaml
name: fullstack-app
services:
  web:
    language: js
    project: ./frontend
  api:
    language: python
    project: ./backend
  graphql:
    language: csharp
    project: ./graphql
```

```bash
$ azd app deps

✓ Prerequisites check passed

📦 Installing dependencies for 3 services

📦 Found Node.js service: web
Installing: ./frontend (pnpm)
Packages: +342
✓ Dependencies installed successfully

🐍 Found Python service: api
./backend (uv)
Installed 67 packages in 1.2s
✓ Dependencies installed successfully

🔷 Found .NET service: graphql
./graphql
Restored ./graphql/GraphQL.csproj (in 3.1 sec)
✓ Dependencies restored successfully

✓ All dependencies installed successfully
```

### Example 2: Python with Poetry

Project structure:
```
myapp/
  src/
    api/
      pyproject.toml
      poetry.lock
      app/
        main.py
```

```bash
$ azd app deps

🐍 Found Python service: api
./src/api (poetry)

poetry install
Installing dependencies from lock file
Package operations: 45 installs, 0 updates, 0 removals
  • Installing certifi (2024.2.2)
  • Installing charset-normalizer (3.3.2)
  ...
✓ Dependencies installed successfully
```

### Example 3: Monorepo with Multiple Services

```yaml
# azure.yaml
name: monorepo
services:
  admin:
    language: js
    project: ./apps/admin
  customer:
    language: js
    project: ./apps/customer
  shared-api:
    language: python
    project: ./services/api
```

```bash
$ azd app deps

📦 Installing dependencies for 3 services

📦 Found Node.js service: admin
Installing: ./apps/admin (pnpm)
✓ Dependencies installed successfully

📦 Found Node.js service: customer
Installing: ./apps/customer (pnpm)
✓ Dependencies installed successfully

🐍 Found Python service: shared-api
./services/api (pip)
✓ Dependencies installed successfully
```
