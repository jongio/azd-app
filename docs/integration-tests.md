# Integration Tests

This directory contains integration tests for the App extension. Integration tests differ from unit tests in that they:

- Execute real commands and processes
- Interact with external tools (package managers, Docker, etc.)
- May modify the filesystem
- Take longer to run
- Require specific tools to be installed

## Running Integration Tests

### Using Mage

```bash
# Run only integration tests
mage testIntegration

# Run all tests (unit + integration)
mage testAll

# Run only unit tests (default)
mage test
```

### Using Go Directly

```bash
# Run integration tests
cd src
go test -v -tags=integration ./...

# Run specific integration test package
go test -v -tags=integration ./internal/installer

# Run specific integration test
go test -v -tags=integration -run TestInstallNodeDependenciesIntegration ./internal/installer
```

### Using PowerShell

```powershell
# Run integration tests
cd src
go test -v -tags=integration ./...

# Run with timeout
go test -v -tags=integration -timeout=10m ./...
```

## Test Organization

Integration tests are organized by package and use Go build tags:

```go
// go:build integration
// +build integration

package installer
```

This allows them to be:
- **Excluded from regular test runs** (no build tag)
- **Included only when explicitly requested** (`-tags=integration`)
- **Kept in the same package** as the code they test

## Test Categories

### Installer Integration Tests
**Location:** `src/internal/installer/installer_integration_test.go`

Tests actual package installation:
- `TestInstallNodeDependenciesIntegration` - npm, pnpm, yarn installation
- `TestRestoreDotnetProjectIntegration` - dotnet restore
- `TestSetupPythonVirtualEnvIntegration` - pip, uv, poetry environment setup

**Requirements:**
- Node.js (for npm tests)
- pnpm (optional, for pnpm tests)
- yarn (optional, for yarn tests)
- .NET SDK (for dotnet tests)
- Python (for Python tests)
- uv/poetry (optional, for advanced Python tests)

### Runner Integration Tests
**Location:** `src/internal/runner/runner_integration_test.go`

Tests long-running processes:
- `TestRunAspireIntegration` - Aspire project execution
- `TestRunPnpmScriptIntegration` - pnpm script execution
- `TestRunDockerComposeIntegration` - Docker Compose execution

**Requirements:**
- .NET Aspire CLI (for Aspire tests)
- pnpm (for pnpm tests)
- Docker (for Docker Compose tests)

### Commands Integration Tests
**Location:** `src/cmd/app/commands/commands_integration_test.go`

Tests end-to-end command execution:
- `TestRunReqsIntegration` - Full prerequisite checking
- `TestCheckPrerequisiteIntegration` - Individual prerequisite checks
- `TestCheckIsRunningIntegration` - Service running checks

**Requirements:**
- Various tools depending on test (Node.js, Go, Docker, etc.)

## Environment Setup

### CI/CD Pipeline

Integration tests should run in a separate CI job with:

```yaml
# GitHub Actions example
test-integration:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    - name: Install dependencies
      run: |
        npm install -g pnpm
        pip install poetry uv
    - name: Run integration tests
      run: mage testIntegration
      timeout-minutes: 15
```

### Local Development

Before running integration tests locally:

```bash
# Install required tools
winget install OpenJS.NodeJS
winget install Microsoft.DotNet.SDK.8
winget install Python.Python.3.12

# Install package managers
npm install -g pnpm yarn
pip install poetry uv

# Install Docker Desktop (optional)
# Install .NET Aspire CLI (optional)
dotnet workload install aspire
```

## Best Practices

### 1. Use Build Tags
Always mark integration tests with build tags:
```go
// go:build integration
// +build integration
```

### 2. Check for Dependencies
Skip tests gracefully when dependencies are missing:
```go
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}

// Check for specific tool
if err != nil {
    t.Logf("Error: %v (may be expected if tool not installed)", err)
    t.Skip("Skipping due to missing tool")
}
```

### 3. Clean Up Resources
Always use `t.TempDir()` and defer cleanup:
```go
tempDir := t.TempDir() // Automatically cleaned up
defer func() {
    // Any additional cleanup
}()
```

### 4. Set Timeouts
Use context with timeout for long-running processes:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 5. Isolate Tests
Each test should:
- Use its own temporary directory
- Not depend on other tests
- Clean up after itself
- Not modify global state

## Troubleshooting

### Test Hangs
- Check for missing `cancel()` on contexts
- Verify background processes are properly terminated
- Increase timeout values if needed

### Missing Dependencies
- Review error messages - tests skip when tools are missing
- Install required tools per the Environment Setup section
- Some tests are optional and only run when specific tools are available

### Permission Errors
- Run terminal as administrator on Windows
- Check file permissions in temp directories
- Ensure write access to test directories

## Performance

Integration tests are slower than unit tests:
- **Unit tests:** ~2-5 seconds total
- **Integration tests:** ~2-5 minutes total (varies by installed tools)

Run unit tests frequently during development, integration tests before commits.
