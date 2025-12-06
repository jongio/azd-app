# azd app health

## Overview

The `health` command provides comprehensive health monitoring for services running locally or deployed to Azure. It supports two modes: **static** (point-in-time snapshot) and **streaming** (real-time continuous monitoring). The command intelligently monitors `/health` endpoints when available and falls back to process-level health checks for services without dedicated health endpoints.

## Purpose

- **Health Monitoring**: Check the health status of all services or specific services
- **Real-Time Updates**: Stream health data continuously for live monitoring
- **Intelligent Detection**: Automatically detect and use `/health` endpoints or fall back to process checks
- **Multi-Level Checks**: Support HTTP health endpoints, TCP port checks, and process-level monitoring
- **Status Reporting**: Provide actionable health status with detailed error information
- **Integration Ready**: Output formats suitable for dashboards, alerting, and automation

## Command Usage

```bash
azd app health [flags]
```

### Flags

#### Basic Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | `-s` | string | | Monitor specific service(s) only (comma-separated) |
| `--stream` | | bool | `false` | Enable streaming mode for real-time updates |
| `--interval` | `-i` | duration | `5s` | Interval between health checks in streaming mode |
| `--output` | `-o` | string | `text` | Output format: 'text', 'json', 'table' |
| `--endpoint` | | string | `/health` | Default health endpoint path to check |
| `--timeout` | | duration | `5s` | Timeout for each health check |
| `--all` | | bool | `false` | Show health for all projects on this machine |
| `--verbose` | `-v` | bool | `false` | Show detailed health check information |

#### Profile and Logging Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--profile` | string | | Health profile to use (development, production, ci, staging, or custom) |
| `--log-level` | string | `info` | Log level: debug, info, warn, error |
| `--log-format` | string | `pretty` | Log format: json, pretty, text |
| `--save-profiles` | bool | `false` | Save sample health profiles to .azd/health-profiles.yaml |

#### Metrics Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--metrics` | bool | `false` | Enable Prometheus metrics exposition |
| `--metrics-port` | int | `9090` | Port for Prometheus metrics endpoint |

#### Circuit Breaker Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--circuit-breaker` | bool | `false` | Enable circuit breaker pattern |
| `--circuit-break-count` | int | `5` | Number of failures before opening circuit |
| `--circuit-break-timeout` | duration | `60s` | Circuit breaker timeout duration |

#### Rate Limiting and Caching Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rate-limit` | int | `0` | Max health checks per second per service (0 = unlimited) |
| `--cache-ttl` | duration | `0` | Cache TTL for health results (0 = no caching) |

## Execution Flow

### Overall Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app health                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Flags & Validate Arguments                            │
│  - Validate interval (must be >= 1s)                         │
│  - Validate timeout (must be >= 1s, <= 60s)                  │
│  - Validate output format                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Discover Services                                           │
│  - Load service registry from .azure/services.json           │
│  - Filter by --service flag if specified                     │
│  - Load azure.yaml for service definitions                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Detect Health Endpoints                                     │
│  - Check for custom health endpoint in azure.yaml            │
│  - Try common paths: /health, /healthz, /ready, /alive       │
│  - Fall back to port check if no HTTP endpoint               │
│  - Fall back to process check as last resort                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              --stream?             No
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────────┐
        │ Streaming Mode   │ │ Static Mode     │
        └──────────────────┘ └─────────────────┘
```

### Static Mode Flow (Default)

```
┌─────────────────────────────────────────────────────────────┐
│  Perform Health Checks (Parallel)                            │
│  - Check all services concurrently                           │
│  - Apply timeout to each check                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Aggregate Results                                           │
│  - Collect health status for each service                    │
│  - Calculate overall health score                            │
│  - Identify unhealthy services                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Format & Display Output                                     │
│  - Format according to --output flag                         │
│  - Display timestamp                                         │
│  - Show summary statistics                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Exit with Status Code                                       │
│  - 0: All services healthy                                   │
│  - 1: One or more services unhealthy                         │
│  - 2: Error performing health checks                         │
└─────────────────────────────────────────────────────────────┘
```

### Streaming Mode Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Initialize Stream                                           │
│  - Clear terminal (if TTY)                                   │
│  - Set up signal handlers (Ctrl+C)                           │
│  - Display header                                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Start Monitoring Loop                                       │
│  while not interrupted:                                      │
│    ├─ Perform Health Checks (Parallel)                       │
│    ├─ Update Display                                         │
│    │  - Refresh health status                                │
│    │  - Show changes since last check                        │
│    │  - Update timestamps                                    │
│    ├─ Write to Output Stream                                 │
│    │  - JSON stream (newline-delimited)                      │
│    │  - Text updates (if TTY)                                │
│    └─ Wait for Interval                                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Graceful Shutdown (on Ctrl+C or error)                      │
│  - Display final summary                                     │
│  - Show total checks performed                               │
│  - Exit with appropriate code                                │
└─────────────────────────────────────────────────────────────┘
```

