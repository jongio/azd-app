<!-- NEXT: 1 -->
# Key Vault Resolution - Tasks

## Phase 1: Core Infrastructure (P0)

### 1. Add Azure SDK Dependencies
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 15 minutes

Add Azure Key Vault SDK dependencies to go.mod.

**Actions**:
1. Run: `cd cli && go get github.com/Azure/azure-sdk-for-go/sdk/azidentity@v1.13.1`
2. Run: `cd cli && go get github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets@v1.4.0`
3. Verify: `go mod tidy`
4. Verify: `mage build`

**Files**:
- `cli/go.mod` (modified)
- `cli/go.sum` (modified)

**Acceptance Criteria**:
- Dependencies added to go.mod
- Build succeeds
- No version conflicts

---

### 2. Create Key Vault Resolver Package
**Status**: TODO
**Priority**: P0 - Critical  
**Estimated**: 2-3 hours
**Depends On**: Task 1

Port Key Vault resolver from azd-exec with adaptations for azd-app.

**Actions**:
1. Create `cli/src/internal/keyvault/keyvault.go`
2. Port `KeyVaultResolver` struct from azd-exec
3. Port regex patterns for reference matching
4. Implement `NewKeyVaultResolver()` using DefaultAzureCredential
5. Implement `ResolveReference()` for single reference resolution
6. Implement `ResolveEnvironmentMap()` for map[string]string (NEW - different from azd-exec)
7. Implement `IsKeyVaultReference()` helper
8. Implement `resolveBySecretURI()` for SecretUri format
9. Implement `resolveByVaultNameAndSecret()` for VaultName format
10. Implement `getClient()` with caching and RWMutex
11. Implement `getSecretValue()` to fetch from Azure
12. Add comprehensive godoc comments

**Interface**:
```go
package keyvault

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

**Files**:
- `cli/src/internal/keyvault/keyvault.go` (new, ~200 lines)

**Acceptance Criteria**:
- All functions implemented
- godoc comments complete
- Compiles without errors
- Supports both reference formats
- Client caching works
- Graceful error handling

---

### 3. Create Unit Tests
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 2-3 hours
**Depends On**: Task 2

Create comprehensive unit tests for Key Vault resolver.

**Actions**:
1. Create `cli/src/internal/keyvault/keyvault_test.go`
2. Test `IsKeyVaultReference()` with valid/invalid references
3. Test reference pattern matching (both formats)
4. Test reference parsing (extract vault, secret, version)
5. Test `ResolveEnvironmentMap()` with mixed env vars
6. Test error scenarios (invalid format, empty reference)
7. Test client caching logic
8. Mock-based tests (no Azure connection required)
9. Verify 80%+ code coverage

**Test Cases**:
```go
func TestIsKeyVaultReference(t *testing.T)
func TestKeyVaultReferencePatterns(t *testing.T)
func TestResolveEnvironmentMap_NoReferences(t *testing.T)
func TestResolveEnvironmentMap_WithReferences(t *testing.T)
func TestKeyVaultResolver_InvalidFormats(t *testing.T)
func TestKeyVaultResolver_ClientCaching(t *testing.T)
```

**Files**:
- `cli/src/internal/keyvault/keyvault_test.go` (new, ~300 lines)

**Acceptance Criteria**:
- 80%+ code coverage
- All tests pass
- No Azure dependencies in unit tests
- Fast execution (<1 second)

---

### 4. Create Integration Tests
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 1 hour
**Depends On**: Task 2

Create integration tests that use real Azure Key Vault.

**Actions**:
1. Create `cli/src/internal/keyvault/keyvault_integration_test.go`
2. Add build tag: `//go:build integration`
3. Test real Key Vault resolution (requires Azure credentials)
4. Test both reference formats with real vault
5. Test error scenarios (invalid vault, missing secret)
6. Document test setup requirements

**Test Setup**:
```bash
# Required environment variables for integration tests
export TEST_KEYVAULT_NAME=test-vault
export TEST_KEYVAULT_SECRET_NAME=test-secret

# Azure credentials required (az login or DefaultAzureCredential)
```

**Files**:
- `cli/src/internal/keyvault/keyvault_integration_test.go` (new, ~200 lines)

**Acceptance Criteria**:
- Tests tagged with `//go:build integration`
- Skipped in normal test runs
- Pass when run with `go test -tags integration`
- Clear documentation for setup

---

## Phase 2: Service Integration (P0)

### 5. Add Context Parameter to ResolveEnvironment
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 1-2 hours
**Depends On**: Task 2

Modify `ResolveEnvironment()` signature to accept context for Key Vault resolution.

**Current Signature**:
```go
func ResolveEnvironment(service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error)
```

**New Signature**:
```go
func ResolveEnvironment(ctx context.Context, service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error)
```

