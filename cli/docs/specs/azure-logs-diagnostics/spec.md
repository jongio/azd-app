# Azure Logs Diagnostics System - Specification

## Overview
Zero-touch Azure log streaming with comprehensive diagnostics to help users troubleshoot configuration issues per host type.

## Design Principles
- **Zero configuration**: Logs stream automatically when properly configured in Azure
- **Manual diagnostics**: User clicks "Diagnostics" button to troubleshoot
- **Host-specific guidance**: Each Azure host type has specific requirements and setup guides
- **No auto-refresh**: User manually refreshes to avoid excessive API calls

## UI Components

### 1. Diagnostic Button Placement
```
[Local] [Azure] [🔍 Diagnostics]
```
- Located immediately after Azure tab button
- Always visible when Azure tab is active
- Opens full-screen modal

### 2. Diagnostic Modal Structure
```
┌─────────────────────────────────────────────┐
│  Azure Logs Diagnostics            [✕]      │
│  Last checked: 2 min ago    [🔄 Refresh]   │
├─────────────────────────────────────────────┤
│                                             │
│  ✅ containerapp-api (Container Apps)       │
│     Streaming • 1,234 logs in 15min        │
│     [Details →]                             │
│                                             │
│  ❌ functions-worker (Azure Functions)      │
│     Not configured                          │
│     [View Setup Guide →]                    │
│                                             │
│  ⚠️  appservice-web (App Service)           │
│     Partial - Missing diagnostic settings   │
│     [View Setup Guide →]                    │
│                                             │
└─────────────────────────────────────────────┘
```

### 3. No Logs Prompt (In Log Pane)
When a service has no logs, instead of "No logs available":
```
⚠️  No logs detected for functions-worker

This could mean:
• Service hasn't generated logs yet
• Logging not configured
• Logs taking time to propagate (5-10 min)

[View Diagnostic Details →]
```

## Backend Architecture

### API Endpoints

#### GET /api/azure/diagnostics
Returns comprehensive diagnostic status for all services.

**Response:**
```json
{
  "workspaceId": "/subscriptions/.../workspaces/...",
  "workspaceName": "workspace-xyz",
  "lastChecked": "2024-12-29T10:23:45Z",
  "services": {
    "containerapp-api": {
      "hostType": "containerApp",
      "status": "healthy",
      "logCount": 1234,
      "lastLogTime": "2024-12-29T10:23:45Z",
      "requirements": [
        {
          "name": "Log Analytics Workspace",
          "status": "met",
          "description": "Connected to workspace-xyz"
        },
        {
          "name": "Diagnostic Settings",
          "status": "met",
          "description": "Streaming ContainerAppConsoleLogs"
        }
      ],
      "setupGuide": null
    },
    "functions-worker": {
      "hostType": "function",
      "status": "not-configured",
      "logCount": 0,
      "requirements": [
        {
          "name": "Application Insights",
          "status": "not-met",
          "description": "APPLICATIONINSIGHTS_CONNECTION_STRING not set"
        }
      ],
      "setupGuide": {
        "title": "Azure Functions - Enable Log Streaming",
        "steps": [...]
      }
    }
  }
}
```

### Validation Logic Per Host Type

#### Container Apps
**Requirements:**
1. Log Analytics workspace exists
2. Diagnostic settings configured with:
   - ContainerAppConsoleLogs
   - ContainerAppSystemLogs
   - Destination: Log Analytics workspace

**Validation:**
- Query: `GET /providers/Microsoft.App/containerApps/{name}/providers/Microsoft.Insights/diagnosticSettings`
- Check logs: Query ContainerAppConsoleLogs table (last 15 min)

**Status:**
- ✅ healthy: Logs flowing
- ⚠️ partial: Diagnostic settings exist but no logs
- ❌ not-configured: No diagnostic settings

#### Azure Functions
**Requirements:**
1. Application Insights connection string in environment
2. (Optional) Diagnostic settings for FunctionAppLogs

