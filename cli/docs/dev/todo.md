# Technical Debt and Deferred Improvements

This document tracks improvements identified during code review, organized by priority. Items are evaluated against security impact, user experience, maintenance burden, API stability, and test value.

---

## 🔴 HIGH PRIORITY

*All high priority items have been completed.*

---

## ⚠️ MEDIUM PRIORITY

### MCP Tool for Azure YAML Configuration

**Status:** Planned  
**Priority:** Medium  
**Effort:** Medium

**Description**
Create a new MCP tool that helps Copilot configure the azure.yaml file for any project to take full advantage of azd-app features. When a user says "set this project up for azd app", Copilot should be able to modify their azure.yaml to support all relevant configuration options.

**Current MCP Tools** (in `cli/src/cmd/app/commands/mcp.go`)
- Observability: `get_services`, `get_service_logs`, `get_project_info`
- Operations: `run_services`, `stop_services`, `start_service`, `restart_service`, `install_dependencies`
- Configuration: `check_requirements`, `get_environment_variables`, `set_environment_variable`
- Resources: `azure://project/azure.yaml`, `azure://project/services/configs`

**Proposed New Tools**

1. **`configure_service`** - Configure a service in azure.yaml with azd-app features
   - Set `entrypoint` (e.g., main.py, app.py)
   - Set `command` for custom run commands
   - Set `type` (http, process, container)
   - Set `mode` (watch, task, build)
   - Configure `ports` (Docker Compose style)
   - Configure `environment` variables
   - Configure `healthcheck` (http, tcp, process, output patterns)
   - Set `uses` for service dependencies

2. **`get_azure_yaml_schema`** - Return the azure.yaml JSON schema
   - Helps Copilot understand all available configuration options
   - Returns schema from `schemas/v1.1/azure.yaml.json`

3. **`detect_project_services`** - Analyze project structure and suggest service configurations
   - Detect language/framework from project files
   - Suggest appropriate entry points
   - Suggest appropriate health check types
   - Suggest port configurations
   - Return recommended azure.yaml configuration

4. **`validate_azure_yaml`** - Validate azure.yaml against schema
   - Report validation errors
   - Suggest fixes for common issues

**Key Azure YAML Features to Expose**

From `schemas/v1.1/azure.yaml.json`:
- `entrypoint`: Entry point file (azd-app addition)
- `command`: Custom run command
- `type`: Service type (http, process, container)
- `mode`: Execution mode (watch, task, build)
- `ports`: Docker Compose style port mappings
- `environment`: Environment variables (array or object format)
- `healthcheck`: Health check configuration
  - `type`: http, tcp, process, output, none
  - `path`: HTTP endpoint path
  - `pattern`: Regex for output-based health checks
  - `interval`, `timeout`, `retries`, `start_period`
- `uses`: Service/resource dependencies
- `logs.filters`: Log filtering configuration

**Example Usage**

User: "Set up my Python Flask API for azd app"

Copilot uses `detect_project_services` to analyze the project, then `configure_service` to update azure.yaml:

```yaml
services:
  api:
    language: python
    project: ./api
    entrypoint: app.py
    ports: ["5000"]
    healthcheck:
      type: http
      path: /health
      interval: 10s
    environment:
      FLASK_ENV: development
```

**Location**
- New tools in `cli/src/cmd/app/commands/mcp.go`
- May need new internal package for azure.yaml manipulation

**Rationale**
- Improves user experience for azd-app adoption
- Enables AI-assisted configuration
- Reduces manual azure.yaml editing errors
- Leverages full schema capabilities

---

### Add Animations to Dashboard UI Components

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Medium

**Description**
Add smooth animations to dashboard components for better visual feedback and user experience.

**Affected Components**
1. **Services Grid View** (`ServiceCard.tsx`)
   - Entry animations when cards appear
   - Status change transitions (healthy → unhealthy)
   - Hover state animations
   
2. **Services Table View** (`ServiceTable.tsx`)
   - Row entry animations on initial load
   - Status badge transitions
   - Sorting animation when columns change
   
3. **Header Status Indicators** (`Header.tsx`, `ServiceStatusCard.tsx`)
   - Health count change animations (number transitions)
   - Status pill color transitions
   - Connection status pulse animations

