#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Create a new release draft for azd-app CLI
.DESCRIPTION
    This script triggers the GitHub Actions workflow to create a draft release.
    After the workflow completes, you can review and publish the release in GitHub.
.PARAMETER Version
    The semantic version number (e.g., 1.2.3)
.PARAMETER DryRun
    Show what would happen without actually triggering the workflow
.EXAMPLE
    .\release.ps1 -Version 1.2.3
    Creates a draft release for version 1.2.3
.EXAMPLE
    .\release.ps1 -Version 1.2.3 -DryRun
    Shows what would happen without triggering the workflow
#>

param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^\d+\.\d+\.\d+$', ErrorMessage = "Version must be in format X.Y.Z (e.g., 1.2.3)")]
    [string]$Version,
    
    [Parameter(Mandatory = $false)]
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# Check if gh CLI is installed
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Write-Error "GitHub CLI (gh) is not installed. Install from: https://cli.github.com/"
    exit 1
}

# Check if authenticated
$authStatus = gh auth status 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error "Not authenticated with GitHub. Run: gh auth login"
    exit 1
}

# Get current directory and ensure we're in the repo
$repoRoot = git rev-parse --show-toplevel 2>$null
if (-not $repoRoot) {
    Write-Error "Not in a git repository"
    exit 1
}

# Check if on main branch
$currentBranch = git branch --show-current
if ($currentBranch -ne 'main') {
    Write-Warning "⚠️  You're on branch '$currentBranch', not 'main'"
    $continue = Read-Host "Continue anyway? (y/N)"
    if ($continue -ne 'y') {
        Write-Host "Aborted."
        exit 0
    }
}

# Check for uncommitted changes
$gitStatus = git status --porcelain
if ($gitStatus) {
    Write-Warning "⚠️  You have uncommitted changes:"
    git status --short
    $continue = Read-Host "Continue anyway? (y/N)"
    if ($continue -ne 'y') {
        Write-Host "Aborted."
        exit 0
    }
}

# Check if tag already exists
$tagExists = git tag -l "cli-v$Version"
if ($tagExists) {
    Write-Error "❌ Tag cli-v$Version already exists. Choose a different version or delete the tag first."
    exit 1
}

# Check if release already exists on GitHub
$existingRelease = gh release view "cli-v$Version" 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Error "❌ Release cli-v$Version already exists on GitHub."
    exit 1
}

Write-Host ""
Write-Host "🚀 Release Plan" -ForegroundColor Cyan
Write-Host "═══════════════" -ForegroundColor Cyan
Write-Host "Version:        $Version" -ForegroundColor White
Write-Host "Tag:            cli-v$Version" -ForegroundColor White
Write-Host "Branch:         $currentBranch" -ForegroundColor White
Write-Host "Repository:     $(gh repo view --json nameWithOwner -q .nameWithOwner)" -ForegroundColor White
Write-Host ""
Write-Host "This will:" -ForegroundColor Yellow
Write-Host "  1. Update version.txt to $Version" -ForegroundColor Gray
Write-Host "  2. Commit and push the version change" -ForegroundColor Gray
Write-Host "  3. Build binaries for all platforms" -ForegroundColor Gray
Write-Host "  4. Update registry.json with checksums and URLs" -ForegroundColor Gray
Write-Host "  5. Commit and push registry.json" -ForegroundColor Gray
Write-Host "  6. Create tag cli-v$Version" -ForegroundColor Gray
Write-Host "  7. Create a DRAFT release on GitHub" -ForegroundColor Gray
Write-Host ""
Write-Host "After completion, you can:" -ForegroundColor Green
Write-Host "  • Review the draft release at:" -ForegroundColor Gray
Write-Host "    https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases" -ForegroundColor Blue
Write-Host "  • Click 'Publish release' to make it official" -ForegroundColor Gray
Write-Host ""

if ($DryRun) {
    Write-Host "🔍 DRY RUN - No actions will be taken" -ForegroundColor Magenta
    exit 0
}

$confirm = Read-Host "Proceed with creating draft release? (y/N)"
if ($confirm -ne 'y') {
    Write-Host "Aborted."
    exit 0
}

Write-Host ""
Write-Host "⏳ Triggering release workflow..." -ForegroundColor Yellow

try {
    # Trigger the workflow
    gh workflow run release-draft.yml -f "version=$Version"
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to trigger workflow"
        exit 1
    }
    
    Write-Host "✅ Workflow triggered successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📊 Monitor progress:" -ForegroundColor Cyan
    Write-Host "   gh run watch" -ForegroundColor Blue
    Write-Host ""
    Write-Host "Or view in browser:" -ForegroundColor Cyan
    Write-Host "   gh run list --workflow=release-draft.yml --limit 1" -ForegroundColor Blue
    Write-Host ""
    Write-Host "Once complete, review and publish at:" -ForegroundColor Cyan
    Write-Host "   https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases" -ForegroundColor Blue
    
} catch {
    Write-Error "Failed to trigger release: $_"
    exit 1
}
