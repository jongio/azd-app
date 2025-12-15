# Azure Cloud Log Streaming

Stream logs from your Azure-deployed services directly into the azd-app dashboard.

## Overview

Azure log streaming enables you to view logs from Container Apps, App Service, and Azure Functions in the same dashboard as your local development logs. This provides a unified view of your application across development and production environments.

## Prerequisites

1. **Azure CLI** - Logged in with `az login`
2. **azd environment** - Project provisioned with `azd provision`
3. **Log Analytics Workspace** - Configured to receive logs from your services

## Quick Start

1. **Provision Azure resources with Log Analytics:**

```bash
azd provision
```

This creates your Log Analytics workspace and configures diagnostic settings. The workspace ID is automatically detected from bicep outputs.

2. **Start the dashboard:**

```bash
azd app run
```

3. **Switch to Azure mode** using the mode toggle in the dashboard header

The dashboard now shows:
- **Timeframe selector**: Choose `15m`, `30m`, `6h`, or `24h` to control the time window
- **Sync interval**: Configure auto-refresh at `10s`, `30s`, `1m`, or `5m`
- **View Query**: Inspect or edit the KQL query used for each service

4. **Optional: Configure analytics in azure.yaml**

```yaml
logs:
  analytics:
    pollingInterval: "30s"     # Auto-refresh interval
    defaultTimespan: "1h"      # Default time window
```

## Required Infrastructure

### Bicep Outputs

Your infrastructure must output the Log Analytics workspace information. Add these outputs to your main.bicep:

```bicep
// In main.bicep
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = monitoring.outputs.logAnalyticsWorkspaceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = monitoring.outputs.logAnalyticsWorkspaceName
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = monitoring.outputs.logAnalyticsWorkspaceGuid
```

### Log Analytics Workspace Module

```bicep
// infra/core/monitor/monitoring.bicep
param name string
param location string = resourceGroup().location
param tags object = {}

resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: name
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

output logAnalyticsWorkspaceId string = logAnalyticsWorkspace.id
output logAnalyticsWorkspaceName string = logAnalyticsWorkspace.name
output logAnalyticsWorkspaceGuid string = logAnalyticsWorkspace.properties.customerId
```

> **Important**: The `customerId` property is the workspace GUID required for Log Analytics queries. This is different from the resource ID.

### Diagnostic Settings for Container Apps

Container Apps must have diagnostic settings configured to send logs to Log Analytics:

```bicep
// infra/core/host/container-app.bicep
param containerAppId string
param logAnalyticsWorkspaceId string

resource diagnosticSettings 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'container-app-logs'
  scope: containerApp  // Reference to your container app resource
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'ContainerAppConsoleLogs'
        enabled: true
      }
      {
        category: 'ContainerAppSystemLogs'
        enabled: true
      }
    ]
  }
}
```

### Diagnostic Settings for App Service

```bicep
// infra/core/host/app-service.bicep
param appServiceId string
param logAnalyticsWorkspaceId string

resource diagnosticSettings 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'app-service-logs'
  scope: appService  // Reference to your app service resource
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'AppServiceConsoleLogs'
        enabled: true
      }
      {
        category: 'AppServiceHTTPLogs'
        enabled: true
      }
      {
        category: 'AppServiceAppLogs'
        enabled: true
      }
    ]
  }
}
```

### Diagnostic Settings for Azure Functions

```bicep
// infra/core/host/function-app.bicep
param functionAppId string
param logAnalyticsWorkspaceId string

resource diagnosticSettings 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'function-app-logs'
  scope: functionApp  // Reference to your function app resource
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      {
        category: 'FunctionAppLogs'
        enabled: true
      }
    ]
  }
}
```

## Configuration Options

### azure.yaml Schema

The new analytics-based configuration separates global workspace settings from service-level table/query overrides:

#### Project-Level (Global) Configuration

Configure workspace connection and default polling behavior:

```yaml
logs:
  analytics:
    workspace: "my-workspace-id"      # Log Analytics workspace ID (optional, auto-detected from bicep outputs)
    pollingInterval: "30s"            # How often to fetch new logs (default: 10s)
    defaultTimespan: "1h"             # Initial log history to fetch (default: 30m)
```

- **workspace**: Azure Log Analytics workspace ID. If omitted, automatically detected from bicep outputs (`AZURE_LOG_ANALYTICS_WORKSPACE_ID`)
- **pollingInterval**: Frequency of auto-refresh in Azure mode. Options: `10s`, `30s`, `1m`, `5m`
- **defaultTimespan**: Default time window for queries. Options: `15m`, `30m`, `1h`, `6h`, `24h`

#### Service-Level Configuration

Override log tables or provide custom KQL queries per service:

```yaml
services:
  api:
    host: containerApp
    logs:
      analytics:
        # Option 1: Specify tables to query (uses auto-generated KQL)
        tables:
          - ContainerAppConsoleLogs_CL
          - ContainerAppSystemLogs_CL
        
        # Option 2: Provide custom KQL query (takes precedence over tables)
        query: |
          ContainerAppConsoleLogs_CL
          | where ContainerAppName_s == 'api'
          | where Log_s !contains "health"
          | project TimeGenerated, Log_s, Stream_s
          | order by TimeGenerated desc
```

**Service analytics options:**

- **tables**: Array of Log Analytics table names. When specified, azd generates a KQL query using these tables with automatic service name filtering.
- **query**: Custom KQL query string. Use `{serviceName}` and `{timespan}` placeholders. This takes precedence over `tables`.

**Examples:**

