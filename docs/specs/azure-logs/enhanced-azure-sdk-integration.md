# Enhanced Azure SDK Integration for Logs Implementation

## Executive Summary

Analysis of Azure SDK for Go to identify additional monitoring capabilities beyond the currently implemented Log Analytics queries. This spec proposes enhancements using Azure Monitor Metrics, Application Insights, and ingestion SDKs to provide richer observability.

## Current Implementation Status

### ✅ Currently Implemented (Production)
- **Log Analytics Query SDK** (`sdk/monitor/query/azlogs`)
  - KQL query execution
  - Workspace queries
  - Custom time ranges
  - Multiple workspace queries
  
### 📦 Available Packages in go.mod
```go
github.com/Azure/azure-sdk-for-go/sdk/azcore v1.20.0
github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs v1.2.0
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0
```

## Proposed Additional SDKs

### 1. Azure Monitor Metrics SDK (High Priority)

**Package**: `github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics`

**Use Cases**:
- Real-time performance metrics (CPU, memory, requests/sec)
- Resource-level metrics without Log Analytics dependency
- Lower latency than log queries (metrics are near real-time)
- Alerting and threshold monitoring

**Benefits**:
- Complements log data with numeric metrics
- Faster for time-series data
- Standard Azure resource metrics available without custom logging

**Implementation Example**:
```go
// Query CPU percentage for Container App
client, _ := azmetrics.NewClient(credential, nil)
res, err := client.QueryResource(ctx, resourceID,
    &azmetrics.QueryResourceOptions{
        Timespan:    to.Ptr(azmetrics.NewTimeInterval(startTime, endTime)),
        Interval:    to.Ptr("PT1M"), // 1 minute intervals
        MetricNames: []string{"CpuPercentage", "MemoryPercentage", "Requests"},
        Aggregation: []*azmetrics.AggregationType{
            to.Ptr(azmetrics.AggregationTypeAverage),
        },
    })
```

**Data Available**:
- Container Apps: CPU%, Memory%, Request count, Response time
- App Service: CPU%, Memory%, HTTP requests, Response time
- Functions: Execution count, Execution units, Errors

### 2. Application Insights SDK (Medium Priority)

**Package**: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/applicationinsights/armapplicationinsights`

**Use Cases**:
- Telemetry management
- Query Application Insights tables directly
- Access to distributed tracing data
- Dependency maps and metrics

**Benefits**:
- Richer telemetry than raw logs
- Pre-aggregated performance data
- Exception and dependency tracking
- Request correlation across services

**Implementation Example**:
```go
// Create App Insights client
factory, _ := armapplicationinsights.NewClientFactory(subscriptionID, cred, nil)
client := factory.NewComponentsClient()

// Get App Insights component for a service
component, _ := client.Get(ctx, resourceGroup, componentName, nil)

// Use Query API to access telemetry
// (requires separate Application Insights Query SDK)
```

### 3. Azure Monitor Management SDK (Medium Priority)

**Package**: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor`

**Use Cases**:
- Diagnostic settings management
- Auto-configure log collection
- Alert rule management
- Metric namespace discovery

**Benefits**:
- Programmatic setup of log collection
- Validate diagnostic settings during `azd provision`
- Auto-detect available metrics per resource type

**Implementation Example**:
```go
// Check if diagnostic settings exist
factory, _ := armmonitor.NewClientFactory(subscriptionID, cred, nil)
diagClient := factory.NewDiagnosticSettingsClient()

settings, err := diagClient.List(ctx, resourceID, nil)
if err != nil || len(settings.Value) == 0 {
    // Guide user to enable diagnostic settings
}

// List available metrics for a resource
metricsClient := factory.NewMetricDefinitionsClient()
defs, _ := metricsClient.List(ctx, resourceID, nil)
```

### 4. Azure Monitor Ingestion SDK (Low Priority - Future)

**Package**: `github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs`

**Use Cases**:
- Custom log ingestion
- Local development log forwarding to Azure
- Unified logging experience

**Benefits**:
- Forward local logs to Azure for unified view
- Custom telemetry from azd-app extension

**Note**: Not needed for current read-only implementation

## Proposed Architecture Enhancements

### Current Architecture
```
azd-app CLI/Dashboard
    ↓
azlogs.Client (Log Analytics)
    ↓
KQL Queries → Logs
```

