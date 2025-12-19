# Test Project Mapping Analysis

**Date**: December 19, 2025  
**Purpose**: Map every test project to its actual usage in integration tests

---

## REFERENCED TEST PROJECTS

### ✅ Actively Used in Integration Tests

#### Package Managers (5 of 7 projects used)

| Project | Test File | Line | Test Name | Purpose |
|---------|-----------|------|-----------|---------|
| `test-npm-project` | installer_integration_test.go | 30 | TestInstallNodeDependenciesIntegration | Tests npm dependency installation in temp directory (inline) |
| `test-npm-project` | generate_integration_test.go | 25 | TestGenerateIntegration | Tests azure.yaml generation for npm projects |
| `test-pnpm-project` | installer_integration_test.go | 46 | TestInstallNodeDependenciesIntegration | Tests pnpm dependency installation in temp directory (inline) |
| `test-pnpm-project` | generate_integration_test.go | 31 | TestGenerateIntegration | Tests azure.yaml generation for pnpm projects |
| `test-yarn-project` | installer_integration_test.go | 66 | TestInstallNodeDependenciesIntegration | Tests yarn dependency installation in temp directory (inline) |
| `test-python-project` | generate_integration_test.go | 37 | TestGenerateIntegration | Tests azure.yaml generation for Python/pip projects |
| `test-poetry-project` | installer_integration_test.go | 198 | TestSetupPythonVirtualEnvIntegration | Tests Poetry virtual env setup in temp directory (inline) |
| `test-poetry-project` | generate_integration_test.go | 43 | TestGenerateIntegration | Tests azure.yaml generation for Poetry projects |
| `test-npm-workspace` | workspace_integration_test.go | 13 | TestNpmWorkspaceIntegration | Tests npm workspace detection and handling |
| `test-npm-workspace` | workspace_integration_test.go | 98 | TestNpmWorkspaceHasWorkspaces | Tests HasNpmWorkspaces() function |

**Note**: Most installer tests create temporary directories and inline package.json content rather than using actual test projects. This is intentional for isolation.

#### Integration Projects (4 of 11 projects used)

| Project | Test File | Line | Test Name | Purpose |
|---------|-----------|------|-----------|---------|
| `aspire-test` | runner_integration_test.go | 29 | TestAspireIntegration | Tests Aspire runner integration |
| `aspire-test` | aspire_test.go | 33 | TestAspireManifest | Tests Aspire manifest parsing |
| `aspire-test` | generate_integration_test.go | 49 | TestGenerateIntegration | Tests azure.yaml generation for Aspire |
| `health-test` | health_e2e_test.go | 23 | TestHealthE2E | E2E test for health monitoring |
| `polyglot-test` | integration_test.go | 312-314 | TestIntegration | Tests mixed-language project support |
| `discovery-test` | discovery_test.go | 251, 399-401 | TestDiscovery | Tests service discovery |

#### Top-Level Projects (1 of 1 used)

| Project | Test File | Line | Test Name | Purpose |
|---------|-----------|------|-----------|---------|
| `azure-deploy-test` | N/A | - | Manual testing only | Azure deployment validation (not automated) |

---

## ❌ ORPHANED TEST PROJECTS (No References in Test Code)

### Functions Category (12 projects - 0% coverage)

**All Azure Functions test projects are orphaned!**

| Project | Purpose (from README) | Status |
|---------|----------------------|--------|
| `functions-dotnet-isolated` | Validate isolated worker model | ❌ No test references |
| `functions-invalid-corrupt-host` | Error handling for corrupt host.json | ❌ No test references |
| `functions-invalid-no-functions` | Error handling for missing functions | ❌ No test references |
| `functions-invalid-no-host` | Error handling for missing host.json | ❌ No test references |
| `functions-minimal` | Minimal valid Functions project | ❌ No test references |
| `functions-nodejs-v3` | Legacy Node.js v3 support | ❌ No test references |
| `functions-nodejs-v4` | Modern Node.js v4 with TypeScript | ❌ No test references |
| `functions-python-v1` | Legacy Python v1 (function.json) | ❌ No test references |
| `functions-python-v2` | Modern Python v2 (decorator model) | ❌ No test references |
| `functions-typescript-v4` | TypeScript v4 support | ❌ No test references |
| `logicapp-ai-agent-style` | Complex Logic Apps with AI | ❌ No test references |
| `logicapp-test` | Basic Logic Apps workflow | ❌ No test references |

