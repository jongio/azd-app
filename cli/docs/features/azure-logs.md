# Azure Cloud Log Streaming

Stream logs from your Azure-deployed services directly into the azd-app dashboard.

## Overview

Azure log streaming enables you to view logs from Container Apps, App Service, and Azure Functions in the same dashboard as your local development logs. This provides a unified view of your application across development and production environments.

## Azure Logs Setup Guide

The **Azure Logs Setup Guide** is an interactive 4-step wizard that helps you configure your project to stream logs from Azure services to the dashboard. The guide automatically detects your current configuration status and provides actionable guidance for each setup step.

### Accessing the Setup Guide

The setup guide appears automatically when:
1. You click the **Azure** button in the mode toggle header
2. Azure logs are not yet fully configured

You can also access the setup guide from:
- **Diagnostics Modal**: Click the "Fix Setup" button when Azure logs show configuration issues
- **Error Displays**: Click the "Setup Guide" button shown in Azure log error messages
- **Deep Linking**: Jump directly to a specific step (e.g., from an error about missing diagnostic settings)

### Setup Steps

#### Step 1: Log Analytics Workspace

Configure a Log Analytics workspace in Azure to collect your application logs.

**What the guide checks:**
- ✅ Workspace is deployed in Azure
- ✅ Workspace ID is configured in your environment
- ✅ Bicep outputs are properly configured

**Quick fixes:**
- View example Bicep code for creating a workspace
- Copy-paste configuration for `azure.yaml`
- Auto-detect workspace after provisioning

**Common issues:**
- "Workspace not deployed": Run `azd provision` to create Azure resources
- "Missing workspace ID": Add Bicep outputs `AZURE_LOG_ANALYTICS_WORKSPACE_ID` and `AZURE_LOG_ANALYTICS_WORKSPACE_GUID`
- "Invalid workspace ID": Verify the workspace exists in your subscription

#### Step 2: Authentication & Permissions

Sign in to Azure and verify you have Log Analytics Reader permissions.

**What the guide checks:**
- ✅ Azure CLI authentication is active
- ✅ User has "Log Analytics Reader" role
- ✅ Can access the workspace successfully

**Quick fixes:**
- One-click login with `az login`
- Copy-paste role assignment commands
- Link to Azure Portal for manual role assignment

**Common issues:**
- "Not authenticated": Click "Sign In" to authenticate with Azure CLI
- "Permission denied": Assign "Log Analytics Reader" role to your user account
- "Authentication expired": Re-run `az login` to refresh credentials

#### Step 3: Diagnostic Settings

Enable diagnostic settings for your Azure services to send logs to the Log Analytics workspace.

**What the guide checks:**
- ✅ Automatically detects all services in your environment
- ✅ Verifies diagnostic settings configuration status for each service
- ✅ Displays aggregated status across all services
- ✅ Identifies resource types and required log categories

**Features:**
- **Aggregated status view**: See configuration status for all services at a glance
  - Green checkmark: All services configured
  - Orange warning: Partial or no configuration
- **Service list**: Shows each service with its resource type and status icon
  - ✓ Configured (green)
  - ○ Not Configured (gray)
  - ! Error (red)
- **Single Bicep template**: Generate consolidated template for all services at once (no per-service copying)
- **Auto-refresh**: Click "Recheck" after deploying changes to verify configuration
- **Clear guidance**: Step-by-step instructions for fixing configuration issues

**Bicep template modal:**
When you click "Show Bicep Template", a modal appears with:
- **Unified template**: Single Bicep file for all detected services
- **Integration instructions**: Collapsible section with step-by-step guidance on where to add the template
- **Syntax highlighting**: Easy-to-read Bicep code with proper formatting
- **Copy All**: One-click copy of entire template to clipboard
- **Download**: Save template as `diagnostic-settings.bicep` file
- **Service detection**: Automatically includes only services found in your environment

