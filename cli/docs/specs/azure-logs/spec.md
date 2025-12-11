# Azure Cloud Log Streaming Specification

## Overview

Enable the azd-app dashboard to display real-time and historical logs from Azure-deployed services alongside local service logs, providing a unified logging experience for developers debugging applications across local and cloud environments.

## Problem Statement

Currently, the azd-app dashboard excels at displaying logs from locally running services via WebSocket streaming. However, once services are deployed to Azure, developers must switch contexts to Azure Portal, CLI tools, or multiple Azure-specific interfaces to view logs. This creates friction in the development workflow, especially when debugging issues that span local and cloud environments.

## Goals

1. Stream logs from Azure-deployed services into the existing dashboard UI
2. Support all major Azure compute platforms used with azd
3. Maintain consistency with existing local log viewing experience
4. Minimize authentication friction by leveraging existing azd credentials
5. Provide both real-time streaming and historical log query capabilities

## Non-Goals

1. Replace Azure Monitor, Log Analytics, or Application Insights as primary log analysis tools
2. Support non-azd-managed Azure resources
3. Implement log alerting or automated analysis
4. Store Azure logs locally beyond the existing in-memory buffer

## Supported Azure Compute Services

| Service | Real-time Streaming | Historical Queries | Priority |
|---------|--------------------|--------------------|----------|
| Azure Container Apps | Yes (via CLI API) | Yes (Log Analytics) | P0 |
| Azure App Service | Yes (via log stream) | Yes (Log Analytics) | P0 |
| Azure Functions | Yes (Live Metrics) | Yes (App Insights) | P1 |
| Azure Kubernetes Service | Yes (Live Logs) | Yes (Container Insights) | P1 |
| Azure Container Instances | Yes (attach) | Yes (Log Analytics) | P2 |

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Dashboard (React)                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │
│  │   LogsPane      │  │   LogsView      │  │   Source Toggle         │ │
│  │   (per service) │  │   (unified)     │  │   [Local] [Azure] [All] │ │
│  └────────┬────────┘  └────────┬────────┘  └────────────┬────────────┘ │
│           │                    │                        │               │
│           └────────────────────┼────────────────────────┘               │
│                               │                                         │
│                    WebSocket /api/logs/stream                           │
└───────────────────────────────┼─────────────────────────────────────────┘
                                │
┌───────────────────────────────┼─────────────────────────────────────────┐
│                        Backend (Go)                                      │
│  ┌────────────────────────────┴────────────────────────────────────┐   │
│  │                     Log Aggregator                               │   │
│  │   ┌──────────────────┐      ┌──────────────────┐                │   │
│  │   │ Local LogBuffer  │      │  Azure LogBuffer │                │   │
│  │   │  (existing)      │      │    (new)         │                │   │
│  │   └────────┬─────────┘      └────────┬─────────┘                │   │
│  │            │                         │                           │   │
│  └────────────┼─────────────────────────┼───────────────────────────┘   │
│               │                         │                               │
│   ┌───────────┴───────────┐   ┌────────┴──────────────────────────┐   │
│   │  Service Executor     │   │     Azure Log Provider            │   │
│   │  stdout/stderr pipes  │   │  ┌─────────────────────────────┐  │   │
│   └───────────────────────┘   │  │  Log Analytics Query Client │  │   │
│                               │  │  Container Apps Log Stream  │  │   │
│                               │  │  App Service Log Stream     │  │   │
│                               │  │  Functions Live Metrics     │  │   │
│                               │  │  AKS Container Insights     │  │   │
│                               │  └─────────────────────────────┘  │   │
│                               └───────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                         │
                                         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Azure                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │
