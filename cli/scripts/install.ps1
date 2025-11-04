#!/usr/bin/env pwsh
#Requires -Version 7.4
# Install script - installs the extension locally for azd to use
param(
    [switch]$Uninstall = $false
)

$ErrorActionPreference = "Stop"

$ExtensionId = "jongio.azd.app"
$Namespace = "app"
$Version = "0.1.0"
$BinaryName = "app.exe"

# Get script directory (scripts folder) and navigate to cli root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $scriptDir

$AzdConfigPath = "$env:USERPROFILE\.azd\config.json"
$ExtensionInstallDir = "$env:USERPROFILE\.azd\extensions\$ExtensionId\$Version"

Push-Location $ProjectRoot

try {
    if ($Uninstall) {
        Write-Host "Uninstalling App Extension..." -ForegroundColor Yellow

        # Remove from config.json
        if (Test-Path $AzdConfigPath) {
            $config = Get-Content $AzdConfigPath -Raw | ConvertFrom-Json
            if ($config.extension.installed.PSObject.Properties.Name -contains $ExtensionId) {
                $config.extension.installed.PSObject.Properties.Remove($ExtensionId)
                $config | ConvertTo-Json -Depth 10 | Set-Content $AzdConfigPath
                Write-Host "   ✓ Removed from config.json" -ForegroundColor Gray
            }
        }

        # Remove extension files
        $extensionBaseDir = "$env:USERPROFILE\.azd\extensions\$ExtensionId"
        if (Test-Path $extensionBaseDir) {
            Remove-Item $extensionBaseDir -Recurse -Force
            Write-Host "   ✓ Removed extension files" -ForegroundColor Gray
        }

        Write-Host "`n✅ Extension uninstalled!" -ForegroundColor Green
        exit 0
    }

    Write-Host "🚀 Installing App Extension..." -ForegroundColor Cyan

    # Smart dashboard detection - only rebuild if source files changed
    $shouldBuildDashboard = $false
    $dashboardDistPath = "src\internal\dashboard\dist"
    $dashboardSrcPath = "dashboard\src"
    
    if (-not (Test-Path $dashboardDistPath)) {
        $shouldBuildDashboard = $true
        Write-Host "📊 Dashboard not built yet" -ForegroundColor Yellow
    }
    elseif (Test-Path $dashboardSrcPath) {
        $distTime = (Get-Item $dashboardDistPath).LastWriteTime
        $srcFiles = Get-ChildItem $dashboardSrcPath -Recurse -File
        $newestSrc = ($srcFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1).LastWriteTime
        
        if ($newestSrc -gt $distTime) {
            $shouldBuildDashboard = $true
            Write-Host "📊 Dashboard source changed, rebuild needed" -ForegroundColor Yellow
        }
        else {
            Write-Host "📊 Dashboard up to date, skipping build" -ForegroundColor Green
        }
    }

    # Build dashboard if needed
    if ($shouldBuildDashboard) {
        Write-Host "`n📊 Building dashboard..." -ForegroundColor Yellow
        Push-Location "dashboard"
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
    }

    # Use azd x build which handles everything automatically
    Write-Host "`n📦 Building and installing with 'azd x build'..." -ForegroundColor Yellow
    azd x build

    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed!" -ForegroundColor Red
        exit 1
    }

    # Ensure the extension directory exists and copy files
    Write-Host "`n📁 Installing extension files..." -ForegroundColor Yellow

    # Kill any running processes
    Get-Process -Name "app","azd" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 200

    if (-not (Test-Path $ExtensionInstallDir)) {
        New-Item -ItemType Directory -Path $ExtensionInstallDir -Force | Out-Null
    }

    # Copy binary and manifest
    try {
        Copy-Item "bin\$BinaryName" "$ExtensionInstallDir\$BinaryName" -Force -ErrorAction Stop
        Copy-Item "extension.yaml" "$ExtensionInstallDir\extension.yaml" -Force -ErrorAction Stop
        Write-Host "   ✓ Copied extension files" -ForegroundColor Gray
    } catch {
        Write-Host "❌ Failed to copy files - $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Register in config.json
    Write-Host "`n📝 Registering extension in azd config..." -ForegroundColor Yellow

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

    $config.extension.installed | Add-Member -NotePropertyName $ExtensionId -NotePropertyValue $extensionEntry -Force
    $config | ConvertTo-Json -Depth 10 | Set-Content $AzdConfigPath

    Write-Host "   ✓ Registered in $AzdConfigPath" -ForegroundColor Gray

    # Activate the extension via azd extension install
    Write-Host "`n🔧 Activating extension..." -ForegroundColor Yellow
    azd extension install $ExtensionId *> $null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   ✓ Extension activated" -ForegroundColor Gray
    }

    Write-Host "`n✅ Extension installed successfully!" -ForegroundColor Green

    Write-Host "`nTry it now:" -ForegroundColor Cyan
    Write-Host "  azd app version" -ForegroundColor White
} finally {
    Pop-Location
}
