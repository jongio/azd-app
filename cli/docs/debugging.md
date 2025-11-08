# Zero-Friction F5 Debugging

Enable seamless debugging for all services orchestrated by `azd app run` with zero manual configuration.

## Overview

The `azd app run --debug` command automatically:
- Configures debug ports for each service based on language
- Generates VS Code launch and task configurations
- Starts services with debuggers enabled
- Shows debug connection information

## Quick Start

### First-Time Setup

```bash
$ azd app run --debug
🔍 Detected services: api (python), web (node)
🛠️  Generated .vscode/launch.json and tasks.json...
✅ Debug configuration created!
🐛 Starting services in debug mode...
   ✅ api running (debug: localhost:5678)
   ✅ web running (debug: localhost:9229)

📖 To debug:
   1. Press F5 in VS Code
   2. Select "🚀 Debug ALL Services" to attach to all services
   3. Or select individual service to debug
   
🌐 Dashboard: http://localhost:4280
```

### VS Code F5 Experience

When you press **F5** in VS Code, the dropdown shows:

1. **🚀 Debug ALL Services** ← Starts `azd app run --debug` then attaches to all
2. **🔌 Attach to ALL (already running)** ← Just attaches (if services already running)
3. 🔌 api (python) ← Individual service
4. 🔌 web (node) ← Individual service

## Supported Languages

### Node.js / TypeScript
- **Debug Port**: 9229 (base), 9230+ for additional services
- **Protocol**: Chrome DevTools Protocol (Inspector)
- **Debug Flags**: `--inspect=0.0.0.0:9229` or `--inspect-brk=0.0.0.0:9229`
- **Debugger**: VS Code built-in Node.js debugger

### Python
- **Debug Port**: 5678 (base), 5679+ for additional services
- **Protocol**: debugpy (DAP)
- **Debug Flags**: `-m debugpy --listen 0.0.0.0:5678`
- **Debugger**: VS Code Python extension (debugpy)
- **Requirements**: `debugpy` must be installed in your virtual environment

### Go
- **Debug Port**: 2345 (base), 2346+ for additional services
- **Protocol**: Delve
- **Command**: Replaced with `dlv debug --headless --listen=:2345`
- **Debugger**: VS Code Go extension
- **Requirements**: `dlv` (Delve) must be installed

### .NET
- **Debug Port**: Not used (attach by process name)
- **Protocol**: CoreCLR
- **Attach Method**: By process name (no debug port required)
- **Debugger**: VS Code C# extension
- **Notes**: No special startup flags needed; attach by process name. Debug port assignment happens but is not used for .NET debugging.

### Java
- **Debug Port**: 5005 (base), 5006+ for additional services
- **Protocol**: JDWP (Java Debug Wire Protocol)
- **Debug Flags**: `-agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=*:5005`
- **Debugger**: VS Code Java extension

### Rust
- **Debug Port**: 4711 (base), 4712+ for additional services
- **Protocol**: LLDB/GDB
- **Status**: Manual configuration required
- **Debugger**: VS Code CodeLLDB extension

## Command Reference

### `azd app run --debug`

Start services with debuggers enabled.

```bash
azd app run --debug
```

**Options:**
- `--debug` - Enable debug mode for all services
- `--wait-for-debugger` - Pause services until debugger attaches
- `--regenerate-debug-config` - Force regenerate VS Code configurations

### Wait for Debugger

Use `--wait-for-debugger` to pause service startup until a debugger connects. Useful for debugging initialization code.

```bash
azd app run --debug --wait-for-debugger
```

**Effect by Language:**
- **Node.js**: Uses `--inspect-brk` instead of `--inspect`
- **Python**: Adds `--wait-for-client` flag
- **Go**: Sets `--continue=false` in dlv
- **Java**: Sets `suspend=y` in JDWP agent

### Regenerate Debug Config

Force regenerate `.vscode/launch.json` and `tasks.json` when services change:

