<#
.SYNOPSIS
    Automated test for container auth (local.credentials: azd in azure.yaml).
.DESCRIPTION
    Tests the full container credential forwarding flow end-to-end:
    1. Builds the azd CLI shim for Linux
    2. Starts the mTLS auth server
    3. Runs a .NET SDK container with shim + certs injected
    4. Validates: shim mounted, certs present, env vars set
    5. Validates: token acquisition via DefaultAzureCredential
    6. Validates: Azure ARM API call from inside container (lists resource groups)
    7. Cleans up all resources
.NOTES
    Prerequisites: Go 1.22+, Docker, `azd auth login`
    Run from: cli/tests/projects/integration/container-auth-test/
#>

param(
    [switch]$SkipAzure,      # Skip tests that require Azure subscription
    [switch]$KeepRunning,    # Don't clean up (for debugging)
    [switch]$Verbose         # Show detailed output
)

$ErrorActionPreference = "Stop"
$TestDir = $PSScriptRoot
$RepoRoot = (Resolve-Path (Join-Path $TestDir "..\..\..\..\..")).Path
$CliDir = Join-Path $RepoRoot "cli"
$ShimSrcDir = Join-Path $CliDir "src\internal\containerauth\shim"

# Track PIDs and containers for cleanup
$script:authServerPid = $null
$script:containerName = "azd-container-auth-test"
$script:certsDir = $null
$script:shimPath = $null
$script:passed = 0
$script:failed = 0
$script:skipped = 0

function Write-TestHeader {
    Write-Host ""
    Write-Host "================================================" -ForegroundColor Cyan
    Write-Host "  Container Auth Integration Test" -ForegroundColor Cyan
    Write-Host "================================================" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Step([string]$step, [string]$desc) {
    Write-Host "[$step] $desc" -ForegroundColor Yellow
}

function Write-Pass([string]$msg) {
    $script:passed++
    Write-Host "  ✓ PASS: $msg" -ForegroundColor Green
}

function Write-Fail([string]$msg) {
    $script:failed++
    Write-Host "  ✗ FAIL: $msg" -ForegroundColor Red
}

function Write-Skip([string]$msg) {
    $script:skipped++
    Write-Host "  ○ SKIP: $msg" -ForegroundColor DarkYellow
}

function Write-Detail([string]$msg) {
    if ($Verbose) { Write-Host "  $msg" -ForegroundColor DarkGray }
}

function Cleanup {
    Write-Host ""
    Write-Step "CLEANUP" "Stopping resources..."

    # Stop container
    $ErrorActionPreference = "Continue"
    docker stop $script:containerName 2>&1 | Out-Null
    docker rm $script:containerName 2>&1 | Out-Null
    $ErrorActionPreference = "Stop"
    Write-Detail "Container removed"

    # Stop auth server
    if ($script:authServerPid) {
        Stop-Process -Id $script:authServerPid -Force -ErrorAction SilentlyContinue
        Write-Detail "Auth server stopped (PID $($script:authServerPid))"
    }

    # Clean temp files
    if ($script:certsDir -and (Test-Path $script:certsDir)) {
        Remove-Item $script:certsDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($script:shimPath -and (Test-Path $script:shimPath)) {
        $shimDir = Split-Path $script:shimPath -Parent
        Remove-Item $shimDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    Write-Detail "Temp files cleaned"
}

# Register cleanup on exit
$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Cleanup } -ErrorAction SilentlyContinue
trap { Cleanup; break }

# ============================================================
Write-TestHeader

# --- Step 1: Verify prerequisites ---
Write-Step "1/7" "Checking prerequisites..."

# Check Docker
$dockerAvailable = $false
try {
    docker info 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) { $dockerAvailable = $true }
} catch {}

if (-not $dockerAvailable) {
    Write-Fail "Docker is not running"
    Write-Host "  Please start Docker Desktop and try again." -ForegroundColor Red
    exit 1
}
Write-Pass "Docker is running"

# Check Go
$goAvailable = $false
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -eq 0) { $goAvailable = $true }
} catch {}

if (-not $goAvailable) {
    Write-Fail "Go is not installed"
    exit 1
}
Write-Pass "Go is available ($goVersion)"

