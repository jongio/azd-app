# Design Evaluation: Key Vault Resolution Integration

## Core Design Decision

**Question**: Where should Key Vault reference resolution happen?

## Option 1: Central Integration in ResolveEnvironment() ⭐ SELECTED

### Implementation
```go
// In cli/src/internal/service/environment.go
func ResolveEnvironment(ctx context.Context, service Service, ...) (map[string]string, error) {
    // Existing environment merging logic
    env := make(map[string]string)
    // ... merge OS env, Azure env, .env file, service env ...
    
    // NEW: Key Vault resolution (single integration point)
    if hasKeyVaultReferences(env) {
        resolver, err := keyvault.NewKeyVaultResolver()
        if err != nil {
            logWarning("Failed to create Key Vault resolver: %v", err)
        } else {
            env, _ = resolver.ResolveEnvironmentMap(ctx, env)
        }
    }
    
    return env, nil
}
```

### Pros ✅
1. **Single Point of Integration**
   - Modify one function
   - All commands benefit automatically
   - Consistent behavior across entire codebase

2. **Minimal Code Changes**
   - ~20 lines in `ResolveEnvironment()`
   - ~5 lines per caller (add `ctx` parameter)
   - Total: ~50-100 lines modified

3. **Automatic Coverage**
   - `azd app run` ✅
   - `azd app start` ✅
   - `azd app restart` ✅
   - `azd app test` ✅
   - Hook execution ✅
   - Any future commands ✅

4. **Easy to Test**
   - Test `ResolveEnvironment()` once
   - All commands automatically tested
   - Single unit test suite

5. **Easy to Maintain**
   - Changes in one place
   - Clear ownership
   - Single source of truth

6. **Performance Optimized**
   - Single fast-path check: `hasKeyVaultReferences()`
   - No overhead if no references (<1ms)
   - Client caching works globally

### Cons ❌
1. **Function Signature Change**
   - Need to add `ctx context.Context` parameter
   - Requires updating 10-15 call sites
   - **Mitigation**: Compiler catches all call sites, mechanical change

2. **Slightly Less Granular Control**
   - Can't disable for specific commands easily
   - **Mitigation**: Add feature flag if needed later

3. **Context Required**
   - Callers must have context available
   - **Mitigation**: All callers already have context

### Impact Analysis
```
Commands automatically covered: 10+
Lines of code changed: ~50-100
Test suites to update: 1 (environment_test.go)
Risk level: LOW ✅
Effort: 2-3 days
```

---

## Option 2: Per-Command Integration ❌ NOT SELECTED

### Implementation
```go
// In EACH command file
func runCommand(ctx context.Context, ...) error {
    // Get environment
    env := service.GetEnvironment()
    
    // Resolve Key Vault references (REPEATED IN EACH COMMAND)
    if hasKeyVaultReferences(env) {
        resolver, _ := keyvault.NewKeyVaultResolver()
        env, _ = resolver.ResolveEnvironmentMap(ctx, env)
    }
    
    // Use resolved env
    cmd.Env = toEnvArray(env)
    return cmd.Run()
}
```

### Pros ✅
1. **Fine-Grained Control**
   - Can enable/disable per command
   - Can customize error handling per command

2. **No Signature Changes**
   - Don't need to modify `ResolveEnvironment()`
   - No cascading changes

### Cons ❌
1. **Code Duplication**
   - Same logic copied to 10+ commands
   - Maintenance nightmare
   - Inconsistent implementations likely

2. **Easy to Miss Commands**
   - New commands may forget to add resolution
   - Existing commands may be missed
   - No compiler enforcement

3. **More Code to Change**
   - 10-15 command files to modify
   - ~500-1000 lines of code changes
   - Higher bug risk

4. **Harder to Test**
   - Need tests for each command
   - More test code to maintain
   - Inconsistent test coverage

5. **Performance Issues**
   - Multiple resolver creations
   - Cache doesn't work across commands
   - More overhead

### Impact Analysis
```
Commands to manually modify: 10+
Lines of code changed: ~500-1000
Test suites to update: 10+
Risk level: MEDIUM-HIGH ❌
Effort: 5-7 days
```

---

## Option 3: Middleware/Interceptor Pattern

### Implementation
```go
// Create command middleware
type CommandMiddleware interface {
    Before(ctx context.Context, env map[string]string) (map[string]string, error)
    After(ctx context.Context, result interface{}) error
}

type KeyVaultMiddleware struct {
    resolver *keyvault.KeyVaultResolver
}

func (m *KeyVaultMiddleware) Before(ctx context.Context, env map[string]string) (map[string]string, error) {
    return m.resolver.ResolveEnvironmentMap(ctx, env)
}

// Register middleware globally
RegisterMiddleware(&KeyVaultMiddleware{})
```

### Pros ✅
1. **Extensible Pattern**
   - Easy to add more middleware
   - Clean separation of concerns
   - Testable in isolation

2. **Global Configuration**
   - Single place to enable/disable
   - Can add priority/ordering

3. **No Command Changes**
   - Commands don't need modification
   - Transparent injection

### Cons ❌
1. **Over-Engineering**
   - Adds complexity for single feature
   - Need middleware framework
   - More abstraction layers

2. **Hidden Behavior**
   - Less obvious what's happening
   - Harder to debug
   - Implicit behavior can be confusing

3. **More Code**
   - Need middleware infrastructure (~200 lines)
   - Need registration system
   - Need execution framework

4. **Not Idiomatic for Go**
   - Go prefers explicit over implicit
   - Middleware more common in web frameworks

