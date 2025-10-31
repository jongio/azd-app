# DevStack Extension - Quick Reference

## ✅ Your Extension is Now Working!

You can use it with:
```powershell
azd devstack hi
```

## 🔄 Development Workflow

### After Making Code Changes

**Option 1: Automatic rebuild (recommended)**
```powershell
azd x watch
```
This watches for file changes and automatically rebuilds and reinstalls.

**Option 2: Manual rebuild**
```powershell
azd x build
```

**Option 3: Full reinstall**
```powershell
.\install-local.ps1
```

### Testing Your Extension

```powershell
# Test the hi command
azd devstack hi

# See all available commands
azd devstack --help

# Test a specific command with help
azd devstack hi --help
```

## 📁 Key Files

- `extension.yaml` - Extension metadata and configuration
- `main.go` - Main entry point, registers all commands
- `cmd_hi.go` - Implementation of the "hi" command
- `install-local.ps1` - Installation script for local development

## ➕ Adding New Commands

1. **Create a new file** (e.g., `cmd_mycommand.go`):

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newMyCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Description of my command",
        Long:  "Detailed description of what this command does",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("My command is running!")
            return nil
        },
    }
}
```

2. **Register it in main.go**:

```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "devstack",
        Short: "DevStack - Developer productivity commands for Azure Developer CLI",
    }

    rootCmd.AddCommand(newHiCommand())
    rootCmd.AddCommand(newMyCommand())  // Add this line

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

3. **Rebuild**:

```powershell
azd x build
```

4. **Test it**:

```powershell
azd devstack mycommand
```

## 🐛 Troubleshooting

**Extension not found?**
```powershell
# Reinstall
.\install-local.ps1

# Check it's installed
Get-Content "$env:USERPROFILE\.azd\config.json" | ConvertFrom-Json | Select-Object -ExpandProperty extension | Select-Object -ExpandProperty installed
```

**Build errors?**
```powershell
# Clean build
Remove-Item bin -Recurse -Force
azd x build
```

**Want to start fresh?**
```powershell
# Uninstall
.\install-local.ps1 -Uninstall

# Reinstall
.\install-local.ps1
```

## 📚 Useful Commands

```powershell
# List all installed extensions
azd extension list --installed

# See azd developer extension commands
azd x --help

# Build for all platforms (when ready to publish)
azd x build --all

# Watch for changes during development
azd x watch

# Package the extension
azd x pack
```

## 🚀 Next Steps

1. Add more commands by creating new `cmd_*.go` files
2. Implement useful developer workflows
3. Test with `azd x watch` for rapid iteration
4. When ready, publish with `azd x release` and `azd x publish`

## 📖 Resources

- [Extension Framework Docs](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/)
