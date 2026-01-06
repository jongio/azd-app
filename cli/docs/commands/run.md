# azd app run

## Overview

The `run` command starts your development environment by orchestrating services defined in `azure.yaml`, or by auto-detecting and running .NET Aspire, pnpm, or docker compose projects. It provides two runtime modes: **azd** (default) with the azd dashboard, or **aspire** for native .NET Aspire experience.

## Purpose

- **Service Orchestration**: Start multiple services in correct dependency order
- **Lifecycle Hooks**: Execute prerun/postrun scripts for setup and notifications
- **Multi-Runtime Support**: Support Node.js, Python, .NET, Azure Functions, Logic Apps, and container-based services
- **Development Dashboard**: Provide real-time service monitoring and log viewing
- **Port Management**: Automatically assign and manage service ports
- **Environment Variables**: Inject Azure and service-specific environment variables
- **Signal Handling**: Gracefully shutdown all services on Ctrl+C
- **Service Registry**: Track running services for inter-command communication

## Command Usage

```bash
azd app run [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Run specific service(s) only (comma-separated) |
| `--runtime` | | string | `azd` | Runtime mode: 'azd' or 'aspire' |
| `--env-file` | | string | | Load environment variables from .env file |
| `--verbose` | `-v` | bool | `false` | Enable verbose logging |
| `--dry-run` | | bool | `false` | Show execution plan without starting services |
| `--restart-containers` | | bool | `false` | Restart containers even if they are already running |
| `--force` | | bool | `false` | Force clean dependency reinstall (passes --force to deps) |
| `--web` | `-w` | bool | `false` | Open dashboard in browser |

## Dashboard Browser Launch

By default, the dashboard URL is displayed but the browser is not opened automatically. Use the `--web` flag to open the dashboard in your system's default browser.

### Command-Line Examples

```bash
# Start services without opening browser (default)
azd app run

# Open dashboard in system browser
azd app run --web
azd app run -w
```

**Timing**: Browser launches immediately after dashboard server is ready and displays:
```
  Dashboard  http://localhost:4280
  Opening in System Default Browser...
```

**Errors**: If browser launch fails, a warning is displayed but the dashboard continues running:
```
⚠️  Could not open browser automatically. Dashboard available at: http://localhost:4280
```

**Non-blocking**: Browser launch happens asynchronously and never blocks dashboard startup.

## Lifecycle Hooks

The `run` command supports **prerun** and **postrun** hooks that execute automatically before and after service orchestration. These are similar to azd's `preprovision` and `postprovision` hooks.

### Hook Types

#### `prerun` Hook
Executes **before** stardefault bs. Common uses:
- Database migrations
- Environment validation
- Dependency checks
- Setting up test data
- Pre-flight checks

#### `postrun` Hook
Executes **after** all services are ready and running. Common uses:
- Notifications (Slack, email, etc.)
- Opening browser windows
- Running integration tests
- Logging startup information
- Service discovery registration

### Hook Configuration

Hooks are configured in the `hooks` section of `azure.yaml`:

```yaml
name: my-app

hooks:
  prerun:
    run: ./scripts/setup.sh
    shell: bash
    continueOnError: false
  postrun:
    run: echo "All services ready!"
    shell: sh

services:
  web:
    language: js
    project: ./frontend
```

### Hook Properties

| Property | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `run` | string | ✅ | | Script or command to execute |
| `shell` | string | ❌ | Platform default | Shell to use (sh, bash, pwsh, powershell, cmd) |
| `continueOnError` | boolean | ❌ | `false` | Continue if hook fails |
| `interactive` | boolean | ❌ | `false` | Allow user interaction |
| `windows` | object | ❌ | | Windows-specific override |
| `posix` | object | ❌ | | POSIX-specific override |

### Platform-Specific Hooks

Different scripts can run on Windows vs POSIX (Linux/macOS):

```yaml
hooks:
  prerun:
    windows:
      run: .\scripts\setup.ps1
      shell: pwsh
    posix:
      run: ./scripts/setup.sh
      shell: bash
```

### Hook Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Parse azure.yaml                                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Prerun Hook (if configured)                         │
│  - Run before starting any services                          │
│  - Stop if hook fails (unless continueOnError=true)          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Start Services                                              │
│  - Orchestrate services in parallel                          │
│  - Wait for all services to be ready                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Postrun Hook (if configured)                        │
│  - Run after all services are ready                          │
│  - Warning if fails (services continue running)              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Display Dashboard & Monitor Services                        │
└─────────────────────────────────────────────────────────────┘
```

