# Service States and Health

This document describes the service state model used by `azd app` to track and display service status.

## Key Concepts

`azd app` separates two independent concepts:

1. **Lifecycle State** - Is the process running? (managed by the service orchestrator)
2. **Health Status** - Is the service responding correctly? (determined by health checks)

These are tracked independently because:
- A running service can be unhealthy (process is up, but health checks fail)
- A stopped service has no health status (it's not running)
- A starting service has unknown health (health checks haven't passed yet)

## Service Types

Service type defines how the service is accessed (protocol level):

| Type | Description | Health Check Method | Example |
|------|-------------|---------------------|---------|
| `http` | Serves HTTP/HTTPS traffic | HTTP endpoint check | Web APIs, frontends |
| `tcp` | Raw TCP connections | Port connectivity check | Databases, gRPC services |
| `process` | No network endpoint | Process running check | Build tools, workers |

### Configuring Service Type

Service type is auto-detected based on:
1. Explicit `healthcheck.type` in `azure.yaml`
2. Presence of ports (defaults to `http`)
3. No ports (defaults to `process`)

```yaml
services:
  # Auto-detected as 'http' (has ports)
  api:
    ports: ["8080"]
    healthcheck:
      test: "http://localhost:8080/health"
  
  # Auto-detected as 'process' (no ports)
  worker:
    project: ./worker
    healthcheck:
      type: process
```

## Service Modes

Service mode defines the lifecycle behavior for process-type services:

| Mode | Description | Expected States | Use Case |
|------|-------------|-----------------|----------|
| `watch` | Continuous, watches for changes | starting → watching | `tsc --watch`, `nodemon` |
| `build` | One-time build, exits on completion | starting → building → built/failed | `tsc`, `go build` |
| `daemon` | Long-running background process | starting → running | MCP servers, workers |
| `task` | One-time task run on demand | starting → completed/failed | Migrations, scripts |

### Configuring Service Mode

```yaml
services:
  # TypeScript compiler in watch mode
  tsc-watch:
    project: ./frontend
    healthcheck:
      type: output
      pattern: "Found 0 errors"
    # Mode auto-detected from command or can be explicit
  
  # One-time build
  build-assets:
    project: ./assets
    healthcheck:
      type: process
    # Completes and shows "Built" or "Failed"
```

## Lifecycle States

These describe what phase the process is in:

| State | Description | Visual | When |
|-------|-------------|--------|------|
| `not-started` | Never been started | ○ Gray | Initial state |
| `starting` | Process is being launched | ◐ Yellow (spin) | After start command |
| `running` | Process is actively running | ● Green (pulse) | Process confirmed running |
| `stopping` | Process is being terminated | ◑ Yellow | After stop command |
| `stopped` | Process intentionally stopped | ◉ Gray | After stop completes |
| `restarting` | Process is being restarted | ↻ Blue (spin) | During restart |
| `completed` | Process finished successfully | ✓ Green | Build/task mode success |
| `failed` | Process exited with error | ✗ Red | Build/task mode error |

### Process-Specific States

For watch/build mode services:

| State | Description | Visual | Mode |
|-------|-------------|--------|------|
| `watching` | Actively watching for file changes | 👁 Green | Watch mode |
| `building` | Currently building | 🔨 Yellow (pulse) | Build mode |
| `built` | Build completed successfully | ✓ Green | Build mode |

## Health Status

These describe the service's ability to handle requests (only meaningful when running):

| Status | Description | Visual | Meaning |
|--------|-------------|--------|---------|
| `healthy` | All health checks passing | ● Green | Service is ready |
| `degraded` | Some checks failing | ⚠ Amber | Service partially available |
| `unhealthy` | Health checks failing | ✗ Red | Service not responding |
| `unknown` | Health not yet determined | ? Gray | Startup grace period |

### Health Check Types

| Check Type | How It Works | Best For |
|------------|--------------|----------|
| `http` | HTTP GET to endpoint | Web services with health endpoints |
| `tcp` | TCP connection to port | Databases, message queues |
| `process` | Checks if PID is running | Background workers, build tools |
| `output` | Matches regex in stdout | Services that log readiness |

## State + Health Combinations

The UI displays a combined status based on both lifecycle state and health:

| Lifecycle | Health | Display | Badge |
|-----------|--------|---------|-------|
| `stopped` | (any) | Stopped | Gray |
| `stopping` | (any) | Stopping | Gray |
| `starting` | (any) | Starting | Yellow |
| `running` | `healthy` | Running | Green |
| `running` | `degraded` | Degraded | Amber |
| `running` | `unhealthy` | Unhealthy | Red |
| `running` | `unknown` | Running | Green* |
| `watching` | (any) | Watching | Green |
| `building` | (any) | Building | Yellow |
| `built` | (any) | Built | Green |
| `completed` | (any) | Completed | Green |
| `failed` | (any) | Failed | Red |

*Running with unknown health shows as healthy for process-type services that don't have HTTP health checks.

## Configuration Examples

### HTTP Service with Health Check

```yaml
services:
  api:
    language: python
    project: ./api
    ports: ["8080"]
    healthcheck:
      test: "http://localhost:8080/health"
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
```

States: `starting` → `running` (health: `unknown` → `healthy`)

### Watch Mode Service

```yaml
services:
  tsc:
    project: ./frontend
    healthcheck:
      type: output
      pattern: "Found 0 errors. Watching for file changes."
```

States: `starting` → `building` → `watching`

### Build Mode Service

```yaml
services:
  assets:
    project: ./static
    healthcheck:
      type: process
```

States: `starting` → `building` → `built` or `failed`

### Background Worker (Daemon)

```yaml
services:
  worker:
    project: ./worker
    healthcheck:
      type: process
```

States: `starting` → `running`

### Disabled Health Check

```yaml
services:
  setup:
    project: ./scripts
    healthcheck: false  # or type: none
```

No health monitoring, only process lifecycle.

## Dashboard Display

The dashboard shows service status consistently across all views:

### Table/Grid View
- Status column shows lifecycle state with appropriate icon
- Color indicates health when running
- Animations indicate active states (heartbeat for healthy, pulse for building)

### Console View
- Health badge in pane header
- Real-time health updates via SSE stream

### Metrics View
- Aggregated health counts
- Service type/mode badges

## API Reference

### Health Stream (SSE)

```
GET /api/health/stream
```

Returns real-time health events:

```json
{
  "type": "health",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": [
    {
      "serviceName": "api",
      "status": "healthy",
      "checkType": "http",
      "endpoint": "http://localhost:8080/health",
      "responseTime": 15000000,
      "serviceType": "http",
      "serviceMode": null
    }
  ],
  "summary": {
    "total": 3,
    "healthy": 2,
    "degraded": 0,
    "unhealthy": 0,
    "starting": 1,
    "stopped": 0,
    "unknown": 0,
    "overall": "healthy"
  }
}
```

### Services API

```
GET /api/services
```

Returns service information including lifecycle state:

```json
{
  "services": [
    {
      "name": "api",
      "local": {
        "status": "running",
        "health": "healthy",
        "serviceType": "http",
        "serviceMode": null,
        "port": 8080,
        "pid": 12345
      }
    }
  ]
}
```

## See Also

- [Health Check Configuration](../schema/azure.yaml.md#healthcheck-new)
- [Service Types](../schema/azure.yaml.md#service-object)
- [Dashboard Guide](./dashboard.md)
