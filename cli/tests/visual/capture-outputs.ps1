# Visual Test Runner - Captures terminal output for visual regression testing
# Runs azd deps at different widths and captures output for screenshot comparison

param(
    [int[]]$Widths = @(40, 50, 60, 80, 100, 120, 140),
    [int]$Height = 30,
    [switch]$CleanFirst
)

$ErrorActionPreference = "Stop"

Write-Host "=== Visual Test Output Capture ===" -ForegroundColor Cyan
Write-Host "Terminal widths to test: $($Widths -join ', ')" -ForegroundColor Gray
Write-Host "Terminal height: $Height" -ForegroundColor Gray
Write-Host ""

# Paths
$scriptDir = $PSScriptRoot
$outputDir = Join-Path $scriptDir "output"
$azdPath = "C:\code\azd-app\cli\bin\azd.exe"
$projectPath = "C:\code\azd-app\cli\tests\projects\fullstack-test"

# Create output directory
if (!(Test-Path $outputDir)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
}

# Clean old outputs
if ($CleanFirst) {
    Write-Host "Cleaning old outputs..." -ForegroundColor Yellow
    Remove-Item "$outputDir\*.txt" -Force -ErrorAction SilentlyContinue
}

# Build azd
Write-Host "[1/3] Building azd..." -ForegroundColor Cyan
Push-Location "C:\code\azd-app\cli"
$buildOutput = go build -o bin/azd.exe ./src/cmd/app 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    Write-Host $buildOutput -ForegroundColor Red
    Pop-Location
    exit 1
}
Pop-Location
Write-Host "✓ Build successful" -ForegroundColor Green

# Function to capture output at specific width
function Capture-AtWidth {
    param(
        [int]$Width,
        [int]$Height
    )
    
    $timestamp = Get-Date -Format "HHmmss"
    $outputFile = Join-Path $outputDir "width_${Width}_${timestamp}.txt"
    
    Write-Host "  Testing width $Width..." -ForegroundColor Gray
    
    # Clean node_modules
    Push-Location $projectPath
    Remove-Item -Recurse -Force web\node_modules -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force api\.venv -ErrorAction SilentlyContinue
    
    # Set environment variables for terminal size
    $env:COLUMNS = $Width
    $env:LINES = $Height
    
    # Capture output (raw with ANSI codes)
    try {
        & $azdPath deps --clean 2>&1 | Out-File -FilePath $outputFile -Encoding UTF8
        Write-Host "    ✓ Captured to: $(Split-Path $outputFile -Leaf)" -ForegroundColor Green
    }
    catch {
        Write-Host "    ❌ Error: $_" -ForegroundColor Red
    }
    finally {
        Pop-Location
    }
    
    return $outputFile
}

# Capture outputs at different widths
Write-Host "`n[2/3] Capturing outputs..." -ForegroundColor Cyan

$capturedFiles = @()
foreach ($width in $Widths) {
    $file = Capture-AtWidth -Width $width -Height $Height
    $capturedFiles += @{
        Width = $width
        File = $file
        FileName = Split-Path $file -Leaf
    }
    
    # Small delay between captures
    Start-Sleep -Milliseconds 500
}

# Generate metadata file for Playwright
Write-Host "`n[3/3] Generating test metadata..." -ForegroundColor Cyan

$metadata = @{
    timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    widths = $Widths
    height = $Height
    captures = $capturedFiles
}

$metadataJson = $metadata | ConvertTo-Json -Depth 10
$metadataFile = Join-Path $outputDir "metadata.json"
$metadataJson | Out-File -FilePath $metadataFile -Encoding UTF8

Write-Host "✓ Metadata saved to: $(Split-Path $metadataFile -Leaf)" -ForegroundColor Green

# Summary
Write-Host "`n=== Capture Complete ===" -ForegroundColor Cyan
Write-Host "Captured $($capturedFiles.Count) terminal outputs" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. cd $scriptDir" -ForegroundColor Gray
Write-Host "  2. npm install" -ForegroundColor Gray
Write-Host "  3. npm test" -ForegroundColor Gray
Write-Host ""
Write-Host "This will generate visual screenshots of all captured outputs" -ForegroundColor Yellow
