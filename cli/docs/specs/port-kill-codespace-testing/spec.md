# Port Kill Codespace Testing Spec

## Overview

The port killing functionality is not working effectively in GitHub Codespaces. Despite implementation of process tree killing in `portmanager/process.go`, real-world testing reveals the port kill doesn't reliably free ports in containerized environments.

## Problem Statement

1. **Port killing not effective**: User reported "kill port issue is still there, it's not killing the port effectively" in Codespace
2. **No automated cross-platform testing**: Current tests mock behaviors or run only locally
3. **No container-specific testing**: Codespace/Docker environments have different process semantics

## Root Cause Analysis

### Potential Issues

1. **Process detection**: `ss -tlnp` and `lsof` may behave differently in containers
2. **Permission issues**: Container processes may require different termination signals
3. **Namespace isolation**: PIDs may not translate correctly across container namespaces
4. **Zombie processes**: Child processes may become zombies without proper reaping
5. **Port TIME_WAIT**: TCP TIME_WAIT state may persist longer in containers

### Current Implementation Analysis

From `process.go`:

```go
// Unix kill command
script := fmt.Sprintf(`
# Kill child processes first
for child in $(pgrep -P %d 2>/dev/null); do
    kill -TERM "$child" 2>/dev/null
done
sleep 0.1
# Force kill any remaining children
for child in $(pgrep -P %d 2>/dev/null); do
    kill -9 "$child" 2>/dev/null
done
# Kill the parent process
kill -TERM %d 2>/dev/null
sleep 0.1
kill -9 %d 2>/dev/null || true
`, pid, pid, pid, pid)
```

**Potential gaps:**
- `pgrep -P` may not work in all container environments
- No verification that port was actually freed
- 0.1 second sleeps may be too short in containers

## Requirements

### Functional Requirements

1. **FR1**: Create test that spawns actual external process on a port, then kills it
2. **FR2**: Test must verify port is actually freed after kill (not just process dead)
3. **FR3**: Test must run in CI on ubuntu-latest, windows-latest, macos-latest
4. **FR4**: Add specific test for Codespaces environment (via devcontainer)
5. **FR5**: Test process tree killing with child processes (e.g., Node.js forking)

### Non-Functional Requirements

1. **NFR1**: Tests must not hang indefinitely (max 30 second timeout)
2. **NFR2**: Tests must be deterministic (no race conditions)
3. **NFR3**: Tests should provide diagnostic output on failure

## Design

### Test Strategy

1. **External Process Spawning**
   - Spawn a real process (e.g., Python/Node simple server) that holds a port
   - Use subprocess with explicit port binding
   - Test process actually holds port before kill attempt

2. **Kill Verification**
   - After kill, verify process is dead via PID check
   - After kill, verify port is available via bind attempt
   - Retry verification with backoff for TIME_WAIT

3. **Container-Specific Testing**
   - Add GitHub Actions workflow that runs tests in container
   - Use `services` or Docker-in-Docker for realistic container testing
   - Add devcontainer task for local Codespace testing

### Test Matrix

| Test | Windows | macOS | Linux | Codespace |
|------|---------|-------|-------|-----------|
| Simple port kill | ✓ | ✓ | ✓ | ✓ |
| Process tree kill | ✓ | ✓ | ✓ | ✓ |
| Node.js with workers | ✓ | ✓ | ✓ | ✓ |
| Python with workers | ✓ | ✓ | ✓ | ✓ |
| Rapid kill-rebind | ✓ | ✓ | ✓ | ✓ |

### Test Implementation

Create new test file: `portmanager/portmanager_kill_integration_test.go`

```go
//go:build integration

// Tests that spawn external processes and verify kill behavior
```

Key test functions:
- `TestKillExternalProcess_SimpleServer`
- `TestKillExternalProcess_WithChildren`
- `TestKillExternalProcess_VerifyPortFreed`
- `TestKillExternalProcess_RapidRebind`

### CI Workflow Updates

Add new workflow: `.github/workflows/port-kill-tests.yml`

- Runs on all three platforms
- Uses integration test tag
- Includes container job for realistic Codespace testing
- Reports diagnostics on failure

### Diagnostic Improvements

Add verbose logging on kill failure:
- Show process tree before kill
- Show running processes after kill attempt
- Show port state (netstat/ss output)
- Show any error messages from kill commands

## Implementation Plan

1. Create test helper to spawn server processes
2. Implement cross-platform server stubs (Python preferred for portability)
3. Add integration tests with `//go:build integration` tag
4. Update CI to run integration tests
5. Add container-specific CI job
6. Add diagnostic improvements to kill function
7. Fix any issues discovered

## Success Criteria

1. All integration tests pass on Windows, macOS, Linux, and Codespace
2. Port is verified free within 5 seconds after kill
3. No test flakiness (100% pass rate over 10 runs)
4. Clear diagnostic output on any failure
