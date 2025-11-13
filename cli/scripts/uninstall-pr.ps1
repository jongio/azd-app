#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Uninstall a PR build of the azd app extension
.DESCRIPTION
    Uninstalls the PR extension build and removes the PR registry source
.PARAMETER PrNumber
    The PR number (optional - if not provided, removes all PR sources)
.EXAMPLE
    .\uninstall-pr.ps1 -PrNumber 123
.EXAMPLE
    iex "& { $(irm https://raw.githubusercontent.com/jongio/azd-app/main/cli/scripts/uninstall-pr.ps1) } -PrNumber 123"
#>

param(
    [Parameter(Mandatory=$false)]
    [int]$PrNumber
)

$ErrorActionPreference = 'Stop'
$extensionId = "jongio.azd.app"

Write-Host "🗑️  Uninstalling azd app PR build" -ForegroundColor Cyan
Write-Host ""

# Uninstall extension
Write-Host "📦 Removing extension..." -ForegroundColor Gray
azd extension uninstall $extensionId 2>&1 | Out-Null
Write-Host "   ✓" -ForegroundColor DarkGray

# Remove PR registry sources
if ($PrNumber) {
    Write-Host "🔗 Removing PR #$PrNumber registry source..." -ForegroundColor Gray
    azd extension source remove "pr-$PrNumber" 2>&1 | Out-Null
    Write-Host "   ✓" -ForegroundColor DarkGray
} else {
    Write-Host "🔗 Removing all PR registry sources..." -ForegroundColor Gray
    $sources = azd extension source list 2>&1 | Select-String "pr-\d+"
    foreach ($source in $sources) {
        $sourceName = ($source -split '\s+')[0]
        azd extension source remove $sourceName 2>&1 | Out-Null
    }
    Write-Host "   ✓" -ForegroundColor DarkGray
}

# Clean up local registry file if it exists
if (Test-Path "pr-registry.json") {
    Write-Host "🧹 Cleaning up registry file..." -ForegroundColor Gray
    Remove-Item "pr-registry.json"
    Write-Host "   ✓" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "✅ Uninstall complete!" -ForegroundColor Green
Write-Host ""
Write-Host "To install the stable version:" -ForegroundColor Gray
Write-Host "  azd extension install $extensionId" -ForegroundColor White
