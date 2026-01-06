# Command-by-Command Analysis for Key Vault Resolution

## Executive Summary

**Key Finding**: All commands that use environment variables flow through a **single choke point**: `service.ResolveEnvironment()` in `cli/src/internal/service/environment.go`. This makes implementation extremely clean - we modify one function and all commands benefit.

## Integration Architecture

```
┌────────────────────────────────────────────────────────────┐
│                   All Commands                              │
│  run, start, restart, test, hooks, logs, etc.              │
└──────────────────────┬─────────────────────────────────────┘
                       │
                       ├──► Direct calls to ResolveEnvironment()
                       ├──► ServiceController.loadEnvVars()
                       └──► executor.ExecuteHook()
                       │
                       ▼
┌────────────────────────────────────────────────────────────┐
│        service.ResolveEnvironment() ⭐ INTEGRATION POINT    │
│                                                             │
│  CURRENT FLOW:                                             │
│  1. Start with OS environment                              │
│  2. Merge Azure environment                                │
│  3. Merge .env file                                        │
│  4. Merge service URLs                                     │
│  5. Merge service-specific env                             │
│  6. Return merged map                                      │
│                                                             │
│  NEW STEP (add after step 5):                              │
│  6. Resolve Key Vault references ⭐ NEW                     │
│  7. Return merged map with resolved secrets                │
└────────────────────────────────────────────────────────────┘
                       │
                       ▼
┌────────────────────────────────────────────────────────────┐
│          keyvault.ResolveEnvironmentMap()                   │
│  - Iterate through map                                     │
│  - Detect Key Vault references                             │
│  - Resolve each reference                                  │
│  - Replace with actual secret value                        │
│  - Graceful error handling                                 │
└────────────────────────────────────────────────────────────┘
```

## Command Analysis

### Category 1: Service Execution Commands

#### 1. `azd app run` - Run development environment
**File**: `cli/src/cmd/app/commands/run.go`

**Environment Flow**:
```
runWithServices()
  └─► runAzdMode()
       └─► runServices()
            └─► service.Prepare() for each service
                 └─► runtime.Env populated
                      └─► Uses ResolveEnvironment() ✅
```

**Key Vault Usage**:
- ✅ Automatically supported via `ResolveEnvironment()`
- ✅ All services get resolved secrets
- ✅ Used for database passwords, API keys, etc.

**Test Case**:
```bash
export DATABASE_URL="@Microsoft.KeyVault(VaultName=myvault;SecretName=db-url)"
azd app run --service api
# API service should receive resolved DATABASE_URL
```

---

#### 2. `azd app start` - Start stopped services
**File**: `cli/src/cmd/app/commands/start.go`

**Environment Flow**:
```
runStart()
  └─► ServiceController.StartService()
       └─► loadEnvVars()
            └─► Merges os.Environ() + runtime.Env
                 └─► runtime.Env from ResolveEnvironment() ✅
```

**Key Vault Usage**:
- ✅ Automatically supported via `loadEnvVars()` → `ResolveEnvironment()`
- ✅ Secrets available when service restarts

**Test Case**:
```bash
azd app start --service api
# Should reload Key Vault references
```

---

#### 3. `azd app restart` - Restart services
**File**: `cli/src/cmd/app/commands/restart.go`

**Environment Flow**: Same as `start`

**Key Vault Usage**:
- ✅ Automatically supported
- ✅ Fresh secret resolution on restart

---

#### 4. `azd app test` - Run tests
**File**: `cli/src/cmd/app/commands/test.go`

**Environment Flow**:
```
runTests()
  └─► testing.RunTests()
       └─► Test execution with environment
            └─► Uses process environment
                 └─► Inherits from parent (os.Environ())
                      └─► May need explicit resolution ⚠️
```

**Key Vault Usage**:
- ⚠️ **Needs Investigation**: Tests may need explicit environment resolution
- May require passing resolved environment to test executor
- Tests often need secrets for integration/e2e testing

**Test Case**:
```bash
export DB_PASSWORD="@Microsoft.KeyVault(VaultName=test;SecretName=db-pass)"
azd app test --type integration
# Integration tests should receive resolved password
```

**Action Required**: Verify test runner inherits resolved environment

---

### Category 2: Hook Execution

#### 5. Prerun/Postrun Hooks
**File**: `cli/src/cmd/app/commands/run.go`

**Environment Flow**:
```
executePrerunHook()
  └─► buildHookEnvironmentVariables()
       └─► Builds env map from services
            └─► Each service.GetEnvironment()
                 └─► Returns environment from azure.yaml
                      └─► ⚠️ May not use ResolveEnvironment()
```

