# azd app info

## Overview

The `info` command displays comprehensive information about running services, including URLs, status, health, metadata, and environment variables. It provides a unified view of both local development and Azure-deployed services.

## Purpose

- **Service Discovery**: Find all defined and running services
- **Status Monitoring**: Check service health and status
- **URL Access**: Get local and Azure URLs for services
- **Environment Inspection**: View service-specific environment variables
- **Metadata Display**: Show service configuration and runtime information
- **Cross-Project Visibility**: Optionally view services from all projects

## Command Usage

```bash
azd app info [flags]
```

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | | bool | `false` | Show services from all projects on this machine |
| `--cwd` | `-C` | string | | Sets the current working directory |

## Execution Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    azd app info                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Determine Project Directory                                 │
│  - Default: Current directory                                │
│  - If --cwd: Use specified directory                         │
│  - If --all: Get all registries (future)                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Get Service Registry                                        │
│  - Load from .azure/registry/services.json                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Validate & Clean Stale Services                             │
│  - Check if port is still listening                          │
│  - Check if PID is still running                             │
│  - Remove stale entries                                      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Get Service Information                                     │
│  - Merge azure.yaml definitions                              │
│  - Merge registry runtime data                               │
│  - Merge Azure environment variables                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Get Azure Environment Values                                │
│  - Execute: azd env get-values --output json                 │
│  - Parse environment variables                               │
│  - Extract service URLs and config                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Display Service Information                                 │
│  - Format: text (default) or JSON                            │
│  - Show all service details                                  │
└─────────────────────────────────────────────────────────────┘
```

## Service Information Sources

### Data Merging

The `info` command merges data from multiple sources:

```
┌─────────────────────────────────────────────────────────────┐
│  azure.yaml (Service Definitions)                            │
│  - Service name                                              │
│  - Language                                                  │
│  - Project directory                                         │
│  - Host configuration                                        │
└─────────────────────────────────────────────────────────────┘
                            +
┌─────────────────────────────────────────────────────────────┐
│  Service Registry (Runtime State)                            │
│  - Running status (starting/running/error/stopped)           │
│  - Health status (healthy/unhealthy/unknown)                 │
│  - Port number                                               │
│  - Process ID (PID)                                          │
│  - Start time                                                │
│  - Last checked time                                         │
└─────────────────────────────────────────────────────────────┘
                            +
┌─────────────────────────────────────────────────────────────┐
│  Azure Environment (azd context)                             │
│  - Azure URLs (deployed endpoints)                           │
│  - Resource names                                            │
│  - Container image names                                     │
│  - Service-specific environment variables                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Unified ServiceInfo                                         │
│  - Complete service information                              │
│  - Both local and Azure data                                 │
└─────────────────────────────────────────────────────────────┘
```

### ServiceInfo Structure

```go
type ServiceInfo struct {
    Name      string              // Service name
    Language  string              // Programming language
    Framework string              // Framework/package manager
    Project   string              // Project directory path
    
    Local     *LocalServiceInfo   // Local development info
    Azure     *AzureServiceInfo   // Azure deployment info
    
    EnvironmentVars map[string]string  // Service env vars
}

type LocalServiceInfo struct {
    Status      string     // starting/running/error/stopped
    Health      string     // healthy/unhealthy/unknown
    Port        int        // Local port
    URL         string     // Local URL
    PID         int        // Process ID
    StartTime   *time.Time // When started
    LastChecked *time.Time // Last health check
}

type AzureServiceInfo struct {
    URL          string  // Azure endpoint URL
    ResourceName string  // Azure resource name
    ImageName    string  // Container image name
}
```

## Service Validation

### Port and Process Checking

The command validates running services by checking:

```
┌─────────────────────────────────────────────────────────────┐
│  For Each Registered Service:                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check if Port is Listening                                  │
│  - Windows: netstat -an | findstr ":PORT.*LISTENING"         │
│  - Unix: netstat -ln | grep ":PORT.*LISTEN"                  │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│  Check if Process is Running                                 │
│  - Windows: tasklist /FI "PID eq {pid}"                      │
│  - Unix: os.FindProcess + signal 0                           │
└─────────────────────────────────────────────────────────────┘
                            ↓
                    ┌───────┴────────┐
                    │                │
        Port Listening?      Port Not Listening
                    │                │
                    ↓                ↓
            ┌─────────────┐   ┌──────────────────┐
            │ Service      │   │ Service Not      │
            │ RUNNING      │   │ Running → Remove │
            │ Update health│   │ from Registry    │
            └─────────────┘   └──────────────────┘
