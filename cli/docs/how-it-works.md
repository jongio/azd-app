# How the App Extension Works

## 🎯 The Complete Flow

### Step 1: Extension Structure
When you run `azd app hi`, here's what happens behind the scenes:

```
azd app hi
     ↓
azd checks config.json for "App" namespace
     ↓
Finds: extension.installed.App.azd.App
     ↓
Executes: ~/.azd/extensions/App.azd.App/0.1.0/App.exe hi
     ↓
Your Go binary runs the "hi" command
```

## 📋 Required Components

### 1. Extension Binary
- **Location**: `~/.azd/extensions/App.azd.App/0.1.0/App.exe`
- **Created by**: `azd x build`
- **Source**: Your Go code compiled from `main.go` and `cmd_*.go` files

### 2. Extension Manifest
- **Location**: `~/.azd/extensions/App.azd.App/0.1.0/extension.yaml`
- **Created by**: Copied during `azd x build`
- **Purpose**: Metadata about your extension (id, version, capabilities)

### 3. Config Registration
- **Location**: `~/.azd/config.json`
- **Section**: `extension.installed.App.azd.App`
- **Created by**: Our `install-local.ps1` script
- **Purpose**: Tells azd about the namespace and where to find the binary

## 🔧 The Magic: How Commands Work

### File Structure
```
Appazdextension/
├── main.go              # Entry point - registers all commands
├── cmd_hi.go            # Individual command implementation
├── cmd_check.go         # Another command implementation
└── extension.yaml       # Extension metadata
```

### Command Registration Flow

**1. In `cmd_hi.go`:**
```go
func newHiCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "hi",                    // Command name
        Short: "Say hello...",          // Short description
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("👋 Hi!")       // Command logic
            return nil
        },
    }
}
```

**2. In `main.go`:**
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "App",              // This becomes the namespace
        Short: "App Extension",
    }
    
    // Register all commands here
    rootCmd.AddCommand(newHiCommand())
    rootCmd.AddCommand(newCheckCommand())
    
    rootCmd.Execute()
}
```

**3. When you run `azd app hi`:**
- azd looks up "App" namespace in config.json
- Executes the binary with argument "hi"
- Binary's main() runs, creates rootCmd with "App"
- Cobra matches "hi" to newHiCommand()
- RunE function executes your code

## 🛠️ How to Add a New Command

### Method 1: Use the Generator Script (Recommended)

```powershell
# Create a new command
.\new-command.ps1 -CommandName mycommand -ShortDescription "My new command" -Install

# This will:
# 1. Create cmd_mycommand.go
# 2. Register it in main.go
# 3. Build and install the extension
# 4. Command is ready: azd app mycommand
```

### Method 2: Manual Steps

**Step 1: Create `cmd_mycommand.go`**
```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newMycommandCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Description of mycommand",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("My command is running!")
            return nil
        },
    }
}
```

**Step 2: Register in `main.go`**
```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "App",
        Short: "App - Developer productivity commands",
    }

    rootCmd.AddCommand(newHiCommand())
    rootCmd.AddCommand(newMycommandCommand())  // Add this line

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**Step 3: Build and Install**
```powershell
.\install-local.ps1
```

**Step 4: Test**
```powershell
azd app mycommand
```

## 🔄 The Installation Process

### What `install-local.ps1` Does:

```powershell
# 1. Builds the extension
azd x build
    ↓
# Compiles Go code → App.exe
# Places in: ~/.azd/extensions/App.azd.App/0.1.0/
# Copies extension.yaml there too

# 2. Registers in config.json
# Adds entry to ~/.azd/config.json:
{
  "extension": {
    "installed": {
      "App.azd.App": {
        "id": "App.azd.App",
        "namespace": "App",        # This is the key!
        "path": "extensions\\App.azd.App\\0.1.0\\App.exe",
        "version": "0.1.0",
        ...
      }
    }
  }
}

# 3. Now azd knows:
# - "App" namespace → run this binary
# - Binary location
# - What capabilities it has
```

## 📊 Key Concepts

### Namespace
- Defined in `extension.yaml` and `main.go`
- This is what you type: `azd [namespace] [command]`
- Example: `azd app hi` → namespace="App", command="hi"

### Command Registration
- Each `cmd_*.go` file creates a cobra.Command
- `main.go` registers them with rootCmd.AddCommand()
- Cobra handles routing "hi" to the right function

### Binary Execution
- azd doesn't "load" your extension
- It executes it as a separate process
- Like running: `App.exe hi`
- azd just knows where to find it via config.json

## 🎓 Pattern to Follow

For every new command, follow this pattern:

1. **Create** `cmd_[name].go` with `new[Name]Command()` function
2. **Register** in `main.go` with `rootCmd.AddCommand(new[Name]Command())`
3. **Build** with `.\install-local.ps1` or `azd x build`
4. **Test** with `azd app [name]`

## 🔍 Debugging Tips

### Command not found?
```powershell
# Check if registered in config.json
Get-Content "$env:USERPROFILE\.azd\config.json" | ConvertFrom-Json | 
  Select-Object -ExpandProperty extension | 
  Select-Object -ExpandProperty installed

# Should show App.azd.App entry
```

### Binary not updating?
```powershell
# Force rebuild
Remove-Item bin -Recurse -Force
.\install-local.ps1
```

### See what azd is doing?
```powershell
# Run with debug
$env:AZD_DEBUG = "true"
azd app hi
```

## 📚 File Templates

### New Command Template (`cmd_[name].go`)
```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func new[Name]Command() *cobra.Command {
    return &cobra.Command{
        Use:   "[name]",
        Short: "Short description",
        Long:  "Long description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Your logic here
            return nil
        },
    }
}
```

### Main Registration Pattern
```go
// In main.go
rootCmd.AddCommand(new[Name]Command())
```

## 🚀 Quick Reference

```powershell
# Create new command (auto-installs)
.\new-command.ps1 -CommandName test -Install

# Manual rebuild
.\install-local.ps1

# Watch for changes (auto-rebuild)
azd x watch

# Test
azd app test

# Uninstall
.\install-local.ps1 -Uninstall
```

---

**Summary**: The extension works by:
1. Go binary with cobra commands
2. `azd x build` places binary in right location  
3. Config.json registers the namespace
4. azd executes binary when you type `azd app [command]`
5. Cobra routes to the right command handler
