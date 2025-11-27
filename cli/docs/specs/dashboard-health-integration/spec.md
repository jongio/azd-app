# Dashboard Health Monitoring Integration

## Overview

Integrate the comprehensive health monitoring capabilities from `azd app health` command into the dashboard for real-time health visibility.

## Problem Statement

Currently, the dashboard receives service status updates via WebSocket but lacks:
1. Detailed health check information (response time, endpoint, check type)
2. Continuous health monitoring with configurable intervals
3. Health history and trends
4. Proactive health checks (independent of service state changes)

The `azd app health` command has robust health monitoring but is CLI-only.

## Goals

1. Expose health monitoring data to the dashboard via a new streaming API
2. Enhance dashboard UI to display detailed health information
3. Enable continuous health monitoring in the dashboard
4. Maintain backward compatibility with existing WebSocket architecture

## Proposed Solution

### API Changes

#### New Endpoint: `/api/health/stream` (Server-Sent Events)

Provides real-time health updates using SSE (simpler than WebSocket for one-way streaming).

```
GET /api/health/stream?interval=5s&service=api,web
Accept: text/event-stream

Response:
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: {"type":"health","timestamp":"2024-11-27T10:30:00Z","services":[...]}

event: health-change
data: {"service":"api","oldStatus":"healthy","newStatus":"unhealthy","reason":"connection timeout"}

event: heartbeat
data: {"timestamp":"2024-11-27T10:30:10Z"}
```

**Query Parameters:**
- `interval` (optional): Health check interval (default: 5s, min: 1s, max: 60s)
- `service` (optional): Comma-separated service filter

**Event Types:**
- `health`: Full health report for all services
- `health-change`: Status change notification for a single service
- `heartbeat`: Keep-alive signal (every 30s)

#### New Endpoint: `/api/health` (REST)

One-shot health check for all services.

```
GET /api/health?service=api

Response:
{
  "timestamp": "2024-11-27T10:30:00Z",
  "services": [
    {
      "serviceName": "api",
      "status": "healthy",
      "checkType": "http",
      "endpoint": "http://localhost:8080/health",
      "responseTime": 45,
      "statusCode": 200,
      "uptime": 3600,
      "details": {
        "database": "healthy",
        "cache": "healthy"
      }
    }
  ],
  "summary": {
    "total": 3,
    "healthy": 2,
    "degraded": 1,
    "unhealthy": 0,
    "overall": "degraded"
  }
}
```

### Type Enhancements

#### Extended LocalServiceInfo

```typescript
export interface LocalServiceInfo {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
  url?: string
  port?: number
  pid?: number
  startTime?: string
  lastChecked?: string
  // New health details
  healthDetails?: HealthDetails
}

export interface HealthDetails {
  checkType: 'http' | 'port' | 'process'
  endpoint?: string           // For HTTP checks
  responseTime?: number       // Milliseconds
  statusCode?: number         // HTTP status code
  uptime?: number            // Seconds since start
  lastError?: string         // Most recent error
  consecutiveFailures?: number
  details?: Record<string, unknown>  // From health endpoint response
}
```

#### Health Streaming Event Types

```typescript
export interface HealthEvent {
  type: 'health' | 'health-change' | 'heartbeat'
  timestamp: string
}

export interface HealthReportEvent extends HealthEvent {
  type: 'health'
  services: HealthCheckResult[]
  summary: HealthSummary
}

export interface HealthChangeEvent extends HealthEvent {
  type: 'health-change'
  service: string
  oldStatus: string
  newStatus: string
  reason?: string
}

export interface HeartbeatEvent extends HealthEvent {
  type: 'heartbeat'
}

export interface HealthCheckResult {
  serviceName: string
  status: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
  checkType: 'http' | 'port' | 'process'
  endpoint?: string
  responseTime: number
  statusCode?: number
  error?: string
  uptime?: number
  details?: Record<string, unknown>
}

export interface HealthSummary {
  total: number
  healthy: number
  degraded: number
  unhealthy: number
  unknown: number
  overall: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
}
```

### Dashboard Changes

#### New Hook: useHealthStream

```typescript
export function useHealthStream(options: {
  enabled?: boolean
  interval?: number  // seconds
  services?: string[]
}) {
  // Returns:
  // - healthReport: HealthReportEvent | null
  // - changes: HealthChangeEvent[]
  // - connected: boolean
  // - error: string | null
}
```

#### UI Enhancements

1. **ServiceCard**: Show health details (response time, check type, uptime)
2. **ServiceStatusCard**: Display health summary with drill-down
3. **New HealthPanel component**: Dedicated health monitoring view
4. **Health trend indicators**: Show if health is improving/declining

### Backend Changes

#### New handlers in server.go

```go
// Add to setupRoutes()
s.mux.HandleFunc("/api/health", s.handleHealthCheck)
s.mux.HandleFunc("/api/health/stream", s.handleHealthStream)

// handleHealthCheck performs a one-shot health check
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {}

// handleHealthStream provides SSE health updates
func (s *Server) handleHealthStream(w http.ResponseWriter, r *http.Request) {}
```

#### Integration with healthcheck package

Reuse existing `healthcheck.HealthMonitor` for actual health checks.

## Implementation Phases

### Phase 1: Backend API (MVP)
- Add `/api/health` REST endpoint
- Add `/api/health/stream` SSE endpoint
- Integrate with existing healthcheck.HealthMonitor

### Phase 2: Dashboard Integration
- Add useHealthStream hook
- Enhance ServiceCard with health details
- Update types

### Phase 3: Health UI Enhancements
- Add HealthPanel component
- Add health trend visualization
- Add health history

## Non-Goals (Out of Scope)

- Alerting/notifications for health changes (separate feature)
- Health metrics export to external systems
- Custom health check configurations via UI
- Historical health data persistence

## Alternatives Considered

1. **Extend existing WebSocket**: Would complicate the existing simple update model
2. **Polling REST endpoint**: Higher latency, more network overhead
3. **WebSocket for health**: More complex than needed for one-way data flow

SSE chosen because:
- Native browser support with auto-reconnection
- Simpler server implementation than WebSocket
- Perfect for one-way streaming
- Works through proxies/firewalls

## Dependencies

- Existing `healthcheck` package
- SSE library (or implement directly with `http.Flusher`)

## Success Criteria

1. Dashboard shows real-time health with <2s latency
2. Health details visible in service cards
3. Health changes are instantly reflected
4. No regression in existing WebSocket functionality
5. Performance: <5% CPU overhead from health monitoring