**Current Implementation** (from `run_hooks_test.go`):
```go
func buildHookEnvironmentVariables(services []*service.Service) []string {
    envMap := make(map[string]string)
    
    // Merge all service environments
    for _, svc := range services {
        for k, v := range svc.GetEnvironment() {
            envMap[k] = v
        }
    }
    
    // Convert to []string
    var env []string
    for k, v := range envMap {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    return env
}
```

**Key Vault Usage**:
- ⚠️ **Needs Modification**: Hooks build env directly without `ResolveEnvironment()`
- ❌ Currently Key Vault references NOT resolved in hooks
- **Fix**: Call `ResolveEnvironment()` before building hook env

**Action Required**:
```go
// MODIFY buildHookEnvironmentVariables()
func buildHookEnvironmentVariables(ctx context.Context, services []*service.Service, azureEnv map[string]string) ([]string, error) {
    envMap := make(map[string]string)
    
    // For each service, use ResolveEnvironment() instead of GetEnvironment()
    for _, svc := range services {
        resolved, err := service.ResolveEnvironment(ctx, *svc, azureEnv, "", nil)
        if err != nil {
            return nil, err
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

---

#### 6. Service Lifecycle Hooks
**File**: `cli/src/internal/service/hooks.go` (if exists)

**Environment Flow**: Similar to prerun/postrun hooks

**Action Required**: Same fix as above

---

### Category 3: Azure Operations

#### 7. `azd app logs` - Azure log streaming
**File**: `cli/src/cmd/app/commands/logs.go`

**Environment Flow**:
```
runLogs()
  └─► Azure authentication
       └─► Uses DefaultAzureCredential
            └─► Credentials from environment variables
                 └─► os.Environ() (includes AZURE_* vars)
```

**Key Vault Usage**:
- ℹ️ **Indirect Benefit**: Azure credentials may come from Key Vault
- ℹ️ Log queries themselves don't use service environment
- ✅ If workspace ID is in env var with Key Vault reference, it would be resolved

**Test Case**:
```bash
export LOG_ANALYTICS_WORKSPACE="@Microsoft.KeyVault(VaultName=ops;SecretName=workspace-id)"
azd app logs --service api
# Should use resolved workspace ID
```

---

#### 8. `azd app health` - Health checks
**File**: `cli/src/cmd/app/commands/health.go`

**Environment Flow**:
```
runHealth()
  └─► HTTP health check endpoints
       └─► May need authentication headers
            └─► From environment variables
```

**Key Vault Usage**:
- ℹ️ **Potential Benefit**: Health endpoints may require API keys
- ℹ️ Not currently using service environment directly
- ✅ Could benefit if health check credentials stored in Key Vault

---

### Category 4: Information Commands

#### 9. `azd app info` - Environment information
**File**: `cli/src/cmd/app/commands/info.go`

**Environment Flow**: Displays environment variables

**Key Vault Usage**:
- ⚠️ **Security Consideration**: Should we display resolved values or references?
- **Recommendation**: Show references, not resolved secrets
- **Option**: Add `--resolve-secrets` flag (with warning)

**Design Decision Required**:
```bash
# Current behavior (after implementation)
azd app info
# Shows: DATABASE_PASSWORD=@Microsoft.KeyVault(...)

# Possible new flag
azd app info --resolve-secrets
# WARNING: This will display actual secret values
# Shows: DATABASE_PASSWORD=actual_secret_value
```

---

#### 10. Dashboard Environment Panel
**File**: `cli/dashboard/src/components/EnvironmentPanel.tsx`

**Environment Flow**: Displays environment variables from API

**Key Vault Usage**:
- ⚠️ **Security Consideration**: UI should mask secret values
- **Recommendation**: 
  - Show references by default
  - Add "Reveal" button with confirmation
  - Mask resolved values: `DATABASE_PASSWORD=****** (resolved from Key Vault)`

**Design Pattern**:
```tsx
interface EnvironmentVariable {
  key: string
  value: string
  isKeyVaultReference: boolean
  isResolved: boolean
}