│  │ Container Apps  │  │   App Service   │  │      Functions          │ │
│  │  Log Stream API │  │  Kudu/LogStream │  │   App Insights          │ │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────────────┘ │
│           │                    │                    │                   │
│           └────────────────────┼────────────────────┘                   │
│                               │                                         │
│                    ┌──────────┴──────────┐                              │
│                    │   Log Analytics     │                              │
│                    │     Workspace       │                              │
│                    └─────────────────────┘                              │
└─────────────────────────────────────────────────────────────────────────┘
```

### Azure Resource Discovery

Leverage existing `azd env get-values` patterns to discover deployed Azure resources:

```
SERVICE_{NAME}_URL         → Deployed service endpoint
SERVICE_{NAME}_NAME        → Azure resource name
AZURE_SUBSCRIPTION_ID      → Subscription context
AZURE_RESOURCE_GROUP_NAME  → Resource group
AZURE_ENV_NAME             → Environment name
```

Additional discovery via Azure Resource Manager:
- Query resource group for resources matching service names
- Detect resource type (Microsoft.App/containerApps, Microsoft.Web/sites, etc.)
- Cache resource metadata for efficient access

### Authentication Strategy

1. **Primary: azd Credential Chain**
   - Use credentials already configured via `azd auth login`
   - Access via azd extension framework (`AZD_ACCESS_TOKEN`)
   - Falls back through DefaultAzureCredential chain

2. **Credential Flow**
   ```
   azd auth login (user)
         │
         ▼
   AZD_ACCESS_TOKEN (extension context)
         │
         ▼
   Azure SDK DefaultAzureCredential
         │
         ▼
   Azure APIs (Log Analytics, ARM, etc.)
   ```

3. **Required Permissions**
   - `Log Analytics Reader` on workspace
   - `Reader` on compute resources
   - For live streaming: compute-specific permissions

### Log Streaming Strategies

#### Strategy 1: Log Analytics Unified Queries (Recommended Primary)

Use Azure Monitor Logs API with KQL for consistent cross-service log access:

```kql
// Container Apps
ContainerAppConsoleLogs_CL
| where ContainerAppName_s == "{serviceName}"
| where TimeGenerated > ago(5m)
| project TimeGenerated, Log_s, Stream_s

// App Service  
AppServiceConsoleLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago(5m)
| project TimeGenerated, ResultDescription

// Functions
FunctionAppLogs
| where _ResourceId contains "{serviceName}"
| where TimeGenerated > ago(5m)
| project TimeGenerated, Message, Level
```

**Pros:**
- Unified API across all services
- Full KQL query capabilities
- Historical data access
- Cross-service queries possible

**Cons:**
- 30-90 second ingestion delay
- Requires Log Analytics workspace configuration
- Cost per GB ingested

#### Strategy 2: Service-Specific Real-time APIs

For low-latency real-time streaming, use service-specific APIs:

**Container Apps:**
```bash
# Via Azure CLI API equivalent
POST /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.App/containerApps/{app}/getLogStream
```

**App Service:**
```
GET https://{app}.scm.azurewebsites.net/api/logstream
Authorization: Bearer {token}
```

**Functions:**
- Application Insights Live Metrics Stream
- QuickPulse SDK integration

**Pros:**
- Near real-time (seconds latency)
- No Log Analytics dependency for basic streaming

**Cons:**
- Service-specific implementations
- Different authentication per service
- No historical data

### Hybrid Approach (Recommended)

Implement both strategies with intelligent switching:

1. **Default Mode: Polling Log Analytics**
   - Query every 5-10 seconds
   - Show logs from past N minutes on connect
   - Works for all services uniformly

2. **Live Mode: Real-time Streaming**
   - Activated by user toggle
   - Uses service-specific APIs
   - Falls back to polling on failure

3. **Historical Mode: Log Analytics Query**
   - User-specified time range
   - Full KQL search/filter
   - Export capability

### Data Model Extension

Extend existing `LogEntry` type:

```go
type LogEntry struct {
    Service   string    `json:"service"`
    Message   string    `json:"message"`
    Level     LogLevel  `json:"level"`
    Timestamp time.Time `json:"timestamp"`
    IsStderr  bool      `json:"isStderr"`
    
    // New fields for Azure logs
    Source    LogSource `json:"source"`    // LOCAL, AZURE
    AzureMetadata *AzureLogMetadata `json:"azureMetadata,omitempty"`
}