## Health Check Strategies

The command uses a cascading strategy to determine the best health check method for each service:

### 1. HTTP Health Endpoint (Preferred)

**Priority**: Highest

**When Used**:
- Service has explicit `healthCheck.endpoint` in azure.yaml
- Service responds to common health endpoints

**Detection Logic**:
```
1. Check azure.yaml for explicit health endpoint:
   services:
     api:
       healthCheck:
         endpoint: /api/health
         type: http

2. Try common health endpoint paths in order:
   - /health
   - /healthz
   - /ready
   - /alive
   - /ping

3. Use first endpoint that returns 2xx or 3xx status
```

**Health Check Process**:
```
1. Construct URL: http://localhost:{port}{endpoint}
2. Send HTTP GET request (with timeout)
3. Evaluate response:
   - 200-299: Healthy
   - 300-399: Healthy (redirect accepted)
   - 400-499: Unhealthy (client error)
   - 500-599: Unhealthy (server error)
   - Timeout/Connection Error: Unhealthy
```

**Response Data Extraction**:
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "2024-11-08T10:30:00Z",
  "version": "1.2.3",
  "checks": {
    "database": "healthy",
    "cache": "healthy",
    "queue": "degraded"
  },
  "details": {
    "uptime": 3600,
    "requests": 12345
  }
}
```

### 2. TCP Port Check (Fallback)

**Priority**: Medium

**When Used**:
- No HTTP health endpoint found
- Service expected to listen on a port
- Service is not HTTP-based

**Health Check Process**:
```
1. Attempt TCP connection to localhost:{port}
2. Connection succeeds → Healthy
3. Connection refused/timeout → Unhealthy
4. Timeout: 2 seconds
```

**Use Cases**:
- Database services (PostgreSQL, Redis, etc.)
- gRPC services
- TCP-based services
- Services during startup (before HTTP ready)

### 3. Process Health Check (Last Resort)

**Priority**: Lowest

**When Used**:
- Service has no port configured
- Port check fails but process should be running
- Background worker services
- Services without network listeners

**Health Check Process**:
```
1. Check if PID exists in registry
2. Verify process is running (OS-specific):
   - Unix: kill -0 {pid}
   - Windows: tasklist /FI "PID eq {pid}"
