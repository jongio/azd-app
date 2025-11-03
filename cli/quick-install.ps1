#!/usr/bin/env pwsh
#Requires -Version 7.4
# Quick install script - just builds and copies, skips azd x build
param(
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

# Get script directory (repo root)
$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $repoRoot

try {
    $extensionDir = "$env:USERPROFILE\.azd\extensions\jongio.azd.app\0.1.0"

    if (-not $SkipBuild) {
        Write-Host "🔨 Building..." -ForegroundColor Cyan
        go build -o bin\app.exe .\src\cmd\app
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Build failed" -ForegroundColor Red
            exit 1
        }
        }

    Write-Host "📦 Copying to extension directory..." -ForegroundColor Cyan

    # Kill any running processes
    Get-Process -Name "app","azd" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 200

    # Ensure directory exists
    if (-not (Test-Path $extensionDir)) {
        New-Item -ItemType Directory -Path $extensionDir -Force | Out-Null
    }

    # Copy files
    # Copy files
    Copy-Item -Path "bin\app.exe" -Destination $extensionDir -Force
    Copy-Item -Path "extension.yaml" -Destination $extensionDir -Force

    Write-Host "✅ Quick install complete!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Test it: cd tests\projects\fullstack-test; azd app run" -ForegroundColor Yellow
} finally {
    Pop-Location
}