type LogSource string
const (
    LogSourceLocal LogSource = "local"
    LogSourceAzure LogSource = "azure"
)

type AzureLogMetadata struct {
    ResourceID    string `json:"resourceId"`
    ResourceType  string `json:"resourceType"`   // containerApp, appService, function
    ContainerName string `json:"containerName,omitempty"`
    InstanceID    string `json:"instanceId,omitempty"`
}
```

### MCP Server Integration

The existing MCP server (`azd app mcp serve`) exposes tools for AI assistant integration. Extend it to support Azure log queries with mode awareness.

#### Extended `get_service_logs` Tool

The existing tool gains a `source` parameter:

```json
{
  "name": "get_service_logs",
  "description": "Get logs from services. Supports local running services and Azure-deployed services.",
  "parameters": {
    "projectDir": "Optional project directory path",
    "serviceName": "Optional service name filter",
    "tail": "Number of recent log lines (default: 100)",
    "level": "Filter: info|warn|error|debug|all",
    "since": "Duration: 5m, 1h, 30s",
    "source": "Log source: auto|local|azure (default: auto)"
  }
}
```

**Source Resolution Logic:**
1. If `source` parameter provided → use that
2. Else if dashboard is running → match dashboard's current mode
3. Else default to `auto` (local if services running, else azure)

#### New `get_azure_logs` Tool

Dedicated tool for Azure-specific queries:

```json
{
  "name": "get_azure_logs",
  "description": "Query logs from Azure-deployed services via Log Analytics",
  "parameters": {
    "projectDir": "Optional project directory path",
    "serviceName": "Service name to query (required)",
    "timespan": "Time window: 5m, 1h, 24h (default: 30m)",
    "query": "Optional custom KQL query (overrides default)",
    "limit": "Max results to return (default: 100)"
  }
}
```

#### New `get_azure_status` Tool

```json
{
  "name": "get_azure_status",
  "description": "Get Azure connection status and discovered resources",
  "parameters": {
    "projectDir": "Optional project directory path"
  },
  "returns": {
    "connected": "boolean",
    "subscription": "Azure subscription ID",
    "resourceGroup": "Resource group name",
    "workspace": "Log Analytics workspace ID",
    "resources": ["List of discovered Azure resources with types"]
  }
}
```

#### Mode Synchronization

The MCP server can query and respect the dashboard's current mode:

1. **Dashboard Running**: MCP queries dashboard API for current source mode
2. **Dashboard Not Running**: MCP uses azure.yaml config or parameter override
3. **Explicit Override**: `source` parameter always takes precedence

This ensures AI assistants see the same logs the user sees in the dashboard.

### API Design

#### New Endpoints

```
GET  /api/azure/services                    # List Azure-deployed services
GET  /api/azure/logs?service={name}         # Fetch historical Azure logs
WS   /api/azure/logs/stream?service={name}  # Stream Azure logs
POST /api/azure/logs/query                  # KQL query execution
GET  /api/azure/status                      # Azure connection status
GET  /api/mode                              # Get current dashboard mode
PUT  /api/mode                              # Set dashboard mode
```

#### Extended Existing Endpoints

```
WS   /api/logs/stream?source=all|local|azure   # Unified stream with source filter
GET  /api/logs?source=all|local|azure          # Unified logs with source filter
GET  /api/services?source=all|local|azure      # Services with source filter
```

#### Mode API

```
GET /api/mode
Response: { "mode": "local", "available": ["local", "azure", "all"] }