3. Process exists and running → Healthy
4. Process not found → Unhealthy
```

**Limitations**:
- Cannot detect if process is hung or deadlocked
- Only confirms process existence, not functionality
- Least reliable health indicator

## Health Status Values

Health status describes whether a running service is responding correctly. This is separate from lifecycle state (whether the process is running).

| Status | Meaning | Criteria | Color |
|--------|---------|----------|-------|
| `healthy` | Service fully operational | HTTP 2xx/3xx, port listening, or process running | Green ✓ |
| `degraded` | Service running but with issues | HTTP returns degraded status | Amber ⚠ |
| `unhealthy` | Service not functioning | HTTP 4xx/5xx, port not listening, process dead | Red ✗ |
| `starting` | Service is initializing | Startup grace period, health not yet determined | Yellow ○ |
| `unknown` | Health status cannot be determined | No health check available or check failed | Gray ? |

### Health Check Result

```go
type HealthCheckResult struct {
    ServiceName  string                 `json:"serviceName"`
    Status       string                 `json:"status"` // healthy, degraded, unhealthy, unknown
    CheckType    string                 `json:"checkType"` // http, tcp, process, output
    Endpoint     string                 `json:"endpoint,omitempty"` // For HTTP checks
    ResponseTime time.Duration          `json:"responseTime"` // Milliseconds
    StatusCode   int                    `json:"statusCode,omitempty"` // HTTP status code
    Error        string                 `json:"error,omitempty"` // Error message if unhealthy
    Timestamp    time.Time              `json:"timestamp"`
    Details      map[string]interface{} `json:"details,omitempty"` // Extracted from health endpoint
    
    // Additional metadata
    Port         int                    `json:"port,omitempty"`
    PID          int                    `json:"pid,omitempty"`
    Uptime       time.Duration          `json:"uptime,omitempty"`
}
```

## Configuration in azure.yaml

Services can specify health check configuration using **Docker Compose-compatible format**:

```yaml
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
    healthcheck:              # Docker Compose compatible key
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/health"]
      # Alternative test formats:
      # test: curl -f http://localhost:8080/api/health || exit 1
      # test: ["CMD-SHELL", "curl -f http://localhost:8080/api/health || exit 1"]
      interval: 10s           # Time between checks (default: 30s)
      timeout: 5s             # Maximum time for check (default: 30s)
      retries: 3              # Consecutive failures before unhealthy (default: 3)
      start_period: 40s       # Grace period for container initialization (default: 0s)
      start_interval: 5s      # Interval during start period (default: 5s)
  
  worker:
    language: python
    project: ./worker
    # No healthcheck defined - will use process check as fallback
  
  redis:
    language: other
    project: ./redis
    ports:
      - "6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 3
  
  db:
    language: other
    project: ./db
    ports:
      - "5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 5
      start_period: 60s
```

**Docker Compose Compatibility**: The `healthcheck` configuration matches Docker Compose specification exactly, making it easy to transition between local development and containerized deployments.

**Test Command Formats**:
- `["CMD", "executable", "arg1", "arg2"]` - Direct command execution
- `["CMD-SHELL", "shell-command"]` - Shell command execution
- `"shell-command"` - String format (automatically wrapped with CMD-SHELL)
- `["NONE"]` - Disable health check

### Skipping Health Checks for Build/Watch Services

For services that don't serve HTTP endpoints (like TypeScript compilers in watch mode, build watchers, or background processors), you can disable health checks entirely:

**Option 1: Boolean false (simplest)**
```yaml
services:
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck: false  # Skip health checks entirely
```

**Option 2: Using disable flag**
```yaml
services:
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck:
      disable: true
```

**Option 3: Using type: none**
```yaml
services:
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck:
      type: none
```

**Option 4: Using test: ["NONE"]**
```yaml
services:
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck:
      test: ["NONE"]
```

When health checks are disabled:
- No port is auto-assigned for the service
- The service is always considered "healthy" (running)
- Health monitoring falls back to process-level checks (is the process running?)
- Dependent services using `uses:` will not be affected

**Example: TypeScript compiler with dependent API server**
```yaml
services:
  # TypeScript compiler for API (watch mode) - no HTTP endpoint
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck: false  # Skip HTTP health checks
    environment:
      NODE_ENV: development

  # API server with nodemon - serves HTTP on port 3001
  api:
    language: ts
    project: ./services/api
    ports:
      - "3001"
    uses:
      - api-tsc  # Depends on api-tsc being healthy (process running)
    healthcheck:
      test: "http://localhost:3001/health"
      interval: 10s
```

### Health Check Types

The `type` field supports the following values:

| Type | Description | Use Case |
|------|-------------|----------|
| `http` | HTTP endpoint check (default) | Web servers, APIs |
| `tcp` | TCP port check | Databases, gRPC services |
| `process` | Process existence check | Background workers |
| `output` | Pattern matching in stdout | Build tools, watchers |
| `none` | Skip health checks | Build/watch services |

### Service Types and Modes

Services are categorized by their **type** (protocol) and **mode** (lifecycle behavior):

**Service Types:**

| Type | Description | Auto-Detection |
|------|-------------|----------------|
| `http` | HTTP traffic with health endpoint | Default when service has ports |
| `tcp` | Raw TCP connections | Explicit configuration |
| `process` | No network endpoint | Default when no ports defined |

**Service Modes (for process type):**

| Mode | Description | Detection |
|------|-------------|-----------|
| `watch` | Continuously watching for file changes | Commands with `--watch`, `-w`, or watch tools (nodemon, air, etc.) |
| `build` | One-time build, exits on completion | Commands with `build`, `compile`, `tsc` without watch |
| `daemon` | Long-running background process | Explicit configuration |
| `task` | On-demand one-time execution | Explicit configuration |

**Configuration Example:**

```yaml
services:
  # HTTP service (auto-detected from ports)
  api:
    language: go
    project: ./api
    ports:
      - "8080"
    healthcheck:
      test: "http://localhost:8080/health"
  
  # Process service in watch mode (auto-detected)
  api-tsc:
    language: ts
    project: ./services/api-tsc
    type: process        # Explicit: no network endpoint
    mode: watch          # Explicit: watching for changes
    # Auto-detected from: npm run watch, tsc --watch, nodemon, etc.
  
  # Process service in build mode
  frontend-build:
    language: ts
    project: ./frontend
    type: process
    mode: build          # One-time build, exits when done