**Impact**: Zero automated validation of Azure Functions support despite 12 test projects!

### Integration Category (7 of 11 projects orphaned)

| Project | Purpose (from README) | Status |
|---------|----------------------|--------|
| `azure` | Configuration file variants | ❌ No test references |
| `azure-logs-test` | Azure logs API testing | ⚠️ Referenced in comment only (serviceinfo_test.go:45) |
| `boundary-test` | Workspace boundary checking | ❌ No test references |
| `containers-test` | Container service testing | ⚠️ Referenced in comment only (detector_test.go:1316) |
| `env-formats-test` | Environment variable handling | ❌ No test references |
| `go-api` | Go language support | ❌ No test references |
| `hooks-test` | Hook execution | ⚠️ Inline test only (hooks_integration_test.go:20) |
| `lifecycle-test` | Service state transitions | ❌ No test references |

**Note**: `hooks-test` project exists but tests create inline azure.yaml instead of using the actual project.

### Orchestration Category (3 of 3 projects orphaned)

| Project | Purpose (from README) | Status |
|---------|----------------------|--------|
| `azure-deploy-test` | Azure deployment with Container Apps | ⚠️ Manual testing only |
| `fullstack-test` | Multi-service orchestration | ❌ No test references |
| `process-services-test` | Service types and modes | ❌ No test references |

### Package Managers Category (2 of 7 projects orphaned)

| Project | Purpose (from README) | Status |
|---------|----------------------|--------|
| `test-package-manager-override` | packageManager field overrides lock files | ❌ No test references |
| `test-pnpm-workspace` | pnpm workspaces with monorepo | ⚠️ DELETED but still referenced! |

### Reqs-Generate Category (4 of 4 projects orphaned)

| Project | Purpose | Status |
|---------|---------|--------|
| `complete-reqs` | Complete requirements validation | ❌ No test references |
| `empty-reqs` | Empty requirements handling | ❌ No test references |
| `no-reqs` | No requirements file | ❌ No test references |
| `partial-reqs` | Partial requirements validation | ❌ No test references |

### Test Frameworks Category (9 projects - 0% coverage)

**All test framework projects are orphaned!**

| Project | Purpose | Status |
|---------|---------|--------|
| `test-frameworks/dotnet/*` | xUnit and NUnit testing | ❌ No test references |
| `test-frameworks/go/*` | Go testing and testify | ❌ No test references |
| `test-frameworks/node/*` | Jest, Vitest, alternatives | ❌ No test references |
| `test-frameworks/python/*` | pytest and unittest | ❌ No test references |
| `test-frameworks/failing/*` | Test failure handling | ❌ No test references |

**Impact**: Zero automated validation of `azd app test` command despite 9 test projects!

---

## 🔴 BROKEN REFERENCES (Deleted Projects Still Referenced)

### Critical Issues

| Deleted Project | Referenced In | Line | Test Name | Impact |
|----------------|---------------|------|-----------|--------|
| `test-pnpm-workspace` | workspace_integration_test.go | 131 | TestPnpmWorkspaceIntegration | **Test will fail** - Project deleted but 4 tests still reference it |
| `test-pnpm-workspace` | workspace_integration_test.go | 215 | TestPnpmWorkspaceHasWorkspaces | **Test will fail** - Missing project directory |
| `test-uv-project` | installer_integration_test.go | 178 | TestSetupPythonVirtualEnvIntegration | **Test creates inline** - Creates temp directory with inline pyproject.toml (not a broken reference) |

