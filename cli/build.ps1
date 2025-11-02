# Build script for App Extension (Windows)

param(
    [switch]$All = $false
)

$ExtensionName = "app"
$OutputDir = "bin"

Write-Host "Building App Extension..." -ForegroundColor Cyan

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
    go build -o "$OutputDir\windows-amd64\$ExtensionName.exe" .\src\cmd\app
    
    Write-Host "Building for Windows (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\windows-arm64\$ExtensionName.exe" .\src\cmd\app
    
    # Linux
    Write-Host "Building for Linux (AMD64)..." -ForegroundColor Green
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o "$OutputDir\linux-amd64\$ExtensionName" .\src\cmd\app
    
    Write-Host "Building for Linux (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\linux-arm64\$ExtensionName" .\src\cmd\app
    
    # macOS
    Write-Host "Building for macOS (AMD64)..." -ForegroundColor Green
    $env:GOOS = "darwin"
    $env:GOARCH = "amd64"
    go build -o "$OutputDir\darwin-amd64\$ExtensionName" .\src\cmd\app
    
    Write-Host "Building for macOS (ARM64)..." -ForegroundColor Green
    $env:GOARCH = "arm64"
    go build -o "$OutputDir\darwin-arm64\$ExtensionName" .\src\cmd\app
    
    # Reset environment variables
    $env:GOOS = ""
    $env:GOARCH = ""
    
    Write-Host "✅ Build complete for all platforms!" -ForegroundColor Green
} else {
    Write-Host "Building for current platform..." -ForegroundColor Yellow
    go build -o "$OutputDir\$ExtensionName.exe" .\src\cmd\app
    Write-Host "✅ Build complete!" -ForegroundColor Green
}

Write-Host "Binaries are in the '$OutputDir' directory" -ForegroundColor Cyan
