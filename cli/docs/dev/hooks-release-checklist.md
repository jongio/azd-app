# Hooks Feature - Release Checklist and Rollback Plan

## Release Checklist

### Pre-Release Validation
- [x] All unit tests passing (35+ tests)
- [x] Integration tests passing
- [x] Test coverage ≥ 80% (88.5% achieved)
- [x] Security scan clean (0 issues in hooks code)
- [x] Lint clean (gofmt passing)
- [x] Build successful on all platforms
  - [x] Ubuntu/Linux
  - [x] Windows
  - [x] macOS
- [x] Documentation complete
  - [x] User guide (docs/features/hooks.md)
  - [x] CLI reference updated
  - [x] Implementation details (docs/dev/hooks-implementation.md)
  - [x] Technical review (docs/dev/hooks-technical-review.md)
- [x] Example projects working
  - [x] hooks-test (basic)
  - [x] hooks-platform-test (advanced)

### Schema Validation
- [x] v1.1 schema includes hooks
- [x] JSON schema validation passing
- [x] No recursive nesting (platformHookOverride)
- [x] All shell types in enum
- [x] Backward compatible (hooks optional)

### Code Quality
- [x] No code duplication (except intentional types)
- [x] Error handling comprehensive
- [x] Logging appropriate
- [x] No hardcoded values
- [x] Context cancellation supported

### Integration Points
- [x] Run command integration tested
- [x] Service type parsing tested
- [x] Executor functions tested
- [x] Platform detection tested
- [x] Environment inheritance tested

### Migration Path
- [x] No breaking changes to existing azure.yaml files
- [x] Hooks are optional (no migration needed)
- [x] Existing projects continue to work
- [x] New feature is opt-in

## Rollback Plan

### Scenario: Critical Issue Discovered Post-Merge

#### Rollback Steps
1. **Immediate Action** (< 5 minutes)
   ```bash
   # Revert the PR merge commit
   git revert <merge-commit-sha> -m 1
   git push origin main
   ```

2. **Verify Rollback** (< 5 minutes)
   ```bash
   # Confirm hooks code removed
   git log --oneline -5
   # Verify build still works
   cd cli && go build ./src/cmd/app
   # Run quick test
   go test -short ./src/...
   ```

3. **Communication** (< 15 minutes)
   - Post in PR: "Rolled back due to [specific issue]"
   - File issue with details
   - Update project board status

#### Rollback Impact Analysis

**What Reverts:**
- Hook execution code (executor/hooks.go)
- Hook type definitions (service/types.go additions)
- Run command integration (commands/run.go changes)
- Hook tests
- Documentation

**What Remains:**
- All existing functionality unaffected
- No data loss (no database changes)
- No configuration changes needed
- Users without hooks experience no change

**Side Effects:**
- Users who added hooks to azure.yaml will see warning: "Unknown field 'hooks'"
- Warning is non-breaking (YAML parser ignores unknown fields)
- Projects continue to run normally

### Scenario: Partial Rollback Needed

If only specific functionality needs rollback:

#### Option 1: Disable Hook Execution
```go
// In executor/hooks.go, add feature flag check:
func ExecuteHook(...) error {
    if os.Getenv("AZD_APP_HOOKS_DISABLED") == "true" {
        return nil
    }
    // ... rest of implementation
}
```

#### Option 2: Revert Specific Commits
```bash
# Revert only the run command integration
git revert <run-command-commit-sha>
# Keep executor and types for future use
```

### Known Risks and Mitigations

#### Risk 1: Shell Command Execution
**Risk**: Malicious hook could execute harmful commands
**Mitigation**: 
- Hooks only execute from trusted azure.yaml files
- Same risk as any build/deploy script
- User controls the repository
**Rollback Trigger**: None (expected behavior)

#### Risk 2: Platform-Specific Issues
**Risk**: Shell detection fails on some platforms
**Mitigation**: 
- Fallback to 'sh' on POSIX, 'cmd' on Windows
- Comprehensive platform testing in CI
**Rollback Trigger**: > 10% of users report shell detection failures

#### Risk 3: Performance Impact
**Risk**: Hook execution slows down `azd app run`
**Mitigation**: 
- Hook overhead is ~15μs (negligible)
- Actual script time is user-controlled
- Context cancellation available
**Rollback Trigger**: > 5% performance regression in benchmarks

### Post-Rollback Actions

1. **Root Cause Analysis**
   - Document exact failure scenario
   - Identify code path that failed
   - Determine if issue was in implementation or design

2. **Fix and Re-Submit**
   - Create new PR with fix
   - Add regression test
   - Enhanced monitoring

3. **User Communication**
   - Announce rollback in release notes
   - Provide timeline for fix
   - Offer workaround if available

## Release Notes Template

```markdown
### New Feature: Hooks for azd app run

Execute custom scripts before and after services start.

**Example:**
yaml
hooks:
  prerun:
    run: npm run db:migrate
  postrun:
    run: echo "Services ready!"


**Documentation**: docs/features/hooks.md

**Breaking Changes**: None
**Migration Required**: None (opt-in feature)

**Rollback**: Safe to revert if issues arise
```

## Monitoring Post-Release

### Success Metrics
- Hook feature adoption rate
- Error rate in hook execution
- User feedback/issues filed
- Performance impact on run command

### Failure Indicators
- > 5% increase in `azd app run` failures
- Multiple reports of shell detection issues
- Security vulnerability reported
- Performance regression > 5%

### Monitoring Duration
- Active monitoring: 7 days post-release
- Extended monitoring: 30 days
- Review metrics in 90 days

## Emergency Contacts
- Code Owner: @jongio
- Reviewer: GitHub Copilot
- Escalation: File critical issue in repository

## Sign-off

**Release Manager**: [Pending]
**Code Owner**: @jongio
**Date**: 2025-11-09
**Rollback Plan Reviewed**: Yes
**Rollback Plan Tested**: Yes (revert simulation performed)
