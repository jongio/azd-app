# azd-core Integration Opportunities Analysis

**Date**: 2026-01-10  
**Status**: Comprehensive analysis of remaining integration opportunities

## Executive Summary

azd-app and azd-exec have **partially integrated** azd-core packages. This document identifies concrete integration opportunities for the **NOT YET INTEGRATED** packages.

### Current Integration Status

| Package | azd-exec | azd-app | Status |
|---------|----------|---------|---------|
| **shellutil** | ✅ DONE | ❌ Not used | Integrated in azd-exec only |
| **keyvault** | ✅ DONE | ✅ DONE | Fully integrated |
| **fileutil** | ❌ Not used | ✅ DONE | Integrated in azd-app only |
| **security** | ❌ Not used | ✅ DONE | Integrated in azd-app only |
| **browser** | ❌ Not used | ✅ DONE | Integrated in azd-app only |
| **procutil** | ❌ Not used | ✅ DONE | Integrated in azd-app only |
| **pathutil** | ❌ Not used | ✅ PARTIAL | Only used in reqs command |
| **env** | ❌ Not used | ❌ Not used | **NOT INTEGRATED** |

---

## 1. env/ Package - Environment Variable Management

**Status**: ❌ NOT INTEGRATED in either project  
**Package**: `github.com/jongio/azd-core/env`

### What It Provides

```go
// Get environment variable with default
value := env.Get("API_KEY", "default-value")

// Get required environment variable (panics if missing)
value := env.MustGet("DATABASE_URL")

// Set environment variable
env.Set("NODE_ENV", "production")

// Get all environment variables as map
envMap := env.GetAll()

// Check if variable exists
if env.Has("DEBUG") {
    // ...
}
```

### Integration Opportunities in azd-app

#### 1. Replace os.Environ() Map Building Pattern

**Current Pattern** (appears in 5+ files):
```go
// internal/serviceinfo/serviceinfo.go
environmentCache = make(map[string]string)
for _, env := range os.Environ() {
    parts := strings.SplitN(env, "=", 2)
    if len(parts) != 2 {
        continue
    }
    environmentCache[parts[0]] = parts[1]
}
```

**With azd-core/env**:
```go
environmentCache = env.GetAll()
```

