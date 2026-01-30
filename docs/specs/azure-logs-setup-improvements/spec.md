# Azure Logs Setup Detection - Technical Specification

**Status**: Draft  
**Priority**: P0 (User blocker)  
**Effort**: Medium (3-5 days)

## Problem Statement

Users deploy Log Analytics workspace via Bicep but setup guide reports "workspace not configured". This creates confusion because the workspace EXISTS in Azure, but the required environment variables aren't populated due to missing Bicep outputs.

**Current user experience**:
1. User writes Bicep with Log Analytics workspace
2. Runs `azd provision` - workspace deploys ✅
3. Opens Azure Logs setup guide - shows "not configured" ❌
4. Docs suggest adding azure.yaml config (shouldn't be needed)
5. User confused: "But I deployed it!"

**Root cause**:
- Bicep creates workspace but doesn't output workspace details
- .env file missing: `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` (critical for queries)
- Detection logic doesn't distinguish "deployed" vs "not configured"
- Docs make azure.yaml seem required when it's optional

## Solution Overview

### 1. Immediate Fix
Add missing Bicep outputs to populate environment variables automatically.

### 2. Better Detection
Enhance setup guide to detect "deployed but not configured" state and provide actionable fix guidance.

### 3. Eliminate azure.yaml Requirement
Make workspace detection fully automatic with azure.yaml only for overrides.

### 4. Improve Auto-Discovery
Make fallback discovery more reliable with caching, better errors, and multi-RG support.

## Technical Design

### Detection State Machine

```
┌─────────────────────────────────────────────────────────────┐
│ Workspace Detection Flow                                    │
└─────────────────────────────────────────────────────────────┘

Check AZURE_LOG_ANALYTICS_WORKSPACE_GUID env var
    │
    ├─ Found ──────────────────────────────► "configured" ✅
    │                                        (Ready to use)
    │
    └─ Not found
         │
         Check AZURE_LOG_ANALYTICS_WORKSPACE_ID env var
         │
         ├─ Found ─────────────────────────► "configured" ✅
         │                                   (Can extract GUID)
         │
         └─ Not found
              │
              Check azure.yaml (logs.analytics.workspace)
              │
              ├─ Found
              │    │
              │    Verify in Azure
              │    │
              │    ├─ Exists ──────────────► "configured" ✅
              │    │                         (Via azure.yaml)
              │    │
              │    └─ Not exists ──────────► "not-deployed" ⚠️
              │                              (Need azd provision)
              │
              └─ Not found
                   │
                   Auto-discover from resource group
                   │
                   ├─ Found ──────────────► "deployed-not-configured" ⚠️
                   │                        (Missing Bicep outputs)
                   │                        ▶ Show fix instructions
                   │
                   ├─ Not found ─────────► "missing" ❌
                   │                        (Not deployed)
                   │
                   └─ Error ──────────────► "error" ❌
                                            (Permission/API issue)
```

### Status Values

```typescript
type WorkspaceStatus = 
  | "configured"                  // ✅ Ready: GUID/ID in env or azure.yaml deployed
  | "deployed-not-configured"     // ⚠️ In Azure but missing Bicep outputs
  | "not-deployed"                // ⚠️ Referenced in azure.yaml but not provisioned
  | "missing"                     // ❌ Not configured anywhere
  | "error"                       // ❌ Check failed (permissions, API, etc.)
```

### User Guidance by Status

#### configured ✅
```
Log Analytics workspace is configured and ready.
Workspace: log-abc123xyz... (first/last chars shown)
Source: environment variable
```

#### deployed-not-configured ⚠️
```
Log Analytics workspace found in Azure, but not configured in environment.

Your Bicep file is missing these outputs:

┌──────────────────────────────────────────────────────────────┐
│ # Add to infra/main.bicep outputs section                    │
│                                                               │
│ output AZURE_LOG_ANALYTICS_WORKSPACE_ID string =             │
│   logAnalytics.outputs.resourceId                            │
│ output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string =           │
│   logAnalytics.outputs.name                                  │
│ output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string =           │
│   logAnalytics.outputs.customerId                            │
└──────────────────────────────────────────────────────────────┘

After adding outputs, run:
  azd provision

Then click "Recheck" below.

[Copy Bicep Code] [Recheck]
```

#### not-deployed ⚠️
```
Workspace configured in azure.yaml but not deployed to Azure.

Run this command to provision resources:
  azd provision

After provisioning, click "Recheck".
```

#### missing ❌
```
Log Analytics workspace not found.

Option 1: Add Bicep outputs (recommended)
  [Show Bicep Example]

Option 2: Configure in azure.yaml  
  [Show azure.yaml Example]

After configuration:
  azd provision
```

#### error ❌
```
Failed to check workspace configuration.

Error: {detailed error message}

Possible causes:
- Missing Azure permissions (need "Reader" role)
- Azure CLI not authenticated
- Network connectivity issue

Fix:
  azd auth login
  
[Retry Check]
```

## Implementation Details

### Backend Changes

#### azure_setup.go - Enhanced Detection

```go
func (s *Server) checkWorkspaceState() WorkspaceState {
    state := WorkspaceState{
        Status:  StatusMissing,
        Message: MsgWorkspaceNotConfigured,
    }

    // Priority 1: Check env vars (GUID preferred, ID fallback)
    if workspaceID := getWorkspaceIDFromEnv(); workspaceID != "" {
        state.Status = StatusConfigured
        state.WorkspaceID = workspaceID
        state.Source = "environment"
        state.Message = fmt.Sprintf("Workspace configured: %s", truncate(workspaceID))
        return state
    }

    // Priority 2: Check azure.yaml
    if azureYaml := loadAzureYaml(); azureYaml.Logs.Analytics.Workspace != "" {
        wsRef := azureYaml.Logs.Analytics.Workspace
        
        // Verify deployment
        deployed := verifyWorkspaceInAzure(ctx, wsRef)
        if deployed {
            state.Status = StatusConfigured
            state.Source = "azure.yaml"
        } else {
            state.Status = StatusNotDeployed
            state.Message = "Workspace in azure.yaml not deployed. Run: azd provision"
        }
        return state
    }

    // Priority 3: Auto-discover (NEW LOGIC)
    discovered, err := attemptAutoDiscovery(ctx)
    if err != nil {
        state.Status = StatusError
        state.Message = fmt.Sprintf("Discovery failed: %v", err)
        return state
    }
    
    if discovered != "" {
        // Found in Azure but not in environment
        state.Status = StatusDeployedNotConfigured  // NEW STATUS
        state.WorkspaceID = discovered
        state.Source = "auto-discovered"
        state.Message = "Workspace deployed but Bicep outputs missing"
        return state
    }

    // Not found anywhere
    state.Status = StatusMissing
    state.Message = "Log Analytics workspace not found"
    return state
}
```

#### discovery.go - Cached Auto-Discovery

```go
// Global cache for auto-discovery results
var discoveryCache = struct {
    sync.RWMutex
    workspaceGUID string
    resourceID    string
    timestamp     time.Time
}{}

const discoveryCacheTTL = 5 * time.Minute

func (d *ResourceDiscovery) detectLogAnalyticsWorkspace(
    ctx context.Context, 
    subscriptionID, 
    resourceGroup string,
) (string, error) {
    // Check cache first
    discoveryCache.RLock()
    if time.Since(discoveryCache.timestamp) < discoveryCacheTTL {
        cached := discoveryCache.workspaceGUID
        discoveryCache.RUnlock()
        if cached != "" {
            return cached, nil
        }
    }
    discoveryCache.RUnlock()

    // Query Azure Resource Manager
    client, err := armresources.NewClient(subscriptionID, d.credential, nil)
    if err != nil {
        return "", &DiscoveryError{Type: "auth", Cause: err}
    }

    pager := client.NewListByResourceGroupPager(resourceGroup, &armresources.ClientListByResourceGroupOptions{
        Filter: to.Ptr("resourceType eq 'Microsoft.OperationalInsights/workspaces'"),
    })

    for pager.More() {
        page, err := pager.NextPage(ctx)
        if err != nil {
            return "", &DiscoveryError{Type: "api", Cause: err}
        }

        for _, resource := range page.Value {
            if resource.Type != nil && 
               strings.EqualFold(*resource.Type, "Microsoft.OperationalInsights/workspaces") {
                
                // Get workspace GUID (customerId property)
                workspaceGUID, err := getWorkspaceCustomerID(ctx, client, *resource.ID)
                if err != nil {
                    slog.Warn("failed to get workspace GUID", "resourceId", *resource.ID, "error", err)
                    continue
                }

                // Update cache
                discoveryCache.Lock()
                discoveryCache.workspaceGUID = workspaceGUID
                discoveryCache.resourceID = *resource.ID
                discoveryCache.timestamp = time.Now()
                discoveryCache.Unlock()

                return workspaceGUID, nil
            }
        }
    }

    return "", &DiscoveryError{Type: "not-found"}
}

// Get workspace GUID (customerId) from resource
func getWorkspaceCustomerID(ctx context.Context, client *armresources.Client, resourceID string) (string, error) {
    // Parse resource ID to extract subscription, RG, name
    parsed := parseResourceID(resourceID)
    
    // Query workspace-specific API for customerId
    wsClient, err := armoperationalinsights.NewWorkspacesClient(parsed.Subscription, client credential, nil)
    if err != nil {
        return "", err
    }
    
    ws, err := wsClient.Get(ctx, parsed.ResourceGroup, parsed.Name, nil)
    if err != nil {
        return "", err
    }
    
    if ws.Properties.CustomerID == nil {
        return "", fmt.Errorf("workspace customerId is nil")
    }
    
    return *ws.Properties.CustomerID, nil
}

// Typed errors for better handling
type DiscoveryError struct {
    Type  string // "auth", "permission", "api", "not-found"
    Cause error
}

func (e *DiscoveryError) Error() string {
    switch e.Type {
    case "auth":
        return "Authentication failed. Run: azd auth login"
    case "permission":
        return "Missing permissions. Need 'Reader' role on resource group"
    case "api":
        return fmt.Sprintf("Azure API error: %v", e.Cause)
    case "not-found":
        return "No Log Analytics workspace found in resource group"
    default:
        return fmt.Sprintf("Discovery error: %v", e.Cause)
    }
}
```

### Frontend Changes

#### WorkspaceSetupStep.tsx - New State Handling

```tsx
const WorkspaceSetupStep: React.FC<WorkspaceSetupStepProps> = ({ onValidationChange }) => {
    const [workspaceState, setWorkspaceState] = useState<WorkspaceState | null>(null);
    const [isChecking, setIsChecking] = useState(false);

    // Fetch workspace state from API
    const checkWorkspace = async () => {
        setIsChecking(true);
        try {
            const resp = await fetch('/api/azure/setup-state');
            const data = await resp.json();
            setWorkspaceState(data.workspace);
            
            // Validate: configured or deployed-not-configured both allow proceeding
            const isValid = ['configured', 'deployed-not-configured'].includes(data.workspace.status);
            onValidationChange(isValid);
        } catch (err) {
            console.error('Failed to check workspace:', err);
            setWorkspaceState({ status: 'error', message: String(err) });
            onValidationChange(false);
        } finally {
            setIsChecking(false);
        }
    };

    useEffect(() => { checkWorkspace(); }, []);

    // Render based on status
    return (
        <div className="workspace-setup-step">
            <h3>Log Analytics Workspace</h3>
            
            {workspaceState?.status === 'configured' && (
                <Alert variant="success">
                    <CheckCircle className="icon" />
                    <div>
                        <strong>Workspace configured</strong>
                        <p>{workspaceState.message}</p>
                    </div>
                </Alert>
            )}

            {workspaceState?.status === 'deployed-not-configured' && (
                <>
                    <Alert variant="warning">
                        <AlertTriangle className="icon" />
                        <div>
                            <strong>Workspace deployed but not configured</strong>
                            <p>Your Bicep file is missing required outputs.</p>
                        </div>
                    </Alert>

                    <CollapsibleSection
                        id="fix-bicep-outputs"
                        title="Fix: Add Bicep Outputs"
                        defaultExpanded={true}
                    >
                        <p>Add these outputs to your <code>infra/main.bicep</code>:</p>
                        
                        <CodeBlock
                            language="bicep"
                            code={BICEP_OUTPUTS_FIX}
                            copyable={true}
                        />

                        <p>Then run:</p>
                        <CodeBlock
                            language="bash"
                            code="azd provision"
                            copyable={true}
                        />

                        <Button onClick={checkWorkspace} variant="secondary">
                            <RefreshCw className={cn("icon", isChecking && "animate-spin")} />
                            Recheck
                        </Button>
                    </CollapsibleSection>
                </>
            )}

            {workspaceState?.status === 'not-deployed' && (
                <Alert variant="warning">
                    <AlertTriangle className="icon" />
                    <div>
                        <strong>Workspace not deployed</strong>
                        <p>Run <code>azd provision</code> to deploy resources.</p>
                    </div>
                </Alert>
            )}

            {workspaceState?.status === 'missing' && (
                <>
                    <Alert variant="error">
                        <Circle className="icon" />
                        <div>
                            <strong>Log Analytics workspace not found</strong>
                            <p>Configure a workspace to collect logs from Azure services.</p>
                        </div>
                    </Alert>

                    <CollapsibleSection id="bicep-example" title="Option 1: Add to Bicep (Recommended)">
                        <CodeBlock language="bicep" code={BICEP_WORKSPACE_EXAMPLE} copyable={true} />
                    </CollapsibleSection>

                    <CollapsibleSection id="yaml-example" title="Option 2: Configure in azure.yaml">
                        <CodeBlock language="yaml" code={AZURE_YAML_EXAMPLE} copyable={true} />
                    </CollapsibleSection>
                </>
            )}

            {workspaceState?.status === 'error' && (
                <Alert variant="error">
                    <AlertTriangle className="icon" />
                    <div>
                        <strong>Failed to check workspace</strong>
                        <p>{workspaceState.message}</p>
                        <Button onClick={checkWorkspace} variant="secondary" className="mt-2">
                            Retry
                        </Button>
                    </div>
                </Alert>
            )}
        </div>
    );
};

// Code snippets
const BICEP_OUTPUTS_FIX = `
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId
`.trim();

const BICEP_WORKSPACE_EXAMPLE = `
// Create Log Analytics workspace
module logAnalytics 'br/public:avm/res/operational-insights/workspace:0.12.0' = {
  name: 'log-analytics'
  params: {
    name: 'log-\${resourceToken}'
    location: location
    skuName: 'PerGB2018'
    dataRetention: 30
  }
}

// Expose outputs
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = logAnalytics.outputs.resourceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = logAnalytics.outputs.name
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.outputs.customerId
`.trim();

const AZURE_YAML_EXAMPLE = `
logs:
  analytics:
    workspace: \${AZURE_LOG_ANALYTICS_WORKSPACE_ID}
`.trim();
```

## Bicep Module Verification

Need to verify AVM module outputs for Log Analytics workspace:

```bicep
// Module: br/public:avm/res/operational-insights/workspace:0.12.0

// Expected outputs (verify in module source):
- resourceId: string          // Full ARM resource ID
- name: string                // Workspace name
- customerId: string          // Workspace GUID (for queries) ⚠️ verify this exists
```

**If `customerId` not available** in AVM outputs, use direct reference:
```bicep
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = logAnalytics.properties.customerId
```

## Testing Plan

### Unit Tests

```go
// azure_setup_test.go
func TestCheckWorkspaceState_DeployedNotConfigured(t *testing.T) {
    // Mock: no env vars, workspace exists in Azure
    // Expect: status = "deployed-not-configured"
}

func TestCheckWorkspaceState_Configured(t *testing.T) {
    // Mock: GUID in env
    // Expect: status = "configured", source = "environment"
}

func TestCheckWorkspaceState_NotDeployed(t *testing.T) {
    // Mock: azure.yaml has workspace, not in Azure
    // Expect: status = "not-deployed"
}

// discovery_test.go
func TestAutoDiscovery_Caching(t *testing.T) {
    // First call: queries Azure
    // Second call within 5min: uses cache
    // After 5min: queries again
}

func TestAutoDiscovery_PermissionDenied(t *testing.T) {
    // Mock ARM API 403
    // Expect: typed error with helpful message
}
```

### E2E Tests

```typescript
// workspace-setup.spec.ts
describe('Workspace Setup Step', () => {
  test('shows deployed-not-configured with fix instructions', async () => {
    // Mock API: workspace in Azure, no env vars
    // Expect: warning alert + Bicep snippet + provision command
  });

  test('recheck button updates state', async () => {
    // Initial: deployed-not-configured
    // User fixes Bicep, clicks recheck
    // Expect: status changes to configured
  });

  test('copy buttons work for code snippets', async () => {
    // Click copy on Bicep snippet
    // Expect: clipboard contains correct output code
  });
});
```

## Rollout Plan

### Phase 1: Immediate (Week 1)
- ✅ Fix temp.bicep (add 3 outputs)
- ✅ Document analysis and tasks
- Add "deployed-not-configured" status to backend
- Update detection logic (checkWorkspaceState)

### Phase 2: UX Improvements (Week 2)
- Update WorkspaceSetupStep component
- Add Bicep snippet UI
- Add "Recheck" button
- Update status messages

### Phase 3: Robustness (Week 3)
- Add discovery caching
- Improve error handling
- Multi-resource-group support
- Telemetry for discovery metrics

### Phase 4: Documentation (Week 4)
- Update all docs to clarify azure.yaml optional
- Add troubleshooting guide
- Update examples
- Create migration guide for existing projects

## Success Metrics

**Quantitative**:
- Reduce "workspace not found" support tickets by 80%
- Auto-discovery cache hit rate >90%
- Setup completion rate increase from 60% → 95%

**Qualitative**:
- Users understand "deployed vs configured" distinction
- No confusion about azure.yaml requirement
- Clear actionable error messages

## Related Documents

- [analysis.md](./analysis.md) - Root cause investigation
- [tasks.md](./tasks.md) - Implementation tasks and checklist
- [../azure-logs-diagnostics-ui-spec.md](../azure-logs-diagnostics-ui-spec.md) - Original diagnostics UI spec