### Hook Examples

**Database Migration:**
```yaml
hooks:
  prerun:
    run: npm run db:migrate
    shell: bash
```

**Multi-Step Setup:**
```yaml
hooks:
  prerun:
    run: |
      echo "Validating environment..."
      npm run validate
      npm run db:migrate
      npm run seed
    shell: bash
```

**Post-Startup Notification:**
```yaml
hooks:
  postrun:
    run: |
      curl -X POST $SLACK_WEBHOOK \
        -d '{"text":"Dev environment ready!"}'
    shell: bash
    continueOnError: true
```

**Cross-Platform Setup:**
```yaml
hooks:
  prerun:
    windows:
      run: |
        Write-Host "Windows setup"
        & .\scripts\setup.ps1
      shell: pwsh
    posix:
      run: |
        echo "POSIX setup"
        ./scripts/setup.sh
      shell: bash
```

For complete hook documentation, see [`hooks.md`](../hooks.md).

## Execution Flow

### Overall Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app run                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Validate Runtime Mode                                       │
│  - Must be 'azd' or 'aspire'                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Dependency Chain                                    │
│  1. reqs (check prerequisites)                               │
│  2. deps (install dependencies)                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Find azure.yaml                                             │
│  - Search current directory and parents                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              runtime=azd       runtime=aspire
                    │                │
                    ↓                ↓
        ┌────────────────────┐ ┌──────────────────┐
        │  AZD Mode          │ │ Aspire Mode      │
        │  (Orchestration)   │ │ (dotnet run)     │
        └────────────────────┘ └──────────────────┘
```

### AZD Mode Flow (Default)

```
┌─────────────────────────────────────────────────────────────┐
│  Parse azure.yaml                                            │
│  - Read services section                                     │
│  - Extract service configurations                            │
│  - Read hooks section                                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Prerun Hook (if configured)                         │
│  - Setup, migrations, validation                             │
│  - FAIL if error and continueOnError=false                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check for Services                                          │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
            No Services         Has Services
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────────┐
        │ Show Message     │ │ Continue        │
        │ Exit Gracefully  │ │                 │
        └──────────────────┘ └─────────────────┘
                                      ↓
┌─────────────────────────────────────────────────────────────┐
│  Apply Service Filter (if --service specified)               │
│  - Keep only requested services                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detect Service Runtimes                                     │
│  - Determine language/framework                              │
│  - Assign ports (avoid conflicts)                            │
│  - Build command and arguments                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
               --dry-run?           No
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────────────┐
        │ Show Execution   │ │ Execute Services    │
        │ Plan & Exit      │ │                     │
        └──────────────────┘ └─────────────────────┘
                                      ↓
┌─────────────────────────────────────────────────────────────┐
│  Load Environment Variables                                  │
│  - Azure environment (from azd)                              │
│  - Custom .env file (if --env-file)                          │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Orchestrate Services                                        │
│  - Start services in parallel                                │
│  - Register in service registry                              │
│  - Collect logs                                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Validate Orchestration                                      │
│  - Check all services started successfully                   │
│  - Verify all services are ready                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute Postrun Hook (if configured)                        │
│  - Notifications, tests, registration                        │
│  - WARN if error (services continue running)                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Start Dashboard Server                                      │
│  - Launch web-based dashboard                                │
│  - Display dashboard URL                                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Wait for Shutdown Signal (Ctrl+C)                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Graceful Shutdown                                           │
│  1. Stop dashboard                                           │
│  2. Stop all services                                        │
│  3. Unregister from registry                                 │
└─────────────────────────────────────────────────────────────┘
```

### Aspire Mode Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Find Aspire AppHost Project                                 │
│  - Search for AppHost.cs or Program.cs                       │
│  - In a .csproj project                                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              Found AppHost       Not Found
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────────┐
        │ Continue         │ │ Error & Exit    │
        └──────────────────┘ └─────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────────┐
│  Execute: dotnet run --project <AppHost.csproj>              │
│  - Inherits all azd environment variables                    │
│  - Aspire dashboard starts automatically                     │
│  - User presses Ctrl+C to stop                               │
└─────────────────────────────────────────────────────────────┘
```

