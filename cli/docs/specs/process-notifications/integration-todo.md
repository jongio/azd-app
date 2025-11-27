# Notifications Command Integration Checklist

## Status
The notifications CLI commands are implemented but **not yet wired** into the azd app command structure.

## Implementation Complete ✅
- [x] Command implementation (`cli/src/cmd/notifications.go`)
- [x] Subcommands: list, mark-read, clear, stats
- [x] Backend integration (database, pipeline)
- [x] Documentation (`cli/docs/commands/notifications.md`)

## Integration Required ⏳

### 1. Register with App Command
The `newNotificationsCmd()` function needs to be registered with the main app command.

**File to modify:** `cli/src/cmd/app/main.go`

**Add to rootCmd:**
```go
import "github.com/jongio/azd-app/cli/src/cmd"

// In main() function, after rootCmd is created:
rootCmd.AddCommand(cmd.newNotificationsCmd())
```

**Alternative location:** `cli/src/cmd/app/commands/core.go` or create new `cli/src/cmd/app/commands/notifications.go`

### 2. Move Command File (Optional)
Consider moving the notifications command to align with other commands:

**Current location:**
```
cli/src/cmd/notifications.go
```

**Suggested location:**
```
cli/src/cmd/app/commands/notifications.go
```

This would be consistent with other commands like `run.go`, `logs.go`, etc.

### 3. Update Package Structure (If moved)
If moving to `commands/` package:

**Change:**
```go
package cmd
```

**To:**
```go
package commands
```

**And export function:**
```go
func NewNotificationsCommand() *cobra.Command {
    // ... (rename from newNotificationsCmd)
}
```

### 4. Test Integration
After wiring up, verify commands work:

```bash
azd app notifications --help
azd app notifications list
azd app notifications stats
```

## Quick Integration (Minimal Changes)

**Option 1: Keep current location, just register it**

In `cli/src/cmd/app/main.go`:
```go
import (
    appcmd "github.com/jongio/azd-app/cli/src/cmd"
    // ... other imports
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "app",
        Short: "App - Automate your development environment setup",
        // ...
    }

    // Add existing commands
    rootCmd.AddCommand(commands.NewRunCommand())
    rootCmd.AddCommand(commands.NewLogsCommand())
    rootCmd.AddCommand(commands.NewInfoCommand())
    rootCmd.AddCommand(commands.NewReqsCommand())
    rootCmd.AddCommand(commands.NewDepsCommand())
    rootCmd.AddCommand(commands.NewVersionCommand())
    
    // Add notifications command
    rootCmd.AddCommand(appcmd.NewNotificationsCmd())  // Make function exported

    // ...
}
```

**Option 2: Move to commands package (recommended)**

1. Move `cli/src/cmd/notifications.go` → `cli/src/cmd/app/commands/notifications.go`
2. Change package from `cmd` to `commands`
3. Export function: `NewNotificationsCommand()`
4. Register in `main.go`:
   ```go
   rootCmd.AddCommand(commands.NewNotificationsCommand())
   ```

## Files to Update

- [ ] `cli/src/cmd/app/main.go` - Register command
- [ ] `cli/src/cmd/notifications.go` - Export function OR move file
- [ ] `cli/extension.yaml` - Add command metadata (if needed)
- [ ] `cli/README.md` - Add to command list
- [ ] `cli/docs/cli-reference.md` - Add notifications section

## Testing After Integration

```bash
# Build
cd cli
mage build

# Test commands
./bin/azd-app-windows-amd64.exe app notifications --help
./bin/azd-app-windows-amd64.exe app notifications list
./bin/azd-app-windows-amd64.exe app notifications stats

# Test with actual database
./bin/azd-app-windows-amd64.exe app run  # Generate some notifications
./bin/azd-app-windows-amd64.exe app notifications list
```

## Current Command Path

❌ **NOT working yet:**
```bash
azd notifications list        # Wrong - not registered
azd app notifications list    # Correct path but not wired up yet
```

✅ **Will work after integration:**
```bash
azd app notifications list
azd app notifications mark-read 1
azd app notifications clear --older-than 7d
azd app notifications stats
```
