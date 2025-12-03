# Tasks: Health Check for Portless Services

## Progress: 4/5 tasks complete

---

## Task 1: Fix NeedsPort() logic in types.go

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Update `NeedsPort()` method in `cli/src/internal/service/types.go` to return `false` when no ports are explicitly defined in `azure.yaml`.

**Changes Made**:
- Simplified `NeedsPort()` to only return `true` when `len(s.Ports) > 0`
- Removed default behavior of returning `true` for services without ports
- Services without explicit ports now use process-based health checks

---

## Task 2: Update healthcheck_config_test.go for new NeedsPort behavior

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Update unit tests in `cli/src/internal/service/healthcheck_config_test.go` for the `TestService_NeedsPort` test function to reflect the new behavior where services without ports don't need a port.

**Changes Made**:
- Updated test case "no ports no healthcheck" to expect `false`
- Updated test case "healthcheck type process" to expect `false`
- Added new test cases for "healthcheck type http with ports" and "multiple ports"
- All tests pass

---

## Task 3: Verify detector.go health check type selection

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Verified that `DetectServiceRuntime()` in `cli/src/internal/service/detector.go` correctly sets health check type to "process" when `NeedsPort()` returns false.

**Verification**:
- When `NeedsPort()` returns false, `runtime.Port = 0` (line 121)
- When `NeedsPort()` returns false and health check type is "http", it changes to "process" (lines 123-125)
- All service tests pass including the new NeedsPort behavior

---

## Task 4: Fix 400 Bad Request causing "degraded" status for debug ports

**Agent**: Developer  
**Status**: ✅ DONE

**Description**:
Services like `electron` with debug/inspector ports (e.g., `--inspect=5858`) were marked as "degraded" because the Node.js inspector returns HTTP 400 Bad Request for HTTP health check requests. The health monitor should cascade to port/process check instead of marking as degraded.

**Root Cause**:
- Node.js inspector (debug port 5858) uses WebSocket protocol, not HTTP
- When HTTP requests are sent to it, it returns 400 Bad Request
- The health monitor interpreted 4xx as "degraded" instead of "not an HTTP endpoint"

**Changes Made**:
- Updated `tryHTTPHealthCheck()` in `monitor.go` to skip 400 Bad Request responses
- This allows cascading to port check (port is open) → healthy
- Added test `TestTryHTTPHealthCheck_Skips400BadRequest`
- All healthcheck tests pass

---

## Task 5: Integration Test with Portless Services

**Agent**: Tester  
**Status**: 🔲 TODO

**Description**:
Test the fix with a project that has portless services (e.g., TypeScript compilers in watch mode) and services with debug ports.

**Acceptance Criteria**:
- [ ] `azd app run` starts without assigning ports to portless services
- [ ] Health checks use process-based monitoring for portless services
- [ ] Services with ports get ports assigned correctly
- [ ] Services with debug ports show "healthy" (not "degraded") via port check
- [ ] Dashboard shows correct health status for all services
- [ ] No errors in logs related to health checks for portless services