## Service Orchestration Details

### Service Runtime Detection

For each service in `azure.yaml`, the runtime detection process:

```
┌─────────────────────────────────────────────────────────────┐
│  Read Service Configuration                                  │
│  - name: Service identifier                                  │
│  - language: js/python/csharp/dotnet                         │
│  - project: Directory path                                   │
│  - host: Deployment target (containerapp, etc.)              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detect Language/Framework                                   │
│  - Check language field in azure.yaml                        │
│  - Scan project directory for markers                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Determine Execution Strategy                                │
│                                                              │
│  Azure Functions (host: function)                            │
│    → Detect variant (Logic Apps, Node.js, Python, .NET,     │
│       Java)                                                  │
│    → Assign port (default 7071)                             │
│    → Run with: func start --port <port>                     │
│                                                              │
│  Node.js (language: js)                                      │
│    → Check package.json for dev/start script                │
│    → Use detected package manager (pnpm/npm/yarn)            │
│                                                              │
│  Python (language: python)                                   │
│    → Look for main.py, app.py, manage.py                    │
│    → Activate virtual environment if exists                  │
│    → Run with appropriate command                            │
│                                                              │
│  .NET (language: csharp/dotnet)                              │
│    → Find .csproj file                                       │
│    → Run with dotnet run                                     │
│                                                              │
│  Aspire (detected AppHost)                                   │
│    → Run AppHost.csproj with dotnet run                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Assign Port                                                 │
│  - Track used ports to avoid conflicts                       │
│  - Assign next available port (starting from 3000)           │
│  - Respect explicit port configurations                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Build ServiceRuntime                                        │
│  {                                                           │
│    Name:       "web",                                        │
│    Language:   "js",                                         │
│    Framework:  "pnpm",                                       │
│    Port:       3000,                                         │
│    Command:    "pnpm",                                       │
│    Args:       ["run", "dev"],                               │
│    WorkingDir: "/path/to/project",                           │
│    Env:        map[string]string{...}                        │
│  }                                                           │
└─────────────────────────────────────────────────────────────┘
```

### Parallel Service Startup

Services start **in parallel** for faster development environment initialization:

```
Time →
0s        2s        4s        6s        8s
│         │         │         │         │
├─ web ──┴─────────┐
│                  ✓ Ready
├─ api ──┴─────────────┐
│                      ✓ Ready
├─ db ───┴──┐
│           ✓ Ready
└─ cache ─┴──┐
             ✓ Ready

Sequential: 8s total
Parallel:   4s total (fastest service)
```

**Orchestration Process**:

1. **Start All Services**: Launch in parallel goroutines
2. **Register in Registry**: Track service metadata and status
3. **Collect Logs**: Capture stdout/stderr in real-time
4. **Monitor Health**: Update service status (starting → running)
5. **Report URLs**: Display access URLs as services become ready

### Service Registry

Each running service is registered with metadata:

```go
ServiceRegistryEntry {
    Name:       "web"
    ProjectDir: "/path/to/project"
    Port:       3000
    URL:        "http://localhost:3000"
    AzureURL:   "https://web-xyz.azurewebsites.net"
    Language:   "js"
    Framework:  "pnpm"
    Status:     "running"
    Health:     "healthy"
    PID:        12345
    StartTime:  "2024-11-04T10:30:00Z"
}
```

**Registry Location**: `.azure/registry/services.json`

**Registry Benefits**:
- Other commands can query running services
- Service discovery for inter-service communication
- Status tracking across command invocations
- Clean shutdown and cleanup

### Port Management

```
┌─────────────────────────────────────────────────────────────┐
│  Port Assignment Strategy                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Start at Default Port: 3000                                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  For Each Service:                                           │
│    1. Check if port is in use                                │
│    2. If used, increment and retry                           │
│    3. Assign port and mark as used                           │
│    4. Continue to next service                               │
└─────────────────────────────────────────────────────────────┘

Example:
  web     → 3000 (first service)
  api     → 3001 (3000 used)
  worker  → 3002 (3000, 3001 used)
```

**Explicit Port Configuration** (future enhancement):
```yaml
services:
  web:
    language: js
    project: ./web
    port: 8080  # Explicit port assignment
```

### Environment Variable Injection

Services receive environment variables from multiple sources:

