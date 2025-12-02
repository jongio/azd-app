# Process Services Test Project

This test project demonstrates all service types and modes supported by azd-app.

## Service Types

| Type | Description | Example Services |
|------|-------------|------------------|
| `http` | HTTP endpoints (default with ports) | web-watch, api, api-watch |
| `tcp` | Raw TCP services | tcp-server |
| `process` | No network endpoint | cli-*, mcp-server, data-processor, go-* |

## Service Modes

| Mode | Description | Example Services |
|------|-------------|------------------|
| `watch` | Continuous file watching | web-watch, api-watch, cli-watch, go-watch |
| `build` | One-time build | cli-build, go-build |
| `daemon` | Long-running background | mcp-server |
| `task` | On-demand execution | data-processor |

## Services Overview

### HTTP Services (type: http)

1. **web-watch** - TypeScript/Express with nodemon (watch mode auto-detected)
2. **api** - Python/FastAPI (standard running mode)
3. **api-watch** - Python/FastAPI with --reload (watch mode auto-detected)

### TCP Services (type: tcp)

4. **tcp-server** - Node.js TCP socket server

### Process Services (type: process)

5. **cli-watch** - TypeScript CLI with nodemon (explicit watch mode)
6. **cli-build** - TypeScript CLI build (explicit build mode)
7. **mcp-server** - MCP server daemon (explicit daemon mode)
8. **data-processor** - Python data processor (explicit task mode)
9. **go-watch** - Go tool with air.toml (watch mode auto-detected)
10. **go-build** - Go build command (build mode auto-detected)
11. **dotnet-watch** - .NET with dotnet watch (watch mode auto-detected)

## Running the Test

```bash
cd cli/tests/projects/process-services-test

# Start all services
azd-app start

# Check health status - should show different types/modes
azd-app health

# View in dashboard
azd-app dashboard
```

## Expected Dashboard Behavior

- **HTTP services**: Show port, health status (healthy/unhealthy/degraded)
- **TCP services**: Show port, connection status
- **Process services**: Show mode badge (watch/build/daemon/task), process-specific status:
  - `watching` - File watcher active
  - `building` - Build in progress
  - `built` - Build completed
  - `failed` - Process/build failed
  - `running` - Daemon running

## Status Indicators

| Status | Icon | Color |
|--------|------|-------|
| watching | Eye | Green |
| building | Hammer | Yellow (animated) |
| built | CheckCircle | Green |
| failed | XCircle | Red |
| running (process) | Cog | Green |
