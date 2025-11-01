# New Command Generator for App Extension
# This script creates a new command file and registers it automatically

param(
    [Parameter(Mandatory=$true)]
    [string]$CommandName,
    
    [Parameter(Mandatory=$false)]
    [string]$ShortDescription = "Description for $CommandName command",
    
    [Parameter(Mandatory=$false)]
    [string]$LongDescription = "Detailed description for the $CommandName command.",
    
    [switch]$Install = $false
)

$ProjectRoot = $PSScriptRoot
$CommandFileName = "cmd_$CommandName.go"
$CommandFilePath = Join-Path $ProjectRoot $CommandFileName
$MainFilePath = Join-Path $ProjectRoot "main.go"

Write-Host "🚀 Creating new command: $CommandName" -ForegroundColor Cyan

# Check if command file already exists
if (Test-Path $CommandFilePath) {
    Write-Host "❌ Command file $CommandFileName already exists!" -ForegroundColor Red
    exit 1
}

# Step 1: Create the command file
Write-Host "`n📝 Creating $CommandFileName..." -ForegroundColor Yellow

$commandTemplate = @"
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func new$($CommandName.Substring(0,1).ToUpper() + $CommandName.Substring(1))Command() *cobra.Command {
	return &cobra.Command{
		Use:   "$CommandName",
		Short: "$ShortDescription",
		Long:  ``$LongDescription``,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("✨ Running $CommandName command!")
			fmt.Println()
			fmt.Println("TODO: Implement $CommandName logic here")
			
			// Your command logic goes here
			
			return nil
		},
	}
}
"@

$commandTemplate | Out-File -FilePath $CommandFilePath -Encoding utf8
Write-Host "   ✓ Created $CommandFileName" -ForegroundColor Green

# Step 2: Update main.go to register the command
Write-Host "`n📝 Registering command in main.go..." -ForegroundColor Yellow

$mainContent = Get-Content $MainFilePath -Raw

# Generate the function name with proper capitalization
$functionName = "new$($CommandName.Substring(0,1).ToUpper() + $CommandName.Substring(1))Command()"

# Check if already registered
if ($mainContent -match [regex]::Escape("rootCmd.AddCommand($functionName)")) {
    Write-Host "   ⚠ Command already registered in main.go" -ForegroundColor Yellow
} else {
    # Find the "// Add commands" comment and add after the last AddCommand
    # Look for the pattern where commands are added
    $pattern = '(// Add commands\s+(?:rootCmd\.AddCommand\([^)]+\)\s+)+)'
    
    if ($mainContent -match $pattern) {
        # Add the new command after the last AddCommand line
        $commandsSection = $matches[1]
        $newCommandLine = "`trootCmd.AddCommand($functionName)`n"
        $updatedSection = $commandsSection + $newCommandLine
        $mainContent = $mainContent -replace [regex]::Escape($commandsSection), $updatedSection
        
        $mainContent | Out-File -FilePath $MainFilePath -Encoding utf8 -NoNewline
        Write-Host "   ✓ Registered $functionName in main.go" -ForegroundColor Green
    } else {
        # Fallback: Look for any AddCommand and add after it
        if ($mainContent -match '(\trootCmd\.AddCommand\([^)]+\))') {
            $lastMatch = $null
            $matches = [regex]::Matches($mainContent, '\trootCmd\.AddCommand\([^)]+\)')
            if ($matches.Count -gt 0) {
                $lastMatch = $matches[$matches.Count - 1]
                $insertPoint = $lastMatch.Index + $lastMatch.Length
                $before = $mainContent.Substring(0, $insertPoint)
                $after = $mainContent.Substring($insertPoint)
                $mainContent = $before + "`n`trootCmd.AddCommand($functionName)" + $after
                
                $mainContent | Out-File -FilePath $MainFilePath -Encoding utf8 -NoNewline
                Write-Host "   ✓ Registered $functionName in main.go" -ForegroundColor Green
            } else {
                Write-Host "   ❌ Could not find AddCommand pattern in main.go" -ForegroundColor Red
                Write-Host "   Please manually add: rootCmd.AddCommand($functionName)" -ForegroundColor Yellow
            }
        } else {
            Write-Host "   ❌ Could not find AddCommand pattern in main.go" -ForegroundColor Red
            Write-Host "   Please manually add: rootCmd.AddCommand($functionName)" -ForegroundColor Yellow
        }
    }
}

# Step 3: Show summary
Write-Host "`n✅ Command '$CommandName' created successfully!" -ForegroundColor Green
Write-Host "`nFiles created/modified:" -ForegroundColor Cyan
Write-Host "  - $CommandFileName (new)" -ForegroundColor White
Write-Host "  - main.go (updated)" -ForegroundColor White

Write-Host "`nNext steps:" -ForegroundColor Cyan
Write-Host "  1. Edit $CommandFileName to implement your command logic" -ForegroundColor White
Write-Host "  2. Build and install the extension:" -ForegroundColor White
Write-Host "     .\install-local.ps1" -ForegroundColor Gray
Write-Host "  3. Test your command:" -ForegroundColor White
Write-Host "     azd App $CommandName" -ForegroundColor Gray

# Step 4: Optionally build and install
if ($Install) {
    Write-Host "`n🔨 Building and installing extension..." -ForegroundColor Yellow
    & "$ProjectRoot\install-local.ps1"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "`n✅ Ready to test!" -ForegroundColor Green
        Write-Host "   Run: azd App $CommandName" -ForegroundColor White
    }
} else {
    Write-Host "`nTo build and install immediately, run:" -ForegroundColor Cyan
    Write-Host "  .\new-command.ps1 -CommandName $CommandName -Install" -ForegroundColor White
}
