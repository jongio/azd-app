# azd app logs

## Overview

The `logs` command displays output logs from running services for debugging and monitoring. It provides real-time log streaming, filtering, and formatting options to help developers troubleshoot and monitor their applications.

## Purpose

- **Log Viewing**: Display historical logs from running services
- **Real-Time Streaming**: Follow logs as they're generated (tail -f behavior)
- **Service Filtering**: View logs from specific services
- **Level Filtering**: Filter by log severity (info, warn, error, debug)
- **Time-Based Filtering**: Show logs from specific time ranges
- **Multiple Formats**: Output as text or JSON
- **File Output**: Save logs to file for analysis

## Command Usage

```bash
azd app logs [service-name] [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--follow` | `-f` | bool | `false` | Follow log output (tail -f behavior) |
| `--service` | `-s` | string | | Filter by service name(s) (comma-separated) |
| `--tail` | `-n` | int | `100` | Number of lines to show from the end |
| `--since` | | string | | Show logs since duration (e.g., 5m, 1h) |
| `--timestamps` | | bool | `true` | Show timestamps with each log entry |
| `--no-color` | | bool | `false` | Disable colored output |
| `--level` | | string | `all` | Filter by log level (info, warn, error, debug, all) |
| `--format` | | string | `text` | Output format (text, json) |
| `--output` | | string | | Write logs to file instead of stdout |
| `--exclude` | `-e` | string | | Regex patterns to exclude (comma-separated) |
| `--no-builtins` | | bool | `false` | Disable built-in filter patterns |

## Execution Flow

### Basic Log Retrieval

```
┌─────────────────────────────────────────────────────────────┐
│                  azd app logs                                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Get Current Working Directory                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Initialize Log Manager                                      │
│  - Get singleton instance for project                        │
│  - Access log buffers for all services                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check Running Services                                      │
│  - Query service registry                                    │
│  - List available service names                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
           No Services         Has Services
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────────┐
        │ Show Message     │ │ Continue        │
        │ "Run azd app run"│ │                 │
        └──────────────────┘ └─────────────────┘
                                      ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Service Filter                                        │
│  - From positional argument OR --service flag                │
│  - Split comma-separated list                                │
│  - Validate services exist                                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Log Level Filter                                      │
│  - Convert string to LogLevel enum                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Parse Time Range (--since)                                  │
│  - Convert duration string to time.Time                      │
│  - Examples: "5m", "1h", "30s"                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Setup Output Writer                                         │
│  - Default: stdout                                           │
│  - If --output: create file                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Retrieve Logs from Buffer(s)                                │
│  - If no filter: get from all services                       │
│  - If filtered: get from specific services                   │
│  - Apply --tail or --since limit                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Filter Logs by Level                                        │
│  - Apply log level filter (if specified)                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Display Logs                                                │
│  - Format: text or JSON                                      │
│  - Output: stdout or file                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
              --follow?              No
                    │                │
                    ↓                ↓
        ┌──────────────────┐ ┌─────────────┐
        │ Start Following  │ │   Done      │
        │ (see below)      │ │             │
        └──────────────────┘ └─────────────┘
```

### Follow Mode Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Subscribe to Log Streams                                    │
│  - For each service (or filtered services)                   │
│  - Create channel for log entries                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Setup Signal Handling                                       │
│  - Listen for Ctrl+C (SIGINT)                                │
│  - Enable graceful shutdown                                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Merge Subscription Channels                                 │
│  - Combine all service channels into single stream           │
│  - Preserve ordering by timestamp                            │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  ┌────────────────────────────────────────────────────┐     │
│  │  Event Loop (runs until Ctrl+C)                    │     │
│  ├────────────────────────────────────────────────────┤     │
│  │                                                     │     │
│  │  Wait for:                                          │     │
│  │   - New log entry from channel                      │     │
│  │   - Shutdown signal (Ctrl+C)                        │     │
│  │                                                     │     │
│  │  On log entry:                                      │     │
│  │   1. Apply level filter                             │     │
│  │   2. Format (text or JSON)                          │     │
│  │   3. Display immediately                            │     │
│  │                                                     │     │
│  │  On shutdown signal:                                │     │
│  │   1. Unsubscribe from all channels                  │     │
│  │   2. Clean up resources                             │     │
│  │   3. Exit gracefully                                │     │
│  │                                                     │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## Log Buffer System

### Log Collection

Services send logs to buffers managed by the LogManager. Log filtering is applied at collection time, so noisy messages never enter the buffer:

