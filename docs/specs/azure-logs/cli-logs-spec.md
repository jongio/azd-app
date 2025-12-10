# azd app logs CLI Azure Integration

## Overview

Extend the existing `azd app logs` command to support viewing logs from both local development services and Azure-deployed services. The CLI command should reuse the same Azure log retrieval code that powers the dashboard.

## Problem Statement

Currently:
- `azd app logs` only displays logs from locally running services
- Dashboard can view Azure logs via `/api/azure/logs` endpoint
- MCP tools have a `--source` parameter but it's not wired to Azure log retrieval
- Users must open the dashboard to view Azure logs

## Goals

1. Add `--source` flag to `azd app logs` command: `local`, `azure`, `all`
2. Reuse existing Azure log infrastructure (`AzureLogBuffer`, `LogAnalyticsClient`)
3. Support all existing filtering options (`--service`, `--level`, `--since`, `--tail`, etc.)
4. Support `--follow` mode for Azure logs (polling-based)
5. Provide clear error messages when Azure is not configured

## Non-Goals

- Real-time streaming for Azure (use polling interval, ~30s)
- KQL query builder in CLI (use config or dashboard)
- Azure-specific filtering beyond what local logs support

## Architecture

### Code Reuse Strategy

The existing infrastructure in `cli/src/internal/` provides:

1. **Azure Log Buffer** (`service/azure_log_buffer.go`)
   - `AzureLogBuffer` - manages mode, polling, buffering
   - `Initialize()` - sets up credentials, resource discovery
   - `GetRecentLogs()` - returns buffered logs
   - `Subscribe()` - live log subscription

2. **Azure SDK Clients** (`azure/`)
   - `LogAnalyticsClient` - KQL queries to Log Analytics
   - `ResourceDiscovery` - finds Azure resources from azd env
   - `NewAzureCredential()` - auth chain (azd token → DefaultAzureCredential)

3. **Dashboard API** (`dashboard/azure_logs.go`)
   - `/api/azure/logs` - HTTP wrapper around `AzureLogBuffer`
   - `/api/azure/status` - connection status

### New Components

```
cli/src/cmd/app/commands/logs.go
├── logsOptions.source string        // New: "local", "azure", "all"
├── logsExecutor.azureLogBuffer      // New: optional Azure buffer
└── collectLogs()                    // Modified: include Azure source
```

## Implementation

### Phase 1: Add --source Flag

Add new flag to `logsOptions`:

```go
type logsOptions struct {
    // ... existing fields
    source string // "local", "azure", "all" - default "local"
}
```

Command registration:
```go
cmd.Flags().StringVar(&opts.source, "source", "local", 
    "Log source: 'local' (default), 'azure', or 'all'")
```

### Phase 2: Azure Log Collection

Create helper function in logs.go:

```go
func (e *logsExecutor) collectAzureLogs(ctx context.Context, cwd string, 
    targetServices []string, sinceTime time.Time) ([]service.LogEntry, error) {
    
    // 1. Load azure.yaml to get Azure config
    azureYaml, err := service.ParseAzureYaml(cwd)
    if err != nil || azureYaml.Logs == nil || 
       azureYaml.Logs.Azure == nil || !azureYaml.Logs.Azure.Enabled {
        return nil, fmt.Errorf("Azure logging not configured. Add logs.azure.enabled: true to azure.yaml")
    }
    
    // 2. Try dashboard first (if running)
    logs, err := e.collectAzureLogsViaDashboard(ctx, cwd, targetServices, sinceTime)
    if err == nil {
        return logs, nil
    }
    
    // 3. Fall back to direct Azure query
    return e.collectAzureLogsDirect(ctx, cwd, azureYaml, targetServices, sinceTime)
}
```

### Phase 3: Dashboard-First Approach

When dashboard is running, leverage its Azure connection:

