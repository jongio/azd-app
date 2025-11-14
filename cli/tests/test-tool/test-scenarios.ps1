# Test Scenarios for PATH Resolution Fix Feature
# This script automates the test scenarios for manual validation

param(
    [ValidateSet('setup', 'test-basic', 'test-version-mismatch', 'test-not-found', 'cleanup', 'all')]
    [string]$Action = 'all'
)

$ErrorActionPreference = 'Stop'

# Set console encoding to UTF-8 for proper emoji display
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::InputEncoding = [System.Text.Encoding]::UTF8

$TestToolDir = "C:\CustomTools\test-tool"
$TestProjectDir = "C:\temp\path-fix-test"
$ScriptDir = $PSScriptRoot

function Write-TestStep {
    param([string]$Message)
    Write-Host "`n==== $Message ====" -ForegroundColor Cyan
}

function Write-TestResult {
    param([string]$Message, [bool]$Success)
    $color = if ($Success) { 'Green' } else { 'Red' }
    $symbol = if ($Success) { '✓' } else { '✗' }
    Write-Host "$symbol $Message" -ForegroundColor $color
}

function Setup-TestTool {
    Write-TestStep "Setting up test-tool"
    
    # Build test-tool
    Push-Location $ScriptDir
    Write-Host "Building test-tool.exe..."
    go build -o test-tool.exe test-tool.go
    
    # Create custom tools directory
    if (Test-Path $TestToolDir) {
        Write-Host "Removing existing test-tool directory..."
        Remove-Item -Recurse -Force $TestToolDir
    }
    
    Write-Host "Creating $TestToolDir..."
    New-Item -ItemType Directory -Path $TestToolDir -Force | Out-Null
    
    # Move executable
    Write-Host "Installing test-tool.exe to $TestToolDir..."
    Move-Item -Force test-tool.exe $TestToolDir\
    
    Pop-Location
    
    # Verify it's not in PATH
    $inPath = $false
    try {
        $null = Get-Command test-tool -ErrorAction Stop
        $inPath = $true
    } catch {
        $inPath = $false
    }
    
    if ($inPath) {
        Write-Host "Warning: test-tool is already in PATH. Removing from current session..." -ForegroundColor Yellow
        $escapedDir = [regex]::Escape($TestToolDir)
        $env:PATH = $env:PATH -replace ";$escapedDir", ""
    }
    
    Write-TestResult "test-tool installed to $TestToolDir" $true
    Write-TestResult "test-tool NOT in current session PATH" (-not $inPath)
}

function Add-ToUserPath {
    Write-TestStep "Adding test-tool to User PATH (registry only)"
    
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    
    if ($userPath -notlike "*$TestToolDir*") {
        Write-Host "Adding $TestToolDir to User PATH..."
        [Environment]::SetEnvironmentVariable('Path', "$userPath;$TestToolDir", 'User')
        Write-TestResult "Added to User PATH in registry" $true
    } else {
        Write-TestResult "Already in User PATH" $true
    }
    
    # Verify NOT in current session
    $inCurrentSession = $env:PATH -like "*$TestToolDir*"
    Write-TestResult "NOT in current PowerShell session PATH" (-not $inCurrentSession)
}

function Test-BasicScenario {
    Write-TestStep "Test Scenario: Basic PATH Resolution"
    
    # Setup
    Setup-TestTool
    
    # Create test project
    if (Test-Path $TestProjectDir) {
        Remove-Item -Recurse -Force $TestProjectDir
    }
    New-Item -ItemType Directory -Path $TestProjectDir -Force | Out-Null
    
    $azureYaml = @"
name: path-fix-test
reqs:
  - name: test-tool
    minVersion: 2.0.0
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
"@
    
    $azureYaml | Out-File -Encoding utf8 "$TestProjectDir\azure.yaml"
    Write-Host "Created azure.yaml in $TestProjectDir"
    
    Push-Location $TestProjectDir
    
    # Initial check (should fail)
    Write-Host "`nRunning initial check (should fail)..."
    Write-Host "Command: azd app reqs" -ForegroundColor Gray
    & azd app reqs 2>&1 | Out-Host
    
    # Add to PATH (registry only)
    Add-ToUserPath
    
    # Run fix (should succeed)
    Write-Host "`nRunning fix (should find tool)..."
    Write-Host "Command: azd app reqs --fix" -ForegroundColor Gray
    $fixOutput = & azd app reqs --fix 2>&1 | Out-String
    Write-Host $fixOutput
    
    # Check if fix reported success
    if ($fixOutput -match "All requirements now satisfied" -and $fixOutput -match "Fixed 1 of 1 issues") {
        Write-TestResult "✅ Fix succeeded - found and validated test-tool" $true
        Write-Host "Note: test-tool is NOT in current session PATH (expected behavior)" -ForegroundColor Yellow
        Write-Host "      The tool was found via registry PATH refresh" -ForegroundColor Yellow
    } else {
        Write-TestResult "❌ Fix FAILED - did not report success" $false
        Pop-Location
        return
    }
    
    # Create cache by running normal check (this would normally be done in new terminal/session)
    Write-Host "`nRunning normal check to create cache..."
    Write-Host "Command: azd app reqs (will fail due to session PATH limitation)" -ForegroundColor Gray
    & azd app reqs 2>&1 | Out-Null
    
    # Verify cache was created and reflects session state
    if (Test-Path "$TestProjectDir\.azure\cache\reqs_cache.json") {
        $cache = Get-Content "$TestProjectDir\.azure\cache\reqs_cache.json" | ConvertFrom-Json
        Write-TestResult "✅ Cache created successfully" $true
        Write-Host "   Cache shows allPassed=$($cache.allPassed) (false is expected due to session limitation)" -ForegroundColor Gray
    }
    
    Pop-Location
    
    Write-Host "`n" -NoNewline
    Write-TestResult "✅ BASIC SCENARIO TEST PASSED" $true
    Write-Host ""
}

