# App Extension - Local Install Script
# This script installs the extension locally for azd to use
# Based on the official azd extension installation process
#Requires -Version 7.4

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

# Build the dashboard first
Write-Host "`n📊 Building dashboard..." -ForegroundColor Yellow
Push-Location "$ProjectRoot\dashboard"
try {
    # Check if node_modules exists, if not run npm install
    if (-not (Test-Path "node_modules")) {
        Write-Host "   Installing dashboard dependencies..." -ForegroundColor Gray
        npm install
        if ($LASTEXITCODE -ne 0) {
            Write-Host "❌ npm install failed!" -ForegroundColor Red
            Pop-Location
            exit 1
        }
    }
    
    # Build the dashboard
    Write-Host "   Building dashboard bundle..." -ForegroundColor Gray
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Dashboard build failed!" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Write-Host "   ✓ Dashboard built successfully" -ForegroundColor Green
} finally {
    Pop-Location
}

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
try {
    # Try to copy the binary
    Copy-Item "$ProjectRoot\bin\$BinaryName" "$ExtensionInstallDir\$BinaryName" -Force -ErrorAction Stop
    Write-Host "   ✓ Copied $BinaryName to $ExtensionInstallDir" -ForegroundColor Gray
} catch {
    Write-Host "❌ Failed to copy $BinaryName - file is in use by another process" -ForegroundColor Red
    Write-Host "   Attempting to force close handles..." -ForegroundColor Yellow
    
    # Try to kill any processes using the file
    $targetFile = "$ExtensionInstallDir\$BinaryName"
    
    # Method 1: Try to find and kill processes with the file open using handle.exe if available
    $handleExe = Get-Command "handle.exe" -ErrorAction SilentlyContinue
    if ($handleExe) {
        Write-Host "   Using handle.exe to find processes..." -ForegroundColor Gray
        try {
            $handleOutput = & handle.exe -accepteula $targetFile 2>&1
            if ($handleOutput -match "process ID:\s+(\d+)|pid:\s+(\d+)") {
                $processIds = $handleOutput | ForEach-Object { 
                    if ($_ -match "process ID:\s+(\d+)") { 
                        $matches[1] 
                    } elseif ($_ -match "pid:\s+(\d+)") {
                        $matches[1]
                    }
                }
                foreach ($processId in $processIds) {
                    Write-Host "   Killing process PID $processId..." -ForegroundColor Yellow
                    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
                }
            }
        } catch {
            Write-Host "   handle.exe failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    
    # Method 2: Kill any azd or app processes that might be using the file
    Write-Host "   Killing azd and app processes..." -ForegroundColor Yellow
    Get-Process | Where-Object {$_.ProcessName -like "*azd*" -or $_.ProcessName -like "*app*"} | Stop-Process -Force -ErrorAction SilentlyContinue
    
    # Method 3: Try to delete the existing file if it exists
    if (Test-Path $targetFile) {
        Write-Host "   Removing existing file..." -ForegroundColor Yellow
        try {
            Remove-Item $targetFile -Force -ErrorAction Stop
        } catch {
            Write-Host "❌ Still cannot remove existing file: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    
    # Wait a moment for processes to fully exit
    Start-Sleep -Seconds 2
    
    # Try copying again
    Write-Host "   Retrying copy..." -ForegroundColor Yellow
    try {
        Copy-Item "$ProjectRoot\bin\$BinaryName" "$ExtensionInstallDir\$BinaryName" -Force -ErrorAction Stop
        Write-Host "   ✓ Copied $BinaryName to $ExtensionInstallDir (after cleanup)" -ForegroundColor Green
    } catch {
        Write-Host "❌ Copy failed even after cleanup: $($_.Exception.Message)" -ForegroundColor Red
        Write-Host "   Please manually close any azd or app processes and try again." -ForegroundColor Red
        exit 1
    }
}

# Copy extension.yaml (this should not have the same issue)
try {
    Copy-Item "$ProjectRoot\extension.yaml" "$ExtensionInstallDir\extension.yaml" -Force -ErrorAction Stop
    Write-Host "   ✓ Copied extension.yaml" -ForegroundColor Gray
} catch {
    Write-Host "❌ Failed to copy extension.yaml: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

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
