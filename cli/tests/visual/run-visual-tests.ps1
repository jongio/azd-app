# Master runner for progress bar analysis testing
# 1. Captures terminal outputs at multiple widths
# 2. Runs analysis tests
# 3. Shows results

param(
    [int[]]$Widths = @(50, 80, 120),
    [switch]$SkipCapture,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$scriptDir = $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Progress Bar Analysis Tests" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Check dependencies
Write-Host "[Step 1/3] Checking dependencies..." -ForegroundColor Cyan

if (!(Test-Path "$scriptDir\node_modules")) {
    Write-Host "  Installing npm dependencies..." -ForegroundColor Yellow
    Push-Location $scriptDir
    npm install
    Pop-Location
}

Write-Host "  ✓ Dependencies ready" -ForegroundColor Green

# Step 2: Capture outputs
if (!$SkipCapture) {
    Write-Host "`n[Step 2/3] Capturing terminal outputs..." -ForegroundColor Cyan
    & "$scriptDir\capture-outputs.ps1" -Widths $Widths -CleanFirst
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  ❌ Capture failed!" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "`n[Step 2/3] Skipping capture (using existing outputs)" -ForegroundColor Yellow
}

# Step 3: Run analysis tests
if (!$SkipTests) {
    Write-Host "`n[Step 3/3] Running analysis tests..." -ForegroundColor Cyan
    Push-Location $scriptDir
    npm test
    $testExitCode = $LASTEXITCODE
    Pop-Location
    
    if ($testExitCode -ne 0) {
        Write-Host "  ❌ Tests failed!" -ForegroundColor Red
    } else {
        Write-Host "  ✓ All tests passed!" -ForegroundColor Green
    }
} else {
    Write-Host "`n[Step 3/3] Skipping tests" -ForegroundColor Yellow
    $testExitCode = 0
}

# Summary
$comparisonReportPath = "$scriptDir\test-results\comparison-report.json"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test results: $(if ($testExitCode -eq 0) { 'PASSED' } else { 'FAILED' })" -ForegroundColor $(if ($testExitCode -eq 0) { 'Green' } else { 'Red' })
Write-Host ""

if (Test-Path $comparisonReportPath) {
    $report = Get-Content $comparisonReportPath | ConvertFrom-Json
    Write-Host "Analysis Results:" -ForegroundColor Cyan
    foreach ($result in $report.results) {
        $ratio = if ($report.baseline.progressLines -gt 0) { 
            [math]::Round($result.progressLines / $report.baseline.progressLines, 2) 
        } else { 0 }
        
        $status = if ($ratio -le 2.5) { "✓" } else { "❌" }
        $color = if ($ratio -le 2.5) { "Green" } else { "Red" }
        
        Write-Host "  $status Width $($result.width): ratio $ratio" -ForegroundColor $color
    }
}

Write-Host ""
Write-Host "Files generated:" -ForegroundColor Cyan
Write-Host "  Reports: $scriptDir\test-results\" -ForegroundColor Gray

Write-Host ""
if ($testExitCode -eq 0) {
    Write-Host "✓ Analysis complete!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Some tests failed. Review the reports for details." -ForegroundColor Red
    exit 1
}
