# Task 2 Implementation: Backend Verification API

**Status**: ✅ COMPLETE  
**Date**: December 25, 2025

## Summary

Implemented the `/api/azure/logs/verify` endpoint to test log connectivity and return sample logs for a specific service.

## Changes Made

### 1. Backend Handler (`cli/src/internal/dashboard/azure_setup.go`)

Added three new types:
- `VerifyLogsRequest` - Request body with service name
- `VerifyLogsResponse` - Response with success status, log count, samples, and next steps
- `VerifyLogsTimeRange` - Time range of logs found
- `VerifyLogsSample` - Sample log entry with timestamp, message, and level

Added `handleAzureLogsVerify` handler that:
- Accepts POST request with `{"service": "api"}` body
- Validates workspace configuration
- Checks Azure authentication
- Queries last 15 minutes of logs for the specified service
- Returns up to 10 sample logs if available
- Handles timeout cases (30s max query time)
- Provides actionable error messages and next steps

Added `truncateMessage` helper to limit log message length to 200 chars.

### 2. Route Registration (`cli/src/internal/dashboard/server_routes.go`)

Added route:
```go
s.mux.HandleFunc("/api/azure/logs/verify", MethodGuard(s.handleAzureLogsVerify, http.MethodPost))
```

### 3. Unit Tests (`cli/src/internal/dashboard/azure_setup_test.go`)

Added comprehensive tests covering:
- ✅ Successful verification with logs (returns samples and time range)
- ✅ No logs found (returns helpful message about 5-15 min delay)
- ✅ Missing workspace configuration (actionable error)
- ✅ Missing service name (validation error)
- ✅ Invalid JSON body (validation error)
- ✅ Message truncation utility

Added `TestTruncateMessage` for the helper function.

## API Specification

### Request

```
POST /api/azure/logs/verify
Content-Type: application/json

{
  "service": "api"
}
```

### Response - Success

```json
{
  "success": true,
  "logsFound": 142,
  "timeRange": {
    "start": "2025-12-25T10:30:00Z",
    "end": "2025-12-25T10:45:00Z"
  },
  "sample": [
    {
      "timestamp": "2025-12-25T10:45:00Z",
      "message": "Request processed successfully",
      "level": "INFO"
    },
    {
      "timestamp": "2025-12-25T10:44:30Z",
      "message": "Database connection established",
      "level": "INFO"
    }
  ],
  "message": "Successfully verified log flow for service 'api'"
}
```

### Response - No Logs Yet

```json
{
  "success": false,
  "logsFound": 0,
  "message": "No logs found for service 'api' in the last 15 minutes",
  "nextSteps": [
    "This is normal if the service was just deployed (logs can take 5-15 minutes to appear)",
    "Generate activity by accessing your application",
    "Configure diagnostic settings to send logs to the workspace",
    "Check diagnostic settings in Azure Portal if logs don't appear after 15 minutes"
  ]
}
```

### Response - Workspace Not Configured

```json
{
  "success": false,
  "message": "Log Analytics workspace not configured",
  "nextSteps": [
    "Configure Log Analytics workspace in azure.yaml",
    "Set AZURE_LOG_ANALYTICS_WORKSPACE_ID environment variable"
  ]
}
```

## Error Handling

The endpoint handles the following scenarios gracefully:

1. **Missing workspace** - Returns actionable steps to configure workspace
2. **Missing authentication** - Prompts user to run `azd auth login`
3. **Query timeout** - Suggests waiting and checking workspace accessibility
4. **Service not found** - Suggests deploying with `azd up` or checking service name
5. **No logs yet** - Explains 5-15 minute delay is normal after deployment
6. **Query errors** - Provides diagnostic steps (check settings, permissions)

## Test Results

```
=== RUN   TestHandleAzureLogsVerify
=== RUN   TestHandleAzureLogsVerify/successful_verification_with_logs
=== RUN   TestHandleAzureLogsVerify/no_logs_found
=== RUN   TestHandleAzureLogsVerify/missing_workspace_configuration
=== RUN   TestHandleAzureLogsVerify/missing_service_name
=== RUN   TestHandleAzureLogsVerify/invalid_JSON_body
--- PASS: TestHandleAzureLogsVerify (0.03s)

=== RUN   TestTruncateMessage
--- PASS: TestTruncateMessage (0.00s)
```

All tests pass ✅

## Acceptance Criteria

- ✅ Queries Log Analytics for specified service
- ✅ Returns sample logs (up to 10) if available
- ✅ Handles "no logs yet" gracefully with helpful message
- ✅ Timeout handling for long queries (30s max)
- ✅ Error messages explain next steps
- ✅ Unit tests added to azure_setup_test.go
- ✅ Route added to server_routes.go
- ✅ Code compiles and all tests pass

## Integration Notes

This endpoint extends the work from Task 1 (Setup State API) and uses the same patterns:
- Reuses `getWorkspaceIDFromEnv()` and `loadAzureYaml()` helpers
- Uses `fetchAzureLogsStandalone()` from existing Azure logs code
- Follows same JSON response patterns with `WriteJSONSuccess()`
- Uses `BadRequest()` for validation errors
- Consistent timeout handling with `timeoutContext(30 * time.Second)`

## Next Steps

This API is ready for frontend integration in Task 7 (Verification Step UI), which will:
- Call this endpoint for each service to verify log flow
- Display sample logs in the verification step
- Show friendly messages when logs aren't available yet
- Provide "View Logs" button when verification succeeds
