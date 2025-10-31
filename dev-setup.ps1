# DevStack Extension - Quick Setup for Development
# This creates an alias so you can use "azd devstack" during development

Write-Host "Setting up DevStack Extension for development..." -ForegroundColor Cyan

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
Write-Host "  .\bin\devstack.exe hi" -ForegroundColor White

Write-Host "`n Option 2: Add a PowerShell function (Use 'azd devstack')" -ForegroundColor Yellow
Write-Host "  Add this to your PowerShell profile:" -ForegroundColor Gray
Write-Host @"

  function Invoke-AzdDevstack {
      & "C:\code\devstackazdextension\bin\devstack.exe" `$args
  }
  Set-Alias -Name "azd-devstack" -Value Invoke-AzdDevstack

"@ -ForegroundColor White

Write-Host "  Then you can use:" -ForegroundColor Gray
Write-Host "    azd-devstack hi" -ForegroundColor White

Write-Host "`n Option 3: Quick test in current session" -ForegroundColor Yellow
Write-Host "  Run these commands now:" -ForegroundColor Gray

# Create the function in the current session
$Global:DevStackPath = "$PSScriptRoot\bin\devstack.exe"

function Global:Invoke-AzdDevstack {
    & $Global:DevStackPath $args
}

Set-Alias -Name "azd-devstack" -Value Invoke-AzdDevstack -Scope Global

Write-Host @"

  function Global:Invoke-AzdDevstack { & "$PSScriptRoot\bin\devstack.exe" `$args }
  Set-Alias -Name "azd-devstack" -Value Invoke-AzdDevstack -Scope Global
  
"@ -ForegroundColor White

Write-Host "`n✅ Function created in current session!" -ForegroundColor Green
Write-Host "`nTry it now:" -ForegroundColor Cyan
Write-Host "  azd-devstack hi" -ForegroundColor White
Write-Host "`nOr use directly:" -ForegroundColor Cyan  
Write-Host "  .\bin\devstack.exe hi" -ForegroundColor White