### Analysis

**`test-pnpm-workspace`**: 
- **Status**: DELETED
- **References**: 2 tests in workspace_integration_test.go
- **Impact**: Tests will skip if project doesn't exist (using os.Stat check)
- **Action**: Either restore project or remove tests

**`test-uv-project`**:
- **Status**: Never existed as directory (inline test only)
- **References**: Creates temporary directory inline
- **Impact**: No broken reference - working as designed

---

## 📊 COVERAGE SUMMARY

### By Category

| Category | Total Projects | Referenced | Orphaned | Coverage |
|----------|----------------|------------|----------|----------|
| **Functions** | 12 | 0 | 12 | **0%** ❌ |
| **Test Frameworks** | 9 | 0 | 9 | **0%** ❌ |
| **Orchestration** | 3 | 0 | 3 | **0%** ❌ |
| **Reqs-Generate** | 4 | 0 | 4 | **0%** ❌ |
| **Integration** | 11 | 4 | 7 | **36%** ⚠️ |
| **Package Managers** | 7 | 5 | 2 | **71%** ✅ |
| **Discovery** | 1 | 1 | 0 | **100%** ✅ |
| **Top-Level** | 1 | 0 | 1 | **0%** ⚠️ |
| **TOTAL** | **48** | **10** | **38** | **21%** ❌ |

### Test File Coverage

| Test File | Projects Referenced | Purpose |
|-----------|-------------------|---------|
| workspace_integration_test.go | 2 (1 deleted) | npm/pnpm workspace detection |
| installer_integration_test.go | 3 (inline only) | Dependency installation (creates temp dirs) |
| generate_integration_test.go | 5 | azure.yaml generation validation |
| health_e2e_test.go | 1 | Health monitoring E2E |
| discovery_test.go | 1 | Service discovery |
| integration_test.go | 1 | Mixed-language projects |
| runner_integration_test.go | 1 | Aspire integration |
| aspire_test.go | 1 | Aspire manifest parsing |

---

## 🎯 COVERAGE GAPS

### Critical Gaps (0% Coverage)

1. **Azure Functions** (12 projects, 0% coverage)
   - No automated tests for any Functions variant
   - Missing: Node.js v3/v4, Python v1/v2, .NET isolated, TypeScript
   - Missing: Logic Apps (standard and AI-integrated)
   - Missing: Invalid/error scenarios
   - **Impact**: No validation that Azure Functions detection and execution works

2. **Test Frameworks** (9 projects, 0% coverage)
   - No automated tests for `azd app test` command
   - Missing: Jest, Vitest, pytest, unittest, xUnit, NUnit, Go testing, testify
   - Missing: Test discovery, execution, and output parsing
   - **Impact**: No validation of test runner integration

3. **Orchestration** (3 projects, 0% coverage)
   - No automated tests for multi-service scenarios
   - Missing: Port management, cross-service communication
   - Missing: Service types (HTTP, TCP, process)
   - Missing: Watch mode, build mode, daemon mode
   - **Impact**: No validation of complex real-world scenarios

4. **Requirements Generation** (4 projects, 0% coverage)
   - No automated tests for `azd app reqs` command
   - Missing: Complete, empty, no-reqs, partial scenarios
   - **Impact**: No validation of requirements detection

### Medium Gaps (36% Coverage)

5. **Integration Projects** (4 of 11 used)
   - Covered: aspire-test, health-test, polyglot-test, discovery-test
   - Missing: azure, boundary-test, containers-test, env-formats-test, go-api, lifecycle-test
   - Referenced in comments only: azure-logs-test, hooks-test
   - **Impact**: Limited validation of advanced features

### Minor Gaps (71% Coverage)

6. **Package Managers** (5 of 7 used)
   - Covered: npm, pnpm, yarn (partially), python, poetry
   - Missing: test-package-manager-override
   - Broken: test-pnpm-workspace (deleted but referenced)
   - **Impact**: Most common scenarios covered

