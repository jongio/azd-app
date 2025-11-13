#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Restore the stable version of azd app extension
.DESCRIPTION
    Uninstalls PR build, removes PR registries, and installs latest stable version
.EXAMPLE
    .\restore-stable.ps1
.EXAMPLE
    iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/scripts/restore-stable.ps1) }"
#>

$ErrorActionPreference = 'Stop'
$repo = "jongio/azd-app"
$extensionId = "jongio.azd.app"
$stableRegistryUrl = "https://raw.githubusercontent.com/$repo/refs/heads/main/registry.json"

Write-Host "🔄 Restoring stable azd app extension" -ForegroundColor Cyan
Write-Host ""

# Step 1: Uninstall current extension
Write-Host "🗑️  Uninstalling current extension..." -ForegroundColor Gray
azd extension uninstall $extensionId 2>$null
# Ignore errors - extension might not be installed

# Step 2: Remove all PR registry sources
Write-Host "🧹 Removing PR registry sources..." -ForegroundColor Gray
$sources = azd extension source list --output json 2>$null | ConvertFrom-Json
if ($sources) {
    foreach ($source in $sources) {
        if ($source.name -match '^pr-\d+$') {
            Write-Host "   Removing: $($source.name)" -ForegroundColor DarkGray
            azd extension source remove $source.name 2>$null
        }
    }
}

# Step 3: Clean up pr-registry.json files
Write-Host "🧹 Cleaning up pr-registry.json files..." -ForegroundColor Gray
Get-ChildItem -Path $PWD -Filter "pr-registry.json" -ErrorAction SilentlyContinue | Remove-Item -Force
if (Test-Path "$HOME/pr-registry.json") {
    Remove-Item "$HOME/pr-registry.json" -Force
}

# Step 4: Add stable registry source
Write-Host "🔗 Adding stable registry source..." -ForegroundColor Gray
azd extension source remove "app" 2>$null  # Remove if exists
azd extension source add -n "app" -t url -l $stableRegistryUrl
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to add stable registry source" -ForegroundColor Red
    exit 1
}

# Step 5: Install latest stable version
Write-Host "📦 Installing latest stable version..." -ForegroundColor Gray
azd extension install $extensionId
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to install stable extension" -ForegroundColor Red
    exit 1
}

# Step 6: Verify installation
Write-Host ""
Write-Host "✅ Restoration complete!" -ForegroundColor Green
Write-Host ""
Write-Host "🔍 Verifying installation..." -ForegroundColor Gray
$installedVersion = azd app version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   $installedVersion" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "✨ Success! Stable version is installed." -ForegroundColor Green
} else {
    Write-Host "⚠️  Could not verify version" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "You can now use azd app normally:" -ForegroundColor Cyan
Write-Host "  azd app hi" -ForegroundColor White
Write-Host "  azd app reqs" -ForegroundColor White
Write-Host "  azd app run" -ForegroundColor White
