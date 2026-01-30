# Azure Logs Setup Detection Improvements - Summary

**Date**: 2026-01-27 (Updated: 2026-01-30)  
**Status**: Implementation Complete ✅ (All Tasks)

## Problem Solved

Users deploying Log Analytics workspaces via Bicep were confused when the Azure Logs setup guide reported "workspace not configured" even though the workspace was successfully deployed in Azure. This happened because Bicep outputs were missing, so environment variables weren't populated.

## Your Project - Manual Fix Required

### Add These Outputs to Your Bicep File

In your project's `infra/main.bicep` file, add these three outputs to the outputs section:

```bicep
// Log Analytics Workspace (required for Azure Logs)
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId
```

**Important**: Replace `logAnalytics` with the actual name of your Log Analytics module if it's different.

### Then Re-provision

```bash
azd provision
```

This will populate your `.env` file with the workspace details, and the setup guide will automatically detect them.

### Result

Your `.env` file will contain:
```env
AZURE_LOG_ANALYTICS_WORKSPACE_ID="/subscriptions/.../Microsoft.OperationalInsights/workspaces/..."
AZURE_LOG_ANALYTICS_WORKSPACE_NAME="log-cae-devexreview-..."
AZURE_LOG_ANALYTICS_WORKSPACE_GUID="abc123-..."
```

The setup guide will then show: ✅ **Workspace configured** (Source: environment)

## azd-app Improvements - Implemented

### 1. Enhanced Detection (Backend)

**Files Modified**:
- [`cli/src/internal/dashboard/constants.go`](c:\code\azd-app\cli\src\internal\dashboard\constants.go)
- [`cli/src/internal/dashboard/azure_setup.go`](c:\code\azd-app\cli\src\internal\dashboard\azure_setup.go)

**Changes**:
- Added new status: `deployed-not-configured` to distinguish "deployed in Azure but missing Bicep outputs" from "not deployed"
- Enhanced `checkWorkspaceState()` to auto-discover workspaces from Azure resource group as fallback
- Added `generateBicepOutputsFix()` to provide copy-pasteable Bicep code
- Updated error messages to be more specific and actionable

**Detection Flow**:
```
Priority 1: Check AZURE_LOG_ANALYTICS_WORKSPACE_GUID env var
    ↓ Not found
Priority 2: Check AZURE_LOG_ANALYTICS_WORKSPACE_ID env var  
    ↓ Not found
Priority 3: Check azure.yaml logs.analytics.workspace
    ↓ Not found
Priority 4: Auto-discover from Azure (NEW!)
    ↓ Found
Status: "deployed-not-configured" with Bicep fix guidance
```

### 2. Improved UI Guidance (Frontend)

**Files Modified**:
- [`cli/dashboard/src/components/WorkspaceSetupStep.tsx`](c:\code\azd-app\cli\dashboard\src\components\WorkspaceSetupStep.tsx)

**Changes**:
- Added handling for `deployed-not-configured` status
- Display amber warning alert when workspace is deployed but outputs missing
- Show Bicep fix code in a code block for easy copy-paste
- Allow user to proceed to next step (not a hard blocker)

**What Users See**:
```
⚠️ Workspace deployed but Bicep outputs missing

Your Log Analytics workspace exists in Azure, but your Bicep file doesn't 
expose the workspace details to the environment.

[Bicep code block with outputs]

Note: Replace logAnalytics with your workspace module name if different.
```

### 3. Specific Issue Reporting

Updated `collectSetupIssues()` to provide three different guidance messages:

| Status | Severity | Message | Fix |
|--------|----------|---------|-----|
| `missing` | error | "Log Analytics workspace not configured" | "Add Log Analytics workspace to your Bicep infrastructure with required outputs" |
| `deployed-not-configured` | **warning** | "**Log Analytics workspace found in Azure, but Bicep outputs are missing**" | **[Bicep code snippet]** |
| `not-deployed` | warning | "Workspace configured but not deployed" | "azd up" |

## Key Benefits

