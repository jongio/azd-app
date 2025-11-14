# Integration Tests

This document describes the integration test suite for the azd CLI extension.

## Running Integration Tests

Integration tests are tagged with `// +build integration` and must be explicitly enabled:

```powershell
# Run all integration tests
go test -tags=integration -v ./src/cmd/app/commands -timeout 10m

# Run specific integration test
go test -tags=integration -v ./src/cmd/app/commands -run TestReqsFixIntegration_BasicPATHResolution -timeout 10m
```

## PATH Resolution Integration Tests

The `reqs_fix_integration_test.go` file contains comprehensive integration tests for the `--fix` flag functionality.

### Test Coverage

#### TestReqsFixIntegration_BasicPATHResolution
Tests the complete PATH resolution workflow:
1. Builds a test-tool binary
2. Installs it to a custom directory
3. Adds directory to User PATH (Windows registry)
4. Verifies tool is NOT in current session PATH
5. Runs `azd app reqs --fix`
6. Verifies tool is found via registry PATH refresh
7. Validates version checking works correctly

**Expected Result**: Fix succeeds and finds tool in registry PATH

#### TestReqsFixIntegration_VersionMismatch
Tests version validation during fix:
1. Builds test-tool (version 2.5.0)
2. Adds to registry PATH
3. Creates project requiring version 10.0.0
4. Runs `azd app reqs --fix`

**Expected Result**: Fix fails with version mismatch error

#### TestReqsFixIntegration_ToolNotFound
Tests behavior when tool doesn't exist anywhere:
1. Creates project requiring nonexistent tool
2. Runs `azd app reqs --fix`

**Expected Result**: Fix fails with "not found" error

#### TestReqsFixIntegration_CacheClearing
Tests that cache is properly invalidated after successful fix:
1. Creates fake cache entry
2. Runs `azd app reqs --fix`
3. Verifies cache was cleared

**Expected Result**: Old cache is removed after fix

### Platform Support

Currently, these integration tests only run on Windows because they test registry-based PATH resolution. Future versions should add Unix-specific tests for shell profile PATH resolution.

### Test Requirements

- **Windows**: Administrator privileges not required (uses User PATH)
- **Go compiler**: Required to build test-tool
- **Temporary directories**: Tests use `t.TempDir()` for isolation
- **PATH cleanup**: Tests automatically restore PATH state after completion

### Manual Testing

For manual testing scenarios, use the PowerShell test scripts:

```powershell
cd cli\tests\test-tool
.\test-scenarios.ps1 -Action test-basic
.\test-scenarios.ps1 -Action test-version-mismatch
.\test-scenarios.ps1 -Action test-not-found
.\test-scenarios.ps1 -Action cleanup
```

See [test-tool README](../../tests/test-tool/README.md) for details.

## Adding New Integration Tests

When adding integration tests:

1. **Tag with build constraint**: Add `// +build integration` at the top
2. **Use short mode skip**: Add `if testing.Short() { t.Skip(...) }`
3. **Clean up resources**: Use `defer` to ensure cleanup happens
4. **Use temp directories**: Use `t.TempDir()` for isolation
5. **Document expected behavior**: Add clear comments about what's being tested

Example:

```go
// +build integration

package commands

func TestMyIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    testDir := t.TempDir()
    defer cleanup(t, testDir)

    // Test implementation
}
```

## Debugging Integration Tests

To debug integration tests:

```powershell
# Run with verbose output
go test -tags=integration -v ./src/cmd/app/commands -run TestReqsFixIntegration

# Run specific test with more detail
go test -tags=integration -v ./src/cmd/app/commands -run TestReqsFixIntegration_BasicPATHResolution -timeout 10m

# Check what PATH modifications were made
$env:PATH -split ';' | Select-String "CustomTools"

# Verify User PATH in registry
[Environment]::GetEnvironmentVariable('Path', 'User')
```

## CI/CD Integration

Integration tests should run in CI:

```yaml
- name: Run Integration Tests
  run: go test -tags=integration -v ./src/cmd/app/commands -timeout 10m
  env:
    GO111MODULE: on
```

For Windows-specific tests, use conditional execution:

```yaml
- name: Run Windows Integration Tests
  if: runner.os == 'Windows'
  run: go test -tags=integration -v ./src/cmd/app/commands -run TestReqsFixIntegration -timeout 10m
```
