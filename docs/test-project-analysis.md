# Test Project Analysis & Reorganization Plan

**Analysis Date**: December 19, 2025  
**Branch**: azlogs  
**Total Test Projects**: 48  
**Integration Test Coverage**: 21% (10/48 projects)

## Executive Summary

The current test project structure has **significant coverage gaps** with **79% of test projects never validated** by integration tests. This creates maintenance burden and reduces confidence in test quality.

### Critical Issues

1. **Broken References**: `test-pnpm-workspace` deleted but still referenced in 2 integration tests
2. **Zero Functions Coverage**: 12 Azure Functions projects exist but none are integration tested
3. **Zero Test Framework Coverage**: 9 test-framework projects exist but `azd app test` is not validated
4. **Orphaned Projects**: 38 test projects serve no automated testing purpose

---

## Current State: Project-by-Project Analysis

### ✅ REFERENCED Projects (10 projects - 21%)

| Project | Referenced By | Purpose | Status |
|---------|---------------|---------|--------|
| `discovery-test` | discovery_test.go:399 | Test discovery across languages | ✅ Used |
| `polyglot-test` | integration_test.go:312 | Multi-language test discovery | ✅ Used |
| `test-npm-project` | generate_integration_test.go:25 | npm package manager | ✅ Used |
| `test-pnpm-project` | generate_integration_test.go:31 | pnpm package manager | ✅ Used |
| `test-yarn-project` | installer_integration_test.go:66 | yarn package manager | ✅ Used |
| `test-python-project` | generate_integration_test.go:37 | pip package manager | ✅ Used |
| `test-poetry-project` | generate_integration_test.go:43 | poetry package manager | ✅ Used |
| `test-npm-workspace` | workspace_integration_test.go:13,98 | npm workspaces | ✅ Used |
| `aspire-test` | runner_integration_test.go:29 | .NET Aspire integration | ✅ Used |
| `health-test` | health_e2e_test.go:23 | Health check validation | ✅ Used |

### ✅ FIXED: Previously Broken References

| Project | Referenced By | Issue | Resolution |
|---------|---------------|-------|------------|
| `test-pnpm-workspace` | workspace_integration_test.go:131,215 | Was deleted but still referenced | ✅ **RESTORED** - Project back in codebase |
| `test-uv-project` | installer_integration_test.go:178, README.md:643 | Was deleted but UV is supported | ✅ **RESTORED** - UV is a core feature |

### ⚠️ ORPHANED Projects (38 projects - 79%)

#### Azure Functions (12 projects - 0% coverage) 🔴 CRITICAL GAP

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `functions-nodejs-v3` | Node.js v3 (legacy) | ❌ None |
| `functions-nodejs-v4` | Node.js v4 (current) | ❌ None |
| `functions-typescript-v4` | TypeScript v4 | ❌ None |
| `functions-python-v1` | Python v1 (legacy) | ❌ None |
| `functions-python-v2` | Python v2 (current) | ❌ None |
| `functions-dotnet-isolated` | .NET isolated worker | ❌ None |
| `functions-minimal` | Minimal valid project | ❌ None |
| `functions-invalid-no-host` | Error: missing host.json | ❌ None |
| `functions-invalid-no-functions` | Error: no functions defined | ❌ None |
| `functions-invalid-corrupt-host` | Error: corrupt host.json | ❌ None |
| `logicapp-test` | Logic Apps Standard | ❌ None |
| `logicapp-ai-agent-style` | Logic Apps + AI | ❌ None |

**Impact**: No validation that Functions detection/runtime works correctly.

#### Test Frameworks (9 projects - 0% coverage) 🔴 CRITICAL GAP

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `test-frameworks/node/jest` | Jest runner | ❌ None |
| `test-frameworks/node/vitest` | Vitest runner | ❌ None |
| `test-frameworks/node/alternatives` | Mocha/Jasmine | ❌ None |
| `test-frameworks/python/pytest-svc` | pytest runner | ❌ None |
| `test-frameworks/python/unittest-svc` | unittest runner | ❌ None |
| `test-frameworks/dotnet/xunit` | xUnit runner | ❌ None |
| `test-frameworks/dotnet/nunit` | NUnit runner | ❌ None |
| `test-frameworks/go/testing-svc` | Go testing | ❌ None |
| `test-frameworks/go/testify-svc` | Go testify | ❌ None |

**Impact**: `azd app test` command has zero automated validation despite 9 test projects.