PUT /api/mode
Body: { "mode": "azure" }
Response: { "mode": "azure", "status": "connected" }
```

### Dashboard Mode Switching

The dashboard operates in one of three modes, easily switchable by the user:

#### Mode Toggle UI

```
┌─────────────────────────────────────────────────────────────┐
│  azd app dashboard                    [Local] [Azure] [All] │
│                                       ─────────────────────│
│                                       │ ● Connected to Azure │
└─────────────────────────────────────────────────────────────┘
```

**Toggle Behavior:**
- **Local** (default when services running): Shows logs from locally running processes
- **Azure**: Shows logs from Azure-deployed services via Log Analytics/real-time APIs
- **All**: Merged view with source badges on each log entry

**Smart Auto-Detection:**
When `source: auto` is configured:
- If local services are running → default to Local
- If no local services but Azure resources exist → default to Azure
- User can always override with toggle

#### Mode Persistence

1. **Session State**: Current mode persists in browser session storage
2. **URL Parameter**: `?source=azure` allows deep linking to specific mode

#### Visual Indicators

| Element | Local Mode | Azure Mode | All Mode |
|---------|------------|------------|----------|
| Toggle highlight | Blue | Purple | Green |
| Log entry badge | None (implicit) | `[Azure]` badge | `[Local]`/`[Azure]` badges |
| Status indicator | "Services running" | "Connected to Azure" | Both indicators |
| Service list | Local services only | Azure resources only | Both with source icon |

### Dashboard UI Changes

1. **Mode Toggle (Primary)**
   - Prominent button group in header: `[Local] [Azure] [All]`
   - Keyboard shortcut: `Ctrl+Shift+M` to cycle modes
   - Persists preference in session storage
   - Respects azure.yaml default on fresh load

2. **Azure Connection Status**
   - Icon in header: ● green (connected), ○ yellow (connecting), ✕ red (error)
   - Tooltip shows: subscription, resource group, last sync time
   - Click to retry connection on error
   - Shows "Run `azd auth login`" on auth failure

3. **Service Cards Enhancement**
   - Source icon: 💻 (local) or ☁️ (azure) next to service name
   - In All mode: show both if service exists in both contexts
   - Azure services show resource type (Container App, App Service, etc.)

4. **Log Entry Display**
   - Azure logs include subtle `[Azure]` badge with timestamp offset indicator
   - Color coding consistent with local logs (info/warn/error)
   - Click Azure log entry → option to "View in Azure Portal"

5. **Historical Log Panel (Azure mode)**
   - Time range picker: Last 15m, 1h, 6h, 24h, Custom
   - KQL query input (collapsible advanced section)
   - "Load More" pagination for large result sets
   - Export to file (JSON/text)

6. **Settings Panel**
   - Default log source mode selector
   - Azure polling interval slider (5s - 60s)
   - Default time window selector
   - Log Analytics workspace ID override
   - "Test Azure Connection" button

### Configuration

Leverage the existing `logs` schema section in azure.yaml (already supports `filters` with `exclude` patterns and `includeBuiltins`). Extend it to support Azure log sources.

#### azure.yaml Schema Extensions

```yaml
name: my-app

# Project-level log configuration (applies to all services)
logs:
  # Existing filter configuration
  filters:
    exclude: ["npm warn", "Debugger listening"]
    includeBuiltins: true
  
  # NEW: Azure log configuration
  azure:
    enabled: true                    # Enable Azure log streaming (default: false)
    source: auto                     # auto | local | azure (default: auto)
    workspace: ""                    # Optional Log Analytics workspace ID override
    pollingInterval: 10s             # Polling interval for Log Analytics (default: 10s)
    defaultTimespan: 30m             # Default time window for historical queries (default: 30m)
    realtime: false                  # Enable real-time streaming where available (default: false)
    
    # Custom KQL queries per resource type (advanced)
    queries:
      containerApp: |
        ContainerAppConsoleLogs_CL
        | where ContainerAppName_s == "{serviceName}"
        | where TimeGenerated > ago({timespan})
        | project TimeGenerated, Log_s, Stream_s
        | order by TimeGenerated desc
      appService: |
        AppServiceConsoleLogs
        | where _ResourceId contains "{serviceName}"
        | where TimeGenerated > ago({timespan})
        | project TimeGenerated, ResultDescription
        | order by TimeGenerated desc
      function: |
        FunctionAppLogs
        | where _ResourceId contains "{serviceName}"
        | where TimeGenerated > ago({timespan})
        | project TimeGenerated, Message, Level
        | order by TimeGenerated desc