**Supported resource types:**
- Container Apps (`ContainerAppConsoleLogs`, `ContainerAppSystemLogs`)
- App Service (`AppServiceConsoleLogs`, `AppServiceHTTPLogs`, `AppServiceAppLogs`)
- Azure Functions (`FunctionAppLogs`)
- Container Registry, Storage Account, Key Vault (if detected)

**Typical workflow:**
1. Guide auto-detects services and checks configuration status
2. If not configured, click "Show Bicep Template"
3. Follow integration instructions in modal to add template to your infra
4. Run `azd up` to deploy changes
5. Click "Recheck" to verify configuration
6. Click "Next" when all services show as configured

**Common issues:**
- **"Diagnostic settings missing"**: Click "Show Bicep Template" and add the generated template to your infrastructure
- **"Wrong workspace"**: Ensure diagnostic settings use the correct `logAnalyticsWorkspaceId` parameter
- **"Logs not enabled"**: The generated template includes all required log categories automatically
- **"No services found"**: Run `azd provision` to deploy your infrastructure first
- **"Permission denied"**: Ensure you have Reader role on the resource group or resources

#### Step 4: Verification

Test workspace connectivity by querying for recent logs from your services.

**What the guide checks:**
- ✅ All previous steps are complete
- ✅ Can successfully authenticate to workspace
- ✅ Can query logs via Log Analytics API
- ✅ Services are generating logs

**Features:**
- **Real workspace queries**: Actually queries your Log Analytics workspace for logs from the last 15 minutes
- **Per-service verification**: Shows log counts and timestamps for each service individually
- **Status indicators**: 
  - ✓ OK (green): Logs found, service is working
  - ⚠ No logs (orange): No recent logs, but this may be normal
  - ✗ Error (red): Diagnostic settings not configured or query failed
- **Actionable guidance**: Provides specific next steps based on results
- **Service details**: Displays log count and last log timestamp for each service
- **Completion flow**: Success state shows celebration message and "View Logs" button

**Verification states:**

1. **Idle** (Initial):
   - Shows "Ready to verify" message
   - Displays "Start Verification" button
   - No queries executed until user clicks

2. **Verifying** (In Progress):
   - Shows loading spinner
   - Displays "Testing connection to Log Analytics workspace..."
   - Queries workspace for recent logs from each service

3. **Success - All Verified** (Best Case):
   - Green success banner: "All N services verified"
   - Service cards showing log counts (e.g., "5 log entries found in last 15m")
   - Last log timestamp for each service
   - Celebration message: "Setup Complete! 🎉"
   - "View Logs" button to navigate to Azure logs view
   - "Recheck" button to run verification again

4. **Partial Success** (Some Services Working):
   - Orange warning banner: "2 of 3 services verified"
   - Mixed service results:
     - Services with logs: Green checkmark, log count, timestamp
     - Services without logs: Orange warning, explanatory message
   - Guidance list explaining each service status
   - "Retry" button to recheck
   - "View Logs Anyway" button (enabled if at least 1 service has logs)
   - "Complete Setup" button to finish setup flow

5. **Error** (Failed):
   - Red error banner: "Verification failed"
   - Error message with details
   - Service cards showing error reasons (if available)
   - "Retry" button to try again
   - "← Back to Diagnostic Settings" link to fix configuration

**Verification logic:**
- Queries each service's logs table for entries in the last 15 minutes
- Returns log counts and most recent timestamp
- Detects common issues:
  - No diagnostic settings configured
  - Logs not yet ingested (normal 2-5 minute delay)
  - Permission errors
  - Invalid queries
  - Service hasn't generated logs yet (normal if service is idle)

**Typical results:**

```
Services:
● api (Container App)
  15 log entries found in last 15 minutes
  Last log: 12/25/2025, 10:45:23 AM
  
● web (App Service)  
  8 log entries found in last 15 minutes
  Last log: 12/25/2025, 10:44:50 AM
  
● worker (Azure Function)
  No logs found in last 15 minutes
  This may be normal if the function hasn't run yet or if 
  diagnostic settings were just configured (allow 2-5 minutes 
  for ingestion).
```

