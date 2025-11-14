# Technical Debt and Deferred Improvements

This document tracks improvements identified during code review, organized by priority. Items are evaluated against security impact, user experience, maintenance burden, API stability, and test value.

---

## 🔴 HIGH PRIORITY

*All high priority items have been completed.*

---

## ⚠️ MEDIUM PRIORITY

### Dashboard Port Assignment Race Condition

**Status:** Deferred  
**Priority:** Medium  
**Effort:** High

**Description**
TOCTOU race condition between port availability check and HTTP server binding in `dashboard/server.go:336-350`.

**Current State**
- Retry logic with `retryWithAlternativePort()` provides mitigation
- 15 retry attempts with randomized ports in 40000-49999 range
- Race window is small (~100ms)

**Rationale for Deferral**
- Current retry logic handles the issue adequately
- Proper fix requires major refactoring (keep listener open, pass FD to HTTP server)
- Extremely rare in practice (requires concurrent dashboard starts)
- User experience impact is minimal (retry succeeds)

**Future Considerations**
- Refactor during HTTP server architecture review
- Consider using SO_REUSEADDR socket option
- Monitor telemetry for retry frequency

---

### Improve Error Context with Command Output

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Low (30 min)

**Description**
Capture stdout/stderr in installer error messages for better debugging.

**Current State**
- Some functions already capture output (e.g., `setupWithPip`)
- Others only report error without output (e.g., `installNodeDependenciesWithWriter`)

**Implementation**
```go
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("failed to install: %w\nOutput: %s", err, string(output))
}
```

**Locations**
- `installer/installer.go:102` (Node dependencies)
- `installer/installer.go:148` (dotnet restore)

**Impact**: Improves debuggability of installation failures

---

### Make Service Orchestration Timeout Configurable

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (30 min)

**Description**
Allow users to configure the 5-minute service startup timeout via environment variable.

**Current State**
- Hardcoded `DefaultServiceStartTimeout = 5 * time.Minute` in `service/constants.go`
- Used in `orchestrator.go:201`

**Implementation**
```go
func getServiceStartTimeout() time.Duration {
    if val := os.Getenv("AZD_SERVICE_START_TIMEOUT"); val != "" {
        if d, err := time.ParseDuration(val); err == nil {
            return d
        }
    }
    return DefaultServiceStartTimeout
}
```

**Rationale for Deferral**
- 5 minutes is sufficient for most workloads
- No user requests for configurability
- Can be added when needed

---

### Enhance Panic Recovery Logging

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (15 min)

**Description**
Add stack traces to panic recovery in parallel installer.

**Location**: `installer/parallel.go:130-144`

**Implementation**
```go
import "runtime/debug"

defer func() {
    if r := recover(); r != nil {
        stack := string(debug.Stack())
        resultsChan <- ProjectInstallResult{
            Error: fmt.Errorf("panic: %v\nStack: %s", r, stack),
        }
    }
}()
```

**Impact**: Easier debugging of panics in parallel operations

---

### Dynamic Python Version Detection on Windows

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (30 min)

**Description**
Replace hardcoded Python paths with dynamic detection in `pathutil/pathutil.go:103-107`.

**Current State**
```go
"C:\\Program Files\\Python312",
"C:\\Program Files\\Python311",
"C:\\Program Files\\Python310",
```

**Implementation**
- Use `filepath.Glob("C:\\Program Files\\Python3*")` to find all installed versions
- Search in order: latest first

**Rationale for Deferral**
- Current hardcoded list covers most common installations
- Python 3.13+ users will hit this eventually
- Low frequency issue (most Python installs add to PATH correctly)

---

## 📝 LOW PRIORITY

### Replace PowerShell with Windows API Calls

**Status:** Deferred  
**Priority:** Low  
**Effort:** High

**Description**
Replace PowerShell process kill with native Windows API for better performance and security.

**Current State**
- Uses PowerShell with parameter binding (secure)
- `portmanager/portmanager.go:650-676`

**Alternative Implementation**
- `GetTcpTable2()` from `iphlpapi.dll` for port -> PID mapping
- `TerminateProcess()` for process termination
- Requires CGo or syscall package

**Rationale for Deferral**
- Current implementation is secure (uses parameter binding)
- Performance difference negligible for this use case
- Adds platform-specific complexity
- PowerShell is always available on Windows

**Future Considerations**
- Consider during performance optimization pass
- Evaluate if PowerShell startup latency becomes issue

---

### Improve Detector Logging Verbosity Control

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (15 min)

**Description**
Use structured debug logging instead of `log.Printf` in detector to avoid log spam.

**Location**: `detector/detector.go:48, 397`

**Current State**
```go
log.Printf("skipping path %s due to error: %v", path, err)
```

**Improvement**
```go
slog.Debug("skipping path due to error",
    slog.String("path", path),
    slog.String("error", err.Error()))
```