services:
  api:
    host: local
    project: ./api
    # Service-level log configuration (overrides project-level)
    logs:
      filters:
        exclude: ["health check"]
      azure:
        enabled: true
        # Custom query for this specific service
        query: |
          ContainerAppConsoleLogs_CL
          | where ContainerAppName_s == "api"
          | where Log_s !contains "health"
          | project TimeGenerated, Log_s
```

#### Log Source Modes

| Mode | Description |
|------|-------------|
| `auto` | Dashboard decides based on context: shows Azure logs when services not running locally |
| `local` | Force local logs only (existing behavior) |
| `azure` | Force Azure logs only |

The `source` setting establishes the default, but can be overridden at runtime via:
- Dashboard UI toggle
- MCP tool parameter
- API query parameter

#### Default KQL Queries

When no custom query is specified, use built-in queries optimized for each resource type:

| Resource Type | Default Table | Key Fields |
|---------------|---------------|------------|
| Container Apps | `ContainerAppConsoleLogs_CL` | `Log_s`, `Stream_s`, `ContainerAppName_s` |
| App Service | `AppServiceConsoleLogs` | `ResultDescription`, `Level` |
| Functions | `FunctionAppLogs` | `Message`, `Level`, `FunctionName` |
| AKS | `ContainerLogV2` | `LogMessage`, `PodName`, `ContainerName` |
| ACI | Container Logs API | Raw log content |

#### Environment Variables

```
# Azure configuration
AZURE_LOG_ANALYTICS_WORKSPACE_ID   # Default workspace (auto-detected if not set)
AZURE_LOGS_POLLING_INTERVAL        # Seconds (default: 10)
AZURE_LOGS_DEFAULT_TIMESPAN        # Minutes (default: 30)
AZURE_LOGS_SOURCE                  # Default source mode: auto|local|azure