**Understanding "No Logs" status:**

A service showing "No logs" doesn't necessarily indicate a problem. This is normal if:
- The service hasn't generated any activity recently (e.g., function not triggered, idle app)
- Diagnostic settings were just configured (Log Analytics has 2-5 minute ingestion delay)
- The service is starting up or not yet deployed

You can still complete setup and view logs later when services become active.

**Common issues:**
- **"No logs found"**: 
  - Wait 2-5 minutes after configuring diagnostic settings (ingestion delay)
  - Trigger service activity (visit web app, invoke function, etc.)
  - Check that service is running in Azure Portal
- **"Authentication failed"**: 
  - Re-run `azd auth login` or `az login`
  - Verify Log Analytics Reader role is assigned
- **"Diagnostic settings not configured"**: 
  - Click "← Back to Diagnostic Settings" 
  - Complete Step 3 configuration
  - Run `azd up` to deploy changes
- **"Query failed"**: 
  - Verify workspace ID is correct
  - Check network connectivity
  - Ensure workspace exists and is accessible

**Recovery actions:**

Every error state provides clear next steps:
- "Retry" button to rerun verification
- Link back to diagnostic settings if that's the issue
- Option to complete setup anyway if partial success
- Guidance messages explain what to do next

### Setup Guide Features

#### Auto-Detection
The guide continuously monitors your setup status and updates in real-time:
- **Polling**: Checks configuration every 5 seconds
- **Status badges**: Green (✓) for complete, yellow (!) for issues
- **Progress tracking**: Saves your progress across page reloads

#### Deep Linking
Jump directly to specific steps from anywhere in the dashboard:
- Error messages link to relevant setup steps
- Diagnostics modal identifies which step needs attention
- URL supports `?setupStep=auth` query parameters

#### Progress Persistence
Your setup progress is automatically saved:
- **LocalStorage**: Remembers completed steps and current position
- **Step validation**: Each step validates independently
- **Resume anytime**: Close and reopen the guide without losing progress

#### Code Examples
Every step provides copy-paste code snippets:
- **Bicep templates**: Infrastructure-as-code for each resource
- **Azure YAML**: Configuration snippets for `azure.yaml`
- **CLI commands**: PowerShell/Bash commands for manual setup
- **Copy button**: One-click copy with visual confirmation

### Navigation

- **Next/Back buttons**: Navigate sequentially through steps
- **Step indicators**: Click any step to jump directly
- **Keyboard shortcuts**: 
  - `Esc` - Close the setup guide
  - `Arrow keys` - Navigate between steps (when focused)
- **Close button**: Exit the guide and return to dashboard

### Integration Points

The setup guide integrates seamlessly with other dashboard features:

- **ModeToggle**: Triggers setup guide when switching to Azure mode
- **ConsoleView**: Shows "Setup Guide" button in Azure log errors
- **DiagnosticsModal**: Displays "Fix Setup" button for configuration issues
- **AzureErrorDisplay**: Links to specific setup steps based on error type

### Completion

When all steps are complete:
1. ✅ Setup verification shows success for all services
2. 🎉 "Complete Setup" button becomes active
3. Clicking complete automatically switches to Azure logs view
4. Dashboard begins streaming logs from your Azure services

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
    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}  # OPTIONAL - auto-detected if omitted
    pollingInterval: "30s"     # Auto-refresh interval
    defaultTimespan: "1h"      # Default time window
