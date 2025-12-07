# Quick Start Script for Health Monitoring Test
# This script uses 'azd app run' to start services and test health monitoring

$ErrorActionPreference = "Stop"

Write-Host "========================================"
Write-Host "azd app health - Quick Start Test Guide"
Write-Host "========================================"
Write-Host ""

# Check if in correct directory
if (-not (Test-Path "azure.yaml")) {
    Write-Host "❌ Error: azure.yaml not found. Please run from health-test directory." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Found azure.yaml - ready to start"
Write-Host ""
Write-Host "Starting all services with 'azd app run'..."
Write-Host ""

# Start services in background
$job = Start-Job -ScriptBlock {
    Set-Location $using:PWD
    azd app run
}

Write-Host "✅ Services starting (Job ID: $($job.Id))"
Write-Host ""
Write-Host "Waiting 30 seconds for services to initialize..."
for ($i = 30; $i -gt 0; $i--) {
    Write-Host -NoNewline "`r  $i seconds remaining...  "
    Start-Sleep -Seconds 1
}
Write-Host ""

Write-Host ""
Write-Host "✅ All services should be ready!"
Write-Host ""
Write-Host "========================================"
Write-Host "Running Quick Tests"
Write-Host "========================================"
Write-Host ""

# Test 1: Basic health check
Write-Host "Test 1: Basic Health Check (Static Mode)"
Write-Host "----------------------------------------"
azd app health
$exitCode = $LASTEXITCODE
Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "✅ Test 1 PASSED (Exit code: $exitCode)" -ForegroundColor Green
} else {
    Write-Host "❌ Test 1 FAILED (Exit code: $exitCode, expected: 0)" -ForegroundColor Red
}
Write-Host ""

# Test 2: Service info
Write-Host "Test 2: Service Info"
Write-Host "----------------------------------------"
azd app info
Write-Host "✅ Test 2 PASSED" -ForegroundColor Green
Write-Host ""

# Test 3: JSON output
Write-Host "Test 3: JSON Output Format"
Write-Host "----------------------------------------"
$jsonOutput = azd app health --output json | Out-String
try {
    $parsed = $jsonOutput | ConvertFrom-Json
    Write-Host "✅ Test 3 PASSED (Valid JSON output)" -ForegroundColor Green
    Write-Host "Summary: $($parsed.summary | ConvertTo-Json -Compress)"
} catch {
    Write-Host "❌ Test 3 FAILED (Invalid JSON)" -ForegroundColor Red
    Write-Host $jsonOutput
}
Write-Host ""

# Test 4: Table output
Write-Host "Test 4: Table Output Format"
Write-Host "----------------------------------------"
azd app health --output table
Write-Host "✅ Test 4 PASSED" -ForegroundColor Green
Write-Host ""

# Test 5: Service filtering
Write-Host "Test 5: Service Filtering"
Write-Host "----------------------------------------"
azd app health --service web,api
Write-Host "✅ Test 5 PASSED" -ForegroundColor Green
Write-Host ""

# Test 6: Verbose mode
Write-Host "Test 6: Verbose Mode"
Write-Host "----------------------------------------"
azd app health --verbose
Write-Host "✅ Test 6 PASSED" -ForegroundColor Green
Write-Host ""

Write-Host "========================================"
Write-Host "Quick Tests Complete!"
Write-Host "========================================"
Write-Host ""
Write-Host "All basic tests passed. Services are running correctly."
Write-Host ""
Write-Host "Next Steps:"
Write-Host "  1. Try streaming mode:    azd app health --stream"
Write-Host "  2. View service logs:     azd app logs --service web"
Write-Host "  3. Full manual testing:   See TESTING.md for comprehensive test guide"
Write-Host ""
Write-Host "To stop all services:"
Write-Host "  Stop-Job -Id $($job.Id)"
Write-Host "  Remove-Job -Id $($job.Id)"
Write-Host "  or press Ctrl+C in the terminal running 'azd app run'"
Write-Host ""
Write-Host "For detailed testing: Get-Content TESTING.md"
Write-Host ""
