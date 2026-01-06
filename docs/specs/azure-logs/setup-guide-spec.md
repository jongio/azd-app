# Azure Logs Setup Guide - Specification

## Overview

Provide an integrated, step-by-step setup guide within the azd-app dashboard to help users configure Azure log streaming from scratch. The guide detects the current setup state, identifies missing prerequisites, and provides actionable guidance with examples and commands.

## Problem Statement

**Current State**: When users click the Azure logs button but Azure isn't configured:
- The button is clickable but nothing happens
- Users see a tooltip saying "Azure logging not configured. Click to diagnose and fix setup"
- No clear path forward to actually configure Azure logs
- Existing DiagnosticsModal only shows health checks, not setup guidance

**User Pain Points**:
1. Don't know what's missing (workspace? credentials? diagnostic settings?)
2. Don't know where to configure azure.yaml
3. Don't know what bicep outputs are required
4. Don't understand the difference between workspace ID vs GUID
5. Have to piece together docs instead of following a wizard

## Goals

1. **Progressive disclosure**: Show what's missing without overwhelming users
2. **Actionable steps**: Each issue has clear fix with copy/paste commands or config
3. **Integrated experience**: No need to leave dashboard to understand setup
4. **Context-aware**: Detect what's already configured vs missing
5. **Educational**: Teach users the setup model while guiding them

## Non-Goals

1. Auto-provision Azure resources (users should use bicep/terraform)
2. Replace full documentation (guide links to docs for deep dives)
3. Support non-azd workflows (assumes azd project structure)

## User Personas

### Persona 1: New to Azure Logs
- **Goal**: Just wants logs to work
- **Knowledge**: Knows `azd up`, not familiar with Log Analytics
- **Needs**: Step-by-step wizard, example code snippets, validation

### Persona 2: Experienced but Stuck
- **Goal**: Fix specific configuration issue
- **Knowledge**: Has Log Analytics, missing one piece (e.g., diagnostic settings)
- **Needs**: Quick diagnostic, link to specific step, copy commands

### Persona 3: Advanced User
- **Goal**: Customize beyond defaults
- **Knowledge**: Understands KQL, wants custom queries
- **Needs**: Configuration reference, skip to advanced options

## Setup Flow

### Entry Points

1. **Click Azure mode button when not configured**
   - Opens setup guide directly
   - Shows "Complete Setup" as primary action
   
2. **Run Diagnostics button in error states**
   - Opens DiagnosticsModal first
   - "Fix Setup" button opens setup guide with context
   
3. **Settings dialog**
   - "Azure Logs Setup" section
   - Opens guide to review/modify configuration

### Setup State Detection

Query `/api/azure/logs/health` to determine setup state:

```typescript
interface SetupState {
  step: 'workspace' | 'credentials' | 'diagnostic-settings' | 'complete'
  issues: SetupIssue[]
  resources: {
    discovered: number
    configured: number
    services: ServiceSetupStatus[]
  }
}

interface SetupIssue {
  severity: 'error' | 'warning' | 'info'
  category: 'workspace' | 'auth' | 'diagnostic-settings' | 'config'
  message: string
  fix: string // Command or config snippet
  docsUrl?: string
}

interface ServiceSetupStatus {
  name: string
  deployed: boolean
  diagnosticSettings: boolean
  logsFlowing: boolean
}
```

### Wizard Steps

#### Step 1: Log Analytics Workspace

**Detection Logic** (in priority order):
1. Check for `AZURE_LOG_ANALYTICS_WORKSPACE_GUID` environment variable (preferred - set during `azd provision`)
2. Check for `AZURE_LOG_ANALYTICS_WORKSPACE_ID` environment variable (resource ID - extracted from bicep outputs)
3. Auto-detect workspace in resource group using Azure Resource Manager API

**Required Actions**:
1. Create Log Analytics workspace (if needed)
2. Add bicep outputs to infra (recommended but auto-detected if missing)
3. Optionally configure azure.yaml if using non-default env var names

**UI Components**:
- Status badge: ✓ Configured | ⚠ Missing | ○ Not deployed
- Collapsible sections:
  - **What is Log Analytics?** - Brief explanation
  - **Create Workspace** - Azure CLI/Portal instructions
  - **Bicep Example** - Copy/paste monitoring module
  - **azure.yaml Config (Optional)** - YAML snippet with workspace reference

