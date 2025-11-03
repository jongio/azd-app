# How azd app Works

This guide explains the architecture and workflow of the azd app extension.

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                   azd app Extension                 │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Commands Layer                                     │
│  ├── reqs    (Check prerequisites)                  │
│  ├── deps    (Install dependencies)                 │
│  ├── run     (Start services + dashboard)           │
│  ├── info    (Show service info)                    │
│  ├── logs    (Stream logs)                          │
│  └── version (Show version)                         │
│                                                     │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Core Packages                                      │
│  ├── detector/     Project & package mgr detection  │
│  ├── installer/    Dependency installation          │
│  ├── runner/       Service execution                │
│  ├── orchestrator/ Command dependency chains        │
│  ├── executor/     Safe command execution           │
│  ├── security/     Input validation & sanitization  │
│  ├── dashboard/    Web UI server                    │
│  ├── service/      Service management               │
│  └── portmanager/  Port allocation                  │
│                                                     │
├─────────────────────────────────────────────────────┤
│                                                     │
│  External Systems                                   │
│  ├── Azure Developer CLI (azd)                      │
│  ├── Package Managers (npm, pip, dotnet, etc.)     │
│  └── Development Tools (Node.js, Python, .NET)     │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## How Commands Work

### 1. Prerequisites Check (`azd app reqs`)

**Flow:**
```
User runs: azd app reqs
     ↓
Read azure.yaml → Get requirements list
     ↓
Check cache (1-hour TTL) → Valid? Return cached results
     ↓
For each requirement:
  ├── Check if tool is installed (version detection)
  ├── Validate minimum version
  └── If checkRunning: true → Verify service is running
     ↓
Cache results → Display summary
```

**Auto-Generation (`--generate`):**
```
User runs: azd app reqs --generate
     ↓
Scan project directory:
  ├── Found package.json + pnpm-lock.yaml → Add node + pnpm
  ├── Found pyproject.toml → Add python + detected pkg mgr
  ├── Found .csproj → Add dotnet
  └── Found Dockerfile → Add docker
     ↓
Normalize versions:
  ├── Node.js: major only (22.19.0 → 22.0.0)
  ├── Python: major.minor (3.13.9 → 3.13.0)
  └── Others: as detected
     ↓
Merge with existing azure.yaml (no duplicates)
     ↓
Write updated azure.yaml
```

### 2. Install Dependencies (`azd app deps`)

**Flow:**
```
User runs: azd app deps
     ↓
Command orchestrator checks dependencies:
  reqs command not run yet? → Run reqs first
     ↓
Scan workspace for projects:
  ├── Walk directory tree
  ├── Identify project markers (package.json, pyproject.toml, *.csproj)
  └── Group by project type
     ↓
For each project:
  ├── Detect package manager (lock files, config)
  ├── Validate paths (security check)
  └── Queue installation
     ↓
Install concurrently (by type):
  ├── Node.js: npm install / pnpm install / yarn install
  ├── Python: 
  │    ├── Create venv if needed
  │    ├── pip install / poetry install / uv sync
  │    └── Save venv path for later use
  └── .NET: dotnet restore
     ↓
Display summary with success/failure count
```

**Package Manager Detection Logic:**
```
Node.js:
  pnpm-lock.yaml exists? → pnpm
  yarn.lock exists? → yarn
  Otherwise → npm (default)

Python:
  uv.lock exists? → uv
  poetry.lock exists? → poetry
  requirements.txt exists? → pip
  pyproject.toml exists? → Check for poetry/uv markers

.NET:
  .sln file? → dotnet restore on solution
  .csproj files? → dotnet restore on each project
```

### 3. Run Services (`azd app run`)

