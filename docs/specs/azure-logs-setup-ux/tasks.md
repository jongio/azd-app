<!-- NEXT: -->
# Azure Logs Setup UX Improvement Tasks

## Done

### Backend - Diagnostic Settings Detection API

**Assigned**: Developer

Implement API endpoint to check diagnostic settings status for all services in one call.

**Requirements**:
- Endpoint: `GET /api/azure/diagnostic-settings/check`
- Detect services from `azd env get-values` (existing pattern)
- For each service, query Azure Resource Manager API for diagnostic settings
- Return aggregated status: `{serviceName: "configured" | "not-configured" | "error"}`
- Include error details for troubleshooting
- Handle permissions errors gracefully (log as warning, return "error" status)

**Response Format**:
```json
{
  "workspaceId": "/subscriptions/.../loganalytics-workspace",
  "services": {
    "appService": {
      "status": "configured",
      "resourceId": "/subscriptions/.../app-service",
      "diagnosticSettingName": "toLogAnalytics"
    },
    "containerApp": {
      "status": "not-configured",
      "resourceId": "/subscriptions/.../container-app",
      "error": null
    },
    "function": {
      "status": "error",
      "resourceId": "/subscriptions/.../function-app",
      "error": "InsufficientPermissions: User does not have access..."
    }
  }
}
```

