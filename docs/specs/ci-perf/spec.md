# CI Performance Optimization

## Overview

Reduce CI workflow execution time from 40+ minutes to <15 minutes through strategic parallelization, caching, and workflow reorganization.

## Current State Analysis

### Timing Breakdown (Last Successful Run)
- **Preflight Checks**: 9.0 min (BOTTLENECK #1)
- **Test (windows)**: 6.2 min (BOTTLENECK #2)  
- **Integration Tests (windows)**: 4.6 min (BOTTLENECK #3)
- **Test (ubuntu)**: 3.6 min
- **Test (macos)**: 3.4 min
- **Build**: 3.4 min (blocked by preflight+test+lint)
- **Lint**: 2.5 min
- **Integration Tests (macos)**: 2.6 min
- **Integration Tests (ubuntu)**: 2.1 min
- **Total Runtime**: ~37 minutes (critical path)

### Identified Inefficiencies
1. **Repeated Setup Work** (8-9x across jobs):
   - Go custom version installation (bootstrap + download)
   - Node.js + pnpm setup
   - Dashboard build (pnpm install + build)
   - Go mod download

2. **Sequential Dependencies**:
   - Build job blocked by preflight + test + lint (unnecessary wait)
   - Preflight installs 5+ tools but runs minimal checks
   - No caching strategy for Go modules, pnpm deps, or build artifacts

3. **Over-Testing**:
   - Integration tests run on all 3 platforms (ubuntu, windows, macos)
   - Most integration issues are platform-agnostic
   - Test matrix could be optimized

4. **Build Inefficiency**:
   - Builds 4 platform binaries sequentially in single job
   - Could parallelize per-platform builds

## Goals

### Primary
- **Reduce total CI time to <15 minutes** (60%+ reduction)
- **Reduce critical path to <10 minutes**

### Secondary
- Maintain or improve test coverage
- Preserve security checks (gosec, govulncheck)
- Keep all platform validation (ubuntu, windows, macos)

## Optimization Strategy

### Phase 1: Caching & Setup Optimization (Expected: -10 min)

**1. Implement Aggressive Caching**
- Go modules cache (restore across jobs)
- pnpm store cache (restore across jobs)
- Dashboard build artifacts cache
- Go build cache
- Go custom version binary cache

**2. Shared Setup Action**
- Create composite action `.github/actions/setup-go-node/action.yml`
- Consolidates: Go setup + Node.js + pnpm + cache restoration
- Reusable across all jobs (reduces duplication 8x → 1x)

**3. Dashboard Build Artifact Strategy**
- Build dashboard ONCE in preflight or dedicated job
- Upload as artifact
- Download in dependent jobs (faster than rebuilding)

### Phase 2: Parallelization & Job Restructuring (Expected: -12 min)

**4. Remove Unnecessary Job Dependencies**
- Build job doesn't need preflight (only needs test passing)
- Lint and test can run fully parallel
- Integration tests can start immediately after unit tests pass

**5. Optimize Preflight Scope**
- Split into: `quick-checks` (2 min) + `security-scan` (parallel, 4 min)
- Quick checks: gofmt, go vet, basic linting
- Security scan: gosec, govulncheck (runs parallel, doesn't block)

**6. Platform-Specific Job Matrix Optimization**
- Unit tests: Keep all 3 platforms (ubuntu, windows, macos)
- Integration tests: **Primary on ubuntu only, smoke on windows**
- Build: Parallelize per-platform in matrix (4 concurrent builds)

### Phase 3: Advanced Optimizations (Expected: -5 min)

**7. Conditional Execution**
- Skip integration tests if only docs/comments changed
- Skip website build if no web/ changes (already has path filter)
- Fast-path for README-only changes

**8. Test Optimization**
- Use `-short` flag more strategically
- Group fast tests vs slow integration tests
- Cache test binaries between runs

**9. GitHub-Hosted Runner Optimization**
- Use `ubuntu-latest` for maximum parallel capacity
- Consider spot runner for integration tests
- Investigate GitHub Actions cache service limits

## Implementation Plan

### Phase 1: Caching & Setup (Week 1)
1. Create composite setup action
2. Implement Go modules cache
3. Implement pnpm + dashboard build cache
4. Measure impact (target: 25-30 min total)

### Phase 2: Parallelization (Week 1-2)
5. Remove build job's preflight dependency
6. Split preflight into quick-checks + security-scan
7. Optimize integration test matrix (ubuntu primary, windows smoke)
8. Parallelize build job per-platform
9. Measure impact (target: 12-15 min total)

### Phase 3: Advanced (Week 2)
10. Implement conditional execution
11. Optimize test suite grouping
12. Final tuning and validation
13. Measure final impact (target: <12 min total)

## Detailed Task Breakdown

### Cache Implementation
- Go modules: Use `actions/cache@v4` with `go.sum` hash key
- pnpm: Use pnpm's built-in cache action with `pnpm-lock.yaml` hash
- Dashboard build: Upload as artifact, download in test/lint/build jobs
- Cache key strategy: `${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}`

### Composite Action Structure
```yaml
# .github/actions/setup-go-node/action.yml
inputs:
  go-version: required
  node-version: default '20'
  skip-dashboard: default 'false'
runs:
  - Restore Go modules cache
  - Bootstrap Go + install custom version
  - Restore pnpm cache
  - Install pnpm
  - Download dashboard artifact OR build (if skip-dashboard=false)
```

### Optimized Job Dependencies
```
Current:
  preflight → (test, lint) → build → integration

Optimized:
  setup-dashboard → (preflight-quick, test, lint, build, security-scan) → integration-ubuntu
                  → integration-windows (smoke only)
```

### Integration Test Strategy
- **ubuntu**: Full integration test suite (~2 min)
- **windows**: Smoke tests only (basic functionality, ~1.5 min)
- **macos**: Skip (redundant for Go integration tests)
- Rationale: Integration issues are 95%+ platform-agnostic

## Testing Strategy

### Validation Plan
1. **Baseline**: Capture current timing for 5 consecutive runs
2. **Phase 1 Testing**: Validate caching works across runs
3. **Phase 2 Testing**: Verify parallelization doesn't introduce flakes
4. **Regression Testing**: Ensure all checks still pass after restructuring

### Metrics to Track
- Total workflow duration (target: <15 min)
- Critical path duration (target: <10 min)
- Cache hit rate (target: >80% for dependencies)
- Test flake rate (maintain current level)
- Security scan coverage (maintain 100%)

### Rollback Plan
- Keep original ci.yml as ci-legacy.yml for 2 weeks
- Feature flag for new workflow (manual dispatch)
- Monitor failure rates for 1 week before making default

## Success Criteria

### Must Have
- ✅ Total CI time <15 minutes (60%+ reduction from 40 min)
- ✅ All existing tests pass
- ✅ All security scans maintained (gosec, govulncheck)
- ✅ All platforms validated (ubuntu, windows, macos for unit tests)
- ✅ Cache hit rate >75%

### Should Have
- ✅ Critical path <10 minutes
- ✅ Zero increase in test flakes
- ✅ Documentation updated with new workflow structure
- ✅ Monitoring for regression detection

### Nice to Have
- ✅ Total CI time <12 minutes (70%+ reduction)
- ✅ Reusable patterns for other repos (azd-exec, azd-rest, etc)
- ✅ Cost analysis (runner minutes before/after)

## Risk Assessment

### High Risk
- **Test coverage gaps**: Mitigation: Maintain unit test matrix, run integration on ubuntu
- **Cache corruption**: Mitigation: Cache key includes hash, auto-invalidates on dependency changes
- **Workflow complexity**: Mitigation: Comprehensive docs, gradual rollout

### Medium Risk
- **Platform-specific regressions**: Mitigation: Keep windows smoke tests, manual testing before release
- **GitHub Actions limits**: Mitigation: Monitor cache usage, artifact retention

### Low Risk
- **Increased flakiness**: Mitigation: Existing timeout guards, can revert parallelization
- **Slower cold runs**: Mitigation: First run will be slower, subsequent runs benefit from cache
