# App Extension - Local Install Script
# This script installs the extension locally for azd to use
# Based on the official azd extension installation process

param(
    [switch]$Uninstall = $false
)

$ExtensionId = "jongio.azd.app"
$Namespace = "app"
$Version = "0.1.0"
$BinaryName = "app.exe"
$ProjectRoot = $PSScriptRoot

$AzdConfigPath = "$env:USERPROFILE\.azd\config.json"

if ($Uninstall) {
    Write-Host "Uninstalling App Extension..." -ForegroundColor Yellow
    
    # Remove from config.json
    if (Test-Path $AzdConfigPath) {
        $config = Get-Content $AzdConfigPath -Raw | ConvertFrom-Json
        if ($config.extension.installed.PSObject.Properties.Name -contains $ExtensionId) {
            $config.extension.installed.PSObject.Properties.Remove($ExtensionId)
            $config | ConvertTo-Json -Depth 10 | Set-Content $AzdConfigPath
            Write-Host "✅ Removed from config.json" -ForegroundColor Green
        }
    }
    
    Write-Host "✅ Extension uninstalled!" -ForegroundColor Green
    Write-Host "Note: Run 'azd x build' to rebuild and reinstall" -ForegroundColor Cyan
    exit 0
}

Write-Host "🚀 Installing App Extension using azd developer tools..." -ForegroundColor Cyan

# Use azd x build which handles everything automatically
Write-Host "`n📦 Building and installing with 'azd x build'..." -ForegroundColor Yellow
azd x build

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed!" -ForegroundColor Red
    exit 1
}

# Ensure the extension directory exists and copy files manually
# (azd x build sometimes doesn't copy to the right location)
$ExtensionInstallDir = "$env:USERPROFILE\.azd\extensions\$ExtensionId\$Version"
Write-Host "`n📁 Ensuring extension files are in place..." -ForegroundColor Yellow

if (-not (Test-Path $ExtensionInstallDir)) {
    New-Item -ItemType Directory -Path $ExtensionInstallDir -Force | Out-Null
}

# Copy binary and manifest
Copy-Item "$ProjectRoot\bin\$BinaryName" "$ExtensionInstallDir\$BinaryName" -Force
Copy-Item "$ProjectRoot\extension.yaml" "$ExtensionInstallDir\extension.yaml" -Force

Write-Host "   ✓ Copied $BinaryName to $ExtensionInstallDir" -ForegroundColor Gray
Write-Host "   ✓ Copied extension.yaml" -ForegroundColor Gray

# Now register it in config.json so azd can find it
Write-Host "`n� Registering extension in azd config..." -ForegroundColor Yellow

if (-not (Test-Path $AzdConfigPath)) {
    Write-Host "❌ azd config.json not found at $AzdConfigPath" -ForegroundColor Red
    exit 1
}

$config = Get-Content $AzdConfigPath -Raw | ConvertFrom-Json

# Ensure extension.installed exists
if (-not $config.extension) {
    $config | Add-Member -NotePropertyName "extension" -NotePropertyValue @{} -Force
}
if (-not $config.extension.installed) {
    $config.extension | Add-Member -NotePropertyName "installed" -NotePropertyValue @{} -Force
}

# Register the extension
$extensionEntry = @{
    id = $ExtensionId
    namespace = $Namespace
    capabilities = @("custom-commands")
    displayName = "App Extension"
    description = "A collection of developer productivity commands for Azure Developer CLI"
    version = $Version
    usage = "azd app <command> [options]"
    path = "extensions\$ExtensionId\$Version\$BinaryName"
    source = "local"
}

# Add or update the extension entry
$config.extension.installed | Add-Member -NotePropertyName $ExtensionId -NotePropertyValue $extensionEntry -Force

# Save the config
$config | ConvertTo-Json -Depth 10 | Set-Content $AzdConfigPath

Write-Host "   ✓ Registered in $AzdConfigPath" -ForegroundColor Gray

# Success!
Write-Host "`n✅ App Extension installed successfully!" -ForegroundColor Green
Write-Host "`nTry it now:" -ForegroundColor Cyan
Write-Host "  azd app hi" -ForegroundColor White
Write-Host "`nTo uninstall:" -ForegroundColor Cyan
Write-Host "  .\install-local.ps1 -Uninstall" -ForegroundColor White
Write-Host "`nTo rebuild after changes:" -ForegroundColor Cyan
Write-Host "  azd x build" -ForegroundColor White
Write-Host "  (or run this script again)" -ForegroundColor Gray