```
┌─────────────────────────────────────────────────────────────┐
│  Environment Variable Sources (in order)                     │
└─────────────────────────────────────────────────────────────┘

1. Azure Environment (from azd context)
   ├─ AZURE_SUBSCRIPTION_ID
   ├─ AZURE_RESOURCE_GROUP_NAME
   ├─ AZURE_ENV_NAME
   ├─ AZURE_LOCATION
   └─ SERVICE_*_URL (for each deployed service)

2. Custom .env File (if --env-file specified)
   ├─ DATABASE_URL=postgresql://...
   ├─ API_KEY=xyz123
   └─ LOG_LEVEL=debug

3. Service-Specific Variables
   ├─ PORT=3000
   └─ NODE_ENV=development

4. Runtime-Specific Variables
   ├─ ASPNETCORE_ENVIRONMENT=Development
   └─ PYTHONUNBUFFERED=1
```

**Merge Strategy**: Later sources override earlier ones

**Example**:
```bash
# Azure env provides:
AZURE_SUBSCRIPTION_ID=abc123
SERVICE_WEB_URL=https://web.azurewebsites.net

# .env file provides:
DATABASE_URL=postgresql://localhost:5432/db
LOG_LEVEL=debug

# Service receives all:
AZURE_SUBSCRIPTION_ID=abc123
SERVICE_WEB_URL=https://web.azurewebsites.net
DATABASE_URL=postgresql://localhost:5432/db
LOG_LEVEL=debug
PORT=3000
```

## Runtime Modes

### AZD Mode (Default)

**Use When**:
- Working with multi-language projects
- Need unified dashboard across all services
- Want consistent experience regardless of stack
- Azure-first development workflow

**Features**:
- ✅ Custom azd dashboard
- ✅ Works with any project type
- ✅ Service orchestration and monitoring
- ✅ Integrated log viewing
- ✅ Service registry tracking

**Dashboard**:
```
┌────────────────────────────────────────────────────┐
│  AZD Dashboard                                     │
│  http://localhost:4280                             │
├────────────────────────────────────────────────────┤
│                                                    │
│  Services:                                         │
│  ┌──────────────────────────────────────────────┐ │
│  │  web         Running  http://localhost:3000  │ │
│  │  api         Running  http://localhost:3001  │ │
│  │  worker      Running  http://localhost:3002  │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  Logs: [Filter ▼] [Follow ✓]                      │
│  ┌──────────────────────────────────────────────┐ │
│  │ [web] Server started on port 3000            │ │
│  │ [api] Connected to database                  │ │
│  │ [worker] Polling for jobs...                 │ │
│  └──────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────┘
```

### Aspire Mode

**Use When**:
- Working exclusively with .NET Aspire projects
- Need full Aspire tooling integration
- Want native Aspire dashboard features
- Using Aspire-specific components

**Features**:
- ✅ Native .NET Aspire dashboard
- ✅ Full Aspire tooling support
- ✅ Aspire component integration
- ✅ OpenTelemetry out of the box
- ✅ Structured logging and tracing

**Requirements**:
- Must have AppHost.cs or Program.cs
- Must be in a .csproj project
- .NET Aspire SDK installed

**Command**:
```bash
azd app run --runtime aspire
```

**What Happens**:
```bash
# Behind the scenes:
cd /path/to/AppHost
dotnet run --project AppHost.csproj

# Aspire dashboard launches automatically
# All azd environment variables are inherited
```

**Aspire Dashboard**:
```
┌────────────────────────────────────────────────────┐
│  .NET Aspire Dashboard                             │
│  https://localhost:15888                           │
├────────────────────────────────────────────────────┤
│                                                    │
│  Resources:                                        │
│  ┌──────────────────────────────────────────────┐ │
│  │  webfrontend   Running  Logs  Traces         │ │
│  │  apiservice    Running  Logs  Traces         │ │
│  │  postgres      Running  Logs  Metrics        │ │
│  └──────────────────────────────────────────────┘ │
│                                                    │
│  Structured Logs  |  Traces  |  Metrics           │
│  ┌──────────────────────────────────────────────┐ │
│  │ Trace: GET /api/users                        │ │
│  │   → webfrontend (2.3ms)                      │ │
│  │   → apiservice  (15.7ms)                     │ │
│  │     → postgres  (12.1ms)                     │ │
│  └──────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────┘
```

