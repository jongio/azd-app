#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Install a PR build of the azd app extension
.DESCRIPTION
    Uninstalls existing extension, downloads PR registry, and installs the PR build
.PARAMETER PrNumber
    The PR number (e.g., 123)
.PARAMETER Version
    The PR version (e.g., 0.5.7-pr123)
.EXAMPLE
    .\install-pr.ps1 -PrNumber 123 -Version 0.5.7-pr123
.EXAMPLE
    iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/scripts/install-pr.ps1) } -PrNumber 123 -Version 0.5.7-pr123"
#>

param(
    [Parameter(Mandatory=$true)]
    [int]$PrNumber,
    
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = 'Stop'
$repo = "jongio/azd-app"
$extensionId = "jongio.azd.app"
$tag = "azd-ext-jongio-azd-app_${Version}"
$registryUrl = "https://github.com/$repo/releases/download/$tag/pr-registry.json"

Write-Host "🚀 Installing azd app PR #$PrNumber (version $Version)" -ForegroundColor Cyan
Write-Host ""

# Step 0: Kill any running extension processes to avoid "file in use" errors
Write-Host "🛑 Stopping any running extension processes..." -ForegroundColor Gray
$processNames = @("jongio-azd-app-windows-amd64", "jongio-azd-app-windows-arm64", "app")
foreach ($name in $processNames) {
    Get-Process -Name $name -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Milliseconds 500  # Give processes time to fully terminate
Write-Host "   ✓" -ForegroundColor DarkGray

# Step 1: Enable extensions
Write-Host "📋 Enabling azd extensions..." -ForegroundColor Gray
azd config set alpha.extension.enabled on
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to enable extensions" -ForegroundColor Red
    exit 1
}

# Step 2: Uninstall existing extension (force remove any version)
Write-Host "🗑️  Uninstalling existing extension (if any)..." -ForegroundColor Gray
azd extension uninstall $extensionId 2>&1 | Out-Null
# Ignore errors - extension might not be installed

# Also manually remove the extension directory to ensure clean install
$extensionDir = Join-Path $env:USERPROFILE ".azd\extensions\$extensionId"
if (Test-Path $extensionDir) {
    Remove-Item -Path $extensionDir -Recurse -Force -ErrorAction SilentlyContinue
}
Write-Host "   ✓" -ForegroundColor DarkGray

# Step 3: Download PR registry
Write-Host "📥 Downloading PR registry..." -ForegroundColor Gray
$registryPath = Join-Path $PWD "pr-registry.json"
try {
    Invoke-WebRequest -Uri $registryUrl -OutFile $registryPath
    Write-Host "   ✓ Downloaded to: $registryPath" -ForegroundColor DarkGray
} catch {
    Write-Host "❌ Failed to download registry from $registryUrl" -ForegroundColor Red
    Write-Host "   Make sure the PR build exists and is accessible" -ForegroundColor Yellow
    exit 1
}

# Step 4: Add registry source
Write-Host "🔗 Adding PR registry source..." -ForegroundColor Gray
azd extension source remove "pr-$PrNumber" 2>$null  # Remove if exists
azd extension source add -n "pr-$PrNumber" -t file -l $registryPath
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to add registry source" -ForegroundColor Red
    exit 1
}

# Step 5: Install PR version
Write-Host "📦 Installing version $Version..." -ForegroundColor Gray

# Clear any cached extension packages to force fresh download
$cacheDir = Join-Path $env:USERPROFILE ".azd\cache"
if (Test-Path $cacheDir) {
    Get-ChildItem -Path $cacheDir -Filter "*$extensionId*" -ErrorAction SilentlyContinue | 
        Remove-Item -Force -ErrorAction SilentlyContinue
}

azd extension install $extensionId --version $Version
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to install extension" -ForegroundColor Red
    exit 1
}

# Step 6: Verify installation
Write-Host ""
Write-Host "✅ Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "🔍 Verifying installation..." -ForegroundColor Gray
$installedVersion = azd app version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   $installedVersion" -ForegroundColor DarkGray
    if ($installedVersion -match $Version) {
        Write-Host ""
        Write-Host "✨ Success! PR build is ready to test." -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "⚠️  Version mismatch - expected $Version" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠️  Could not verify version" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Try these commands:" -ForegroundColor Cyan
Write-Host "  azd app run" -ForegroundColor White
Write-Host "  azd app reqs" -ForegroundColor White
Write-Host ""
Write-Host "To restore stable version, run:" -ForegroundColor Gray
Write-Host "  iex `"& { `$(irm https://raw.githubusercontent.com/$repo/main/cli/scripts/restore-stable.ps1) }`"" -ForegroundColor White