function Test-VersionMismatch {
    Write-TestStep "Test Scenario: Version Mismatch"
    
    # Setup
    Setup-TestTool
    Add-ToUserPath
    
    # Create test project requiring newer version
    if (Test-Path $TestProjectDir) {
        Remove-Item -Recurse -Force $TestProjectDir
    }
    New-Item -ItemType Directory -Path $TestProjectDir -Force | Out-Null
    
    $azureYaml = @"
name: version-mismatch-test
reqs:
  - name: test-tool
    minVersion: 10.0.0
    command: test-tool
    args: ["--version"]
    versionPrefix: "test-tool version "
"@
    
    $azureYaml | Out-File -Encoding utf8 "$TestProjectDir\azure.yaml"
    Write-Host "Created azure.yaml requiring version 10.0.0 (installed is 2.5.0)"
    
    Push-Location $TestProjectDir
    
    # Run fix (should find but report version mismatch)
    Write-Host "`nRunning fix (should find tool but report version too old)..."
    Write-Host "Command: azd app reqs --fix" -ForegroundColor Gray
    & azd app reqs --fix 2>&1 | Out-Host
    
    Pop-Location
    
    Write-TestResult "Version mismatch scenario completed" $true
}

function Test-NotFound {
    Write-TestStep "Test Scenario: Tool Not Found"
    
    # Create test project with non-existent tool
    if (Test-Path $TestProjectDir) {
        Remove-Item -Recurse -Force $TestProjectDir
    }
    New-Item -ItemType Directory -Path $TestProjectDir -Force | Out-Null
    
    $azureYaml = @"
name: not-found-test
reqs:
  - name: totally-fake-tool-xyz-999
    minVersion: 1.0.0
"@
    
    $azureYaml | Out-File -Encoding utf8 "$TestProjectDir\azure.yaml"
    Write-Host "Created azure.yaml with non-existent tool"
    
    Push-Location $TestProjectDir
    
    # Run fix (should fail gracefully with suggestion)
    Write-Host "`nRunning fix (should report cannot find tool)..."
    Write-Host "Command: azd app reqs --fix" -ForegroundColor Gray
    & azd app reqs --fix 2>&1 | Out-Host
    
    Pop-Location
    
    Write-TestResult "Not found scenario completed" $true
}

function Cleanup-All {
    Write-TestStep "Cleaning up test environment"
    
    # Remove from PATH
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -like "*$TestToolDir*") {
        Write-Host "Removing from User PATH..."
        # Escape backslashes for regex
        $escapedPath = [regex]::Escape($TestToolDir)
        $newPath = $userPath -replace ";$escapedPath", "" -replace "$escapedPath;", ""
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-TestResult "Removed from User PATH" $true
    }
    
    # Delete directories
    if (Test-Path $TestToolDir) {
        Write-Host "Removing $TestToolDir..."
        Remove-Item -Recurse -Force $TestToolDir
        Write-TestResult "Removed test-tool directory" $true
    }
    
    if (Test-Path "C:\CustomTools") {
        $items = Get-ChildItem "C:\CustomTools"
        if ($items.Count -eq 0) {
            Write-Host "Removing empty C:\CustomTools..."
            Remove-Item -Force "C:\CustomTools"
        }
    }
    
    if (Test-Path $TestProjectDir) {
        Write-Host "Removing $TestProjectDir..."
        Remove-Item -Recurse -Force $TestProjectDir
        Write-TestResult "Removed test project directory" $true
    }
    
    Write-Host "`nCleanup complete!" -ForegroundColor Green
}

# Main execution
switch ($Action) {
    'setup' {
        Setup-TestTool
        Add-ToUserPath
    }
    'test-basic' {
        Test-BasicScenario
    }
    'test-version-mismatch' {
        Test-VersionMismatch
    }
    'test-not-found' {
        Test-NotFound
    }
    'cleanup' {
        Cleanup-All
    }
    'all' {
        Write-Host "==================================================" -ForegroundColor Magenta
        Write-Host "  PATH Resolution Fix - Automated Test Suite" -ForegroundColor Magenta
        Write-Host "==================================================" -ForegroundColor Magenta
        
        Test-BasicScenario
        Start-Sleep -Seconds 2
        
        Test-VersionMismatch
        Start-Sleep -Seconds 2
        
        Test-NotFound
        Start-Sleep -Seconds 2
        
        Write-Host "`n==================================================" -ForegroundColor Magenta
        Write-Host "  All Test Scenarios Completed!" -ForegroundColor Magenta
        Write-Host "==================================================" -ForegroundColor Magenta
        
        Write-Host "`nRun cleanup? (Y/N): " -NoNewline -ForegroundColor Yellow
        $response = Read-Host
        if ($response -eq 'Y' -or $response -eq 'y') {
            Cleanup-All
        } else {
            Write-Host "Skipping cleanup. Run with -Action cleanup to clean up later." -ForegroundColor Yellow
        }
    }
}