**Implementation Approach**
- Use CSS transitions for simple state changes
- Consider Framer Motion for complex animations (already available)
- Respect `prefers-reduced-motion` media query
- Keep animations subtle (150-300ms duration)

**Rationale for Deferral**
- Core functionality works without animations
- Needs design review for animation timing/easing
- Should be implemented consistently across all components

---

### Stop Button Intermittent Failure in Console View

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Medium

**Description**
The stop button on service cards in the Console view sometimes doesn't work, particularly on the first click after loading.

**Symptoms**
- First click on stop button does nothing
- Subsequent clicks may work
- Occurs on LogsPane header service action buttons

**Possible Causes**
1. **Event handler not bound**: React event handler may not be attached on first render
2. **Race condition**: Service operations context may not be fully initialized
3. **Click propagation**: Event may be stopped or captured by parent elements
4. **State sync issue**: Button may be checking stale service state

**Investigation Needed**
- Add logging to `ServiceActions` click handlers
- Check if `useServiceOperations` hook is ready on first render
- Verify event propagation through LogsPane header
- Check for any `e.stopPropagation()` issues

**Location**: 
- `dashboard/src/components/ServiceActions.tsx` - Action button handlers
- `dashboard/src/components/LogsPane.tsx` - Header with service actions
- `dashboard/src/hooks/useServiceOperations.ts` - Service operation logic

**Rationale for Deferral**
- Workaround exists (click again)
- Needs debugging session with running services
- May be related to WebSocket connection timing

---

### Dashboard State/Health Not Updating in Real-Time

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Medium

**Description**
Service state and health status sometimes doesn't update in the dashboard until a page refresh. The UI shows stale data even though the backend has the correct state.

**Symptoms**
- Service shows old status (e.g., "running" when actually stopped)
- Health indicators don't update after service state changes
- Page refresh fixes the issue and shows correct state
- More likely to occur after rapid state changes

**Possible Causes**
1. **WebSocket message dropped**: Message may be lost during reconnection
2. **React state not updating**: State update may not trigger re-render
3. **Health report not propagating**: Health stream may miss updates
4. **Stale closure in event handler**: WebSocket handler using old state

**Investigation Needed**
- Add WebSocket message logging to verify messages received
- Check if `healthReport` state updates trigger re-renders
- Verify WebSocket reconnection logic preserves state sync
- Check for race conditions between REST and WebSocket updates

**Location**: 
- `dashboard/src/App.tsx` - Main state management
- `dashboard/src/hooks/useHealthStream.ts` - WebSocket health stream
- `dashboard/src/contexts/ServicesContext.tsx` - Services state context
- `cli/src/internal/dashboard/server.go` - WebSocket server

**Rationale for Deferral**
- Workaround exists (page refresh)
- Needs debugging with network inspection
- May require WebSocket protocol changes

---

### Services Stuck in "Starting" Status in Dashboard

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Medium

**Description**
Services like `data-processor` (type: process, mode: task) and `dotnet-watch` (type: http with ports) show as "starting" in the dashboard even after they're running and healthy.

**Root Cause Analysis**
1. **Build/task mode services**: Fixed logic in `performBuildTaskHealthCheck` to check PID > 0 before grace period for quickly-exiting tasks
2. **HTTP services (dotnet-watch)**: Port may not be populated in health check when registry doesn't have it. Added fallback to azure.yaml ports.
3. **Registry entry missing Type/Mode**: Fixed `service_control.go` to include Type and Mode when registering services

**Files Modified**
- `healthcheck/monitor.go`: Reordered PID check logic, added port fallback from azure.yaml
- `cmd/app/commands/service_control.go`: Added Type/Mode to registry entry
- `healthcheck/build_task_test.go`: Added regression tests

**Remaining Investigation Needed**
- Debug logging added but services were killed during build
- Need to trace full flow with services running to confirm fixes work
- May need additional fixes for how health status flows to dashboard

**Location**: 
- `healthcheck/monitor.go:1325-1395` - Build/task health check logic
- `healthcheck/monitor.go:510-540` - Service list building with azure.yaml fallback
- `cmd/app/commands/service_control.go:301-316` - Registry entry creation

---

### Registry File Permissions Inconsistency

**Status:** Deferred  
**Priority:** Medium  
**Effort:** Low (10 min)

**Description**
ServiceRegistry creates `.azure/` directory with 0750 permissions but writes files with 0600.
PortManager creates `.azure/` with 0700. Inconsistent directory permissions.

