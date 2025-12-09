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
  azure:
    enabled: true              # Enable Azure log streaming
    source: auto               # auto|local|azure
    pollingInterval: 10s       # Log Analytics poll interval
    defaultTimespan: 30m       # Default query time window
    realtime: false            # Use real-time APIs where available
```

### Per-Service Custom Queries

```yaml
services:
  functions-worker:
    logs:
      azure:
        query: |
          FunctionAppLogs
          | where _ResourceId contains "{serviceName}"
          | where TimeGenerated > ago({timespan})
          | project TimeGenerated, FunctionName, Message, Level
```

## Infrastructure Details

### Azure Verified Modules Used

| Module | Version | Purpose |
|--------|---------|---------|
| `avm/res/operational-insights/workspace` | 0.10.0 | Log Analytics workspace |
| `avm/res/insights/component` | 0.6.0 | Application Insights |
| `avm/res/container-registry/registry` | 0.8.0 | Container Registry |
| `avm/res/app/managed-environment` | 0.10.0 | Container Apps Environment |
| `avm/res/app/container-app` | 0.13.0 | Container App |
| `avm/res/web/serverfarm` | 0.4.0 | App Service Plan |
| `avm/res/web/site` | 0.13.0 | Web App / Function App |
| `avm/res/storage/storage-account` | 0.15.0 | Storage for Functions |

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

The workspace is auto-detected from diagnostic settings. Override with:

```yaml
logs:
  azure:
    workspace: "/subscriptions/.../workspaces/..."
```

Or set environment variable:
```bash
export AZURE_LOG_ANALYTICS_WORKSPACE_ID="..."
```

## Related Documentation

- [Azure Cloud Log Streaming Specification](../../../docs/specs/azure-logs/spec.md)
- [Azure Cloud Log Streaming Tasks](../../../docs/specs/azure-logs/tasks.md)
- [azure.yaml Schema Reference](../../docs/schema/azure.yaml.md)
- [Azure Verified Modules](https://azure.github.io/Azure-Verified-Modules/)