```
┌─────────────────────────────────────────────────────────────┐
│  Service Process                                             │
│  - Stdout stream                                             │
│  - Stderr stream                                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Collector (per service)                                 │
│  - Read from stdout/stderr                                   │
│  - Parse log level from output                               │
│  - Create LogEntry with metadata                             │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Filter (applied at collection time)                     │
│  - Built-in patterns (Electron, npm, Node.js noise)         │
│  - azure.yaml logs.filters.exclude patterns                 │
│  - Noisy messages are dropped before buffer                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Service Log Buffer                                          │
│  - Ring buffer (circular, fixed size)                        │
│  - Stores recent entries (after filtering)                   │
│  - Supports subscriptions (for follow mode)                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Manager                                                 │
│  - Central registry of all buffers                           │
│  - Query interface for commands                              │
│  - Loads filter config from azure.yaml at startup            │
└─────────────────────────────────────────────────────────────┘
```

### LogEntry Structure

Each log entry contains:

```go
type LogEntry struct {
    Service    string        // Service name (e.g., "web")
    Message    string        // Log message content
    Timestamp  time.Time     // When log was generated
    Level      LogLevel      // Severity (info/warn/error/debug)
    IsStderr   bool          // From stderr stream?
}
```

### Log Levels

```
┌─────────────────────────────────────────────────────────────┐
│  Log Level Hierarchy                                         │
├─────────────────────────────────────────────────────────────┤
│  DEBUG   (-1)  Most verbose, development info                │
│  INFO    (0)   General information                           │
│  WARN    (1)   Warning messages                              │
│  ERROR   (2)   Error messages                                │
│  ALL     (*)   All levels (default)                          │
└─────────────────────────────────────────────────────────────┘
```

**Level Detection**:

Logs are automatically assigned levels based on content patterns:

| Pattern | Level | Example |
|---------|-------|---------|
| From stderr | ERROR | Any stderr output |
| Contains "error", "fail" | ERROR | "Failed to connect" |
| Contains "warn" | WARN | "Warning: deprecated" |
| Contains "debug", "trace" | DEBUG | "Debug: processing..." |
| Default | INFO | "Server started" |

## Service Filtering

### Single Service

```bash
# Positional argument
azd app logs web

# Or --service flag
azd app logs --service web
```

### Multiple Services

```bash
# Comma-separated list
azd app logs --service web,api
```

**Filter Flow**:

```
Available services:      Filter: "web,api"        Result:
- web                   ──────────────────►       - web
- api                                             - api
- worker
- cache
```

## Time-Based Filtering

### Using --since

Show logs from the last N duration:

```bash
# Last 5 minutes
azd app logs --since 5m

# Last 1 hour
azd app logs --since 1h

# Last 30 seconds
azd app logs --since 30s

# Last 2 days
azd app logs --since 48h
```

**Calculation**:

```
Current Time:        2024-11-04 10:30:00
--since 5m:          2024-11-04 10:25:00
                     ↓
Filter:              timestamp >= 10:25:00
```

### Using --tail

Show last N lines (default: 100):

```bash
# Last 50 lines
azd app logs --tail 50

# Last 200 lines
azd app logs --tail 200
```

## Output Formats

### Text Format (Default)

Human-readable format with optional colors and timestamps:

```
[10:30:45.123] [web] Server started on port 3000
[10:30:46.456] [api] Connected to database
[10:30:47.789] [api] ERROR: Failed to load config
[10:30:48.012] [web] GET /api/users 200 12ms
```

**Color Coding**:
- **Timestamps**: Gray
- **Service names**: Cyan
- **Error messages**: Red
- **Warning messages**: Yellow
- **Debug messages**: Gray
- **Info messages**: Default

**Format String**:
```
[timestamp] [service] message
│          │         └─ Log content (colored by level)
│          └─────────── Service name (cyan)
└────────────────────── Timestamp (gray)
```

### JSON Format

Machine-readable format for parsing and analysis:

```bash
azd app logs --format json
```

**Output**:
```json
{"service":"web","message":"Server started on port 3000","timestamp":"2024-11-04T10:30:45.123Z","level":0,"isStderr":false}
{"service":"api","message":"Connected to database","timestamp":"2024-11-04T10:30:46.456Z","level":0,"isStderr":false}
{"service":"api","message":"ERROR: Failed to load config","timestamp":"2024-11-04T10:30:47.789Z","level":2,"isStderr":true}
```

**Fields**:
| Field | Type | Description |
|-------|------|-------------|
| `service` | string | Service name |
| `message` | string | Log message |
| `timestamp` | string | ISO 8601 timestamp |
| `level` | int | Log level (-1=debug, 0=info, 1=warn, 2=error) |
| `isStderr` | bool | From stderr stream |

## Follow Mode

### Real-Time Streaming

```bash
# Follow all services
azd app logs --follow

# Follow specific service
azd app logs -f --service web

# Follow with filters
azd app logs -f --level error
```