```

**Health Check Behavior by Type/Mode:**

| Type | Mode | Health Strategy | Lifecycle States |
|------|------|-----------------|------------------|
| http | - | HTTP endpoint check | running (health: healthy/degraded/unhealthy) |
| tcp | - | Port listening check | running (health: healthy/unhealthy) |
| process | watch | Output pattern match | starting → watching |
| process | build | Exit code check | starting → building → built/failed |
| process | daemon | Process alive check | starting → running |
| process | task | Exit code check | starting → completed/failed |

> **Note**: For http/tcp services, the health column shows **health status** (healthy/unhealthy). For process services, the states shown are **lifecycle states** (watching, building, etc.) since they don't have traditional health checks.

**Auto-Detection Rules:**

1. **Service Type Detection:**
   - Has `ports` defined → `http` type (default)
   - No `ports` defined → `process` type

2. **Service Mode Detection (for process type):**
   - Command contains `--watch` or `-w` → `watch` mode
   - Command contains watch tools (nodemon, air, watchexec) → `watch` mode
   - Project has `air.toml` or watch script in package.json → `watch` mode
   - Command contains `build`, `compile`, `tsc` (without watch) → `build` mode
   - Default → `daemon` mode

**Language-Specific Watch Detection:**

| Language | Watch Indicators |
|----------|------------------|
| TypeScript/JavaScript | nodemon, ts-node --watch, tsc -w, npm run dev (with watch) |
| Go | air, reflex, air.toml, .air.toml |
| Python | uvicorn --reload, watchdog, watchgod |
| .NET | dotnet watch |
| Rust | cargo-watch |
| Java | spring-boot-devtools, jrebel |

**Output-based health check (experimental)**:
```yaml
services:
  api-tsc:
    language: ts
    project: ./services/api-tsc
    healthcheck:
      type: output
      pattern: "Found 0 errors"  # Regex to match in stdout
      timeout: 60s
```

**Legacy Format** (still supported for backward compatibility):
```yaml
services:
  api:
    healthCheck:              # Legacy camelCase key
      type: http
      endpoint: /api/health
      interval: 5s
      timeout: 3s
      startPeriod: 30s
      retries: 3
```


## Output Formats

### Text Format (Default)

Human-readable format with colors and icons:

```
Health Check (2024-11-08 10:30:00)
=====================================

✓ web                          healthy      (http)
  Endpoint: http://localhost:3000/health
  Response Time: 45ms
  Status Code: 200
  Uptime: 2h 15m

✓ api                          healthy      (http)
  Endpoint: http://localhost:8080/api/health
  Response Time: 23ms
  Status Code: 200
  Details:
    - database: healthy
    - cache: healthy
  Uptime: 2h 15m

⚠ worker                       degraded     (http)
  Endpoint: http://localhost:8081/health
  Response Time: 156ms
  Status Code: 200
  Details:
    - queue: degraded (backlog: 1234)
  Uptime: 2h 10m

✗ db                           unhealthy    (tcp)
  Port: 5432
  Error: connection refused
  
─────────────────────────────────────────────