### Enhanced Architecture
```
azd-app CLI/Dashboard
    ↓
┌─────────────────┬──────────────────┬─────────────────┐
│  azlogs.Client  │ azmetrics.Client │ armmonitor      │
│  (Logs)         │  (Metrics)       │  (Management)   │
└─────────────────┴──────────────────┴─────────────────┘
    ↓                    ↓                    ↓
Log Analytics      Resource Metrics    Diagnostic Settings
```

## Implementation Plan

### Phase 1: Metrics Integration (High Priority)

**Goal**: Add real-time metrics alongside logs

#### Task 1.1: Add Metrics SDK Dependency
```bash
go get github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics
```

**Files**:
- `cli/go.mod` - Add dependency
- `cli/go.sum` - Update checksums

#### Task 1.2: Metrics Client Implementation
**File**: `cli/src/internal/azure/metrics.go` (new)

```go
package azure

import (
    "context"
    "time"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
    "github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
)

type MetricsClient struct {
    client *azmetrics.Client
}

func NewMetricsClient(cred azcore.TokenCredential) (*MetricsClient, error) {
    client, err := azmetrics.NewClient(cred, nil)
    if err != nil {
        return nil, err
    }
    return &MetricsClient{client: client}, nil
}

// GetResourceMetrics queries metrics for a specific resource
func (c *MetricsClient) GetResourceMetrics(
    ctx context.Context,
    resourceID string,
    metricNames []string,
    startTime, endTime time.Time,
) (*MetricsResult, error) {
    // Implementation
}

// GetDefaultMetrics returns standard metrics for resource type
func GetDefaultMetrics(resourceType ResourceType) []string {
    switch resourceType {
    case ResourceTypeContainerApp:
        return []string{"CpuPercentage", "MemoryPercentage", "Requests"}
    case ResourceTypeAppService:
        return []string{"CpuPercentage", "MemoryWorkingSet", "Http5xx"}
    case ResourceTypeFunction:
        return []string{"FunctionExecutionCount", "FunctionExecutionUnits"}
    default:
        return []string{}
    }
}
```

#### Task 1.3: Metrics API Endpoint
**File**: `cli/src/internal/dashboard/azure_metrics.go` (new)

```go
// GET /api/azure/metrics?service={name}&metric={name}&since={duration}
func (s *Server) handleAzureMetrics(w http.ResponseWriter, r *http.Request) {
    // Parse query params
    serviceName := r.URL.Query().Get("service")
    metricName := r.URL.Query().Get("metric")
    since := parseDuration(r.URL.Query().Get("since"), 1*time.Hour)
    
    // Get resource for service
    resource := s.azureDiscovery.GetResource(serviceName)
    
    // Query metrics
    metrics, err := s.metricsClient.GetResourceMetrics(
        r.Context(),
        resource.ResourceID,
        []string{metricName},
        time.Now().Add(-since),
        time.Now(),
    )
    
    // Return JSON
}
```

#### Task 1.4: Dashboard Metrics Panel
**File**: `cli/dashboard/src/components/MetricsPanel.tsx` (new)

```typescript
export function MetricsPanel({ service }: { service: string }) {
  const { data } = useMetrics(service, ['CpuPercentage', 'MemoryPercentage']);
  
  return (
    <div className="metrics-panel">
      <MetricChart metric="CpuPercentage" data={data?.cpu} />
      <MetricChart metric="MemoryPercentage" data={data?.memory} />
    </div>
  );
}
```

**Benefits**:
- Real-time performance visibility
- Faster than log queries
- Reduced Log Analytics costs
- Standard metrics without configuration

**Acceptance Criteria**:
- Metrics query working for Container Apps
- Dashboard displays CPU and memory charts
- Metrics available in CLI (`azd app metrics`)
- Error handling for unsupported resource types

---

### Phase 2: Application Insights Integration (Medium Priority)

**Goal**: Access Application Insights telemetry

#### Task 2.1: Add App Insights SDK
```bash
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/applicationinsights/armapplicationinsights/v2
```

#### Task 2.2: App Insights Discovery
**File**: `cli/src/internal/azure/appinsights.go` (new)

```go
// Discover Application Insights components linked to resources
func (d *ResourceDiscovery) DiscoverAppInsights(ctx context.Context) ([]AppInsightsComponent, error) {
    factory, err := armapplicationinsights.NewClientFactory(d.subscriptionID, d.credential, nil)
    client := factory.NewComponentsClient()
    
    pager := client.NewListByResourceGroupPager(d.resourceGroup, nil)
    // Iterate and return components
}
```

