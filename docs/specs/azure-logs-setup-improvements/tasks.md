# Azure Logs Setup Detection - Improvement Tasks

<!-- NEXT: 1 -->

## Summary

Improve Azure Logs setup detection to eliminate confusion when Log Analytics workspace is deployed but environment variables aren't configured. Remove requirement for azure.yaml updates by making auto-detection smarter and providing better guidance.

## TODO

### 1. Fix temp.bicep - Add Missing Outputs

**Priority**: P0 (Immediate user blocker)

Add missing Log Analytics outputs to expose workspace details to environment.

**Changes to temp.bicep**:
```bicep
// After line 298 (after Container App outputs), add:

// ============================================
// Log Analytics Workspace
// ============================================

output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId
```

**Verification**:
- Check AVM module outputs: `br/public:avm/res/operational-insights/workspace:0.12.0`
- Ensure `.outputs.resourceId`, `.outputs.name`, `.outputs.customerId` are available
- If AVM doesn't expose `customerId`, may need to reference `logAnalytics::properties.customerId` directly

**Why GUID is critical**:
- `.customerId` is the workspace GUID used for Log Analytics queries
- Different from resource ID (which is ARM path)
- Required for `az monitor log-analytics query` and SDK queries

**After fix**:
- Run `azd provision` → outputs populate .env
- Setup guide detects workspace automatically
- No azure.yaml changes needed

---

### 2. Improve Setup Detection - Differentiate States

**Priority**: P0 (Core UX improvement)

Update `checkWorkspaceState()` to distinguish "deployed but not configured" from "not deployed".

**Current behavior** (azure_setup.go:109-148):
```go
func (s *Server) checkWorkspaceState() WorkspaceState {
    // Only checks env vars and azure.yaml
    // Doesn't detect deployed-but-not-outputted state
}
```

**New behavior**:
```go
func (s *Server) checkWorkspaceState() WorkspaceState {
    // 1. Check env vars (GUID, ID) - if present, return "configured" ✅
    
    // 2. Check azure.yaml - if present, return "not-deployed" or "configured"
    
    // 3. NEW: Attempt auto-discovery from resource group
    //    - If workspace found: return "deployed-not-configured" with specific fix
    //    - If not found: return "not-deployed"
}
```

**New status**: `deployed-not-configured`
```go
const (
    StatusConfigured            = "configured"            // Env vars set, ready to use
    StatusDeployedNotConfigured = "deployed-not-configured" // NEW: In Azure but missing outputs
    StatusMissing               = "missing"                // Not in env/yaml
    StatusNotDeployed           = "not-deployed"          // Referenced but not provisioned
    StatusError                 = "error"                 // Check failed
)
```

**Message for deployed-not-configured**:
```
"Log Analytics workspace found in Azure, but Bicep outputs are missing.

Add these outputs to your main.bicep:

output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId

Then run: azd provision"
```

**Files to update**:
- `cli/src/internal/dashboard/azure_setup.go` - Add detection logic
- `cli/dashboard/src/components/WorkspaceSetupStep.tsx` - Handle new status
- `cli/src/internal/azure/discovery.go` - Expose workspace detection as reusable function

---

### 3. Add Bicep Quick Fix to Setup Guide

**Priority**: P1 (Improve self-service)

Add a collapsible "Fix Bicep Outputs" section in the setup guide when workspace is detected but not configured.

**UI component** (WorkspaceSetupStep.tsx):
```tsx
{workspaceState.status === 'deployed-not-configured' && (
  <CollapsibleSection
    id="fix-bicep"
    title="Fix Missing Bicep Outputs"
    defaultExpanded={true}
  >
    <p>Your Log Analytics workspace is deployed, but Bicep outputs are missing.</p>
    
    <h4>Step 1: Add outputs to main.bicep</h4>
    <CodeBlock 
      language="bicep"
      code={BICEP_OUTPUTS_TEMPLATE}
      copyable={true}
    />
    
    <h4>Step 2: Re-provision</h4>
    <CodeBlock
      language="bash"
      code="azd provision"
      copyable={true}
    />
    
    <Alert>
      After provisioning, click "Recheck" to verify setup.
    </Alert>
  </CollapsibleSection>
)}
```

**Template constant**:
```typescript
const BICEP_OUTPUTS_TEMPLATE = `
// Add to your infra/main.bicep outputs section

output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name  
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId
`.trim();
```

**Files to update**:
- `cli/dashboard/src/components/WorkspaceSetupStep.tsx`
- `cli/dashboard/src/components/shared/CodeBlock.tsx` (may need Bicep syntax highlighting)

---

### 4. Remove azure.yaml Requirement from Docs

**Priority**: P1 (Reduce confusion)

Update documentation to make azure.yaml configuration truly optional and only for overrides.

**Current docs** (features/azure-logs.md) say:
```yaml
logs:
  analytics:
    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}  # Suggests this is required
```

**New docs**:
```yaml
# Optional: Only needed for custom configurations
logs:
  analytics:
    # Override auto-detection (only if needed)
    workspace: ${CUSTOM_WORKSPACE_VAR}
    
    # Polling settings
    pollingInterval: 30s
    defaultTimespan: 1h
```