# Check azd auth
$azdAuthOk = $false
try {
    $null = azd auth token --scope "https://management.azure.com/.default" --output json 2>&1
    if ($LASTEXITCODE -eq 0) { $azdAuthOk = $true }
} catch {}

if ($azdAuthOk) {
    Write-Pass "azd auth is logged in"
} else {
    Write-Host "  WARNING: azd auth token failed. Token/Azure tests will fail." -ForegroundColor DarkYellow
    Write-Host "  Run 'azd auth login' to enable full testing." -ForegroundColor DarkYellow
}

# --- Step 2: Build shim binary ---
Write-Step "2/7" "Building azd auth shim for Linux..."

# Detect architecture
$runningOnMac = $IsMacOS -or (Test-Path "/usr/bin/sw_vers" -ErrorAction SilentlyContinue)
$shimArch = "amd64"
if ($runningOnMac) {
    $hostArch = if (Test-Path "/usr/bin/uname") { & uname -m } else { "x86_64" }
    if ($hostArch -eq "arm64") { $shimArch = "arm64" }
}
if ($env:AZD_SHIM_ARCH) { $shimArch = $env:AZD_SHIM_ARCH }

$shimTmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "azd-auth-test-shim-$(Get-Random)"
New-Item -ItemType Directory -Path $shimTmpDir -Force | Out-Null
$script:shimPath = Join-Path $shimTmpDir "azd"

$env:GOOS = "linux"
$env:GOARCH = $shimArch
$env:CGO_ENABLED = "0"
$env:GOWORK = "off"
Push-Location $ShimSrcDir
go build -ldflags="-s -w" -o $script:shimPath . 2>&1
$buildResult = $LASTEXITCODE
Pop-Location
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
Remove-Item Env:GOWORK -ErrorAction SilentlyContinue

if ($buildResult -ne 0) {
    Write-Fail "Failed to build shim binary"
    exit 1
}
Write-Pass "Shim built for linux/$shimArch at $($script:shimPath)"

# --- Step 3: Start mTLS auth server ---
Write-Step "3/7" "Starting mTLS auth server..."

# Build a small Go program that starts the auth server and prints JSON config
$serverTmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "azd-auth-test-server-$(Get-Random)"
New-Item -ItemType Directory -Path $serverTmpDir -Force | Out-Null

$script:certsDir = Join-Path $serverTmpDir "certs"
New-Item -ItemType Directory -Path $script:certsDir -Force | Out-Null

# Use a helper Go script to start the server
$serverScript = Join-Path $serverTmpDir "main.go"
$goModFile = Join-Path $serverTmpDir "go.mod"

$azdCoreDir = (Resolve-Path (Join-Path $RepoRoot "..\azd-core")).Path
$azdCoreDirForwardSlash = $azdCoreDir -replace '\\', '/'

@"
module azd-auth-test-server

go 1.25.6

require github.com/jongio/azd-core v0.0.0

replace github.com/jongio/azd-core => $azdCoreDirForwardSlash
"@ | Set-Content -Path $goModFile -Encoding UTF8

@"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jongio/azd-core/authn"
)

func main() {
	certsDir := os.Args[1]

	srv := &authn.Server{
		Config: authn.ServerConfig{
			Port:          0,
			AllowedScopes: "*",
			CertsDir:      certsDir,
		},
	}

	if err := srv.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start: %v\n", err)
		os.Exit(1)
	}

	info := map[string]interface{}{
		"port":     srv.Port(),
		"certsDir": srv.CertsDir(),
	}
	jsonBytes, _ := json.Marshal(info)
	fmt.Println(string(jsonBytes))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	srv.Stop()
}
"@ | Set-Content -Path $serverScript -Encoding UTF8

# Run go mod tidy to resolve dependencies
$env:GOWORK = "off"
Push-Location $serverTmpDir
go mod tidy 2>&1 | Out-Null
Pop-Location

