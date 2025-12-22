# Command Hooks Implementation

## Overview

Extend the existing hook system (currently only on `run` command) to all lifecycle commands: `test`, `deps`, `start`, `stop`, `restart`, `add`, and `reqs`. This enables automation and customization at every stage of the development workflow.

## Goals

1. **Consistency**: All commands follow the same hook pattern as `run`
2. **Automation**: Enable pre/post automation for all lifecycle operations
3. **Flexibility**: Support platform-specific, error handling, and interactive modes
4. **Observability**: Pass command-specific context via environment variables

## Scope

### Commands Getting Hooks

#### HIGH Priority
1. **`test`** - `pretest`, `posttest`
   - Use cases: Build before test, upload coverage, send notifications
2. **`deps`** - `predeps`, `postdeps`
   - Use cases: Clean dependencies, run security audits
3. **`start`** - `prestart`, `poststart`
   - Use cases: Validate environment, register services
4. **`stop`** - `prestop`, `poststop`
   - Use cases: Drain connections, archive logs
5. **`restart`** - `prerestart`, `postrestart`
   - Use cases: Graceful restart with notifications

#### MEDIUM Priority
6. **`add`** - `preadd`, `postadd`
   - Use cases: Validate compatibility, configure integrations
7. **`reqs`** - `prereqs`, `postreqs`
   - Use cases: Update package managers, configure tools

### Commands NOT Getting Hooks
- `run` - ✅ Already has hooks (prerun, postrun)
- `health`, `info`, `logs`, `version` - Read-only, no state changes
- `listen`, `mcp`, `notifications` - Internal/framework commands

## Schema Design

### Updated azure.yaml Structure

```yaml
name: my-app

hooks:
  # Run command (existing)
  prerun:
    run: echo "Before running services"
  postrun:
    run: echo "After services ready"
  
  # Test command (new)
  pretest:
    run: npm run build
    shell: bash
  posttest:
    run: ./upload-coverage.sh
    continueOnError: true
  
  # Dependencies command (new)
  predeps:
    run: rm -rf node_modules
    continueOnError: true
  postdeps:
    run: npm audit
    continueOnError: true
  
  # Service lifecycle commands (new)
  prestart:
    run: ./validate-env.sh
  poststart:
    run: curl -X POST $WEBHOOK_URL -d "Services started"
  
  prestop:
    run: ./drain-connections.sh
  poststop:
    run: ./archive-logs.sh
  
  prerestart:
    run: ./pre-restart-check.sh
  postrestart:
    run: ./post-restart-verify.sh
  
  # Configuration commands (new)
  preadd:
    run: ./backup-azure-yaml.sh
  postadd:
    run: ./configure-service.sh
  
  prereqs:
    run: ./update-package-managers.sh
  postreqs:
    run: ./configure-tools.sh

services:
  web:
    language: TypeScript
    project: ./frontend
```

### Hook Properties (Same as existing)

All hooks support the same properties as `run` command hooks:
- `run` (required): Script or command to execute
- `shell` (optional): Shell to use (sh, bash, pwsh, powershell, cmd)
- `continueOnError` (optional, default: false): Continue if hook fails
- `interactive` (optional, default: false): Allow user interaction
- `windows` (optional): Windows-specific override
- `posix` (optional): POSIX-specific override

## Environment Variables

Each command type passes specific context to its hooks via environment variables. This follows the existing pattern from `run` command.

### Common Variables (all hooks)
- `AZD_APP_PROJECT_NAME` - Project name from azure.yaml
- `AZD_APP_PROJECT_DIR` - Absolute path to project directory
- `AZD_APP_SERVICE_COUNT` - Number of services defined

### Test Command Variables
- `AZD_APP_TEST_TYPE` - Test type being run (unit, integration, e2e, all)
- `AZD_APP_TEST_COVERAGE` - Whether coverage is enabled (true/false)
- `AZD_APP_TEST_SERVICES` - Comma-separated list of services being tested
- `AZD_APP_TEST_OUTPUT_DIR` - Test results output directory

### Deps Command Variables
- `AZD_APP_DEPS_CLEAN` - Whether --clean flag was used (true/false)
- `AZD_APP_DEPS_SERVICES` - Comma-separated list of services (if filtered)

### Start/Stop/Restart Command Variables
- `AZD_APP_SERVICES` - Comma-separated list of services being operated on
- `AZD_APP_OPERATION` - Operation type (start, stop, restart)

### Add Command Variables
- `AZD_APP_ADD_SERVICE` - Service being added (azurite, cosmos, redis, postgres)
- `AZD_APP_ADD_SERVICE_DISPLAY_NAME` - Display name of service

### Reqs Command Variables
- `AZD_APP_REQS_GENERATE` - Whether --generate flag was used (true/false)

## Hook Execution Order

### Test Command Flow
```
azd app test
  ↓
Parse azure.yaml
  ↓
Execute PRETEST hook ←─────────────────┐
  ↓                                     │
Run tests (all services)                │ STOP if hook fails
  ↓                                     │ (unless continueOnError: true)
Generate coverage reports               │
  ↓                                     │
Execute POSTTEST hook ←─────────────────┘
  ↓
Display results
```

