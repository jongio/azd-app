# Port Management in azd app run

## Overview

The `azd app run` command uses a sophisticated port management system that:
- Honors explicit port configurations from `azure.yaml`
- Auto-detects ports from framework configurations
- Persists port assignments across runs
- Prompts before killing processes on ports
- Automatically cleans up stale processes

## Port Detection Priority

When determining which port to use for a service, the system follows this priority order:

### 1. **azure.yaml Config (EXPLICIT - MANDATORY)**
```yaml
services:
  frontend:
    config:
      port: 3000  # This port is MANDATORY and will never be changed
```

**Behavior:**
- ✅ If port is available → Use it immediately
- ⚠️ If port is in use → Prompt user to kill process
- ❌ If user declines → **FAIL** (no fallback, port is required)

**Why explicit ports are mandatory:**
Many applications have hardcoded port dependencies:
- Frontend apps expect API on specific port
- CORS configurations reference specific ports
- OAuth callbacks use fixed ports
- Development tools (debuggers, profilers) expect specific ports

### 2. **Framework Configuration Files**
Detected from project-specific config:
- **Node.js**: `package.json` scripts (`--port=3000`, `-p 3000`)
- **.NET**: `launchSettings.json` applicationUrl
- **Django**: `settings.py` (PORT or default 8000)
- **Spring Boot**: `application.properties` server.port
- **Aspire**: Detected from AppHost.cs service definitions

**Behavior:**
- ✅ Preferred but flexible
- 🔄 Will find alternative if port is in use

### 3. **Environment Variables**
- `PORT` - Standard port variable
- `{SERVICE_NAME}_PORT` - Service-specific override

### 4. **Framework Defaults**
Built-in defaults for common frameworks:
- Next.js, React, Vue: 3000
- Angular: 4200
- Django: 8000
- Flask: 5000
- .NET Aspire: Dynamic per-service
- Spring Boot: 8080

### 5. **Dynamic Assignment**
Find any available port in range 3000-9999

## Port Persistence

Port assignments are saved in `.azure/ports.json`:

```json
{
  "assignments": {
    "frontend": {
      "serviceName": "frontend",
      "port": 3000,
      "lastUsed": "2025-11-01T10:30:00Z"
    },
    "backend": {
      "serviceName": "backend",
      "port": 8080,
      "lastUsed": "2025-11-01T10:30:00Z"
    }
  }
}
```

**Benefits:**
- Same ports used across multiple runs
- Consistent development environment
- Easier debugging (ports don't change randomly)
- Better for documentation and team collaboration

## Port Cleanup

When `azd app run` starts, it automatically:

1. **Checks assigned ports** from `.azure/ports.json`
2. **Detects if ports are in use** using `netstat` (Windows) or `lsof` (Unix)
3. **Prompts before killing** processes:

```
Port 3000 for service 'frontend' is in use by process 12345. Stop existing process? (y/N):
```

4. **Handles user response:**
   - **`y` or `yes`** → Kill process and continue
   - **`N` or any other** → Find alternative port (or fail if explicit)

## User Prompt Behavior

### For Explicit Ports (azure.yaml config)

```
⚠️  Service 'frontend' requires port 3000 (configured in azure.yaml)
Port 3000 for service 'frontend' is in use by process 12345. Stop existing process? (y/N): N

Error: port 3000 is required for service 'frontend' but is in use and cannot be freed
```

**No fallback** - the port is mandatory.

### For Flexible Ports (auto-detected or default)

```
Port 3000 for service 'api' is in use by process 12345. Stop existing process? (y/N): N
Finding alternative port for api...
Assigned port 3001 to service 'api'
```

**Fallback available** - finds next available port.

## Configuration Examples

### Example 1: Explicit Ports (Recommended for production-like setups)

```yaml
name: my-app
services:
  frontend:
    project: ./web
    config:
      port: 3000  # Never changes, always 3000
  
  backend:
    project: ./api
    config:
      port: 8080  # Never changes, always 8080
```

**Use when:**
- CORS configuration references specific ports
- OAuth callbacks use fixed URLs
- Team uses same port conventions
- App code has hardcoded port references

### Example 2: Flexible Ports (Auto-detection)

```yaml
name: my-app
services:
  frontend:
    project: ./web
    # No port config - reads from package.json or uses defaults
  
  backend:
    project: ./api
    # No port config - reads from framework config
```

**Use when:**
- Ports can be dynamic
- Framework handles port configuration
- Local development only (no production dependencies)

### Example 3: Mixed Approach

```yaml
name: my-app
services:
  frontend:
    project: ./web
    config:
      port: 3000  # Explicit - OAuth callback is http://localhost:3000/callback
  
  backend:
    project: ./api
    # Flexible - can use any available port
    
  database-admin:
    project: ./admin
    # Flexible - just a dev tool, any port works
```

## Technical Implementation

### Port Manager (`portmanager.go`)

```go
// Signature
func AssignPort(serviceName string, preferredPort int, isExplicit bool, cleanStale bool) (int, error)

// isExplicit = true  → Port from azure.yaml, MUST be used
// isExplicit = false → Port is flexible, can find alternative
// cleanStale = true  → Prompt to kill processes on ports
```

### Port Detection (`port.go`)

```go
// Returns (port, isExplicit, error)
func DetectPort(serviceName string, service Service, projectDir string, 
                framework string, usedPorts map[int]bool) (int, bool, error)

// isExplicit = true when port comes from service.Config["port"]
```

### Service Detector (`detector.go`)

```go
// Calls DetectPort and passes isExplicit flag to AssignPort
preferredPort, isExplicit, _ := DetectPort(serviceName, service, projectDir, framework, usedPorts)
port, err := portMgr.AssignPort(serviceName, preferredPort, isExplicit, true)
```

## Best Practices

### ✅ DO

- **Use explicit ports** in `azure.yaml` when apps have port dependencies
- **Document why** specific ports are required (comments in azure.yaml)
- **Use consistent ports** across team (check in azure.yaml)
- **Let framework configs** handle flexible services
- **Review prompts** before killing processes (could be important services)

### ❌ DON'T

- **Hardcode ports** in app code without documenting in azure.yaml
- **Ignore prompts** to kill processes (verify it's safe first)
- **Use explicit ports** unnecessarily (makes local dev harder)
- **Mix port sources** (if in azure.yaml, don't also set in framework config)

## Troubleshooting

### Error: "port X is required but is in use"

**Cause:** Explicit port from azure.yaml is in use, user declined to kill process

**Solutions:**
1. Kill the process manually: `lsof -ti:3000 | xargs kill -9` (Mac/Linux) or `Stop-Process -Id (Get-NetTCPConnection -LocalPort 3000).OwningProcess` (Windows)
2. Find what's using the port and stop it properly
3. If port isn't truly required, remove from azure.yaml config

### Port keeps changing between runs

**Cause:** Port is not persisted or is flexible

**Solutions:**
1. Add explicit port to azure.yaml `config.port`
2. Check `.azure/ports.json` is not in `.gitignore`
3. Verify port detection is finding framework config

### Prompt appears every time

**Cause:** Old process left running from previous session

**Solutions:**
1. Say "yes" to kill the stale process
2. Check for zombie processes: `ps aux | grep node` (look for old instances)
3. Port manager will auto-cleanup if cleanStale=true

## Related Files

- **Port Detection:** `src/internal/service/port.go`
- **Port Manager:** `src/internal/portmanager/portmanager.go`
- **Service Detector:** `src/internal/service/detector.go`
- **Port Persistence:** `.azure/ports.json` (per-project)
- **Example Config:** `tests/projects/azure/azure-explicit-ports.yaml`