**Acceptance Criteria**:
- Returns status for all detected services in single call
- Gracefully handles missing/undeployed services (skip, don't error)
- Includes workspace ID for context
- Error messages are actionable
- Unit tests with mocked Azure ARM API

**Files**:
- `cli/src/internal/azure/diagnostics.go` (new)
- `cli/src/internal/server/handlers_azure.go` (add endpoint)

---

### Backend - Workspace Verification API

**Assigned**: Developer

Implement API endpoint to verify workspace connection by querying for recent logs.

**Requirements**:
- Endpoint: `POST /api/azure/workspace/verify`
- Query workspace for logs from last 15 minutes per service
- Return log counts and sample timestamps
- Identify common issues:
  - No diagnostic settings configured
  - Logs not yet ingested (normal delay)
  - Permission errors
  - Invalid queries
- Provide specific guidance per error type

**Request**:
```json
{
  "services": ["appService", "containerApp", "function"],
  "timespan": "PT15M"
}
```

**Response Format**:
```json
{
  "status": "success" | "partial" | "error",
  "workspace": {
    "id": "/subscriptions/.../workspace",
    "name": "my-workspace"
  },
  "results": {
    "appService": {
      "logCount": 15,
      "lastLogTime": "2025-12-25T10:45:00Z",
      "status": "ok"
    },
    "containerApp": {
      "logCount": 0,
      "lastLogTime": null,
      "status": "no-logs",
      "message": "No logs found. This may be normal if the service hasn't run yet or if diagnostic settings were just configured (allow 2-5 minutes for ingestion)."
    },
    "function": {
      "logCount": 0,
      "status": "error",
      "error": "DiagnosticSettingsNotConfigured: No diagnostic settings found for this resource."
    }
  },
  "guidance": [
    "appService: Logs flowing correctly",
    "containerApp: No recent logs - wait or trigger activity",
    "function: Configure diagnostic settings first"
  ]
}
```

**Acceptance Criteria**:
- Actually queries Log Analytics workspace
- Returns meaningful log counts and timestamps
- Detects diagnostic settings issues
- Provides specific guidance per service
- Handles query errors gracefully
- Unit tests with mocked workspace queries

**Files**:
- `cli/src/internal/azure/verification.go` (new)
- `cli/src/internal/server/handlers_azure.go` (add endpoint)

---

### Backend - Batch Bicep Template Generator

**Assigned**: Developer

Generate consolidated Bicep template for all detected services.

**Requirements**:
- Endpoint: `GET /api/azure/bicep-template`
- Detect service types from environment
- Generate Bicep module with:
  - Diagnostic settings for each service type
  - Log Analytics workspace reference (parameter)
  - All logs categories enabled
  - Retention configured (30 days default)
- Include integration instructions (where to add module, parameters)
- Template should be valid Bicep (syntax-checked)

**Response Format**:
```json
{
  "template": "// Bicep template content...",
  "services": ["appService", "containerApp", "function"],
  "instructions": {
    "summary": "Add this module to your main.bicep",
    "steps": [
      "1. Save this template as infra/modules/diagnostic-settings.bicep",
      "2. Add workspace parameter to main.bicep if not present",
      "3. Reference module in main.bicep for each service",
      "4. Run `azd up` to deploy"
    ]
  },
  "parameters": [
    {
      "name": "logAnalyticsWorkspaceId",
      "description": "Resource ID of Log Analytics workspace",
      "example": "/subscriptions/.../Microsoft.OperationalInsights/workspaces/my-workspace"
    }
  ]
}
```

**Acceptance Criteria**:
- Generated Bicep is syntactically valid
- Includes all detected service types
- Integration instructions are clear
- Template works with standard azd Bicep structure
- Unit tests verify template generation

**Files**:
- `cli/src/internal/azure/bicep.go` (new)
- `cli/src/internal/server/handlers_azure.go` (add endpoint)

---

### Frontend - Aggregated Diagnostic Settings UI

**Assigned**: Designer → Developer

Replace per-service diagnostic settings cards with aggregated status view.

**Requirements**:
- Call `/api/azure/diagnostic-settings/check` on mount
- Show loading state while checking
- Display aggregated status:
  - All configured ✓
  - Partial (N of M configured) ⚠
  - None configured ✗
- List services with status icons (not expandable cards)
- Single "Show Bicep Template" button (not per-service)
- Clear error state if check fails
- Retry button on errors

**UI States**:
1. **Loading**: "Checking diagnostic settings..."
2. **All Configured**: Green checkmark, "All services configured", [Test Connection] button
3. **Partially Configured**: Yellow warning, "2 of 3 services configured", list with icons, [Show Template] button
4. **None Configured**: Orange warning, "3 services need configuration", [Show Template] button
5. **Error**: Red error, "Could not check diagnostic settings", error message, [Retry] button

**Acceptance Criteria**:
- Single API call checks all services
- Status updates automatically
- No per-service expand/collapse
- Template modal shows unified Bicep
- Error states have recovery actions
- Loading states are clear
- Component tests cover all states

**Files**:
- `cli/dashboard/src/components/DiagnosticSettingsStep.tsx` (modify)
- `cli/dashboard/src/hooks/useDiagnosticSettings.ts` (new)

---

### Frontend - Bicep Template Modal

**Assigned**: Developer

Create modal to display unified Bicep template for all services.

**Requirements**:
- Call `/api/azure/bicep-template` when opened
- Syntax-highlighted Bicep code display
- "Copy All" button with toast confirmation
- Integration instructions (expandable sections)
- Download as file option
- Responsive layout
- Keyboard accessible (Esc to close)

**Modal Sections**:
1. **Header**: "Diagnostic Settings Template", close button
2. **Instructions**: Step-by-step integration guide (collapsible)
3. **Template**: Syntax-highlighted Bicep with copy button
4. **Footer**: [Download] [Copy All] [Close]

**Acceptance Criteria**:
- Template loads from API
- Syntax highlighting works (use existing code highlighter)
- Copy button copies entire template
- Download creates .bicep file
- Instructions are clear and actionable
- Accessible (WCAG AA)
- Component tests

**Files**:
- `cli/dashboard/src/components/BicepTemplateModal.tsx` (new)
- `cli/dashboard/src/hooks/useBicepTemplate.ts` (new)

---

### Frontend - Enhanced Verification Step

**Assigned**: Developer

Replace placeholder verification with actual workspace log query verification.

**Requirements**:
- Call `/api/azure/workspace/verify` when step loads
- Show progress: "Testing connection...", "Querying logs..."
- Display results per service:
  - Service name
  - Log count (last 15m)
  - Status icon (✓ logs found, ⚠ no logs, ✗ error)
  - Contextual message
- Summary at top: "2 of 3 services verified"
- Retry button if verification fails
- "Back to Diagnostic Settings" link if no services configured
- "View Logs" button (navigates to dashboard logs view with Azure mode)
- "Complete Setup" button (enabled if at least 1 service has logs)

**UI Flow**:
1. **Loading**: Progress indicator, "Verifying workspace connection..."
2. **Success (all services)**: Green summary, service list with counts, [View Logs] [Complete]
3. **Success (partial)**: Yellow summary, service list, warnings for no-logs, [View Logs] [Complete]
4. **Failure (no logs)**: Orange warning, guidance, [← Back to Diagnostic Settings] [Retry]
5. **Error**: Red error, error message, recovery actions

**Acceptance Criteria**:
- Actually queries workspace for logs
- Shows real log counts and timestamps
- Provides specific guidance per service state
- Error states link back to previous steps
- Success enables completion
- Loading states are informative
- Component tests with mocked API

**Files**:
- `cli/dashboard/src/components/SetupVerification.tsx` (modify)
- `cli/dashboard/src/hooks/useWorkspaceVerification.ts` (new)

---

### Testing - Backend API Tests

**Assigned**: Tester

Unit tests for new Azure APIs.

**Test Coverage**:
- Diagnostic settings check API
  - All services configured
  - No services configured
  - Partial configuration
  - Permission errors
  - Missing resources
- Workspace verification API
  - Successful log query
  - No logs (normal delay)
  - No diagnostic settings
  - Query errors
  - Invalid workspace
- Bicep template generation
  - Single service type
  - Multiple service types
  - Template syntax validation

**Acceptance Criteria**:
- 80%+ coverage on new code
- Mocked Azure ARM and Log Analytics APIs
- Edge cases covered
- Error scenarios tested

**Files**:
- `cli/src/internal/azure/diagnostics_test.go` (new)
- `cli/src/internal/azure/verification_test.go` (new)
- `cli/src/internal/azure/bicep_test.go` (new)

---

### Testing - Frontend Component Tests

**Assigned**: Tester

Component tests for new and modified UI.

**Test Coverage**:
- DiagnosticSettingsStep
  - Loading state
  - All configured state
  - Partial configured state
  - None configured state
  - Error state
  - Retry action
- BicepTemplateModal
  - Template display
  - Copy button
  - Download button
  - Instructions expand/collapse
  - Keyboard navigation
- SetupVerification
  - Loading state
  - Success (all services)
  - Success (partial)
  - No logs found
  - Error state
  - Retry action
  - Navigation actions

**Acceptance Criteria**:
- All UI states tested
- User interactions tested
- Accessibility tested (keyboard nav, ARIA)
- Mock API responses
- 80%+ coverage on components

**Files**:
- `cli/dashboard/src/components/DiagnosticSettingsStep.test.tsx` (modify)
- `cli/dashboard/src/components/BicepTemplateModal.test.tsx` (new)
- `cli/dashboard/src/components/SetupVerification.test.tsx` (modify)

---

### Documentation Update

**Assigned**: Developer

Update documentation to reflect new setup UX.

**Updates Required**:
- `cli/docs/features/azure-logs.md`
  - Update setup guide section
  - Add Bicep template integration steps
  - Update troubleshooting (new error messages)
- `cli/docs/design/components/azure-setup-guide.md` (if exists)
  - Document new flow
  - Update component specs
- README or quickstart
  - Update setup instructions

**Acceptance Criteria**:
- Setup guide reflects new UX
- Bicep integration clearly documented
- Screenshots updated (if present)
- Troubleshooting section current

**Files**:
- `cli/docs/features/azure-logs.md` (modify)
- `cli/dashboard/README.md` (modify if needed)

---

## Task Dependencies

```
Backend APIs → Frontend UI → Testing → Documentation

1. Diagnostic Settings Check API
   ↓
2. DiagnosticSettingsStep UI
   ↓
3. Bicep Template Generator API
   ↓
4. BicepTemplateModal UI
   ↓
5. Workspace Verification API
   ↓
6. SetupVerification UI
   ↓
7. Backend Tests
   ↓
8. Frontend Tests
   ↓
9. Documentation
```

## Priority Order

P0 (Minimum Viable):
1. Diagnostic Settings Check API
2. Aggregated Diagnostic Settings UI
3. Enhanced Verification Step
4. Testing

P1 (Enhanced):
5. Bicep Template Generator API
6. Bicep Template Modal
7. Documentation

## Success Metrics

- Setup time reduced from ~5 minutes to ~2 minutes
- Diagnostic settings configuration goes from 3 actions to 1 action
- Verification step provides actual validation (not placeholder)
- Error recovery paths are clear (100% of error states have next action)
