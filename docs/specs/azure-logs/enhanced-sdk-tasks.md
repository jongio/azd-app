# Enhanced Azure SDK Integration - Tasks

<!-- DEFERRED: Moved to future enhancement, keeping azlogs branch focused on logs only -->

## Overview

**STATUS**: DEFERRED - This enhancement is planned for a future branch after the Azure logs feature is complete and merged.

Implementation tasks for enhanced Azure SDK integration, adding Metrics, Application Insights, and Management SDK capabilities to the existing log analytics implementation.

## Phase 1: Metrics Integration (P0 - High Priority)

### 1. Add Azure Monitor Metrics SDK {#add-metrics-sdk}
**Status**: TODO
**Assigned**: Developer

Add metrics SDK dependency to enable real-time performance monitoring.

**Actions**:
1. Add dependency: `go get github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics`
2. Update `go.mod` and `go.sum`
3. Verify clean build

**Files**:
- `cli/go.mod`
- `cli/go.sum`

**Acceptance Criteria**:
- Dependency added successfully
- Clean `go build`
- All existing tests pass

---

### 2. Implement Metrics Client {#implement-metrics-client}
**Status**: TODO
**Assigned**: Developer

Create metrics query client with support for standard Azure resource metrics.

**Actions**:
1. Create `cli/src/internal/azure/metrics.go`
2. Implement `MetricsClient` struct
3. Implement `GetResourceMetrics()` method
4. Add `GetDefaultMetrics()` helper for resource types
5. Add unit tests with mocked SDK

**Interface**:
```go
type MetricsClient struct {
    client *azmetrics.Client
}

func NewMetricsClient(cred azcore.TokenCredential) (*MetricsClient, error)

func (c *MetricsClient) GetResourceMetrics(
    ctx context.Context,
    resourceID string,
    metricNames []string,
    startTime, endTime time.Time,
) (*MetricsResult, error)

func GetDefaultMetrics(resourceType ResourceType) []string
```

**Files**:
- `cli/src/internal/azure/metrics.go` (new)
- `cli/src/internal/azure/metrics_test.go` (new)

**Acceptance Criteria**:
- Client authenticates with Azure
- Successfully queries resource metrics
- Supports Container Apps, App Service, Functions
- Unit tests with 80% coverage

---

### 3. Add Metrics API Endpoint {#add-metrics-api}
**Status**: TODO
**Assigned**: Developer

Add dashboard API endpoint for querying Azure resource metrics.

**Actions**:
1. Create `cli/src/internal/dashboard/azure_metrics.go`
2. Implement `handleAzureMetrics` handler
3. Add route: `GET /api/azure/metrics`
4. Support query params: `service`, `metric`, `since`, `interval`
5. Return JSON time-series data

**Endpoint**:
```
GET /api/azure/metrics?service=api&metric=CpuPercentage&since=1h
```

**Response**:
```json
{
  "metric": "CpuPercentage",
  "service": "api",
  "resourceId": "/subscriptions/.../",
  "timeRange": {
    "start": "2024-01-15T10:00:00Z",
    "end": "2024-01-15T11:00:00Z"
  },
  "dataPoints": [
    {"timestamp": "2024-01-15T10:00:00Z", "value": 23.5},
    {"timestamp": "2024-01-15T10:01:00Z", "value": 25.2}
  ],
  "aggregation": "Average",
  "unit": "Percent"
}
```

**Files**:
- `cli/src/internal/dashboard/azure_metrics.go` (new)
- `cli/src/internal/dashboard/server.go` (modify)
- `cli/src/internal/dashboard/azure_metrics_test.go` (new)

**Acceptance Criteria**:
- Endpoint returns valid metric data
- Supports multiple metric types
- Error handling for missing/invalid resources
- Integration tests with mocked Azure

---

### 4. Dashboard Metrics Panel Component {#dashboard-metrics-panel}
**Status**: TODO
**Assigned**: Developer

Create React component to display metrics charts in dashboard.

**Actions**:
1. Create `cli/dashboard/src/components/MetricsPanel.tsx`
2. Create `cli/dashboard/src/hooks/useMetrics.ts`
3. Add time-series chart with recharts library
4. Support multiple metrics per service
5. Add metric selector dropdown

**Features**:
- Real-time metric charts (line graphs)
- Time range selector (15m, 1h, 6h, 24h)
- Multi-metric view (CPU, Memory, Requests)
- Auto-refresh every 30s
- Loading and error states

**Files**:
- `cli/dashboard/src/components/MetricsPanel.tsx` (new)
- `cli/dashboard/src/hooks/useMetrics.ts` (new)
- `cli/dashboard/src/types.ts` (modify)
- `cli/dashboard/src/components/MetricsPanel.test.tsx` (new)

