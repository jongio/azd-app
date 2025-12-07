# azd app test Output Fix

## Problem Statement

The `azd app test` command has two display bugs:

1. **Double checkmark**: Output shows `✓ ✓ api: 0 passed...` because `output.Success()` already adds a checkmark, and the format string also contains one
2. **Zero test counts**: Shows `0 passed, 0 total` even when Jest reports `10 passed, 10 total` because Jest writes its summary to stderr, but `RunCommandWithOutput` only captures stdout

## Root Cause Analysis

### Double Checkmark

In `test.go` line 246:
```go
output.Success("✓ %s: %d passed, %d total (%.2fs)", ...)
```

But `output.Success()` in `output.go` already adds a checkmark:
```go
func Success(format string, args ...interface{}) {
    msg := fmt.Sprintf(format, args...)
    check := getIcon(SymbolCheck, ASCIICheck)
    fmt.Printf("%s%s%s %s\n", BrightGreen, check, Reset, msg)
}
```

### Zero Test Counts

In `executor.go` line 91:
```go
output, err := cmd.Output()  // Only captures stdout
```

Jest writes test summary to stderr, so the output is never captured for parsing.

## Solution

1. Remove `✓` from format strings in `test.go` (lines 246, 249, 259)
2. Change `RunCommandWithOutput` to use `cmd.CombinedOutput()` to capture both stdout and stderr

## Files to Modify

- `cli/src/cmd/app/commands/test.go` - Remove ✓ from format strings
- `cli/src/internal/executor/executor.go` - Use CombinedOutput instead of Output

## Testing

- Run `azd app test` in demo folder
- Verify single checkmark in output
- Verify test counts match actual results
