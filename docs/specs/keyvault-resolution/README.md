# Key Vault Resolution Implementation

## Documentation Index

📋 **This Document**: Executive Summary
- [Full Specification](spec.md) - Comprehensive technical specification
- [Command Analysis](command-analysis.md) - Detailed analysis of each command
- [Design Evaluation](design-evaluation.md) - Design options and trade-offs
- [Tasks](tasks.md) - Implementation task breakdown

---

# Executive Summary

## What We're Building

Automatic Azure Key Vault reference resolution for all azd-app commands that use environment variables. Users can reference secrets stored in Azure Key Vault using a special syntax, and azd-app will automatically fetch and inject the actual secret values before running services.

## Why This Matters

**Security Benefits:**
- ✅ No secrets in code or configuration files
- ✅ Centralized secret management in Azure Key Vault
- ✅ Azure RBAC-based access control
- ✅ Audit trail of all secret access
- ✅ Automatic secret rotation support

**Developer Experience:**
```bash
# Before: Secrets in code ❌
export DATABASE_PASSWORD="MySecretPassword123!"

# After: Reference to Key Vault ✅
export DATABASE_PASSWORD="@Microsoft.KeyVault(VaultName=myvault;SecretName=db-password)"
```

## Architecture - The Key Insight

**All environment variable usage flows through ONE function**: `service.ResolveEnvironment()`

This is a **centralized choke point** - modify it once, and all commands automatically support Key Vault:

```
┌───────────────────────────────────────────────────┐
│  ALL Commands: run, start, restart, test, etc.   │
└────────────────────┬──────────────────────────────┘
                     │
                     ▼
┌───────────────────────────────────────────────────┐
│  service.ResolveEnvironment() ⭐ ONE CHANGE HERE  │
│  Add: Detect and resolve Key Vault references    │
└────────────────────┬──────────────────────────────┘
                     │
                     ▼
┌───────────────────────────────────────────────────┐
│  keyvault.KeyVaultResolver (NEW PACKAGE)          │
│  - Pattern matching                               │
│  - Azure SDK integration                          │
│  - Client caching                                 │
└───────────────────────────────────────────────────┘
```

## Scope of Impact

### ✅ Automatic Support (Zero Code Changes)
These commands get Key Vault resolution for FREE by modifying `ResolveEnvironment()`:
1. ✅ `azd app run` - Run development services
2. ✅ `azd app start` - Start stopped services  
3. ✅ `azd app restart` - Restart services
4. ✅ `azd app logs` - Azure log streaming (credentials)

### ⚠️ Requires Small Changes (2 areas)
5. ⚠️ **Hook execution** - Modify `buildHookEnvironmentVariables()` to call `ResolveEnvironment()`
6. ⚠️ **Test command** - Verify environment inheritance works

### 🔍 Design Decisions Needed
7. 🔍 **Info command** - Show references or resolved values?
8. 🔍 **Dashboard** - Mask secrets in UI?

## Implementation Complexity

### LOW RISK ✅
- **Single integration point**: Modify one function
- **Compiler-verified**: Context parameter addition caught by compiler
- **Graceful degradation**: Warnings only, no hard failures
- **Well-isolated**: New package, minimal touching of existing code

### Code Changes Estimate
- **New code**: ~500 lines (keyvault package + tests)
- **Modified code**: ~50 lines (ResolveEnvironment + 10-15 call sites)
- **Total effort**: 4-5 days for P0+P1 features

## Reference Formats (Same as azd-exec)

### Format 1: SecretUri
```
@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret)
```

### Format 2: VaultName + SecretName
```
@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret)
```

Both formats support optional version:
```
@Microsoft.KeyVault(SecretUri=https://myvault.vault.azure.net/secrets/my-secret/v1)
@Microsoft.KeyVault(VaultName=myvault;SecretName=my-secret;SecretVersion=v1)
```

## Performance Impact

| Scenario | Overhead |
|----------|----------|
| No Key Vault references (most common) | <1ms (string check) |
| First reference to a vault | ~100-500ms (auth + network) |
| Subsequent references (cached) | ~50-100ms (network only) |

**Optimization**: Client caching ensures minimal overhead after first resolution.

## Error Handling Philosophy

**"Continue on Error"** - Never block service startup due to Key Vault issues:

```go
if hasKeyVaultReferences(env) {
    resolver, err := keyvault.NewKeyVaultResolver()
    if err != nil {
        // Log warning, continue with unresolved values ⚠️
        log.Printf("Warning: Key Vault resolver creation failed: %v", err)
    } else {
        env, err = resolver.ResolveEnvironmentMap(ctx, env)
        if err != nil {
            // Log warning, continue with unresolved values ⚠️
            log.Printf("Warning: Some Key Vault references failed: %v", err)
        }
    }
}
```

**Benefits:**
- ✅ Services run even without Azure credentials
- ✅ Works offline (with warnings)
- ✅ Graceful degradation in error scenarios
- ✅ Clear feedback to users via warnings

## Implementation Phases

### Phase 1: Core Infrastructure (P0) - 2 days
1. Add Azure SDK dependencies
2. Create `keyvault` package (port from azd-exec)
3. Create unit tests (80% coverage)
4. Create integration tests (with real Azure)

### Phase 2: Service Integration (P0) - 1 day
5. Add `context.Context` parameter to `ResolveEnvironment()`
6. Update 10-15 call sites
7. Add Key Vault resolution logic
8. Test integration

### Phase 3: Hook Integration (P0) - 1 day
9. Modify `buildHookEnvironmentVariables()` to use `ResolveEnvironment()`
10. Test prerun/postrun hooks

### Phase 4: Testing (P1) - 1 day
11. Integration tests for all commands
12. E2E tests
13. Manual testing scenarios

