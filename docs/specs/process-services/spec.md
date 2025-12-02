# Process Services

## Problem Statement

Many development workflows include services that don't serve HTTP endpoints:

- **Build Watchers**: `tsc --watch`, `esbuild --watch`, `vite build --watch`
- **One-time Builds**: `tsc`, `go build`, `dotnet build`
- **Background Daemons**: MCP servers, queue workers, file processors
- **Stdio Servers**: Language servers, MCP servers communicating via stdio

Currently, these services are awkwardly handled:
- Ports are assigned when not needed
- HTTP health checks fail on non-HTTP services
- Status displays show "unhealthy" when the process is running fine
- Users must manually configure `healthcheck: false`

## Solution

Introduce a formal **Service Type** and **Run Mode** system:

- **Type** = How the service is accessed (protocol)
- **Mode** = How the service operates (lifecycle behavior)

## Service Types

| Type | Description | Health Check | Example |
|------|-------------|--------------|---------|
| `http` | Serves HTTP/HTTPS traffic | HTTP endpoint | APIs, web apps |
| `tcp` | Raw TCP connections | Port listening | Databases, gRPC |
| `process` | No network endpoint | Process running | Build tools, workers |

## Run Modes (for `process` type)

| Mode | Lifecycle | Success Criteria | Status When Running |
|------|-----------|------------------|---------------------|
| `watch` | Continuous, restarts on changes | Pattern match or process alive | 🟢 Watching |
| `build` | Run once, exit when complete | Exit code 0 | 🟡 Building → 🟢 Built |
| `daemon` | Long-running background | Process alive | 🟢 Running |
| `task` | Run once on demand | Exit code 0 | 🟡 Running → 🟢 Complete |

## Language Support Matrix

### Watch Mode Commands by Language

| Language | Watch Command | Build Command | Pattern for Ready |
|----------|---------------|---------------|-------------------|
| TypeScript | `tsc --watch` | `tsc` | `Watching for file changes` |
| TypeScript | `tsx watch` | `tsx` | `Watching for changes` |
| JavaScript | `nodemon` | `node` | `watching path` |
| JavaScript | `npm run dev` (varies) | `npm run build` | Framework-specific |
| Go | `air` / `reflex` | `go build` | `watching` |
| Go | `go run . --watch` (custom) | `go run .` | N/A |
| Python | `watchdog` / `watchfiles` | `python` | `Watching` |
| Python | `uvicorn --reload` | `uvicorn` | `Watching for changes` |
| .NET | `dotnet watch` | `dotnet build` | `Started` |
| Rust | `cargo watch` | `cargo build` | `Watching` |
| Java | `mvn spring-boot:run` | `mvn package` | `Started` |

### Package Manager Run Scripts

| Package Manager | Watch Script | Build Script |
|-----------------|--------------|--------------|
| npm | `npm run watch` / `npm run dev` | `npm run build` |
| pnpm | `pnpm watch` / `pnpm dev` | `pnpm build` |
| yarn | `yarn watch` / `yarn dev` | `yarn build` |
| bun | `bun run watch` / `bun run dev` | `bun run build` |

## Functional Requirements

### FR-1: Service Type Configuration

```yaml
services:
  # TypeScript watcher
  api-tsc:
    language: ts
    project: ./services/api-tsc
    type: process
    mode: watch
    healthcheck:
      pattern: "Found 0 errors"
    
  # One-time build step
  prebuild:
    language: ts
    project: ./build
    type: process
    mode: build
    
  # MCP server (stdio daemon)
  mcp-server:
    project: ./mcp
    type: process
    mode: daemon
    
  # HTTP API service
  api:
    language: ts
    project: ./services/api
    type: http  # inferred from ports
    ports:
      - "3001"
```

### FR-2: Auto-Detection Rules

When `type` is not specified:
1. Has `ports` defined → `type: http`
2. No ports → `type: process`

When `mode` is not specified (for `type: process`):
1. Command contains `--watch`, `watch`, `-w` → `mode: watch`
2. Command is build tool (`tsc`, `go build`, `dotnet build`) → `mode: build`
3. Otherwise → `mode: daemon` (default)

### FR-3: Language-Specific Watch Detection

Detect watch mode from project structure:

**TypeScript/JavaScript:**
```yaml
# Detected from package.json scripts
scripts:
  watch: "tsc --watch"      # → mode: watch
  dev: "nodemon src/app.ts" # → mode: watch  
  build: "tsc"              # → mode: build
```

**Go:**
```yaml
# Check for air.toml or .air.toml → mode: watch
# Check for reflex.conf → mode: watch
```

**.NET:**
```yaml
# Command contains "watch" → mode: watch
# dotnet watch run
```

**Python:**
```yaml
# uvicorn with --reload → mode: watch
# watchdog/watchfiles in requirements → potential watch
```

### FR-4: Health Monitoring by Type and Mode

| Type | Mode | Health Check Method |
|------|------|---------------------|
| `http` | - | HTTP GET to endpoint |
| `tcp` | - | TCP port connect |
| `process` | `watch` | Process alive + optional pattern match |
| `process` | `build` | Exit code (0 = success) |
| `process` | `daemon` | Process alive |
| `process` | `task` | Exit code (0 = success) |

### FR-5: Status Display