**Location**: 
- `registry/registry.go:76` - `os.MkdirAll(registryDir, 0750)`
- `portmanager/portmanager.go:153` - `os.MkdirAll(portsDir, 0700)`

**Recommendation**
Standardize to 0700 for directory and 0600 for files for consistency.

**Rationale for Deferral**
- Both permissions are secure (owner-only access)
- 0750 vs 0700 has minimal security difference
- No actual security vulnerability
- Low user impact

---

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

**Note**: Should update to include Python 3.13, 3.14 when they release.

---

### Logs Command: Refactor Global Flag Variables

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium (2-3 hours)

**Description**
Replace global variables for command flags with struct-based options for thread safety and testability.

**Current State** (`commands/logs.go:22-32`)
```go
var (
    logsFollow     bool
    logsService    string
    logsTail       int
    logsSince      string
    logsTimestamps bool
    logsNoColor    bool
    logsLevel      string
    logsFormat     string
    logsFile       string
    logsExclude    string
    logsNoBuiltins bool
)
```

**Proposed Implementation**
```go
type LogsOptions struct {
    Follow     bool
    Service    string
    Tail       int
    Since      string
    Timestamps bool
    NoColor    bool
    Level      string
    Format     string
    File       string
    Exclude    string
    NoBuiltins bool
}

func NewLogsCommand() *cobra.Command {
    opts := &LogsOptions{Tail: 100, Timestamps: true}
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            return runLogs(cmd, args, opts)
        },
    }
    cmd.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "...")
    // ... bind other flags
    return cmd
}
```

**Benefits**
- Thread-safe (no shared mutable state)
- Easier unit testing (inject options directly)
- Clearer function signatures
- Consistent with Go best practices

**Rationale for Deferral**
- Would require refactoring all commands for consistency
- Current implementation works correctly for CLI use
- No concurrent command execution in practice

---

### Logs Command: Unbounded Memory When Reading Large Log Files

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium (1-2 hours)

**Description**
When reading rotated log files without a `--since` filter, all entries are loaded into memory before applying the tail limit. For services with heavy logging, this could load ~3MB of parsed JSON objects.

**Current State** (`commands/logs.go:237-254`)
```go
for _, logFile := range logFiles {
    entries, err := readSingleLogFile(logFile, serviceName, sinceTime)
    // ...
    allEntries = append(allEntries, entries...)  // All entries loaded
}
// Tail applied AFTER loading everything
if tail > 0 && len(allEntries) > tail {
    allEntries = allEntries[len(allEntries)-tail:]
}
```

**Proposed Implementation**
Use a streaming/ring buffer approach:
```go
// Keep only tail entries during reading
ringBuffer := make([]service.LogEntry, 0, tail)
for _, logFile := range logFiles {
    readLogFileStreaming(logFile, func(entry service.LogEntry) {
        if len(ringBuffer) >= tail {
            ringBuffer = ringBuffer[1:]  // Drop oldest
        }
        ringBuffer = append(ringBuffer, entry)
    })
}
```

**Alternative**: Read files in reverse order (newest first), stop when tail limit reached.

**Rationale for Deferral**
- 3MB is acceptable for CLI memory usage
- Most users use `--since` or reasonable `--tail` values
- Would add complexity to log reading logic

---

### Logs Command: Dashboard Ping Latency

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (30 min)

**Description**
The `dashboardClient.Ping(ctx)` call blocks for up to `DashboardAPITimeout` (likely 30s) if the dashboard is unresponsive. This slows down the command when dashboard is down.

**Current State** (`commands/logs.go:89-93`)
```go
if err := dashboardClient.Ping(ctx); err != nil {
    output.Info("No services are currently running")
    // ...
}
```

**Proposed Implementation**
1. Reduce ping timeout specifically for logs command:
```go
pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
if err := dashboardClient.Ping(pingCtx); err != nil {
```

2. Or make timeout configurable per-client:
```go
client := dashboard.NewClient(ctx, cwd, dashboard.WithPingTimeout(2*time.Second))
```

**Rationale for Deferral**
- Dashboard is usually responsive when running
- Only affects edge case (dashboard hung but not crashed)
- Current behavior is safe (just slow)

---