#### Task 2.3: Enhanced Telemetry Queries
- Query `requests` table for HTTP telemetry
- Query `exceptions` table for errors
- Query `dependencies` table for downstream calls
- Query `traces` table for custom logging

**Benefits**:
- Structured telemetry vs raw logs
- Pre-aggregated performance data
- Exception grouping and trends
- Request correlation across services

---

### Phase 3: Management SDK Integration (Medium Priority)

**Goal**: Validate and auto-configure logging

#### Task 3.1: Add Monitor Management SDK
```bash
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor
```

#### Task 3.2: Diagnostic Settings Validation
**File**: `cli/src/internal/azure/diagnostics.go` (new)

```go
func ValidateDiagnosticSettings(ctx context.Context, resourceID string) (*DiagnosticStatus, error) {
    factory, _ := armmonitor.NewClientFactory(subscriptionID, cred, nil)
    client := factory.NewDiagnosticSettingsClient()
    
    settings, err := client.List(ctx, resourceID, nil)
    
    return &DiagnosticStatus{
        Enabled:         len(settings.Value) > 0,
        WorkspaceLinked: hasWorkspace(settings),
        Categories:      getEnabledCategories(settings),
    }, nil
}
```

#### Task 3.3: `azd app logs init` Command
**New Command**: Initialize log collection for a resource

```bash
azd app logs init --service api
```

**Actions**:
1. Check if diagnostic settings exist
2. If not, create with recommended settings:
   - Send logs to Log Analytics workspace
   - Enable container/app logs category
   - Set retention policy
3. Verify workspace is accessible
4. Test query to confirm logs flowing

**Benefits**:
- Automated log setup
- Validation during `azd provision`
- Clear guidance when logging not configured

---

### Phase 4: Advanced Features (Low Priority)

#### Task 4.1: Metric Alerts API
- Query existing alert rules
- Display alerts in dashboard
- CLI command to list active alerts

#### Task 4.2: Custom Metrics Ingestion
- Forward local development metrics to Azure
- Unified dev/prod observability

#### Task 4.3: Operational Insights SDK
```bash
go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights/v2
```

**Use Cases**:
- Workspace management
- Saved queries
- Query packs

---

## SDK Comparison Matrix

| SDK | Current Use | Proposed Use | Priority | Latency | Cost Impact |
|-----|-------------|--------------|----------|---------|-------------|
| `azlogs` | ✅ Production | Current logs | High | 1-5min | Base |
| `azmetrics` | ❌ Not used | Real-time metrics | High | 1-30s | Low |
| `armapplicationinsights` | ❌ Not used | Telemetry query | Medium | 1-5min | Low |
| `armmonitor` | ❌ Not used | Diagnostic validation | Medium | Instant | None |
| `azlogs` (ingestion) | ❌ Not used | Custom logs | Low | 1-5min | Medium |
| `armoperationalinsights` | ❌ Not used | Workspace mgmt | Low | Instant | None |

## Configuration Schema Extensions

### Current Schema
```yaml
logs:
  analytics:
    workspace: ""
    pollingInterval: 30s
    defaultTimespan: 30m
```

### Proposed Extensions
```yaml
logs:
  analytics:
    workspace: ""
    pollingInterval: 30s
    defaultTimespan: 30m
    
  metrics:  # NEW
    enabled: true
    defaultMetrics:
      - CpuPercentage
      - MemoryPercentage
      - Requests
    interval: PT1M  # ISO 8601 duration
    
  applicationInsights:  # NEW
    enabled: auto  # auto, true, false
    componentName: ""  # optional, auto-detect if empty
```

## CLI Command Extensions

### New Commands

```bash
# Query metrics
azd app metrics --service api --metric CpuPercentage --since 1h

# List available metrics for a service
azd app metrics list --service api

# Initialize diagnostic settings
azd app logs init --service api

# Validate log configuration
azd app logs validate

# Show Application Insights telemetry
azd app insights --service api --type requests --since 1h
```

## Dashboard UI Enhancements

### Metrics Panel
- Time-series charts for CPU, Memory, Requests
- Toggle metrics on/off
- Overlay metrics on log timeline

### Service Health Indicators
- Real-time status badges (health based on metrics)
- Error rate indicators
- Performance trend sparklines

### Telemetry Views
- Request table (HTTP requests from App Insights)
- Exception grouping
- Dependency map

## Testing Strategy

