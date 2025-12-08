# Test Command Progress Feedback and Auto-Discovery

## Overview

The `azd app test` command currently lacks progress feedback during test execution and doesn't prompt users to save auto-discovered test configurations. This leads to a poor user experience where the command appears to "stall" without any indication of what's happening.

## Problem Statement

When running `azd app test` on a project without explicit test configuration:

1. **No progress feedback** - Shows "Running all tests..." then appears to hang
2. **Silent test execution** - No indication which service is being tested
3. **Missing validation** - Doesn't verify tests exist before attempting execution
4. **No config save prompt** - Auto-discovered configurations aren't offered to be saved

## Requirements

### 1. Progress Feedback During Test Execution

Display real-time progress for each service:
```
🧪 Running all tests...

  Testing services:
    ▸ web (js/vitest) - Running unit tests...
    ✓ api (ts/jest) - 12 passed, 0 failed (1.2s)
    • admin-server (ts) - Detecting test framework...
    • admin-client (js) - Pending
```

Requirements:
- Show service name and detected language
- Show detected framework when available
- Show current action (detecting, installing deps, running tests, etc.)
- Update in place using ANSI escape codes
- Support non-TTY output (CI) with simple line output

### 2. Pre-Test Validation

Before running tests, validate each service:
- Check if test framework is detectable
- Check if test command exists (package.json scripts, pytest, etc.)
- Check if test files exist
- Report services that will be skipped with reason

Output example:
```
🔍 Analyzing 4 services...
  ✓ web: vitest detected (15 test files)
  ✓ api: jest detected (8 test files)
  ⚠ admin-server: No test script in package.json (skipping)
  ✓ admin-client: vitest detected (3 test files)

Found 3 testable services (1 skipped)
```

### 3. Auto-Discovery Save Prompt

After auto-detecting test configuration, prompt user to save:
```
📋 Discovered test configuration:

services:
  web:
    test:
      framework: vitest
      unit:
        command: pnpm test
  api:
    test:
      framework: jest
      unit:
        command: npm test

Would you like to save this configuration to azure.yaml? [Y/n]
```

Requirements:
- Show discovered config in YAML format
- Only prompt in interactive mode (TTY)
- Flag to skip prompt: `--no-save`
- Flag to always save: `--save`
- Don't overwrite existing test config in azure.yaml

### 4. Smart Streaming Output (Default)

Stream test output by default with intelligent behavior based on context:

**Single service** → Stream directly (no prefix):
```
azd app test --service web
PASS src/App.test.tsx
PASS src/utils.test.tsx
✓ 12 tests passed
```

**Multiple services + sequential** (`--parallel=false`) → Stream with prefix:
```
azd app test --parallel=false
[web] PASS src/App.test.tsx
[web] ✓ 12 tests passed
[api] PASS src/handlers.test.ts
[api] ✓ 8 tests passed
```

**Multiple services + parallel** (default) → Progress bars, failures shown at end:
```
azd app test
  ✓ web (vitest) - 12 passed (1.2s)
  ✓ api (jest) - 8 passed (2.1s)
  ✗ admin (jest) - 5 passed, 2 failed (1.8s)

─── Failures ───
[admin] FAIL src/auth.test.ts
  ✗ should validate token
    Expected: true
    Received: false
```

**CI environment** (no TTY) → Always stream with prefix:
```
[web] PASS src/App.test.tsx
[web] ✓ 12 tests passed
[api] PASS src/handlers.test.ts
```

**Force streaming in parallel mode** → `--stream` flag:
```
azd app test --stream
[web] PASS src/App.test.tsx
[api] PASS src/handlers.test.ts  # interleaved output
[web] PASS src/utils.test.tsx
```

### 5. Timeout and Stall Detection

Add protection against stalled tests:
- Default timeout per service: 10 minutes
- Warning after 30 seconds without output
- Flag to configure: `--timeout 5m`

Stall detection:
```
⚠ web: No output for 30 seconds (test might be hanging)
   Use --stream to see real-time output
```

## Configuration

### New azure.yaml Options

```yaml
test:
  timeout: 10m           # Per-service timeout
  streamOutput: false    # Real-time output
  skipUndetected: true   # Skip services without detectable tests
```

### New CLI Flags

- `--stream` - Force streaming output even in parallel mode
- `--no-stream` - Force progress bar mode (suppress streaming)
- `--timeout <duration>` - Per-service timeout (default: 10m)
- `--save` - Save discovered config to azure.yaml
- `--no-save` - Don't prompt to save config
- `--skip-undetected` - Skip services without detectable tests (default: true)

## Implementation Notes

### Output Mode Selection Logic

```go
func selectOutputMode(opts *TestOptions, serviceCount int) OutputMode {
    // CI/non-TTY: always stream
    if !term.IsTerminal(os.Stdout.Fd()) {
        return OutputModeStream
    }
    // Explicit flags override defaults
    if opts.Stream {
        return OutputModeStream
    }
    if opts.NoStream {
        return OutputModeProgress
    }
    // Single service: stream
    if serviceCount == 1 {
        return OutputModeStream
    }
    // Sequential: stream with prefix
    if !opts.Parallel {
        return OutputModeStreamPrefixed
    }
    // Parallel: progress bars
    return OutputModeProgress
}
```

### Progress Display

Use the existing `MultiProgress` system in `internal/output/progress.go`:
```go
mp := output.NewMultiProgress()
mp.AddBar("web", "Testing web (vitest)")
mp.Start()
// ...
mp.UpdateStatus("web", output.TaskStatusRunning)
mp.Complete("web")
```

### Service Validation

Add a new method to TestOrchestrator:
```go
type ServiceValidation struct {
    Name        string
    Language    string
    Framework   string
    TestFiles   int
    CanTest     bool
    SkipReason  string
}

func (o *TestOrchestrator) ValidateServices() ([]ServiceValidation, error)
```

### Config Save

Add method to write discovered config:
```go
func SaveDiscoveredTestConfig(azureYamlPath string, services []ServiceInfo) error
```

## Success Criteria

- Users see progress feedback within 1 second of starting test command
- Services without tests are clearly reported as skipped
- Auto-discovered config can be saved with single confirmation
- Timeout prevents indefinite hangs
- Works in both interactive and CI environments

## Out of Scope

- Test result caching
- Parallel output streaming (one service at a time)
- Custom progress bar styles
