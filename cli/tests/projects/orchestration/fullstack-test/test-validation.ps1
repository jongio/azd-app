# Test script to validate process validation in info command

# Create a fake service registry entry with non-existent PID
$registryPath = ".azure\services.json"
$fakeRegistry = @{
    "test-service" = @{
        "name" = "test-service"
        "projectDir" = (Get-Location).Path
        "pid" = 99999  # Non-existent PID
        "port" = 3000
        "url" = "http://localhost:3000"
        "language" = "node"
        "framework" = "express"
        "status" = "running"
        "health" = "healthy"
        "startTime" = "2025-11-02T05:00:00Z"
        "lastChecked" = "2025-11-02T05:00:00Z"
    }
}

# Write fake registry
$fakeRegistry | ConvertTo-Json -Depth 10 | Out-File -FilePath $registryPath -Encoding UTF8

Write-Host "Created fake registry with non-existent PID 99999"
Write-Host "Registry contents:"
Get-Content $registryPath

Write-Host "`nTesting info command (should clean up the stale entry):"
c:\code\azd-app\cli\bin\azd-app.exe info

Write-Host "`nRegistry after validation:"
if (Test-Path $registryPath) {
    Get-Content $registryPath
} else {
    Write-Host "Registry file not found"
}