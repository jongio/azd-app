# Azure Logs Setup Detection - Root Cause Analysis

**Date**: 2026-01-27  
**Issue**: Log Analytics workspace deployed but not detected by setup guide

## Root Cause

### 1. Missing Bicep Outputs

**Current situation in temp.bicep**:
```bicep
// Line 268: Log Analytics workspace IS deployed
module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {
  name: 'log-analytics-deployment'
  params: {
    name: 'log-cae-devexreview-${environmentName}'
    location: location
    // ... workspace deployed successfully
  }
}

// BUT: No outputs at bottom of file exposing workspace details!
// Missing:
// - AZURE_LOG_ANALYTICS_WORKSPACE_ID (resource ID)
// - AZURE_LOG_ANALYTICS_WORKSPACE_NAME (workspace name)
// - AZURE_LOG_ANALYTICS_WORKSPACE_GUID (customerId - CRITICAL for queries)
```

**What azd-app test projects do correctly** (azure-logs-test/infra/main.bicep:118-121):
```bicep
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = monitoring.outputs.logAnalyticsWorkspaceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = monitoring.outputs.logAnalyticsWorkspaceName
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = monitoring.outputs.logAnalyticsWorkspaceGuid
```

### 2. Detection Logic Flow

**Current priority order** (discovery.go:151-164):

1. **AZURE_LOG_ANALYTICS_WORKSPACE_GUID** env var ✅ (preferred - used for queries)
2. **AZURE_LOG_ANALYTICS_WORKSPACE_ID** env var ✅ (resource ID fallback)
3. **Auto-detection** from resource group ⚠️ (last resort, requires ARM access)

**Why it fails**:
- temp.bicep doesn't output workspace details → .env doesn't contain GUID/ID
- Auto-detection DOES work but shouldn't be required
- Setup UI shows "missing" because env vars aren't populated

### 3. User Experience Impact

**What user sees**:
```
Log Analytics workspace not configured in azure.yaml or environment
```

**What's actually happening**:
- ✅ Workspace IS deployed in Azure
- ✅ Auto-detection CAN find it (if credentials work)
- ❌ Environment variables are missing (Bicep outputs not defined)
- ❌ Setup guide doesn't distinguish "deployed but not outputted" vs "not deployed"

## Evidence from .env File

**Current .env** (user's environment):
```bash
# Application Insights - present ✅
AZURE_APPINSIGHTS_CONNECTION_STRING="..."
AZURE_APPINSIGHTS_INSTRUMENTATION_KEY="..."

# Log Analytics - MISSING ❌
# AZURE_LOG_ANALYTICS_WORKSPACE_ID=<should be here>
# AZURE_LOG_ANALYTICS_WORKSPACE_NAME=<should be here>  
# AZURE_LOG_ANALYTICS_WORKSPACE_GUID=<should be here>
```

## Comparison with Working Project

### azd-app/cli/tests/projects/integration/azure-logs-test

**Bicep structure**:
```bicep
// infra/modules/monitoring.bicep - Shared module
resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2025-07-01' = { ... }

// Outputs
output logAnalyticsWorkspaceId string = logAnalyticsWorkspace.id
output logAnalyticsWorkspaceName string = logAnalyticsWorkspace.name
output logAnalyticsWorkspaceGuid string = logAnalyticsWorkspace.properties.customerId
```

**Main bicep**:
```bicep
module monitoring './modules/monitoring.bicep' = { ... }

// Expose to environment
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = monitoring.outputs.logAnalyticsWorkspaceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = monitoring.outputs.logAnalyticsWorkspaceName
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = monitoring.outputs.logAnalyticsWorkspaceGuid
```

**Result**: Environment variables populated automatically after `azd provision`

## Why azure.yaml Updates Shouldn't Be Required

**Current docs suggest** (screenshot shows):
```yaml
logs:
  analytics:
    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}
```

**This is wrong because**:
1. If Bicep outputs are correct, workspace is auto-detected from environment
2. azure.yaml should only be needed for OVERRIDES (custom workspace, different env var name)
3. Making it "required" duplicates configuration and creates confusion
4. Detection code ALREADY supports auto-discovery as fallback

**Correct default behavior**:
- Bicep outputs → .env file → auto-detected (no azure.yaml needed)
- Only use azure.yaml if you want to override or customize

## Technical Details

### AVM Module Used (temp.bicep:268)
```bicep
module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {
  // Deploys workspace successfully
  // Has outputs available: .outputs.resourceId, .outputs.name, .outputs.customerId (probably)
}
```

**Missing connection**: These AVM outputs exist but aren't exposed in main.bicep outputs

### Discovery Code (discovery.go:151-164)
```go
// Priority chain works correctly
if wsGUID := envValues["AZURE_LOG_ANALYTICS_WORKSPACE_GUID"]; wsGUID != "" {
    result.LogAnalyticsWorkspaceID = wsGUID  // ✅ Best option
} else if wsID := envValues["AZURE_LOG_ANALYTICS_WORKSPACE_ID"]; wsID != "" {
    result.LogAnalyticsWorkspaceID = wsID    // ✅ Fallback option  
} else if result.SubscriptionID != "" && result.ResourceGroup != "" {
    workspaceID := d.detectLogAnalyticsWorkspace(ctx, result.SubscriptionID, result.ResourceGroup)
    result.LogAnalyticsWorkspaceID = workspaceID  // ⚠️ Last resort (works but shouldn't be needed)
}
```

**The auto-detection IS there** but relying on it means:
- Extra ARM API call (slower)
- Requires ARM Reader permissions
- Can fail if permissions are restricted
- Doesn't scale well (multiple resource groups, cross-subscription)

## Recommendations

See [tasks.md](./tasks.md) for implementation plan addressing:

1. **Immediate fix**: Add missing Bicep outputs to temp.bicep
2. **Better detection**: Improve setup guide to detect "deployed but not configured" state
3. **Better guidance**: Specific error messages with copy-paste fixes
4. **Remove azure.yaml confusion**: Make it truly optional, document ONLY for overrides
5. **Improve auto-discovery reliability**: Better error handling, caching, cross-subscription support