**Add prominent note**:
```markdown
> **Note**: The `workspace` field is OPTIONAL and auto-detected from:
> 1. `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` environment variable (recommended)
> 2. `AZURE_LOG_ANALYTICS_WORKSPACE_ID` environment variable
> 3. Auto-discovery from your resource group
>
> Only specify `workspace` if:
> - Using a custom environment variable name
> - Connecting to a workspace in a different resource group
> - Need to override auto-detection
```

**Files to update**:
- `cli/docs/features/azure-logs.md` - Main feature docs
- `web/src/pages/reference/azure-yaml.astro` - Schema reference
- `cli/docs/commands/logs.md` - Command reference
- `cli/tests/projects/integration/azure-logs-test/README.md` - Example project

**Update setup guide copy** (WorkspaceSetupStep.tsx):
- Remove suggestion to add azure.yaml config
- Focus on Bicep outputs as the primary solution
- Show azure.yaml only as "Advanced Override" option

---

### 5. Improve Auto-Discovery Reliability

**Priority**: P2 (Robustness)

Make auto-discovery more robust when falling back from missing env vars.

**Current issues**:
- ARM API call on every check (no caching)
- Single resource group only (no cross-subscription support)
- Silent failure if permissions missing
- No retry on transient errors

**Improvements**:

**5a. Cache auto-discovery results**
```go
// discovery.go - Add caching layer
type workspaceDiscoveryCache struct {
    workspaceID  string
    timestamp    time.Time
    mu           sync.RWMutex
}

var autoDiscoveryCache = &workspaceDiscoveryCache{}

func (d *ResourceDiscovery) detectLogAnalyticsWorkspace(...) string {
    // Check cache first (5 min TTL)
    autoDiscoveryCache.mu.RLock()
    if time.Since(autoDiscoveryCache.timestamp) < 5*time.Minute {
        cached := autoDiscoveryCache.workspaceID
        autoDiscoveryCache.mu.RUnlock()
        return cached
    }
    autoDiscoveryCache.mu.RUnlock()
    
    // Perform discovery...
    // Update cache...
}
```

**5b. Better error reporting**
```go
// Return typed errors instead of ""
func (d *ResourceDiscovery) detectLogAnalyticsWorkspace(...) (string, error) {
    // ...
    if permissionErr {
        return "", &PermissionError{Scope: "ARM Reader"}
    }
    if notFoundErr {
        return "", &NotFoundError{ResourceType: "Log Analytics workspace"}
    }
}
```

**5c. Support multiple resource groups**
```go
// Search in tagged resource groups if primary not found
func (d *ResourceDiscovery) detectLogAnalyticsWorkspaceMultiRG(ctx context.Context) (string, error) {
    // 1. Check primary resource group (AZURE_RESOURCE_GROUP)
    
    // 2. Search all RGs with tag "azd-env-name" matching current environment
    
    // 3. Return first workspace found
}
```

**Files to update**:
- `cli/src/internal/azure/discovery.go` - Core discovery logic
- `cli/src/internal/azure/standalone_logs.go` - Standalone discovery functions
- `cli/src/internal/dashboard/azure_setup.go` - Setup state checks

---

### 6. Add "Auto-Fix" Button to Setup Guide

**Priority**: P2 (Nice-to-have UX)

When workspace is deployed-not-configured, offer one-click fix that:
1. Detects Bicep file structure
2. Adds missing outputs automatically
3. Prompts user to review changes
4. Offers to run `azd provision`

**UI Flow**:
```tsx
<Button onClick={handleAutoFix} variant="primary">
  Auto-Fix Bicep Outputs
</Button>

// Modal shows:
// 1. Detected Bicep file: infra/main.bicep
// 2. Preview of changes (diff view)
// 3. Confirm button
// 4. After confirm: edit file, show success, prompt to provision
```

**Backend API**:
```
POST /api/azure/setup/auto-fix
{
  "action": "add-bicep-outputs"
}

Response:
{
  "bicepFile": "infra/main.bicep",
  "changes": "...",
  "applied": true
}
```

**Implementation**:
- Detect main.bicep location (scan infra/, or from azure.yaml)
- Parse Bicep AST to find outputs section
- Insert missing outputs with proper formatting
- Preserve comments and spacing

**Files to create/update**:
- `cli/src/internal/dashboard/azure_setup_autofix.go` - Auto-fix logic
- `cli/dashboard/src/components/WorkspaceSetupStep.tsx` - UI
- Consider using Bicep MCP server for safe AST manipulation

**Risks**:
- Bicep parsing can be complex (consider using Bicep Language Server)
- File editing can break formatting
- Need good error handling if file structure unexpected

**Alternative**: Skip parsing, just append to file with clear comment markers

---

### 7. Update Test Coverage

**Priority**: P2

Add tests for deployed-not-configured state.

**Test scenarios**:
1. Workspace deployed, no env vars → auto-discovery finds it → show deployed-not-configured
2. Workspace deployed, GUID set → show configured ✅
3. Workspace not deployed → show not-deployed
4. azure.yaml has workspace, not deployed → show not-deployed
5. Auto-discovery fails (permissions) → graceful degradation with helpful message