```go
func (e *logsExecutor) collectAzureLogsViaDashboard(ctx context.Context, 
    cwd string, targetServices []string, sinceTime time.Time) ([]service.LogEntry, error) {
    
    client, err := e.dashboardClientFactory(ctx, cwd)
    if err != nil {
        return nil, err
    }
    
    if err := client.Ping(ctx); err != nil {
        return nil, err
    }
    
    // Use existing /api/azure/logs endpoint
    return client.GetAzureLogs(ctx, targetServices, e.opts.tail, sinceTime)
}
```

Add to `DashboardClient` interface:
```go
type DashboardClient interface {
    Ping(ctx context.Context) error
    GetServices(ctx context.Context) ([]*serviceinfo.ServiceInfo, error)
    StreamLogs(ctx context.Context, serviceName string, logs chan<- service.LogEntry) error
    GetAzureLogs(ctx context.Context, services []string, tail int, since time.Time) ([]service.LogEntry, error) // NEW
}
```

### Phase 4: Direct Azure Query

When dashboard not running, query Azure directly:

```go
func (e *logsExecutor) collectAzureLogsDirect(ctx context.Context, 
    cwd string, azureYaml *service.AzureYaml, 
    targetServices []string, sinceTime time.Time) ([]service.LogEntry, error) {
    
    // Create temporary Azure log buffer (not persisted)
    azBuffer := service.NewAzureLogBuffer(azureYaml.Logs.Azure, cwd)
    
    // Get Azure context from azd env
    envValues, err := getAzdEnvValues(cwd)
    if err != nil {
        return nil, fmt.Errorf("failed to get Azure environment: %w\nRun 'azd provision' first")
    }
    
    // Initialize (sets up credentials, discovers resources)
    if err := azBuffer.Initialize(ctx, 
        envValues.SubscriptionID, 
        envValues.ResourceGroup,
        envValues.EnvironmentName); err != nil {
        return nil, fmt.Errorf("Azure initialization failed: %w", err)
    }
    defer azBuffer.Close()
    
    // Fetch logs (one-time query, not polling)
    var allLogs []service.LogEntry
    for _, svc := range targetServices {
        logs := azBuffer.GetRecentLogs(svc, e.opts.tail)
        allLogs = append(allLogs, logs...)
    }
    
    // Filter by time if specified
    if !sinceTime.IsZero() {
        allLogs = filterLogsByTime(allLogs, sinceTime)
    }
    
    return allLogs, nil
}
```

### Phase 5: Follow Mode for Azure

For `--follow` with Azure source:

```go
func (e *logsExecutor) followAzureLogs(ctx context.Context, cwd string,
    serviceFilter []string, levelFilter service.LogLevel, 
    logFilter *service.LogFilter, outputWriter io.Writer) error {
    
    // Create and initialize Azure buffer
    azBuffer := service.NewAzureLogBuffer(azureYaml.Logs.Azure, cwd)
    // ... initialize ...
    
    // Switch to Azure mode to start polling
    if err := azBuffer.SetMode(service.LogModeAzure); err != nil {
        return err
    }
    
    // Subscribe to buffered logs
    subscription := azBuffer.Subscribe()
    defer azBuffer.Unsubscribe(subscription)
    
    output.Info("Streaming Azure logs (polling every %s)...", 
        azureYaml.Logs.Azure.PollingInterval)
    
    // Similar to followLogsInMemory but for Azure
    for {
        select {
        case entry := <-subscription:
            if !e.shouldDisplayEntry(entry, levelFilter, logFilter) {
                continue
            }
            displayLogEntry(entry, outputWriter, e.opts)
        case <-sigChan:
            return nil
        }
    }
}
```

### Phase 6: Merged View (--source all)

When `--source all`, interleave local and Azure logs:

