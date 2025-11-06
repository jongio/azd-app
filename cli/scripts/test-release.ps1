#!/usr/bin/env pwsh
# Quick test script for GoReleaser configuration
# This runs GoReleaser in snapshot mode (no actual release)

Write-Host "Testing GoReleaser configuration..." -ForegroundColor Cyan

# Check if goreleaser is installed
if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) {
    Write-Host "Installing GoReleaser..." -ForegroundColor Yellow
    go install github.com/goreleaser/goreleaser/v2@latest
}

# Build dashboard first
Write-Host "Building dashboard..." -ForegroundColor Cyan
Push-Location cli/dashboard
npm ci
npm run build
Pop-Location

# Run goreleaser in snapshot mode (doesn't create a release)
Write-Host "Running GoReleaser in snapshot mode..." -ForegroundColor Cyan
goreleaser release --snapshot --clean --skip=publish -f cli/.goreleaser.yml

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✓ GoReleaser test successful!" -ForegroundColor Green
    Write-Host "Check the 'dist' directory for generated artifacts" -ForegroundColor Green
} else {
    Write-Host "`n✗ GoReleaser test failed" -ForegroundColor Red
    exit 1
}