Summary: 2 healthy, 1 degraded, 1 unhealthy
Overall Status: DEGRADED
```

### Table Format

Compact tabular view:

```
┌──────────┬───────────┬───────────┬──────────────────────────────────────┬──────────┐
│ SERVICE  │ STATUS    │ TYPE      │ ENDPOINT/PORT                        │ RESPONSE │
├──────────┼───────────┼───────────┼──────────────────────────────────────┼──────────┤
│ web      │ healthy   │ http      │ http://localhost:3000/health         │ 45ms     │
│ api      │ healthy   │ http      │ http://localhost:8080/api/health     │ 23ms     │
│ worker   │ degraded  │ http      │ http://localhost:8081/health         │ 156ms    │
│ db       │ unhealthy │ tcp       │ localhost:5432                       │ error    │
└──────────┴───────────┴───────────┴──────────────────────────────────────┴──────────┘
```

### JSON Format

Machine-readable format:

```json
{
  "timestamp": "2024-11-08T10:30:00Z",
  "project": "/path/to/project",
  "services": [
    {
      "serviceName": "web",
      "status": "healthy",
      "checkType": "http",
      "endpoint": "http://localhost:3000/health",
      "responseTime": 45,
      "statusCode": 200,
      "timestamp": "2024-11-08T10:30:00Z",
      "port": 3000,
      "uptime": 8100
    },
    {
      "serviceName": "api",
      "status": "healthy",
      "checkType": "http",
      "endpoint": "http://localhost:8080/api/health",
      "responseTime": 23,
      "statusCode": 200,
      "timestamp": "2024-11-08T10:30:00Z",
      "details": {
        "database": "healthy",
        "cache": "healthy"
      },
      "port": 8080,
      "uptime": 8100
    },
    {
      "serviceName": "worker",
      "status": "degraded",
      "checkType": "http",
      "endpoint": "http://localhost:8081/health",
      "responseTime": 156,
      "statusCode": 200,
      "timestamp": "2024-11-08T10:30:00Z",
      "details": {
        "queue": "degraded",
        "backlog": 1234
      },
      "port": 8081,
      "uptime": 7800
    },
    {
      "serviceName": "db",
      "status": "unhealthy",
      "checkType": "tcp",
      "port": 5432,
      "error": "connection refused",
      "timestamp": "2024-11-08T10:30:00Z"
    }
  ],
  "summary": {
    "total": 4,
    "healthy": 2,
    "degraded": 1,
    "unhealthy": 1,
    "unknown": 0,
    "overall": "degraded"
  }
}
```

### JSON Stream Format (Streaming Mode)

**For CLI (piped/non-TTY)**: Newline-delimited JSON (NDJSON)

```json
{"timestamp":"2024-11-08T10:30:00Z","services":[{"serviceName":"web","status":"healthy","responseTime":45}]}
{"timestamp":"2024-11-08T10:30:05Z","services":[{"serviceName":"web","status":"healthy","responseTime":43}]}
{"timestamp":"2024-11-08T10:30:10Z","services":[{"serviceName":"web","status":"healthy","responseTime":47}]}
```

Each line is a complete JSON object representing one health check cycle. Works with standard Unix tools:

```bash
azd app health --stream --output json | jq '.services[] | select(.status != "healthy")'
```

### Server-Sent Events (SSE) Format (Dashboard API)

**For Dashboard and MCP Integration**: Server-Sent Events provide real-time updates via HTTP

```
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"timestamp":"2024-11-08T10:30:00Z","services":[...]}

data: {"timestamp":"2024-11-08T10:30:05Z","services":[...]}

event: health-change
data: {"service":"api","oldStatus":"healthy","newStatus":"unhealthy"}

event: heartbeat
data: {"timestamp":"2024-11-08T10:30:10Z"}
```

**Benefits**:
- Browser native support (`EventSource` API)
- Auto-reconnection built-in
- Event types for different message kinds
- Simpler than WebSocket (HTTP-based)
- Works through proxies and firewalls

**Dashboard API Endpoint**:
```bash
GET /api/health/stream
```

### WebSocket Format (Optional - Advanced Use Cases)

**For Advanced Dashboard Features**: WebSocket provides bidirectional communication

```javascript
// Client side
const ws = new WebSocket('ws://localhost:4280/health/stream');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'health') {
    // Update health display
  }
};

