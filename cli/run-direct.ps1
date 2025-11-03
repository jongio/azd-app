#!/usr/bin/env pwsh
#Requires -Version 7.4
# Run the app directly without installing as an extension
# Useful for quick testing
param(
    [Parameter(Position=0)]
    [string]$Command = "run",
    
    [string]$ProjectDir = "tests\projects\fullstack-test"
)

$ErrorActionPreference = "Stop"

# Get script directory (repo root)
$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $repoRoot

try {
    $binPath = "bin\app.exe"
    
    # Build if needed
    if (-not (Test-Path $binPath) -or ((Get-Item $binPath).LastWriteTime -lt (Get-Date).AddMinutes(-5))) {
        Write-Host "🔨 Building..." -ForegroundColor Cyan
        go build -o $binPath .\src\cmd\app
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ Build failed" -ForegroundColor Red
            exit 1
        }
    }

    # Run directly
    $fullProjectPath = Join-Path $repoRoot $ProjectDir
    Push-Location $fullProjectPath
    try {
        Write-Host "🚀 Running: $binPath $Command" -ForegroundColor Cyan
        Write-Host ""
        & (Join-Path $repoRoot $binPath) $Command
    } finally {
        Pop-Location
    }
} finally {
    Pop-Location
}