**Validation:**
- Check env: `APPLICATIONINSIGHTS_CONNECTION_STRING` in azd env
- Query diagnostic settings API
- Check logs: Query FunctionAppLogs table

**Status:**
- ✅ healthy: Logs flowing
- ⚠️ partial: App Insights configured but no logs
- ❌ not-configured: No App Insights connection

#### App Service
**Requirements:**
1. Diagnostic settings configured with:
   - AppServiceConsoleLogs
   - AppServiceHTTPLogs
   - Destination: Log Analytics workspace

**Validation:**
- Query: `GET /providers/Microsoft.Web/sites/{name}/providers/Microsoft.Insights/diagnosticSettings`
- Check logs: Query AppServiceConsoleLogs table

**Status:**
- ✅ healthy: Logs flowing
- ⚠️ partial: Diagnostic settings exist but no logs
- ❌ not-configured: No diagnostic settings

#### AKS (Future)
**Requirements:**
1. Container Insights enabled
2. Log Analytics workspace integration

**Validation:**
- Check addon: Container monitoring enabled
- Check logs: Query ContainerLogV2 table

## Setup Guides

### Container Apps Setup Guide
```markdown
# Container Apps - Enable Log Streaming

## Automatic Setup (Recommended)
Your Container App needs diagnostic settings to stream logs.

Run this command:
```bash
azd up
```

This will configure diagnostic settings automatically.

## Manual Setup
1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to your Container App: `{containerAppName}`
3. Click "Diagnostic settings" in the left menu
4. Click "+ Add diagnostic setting"
5. Name: `azd-logs`
6. Select logs:
   - ✅ ContainerAppConsoleLogs
   - ✅ ContainerAppSystemLogs
7. Check "Send to Log Analytics workspace"
8. Select workspace: `{workspaceName}`
9. Click "Save"

Logs should appear within 5-10 minutes.
```

### Functions Setup Guide
```markdown
# Azure Functions - Enable Log Streaming

## Step 1: Configure Application Insights

Add to your `azure.yaml`:

```yaml
services:
  {serviceName}:
    environment:
      APPLICATIONINSIGHTS_CONNECTION_STRING: ${APPLICATIONINSIGHTS_CONNECTION_STRING}
```

## Step 2: Deploy
```bash
azd deploy {serviceName}
```

## Step 3: Verify
Logs should appear in 5-10 minutes.

## Troubleshooting
- Ensure function is processing requests
- Check Application Insights in Azure Portal
- Verify workspace connection
```

### App Service Setup Guide
```markdown
# App Service - Enable Log Streaming

## Automatic Setup
```bash
azd up
```

## Manual Setup
1. Azure Portal → Your App Service: `{appServiceName}`
2. "Diagnostic settings" → "+ Add diagnostic setting"
3. Name: `azd-logs`
4. Select:
   - ✅ AppServiceConsoleLogs
   - ✅ AppServiceHTTPLogs
5. Send to Log Analytics workspace: `{workspaceName}`
6. Save

Logs appear in 5-10 minutes.
```

## Implementation Phases

### Phase 1: Infrastructure
- [ ] Create diagnostic API endpoint
- [ ] Define data structures
- [ ] Implement base validator interface

### Phase 2: Container Apps Validator
- [ ] Diagnostic settings check
- [ ] Log query validation
- [ ] Setup guide generation

### Phase 3: Functions Validator
- [ ] App Insights connection check
- [ ] Environment variable validation
- [ ] Setup guide generation

### Phase 4: App Service Validator
- [ ] Diagnostic settings check
- [ ] Log query validation
- [ ] Setup guide generation

### Phase 5: Frontend
- [ ] Diagnostic modal component
- [ ] Service health cards
- [ ] Setup guide renderer
- [ ] No logs prompt component

### Phase 6: Integration
- [ ] Wire up diagnostic button
- [ ] Connect to API
- [ ] Testing with real Azure resources

### Phase 7: AKS & Others
- [ ] AKS validator
- [ ] Other host types as needed