# Build the server helper
$serverBinExt = if ($IsWindows -or ($env:OS -eq "Windows_NT")) { ".exe" } else { "" }
$serverBin = Join-Path $serverTmpDir "server$serverBinExt"
Push-Location $serverTmpDir
go build -o $serverBin . 2>&1
$serverBuildResult = $LASTEXITCODE
Pop-Location
Remove-Item Env:GOWORK -ErrorAction SilentlyContinue

if ($serverBuildResult -ne 0) {
    Write-Fail "Failed to build auth server helper"
    Cleanup
    exit 1
}

# Start the server
$outputFile = Join-Path $serverTmpDir "output.txt"
$errorFile = Join-Path $serverTmpDir "error.txt"

$serverProc = Start-Process -FilePath $serverBin `
    -ArgumentList $script:certsDir `
    -RedirectStandardOutput $outputFile `
    -RedirectStandardError $errorFile `
    -PassThru -NoNewWindow

$script:authServerPid = $serverProc.Id

# Wait for JSON config
$serverInfo = $null
for ($i = 0; $i -lt 15; $i++) {
    Start-Sleep -Seconds 1
    if (Test-Path $outputFile) {
        $raw = Get-Content $outputFile -Raw -ErrorAction SilentlyContinue
        if ($raw) {
            $firstLine = ($raw -split "`n")[0].Trim()
            if ($firstLine.StartsWith("{")) {
                try { $serverInfo = $firstLine | ConvertFrom-Json; break } catch {}
            }
        }
    }
}

if (-not $serverInfo) {
    Write-Fail "Auth server failed to start within 15s"
    if (Test-Path $errorFile) {
        Write-Host "  Server error:" -ForegroundColor Red
        Get-Content $errorFile | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
    }
    Cleanup
    exit 1
}

$authPort = $serverInfo.port
Write-Pass "Auth server started (PID: $($serverProc.Id), port: $authPort)"

# --- Step 4: Start container with auth injection ---
Write-Step "4/7" "Starting container with auth injection..."

# Determine container host
$containerHost = "host.docker.internal"
# Check for Podman
try {
    $dockerVer = docker --version 2>&1
    if ($dockerVer -match "podman") { $containerHost = "host.containers.internal" }
} catch {}

$testappDir = Join-Path $TestDir "testapp"

# Build docker run args
$dockerArgs = @(
    "run", "-d",
    "--name", $script:containerName,
    "-p", "8080:8080",
    "-e", "ASPNETCORE_URLS=http://+:8080",
    "-e", "AZD_AUTH_HOST=$containerHost",
    "-e", "AZD_AUTH_PORT=$authPort",
    "-e", "AZD_AUTH_CERTS_DIR=/run/secrets/azd-auth",
    "-v", "$($script:shimPath):/usr/local/bin/azd:ro",
    "-v", "$($script:certsDir):/run/secrets/azd-auth:ro",
    "-v", "${testappDir}:/app"
)

# Add --add-host on native Linux (not Docker Desktop)
$runningOnLinux = $IsLinux -or ((Test-Path "/proc/version") -and -not (Test-Path "/mnt/c/Windows"))
if ($runningOnLinux) {
    $dockerArgs += @("--add-host", "host.docker.internal:host-gateway")
}

$dockerArgs += @(
    "mcr.microsoft.com/dotnet/sdk:8.0",
    "dotnet", "run", "--project", "/app/testapp.csproj"
)

Write-Detail "docker $($dockerArgs -join ' ')"

docker @dockerArgs 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Fail "Failed to start container"
    Cleanup
    exit 1
}
Write-Pass "Container started: $($script:containerName)"

# --- Step 5: Wait for app + validate injection ---
Write-Step "5/7" "Waiting for app and validating injection..."

# Wait for health endpoint
$ready = $false
for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Seconds 2
    try {
        $null = Invoke-WebRequest -Uri http://localhost:8080/health -UseBasicParsing -TimeoutSec 2
        $ready = $true; break
    } catch {}
    if ($i % 10 -eq 9) {
        Write-Detail "Still waiting... ($($i+1)s)"
        # Show container logs for debugging
        if ($Verbose) { docker logs $script:containerName --tail 5 2>&1 | ForEach-Object { Write-Detail "  $_" } }
    }
}