**Flow:**
```
User runs: azd app run
     ↓
Command orchestrator checks dependencies:
  deps command not run yet? → Run deps first
  (which runs reqs if needed)
     ↓
Read azure.yaml → Parse services configuration
     ↓
For each service:
  ├── Detect service type (Aspire, Node.js, Python, etc.)
  ├── Determine run command
  ├── Allocate port (if needed)
  └── Prepare environment variables
     ↓
Start dashboard server:
  ├── Allocate random port (40000-49999)
  ├── Serve embedded React app
  ├── Setup WebSocket for live updates
  └── Display dashboard URL
     ↓
Start all services concurrently:
  ├── Execute run commands via executor package
  ├── Capture stdout/stderr to log buffers
  ├── Monitor health (HTTP health checks)
  └── Update dashboard with service status
     ↓
Wait for Ctrl+C:
  ├── Stream logs to dashboard in real-time
  ├── Update health status periodically
  └── Handle service crashes/restarts
     ↓
On exit:
  ├── Stop all services gracefully
  ├── Clean up resources
  └── Display exit summary
```

**Service Detection:**
```
Aspire Project:
  ├── Check for AppHost.cs or Program.cs with Aspire markers
  ├── Run: dotnet run --project <path>
  └── Aspire dashboard handles orchestration

Docker Compose:
  ├── Check for docker-compose.yml
  ├── Run: docker compose up
  └── Parse service ports from config

Node.js Service:
  ├── Check package.json for scripts
  ├── Preferred: dev > start > serve
  └── Run: <package-manager> run <script>

Python Service:
  ├── Detect entry point (main.py, app.py, etc.)
  ├── Or use entrypoint from azure.yaml
  ├── Activate venv if created
  └── Run: python <entrypoint>

Custom Script:
  └── Use config.commands.run from azure.yaml
```

### 4. Service Information (`azd app info`)

**Flow:**
```
User runs: azd app info
     ↓
Read service registry (in-memory state)
     ↓
For each running service:
  ├── Get service name
  ├── Get process ID
  ├── Get assigned port
  ├── Get health status
  ├── Check Azure deployment info (from azd env)
  └── Build URL (local + Azure)
     ↓
Display formatted table or JSON
```

### 5. Log Streaming (`azd app logs`)

**Flow:**
```
User runs: azd app logs [options]
     ↓
Read service registry → Get list of running services
     ↓
Filter by --service flag (if provided)
     ↓
For each service:
  ├── Access log buffer (ring buffer, 10,000 lines max)
  ├── Apply --tail filter (last N lines)
  └── Stream to stdout or file
     ↓
If --follow flag:
  ├── Subscribe to new log entries
  ├── Stream in real-time
  └── Continue until Ctrl+C
```

## Security Architecture

### Input Validation

All user inputs pass through validation:

```go
// Path validation (prevents ../.. attacks)
if err := security.ValidatePath(userPath); err != nil {
    return err
}

// Package manager validation (whitelist only)
if err := security.ValidatePackageManager(pm); err != nil {
    return err
}

// Script name sanitization (blocks shell chars)
safe := security.SanitizeScriptName(userInput)
```

### Safe Command Execution

Never use raw `exec.Command()`:

```go
// ❌ WRONG - No timeout, no context, no validation
cmd := exec.Command("npm", "install")

// ✅ RIGHT - Context-aware, validated, timeout protection
executor.RunCommand("npm", []string{"install"}, projectDir)
```

The executor package provides:
- 30-minute default timeout
- Automatic context cancellation
- Environment variable inheritance
- Proper signal handling
- Error wrapping with context

### Azure Environment Isolation

azd app inherits azd's security context:

```
azd (parent process)
  ├── Sets environment variables:
  │    ├── AZD_SERVER (gRPC address)
  │    ├── AZD_ACCESS_TOKEN (JWT token)
  │    ├── AZURE_SUBSCRIPTION_ID
  │    └── All azd environment vars
  ↓
azd app (child process)
  ├── Inherits via os.Environ()
  ├── Can communicate back to azd via gRPC
  └── All spawned commands also inherit
```