### Deps Command Flow
```
azd app deps
  ↓
Parse azure.yaml (optional)
  ↓
Execute PREDEPS hook ←──────────────────┐
  ↓                                     │
Detect projects                          │ STOP if hook fails
  ↓                                     │ (unless continueOnError: true)
Install dependencies (parallel)         │
  ↓                                     │
Execute POSTDEPS hook ←─────────────────┘
  ↓
Display summary
```

### Start/Stop/Restart Command Flow
```
azd app start/stop/restart
  ↓
Connect to dashboard
  ↓
Get service list
  ↓
Execute PRE* hook ←─────────────────────┐
  ↓                                     │
Perform operation                        │ STOP if hook fails
  ↓                                     │ (unless continueOnError: true)
Execute POST* hook ←────────────────────┘
  ↓
Display results
```

### Add Command Flow
```
azd app add <service>
  ↓
Validate service exists
  ↓
Execute PREADD hook ←───────────────────┐
  ↓                                     │
Update azure.yaml                        │ STOP if hook fails
  ↓                                     │ (unless continueOnError: true)
Execute POSTADD hook ←──────────────────┘
  ↓
Display success message
```

### Reqs Command Flow
```
azd app reqs
  ↓
Parse azure.yaml
  ↓
Execute PREREQS hook ←──────────────────┐
  ↓                                     │
Check all prerequisites                  │ STOP if hook fails
  ↓                                     │ (unless continueOnError: true)
Execute POSTREQS hook ←─────────────────┘
  ↓
Display results
```

## Implementation Details

### Type Updates (internal/service/types.go)

```go
type Hooks struct {
    // Run command (existing)
    Prerun  *Hook `yaml:"prerun,omitempty"`
    Postrun *Hook `yaml:"postrun,omitempty"`
    
    // Test command (new)
    Pretest  *Hook `yaml:"pretest,omitempty"`
    Posttest *Hook `yaml:"posttest,omitempty"`
    
    // Deps command (new)
    Predeps  *Hook `yaml:"predeps,omitempty"`
    Postdeps *Hook `yaml:"postdeps,omitempty"`
    
    // Start command (new)
    Prestart  *Hook `yaml:"prestart,omitempty"`
    Poststart *Hook `yaml:"poststart,omitempty"`
    
    // Stop command (new)
    Prestop  *Hook `yaml:"prestop,omitempty"`
    Poststop *Hook `yaml:"poststop,omitempty"`
    
    // Restart command (new)
    Prerestart  *Hook `yaml:"prerestart,omitempty"`
    Postrestart *Hook `yaml:"postrestart,omitempty"`
    
    // Add command (new)
    Preadd  *Hook `yaml:"preadd,omitempty"`
    Postadd *Hook `yaml:"postadd,omitempty"`
    
    // Reqs command (new)
    Prereqs  *Hook `yaml:"prereqs,omitempty"`
    Postreqs *Hook `yaml:"postreqs,omitempty"`
}

// Add getter methods for each hook (following existing pattern)
func (h *Hooks) GetPretest() *Hook { ... }
func (h *Hooks) GetPosttest() *Hook { ... }
// ... etc for all hooks
```

### Command Updates

Each command file will be updated to:
1. Load azure.yaml (if not already doing so)
2. Call pre-hook before main operation
3. Call post-hook after main operation
4. Pass command-specific environment variables

Pattern (following run.go):
```go
// In test.go
func runTests(opts *TestOptions) error {
    // ... existing code to get working directory ...
    
    // Load azure.yaml for hooks
    azureYaml, err := loadAzureYamlForHooks()
    if err == nil {
        // Execute pretest hook
        if err := executePreHook("test", azureYaml, opts); err != nil {
            return err
        }
        
        // Defer posttest hook
        defer func() {
            if postErr := executePostHook("test", azureYaml, opts); postErr != nil {
                output.Warning("Post-test hook failed: %v", postErr)
            }
        }()
    }
    
    // ... existing test logic ...
}

func executePreHook(command string, azureYaml *service.AzureYaml, opts interface{}) error {
    // Build environment variables specific to this command
    envVars := buildHookEnvVars(command, azureYaml, opts)
    
    // Get the appropriate hook
    var hook *service.Hook
    switch command {
    case "test":
        hook = azureYaml.Hooks.GetPretest()
    case "deps":
        hook = azureYaml.Hooks.GetPredeps()
    // ... etc
    }
    
    if hook == nil {
        return nil
    }
    
    // Execute using existing hook executor
    config := executor.ResolveHookConfig(convertHook(hook))
    config.Env = envVars
    return executor.ExecuteHook(context.Background(), "pre"+command, *config, workingDir)
}
```

## Documentation Updates

### Files to Update

1. **Schema Documentation**
   - `cli/docs/schema/azure.yaml.md` - Add all new hook types to Hooks section