// Can send commands to server
ws.send(JSON.stringify({
  type: 'filter',
  services: ['api', 'web']
}));
```

**Server sends**:
```json
{"type":"health","timestamp":"2024-11-08T10:30:00Z","services":[...]}
{"type":"status-change","service":"api","status":"unhealthy"}
{"type":"heartbeat","timestamp":"2024-11-08T10:30:05Z"}
```

**Use Cases**:
- Pause/resume monitoring from dashboard
- Filter services in real-time
- Request on-demand health checks
- Lower latency than SSE

**Dashboard API Endpoint**:
```bash
ws://localhost:4280/health/ws
```

## Streaming Mode

### Real-Time Monitoring

Streaming mode continuously monitors services and displays updates:

**Terminal Output (Interactive)**:
```
╔═══════════════════════════════════════════════════════════╗
║              Real-Time Health Monitoring                  ║
║  Started: 2024-11-08 10:30:00    Interval: 5s             ║
║  Press Ctrl+C to stop                                     ║
╚═══════════════════════════════════════════════════════════╝

┌─────────────────────────────────────────────────────────────┐
│ Last Update: 10:30:15 (15 seconds ago)                      │
│ Checks Performed: 3                                         │
├─────────────────────────────────────────────────────────────┤
│ ✓ web           healthy   45ms   [███████████] 100%        │
│ ✓ api           healthy   23ms   [███████████] 100%        │
│ ⚠ worker        degraded  156ms  [████████░░] 80%          │
│ ✗ db            unhealthy -      [░░░░░░░░░░] 0%           │
└─────────────────────────────────────────────────────────────┘

Recent Changes:
  10:30:10 - worker: healthy → degraded (queue backlog)
  10:30:05 - db: healthy → unhealthy (connection refused)
```

**Non-Interactive Output (Piped/Redirected)**:
```json
{"timestamp":"2024-11-08T10:30:00Z","services":[...]}
{"timestamp":"2024-11-08T10:30:05Z","services":[...]}
```

### Stream Control

- **Start**: `azd app health --stream`
- **Stop**: Press Ctrl+C (SIGINT)
- **Graceful Shutdown**: Displays final summary before exit

### Performance Considerations

- Checks run in parallel (one goroutine per service)
- Maximum concurrent checks: 10 (configurable)
- Memory efficient (circular buffer for history)
- CPU throttling if system under load

## Integration with Existing Systems

### Service Registry

Health checks read from and update the service registry:

```go
// Read registry for service list
registry := registry.GetRegistry(projectDir)
services := registry.ListAll()

// Update health status in registry
for _, svc := range services {
    result := performHealthCheck(svc)
    registry.UpdateHealth(svc.Name, result.Status, result.Error)
}
```

### Service Info Package

Integrates with existing `serviceinfo` package:

```go
import "github.com/jongio/azd-app/cli/src/internal/serviceinfo"

// Get service information including health
info := serviceinfo.GetServiceInfo(projectDir, serviceName)
health := performHealthCheck(info)

// Update service info with health data
info.Local.Health = health.Status
info.Local.LastChecked = health.Timestamp
```

### Dashboard Integration

Health data available via dashboard API:

```
GET /api/health
GET /api/health/{serviceName}
GET /api/health/stream  (Server-Sent Events)
```

## Error Handling

### Connection Errors

```
✗ api                          unhealthy    (http)
  Endpoint: http://localhost:8080/health
  Error: dial tcp [::1]:8080: connect: connection refused
  Suggestion: Check if service is running with 'azd app info'
```

### Timeout Errors

```
✗ slow-service                 unhealthy    (http)
  Endpoint: http://localhost:9000/health
  Error: health check timed out after 5s
  Suggestion: Increase timeout with --timeout or check service logs
```

### Health Endpoint Returns Error

```
✗ api                          unhealthy    (http)
  Endpoint: http://localhost:8080/api/health
  Status Code: 503
  Error: Service Unavailable
  Response: {"status":"unhealthy","details":{"database":"connection failed"}}
  Suggestion: Check service dependencies and logs
```

### No Services Running

```
No running services found in /path/to/project

Run 'azd app run' to start services
```

### Invalid Configuration

```
Error: Invalid health check configuration for service 'api'
  - interval must be at least 1s (got: 500ms)
  - timeout must be less than interval (got: timeout=10s, interval=5s)
```

## Common Use Cases

### 1. Quick Health Check

```bash
# Check all services
azd app health

# Output:
# ✓ All services healthy (checked 3 services)
```

### 2. Specific Service Health

```bash
# Check single service
azd app health --service api

# Check multiple services
azd app health --service web,api
```

### 3. Continuous Monitoring

```bash
# Stream health updates every 5 seconds
azd app health --stream

