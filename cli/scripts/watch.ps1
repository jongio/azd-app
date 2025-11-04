#!/usr/bin/env pwsh
#Requires -Version 7.4
# Watch for changes and auto-rebuild with debouncing
param()

$ErrorActionPreference = "Stop"

# Get script directory (scripts folder) and navigate to cli root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $scriptDir

Push-Location $ProjectRoot

Write-Host "👀 Watching for changes..." -ForegroundColor Cyan
Write-Host "   Monitoring: Go files (src/**/*.go), Dashboard (dashboard/src/**/*)" -ForegroundColor Gray
Write-Host "   Press Ctrl+C to stop" -ForegroundColor Yellow
Write-Host ""

# Track file modification times
$fileStates = @{}
$lastChange = $null
$debounceSeconds = 2 # Wait 2 seconds after last change

# Initial build
Write-Host "🔨 Initial build..." -ForegroundColor Cyan
try {
    $buildOutput = & "$scriptDir\install.ps1" 2>&1
    
    # Check for success in output instead of exit code (azd x build has a bug)
    $buildSuccess = $buildOutput -match "SUCCESS.*Build completed successfully|✅.*installed successfully"
    
    if ($buildSuccess) {
        # Get expected version from version.txt
        $expectedVersion = (Get-Content "$ProjectRoot\version.txt" -Raw).Trim()
        
        # Get actual version from azd app version
        try {
            $actualVersion = (azd app version 2>&1 | Out-String).Trim()
        } catch {
            $actualVersion = "unknown"
        }
        
        Write-Host "✅ Initial build complete! Version: $actualVersion (expected: $expectedVersion)" -ForegroundColor Green
    } else {
        Write-Host "❌ Initial build failed!" -ForegroundColor Red
        Write-Host "Build output:" -ForegroundColor Red
        $buildOutput | ForEach-Object { Write-Host $_ -ForegroundColor Gray }
    }
} catch {
    Write-Host "❌ Initial build failed with exception!" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
}
Write-Host ""

# Watch patterns
$watchPatterns = @(
    "src/**/*.go",
    "dashboard/src/**/*",
    "dashboard/package.json",
    "dashboard/vite.config.ts",
    "dashboard/tsconfig.json",
    "extension.yaml"
)

while ($true) {
    $changed = $false
    $changedFiles = @()

    # Check all patterns
    foreach ($pattern in $watchPatterns) {
        # Handle recursive patterns
        if ($pattern -like "*/**/*") {
            $parts = $pattern -split '/\*\*/\*'
            $baseDir = $parts[0]
            $ext = if ($parts.Length -gt 1) { $parts[1] } else { "" }
            
            if (Test-Path $baseDir) {
                $files = Get-ChildItem -Path $baseDir -Recurse -File | Where-Object {
                    $ext -eq "" -or $_.Extension -eq $ext -or $_.Name -like "*$ext"
                }
                
                foreach ($file in $files) {
                    $modTime = $file.LastWriteTime
                    if ($fileStates.ContainsKey($file.FullName)) {
                        if ($modTime -gt $fileStates[$file.FullName]) {
                            $fileStates[$file.FullName] = $modTime
                            $changed = $true
                            $changedFiles += $file.Name
                            $lastChange = Get-Date
                        }
                    } else {
                        $fileStates[$file.FullName] = $modTime
                    }
                }
            }
        } else {
            # Non-recursive pattern
            $files = Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue
            foreach ($file in $files) {
                $modTime = $file.LastWriteTime
                if ($fileStates.ContainsKey($file.FullName)) {
                    if ($modTime -gt $fileStates[$file.FullName]) {
                        $fileStates[$file.FullName] = $modTime
                        $changed = $true
                        $changedFiles += $file.Name
                        $lastChange = Get-Date
                    }
                } else {
                    $fileStates[$file.FullName] = $modTime
                }
            }
        }
    }

    if ($changed) {
        Write-Host "🔄 [$(Get-Date -Format 'HH:mm:ss')] Change detected in $($changedFiles.Count) file(s), waiting for more changes..." -ForegroundColor Yellow
    }

    # If we had a change and it's been quiet for debounceSeconds, rebuild
    if ($lastChange -and ((Get-Date) - $lastChange).TotalSeconds -ge $debounceSeconds) {
        Write-Host ""
        Write-Host "🔨 [$(Get-Date -Format 'HH:mm:ss')] Rebuilding and reinstalling..." -ForegroundColor Cyan
        
        try {
            # First build with mage to bump version
            $buildOutput = & mage build 2>&1
            if ($LASTEXITCODE -ne 0) {
                throw "mage build failed"
            }
            
            # Then install
            $buildOutput = & "$scriptDir\install.ps1" 2>&1
            
            # Check for success in output instead of exit code (azd x build has a bug)
            $buildSuccess = $buildOutput -match "SUCCESS.*Build completed successfully|✅.*installed successfully"
            
            if ($buildSuccess) {
                # Get expected version from version.txt
                $expectedVersion = (Get-Content "$ProjectRoot\version.txt" -Raw).Trim()
                
                # Get actual version from azd app version
                try {
                    $actualVersion = (azd app version 2>&1 | Out-String).Trim()
                } catch {
                    $actualVersion = "unknown"
                }
                
                Write-Host "✅ [$(Get-Date -Format 'HH:mm:ss')] Rebuild complete! Version: $actualVersion (expected: $expectedVersion)" -ForegroundColor Green
            } else {
                Write-Host "❌ [$(Get-Date -Format 'HH:mm:ss')] Build failed!" -ForegroundColor Red
                Write-Host "Build output:" -ForegroundColor Red
                $buildOutput | ForEach-Object { Write-Host $_ -ForegroundColor Gray }
            }
        } catch {
            Write-Host "❌ [$(Get-Date -Format 'HH:mm:ss')] Build failed with exception!" -ForegroundColor Red
            Write-Host $_.Exception.Message -ForegroundColor Red
            Write-Host $_.ScriptStackTrace -ForegroundColor Gray
        }
        
        Write-Host ""
        Write-Host "👀 Watching for changes..." -ForegroundColor Cyan
        
        # Reset
        $lastChange = $null
    }

    Start-Sleep -Milliseconds 500
}

Pop-Location