**Actions**:
1. Modify function signature in `cli/src/internal/service/environment.go`
2. Find all callers using grep: `grep -r "ResolveEnvironment" cli/src`
3. Update each caller to pass context (estimated 10-15 call sites)
4. Verify compilation
5. Run existing tests to ensure no breakage

**Callers to Update** (estimated):
- `cli/src/cmd/app/commands/run.go`
- `cli/src/cmd/app/commands/service_control.go`
- `cli/src/internal/service/*.go`
- Test files

**Files**:
- `cli/src/internal/service/environment.go` (modified)
- ~10-15 caller files (modified)

**Acceptance Criteria**:
- Signature updated
- All callers updated
- Compiles without errors
- All existing tests pass

---

### 6. Implement Key Vault Resolution in ResolveEnvironment
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 1-2 hours
**Depends On**: Tasks 2, 5

Add Key Vault reference resolution logic to `ResolveEnvironment()`.

**Actions**:
1. Add `hasKeyVaultReferences()` helper function
2. Add Key Vault resolution logic after environment merging
3. Implement graceful error handling with warnings
4. Add logging for resolution attempts
5. Ensure backward compatibility

**Implementation**:
```go
func ResolveEnvironment(ctx context.Context, service Service, azureEnv map[string]string, dotEnvPath string, serviceURLs map[string]string) (map[string]string, error) {
    // ... existing logic to build env map ...
    
    // NEW: Resolve Key Vault references
    if hasKeyVaultReferences(env) {
        resolver, err := keyvault.NewKeyVaultResolver()
        if err != nil {
            // Log warning but continue
            log.Printf("Warning: Failed to create Key Vault resolver: %v", err)
            log.Printf("Key Vault references will not be resolved")
        } else {
            resolvedEnv, err := resolver.ResolveEnvironmentMap(ctx, env)
            if err != nil {
                // Log warning but continue with unresolved values
                log.Printf("Warning: Failed to resolve some Key Vault references: %v", err)
            } else {
                env = resolvedEnv
            }
        }
    }
    
    return env, nil
}

func hasKeyVaultReferences(env map[string]string) bool {
    for _, value := range env {
        if keyvault.IsKeyVaultReference(value) {
            return true
        }
    }
    return false
}
```

**Files**:
- `cli/src/internal/service/environment.go` (modified)

**Acceptance Criteria**:
- Key Vault resolution integrated
- Graceful error handling
- Warning messages for failures
- Backward compatible
- No breaking changes

---

### 7. Test Service Integration
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 1 hour
**Depends On**: Task 6

Add tests for Key Vault resolution in environment merging.

**Actions**:
1. Add tests to `cli/src/internal/service/environment_test.go`
2. Test environment merging with Key Vault references
3. Test priority order (service env > .env > azure env > OS env)
4. Test error scenarios
5. Mock Key Vault resolver for unit tests

**Test Cases**:
```go
func TestResolveEnvironment_WithKeyVaultReferences(t *testing.T)
func TestResolveEnvironment_KeyVaultError(t *testing.T)
func TestResolveEnvironment_MixedEnvironment(t *testing.T)
```

**Files**:
- `cli/src/internal/service/environment_test.go` (modified)

**Acceptance Criteria**:
- Tests cover Key Vault integration
- Tests pass
- Mock resolver for unit tests (no Azure dependencies)

---

## Phase 3: Hook Integration (P0)

### 8. Modify Hook Environment Building
**Status**: TODO
**Priority**: P0 - Critical
**Estimated**: 2 hours
**Depends On**: Task 6

Update hook execution to use `ResolveEnvironment()` for Key Vault support.

**Current Implementation** (in `run_hooks_test.go`):
```go
func buildHookEnvironmentVariables(services []*service.Service) []string {
    envMap := make(map[string]string)
    for _, svc := range services {
        for k, v := range svc.GetEnvironment() {
            envMap[k] = v
        }
    }
    // Convert to []string...
}
```

**New Implementation**:
```go
func buildHookEnvironmentVariables(ctx context.Context, services []*service.Service, azureEnv map[string]string, dotEnvPath string) ([]string, error) {
    envMap := make(map[string]string)
    
    // For each service, use ResolveEnvironment() to get resolved env
    for _, svc := range services {
        resolved, err := service.ResolveEnvironment(ctx, *svc, azureEnv, dotEnvPath, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve environment for service %s: %w", svc.Name, err)
        }
        for k, v := range resolved {
            envMap[k] = v
        }
    }
    
    // Convert to []string
    var env []string
    for k, v := range envMap {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    return env, nil
}
```

**Actions**:
1. Modify `buildHookEnvironmentVariables()` signature
2. Call `ResolveEnvironment()` for each service
3. Update all callers (prerun, postrun hooks)
4. Add error handling
5. Update tests

