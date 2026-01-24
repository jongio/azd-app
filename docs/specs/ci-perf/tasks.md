# CI Performance Optimization - Tasks

<!-- NEXT: 1 -->

## TODO

### 1. Baseline Performance Measurement
**Priority**: P0  
**Scope**: Investigation  
**Description**: Establish baseline metrics for current CI performance. Capture timing data from 5 consecutive successful runs to understand variance and identify most consistent bottlenecks.
**Acceptance Criteria**:
- Documented average runtime and variance for each job
- Identified top 5 bottlenecks with timing data
- Baseline metrics saved for comparison

### 2. Create Composite Setup Action
**Priority**: P0  
**Scope**: Infrastructure  
**Description**: Create reusable composite action `.github/actions/setup-go-node/action.yml` that consolidates Go setup, Node.js, pnpm, and cache restoration. Reduces code duplication from 8x to 1x.
**Acceptance Criteria**:
- Composite action created with inputs for go-version, node-version, skip-dashboard
- Includes Go modules cache restoration
- Includes pnpm store cache restoration
- Includes Go custom version installation
- Documentation in action.yml with usage examples

### 3. Implement Go Modules Cache
**Priority**: P0  
**Scope**: Optimization  
**Description**: Add `actions/cache@v4` for Go modules across all jobs using `go.sum` hash as cache key. Should restore dependencies instead of downloading on every run.
**Acceptance Criteria**:
- Cache key: `${{ runner.os }}-go-mod-${{ hashFiles('**/go.sum') }}`
- Restore cache before `go mod download`
- Cache hit reduces go mod download time from ~30s to <5s
- Works across all CI jobs (test, lint, build, integration)

### 4. Implement pnpm + Dashboard Build Cache
**Priority**: P0  
**Scope**: Optimization  
**Description**: Cache pnpm store and dashboard build output. Dashboard is built 8+ times currently, should be built once and reused.
**Acceptance Criteria**:
- pnpm store cache with key: `${{ runner.os }}-pnpm-${{ hashFiles('**/pnpm-lock.yaml') }}`
- Dashboard build artifact uploaded after first build
- All jobs download dashboard artifact instead of rebuilding
- Dashboard build time reduced from 8x×60s = 8min total to 1×60s = 1min

### 5. Update All CI Jobs to Use Composite Action
**Priority**: P0  
**Scope**: Refactoring  
**Description**: Replace duplicated setup code in all CI jobs (preflight, test, lint, build, integration) with the new composite action.
**Acceptance Criteria**:
- All jobs use composite action for setup
- No Go/Node/pnpm setup code duplicated
- All jobs restore caches correctly
- CI still passes after changes

### 6. Remove Build Job's Preflight Dependency
**Priority**: P1  
**Scope**: Architecture  
**Description**: Build job currently waits for preflight (9 min) + test + lint. It only needs test to pass. Remove preflight from `needs:` array.
**Acceptance Criteria**:
- Build job needs: [test, lint] (remove preflight)
- Build starts immediately after tests pass
- Saves 9 minutes from critical path

### 7. Split Preflight into Quick Checks + Security Scan
**Priority**: P1  
**Scope**: Architecture  
**Description**: Current preflight (9 min) blocks everything. Split into fast `quick-checks` (2 min) for gofmt/vet and parallel `security-scan` (4 min) for gosec/govulncheck that doesn't block.
**Acceptance Criteria**:
- New job: `quick-checks` (gofmt, go vet) ~2 min
- New job: `security-scan` (gosec, govulncheck, staticcheck) ~4 min  
- security-scan runs parallel, doesn't block build
- quick-checks can block test/lint if desired for fast feedback
- No security tools removed, just reorganized

### 8. Optimize Integration Test Matrix
**Priority**: P1  
**Scope**: Architecture  
**Description**: Integration tests run on all 3 platforms (ubuntu, windows, macos). Most integration issues are platform-agnostic. Run full suite on ubuntu, smoke tests only on windows, skip macos.
**Acceptance Criteria**:
- ubuntu: Full integration test suite (~2 min)
- windows: Smoke tests only (basic functionality, ~1.5 min)
- macos: Removed from integration test matrix
- Document smoke test selection criteria
- Total integration test time reduced from 9.3 min to ~3.5 min

### 9. Parallelize Build Job by Platform
**Priority**: P1  
**Scope**: Optimization  
**Description**: Build currently compiles 4 platform binaries sequentially. Use matrix strategy to build each platform in parallel.
**Acceptance Criteria**:
- Matrix: [windows-amd64, linux-amd64, darwin-amd64, darwin-arm64]
- Each platform builds in parallel (~1 min each)
- All binaries uploaded as single artifact
- Build time reduced from ~3.4 min to ~1.5 min

### 10. Measure Phase 1+2 Performance
**Priority**: P1  
**Scope**: Validation  
**Description**: After implementing caching, composite action, and parallelization changes, measure new CI performance over 5 runs. Compare to baseline.
**Acceptance Criteria**:
- 5 successful consecutive runs captured
- Average runtime documented
- Cache hit rates measured (target: >75%)
- Comparison with baseline showing improvement
- Target: Total time <18 minutes (50%+ reduction)

### 11. Implement Conditional Execution
**Priority**: P2  
**Scope**: Optimization  
**Description**: Skip expensive jobs when only docs/comments change. Use path filters and conditional logic.
**Acceptance Criteria**:
- Skip integration tests if only `.md`, `.txt`, comments changed
- Skip build if only docs changed
- Fast-path for README-only PRs (<2 min)
- Document conditional logic in workflow comments

### 12. Optimize Test Suite Grouping
**Priority**: P2  
**Scope**: Optimization  
**Description**: Use `-short` flag strategically. Group fast tests separately from slow integration tests. Cache test binaries.
**Acceptance Criteria**:
- Fast tests (<5s each) separated from slow tests
- `-short` flag skips slow integration tests in unit test job
- Test binary caching reduces compilation overhead
- Test time reduced by 10-20%

### 13. Final Performance Validation
**Priority**: P2  
**Scope**: Validation  
**Description**: After all optimizations, run final performance measurement. Validate success criteria met.
**Acceptance Criteria**:
- Total CI time <15 minutes (target met)
- Critical path <10 minutes (target met)
- Cache hit rate >75%
- All tests passing
- Security scans maintained
- Zero increase in flake rate

### 14. Documentation and Rollout
**Priority**: P2  
**Scope**: Documentation  
**Description**: Document new workflow structure, update CONTRIBUTING.md, create rollback plan, monitor for 1 week.
**Acceptance Criteria**:
- CONTRIBUTING.md updated with CI optimization details
- Original ci.yml backed up as ci-legacy.yml
- Monitoring plan for failure rates over 1 week
- Cost analysis (runner minutes before/after)
- Success announcement with metrics

## IN PROGRESS

## DONE
