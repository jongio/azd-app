# Hooks Test Projects

Test projects demonstrating hook functionality in azd app.

## hooks-test

Basic example with simple prerun and postrun hooks that echo messages.

**Features:**
- Simple echo-based hooks
- Single service (Node.js)
- Demonstrates basic hook execution

**To test:**
```bash
cd hooks-test
azd app run --dry-run  # Preview without running
# azd app run  # Note: Would start a service that runs for 5 minutes
```

## Platform-Specific Hooks (Removed)

The `hooks-platform-test` project has been removed as its functionality is now covered by the main `hooks-test` project.

**Previously tested:**
- Platform-specific hook overrides (Windows/POSIX)
- External script files
- Shell selection (pwsh, bash)

**Current coverage:**
- Basic hook execution is tested in `hooks-test`
- Platform-specific behavior is validated in unit tests

## Testing Hooks Without Running Services

You can test hook parsing and configuration without actually running services:

```bash
# Test YAML parsing
cd hooks-test
cat azure.yaml

# Verify schema validation
# The yaml-language-server directive in azure.yaml provides IDE validation
```

## Testing Hook Execution

To test actual hook execution in unit tests, see:
- `cli/src/internal/executor/hooks_test.go` - Hook executor tests
- `cli/src/internal/service/hooks_test.go` - YAML parsing tests
- `cli/src/cmd/app/commands/hooks_integration_test.go` - Integration tests

## Expected Behavior

### Successful Execution
1. Prerun hook executes before services start
2. Services start and become ready
3. Postrun hook executes after all services are ready
4. Dashboard shows running services

### Hook Failure (continueOnError: false)
1. Prerun hook fails
2. Execution stops, services don't start
3. Error message displayed

### Hook Failure (continueOnError: true)
1. Prerun hook fails
2. Warning displayed but execution continues
3. Services start normally
4. Postrun hook still executes