```bash
azd app run --debug --regenerate-debug-config
```

## Generated VS Code Configuration

### launch.json

The tool generates a `launch.json` with:
- **Individual attach configurations** for each service
- **Compound configuration** to debug all services at once
- **Pre-launch task** to start services in debug mode

Example for Node.js service:

```json
{
  "type": "node",
  "request": "attach",
  "name": "🔌 api (node)",
  "address": "localhost",
  "port": 9229,
  "skipFiles": ["<node_internals>/**"]
}
```

Example for Python service:

```json
{
  "type": "debugpy",
  "request": "attach",
  "name": "🔌 worker (python)",
  "connect": {
    "host": "localhost",
    "port": 5678
  },
  "pathMappings": [
    {
      "localRoot": "${workspaceFolder}",
      "remoteRoot": "."
    }
  ]
}
```

### tasks.json

The tool generates a `tasks.json` with a background task that:
- Starts all services in debug mode
- Monitors for startup completion
- Allows F5 to wait for services to be ready before attaching

```json
{
  "label": "azd: Start Services (Debug)",
  "type": "shell",
  "command": "azd app run --debug",
  "isBackground": true,
  "problemMatcher": {
    "pattern": {
      "regexp": "^.*$"
    },
    "background": {
      "activeOnStart": true,
      "beginsPattern": "🐛 Starting services in debug mode",
      "endsPattern": "📊 Dashboard:"
    }
  }
}
```

## Debug Information in Dashboard

The dashboard API exposes debug information at `/api/services`:

```json
{
  "services": [
    {
      "name": "api",
      "language": "Python",
      "framework": "Flask",
      "local": {
        "status": "running",
        "health": "healthy",
        "url": "http://localhost:5000",
        "port": 5000,
        "pid": 12345,
        "debug": {
          "enabled": true,
          "port": 5678,
          "protocol": "debugpy",
          "url": "tcp://localhost:5678"
        }
      }
    }
  ]
}
```

## Debug Information in `azd app info`

View debug status of running services:

```bash
$ azd app info

📦 Project: /path/to/project

  ✓ api
  Local URL:    http://localhost:5000
  Language:     Python
  Framework:    Flask
  Port:         5000
  PID:          12345
  Debug:        enabled on port 5678 (debugpy)
  Debug URL:    tcp://localhost:5678
  Status:       running
  Health:       healthy
```

## Troubleshooting

### Python: debugpy Not Found

**Error**: `No module named debugpy`

**Solution**: Install debugpy in your virtual environment:

```bash
pip install debugpy
```

Or add to `requirements.txt`:

```txt
debugpy>=1.6.0
```

### Go: dlv Not Found

**Error**: `exec: "dlv": executable file not found`

**Solution**: Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Port Already in Use

**Error**: Debug port 9229 already in use

**Solution**: 
1. Stop other services using that port
2. Or the tool will automatically increment to the next available port (9230, 9231, etc.)

### VS Code Can't Attach

**Symptoms**: Debugger fails to attach, connection refused

**Troubleshooting**:
1. Verify service is running: `azd app info`
2. Check debug port in output
3. Ensure VS Code has correct language extension installed
4. Try regenerating config: `azd app run --debug --regenerate-debug-config`

### Breakpoints Not Hit

**Possible Causes**:
1. Source maps not configured (TypeScript/JavaScript)
2. Wrong path mappings (Python)
3. Code optimization preventing breakpoints (.NET Release mode)

**Solution**:
- For TypeScript: Ensure `sourceMap: true` in tsconfig.json
- For Python: Check `pathMappings` in launch.json
- For .NET: Use Debug configuration, not Release

## Advanced Usage

### Debug Specific Services Only

```bash
azd app run --debug --service api,worker
```

### Dry Run to See Debug Configuration

Preview debug setup without starting services:

```bash
azd app run --debug --dry-run
```