```

**Rationale**: Port listening is more reliable than PID checking because:
- PIDs can be reused by the OS
- A process may exist but not be serving on the port
- Port availability directly indicates service availability

## Status and Health Indicators

### Status Values

| Status | Icon | Meaning | Color |
|--------|------|---------|-------|
| `running` | ✓ | Service is active and responding | Green |
| `starting` | ○ | Service is initializing | Yellow |
| `error` | ✗ | Service failed to start or crashed | Red |
| `stopped` | ● | Service intentionally stopped | Gray |
| `unknown` | ? | Status cannot be determined | Yellow |

### Health Values

| Health | Meaning | Color |
|--------|---------|-------|
| `healthy` | Service responding normally | Green |
| `unhealthy` | Service not responding to health checks | Red |
| `unknown` | Health status not available | Yellow |

## Environment Variables

### Service-Specific Variables

The command displays environment variables relevant to each service:

```
Pattern Matching:
  SERVICE_{SERVICENAME}_*     (highest priority)
  AZURE_{SERVICENAME}_*       (Azure-specific)

Examples for service "web":
  SERVICE_WEB_URL=https://web.azurewebsites.net
  SERVICE_WEB_ENDPOINT_URL=https://web-api.azurewebsites.net
  AZURE_WEB_RESOURCE_NAME=web-xyz
```

### Azure Environment Integration

```bash
# Behind the scenes, info command executes:
azd env get-values --output json

# Then parses environment variables:
{
  "AZURE_SUBSCRIPTION_ID": "abc123...",
  "SERVICE_WEB_URL": "https://web.azurewebsites.net",
  "SERVICE_API_URL": "https://api.azurewebsites.net",
  ...
}
```

## Output Formats

### Text Format (Default)

Human-readable format with icons and colors:

```
📦 Project: /path/to/project

  ✓ web
    Local URL: http://localhost:3000
    Azure URL: https://web-xyz.azurewebsites.net
    Azure Resource: web-xyz
    Language: js
    Framework: pnpm
    Project: ./src/web
    Port: 3000
    PID: 12345
    Started: 5m ago
    Status: running
    Health: healthy
    
    Environment Variables:
      SERVICE_WEB_URL = https://web-xyz.azurewebsites.net
      SERVICE_WEB_ENDPOINT_URL = https://web-xyz.azurewebsites.net

──────────────────────────────────────────────────────────────

  ○ api
    Local URL: http://localhost:3001 (not running)
    Azure URL: https://api-xyz.azurewebsites.net
    Language: python
    Framework: uv
    Project: ./src/api
    Status: starting
    Health: unknown
```

### JSON Format

Machine-readable format matching dashboard API schema:

```bash
azd app info --output json
```

```json
{
  "project": "/path/to/project",
  "services": [
    {
      "name": "web",
      "language": "js",
      "framework": "pnpm",
      "project": "./src/web",
      "local": {
        "status": "running",
        "health": "healthy",
        "port": 3000,
        "url": "http://localhost:3000",
        "pid": 12345,
        "startTime": "2024-11-04T10:25:00Z",
        "lastChecked": "2024-11-04T10:30:00Z"
      },
      "azure": {
        "url": "https://web-xyz.azurewebsites.net",
        "resourceName": "web-xyz",
        "imageName": "myacr.azurecr.io/web:latest"
      },
      "environmentVars": {
        "SERVICE_WEB_URL": "https://web-xyz.azurewebsites.net"
      }
    }
  ]
}
```

## Project Scoping

### Current Project (Default)

```bash
# Show services in current directory
azd app info
```

### Specific Project

```bash
# Show services in another project
azd app info --cwd /path/to/other/project
```

### All Projects (Future)

```bash
# Show services from all projects on this machine
azd app info --all
```

**Use Case**: Developers working on multiple projects simultaneously

## Common Use Cases

### 1. Check Service Status

```bash
$ azd app info

📦 Project: /Users/dev/myapp

  ✓ web
    Local URL: http://localhost:3000
    Status: running
    Health: healthy

  ✓ api
    Local URL: http://localhost:3001
    Status: running
    Health: healthy
```

### 2. Get Service URLs

```bash
$ azd app info | grep "URL"

    Local URL: http://localhost:3000
    Azure URL: https://web.azurewebsites.net
    Local URL: http://localhost:3001
    Azure URL: https://api.azurewebsites.net
```

### 3. Verify Deployment

```bash
$ azd app info --output json | jq '.services[] | {name, azure}'

{
  "name": "web",
  "azure": {
    "url": "https://web-xyz.azurewebsites.net",
    "resourceName": "web-xyz",
    "imageName": "myacr.azurecr.io/web:latest"
  }
}
```

### 4. Check Environment Config

```bash
$ azd app info

  ✓ api
    Environment Variables:
      SERVICE_API_URL = https://api.azurewebsites.net
      SERVICE_API_DATABASE_URL = postgresql://...
```

### 5. Debug Service Issues

```bash
$ azd app info

  ✗ worker
    Local URL: http://localhost:3002 (not running)
    Status: error
    Health: unhealthy
    
# Service failed - check logs
$ azd app logs --service worker
```

## Integration with Other Commands

### With `azd app run`

```bash
# Terminal 1: Start services
azd app run

