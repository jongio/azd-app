# Per-Project Dashboard Architecture

## Overview

Each project running with `azd app run` gets its own dedicated dashboard instance with a unique port. This allows multiple projects to run simultaneously, each with their own monitoring interface.

## Architecture

### Project Isolation

```
Project A (c:\projects\app-a)
  ├─ Services: frontend, backend
  ├─ Dashboard: http://localhost:3100
  └─ Port assignments: .azure/ports.json
      ├─ azd-app-dashboard: 3100
      ├─ frontend: 3000
      └─ backend: 8080

Project B (c:\projects\app-b)  
  ├─ Services: api, worker
  ├─ Dashboard: http://localhost:3101
  └─ Port assignments: .azure/ports.json
      ├─ azd-app-dashboard: 3101
      ├─ api: 3000
      └─ worker: 8081
```

### Key Features

1. **Project-Scoped Dashboards**
   - Each project directory gets its own dashboard server instance
   - Dashboard runs on a persistent port assigned by the port manager
   - Multiple projects can run simultaneously without conflicts

2. **Port Persistence**
   - Dashboard port is assigned like any other service
   - Stored in `.azure/ports.json` as service `azd-app-dashboard`
   - Prefers port 3100, but will use next available if taken
   - Same dashboard port used across runs for consistency

3. **Automatic Cleanup**
   - Dashboard stops when `azd app run` is interrupted
   - Port is released back to the port manager
   - Server instance is removed from memory

## Implementation Details

### Server Management

**Before (Singleton Pattern):**
```go
var (
    server     *Server      // Global singleton
    serverOnce sync.Once    // Ensures single initialization
)

func GetServer(projectDir string) *Server {
    serverOnce.Do(func() {
        server = &Server{...}
    })
    return server  // Same instance for all projects
}
```

**After (Multi-Instance Pattern):**
```go
var (
    servers   = make(map[string]*Server)  // Map: projectDir → Server
    serversMu sync.Mutex                   // Protects the map
)

func GetServer(projectDir string) *Server {
    serversMu.Lock()
    defer serversMu.Unlock()
    
    absPath, _ := filepath.Abs(projectDir)
    
    // Return existing server for this project
    if srv, exists := servers[absPath]; exists {
        return srv
    }
    
    // Create new server for this project
    srv := &Server{projectDir: absPath, ...}
    servers[absPath] = srv
    return srv
}
```

### Port Assignment

**Dashboard as a Service:**
```go
func (s *Server) Start() (string, error) {
    portMgr := portmanager.GetPortManager(s.projectDir)
    
    // Dashboard is treated like any other service
    // Preferred port: 3100, flexible (can change if in use)
    port, err := portMgr.AssignPort("azd-app-dashboard", 3100, false, true)
    
    s.port = port
    // ... start HTTP server on assigned port
}
```

### Cleanup on Stop

```go
func (s *Server) Stop() error {
    // Release port back to port manager
    portMgr := portmanager.GetPortManager(s.projectDir)
    portMgr.ReleasePort("azd-app-dashboard")
    
    // Remove from servers map
    serversMu.Lock()
    delete(servers, s.projectDir)
    serversMu.Unlock()
    
    // Stop HTTP server
    return s.server.Close()
}
```

## Port Manager Integration

The dashboard leverages the same port management system as application services:

### Port Assignment
```json
// .azure/ports.json
{
  "assignments": {
    "azd-app-dashboard": {
      "serviceName": "azd-app-dashboard",
      "port": 3100,
      "lastUsed": "2025-11-01T15:30:00Z"
    },
    "frontend": {
      "serviceName": "frontend",
      "port": 3000,
      "lastUsed": "2025-11-01T15:30:00Z"
    }
  }
}
```

### Benefits
- ✅ **Consistent URLs**: Same dashboard port across runs
- ✅ **Bookmark-friendly**: Can bookmark dashboard URL
- ✅ **No conflicts**: Multiple projects won't collide
- ✅ **Auto-cleanup**: Stale dashboard processes are detected and cleaned

## User Experience

### Single Project

```bash
cd c:\projects\my-app
azd app run

# Output:
Starting services...
✓ frontend: http://localhost:3000
✓ backend: http://localhost:8080

📊 Dashboard: http://localhost:3100
```

### Multiple Projects Simultaneously

**Terminal 1:**
```bash
cd c:\projects\project-a
azd app run

📊 Dashboard: http://localhost:3100
```

**Terminal 2:**
```bash
cd c:\projects\project-b
azd app run

📊 Dashboard: http://localhost:3101
```

**Both dashboards are accessible:**
- Project A: `http://localhost:3100` (shows services from project-a)
- Project B: `http://localhost:3101` (shows services from project-b)

### Dashboard Restart Detection

If dashboard port is already in use:

```
Port 3100 for service 'azd-app-dashboard' is in use by process 12345. Stop existing process? (y/N): y
Killing process 12345 on port 3100
✓ Dashboard started: http://localhost:3100
```

Or find alternative:

```
Port 3100 for service 'azd-app-dashboard' is in use by process 12345. Stop existing process? (y/N): N
Finding alternative port for azd-app-dashboard...
✓ Dashboard started: http://localhost:3101
```

## WebSocket Support

Each dashboard maintains its own WebSocket connections:

```javascript
// Frontend connects to project-specific dashboard
const ws = new WebSocket('ws://localhost:3100/api/ws');

// Only receives updates for services in this project
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    // data.services = services from this project only
};
```

## Technical Benefits

### Thread Safety
- Mutex-protected server map prevents race conditions
- Safe for concurrent access from multiple projects

### Memory Efficiency
- Servers are removed from map on shutdown
- No memory leaks from abandoned instances

### Port Efficiency
- Ports are released when dashboard stops
- Port manager can reassign ports to other services

### Clean Shutdown
- Signal handling stops dashboard gracefully
- All resources properly cleaned up

## Troubleshooting

### Dashboard Won't Start

**Error:** "Failed to assign port for dashboard"

**Causes:**
1. All ports in range 3100-9999 are in use
2. Permission issues binding to ports

**Solutions:**
```bash
# Check what's using ports
netstat -ano | findstr :3100   # Windows
lsof -i :3100                  # Mac/Linux

# Clean up port assignments
rm .azure/ports.json

# Try again
azd app run
```

### Dashboard Shows Wrong Services

**Cause:** Looking at dashboard from different project

**Solution:** Check the URL - each project has its own dashboard
```bash
# Project A dashboard
http://localhost:3100

# Project B dashboard
http://localhost:3101
```

### Dashboard Port Changes

**Cause:** Port assignment was lost or cleaned up

**Solution:** Dashboard port is persistent in `.azure/ports.json`
- Don't add this file to `.gitignore` if you want consistent ports
- Or set explicit port in azure.yaml (coming soon)

## Future Enhancements

### Explicit Dashboard Port Configuration

```yaml
# azure.yaml
name: my-app

dashboard:
  port: 3100  # Explicit port for dashboard (MANDATORY)

services:
  frontend:
    config:
      port: 3000
```

### Dashboard Discovery Service

```bash
# List all running dashboards
azd app dashboards

# Output:
Running Dashboards:
  • my-app-a: http://localhost:3100 (c:\projects\app-a)
  • my-app-b: http://localhost:3101 (c:\projects\app-b)
```

## Related Files

- **Dashboard Server:** `src/internal/dashboard/server.go`
- **Port Manager:** `src/internal/portmanager/portmanager.go`
- **Run Command:** `src/cmd/app/commands/run.go`
- **Port Persistence:** `.azure/ports.json` (per-project)
