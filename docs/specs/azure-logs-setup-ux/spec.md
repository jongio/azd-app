# Azure Logs Setup UX Improvement

## Problem Statement

The current Diagnostic Settings setup experience is problematic:

1. **Per-service configuration is tedious**: Users must configure diagnostic settings individually for each service (appService, containerApp, function), even though they all need the same Log Analytics workspace connection.

2. **Poor verification**: The current UI shows "Not Configured" status but doesn't provide a clear way to verify that the Log Analytics workspace is actually configured correctly and receiving logs.

3. **Manual Bicep copying**: Users must manually copy Bicep code for each service type, which is error-prone and repetitive.

## Goals

1. **Single workspace configuration**: Configure the Log Analytics workspace once at the project level, not per-service
2. **Automated diagnostic settings**: Auto-generate or verify diagnostic settings for all detected services in one action
3. **Better verification**: Provide a "Test Connection" feature that actually queries the workspace to verify logs are flowing
4. **Clearer guidance**: Show exactly what's configured and what's missing with actionable steps

## Design Principles

- **Zero-config default**: If resources are already configured (via azd provision with proper Bicep), the setup guide should detect this and skip unnecessary steps
- **Batch operations**: Configure all services at once, not one-by-one
- **Verification over configuration**: Focus on verifying the setup works rather than making users configure manually
- **Progressive disclosure**: Show simple status by default, advanced details on demand

## Proposed UX Flow

### Step 1: Workspace Detection (Existing)
- Auto-detect workspace from `azd env get-values` (existing behavior)
- Manual entry if not found
- ✅ Keep as-is

### Step 2: Authentication (Existing)
- Verify Azure credentials
- Check permissions (Reader + Log Analytics Reader)
- ✅ Keep as-is

### Step 3: Diagnostic Settings (NEW)

Instead of showing individual service cards with "Not Configured", show:

#### Option A: Automatic Detection & Verification
```
┌─────────────────────────────────────────────────────┐
│ Diagnostic Settings                                 │
│                                                     │
│ Checking diagnostic settings for your services...  │
│                                                     │
│ ● appService          ✓ Configured                │
│ ● containerApp        ✓ Configured                │
│ ● function            ✓ Configured                │
│                                                     │
│ All services are configured! ✓                     │
│                                                     │
│ [Test Connection →]                                │
└─────────────────────────────────────────────────────┘
```

If not configured:
```
┌─────────────────────────────────────────────────────┐
│ Diagnostic Settings                                 │
│                                                     │
│ ⚠ 3 services need diagnostic settings enabled      │
│                                                     │
│ ○ appService          Not Configured               │
│ ○ containerApp        Not Configured               │
│ ○ function            Not Configured               │
│                                                     │
│ How to fix:                                        │
│ 1. Update your main.bicep with diagnostic settings│
│ 2. Run `azd up` to deploy the changes             │
│                                                     │
│ [Show Bicep Template] [Skip]                       │
└─────────────────────────────────────────────────────┘
```

"Show Bicep Template" opens a modal with:
- Single unified Bicep template for ALL services
- Automatically detects service types in project
- Shows where to add it in their infra structure
- Copy button for entire template

#### Option B: Quick Fix Action
```
┌─────────────────────────────────────────────────────┐
│ Diagnostic Settings                                 │
│                                                     │
│ ⚠ 3 services need diagnostic settings enabled      │
│                                                     │
│ We can configure these automatically by:           │
│ 1. Generating diagnostic settings Bicep templates  │
│ 2. Adding them to your infrastructure              │
│ 3. Running `azd up` to deploy                      │
│                                                     │
│ [Auto-Configure All Services →]  [Manual Setup]    │
└─────────────────────────────────────────────────────┘
```

### Step 4: Verification (ENHANCED)

Current: Shows generic "Setup Verification" with unclear validation

New: Actually test the connection
```
┌─────────────────────────────────────────────────────┐
│ Verification                                        │
│                                                     │
│ Testing connection to Log Analytics workspace...   │
│                                                     │
│ ✓ Workspace accessible                            │
│ ✓ Authentication valid                            │
│ ✓ Querying recent logs...                         │
│                                                     │
│ Found logs from:                                   │
│   ● appService (2 entries in last 15m)            │
│   ● containerApp (5 entries in last 15m)          │
│   ● function (0 entries)                          │
│                                                     │
│ ⚠ No recent logs from function                    │
│   This is normal if the function hasn't run yet.  │
│                                                     │
│ [View Logs →]  [Complete Setup]                   │
└─────────────────────────────────────────────────────┘
```

If verification fails:
```
┌─────────────────────────────────────────────────────┐
│ Verification                                        │
│                                                     │
│ ✗ Could not retrieve logs                         │
│                                                     │
│ Error: No diagnostic settings configured for       │
│ these services. Logs cannot flow to workspace.     │
│                                                     │
│ Fix:                                               │
│ 1. Go back to Diagnostic Settings step            │
│ 2. Configure diagnostic settings                  │
│ 3. Wait 2-5 minutes for logs to appear            │
│ 4. Return here to verify                          │
│                                                     │
│ [← Back to Diagnostic Settings]  [Retry]          │
└─────────────────────────────────────────────────────┘
```

## Implementation Requirements

### Backend Changes (Go)

1. **Add diagnostic settings detection API**
   - `GET /api/azure/diagnostic-settings/check`
   - Returns status for each service: configured/not-configured/error
   - Uses Azure Resource Manager API to check diagnostic settings

2. **Add workspace verification API**
   - `POST /api/azure/workspace/verify`
   - Actually queries workspace for recent logs
   - Returns log counts per service
   - Identifies common issues (no logs, permission errors)

3. **Batch Bicep template generation**
   - Generate consolidated Bicep for all services
   - Include only detected service types
   - Provide integration instructions

### Frontend Changes (React)

1. **DiagnosticSettingsStep.tsx**
   - Call check API on mount
   - Show aggregated status (not per-service cards)
   - Single "Show Template" modal for all services
   - Clear error states with recovery actions

2. **SetupVerification.tsx**
   - Add actual verification logic (call verify API)
   - Show log counts and service status
   - Provide retry mechanism
   - Link back to previous steps on failure

3. **New Component: BicepTemplateModal.tsx**
   - Syntax-highlighted Bicep
   - Copy all button
   - Integration instructions
   - Service-specific sections (collapsible)

## Success Criteria

1. **Faster setup**: Users can set up diagnostic settings in 1 action vs 3
2. **Better validation**: Verification step actually tests log flow
3. **Less confusion**: Clear status shows what's working and what's not
4. **Recovery paths**: Every error state has a clear next action

## Out of Scope

- Automatic infrastructure modification (still requires manual Bicep update + deploy)
- Support for non-Bicep infrastructure (Terraform, ARM templates)
- Log streaming during verification (show counts only)

## Future Enhancements

- Auto-commit Bicep changes to repo (with user confirmation)
- One-click `azd up` from setup wizard
- Live log streaming in verification step
- Export setup checklist for team onboarding