**HTTP Services:**
| State | Display |
|-------|---------|
| Starting | 🟡 Starting |
| Healthy | 🟢 Running |
| Unhealthy | 🔴 Unhealthy |
| Stopped | ⚪ Stopped |

**Process Services - Watch Mode:**
| State | Display |
|-------|---------|
| Starting (pattern not matched) | 🟡 Starting |
| Watching (pattern matched or process alive) | 🟢 Watching |
| Error (process exited) | 🔴 Error |
| Stopped | ⚪ Stopped |

**Process Services - Build Mode:**
| State | Display |
|-------|---------|
| Building | 🟡 Building |
| Built (exit 0) | 🟢 Built |
| Failed (exit non-0) | 🔴 Failed |

**Process Services - Daemon Mode:**
| State | Display |
|-------|---------|
| Starting | 🟡 Starting |
| Running | 🟢 Running |
| Error | 🔴 Error |
| Stopped | ⚪ Stopped |

### FR-6: Dashboard Display

**Service Card Changes:**

For `type: process`:
- Show mode badge: `Watch`, `Build`, `Daemon`
- No port/URL displayed
- Health check type: "Process" or "Output"
- Response time: "-" (not applicable)
- Uptime: Normal display

For `mode: build`:
- Show build duration instead of uptime
- Show "Complete" or "Failed" as final state
- Option to re-run build

### FR-7: Service Actions by Type

| Action | HTTP | Process (daemon/watch) | Process (build) |
|--------|------|------------------------|-----------------|
| Start | ✅ | ✅ | ✅ Run |
| Stop | ✅ | ✅ | ❌ |
| Restart | ✅ | ✅ | ✅ Re-run |
| Open URL | ✅ | ❌ | ❌ |
| View Logs | ✅ | ✅ | ✅ |

### FR-8: Dependencies and Ordering

Process services can be dependencies:

```yaml
services:
  # Build TypeScript first
  api-tsc:
    type: process
    mode: build
    project: ./api
    
  # API depends on TypeScript build
  api:
    type: http
    project: ./api
    ports: ["3001"]
    uses:
      - api-tsc  # Wait for build to complete
```

For `mode: build` dependencies:
- Dependent service waits for exit code 0
- If build fails, dependent service doesn't start

For `mode: watch` dependencies:
- Dependent service waits for pattern match (if configured)
- Or waits for startup grace period

## Configuration Reference

```yaml
services:
  service-name:
    # Service type (protocol/access method)
    type: http | tcp | process  # Default: http if ports, process otherwise
    
    # Run mode (for type: process only)
    mode: watch | build | daemon | task  # Default: daemon
    
    # Language (for auto-detection)
    language: ts | js | python | go | dotnet | java | rust
    
    # Project directory
    project: ./path/to/project
    
    # Explicit entrypoint/command
    entrypoint: "npm run watch"
    
    # Ports (implies type: http)
    ports:
      - "3000"
      
    # Health check configuration
    healthcheck:
      # For process types - stdout pattern matching
      pattern: "Watching for file changes"
      
      # For http types - endpoint path
      path: /health
      
      # Timeout for startup
      timeout: 60s
      
      # Disable health checks entirely
      disable: true
```

## Acceptance Criteria

1. **Type configuration**: Services can specify `type: http | tcp | process`
2. **Mode configuration**: Process services can specify `mode: watch | build | daemon | task`
3. **Auto-detection**: Types and modes are inferred from ports, commands, and project structure
4. **Language support**: Watch/build detection works for TS, JS, Go, Python, .NET, Rust, Java
5. **Health checks**: Appropriate health check method used for each type/mode
6. **Dashboard display**: Status shows type-appropriate indicators and badges
7. **Actions work**: Start/stop/restart work correctly for each type/mode
8. **Dependencies**: Build-mode services can be dependencies with proper ordering
9. **Backward compatible**: Existing configs continue to work

## Non-Goals

- gRPC-specific protocol handling (use `type: tcp` for now)
- WebSocket-specific handling (use `type: http`)
- Automatic restart on build failure (user-triggered only)
- Build caching or incremental builds (tool responsibility)

## Technical Implementation

### Go Backend

**types.go:**
- Add `Type` field to `Service` struct
- Add `Mode` field to `Service` struct
- Add constants: `ServiceTypeHTTP`, `ServiceTypeTCP`, `ServiceTypeProcess`
- Add constants: `ServiceModeWatch`, `ServiceModeBuild`, `ServiceModeDaemon`, `ServiceModeTask`

**detector.go:**
- Add `detectServiceType()` function
- Add `detectServiceMode()` function
- Update `DetectServiceRuntime()` to set type/mode

**monitor.go:**
- Update health check selection based on type/mode
- Add build completion detection for `mode: build`
- Add pattern matching for `mode: watch`

**registry:**
- Add `Type` and `Mode` fields to `ServiceRegistryEntry`
- Include in SSE events

### Dashboard

**types.ts:**
- Add `ServiceType` union: `'http' | 'tcp' | 'process'`
- Add `ServiceMode` union: `'watch' | 'build' | 'daemon' | 'task'`
- Add fields to `LocalServiceInfo`

**service-utils.ts:**
- Update `getStatusDisplay()` for each type/mode combination
- Add `getServiceTypeBadge()` function
- Add `getServiceModeLabel()` function

**Components:**
- Update `ServiceCard` for type/mode display
- Update `StatusCell` for mode-specific status
- Update `ServiceActions` to hide/show based on type