### Mode Comparison

| Feature | AZD Mode | Aspire Mode |
|---------|----------|-------------|
| **Multi-language** | ✅ Yes | ❌ .NET only |
| **Dashboard** | Custom azd | Native Aspire |
| **Orchestration** | azd controls | Aspire controls |
| **Logs** | Text streams | Structured |
| **Tracing** | ❌ No | ✅ OpenTelemetry |
| **Metrics** | ❌ No | ✅ Built-in |
| **Setup** | Any project | Aspire project required |

## Dashboard

### AZD Dashboard Features

**Service Overview**:
- List of all running services
- Status indicators (starting/running/error)
- URLs with clickable links
- Language and framework info

**Log Viewer**:
- Real-time log streaming
- Filter by service
- Follow mode (auto-scroll)
- Search and highlighting

**Service Control**:
- Stop individual services
- Restart services
- View service details

**Access**:
```bash
$ azd app run

📊 Dashboard: http://localhost:4280

# Open in browser to view
```

## Service Filtering

Run specific services only using `--service`:

```bash
# Run single service
azd app run --service web

# Run multiple services
azd app run --service web,api

# Useful for:
# - Testing individual services
# - Reducing resource usage
# - Debugging specific components
```

**Filter Flow**:
```
azure.yaml services:         --service web,api          Result:
- web                       ───────────────────────►    - web
- api                                                   - api
- worker
- cache
```

## Dry-Run Mode

Preview what would be executed without starting services:

```bash
$ azd app run --dry-run

🔍 Dry-run mode: Showing execution plan

web
  Language: js
  Framework: pnpm
  Port: 3000
  Directory: ./src/web
  Command: pnpm run dev

api
  Language: python
  Framework: uv
  Port: 3001
  Directory: ./src/api
  Command: uv run uvicorn app.main:app --reload --port 3001

apphost
  Language: csharp
  Framework: aspire
  Port: 3002
  Directory: ./src/apphost
  Command: dotnet run --project AppHost.csproj
```

**Use Cases**:
- Verify service detection
- Check port assignments
- Validate commands before execution
- Debug configuration issues

## Graceful Shutdown

When you press Ctrl+C:

```
User presses Ctrl+C
         ↓
┌─────────────────────────────────────────┐
│  Intercept SIGINT/SIGTERM               │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│  Display Shutdown Message               │
│  "🛑 Shutting down services..."         │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│  Stop Dashboard Server                  │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│  Stop Services in Parallel              │
│  - Send SIGTERM to each process         │
│  - Wait for graceful shutdown           │
│  - Force kill if timeout (10s)          │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│  Unregister from Service Registry       │
│  - Remove service entries               │
│  - Clean up registry file               │
└─────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────┐
│  Display Success Message                │
│  "All services stopped"                 │
└─────────────────────────────────────────┘
```

## Command Dependency Chain

```
User runs: azd app run
         ↓
┌────────────────────────────────┐
│  Orchestrator Auto-Executes:   │
│                                │
│  1. reqs                       │
│     └─ Check prerequisites     │
│        Exit if not satisfied   │
│                                │
│  2. deps                       │
│     └─ Install dependencies    │
│        Exit if failed          │
│                                │
│  3. run                        │
│     └─ Start services          │
└────────────────────────────────┘
```

This ensures:
- Prerequisites are validated before running
- Dependencies are installed before starting services
- Users don't need to manually run multiple commands

## Configuration

### azure.yaml Structure

```yaml
name: my-app

# Services to run in development
services:
  # Frontend service
  web:
    language: js
    host: local
    project: ./src/web
  
  # Backend API
  api:
    language: python
    host: local
    project: ./src/api
  
  # .NET Aspire AppHost
  apphost:
    language: csharp
    host: local
    project: ./src/apphost

# Resources (databases, caches, etc.)
resources:
  database:
    type: postgres.database
    uses:
      - name: postgres
        type: postgres.server
  
  cache:
    type: redis.database
    uses:
      - name: redis
        type: redis.server
```

### Service Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `language` | string | ✅* | Project language (js/python/csharp/dotnet) |
| `host` | string | ✅ | Deployment target (containerapp, etc.) |
| `project` | string | ✅* | Relative path to project directory |
| `image` | string | ❌ | Docker image for container services |
| `ports` | []string | ❌ | Port mappings (e.g., "3000:3000") |
| `environment` | map | ❌ | Environment variables for the service |

