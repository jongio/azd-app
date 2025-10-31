# Build script for DevStack Extension (Windows)

param(
    [switch]$All = $false
)

$ExtensionName = "devstack"
$OutputDir = "bin"

Write-Host "Building DevStack Extension..." -ForegroundColor Cyan

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

if ($All) {
    Write-Host "Building for all platforms..." -ForegroundColor Yellow
    
    # Windows
    Write-Host "Building for Windows (AMD64)..." -ForegroundColor Green
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -o "$OutputDir\windows-amd64\$ExtensionName.exe" .
    
    Write-Host "Building for Windows (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\windows-arm64\$ExtensionName.exe" .
    
    # Linux
    Write-Host "Building for Linux (AMD64)..." -ForegroundColor Green
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o "$OutputDir\linux-amd64\$ExtensionName" .
    
    Write-Host "Building for Linux (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\linux-arm64\$ExtensionName" .
    
    # macOS
    Write-Host "Building for macOS (AMD64)..." -ForegroundColor Green
    $env:GOOS = "darwin"
    $env:GOARCH = "amd64"
    go build -o "$OutputDir\darwin-amd64\$ExtensionName" .
    
    Write-Host "Building for macOS (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\darwin-arm64\$ExtensionName" .
    
    # Reset environment variables
    $env:GOOS = ""
    $env:GOARCH = ""
    
    Write-Host "✅ Build complete for all platforms!" -ForegroundColor Green
} else {
    Write-Host "Building for current platform..." -ForegroundColor Yellow
    go build -o "$OutputDir\$ExtensionName.exe" .
    Write-Host "✅ Build complete!" -ForegroundColor Green
}

Write-Host "Binaries are in the '$OutputDir' directory" -ForegroundColor Cyan
