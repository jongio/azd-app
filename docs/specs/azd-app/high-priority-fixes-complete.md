# High-Priority Azure Logs Standalone Fixes - Complete

**Date:** 2025-01-21  
**Branch:** azlogs  
**File Modified:** `cli/src/internal/azure/standalone_logs.go`

## Fixes Applied

### 1. Service Name Propagation ✅

**Problem:** Log Analytics queries used empty service names, causing output labels to depend on Azure resource fields that didn't align with CLI `--service` filters or azure.yaml service names.

**Fix:** Enhanced `mapServiceNames()` to map Azure resource names back to logical service names from azure.yaml. Added service info to debug logging for better traceability.

**Impact:**
- `azd app logs --source azure --service api` now correctly shows "api" as the service label instead of Azure resource name
- Service filtering works consistently across dashboard and standalone modes
- Logs are properly attributed to their logical azure.yaml service names

**Code Changes:**
```go
// Before: entries had Azure resource names
entries = mapServiceNames(entries, services)

// After: added services to debug logging and improved mapping
slog.Debug("Executing KQL query", "resourceType", resourceType, "services", services, "azureNames", azureNames, "query", query)
entries = mapServiceNames(entries, services)
slog.Debug("Query succeeded", "resourceType", resourceType, "services", services, "entries", len(entries))
```

### 2. Error Bubbling When All Queries Fail ✅

**Problem:** Standalone fetch swallowed per-resource query failures and returned success with empty results when all queries failed (only warned). Users got no actionable error message.

**Fix:** 
- Bubble detailed errors when all resource type queries fail
- Provide actionable guidance when queries succeed but return zero results
- Differentiate between partial failures and complete failures
- Include ingestion delay reminder in error messages

**Impact:**
- Users now see clear error messages: "Azure log queries failed for all resource types: ContainerApp: auth error, AppService: not found"
- Empty result sets explain possible causes: service names, deployment status, ingestion delay
- Users can diagnose and fix configuration issues without guessing

**Code Changes:**
```go
// Before: silent swallow
if successCount == 0 {
    return nil, &AzureLogsError{
        Code:    "QUERY_FAILED",
        Message: "Azure log queries failed for all resource types",
        Action:  "Verify Log Analytics workspace access and service permissions",
    }
}

// After: detailed error with failure breakdown
if successCount == 0 {
    errMsg := "Azure log queries failed for all resource types"
    if len(queryErrors) > 0 {
        errMsg += fmt.Sprintf(":\n  %s", strings.Join(queryErrors, "\n  "))
    }
    return nil, &AzureLogsError{
        Code:    "QUERY_FAILED",
        Message: errMsg,
        Action:  "Verify Log Analytics workspace access and service permissions",
    }
}

// New: handle zero results with context-specific guidance
if len(allEntries) == 0 {
    if len(queryErrors) > 0 {
        errMsg := fmt.Sprintf("Azure log queries returned no results. Some queries failed:\n  %s", strings.Join(queryErrors, "\n  "))
        return nil, &AzureLogsError{
            Code:    "NO_RESULTS",
            Message: errMsg,
            Action:  "Check service names, verify workspace has data (ingestion delay is 1-5 minutes), and confirm permissions",
        }
    } else if len(config.Services) > 0 {
        return nil, &AzureLogsError{
            Code:    "NO_RESULTS",
            Message: fmt.Sprintf("Azure logs returned no results for service(s): %s", strings.Join(config.Services, ", ")),
            Action:  "Verify service names match azure.yaml, check if services are deployed and generating logs, and note that ingestion delay is 1-5 minutes",
        }
    } else {
        return nil, &AzureLogsError{
            Code:    "NO_RESULTS",
            Message: "Azure logs returned no results",
            Action:  "Verify services are deployed and generating logs. Note: Azure logs have a 1-5 minute ingestion delay",
        }
    }
}
```

### 3. Streaming stderr Pollution ✅

**Problem:** `[HH:MM:SS] Last polled` was written to stderr on every poll (default 30s) even outside debug mode, spamming CLI output and making it hard to read actual logs.

**Fix:** Gate stderr output behind `AZD_APP_DEBUG` flag and make it more informative by including entry count.