# From azd environment (auto-populated)
AZURE_SUBSCRIPTION_ID              # Subscription for resource discovery
AZURE_RESOURCE_GROUP_NAME          # Resource group for deployed services
AZURE_ENV_NAME                     # azd environment name
```

### Error Handling

1. **Authentication Failures**
   - Prompt: "Azure authentication required. Run `azd auth login`"
   - Graceful fallback to local-only mode

2. **Permission Errors**
   - Show specific missing role
   - Link to Azure RBAC documentation

3. **Resource Not Found**
   - Service not deployed: "Service '{name}' not found in Azure"
   - Suggest running `azd provision`

4. **Rate Limiting**
   - Implement exponential backoff
   - Cache queries where appropriate

5. **Network Errors**
   - Retry with backoff
   - Maintain local log display during outage

### Performance Considerations

1. **Polling Efficiency**
   - Use `TimeGenerated > ago(10s)` to minimize data transfer
   - Track last fetched timestamp per service
   - Deduplicate logs by timestamp + message hash

2. **Connection Management**
   - Reuse HTTP clients with connection pooling
   - Limit concurrent Azure API connections
   - Implement request queuing

3. **Memory Management**
   - Same 1000-entry buffer limit for Azure logs
   - Separate buffers per service+source
   - LRU eviction for historical queries

4. **Caching**
   - Cache resource metadata (resource IDs, types)
   - Cache authentication tokens
   - Cache Log Analytics workspace ID

### Security Considerations

1. **Credential Handling**
   - Never log or expose tokens
   - Use secure token storage
   - Respect token expiration

2. **Data Sensitivity**
   - Logs may contain sensitive data
   - Warn users about local caching
   - Clear logs on session end option

3. **Network Security**
   - TLS for all Azure API calls
   - Validate SSL certificates
   - Respect private endpoints if configured

## Dependencies

### Go Packages (New)

```go
github.com/Azure/azure-sdk-for-go/sdk/azidentity          // Credentials
github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs // Log Analytics
github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources // Resource discovery
```

### Frontend (Existing sufficient)

- React, TypeScript (existing)
- WebSocket client (existing)
- No new major dependencies required

## Testing Strategy

1. **Unit Tests**
   - Mock Azure API responses
   - Test log parsing/transformation
   - Test error handling paths

2. **Integration Tests**
   - Use Azure SDK test fixtures
   - Test against live Azure (optional, CI flag)
   - Test credential flows

3. **E2E Tests**
   - Mock Azure backend for dashboard tests
   - Test source toggle behavior
   - Test error state displays

## Rollout Plan

### Phase 1: Foundation (P0)
- Azure resource discovery via azd
- Log Analytics query client
- Container Apps log support
- Basic UI source toggle

### Phase 2: Service Expansion (P1)
- App Service log streaming
- Azure Functions log support
- Real-time streaming mode

### Phase 3: Advanced Features (P2)
- AKS Container Insights integration
- KQL query builder UI
- Historical log export
- Azure Container Instances support

### Phase 4: Polish (P3)
- Performance optimization
- Advanced filtering
- Cross-service log correlation

## Success Metrics

1. **Functional**
   - Logs appear within 30 seconds of generation
   - All P0 compute services supported
   - Graceful degradation on errors

2. **User Experience**
   - Single click to switch local/Azure logs
   - No additional authentication steps
   - Consistent UI between local and Azure logs

3. **Performance**
   - Dashboard remains responsive with Azure logs
   - Memory usage within existing limits
   - No significant increase in API latency

## Open Questions

1. **Log Analytics Workspace Discovery**
   - Should we auto-detect workspace from resource diagnostic settings?
   - Or require explicit configuration?
   - **Decision**: Auto-detect with manual override option

2. **Multi-environment Support**
   - Show logs from multiple azd environments simultaneously?
   - **Decision**: Single environment at a time, environment switcher in UI

3. **Log Retention in Dashboard**
   - Separate buffer limits for local vs Azure?
   - **Decision**: Same limits, unified buffer management

4. **Offline Mode**
   - Cache last-fetched Azure logs for offline viewing?
   - **Decision**: Defer to Phase 4, focus on live connectivity first

## Appendix

### KQL Reference Queries

**Container Apps Console Logs:**
```kql
ContainerAppConsoleLogs_CL
| where ContainerAppName_s == "api"
| where TimeGenerated > ago(5m)
| project TimeGenerated, Log_s, Stream_s, RevisionName_s
| order by TimeGenerated desc
```

**App Service Logs:**
```kql
AppServiceConsoleLogs
| where _ResourceId contains "myapp"
| where TimeGenerated > ago(5m)
| project TimeGenerated, ResultDescription, Level
| order by TimeGenerated desc
```

**Function App Logs:**
```kql
FunctionAppLogs
| where FunctionName != ""
| where TimeGenerated > ago(5m)
| project TimeGenerated, FunctionName, Message, Level
| order by TimeGenerated desc
```

**AKS Container Logs:**
```kql
ContainerLogV2
| where PodNamespace == "default"
| where TimeGenerated > ago(5m)
| project TimeGenerated, PodName, ContainerName, LogMessage
| order by TimeGenerated desc
```

### Azure SDK Code Examples

**Credential Setup:**
```go
import "github.com/Azure/azure-sdk-for-go/sdk/azidentity"

cred, err := azidentity.NewDefaultAzureCredential(nil)
if err != nil {
    // Fallback to azd token if available
    cred, err = NewAzdTokenCredential(os.Getenv("AZD_ACCESS_TOKEN"))
}
```

**Log Analytics Query:**
```go
import "github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"

client, _ := azlogs.NewClient(cred, nil)
resp, _ := client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
    Query:    to.Ptr("ContainerAppConsoleLogs_CL | take 100"),
    Timespan: to.Ptr("PT5M"),
}, nil)
```