**Behavior**:
- Displays existing logs first (respecting --tail)
- Then streams new logs as they arrive
- Updates in real-time
- Continues until Ctrl+C

**Subscription Mechanism**:

```
┌─────────────────────────────────────────────────────────────┐
│  Log Buffer (per service)                                    │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Ring Buffer: [entry1, entry2, ..., entryN]        │     │
│  └────────────────────────────────────────────────────┘     │
│                         ↓                                    │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Subscribers (channels)                             │     │
│  │  - Channel 1 (logs command, user A)                │     │
│  │  - Channel 2 (logs command, user B)                │     │
│  │  - Channel 3 (dashboard)                            │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
│  When new log arrives:                                       │
│   → Add to buffer                                            │
│   → Send to all subscribers                                  │
└─────────────────────────────────────────────────────────────┘
```

## File Output

### Save Logs to File

```bash
# Save to file
azd app logs --output debug.log

# Follow and save
azd app logs -f --output live.log

# JSON to file
azd app logs --format json --output data.jsonl
```

**Security**: Output path is validated to prevent path traversal attacks

## Common Use Cases

### 1. View Recent Logs

```bash
# Last 100 lines (default)
azd app logs

# Last 50 lines
azd app logs --tail 50
```

### 2. Monitor Specific Service

```bash
# Follow web service logs
azd app logs -f --service web
```

### 3. Debugging Errors

```bash
# Show only errors
azd app logs --level error

# Follow errors in real-time
azd app logs -f --level error
```

### 4. Time-Based Investigation

```bash
# Logs from last 10 minutes
azd app logs --since 10m

# Errors from last hour
azd app logs --since 1h --level error
```

### 5. Export for Analysis

```bash
# Export as JSON
azd app logs --format json --output logs.jsonl

# Can then analyze with jq:
cat logs.jsonl | jq 'select(.level == 2)'
```

### 6. Multi-Service Monitoring

```bash
# Monitor web and api together
azd app logs -f --service web,api

# Useful for debugging interactions
```

## Advanced Filtering

### Combining Filters

Filters can be combined for precise log retrieval:

```bash
# Service + Level + Time
azd app logs --service api --level error --since 5m

# Multiple services + Follow + No timestamps
azd app logs -f --service web,api --timestamps=false

# Save errors from last hour to file
azd app logs --level error --since 1h --output errors.log
```

**Filter Application Order**:

```
1. Service Filter    (select services)
    ↓
2. Time Filter      (--since or --tail)
    ↓
3. Level Filter     (--level)
    ↓
4. Pattern Filter   (--exclude, azure.yaml logFilters, built-ins)
    ↓
5. Format/Display   (--format, --timestamps, --no-color)
```

## Pattern-Based Filtering

### Overview

Pattern-based filtering suppresses noisy log output that doesn't indicate real problems. Patterns are applied using case-insensitive regex matching.

**Filtering happens at two levels:**
1. **Collection time** (`azd app run`): Noisy messages are filtered out before they enter the log buffer or are written to log files
2. **Display time** (`azd app logs --exclude`): Additional command-line patterns for ad-hoc filtering

### Built-In Patterns

By default, azd app logs suppresses common noise patterns:

| Pattern | Description |
|---------|-------------|
| `Request Autofill.enable failed` | Electron DevTools protocol errors |
| `npm warn Unknown env config` | npm registry credential warnings |
| `Debugger listening on ws://` | Node.js debugger messages |
| `ExperimentalWarning:` | Node.js experimental feature warnings |
| `DeprecationWarning:` | Node.js deprecation warnings |

### Command-Line Filtering

```bash
# Exclude specific patterns
azd app logs --exclude "pattern1,pattern2"

# Disable built-in patterns (show all logs)
azd app logs --no-builtins

# Combine custom patterns with built-ins
azd app logs --exclude "my custom pattern"
```

### Azure.yaml Configuration

Configure project-level log settings in `azure.yaml` under the `logs` section:

```yaml
# Logging configuration
logs:
  filters:
    exclude:
      - "Optional service not available"
      - "Cache miss"
      - "Retry attempt"
    includeBuiltins: true  # Include built-in patterns (default: true)
```

Service-level logging configuration:

```yaml
services:
  web:
    project: ./frontend
    logs:
      filters:
        exclude:
          - "Hot Module Replacement"
```

### Filter Priority

1. **Command-line `--exclude`**: Always applied
2. **Command-line `--no-builtins`**: Overrides azure.yaml and defaults
3. **azure.yaml `logs.filters`**: Project-level patterns
4. **Built-in patterns**: Applied by default unless disabled

### Examples

