# Key Vault Reference Resolution for azd-app

## Overview

Implement automatic Azure Key Vault reference resolution for environment variables across all azd-app commands. This allows users to store secrets in Azure Key Vault and reference them using a standardized format, with automatic resolution at runtime.

## Background

Based on the successful implementation in [azd-exec](https://github.com/jongio/azd-exec), where Key Vault references in environment variables are automatically resolved before script execution. This feature provides:

- Centralized secret management in Azure Key Vault
- No secrets in code or configuration files
- Azure RBAC-based access control
- Audit trail of secret access
- Automatic secret rotation support

## Goals

1. **Automatic Resolution**: Transparently resolve Key Vault references in environment variables
2. **Wide Command Coverage**: Support all commands that use environment variables
3. **Graceful Degradation**: Continue execution with warnings if resolution fails
4. **Performance**: Cache Key Vault clients per vault URL
5. **Security**: Use DefaultAzureCredential, no credential storage
6. **Backward Compatible**: No breaking changes to existing functionality

## Reference Formats

Support two reference formats (matching azd-exec):

### Format 1: SecretUri
```
@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)
@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret/abc123)
```

### Format 2: VaultName + SecretName
```
@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)
@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret;SecretVersion=abc123)
```

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Command Layer                             │
│  (run, start, test, logs, hooks, etc.)                      │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Environment Resolution Layer                     │
│  service.ResolveEnvironment() - INTEGRATION POINT            │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│           Key Vault Resolver (NEW)                           │
│  keyvault.KeyVaultResolver                                   │
│  - Pattern matching (2 formats)                              │
│  - Azure SDK integration                                     │
│  - Client caching                                            │
│  - Error handling                                            │
└─────────────────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Azure Key Vault SDK                             │
│  github.com/Azure/azure-sdk-for-go/sdk/...                  │
└─────────────────────────────────────────────────────────────┘
```

### Integration Point

The **single point of integration** is `service.ResolveEnvironment()` in `cli/src/internal/service/environment.go`. This function is called by all commands that need environment variables, making it the perfect place to inject Key Vault resolution.

## Commands That Need Environment Variables

### Category 1: Direct Service Execution (HIGH PRIORITY)

Commands that directly execute user code and MUST have resolved secrets:

1. **`azd app run`** - Starts development services
   - File: `cli/src/cmd/app/commands/run.go`
   - Integration: `runAzdMode()` → `runServices()` → uses `service.ResolveEnvironment()`
   - Priority: **P0 - Critical**

2. **`azd app start`** - Starts stopped services
   - File: `cli/src/cmd/app/commands/start.go`
   - Integration: `runStart()` → `ServiceController.StartService()` → uses `loadEnvVars()`
   - Priority: **P0 - Critical**

3. **`azd app restart`** - Restarts services
   - File: `cli/src/cmd/app/commands/restart.go`
   - Integration: Same as start
   - Priority: **P0 - Critical**

4. **`azd app test`** - Runs tests
   - File: `cli/src/cmd/app/commands/test.go`
   - Integration: `runTests()` → test execution with environment
   - Priority: **P0 - Critical**

### Category 2: Hook Execution (HIGH PRIORITY)

Commands that execute hooks which may need secrets:

5. **Prerun/Postrun Hooks**
   - File: `cli/src/cmd/app/commands/run.go`
   - Functions: `executePrerunHook()`, `executePostrunHook()`
   - Integration: Uses `executor.ExecuteHook()` with environment
   - Priority: **P0 - Critical**

6. **Service Lifecycle Hooks**
   - File: `cli/src/internal/service/hooks.go`
   - Integration: Hook execution contexts
   - Priority: **P1 - High**

### Category 3: Azure Operations (MEDIUM PRIORITY)

Commands that interact with Azure and may need credentials:

7. **`azd app logs`** - Azure log streaming
   - File: `cli/src/cmd/app/commands/logs.go`
   - Integration: Azure credentials for log queries
   - Priority: **P1 - High**

8. **`azd app health`** - Health checks
   - File: `cli/src/cmd/app/commands/health.go`
   - Integration: May need secrets for health endpoints
   - Priority: **P2 - Medium**

### Category 4: Information Commands (LOW PRIORITY)

Commands that display information but may benefit from resolution:

9. **`azd app info`** - Environment information
   - File: `cli/src/cmd/app/commands/info.go`
   - Integration: Display resolved values
   - Priority: **P3 - Low**

10. **Dashboard Environment Panel**
    - File: `cli/dashboard/src/components/EnvironmentPanel.tsx`
    - Integration: Display resolved values in UI
    - Priority: **P3 - Low**

## Design Approach

### Option 1: Central Integration (RECOMMENDED)

**Modify `service.ResolveEnvironment()` to resolve Key Vault references**

**Pros:**
- Single point of integration
- All commands benefit automatically
- Consistent behavior
- Minimal code changes
- Easy to test

**Cons:**
- None significant

**Implementation:**
```go
// In cli/src/internal/service/environment.go
func ResolveEnvironment(...) (map[string]string, error) {
    // ... existing logic to build env map ...
    
    // NEW: Resolve Key Vault references
    if hasKeyVaultReferences(env) {
        resolver, err := keyvault.NewKeyVaultResolver()
        if err != nil {
            // Log warning but continue with unresolved values
            logKeyVaultWarning(err)
        } else {
            env, err = resolver.ResolveEnvironmentMap(ctx, env)
            if err != nil {
                // Log warning but continue with unresolved values
                logKeyVaultWarning(err)
            }
        }
    }
    
    return env, nil
}
```

### Option 2: Per-Command Integration

**Add Key Vault resolution to each command individually**

**Pros:**
- Fine-grained control
- Can disable for specific commands

**Cons:**
- Many code changes
- Inconsistent behavior risk
- Harder to maintain
- More testing required

**Decision: Use Option 1 - Central Integration**

## Implementation Plan

### Phase 1: Core Infrastructure (P0)

#### Task 1: Add Azure SDK Dependencies
```bash
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.13.1
go get github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets@v1.4.0
```

#### Task 2: Create Key Vault Resolver Package
- **File**: `cli/src/internal/keyvault/keyvault.go`
- **Content**: Port from azd-exec with adaptations
- **Components**:
  - `KeyVaultResolver` struct
  - Pattern matching (2 regex patterns)
  - `ResolveReference()` method
  - `ResolveEnvironmentMap()` method (NEW - for map[string]string)
  - Client caching with RWMutex
  - Error handling

#### Task 3: Create Unit Tests
- **File**: `cli/src/internal/keyvault/keyvault_test.go`
- **Coverage**: Pattern matching, reference validation, format parsing
- **Target**: 80% coverage

#### Task 4: Create Integration Tests
- **File**: `cli/src/internal/keyvault/keyvault_integration_test.go`
- **Tags**: `//go:build integration`
- **Tests**: Real Azure Key Vault resolution (requires setup)

### Phase 2: Service Integration (P0)

#### Task 5: Modify ResolveEnvironment
- **File**: `cli/src/internal/service/environment.go`
- **Changes**:
  - Add `hasKeyVaultReferences()` helper
  - Call Key Vault resolver before returning
  - Graceful error handling with warnings
  - Add context parameter (may require signature change)

#### Task 6: Update ResolveEnvironment Callers
- Review all callers of `ResolveEnvironment()`
- Add context parameter where needed
- Ensure proper error handling

#### Task 7: Test Service Integration
- **File**: `cli/src/internal/service/environment_test.go`
- Add tests for Key Vault resolution in environment merging

### Phase 3: Command Coverage (P0)

#### Task 8: Test Critical Commands
Test that Key Vault resolution works for:
- `azd app run`
- `azd app start`
- `azd app test`
- Hook execution

Create integration tests in `cli/src/cmd/app/commands/*_integration_test.go`

### Phase 4: Documentation (P1)

#### Task 9: Update Documentation
- **README.md**: Add Key Vault integration section
- **docs/features/keyvault.md**: Comprehensive guide
- **Examples**: Create demo scripts showing usage

#### Task 10: Add CLI Help Text
Update command help to mention Key Vault support

### Phase 5: Advanced Features (P2)

#### Task 11: Dashboard Support
- Display resolved values in environment panel
- Mask secret values for security
- Show resolution status

#### Task 12: Diagnostic Logging
- Add verbose logging option
- Show which secrets are being resolved
- Performance metrics

## Technical Specifications

### Key Vault Resolver Interface

```go
package keyvault

import (
    "context"
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
    "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

type KeyVaultResolver struct {
    credential *azidentity.DefaultAzureCredential
    clients    map[string]*azsecrets.Client
    mu         sync.RWMutex
}

func NewKeyVaultResolver() (*KeyVaultResolver, error)
func (r *KeyVaultResolver) ResolveReference(ctx context.Context, reference string) (string, error)
func (r *KeyVaultResolver) ResolveEnvironmentMap(ctx context.Context, env map[string]string) (map[string]string, error)
func IsKeyVaultReference(value string) bool
```

### Environment Resolution Signature Change

**Current:**
```go
func ResolveEnvironment(service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error)
```

**Proposed:**
```go
func ResolveEnvironment(ctx context.Context, service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error)
```

**Impact Analysis:**
- Requires updating ~10-15 call sites
- All callers already have context available
- Breaking change but internal API only
- Will be tested by existing tests

### Error Handling Strategy

1. **Credential Errors**: Warn and continue with unresolved reference
2. **Network Errors**: Warn and continue with unresolved reference
3. **Vault Not Found**: Warn and continue with unresolved reference
4. **Secret Not Found**: Warn and continue with unresolved reference
5. **Invalid Format**: Warn and continue with unresolved reference

**Rationale**: Services should still run even if secret resolution fails. This allows:
- Offline development
- Missing permissions scenarios
- Graceful degradation

### Performance Considerations

1. **Client Caching**: Key Vault clients cached per vault URL
2. **Lazy Resolution**: Only resolve if references detected
3. **Parallel Resolution**: Resolve multiple secrets in parallel (future enhancement)
4. **Detection**: Fast string prefix check before regex matching

**Expected Performance:**
- No references: <1ms overhead (string check only)
- First resolution: ~100-500ms (authentication + HTTPS)
- Cached client: ~50-100ms (HTTPS call only)

### Security Considerations

1. **Credential Management**
   - Use `DefaultAzureCredential` (same as azd)
   - No credential storage in extension
   - Leverages existing Azure authentication

2. **Secret Handling**
   - Secrets never logged
   - Secrets never displayed in output (except when explicitly requested)
   - Resolved values only in process memory
   - No secret caching (always fetch fresh)

3. **Error Messages**
   - May reveal vault/secret names in warnings
   - Consider masking option for production
   - Document secure logging practices

4. **Audit Trail**
   - All secret access logged in Azure Key Vault
   - Users should monitor audit logs
   - Document audit log review process

## Testing Strategy

### Unit Tests
- Pattern matching validation
- Reference format parsing
- Client caching logic
- Error scenarios
- Target: 80% code coverage

### Integration Tests
- Real Azure Key Vault resolution
- Both reference formats
- Error handling (invalid vault, missing secret)
- Requires test Key Vault setup

### E2E Tests
- Full command execution with Key Vault references
- `azd app run` with Key Vault env vars
- Hook execution with secrets
- Multiple services with different vaults

### Manual Testing Scenarios

1. **Happy Path**
   ```bash
   # Create Key Vault secret
   az keyvault secret set --vault-name myvault --name db-password --value "secret123"
   
   # Set env var with reference
   export DATABASE_PASSWORD="@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"
   
   # Run service - password automatically resolved
   azd app run --service api
   ```

2. **Missing Vault**
   ```bash
   export API_KEY="@Microsoft.KeyVault(VaultName=nonexistent;SecretName=key)"
   azd app run
   # Should warn but continue
   ```

3. **No Credentials**
   ```bash
   # Without Azure login
   export SECRET="@Microsoft.KeyVault(VaultName=test;SecretName=secret)"
   azd app run
   # Should warn but continue
   ```

4. **Mixed Environment**
   ```bash
   export NORMAL_VAR="plain value"
   export SECRET_VAR="@Microsoft.KeyVault(VaultName=vault;SecretName=secret)"
   export ANOTHER_VAR="another plain value"
   azd app run
   # Should resolve SECRET_VAR, leave others unchanged
   ```

## Migration Guide

### For Users

No migration required! Key Vault resolution is:
- Opt-in by using reference format
- Backward compatible
- No changes to existing workflows

### For Developers

If you maintain code that calls `service.ResolveEnvironment()`:

**Before:**
```go
env, err := service.ResolveEnvironment(svc, azureEnv, dotEnvPath, serviceURLs)
```

**After:**
```go
env, err := service.ResolveEnvironment(ctx, svc, azureEnv, dotEnvPath, serviceURLs)
```

## Success Criteria

- ✅ All P0 commands support Key Vault resolution
- ✅ Unit tests with 80% coverage
- ✅ Integration tests pass
- ✅ Documentation complete
- ✅ No breaking changes to user workflows
- ✅ Performance overhead <1ms when no references
- ✅ Graceful error handling with warnings
- ✅ Real-world testing with Azure Key Vault

## Future Enhancements

1. **Configuration Options**
   - `--kv-resolve=false` flag to disable
   - `--kv-warn=false` to suppress warnings
   - Timeout configuration

2. **Performance**
   - Parallel resolution of multiple secrets
   - TTL-based caching of secret values
   - Pre-fetching on service initialization

3. **Additional Features**
   - Support for Key Vault certificates
   - Managed identity support
   - Offline mode with cached secrets

4. **Dashboard Integration**
   - Show resolution status per service
   - Mask secret values in UI
   - Resolution metrics and timing

## Dependencies

### Direct Dependencies
```
github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.4.0
```

### Transitive Dependencies
- `github.com/Azure/azure-sdk-for-go/sdk/azcore` v1.20.0
- MSAL for Go v1.6.0 (authentication)
- Standard Go packages

All dependencies are from Microsoft and well-maintained.

## Timeline Estimate

- **Phase 1** (Core Infrastructure): 1-2 days
- **Phase 2** (Service Integration): 1 day
- **Phase 3** (Command Coverage): 1 day
- **Phase 4** (Documentation): 1 day
- **Phase 5** (Advanced Features): 2-3 days (optional)

**Total**: 4-5 days for P0+P1, 7-8 days including P2

## Open Questions

1. Should we add a global flag `--no-kv-resolve` to disable resolution?
2. Should we cache resolved secret values (with TTL)?
3. Should we support Azure Key Vault certificates?
4. Should we add metrics/telemetry for resolution performance?

## References

- [azd-exec Key Vault Implementation](https://github.com/jongio/azd-exec/blob/main/cli/src/internal/executor/keyvault.go)
- [Azure Key Vault SDK Documentation](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets)
- [DefaultAzureCredential Documentation](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential)
