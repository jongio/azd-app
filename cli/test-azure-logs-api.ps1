#!/usr/bin/env pwsh
# Test script for Azure logs dashboard API endpoint

param(
    [int]$Port = 53280,
    [string]$Service = "",
    [string]$Since = "1h",
    [int]$Tail = 100
)

$baseUrl = "http://localhost:$Port"

Write-Host "`n=== Testing Azure Logs Dashboard API ===" -ForegroundColor Cyan
Write-Host "Base URL: $baseUrl`n" -ForegroundColor Gray

# Test 1: Ping endpoint
Write-Host "[Test 1] Checking dashboard is running..." -ForegroundColor Yellow
try {
    $ping = Invoke-RestMethod -Uri "$baseUrl/api/ping" -Method GET -ErrorAction Stop
    Write-Host "✓ Dashboard is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Dashboard not running on port $Port" -ForegroundColor Red
    Write-Host "  Start with: azd app run" -ForegroundColor Gray
    exit 1
}

# Test 2: Get all Azure logs
Write-Host "`n[Test 2] Fetching all Azure logs (last $Since)..." -ForegroundColor Yellow
$url = "$baseUrl/api/azure/logs?since=$Since&tail=$Tail"
Write-Host "URL: $url" -ForegroundColor Gray

try {
    $response = Invoke-RestMethod -Uri $url -Method GET -ErrorAction Stop
    
    Write-Host "✓ Request successful" -ForegroundColor Green
    Write-Host "`nResponse:" -ForegroundColor White
    Write-Host "  Status: $($response.status)" -ForegroundColor Cyan
    Write-Host "  Count: $($response.count)" -ForegroundColor Cyan
    Write-Host "  Timestamp: $($response.timestamp)" -ForegroundColor Cyan
    
    if ($response.status -eq "ok") {
        Write-Host "`n  Sample logs:" -ForegroundColor White
        $response.logs | Select-Object -First 3 | ForEach-Object {
            Write-Host "    - [$($_.timestamp)] $($_.service): $($_.message.Substring(0, [Math]::Min(60, $_.message.Length)))..." -ForegroundColor Gray
        }
    } elseif ($response.error) {
        Write-Host "`n  Error Details:" -ForegroundColor Red
        Write-Host "    Code: $($response.error.code)" -ForegroundColor Yellow
        Write-Host "    Message: $($response.error.message)" -ForegroundColor Yellow
        Write-Host "    Action: $($response.error.action)" -ForegroundColor Yellow
        if ($response.error.command) {
            Write-Host "    Command: $($response.error.command)" -ForegroundColor Cyan
        }
        Write-Host "    Docs: $($response.error.docsUrl)" -ForegroundColor Blue
    }
    
} catch {
    Write-Host "✗ Request failed" -ForegroundColor Red
    Write-Host "  Status: $($_.Exception.Response.StatusCode.value__)" -ForegroundColor Yellow
    Write-Host "  Message: $($_.Exception.Message)" -ForegroundColor Yellow
    
    # Try to read error response body
    if ($_.Exception.Response) {
        try {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $errorBody = $reader.ReadToEnd()
            $errorJson = $errorBody | ConvertFrom-Json
            
            Write-Host "`n  Error Response:" -ForegroundColor Red
            Write-Host "    Status: $($errorJson.status)" -ForegroundColor Yellow
            if ($errorJson.error) {
                Write-Host "    Code: $($errorJson.error.code)" -ForegroundColor Yellow
                Write-Host "    Message: $($errorJson.error.message)" -ForegroundColor Yellow
                Write-Host "    Action: $($errorJson.error.action)" -ForegroundColor Yellow
                if ($errorJson.error.command) {
                    Write-Host "    Command: $($errorJson.error.command)" -ForegroundColor Cyan
                }
                Write-Host "    Docs: $($errorJson.error.docsUrl)" -ForegroundColor Blue
            }
        } catch {
            Write-Host "  (Could not parse error response)" -ForegroundColor Gray
        }
    }
}

# Test 3: Get logs for specific service (if provided)
if ($Service) {
    Write-Host "`n[Test 3] Fetching logs for service: $Service..." -ForegroundColor Yellow
    $url = "$baseUrl/api/azure/logs?service=$Service&since=$Since&tail=$Tail"
    Write-Host "URL: $url" -ForegroundColor Gray
    
    try {
        $response = Invoke-RestMethod -Uri $url -Method GET -ErrorAction Stop
        
        Write-Host "✓ Request successful" -ForegroundColor Green
        Write-Host "  Status: $($response.status)" -ForegroundColor Cyan
        Write-Host "  Count: $($response.count)" -ForegroundColor Cyan
        
        if ($response.status -eq "ok" -and $response.count -gt 0) {
            Write-Host "`n  Sample logs from $Service`:" -ForegroundColor White
            $response.logs | Select-Object -First 3 | ForEach-Object {
                Write-Host "    - [$($_.timestamp)] $($_.message.Substring(0, [Math]::Min(60, $_.message.Length)))..." -ForegroundColor Gray
            }
        }
    } catch {
        Write-Host "✗ Request failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Test 4: Check Azure status
Write-Host "`n[Test 4] Checking Azure status..." -ForegroundColor Yellow
try {
    $status = Invoke-RestMethod -Uri "$baseUrl/api/azure/status" -Method GET -ErrorAction Stop
    
    Write-Host "✓ Azure status retrieved" -ForegroundColor Green
    Write-Host "  Mode: $($status.mode)" -ForegroundColor Cyan
    Write-Host "  Enabled: $($status.enabled)" -ForegroundColor Cyan
    Write-Host "  Connected: $($status.connected)" -ForegroundColor Cyan
    if ($status.workspaceId) {
        Write-Host "  Workspace: $($status.workspaceId)" -ForegroundColor Cyan
    }
} catch {
    Write-Host "✗ Failed to get Azure status: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n=== Tests Complete ===" -ForegroundColor Cyan