## Dashboard Architecture

### Technology Stack

- **Frontend**: React 18 + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **UI Components**: shadcn/ui
- **Backend**: Go HTTP server with WebSockets

### Data Flow

```
Service Process
  ├── Writes to stdout/stderr
  ↓
Service Manager
  ├── Captures logs in ring buffer
  ├── Performs HTTP health checks
  ├── Updates service state
  ↓
Dashboard Server (Go)
  ├── Exposes REST API (/api/services, /api/logs)
  ├── Sends real-time updates via WebSocket
  ↓
React Frontend
  ├── Displays service cards with status
  ├── Streams logs in real-time
  └── Updates UI on state changes
```

### Embedded Resources

The dashboard is embedded in the binary:

```go
//go:embed dashboard/dist/*
var dashboardFS embed.FS

// Serve static files from embedded FS
http.Handle("/", http.FileServer(http.FS(distFS)))
```

No external dependencies needed to run the dashboard!

## Command Dependency Chain

Commands can depend on each other using the orchestrator pattern:

```go
type CommandFunc func() error

orchestrator := orchestrator.New()

// reqs has no dependencies
orchestrator.Register("reqs", reqs.Run)

// deps depends on reqs
orchestrator.Register("deps", deps.Run, "reqs")

// run depends on deps (which depends on reqs)
orchestrator.Register("run", run.Run, "deps")

// Execute run → automatically runs deps → automatically runs reqs
orchestrator.Execute("run")
```

**Features:**
- Automatic dependency resolution
- Memoization (each command runs once)
- Cycle detection (prevents infinite loops)
- Error propagation (failed dependency stops chain)

## Performance Optimizations

### Caching
- **Prerequisites check**: 1-hour cache to avoid repeated tool detection
- **Service registry**: In-memory state for fast queries
- **Log buffers**: Ring buffer (10,000 lines) prevents memory bloat

### Concurrency
- **Dependency installation**: Concurrent installation by project type
- **Service startup**: All services start in parallel
- **Health checks**: Non-blocking background checks
- **Log streaming**: Buffered channels for efficient streaming

### Resource Management
- **Port allocation**: Random ports avoid conflicts
- **Process cleanup**: Proper signal handling ensures clean shutdown
- **File descriptors**: Limited concurrency prevents exhaustion

## Extension Points

### Custom Commands

Add new commands by implementing Cobra command:

```go
func newMyCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "My custom command",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Your logic here
            return nil
        },
    }
}

// Register in main.go
rootCmd.AddCommand(newMyCommand())
```

### Custom Service Types

Add support for new project types by implementing detector:

```go
type MyDetector struct{}

func (d *MyDetector) Detect(dir string) (string, error) {
    // Check for your project markers
    if exists(filepath.Join(dir, "my-config.yml")) {
        return "my-project-type", nil
    }
    return "", nil
}
```

### Custom Installers

Implement installer interface for new package managers:

```go
func InstallMyPackageManager(projectDir string) error {
    return executor.RunCommand(
        "my-pm",
        []string{"install"},
        projectDir,
    )
}
```

## Troubleshooting

### Enable Debug Logging

```bash
export AZD_DEBUG=1
azd app run
```

### Check Extension Binary

```bash
# View extension location
azd extension list

# Check binary works standalone
~/.azd/extensions/app.azd.app/<version>/app version
```

### Common Issues

**Prerequisites fail:**
- Check `~/.azd/cache/reqs-cache.json` for cached values
- Delete cache to force re-check

**Services won't start:**
- Check logs with `azd app logs`
- Verify ports aren't in use
- Check azure.yaml syntax

**Dashboard won't load:**
- Check firewall settings
- Try different port range
- Check browser console for errors

## Learn More