### Impact Analysis
```
New infrastructure needed: Yes (~200 lines)
Commands to modify: 0
Risk level: MEDIUM (new pattern)
Effort: 4-5 days
```

---

## Comparison Matrix

| Aspect | Option 1: Central | Option 2: Per-Command | Option 3: Middleware |
|--------|------------------|----------------------|---------------------|
| **Code Changes** | ~50-100 lines | ~500-1000 lines | ~300 lines |
| **Risk** | LOW ✅ | MEDIUM-HIGH ❌ | MEDIUM |
| **Maintenance** | EASY ✅ | HARD ❌ | MEDIUM |
| **Consistency** | GUARANTEED ✅ | AT RISK ❌ | GOOD |
| **Test Effort** | LOW ✅ | HIGH ❌ | MEDIUM |
| **Extensibility** | GOOD | POOR ❌ | EXCELLENT ✅ |
| **Go Idioms** | EXCELLENT ✅ | GOOD | FAIR |
| **Debug-ability** | EXCELLENT ✅ | GOOD | FAIR |
| **Performance** | OPTIMAL ✅ | SUBOPTIMAL ❌ | GOOD |
| **Effort** | 2-3 days ✅ | 5-7 days ❌ | 4-5 days |

---

## Decision Rationale

### Why Option 1 is Best

1. **Existing Architecture Supports It**
   - `ResolveEnvironment()` already exists as the integration point
   - All commands already use it
   - Natural fit with current design

2. **Principle of Least Surprise**
   - Developers expect environment resolution to happen in `ResolveEnvironment()`
   - No hidden middleware magic
   - Explicit and clear

3. **Go Best Practices**
   - Explicit over implicit ✅
   - Simple over complex ✅
   - Less code is better ✅

4. **Risk Management**
   - Compiler-enforced changes (context parameter)
   - Single point of failure (easier to fix)
   - Isolated to one package

5. **Future-Proof**
   - Easy to add feature flags
   - Easy to extend with other secret sources
   - Easy to add caching/optimization

### Trade-offs Accepted

1. **Signature Change**: Acceptable
   - Internal API only
   - Compiler catches all call sites
   - Mechanical change, low risk

2. **Context Required**: Acceptable
   - All callers already have context
   - Best practice in Go
   - Enables proper cancellation

3. **Less Granular Control**: Acceptable
   - Can add feature flags if needed
   - Consistent behavior is more important
   - YAGNI (You Aren't Gonna Need It)

---

## Alternative Considered: Decorator Pattern

Could wrap `ResolveEnvironment()`:

```go
func ResolveEnvironmentWithKeyVault(ctx context.Context, ...) (map[string]string, error) {
    env, err := ResolveEnvironment(service, azureEnv, ...)
    if err != nil {
        return nil, err
    }
    
    // Add Key Vault resolution
    return resolveKeyVaultReferences(ctx, env)
}
```

**Rejected Because**:
- ❌ Two functions doing similar things (confusing)
- ❌ Need to update all call sites anyway
- ❌ No benefit over modifying original function
- ❌ Creates tech debt (which function to use?)

---

## Implementation Strategy

### Phase 1: Foundation
1. Create `keyvault` package (isolated, testable)
2. Unit tests (no dependencies on rest of codebase)
3. Integration tests (optional, requires Azure)

### Phase 2: Integration
4. Add `ctx context.Context` to `ResolveEnvironment()` signature
5. Compiler shows all call sites that need updating
6. Update each call site mechanically (add `ctx`)
7. Run existing tests - should all pass

### Phase 3: Key Vault Logic
8. Add Key Vault resolution to `ResolveEnvironment()`
9. Add `hasKeyVaultReferences()` helper
10. Add error handling and logging
11. Test with real Key Vault references

### Phase 4: Polish
12. Update documentation
13. Add examples
14. Performance testing
15. Security review

---

## Risk Mitigation

### Risk: Breaking Existing Functionality
**Mitigation**:
- Extensive unit tests
- Integration tests
- Graceful error handling (warnings only)
- Feature can be disabled if needed

### Risk: Performance Regression
**Mitigation**:
- Fast-path for no references (<1ms)
- Client caching
- Performance benchmarks
- Monitoring in tests

### Risk: Security Issues
**Mitigation**:
- Use Microsoft's official SDK
- DefaultAzureCredential (well-tested)
- No secret caching
- Comprehensive security review

### Risk: Complexity Creep
**Mitigation**:
- Simple design (single integration point)
- Clear documentation
- Examples for common scenarios
- No over-engineering

---

## Success Metrics

### Functionality ✅
- [ ] All P0 commands support Key Vault
- [ ] Both reference formats work
- [ ] Error handling is graceful
- [ ] Tests have 80% coverage

### Performance ✅
- [ ] No references: <1ms overhead
- [ ] First resolution: <500ms
- [ ] Cached resolution: <100ms

### Code Quality ✅
- [ ] No duplication
- [ ] Clear, idiomatic Go
- [ ] Well-documented
- [ ] Comprehensive tests

### User Experience ✅
- [ ] Works out of the box
- [ ] Clear error messages
- [ ] Good documentation
- [ ] Examples provided

---

## Conclusion

**Option 1: Central Integration** is the clear winner:

- ✅ Lowest risk
- ✅ Least code
- ✅ Best maintainability
- ✅ Most consistent
- ✅ Fastest implementation
- ✅ Most testable

This design leverages the existing architecture perfectly and follows Go best practices. The single integration point makes it easy to implement, test, and maintain.