2. **Command Documentation**
   - `cli/docs/commands/test.md` - Add hooks section
   - `cli/docs/commands/deps.md` - Add hooks section
   - `cli/docs/commands/start.md` - Add hooks section
   - `cli/docs/commands/stop.md` - Add hooks section
   - `cli/docs/commands/restart.md` - Add hooks section
   - `cli/docs/commands/add.md` - Add hooks section
   - `cli/docs/commands/reqs.md` - Add hooks section

3. **Feature Documentation**
   - `cli/docs/features/hooks.md` - Add examples for all new hooks

4. **CLI Reference**
   - Auto-generated from commands - no direct changes needed

## Testing Strategy

### Unit Tests

For each command, add tests following the pattern from `run_hooks_test.go`:

1. **Success cases**
   - `TestExecutePreXXXHook_Success`
   - `TestExecutePostXXXHook_Success`

2. **Failure cases**
   - `TestExecutePreXXXHook_Failure`
   - `TestExecutePostXXXHook_FailureWithContinue`

3. **No hooks configured**
   - `TestExecutePreXXXHook_NoHooks`

4. **Environment variables**
   - `TestBuildXXXHookEnvironmentVariables`

### Integration Tests

Add integration tests in `*_hooks_integration_test.go` files for each command:
- Hook execution with real scripts
- Platform-specific hooks
- Error handling with continueOnError

## Examples

### Test Workflow
```yaml
hooks:
  pretest:
    run: |
      echo "Building before tests..."
      npm run build
    shell: bash
  posttest:
    run: |
      # Upload coverage to Codecov
      curl -s https://codecov.io/bash | bash
      
      # Send Slack notification
      curl -X POST $SLACK_WEBHOOK \
        -d "{\"text\":\"Tests completed with ${AZD_APP_TEST_TYPE} coverage\"}"
    shell: bash
    continueOnError: true
```

### Dependencies Workflow
```yaml
hooks:
  predeps:
    run: |
      echo "Cleaning old dependencies..."
      rm -rf node_modules .venv
    continueOnError: true
  postdeps:
    run: |
      # Security audit
      npm audit --production
      
      # Generate SBOM
      cyclonedx-npm --output-file sbom.json
    continueOnError: true
```

### Service Lifecycle Workflow
```yaml
hooks:
  prestart:
    run: ./scripts/validate-ports.sh
  poststart:
    run: |
      echo "Services started: ${AZD_APP_SERVICES}"
      ./scripts/health-check.sh
      
  prestop:
    run: ./scripts/drain-connections.sh
  poststop:
    run: |
      echo "Archiving logs..."
      ./scripts/archive-logs.sh
      
  prerestart:
    run: ./scripts/backup-state.sh
  postrestart:
    run: ./scripts/verify-health.sh
```

### Configuration Workflow
```yaml
hooks:
  preadd:
    run: |
      # Backup current config
      cp azure.yaml azure.yaml.backup
    shell: bash
  postadd:
    run: |
      # Service: ${AZD_APP_ADD_SERVICE}
      echo "Configuring ${AZD_APP_ADD_SERVICE_DISPLAY_NAME}..."
      ./scripts/configure-${AZD_APP_ADD_SERVICE}.sh
    shell: bash
    
  prereqs:
    run: |
      # Update package managers
      brew update || apt-get update || echo "No package manager to update"
    continueOnError: true
  postreqs:
    run: |
      # Configure newly installed tools
      ./scripts/setup-tools.sh
    shell: bash
```

## Success Criteria

1. ✅ All 7 commands have pre/post hooks implemented
2. ✅ Schema updated in `types.go` with all 14 new hooks
3. ✅ All command files execute hooks with proper error handling
4. ✅ Command-specific environment variables passed to hooks
5. ✅ All documentation updated (schema, commands, features)
6. ✅ Unit tests for each command's hooks
7. ✅ Integration tests verify hooks execute correctly
8. ✅ Examples demonstrate real-world use cases

## Non-Goals

- Changing existing `run` command hook behavior
- Adding hooks to read-only commands (health, info, logs, version)
- Adding hooks to internal commands (listen, mcp, notifications)
- Changing hook schema structure (keep all hooks in single `hooks:` section)

## Timeline

1. **Phase 1**: Schema & Infrastructure (1 day)
   - Update `types.go` with all hook fields
   - Add getter methods
   - Update schema validation

2. **Phase 2**: High Priority Commands (2 days)
   - Implement test hooks
   - Implement deps hooks
   - Implement start/stop/restart hooks
   - Unit tests for each

3. **Phase 3**: Medium Priority Commands (1 day)
   - Implement add hooks
   - Implement reqs hooks
   - Unit tests for each

4. **Phase 4**: Documentation (1 day)
   - Update schema docs
   - Update command docs
   - Update features docs
   - Add examples

5. **Phase 5**: Integration Tests (1 day)
   - Add integration tests for each command
   - Test platform-specific hooks
   - Test error handling

Total: ~6 days