**Acceptance Criteria**:
- Charts display metric data correctly
- Time range selection works
- Auto-refresh functional
- Accessible (WCAG AA)
- Component tests pass

---

### 5. Extend azure.yaml Schema for Metrics {#extend-schema-metrics}
**Status**: TODO
**Assigned**: Developer

Add metrics configuration to azure.yaml schema.

**Schema Addition**:
```yaml
logs:
  analytics:
    workspace: ""
  
  metrics:  # NEW
    enabled: true
    defaultMetrics:
      - CpuPercentage
      - MemoryPercentage
      - Requests
    interval: PT1M  # ISO 8601 duration
    refreshInterval: 30s
```

**Actions**:
1. Update `schemas/v1.1/azure.yaml.json`
2. Add `metricsConfig` schema definition
3. Add validation rules
4. Update documentation

**Files**:
- `schemas/v1.1/azure.yaml.json`
- `cli/docs/features/azure-logs.md` (update)

**Acceptance Criteria**:
- Schema validates metrics config
- Backward compatible (metrics optional)
- Documentation complete

---

### 6. CLI Metrics Command {#cli-metrics-command}
**Status**: TODO
**Assigned**: Developer

Add `azd app metrics` command for CLI metric queries.

**Command**:
```bash
azd app metrics --service api --metric CpuPercentage --since 1h
azd app metrics list --service api
```

**Actions**:
1. Create `cli/src/cmd/app/commands/metrics.go`
2. Implement `metricsCommand` with flags
3. Add `metricsExecutor` for query logic
4. Support output formats (table, json)
5. Add command tests

**Files**:
- `cli/src/cmd/app/commands/metrics.go` (new)
- `cli/src/cmd/app/commands/metrics_test.go` (new)
- `cli/docs/commands/metrics.md` (new)

**Acceptance Criteria**:
- Command queries metrics successfully
- List command shows available metrics
- Output formats work correctly
- Help text clear and complete

---

### 7. Integrate Metrics into Service Details {#integrate-metrics-ui}
**Status**: TODO
**Assigned**: Developer

Add metrics section to service detail view in dashboard.

**Actions**:
1. Update `cli/dashboard/src/components/ServiceDetails.tsx`
2. Add "Metrics" tab alongside "Logs" tab
3. Embed `MetricsPanel` component
4. Show health indicators based on metrics
5. Add performance trend sparklines

**Features**:
- Tab navigation: Logs | Metrics | Info
- Real-time health badges (CPU high, memory OK, etc.)
- Quick metric overview (current values)
- Link to full metrics view

**Files**:
- `cli/dashboard/src/components/ServiceDetails.tsx` (modify)
- `cli/dashboard/src/components/ServiceHealthBadge.tsx` (new)

**Acceptance Criteria**:
- Metrics tab displays correctly
- Health indicators accurate
- Tab switching smooth
- Mobile responsive

---

## Phase 2: Application Insights Integration (P1 - Medium Priority)

### 8. Add Application Insights SDK {#add-appinsights-sdk}
**Status**: TODO
**Assigned**: Developer

Add Application Insights SDK for telemetry access.

**Actions**:
1. Add dependency: `go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/applicationinsights/armapplicationinsights/v2`
2. Update dependencies
3. Verify build

**Files**:
- `cli/go.mod`
- `cli/go.sum`

**Acceptance Criteria**:
- Dependency added
- Clean build
- Tests pass

---

### 9. Implement App Insights Discovery {#implement-appinsights-discovery}
**Status**: TODO
**Assigned**: Developer

Auto-discover Application Insights components linked to resources.

**Actions**:
1. Extend `cli/src/internal/azure/discovery.go`
2. Add `DiscoverAppInsights()` method
3. Link App Insights to services
4. Cache component details

**Files**:
- `cli/src/internal/azure/discovery.go` (modify)
- `cli/src/internal/azure/appinsights.go` (new)
- `cli/src/internal/azure/appinsights_test.go` (new)

**Acceptance Criteria**:
- Discovers linked App Insights components
- Maps components to services
- Handles missing components gracefully

---

### 10. App Insights Telemetry Query {#appinsights-telemetry-query}
**Status**: TODO
**Assigned**: Developer

Query Application Insights tables for structured telemetry.

**Tables**:
- `requests` - HTTP request telemetry
- `exceptions` - Exception logs
- `dependencies` - Downstream call telemetry
- `traces` - Custom trace logs

**Actions**:
1. Create `cli/src/internal/azure/appinsights_query.go`
2. Implement query methods per table type
3. Add API endpoints for telemetry data
4. Transform to common format

**Files**:
- `cli/src/internal/azure/appinsights_query.go` (new)
- `cli/src/internal/dashboard/azure_telemetry.go` (new)

