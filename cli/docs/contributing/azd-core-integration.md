---
description: Developer guide for azd-core integration - local development and CI/CD workflows
project: azd-app
date: 2026-01-08
status: active
---

# azd-core Integration Guide for Contributors

This document explains how the azd-app CLI integrates with `azd-core` and provides guidance for local development and CI/CD workflows.

Note (2026-01-09): `azd-core v0.1.0` is now released and pinned in the CLI. The `replace` directive has been removed from `cli/go.mod`. For local development, continue to use a `go.work` workspace; for CI-simulation locally, set `GOWORK=off`.

## Overview

The azd-app CLI uses `azd-core` for Key Vault environment variable reference resolution. The `azd-core` module provides:

- `keyvault.NewKeyVaultResolver()`: Creates a Key Vault resolver with Azure authentication
- `keyvault.IsKeyVaultReference()`: Detects Azure Key Vault reference formats
- `keyvault.ResolveEnvironmentVariables()`: Resolves references to actual secret values

## Module Structure

```
c:\code\
├── azd-app/
│   ├── cli/                    # Main CLI code
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── src/internal/service/environment.go
│   └── ...
├── azd-core/
│   ├── go.mod
│   ├── keyvault/
│   │   ├── keyvault.go
│   │   └── keyvault_test.go
│   └── ...
└── go.work                     # Go workspace for local development
```

## Local Development Setup

### Prerequisites

- Go 1.25.5 or later
- Azure CLI (`az` command available)

### Initial Setup

1. **Clone both repositories**:
   ```bash
   git clone https://github.com/jongio/azd-app.git c:\code\azd-app
   git clone https://github.com/jongio/azd-core.git c:\code\azd-core
   ```

2. **Navigate to the workspace root**:
   ```bash
   cd c:\code
   ```

3. **The `go.work` file automatically enables the workspace**:
   ```
   go 1.25.5
   
   use (
       ./azd-app/cli
       ./azd-core
   )
   ```

### Building and Testing

**From the workspace root (`c:\code`):**

```bash
# Build the CLI
cd azd-app/cli && mage install

# Run tests (uses go.work for azd-core resolution)
cd azd-app/cli && go test ./...

# Run tests with coverage
cd azd-app/cli && go test ./... -coverprofile=coverage.out

# Check specific KV test coverage
cd azd-app/cli && go test ./src/internal/service -run TestResolveEnvironment
```

### Using the go.work Workspace

The `go.work` file in `c:\code` tells Go to:

1. Use local `azd-app/cli` and `azd-core` directories instead of fetching from GitHub
2. Link the modules directly, bypassing version constraints
3. Simplify local development and testing

**Important**: Do NOT commit `go.work` to the repository. It's used only for local development.

### Handling Azure Credentials in Local Dev

When running tests locally that interact with Key Vault, ensure Azure credentials are available:

```bash
# Option 1: Use Azure CLI
az login

# Option 2: Use a service principal with environment variables
export AZURE_CLIENT_ID="<client-id>"
export AZURE_TENANT_ID="<tenant-id>"
export AZURE_CLIENT_SECRET="<client-secret>"

# Then run tests
go test ./...
```

Tests that fail due to auth issues (e.g., 403 Forbidden) are expected and indicate proper credential validation. The integration uses graceful degradation, so these won't fail the test suite.

## CI/CD Configuration

### Go Module Resolution in CI

Unlike local development, CI/CD environments:

1. **Do NOT use `go.work`** - It's excluded from the repository
2. **Pin `azd-core` to a specific version** in `go.mod`:
   ```go
   require github.com/jongio/azd-core v0.1.0
   ```
3. **Use `replace` directives only for local development** (via `go.mod`):
   ```go
   replace github.com/jongio/azd-core v0.0.0 => ../../azd-core
   ```

### Pinning azd-core in go.mod

For CI and releases, `cli/go.mod` pins `azd-core` to a tagged version and does not include a local `replace` directive:

```go
require github.com/jongio/azd-core v0.1.0
```

For local development, rely on `go.work` to use a local checkout of `azd-core`. Do not commit `replace` directives. To simulate CI locally, run with `GOWORK=off`:

```powershell
$env:GOWORK='off'
go mod tidy
go list -m all | Select-String "github.com/jongio/azd-core"
```

### GitHub Actions Workflow

CI workflows automatically:

1. Clone both `azd-app` and `azd-core` if needed (determined by which versions are pinned)
2. Run `go test ./...` to verify Key Vault integration
3. Run other test suites (coverage, lint, etc.)

Example workflow setup:

```yaml
- name: Run tests
  run: |
    cd cli
    go test ./... -v -coverprofile=coverage.out
```

## Integration Points

### Key Files

1. **`cli/src/internal/service/environment.go`**:
   - `ResolveEnvironment()`: Main function that orchestrates resolution
   - `resolveKeyVaultReferences()`: Calls azd-core resolver
   - `hasKeyVaultReferences()`: Detection helper

