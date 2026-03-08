# Spec: Azure Developer CLI Extension Framework Enhancements Adoption

## Summary

This spec identifies recent enhancements to the Azure Developer CLI (azd) extension framework and proposes their adoption across azd-exec, azd-app, and azd-core projects to improve observability, debugging, reliability, and developer experience.

## Background

The azure-dev project has introduced several significant enhancements to the extension framework over the past 3-6 months:

1. **Distributed Tracing** (Dec 2025) - OpenTelemetry integration with W3C Trace Context
2. **Structured Error Handling** (Dec 2025) - Unified error model with origin tracking
3. **Interactive Extension Support** (Dec 2025) - TUI/interactive mode for extensions
4. **ServiceContext in Lifecycle Events** (Oct 2025) - Enhanced event context with build artifacts
5. **Automatic Event Handler Cleanup** (Oct 2025) - Context-based lifecycle management
6. **Duplicate Event Registration Prevention** (Oct 2025) - Improved event dispatcher
7. **WaitForDebugger Export** (Jan 2026) - Public debugger attachment API
8. **Additional Properties Support** (Nov 2025) - Extension-specific config in azure.yaml
9. **Unknown Properties in Schema** (Nov 2025) - azure.yaml schema extensibility

## Motivation

### Current State
- **azd-exec**: Basic extension with command execution, no tracing or structured errors
- **azd-app**: Advanced extension with lifecycle events, service targets, MCP server, but lacks observability
- **azd-core**: Shared utilities with no extension-specific enhancements

### Pain Points
1. **Limited observability** - Cannot trace extension execution or correlate with Azure SDK calls
2. **Generic error handling** - No structured error categorization (local vs service vs tool)
3. **Manual debugging** - No standardized debugger attachment pattern
4. **Missed lifecycle context** - ServiceContext not exposed in event handlers
5. **Memory leaks risk** - Manual event handler cleanup
6. **Config limitations** - Cannot store extension-specific config in azure.yaml

## Goals

1. **Enhance Observability**: Add distributed tracing to azd-exec and azd-app
2. **Improve Error Handling**: Adopt structured error model across all extensions
3. **Better Developer Experience**: Export debugger support, interactive mode
4. **Use Event Context**: Utilize ServiceContext in azd-app lifecycle handlers
5. **Improve Reliability**: Auto cleanup, duplicate prevention
6. **Enable Extensibility**: Support additional properties for extension config

## Non-Goals

- Migrating azd-exec or azd-app to new language (both remain Go)
- Changing existing public APIs (additive changes only)
- Backporting changes to older azd versions
- Creating new extension capabilities beyond framework adoption

## Detailed Design

### 1. Distributed Tracing Integration

#### Overview
Adopt OpenTelemetry-based tracing with W3C Trace Context propagation from azd into extensions.

#### Changes Required

**azd-exec/cli/src/cmd/exec/main.go**
```go
import (
    "github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

func main() {
    // BEFORE:
    // rootCmd := newRootCmd()
    
    // AFTER:
    ctx := azdext.NewContext() // Hydrates trace context from TRACEPARENT env
    rootCmd := newRootCmd()
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**azd-app/cli/src/cmd/app/main.go**
```go
import (
    "github.com/azure/azure-dev/cli/azd/pkg/azdext"
)

