# Health Monitoring

The `azd app` health monitor continuously checks service health every 5 seconds and provides real-time status updates through the dashboard.

## Health Check Hierarchy

The health monitor uses a three-tier approach to determine service health:

1. **Process Check** - Verifies the process is running and not zombie
2. **Port Check** - Tests if the service port is listening (most reliable)
3. **HTTP Health Check** - Calls `/health`, `/healthz`, or `/api/health` endpoints

## Error Detection

The health monitor scans service logs (last 30 seconds) for common error patterns and classifies them by severity:

### Critical Errors (⛔)

Critical errors indicate complete service failure and are shown in red:

**Process Crashes:**
- `panic:`, `fatal error:`, `fatal:`
- `segmentation fault`, `core dumped`, `stack overflow`

**Port/Network Failures:**
- `address already in use`, `eaddrinuse`
- `port.*already in use`, `failed to bind`
- `listen eaddrinuse`, `bind: address already in use`

**Startup Failures:**
- `application failed to start`, `failed to start`
- `startup failed`, `initialization failed`
- `bootstrap failed`, `failed to initialize`

**Module/Import Errors:**
- Node.js: `cannot find module`, `module not found`
- Python: `modulenotfounderror`, `importerror:`, `no module named`, `cannot import name`
- Java: `could not find or load main class`

**Database Connection:**
- `connection refused`, `econnrefused`
- `cannot connect to database`, `database connection failed`
- `connection timeout`, `no connection to server`

**Authentication:**
- `authentication failed`, `authorization failed`
- `access denied`, `permission denied`
- `credentials invalid`, `unauthorized`

**Out of Memory:**
- `out of memory`, `oom`, `outofmemoryerror`
- `cannot allocate memory`, `memory limit exceeded`

### Warnings (⚠️)

Warnings indicate potential issues but the service may still be functional. Shown in yellow:

**Configuration Issues:**
- `configuration error`, `config error`, `invalid configuration`
- `missing required`, `environment variable.*not set`

**Deprecation:**
- `deprecated`, `deprecation warning`

**Performance:**
- `timeout`, `request timeout`, `response timeout`
- `slow query`, `high memory usage`, `high cpu usage`

**API/Service Issues:**
- `service unavailable`, `503`, `502 bad gateway`, `504 gateway timeout`
- `api error`, `rate limit`, `too many requests`, `429`

**SSL/TLS:**
- `certificate`, `ssl error`, `tls handshake`
- `certificate verify failed`, `certificate has expired`

**File System:**
- `no such file or directory`, `enoent`, `file not found`
- `cannot read file`, `permission denied`
- `disk full`, `no space left`

**Docker/Container:**
- `container.*exited`, `container.*failed`
- `image not found`, `pull access denied`

**Cloud Provider:**
- `throttling`, `quota exceeded`
- `resource not found`, `service limit`

**Framework-Specific:**
- ASP.NET: `unhandled exception`, `system.exception`
- Express.js: `express deprecated`
- Django: `django.core.exceptions`, `operationalerror`
- Spring Boot: `error starting applicationcontext`, `bean creation exception`
- FastAPI: `validation error`, `422 unprocessable entity`

## Pattern Matching

Error patterns support basic regex syntax:
- Use `.*` for wildcards (e.g., `port.*already in use`)
- All matching is case-insensitive
- Patterns are checked in order (critical first, then warnings)

## Dashboard Display

### Service Card View
- Critical errors: Red banner with ⛔ icon and "Error Detected" header
- Warnings: Yellow banner with ⚠️ icon and "Warning Detected" header

### Table View
- Error messages shown below the status badge
- Critical: Red text with ⛔ emoji
- Warnings: Yellow text with ⚠️ emoji
- Hover to see full error message (truncated at ~150 chars)

## Implementation Details

**Location:** `cli/src/internal/healthmonitor/healthmonitor.go`

**Key Functions:**
- `checkLogsForErrors()` - Scans recent logs for error patterns
- `matchesPattern()` - Supports regex patterns with `.*` wildcards
- `truncateMessage()` - Intelligently truncates at word boundaries

**Scan Window:** Last 30 seconds of logs

**Message Length:** Truncated to ~150 characters with word boundary detection