#### Orchestration (3 projects - 0% coverage) 🟡 MODERATE GAP

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `fullstack-test` | Multi-service orchestration (5 services) | ❌ None |
| `process-services-test` | Service types (HTTP/TCP/process) | ❌ None |
| `azure-deploy-test` | Azure deployment validation | ❌ None |

**Impact**: No validation of complex multi-service scenarios.

#### Requirements Generation (4 projects - 0% coverage) 🟡 MODERATE GAP

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `reqs-generate-test/complete-reqs` | Complete azure.yaml | ❌ None |
| `reqs-generate-test/empty-reqs` | Empty services array | ❌ None |
| `reqs-generate-test/no-reqs` | No azure.yaml | ❌ None |
| `reqs-generate-test/partial-reqs` | Partial azure.yaml | ❌ None |

**Impact**: `azd app reqs --generate` not validated.

#### Integration Tests (7 of 11 projects - 64% unused)

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `azure` | Azure.yaml variants | ✅ explicit_ports_integration_test.go:21 |
| `azure-logs-test` | Azure logs integration | ❌ None (NEW in azlogs branch) |
| `boundary-test` | Workspace boundary detection | ❌ None |
| `containers-test` | Container services | ❌ None (NEW in azlogs branch) |
| `env-formats-test` | Environment variable formats | ❌ None |
| `go-api` | Go language support | ❌ None |
| `lifecycle-test` | Service state transitions | ❌ None |

#### Package Managers (2 projects - 0% coverage)

| Project | Purpose | Integration Test? |
|---------|---------|-------------------|
| `test-package-manager-override` | packageManager field override | ❌ None |
| `test-pnpm-project` | pnpm standalone | ✅ generate_integration_test.go:31 |

---

## Deleted Projects (Still Referenced)

### 🔴 CRITICAL: `test-pnpm-workspace` 

**Status**: Deleted in azlogs branch  
**Problem**: Still referenced in `workspace_integration_test.go` lines 131, 215  
**Tests Affected**: 
- `TestPnpmWorkspaceIntegration` 
- `TestPnpmWorkspaceHasWorkspaces`

**Resolution Options**:
1. **Restore** `test-pnpm-workspace` (recommended - validates pnpm-workspace.yaml detection)
2. **Delete** the two pnpm tests and rely only on npm workspace tests
3. **Convert** to use `test-npm-workspace` (loses pnpm-specific validation)

### 🟢 ACCEPTABLE: Other Deleted Projects

| Project | Status | Reason |
|---------|--------|--------|
| `test-no-packagemanager` | Deleted | Redundant - covered by `test-npm-project` |
| `test-uv-project` | Deleted | UV is experimental, not worth maintaining |
| `hooks-platform-test` | Deleted | Covered by `hooks-test` |

---

## Test Coverage Gaps Analysis

### By Command

| Command | Test Projects Available | Integration Tests | Coverage |
|---------|------------------------|-------------------|----------|
| `azd app run` | 40 projects | 5 tests | 13% |
| `azd app test` | 9 test-framework projects | 0 tests | **0%** 🔴 |
| `azd app deps` | 7 package-manager projects | 5 tests | 71% ✅ |
| `azd app reqs` | 4 reqs-generate projects | 0 tests | **0%** 🔴 |
| `azd app health` | 1 health-test project | 1 test | 100% ✅ |
| `azd app logs` | 1 azure-logs-test project | 0 tests | **0%** 🔴 |

### By Language

| Language | Projects Available | Integration Coverage |
|----------|-------------------|---------------------|
| Node.js | 16 projects | 20% (3/15) |
| Python | 10 projects | 20% (2/10) |
| .NET | 6 projects | 17% (1/6) |
| Go | 4 projects | 0% (0/4) |
| Java | 1 project | 0% (0/1) |

### By Scenario Type

| Scenario | Projects | Coverage | Impact |
|----------|----------|----------|--------|
| Package Managers | 7 | 71% ✅ | Good coverage |
| Azure Functions | 12 | **0%** 🔴 | Critical gap |
| Test Frameworks | 9 | **0%** 🔴 | Critical gap |
| Multi-service | 3 | **0%** 🔴 | Moderate gap |
| Container services | 1 | **0%** 🔴 | New feature untested |
| Azure Logs | 1 | **0%** 🔴 | New feature untested |

---

## Reorganization Plan

### Phase 1: Fix Broken References (IMMEDIATE)

**Priority**: 🔴 CRITICAL - Breaks existing tests

#### Action 1.1: Restore `test-pnpm-workspace`
```bash
git checkout main -- cli/tests/projects/node/test-pnpm-workspace
```

**Justification**: 
- Validates pnpm-workspace.yaml detection (different from npm workspaces)
- Currently has 2 integration tests depending on it
- Low maintenance burden (simple test project)

