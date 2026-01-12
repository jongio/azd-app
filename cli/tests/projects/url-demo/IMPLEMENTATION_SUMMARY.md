# Task 4: Console Output Formatting - Implementation Summary

## Overview
Successfully implemented console output formatting to display custom URLs alongside deployment URLs when configured in `azure.yaml`.

## Files Changed

### 1. `src/internal/serviceinfo/serviceinfo.go`
**Changes:**
- Added `AltURL` field to `AzureServiceInfo` struct
- Updated `mergeServiceInfo` function to extract `url` from service config
- Preserved `url` when overlaying Azure service information from environment

**Key Code:**
```go
type AzureServiceInfo struct {
    URL          string `json:"url,omitempty"`
    ResourceName string `json:"resourceName,omitempty"`
    ImageName    string `json:"imageName,omitempty"`
    AltURL       string `json:"altUrl,omitempty"` // Alternate URL from config
}
```

### 2. `src/cmd/app/commands/info.go`
**Changes:**
- Updated console output formatting to display both deployment URL and access URL
- Implemented conditional logic to show:
  - "Deployment URL" + "Access URL" when both are present
  - "Azure URL" when only deployment URL exists (backward compatible)
  - "Access URL" when only custom URL is configured

**Key Code:**
```go
if svc.Azure != nil {
    if svc.Azure.URL != "" {
        if svc.Azure.AltURL != "" {
            cliout.Label("  Deployment URL", svc.Azure.URL)
            cliout.Label("  Access URL", svc.Azure.AltURL)
        } else {
            cliout.Label("  Azure URL", svc.Azure.URL)
        }
    } else if svc.Azure.AltURL != "" {
        cliout.Label("  Access URL", svc.Azure.AltURL)
    }
}
```

### 3. `src/internal/serviceinfo/serviceinfo_test.go`
**Changes:**
- Added `TestMergeServiceInfo_WithAltURL` test case
- Verifies that custom URLs are correctly extracted from service config
- Tests both services with and without custom URLs

### 4. `src/cmd/app/commands/info_test.go`
**Changes:**
- Added `TestPrintInfoDefault_WithAltURL` test case
- Verifies console output formatting with mixed scenarios
- Tests service with url and service without url

## Test Results

### Unit Tests
All tests passing:
```
✓ TestMergeServiceInfo_WithAltURL
✓ TestPrintInfoDefault_WithAltURL
✓ TestFormatStatus
✓ TestFormatHealth
✓ TestRunInfo*
```

### Example Console Output

**Service with custom URL:**
```
ℹ    ● web
     Deployment URL: https://web-abc123.azurecontainerapps.io
     Access URL: https://myapp.example.com
     Language:  node
     Framework: express
     Project:   ./src/web
     Status:    not-running
```

**Service without custom URL:**
```
ℹ    ● api
     Azure URL: https://api-abc123.azurewebsites.net
     Language:  python
     Framework: flask
     Project:   ./src/api
     Status:    not-running
```

### JSON Output
The JSON output correctly includes the `altUrl` field:
```json
{
  "azure": {
    "url": "https://web-abc123.azurecontainerapps.io",
    "altUrl": "https://myapp.example.com"
  }
}
```

## Acceptance Criteria - ✅ All Met

- ✅ Console output displays custom URLs when configured
- ✅ Clear distinction between "Deployment URL" and "Access URL"
- ✅ Backward compatible - services without url show only deployment URL
- ✅ Consistent formatting across different commands (`azd app info`)
- ✅ All tests pass
- ✅ No breaking changes to existing console output for services without url

## Integration with Previous Tasks

This implementation builds on:
- **Task 1**: ServiceConfig with URL field (already implemented)
- **Task 2**: Configuration validation (already implemented)
- **Task 3**: Dashboard UI updates (separate task)

The console output now correctly reads the `url` from `azure.yaml` and displays it alongside deployment URLs.

## Edge Cases Handled

1. **Service with url but no Azure deployment URL**: Shows only "Access URL"
2. **Service with Azure deployment URL but no url**: Shows only "Azure URL" (backward compatible)
3. **Service with both URLs**: Shows "Deployment URL" and "Access URL"
4. **Service with neither**: Shows no URL fields

## Example Usage

### Configuration
```yaml
services:
  web:
    language: node
    host: containerapp
    project: ./src/web
    url: https://myapp.example.com
```

### Command
```bash
azd app info
```

### Output
```
📦 Project: /path/to/project

ℹ    ● web
     Deployment URL: https://web-abc123.azurecontainerapps.io
     Access URL: https://myapp.example.com
     Language:  node
     Framework: express
     Status:    not-running
```

## Notes

- The implementation maintains full backward compatibility
- JSON output includes the url field when present
- Console formatting is consistent with existing patterns
- All error handling is preserved from previous implementations
