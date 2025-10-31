# Local Installation Guide for DevStack Extension

## Option 1: Quick Test (Direct Binary)

You can test the extension directly without azd integration:

```powershell
# From the project directory
.\bin\devstack.exe hi
```

## Option 2: Add to PATH (Use as standalone CLI)

Add the bin directory to your PATH so you can call it from anywhere:

```powershell
# Add to current session
$env:PATH += ";C:\code\devstackazdextension\bin"

# Test it
devstack hi

# To make it permanent, add to your PowerShell profile:
code $PROFILE
# Add this line:
# $env:PATH += ";C:\code\devstackazdextension\bin"
```

## Option 3: Full azd Extension Integration (Recommended)

To use it as `azd devstack`, you need to set up a local extension registry:

### Step 1: Create Local Extension Registry

Create a local registry file at `C:\code\devstackazdextension\local-registry.json`:

```json
{
  "$schema": "https://raw.githubusercontent.com/Azure/azure-dev/main/cli/azd/extensions/registry.schema.json",
  "extensions": [
    {
      "id": "devstack.azd.devstack",
      "name": "DevStack Extension",
      "description": "A collection of developer productivity commands for Azure Developer CLI",
      "publisher": "devstack",
      "version": "0.1.0",
      "tags": ["developer", "productivity"],
      "platforms": {
        "windows-amd64": {
          "url": "file:///C:/code/devstackazdextension/bin/devstack.exe",
          "sha256": ""
        }
      }
    }
  ]
}
```

### Step 2: Add Local Extension Source to azd

```powershell
# Add your local registry as an extension source
azd extension source add -n local -t file -l "C:\code\devstackazdextension\local-registry.json"

# Verify it was added
azd extension source list
```

### Step 3: Install the Extension

```powershell
# Install from your local source
azd extension install devstack.azd.devstack -s local

# Verify installation
azd extension list --installed
```

### Step 4: Use It!

```powershell
azd devstack hi
```

## Option 4: Copy to azd Extensions Directory (Manual Install)

Azd stores extensions in a specific directory. You can manually copy your binary there:

```powershell
# Find azd's extension directory
$azdExtDir = "$env:USERPROFILE\.azd\extensions\devstack.azd.devstack\0.1.0"

# Create the directory
New-Item -ItemType Directory -Path $azdExtDir -Force

# Copy your binary
Copy-Item .\bin\devstack.exe "$azdExtDir\devstack.exe"

# Copy the extension manifest
Copy-Item .\extension.yaml "$azdExtDir\extension.yaml"

# Create a registration file
@"
{
  "id": "devstack.azd.devstack",
  "version": "0.1.0",
  "installedAt": "$(Get-Date -Format 'yyyy-MM-ddTHH:mm:ss')"
}
"@ | Out-File -FilePath "$env:USERPROFILE\.azd\extensions\installed.json" -Encoding utf8
```

Then test:
```powershell
azd devstack hi
```

## Recommended Approach for Development

Use **Option 3** for the most realistic testing, or create a simple PowerShell alias:

```powershell
# Add to your $PROFILE
function azd {
    param([Parameter(ValueFromRemainingArguments)]$args)
    
    if ($args[0] -eq "devstack") {
        & "C:\code\devstackazdextension\bin\devstack.exe" $args[1..($args.Length-1)]
    } else {
        & azd.exe $args
    }
}
```

This intercepts `azd devstack` calls and routes them to your local binary while keeping normal `azd` commands working.

## Rebuild and Update

Whenever you make changes:

```powershell
# Rebuild
.\build.ps1

# If using Option 4, recopy the binary
Copy-Item .\bin\devstack.exe "$env:USERPROFILE\.azd\extensions\devstack.azd.devstack\0.1.0\devstack.exe" -Force
```