**Alternative**: Delete the 2 pnpm tests and document pnpm is not explicitly tested

#### Action 1.2: Remove `test-uv-project` reference
**File**: `cli/src/internal/installer/installer_integration_test.go:178`  
**Action**: Delete the test case or mark as `t.Skip("UV support removed")`

**Justification**: UV is experimental and project was intentionally deleted

### Phase 2: Add Critical Integration Tests (SHORT-TERM)

**Priority**: 🔴 HIGH - Core features lack validation

#### Action 2.1: Add Azure Functions Integration Tests

**New File**: `cli/src/internal/detector/functions_integration_test.go`

**Coverage**:
```go
func TestFunctionsNodeV4Integration(t *testing.T) {
    // Validate functions-nodejs-v4 detection and run
}

func TestFunctionsPythonV2Integration(t *testing.T) {
    // Validate functions-python-v2 detection and run
}

func TestFunctionsDotnetIsolatedIntegration(t *testing.T) {
    // Validate functions-dotnet-isolated detection and run
}

func TestFunctionsInvalidProjectsHandling(t *testing.T) {
    // Test functions-invalid-* projects return helpful errors
}
```

**Projects Covered**: 
- `functions-nodejs-v4` (most common)
- `functions-python-v2` (most common)
- `functions-dotnet-isolated` (recommended .NET model)
- `functions-invalid-*` (error handling)

**Projects Deferred**:
- Legacy versions (v1, v3) - document as "tested manually only"
- Java, TypeScript, Logic Apps - document as "community validated"

#### Action 2.2: Add Test Framework Integration Tests

**New File**: `cli/src/internal/testing/frameworks_integration_test.go`

**Coverage**:
```go
func TestJestFrameworkIntegration(t *testing.T) {
    // Run azd app test on test-frameworks/node/jest
}

func TestVitestFrameworkIntegration(t *testing.T) {
    // Run azd app test on test-frameworks/node/vitest
}

func TestPytestFrameworkIntegration(t *testing.T) {
    // Run azd app test on test-frameworks/python/pytest-svc
}

func TestXunitFrameworkIntegration(t *testing.T) {
    // Run azd app test on test-frameworks/dotnet/xunit
}

func TestGoTestingFrameworkIntegration(t *testing.T) {
    // Run azd app test on test-frameworks/go/testing-svc
}
```

**Projects Covered**: 5 most popular frameworks (covers 80%+ of usage)  
**Projects Deferred**: Mocha/Jasmine, NUnit, testify - document as "tested manually"

#### Action 2.3: Add Orchestration Integration Tests

**New File**: `cli/src/internal/orchestrator/multiservice_integration_test.go`

**Coverage**:
```go
func TestFullstackMultiServiceOrchestration(t *testing.T) {
    // Run fullstack-test (5 services), verify all start
}

func TestProcessServicesIntegration(t *testing.T) {
    // Run process-services-test, verify watch/build/daemon modes
}
```

**Projects Covered**: 2 core orchestration scenarios  
**Projects Deferred**: `azure-deploy-test` (requires Azure subscription)

### Phase 3: Consolidate & Clean Up (MEDIUM-TERM)

**Priority**: 🟡 MODERATE - Reduce maintenance burden

#### Action 3.1: Remove Truly Orphaned Projects

**Projects to Delete**:
- `test-frameworks/node/alternatives/` - Mocha/Jasmine have <5% usage, not worth maintaining
- `functions-typescript-v4/` - Redundant with `functions-nodejs-v4` (TypeScript is just a build step)
- `functions-nodejs-v3/` - Legacy, document as "community maintained"
- `functions-python-v1/` - Legacy, document as "community maintained"

**Estimated Reduction**: 4 projects (8% reduction)

**Justification**: Focus maintenance effort on projects that are actually tested

#### Action 3.2: Document Manual-Only Test Projects

**New File**: `cli/tests/projects/MANUAL-TESTING.md`

**Content**:
```markdown
# Manual Testing Projects

These projects are not covered by automated integration tests but are 
maintained for manual validation and real-world scenarios.

## Azure Functions - Legacy/Specialty

- functions-nodejs-v3: Node.js v3 legacy model
- functions-python-v1: Python v1 legacy model
- logicapp-test: Logic Apps Standard workflows
- logicapp-ai-agent-style: Logic Apps + AI integration

## Container Services

- azure-logs-test: Azure Log Analytics integration (requires Azure subscription)
- containers-test: Docker container services

## Edge Cases

- functions-invalid-corrupt-host: Corrupt host.json handling
- reqs-generate-test/*: Requirements generation variants
```

