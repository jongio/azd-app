# Master runner for visual testing
# 1. Captures terminal outputs at multiple widths
# 2. Runs Playwright visual tests
# 3. Opens HTML report

param(
    [int[]]$Widths = @(40, 50, 60, 80, 100, 120, 140),
    [switch]$SkipCapture,
    [switch]$SkipTests,
    [switch]$OpenReport = $true
)

$ErrorActionPreference = "Stop"

$scriptDir = $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Visual Testing Suite for azd Progress Bars" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Check dependencies
Write-Host "[Step 1/4] Checking dependencies..." -ForegroundColor Cyan

if (!(Test-Path "$scriptDir\node_modules")) {
    Write-Host "  Installing npm dependencies..." -ForegroundColor Yellow
    Push-Location $scriptDir
    npm install
    Pop-Location
}

# Check Playwright browsers
$playwrightCheck = & npx playwright --version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "  Installing Playwright browsers..." -ForegroundColor Yellow
    Push-Location $scriptDir
    npx playwright install chromium
    Pop-Location
}

Write-Host "  ✓ Dependencies ready" -ForegroundColor Green

# Step 2: Capture outputs
if (!$SkipCapture) {
    Write-Host "`n[Step 2/4] Capturing terminal outputs..." -ForegroundColor Cyan
    $widthsParam = $Widths -join ','
    & "$scriptDir\capture-outputs.ps1" -Widths $Widths -CleanFirst
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  ❌ Capture failed!" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "`n[Step 2/4] Skipping capture (using existing outputs)" -ForegroundColor Yellow
}

# Step 3: Run Playwright tests
if (!$SkipTests) {
    Write-Host "`n[Step 3/4] Running Playwright visual tests..." -ForegroundColor Cyan
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
    Write-Host "`n[Step 3/4] Skipping tests" -ForegroundColor Yellow
    $testExitCode = 0
}

# Step 4: Generate summary and open report
Write-Host "`n[Step 4/4] Generating summary..." -ForegroundColor Cyan

# Count screenshots
$screenshotCount = (Get-ChildItem "$scriptDir\screenshots\*.png" -ErrorAction SilentlyContinue).Count

# Check for comparison report
$comparisonReportPath = "$scriptDir\test-results\comparison-report.json"
$visualComparisonPath = "$scriptDir\test-results\visual-comparison.html"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Screenshots generated: $screenshotCount" -ForegroundColor White
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
Write-Host "  Screenshots: $scriptDir\screenshots\" -ForegroundColor Gray
Write-Host "  Test report: $scriptDir\test-results\html-report\" -ForegroundColor Gray
Write-Host "  Comparison:  $visualComparisonPath" -ForegroundColor Gray

# Open reports
if ($OpenReport) {
    Write-Host ""
    Write-Host "Opening reports..." -ForegroundColor Yellow
    
    if (Test-Path $visualComparisonPath) {
        Start-Process $visualComparisonPath
    }
    
    Start-Process "$scriptDir\test-results\html-report\index.html"
}

Write-Host ""
if ($testExitCode -eq 0) {
    Write-Host "✓ Visual testing complete!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Some tests failed. Review the reports for details." -ForegroundColor Red
    exit 1
}
