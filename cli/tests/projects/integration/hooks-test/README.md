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

## hooks-platform-test

Advanced example with platform-specific hooks and external scripts.

**Features:**
- Platform-specific hook overrides (Windows/POSIX)
- External script files
- Python FastAPI service
- Demonstrates shell selection (pwsh, bash)

**To test:**
```bash
cd hooks-platform-test
azd app run --dry-run  # Preview without running
# azd app run  # Note: Requires Python and FastAPI installed
```

**Platform behavior:**
- **Windows**: Uses PowerShell (pwsh) with Windows-specific messages
- **Linux/macOS**: Uses Bash with POSIX-specific messages

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
