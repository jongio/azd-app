# AZD App Dashboard - Implementation Summary

## Overview
Implemented an embedded web dashboard and service registry system for the azd app extension. The system tracks running services across all projects on a machine and provides both CLI and web interfaces for monitoring.

## What Was Built

### 1. Service Registry (`src/internal/registry/registry.go`)
Global persistent registry for tracking running services across all projects.

**Features:**
- **Persistent Storage**: Saves to `~/.azd/app/services.json`
- **Cross-Process Tracking**: Multiple `azd app` instances can query the same registry
- **Automatic Cleanup**: Removes stale entries for stopped processes
- **Project Filtering**: List services by project directory

**Data Structure:**
```go
type ServiceRegistryEntry struct {
    Name        string    // Service name
    ProjectDir  string    // Absolute path to project
    PID         int       // Process ID
    Port        int       // Port number
    URL         string    // Service URL
    Language    string    // Language (js, python, etc.)
    Framework   string    // Framework (nextjs, django, etc.)
    Status      string    // starting, ready, stopping, stopped, error
    Health      string    // healthy, unhealthy, unknown
    StartTime   time.Time // When service started
    LastChecked time.Time // Last health check
    Error       string    // Error message if any
}
```

**API:**
- `GetRegistry()` - Get global singleton instance
- `Register(entry)` - Add/update service
- `Unregister(projectDir, serviceName)` - Remove service
- `UpdateStatus(projectDir, serviceName, status, health)` - Update status
- `GetService(projectDir, serviceName)` - Get single service
- `ListAll()` - Get all services from all projects
- `ListByProject(projectDir)` - Get services for specific project
- `Clear()` - Remove all entries

### 2. URLs Command (`src/cmd/app/commands/urls.go`)
CLI command to list running services and their URLs.

**Usage:**
```bash
# Show services in current project
azd app urls

# Show all services from all projects
azd app urls --all

# Show services from specific project
azd app urls --project /path/to/project
```

**Output:**
```
[Project: ./my-app]
────────────────────────────────────────────────────────────
  ✓ web                  http://localhost:3000
     nextjs        js (ready)
  ✓ api                  http://localhost:8000
     fastapi       python (ready)
```

**Status Icons:**
- `✓` (green) - Ready & healthy
- `○` (yellow) - Starting
- `✗` (red) - Error or unhealthy
- `●` (gray) - Stopped
- `?` (yellow) - Unknown status

### 3. Dashboard Server (`src/internal/dashboard/server.go`)
Embedded HTTP server with REST API and WebSocket support.

**Features:**
- **Embedded Static Files**: Uses `embed.FS` to bundle HTML/CSS/JS in binary
- **Auto Port Selection**: Finds available port in range 3100-3200
- **REST API Endpoints**:
  - `GET /api/services` - Services for current project (JSON)
  - `GET /api/services/all` - All services from all projects (JSON)
  - `GET /api/ws` - WebSocket for live updates
- **Fallback HTML**: Simple dashboard if React app not built yet
- **WebSocket Broadcasts**: Real-time service status updates

**Embedded Files:**
Located in `src/internal/dashboard/dist/`
- Currently: Placeholder `index.html`
- Future: Built React app (HTML, CSS, JS, assets)

**Server Lifecycle:**
```go
server := dashboard.GetServer(projectDir)
url, err := server.Start()  // Returns http://localhost:3100 (or next available)
// ... server runs ...
server.Stop()  // Graceful shutdown
```

### 4. Integration with Orchestrator
Updated `src/internal/service/orchestrator.go` to register services.

**Lifecycle:**
1. **Starting**: Register with status "starting", health "unknown"
2. **Process Started**: Update with PID
3. **Health Check**: Update status to "ready", health to "healthy"
4. **Error**: Update status to "error", health to "unhealthy"
5. **Stopping**: Update status to "stopping"
6. **Stopped**: Unregister from registry

**Registry Updates:**
- `OrchestrateServices()` - Registers each service before starting
- `PerformHealthCheck()` - Updates status to "ready" / "healthy" on success
- `StopAllServices()` - Unregisters services after stopping

### 5. Enhanced Run Command
Updated `src/cmd/app/commands/run.go` to start dashboard.

**New Output:**
```
╔══════════════════════════════════════╗
║  Starting 2 service(s)...            ║
╚══════════════════════════════════════╝

[15:04:05] [web] Starting nextjs service on port 3000
[15:04:09] [web] ✓ Service ready at http://localhost:3000
[15:04:09] [api] Starting fastapi service on port 8000
[15:04:13] [api] ✓ Service ready at http://localhost:8000

╔══════════════════════════════════════╗
║  All services ready!                 ║
╚══════════════════════════════════════╝

Service URLs:
  web: http://localhost:3000
  api: http://localhost:8000

📊 Dashboard: http://localhost:3100
```