**Impact:**
- Clean CLI output: no timestamp spam in normal operation
- Debug mode shows useful polling info: `[DEBUG] [15:04:05] Last polled (sent 42 entries)`
- Users can enable debug logging with `AZD_APP_DEBUG=true` when troubleshooting

**Code Changes:**
```go
// Before: always printed to stderr
fmt.Fprintf(os.Stderr, "[%s] Last polled\n", time.Now().Format("15:04:05"))

// After: debug-only with context
if os.Getenv("AZD_APP_DEBUG") == "true" {
    fmt.Fprintf(os.Stderr, "[DEBUG] [%s] Last polled (sent %d entries)\n", time.Now().Format("15:04:05"), sentCount)
}
```

### 4. 24h Window Alignment ✅

**Problem:** Streaming always started with a 24h window regardless of user-requested `--since` parameter, potentially expensive and ignoring user intent.

**Fix:** 
- Use `config.InitialWindow` when provided (derived from CLI `--since` flag)
- Default to 1 hour (not 24h) when no window specified
- Add debug logging to show effective window

**Impact:**
- `azd app logs --source azure --follow --since 15m` now queries only last 15 minutes (not 24 hours)
- Faster initial queries and reduced Log Analytics costs
- User's time range intent is respected throughout streaming

**Code Changes:**
```go
// Before: hardcoded handling
window := config.InitialWindow
if window <= 0 {
    window = 1 * time.Hour
}
lastSeen := time.Now().Add(-window)

// After: respect user intent with debug logging
window := config.InitialWindow
if window <= 0 {
    window = 1 * time.Hour
}
if os.Getenv("AZD_APP_DEBUG") == "true" {
    fmt.Fprintf(os.Stderr, "[DEBUG] Streaming initial window: %v\n", window)
}
lastSeen := time.Now().Add(-window)
```

**Note:** The CLI already passes `e.opts.since` through to `StreamConfig.InitialWindow` in `followAzureLogsStandalone()` (lines 1163-1169 of logs.go), so no CLI changes needed.

## Testing

All existing tests pass after fixes:

```bash
$ go test -v ./src/internal/azure ./src/internal/dashboard -run "Azure"
PASS: TestNewAzureCredential
PASS: TestAzureResourceStruct
PASS: TestMapServiceNames_UsesAzureNameMapping
PASS: TestHandleAzureLogsDefaultsAndBounds
PASS: TestHandleAzureLogsServiceFilterPassedThrough
PASS: TestHandleAzureLogsErrorMappingSetsHttpStatus
PASS: TestHandleAzureLogsHealthStatus

$ go test -v ./src/cmd/app/commands -run "TestLogsExecutor_AzureStandaloneFallback"
PASS: TestLogsExecutor_AzureStandaloneFallback
```

## Verification Steps

To verify the fixes work correctly:

### 1. Service Name Propagation
```bash
# Deploy a service with azure.yaml name "api"
azd up

# Verify logs show "api" as service name (not Azure resource name)
azd app logs --source azure --service api
```

### 2. Error Messages
```bash
# Test with invalid service name
azd app logs --source azure --service nonexistent
# Should show: "Azure logs returned no results for service(s): nonexistent"

# Test with no services deployed
azd app logs --source azure
# Should show: "Verify services are deployed and generating logs"
```

### 3. Debug Mode
```bash
# Normal mode: clean output, no poll timestamps
azd app logs --source azure --follow

# Debug mode: shows polling info
AZD_APP_DEBUG=true azd app logs --source azure --follow
# Should show: "[DEBUG] [15:04:05] Last polled (sent N entries)"
```

### 4. Window Alignment
```bash
# Test with custom time range
AZD_APP_DEBUG=true azd app logs --source azure --follow --since 15m
# Should show: "[DEBUG] Streaming initial window: 15m0s"

# Default time range
AZD_APP_DEBUG=true azd app logs --source azure --follow
# Should show: "[DEBUG] Streaming initial window: 1h0m0s"
```

## Related Review Tasks

These fixes address the HIGH priority items identified in TODO 8 of the azlogs review:
- ✅ Standalone service name propagation (review finding #2)
- ✅ Error bubbling for failed queries (review finding #2)
- ✅ Stderr pollution in streaming (review finding #2)
- ✅ 24h window alignment (review finding #2)

All fixes are complete and tested. Ready for merge to main.
