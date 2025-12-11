# Manual Test Commands for Azure Logs API

## Prerequisites
```powershell
# 1. Build the CLI
cd c:\code\azd-app-2\cli
mage build

# 2. Start the dashboard (with Azure-enabled project)
cd c:\code\azd-app-2\cli\tests\projects\integration\azure-logs-test
azd app run
```

## Test Commands

### 1. Check dashboard is running
```powershell
curl.exe http://localhost:53280/api/ping
# Expected: {"message":"pong"}
```

### 2. Get all Azure logs (default: last 1 hour)
```powershell
curl.exe http://localhost:53280/api/azure/logs
```

Expected success response:
```json
{
  "status": "ok",
  "logs": [
    {
      "service": "containerapp-api",
      "message": "log message here",
      "level": 0,
      "timestamp": "2025-12-10T20:00:00Z",
      "source": "azure",
      "azureMetadata": {
        "resourceType": "containerapp",
        "containerName": "api"
      }
    }
  ],
  "count": 10,
  "timestamp": "2025-12-10T20:05:00Z"
}
```

Expected error response (not authenticated):
```json
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

### 3. Get logs for specific service
```powershell
curl.exe "http://localhost:53280/api/azure/logs?service=containerapp-api"
```

### 4. Get logs with custom time range
```powershell
# Last 30 minutes
curl.exe "http://localhost:53280/api/azure/logs?since=30m"

# Last 2 hours
curl.exe "http://localhost:53280/api/azure/logs?since=2h"

# Last 15 minutes
curl.exe "http://localhost:53280/api/azure/logs?since=15m"
```

### 5. Limit number of logs returned
```powershell
# Get only 10 logs
curl.exe "http://localhost:53280/api/azure/logs?tail=10"

# Get 100 logs from specific service
curl.exe "http://localhost:53280/api/azure/logs?service=api&tail=100"
```

### 6. Combined filters
```powershell
curl.exe "http://localhost:53280/api/azure/logs?service=containerapp-api&since=1h&tail=50"
```

### 7. Check Azure status
```powershell
curl.exe http://localhost:53280/api/azure/status
```

Expected:
```json
{
  "mode": "azure",
  "connected": true,
  "enabled": true,
  "resourceCount": 3,
  "workspaceId": "guid-here"
}
```

## Using PowerShell for Better Formatting

```powershell
# Pretty print JSON response
$response = Invoke-RestMethod -Uri "http://localhost:53280/api/azure/logs?since=30m"
$response | ConvertTo-Json -Depth 5

# Show just the status and count
$response = Invoke-RestMethod -Uri "http://localhost:53280/api/azure/logs"
Write-Host "Status: $($response.status)"
Write-Host "Count: $($response.count)"

# Show error details if present
if ($response.error) {
    Write-Host "`nError Details:"
    Write-Host "  Code: $($response.error.code)"
    Write-Host "  Message: $($response.error.message)"
    Write-Host "  Action: $($response.error.action)"
    Write-Host "  Command: $($response.error.command)"
    Write-Host "  Docs: $($response.error.docsUrl)"
}

# Show log samples
if ($response.logs) {
    Write-Host "`nSample Logs:"
    $response.logs | Select-Object -First 5 | ForEach-Object {
        Write-Host "  [$($_.timestamp)] $($_.service): $($_.message.Substring(0, [Math]::Min(80, $_.message.Length)))..."
    }
}
```

## Using the Test Script

```powershell
# Run all tests
cd c:\code\azd-app-2\cli
.\test-azure-logs-api.ps1

# Test with custom parameters
.\test-azure-logs-api.ps1 -Port 53280 -Service "containerapp-api" -Since "30m" -Tail 50
```

## Troubleshooting

### Dashboard not responding
```powershell
# Check if dashboard is running
Get-Process | Where-Object { $_.ProcessName -match "azd" }

# Restart dashboard
cd c:\code\azd-app-2\cli\tests\projects\integration\azure-logs-test
azd app run
```

### Port already in use
```powershell
# Find process using port
Get-NetTCPConnection -LocalPort 53280 | Select-Object OwningProcess
Get-Process -Id <PID>

# Kill the process
Stop-Process -Id <PID> -Force
```

### Authentication errors
```powershell
# Login to Azure
azd auth login

# Verify environment variables
azd env get-values | Select-String "AZURE_"
```

## Expected Error Codes

| Code | HTTP Status | Meaning | Fix Command |
|------|-------------|---------|-------------|
| `AUTH_EXPIRED` | 401 | Azure auth expired | `azd auth login` |
| `AUTH_REQUIRED` | 401 | Not authenticated | `azd auth login` |
| `NOT_DEPLOYED` | 503 | Resources not deployed | `azd up` |
| `NO_WORKSPACE` | 503 | Log Analytics not configured | `azd up` or set env var |
| `NO_PERMISSION` | 403 | Missing permissions | Grant Log Analytics Reader role |
| `UNKNOWN` | 500 | Other error | Check details |
