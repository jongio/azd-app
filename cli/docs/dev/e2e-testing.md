# E2E Integration Testing for Health Command

## Overview

Comprehensive end-to-end integration tests that verify the `azd app health` command works correctly across all platforms in real-world scenarios.

## Test Coverage

### TestHealthCommandE2E_FullWorkflow
Complete workflow test that:
1. ✅ Installs dependencies (`azd app deps`)
2. ✅ Starts all services (`azd app run`)
3. ✅ Runs basic health check
4. ✅ Tests JSON output format and validation
5. ✅ Tests table output format
6. ✅ Tests service filtering
7. ✅ Tests verbose mode
8. ✅ Tests streaming mode (real-time updates)
9. ✅ Verifies service info command
10. ✅ Cleans up processes properly

### TestHealthCommandE2E_ErrorCases
Error handling scenarios:
- ✅ No services running (reports unhealthy gracefully)
- ✅ Invalid output format (proper error message)
- ✅ Invalid interval parameter (validation works)

### TestHealthCommandE2E_CrossPlatform
Platform-specific functionality:
- ✅ Process checking works on Windows/Linux/macOS
- ✅ Services detected correctly on each platform
- ✅ Platform-specific process management

## Running E2E Tests

### Locally

```bash
# From cli directory
cd cli

# Run all E2E tests
mage testE2E

# Or use go test directly
go test -v -tags=integration -timeout=15m ./src/cmd/app/commands -run TestHealthCommandE2E

# Run specific test
go test -v -tags=integration ./src/cmd/app/commands -run TestHealthCommandE2E_FullWorkflow
```

### In CI/CD

**Automatic triggers:**
- Every PR that changes health command code
- Changes to healthcheck package
- Changes to health test project
- Changes to E2E test file

**Manual trigger:**
```bash
# Via GitHub CLI
gh workflow run health-e2e.yml

# Or use GitHub UI:
# Actions → Health Command E2E Tests → Run workflow
```

**Matrix testing:**
- ✅ Ubuntu (latest)
- ✅ Windows (latest)
- ✅ macOS (latest)
- ✅ Go 1.25

## Test Project

**Location:** `cli/tests/projects/health-test`

**Services:**
1. **web** - Node.js with HTTP health endpoint
2. **api** - Python Flask with /healthz endpoint
3. **database** - Node.js TCP server (port check only)
4. **worker** - Python background process (process check only)
5. **admin** - Node.js with authentication

**Configuration:** `azure.yaml` with Docker Compose-compatible healthcheck definitions

## CI/CD Workflows

### Main CI Workflow
**File:** `.github/workflows/ci.yml`

Includes integration test job that:
- Installs all test project dependencies
- Builds dashboard assets
- Runs E2E tests with 15-minute timeout
- Uploads logs on failure
- Runs on all platforms (Ubuntu, Windows, macOS)

### Health E2E Workflow
**File:** `.github/workflows/health-e2e.yml`

Dedicated E2E testing workflow:
- Manual dispatch with OS selection
- Auto-triggers on health command changes
- 20-minute timeout per platform
- Separates test phases (full workflow, error cases, cross-platform)
- Detailed test reports in GitHub step summary
- Artifact uploads for debugging

## Key Features

### Cross-Platform Testing
- Windows process checking (different than Unix)
- Platform-specific cleanup (taskkill on Windows)
- Path handling differences

### Real Service Testing
- Actual Node.js and Python services
- Real HTTP servers and health endpoints
- TCP port listening
- Background workers

### Comprehensive Validation
- Exit code checking
- JSON schema validation
- Output format verification
- Service detection accuracy
- Timing and timeout handling

### Robust Cleanup
- Context cancellation
- Process kill on all platforms
- Temporary directory cleanup
- Graceful shutdown attempts

## Test Execution Flow

```
1. Build azd binary in temp directory
   ↓
2. Install test project dependencies
   ↓
3. Start services with azd app run (background)
   ↓
4. Wait for initialization (90s)
   ↓
5. Run health checks (various modes)
   ↓
6. Validate outputs and behaviors
   ↓
7. Cleanup: Stop services, kill processes
   ↓
8. Remove temp binary
```

## Debugging Failed Tests

### Local debugging:
```bash
# Run with verbose output
go test -v -tags=integration ./src/cmd/app/commands -run TestHealthCommandE2E_FullWorkflow

# Check for orphaned processes
# Windows:
tasklist | findstr "node python"
# Unix:
ps aux | grep -E "node|python"

# Manual cleanup if needed
# Windows:
taskkill /F /IM node.exe
# Unix:
pkill -9 node
```

### CI debugging:
1. Check "Collect logs on failure" step for system info
2. Download "e2e-test-logs-*" artifacts
3. Review GitHub step summary for test report
4. Look for timeout issues (may need to increase serviceTimeout)

## Configuration

### Timeouts
- **healthTimeout**: 5 minutes (overall test timeout)
- **serviceTimeout**: 90 seconds (service initialization wait)
- **Individual commands**: 10-30 seconds (context timeout)

### Dependencies
- Go 1.25
- Node.js 20
- Python 3.11
- Platform-specific tools (npm, pip)

## Maintenance

### Adding New Tests
1. Add test function to `health_e2e_test.go`
2. Follow naming convention: `TestHealthCommandE2E_<Feature>`
3. Use `testing.Short()` check for unit test mode
4. Ensure proper cleanup with defer
5. Add to workflow if needed

### Updating Test Project
1. Modify `tests/projects/health-test/azure.yaml`
2. Update service implementations if needed
3. Re-run E2E tests to verify changes
4. Update README if service behavior changes

## Success Metrics

✅ All tests pass on all platforms (Ubuntu, Windows, macOS)
✅ No orphaned processes after test completion
✅ Tests complete within timeout (< 15 minutes total)
✅ Exit codes match expectations (0 for healthy, 1 for unhealthy)
✅ JSON output parses correctly
✅ Service detection works (5 services found)
✅ Cleanup succeeds without errors