### Before
- ❌ User confused: "I deployed a workspace, why does it say 'not configured'?"
- ❌ No distinction between "not deployed" and "deployed but missing outputs"
- ❌ Generic error message with no actionable fix
- ❌ Docs suggested azure.yaml config as required (shouldn't be)

### After
- ✅ Clear detection: "Workspace deployed but Bicep outputs missing"
- ✅ Auto-discovery finds deployed workspace even when env vars missing
- ✅ Copy-paste Bicep fix provided directly in UI
- ✅ Specific guidance for each scenario
- ✅ azure.yaml remains optional (only needed for overrides)

## Technical Implementation

### Auto-Discovery Logic

When environment variables are missing, the setup guide now:

1. Creates Azure credential
2. Uses existing `ResourceDiscovery` service to scan resource group
3. Finds Log Analytics workspace (if deployed)
4. Returns `deployed-not-configured` status with workspace ID
5. Provides Bicep outputs fix

**Performance**: Auto-discovery has 15-second timeout to keep UI responsive.

### Backward Compatibility

All changes are backward compatible:
- Existing detection paths still work (env vars, azure.yaml)
- New status added to existing status enum
- Frontend gracefully handles old API responses (won't have bicepFix field)

## Testing Recommendations

### Manual Test Scenarios

1. **Workspace deployed, outputs missing** (your scenario)
   - Deploy workspace via Bicep without outputs
   - Run `azd provision`
   - Open setup guide → should show "deployed-not-configured" with Bicep fix

2. **Workspace deployed, outputs present**
   - Add missing outputs
   - Run `azd provision`
   - Open setup guide → should show "configured" ✅

3. **Workspace not deployed**
   - Remove workspace from Bicep
   - Run `azd provision`
   - Open setup guide → should show "missing"

4. **azure.yaml override**
   - Add `logs.analytics.workspace` to azure.yaml
   - Should show "not-deployed" until provisioned

### Automated Tests (Recommended)

Add unit tests in `azure_setup_test.go`:

```go
func TestCheckWorkspaceState_DeployedNotConfigured(t *testing.T) {
    // Mock: no env vars, workspace exists in Azure
    // Expect: status = "deployed-not-configured", bicepFix populated
}

func TestCheckWorkspaceState_AutoDiscoveryFails(t *testing.T) {
    // Mock: no env vars, discovery API error
    // Expect: status = "missing" (fallback gracefully)
}
```

## Documentation Updates Needed

To complete this improvement, update these docs to clarify that azure.yaml is optional:

1. **cli/docs/features/azure-logs.md**
   - Emphasize Bicep outputs as primary method
   - Show azure.yaml only as "Advanced Override" option

2. **web/src/pages/reference/azure-yaml.astro**
   - Update schema docs to say `logs.analytics.workspace` is optional
   - Note auto-detection from environment variables

3. **Example projects**
   - Ensure all examples have proper Bicep outputs
   - Remove azure.yaml config from examples (to show it's optional)

## All Features Now Implemented ✅

All P2 "polish" features from tasks.md have been implemented:

- **Auto-fix button**: ✅ `POST /api/azure/setup/auto-fix` endpoint + UI button
- **Cached auto-discovery**: ✅ Already existed with 5 min TTL
- **Multi-resource-group support**: ✅ `DetectLogAnalyticsWorkspaceMultiRG()` function
- **Better error handling**: ✅ `DiscoveryError` typed errors (auth, permission, not-found, timeout)

Only deferred items are telemetry (can be added later as needed).

## Files Changed

### Modified
1. [`cli/src/internal/dashboard/constants.go`](c:\code\azd-app\cli\src\internal\dashboard\constants.go) - Added status constant
2. [`cli/src/internal/dashboard/azure_setup.go`](c:\code\azd-app\cli\src\internal\dashboard\azure_setup.go) - Enhanced detection, added auto-discovery
3. [`cli/dashboard/src/components/WorkspaceSetupStep.tsx`](c:\code\azd-app\cli\dashboard\src\components\WorkspaceSetupStep.tsx) - Added UI for new status + auto-fix button
4. [`cli/src/internal/azure/discovery.go`](c:\code\azd-app\cli\src\internal\azure\discovery.go) - Typed errors, multi-RG support
5. [`cli/src/internal/dashboard/server_routes.go`](c:\code\azd-app\cli\src\internal\dashboard\server_routes.go) - Added auto-fix route
6. [`temp.bicep`](c:\code\azd-app\temp.bicep) - Added Log Analytics outputs

### New Files
7. [`cli/src/internal/dashboard/azure_setup_autofix.go`](c:\code\azd-app\cli\src\internal\dashboard\azure_setup_autofix.go) - Auto-fix backend handler
8. [`cli/src/internal/dashboard/azure_setup_autofix_test.go`](c:\code\azd-app\cli\src\internal\dashboard\azure_setup_autofix_test.go) - Auto-fix tests

### Test Files Updated
9. [`cli/src/internal/dashboard/azure_setup_test.go`](c:\code\azd-app\cli\src\internal\dashboard\azure_setup_test.go) - Tests for deployed-not-configured
10. [`cli/src/internal/azure/discovery_test.go`](c:\code\azd-app\cli\src\internal\azure\discovery_test.go) - Tests for typed errors

### Documentation Created
11. [`docs/specs/azure-logs-setup-improvements/analysis.md`](c:\code\azd-app\docs\specs\azure-logs-setup-improvements\analysis.md) - Root cause analysis
12. [`docs/specs/azure-logs-setup-improvements/tasks.md`](c:\code\azd-app\docs\specs\azure-logs-setup-improvements\tasks.md) - Full task breakdown
13. [`docs/specs/azure-logs-setup-improvements/spec.md`](c:\code\azd-app\docs\specs\azure-logs-setup-improvements\spec.md) - Technical specification
14. [`docs/specs/azure-logs-setup-improvements/SUMMARY.md`](c:\code\azd-app\docs\specs\azure-logs-setup-improvements\SUMMARY.md) - This file

## Next Steps

### For Your Project (Immediate)
1. Add the three Bicep outputs to your `infra/main.bicep`
2. Run `azd provision`
3. Verify `.env` file has workspace variables
4. Setup guide should now show configured ✅

### For azd-app (Before Merge)
1. Test the changes manually with a project missing outputs
2. Add unit tests for new detection logic
3. Update documentation (features/azure-logs.md)
4. Create PR with test results

### For azd-app (Future)
1. Consider implementing auto-fix button (Phase 4 from tasks.md)
2. Add telemetry to track how often "deployed-not-configured" occurs
3. Cache auto-discovery results for performance
4. Extend to support cross-subscription scenarios

---

## Summary

**Problem**: Workspace deployed but setup guide says "not configured"  
**Root Cause**: Missing Bicep outputs → environment variables not populated  
**Solution**: Auto-detect deployed workspace + provide specific Bicep fix guidance  
**Impact**: Better UX for users, fewer support tickets, clearer error messages  
**Status**: ✅ Core implementation complete, ready for testing