**Files to create/update**:
- `cli/src/internal/dashboard/azure_setup_test.go` - Unit tests
- `cli/src/internal/azure/discovery_test.go` - Discovery tests
- `cli/tests/e2e/azure_logs_setup_test.go` - E2E tests (if exists)

---

## Done

### Task 1: Fix temp.bicep - Add Missing Outputs ✅
Added `AZURE_LOG_ANALYTICS_WORKSPACE_ID`, `NAME`, and `GUID` outputs to temp.bicep.

### Task 2: Improve Setup Detection - Differentiate States ✅
Enhanced `checkWorkspaceState()` with `deployed-not-configured` status and auto-discovery fallback.

### Task 3: Add Bicep Quick Fix to Setup Guide ✅
WorkspaceSetupStep.tsx displays bicepFix code block with copy functionality.

### Task 4: Remove azure.yaml Requirement from Docs ✅
Documentation already clarifies azure.yaml workspace config is optional.

### Task 5: Improve Auto-Discovery Reliability ✅
- Added `DiscoveryError` typed errors (auth, permission, not-found, timeout)
- Added `DetectLogAnalyticsWorkspaceMultiRG()` for cross-RG search
- Cache already existed with 5 min TTL

### Task 6: Add "Auto-Fix" Button to Setup Guide ✅
- Backend: `POST /api/azure/setup/auto-fix` endpoint in `azure_setup_autofix.go`
- Frontend: Auto-fix button with success/error feedback
- Detects Log Analytics module and adds missing outputs

### Task 7: Update Test Coverage ✅
- `TestDiscoveryError` - typed error messages
- `TestIsDiscoveryNotFound` - error classification
- `TestHandleAzureSetupAutoFix` - auto-fix endpoint (6 scenarios)
- `TestDetectLogAnalyticsModuleName` - Bicep parsing
- `TestInsertBicepOutputs` - Bicep modification

---

## Acceptance Criteria

### For Task 1 (Fix temp.bicep)
- [x] temp.bicep has 3 new outputs (ID, NAME, GUID)
- [x] `azd provision` populates .env with workspace details
- [x] Setup guide shows "configured" status
- [x] No azure.yaml changes required

### For Task 2 (Improve Detection)
- [x] New status "deployed-not-configured" recognized
- [x] Setup guide shows specific message with fix instructions
- [x] Auto-discovery runs as fallback (with caching)
- [x] Error states handled gracefully

### For Task 3 (Quick Fix UI)
- [x] Collapsible section shows in deployed-not-configured state
- [x] Bicep code snippet is copy-pasteable
- [x] "Recheck" button verifies after fix
- [x] Mobile-responsive layout

### For Task 4 (Docs Update)
- [x] All docs clarify azure.yaml is optional
- [x] Examples show auto-detection as default
- [x] Override scenarios documented clearly
- [x] Confusion points addressed (FAQs)

### For Task 5 (Auto-Discovery)
- [x] Results cached (5 min TTL)
- [x] Permission errors reported clearly
- [x] Multi-resource-group search supported
- [ ] Telemetry for discovery success/failure rates (deferred)

### For Task 6 (Auto-Fix)
- [x] Button appears in correct state
- [x] Preview shows exact changes (manual section)
- [x] File editing preserves formatting
- [x] Error handling for edge cases
- [ ] Telemetry for auto-fix usage (deferred)

### For Task 7 (Tests)
- [x] All scenarios covered
- [x] Mocked Azure APIs
- [x] Fast execution (<5s total)
- [ ] CI integration

---

## Implementation Order

**Phase 1 - Immediate Fix** (P0 - unblock user):
1. Task 1 - Fix temp.bicep outputs

**Phase 2 - Better Detection** (P0 - core UX):
2. Task 2 - Deployed-not-configured status
3. Task 3 - Quick fix UI component

**Phase 3 - Reduce Confusion** (P1 - docs):
4. Task 4 - Update docs to make azure.yaml optional

**Phase 4 - Robustness** (P2 - polish):
5. Task 5 - Improve auto-discovery
6. Task 6 - Auto-fix button
7. Task 7 - Test coverage

---

## Success Metrics

**Before**:
- ❌ User confused: workspace deployed but setup guide says "missing"
- ❌ Docs suggest azure.yaml config is required
- ❌ No specific guidance for "deployed but not outputted" case
- ❌ Auto-discovery not cached (slow)

**After**:
- ✅ Setup guide detects all states correctly
- ✅ Specific fix instructions with copy-paste code
- ✅ azure.yaml truly optional (only for overrides)
- ✅ Auto-discovery cached and reliable
- ✅ Optional: One-click auto-fix for common case

**User experience**:
- From: "Why isn't it working? I deployed everything!"
- To: "Oh, I just need to add these 3 outputs. Done!"

---

## Related Files

**Analysis**: [analysis.md](./analysis.md) - Root cause details  
**Spec**: [spec.md](./spec.md) - Technical specification (to be created)
**PR**: (Link to PR once created)
