#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Run integration tests for the App extension.

.DESCRIPTION
    This script runs integration tests with proper setup and cleanup.
    Integration tests require external tools and may take several minutes.

.PARAMETER Package
    Specific package to test (e.g., "installer", "runner", "commands")

.PARAMETER Test
    Specific test to run (e.g., "TestInstallNodeDependenciesIntegration")

.PARAMETER Timeout
    Timeout for the test run (default: 10m)

.PARAMETER Verbose
    Show verbose output

.EXAMPLE
    .\test-integration.ps1
    Run all integration tests

.EXAMPLE
    .\test-integration.ps1 -Package installer
    Run only installer integration tests

.EXAMPLE
    .\test-integration.ps1 -Package installer -Test TestInstallNodeDependenciesIntegration
    Run a specific integration test

.EXAMPLE
    .\test-integration.ps1 -Verbose
    Run with verbose output
#>

param(
    [string]$Package = "",
    [string]$Test = "",
    [string]$Timeout = "10m",
    [switch]$Verbose
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "🧪 Running Integration Tests" -ForegroundColor Cyan
Write-Host ""

# Change to src directory
Push-Location -Path "src"

try {
    # Build test arguments
    $testArgs = @(
        "test",
        "-tags=integration",
        "-timeout=$Timeout"
    )

    if ($Verbose) {
        $testArgs += "-v"
    }

    if ($Test) {
        $testArgs += "-run=$Test"
    }

    # Determine which packages to test
    if ($Package) {
        $packagePath = switch ($Package) {
            "installer" { "./internal/installer" }
            "runner" { "./internal/runner" }
            "commands" { "./cmd/app/commands" }
            default { 
                Write-Host "❌ Unknown package: $Package" -ForegroundColor Red
                Write-Host "Valid packages: installer, runner, commands" -ForegroundColor Yellow
                exit 1
            }
        }
        $testArgs += $packagePath
    } else {
        $testArgs += "./..."
    }

    Write-Host "📦 Test arguments: go $($testArgs -join ' ')" -ForegroundColor Gray
    Write-Host ""

    # Check for required tools
    Write-Host "🔍 Checking for required tools..." -ForegroundColor Cyan
    
    $tools = @{
        "go" = "Go"
        "node" = "Node.js (optional for some tests)"
        "dotnet" = ".NET SDK (optional for some tests)"
        "python" = "Python (optional for some tests)"
    }

    foreach ($tool in $tools.Keys) {
        if (Get-Command $tool -ErrorAction SilentlyContinue) {
            Write-Host "  ✓ $($tools[$tool])" -ForegroundColor Green
        } else {
            Write-Host "  ⚠ $($tools[$tool]) not found" -ForegroundColor Yellow
        }
    }
    Write-Host ""

    Write-Host "⏱️  Timeout: $Timeout" -ForegroundColor Gray
    Write-Host "📝 Note: Integration tests may take several minutes and some may be skipped if tools are not installed" -ForegroundColor Gray
    Write-Host ""

    # Run the tests
    Write-Host "🚀 Running tests..." -ForegroundColor Cyan
    Write-Host ""

    $startTime = Get-Date
    & go @testArgs

    $exitCode = $LASTEXITCODE
    $duration = (Get-Date) - $startTime

    Write-Host ""
    Write-Host "⏱️  Duration: $($duration.ToString('mm\:ss'))" -ForegroundColor Gray

    if ($exitCode -eq 0) {
        Write-Host "✅ Integration tests passed!" -ForegroundColor Green
    } else {
        Write-Host "❌ Integration tests failed!" -ForegroundColor Red
        Write-Host ""
        Write-Host "💡 Troubleshooting tips:" -ForegroundColor Yellow
        Write-Host "  - Some tests may fail if required tools are not installed" -ForegroundColor Gray
        Write-Host "  - Check docs/integration-tests.md for setup instructions" -ForegroundColor Gray
        Write-Host "  - Run with -Verbose flag for more details" -ForegroundColor Gray
    }

    exit $exitCode

} finally {
    Pop-Location
}