*Required for application services, not required for container services.

### Container Services

Container services are defined using the `image` field instead of `language` and `project`. They are started as Docker containers alongside your application services.

```yaml
services:
  # Application service
  api:
    language: python
    project: ./backend
  
  # Container service
  azurite:
    host: local
    image: mcr.microsoft.com/azure-storage/azurite:latest
    ports:
      - "10000:10000"
      - "10001:10001"
      - "10002:10002"
```

Container services require Docker to be installed and running. The `azd app reqs` command will automatically check for Docker when container services are defined.

Use `azd app add` to easily add well-known container services like Azurite, Cosmos DB emulator, Redis, or PostgreSQL.

### Environment File Format

`.env` file (used with `--env-file`):

```bash
# Database configuration
DATABASE_URL=postgresql://localhost:5432/mydb
DATABASE_POOL_SIZE=20

# API keys
OPENAI_API_KEY=sk-xyz123
STRIPE_API_KEY=sk_test_abc456

# Application settings
LOG_LEVEL=debug
ENABLE_METRICS=true
```

## Output Examples

### Successful Startup

```bash
$ azd app run

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

web              → http://localhost:3000
api              → http://localhost:3001
worker           → http://localhost:3002

📊 Dashboard: http://localhost:4280

💡 Press Ctrl+C to stop all services
```

### With Service Filter

```bash
$ azd app run --service web,api

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting 2 services (filtered)

web              → http://localhost:3000
api              → http://localhost:3001

📊 Dashboard: http://localhost:4280

💡 Press Ctrl+C to stop all services
```

### Aspire Mode

```bash
$ azd app run --runtime aspire

🚀 Running Aspire in native mode
  Directory: ./src/apphost
  Project: AppHost.csproj

💡 Aspire dashboard will start automatically
💡 All azd environment variables are available to your app

💡 Press Ctrl+C to stop

Building...
  AppHost -> ./src/apphost/bin/Debug/net8.0/AppHost.dll
info: Aspire.Hosting.DistributedApplication[0]
      Aspire version: 8.0.0
info: Aspire.Hosting.DistributedApplication[0]
      Distributed application starting.
info: Aspire.Hosting.DistributedApplication[0]
      Dashboard: https://localhost:15888
```

### Shutdown

```bash
^C

🛑 Shutting down services...
✓ All services stopped
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Services ran and shutdown gracefully |
| 1 | Failure | Service startup failed or runtime error |
| 2 | Validation | Invalid flags or configuration |

## Common Use Cases

### 1. Daily Development

```bash
# Start all services
azd app run

# Services run until Ctrl+C
# Dashboard available at http://localhost:4280
```

### 2. Work on Specific Service

```bash
# Run only the service you're working on
azd app run --service web

# Faster startup, fewer resources
```

### 3. Debug Startup Issues

```bash
# Use verbose logging
azd app run --verbose

# Or preview execution plan
azd app run --dry-run
```

### 4. Custom Environment

```bash
# Load custom environment variables
azd app run --env-file .env.local

# Useful for local database URLs, API keys, etc.
```

### 5. Aspire Development

```bash
# Use native Aspire experience
azd app run --runtime aspire

# Get Aspire dashboard, tracing, metrics
```

### 6. Force Clean Dependency Reinstall

```bash
# Force clean dependency reinstall before running
azd app run --force

# Combine with other flags
azd app run --force --web
```

## Troubleshooting

### Issue: "No services defined in azure.yaml"

**Cause**: `azure.yaml` has no `services` section

**Solution**:
```yaml
# Add services to azure.yaml
services:
  web:
    language: js
    project: ./src/web
```

### Issue: Service fails to start

**Check Prerequisites**:
```bash
azd app reqs --no-cache
```

**Check Dependencies**:
```bash
azd app deps
```

**Use Verbose Logging**:
```bash
azd app run --verbose
```

### Issue: Port already in use

**Cause**: Another process using the assigned port

**Solution**:
```bash
# Find process using port
netstat -ano | findstr :3000  # Windows
lsof -i :3000                 # macOS/Linux