if (-not $ready) {
    Write-Fail "Container app did not become healthy within 120s"
    Write-Host "  Container logs:" -ForegroundColor Red
    docker logs $script:containerName --tail 30 2>&1 | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
    if (-not $KeepRunning) { Cleanup }
    exit 1
}
Write-Pass "App is healthy"

# Validate injection via /check endpoint
try {
    $checkResponse = Invoke-WebRequest -Uri http://localhost:8080/check -UseBasicParsing -TimeoutSec 10
    $check = $checkResponse.Content | ConvertFrom-Json

    if ($check.shimMounted) { Write-Pass "Shim binary is mounted at /usr/local/bin/azd" }
    else { Write-Fail "Shim binary not mounted" }

    if ($check.caCertPresent -and $check.clientCertPresent -and $check.clientKeyPresent) {
        Write-Pass "mTLS certificates are present"
    } else {
        Write-Fail "mTLS certificates missing (ca=$($check.caCertPresent) cert=$($check.clientCertPresent) key=$($check.clientKeyPresent))"
    }

    if ($check.authHost -ne "(not set)" -and $check.authPort -ne "(not set)") {
        Write-Pass "Auth env vars set (host=$($check.authHost), port=$($check.authPort))"
    } else {
        Write-Fail "Auth env vars not set (host=$($check.authHost), port=$($check.authPort))"
    }
} catch {
    Write-Fail "/check endpoint failed: $($_.Exception.Message)"
}

# --- Step 6: Validate token acquisition ---
Write-Step "6/7" "Validating token acquisition via DefaultAzureCredential..."

if (-not $azdAuthOk) {
    Write-Skip "Token test skipped (azd auth login required)"
} else {
    try {
        $tokenResponse = Invoke-WebRequest -Uri http://localhost:8080/token -UseBasicParsing -TimeoutSec 30
        $token = $tokenResponse.Content | ConvertFrom-Json

        if ($token.success) {
            Write-Pass "Token acquired via DefaultAzureCredential ($($token.durationMs)ms)"
            Write-Detail "Token prefix: $($token.tokenPrefix)"
            Write-Detail "Expires: $($token.expiresOn)"
        } else {
            Write-Fail "Token acquisition failed: $($token.message)"
        }
    } catch {
        Write-Fail "Token request failed: $($_.Exception.Message)"
        Write-Host "  Container logs:" -ForegroundColor Red
        docker logs $script:containerName --tail 10 2>&1 | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
    }
}

# --- Step 7: Validate Azure API call ---
Write-Step "7/7" "Validating Azure API call from container..."

if ($SkipAzure -or -not $azdAuthOk) {
    Write-Skip "Azure API test skipped"
} else {
    try {
        $azureResponse = Invoke-WebRequest -Uri http://localhost:8080/azure -UseBasicParsing -TimeoutSec 60
        $azure = $azureResponse.Content | ConvertFrom-Json

        if ($azure.success) {
            Write-Pass "Azure ARM call succeeded from container"
            Write-Host "    Subscription: $($azure.subscription)" -ForegroundColor White
            Write-Host "    Resource groups: $($azure.count)" -ForegroundColor White
            foreach ($rg in $azure.resourceGroups) {
                Write-Host "      - $rg" -ForegroundColor DarkGray
            }
        } else {
            Write-Fail "Azure API call failed: $($azure.message)"
        }
    } catch {
        Write-Fail "Azure API request failed: $($_.Exception.Message)"
    }
}

# --- Results ---
Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  Results: $($script:passed) passed, $($script:failed) failed, $($script:skipped) skipped" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# --- Cleanup ---
if ($KeepRunning) {
    Write-Host "Container is still running. Access at http://localhost:8080" -ForegroundColor Yellow
    Write-Host "Clean up manually: docker stop $($script:containerName) && docker rm $($script:containerName)" -ForegroundColor Yellow
    Write-Host "Stop auth server: Stop-Process -Id $($script:authServerPid)" -ForegroundColor Yellow
} else {
    Cleanup
    Write-Host "All resources cleaned up." -ForegroundColor Green
}

# Exit with appropriate code
if ($script:failed -gt 0) { exit 1 }
exit 0