```

> **Note**: The `workspace` field is optional. If omitted, the workspace is automatically discovered using:
> 1. `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` environment variable (recommended)
> 2. `AZURE_LOG_ANALYTICS_WORKSPACE_ID` environment variable
> 3. Auto-discovery from your resource group
>
> Only specify `workspace` if you're using a custom environment variable name or need to override the default detection.

## Required Infrastructure

### Bicep Outputs (Recommended)

For best performance, output the Log Analytics workspace information from your infrastructure:

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

### Diagnostic Settings - Unified Approach

**New in Setup Guide**: The Azure Logs Setup Guide now generates a unified Bicep template for all your services in one file, replacing the need for per-service configuration.

#### Using the Setup Guide Template Generator

The recommended approach is to use the setup guide's built-in template generator:

1. Run `azd app run` and switch to Azure mode
2. Open the Azure Logs Setup Guide
3. Navigate to Step 3: Diagnostic Settings
4. Click "Show Bicep Template"
5. The modal will display a complete template for all detected services

The generated template includes:
- Diagnostic settings for all detected service types
- Correct log categories for each resource type
- Log Analytics workspace parameter integration
- Comments and integration instructions

**Generated template structure:**

```bicep
// infra/modules/diagnostic-settings.bicep
// Auto-generated by Azure Logs Setup Guide

param logAnalyticsWorkspaceId string

// Container Apps Diagnostic Settings
resource containerAppDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = [for app in containerApps: {
  name: 'containerapp-logs-${app.name}'
  scope: app
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      { category: 'ContainerAppConsoleLogs', enabled: true }
      { category: 'ContainerAppSystemLogs', enabled: true }
    ]
  }
}]

// App Service Diagnostic Settings
resource appServiceDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = [for app in appServices: {
  name: 'appservice-logs-${app.name}'
  scope: app
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      { category: 'AppServiceConsoleLogs', enabled: true }
      { category: 'AppServiceHTTPLogs', enabled: true }
      { category: 'AppServiceAppLogs', enabled: true }
    ]
  }
}]