# Kill process or change port
```

### Issue: Environment variables not available

**Check azd Context**:
```bash
azd env get-values
```

**Verify .env File**:
```bash
cat .env.local
azd app run --env-file .env.local --verbose
```

### Issue: Aspire mode fails

**Check for AppHost**:
```bash
# Ensure AppHost.cs exists
ls ./src/apphost/AppHost.cs

# Or Program.cs with Aspire
ls ./src/apphost/Program.cs
```

**Verify Aspire SDK**:
```bash
dotnet workload list
# Should show: aspire
```

## Best Practices

1. **Use azd Mode by Default**: Works with all project types
2. **Filter Services**: Use `--service` to run only what you need
3. **Environment Files**: Keep local config in `.env.local` (gitignored)
4. **Aspire for .NET**: Use `--runtime aspire` for Aspire projects
5. **Check Prerequisites**: Run `azd app reqs` if services fail to start
6. **Monitor Dashboard**: Use dashboard for real-time service monitoring
7. **Graceful Shutdown**: Always use Ctrl+C to stop services cleanly

## Security Considerations

### Trust Model

The `run` command executes commands defined in `azure.yaml` with your user permissions. When you run `azd app run`, you implicitly trust:

- **Hook scripts** (prerun/postrun)
- **Service commands** (command, entrypoint)
- **Framework-detected commands** (npm run dev, python app.py, etc.)

**This follows the same trust model as:**
- npm scripts (package.json)
- Makefile targets  
- docker-compose.yml commands
- Azure Developer CLI (azd) hooks

### Security Guidance

1. **Review azure.yaml before running**: Especially in cloned/downloaded projects
2. **Inspect hook scripts**: Check what prerun/postrun scripts do
3. **Treat azure.yaml like code**: It defines what commands execute
4. **Use version control**: Track changes to commands and hooks

### What the Run Command Can Execute

- **Hooks**: Arbitrary shell commands via prerun/postrun
- **Service commands**: Any command specified via `command` or `entrypoint`
- **Detected commands**: Package manager scripts, framework CLIs

### Recommended Practices

- **Audit third-party templates**: Review azure.yaml before running
- **Use explicit commands**: Prefer `command:` over auto-detection for clarity
- **Don't run as root/admin**: Use least privilege principle

## Related Commands

- [`azd app reqs`](./reqs.md) - Check prerequisites (runs automatically)
- [`azd app deps`](./deps.md) - Install dependencies (runs automatically)
- [`azd app logs`](./logs.md) - View service logs
- [`azd app info`](./info.md) - Show running service information

## Related Documentation

- [Lifecycle Hooks](../hooks.md) - Complete hook configuration guide
- [Azure Functions Support](../features/azure-functions.md) - Detailed Azure Functions documentation

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
```

```bash
$ azd app run

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

web              → http://localhost:3000
api              → http://localhost:3001

📊 Dashboard: http://localhost:4280
```

### Example 2: Azure Functions Multi-Language App

```yaml
# azure.yaml
name: functions-app
services:
  workflows:
    project: ./logicapp
    host: function
  api:
    project: ./functions-python
    host: function
  processor:
    project: ./functions-dotnet
    host: function
```

```bash
$ azd app run

✓ Prerequisites check passed
✓ Dependencies installed

🚀 Starting services

workflows        → http://localhost:7071  [Logic Apps Standard]
api              → http://localhost:7072  [Azure Functions (Python)]
processor        → http://localhost:7073  [Azure Functions (.NET)]

📊 Dashboard: http://localhost:4280
```

### Example 3: Aspire Microservices

```bash
$ azd app run --runtime aspire

🚀 Running Aspire in native mode
  Directory: ./src/AppHost
  Project: AppHost.csproj

Dashboard: https://localhost:15888

Resources:
  webfrontend   → https://localhost:7001
  apiservice    → https://localhost:7002
  postgres      → localhost:5432
  redis         → localhost:6379
```

### Example 4: Development with Custom Config

```bash
# .env.local
DATABASE_URL=postgresql://localhost:5432/dev_db
API_KEY=dev_key_123
LOG_LEVEL=debug

$ azd app run --env-file .env.local --service api --verbose

Loading environment from: .env.local
  DATABASE_URL=postgresql://localhost:5432/dev_db
  API_KEY=***
  LOG_LEVEL=debug

Starting service: api
  Command: uv run uvicorn app.main:app --reload --port 3001
  Working directory: ./src/api
  Environment variables: 12 total

api              → http://localhost:3001
```