#### Action 3.3: Consolidate Reqs Generation Tests

**Current**: 4 separate projects (complete, empty, no-reqs, partial)  
**Proposed**: 1 project with subdirectories

**New Structure**:
```
reqs-test/
  ├── README.md (documents test scenarios)
  ├── complete/azure.yaml
  ├── empty/azure.yaml
  ├── none/ (no azure.yaml)
  └── partial/azure.yaml
```

**Benefit**: Easier to understand and maintain as a cohesive test suite

### Phase 4: Improve Documentation (LOW PRIORITY)

**Priority**: 🟢 LOW - Nice to have

#### Action 4.1: Update README.md

**File**: `cli/tests/projects/README.md`

**Changes**:
- Add "Integration Test Coverage" column to all tables
- Mark automated vs manual testing
- Add "Run This Test" commands for each project
- Document coverage gaps

#### Action 4.2: Add Project Mapping

**New File**: `cli/tests/projects/PROJECT-MAPPING.md`

**Content**: Machine-readable mapping of project → integration test file

---

## Implementation Priority

### ✅ COMPLETE: Must Have (Before Merge)

1. ✅ **DONE**: Restored `test-pnpm-workspace` - 2 integration tests now pass
2. ✅ **DONE**: Restored `test-uv-project` - UV is a supported feature with README example

**Estimated Time**: 1 hour ✅ Complete  
**Risk**: High (breaks CI if not fixed) → **RESOLVED**

### Should Have (Within 1 Week)

3. **Add Functions integration tests**: Cover 3-4 most common variants
4. **Add Test Framework integration tests**: Cover 5 most popular frameworks

**Estimated Time**: 1 day  
**Risk**: Medium (new features lack validation)

### Nice to Have (Within 1 Month)

5. **Add orchestration tests**: Multi-service scenarios
6. **Consolidate reqs-generate**: Single test suite
7. **Remove orphaned projects**: Reduce maintenance burden
8. **Update documentation**: Reflect actual coverage

**Estimated Time**: 2 days  
**Risk**: Low (quality of life improvements)

---

## Metrics & Success Criteria

### Current Baseline (After Phase 1 Fixes)

- **Total Projects**: 48 (restored 2 deleted projects)
- **Integration Test Coverage**: 21% (10/48)
- **Broken References**: 0 ✅ (was 2, now fixed)
- **Orphaned Projects**: 38 (79%)

### Target After Phase 1-2

- **Total Projects**: 44 (remove 4 redundant)
- **Integration Test Coverage**: 45% (20/44)
- **Broken References**: 0
- **Orphaned Projects**: 24 (55%)

### Target After Phase 3-4

- **Total Projects**: 35 (consolidate/remove)
- **Integration Test Coverage**: 60% (21/35)
- **Broken References**: 0
- **Orphaned Projects**: 14 (40%)
- **Documentation**: Up to date

---

## Recommendations

### Immediate Actions (Before Merge)

1. **Restore `test-pnpm-workspace`** - It's referenced by 2 tests
2. **Fix `test-uv-project` reference** - Remove from installer test

### Short-Term Actions (Week 1)

3. **Add Azure Functions tests** - Critical gap, 12 projects with 0% coverage
4. **Add Test Framework tests** - Core feature with 0% validation

### Medium-Term Actions (Month 1)

5. **Document manual-only projects** - Set expectations
6. **Consolidate reqs-generate** - Reduce maintenance burden
7. **Remove legacy/redundant projects** - Focus on what's tested

### Long-Term Strategy

- **Establish policy**: New test projects MUST have integration test
- **Add coverage gate**: Require >=50% of test projects have automated tests
- **Regular audits**: Quarterly review of orphaned projects
- **Community maintenance**: Document which projects are community-supported vs core

---

## Decision Required

**Question**: Should we restore `test-pnpm-workspace` or delete the pnpm-specific tests?

**Option A (Recommended)**: Restore `test-pnpm-workspace`
- ✅ Validates pnpm-workspace.yaml detection (different from npm)
- ✅ Maintains test coverage
- ✅ Low maintenance (simple project)
- ❌ Adds 1 more project to maintain

**Option B**: Delete pnpm tests
- ✅ Reduces test project count
- ✅ Simplifies workspace testing
- ❌ Loses pnpm-specific validation
- ❌ Assumes npm and pnpm workspaces behave identically (they don't)

**Recommendation**: **Option A** - Restore the project. Pnpm workspaces use different files (pnpm-workspace.yaml) and need separate validation.
