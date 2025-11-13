# Parallel Dependency Installation Output Specification

## Overview
When installing dependencies for multiple projects concurrently, each project should display its own progress indicator and final status on separate, persistent lines.

## Requirements

### During Installation
1. **Concurrent Display**: Each project displays on its own line with an animated progress indicator
2. **Non-Overlapping Output**: Multiple projects install concurrently with separate, non-interfering progress displays
3. **Real-time Activity**: Progress shows live activity (spinning/counting) while npm/pip/dotnet runs
4. **Activity Feedback**: As the underlying process writes output, the progress bar updates to show work is happening

### After Completion
1. **Persistent Status Lines**: Each project shows a final status line that remains visible (doesn't disappear)
2. **Success Format**: `✓ <project-name> (<package-manager>)` displayed in green
3. **Failure Format**: `✗ <project-name> (<package-manager>)` displayed in red
4. **Complete Summary**: All final status lines remain visible so user can see the complete summary of what succeeded/failed
5. **Overall Summary**: After all projects complete, show total count: `✓ Installed N project(s)` or error summary

## Visual Layout

### Expected Output Flow
```
Installing Dependencies

web (npm) (15/-, 8 it/s) [1s]    ← animated progress bar
api (pip) (12/-, 6 it/s) [1s]    ← animated progress bar

[after completion - progress bars cleared and replaced with:]

✓ web (npm)                       ← persistent success line
✓ api (pip)                       ← persistent success line

✓ Installed 2 project(s)          ← overall summary
```

### Error Case
```
Installing Dependencies

web (npm) (23/-, 9 it/s) [2s]    ← animated progress bar
api (pip) (8/-, 4 it/s) [1s]     ← animated progress bar

[after completion:]

✓ web (npm)                       ← success
✗ api (pip)                       ← failure

✗ Failed to install 1 project(s)  ← error summary
  • api: failed to install requirements: ...
```

## Implementation Details

### Progress Display
- **Type**: Indeterminate spinner bar (unknown total)
- **Updates**: Incremented on each write from npm/pip/dotnet process
- **Cleanup**: On completion, clear the progress bar line before printing status

### Output Flow
1. Create progress bar for each project with project name + package manager as description
2. Route process stdout/stderr to update progress on each write
3. When installation completes:
   - Stop the progress bar
   - Clear the progress bar line (overwrite with spaces + carriage return)
   - Print final status: `✓ <name>` or `✗ <name>` on a fresh line
4. After all projects complete, print overall summary

### Concurrency Handling
- Each project runs concurrently
- All projects must complete before showing summary
- Thread-safe handling of concurrent progress updates
- Progress display library handles concurrent rendering to terminal

### Status Messages
- **Success**: Green checkmark + project identifier
- **Failure**: Red X + project identifier + error details in summary
- **Summary**: Total count of successful/failed installations

## Edge Cases

### No Projects Found
Show message: "No projects found"

### All Skipped (Already Installed)
Still show per-project status but indicate they were skipped

### Mixed Success/Failure
Show all individual statuses, then error summary listing only failures

### JSON Output Mode
Suppress all progress bars and status lines, output only JSON result at end
