# postrun.ps1 - Windows PowerShell post-startup script

Write-Host "✅ Fullstack application is running!" -ForegroundColor Green
Write-Host "📍 API available at: http://localhost:5000" -ForegroundColor Cyan
Write-Host "📍 Web available at: http://localhost:5001" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press Ctrl+C to stop all services" -ForegroundColor Yellow
exit 0
