# Azure Logs Dashboard API Implementation Summary

## What Was Implemented

### 1. Enhanced `/api/azure/logs` Endpoint

**Location**: `cli/src/internal/dashboard/azure_logs.go`

**Handler**: `handleAzureLogs`

### 2. Structured Response Format

Added two new types:

```go
type AzureLogsResponse struct {
    Status    string              `json:"status"`              // "ok" | "error"
    Logs      []service.LogEntry  `json:"logs,omitempty"`      // Log entries
    Count     int                 `json:"count"`               // Number of logs returned
    Timestamp time.Time           `json:"timestamp"`           // Response timestamp
    Error     *ErrorInfo          `json:"error,omitempty"`     // Error details if status=error
}

type ErrorInfo struct {
    Message string `json:"message"`         // Human-readable error message
    Code    string `json:"code"`            // Error code: "AUTH_EXPIRED", "NOT_DEPLOYED", etc.
    Action  string `json:"action"`          // What the user should do
    Command string `json:"command"`         // CLI command to run (optional)
    DocsURL string `json:"docsUrl"`         // Documentation URL
}
```

### 3. Query Parameters Supported

- `service=<name>`: Filter logs by service name
- `since=<duration>`: Time range (e.g., "1h", "30m", "2h") - Default: 1 hour
- `tail=<number>`: Max number of logs to return - Default: 500, Max: 10000

### 4. Error Handling with Documentation Links

The endpoint maps Azure errors to structured `ErrorInfo` with:
- Error codes: `AUTH_EXPIRED`, `AUTH_REQUIRED`, `NOT_DEPLOYED`, `NO_WORKSPACE`, `NO_PERMISSION`, `UNKNOWN`
- Actionable messages
- CLI commands to fix issues
- Documentation URLs pointing to troubleshooting guides

Error mapping:
- `AUTH_EXPIRED`, `AUTH_REQUIRED` → HTTP 401, docs: `https://aka.ms/azd/app/logs/troubleshoot#auth`
- `NOT_DEPLOYED` → HTTP 503, docs: `https://aka.ms/azd/app/logs/setup`
- `NO_WORKSPACE` → HTTP 503, docs: `https://aka.ms/azd/app/logs/configure`
- `NO_PERMISSION` → HTTP 403, docs: `https://aka.ms/azd/app/logs/troubleshoot#permissions`
- Other errors → HTTP 500, docs: `https://aka.ms/azd/app/logs/troubleshoot`

### 5. Integration with Standalone Logs Fetcher

The endpoint uses `azure.FetchAzureLogsStandalone()` which:
- Queries Azure Log Analytics directly
- Supports multiple resource types (Container Apps, App Service, Functions, AKS, ACI)
- Maps azure.yaml service names to Azure resource names
- Handles authentication via DefaultAzureCredential

### 6. Type Conversion

Added `convertAzureLogLevel()` helper to convert between `azure.LogLevel` and `service.LogLevel`.

Converts `azure.LogEntry` to `service.LogEntry` with proper metadata mapping.

## Example Usage

### Success Response
```bash
GET /api/azure/logs?since=30m&service=api

{
  "status": "ok",
  "logs": [
    {
      "service": "containerapp-api",
      "message": "Server started on port 3000",
      "level": 0,
      "timestamp": "2025-12-10T20:00:00Z",
      "source": "azure",
      "azureMetadata": {
        "resourceType": "containerapp",
        "containerName": "api",
        "instanceId": "ca-123-abc"
      }
    }
  ],
  "count": 1,
  "timestamp": "2025-12-10T20:05:00Z"
}
```

### Error Response
```bash
GET /api/azure/logs

{
  "status": "error",
  "count": 0,
  "timestamp": "2025-12-10T20:05:00Z",
  "error": {
    "message": "Azure authentication required",
    "code": "AUTH_REQUIRED",
    "action": "Run 'azd auth login' to authenticate",
    "command": "azd auth login",
    "docsUrl": "https://aka.ms/azd/app/logs/troubleshoot#auth"
  }
}
```

## Testing Instructions

### 1. Build the CLI
```powershell
cd c:\code\azd-app-2\cli
mage build
```

### 2. Start the dashboard
```powershell
cd c:\code\azd-app-2\cli\tests\projects\integration\azure-logs-test
azd app run
```

### 3. Test the endpoint
```powershell
# Get all Azure logs from last hour
Invoke-RestMethod -Uri "http://localhost:<PORT>/api/azure/logs"

# Get logs for specific service from last 30 minutes
Invoke-RestMethod -Uri "http://localhost:<PORT>/api/azure/logs?service=api&since=30m"

# Get recent 100 logs
Invoke-RestMethod -Uri "http://localhost:<PORT>/api/azure/logs?tail=100"

# Test error response (if not authenticated)
Invoke-RestMethod -Uri "http://localhost:<PORT>/api/azure/logs" -ErrorAction Stop
```

## Build Status

✅ **Compilation**: Successfully built with no errors
✅ **Type Safety**: All type conversions handled correctly
✅ **Error Handling**: Comprehensive error mapping with docs links
✅ **Code Quality**: Follows existing patterns in the codebase

## Integration Points

- **Standalone Logs**: Uses `azure.FetchAzureLogsStandalone()` from `cli/src/internal/azure/standalone_logs.go`
- **Log Manager**: Integrates with existing log management infrastructure
- **Error Types**: Uses `azure.AzureLogsError` for structured error handling
- **HTTP Helpers**: Uses existing `writeJSON()` and `writeJSONError()` helpers

## Next Steps for Testing

1. ✅ Build completed successfully
2. ⏳ Start dashboard with test project
3. ⏳ Verify endpoint returns logs when authenticated
4. ⏳ Verify error responses have all required fields
5. ⏳ Test service filtering works
6. ⏳ Test time range filtering works
7. ⏳ Verify documentation URLs are correct

## Files Modified

- `cli/src/internal/dashboard/azure_logs.go` - Enhanced `handleAzureLogs` function with:
  - New response types (`AzureLogsResponse`, `ErrorInfo`)
  - Query parameter parsing for `service`, `since`, `tail`
  - Structured error handling with docs links
  - Integration with standalone logs fetcher
  - Type conversion helpers

## Status

**Implementation**: ✅ **Complete**
**Build**: ✅ **Success**
**Testing**: ⏳ **Pending** (requires running dashboard instance)

The endpoint is fully implemented and ready for use. It follows the spec requirements:
- ✅ Structured JSON responses with status field
- ✅ Error responses with actionable guidance and docs URLs
- ✅ Support for service and time filters
- ✅ Integration with existing Azure logs infrastructure
- ✅ Proper HTTP status codes for different error types