**Files to Update**:
- [internal/serviceinfo/serviceinfo.go](c:\code\azd-app\cli\src\internal\serviceinfo\serviceinfo.go#L30-L40)
- [internal/dashboard/service_operations.go](c:\code\azd-app\cli\src\internal\dashboard\service_operations.go#L590-L600)

**Impact**: Simpler, cleaner code; eliminates manual splitting logic

#### 2. Environment Variable Access with Defaults

**Current Pattern**:
```go
// Multiple test files
originalCodespaces := os.Getenv("CODESPACES")
if originalCodespaces == "" {
    // handle default
}
```

**With azd-core/env**:
```go
originalCodespaces := env.Get("CODESPACES", "")
```

**Files to Update**:
- [internal/security/validation_test.go](c:\code\azd-app\cli\src\internal\security\validation_test.go#L430-L440)
- [internal/testing/output_mode_test.go](c:\code\azd-app\cli\src\internal\testing\output_mode_test.go#L185-L195)

**Impact**: Reduces boilerplate for default value handling

---

## 2. pathutil/ Package - Tool Discovery and PATH Management

**Status**: ✅ PARTIAL in azd-app (only reqs command), ❌ NOT in azd-exec  
**Package**: `github.com/jongio/azd-core/pathutil`

### What It Provides

```go
// Find tool in PATH (auto .exe handling on Windows)
nodePath := pathutil.FindToolInPath("node")

// Search common installation directories
pythonPath := pathutil.SearchToolInSystemPath("python")

// Get installation URL for missing tools
suggestion := pathutil.GetInstallSuggestion("docker")
// Returns: "https://docs.docker.com/get-docker/"

// Refresh PATH from system (Windows registry, Unix environment)
newPath, err := pathutil.RefreshPATH()
```

### Integration Opportunities in azd-app

#### 1. Replace Custom Browser Executable Finding

**Current Code**: [internal/notify/notify_windows.go](c:\code\azd-app\cli\src\internal\notify\notify_windows.go#L216-L232)
```go
func (w *windowsNotifier) findAzdExecutable() string {
    // Try to find azd in PATH first
    if path, err := exec.LookPath("azd"); err == nil {
        return path
    }

    // Check common installation locations
    locations := []string{
        filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Azure Dev CLI", "azd.exe"),
        filepath.Join(os.Getenv("ProgramFiles"), "Azure Dev CLI", "azd.exe"),
        filepath.Join(os.Getenv("ProgramFiles(x86)"), "Azure Dev CLI", "azd.exe"),
    }
    for _, loc := range locations {
        if _, err := os.Stat(loc); err == nil {
            return loc
        }
    }
    return ""
}
```

**With azd-core/pathutil**:
```go
func (w *windowsNotifier) findAzdExecutable() string {
    // Try PATH first
    if path := pathutil.FindToolInPath("azd"); path != "" {
        return path
    }
    
    // Search system directories (includes Program Files automatically)
    return pathutil.SearchToolInSystemPath("azd")
}
```

**Impact**: 
- Eliminates 15 lines of custom code
- Cross-platform (works on Linux/macOS too)
- Searches more locations (Homebrew, /usr/local/bin, etc.)

#### 2. Enhance Prerequisites Checker with Installation Suggestions

**Current Code**: [cmd/app/commands/reqs.go](c:\code\azd-app\cli\src\cmd\app\commands\reqs.go#L110-L140) already uses `pathutil.FindToolInPath()`, but could be enhanced:

**Enhancement**:
```go
// When tool is missing, provide installation URL
toolPath := pathutil.FindToolInPath("docker")
if toolPath == "" {
    suggestion := pathutil.GetInstallSuggestion("docker")
    if suggestion != "" {
        output.Warning("Docker not found. Install from: %s", suggestion)
    }
}
```

**Impact**: Better user experience with actionable error messages

#### 3. Replace exec.LookPath Usage

**Current Pattern** (appears in multiple files):
```go
// azd-exec: internal/executor/command_builder_test.go
_, err := exec.LookPath(cmd.Args[0])
```

**With azd-core/pathutil**:
```go
path := pathutil.FindToolInPath(cmd.Args[0])
if path == "" {
    // handle not found
}
```

**Files to Update**:
- azd-exec: [internal/executor/command_builder_test.go](c:\code\azd-exec\cli\src\internal\executor\command_builder_test.go#L105)
- azd-app: [internal/notify/notify_windows.go](c:\code\azd-app\cli\src\internal\notify\notify_windows.go#L218)

**Impact**: Automatic .exe handling on Windows, cleaner API

---

## 3. procutil/ Package - Process Detection

**Status**: ✅ DONE in azd-app, ❌ NOT in azd-exec  
**Package**: `github.com/jongio/azd-core/procutil`

### What It Provides

```go
// Check if process is running (uses gopsutil for reliability)
if procutil.IsProcessRunning(pid) {
    fmt.Println("Process is alive")
}
```

### azd-app Integration Status

azd-app **already uses** procutil from azd-core:
- [internal/healthcheck/checker.go](c:\code\azd-app\cli\src\internal\healthcheck\checker.go#L20)
- [internal/monitor/state_monitor.go](c:\code\azd-app\cli\src\internal\monitor\state_monitor.go#L13)

### Integration Opportunities in azd-exec

**Status**: azd-exec has NO process detection needs currently  
**Reason**: azd-exec only runs scripts, doesn't manage long-running processes

**Potential Future Use Case**: If azd-exec adds background script execution or daemon management

---

## 4. browser/ Package - Cross-Platform Browser Launching

**Status**: ✅ DONE in azd-app, ❌ NOT in azd-exec  
**Package**: `github.com/jongio/azd-core/browser`

### What It Provides

```go
// Launch URL in system default browser
err := browser.Launch(browser.LaunchOptions{
    URL:     "http://localhost:4280",
    Target:  browser.TargetSystem,
    Timeout: 5 * time.Second,
})
```

### azd-app Integration Status

azd-app **already uses** browser from azd-core:
- [cmd/app/commands/run.go](c:\code\azd-app\cli\src\cmd\app\commands\run.go#L23) - Dashboard browser launch
- [cmd/app/commands/run_test.go](c:\code\azd-app\cli\src\cmd\app\commands\run_test.go#L15) - Tests

### Integration Opportunities in azd-exec

**Status**: azd-exec has NO browser launching needs currently  
**Reason**: azd-exec is a command-line script executor with no UI components

**Potential Future Use Case**: If azd-exec adds web-based logs viewer or debugging UI

---

## 5. What's Already Integrated

### ✅ azd-app Successfully Integrated

1. **fileutil** - Used in 8 files for atomic JSON writes:
   - internal/config/notifications.go
   - internal/config/config.go
   - internal/cache/reqs_cache.go
   - internal/detector/detector.go
   - internal/detector/detector_functions.go
   - internal/service/detector_helpers.go
   - internal/dashboard/classifications.go

2. **security** - Used in 20+ files for:
   - Path validation (ValidatePath)
   - Service name validation (ValidateServiceName)
   - Script name sanitization (SanitizeScriptName)
   - Package manager validation (ValidatePackageManager)

3. **keyvault** - Used in service environment resolution:
   - internal/service/environment.go - Azure Key Vault reference resolution

4. **browser** - Used in dashboard:
   - cmd/app/commands/run.go - Opens dashboard in browser

5. **procutil** - Used in health checks and monitoring:
   - internal/healthcheck/checker.go
   - internal/monitor/state_monitor.go

6. **pathutil** - Used in prerequisites checker:
   - cmd/app/commands/reqs.go

### ✅ azd-exec Successfully Integrated

1. **shellutil** - Used for shell detection:
   - internal/executor/constants.go
   - internal/executor/executor.go
   - internal/executor/command_shell_test.go

2. **keyvault** - Used for Key Vault reference resolution:
   - internal/executor/executor.go
   - Multiple test files

---

## 6. Implementation Priority

### High Priority (Clear Value, Low Risk)

1. **env.GetAll() in azd-app** - Replace manual os.Environ() parsing
   - **Effort**: 30 minutes
   - **Impact**: Eliminates duplicate code in 2 files
   - **Risk**: None (drop-in replacement)

2. **pathutil enhancement in azd-app reqs** - Add installation suggestions
   - **Effort**: 1 hour
   - **Impact**: Better UX when prerequisites missing
   - **Risk**: None (additive feature)

3. **pathutil in notify_windows.go** - Replace custom azd executable finding
   - **Effort**: 30 minutes
   - **Impact**: Simpler, cross-platform code
   - **Risk**: Low (existing code is Windows-only anyway)

### Medium Priority (Good Value, Needs Testing)

4. **env.Get() with defaults in tests** - Replace os.Getenv with default handling
   - **Effort**: 2 hours (20+ test files)
   - **Impact**: Cleaner test code
   - **Risk**: Low (tests only)

### Low Priority (Nice to Have)

5. **exec.LookPath → pathutil.FindToolInPath** - Standardize tool finding
   - **Effort**: 1 hour
   - **Impact**: Consistency across codebase
   - **Risk**: Low (same behavior)

---

## 7. Packages NOT Needed (and Why)

### env/ in azd-exec
**Reason**: azd-exec already has comprehensive environment variable handling via azd-core/keyvault integration. The env package doesn't add value beyond what's already achieved.

### browser/ in azd-exec
**Reason**: azd-exec is a CLI script executor with no UI. Browser launching would be out of scope.

### procutil/ in azd-exec
**Reason**: azd-exec runs scripts synchronously and terminates. No process management needed.

---

## 8. Next Steps

1. **Immediate** (This Week):
   - Integrate env.GetAll() in azd-app serviceinfo and dashboard
   - Add pathutil.GetInstallSuggestion() to reqs command

2. **Short Term** (Next 2 Weeks):
   - Replace notify_windows.go custom code with pathutil
   - Update test files to use env.Get() with defaults

3. **Long Term** (Next Month):
   - Standardize all exec.LookPath to pathutil.FindToolInPath
   - Document integration patterns in CONTRIBUTING.md

---

## 9. File-Specific Integration Map

### Files That Should Use env.GetAll()

| File | Line | Current Code | Benefit |
|------|------|--------------|---------|
| [serviceinfo.go](c:\code\azd-app\cli\src\internal\serviceinfo\serviceinfo.go#L30) | 30-40 | Manual os.Environ() parsing | -10 lines, cleaner |
| [service_operations.go](c:\code\azd-app\cli\src\internal\dashboard\service_operations.go#L590) | 590-600 | Manual os.Environ() parsing | -10 lines, cleaner |

### Files That Should Use pathutil

| File | Line | Current Code | Benefit |
|------|------|--------------|---------|
| [notify_windows.go](c:\code\azd-app\cli\src\internal\notify\notify_windows.go#L216) | 216-232 | Custom exec finding | -15 lines, cross-platform |
| [reqs.go](c:\code\azd-app\cli\src\cmd\app\commands\reqs.go#L110) | 110+ | Already uses pathutil, needs GetInstallSuggestion() | Better UX |

### Files That Should Use env.Get()

Over 20 test files that currently use:
```go
originalVar := os.Getenv("VAR")
// manual default handling
```

Could use:
```go
originalVar := env.Get("VAR", "default")
```

---

## Conclusion

**Major Finding**: azd-app and azd-exec have **successfully integrated most azd-core packages** where they make sense.

**Remaining Opportunities**:
1. **env package**: Clear wins in azd-app for cleaner environment variable handling (2-3 files)
2. **pathutil package**: Already partially integrated in azd-app, needs minor enhancements (1-2 files)
3. **All other packages**: Either already integrated or not applicable to the project's scope

**Key Insight**: The integration story is actually **much better than expected**. Most packages are already in use where they provide value. The remaining opportunities are minor cleanups and UX enhancements, not major refactoring.