func main() {
    // AFTER:
    ctx := azdext.NewContext()
    rootCmd := &cobra.Command{...}
    // Register commands...
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

**azd-app - Azure SDK Correlation**
For commands that make Azure SDK calls (logs, health checks):
```go
import (
    "github.com/azure/azure-dev/cli/azd/pkg/azsdk"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

// In Azure client initialization:
clientOptions := &policy.ClientOptions{
    PerCallPolicies: []policy.Policy{
        azsdk.NewMsCorrelationPolicy(), // Correlates with parent trace
    },
}
```

#### Benefits
- End-to-end trace visibility from azd through extensions to Azure services
- Improved debugging with correlation IDs
- Better understanding of performance bottlenecks
- Foundation for telemetry and monitoring

#### Dependencies
- Requires azd >= 1.22.0 (includes tracing support)
- No additional packages needed (azdext already available)

### 2. Structured Error Handling

#### Overview
Adopt the new `ExtensionError` protobuf message for consistent error categorization and telemetry.

#### Error Model
```protobuf
enum ErrorOrigin {
  ERROR_ORIGIN_UNSPECIFIED = 0;  // Unknown
  ERROR_ORIGIN_LOCAL       = 1;  // Config, filesystem, auth, validation
  ERROR_ORIGIN_SERVICE     = 2;  // HTTP/gRPC upstream services
  ERROR_ORIGIN_TOOL        = 3;  // Subprocess, external tools
}

message ServiceErrorDetail {
  string error_code   = 1;  // e.g., "Conflict", "NotFound"
  int32 status_code   = 2;  // HTTP status (409, 404, 500)
  string service_name = 3;  // e.g., "ai.azure.com"
}

message ExtensionError {
  string message = 2;
  string details = 3;
  ErrorOrigin origin = 4;
  oneof source {
    ServiceErrorDetail service_error = 10;
  }
}
```

#### Changes Required

**azd-core - New Error Package**
Create `c:\code\azd-core\errors\errors.go`:
```go
package errors

type ErrorOrigin int

const (
    ErrorOriginUnspecified ErrorOrigin = iota
    ErrorOriginLocal
    ErrorOriginService
    ErrorOriginTool
)

type ServiceErrorDetail struct {
    ErrorCode   string
    StatusCode  int
    ServiceName string
}

type ExtensionError struct {
    Message      string
    Details      string
    Origin       ErrorOrigin
    ServiceError *ServiceErrorDetail
}

func (e *ExtensionError) Error() string {
    if e.Details != "" {
        return fmt.Sprintf("%s: %s", e.Message, e.Details)
    }
    return e.Message
}

// Factory functions
func NewLocalError(message, details string) *ExtensionError {
    return &ExtensionError{
        Message: message,
        Details: details,
        Origin:  ErrorOriginLocal,
    }
}

func NewServiceError(message, details string, code string, status int, service string) *ExtensionError {
    return &ExtensionError{
        Message: message,
        Details: details,
        Origin:  ErrorOriginService,
        ServiceError: &ServiceErrorDetail{
            ErrorCode:   code,
            StatusCode:  status,
            ServiceName: service,
        },
    }
}

func NewToolError(message, details string) *ExtensionError {
    return &ExtensionError{
        Message: message,
        Details: details,
        Origin:  ErrorOriginTool,
    }
}
```

**azd-exec - Adopt Structured Errors**
```go
import "github.com/jongio/azd-core/errors"

// In executor.Execute():
if _, err := os.Stat(scriptPath); err != nil {
    return errors.NewLocalError(
        fmt.Sprintf("script not found: %s", scriptPath),
        err.Error(),
    )
}

// For shell execution failures:
if exitErr, ok := err.(*exec.ExitError); ok {
    return errors.NewToolError(
        fmt.Sprintf("script exited with code %d", exitErr.ExitCode()),
        string(exitErr.Stderr),
    )
}
```

**azd-app - Adopt Structured Errors**
```go
import "github.com/jongio/azd-core/errors"

// In logs command when Azure SDK fails:
if err != nil {
    if respErr, ok := err.(*azcore.ResponseError); ok {
        return errors.NewServiceError(
            "failed to query Azure logs",
            respErr.Error(),
            respErr.ErrorCode,
            respErr.StatusCode,
            "logs.azure.com",
        )
    }
}
```

#### Benefits
- Consistent error categorization across all extensions
- Better telemetry and error analytics
- Improved error messages for users
- Foundation for retry logic based on error origin

### 3. WaitForDebugger Export

#### Overview
Use the newly exported `WaitForDebugger` function for standardized debugger attachment.

#### Changes Required

**azd-exec/cli/src/cmd/exec/commands/listen.go**
```go
import "github.com/azure/azure-dev/cli/azd/pkg/azdext"

func newListenCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "listen",
        Short: "Listen for azd extension events (internal)",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            
            azdClient, err := azdext.NewAzdClient()
            if err != nil {
                return err
            }
            
            // ADDED: Enable debugger attachment via AZD_EXT_DEBUG=1
            azdext.WaitForDebugger(ctx, azdClient)
            
            // Extension host initialization...
        },
    }
}
```

**azd-app/cli/src/cmd/app/commands/listen.go** - Same pattern

#### Benefits
- Consistent debugging experience across all extensions
- Standardized via environment variable (`AZD_EXT_DEBUG=1`)
- User-friendly prompt with PID for attaching debugger
- No custom implementation needed

### 4. ServiceContext in Lifecycle Events

#### Overview
Update azd-app lifecycle event handlers to utilize the new ServiceContext which contains artifacts from all lifecycle phases.

#### Current vs Enhanced

**BEFORE:**
```go
WithServiceEventHandler("prepackage", func(ctx context.Context, args *azdext.ServiceEventArgs) error {
    // args.Service.Name available
    // args.Project available
    // No access to build artifacts, restore info, etc.
}, nil)
```

**AFTER:**
```go
WithServiceEventHandler("postpackage", func(ctx context.Context, args *azdext.ServiceEventArgs) error {
    // NEW: Access to ServiceContext
    fmt.Printf("Service: %s\n", args.Service.Name)
    fmt.Printf("Package Artifacts: %d\n", len(args.ServiceContext.Package))
    
    // Can inspect build outputs, package details
    for _, artifact := range args.ServiceContext.Package {
        fmt.Printf("  - %s\n", artifact.Path)
    }
    
    return nil
}, nil)
```

#### ServiceContext Structure
```go
type ServiceContext struct {
    Restore *RestoreResult  // From restore phase
    Build   *BuildResult    // From build phase  
    Package *PackageResult  // From package phase
    Deploy  *DeployResult   // From deploy phase
}
```

#### Changes Required

**azd-app/cli/src/cmd/app/commands/listen.go**
Update all service event handlers to use ServiceContext:
```go
.WithServiceEventHandler("postbuild", func(ctx context.Context, args *azdext.ServiceEventArgs) error {
    // Example: Analyze build artifacts
    if args.ServiceContext.Build != nil {
        for _, artifact := range args.ServiceContext.Build.Artifacts {
            fmt.Printf("Built: %s (size: %d)\n", artifact.Path, artifact.Size)
        }
    }
    return nil
}, nil)
```

#### Use Cases for azd-app
1. **Dashboard**: Display real-time build/package progress with artifact counts
2. **Notifications**: Alert on package size changes or build artifact counts
3. **Health**: Validate that expected artifacts were produced
4. **Testing**: Access to build outputs for post-build test execution

#### Benefits
- Richer context in lifecycle events
- Better telemetry and logging
- Enable artifact-aware automation
- Foundation for build caching, validation

### 5. Automatic Event Handler Cleanup

#### Overview
Adopt context-based automatic cleanup to prevent memory leaks and duplicate event registrations.

#### Current Risk
Manual event handler registration without cleanup can lead to:
- Memory leaks in long-running processes
- Duplicate event firing
- Context leaks

#### Changes Required

**azd-app - Use Context Lifecycle**
The framework now automatically removes handlers when context is cancelled:
```go
// In lifecycle event registration (listen.go):
// The context passed to WithServiceEventHandler determines handler lifetime
// When context is cancelled, handlers are auto-removed

func newListenCommand() *cobra.Command {
    return &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context() // Root context
            
            host := azdext.NewExtensionHost(extensionId).
                WithServiceEventHandler("predeploy", handler, nil)
            
            // When this command exits and ctx is cancelled,
            // all registered handlers are automatically cleaned up
            return host.Run(ctx)
        },
    }
}
```

#### Benefits
- No memory leaks from orphaned handlers
- Automatic cleanup on process exit
- Simplified handler management
- Works with existing code (no changes needed, just benefit from framework)

### 6. Additional Properties Support in azure.yaml

#### Overview
Enable extension-specific configuration in `azure.yaml` using the new `additionalProperties` support.

#### Schema Changes
The azure.yaml JSON schema now allows `additionalProperties: true` at project and service levels.

#### Example: azd-app Configuration

**Before** (limited to standard azure.yaml fields):
```yaml
name: my-app
services:
  api:
    project: ./src/api
    host: containerapp
    language: dotnet
```

**After** (with extension config):
```yaml
name: my-app

# Project-level azd-app config
app:
  notifications:
    enabled: true
    channels: ["slack", "email"]
  dashboard:
    port: 3100
  health:
    interval: 30s

services:
  api:
    project: ./src/api
    host: containerapp
    language: dotnet
    
    # Service-level azd-app config
    app:
      test:
        coverage: 80
        framework: "go"
      monitoring:
        alerts:
          - cpu > 80%
          - memory > 90%
```

#### Implementation

**azd-app - Config Reading**
```go
import "github.com/azure/azure-dev/cli/azd/pkg/config"

// In commands that need extension config:
func (c *RunCommand) Execute(ctx context.Context) error {
    // Load project config (from azure.yaml)
    projectConfig, err := loadProjectConfig(ctx)
    if err != nil {
        return err
    }
    
    // Extract app-specific config from AdditionalProperties
    cfg := config.NewConfig(projectConfig.AdditionalProperties)
    
    type AppConfig struct {
        Notifications struct {
            Enabled  bool     `yaml:"enabled"`
            Channels []string `yaml:"channels"`
        } `yaml:"notifications"`
        Dashboard struct {
            Port int `yaml:"port"`
        } `yaml:"dashboard"`
    }
    
    var appConfig AppConfig
    if found, err := cfg.GetSection("app", &appConfig); err != nil {
        return err
    } else if found {
        // Use the config
        if appConfig.Dashboard.Port > 0 {
            dashboardPort = appConfig.Dashboard.Port
        }
    }
}
```

**Service-Level Config**
```go
// For service-specific config:
for serviceName, serviceConfig := range projectConfig.Services {
    serviceCfg := config.NewConfig(serviceConfig.AdditionalProperties)
    
    type ServiceAppConfig struct {
        Test struct {
            Coverage  int    `yaml:"coverage"`
            Framework string `yaml:"framework"`
        } `yaml:"test"`
    }
    
    var svcAppConfig ServiceAppConfig
    if found, err := serviceCfg.GetSection("app", &svcAppConfig); err == nil && found {
        // Use service-specific config
        coverageThreshold = svcAppConfig.Test.Coverage
    }
}
```

#### Benefits
- Extension configuration lives with project config
- Type-safe config extraction
- No separate config files needed
- Validates with azure.yaml schema

### 7. Interactive Extension Support

#### Overview
Enable azd-app dashboard and other interactive features to work properly when launched as an extension.

#### Current State
azd-app dashboard runs as background process, may have issues with interactive azd commands.

#### Enhancement
Use the new interactive extension support for TUI features.

#### Changes Required

**azd-app/cli/src/cmd/app/commands/start.go**
```go
// Mark dashboard start as background/interactive appropriately
func NewStartCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "start",
        Short: "Start the dashboard server",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Dashboard is interactive/long-running
            // Framework handles proper terminal/process management
            return dashboard.Start(cmd.Context(), port)
        },
    }
}
```

#### Benefits
- Proper terminal handling for TUI
- Compatible with azd workflows
- Better process lifecycle management

## Implementation Plan

### Phase 1: Foundation (Week 1)
**Priority: P0**
- [ ] Add structured error package to azd-core
- [ ] Update azd-exec to use `azdext.NewContext()` for tracing
- [ ] Update azd-app to use `azdext.NewContext()` for tracing
- [ ] Add WaitForDebugger to both extensions' listen commands
- [ ] Update go.mod dependencies if needed

**Deliverables:**
- azd-core/errors package with factory functions
- Tracing-enabled main.go in both extensions
- Debugger support in listen commands

### Phase 2: Error Handling (Week 2)
**Priority: P1**
- [ ] Refactor azd-exec error handling to use structured errors
- [ ] Refactor azd-app error handling to use structured errors
- [ ] Add error origin categorization throughout
- [ ] Add tests for error handling

**Deliverables:**
- All errors categorized by origin (local/service/tool)
- ServiceErrorDetail populated for Azure SDK errors
- Improved error messages

### Phase 3: Event Enhancements (Week 2)
**Priority: P1**
- [ ] Update azd-app lifecycle handlers to use ServiceContext
- [ ] Enhance dashboard to show artifact information
- [ ] Enhance notifications with build/package details
- [ ] Add tests for event context usage

**Deliverables:**
- ServiceContext utilized in all lifecycle handlers
- Dashboard shows artifact counts/details
- Richer notification content

### Phase 4: Azure SDK Correlation (Week 3)
**Priority: P2**
- [ ] Add correlation policy to azd-app Azure clients
- [ ] Verify trace correlation in logs command
- [ ] Verify trace correlation in health checks
- [ ] Add trace context to MCP server operations

**Deliverables:**
- End-to-end tracing from azd → app → Azure APIs
- Correlation IDs in all Azure SDK calls
- Trace context in MCP operations

### Phase 5: Config Extensibility (Week 3-4)
**Priority: P2**
- [ ] Design azd-app config schema for azure.yaml
- [ ] Implement config reading in relevant commands
- [ ] Update documentation with config examples
- [ ] Add tests for config extraction

**Deliverables:**
- Support for `app:` section in azure.yaml
- Project and service-level config
- Type-safe config extraction
- Documentation and examples

### Phase 6: Testing & Documentation (Week 4)
**Priority: P1**
- [ ] Integration tests for tracing
- [ ] Integration tests for error handling
- [ ] Update CONTRIBUTING.md with debugging instructions
- [ ] Update README.md with tracing/config features
- [ ] Add examples directory with azure.yaml samples

**Deliverables:**
- Comprehensive test coverage
- User-facing documentation
- Developer documentation
- Example configurations

## Testing Strategy

### Unit Tests
- Error factory functions (azd-core/errors)
- Config extraction logic
- ServiceContext usage

### Integration Tests
- Tracing context propagation (verify TRACEPARENT)
- Error categorization across error types
- Config loading from azure.yaml with AdditionalProperties
- ServiceContext populated in lifecycle events

### Manual Testing
- Debugger attachment with `AZD_EXT_DEBUG=1`
- Trace correlation in Azure Portal
- Dashboard showing artifact counts
- Extension config in azure.yaml

### Regression Testing
- Existing azd-exec functionality unchanged
- Existing azd-app functionality unchanged
- Backward compatibility with older azd versions

## Risks and Mitigations

### Risk 1: azd Version Compatibility
**Impact:** Features require azd >= 1.22.0

**Mitigation:**
- Document minimum azd version in README
- Graceful degradation where possible
- Version check at extension startup

### Risk 2: Breaking Changes in Framework
**Impact:** Future azd updates may change APIs

**Mitigation:**
- Follow azd's semantic versioning
- Pin to specific azd version ranges in go.mod
- Monitor azure-dev changelogs
- Maintain compatibility layer if needed

### Risk 3: Adoption Complexity
**Impact:** Team needs to learn new patterns

**Mitigation:**
- Comprehensive documentation
- Code examples in spec
- Incremental rollout (start with tracing)
- Pair programming sessions

### Risk 4: Performance Overhead
**Impact:** Tracing may add latency

**Mitigation:**
- Tracing is opt-in (enabled via env vars)
- Minimal overhead in production
- Benchmark before/after

### Risk 5: Config Schema Evolution
**Impact:** azure.yaml structure may change

**Mitigation:**
- Use versioned config sections
- Graceful handling of missing config
- Validate config at load time

## Success Metrics

### Observability
- [ ] 100% of extension commands emit trace context
- [ ] All Azure SDK calls correlated with parent trace
- [ ] Trace data visible in Azure Application Insights

### Error Handling
- [ ] 100% of errors categorized by origin
- [ ] ServiceErrorDetail populated for all Azure errors
- [ ] Error messages include actionable details

### Developer Experience
- [ ] Debugger attachment works in <5 seconds
- [ ] Extension config documented with 5+ examples
- [ ] Zero memory leaks from event handlers

### Reliability
- [ ] No duplicate event registrations
- [ ] Clean handler cleanup on context cancellation
- [ ] Zero regressions in existing functionality

## Open Questions

1. **Q:** Should we add tracing to azd-core utilities?  
   **A:** Yes, in Phase 4 - add trace spans for Key Vault operations, file utilities, etc.

2. **Q:** Should azd-exec support additional properties config?  
   **A:** Low priority - azd-exec is simple, config via flags is sufficient. Defer to future.

3. **Q:** How to handle trace data in CI/CD?  
   **A:** Document `AZD_TRACE_LOG_URL` for CI environments, send to OTLP endpoint.

4. **Q:** Should we version the extension config schema?  
   **A:** Yes, use `app.version: 1` in azure.yaml for future-proofing.

5. **Q:** Impact on azd-core consumers outside azd-exec/azd-app?  
   **A:** azd-core errors package is generic, can be used by any Go project. No dependencies on azd.

## Alternatives Considered

### Alternative 1: Custom Tracing Solution
**Pros:** Full control, no azd dependency  
**Cons:** Reinventing the wheel, no correlation with azd traces  
**Decision:** Rejected - use existing framework

### Alternative 2: Separate Config File (app-config.yaml)
**Pros:** Clear separation, easier validation  
**Cons:** Multiple config files, fragmentation, user confusion  
**Decision:** Rejected - use azure.yaml AdditionalProperties

### Alternative 3: Manual Error Tagging
**Pros:** Simple, no new types  
**Cons:** Inconsistent, hard to analyze, poor telemetry  
**Decision:** Rejected - use structured errors

## References

- [azd Extension Framework Docs](https://github.com/Azure/azure-dev/blob/main/cli/azd/docs/extension-framework.md)
- [Distributed Tracing PR #6321](https://github.com/Azure/azure-dev/pull/6321)
- [Additional Properties Support PR #6196](https://github.com/Azure/azure-dev/pull/6196)
- [Automatic Handler Cleanup PR #5960](https://github.com/Azure/azure-dev/pull/5960)
- [ServiceContext in Events PR #6002](https://github.com/Azure/azure-dev/pull/6002)
- [WaitForDebugger Export PR #6433](https://github.com/Azure/azure-dev/pull/6433)

## Appendix A: Code Examples

### Example: Full Tracing Setup in azd-app

```go
// main.go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/azure/azure-dev/cli/azd/pkg/azdext"
    "github.com/jongio/azd-app/cli/src/cmd/app/commands"
)

func main() {
    // Initialize context with trace propagation
    ctx := azdext.NewContext()
    
    rootCmd := &cobra.Command{
        Use:   "app",
        Short: "App - Automate your development environment setup",
    }
    
    // Register commands...
    rootCmd.AddCommand(
        commands.NewRunCommand(),
        commands.NewLogsCommand(),
        // ...
    )
    
    if err := rootCmd.ExecuteContext(ctx); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### Example: Structured Error in azd-exec

```go
// executor/executor.go
package executor

import (
    "github.com/jongio/azd-core/errors"
)

func (e *Executor) Execute(ctx context.Context, scriptPath string) error {
    // Validate script exists (LOCAL error)
    if _, err := os.Stat(scriptPath); err != nil {
        return errors.NewLocalError(
            fmt.Sprintf("script not found: %s", scriptPath),
            err.Error(),
        )
    }
    
    // Execute script
    cmd := exec.CommandContext(ctx, shellPath, scriptPath)
    if err := cmd.Run(); err != nil {
        // Categorize as TOOL error (subprocess failure)
        if exitErr, ok := err.(*exec.ExitError); ok {
            return errors.NewToolError(
                fmt.Sprintf("script failed with exit code %d", exitErr.ExitCode()),
                string(exitErr.Stderr),
            )
        }
        return err
    }
    
    return nil
}
```

### Example: ServiceContext Usage in azd-app

```go
// commands/listen.go
package commands

func newListenCommand() *cobra.Command {
    return &cobra.Command{
        Use: "listen",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()
            azdClient, _ := azdext.NewAzdClient()
            
            // Enable debugger
            azdext.WaitForDebugger(ctx, azdClient)
            
            host := azdext.NewExtensionHost(extensionId).
                WithServiceEventHandler("postbuild", func(ctx context.Context, args *azdext.ServiceEventArgs) error {
                    // Access build artifacts via ServiceContext
                    if args.ServiceContext.Build != nil {
                        fmt.Printf("Built %s: %d artifacts\n",
                            args.Service.Name,
                            len(args.ServiceContext.Build.Artifacts))
                        
                        // Send notification with artifact details
                        notifier.Send(Notification{
                            Type: "build_complete",
                            Service: args.Service.Name,
                            ArtifactCount: len(args.ServiceContext.Build.Artifacts),
                        })
                    }
                    return nil
                }, nil)
            
            return host.Run(ctx)
        },
    }
}
```

### Example: Azure SDK Correlation in azd-app

```go
// internal/logs/client.go
package logs

import (
    "github.com/Azure/azure-sdk-for-go/sdk/azcore"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
    "github.com/azure/azure-dev/cli/azd/pkg/azsdk"
)

func NewLogsClient(credential azcore.TokenCredential) (*azlogs.Client, error) {
    // Add correlation policy for trace propagation
    clientOptions := &policy.ClientOptions{
        PerCallPolicies: []policy.Policy{
            azsdk.NewMsCorrelationPolicy(), // Correlates with parent azd trace
        },
    }
    
    return azlogs.NewClient(credential, clientOptions)
}
```

## Appendix B: Configuration Schema

### azd-app Extension Config in azure.yaml

```yaml
# Project-level config
name: my-application

app:
  version: 1  # Config schema version
  
  # Dashboard configuration
  dashboard:
    enabled: true
    port: 3100
    refresh: 5s
  
  # Notifications
  notifications:
    enabled: true
    channels:
      - type: slack
        webhook: ${SLACK_WEBHOOK}
      - type: email
        recipients: ["team@company.com"]
  
  # Health monitoring
  health:
    enabled: true
    interval: 30s
    endpoints:
      - name: api-health
        url: https://${SERVICE_API_ENDPOINT_URL}/health
        timeout: 10s
  
  # Test defaults
  test:
    coverage: 80
    timeout: 10m

services:
  api:
    project: ./src/api
    host: containerapp
    language: dotnet
    
    # Service-specific app config
    app:
      test:
        coverage: 90  # Override project default
        framework: dotnet
      monitoring:
        metrics: true
        alerts:
          - cpu > 80%
          - memory > 90%
  
  web:
    project: ./src/web
    host: staticwebapp
    language: ts
    
    app:
      test:
        framework: vitest
        coverage: 75
```

---

**Status:** Draft for Review  
**Author:** Manager Agent  
**Date:** 2026-01-10  
**Version:** 1.0  
