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

## Health Diagnostics

### Overview

Health diagnostic information provides detailed insights into service health status beyond simple "healthy" or "unhealthy" labels. This information is available through dashboard tooltips and the `azd app health` command.

### Available Diagnostic Information

When a service's health is checked, the following information is collected:

| Field | Description | Example |
|-------|-------------|---------|
| **Check Type** | Method used for health check | HTTP, TCP, Process |
| **Endpoint/Target** | What was checked | `http://localhost:8080/health` |
| **Status Code** | HTTP response code (HTTP only) | 200, 503, 404 |
| **Response Time** | How long the check took | 45ms |
| **Error Message** | Primary error if unhealthy | "Connection refused" |
| **Error Details** | Extended error information | "Database connection pool exhausted" |
| **Consecutive Failures** | Failure count since last success | 3 |
| **Uptime** | Time since service started | 15m 47s |

### Enhanced Error Details

Health checks can provide two levels of error information:

**Primary Error** (`error` field):
- Basic error message from the health check
- Example: "HTTP 503: Service Unavailable"
- Always present when status is unhealthy

**Extended Error Details** (`errorDetails` field):
- Parsed from health endpoint response body
- Provides context about what's failing internally
- Example: "Database connection pool exhausted"
- Optional, depends on health endpoint implementation

**Configuration Example:**

```python
# Implement a health endpoint that returns detailed error information
@app.get("/health")
def health():
    try:
        db.execute("SELECT 1")
        return {"status": "healthy"}
    except PoolExhausted:
        return {
            "status": "unhealthy",
            "error": "Database connection failed",
            "details": "Connection pool exhausted (50/50 in use)"
        }, 503
```

### Suggested Actions Feature

When a service becomes unhealthy, the system automatically generates **suggested actions** to help you diagnose and fix the problem. These suggestions are context-aware based on:

- Health check type (HTTP, TCP, Process)
- HTTP status code (for HTTP checks)
- Error patterns
- Service configuration

**How Suggested Actions Work:**

```
Error Detected
      ↓
Analyze Context:
  • Check Type: HTTP
  • Status Code: 503
  • Error: Service Unavailable
      ↓
Generate Suggestions:
  ✓ Check service logs: azd app logs --service api
  ✓ Verify database is running
  ✓ Check network connectivity
  ✓ Review connection pool settings
      ↓
Display in Tooltip + Diagnostic Report
```

**Suggested Actions by Error Type:**

| Error Type | Common Suggestions |
|------------|-------------------|
| **HTTP 503** | Check dependencies, verify database/cache/queue |
| **HTTP 500-599** | Check logs, review stack traces, recent deployments |
| **HTTP 404** | Verify endpoint path, check health check configuration |
| **HTTP 401/403** | Check credentials, verify API keys |
| **Connection Refused** | Verify service is running, check port configuration |
| **Timeout** | Check network connectivity, firewall rules |
| **Process Not Running** | Check service logs, verify start command |

**Backend Integration:**

The backend can provide custom suggestions via the health endpoint:

```json
{
  "status": "unhealthy",
  "checks": {
    "database": "failed"
  },
  "suggestion": "Database connection pool settings may need adjustment"
}
```

This suggestion appears in the tooltip's "Suggested Actions" section.

### Consecutive Failure Tracking

The health system tracks how many times a service has failed consecutively. This helps:

- Identify persistent vs. transient issues
- Trigger appropriate alerts (e.g., alert after 3 consecutive failures)
- Provide context in diagnostic reports

**How It Works:**

```
Check 1: unhealthy → Consecutive Failures: 1
Check 2: unhealthy → Consecutive Failures: 2
Check 3: unhealthy → Consecutive Failures: 3
Check 4: healthy   → Consecutive Failures: 0 (reset)
Check 5: unhealthy → Consecutive Failures: 1
```

**Configuration:**

```yaml
services:
  api:
    healthcheck:
      test: "http://localhost:8080/health"
      retries: 3  # Mark unhealthy after 3 consecutive failures
```

**Viewing Consecutive Failures:**

- Dashboard tooltip shows: `Consecutive Failures: 3`
- CLI output includes failure count
- Diagnostic report documents failure history

### Diagnostic Report Format

When you copy diagnostics from a tooltip or export from CLI, you get a structured markdown report:

```markdown
# Service Health Diagnostic Report
**Service**: api
**Status**: unhealthy
**Timestamp**: 2025-12-29T10:30:45Z

## Health Check
- **Type**: HTTP GET
- **Endpoint**: http://localhost:8080/health
- **Status Code**: 503 Service Unavailable
- **Response Time**: 45ms
- **Consecutive Failures**: 3

## Error
Database connection failed: timeout after 5s

## Service Info
- **Uptime**: 15m 47s
- **PID**: 12345
- **Port**: 8080

## Suggested Actions
1. Check service logs: `azd app logs --service api`
2. Verify database is running
3. Check network connectivity
4. Review connection pool settings

---
Generated by azd app health
```

This format is designed to be:
- **Readable**: Easy to understand at a glance
- **Shareable**: Copy/paste into chat, email, or issues
- **Actionable**: Includes commands you can run immediately
- **Complete**: Contains all diagnostic context needed

### Accessing Diagnostic Information

**Dashboard:**
- Hover over health icon on service cards
- Tooltip appears with full diagnostic details
- Click "Copy Diagnostics" to copy report

**CLI:**
- Run `azd app health` to see health status
- Use `--verbose` flag for extended information
- Export with `--format json` for programmatic access

**MCP (Model Context Protocol):**
- AI assistants can query health status
- Automatic correlation with logs and errors
- Context-aware troubleshooting suggestions

### Troubleshooting with Diagnostics

**Example Workflow:**

1. **Notice unhealthy service** in dashboard (red health icon)
2. **Hover on health icon** to see diagnostic tooltip
3. **Read error details**: "Database connection failed: timeout after 5s"
4. **Check consecutive failures**: 3 (persistent issue)
5. **Review suggested actions**:
   - Check service logs: `azd app logs --service api`
   - Verify database is running
6. **Copy diagnostics** for sharing with team
7. **Run suggested command**: `azd app logs --service api --level error`
8. **Identify root cause**: Connection pool exhausted
9. **Apply fix**: Increase pool size in configuration
10. **Monitor recovery**: Health icon turns green when resolved

See [Health Check Troubleshooting](../troubleshooting/health-checks.md) for detailed troubleshooting scenarios.

## See Also

- [Health Check Configuration](../schema/azure.yaml.md#healthcheck-new)
- [Service Types](../schema/azure.yaml.md#service-object)
- [Dashboard Guide](./dashboard.md)
- [Health Check Command](../commands/health.md)
- [Health Check Troubleshooting](../troubleshooting/health-checks.md)
