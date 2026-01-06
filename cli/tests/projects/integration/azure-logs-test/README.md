# Azure Logs Test Project

This test project contains services for every Azure host type currently supported by the `azd app logs` command with Azure cloud log streaming. It validates log streaming from Azure-deployed services into the azd-app dashboard.

## Supported Host Types

| Host Type | Service | Language | Priority | Log Source |
|-----------|---------|----------|----------|------------|
| Container Apps | `containerapp-api` | Node.js | P0 | ContainerAppConsoleLogs_CL |
| App Service | `appservice-web` | Python | P0 | AppServiceConsoleLogs |
| Azure Functions | `functions-worker` | TypeScript | P1 | FunctionAppLogs |

## Prerequisites

```bash
# Install required tools
azd auth login
az login

# Install azd app extension
azd config set alpha.extensions.enabled on
azd extension install jongio.azd.app
```

## Deployment

### 1. Initialize Environment

```bash
cd cli/tests/projects/azure-logs-test

# Create a new azd environment
azd env new azure-logs-test

# Set location (choose a region that supports all services)
azd env set AZURE_LOCATION eastus2
```

### 2. Provision Infrastructure

```bash
# Provision all Azure resources
azd provision
```

This creates:
- **Resource Group**: `rg-azure-logs-test`
- **Log Analytics Workspace**: Central hub for all logs
- **Application Insights**: For Functions telemetry
- **Container Registry**: For Container Apps
- **Container Apps Environment**: With log streaming enabled
- **App Service Plan + Web App**: With diagnostic settings
- **Function App**: With App Insights integration

### 3. Deploy Services

```bash
# Deploy all services
azd deploy
```

## Testing Log Streaming

### Local Development (Dashboard)

```bash
# Start local services with dashboard
azd app run

# The dashboard shows logs from locally running services
# Toggle to "Azure" mode to see Azure-deployed service logs
```

### Dashboard Mode Switching

The dashboard header contains a mode toggle:
- **[Local]**: Shows logs from locally running services (default when services are running)
- **[Azure]**: Shows logs from Azure-deployed services via Log Analytics
- **[All]**: Merged view with source badges on each log entry

### MCP Tool Testing

```bash
# Start MCP server
azd app mcp serve

# Use get_service_logs tool with source parameter:
# - source: "local" - Local logs only
# - source: "azure" - Azure logs only
# - source: "auto" - Matches dashboard mode or config
```

### Generate Test Logs

Each service has endpoints to generate sample logs:

```bash
# Container Apps
curl https://<containerapp-api-url>/generate-logs?count=10

# App Service
curl https://<appservice-web-url>/generate-logs?count=10

# Azure Functions
curl https://<functions-worker-url>/api/generate-logs?count=10
```

### Verify Log Analytics Queries

```bash
# Open Azure Portal > Log Analytics Workspace
# Run these queries:

# Container Apps
ContainerAppConsoleLogs_CL
| where ContainerAppName_s contains "containerapp-api"
| take 100

# App Service
AppServiceConsoleLogs
| where _ResourceId contains "appservice-web"
| take 100

# Functions
FunctionAppLogs
| where _ResourceId contains "func-"
| take 100
```

## Configuration

### azure.yaml Log Settings

```yaml
logs:
  filters:
    exclude: ["health check", "readiness probe"]
    includeBuiltins: true
  analytics:
    pollingInterval: 10s       # Log Analytics poll interval
    defaultTimespan: 30m       # Default query time window
    realtime: false            # Use low-latency streaming when supported
```

### Per-Service Custom Queries

```yaml
services:
  functions-worker:
    logs:
      analytics:
        query: |
          FunctionAppLogs
          | where _ResourceId contains "{serviceName}"
          | where TimeGenerated > ago({timespan})
          | project TimeGenerated, FunctionName, Message, Level
```

## Infrastructure Details

### Bicep Resources

All infrastructure uses native Bicep resources with diagnostic settings configured to send logs to the central Log Analytics workspace:

| Resource Type | Purpose |
|---------------|---------|
| `Microsoft.OperationalInsights/workspaces` | Log Analytics workspace |
| `Microsoft.Insights/components` | Application Insights |
| `Microsoft.ContainerRegistry/registries` | Container Registry |
| `Microsoft.App/managedEnvironments` | Container Apps Environment |
| `Microsoft.App/containerApps` | Container App |
| `Microsoft.Web/serverfarms` | App Service Plan |
| `Microsoft.Web/sites` | Web App / Function App |
| `Microsoft.Storage/storageAccounts` | Storage for Functions |

### Diagnostic Settings

All services are configured to send logs to the central Log Analytics workspace:

| Service | Log Categories |
|---------|---------------|
| Container Apps | Console logs via managed environment |
| App Service | AppServiceHTTPLogs, AppServiceConsoleLogs, AppServiceAppLogs |
| Functions | FunctionAppLogs + Application Insights |

## Cleanup

```bash
# Delete all Azure resources
azd down --force --purge
```

## Troubleshooting

### No Azure Logs Appearing

1. Check Azure connection status in dashboard header
2. Verify `azd auth login` is current
3. Ensure diagnostic settings are configured (check Azure Portal)
4. Log Analytics has 30-90 second ingestion delay

### Permission Errors

Required roles:
- `Log Analytics Reader` on workspace
- `Reader` on compute resources

### Log Analytics Workspace Not Found

The workspace is **automatically detected** in the following priority order:

1. `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` environment variable (recommended - set by `azd provision`)
2. `AZURE_LOG_ANALYTICS_WORKSPACE_ID` environment variable (resource ID)
3. Auto-discovery from resource group using Azure Resource Manager API

**Optional Override** - only needed if using custom environment variable names:

```yaml
logs:
  analytics:
    workspace: ${CUSTOM_WORKSPACE_VAR}  # Use your custom env var instead of default
```

Or set environment variables manually:
```bash
export AZURE_LOG_ANALYTICS_WORKSPACE_ID="..."
export AZURE_LOG_ANALYTICS_WORKSPACE_GUID="..."  # preferred for query APIs
```

## Related Documentation

- [Azure Cloud Log Streaming Specification](../../../docs/specs/azure-logs/spec.md)
- [Azure Cloud Log Streaming Tasks](../../../docs/specs/azure-logs/tasks.md)
- [azure.yaml Schema Reference](../../docs/schema/azure.yaml.md)
- [Azure Verified Modules](https://azure.github.io/Azure-Verified-Modules/)
