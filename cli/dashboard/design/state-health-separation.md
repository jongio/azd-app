# Service State and Health Separation

## Problem

Currently we conflate two distinct concepts:
1. **Lifecycle State** - Is the process running?
2. **Health Status** - Is the service working correctly?

This causes UI inconsistencies where services show "Starting" when they're actually running but health checks haven't passed yet.

## Design

### Lifecycle State (ProcessState)

Describes the process lifecycle - managed by the service orchestrator.

| State | Description | Applicable To |
|-------|-------------|---------------|
| `not-started` | Never been started | All |
| `starting` | Process is being launched | All |
| `running` | Process is actively running | http, tcp, daemon, watch |
| `stopping` | Process is being terminated | All |
| `stopped` | Process has been intentionally stopped | All |
| `restarting` | Process is being restarted | All |
| `completed` | Process finished successfully (exit 0) | build, task |
| `failed` | Process exited with error (exit != 0) | build, task |

### Health Status (HealthStatus)

Describes the service's ability to handle requests - determined by health checks.

| Status | Description | When |
|--------|-------------|------|
| `healthy` | Service is responding correctly | Health checks pass |
| `degraded` | Service responding but with issues | Partial failures, slow responses |
| `unhealthy` | Service not responding correctly | Health checks fail |
| `unknown` | Health cannot be determined | No health check configured, or not yet checked |
| `n/a` | Health check not applicable | Process is stopped, or build/task mode |

### Display Logic

The UI should show BOTH state and health when relevant:

| Process State | Health Status | Display |
|---------------|---------------|---------|
| running | healthy | ✓ Running (green) |
| running | degraded | ⚠ Running - Degraded (amber) |
| running | unhealthy | ✗ Running - Unhealthy (red) |
| running | unknown | ○ Running (gray indicator) |
| starting | * | ◐ Starting (yellow, animated) |
| stopping | * | ◐ Stopping (gray, animated) |
| stopped | n/a | ○ Stopped (gray) |
| completed | n/a | ✓ Completed (green) |
| failed | n/a | ✗ Failed (red) |
| not-started | n/a | ○ Not Started (gray) |

### Process Type Considerations

**HTTP/TCP Services (type: http, tcp)**
- Always have health checks (HTTP endpoint or port check)
- Health status is meaningful when running

**Process Services (type: process)**
- May or may not have health checks
- Health based on: process running (daemon/watch) or exit code (build/task)

**Mode-Specific Behavior**
- `watch`: Running continuously, health = process alive
- `build`: Completes, health = exit code 0
- `daemon`: Running continuously, health = process alive  
- `task`: Completes, health = exit code 0

### Component Responsibilities

1. **Services API** (`/api/services`): Provides lifecycle state
2. **Health Stream** (`/api/health/stream`): Provides health status
3. **UI Components**: Display both independently

### Migration Path

1. Keep `HealthStatus` type but remove `'starting'` - that's a lifecycle state
2. Add `ProcessState` type for lifecycle
3. Update `getStatusDisplay()` to take both state and health
4. Update all components to pass both values
5. Backend: Ensure health stream returns pure health (not lifecycle)

## Files to Update

### Types
- `types.ts`: Add ProcessState, update HealthStatus

### Utils  
- `service-utils.ts`: Update getStatusDisplay, getEffectiveStatus

### Components
- `ServiceTableRow.tsx`
- `ServiceCard.tsx`
- `StatusCell.tsx`
- `LogsPane.tsx`
- `LogsMultiPaneView.tsx`
- `ModernMetricsView.tsx`
- `PerformanceMetrics.tsx`

### Backend (if needed)
- `healthcheck/monitor.go`: Ensure clean separation
