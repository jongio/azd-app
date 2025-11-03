#!/usr/bin/env pwsh
#Requires -Version 7.4
# Watch for changes and auto-rebuild
param(
    [string]$Path = "src"
)

# Get script directory (repo root)
$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$watchPath = Join-Path $repoRoot $Path

Write-Host "👀 Watching $watchPath for changes..." -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop" -ForegroundColor Yellow
Write-Host ""

$lastBuild = Get-Date

while ($true) {
    $changed = Get-ChildItem -Path $watchPath -Recurse -Filter "*.go" | 
        Where-Object { $_.LastWriteTime -gt $lastBuild }
    
    if ($changed) {
        $lastBuild = Get-Date
        Write-Host "[$(Get-Date -Format 'HH:mm:ss')] Changes detected, rebuilding..." -ForegroundColor Cyan
        
        & "$repoRoot\quick-install.ps1"
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[$(Get-Date -Format 'HH:mm:ss')] ✅ Ready!" -ForegroundColor Green
            Write-Host ""
        }
    }
    
    Start-Sleep -Milliseconds 500
}
