# Azure Cloud Log Streaming

Stream logs from your Azure-deployed services directly into the azd-app dashboard.

## Overview

Azure log streaming enables you to view logs from Container Apps, App Service, and Azure Functions in the same dashboard as your local development logs. This provides a unified view of your application across development and production environments.

## Prerequisites

1. **Azure CLI** - Logged in with `az login`
2. **azd environment** - Project provisioned with `azd provision`
3. **Log Analytics Workspace** - Configured to receive logs from your services

## Quick Start

1. **Enable Azure logs in azure.yaml:**

```yaml
logs:
  azure:
    enabled: true
```

2. **Start the dashboard:**

```bash
azd app run
```

3. **Switch to Azure mode** using the mode toggle in the dashboard header, or use the CLI:

```bash
azd app logs --source azure
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

```yaml
logs:
  azure:
    enabled: true                    # Enable Azure log streaming
    pollingInterval: 10s             # How often to fetch new logs (default: 10s)
    defaultTimespan: 30m             # Initial log history to fetch (default: 30m)
    
    # Custom KQL queries per resource type (optional)
    queries:
      containerApp: |
        ContainerAppConsoleLogs_CL
        | where ContainerAppName_s == '{serviceName}'
        | project TimeGenerated, Log_s, Stream_s
        | order by TimeGenerated desc
      appService: |
        AppServiceConsoleLogs
        | where _ResourceId contains '{serviceName}'
        | project TimeGenerated, ResultDescription
        | order by TimeGenerated desc
```

### Service-Level Overrides

Override Azure log settings for specific services:

```yaml
services:
  api:
    host: containerApp
    logs:
      azure:
        enabled: true
        queries:
          containerApp: |
            ContainerAppConsoleLogs_CL
            | where ContainerAppName_s == 'api'
            | where Log_s !contains "health"
            | project TimeGenerated, Log_s
```

## CLI Usage

### View Azure Logs

```bash
# View recent Azure logs
azd app logs --source azure

# View specific service
azd app logs --source azure api

# Follow Azure logs (live streaming)
azd app logs --source azure --follow

# View both local and Azure logs
azd app logs --source all

# Filter by log level
azd app logs --source azure --level error

# View last hour of logs
azd app logs --source azure --since 1h
```

### Dashboard Mode

The dashboard supports three log modes:

- **Local** (default) - Logs from locally running services
- **Azure** - Logs from Azure-deployed services
- **All** - Combined view of both sources

Toggle modes using the mode selector in the dashboard header or with keyboard shortcut `Ctrl+Shift+M`.

## Troubleshooting

### "Azure logs not configured"

**Cause**: Missing workspace information in azd environment.

**Solution**: 
1. Ensure your bicep outputs include `AZURE_LOG_ANALYTICS_WORKSPACE_ID` and `AZURE_LOG_ANALYTICS_WORKSPACE_GUID`
2. Run `azd provision` to update environment values
3. Verify with `azd env get-values | grep LOG_ANALYTICS`

### "No Azure logs found"

**Cause**: Diagnostic settings not configured or logs not yet ingested.

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
3. Note: Log Analytics has ingestion delay of 1-5 minutes

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

### Logs appear in Azure Portal but not in dashboard

**Cause**: KQL query mismatch or different log table schema.

**Solution**:
1. Test the default query in Azure Portal Log Analytics
2. Check if custom query in azure.yaml has syntax errors
3. Verify service names match between azd and Azure resources

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
