# Dashboard Health Monitoring Integration - Tasks

## Progress: 0/12 TODO, 0 IN PROGRESS, 0 DONE

## Task List

### Phase 1: Backend API

#### Task 1.1: Add /api/health REST endpoint
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Add REST endpoint for one-shot health checks
- **Acceptance Criteria**:
  - GET `/api/health` returns health report for all services
  - GET `/api/health?service=api` filters to specific service
  - Response matches `HealthReport` JSON schema
  - Uses existing `healthcheck.HealthMonitor`
  - Timeout defaults to 5s
- **Files**: `cli/src/internal/dashboard/server.go`

#### Task 1.2: Add /api/health/stream SSE endpoint
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Add Server-Sent Events endpoint for health streaming
- **Acceptance Criteria**:
  - GET `/api/health/stream` opens SSE connection
  - Sends `health` event at configurable interval (default 5s)
  - Sends `health-change` event when status changes
  - Sends `heartbeat` every 30s
  - Supports `interval` and `service` query params
  - Graceful cleanup on client disconnect
- **Files**: `cli/src/internal/dashboard/server.go`

#### Task 1.3: Create health streaming integration
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Integrate healthcheck.HealthMonitor with dashboard streaming
- **Acceptance Criteria**:
  - Reuse existing `healthcheck.HealthMonitor` for checks
  - Track previous health states for change detection
  - Rate limit health checks per service
  - Handle concurrent SSE connections
- **Files**: `cli/src/internal/dashboard/health_stream.go` (new)

### Phase 2: Dashboard Frontend

#### Task 2.1: Create useHealthStream hook
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: React hook for consuming health SSE stream
- **Acceptance Criteria**:
  - Connects to `/api/health/stream` SSE endpoint
  - Parses `health`, `health-change`, and `heartbeat` events
  - Auto-reconnects on disconnect (with backoff)
  - Exposes `healthReport`, `changes`, `connected`, `error`
  - Configurable interval and service filter
  - Cleanup on unmount
- **Files**: `cli/dashboard/src/hooks/useHealthStream.ts` (new)

#### Task 2.2: Add HealthDetails types
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Add TypeScript types for health monitoring
- **Acceptance Criteria**:
  - Add `HealthDetails` interface
  - Add `HealthEvent`, `HealthReportEvent`, `HealthChangeEvent` types
  - Add `HealthCheckResult`, `HealthSummary` types
  - Update `LocalServiceInfo` with optional `healthDetails`
- **Files**: `cli/dashboard/src/types.ts`

#### Task 2.3: Update ServiceCard with health details
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Show detailed health information in service cards
- **Acceptance Criteria**:
  - Display response time when available
  - Show check type (HTTP/port/process)
  - Show health endpoint for HTTP checks
  - Display uptime
  - Show last error if unhealthy
- **Files**: `cli/dashboard/src/components/ServiceCard.tsx`

#### Task 2.4: Update ServiceStatusCard with health summary
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Enhance status card with health summary data
- **Acceptance Criteria**:
  - Show degraded count separate from errors
  - Tooltip with health summary details
  - Indicator for active health monitoring
- **Files**: `cli/dashboard/src/components/ServiceStatusCard.tsx`

### Phase 3: Integration & Testing

#### Task 3.1: Integrate health stream in App
- **Agent**: Developer
- **Status**: 🔲 TODO
- **Description**: Wire up health streaming in main App component
- **Acceptance Criteria**:
  - useHealthStream called at App level
  - Health data passed to ServiceCard components
  - Health changes trigger notifications
  - Graceful fallback if health stream fails
- **Files**: `cli/dashboard/src/App.tsx`

#### Task 3.2: Write unit tests for useHealthStream
- **Agent**: Tester
- **Status**: 🔲 TODO
- **Description**: Unit tests for health stream hook
- **Acceptance Criteria**:
  - Test connection establishment
  - Test event parsing for all types
  - Test reconnection logic
  - Test cleanup on unmount
  - Coverage ≥80%
- **Files**: `cli/dashboard/src/hooks/useHealthStream.test.ts` (new)

#### Task 3.3: Write backend tests for health endpoints
- **Agent**: Tester
- **Status**: 🔲 TODO
- **Description**: Tests for REST and SSE health endpoints
- **Acceptance Criteria**:
  - Test `/api/health` REST endpoint
  - Test `/api/health/stream` SSE endpoint
  - Test service filtering
  - Test interval configuration
  - Test error handling
- **Files**: `cli/src/internal/dashboard/health_stream_test.go` (new)

#### Task 3.4: E2E test health monitoring flow
- **Agent**: Tester
- **Status**: 🔲 TODO
- **Description**: End-to-end test of health monitoring in dashboard
- **Acceptance Criteria**:
  - Dashboard shows health details
  - Health updates in real-time
  - Health changes reflected correctly
  - Degraded state displays properly
- **Files**: `cli/dashboard/e2e/health-monitoring.spec.ts` (new)