### Logs Command: WebSocket Rate Limiting Warning

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (20 min)

**Description**
When WebSocket log streaming drops messages due to channel congestion, there's no warning to the user. Messages are silently lost.

**Current State** (`dashboard/client.go:282-286`)
```go
select {
case logs <- entry:
case <-time.After(100 * time.Millisecond):
    // Drop if channel is full/slow - NO WARNING
case <-ctx.Done():
    return ctx.Err()
}
```

**Proposed Implementation**
Add a dropped message counter and periodic warning:
```go
var droppedCount int64
select {
case logs <- entry:
case <-time.After(100 * time.Millisecond):
    atomic.AddInt64(&droppedCount, 1)
    if droppedCount == 1 || droppedCount%100 == 0 {
        slog.Warn("log entries dropped due to slow consumer", 
            slog.Int64("dropped", droppedCount))
    }
case <-ctx.Done():
    return ctx.Err()
}
```

**Rationale for Deferral**
- Dropped messages are rare in practice
- User can increase terminal scroll buffer if needed
- Adding warning may clutter output

---

### Logs Command: Add Grep/Search Functionality

**Status:** Deferred  
**Priority:** Low  
**Effort:** Medium (2-3 hours)

**Description**
Add `--grep` flag to filter log output by pattern matching, similar to `grep` or `docker logs --grep`.

**Proposed Flags**
- `--grep <pattern>`: Filter logs to show only lines matching the regex pattern
- `--context <N>`: Show N lines before and after each match (like `grep -C`)
- `--highlight`: Highlight matching text in output (ANSI colors)

**Example Usage**
```bash
azd app logs --grep "error|warning" --context 2
azd app logs -f --grep "database" --highlight
azd app logs --service api --grep "request.*500"
```

**Implementation Approach**
1. Add flags to `NewLogsCommand()`
2. Compile regex pattern in `validateLogsInputs()`
3. Filter in `displayLogsText()` before output
4. For `--context`, maintain a ring buffer of recent lines
5. For `--highlight`, wrap matches with ANSI color codes

**Rationale for Deferral**
- Users can pipe to `grep` as workaround (`azd app logs | grep pattern`)
- Medium effort feature
- Core logs functionality is complete

---

### Logs Command: Shell Completion for Service Names

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (1 hour)

**Description**
Add shell completion for the `--service` flag to autocomplete available service names.

**Implementation**
```go
cmd.RegisterFlagCompletionFunc("service", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Read azure.yaml to get service names
    services := getServiceNamesFromConfig()
    return services, cobra.ShellCompDirectiveNoFileComp
})
```

**Benefits**
- Tab completion for service names
- Reduces typos
- Better CLI UX

**Rationale for Deferral**
- Project doesn't currently use shell completions anywhere
- Would need to add completion infrastructure first
- Low priority enhancement

---

## 📝 LOW PRIORITY

### Add Stack Traces to Panic Recovery

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (20 min)

**Description**
Enhance panic recovery handlers with stack traces for easier debugging.

**Locations**: 
- `installer/parallel.go:133-138`
- `commands/run.go:276-279, 349-352`

**Implementation**
```go
import "runtime/debug"

defer func() {
    if r := recover(); r != nil {
        stack := string(debug.Stack())
        output.Error("Panic: %v\nStack: %s", r, stack)
    }
}()
```

**Benefits**
- Faster root cause analysis
- Better production debugging
- No performance impact (only on panic)

**Rationale for Deferral**
- Panics are rare in production
- Current recovery prevents crashes
- Can add when needed for specific issues

---

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

### Migrate to Structured Logging (slog)

**Status:** Deferred  
**Priority:** Low  
**Effort:** Low (30 min)

**Description**
Migrate from `log.Printf` to structured logging with `slog` package for better observability.

**Locations**: 
- `detector/detector.go:48, 397` - path traversal errors
- Other log.Printf calls throughout codebase

**Current State**
```go
log.Printf("skipping path %s due to error: %v", path, err)
```

**Improvement**
```go
slog.Debug("skipping path during detection",
    slog.String("path", path),
    slog.String("error", err.Error()))
```

**Benefits**
- Structured logs easier to parse/filter
- Better observability in production
- Log level control (debug/info/warn/error)

**Rationale for Deferral**
- Current logging is functional
- Low priority enhancement
- Can migrate incrementally

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