# Custom interval
azd app health --stream --interval 10s
```

### 4. CI/CD Integration

```bash
# Check health and exit with code
azd app health --output json

# Exit codes:
# 0 = all healthy
# 1 = one or more unhealthy
# 2 = error checking health

# In CI pipeline:
azd app run &
sleep 10
azd app health || exit 1
```

### 5. Scripting and Automation

```bash
# Get health as JSON for processing
azd app health --output json | jq '.summary.unhealthy'

# Monitor until healthy
while ! azd app health --service api > /dev/null 2>&1; do
  echo "Waiting for api to be healthy..."
  sleep 2
done
```

### 6. Debugging Startup Issues

```bash
# Verbose output for troubleshooting
azd app health --verbose

# Output:
# Checking web...
#   Trying /health... 200 OK (45ms)
#   Health endpoint found: /health
# ✓ web healthy
```

### 7. Custom Health Endpoint

```bash
# Override default health endpoint
azd app health --endpoint /api/status

# Per-service endpoints configured in azure.yaml
```

### 8. Long-Running Monitoring

```bash
# Stream to file for analysis
azd app health --stream --output json > health.log

# Parse later:
cat health.log | jq -r '.services[] | select(.status != "healthy") | .serviceName'
```

## Advanced Features

### Health History Tracking

```bash
# Show health history (last 10 checks)
azd app health --history

# Output:
# api:
#   10:30:00 - healthy (45ms)
#   10:30:05 - healthy (43ms)
#   10:30:10 - degraded (156ms)
#   10:30:15 - unhealthy (timeout)
#   10:30:20 - healthy (52ms)
```

### Alerting Thresholds

```yaml
# azure.yaml
services:
  api:
    healthCheck:
      endpoint: /health
      alerts:
        responseTime: 1000ms  # Alert if > 1s
        failureThreshold: 3   # Alert after 3 consecutive failures
```

### Health Metrics Export

```bash
# Export health metrics to Prometheus
azd app health --export prometheus --output metrics.txt

# Output (Prometheus format):
# azd_service_health{service="web"} 1
# azd_service_health{service="api"} 1
# azd_service_health{service="worker"} 0.5
# azd_service_response_time{service="web"} 0.045
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | All services healthy |
| 1 | Unhealthy | One or more services unhealthy or degraded |
| 2 | Error | Error performing health checks or invalid configuration |
| 130 | Interrupted | User pressed Ctrl+C in streaming mode (normal) |

## Best Practices

1. **Define Health Endpoints**: Implement `/health` endpoints in all HTTP services
2. **Structured Health Responses**: Return JSON with detailed status information
3. **Appropriate Timeouts**: Set timeouts based on service complexity (3-5s typical)
4. **Grace Periods**: Allow startup time before marking unhealthy
5. **Dependency Checks**: Include upstream dependency status in health responses
6. **Use Streaming for Monitoring**: Long-running health monitoring in development
7. **CI/CD Integration**: Use exit codes to fail builds on unhealthy services
8. **Custom Endpoints**: Configure service-specific health paths when needed

## Related Commands

- [`azd app run`](./run.md) - Start services (health checks run during startup)
- [`azd app info`](./info.md) - Show service information (includes last known health)
- [`azd app logs`](./logs.md) - View service logs (useful when health checks fail)

## Related Documentation