**Files**:
- `cli/src/cmd/app/commands/run.go` (modified - hook execution)
- `cli/src/cmd/app/commands/run_hooks_test.go` (modified)

**Acceptance Criteria**:
- Hooks receive resolved Key Vault values
- Tests updated
- All hook tests pass
- No breaking changes to hook behavior

---

## Phase 4: Command Testing (P1)

### 9. Test Run Command with Key Vault
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 1 hour
**Depends On**: Task 6

Create integration test for `azd app run` with Key Vault references.

**Actions**:
1. Create test in `cli/src/cmd/app/commands/run_test.go`
2. Set environment variable with Key Vault reference
3. Run command in test mode
4. Verify service receives resolved value
5. Test error scenarios

**Test Case**:
```go
func TestRunCommand_KeyVaultResolution(t *testing.T) {
    // Requires integration test setup
    os.Setenv("DATABASE_URL", "@Microsoft.KeyVault(VaultName=test;SecretName=db-url)")
    defer os.Unsetenv("DATABASE_URL")
    
    // Run command
    // Verify service receives resolved value
}
```

**Files**:
- `cli/src/cmd/app/commands/run_integration_test.go` (new or modify existing)

**Acceptance Criteria**:
- Integration test passes with real Key Vault
- Test documented with setup requirements

---

### 10. Test Service Control Commands
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 1 hour
**Depends On**: Task 6

Test start/restart commands with Key Vault references.

**Actions**:
1. Create tests for start/restart with Key Vault env vars
2. Verify resolution happens on each start
3. Test secret rotation scenario
4. Test error handling

**Files**:
- New test file or extend existing service control tests

**Acceptance Criteria**:
- Start/restart tests pass
- Secrets re-resolved on each start
- Error scenarios handled

---

### 11. Test Command with Key Vault
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 1 hour
**Depends On**: Task 6

Verify test command environment inheritance and Key Vault resolution.

**Actions**:
1. Investigate how test runner gets environment
2. Ensure resolved environment passed to tests
3. Add integration test for test command
4. Document any special handling needed

**Files**:
- `cli/src/cmd/app/commands/test_test.go` (or new integration test)

**Acceptance Criteria**:
- Tests receive resolved Key Vault values
- Integration tests pass
- Documented behavior

---

## Phase 5: Documentation (P1)

### 12. Update README
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 30 minutes
**Depends On**: Tasks 1-11

Add Key Vault integration section to main README.

**Actions**:
1. Add "🔐 Azure Key Vault Integration" section
2. Document supported reference formats
3. Add usage examples
4. Document authentication methods
5. Add troubleshooting section

**Files**:
- `cli/README.md` (modified)

**Acceptance Criteria**:
- Clear documentation
- Examples work
- Authentication explained

---

### 13. Create Feature Documentation
**Status**: TODO
**Priority**: P1 - High
**Estimated**: 1 hour
**Depends On**: Tasks 1-11

Create comprehensive Key Vault feature documentation.

**Actions**:
1. Create `cli/docs/features/keyvault.md`
2. Document all reference formats with examples
3. Document authentication flow
4. Add troubleshooting guide
5. Document security best practices
6. Add FAQ section

**Content Outline**:
- Overview
- Reference Formats
- Usage Examples (per command)
- Authentication
- Error Handling
- Security Considerations
- Troubleshooting
- FAQ

**Files**:
- `cli/docs/features/keyvault.md` (new)

**Acceptance Criteria**:
- Comprehensive documentation
- Examples for all commands
- Troubleshooting guide complete

---

### 14. Create Demo Examples
**Status**: TODO
**Priority**: P2 - Medium
**Estimated**: 30 minutes
**Depends On**: Tasks 1-11

Create demo scripts showing Key Vault usage.

**Actions**:
1. Create `cli/examples/keyvault-demo.sh` (Bash)
2. Create `cli/examples/keyvault-demo.ps1` (PowerShell)
3. Show setup steps
4. Show all reference formats
5. Show error scenarios

**Files**:
- `cli/examples/keyvault-demo.sh` (new)
- `cli/examples/keyvault-demo.ps1` (new)

**Acceptance Criteria**:
- Working demo scripts
- Clear setup instructions
- Both formats demonstrated

---

### 15. Update Command Help Text
**Status**: TODO
**Priority**: P2 - Medium
**Estimated**: 30 minutes
**Depends On**: Tasks 1-11

Update command help to mention Key Vault support.

**Actions**:
1. Update `azd app run --help` text
2. Update `azd app start --help` text
3. Update `azd app test --help` text
4. Add Key Vault examples to help text

**Files**:
- `cli/src/cmd/app/commands/run.go` (modified)
- `cli/src/cmd/app/commands/start.go` (modified)
- `cli/src/cmd/app/commands/test.go` (modified)