**Example Bicep Output** (recommended):
```bicep
output AZURE_LOG_ANALYTICS_WORKSPACE_ID string = monitoring.outputs.logAnalyticsWorkspaceId
output AZURE_LOG_ANALYTICS_WORKSPACE_NAME string = monitoring.outputs.logAnalyticsWorkspaceName
output AZURE_LOG_ANALYTICS_WORKSPACE_GUID string = monitoring.outputs.logAnalyticsWorkspaceGuid
```

**Auto-Detection** (default behavior):
The workspace is automatically discovered if you have:
- A Log Analytics workspace in your resource group
- `AZURE_SUBSCRIPTION_ID` and `AZURE_RESOURCE_GROUP_NAME` environment variables (automatically set by `azd provision`)

**Example azure.yaml Config (optional - only needed if using custom env var names)**:
```yaml
logs:
  analytics:
    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}  # Optional - auto-detected if not specified
```

**Note**: The `workspace` field in azure.yaml is **optional**. It's only needed if:
- You're using a custom environment variable name instead of the default `AZURE_LOG_ANALYTICS_WORKSPACE_ID` or `AZURE_LOG_ANALYTICS_WORKSPACE_GUID`
- You want to explicitly specify a workspace when multiple workspaces exist in the resource group

#### Step 2: Authentication

**Detection Logic**:
- Check if `azd auth login` has been run
- Test credential scope for Log Analytics API
- Verify permissions on workspace

**Required Actions**:
1. Run `azd auth login` if not authenticated
2. Verify account has `Log Analytics Reader` role

**UI Components**:
- Login status indicator
- "Sign In" button (runs `azd auth login` via API)
- Permission check results
- Role assignment instructions (link to portal)

**Example Permission Check**:
```
✓ Authenticated as: user@example.com
✓ Subscription access: Contributor
✗ Log Analytics Reader: Missing
```

**Fix Command**:
```bash
az role assignment create \
  --assignee user@example.com \
  --role "Log Analytics Reader" \
  --scope /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{workspace}
```

#### Step 3: Diagnostic Settings

**Detection Logic**:
- Discover deployed Azure resources via `azd env get-values`
- Check each resource for diagnostic settings
- Verify diagnostic settings point to correct workspace

**Required Actions** (per service):
1. Enable diagnostic settings
2. Configure log categories
3. Link to Log Analytics workspace

**UI Components**:
- Service-by-service status table:
  - Service name
  - Resource type
  - Diagnostic settings (✓/✗)
  - Quick fix button per service
- Bulk "Enable All" button
- Per-service bicep examples

**Example Diagnostic Settings (Container App)**:
```bicep
resource diagnosticSettings 'Microsoft.Insights/diagnosticSettings@2021-05-01-preview' = {
  name: 'logs-to-analytics'
  scope: containerApp
  properties: {
    workspaceId: logAnalyticsWorkspace.id
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

#### Step 4: Verification

**Detection Logic**:
- Query Log Analytics for recent logs from each service
- Show timestamp of most recent log entry
- Test KQL query execution

**Verification Steps**:
1. Workspace connected ✓
2. Authenticated ✓
3. Diagnostic settings configured ✓
4. Logs flowing ✓

**UI Components**:
- Success state: "All set! Click 'View Logs' to start streaming"
- Per-service verification:
  - Last log received: 2 minutes ago ✓
  - No logs yet (may take 5-15 minutes after deployment) ⏱
- "View Sample Logs" button to see actual data
- "Advanced Configuration" link (custom queries, polling interval)

### Advanced Configuration (Optional)

Collapsible section for experienced users:

**Polling Configuration**:
```yaml
logs:
  analytics:
    workspace: ${AZURE_LOG_ANALYTICS_WORKSPACE_ID}  # Optional - auto-detected if omitted
    pollingInterval: 10s  # How often to query (default: 10s)
    defaultTimespan: 30m  # Lookback window (default: 30m)
```

**Custom KQL Queries**:
```yaml
services:
  api:
    logs:
      analytics:
        query: |
          ContainerAppConsoleLogs_CL
          | where ContainerName_s == "api"
          | where Log_s contains "ERROR"
          | project TimeGenerated, Log_s