- [Service States and Health](../features/service-states.md) - Understanding lifecycle states vs health status
- [azure.yaml Health Configuration](../schema/azure.yaml.md#healthcheck-new) - Health check configuration reference

## Examples

### Example 1: Basic Health Check

```bash
$ azd app health

Health Check (2024-11-08 10:30:00)
=====================================

✓ web                          healthy      (http)
  Response Time: 45ms

✓ api                          healthy      (http)
  Response Time: 23ms

─────────────────────────────────────────────

Summary: 2 healthy, 0 degraded, 0 unhealthy
Overall Status: HEALTHY
```

### Example 2: Streaming Mode

```bash
$ azd app health --stream --interval 3s

╔═══════════════════════════════════════════════════════════╗
║              Real-Time Health Monitoring                  ║
║  Started: 2024-11-08 10:30:00    Interval: 3s             ║
║  Press Ctrl+C to stop                                     ║
╚═══════════════════════════════════════════════════════════╝

[Updates every 3 seconds...]

^C
Stopping health monitoring...
Total checks: 24
Uptime: 100% (web), 95.8% (api)
```

### Example 3: JSON Output for Automation

```bash
$ azd app health --output json

{
  "timestamp": "2024-11-08T10:30:00Z",
  "project": "/Users/dev/myapp",
  "services": [
    {
      "serviceName": "web",
      "status": "healthy",
      "checkType": "http",
      "endpoint": "http://localhost:3000/health",
      "responseTime": 45,
      "statusCode": 200,
      "timestamp": "2024-11-08T10:30:00Z"
    }
  ],
  "summary": {
    "total": 1,
    "healthy": 1,
    "overall": "healthy"
  }
}
```

### Example 4: Unhealthy Service

```bash
$ azd app health

Health Check (2024-11-08 10:30:00)
=====================================

✗ api                          unhealthy    (http)
  Endpoint: http://localhost:8080/api/health
  Status Code: 503
  Error: Service Unavailable
  Response Body: {
    "status": "unhealthy",
    "checks": {
      "database": "connection failed"
    }
  }
  
  Troubleshooting:
    1. Check service logs: azd app logs --service api
    2. Verify dependencies are running
    3. Check database connection settings

─────────────────────────────────────────────

Summary: 0 healthy, 0 degraded, 1 unhealthy
Overall Status: UNHEALTHY

# Exit code: 1
```

### Example 5: Custom Configuration

```yaml
# azure.yaml
services:
  api:
    language: python
    project: ./api
    ports:
      - "8080"
    healthCheck:
      type: http
      endpoint: /api/v1/health
      timeout: 10s
      interval: 5s
      startPeriod: 30s
      headers:
        X-Health-Token: secret123
```

```bash
$ azd app health --service api --verbose

Checking api health...
  Type: http
  Endpoint: http://localhost:8080/api/v1/health
  Timeout: 10s
  Headers: X-Health-Token: ***
  
  Sending GET request...
  Response received in 234ms
  Status Code: 200
  Response Body: {
    "status": "healthy",
    "version": "1.2.3",
    "uptime": 3600,
    "dependencies": {
      "database": "healthy",
      "cache": "healthy",
      "queue": "healthy"
    }
  }

✓ api                          healthy      (http)
  Response Time: 234ms
  Version: 1.2.3
  Dependencies: 3/3 healthy
```

## Implementation Notes

### Not Implemented Yet

This is a **specification document only**. Implementation will come after this design is reviewed and approved.

### Dependencies

- Existing health check infrastructure in `cli/src/internal/service/health.go`
- Service registry in `cli/src/internal/registry/`
- Service info package in `cli/src/internal/serviceinfo/`
- Output formatting utilities in `cli/src/internal/output/`

### Future Enhancements

- **Alerts**: Email/webhook notifications on health changes
- **Metrics Storage**: Store health history in database
- **Grafana Dashboard**: Pre-built dashboard for health visualization
- **Health Trends**: Show health trends over time
- **Predictive Alerts**: ML-based prediction of upcoming failures
- **Distributed Tracing**: Correlate health with distributed traces
- **Custom Health Checks**: Plugin system for custom health check logic

### MCP (Model Context Protocol) Integration

Health monitoring will be integrated with the MCP server (coming in separate PR) to provide AI-assisted development features:

**MCP Tool: `health_check`**
```json
{
  "name": "health_check",
  "description": "Check health status of services",
  "inputSchema": {
    "type": "object",
    "properties": {
      "service": {
        "type": "string",
        "description": "Optional service name to check"
      },
      "stream": {
        "type": "boolean",
        "description": "Enable streaming mode"
      }
    }
  }
}
```

**MCP Resource: `health://status`**
```json
{
  "uri": "health://status",
  "name": "Health Status",
  "description": "Real-time health status of all services",
  "mimeType": "application/json"
}
```

**Integration Points**:
- AI can query service health status
- AI can diagnose unhealthy services
- AI can correlate health with logs and errors
- AI can suggest fixes based on health patterns
- Real-time health updates via Server-Sent Events

**Use Cases**:
- "What services are unhealthy?" → AI queries health status
- "Why is the API failing?" → AI checks health + logs + recent changes
- "Monitor health while I develop" → AI streams health updates
- "Alert me if any service becomes unhealthy" → AI monitors via SSE