**Acceptance Criteria**:
- Help text mentions Key Vault
- Examples in help text
- Clear and concise

---

## Phase 6: Advanced Features (P2)

### 16. Dashboard Secret Masking
**Status**: TODO
**Priority**: P2 - Medium
**Estimated**: 2-3 hours
**Depends On**: Task 6

Add Key Vault support to dashboard environment panel with secret masking.

**Actions**:
1. Modify environment API to flag Key Vault references
2. Add `isKeyVaultReference` and `isResolved` fields
3. Update `EnvironmentPanel.tsx` to mask resolved secrets
4. Add "Reveal" button with confirmation
5. Show "Resolved from Key Vault" badge
6. Add tooltip with reference string

**UI Design**:
```tsx
{isKeyVaultReference && isResolved ? (
  <div>
    <span>********</span>
    <Badge>Key Vault</Badge>
    <Button onClick={handleReveal}>Reveal</Button>
  </div>
) : (
  <span>{value}</span>
)}
```

**Files**:
- `cli/dashboard/src/components/EnvironmentPanel.tsx` (modified)
- `cli/src/internal/dashboard/server.go` (modified - API endpoint)

**Acceptance Criteria**:
- Secrets masked by default
- Reveal requires confirmation
- Badge shows Key Vault source
- Tooltip shows reference

---

### 17. Info Command Secret Handling
**Status**: TODO
**Priority**: P2 - Medium
**Estimated**: 1 hour
**Depends On**: Task 6

Add `--resolve-secrets` flag to info command.

**Actions**:
1. Add `--resolve-secrets` flag to info command
2. Default behavior: show references, not values
3. With flag: show resolved values with warning
4. Add security warning to help text

**Implementation**:
```bash
# Default (safe)
azd app info
# Shows: DATABASE_PASSWORD=@Microsoft.KeyVault(...)

# With flag (warning displayed)
azd app info --resolve-secrets
# WARNING: This will display actual secret values
# Shows: DATABASE_PASSWORD=actual_secret_value
```

**Files**:
- `cli/src/cmd/app/commands/info.go` (modified)

**Acceptance Criteria**:
- Default behavior safe (shows references)
- Flag works with warning
- Help text explains security implications

---

### 18. Add Diagnostic Logging
**Status**: TODO
**Priority**: P3 - Low
**Estimated**: 1 hour
**Depends On**: Task 6

Add verbose logging for Key Vault resolution.

**Actions**:
1. Add debug logging to Key Vault resolver
2. Log which secrets are being resolved
3. Log resolution timing
4. Log cache hits/misses
5. Add `--verbose` support

**Log Output**:
```
[DEBUG] Detected 2 Key Vault references in environment
[DEBUG] Resolving: DATABASE_PASSWORD from vault 'myvault'
[DEBUG] Resolution took 150ms (cached client)
[DEBUG] Resolving: API_KEY from vault 'ops-vault'
[DEBUG] Resolution took 120ms (cached client)
```

**Files**:
- `cli/src/internal/keyvault/keyvault.go` (modified)

**Acceptance Criteria**:
- Verbose logging works
- Timing information accurate
- No secrets in logs

---

### 19. Performance Metrics
**Status**: TODO
**Priority**: P3 - Low
**Estimated**: 1 hour
**Depends On**: Task 18

Add performance metrics for Key Vault resolution.

**Actions**:
1. Track resolution attempts
2. Track cache hits/misses
3. Track resolution duration
4. Display metrics in verbose mode

**Files**:
- `cli/src/internal/keyvault/keyvault.go` (modified)

**Acceptance Criteria**:
- Metrics tracked
- Displayed in verbose mode
- Useful for debugging

---

## Testing Checklist

### Unit Tests
- [ ] Key Vault resolver pattern matching
- [ ] Reference format parsing
- [ ] Environment map resolution
- [ ] Error scenarios
- [ ] Client caching

### Integration Tests
- [ ] Real Azure Key Vault resolution
- [ ] Both reference formats
- [ ] Error scenarios (invalid vault, missing secret)
- [ ] Multiple vaults

### E2E Tests
- [ ] `azd app run` with Key Vault env vars
- [ ] `azd app start` with Key Vault env vars
- [ ] `azd app test` with Key Vault env vars
- [ ] Hook execution with Key Vault references
- [ ] Multiple services with different vaults

### Manual Testing
- [ ] Happy path (valid reference, exists)
- [ ] Missing vault
- [ ] Missing secret
- [ ] No Azure credentials
- [ ] Mixed environment (some KV, some plain)
- [ ] Service restart re-resolves secrets
- [ ] Dashboard displays masked secrets
- [ ] Info command behavior

---

## Done

_Completed tasks will be moved here as work progresses_