**Dashboard Features:**
- Automatically starts when services run
- Shows all services in current project
- Updates in real-time via WebSocket
- Stops gracefully on Ctrl+C

## Architecture

### Registry Persistence
```
~/.azd/
└── app/
    └── services.json  # Global service registry
```

**JSON Format:**
```json
{
  "/home/user/my-app:web": {
    "name": "web",
    "projectDir": "/home/user/my-app",
    "pid": 12345,
    "port": 3000,
    "url": "http://localhost:3000",
    "language": "js",
    "framework": "nextjs",
    "status": "ready",
    "health": "healthy",
    "startTime": "2025-11-01T15:04:05Z",
    "lastChecked": "2025-11-01T15:04:09Z"
  }
}
```

### Dashboard Architecture
```
Dashboard Server (port 3100-3200)
├── Static Files (embedded)
│   └── dist/index.html (React app)
├── REST API
│   ├── GET /api/services - Current project
│   └── GET /api/services/all - All projects
└── WebSocket
    └── /api/ws - Live updates
```

### Data Flow
```
Service Orchestration
  ↓ Register
Service Registry (in-memory + disk)
  ↓ Query
Dashboard Server
  ↓ HTTP/WebSocket
Web Browser / CLI
```

## Usage Examples

### Start Services with Dashboard
```bash
cd my-app
azd app run

# Output includes:
# - Service startup logs
# - Service URLs
# - Dashboard URL
```

### Check Running Services
```bash
# Current project
azd app urls

# All projects
azd app urls --all

# Specific project
azd app urls --project ~/other-app
```

### Access Dashboard
```bash
# Open in browser (URL printed by `azd app run`)
http://localhost:3100

# Or query API directly
curl http://localhost:3100/api/services
curl http://localhost:3100/api/services/all
```

### Automation Scripts
```bash
# Get all service URLs for automation
azd app urls --all --format json  # Future enhancement

# Example script to check if services are ready
#!/bin/bash
SERVICES=$(curl -s http://localhost:3100/api/services)
echo "$SERVICES" | jq -r '.[] | select(.status=="ready") | .url'
```

## File Structure
```
src/
├── cmd/app/commands/
│   └── urls.go              # New: urls command
├── internal/
│   ├── registry/
│   │   └── registry.go      # New: service registry
│   ├── dashboard/
│   │   ├── server.go        # New: HTTP server
│   │   └── dist/
│   │       └── index.html   # Placeholder (React app goes here)
│   └── service/
│       └── orchestrator.go  # Updated: registry integration
└── main.go                  # Updated: added urls command
```

## Dependencies Added
- `github.com/gorilla/websocket` - WebSocket support for dashboard

## Testing
```bash
# Build
go build ./...  # ✅ Success

# Test
go test ./...   # ✅ All tests pass
```

## Future Enhancements (React Dashboard)
The React dashboard project still needs to be created:

### Dashboard Directory Structure (To Be Created)
```
dashboard/
├── package.json           # Vite + React + Tailwind 4
├── vite.config.ts         # Build to src/internal/dashboard/dist
├── tailwind.config.js     # Tailwind 4 config
├── src/
│   ├── App.tsx            # Main app component
│   ├── components/
│   │   ├── ServiceCard.tsx      # Service status card
│   │   ├── ServiceList.tsx      # List of services
│   │   ├── ProjectFilter.tsx    # Filter by project
│   │   └── ui/                  # shadcn/ui components
│   │       ├── card.tsx
│   │       ├── badge.tsx
│   │       └── ...
│   ├── hooks/
│   │   ├── useServices.ts       # Fetch & WebSocket hook
│   │   └── useWebSocket.ts      # WebSocket connection
│   └── lib/
│       └── api.ts               # API client
└── index.html
```

### Dashboard Features (To Be Implemented)
- **Real-time Updates**: WebSocket connection to `/api/ws`
- **Service Cards**: Show name, status, health, port, URL
- **Color-Coded Status**: Green (ready), yellow (starting), red (error)
- **Project Filtering**: Toggle between current project / all projects
- **Service Actions**: View logs, restart service (future)
- **Health Indicators**: Visual health check status
- **Responsive Design**: Mobile-friendly layout

### Build Integration
```json
// package.json script
"build": "vite build --outDir ../src/internal/dashboard/dist"
```

After building, the React app will be embedded in the Go binary via `embed.FS`.

## Summary

✅ **Completed:**
1. Service registry with persistent storage
2. `azd app urls` command for listing services
3. Embedded dashboard server with REST API & WebSocket
4. Integration with service orchestration
5. Dashboard URL printed after `azd app run`
6. Cross-project service tracking

🚧 **Remaining:**
1. Build React dashboard with Vite + Tailwind 4 + shadcn/ui
2. Implement service cards and real-time updates
3. Add project filtering UI
4. Build and embed React app in binary

The backend infrastructure is complete and ready for the React frontend!
