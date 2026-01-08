<#
Adds or removes Windows Firewall rules for test ports.

Usage:
  # Add a single port
  .\add-firewall-rule.ps1 -Action Add -Port 12345 -Name "azd-app-test-12345"

  # Remove a rule by name
  .\add-firewall-rule.ps1 -Action Remove -Name "azd-app-test-12345"
#>

param(
    [Parameter(Mandatory=$true)]
    [ValidateSet('Add','Remove')]
    [string]$Action,

    [int]$Port,

    [string]$Name = "azd-app-test-rule",

    [string]$PortRange
)

function Ensure-Admin {
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Error "This script must be run as Administrator. Right-click PowerShell and 'Run as Administrator'."
        exit 1
    }
}

if ($Action -eq 'Add') {
    Ensure-Admin
    if ($Port -ne $null) {
        Write-Output "Adding firewall rule for port $Port with name '$Name'"
        New-NetFirewallRule -DisplayName $Name -Direction Inbound -Action Allow -Protocol TCP -LocalPort $Port -Profile Any -ErrorAction Stop
        Write-Output "Added rule $Name"
    } elseif ($PortRange) {
        Write-Output "Adding firewall rule for port range $PortRange with name '$Name'"
        New-NetFirewallRule -DisplayName $Name -Direction Inbound -Action Allow -Protocol TCP -LocalPort $PortRange -Profile Any -ErrorAction Stop
        Write-Output "Added rule $Name"
    } else {
        Write-Error "Specify -Port or -PortRange when adding a rule."
        exit 1
    }
} elseif ($Action -eq 'Remove') {
    Ensure-Admin
    Write-Output "Removing firewall rule with name '$Name'"
    Get-NetFirewallRule -DisplayName $Name -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue
    Write-Output "Removed rule(s) named $Name"
}