# Terminal 2: Check status
azd app info
```

### With `azd app logs`

```bash
# Get service list
azd app info

# View logs for specific service
azd app logs --service web
```

### With Dashboard

The dashboard uses the same data source:

```
azd app info (CLI)
       ↓
ServiceInfo API
       ↓
Dashboard (Web UI)
```

Both access the unified `serviceinfo` package for consistency.

## Service Registry

### Registry Location

```
project-root/
  .azure/
    registry/
      services.json
```

### Registry Structure

```json
{
  "web": {
    "name": "web",
    "projectDir": "/path/to/project",
    "port": 3000,
    "url": "http://localhost:3000",
    "azureUrl": "https://web.azurewebsites.net",
    "language": "js",
    "framework": "pnpm",
    "status": "running",
    "health": "healthy",
    "pid": 12345,
    "startTime": "2024-11-04T10:25:00Z"
  }
}
```

**Registry Management**:
- Updated by `azd app run` when services start
- Cleaned by `azd app info` (removes stale entries)
- Queried by `azd app logs` for service discovery

## Time Formatting

### Relative Time

For recent events (< 1 day):

| Duration | Display |
|----------|---------|
| < 1 minute | "30s ago" |
| < 1 hour | "15m ago" |
| < 24 hours | "5h ago" |

### Absolute Time

For older events (≥ 1 day):

```
2024-11-04 10:25:00
```

## Error Handling

### Common Scenarios

| Scenario | Behavior |
|----------|----------|
| No azure.yaml | Shows message with suggestion |
| No running services | Shows available services as "not running" |
| Stale registry entries | Automatically cleaned and removed |
| Azure CLI not available | Shows local info only |

### Example: No Services Running

```bash
$ azd app info

📦 Project: /Users/dev/myapp

No services are currently running

Run 'azd app run' to start services
```

### Example: No azure.yaml

```bash
$ azd app info

📦 Project: /Users/dev/myapp

No services defined in azure.yaml

Run 'azd app reqs --generate' to create azure.yaml with service definitions
```

## Performance Considerations

### Validation Overhead

```
Per service validation:
  - Port check: ~10ms
  - PID check: ~5ms
  - Total: ~15ms per service

For 10 services: ~150ms
```

**Optimization**: Validation runs in parallel where possible

### Registry Cleanup

- Happens automatically on every `info` invocation
- Removes only stale entries (port not listening)
- Minimal performance impact

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Information displayed successfully |
| 1 | Failure | Error accessing registry or azure.yaml |

## Best Practices

1. **Check Before Debugging**: Run `info` before investigating issues
2. **Use JSON for Automation**: Parse JSON output in scripts
3. **Verify Deployments**: Check Azure URLs after deployment
4. **Monitor Health**: Watch for health status changes
5. **Cross-Reference with Logs**: Use `info` to identify services, then check `logs`

## Related Commands

- [`azd app run`](./run.md) - Start services (populates registry)
- [`azd app logs`](./logs.md) - View service logs
- `azd env get-values` - Get Azure environment variables

## Examples

### Example 1: Full Service Information

```bash
$ azd app info

📦 Project: /Users/dev/fullstack-app

  ✓ web
    Local URL: http://localhost:3000
    Azure URL: https://web-abc123.azurewebsites.net
    Azure Resource: web-abc123
    Docker Image: myacr.azurecr.io/web:latest
    Language: js
    Framework: pnpm
    Project: ./frontend
    Port: 3000
    PID: 45678
    Started: 10m ago
    Checked: 2s ago
    Status: running
    Health: healthy
    
    Environment Variables:
      SERVICE_WEB_URL = https://web-abc123.azurewebsites.net

──────────────────────────────────────────────────────────────

  ✓ api
    Local URL: http://localhost:3001
    Azure URL: https://api-abc123.azurewebsites.net
    Azure Resource: api-abc123
    Language: python
    Framework: uv
    Project: ./backend
    Port: 3001
    PID: 45679
    Started: 10m ago
    Status: running
    Health: healthy
```

### Example 2: JSON Output for Automation

```bash
$ azd app info --output json | jq '.services[] | {name, status: .local.status, url: .azure.url}'

{
  "name": "web",
  "status": "running",
  "url": "https://web-abc123.azurewebsites.net"
}
{
  "name": "api",
  "status": "running",
  "url": "https://api-abc123.azurewebsites.net"
}
```

### Example 3: Specific Project

```bash
$ azd app info --cwd /Users/dev/other-project

📦 Project: /Users/dev/other-project

  ✓ service1
    Local URL: http://localhost:4000
    Status: running
```

### Example 4: Service Not Running

```bash
$ azd app info

📦 Project: /Users/dev/myapp

  ● web
    Local URL: http://localhost:3000 (not running)
    Azure URL: https://web.azurewebsites.net
    Language: js
    Project: ./src/web
    Status: stopped
```
