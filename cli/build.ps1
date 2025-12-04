#!/usr/bin/env pwsh
# Build script called by azd x build
# This handles pre-build steps like dashboard compilation

$ErrorActionPreference = 'Stop'

# Get the directory of the script (cli folder)
$EXTENSION_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path

# Change to the script directory
Set-Location -Path $EXTENSION_DIR

# Helper function to kill extension processes
# This is necessary on Windows where binaries cannot be overwritten while in use
function Stop-ExtensionProcesses {
    $binaryName = "app"
    $extensionId = "jongio.azd.app"
    $extensionBinaryPrefix = $extensionId -replace '\.', '-'

    # Kill processes by name silently (ignore errors if not running)
    taskkill /F /IM "$binaryName.exe" 2>$null | Out-Null
    foreach ($arch in @("windows-amd64", "windows-arm64")) {
        $procName = "$extensionBinaryPrefix-$arch.exe"
        taskkill /F /IM $procName 2>$null | Out-Null
    }
    
    # Also kill any processes running from the installed extension directory
    # This catches processes that azd x watch/install started
    $installedExtensionDir = Join-Path $env:USERPROFILE ".azd\extensions\$extensionId"
    if (Test-Path $installedExtensionDir) {
        Get-Process | Where-Object { 
            $_.Path -and $_.Path.StartsWith($installedExtensionDir) 
        } | ForEach-Object {
            Write-Host "  Stopping process: $($_.Name) (PID: $($_.Id))" -ForegroundColor Gray
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
        }
    }
    
    # Give processes time to fully terminate and release file handles
    Start-Sleep -Milliseconds 500
}

# Check if we need to rebuild the Go binary
# This happens when: Go files changed OR embedded dashboard dist changed
$needsGoBuild = $false
$existingBinaries = Get-ChildItem -Path "bin" -Filter "*.exe" -ErrorAction SilentlyContinue | Where-Object { $_.Name -notlike "*.old" }

if (-not $existingBinaries) {
    # No binary exists, definitely need to build
    $needsGoBuild = $true
    Write-Host "No existing binary found, will build" -ForegroundColor Yellow
} else {
    $newestBinary = $existingBinaries | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    $binaryTime = $newestBinary.LastWriteTime
    
    # Check Go source files
    $goFiles = Get-ChildItem -Path "src" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue
    if ($goFiles) {
        $newestGoFile = $goFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1
        if ($newestGoFile.LastWriteTime -gt $binaryTime) {
            $needsGoBuild = $true
            Write-Host "Go source files changed, will rebuild" -ForegroundColor Yellow
        }
    }
    
    # Check embedded dashboard dist (it's compiled into the binary)
    $dashboardDistPath = "src\internal\dashboard\dist"
    if (Test-Path $dashboardDistPath) {
        $distFiles = Get-ChildItem -Path $dashboardDistPath -Recurse -File -ErrorAction SilentlyContinue
        if ($distFiles) {
            $newestDistFile = $distFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1
            if ($newestDistFile.LastWriteTime -gt $binaryTime) {
                $needsGoBuild = $true
                Write-Host "Embedded dashboard dist changed, will rebuild" -ForegroundColor Yellow
            }
        }
    }
}

# Only kill extension processes if we're actually going to rebuild the binary
# This prevents unnecessary restarts when dashboard source changes (vite watcher handles that separately)
if ($needsGoBuild) {
    Write-Host "Stopping extension processes before rebuild..." -ForegroundColor Yellow
    Stop-ExtensionProcesses
} else {
    # Nothing to rebuild - exit early to prevent azd x watch from trying to install
    Write-Host "  ✓ Binary up to date, skipping build" -ForegroundColor Green
    exit 0
}

Write-Host "Building App Extension..." -ForegroundColor Cyan

# Build dashboard first (if needed)
$dashboardDistPath = "src\internal\dashboard\dist"
$dashboardSrcPath = "dashboard\src"

$shouldBuildDashboard = $false

if (-not (Test-Path $dashboardDistPath)) {
    $shouldBuildDashboard = $true
    Write-Host "Dashboard not built yet" -ForegroundColor Yellow
}
elseif (Test-Path $dashboardSrcPath) {
    $distTime = (Get-Item $dashboardDistPath).LastWriteTime
    $srcFiles = Get-ChildItem $dashboardSrcPath -Recurse -File
    $newestSrc = ($srcFiles | Sort-Object LastWriteTime -Descending | Select-Object -First 1).LastWriteTime
    
    if ($newestSrc -gt $distTime) {
        $shouldBuildDashboard = $true
        Write-Host "Dashboard source changed, rebuild needed" -ForegroundColor Yellow
    }
}

