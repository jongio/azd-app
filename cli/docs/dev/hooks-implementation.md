# Hooks Implementation Summary

## Overview
This implementation adds `prerun` and `postrun` hook support to the `azd app run` command, similar to how azd supports `preprovision` and `postprovision` hooks.

## Architecture

### 1. Schema Layer (`schemas/v1.1/azure.yaml.json`)
- Added `hooks` property to root schema
- Defined `hook` and `platformHook` definitions
- Supports properties:
  - `run`: Script or command to execute (required)
  - `shell`: Shell to use (sh, bash, pwsh, etc.)
  - `continueOnError`: Error handling behavior
  - `interactive`: User interaction support
  - `windows` / `posix`: Platform-specific overrides

### 2. Type Definitions (`src/internal/service/types.go`)
```go
type Hooks struct {
    Prerun  *Hook
    Postrun *Hook
}

type Hook struct {
    Run             string
    Shell           string
    ContinueOnError bool
    Interactive     bool
    Windows         *PlatformHook
    Posix           *PlatformHook
}

type PlatformHook struct {
    Run             string
    Shell           string
    ContinueOnError *bool  // Pointer to allow explicit false
    Interactive     *bool
}
```

### 3. Hook Executor (`src/internal/executor/hooks.go`)

**Key Functions:**
- `ExecuteHook()`: Main entry point for hook execution
- `ResolveHookConfig()`: Resolves platform-specific overrides
- `prepareHookCommand()`: Builds exec.Cmd based on shell type
- `getDefaultShell()`: Platform-aware default shell selection

**Features:**
- Cross-platform shell support (sh, bash, pwsh, powershell, cmd)
- Platform detection (Windows vs POSIX)
- Error handling with continueOnError
- Interactive mode support
- Environment variable inheritance

### 4. Run Command Integration (`src/cmd/app/commands/run.go`)

**Execution Flow:**
1. Parse `azure.yaml`
2. Execute `prerun` hook (if configured)
3. Start all services
4. Wait for services to be ready
5. Execute `postrun` hook (if configured)
6. Start dashboard and monitor

**Helper Functions:**
- `executePrerunHook()`: Executes prerun before services start
- `executePostrunHook()`: Executes postrun after services ready
- `convertPlatformHook()`: Converts service.PlatformHook to executor.PlatformHook

## Execution Details

### Hook Execution Context
- **Working Directory**: Directory containing `azure.yaml`
- **Environment**: Inherits all env vars including azd context
- **Shell Selection**: Platform-aware with fallbacks
- **Error Handling**: Respects `continueOnError` flag

### Platform-Specific Behavior

**Windows (default shells in order):**
1. `pwsh` (PowerShell Core) - preferred
2. `powershell` (Windows PowerShell)
3. `cmd` (Command Prompt)

**POSIX/Linux/macOS (default shells in order):**
1. `bash` - preferred
2. `sh` - fallback

### Shell Command Construction

**POSIX shells (sh, bash, zsh):**
```bash
<shell> -c "<script>"
```

**PowerShell:**
```pwsh
pwsh -Command "<script>"
```

**CMD:**
```cmd
cmd /c "<script>"
```

## Testing

### Unit Tests (15+ tests)
- **executor/hooks_test.go**: Hook execution logic
  - Default shell selection
  - Config resolution
  - Platform overrides
  - Command preparation
  - Execution with various scenarios

- **service/hooks_test.go**: YAML parsing
  - Basic hooks
  - Platform-specific hooks
  - Boolean overrides
  - Partial configurations

### Integration Tests
- **commands/hooks_integration_test.go**: End-to-end scenarios
  - Basic hook execution
  - Failure handling
  - ContinueOnError behavior

### Test Projects
- `tests/projects/hooks-test`: Basic example
- `tests/projects/hooks-platform-test`: Platform-specific example

## Documentation

### User Documentation
- **docs/hooks.md**: Comprehensive guide
  - Overview and concepts
  - Configuration reference
  - Platform-specific hooks
  - 10+ examples
  - Best practices
  - Troubleshooting

### Updated Documentation
- **docs/cli-reference.md**: Added hooks section to run command
- **README.md**: Added hooks example

## Example Configurations

### Basic
```yaml
hooks:
  prerun:
    run: echo "Starting..."
  postrun:
    run: echo "Ready!"
```

### Platform-Specific
```yaml
hooks:
  prerun:
    run: echo "default"
    windows:
      run: Write-Host "Windows"
      shell: pwsh
    posix:
      run: echo "POSIX"
      shell: bash
```

### Advanced
```yaml
hooks:
  prerun:
    run: npm run db:migrate
    shell: bash
    continueOnError: false
  postrun:
    run: curl -X POST $WEBHOOK
    continueOnError: true
    interactive: false
```

## Design Decisions

### 1. Platform Overrides vs Conditional Logic
**Chosen**: Platform-specific sections in YAML
**Rationale**: Cleaner YAML, easier to maintain, follows azd pattern

### 2. Shell Detection
**Chosen**: Automatic with overrides
**Rationale**: Works out of the box while allowing customization

### 3. Error Handling
**Chosen**: `continueOnError` flag
**Rationale**: Matches azd behavior, clear semantics

### 4. Execution Timing
**Chosen**: 
- Prerun: Before any services start
- Postrun: After all services ready
**Rationale**: Matches preprovision/postprovision semantics

### 5. Working Directory
**Chosen**: azure.yaml directory
**Rationale**: Consistent, predictable, matches azd

### 6. Environment Variables
**Chosen**: Inherit all from parent
**Rationale**: Access to azd context, user's environment

## Security Considerations

### Command Injection Prevention
- Uses `exec.CommandContext()` with separate args
- No shell expansion in command construction
- Scripts execute in isolated shell processes

### Environment Variable Safety
- Inherits only, doesn't modify
- No automatic env var expansion in hook scripts
- User controls environment through shell scripts

### File System Access
- Executes in azure.yaml directory
- User responsible for script permissions
- Scripts can access any files user can access

## Future Enhancements (Not in Scope)

1. **Service-level hooks**: Per-service prestart/poststart
2. **Hook templating**: Variable substitution in hook commands
3. **Hook output capture**: Structured logging of hook output
4. **Hook timeout**: Configurable timeout per hook
5. **Async postrun**: Don't block dashboard on postrun
6. **Hook dependencies**: Order hooks based on dependencies

## Compatibility

### Backward Compatibility
- Hooks are optional - existing configs work unchanged
- No breaking changes to existing functionality
- Gracefully handles missing hooks section

### Forward Compatibility
- Schema versioned (v1.1)
- Extensible design for future hook types
- Platform overrides pattern can extend to new platforms

## Validation

### Build
✅ `mage build` - Successful compilation

### Tests
✅ `mage test` - All tests pass
✅ Unit tests - 15+ passing
✅ Integration tests - All passing
✅ No test regressions

### Code Quality
✅ `mage fmt` - Code formatted
✅ No linter errors
✅ CodeQL - No security alerts

### Documentation
✅ Comprehensive user guide
✅ API documentation
✅ Example projects
✅ Updated CLI reference