**Impact**: Reduces log noise in large codebases with many permission-denied directories

---

### Extract Magic Numbers to Constants

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (15 min)

**Description**
Extract remaining magic numbers to named constants.

**Locations**
- `dashboard/server.go:401`: `time.Sleep(100 * time.Millisecond)`
- `dashboard/server.go:454`: `time.Sleep(100 * time.Millisecond)`

**Suggested Constants**
```go
const (
    ServerStartupDelay = 100 * time.Millisecond
)
```

---

### Global Orchestrator Dependency Injection

**Status:** Deferred  
**Priority:** Low  
**Effort:** High

**Description**
Refactor the global orchestrator to use dependency injection instead of package-level state.

**Rationale for Deferral**
- Requires major refactoring across multiple packages
- Low security impact
- Current implementation is functional
- Would be breaking change for internal code

**Future Considerations**
- Consider during next major architectural refactor
- Could improve testability
- Would reduce package-level coupling

---

### Runner Test Coverage

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium

**Description**
Increase unit test coverage for runner package beyond current 37.5%.

**Rationale for Deferral**
- Existing integration tests provide adequate coverage
- Runner has good integration test coverage
- Functions primarily orchestrate external processes
- Unit testing would require extensive mocking
- Current coverage adequate for reliability

**Future Considerations**
- Add integration tests for new runner functions
- Consider table-driven tests for edge cases
- Focus on error handling paths

---

### Functional Options Pattern

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium

**Description**
Implement functional options pattern for internal packages (e.g., installer, runner, executor).

**Rationale for Deferral**
- Breaking API change for internal code
- Low ROI given internal-only usage
- Current explicit parameter approach is clear
- Would add complexity without significant benefit
- Not a common pattern in Go CLI tools

**Future Considerations**
- Consider if APIs become public
- Evaluate if configuration complexity grows significantly
- Review if option combinations become problematic

---

## 📋 DOCUMENTATION NEEDS

*All current documentation needs have been completed.*

---

## ✅ COMPLETED

### WebSocket Origin Validation

**Status:** ✅ Fixed (Nov 14, 2025)  
**Priority:** Critical  

**Description**
Fixed Cross-Site WebSocket Hijacking (CSWSH) vulnerability in dashboard server.

**Implementation**
- Added origin validation to only allow localhost connections
- Prevents malicious websites from connecting to local dashboard
- Empty origin allowed for non-browser WebSocket clients

**Location**: `dashboard/server.go:40-56`

---

### WebSocket Security Tests

**Status:** ✅ Completed (Nov 14, 2025)  
**Priority:** High  

**Description**
Added comprehensive tests for WebSocket origin validation to prevent CSWSH vulnerability regressions.

**Implementation**
- 13 test cases covering allowed/blocked origins
- Specific CSWSH attack scenario testing
- Edge case coverage (IDN homograph, subdomain attacks)

**Location**: `dashboard/server_security_test.go`
**Test Results**: All passing

---

### Document Windows PATH Refresh Limitations

**Status:** ✅ Completed (Nov 14, 2025)  
**Priority:** Medium  

**Description**
Added documentation explaining PATH refresh behavior and limitations for both Windows and Unix systems.

**Implementation**
- Added detailed PATH refresh behavior explanation
- Documented Windows registry refresh limitations
- Explained Unix shell profile requirements
- Included troubleshooting steps for users

**Location**: `cli/docs/commands/reqs.md` (troubleshooting section)

---

### Unix Process Kill Command Injection

**Status:** ✅ Fixed (PR #48)  
**Priority:** High  

**Description**
Fixed command injection vulnerability in Unix process kill function.

**Previous Code** (vulnerable):
```go
shScript := fmt.Sprintf("lsof -ti:%d | xargs -r kill -9", port)
exec.Command("sh", "-c", shScript)
```

**Fixed Code**:
```go
output, err := exec.CommandContext(ctx, "lsof", "-ti:"+portStr).Output()
pids := strings.Fields(strings.TrimSpace(string(output)))
for _, pid := range pids {
    if _, err := strconv.Atoi(pid); err != nil { continue }
    exec.CommandContext(ctx, "kill", "-9", pid).Run()
}
```

---

## 🎯 Review Criteria

Before moving any deferred item to active development:

1. **Security Impact**: Does it address a security vulnerability?
2. **User Impact**: Does it directly improve user experience?
3. **Maintenance Burden**: Does it reduce ongoing maintenance costs?
4. **API Stability**: Is it worth the breaking change?
5. **Test Value**: Does it meaningfully improve test reliability?

**Activation Threshold**: Items should meet at least 2 of the above criteria.

**Priority Definitions**:
- 🔴 **High**: Security issues, critical bugs, or high user impact
- ⚠️ **Medium**: Quality improvements with moderate user impact
- 📝 **Low**: Nice-to-have improvements, refactoring, optimization
