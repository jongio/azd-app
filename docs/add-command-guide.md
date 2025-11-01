# 🎯 Adding New Commands - Complete Guide

## TL;DR - Quick Command Creation

```powershell
# One command to create, build, and install:
.\new-command.ps1 -CommandName mycommand -ShortDescription "My command" -Install

# Test it:
azd app mycommand
```

## 📋 What You Need to Know

### How Commands Show Up in `azd app`

When you run `azd app hi`, here's the flow:

1. **azd looks up the namespace** in `~/.azd/config.json`
2. **Finds the binary path**: `extensions\App.azd.App\0.1.0\App.exe`
3. **Executes it**: `App.exe hi`
4. **Your Go code runs**: Cobra routes "hi" to `newHiCommand()`
5. **Command executes**: Your RunE function runs

### The Two Files That Matter

**1. `cmd_[name].go`** - Your command implementation
```go
func newMyCommandCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "Description",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("My command logic here!")
            return nil
        },
    }
}
```

**2. `main.go`** - Command registration
```go
func main() {
    rootCmd := &cobra.Command{
        Use: "App",  // This is the namespace
    }
    
    rootCmd.AddCommand(newMyCommandCommand())  // Register here
    
    rootCmd.Execute()
}
```

## 🚀 Method 1: Automated (Recommended)

### Use the Generator Script

```powershell
# Create command only
.\new-command.ps1 -CommandName test -ShortDescription "Test command"

# Create AND install immediately
.\new-command.ps1 -CommandName test -ShortDescription "Test command" -Install
```

**What it does:**
1. ✅ Creates `cmd_test.go` with proper template
2. ✅ Updates `main.go` to register the command
3. ✅ (If -Install) Builds and installs the extension
4. ✅ Ready to use: `azd app test`

### Then Edit Your Command

Open `cmd_test.go` and implement your logic:

```go
func newTestCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "test",
        Short: "Test command",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Your implementation here
            fmt.Println("Testing...")
            
            // Can access arguments
            if len(args) > 0 {
                fmt.Println("Args:", args)
            }
            
            return nil
        },
    }
}
```

### Rebuild and Test

```powershell
# Rebuild
.\install-local.ps1

# Test
azd app test
```

## 🛠️ Method 2: Manual (Full Control)

### Step 1: Create `cmd_[name].go`

```go
package main

import (
    "fmt"
    "github.com/spf13/cobra"
)

func newMyCommandCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "mycommand",
        Short: "One-line description",
        Long:  `Detailed multi-line description
of what this command does.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("Executing mycommand!")
            
            // Your logic here
            
            return nil
        },
    }
}
```

**Naming Convention:**
- File: `cmd_mycommand.go`
- Function: `newMycommandCommand()` (capitalize first letter of command name)
- Use: `"mycommand"` (lowercase command name)

### Step 2: Register in `main.go`

Add one line:

```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "App",
        Short: "App - Developer productivity commands",
    }

    rootCmd.AddCommand(newHiCommand())
    rootCmd.AddCommand(newPrereqsCommand())
    rootCmd.AddCommand(newStatusCommand())
    rootCmd.AddCommand(newMycommandCommand())  // <-- Add this

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### Step 3: Build and Install

```powershell
.\install-local.ps1
```

**What this script does:**
1. Runs `azd x build` (compiles Go code)
2. Copies binary to `~/.azd/extensions/App.azd.App/0.1.0/`
3. Copies `extension.yaml` there too
4. Registers extension in `~/.azd/config.json`

### Step 4: Test

```powershell
azd app mycommand
```

## 🔄 Development Workflow

### Quick Iteration

```powershell
# Option 1: Watch mode (auto-rebuild on changes)
azd x watch

# Option 2: Manual rebuild
.\install-local.ps1

# Option 3: Just rebuild (if already registered)
azd x build
Copy-Item ".\bin\App.exe" "$env:USERPROFILE\.azd\extensions\App.azd.App\0.1.0\" -Force
```

### Testing

```powershell
# Test your command
azd app mycommand

# Test with arguments
azd app mycommand arg1 arg2

# See all commands
azd app --help

# See command help
azd app mycommand --help
```

## 📚 Command Templates

### Basic Command

```go
func newBasicCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "basic",
        Short: "A basic command",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("Basic command executed!")
            return nil
        },
    }
}
```

### Command with Flags

```go
func newFlagsCommand() *cobra.Command {
    var name string
    var count int
    
    cmd := &cobra.Command{
        Use:   "greet",
        Short: "Greet someone",
        RunE: func(cmd *cobra.Command, args []string) error {
            for i := 0; i < count; i++ {
                fmt.Printf("Hello, %s!\n", name)
            }
            return nil
        },
    }
    
    cmd.Flags().StringVarP(&name, "name", "n", "World", "Name to greet")
    cmd.Flags().IntVarP(&count, "count", "c", 1, "Number of times to greet")
    
    return cmd
}

// Usage: azd app greet --name John --count 3
```

### Command with Required Arguments

```go
func newRequiredArgsCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "deploy [environment]",
        Short: "Deploy to an environment",
        Args:  cobra.ExactArgs(1),  // Requires exactly 1 argument
        RunE: func(cmd *cobra.Command, args []string) error {
            env := args[0]
            fmt.Printf("Deploying to %s...\n", env)
            return nil
        },
    }
}

// Usage: azd app deploy production
```

### Command with Subcommands

```go
func newParentCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration",
    }
    
    cmd.AddCommand(&cobra.Command{
        Use:   "get [key]",
        Short: "Get a config value",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Printf("Getting config: %s\n", args[0])
            return nil
        },
    })
    
    cmd.AddCommand(&cobra.Command{
        Use:   "set [key] [value]",
        Short: "Set a config value",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Printf("Setting %s = %s\n", args[0], args[1])
            return nil
        },
    })
    
    return cmd
}

// Usage: 
// azd app config get mykey
// azd app config set mykey myvalue
```

## 🐛 Troubleshooting

### Command not showing up?

```powershell
# Check if registered in main.go
Get-Content main.go | Select-String "rootCmd.AddCommand"

# Rebuild
.\install-local.ps1

# Verify binary has the command
.\bin\App.exe --help
```

### Command shows but doesn't work?

```powershell
# Test binary directly
.\bin\App.exe mycommand

# Test installed binary
& "$env:USERPROFILE\.azd\extensions\App.azd.App\0.1.0\App.exe" mycommand

# Reinstall
.\install-local.ps1 -Uninstall
.\install-local.ps1
```

### Build errors?

```powershell
# Clean build
Remove-Item bin -Recurse -Force
go build -o bin/App.exe .

# Check for Go errors
go vet ./...
```

## 📖 Summary

**To add a new command that shows up in `azd app`:**

1. **Create** `cmd_[name].go` with `new[Name]Command()` function
2. **Register** in `main.go`: `rootCmd.AddCommand(new[Name]Command())`
3. **Build** with `.\install-local.ps1`
4. **Test** with `azd app [name]`

**Or use the generator:**

```powershell
.\new-command.ps1 -CommandName [name] -Install
```

**Key files modified:**
- `cmd_[name].go` ← Your new command
- `main.go` ← Registration
- Binary rebuilt and installed automatically

That's it! Your command will now show up when you run `azd app --help` and can be executed with `azd app [name]`.
