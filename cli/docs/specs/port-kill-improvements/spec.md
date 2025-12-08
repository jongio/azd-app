# Port Kill Improvements Spec

## Overview

The port killing feature used by `azd app run` when a port conflict is detected has two issues:

1. **Port killing doesn't actually work**: The process on the port is not reliably killed, especially on Windows where child processes may hold the port
2. **Always prompts user**: There's no way to configure automatic port killing without prompting every time

## Problem Analysis

### Issue 1: Port Kill Not Working

The current implementation in `portmanager/process.go` uses:

```go
// Windows
psScript := fmt.Sprintf("Stop-Process -Id %d -Force -ErrorAction SilentlyContinue", pid)
```

**Problems:**
- Only kills the parent process, not child processes that may be holding the port
- Node.js apps spawn child processes that inherit the port
- Python Flask/Django apps spawn worker processes
- .NET apps may have child processes

**Solution**: Kill the entire process tree, not just the parent process.

On Windows, use:
```powershell
$process = Get-CimInstance Win32_Process -Filter "ProcessId = $pid"
$children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $pid"
foreach ($child in $children) { Stop-Process -Id $child.ProcessId -Force -ErrorAction SilentlyContinue }
Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
```

On Unix, use:
```bash
pkill -TERM -P $pid 2>/dev/null || true  # Kill children first
kill -9 $pid 2>/dev/null || true          # Then kill parent
```

### Issue 2: No "Always Kill" Configuration

Users who always want to kill processes on conflicting ports must respond to the prompt every time. This is especially annoying in development workflows where ports are frequently reused.

**Solution**: Add a user preference `alwaysKillPortConflicts` that:
- When `true`, automatically kills processes on conflicting ports without prompting
- Can be set via `azd config set` or via the first-time prompt
- Stored in azd's user config via the existing `azdconfig.ConfigClient` interface

## Requirements

### Functional Requirements

1. **FR1**: When killing a process on a port, kill all child processes first, then the parent
2. **FR2**: Add a new user preference `alwaysKillPortConflicts` (boolean)
3. **FR3**: When the preference is `true`, skip the prompt and automatically kill the process
4. **FR4**: Add a new option in the prompt: "4) Always kill processes (remember this choice)"
5. **FR5**: Provide a way to reset the preference via config

### Non-Functional Requirements

1. **NFR1**: Process tree killing must complete within 5 seconds
2. **NFR2**: Preference must persist across sessions
3. **NFR3**: Works on Windows, macOS, and Linux

## Design

### Configuration Key

Store in azd user config at: `app.preferences.alwaysKillPortConflicts`

Value: `"true"` or `"false"` (string for config compatibility)

### Updated Prompt

When a port conflict is detected:

```
⚠️  Service 'api' requires port 3000 (configured in azure.yaml)
This port is currently in use by node (PID 1234).

Options:
  1) Always kill processes (don't ask again)
  2) Kill the process using port 3000
  3) Assign a different port automatically
  4) Cancel

Choose (1/2/3/4): 
```

### Process Tree Kill Algorithm

```
1. Get PID of process on port
2. Find all child processes (recursive)
3. Sort children by depth (deepest first)
4. Send SIGTERM/Stop-Process to all children
5. Wait up to 100ms
6. Send SIGKILL/Force stop to remaining children
7. Kill parent process
8. Verify port is free (with retries)
```

## API Changes

### ConfigClient Interface (azdconfig/config.go)

No changes needed - use existing `GetPreference` and `SetPreference` methods.

### PortManager Changes

1. Add `getAlwaysKillPreference()` method
2. Add `setAlwaysKillPreference(value bool)` method  
3. Update `AssignPort()` to check preference before prompting
4. Update kill logic to handle process trees

## Test Plan

1. Unit test: Process tree killing on Windows (mock)
2. Unit test: Process tree killing on Unix (mock)
3. Unit test: Preference storage and retrieval
4. Unit test: Prompt skipped when preference is true
5. Integration test: Kill Node.js process with workers
6. Integration test: Preference persists across runs

## Migration

No migration needed - new feature with sensible defaults (prompt behavior unchanged if preference not set).

## Future Considerations

- Could add per-project preferences (some projects always kill, others don't)
- Could add port-specific preferences (always kill on port 3000, prompt for others)
- Could show warning when auto-killing is enabled