```bash
# Suppress Ollama connection failures
azd app logs --exclude "Ollama not available"

# Show only real errors (no warnings, no noise)
azd app logs --level error

# See all logs including noise (debugging filters)
azd app logs --no-builtins
```

## Timestamps

### Timestamp Control

```bash
# With timestamps (default)
azd app logs
[10:30:45.123] [web] Server started

# Without timestamps
azd app logs --timestamps=false
[web] Server started
```

**Format**: `HH:MM:SS.mmm` (24-hour with milliseconds)

## Color Output

### Color Modes

```bash
# With colors (default)
azd app logs

# No colors (for piping or terminals without color support)
azd app logs --no-color

# Automatically disabled when piping
azd app logs | grep error
```

## Integration with Service Registry

The logs command integrates with the service registry:

```
┌─────────────────────────────────────────────────────────────┐
│  Service Registry                                            │
│  - Tracks running services                                   │
│  - Maps service names to log buffers                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Log Manager                                                 │
│  - Gets list of services from registry                       │
│  - Access log buffers                                        │
│  - Validates service names                                   │
└─────────────────────────────────────────────────────────────┘
```

**Benefits**:
- Only shows logs from currently running services
- Validates service names before querying
- Automatically updates when services start/stop

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| No services running | `azd app run` not active | Run `azd app run` first |
| Service not found | Invalid service name | Check `azd app info` for service list |
| Invalid duration | Bad --since format | Use format like "5m", "1h", "30s" |
| Permission denied | Can't write to --output | Check file permissions |

**Example Error**:
```bash
$ azd app logs --service invalid

Error: service 'invalid' not found (available: web, api, worker)
```

## Performance Considerations

### Buffer Size

- Log buffers are **ring buffers** (circular, fixed size)
- Default capacity: **1000 entries per service**
- Older entries are dropped when buffer is full
- Follow mode subscribes to live stream (no buffer limit)

### Memory Usage

```
Per Service:
  Buffer size: 1000 entries
  Avg entry size: ~200 bytes
  Memory: ~200 KB per service

For 10 services: ~2 MB total
```

### Follow Mode Overhead

- **Minimal CPU**: Event-driven (no polling)
- **Minimal Memory**: Only buffered entries waiting to display
- **Network**: N/A (local only)

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Logs displayed successfully |
| 1 | Failure | Error retrieving or displaying logs |
| 2 | Validation | Invalid arguments or service not found |

## Best Practices

1. **Use Follow for Development**: `azd app logs -f` for real-time monitoring
2. **Filter by Service**: Reduce noise when debugging specific services
3. **Export Errors**: Save error logs for later analysis
4. **Combine with grep**: `azd app logs | grep "pattern"` for ad-hoc searching
5. **JSON for Automation**: Use `--format json` in scripts
6. **Tail for Performance**: Use `--tail` to limit output for large log volumes

## Related Commands

- [`azd app run`](./run.md) - Start services that generate logs
- [`azd app info`](./info.md) - Show running services

## Examples

### Example 1: Basic Log Viewing

```bash
$ azd app logs

[10:30:45.123] [web] Server listening on port 3000
[10:30:45.456] [api] Database connected
[10:30:46.789] [web] GET / 200 15ms
[10:30:47.012] [api] GET /api/users 200 8ms
```

### Example 2: Follow Specific Service

```bash
$ azd app logs -f --service web

[10:30:45.123] [web] Server listening on port 3000
[10:30:46.789] [web] GET / 200 15ms
[10:30:48.234] [web] GET /about 200 5ms
^C
```

### Example 3: Error Investigation

```bash
$ azd app logs --level error --since 1h

[09:45:23.456] [api] ERROR: Connection timeout to database
[10:12:34.789] [worker] ERROR: Failed to process job #1234
[10:25:45.012] [api] ERROR: Validation failed for user input
```

### Example 4: JSON Export

```bash
$ azd app logs --format json --output logs.jsonl

$ cat logs.jsonl | jq -r 'select(.level == 2) | .message'
ERROR: Connection timeout to database
ERROR: Failed to process job #1234
ERROR: Validation failed for user input
```

### Example 5: Multi-Service Monitoring

```bash
$ azd app logs -f --service web,api --timestamps=false

[web] Server started
[api] Database connected
[web] GET /api/users → proxying to api
[api] GET /api/users 200 12ms
[web] GET /api/users 200 15ms
```

### Example 6: Debugging with Filters

```bash
$ azd app logs --service api --level warn --since 30m --no-color

[10:15:23.456] [api] Warning: Slow query detected (250ms)
[10:28:34.789] [api] Warning: Cache miss rate high (75%)
[10:35:45.012] [api] Warning: Connection pool nearly exhausted
```