- [Getting Started Guide](../../GETTING-STARTED.md)
- [Command Dependency Chain](dev/command-dependency-chain.md)
- [Azure Environment Context](dev/azd-environment-context.md)
- [Dashboard Implementation](dev/dashboard-per-project.md)
- [Contributing Guide](../../CONTRIBUTING.md)

### Method 2: Manual Steps

**Step 1: Create `cmd_mycommand.go`**
```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newMycommandCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Description of mycommand",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("My command is running!")
            return nil
        },
    }
}
```

**Step 2: Register in `main.go`**
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "App",
        Short: "App - Developer productivity commands",
    }

    rootCmd.AddCommand(newHiCommand())
    rootCmd.AddCommand(newMycommandCommand())  // Add this line

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**Step 3: Build and Install**
```powershell
.\install-local.ps1
```

**Step 4: Test**
```powershell
azd app mycommand
```

## 🔄 The Installation Process

### What `install-local.ps1` Does:

```powershell
# 1. Builds the extension
azd x build
    ↓
# Compiles Go code → App.exe
# Places in: ~/.azd/extensions/App.azd.App/0.1.0/
# Copies extension.yaml there too

# 2. Registers in config.json
# Adds entry to ~/.azd/config.json:
{
  "extension": {
    "installed": {
      "App.azd.App": {
        "id": "App.azd.App",
        "namespace": "App",        # This is the key!
        "path": "extensions\\App.azd.App\\0.1.0\\App.exe",
        "version": "0.1.0",
        ...
      }
    }
  }
}

# 3. Now azd knows:
# - "App" namespace → run this binary
# - Binary location
# - What capabilities it has
```

## 📊 Key Concepts

### Namespace
- Defined in `extension.yaml` and `main.go`
- This is what you type: `azd [namespace] [command]`
- Example: `azd app hi` → namespace="App", command="hi"

### Command Registration
- Each `cmd_*.go` file creates a cobra.Command
- `main.go` registers them with rootCmd.AddCommand()
- Cobra handles routing "hi" to the right function

### Binary Execution
- azd doesn't "load" your extension
- It executes it as a separate process
- Like running: `App.exe hi`
- azd just knows where to find it via config.json

## 🎓 Pattern to Follow

For every new command, follow this pattern:

1. **Create** `cmd_[name].go` with `new[Name]Command()` function
2. **Register** in `main.go` with `rootCmd.AddCommand(new[Name]Command())`
3. **Build** with `.\install-local.ps1` or `azd x build`
4. **Test** with `azd app [name]`

## 🔍 Debugging Tips

### Command not found?
```powershell
# Check if registered in config.json
Get-Content "$env:USERPROFILE\.azd\config.json" | ConvertFrom-Json | 
  Select-Object -ExpandProperty extension | 
  Select-Object -ExpandProperty installed

# Should show App.azd.App entry
```

### Binary not updating?
```powershell
# Force rebuild
Remove-Item bin -Recurse -Force
.\install-local.ps1
```

### See what azd is doing?
```powershell
# Run with debug
$env:AZD_DEBUG = "true"
azd app hi
```

## 📚 File Templates

### New Command Template (`cmd_[name].go`)
```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func new[Name]Command() *cobra.Command {
    return &cobra.Command{
        Use:   "[name]",
        Short: "Short description",
        Long:  "Long description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Your logic here
            return nil
        },
    }
}
```

### Main Registration Pattern
```go
// In main.go
rootCmd.AddCommand(new[Name]Command())
```

## 🚀 Quick Reference

```powershell
# Create new command (auto-installs)
.\new-command.ps1 -CommandName test -Install

# Manual rebuild
.\install-local.ps1

# Watch for changes (auto-rebuild)
azd x watch

# Test
azd app test

# Uninstall
.\install-local.ps1 -Uninstall
```

---

**Summary**: The extension works by:
1. Go binary with cobra commands
2. `azd x build` places binary in right location  
3. Config.json registers the namespace
4. azd executes binary when you type `azd app [command]`
5. Cobra routes to the right command handler