```

## Component Architecture

### New Components

#### `AzureSetupGuide.tsx`
Primary wizard component with step navigation.

**Props**:
```typescript
interface AzureSetupGuideProps {
  isOpen: boolean
  onClose: () => void
  onComplete?: () => void
  initialStep?: SetupStep
}
```

**Features**:
- Stepper UI (1 → 2 → 3 → 4)
- Collapsible help sections
- Code copy buttons
- In-guide diagnostic tests
- Progress persistence (localStorage)

#### `WorkspaceSetupStep.tsx`
Workspace configuration step.

#### `AuthSetupStep.tsx`
Authentication verification step.

#### `DiagnosticSettingsStep.tsx`
Service-by-service diagnostic settings.

#### `SetupVerification.tsx`
Final verification and testing step.

### Modified Components

#### `ModeToggle.tsx`
- Click Azure button when disabled → Open setup guide
- Show setup progress indicator (e.g., "2/4 steps complete")

#### `DiagnosticsModal.tsx`
- Add "Fix Setup" button when health checks fail
- Pass failure context to setup guide (deep link to specific step)

#### `AzureErrorDisplay.tsx`
- "Setup Guide" button in error states
- Link to relevant setup step based on error type

## API Additions

### `GET /api/azure/logs/setup-state`

Returns current setup state and issues:

```json
{
  "step": "diagnostic-settings",
  "overallStatus": "incomplete",
  "workspace": {
    "configured": true,
    "exists": true,
    "workspaceId": "/subscriptions/.../workspaces/...",
    "workspaceName": "my-workspace",
    "workspaceGuid": "abc-123-..."
  },
  "authentication": {
    "authenticated": true,
    "principal": "user@example.com",
    "hasPermissions": true,
    "scopes": ["https://api.loganalytics.io/.default"]
  },
  "services": [
    {
      "name": "api",
      "deployed": true,
      "resourceType": "Microsoft.App/containerApps",
      "resourceId": "/subscriptions/.../containerApps/api",
      "diagnosticSettings": {
        "configured": false,
        "workspaceLinked": false,
        "categories": []
      },
      "logsFlowing": false,
      "lastLogTimestamp": null
    }
  ],
  "issues": [
    {
      "severity": "error",
      "category": "diagnostic-settings",
      "service": "api",
      "message": "Diagnostic settings not configured for service 'api'",
      "fix": "See bicep example in setup guide",
      "docsUrl": "https://learn.microsoft.com/azure/azure-monitor/..."
    }
  ],
  "nextSteps": [
    "Configure diagnostic settings for 'api' service",
    "Run 'azd up' to apply infrastructure changes"
  ]
}
```

### `POST /api/azure/logs/verify`

Test connectivity and log flow:

```json
Request:
{
  "service": "api"
}