### Phase 5: Documentation (P1) - 1 day
14. Update README
15. Create feature documentation
16. Create demo examples
17. Update command help

### Phase 6: Advanced (P2) - 2-3 days (Optional)
18. Dashboard secret masking
19. Info command `--resolve-secrets` flag
20. Diagnostic logging
21. Performance metrics

## Testing Strategy

### Unit Tests (No Azure Required)
```go
func TestIsKeyVaultReference(t *testing.T)
func TestResolveEnvironmentMap_NoReferences(t *testing.T)
func TestResolveEnvironmentMap_WithReferences(t *testing.T)
func TestKeyVaultResolver_InvalidFormats(t *testing.T)
```

### Integration Tests (Real Azure)
```go
//go:build integration

func TestKeyVaultIntegration_SecretUriFormat(t *testing.T)
func TestKeyVaultIntegration_VaultNameFormat(t *testing.T)
func TestKeyVaultIntegration_InvalidVault(t *testing.T)
```

### E2E Tests
```bash
# Set Key Vault reference
export DATABASE_URL="@Microsoft.KeyVault(VaultName=test;SecretName=db-url)"

# Run command - verify resolution
azd app run --service api

# Verify service received resolved value
```

## Security Considerations

### ✅ Strengths
1. **No credential storage** - Uses DefaultAzureCredential
2. **No secret leakage** - Secrets only in process memory
3. **Audit trail** - All access logged in Azure Key Vault
4. **RBAC enforcement** - Azure manages access control

### ⚠️ Considerations
1. **Warning messages may reveal vault/secret names**
   - Recommendation: Document secure logging practices
   - Future: Add `--kv-quiet` flag to suppress names

2. **Dashboard UI displays resolved values**
   - Recommendation: Mask by default, add "Reveal" button
   - Future: Implement in Phase 6

3. **Info command may show secrets**
   - Recommendation: Show references by default
   - Future: Add `--resolve-secrets` flag with warning

## Success Criteria

**Must Have (P0):**
- ✅ `azd app run` resolves Key Vault references
- ✅ `azd app start/restart` resolves Key Vault references
- ✅ Hooks resolve Key Vault references
- ✅ Unit tests with 80% coverage
- ✅ Integration tests pass
- ✅ Graceful error handling
- ✅ No breaking changes

**Should Have (P1):**
- ✅ All commands tested
- ✅ Documentation complete
- ✅ Demo examples working

**Nice to Have (P2):**
- ✅ Dashboard secret masking
- ✅ Diagnostic logging
- ✅ Performance metrics

## Example Usage

### Setting Up
```bash
# 1. Create Key Vault secret
az keyvault secret set \
  --vault-name my-dev-vault \
  --name database-password \
  --value "SuperSecretPassword123!"

# 2. Reference in azure.yaml
services:
  api:
    environment:
      DATABASE_PASSWORD: "@Microsoft.KeyVault(VaultName=my-dev-vault;SecretName=database-password)"

# 3. Run service - password automatically resolved
azd app run --service api
```

### Or Using Environment Variables
```bash
# Set env var with Key Vault reference
export DATABASE_PASSWORD="@Microsoft.KeyVault(VaultName=my-dev-vault;SecretName=database-password)"

# Run service
azd app run --service api
# api service receives: DATABASE_PASSWORD="SuperSecretPassword123!"
```

### Multiple Secrets, Multiple Vaults
```yaml
services:
  api:
    environment:
      # Production database in secure vault
      DATABASE_URL: "@Microsoft.KeyVault(VaultName=prod-vault;SecretName=db-url)"
      
      # API keys in different vault
      OPENAI_API_KEY: "@Microsoft.KeyVault(VaultName=api-keys-vault;SecretName=openai-key)"
      
      # Normal env vars still work
      LOG_LEVEL: "info"
      PORT: "3000"
```

## Migration Path

**For End Users**: ✅ Zero migration required
- Existing workflows unchanged
- Opt-in by using Key Vault references
- Backward compatible

**For Developers**: ⚠️ Minor signature change
```go
// Before
env, err := service.ResolveEnvironment(svc, azureEnv, dotEnvPath, urls)

// After
env, err := service.ResolveEnvironment(ctx, svc, azureEnv, dotEnvPath, urls)
```

Compiler will catch all call sites that need updating.

## Dependencies

```go
// Direct dependencies (from Microsoft, well-maintained)
github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets v1.4.0
```

## Documentation Structure

```
cli/
├── README.md (add Key Vault section)
├── docs/
│   └── features/
│       └── keyvault.md (comprehensive guide)
├── examples/
│   ├── keyvault-demo.sh
│   └── keyvault-demo.ps1
└── src/internal/keyvault/
    ├── keyvault.go
    ├── keyvault_test.go
    └── keyvault_integration_test.go
```

## Next Steps

1. **Review this spec** - Confirm design approach
2. **Start Phase 1** - Add dependencies, create keyvault package
3. **Test early** - Integration tests with real Azure Key Vault
4. **Iterate** - Refine based on real-world usage

## Questions for Discussion

1. **Dashboard masking**: Should we implement in P0 or P2?
2. **Info command**: Default to showing references or resolved values?
3. **Feature flag**: Add `--no-kv-resolve` global flag?
4. **Caching**: Should we cache resolved secret values (with TTL)?
5. **Telemetry**: Track Key Vault resolution metrics?

## References

- **Full Spec**: [docs/specs/keyvault-resolution/spec.md](spec.md)
- **Command Analysis**: [docs/specs/keyvault-resolution/command-analysis.md](command-analysis.md)
- **Tasks**: [docs/specs/keyvault-resolution/tasks.md](tasks.md)
- **azd-exec Implementation**: https://github.com/jongio/azd-exec/blob/main/cli/src/internal/executor/keyvault.go