// Azure Functions Diagnostic Settings
resource functionAppDiagnostics 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = [for func in functionApps: {
  name: 'function-logs-${func.name}'
  scope: func
  properties: {
    workspaceId: logAnalyticsWorkspaceId
    logs: [
      { category: 'FunctionAppLogs', enabled: true }
    ]
  }
}]
```

**Integration steps** (shown in the modal):

1. Save the generated template as `infra/modules/diagnostic-settings.bicep`
2. Add the workspace parameter to your `main.bicep` if not already present:
   ```bicep
   param logAnalyticsWorkspaceId string = monitoring.outputs.logAnalyticsWorkspaceId
   ```
3. Reference the module in `main.bicep`:
   ```bicep
   module diagnosticSettings './modules/diagnostic-settings.bicep' = {
     name: 'diagnostic-settings'
     params: {
       logAnalyticsWorkspaceId: logAnalyticsWorkspaceId
     }
   }
   ```
4. Run `azd up` to deploy the changes
5. Return to setup guide and click "Recheck" in Step 3

#### Manual Configuration (Per-Service)

If you prefer to configure diagnostic settings manually, here are examples for each service type:

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

> **Tip**: Using the setup guide's template generator is faster and less error-prone than manual configuration, as it automatically detects your services and generates the correct template structure.

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

### Setup Guide Issues

#### Setup guide won't open

**Symptoms**: Clicking "Azure" mode doesn't show the setup guide.

**Cause**: Setup guide only appears when Azure logs are not fully configured.

**Solution**:
1. Check if Azure logs are already working - if logs appear, setup is complete
2. Try opening diagnostics modal (`Ctrl+Shift+D`) to see current status
3. Clear browser localStorage and refresh: `localStorage.clear()`

#### Setup guide shows incorrect status

**Symptoms**: Step shows as incomplete but you've already configured it.

**Cause**: Setup state polling may be delayed or cached.

**Solution**:
1. Click the **Refresh** or **Recheck** button in the affected step
2. Wait 5-10 seconds for auto-refresh to detect changes
3. Close and reopen the setup guide to reset state
4. Run `azd env get-values` to verify environment variables are set

#### Can't proceed past a step

**Symptoms**: "Next" button is disabled on a step.

**Cause**: Step validation requires all checks to pass before proceeding.

**Solution**:
1. Look for error messages or warnings in red/orange boxes
2. Use the "Recheck" button to revalidate after making changes
3. Verify changes in Azure Portal if auto-detection isn't working
4. Check the specific issue below based on which step is failing

#### Deep linking doesn't work

**Symptoms**: URL with `?setupStep=auth` doesn't open the correct step.

**Cause**: Setup guide might be closed or step name is incorrect.

**Solution**:
1. Ensure setup guide is open (click Azure button)
2. Valid step names: `workspace`, `auth`, `diagnostic-settings`, `verification`
3. Check browser console for any JavaScript errors

### Diagnostic Settings (Step 3) Issues

#### "Could not check diagnostic settings"

**Symptoms**: Red error banner in Step 3 with error message.

**Cause**: API cannot query Azure Resource Manager for diagnostic settings status.

**Solution**:
1. Ensure you have **Reader** role on resources or resource group
2. Verify authentication: `azd auth login` or `az login`
3. Check network connectivity to Azure
4. Click **Retry** after fixing authentication/permissions
5. If persistent, click **Skip This Step** to proceed anyway

#### "No services found"

**Symptoms**: Orange warning "No Azure services were discovered in your project"

**Cause**: No Azure resources detected in environment, or resources not yet deployed.

**Solution**:
1. Run `azd provision` to deploy your infrastructure
2. Verify services exist in Azure Portal
3. Check `azd env get-values` shows service resource IDs
4. Click **Recheck** after provisioning

#### Diagnostic settings status stuck on "Checking..."

**Symptoms**: Loading spinner doesn't complete.

**Cause**: API timeout or network issue.

**Solution**:
1. Check browser Developer Console (F12) for network errors
2. Verify `azd app run` is still running
3. Refresh the browser page
4. Restart `azd app run` if unresponsive

#### Service shows as "Not Configured" after deploying Bicep

**Symptoms**: Deployed diagnostic settings but status still shows gray circle.

**Cause**: Bicep deployment succeeded but configuration not detected, or template was deployed to wrong resource.

**Solution**:
1. Click **Recheck** in Step 3 to refresh status
2. Wait 30-60 seconds for Azure Resource Manager to update
3. Verify diagnostic settings in Azure Portal:
   ```bash
   az monitor diagnostic-settings list --resource <resource-id>
   ```
4. Ensure diagnostic settings `workspaceId` matches your Log Analytics workspace
5. Check that the Bicep template was added to the correct resource scope

#### "Permission denied" when checking settings

**Symptoms**: Service shows error status with permission message.

**Cause**: User doesn't have Reader role on the resource.

**Solution**:
1. Assign Reader role to your user account:
   ```bash
   az role assignment create \
     --assignee <user-email> \
     --role "Reader" \
     --scope <resource-id>
   ```
2. Wait 5-10 minutes for RBAC permissions to propagate
3. Re-authenticate: `az logout && az login`
4. Click **Retry** in Step 3

#### Bicep template modal won't load

**Symptoms**: Modal shows loading spinner indefinitely or displays error.

**Cause**: Template generation API failed or timed out.

**Solution**:
1. Check browser console for errors
2. Verify services were detected (check Step 3 service list)
3. Close modal and click "Show Bicep Template" again
4. If persistent, manually create diagnostic settings (see Manual Configuration section above)

### Verification (Step 4) Issues

#### "Verification failed" error

**Symptoms**: Red error banner in Step 4 after clicking "Start Verification".

**Cause**: Cannot query Log Analytics workspace or authentication failed.

**Solution**:
1. Verify authentication: `azd auth login` or `az login`
2. Ensure **Log Analytics Reader** role is assigned:
   ```bash
   az role assignment create \
     --assignee <user-email> \
     --role "Log Analytics Reader" \
     --scope <workspace-id>
   ```
3. Wait 5-10 minutes after assigning role for propagation
4. Check workspace exists and is accessible:
   ```bash
   az monitor log-analytics workspace show \
     --resource-group <rg> \
     --workspace-name <name>
   ```
5. Click **Retry**

#### All services show "No logs found"

**Symptoms**: Orange warning icons, "0 log entries found in last 15 minutes" for all services.

**Cause**: Services haven't generated logs, or diagnostic settings just configured (ingestion delay).

**Solution**:
1. **Wait 2-5 minutes** after configuring diagnostic settings (normal ingestion delay)
2. **Trigger service activity**:
   - Visit web app URL
   - Invoke Azure Function
   - Send traffic to container app
3. Click **Retry** after generating activity
4. Check if services are actually running in Azure Portal
5. Verify diagnostic settings are configured (go back to Step 3)
6. You can still click **Complete Setup** - logs will appear when services become active

#### Some services show logs, some don't (partial success)

**Symptoms**: Orange warning banner: "2 of 3 services verified"

**Cause**: Normal if some services are idle or haven't generated recent activity.

**Solution**:
- **This is often normal** - idle services won't have logs
- Services showing "No logs" with message "may be normal if service hasn't run yet" are OK
- Click **View Logs Anyway** to view logs from working services
- Click **Complete Setup** to finish - logs from other services will appear when they become active
- To verify all services, trigger activity and click **Retry**

#### "Diagnostic settings not configured" in verification

**Symptoms**: Red error icon for service: "DiagnosticSettingsNotConfigured"

**Cause**: Diagnostic settings missing or not properly configured for this service.

**Solution**:
1. Click **← Back to Diagnostic Settings** to return to Step 3
2. Verify service shows as "Configured" in Step 3
3. If not, click "Show Bicep Template" and deploy the template
4. Run `azd up` to apply changes
5. Wait 2-5 minutes for ingestion to begin
6. Return to Step 4 and click **Retry**

#### "Authentication failed" during verification

**Symptoms**: Error about authentication or permissions when querying workspace.

**Cause**: Azure CLI auth expired or Log Analytics Reader role missing.

**Solution**:
1. Re-authenticate:
   ```bash
   az logout
   az login
   ```
2. Verify Log Analytics Reader role:
   ```bash
   az role assignment list --assignee <user-email> --scope <workspace-id>
   ```
3. Assign role if missing (see "Verification failed" above)
4. Click **Retry** after fixing authentication

#### Verification queries timeout

**Symptoms**: Long wait time, then error "Query timed out" or network error.

**Cause**: Large workspace, slow network, or workspace query throttling.

**Solution**:
1. Check network connectivity
2. Verify workspace is in same region (latency)
3. Try again during off-peak hours if workspace is heavily used
4. Click **Retry** - timeouts are often transient

#### Verification stuck on "Testing connection..."

**Symptoms**: Loading spinner doesn't complete, no results shown.

**Cause**: API timeout or hung request.

**Solution**:
1. Wait 30 seconds to see if it completes
2. Check browser Developer Console (F12) for errors
3. Refresh the entire dashboard page
4. Restart `azd app run`
5. Click **Start Verification** again

### Azure Logs Viewing Issues

### "Azure logs not configured"

**Cause**: Missing workspace information in azd environment, or setup guide not completed.

**Solution**: 
1. Open the **Azure Logs Setup Guide** (click Azure button)
2. Follow all 4 steps to complete setup
3. Ensure your bicep outputs include `AZURE_LOG_ANALYTICS_WORKSPACE_ID` and `AZURE_LOG_ANALYTICS_WORKSPACE_GUID`
4. Run `azd provision` to update environment values
5. Verify with `azd env get-values | grep LOG_ANALYTICS`
6. Alternatively, set `logs.analytics.workspace` explicitly in azure.yaml

### "No Azure logs found"

**Cause**: Diagnostic settings not configured, logs not yet ingested, or time window too narrow.

**Solution**:
1. Open the **Azure Logs Setup Guide** and complete all steps, especially Step 3 (Diagnostic Settings) and Step 4 (Verification)
2. Verify diagnostic settings exist:
   ```bash
   az monitor diagnostic-settings list --resource <resource-id>
   ```
3. Check if logs exist in Log Analytics:
   ```bash
   az monitor log-analytics query \
     -w <workspace-id> \
     --analytics-query "ContainerAppConsoleLogs_CL | take 10"
   ```
4. **Expand time window**: Use the timeframe selector in the dashboard to try `1h`, `6h`, or `24h`
5. Note: Log Analytics has ingestion delay of 1-5 minutes

### "Authentication failed"

**Cause**: Azure credentials not valid or expired.

**Solution**:
1. Open the **Azure Logs Setup Guide** and complete Step 2 (Authentication)
2. Re-authenticate with Azure:
   ```bash
   az login
   azd auth login
   ```
3. Verify subscription access:
   ```bash
   az account show
   ```

### "Workspace not found"

**Cause**: Log Analytics workspace ID is incorrect or inaccessible.

**Solution**:
1. Open the **Azure Logs Setup Guide** and verify Step 1 shows workspace as "Configured"
2. Verify workspace exists:
   ```bash
   az monitor log-analytics workspace show --resource-group <rg> --workspace-name <name>
   ```
3. Check you have Reader access to the workspace
4. Verify `AZURE_LOG_ANALYTICS_WORKSPACE_ID` in `.azure/<env>/.env` matches the actual workspace

### Logs appear in Azure Portal but not in dashboard

**Cause**: Table name mismatch, KQL syntax error, or service name filtering issue.

**Solution**:
1. Complete the **Azure Logs Setup Guide** to verify all configuration
2. Click **View Query** in the dashboard's Azure logs bar to see the KQL being executed
3. Copy the query and test it in Azure Portal's Log Analytics query editor
4. Check if the service name placeholder `{serviceName}` matches your Azure resource name
5. Override with a custom query in azure.yaml if default tables don't match your setup:
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

### General Troubleshooting Tips

#### Setup guide validation stuck

**Symptoms**: A step keeps showing "Checking..." or spinner never stops.

**Cause**: API call to setup state endpoint is failing or timing out.

**Solution**:
1. Check browser Developer Console (F12) for network errors
2. Verify `azd app run` is still running in terminal
3. Refresh the entire dashboard page
4. Restart `azd app run` if API is unresponsive

#### "Permission denied" after role assignment

**Symptoms**: Added required role but setup guide still shows permission denied.

**Cause**: Azure RBAC permissions can take 5-10 minutes to propagate.

**Solution**:
1. Wait 5-10 minutes after assigning the role
2. Click "Recheck" or "Retry" button in the affected step
3. Try logging out and back in:
   ```bash
   az logout
   az login
   ```
4. Verify role assignment in Azure Portal:
   - Navigate to resource → Access control (IAM)
   - Check "Role assignments" tab for your user

### Manual Setup Override

If the setup guide doesn't work for your environment, you can configure manually:

1. **Set workspace ID** in `.azure/<env>/.env`:
   ```bash
   AZURE_LOG_ANALYTICS_WORKSPACE_ID=/subscriptions/.../workspaces/my-workspace
   AZURE_LOG_ANALYTICS_WORKSPACE_GUID=12345678-1234-1234-1234-123456789abc
   ```

2. **Or configure in azure.yaml**:
   ```yaml
   logs:
     analytics:
       workspace: "/subscriptions/.../workspaces/my-workspace"
   ```

3. **Manually create diagnostic settings in Azure Portal**:
   - Navigate to each resource
   - Select "Diagnostic settings"
   - Click "Add diagnostic setting"
   - Select log categories
   - Choose "Send to Log Analytics workspace"
   - Select your workspace
   - Save

4. **Restart dashboard**: `azd app run`

5. **Verify**: Switch to Azure mode - logs should appear if diagnostic settings are correct

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