Output:
```
ℹ  api
   Language:    Python
   Framework:   Flask
   Port:        5000
   Directory:   /path/to/api
   Command:     python [-m flask run --host 0.0.0.0 --port 5000]
   Debug:       enabled on port 5678 (debugpy)
```

### Multiple Services of Same Language

When you have multiple services in the same language, debug ports are automatically incremented:

**Example**: Two Node.js services
- `api`: Debug port 9229
- `gateway`: Debug port 9230

**Example**: Three Python services
- `worker1`: Debug port 5678
- `worker2`: Debug port 5679
- `worker3`: Debug port 5680

## Best Practices

### 1. Use Wait for Debugger Sparingly

Only use `--wait-for-debugger` when debugging initialization code. It pauses service startup until debugger attaches, which can slow down development.

### 2. Install Language Extensions

Ensure you have the appropriate VS Code extensions installed:
- **Node.js**: Built-in (no extension needed)
- **Python**: Python extension (`ms-python.python`)
- **Go**: Go extension (`golang.go`)
- **.NET**: C# extension (`ms-dotnettools.csharp`)
- **Java**: Java Extension Pack (`vscjava.vscode-java-pack`)

### 3. Keep Dependencies Updated

For Python projects, ensure `debugpy` is in your `requirements.txt` or `pyproject.toml`:

```txt
# requirements.txt
debugpy>=1.6.0
```

### 4. Commit .vscode to Git

Consider committing `.vscode/launch.json` and `.vscode/tasks.json` to your repository so team members get debug configuration automatically.

```bash
git add .vscode/launch.json .vscode/tasks.json
git commit -m "Add debug configurations"
```

### 5. Use Dashboard for Multi-Service Debugging

Open the dashboard (`http://localhost:4280`) to see all services and their debug status in one place.

## Architecture

### Debug Port Assignment

Debug ports are assigned based on:
1. **Language default**: Each language has a default base port
2. **Service index**: Multiple services of same language get incremented ports
3. **Deterministic**: Same services always get same ports (based on order in azure.yaml)

### Debug Flag Injection

Debug flags are injected at service startup based on language:
- **Node.js**: Modifies command args to insert `--inspect` flag
- **Python**: Wraps command with `python -m debugpy`
- **Go**: Replaces command with `dlv debug`
- **.NET**: No modification (attach by PID)
- **Java**: Sets `JAVA_TOOL_OPTIONS` environment variable

### Registry Integration

Debug information is stored in the service registry (`.azure/services.json`) and exposed via:
- Dashboard API (`/api/services`)
- Info command (`azd app info`)

## FAQ

**Q: Do I need to install anything extra?**

A: For most languages, no. For Python, you need `debugpy`. For Go, you need `dlv` (Delve).

**Q: Can I debug in production?**

A: No, this feature is for local development only. Do not use `--debug` flag in production.

**Q: Does this work with Docker services?**

A: Not yet. Currently supports services run directly (not in containers).

**Q: Can I customize debug ports?**

A: Currently, ports are assigned automatically. Custom port assignment is planned for a future release.

**Q: Does this work with remote debugging?**

A: Yes, debug servers listen on `0.0.0.0`, allowing remote connections. However, the generated VS Code config uses `localhost`.

**Q: How do I debug a specific service only?**

A: Use the individual attach configurations in VS Code, or use `--service` flag:
```bash
azd app run --debug --service api
```

## See Also

- [VS Code Debugging Guide](https://code.visualstudio.com/docs/editor/debugging)
- [Node.js Debugging](https://code.visualstudio.com/docs/nodejs/nodejs-debugging)
- [Python Debugging](https://code.visualstudio.com/docs/python/debugging)
- [Go Debugging](https://code.visualstudio.com/docs/languages/go#_debugging)
- [.NET Debugging](https://code.visualstudio.com/docs/languages/csharp#_debugging)
- [Java Debugging](https://code.visualstudio.com/docs/java/java-debugging)