---

## 🔧 RECOMMENDED ACTIONS

### Immediate Actions (High Priority)

1. **Fix Broken Reference**
   - [ ] Remove or restore `test-pnpm-workspace` references in workspace_integration_test.go
   - Lines: 131, 215
   - Options: (a) Restore deleted project, or (b) Remove tests

2. **Add Functions Test Coverage**
   - [ ] Create `functions_integration_test.go`
   - [ ] Test detection for all 12 Functions variants
   - [ ] Test execution with `azd app run`
   - [ ] Validate error handling (invalid projects)

3. **Add Test Framework Coverage**
   - [ ] Create `test_frameworks_integration_test.go`
   - [ ] Test `azd app test` for all 9 framework variants
   - [ ] Validate test discovery and execution
   - [ ] Test output parsing and reporting

4. **Add Orchestration Coverage**
   - [ ] Create `orchestration_integration_test.go`
   - [ ] Test fullstack-test multi-service orchestration
   - [ ] Test process-services-test service types
   - [ ] Validate port management and cross-service communication

### Medium Priority

5. **Add Requirements Coverage**
   - [ ] Create `reqs_integration_test.go`
   - [ ] Test all 4 reqs-generate-test scenarios
   - [ ] Validate requirement detection and generation

6. **Complete Integration Coverage**
   - [ ] Add tests for: boundary-test, env-formats-test, go-api, lifecycle-test
   - [ ] Convert comment references to actual tests (azure-logs-test, containers-test)
   - [ ] Use actual hooks-test project instead of inline

### Low Priority

7. **Add Missing Package Manager Coverage**
   - [ ] Test test-package-manager-override
   - [ ] Validate packageManager field override behavior

---

## 📝 NOTES

### Test Strategy Patterns Observed

1. **Inline vs. Project-Based Testing**
   - Installer tests prefer creating temp directories with inline content
   - This is good for isolation but means test projects aren't validated
   - Trade-off: Project consistency vs. test isolation

2. **Manual vs. Automated Testing**
   - azure-deploy-test appears to be for manual testing only
   - No automated deployment validation

3. **Comment-Only References**
   - Some projects referenced in comments but not actual tests
   - Examples: azure-logs-test, containers-test, hooks-test

### Duplicate/Redundant Projects

No obvious duplicates found. Each project serves a distinct purpose even if not currently tested.

### Test Projects That Should Exist But Don't

Based on README claims vs actual projects:
- ✅ All documented projects exist
- ❌ But most lack automated test coverage

---

## 🎓 LESSONS LEARNED

1. **Documentation vs. Reality Gap**: README describes 40+ test projects, but only 10 (21%) are used in automated tests

2. **Azure Functions Blind Spot**: Despite 12 Functions projects, zero automated validation

3. **Test Command Blind Spot**: Despite 9 test-framework projects, `azd app test` has no integration tests

4. **Manual Testing Risk**: Relying on manual testing for complex scenarios (orchestration, deployment)

5. **Maintenance Debt**: Deleted projects still referenced in tests (test-pnpm-workspace)

6. **Coverage Illusion**: Having test projects ≠ having test coverage

---

## 📈 RECOMMENDED TEST COVERAGE TARGET

Current: **21%** (10/48 projects)  
Target: **80%** (38/48 projects)  

**Priorities by Category**:
1. Functions: 0% → 80% (add 10 of 12 projects)
2. Test Frameworks: 0% → 80% (add 7 of 9 projects)
3. Orchestration: 0% → 100% (add all 3 projects)
4. Integration: 36% → 70% (add 4 more projects)
5. Package Managers: 71% → 85% (add 1 more project)
6. Reqs-Generate: 0% → 75% (add 3 of 4 projects)

**Total New Tests Needed**: ~28 integration tests across 6-8 new test files

---

**End of Analysis**
