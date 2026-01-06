# Task #2 Completion: Diagnostic Settings Check API

## Summary

Successfully implemented the diagnostic settings check API for the Azure Logs setup UX improvement as specified in Task #2 of `docs/specs/azure-logs-setup-ux/tasks.md`.

## Implemented Components

### 1. Core Logic: `cli/src/internal/azure/diagnostics.go`

**Key Features:**
- `DiagnosticSettingsChecker` - Main checker that queries Azure Management API
- `CheckAllServices()` - Checks diagnostic settings for all discovered services in a single operation
- `CheckSingleService()` - Checks a specific service by name
- Smart workspace matching - Handles different workspace ID formats (full resource ID, name only, GUID)
- Graceful error handling - Distinguishes between "not configured", "configured", and "error" states

**API Integration:**
- Uses Azure Management API: `https://management.azure.com/{resourceUri}/providers/Microsoft.Insights/diagnosticSettings`
- API Version: `2021-05-01-preview`
- Authenticates using existing credential chain (azd token, Azure CLI, etc.)

**Status Values:**
- `configured` - Diagnostic settings exist and point to the expected Log Analytics workspace
- `not-configured` - No diagnostic settings found or workspace not configured
- `error` - Permission denied, API errors, or settings point to wrong workspace

### 2. API Endpoint: `cli/src/internal/dashboard/azure_logs_handlers.go`

**Endpoint:** `GET /api/azure/diagnostic-settings/check`

**Handler:** `handleAzureDiagnosticSettingsCheck`

**Response Format:**
```json
{
  "workspaceId": "/subscriptions/.../workspaces/my-workspace",
  "services": {
    "api": {
      "status": "configured",
      "resourceId": "/subscriptions/.../Microsoft.Web/sites/api",
      "diagnosticSettingName": "toLogAnalytics",
      "workspaceId": "/subscriptions/.../workspaces/my-workspace"
    },
    "web": {
      "status": "not-configured",
      "resourceId": "/subscriptions/.../Microsoft.Web/sites/web",
      "error": "No diagnostic settings found"
    },
    "function": {
      "status": "error",
      "resourceId": "/subscriptions/.../Microsoft.Web/sites/function",
      "error": "Insufficient permissions"
    }
  }
}
```

**Error Handling:**
- 401 Unauthorized - No Azure credentials available
- 504 Gateway Timeout - Request timed out (30 second timeout)
- 500 Internal Server Error - Discovery or other failures

### 3. Routing: `cli/src/internal/dashboard/server_routes.go`

Added route:
```go
s.mux.HandleFunc("/api/azure/diagnostic-settings/check", 
    MethodGuard(s.handleAzureDiagnosticSettingsCheck, http.MethodGet))
```

### 4. Unit Tests: `cli/src/internal/azure/diagnostics_test.go`

**Test Coverage:**
- ✅ Workspace matching logic (exact match, case insensitive, resource ID extraction)
- ✅ Workspace name extraction from resource IDs
- ✅ Different diagnostic settings configurations
- ✅ Error scenarios (404, 403, 500)
- ✅ Edge cases (storage account only, wrong workspace, no settings)
- ✅ JSON serialization/deserialization
- ✅ Status constant values

**Test Results:**
```
PASS: TestDiagnosticSettingsChecker_CheckDiagnosticSettings
PASS: TestWorkspaceMatches (8 sub-tests)
PASS: TestExtractWorkspaceName (6 sub-tests)
PASS: TestDiagnosticSettingsResponse_Serialization
PASS: TestDiagnosticSettingsStatus_StringValues
```

## Implementation Details

### Resource Discovery Integration

The checker integrates with the existing `ResourceDiscovery` system:
1. Discovers all services from `azd env get-values`
2. Maps service names to Azure resource IDs
3. Queries diagnostic settings for each resource
4. Returns aggregated status

### Workspace ID Matching

Handles multiple workspace identifier formats:
- Full resource ID: `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.OperationalInsights/workspaces/{name}`
- Workspace name only: `my-workspace`
- GUID format (for future Log Analytics API integration)
- Case-insensitive comparison
- Extracts workspace name from resource IDs for comparison

### Permission Handling

Gracefully handles Azure RBAC scenarios:
- Returns `error` status with actionable message for 403 Forbidden
- Logs debug information for troubleshooting
- Doesn't fail entire check if one service has permission issues
- Continues checking other services

### Performance

- Single API call per service (parallel execution possible)
- 30 second timeout to prevent hanging
- Reuses existing credential and discovery infrastructure
- Caches discovery results (5 minute TTL)

## Acceptance Criteria ✅

All acceptance criteria from `tasks.md` met:

✅ **Returns status for all detected services in single call**
   - `CheckAllServices()` returns map of all service statuses

✅ **Gracefully handles missing/undeployed services**
   - Returns `not-configured` status with clear error message
   - Doesn't throw errors for missing services

✅ **Includes workspace ID for context**
   - Response includes expected workspace ID at top level
   - Each configured service includes actual workspace ID

✅ **Error messages are actionable**
   - "No diagnostic settings found for this resource"
   - "Insufficient permissions to check diagnostic settings"
   - "Diagnostic settings configured but not sending to expected workspace"

✅ **Unit tests with mocked Azure ARM API**
   - 20+ test cases covering all scenarios
   - Mock HTTP server for API responses
   - Tests for workspace matching logic
   - Edge case coverage

## Files Created/Modified

### Created:
- `cli/src/internal/azure/diagnostics.go` (458 lines)
- `cli/src/internal/azure/diagnostics_test.go` (403 lines)

### Modified:
- `cli/src/internal/dashboard/azure_logs_handlers.go` (+36 lines)
- `cli/src/internal/dashboard/server_routes.go` (+1 line)

## Next Steps

The diagnostic settings check API is ready for frontend integration. Next tasks in the sequence:

1. **Task #3**: Workspace Verification API (check if logs are actually flowing)
2. **Task #4**: Bicep Template Generator API
3. **Task #5**: Frontend - Aggregated Diagnostic Settings UI
4. **Task #6**: Frontend - Bicep Template Modal
5. **Task #7**: Frontend - Enhanced Verification Step

## Testing Instructions

### Build and Test:
```bash
cd cli
mage build
cd src/internal/azure
go test -v -run "TestDiagnostic|TestWorkspace|TestExtract"
```

### Manual API Testing (requires deployed Azure resources):
```bash
# Start the app
azd app run

# In another terminal, test the endpoint
curl http://localhost:4280/api/azure/diagnostic-settings/check
```

### Expected Response (example):
```json
{
  "workspaceId": "/subscriptions/abc-123/resourceGroups/rg-test/providers/Microsoft.OperationalInsights/workspaces/workspace-test",
  "services": {
    "api": {
      "status": "configured",
      "resourceId": "/subscriptions/abc-123/resourceGroups/rg-test/providers/Microsoft.Web/sites/api-test",
      "diagnosticSettingName": "toLogAnalytics",
      "workspaceId": "/subscriptions/abc-123/resourceGroups/rg-test/providers/Microsoft.OperationalInsights/workspaces/workspace-test"
    }
  }
}
```

## Notes

- Implementation follows existing patterns from `azure_setup.go` and other Azure integration code
- Uses the same credential chain and discovery mechanisms
- Error handling matches existing API endpoints
- All tests pass and code builds successfully
- Ready for frontend integration

---

**Status:** ✅ Complete
**Build Status:** ✅ Passing
**Tests Status:** ✅ All tests passing
**Review Status:** Ready for review