**Acceptance Criteria**:
- Query App Insights tables successfully
- Request telemetry includes duration, status
- Exception grouping works
- Dependency data shows call chains

---

## Phase 3: Management SDK Integration (P2 - Medium Priority)

### 11. Add Azure Monitor Management SDK {#add-management-sdk}
**Status**: TODO
**Assigned**: Developer

Add management SDK for diagnostic settings validation.

**Actions**:
1. Add dependency: `go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor`
2. Update dependencies

**Files**:
- `cli/go.mod`
- `cli/go.sum`

**Acceptance Criteria**:
- Dependency added
- Clean build

---

### 12. Diagnostic Settings Validation {#diagnostic-settings-validation}
**Status**: TODO
**Assigned**: Developer

Validate that resources have diagnostic settings configured correctly.

**Actions**:
1. Create `cli/src/internal/azure/diagnostics.go`
2. Implement `ValidateDiagnosticSettings()` function
3. Check for workspace linkage
4. Verify enabled log categories
5. Add validation to status endpoint

**Files**:
- `cli/src/internal/azure/diagnostics.go` (new)
- `cli/src/internal/azure/diagnostics_test.go` (new)

**Acceptance Criteria**:
- Detects missing diagnostic settings
- Validates workspace configuration
- Returns actionable error messages
- Unit tests with mocked ARM API

---

### 13. Logs Init Command {#logs-init-command}
**Status**: TODO
**Assigned**: Developer

Add command to initialize log collection for resources.

**Command**:
```bash
azd app logs init --service api
azd app logs validate
```

**Actions**:
1. Create `cli/src/cmd/app/commands/logs_init.go`
2. Check diagnostic settings
3. Create settings if missing
4. Link to workspace
5. Verify logs flowing

**Files**:
- `cli/src/cmd/app/commands/logs_init.go` (new)
- `cli/docs/commands/logs.md` (update)

**Acceptance Criteria**:
- Creates diagnostic settings automatically
- Links to detected workspace
- Verifies configuration works
- Clear output messages

---

## Phase 4: Advanced Features (P3 - Low Priority)

### 14. Metric Alerts Integration {#metric-alerts}
**Status**: TODO
**Assigned**: Developer

Display active metric alerts in dashboard.

**Actions**:
1. Query alert rules API
2. Show active alerts in header
3. Alert details modal
4. Link to Azure Portal for management

**Acceptance Criteria**:
- Alerts displayed correctly
- Severity colors clear
- Links to Portal work

---

### 15. Custom Metrics Ingestion {#custom-metrics-ingestion}
**Status**: TODO
**Assigned**: Developer

Forward local development metrics to Azure for unified view.

**Actions**:
1. Add ingestion SDK
2. Configure Data Collection Endpoint
3. Forward local metrics during development
4. Toggle in config

**Acceptance Criteria**:
- Local metrics appear in Azure
- No performance impact
- Opt-in feature

---

## Task Dependencies

```
Phase 1 (Metrics):
1 → 2 → 3 → 4
         ↓
    5 → 6 → 7

Phase 2 (App Insights):
8 → 9 → 10

Phase 3 (Management):
11 → 12 → 13

Phase 4 (Advanced):
14 (parallel)
15 (parallel)
```

## Success Metrics

### Phase 1 Complete
- ✅ Metrics API functional
- ✅ Dashboard displays CPU/memory charts
- ✅ CLI metrics command works
- ✅ 80% test coverage
- ✅ Documentation complete

### Phase 2 Complete
- ✅ App Insights auto-discovered
- ✅ Request telemetry visible
- ✅ Exception grouping works

### Phase 3 Complete
- ✅ Diagnostic settings validated
- ✅ Auto-init functional
- ✅ Clear setup guidance

## Estimated Timeline

| Phase | Tasks | Effort | Duration |
|-------|-------|--------|----------|
| Phase 1 | 1-7 | 3 weeks | Sprint 1-2 |
| Phase 2 | 8-10 | 2 weeks | Sprint 3 |
| Phase 3 | 11-13 | 1 week | Sprint 4 |
| Phase 4 | 14-15 | 1 week | Sprint 5 |

**Total**: ~7 weeks for full implementation

## Priority Justification

**Phase 1 (High Priority)**:
- High user value (real-time metrics)
- Low implementation risk
- Complements existing logs
- Quick wins for observability

**Phase 2 (Medium Priority)**:
- Valuable for App Insights users
- Requires App Insights setup
- More complex integration

**Phase 3 (Medium Priority)**:
- Improves onboarding
- Reduces support burden
- Lower user-facing value

**Phase 4 (Low Priority)**:
- Nice-to-have features
- Limited user demand
- Complex implementation
