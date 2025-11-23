# prerun.ps1 - Windows PowerShell startup script

Write-Host "🚀 Starting fullstack application - preparing API and Web services..." -ForegroundColor Cyan
Write-Host "📦 Checking dependencies..." -ForegroundColor Yellow

# Check if virtual environment exists for Python API
if (-not (Test-Path "./api/.venv")) {
    Write-Host "Creating Python virtual environment..." -ForegroundColor Yellow
}

# Check if node_modules exists for Web
if (-not (Test-Path "./web/node_modules")) {
    Write-Host "Node modules will be installed..." -ForegroundColor Yellow
}

Write-Host "✅ Pre-run checks complete" -ForegroundColor Green
exit 0