```yaml
# Example 1: Container App with default tables
services:
  web:
    host: containerApp
    logs:
      analytics: {}  # Uses default ContainerAppConsoleLogs_CL

# Example 2: Azure Functions with specific tables
services:
  api:
    host: function
    logs:
      analytics:
        tables:
          - FunctionAppLogs
          - AppServiceConsoleLogs

# Example 3: Custom query with filters
services:
  worker:
    host: containerApp
    logs:
      analytics:
        query: |
          union ContainerAppConsoleLogs_CL, ContainerAppSystemLogs_CL
          | where ContainerAppName_s == '{serviceName}'
          | where TimeGenerated > ago({timespan})
          | where Log_s !contains "DEBUG"
          | project TimeGenerated, Log_s, Level_s
          | order by TimeGenerated desc
          | take 1000
```

## CLI Usage

### View Azure Logs

```bash
# View recent Azure logs (uses default timespan from config)
azd app logs --source azure

# View specific service
azd app logs --source azure api

# View last hour of logs
azd app logs --source azure --since 1h

# Follow Azure logs (live streaming with polling)
azd app logs --source azure --follow

# View both local and Azure logs
azd app logs --source all

# Filter by log level
azd app logs --source azure --level error
```

### Dashboard Mode

The dashboard supports three log modes:

- **Local** (default) - Logs from locally running services
- **Azure** - Logs from Azure-deployed services with:
  - **Timeframe selector**: Adjust time window (15m, 30m, 1h, 6h, 24h, or custom)
  - **Sync interval**: Control auto-refresh frequency (10s, 30s, 1m, 5m)
  - **Query viewer**: View and edit KQL queries per service
- **All** - Combined view of both sources

Toggle modes using the mode selector in the dashboard header or with keyboard shortcut `Ctrl+Shift+M`.

## Troubleshooting

### "Azure logs not configured"

**Cause**: Missing workspace information in azd environment.

**Solution**: 
1. Ensure your bicep outputs include `AZURE_LOG_ANALYTICS_WORKSPACE_ID` and `AZURE_LOG_ANALYTICS_WORKSPACE_GUID`
2. Run `azd provision` to update environment values
3. Verify with `azd env get-values | grep LOG_ANALYTICS`
4. Alternatively, set `logs.analytics.workspace` explicitly in azure.yaml

### "No Azure logs found"

**Cause**: Diagnostic settings not configured, logs not yet ingested, or time window too narrow.

**Solution**:
1. Verify diagnostic settings exist:
   ```bash
   az monitor diagnostic-settings list --resource <resource-id>
   ```
2. Check if logs exist in Log Analytics:
   ```bash
   az monitor log-analytics query \
     -w <workspace-id> \
     --analytics-query "ContainerAppConsoleLogs_CL | take 10"
   ```
3. **Expand time window**: Use the timeframe selector in the dashboard to try `1h`, `6h`, or `24h`
4. Note: Log Analytics has ingestion delay of 1-5 minutes

### "Authentication failed"

**Cause**: Azure credentials not valid or expired.

**Solution**:
1. Re-authenticate with Azure:
   ```bash
   az login
   azd auth login
   ```
2. Verify subscription access:
   ```bash
   az account show
   ```

### "Workspace not found"

**Cause**: Log Analytics workspace ID is incorrect or inaccessible.

**Solution**:
1. Verify workspace exists:
   ```bash
   az monitor log-analytics workspace show --resource-group <rg> --workspace-name <name>
   ```
2. Check you have Reader access to the workspace
3. Verify `AZURE_LOG_ANALYTICS_WORKSPACE_ID` in `.azure/<env>/.env` matches the actual workspace

### Logs appear in Azure Portal but not in dashboard

**Cause**: Table name mismatch, KQL syntax error, or service name filtering issue.

**Solution**:
1. Click **View Query** in the dashboard's Azure logs bar to see the KQL being executed
2. Copy the query and test it in Azure Portal's Log Analytics query editor
3. Check if the service name placeholder `{serviceName}` matches your Azure resource name
4. Override with a custom query in azure.yaml if default tables don't match your setup:
   ```yaml
   services:
     myservice:
       logs:
         analytics:
           query: |
             ContainerAppConsoleLogs_CL
             | where TimeGenerated > ago({timespan})
             | project TimeGenerated, Log_s
   ```

### Slow or stale logs in Azure mode

**Cause**: Polling interval too long, or Log Analytics ingestion delay.

**Solution**:
1. Adjust **Sync interval** in the dashboard to `10s` for faster refresh
2. Use the **Refresh** button to manually fetch latest logs
3. Set shorter `pollingInterval` in azure.yaml:
   ```yaml
   logs:
     analytics:
       pollingInterval: "10s"
   ```
4. Note: Azure Log Analytics has a 1-5 minute ingestion delay; you cannot get true real-time logs

## Permissions Required

The following Azure RBAC roles are needed:

| Role | Scope | Purpose |
|------|-------|---------|
| Reader | Resource Group | List Azure resources |
| Log Analytics Reader | Log Analytics Workspace | Query logs |

Minimum permissions can be assigned with:

```bash
# Grant Log Analytics Reader to workspace
az role assignment create \
  --assignee <user-or-service-principal> \
  --role "Log Analytics Reader" \
  --scope /subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.OperationalInsights/workspaces/<workspace>
```

## Known Limitations

1. **Ingestion Delay**: Log Analytics has a 1-5 minute ingestion delay. Real-time streaming uses polling to approximate live logs.

2. **Query Limits**: Log Analytics queries are limited to 500,000 records per query. Use `--tail` and `--since` to limit results.

3. **Authentication**: Azure logs require Azure CLI or azd authentication. Managed identity is not yet supported for local development.

4. **Resource Types**: Currently supports Container Apps, App Service, and Azure Functions. AKS and ACI support planned for future releases.