2. **`cli/go.mod`**:
   - Declares `azd-core` dependency
   - Contains local `replace` directive for dev

3. **Tests: `cli/src/internal/service/environment_test.go`**:
   - `TestHasKeyVaultReferences`: Tests reference detection
   - `TestResolveEnvironmentWithKeyVaultReferences`: Tests full resolution flow
   - `TestEnvMapToSlice`/`TestEnvSliceToMap`: Tests helper functions

### How It Works

1. **Environment Loading**: `ResolveEnvironment()` merges variables from multiple sources
2. **Reference Detection**: `hasKeyVaultReferences()` checks if any variables contain Key Vault references
3. **Resolution**: If references exist:
   - Create a `KeyVaultResolver` from `azd-core`
   - Call `ResolveEnvironmentVariables()` with graceful degradation enabled
   - Log warnings but continue on errors
   - Return resolved variables or original values on failure

### Reference Formats Supported

All formats defined by `azd-core/keyvault`:

1. **SecretUri format**:
   ```
   @Microsoft.KeyVault(SecretUri=https://vault.vault.azure.net/secrets/name/version)
   ```

2. **VaultName format**:
   ```
   @Microsoft.KeyVault(VaultName=vault;SecretName=name;SecretVersion=version)
   ```

3. **akvs format**:
   ```
   akvs://guid/vault/secret/version
   ```

## Testing Guide

### Unit Tests

```bash
# Run all environment tests
go test ./src/internal/service -run TestResolveEnvironment

# Run specific KV tests
go test ./src/internal/service -run TestHasKeyVaultReferences
go test ./src/internal/service -run TestResolveEnvironmentWithKeyVaultReferences
```

### Integration Tests

Some tests may require actual Azure resources:

```bash
# These will attempt real Key Vault access if credentials are available
go test ./... -v
```

### Coverage Requirements

Target: ≥80% coverage for KV-related code

```bash
go test ./src/internal/service -coverprofile=coverage.out
go tool cover -html=coverage.out  # View in browser
go tool cover -func=coverage.out  # View coverage by function
```

## Debugging

### Enable Debug Output

```bash
# Set debug environment variable
export AZD_DEBUG=true

# Run application
azd app run
```

When `AZD_DEBUG=true`, Key Vault warning messages include variable names for troubleshooting:

```
Warning: failed to resolve Key Vault reference for DB_PASSWORD: failed to get secret...
```

### Test Failures

If Key Vault tests fail with authentication errors:

1. **Expected**: 403 Forbidden errors (auth validation)
2. **Expected**: "failed to create resolver" (no credentials)
3. **Unexpected**: Panic or test suite crash

For troubleshooting:

```bash
# Check Azure credentials
az account show

# Re-authenticate
az login

# Verify Key Vault access
az keyvault secret show --vault-name <vault> --name <secret>
```

## Version Management

### Current Status

- `azd-core`: Local development version (under `c:\code\azd-core`)
- `azd-app`: Depends on local `azd-core`

### Future Releases

When `azd-core` is published to GitHub:

1. Create a git tag in `azd-core`: `git tag v0.1.0`
2. Update `azd-app/cli/go.mod`:
   ```go
   require github.com/jongio/azd-core v0.1.0
   ```
3. Remove the `replace` directive
4. Run `go mod tidy`
5. Test in CI to verify resolution

## Updating azd-core

If you need to modify `azd-core`:

1. **Make changes in `c:\code\azd-core`**
2. **Test with azd-app**:
   ```bash
   cd c:\code\azd-app\cli
   go test ./...
   ```
3. **Go workspace automatically picks up changes** - no need to run `go mod tidy`
4. **Commit changes to both repos** (when ready to push)

## Common Issues

### Issue: "repository not found"

**Cause**: Go trying to fetch `azd-core` from GitHub instead of using local `go.work`

**Solution**: Ensure you're running commands from `c:\code` (workspace root) or `c:\code\azd-app\cli` where `go.work` is accessible

```bash
# Correct: Run from workspace root
cd c:\code
go work use  # Verify workspace is active

# Correct: Let Go find go.work automatically
cd c:\code\azd-app\cli
go test ./...  # Works because go.work is in parent directories
```

### Issue: "version v0.0.0 invalid"

**Cause**: `go.mod` has `v0.0.0` which is not a valid semantic version for remote lookup

**Solution**: Ensure the `replace` directive is in place in `go.mod` or update the version to a real release

### Issue: Tests fail with Key Vault timeouts

**Cause**: Key Vault is unreachable (network issue or authentication takes too long)

**Solution**: 
1. Check internet connectivity
2. Re-authenticate: `az login`
3. Tests will gracefully degrade and continue

## Related Documentation

- [azd-core Integration Spec](../../specs/azd-app/azd-core-integration.md)
- [Key Vault Integration Guide](../features/keyvault-integration.md)
- [Go Workspaces Documentation](https://go.dev/ref/mod#workspaces)