if ($shouldBuildDashboard) {
    Write-Host "Building dashboard..." -ForegroundColor Yellow
    Push-Location "dashboard"
    try {
        if (-not (Test-Path "node_modules")) {
            Write-Host "  Installing dashboard dependencies..." -ForegroundColor Gray
            npm install --silent
            if ($LASTEXITCODE -ne 0) {
                Write-Host "ERROR: npm install failed" -ForegroundColor Red
                exit 1
            }
        }

        Write-Host "  Building dashboard bundle..." -ForegroundColor Gray
        npm run build --silent
        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: Dashboard build failed" -ForegroundColor Red
            exit 1
        }
        Write-Host "  ✓ Dashboard built successfully" -ForegroundColor Green
    } finally {
        Pop-Location
    }
} else {
    Write-Host "  ✓ Dashboard up to date" -ForegroundColor Green
}

# Create a safe version of EXTENSION_ID replacing dots with dashes
$EXTENSION_ID_SAFE = $env:EXTENSION_ID -replace '\.', '-'

# Define output directory
$OUTPUT_DIR = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { Join-Path $EXTENSION_DIR "bin" }

# Create output directory if it doesn't exist
if (-not (Test-Path -Path $OUTPUT_DIR)) {
    New-Item -ItemType Directory -Path $OUTPUT_DIR | Out-Null
}

# Get Git commit hash and build date
try {
    $COMMIT = git rev-parse HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { $COMMIT = "unknown" }
} catch {
    $COMMIT = "unknown"
}
$BUILD_DATE = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

# Read version from version.txt if EXTENSION_VERSION not set
if (-not $env:EXTENSION_VERSION) {
    if (Test-Path "extension.yaml") {
        $yamlContent = Get-Content "extension.yaml" -Raw
        if ($yamlContent -match 'version:\s*(\S+)') {
            $env:EXTENSION_VERSION = $matches[1]
        } else {
            $env:EXTENSION_VERSION = "0.0.0-dev"
        }
    } else {
        $env:EXTENSION_VERSION = "0.0.0-dev"
    }
}

Write-Host "Building version $env:EXTENSION_VERSION" -ForegroundColor Cyan

# List of OS and architecture combinations
if ($env:EXTENSION_PLATFORM) {
    $PLATFORMS = @($env:EXTENSION_PLATFORM)
}
else {
    $PLATFORMS = @(
        "windows/amd64",
        "windows/arm64",
        "darwin/amd64",
        "darwin/arm64",
        "linux/amd64",
        "linux/arm64"
    )
}

$APP_PATH = "github.com/jongio/azd-app/cli/src/cmd/app/commands"

# Loop through platforms and build
foreach ($PLATFORM in $PLATFORMS) {
    $OS, $ARCH = $PLATFORM -split '/'

    $OUTPUT_NAME = Join-Path $OUTPUT_DIR "$EXTENSION_ID_SAFE-$OS-$ARCH"

    if ($OS -eq "windows") {
        $OUTPUT_NAME += ".exe"
    }

    Write-Host "  Building for $OS/$ARCH..." -ForegroundColor Gray

    # Handle locked files on Windows by renaming instead of deleting
    if (Test-Path -Path $OUTPUT_NAME) {
        $backupName = "$OUTPUT_NAME.old"
        try {
            # Try to remove old backup first
            if (Test-Path -Path $backupName) {
                Remove-Item -Path $backupName -Force -ErrorAction SilentlyContinue
            }
            # Rename current file (works even if running)
            Move-Item -Path $OUTPUT_NAME -Destination $backupName -Force -ErrorAction Stop
        } catch {
            # If rename fails, file might not be locked - try direct delete
            Remove-Item -Path $OUTPUT_NAME -Force -ErrorAction SilentlyContinue
        }
    }

    # Set environment variables for Go build
    $env:GOOS = $OS
    $env:GOARCH = $ARCH

    $ldflags = "-s -w -X '$APP_PATH.Version=$env:EXTENSION_VERSION' -X '$APP_PATH.BuildTime=$BUILD_DATE' -X '$APP_PATH.Commit=$COMMIT'"

    go build `
        "-ldflags=$ldflags" `
        -o $OUTPUT_NAME `
        ./src/cmd/app

    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Build failed for $OS/$ARCH" -ForegroundColor Red
        exit 1
    }
}

Write-Host "`n✓ Build completed successfully!" -ForegroundColor Green
Write-Host "  Binaries are located in the $OUTPUT_DIR directory." -ForegroundColor Gray
