# App Extension - Quick Setup for Development
# This creates an alias so you can use "azd App" during development

Write-Host "Setting up App Extension for development..." -ForegroundColor Cyan

# Build the extension
Write-Host "`n📦 Building extension..." -ForegroundColor Yellow
& "$PSScriptRoot\build.ps1"

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host "`n✅ Build complete!" -ForegroundColor Green
Write-Host "`nTo use the extension during development, you have two options:" -ForegroundColor Cyan

Write-Host "`n Option 1: Use directly (Recommended for testing)" -ForegroundColor Yellow
Write-Host "  .\bin\App.exe hi" -ForegroundColor White

Write-Host "`n Option 2: Add a PowerShell function (Use 'azd App')" -ForegroundColor Yellow
Write-Host "  Add this to your PowerShell profile:" -ForegroundColor Gray
Write-Host @"

  function Invoke-AzdApp {
      & "C:\code\Appazdextension\bin\App.exe" `$args
  }
  Set-Alias -Name "azd-App" -Value Invoke-AzdApp

"@ -ForegroundColor White

Write-Host "  Then you can use:" -ForegroundColor Gray
Write-Host "    azd-App hi" -ForegroundColor White

Write-Host "`n Option 3: Quick test in current session" -ForegroundColor Yellow
Write-Host "  Run these commands now:" -ForegroundColor Gray

# Create the function in the current session
$Global:AppPath = "$PSScriptRoot\bin\App.exe"

function Global:Invoke-AzdApp {
    & $Global:AppPath $args
}

Set-Alias -Name "azd-App" -Value Invoke-AzdApp -Scope Global

Write-Host @"

  function Global:Invoke-AzdApp { & "$PSScriptRoot\bin\App.exe" `$args }
  Set-Alias -Name "azd-App" -Value Invoke-AzdApp -Scope Global
  
"@ -ForegroundColor White

Write-Host "`n✅ Function created in current session!" -ForegroundColor Green
Write-Host "`nTry it now:" -ForegroundColor Cyan
Write-Host "  azd-App hi" -ForegroundColor White
Write-Host "`nOr use directly:" -ForegroundColor Cyan  
Write-Host "  .\bin\App.exe hi" -ForegroundColor White