```go
func (e *logsExecutor) collectAllLogs(ctx context.Context, ...) ([]service.LogEntry, error) {
    var allLogs []service.LogEntry
    
    // Collect local logs
    localLogs, _ := e.collectLogs(ctx, cwd, targetServices, logManager, sinceTime)
    allLogs = append(allLogs, localLogs...)
    
    // Collect Azure logs (non-fatal if not configured)
    azureLogs, err := e.collectAzureLogs(ctx, cwd, targetServices, sinceTime)
    if err != nil {
        output.Warn("Azure logs unavailable: %s", err)
    } else {
        allLogs = append(allLogs, azureLogs...)
    }
    
    // Sort by timestamp for proper interleaving
    service.SortLogEntries(allLogs)
    return allLogs, nil
}
```

## CLI Examples

```bash
# View local logs (default, unchanged behavior)
azd app logs

# View Azure logs
azd app logs --source azure

# View both local and Azure logs
azd app logs --source all

# Filter Azure logs by service
azd app logs --source azure --service api

# Follow Azure logs
azd app logs --source azure --follow

# Azure logs with level filter
azd app logs --source azure --level error

# Azure logs from last hour
azd app logs --source azure --since 1h

# Export Azure logs to file
azd app logs --source azure --file azure-logs.txt

# JSON output for processing
azd app logs --source azure --format json
```

## Error Handling

### Azure Not Configured
```
Error: Azure logging not configured.
To enable Azure log viewing:
  1. Add to azure.yaml:
     logs:
       azure:
         enabled: true
  2. Run 'azd provision' to deploy resources

For more info: https://aka.ms/azd-app/azure-logs
```

### Not Deployed
```
Error: No Azure resources found.
Run 'azd provision' or 'azd deploy' to deploy your services.
```

### Authentication Failed
```
Error: Azure authentication failed.
Run 'az login' or 'azd auth login' to authenticate.
```

### No Log Analytics Workspace
```
Warning: No Log Analytics workspace configured.
Azure logs require a Log Analytics workspace. Configure one in azure.yaml:
  logs:
    azure:
      workspace: <workspace-guid>

Or enable diagnostic settings on your Azure resources.
```

## Testing

### Unit Tests
- `logs_azure_test.go` - test Azure log collection
- Mock `AzureLogBuffer` for isolated testing
- Test source flag validation
- Test log merging for `--source all`

### Integration Tests
- Mock Azure API responses
- Test dashboard fallback behavior
- Test credential chain

### Manual Testing
```bash
# Test with azure-logs-test project
cd cli/tests/projects/integration/azure-logs-test
azd provision  # Deploy test resources
azd app logs --source azure
```

## Tasks

### TODO: Add --source flag to logs command {#add-source-flag}
**Assigned**: Developer
Add `--source` flag with validation (local|azure|all)

### TODO: Implement Azure log collection via dashboard {#azure-dashboard-collection}
**Assigned**: Developer
Add `GetAzureLogs` to DashboardClient, implement collection

### TODO: Implement direct Azure log collection {#azure-direct-collection}
**Assigned**: Developer
Create `collectAzureLogsDirect` using AzureLogBuffer

### TODO: Implement Azure follow mode {#azure-follow-mode}
**Assigned**: Developer
Add `followAzureLogs` with polling-based streaming

### TODO: Implement merged view {#merged-view}
**Assigned**: Developer
Implement `--source all` with proper interleaving

### TODO: Add error handling and user guidance {#error-handling}
**Assigned**: Developer
Clear error messages with actionable guidance

### TODO: Write unit tests {#cli-logs-tests}
**Assigned**: Tester
Unit tests for Azure log collection paths

## File Changes

| File | Change |
|------|--------|
| `cli/src/cmd/app/commands/logs.go` | Add `--source` flag, Azure collection methods |
| `cli/src/internal/dashboard/client.go` | Add `GetAzureLogs` method |
| `cli/src/cmd/app/commands/logs_azure_test.go` | New test file |

## Dependencies

- Existing `service.AzureLogBuffer` (no changes needed)
- Existing `azure.LogAnalyticsClient` (no changes needed)
- Existing `dashboard.handleAzureLogs` (no changes needed)