Response:
{
  "success": true,
  "logsFound": 142,
  "timeRange": {
    "start": "2025-01-01T10:00:00Z",
    "end": "2025-01-01T10:30:00Z"
  },
  "sample": [
    {
      "timestamp": "2025-01-01T10:29:45Z",
      "message": "GET /api/health 200"
    }
  ]
}
```

## UX Flow Examples

### Scenario 1: First-Time Setup

1. User clicks Azure mode button
2. Setup guide opens at Step 1 (Workspace)
3. Shows "Workspace not configured" status
4. Expands bicep example section
5. User copies bicep, adds to infra/main.bicep
6. User runs `azd up`
7. Guide auto-detects workspace, marks Step 1 complete
8. Advances to Step 2 (Authentication)
9. Shows "Sign in required"
10. User clicks "Sign In" button
11. Guide detects successful auth, marks Step 2 complete
12. Advances to Step 3 (Diagnostic Settings)
13. Shows table: `api` service missing diagnostic settings
14. User clicks "Show Bicep Example" for api
15. User copies diagnostic settings, adds to api.bicep
16. User runs `azd up`
17. Guide marks Step 3 complete
18. Advances to Step 4 (Verification)
19. Shows "Waiting for logs..." (5-15 min delay normal)
20. After logs arrive, shows "✓ All set!"
21. User clicks "View Logs" → Guide closes, Azure mode activates

### Scenario 2: Missing Diagnostic Settings

1. User has workspace + auth configured
2. User clicks Azure mode button
3. Logs don't appear, error shows
4. User clicks "Run Diagnostics"
5. DiagnosticsModal shows "Diagnostic settings not found"
6. User clicks "Fix Setup"
7. Setup guide opens directly to Step 3 (Diagnostic Settings)
8. Shows which services need configuration
9. User follows bicep examples, deploys
10. Verification passes
11. Logs start flowing

### Scenario 3: Permission Issue

1. User has everything configured
2. User clicks Azure mode button
3. Error: "Permission denied"
4. User clicks "Run Diagnostics"
5. DiagnosticsModal shows "Missing Log Analytics Reader role"
6. User clicks "Fix Setup"
7. Setup guide opens to Step 2 (Authentication)
8. Shows exact `az role assignment create` command
9. User copies and runs command
10. Clicks "Retest" in guide
11. Verification passes
12. User returns to dashboard, logs work

## Design Principles

### 1. Progressive Complexity
- Start simple: "You need a workspace"
- Expand on click: Full bicep examples, KQL customization
- Expert mode: Link to advanced docs

### 2. Validation Before Next Step
- Can't advance to Step 2 until Step 1 configured
- Real-time detection (check every 5s when guide open)
- Manual "Recheck" button

### 3. Copy/Paste First
- Every fix has a code snippet
- One-click copy buttons
- Syntax highlighting

### 4. Visual Progress
- Stepper shows 1→2→3→4 with checkmarks
- Overall progress bar
- Per-step status indicators

### 5. Context Preservation
- Remembers last step (localStorage)
- Deep linking to specific step from errors
- Prefills known values (subscription ID, resource group)

## Accessibility

- **WCAG AA compliance**: Keyboard navigation through steps
- **Screen reader support**: Status announcements, step transitions
- **High contrast**: Status colors meet contrast ratios
- **Focus management**: Auto-focus on step load, proper tab order
- **Error announcements**: Live region for validation failures

## Performance

- **Lazy load steps**: Only render current step component
- **Debounced validation**: Don't spam API during setup
- **Cached state**: Store setup progress locally
- **Optimistic UI**: Show "checking..." immediately

## Security

- **No credential display**: Never show tokens/keys in guide
- **Sanitized output**: Mask sensitive data in examples
- **Link validation**: Only official Microsoft Learn docs
- **CSP compliance**: Inline code snippets properly escaped

## Testing Strategy

### Unit Tests
- Each step component in isolation
- Setup state parsing logic
- Validation functions

### Integration Tests
- Full wizard flow with mocked API
- Deep linking from error states
- Progress persistence across sessions

### E2E Tests
- Complete setup workflow (with test infra)
- Error recovery paths
- Multiple service configurations

### Accessibility Tests
- Keyboard-only navigation
- Screen reader announcements
- Color contrast validation

## Documentation

### User Documentation
- `cli/docs/features/azure-logs-setup.md` - Setup guide walkthrough
- Add setup guide section to main Azure logs docs
- Add troubleshooting section referencing guide

### Developer Documentation
- Component API reference
- Setup state API specification
- Extension points for custom resource types

## Metrics

Track setup guide effectiveness:
- **Completion rate**: % of users who complete all 4 steps
- **Drop-off points**: Which step users abandon
- **Time to complete**: Median time from open to verification
- **Error frequency**: Most common setup issues
- **Help section usage**: Which expandable sections clicked most

## Future Enhancements

### Phase 2
- **One-click bicep generation**: Generate complete monitoring module
- **Azure portal integration**: Link directly to resource in portal
- **Template library**: Pre-built configs for common scenarios

### Phase 3
- **Automated diagnostic settings**: Enable via ARM API (requires elevated permissions)
- **Log sampling**: Show live log preview during setup
- **Multi-workspace support**: Configure different workspaces per service

## Success Criteria

1. **Discoverability**: 90% of users find setup guide when Azure not configured
2. **Completion**: 70% of users who start guide complete all steps
3. **Time to value**: Median time from "not configured" to "logs flowing" < 15 minutes (excluding deployment time)
4. **Support reduction**: 50% reduction in "Azure logs not working" issues
5. **User satisfaction**: Positive feedback on setup experience

## Acceptance Criteria

- [ ] Setup guide opens when clicking Azure mode button (not configured)
- [ ] All 4 setup steps implemented with validation
- [ ] Workspace step detects configuration from azure.yaml and env
- [ ] Auth step tests Log Analytics API access
- [ ] Diagnostic settings step shows per-service status
- [ ] Verification step queries actual logs
- [ ] Code snippets copyable with one click
- [ ] Deep linking from error states works
- [ ] Progress persists across page reloads
- [ ] WCAG AA accessible (keyboard nav, screen readers)
- [ ] Unit tests >=80% coverage on new components
- [ ] E2E test covering full happy path
- [ ] Documentation updated with setup guide reference

---

**References**:
- [Azure Logs Spec](spec.md)
- [Azure Logs Tasks](tasks.md)
- [Azure Error Display Component](../../cli/dashboard/src/components/AzureErrorDisplay.tsx)
- [Diagnostics Modal Component](../../cli/dashboard/src/components/DiagnosticsModal.tsx)