// Display logic
{isKeyVaultReference && isResolved ? (
  <span title={value}>
    ******** <Badge>Key Vault</Badge>
  </span>
) : (
  <span>{value}</span>
)}
```

---

## Additional Integration Points

### ServiceController.loadEnvVars()
**File**: `cli/src/cmd/app/commands/service_control.go`

**Current Implementation**:
```go
func (c *ServiceController) loadEnvVars(runtime *service.ServiceRuntime) map[string]string {
    envVars := make(map[string]string)
    
    // Merge os.Environ()
    for _, e := range os.Environ() {
        pair := strings.SplitN(e, "=", 2)
        if len(pair) == 2 {
            envVars[pair[0]] = pair[1]
        }
    }
    
    // Merge runtime.Env (from ResolveEnvironment)
    for k, v := range runtime.Env {
        envVars[k] = v
    }
    
    return envVars
}
```

**Key Vault Support**:
- ✅ `runtime.Env` comes from `ResolveEnvironment()`
- ✅ Automatically includes resolved secrets
- ✅ No changes needed if we modify `ResolveEnvironment()`

---

## Summary of Required Changes

### ✅ Automatic Support (No Changes Required)

These commands automatically get Key Vault resolution by modifying `ResolveEnvironment()`:

1. ✅ `azd app run` - Uses `ResolveEnvironment()` directly
2. ✅ `azd app start` - Uses `runtime.Env` from `ResolveEnvironment()`
3. ✅ `azd app restart` - Same as start
4. ✅ `azd app logs` - Indirect benefit for credentials

### ⚠️ Requires Modification

5. ⚠️ **Hook Execution** - Must modify `buildHookEnvironmentVariables()`
   - Change signature to accept context
   - Call `ResolveEnvironment()` instead of `GetEnvironment()`

6. ⚠️ **Test Command** - May need to pass resolved environment
   - Verify test runner environment inheritance
   - May need explicit environment resolution

### 🔍 Design Decisions Required

7. 🔍 **Info Command** - How to handle secret display?
   - Show references or resolved values?
   - Add `--resolve-secrets` flag?

8. 🔍 **Dashboard** - How to display secrets in UI?
   - Mask resolved values?
   - Show "Resolved from Key Vault" badge?
   - Add reveal button?

---

## Implementation Priority

### Phase 1: Core (P0)
1. Create `keyvault` package
2. Modify `service.ResolveEnvironment()` 
3. Update function signature (add `context.Context`)
4. Update all `ResolveEnvironment()` callers (~10-15 call sites)

### Phase 2: Hooks (P0)
5. Modify `buildHookEnvironmentVariables()` to use `ResolveEnvironment()`
6. Test prerun/postrun hooks with Key Vault references

### Phase 3: Testing (P1)
7. Verify test command environment inheritance
8. Add integration tests for all commands

### Phase 4: UI/UX (P2)
9. Decide on info command behavior
10. Implement dashboard masking/reveal

---

## Test Strategy by Command

### Integration Tests Required

```go
// Test azd app run
func TestRunCommand_KeyVaultResolution(t *testing.T) {
    // Set env var with Key Vault reference
    os.Setenv("DATABASE_URL", "@Microsoft.KeyVault(VaultName=test;SecretName=db-url)")
    defer os.Unsetenv("DATABASE_URL")
    
    // Run command
    // Verify service receives resolved value
}

// Test hooks
func TestHooks_KeyVaultResolution(t *testing.T) {
    // azure.yaml with prerun hook that uses env var
    // Env var has Key Vault reference
    // Verify hook receives resolved value
}

// Test start/restart
func TestServiceControl_KeyVaultResolution(t *testing.T) {
    // Start service with Key Vault env var
    // Stop and restart
    // Verify resolution happens on each start
}
```

---

## Risk Analysis

### Low Risk
- ✅ Core `ResolveEnvironment()` modification: Well-isolated, single function
- ✅ Signature change: Internal API only, compiler catches all call sites
- ✅ Graceful degradation: Warnings only, no hard failures

### Medium Risk
- ⚠️ Hook execution: Requires changing environment building logic
- ⚠️ Test command: May need careful environment handling
- ⚠️ Performance: First resolution may add latency

### Mitigation
- Comprehensive unit and integration tests
- Feature flag option to disable resolution
- Clear documentation and examples
- Phased rollout (run command first, then others)

---

## Performance Impact

### No Key Vault References (Most Common Case)
- Overhead: <1ms (simple string check)
- No network calls
- No credential initialization

### With Key Vault References
- First reference: ~100-500ms (credential + network)
- Subsequent references (same vault): ~50-100ms (cached client)
- Parallel resolution possible (future optimization)

### Optimization Strategy
1. Fast-path detection: Check for "@Microsoft.KeyVault(" prefix
2. Lazy credential creation: Only create when needed
3. Client caching: One client per vault URL
4. Future: Parallel secret resolution

---

## Conclusion

**Key Takeaway**: The implementation is remarkably clean due to the centralized `ResolveEnvironment()` function. Most commands get Key Vault support automatically with **zero changes** once we modify that one function.

**Only 2 areas need explicit work**:
1. Hook execution (modify environment building)
2. Test command (verify environment inheritance)

This is a **low-risk, high-impact feature** with excellent architectural fit.
