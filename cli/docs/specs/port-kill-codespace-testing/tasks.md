# Port Kill Codespace Testing Tasks

## Overview
Create robust cross-platform and container testing for port kill functionality.

Spec: [spec.md](spec.md)

---

## Tasks

### 1. Create test helper for spawning server processes
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/testutil_test.go`

**Description**:
Create test utilities to spawn external processes that listen on ports. These processes must be killable and verifiable.

**Acceptance Criteria**:
- [x] Create `SpawnPythonHTTPServer(port int) (*TestServer, error)` helper
- [x] Server should bind to specified port and hold it
- [x] Support graceful and forceful termination via `TestServer.Stop()`
- [x] Cross-platform support (Python with platform-specific process group handling)
- [x] Return process PID and allow verification of running state via `IsRunning()`
- [x] Include timeout handling via `WithTimeout()` option
- [x] Platform-specific files: `testutil_unix_test.go`, `testutil_windows_test.go`

---

### 2. Implement integration tests for actual process killing
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/portmanager_kill_integration_test.go`

**Description**:
Create integration tests tagged with `//go:build integration` that spawn real processes and verify they are killed.

**Acceptance Criteria**:
- [x] `TestKillExternalProcess_SimpleServer`: Spawn server, kill via PortManager, verify port free
- [x] `TestKillExternalProcess_VerifyPortFreed`: After kill, bind to port to prove it's available
- [x] `TestKillExternalProcess_RapidRebind`: Kill and immediately rebind (3 iterations)
- [x] Each test has 30 second timeout (via `WithTimeout(15*time.Second)` for spawns)
- [x] Tests skip gracefully if Python not available (`IsPythonAvailable()`)
- [x] Diagnostic output on failure via `CollectDiagnostics()` helper
- [x] Additional tests: `TestKillExternalProcess_ProcessDetection`, `TestKillExternalProcess_NoProcessOnPort`, `TestKillExternalProcess_MultipleKills`, `TestKillExternalProcess_Timeout`, `TestDiagnostics_CollectInfo`

---

### 3. Implement process tree killing tests
- **Status**: TODO
- **Assignee**: Developer
- **File**: `src/internal/portmanager/portmanager_kill_integration_test.go`

**Description**:
Test that child processes are properly killed along with parent.

**Acceptance Criteria**:
- [ ] `TestKillExternalProcess_WithChildren`: Spawn process that forks children, verify all killed
- [ ] `TestKillExternalProcess_NodeWorker`: Test with Node.js cluster/worker scenario
- [ ] Verify no orphan processes after kill
- [ ] Verify port freed after killing process tree

---

### 4. Add CI workflow for port kill integration tests
- **Status**: DONE
- **Assignee**: DevOps
- **File**: `.github/workflows/ci.yml` (integrated into existing workflow)

**Description**:
Add port kill integration tests to existing CI workflow.

**Acceptance Criteria**:
- [x] Integrated into existing `integration` job (not a separate workflow)
- [x] Matrix: ubuntu-latest, windows-latest, macos-latest (already in CI)
- [x] Python 3.x already installed by existing CI
- [x] Run with `-tags=integration` flag
- [x] 10 minute timeout for port manager tests
- [x] Uses existing artifact upload on failure

---

### 5. Add diagnostic logging to kill function
- **Status**: DONE
- **Assignee**: Developer
- **File**: `src/internal/portmanager/process.go`

**Description**:
Improve error reporting when port kill fails.

**Acceptance Criteria**:
- [x] Log the command being executed (via slog.Debug)
- [x] Log stdout/stderr from kill commands
- [x] Include process name in initial termination log
- [x] Add structured logging with port, PID, error details
- [x] Timeout detection with specific warning

---

### 6. Fix identified issues from testing
- **Status**: IN PROGRESS
- **Assignee**: Developer
- **File**: `src/internal/portmanager/process.go`

**Description**:
Fix any issues discovered during integration testing.

**Notes**:
Tests pass on Windows. Need to run in CI/Codespace to verify Linux/macOS and container environments.

**Acceptance Criteria**:
- [ ] Run tests in CI on all 3 platforms
- [ ] Run container test to validate Codespace-like environment
- [ ] Address any platform-specific failures
- [ ] Fix container-specific issues (if any)
- [ ] Update kill timeouts if needed
- [ ] Add fallback mechanisms for unreliable commands

---

### 7. Add devcontainer test task
- **Status**: TODO
- **Assignee**: Developer
- **File**: `.devcontainer/devcontainer.json`

**Description**:
Add VS Code task to run port kill tests in devcontainer.

**Acceptance Criteria**:
- [ ] Add task or script to run integration tests
- [ ] Document how to run tests in Codespace
- [ ] Add README section for testing in Codespace

---

## Dependencies

- ~~Task 2 depends on Task 1 (need test helpers)~~ DONE
- ~~Task 3 depends on Task 1 (need test helpers)~~ BLOCKED (needs Node.js worker test setup)
- ~~Task 4 depends on Tasks 2-3 (need tests to run)~~ DONE
- Task 6 depends on CI results (need to run in CI)
- Tasks 5 and 7 can run in parallel

## Progress Summary

**Completed:**
- Task 1: Test helper utilities (`testutil_test.go`, platform-specific files)
- Task 2: Integration tests (8 new tests, all passing on Windows)
- Task 4: CI workflow (`.github/workflows/port-kill-tests.yml`)
- Task 5: Diagnostic logging (enhanced `process.go`)

**Remaining:**
- Task 3: Process tree killing tests (Node.js worker scenario)
- Task 6: Fix issues from CI (awaiting CI run)
- Task 7: Devcontainer test task

## Notes

The root cause may be:
1. `pgrep -P` not available or behaving differently in containers
2. Process namespace isolation preventing kill
3. TIME_WAIT state taking longer in containers
4. Permission issues with container processes

Integration tests will help identify the specific issue in Codespaces.