### Unit Tests
- Mock Azure SDK clients
- Test metric parsing
- Test configuration validation

### Integration Tests
- Mock Azure Management API responses
- Test metric query flow
- Test diagnostic settings validation

### E2E Tests
- Test with real Azure resources (optional)
- Metrics display in dashboard
- CLI commands with mocked backends

## Migration Path

### Phase 1: Non-Breaking Addition
- Add metrics SDK alongside existing logs
- New optional features
- Existing logs functionality unchanged

### Phase 2: Configuration
- Extend schema with backward compatibility
- Default to logs-only mode
- Opt-in for metrics and App Insights

### Phase 3: Integration
- Unified observability view
- Metrics overlay on logs timeline
- Cross-correlation features

## Dependencies and Prerequisites

### Azure Resources Required
- Log Analytics Workspace (existing requirement)
- Application Insights component (optional)
- Diagnostic settings enabled on resources

### Azure Permissions Required
```
# Current
- Log Analytics Reader

# Additional for Metrics
- Monitoring Reader

# Additional for Management
- Reader (subscription/resource group level)

# Optional for App Insights
- Application Insights Component Contributor
```

## Success Metrics

### Phase 1 (Metrics)
- ✅ Metrics API returns data for Container Apps
- ✅ Dashboard displays real-time CPU/memory charts
- ✅ CLI `azd app metrics` command functional
- ✅ Latency <5s for metric queries

### Phase 2 (App Insights)
- ✅ Auto-detect App Insights components
- ✅ Request telemetry visible in dashboard
- ✅ Exception grouping working

### Phase 3 (Management)
- ✅ Diagnostic settings validation
- ✅ Auto-configuration during provision
- ✅ Clear setup guidance

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Additional dependencies | Medium | Keep as optional features |
| Increased complexity | Medium | Clear separation of concerns |
| Cost increase | Low | Metrics free for Azure resources |
| Breaking changes | Low | Backward compatible schema |

## Recommendations

### Immediate (Next Sprint)
1. **Add Metrics SDK** - High value, low risk
2. **Implement basic metrics API** - CPU, Memory, Requests
3. **Dashboard metrics panel** - Simple time-series charts

### Short-term (1-2 months)
4. **Application Insights integration** - If services use App Insights
5. **Diagnostic settings validation** - Improve onboarding
6. **CLI metrics command** - Power user feature

### Long-term (3-6 months)
7. **Advanced telemetry features** - Request correlation, dependency maps
8. **Metric alerts** - Alerting integration
9. **Custom metrics** - Local dev forwarding

## Blockers

### Azure Extension Framework Auth Service

**Status**: Blocked (upstream dependency)  
**Priority**: High  
**Reference**: [Technical Debt - High Priority](../../../cli/docs/dev/todo.md#request-azd-extension-framework-auth-service-for-custom-scopes)

The azd extension framework provides `AZD_ACCESS_TOKEN` for gRPC communication, but this token is scoped to Azure Resource Manager (`management.azure.com`). Extensions that need to call other Azure APIs (like Log Analytics `api.loganalytics.io`) cannot use this token.

**Current Workaround**:
Using `DefaultAzureCredential` which relies on Azure CLI credentials from `azd auth login`. This works but bypasses the extension framework's auth model.

**Impact on This Spec**:
- All Azure SDK integrations (logs, metrics, App Insights, management) currently use the workaround
- Inconsistent credential handling between ARM calls and other Azure API calls
- Cannot leverage azd's token caching/refresh for non-ARM APIs

**Required Before Implementation**:
Request azd core team to add a gRPC service to request tokens with custom scopes. See [todo.md](../../../cli/docs/dev/todo.md#request-azd-extension-framework-auth-service-for-custom-scopes) for detailed specification.

## Conclusion

The Azure SDK for Go provides rich monitoring capabilities beyond Log Analytics queries. Adding the Metrics SDK (Phase 1) provides immediate value with minimal risk:

✅ **Pros**:
- Real-time performance data
- Lower latency than logs
- Standard metrics without configuration
- Complements existing log functionality

⚠️ **Cons**:
- Additional dependency
- Slightly more complex architecture
- Requires metrics-specific UI

**Status**: DEFERRED - This enhancement will be implemented in a future branch after the Azure logs feature is complete and merged to main. The current `azlogs` branch remains focused on log streaming functionality only.

**Recommendation**: Proceed with Phase 1 (Metrics Integration) as a high-priority enhancement in a separate feature branch after logs implementation is stable.
