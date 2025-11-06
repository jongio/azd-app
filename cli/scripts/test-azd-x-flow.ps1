#!/usr/bin/env pwsh
# Test script for azd x release flow
# This simulates what the GitHub Actions workflow will do

param(
    [string]$Version = "0.4.3-test",
    [switch]$SkipBuild,
    [switch]$SkipPack,
    [switch]$SkipPublish
)

$ErrorActionPreference = 'Stop'

Write-Host "🧪 Testing azd x release flow" -ForegroundColor Cyan
Write-Host "Version: $Version" -ForegroundColor Yellow
Write-Host ""

# Check if azd is installed
Write-Host "Checking azd installation..." -ForegroundColor Gray
try {
    $azdVersion = azd version 2>&1
    Write-Host "✅ azd installed: $azdVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ azd not installed. Install from https://aka.ms/azd" -ForegroundColor Red
    exit 1
}

# Check if microsoft.azd.extensions is installed
Write-Host "Checking azd extensions..." -ForegroundColor Gray
try {
    $extensions = azd extension list --output json | ConvertFrom-Json
    $hasExtension = $extensions | Where-Object { $_.id -eq "microsoft.azd.extensions" }
    if ($hasExtension) {
        Write-Host "✅ microsoft.azd.extensions installed" -ForegroundColor Green
    } else {
        Write-Host "Installing microsoft.azd.extensions..." -ForegroundColor Yellow
        azd extension install microsoft.azd.extensions
    }
} catch {
    Write-Host "⚠️  Could not verify extensions" -ForegroundColor Yellow
}

# Navigate to cli directory
$cliDir = Join-Path $PSScriptRoot ".."
Set-Location $cliDir

Write-Host ""
Write-Host "=== Step 1: Build Dashboard ===" -ForegroundColor Cyan
if (Test-Path "dashboard") {
    Set-Location "dashboard"
    if (Test-Path "package-lock.json") {
        Write-Host "Running npm ci..." -ForegroundColor Gray
        npm ci
        Write-Host "Running npm run build..." -ForegroundColor Gray
        npm run build
        Write-Host "✅ Dashboard built" -ForegroundColor Green
    } else {
        Write-Host "⚠️  No package-lock.json found, skipping dashboard build" -ForegroundColor Yellow
    }
    Set-Location ".."
} else {
    Write-Host "⚠️  No dashboard directory found" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Step 2: Build Extension Binaries ===" -ForegroundColor Cyan
if (-not $SkipBuild) {
    $env:EXTENSION_ID = "jongio.azd.app"
    $env:EXTENSION_VERSION = $Version
    
    Write-Host "Building for all platforms..." -ForegroundColor Gray
    Write-Host "  EXTENSION_ID=$env:EXTENSION_ID" -ForegroundColor DarkGray
    Write-Host "  EXTENSION_VERSION=$env:EXTENSION_VERSION" -ForegroundColor DarkGray
    
    azd x build --all
    
    if (Test-Path "bin") {
        $binaries = Get-ChildItem "bin" -Filter "jongio-azd-app-*"
        Write-Host "✅ Built $($binaries.Count) binaries:" -ForegroundColor Green
        $binaries | ForEach-Object { Write-Host "   - $($_.Name)" -ForegroundColor DarkGray }
    } else {
        Write-Host "❌ No binaries found in bin/" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "⏭️  Skipped (--SkipBuild)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Step 3: Package Extension ===" -ForegroundColor Cyan
if (-not $SkipPack) {
    Write-Host "Packaging extension..." -ForegroundColor Gray
    azd x pack
    
    $registryPath = Join-Path $env:USERPROFILE ".azd\registry\jongio.azd.app\$Version"
    if (Test-Path $registryPath) {
        $archives = Get-ChildItem $registryPath -Filter "*.zip","*.tar.gz"
        Write-Host "✅ Packaged $($archives.Count) archives:" -ForegroundColor Green
        $archives | ForEach-Object { Write-Host "   - $($_.Name)" -ForegroundColor DarkGray }
        Write-Host "   Location: $registryPath" -ForegroundColor DarkGray
    } else {
        Write-Host "❌ No packages found in $registryPath" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "⏭️  Skipped (--SkipPack)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Step 4: Update Registry (Local Mode) ===" -ForegroundColor Cyan
if (-not $SkipPublish) {
    $registryFile = Join-Path $cliDir "..\registry.json"
    Write-Host "Publishing to $registryFile..." -ForegroundColor Gray
    
    # Backup registry.json
    $backupFile = "$registryFile.backup"
    Copy-Item $registryFile $backupFile -Force
    Write-Host "   Backed up to $backupFile" -ForegroundColor DarkGray
    
    try {
        azd x publish --registry "../registry.json" --version $Version
        
        if (Test-Path $registryFile) {
            $registry = Get-Content $registryFile -Raw | ConvertFrom-Json
            $extension = $registry.extensions.'jongio.azd.app'
            $versionEntry = $extension.versions | Where-Object { $_.version -eq $Version }
            
            if ($versionEntry) {
                Write-Host "✅ Registry updated with version $Version" -ForegroundColor Green
                Write-Host "   Artifacts:" -ForegroundColor DarkGray
                $versionEntry.artifacts.PSObject.Properties | ForEach-Object {
                    Write-Host "     - $($_.Name): $($_.Value.url)" -ForegroundColor DarkGray
                }
            } else {
                Write-Host "⚠️  Version $Version not found in registry" -ForegroundColor Yellow
            }
        }
    } catch {
        Write-Host "❌ Failed to publish: $_" -ForegroundColor Red
        # Restore backup
        Copy-Item $backupFile $registryFile -Force
        Write-Host "   Restored registry.json from backup" -ForegroundColor Yellow
        exit 1
    }
} else {
    Write-Host "⏭️  Skipped (--SkipPublish)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Test Summary ===" -ForegroundColor Cyan
Write-Host "✅ All steps completed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps for full workflow test:" -ForegroundColor Yellow
Write-Host "1. Commit and push the updated release.yml workflow" -ForegroundColor Gray
Write-Host "2. Run 'Prepare Release' workflow to create version bump PR" -ForegroundColor Gray
Write-Host "3. Merge the release PR" -ForegroundColor Gray
Write-Host "4. Run 'Release' workflow with the version number" -ForegroundColor Gray
Write-Host ""
Write-Host "To clean up this test:" -ForegroundColor Yellow
Write-Host "  - Restore registry.json: Copy-Item registry.json.backup registry.json -Force" -ForegroundColor Gray
Write-Host "  - Remove test artifacts: Remove-Item ~\.azd\registry\jongio.azd.app\$Version -Recurse -Force" -ForegroundColor Gray
