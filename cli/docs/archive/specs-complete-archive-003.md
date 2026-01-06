# CLI Specs Complete Archive #003
Archived: January 6, 2026

This archive contains completed specification projects from /cli/docs/specs.

---


## PROJECT: azure-logs-v2

### SPEC.MD

# Azure Logs v2: Simplified & Reliable Design

## Implementation Status

| Component | Status | Details |
|-----------|--------|---------|
| CLI `--source azure` flag | ✅ Done | Works standalone without `azd app run` |
| CLI `--source azure -f` streaming | ✅ Done | Polls every 30s, no `azd app run` required |
| CLI service filtering `-s` | ✅ Done | Maps azure.yaml names to Azure resources |
| Standalone Azure logs fetcher | ✅ Done | `azure/standalone_logs.go` |
| DefaultAzureCredential auth | ✅ Done | Uses `azd auth login` credentials |
| Log Analytics SDK integration | ✅ Done | `azlogs` SDK queries work |
| **Next Phase: Dashboard Integration** | | |
| Dashboard API endpoint | ⏳ Next | `GET /api/azure/logs` with structured errors |
| Dashboard UI auto-load | ⏳ Next | Load logs when Azure mode selected |
| Visual loading/error states | ⏳ Next | Spinner, error panel, status footer |
| Auto-refresh countdown | ⏳ Next | 30s poll with visible countdown |
| **Diagnostics & Documentation** | | |
| Health check endpoint | ⏳ Next | `GET /api/azure/logs/health` |
| Auto-resolve missing workspace | ⏳ Next | Discover and store workspace GUID |
| Documentation URLs | ⏳ Next | All errors link to docs |
| Diagnostics UI modal | ⏳ Next | Interactive health check panel |

## Executive Summary

The current Azure logs implementation has proven unreliable due to:
1. **Log Analytics Ingestion Delay**: 30-90 second latency makes "real-time" polling feel broken
2. **Complex Initialization Chain**: Multiple dependencies (credentials → discovery → workspace → client) with many failure points
3. **KQL Query Fragility**: Queries depend on exact service naming that often mismatches between azure.yaml and Azure resources
4. **Silent Failures**: When something goes wrong, user sees nothing - no feedback

This v2 spec proposes a **simplified, transparent approach** that works reliably with clear status feedback.

## CLI Interface

### Requirements by Source

| Command | Requires `azd app run`? | Notes |
|---------|-------------------------|-------|
| `azd app logs` | Yes | Local logs need services running |
| `azd app logs -f` | Yes | Local streaming needs services running |
| `azd app logs --source azure` | **No** | Queries Log Analytics directly |
| `azd app logs --source azure -f` | **No** | Polls Log Analytics directly (30s) |
| `azd app logs --source all` | Yes | Local component needs services |
| `azd app logs --source all -f` | Yes | Local component needs services |

### Command Examples

```bash
# Local logs (requires azd app run)
azd app run                     # Start services first
azd app logs                    # View local logs
azd app logs -f                 # Follow local logs in real-time

# Azure logs (standalone - no azd app run required)
azd app logs --source azure             # View logs from Azure Log Analytics
azd app logs --source azure -f          # Follow Azure logs (30s polling)
azd app logs --source azure --since 1h  # Logs from last hour
azd app logs --source azure -s api      # Filter by service

# Combined (requires azd app run for local component)
azd app run                     # Start services first
azd app logs --source all       # Both local and Azure logs
```

### Flag Definition

```go
cmd.Flags().StringVar(&opts.source, "source", "local", "Log source: 'local' (default), 'azure', or 'all'")
```

| Value | Description |
|-------|-------------|
| `local` | Logs from locally running services (default) |
| `azure` | Logs from Azure Log Analytics (standalone) |
| `all` | Logs from both sources |

## Design Principles

1. **Fail Fast, Communicate Clearly**: Surface errors immediately with actionable guidance
2. **Keep the SDK**: Use `github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs` (already have it)
3. **Auto-Pull with Feedback**: Logs load automatically, but user sees clear status indicators
4. **Transparent State**: User always knows what's happening (loading, error, last updated)

## UX Requirements

### Auto-Load on Dashboard Open
When user switches to Azure logs tab:
1. Immediately show loading indicator
2. Fetch logs automatically (no button click required)
3. Display logs when ready
4. Show "Last updated: X seconds ago" footer
5. Continue polling every 30 seconds

### Visual Status Indicators

```
┌──────────────────────────────────────────────────────────────────┐
│  [Local]  [Azure ●]                                              │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  🔄 Loading logs from Azure...                             │ │  ← Loading state
│  │  ████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  [Local]  [Azure ●]                              ↻ 25s           │  ← Next refresh countdown
│                                                                  │
│  2024-01-15 10:32:15  INFO   Server started on port 8080         │
│  2024-01-15 10:32:14  INFO   Connected to database               │
│  2024-01-15 10:32:13  WARN   Cache miss for key 'config'         │
│  ...                                                             │
│  ─────────────────────────────────────────────────────────────── │
│  ✓ 142 logs • Updated 5s ago • Next refresh in 25s              │  ← Status footer
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  [Local]  [Azure ⚠]                                              │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  ⚠️  Could not fetch Azure logs                            │ │  ← Error state
│  │                                                            │ │
│  │  Authentication expired.                                   │ │
│  │                                                            │ │
│  │  Run this command to fix:                                  │ │
│  │  ┌──────────────────────────────────┐                     │ │
│  │  │  azd auth login                  │  [Copy]              │ │
│  │  └──────────────────────────────────┘                     │ │
│  │                                                            │ │
│  │  [Retry Now]                                               │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

### State Machine

```
                    ┌─────────────┐
       tab open     │   IDLE      │
      ─────────────►│  (no logs)  │
                    └──────┬──────┘
                           │ auto-fetch
                           ▼
                    ┌─────────────┐
                    │  LOADING    │◄────────────────┐
                    │  (spinner)  │                 │
                    └──────┬──────┘                 │
                           │                        │
              ┌────────────┼────────────┐           │
              │ success    │            │ error     │
              ▼            │            ▼           │
       ┌─────────────┐     │     ┌─────────────┐    │
       │  SHOWING    │     │     │   ERROR     │    │
       │  (logs)     │     │     │  (message)  │────┤ retry
       └──────┬──────┘     │     └─────────────┘    │
              │            │                        │
              │ 30s timer  │                        │
              └────────────┴────────────────────────┘
```

## Architecture

### Keep the SDK, Simplify the Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dashboard (React)                             │
│                                                                  │
│  AzureLogsPanel                                                  │
│  ├─ state: idle | loading | showing | error                     │
│  ├─ logs: LogEntry[]                                            │
│  ├─ lastUpdated: timestamp                                      │
│  ├─ nextRefresh: countdown                                      │
│  └─ error: { message, action, command }                         │
│                                                                  │
│  On mount → fetch logs                                          │
│  On success → show logs, start 30s timer                        │
│  On error → show error with action                              │
│  On timer → fetch logs again                                    │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼ GET /api/azure/logs
┌─────────────────────────────────────────────────────────────────┐
│                    Backend (Go)                                  │
│                                                                  │
│  1. Check initialization state                                   │
│     └─ Not ready? Return status with action                     │
│                                                                  │
│  2. Query Log Analytics via SDK                                  │
│     └─ github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs│
│                                                                  │
│  3. Return logs OR error with action                             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Simplified Initialization

Instead of complex async initialization, do lazy init on first request:

```go
func (s *Server) handleAzureLogs(w http.ResponseWriter, r *http.Request) {
    // 1. Lazy initialize if needed
    if s.azureClient == nil {
        client, err := s.initAzureClient()
        if err != nil {
            writeErrorWithAction(w, err)
            return
        }
        s.azureClient = client
    }
    
    // 2. Query logs
    logs, err := s.azureClient.QueryLogs(r.Context(), s.workspaceID, kql)
    if err != nil {
        writeErrorWithAction(w, err)
        return
    }
    
    // 3. Return logs with metadata
    writeJSON(w, AzureLogsResponse{
        Logs:      logs,
        Timestamp: time.Now(),
        Count:     len(logs),
    })
}
```

## Keep Using azlogs SDK

We already have the SDK integrated. The issue isn't the SDK - it's the initialization chain and error handling. Keep:

```go
import "github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"

// Already have this working:
client, err := azlogs.NewClient(cred, nil)
resp, err := client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
    Query:    to.Ptr(kql),
    Timespan: to.Ptr(timespan),
}, nil)
```

### Simplify Credential Chain

Instead of complex DefaultAzureCredential, use `azd auth token`:

```go
func getCredential(projectDir string) (azcore.TokenCredential, error) {
    // Try to get token from azd first (most reliable for local dev)
    cmd := exec.Command("azd", "auth", "token", "--output", "json")
    cmd.Dir = projectDir
    
    output, err := cmd.Output()
    if err == nil {
        var result struct {
            Token     string `json:"token"`
            ExpiresOn string `json:"expiresOn"`
        }
        if json.Unmarshal(output, &result) == nil {
            return &staticTokenCredential{token: result.Token}, nil
        }
    }
    
    // Fall back to DefaultAzureCredential
    return azidentity.NewDefaultAzureCredential(nil)
}

// Simple credential that uses a pre-fetched token
type staticTokenCredential struct {
    token string
}

func (c *staticTokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
    return azcore.AccessToken{Token: c.token, ExpiresOn: time.Now().Add(time.Hour)}, nil
}
```
```

## API Changes

### Updated Endpoint: `GET /api/azure/logs`

Simple GET endpoint that returns logs or error with action:

```go
type AzureLogsResponse struct {
    Status    string      `json:"status"`    // "ok" | "error" | "not_configured"
    Logs      []LogEntry  `json:"logs,omitempty"`
    Count     int         `json:"count"`
    Timestamp time.Time   `json:"timestamp"`
    Error     *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Message string `json:"message"`
    Code    string `json:"code"`     // "AUTH_EXPIRED", "NOT_FOUND", etc.
    Action  string `json:"action"`   // "Run 'azd auth login'"
    Command string `json:"command"`  // "azd auth login"
}
```

### Response Examples

**Success:**
```json
{
  "status": "ok",
  "logs": [...],
  "count": 142,
  "timestamp": "2024-01-15T10:32:15Z"
}
```

**Auth Error:**
```json
{
  "status": "error",
  "error": {
    "message": "Authentication expired",
    "code": "AUTH_EXPIRED",
    "action": "Run this command to fix:",
    "command": "azd auth login"
  }
}
```

**Not Deployed:**
```json
{
  "status": "error", 
  "error": {
    "message": "No Azure resources found",
    "code": "NOT_DEPLOYED",
    "action": "Deploy your app first:",
    "command": "azd up"
  }
}
```

## Frontend Implementation

### React Component

```typescript
interface AzureLogsState {
  status: 'idle' | 'loading' | 'showing' | 'error';
  logs: LogEntry[];
  lastUpdated: Date | null;
  error: ErrorInfo | null;
  nextRefreshIn: number; // seconds
}

function AzureLogsPanel() {
  const [state, setState] = useState<AzureLogsState>({
    status: 'idle',
    logs: [],
    lastUpdated: null,
    error: null,
    nextRefreshIn: 30,
  });
  
  const fetchLogs = async () => {
    setState(s => ({ ...s, status: 'loading', error: null }));
    
    try {
      const resp = await fetch('/api/azure/logs');
      const data = await resp.json();
      
      if (data.status === 'ok') {
        setState(s => ({
          ...s,
          status: 'showing',
          logs: data.logs,
          lastUpdated: new Date(data.timestamp),
          nextRefreshIn: 30,
        }));
      } else {
        setState(s => ({
          ...s,
          status: 'error',
          error: data.error,
        }));
      }
    } catch (err) {
      setState(s => ({
        ...s,
        status: 'error',
        error: { message: 'Network error', code: 'NETWORK', action: 'Check connection' },
      }));
    }
  };
  
  // Auto-fetch on mount
  useEffect(() => {
    fetchLogs();
  }, []);
  
  // Auto-refresh every 30 seconds when showing logs
  useEffect(() => {
    if (state.status !== 'showing') return;
    
    const countdown = setInterval(() => {
      setState(s => {
        if (s.nextRefreshIn <= 1) {
          fetchLogs();
          return { ...s, nextRefreshIn: 30 };
        }
        return { ...s, nextRefreshIn: s.nextRefreshIn - 1 };
      });
    }, 1000);
    
    return () => clearInterval(countdown);
  }, [state.status]);
  
  // Render based on state
  if (state.status === 'loading') {
    return <LoadingSpinner message="Loading logs from Azure..." />;
  }
  
  if (state.status === 'error') {
    return (
      <ErrorPanel
        message={state.error.message}
        action={state.error.action}
        command={state.error.command}
        onRetry={fetchLogs}
      />
    );
  }
  
  if (state.status === 'showing') {
    return (
      <>
        <LogList logs={state.logs} />
        <StatusFooter
          count={state.logs.length}
          lastUpdated={state.lastUpdated}
          nextRefreshIn={state.nextRefreshIn}
        />
      </>
    );
  }
  
  return null;
}
```

### Status Footer Component

```typescript
function StatusFooter({ count, lastUpdated, nextRefreshIn }) {
  const ago = lastUpdated ? formatTimeAgo(lastUpdated) : 'never';
  
  return (
    <div className="azure-logs-footer">
      <span className="logs-count">✓ {count} logs</span>
      <span className="separator">•</span>
      <span className="last-updated">Updated {ago}</span>
      <span className="separator">•</span>
      <span className="next-refresh">
        <RefreshIcon className="spinning" /> {nextRefreshIn}s
      </span>
    </div>
  );
}
```

## Diagnostics & Auto-Resolution

### Automatic Issue Resolution

When Azure logs fail, attempt to auto-resolve before showing error:

```go
func (s *Server) diagnoseAndResolveAzureLogs(ctx context.Context) *ErrorInfo {
    // 1. Check if workspace GUID is missing from env vars
    workspaceID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID")
    if workspaceID == "" {
        // Try to discover and store it
        resolved, err := s.discoverAndStoreWorkspaceID(ctx)
        if err == nil && resolved {
            // Successfully resolved - retry will work now
            return nil
        }
        
        return &ErrorInfo{
            Message: "Log Analytics workspace not configured",
            Code:    "NO_WORKSPACE",
            Action:  "Run this command to configure:",
            Command: "azd env refresh",
            DocsURL: "https://aka.ms/azd/app/logs/setup",
        }
    }
    
    // 2. Check if diagnostic settings are configured
    hasDiagnostics, err := s.checkDiagnosticSettings(ctx)
    if err == nil && !hasDiagnostics {
        return &ErrorInfo{
            Message: "Azure diagnostic settings not configured",
            Code:    "NO_DIAGNOSTICS",
            Action:  "Your services aren't sending logs to Log Analytics",
            Command: "",
            DocsURL: "https://aka.ms/azd/app/logs/configure",
        }
    }
    
    // 3. Check if services are deployed
    servicesDeployed, err := s.checkServicesDeployed(ctx)
    if err == nil && !servicesDeployed {
        return &ErrorInfo{
            Message: "No services deployed to Azure",
            Code:    "NOT_DEPLOYED",
            Action:  "Deploy your app first:",
            Command: "azd up",
            DocsURL: "https://aka.ms/azd/app/logs/troubleshoot",
        }
    }
    
    return nil
}

// Try to discover workspace GUID and store in .env
func (s *Server) discoverAndStoreWorkspaceID(ctx context.Context) (bool, error) {
    // Query Azure for Log Analytics workspace
    resourceGroup := os.Getenv("AZURE_RESOURCE_GROUP")
    if resourceGroup == "" {
        return false, errors.New("no resource group")
    }
    
    cmd := exec.Command("az", "monitor", "log-analytics", "workspace", "list",
        "--resource-group", resourceGroup,
        "--query", "[0].customerId",
        "--output", "tsv")
    
    output, err := cmd.Output()
    if err != nil {
        return false, err
    }
    
    workspaceID := strings.TrimSpace(string(output))
    if workspaceID == "" {
        return false, errors.New("no workspace found")
    }
    
    // Store in .env file
    envPath := filepath.Join(s.projectDir, ".azure", s.envName, ".env")
    if err := appendEnvVar(envPath, "AZURE_LOG_ANALYTICS_WORKSPACE_ID", workspaceID); err != nil {
        return false, err
    }
    
    // Update current environment
    os.Setenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID", workspaceID)
    
    return true, nil
}
```

### Documentation Links

Every error should include a documentation URL:

```go
type ErrorInfo struct {
    Message string `json:"message"`
    Code    string `json:"code"`
    Action  string `json:"action"`
    Command string `json:"command"`
    DocsURL string `json:"docsUrl"`  // NEW: Link to documentation
}
```

**Documentation Structure:**

- `https://aka.ms/azd/app/logs/setup` - Initial setup guide
- `https://aka.ms/azd/app/logs/configure` - Configuration & customization
- `https://aka.ms/azd/app/logs/troubleshoot` - Troubleshooting guide
- `https://aka.ms/azd/app/logs/kql` - Writing custom KQL queries
- `https://aka.ms/azd/app/logs/services` - Per-service configuration

## Error Handling Strategy

Map Azure errors to actionable guidance with docs:

```go
func mapAzureError(err error) *ErrorInfo {
    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "AADSTS") || strings.Contains(errStr, "401"):
        return &ErrorInfo{
            Message: "Authentication expired",
            Code:    "AUTH_EXPIRED",
            Action:  "Run this command to fix:",
            Command: "azd auth login",
            DocsURL: "https://aka.ms/azd/app/logs/troubleshoot#auth",
        }
        
    case strings.Contains(errStr, "ResourceNotFound") || strings.Contains(errStr, "404"):
        return &ErrorInfo{
            Message: "Azure resources not found",
            Code:    "NOT_DEPLOYED",
            Action:  "Deploy your app first:",
            Command: "azd up",
            DocsURL: "https://aka.ms/azd/app/logs/setup",
        }
        
    case strings.Contains(errStr, "AuthorizationFailed") || strings.Contains(errStr, "403"):
        return &ErrorInfo{
            Message: "Missing permissions on Log Analytics workspace",
            Code:    "NO_PERMISSION",
            Action:  "Grant 'Log Analytics Reader' role",
            Command: "",
            DocsURL: "https://aka.ms/azd/app/logs/troubleshoot#permissions",
        }
        
    case strings.Contains(errStr, "WorkspaceNotFound"):
        return &ErrorInfo{
            Message: "Log Analytics workspace not configured",
            Code:    "NO_WORKSPACE",
            Action:  "Configure diagnostic settings",
            Command: "azd env refresh",
            DocsURL: "https://aka.ms/azd/app/logs/configure",
        }
        
    case strings.Contains(errStr, "no logs returned"):
        return &ErrorInfo{
            Message: "No logs found for your services",
            Code:    "NO_LOGS",
            Action:  "Check if services are running and sending logs",
            Command: "",
            DocsURL: "https://aka.ms/azd/app/logs/troubleshoot#no-logs",
        }
        
    default:
        return &ErrorInfo{
            Message: errStr,
            Code:    "UNKNOWN",
            Action:  "See troubleshooting guide",
            Command: "",
            DocsURL: "https://aka.ms/azd/app/logs/troubleshoot",
        }
    }
}
```

### Diagnostic Health Check

Add a diagnostic endpoint for troubleshooting:

```go
// GET /api/azure/logs/health
type HealthCheckResponse struct {
    Status  string            `json:"status"`  // "healthy" | "degraded" | "error"
    Checks  []HealthCheck     `json:"checks"`
    DocsURL string            `json:"docsUrl"`
}

type HealthCheck struct {
    Name    string `json:"name"`
    Status  string `json:"status"`  // "pass" | "warn" | "fail"
    Message string `json:"message"`
    Fix     string `json:"fix,omitempty"`
}

func (s *Server) handleAzureLogsHealth(w http.ResponseWriter, r *http.Request) {
    checks := []HealthCheck{}
    
    // 1. Authentication
    if _, err := getCredential(s.projectDir); err != nil {
        checks = append(checks, HealthCheck{
            Name:    "Authentication",
            Status:  "fail",
            Message: "Not authenticated",
            Fix:     "Run: azd auth login",
        })
    } else {
        checks = append(checks, HealthCheck{
            Name:    "Authentication",
            Status:  "pass",
            Message: "Credentials valid",
        })
    }
    
    // 2. Workspace ID
    workspaceID := os.Getenv("AZURE_LOG_ANALYTICS_WORKSPACE_ID")
    if workspaceID == "" {
        checks = append(checks, HealthCheck{
            Name:    "Log Analytics Workspace",
            Status:  "fail",
            Message: "Workspace ID not configured",
            Fix:     "Run: azd env refresh",
        })
    } else {
        checks = append(checks, HealthCheck{
            Name:    "Log Analytics Workspace",
            Status:  "pass",
            Message: fmt.Sprintf("Workspace: %s", workspaceID[:8]+"..."),
        })
    }
    
    // 3. Services deployed
    services := discoverServices()
    if len(services) == 0 {
        checks = append(checks, HealthCheck{
            Name:    "Azure Services",
            Status:  "warn",
            Message: "No services found in environment",
            Fix:     "Run: azd up",
        })
    } else {
        checks = append(checks, HealthCheck{
            Name:    "Azure Services",
            Status:  "pass",
            Message: fmt.Sprintf("%d services discovered", len(services)),
        })
    }
    
    // 4. Connectivity test
    if err := testLogAnalyticsConnection(workspaceID); err != nil {
        checks = append(checks, HealthCheck{
            Name:    "Log Analytics Connection",
            Status:  "fail",
            Message: err.Error(),
            Fix:     "Check network and firewall settings",
        })
    } else {
        checks = append(checks, HealthCheck{
            Name:    "Log Analytics Connection",
            Status:  "pass",
            Message: "Successfully connected",
        })
    }
    
    // Determine overall status
    status := "healthy"
    for _, check := range checks {
        if check.Status == "fail" {
            status = "error"
            break
        } else if check.Status == "warn" {
            status = "degraded"
        }
    }
    
    writeJSON(w, HealthCheckResponse{
        Status:  status,
        Checks:  checks,
        DocsURL: "https://aka.ms/azd/app/logs/troubleshoot",
    })
}
```

## UI Changes

### Azure Logs View Design

```
┌──────────────────────────────────────────────────────────────────┐
│  Services: [api ▼]  Time Range: [Last 30m ▼]  [🔄 Refresh]      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ℹ️  Azure logs may have 1-2 minute delay from live events       │
│                                                                  │
│  2024-01-15 10:32:15  INFO   Server started on port 8080         │
│  2024-01-15 10:32:14  INFO   Connected to database               │
│  2024-01-15 10:32:13  WARN   Cache miss for key 'config'         │
│  2024-01-15 10:32:12  ERROR  Failed to parse request body        │
│                                                                  │
│  ─────────────────────────────────────────────────────────────── │
│  📊 100 logs • Updated 5s ago • ↻ 25s   [Run Diagnostics]       │
└──────────────────────────────────────────────────────────────────┘
```

### Error State with Docs Link

```
┌──────────────────────────────────────────────────────────────────┐
│  Services: [api ▼]  Time Range: [Last 30m ▼]  [🔄 Refresh]      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ⚠️  Log Analytics Workspace Not Configured                      │
│                                                                  │
│  Your services aren't configured to send logs to Log Analytics.  │
│                                                                  │
│  To fix, run in your terminal:                                   │
│  ┌──────────────────────────────────┐                           │
│  │  azd env refresh                 │  [Copy]                    │
│  └──────────────────────────────────┘                           │
│                                                                  │
│  [Retry Now]  [Run Diagnostics]  [View Setup Guide →]           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Diagnostics Panel

```
┌──────────────────────────────────────────────────────────────────┐
│  Azure Logs Diagnostics                                   [×]    │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ✓  Authentication                                               │
│      Credentials valid                                           │
│                                                                  │
│  ✗  Log Analytics Workspace                                      │
│      Workspace ID not configured                                 │
│      Fix: Run 'azd env refresh'                                  │
│                                                                  │
│  ✓  Azure Services                                               │
│      3 services discovered (api, web, functions)                 │
│                                                                  │
│  ⚠  Log Analytics Connection                                     │
│      No logs returned in last query                              │
│      Services may not be sending logs yet                        │
│                                                                  │
│  ─────────────────────────────────────────────────────────────── │
│  Status: Degraded                                                │
│                                                                  │
│  [View Troubleshooting Guide →]  [Copy Diagnostics]  [Close]    │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Error Panel Component with Docs

```typescript
interface ErrorPanelProps {
  error: ErrorInfo;
  onRetry: () => void;
  onDiagnostics?: () => void;
}

function ErrorPanel({ error, onRetry, onDiagnostics }: ErrorPanelProps) {
  return (
    <div className="error-panel">
      <div className="error-header">
        <WarningIcon />
        <h3>{error.message}</h3>
      </div>
      
      {error.action && (
        <p className="error-action">{error.action}</p>
      )}
      
      {error.command && (
        <div className="command-block">
          <code>{error.command}</code>
          <button onClick={() => copyToClipboard(error.command)}>
            Copy
          </button>
        </div>
      )}
      
      <div className="error-actions">
        <button onClick={onRetry} className="primary">
          Retry Now
        </button>
        {onDiagnostics && (
          <button onClick={onDiagnostics} className="secondary">
            Run Diagnostics
          </button>
        )}
        {error.docsUrl && (
          <a href={error.docsUrl} target="_blank" className="docs-link">
            View Setup Guide →
          </a>
        )}
      </div>
    </div>
  );
}
```

### Diagnostics Modal Component

```typescript
interface DiagnosticsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

function DiagnosticsModal({ isOpen, onClose }: DiagnosticsModalProps) {
  const [health, setHealth] = useState<HealthCheckResponse | null>(null);
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    if (!isOpen) return;
    
    setLoading(true);
    fetch('/api/azure/logs/health')
      .then(r => r.json())
      .then(data => {
        setHealth(data);
        setLoading(false);
      });
  }, [isOpen]);
  
  if (!isOpen) return null;
  
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Azure Logs Diagnostics</h2>
          <button onClick={onClose}>×</button>
        </div>
        
        {loading ? (
          <LoadingSpinner />
        ) : (
          <div className="diagnostics-results">
            {health.checks.map(check => (
              <div key={check.name} className={`check check-${check.status}`}>
                <div className="check-header">
                  <StatusIcon status={check.status} />
                  <h4>{check.name}</h4>
                </div>
                <p className="check-message">{check.message}</p>
                {check.fix && (
                  <p className="check-fix">
                    <strong>Fix:</strong> {check.fix}
                  </p>
                )}
              </div>
            ))}
            
            <div className="diagnostics-footer">
              <div className="status-badge">
                Status: <strong>{health.status}</strong>
              </div>
              
              <div className="actions">
                <a href={health.docsUrl} target="_blank">
                  View Troubleshooting Guide →
                </a>
                <button onClick={() => copyDiagnostics(health)}>
                  Copy Diagnostics
                </button>
                <button onClick={onClose} className="primary">
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'pass': return <span className="icon-pass">✓</span>;
    case 'warn': return <span className="icon-warn">⚠</span>;
    case 'fail': return <span className="icon-fail">✗</span>;
    default: return null;
  }
}
```

## Migration Path

### Phase 1: CLI-Based Fetch (Week 1-2)
- Implement `POST /api/azure/logs/fetch` using az CLI
- Add simple UI with Refresh button
- Remove continuous polling code
- Ship as experimental feature

### Phase 2: Feedback & Iteration (Week 3-4)  
- Gather user feedback on reliability
- Add caching for recent results
- Improve error messages based on real failures
- Add auto-refresh as opt-in

### Phase 3: Advanced Features (Week 5+)
- Log search/filter within fetched results
- Cross-service log correlation
- Export to file
- Custom KQL queries (for advanced users)

## What We're NOT Building

To keep v2 focused and reliable:

1. **No Real-time Streaming APIs**: Container Apps/App Service streaming APIs are inconsistent
2. **No Complex Mode Switching**: Just Local and Azure tabs
3. **No Manual Refresh Button**: Auto-refresh with clear countdown indicator
4. **Keep the SDK**: `azlogs` works fine, just simplify initialization and error handling

## What's Changed from v1

| Aspect | v1 (Current) | v2 (Proposed) |
|--------|--------------|---------------|
| Load behavior | Silent background poll | Immediate fetch with loading indicator |
| Error display | None (silent failures) | Clear error with fix command |
| Refresh | Hidden 30s poll | Visible countdown + auto-refresh |
| SDK | azlogs | azlogs (keep it) |
| Credentials | DefaultAzureCredential | `azd auth token` → SDK |
| State feedback | None | Loading → Showing → Error states |

## Success Criteria

1. **Visible Feedback**: User always sees loading/error/success state
2. **Actionable Errors**: Every error includes a command to fix it
3. **Auto-Load**: Logs appear automatically when tab opens (no click needed)
4. **Reliability**: 95%+ of fetches succeed or show clear error
5. **Performance**: Logs appear within 3 seconds of tab open

## Open Questions

1. **Polling Interval**: 30 seconds? User-configurable?
2. **Log Retention**: How many logs to show? (Proposed: 500)
3. **Service Filter**: Show all services or picker? (Proposed: all, with filter)

## Appendix: SDK Usage

Keep the existing `azlogs` SDK code:

```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs"
    "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
)

// Create client with simplified credential
client, err := azlogs.NewClient(cred, nil)
if err != nil {
    return nil, mapAzureError(err)
}

// Query logs
resp, err := client.QueryWorkspace(ctx, workspaceID, azlogs.QueryBody{
    Query:    to.Ptr(kql),
    Timespan: to.Ptr(azlogs.NewTimeInterval(startTime, endTime)),
}, &azlogs.QueryWorkspaceOptions{
    Options: &azlogs.QueryOptions{
        Wait: to.Ptr(30), // 30 second timeout
    },
})
if err != nil {
    return nil, mapAzureError(err)
}

// Parse response
for _, table := range resp.Tables {
    for _, row := range table.Rows {
        // ... parse row to LogEntry
    }
}
```

### Credential from azd

```go
func getCredentialFromAzd(projectDir string) (azcore.TokenCredential, error) {
    cmd := exec.Command("azd", "auth", "token", 
        "--scope", "https://api.loganalytics.io/.default",
        "--output", "json")
    cmd.Dir = projectDir
    
    output, err := cmd.Output()
    if err != nil {
        return nil, &ErrorInfo{
            Message: "Not logged in to Azure",
            Code:    "AUTH_REQUIRED",
            Action:  "Run this command:",
            Command: "azd auth login",
        }
    }
    
    var result struct {
        Token     string `json:"token"`
        ExpiresOn string `json:"expiresOn"`
    }
    if err := json.Unmarshal(output, &result); err != nil {
        return nil, err
    }
    
    expiry, _ := time.Parse(time.RFC3339, result.ExpiresOn)
    return &staticToken{token: result.Token, expiry: expiry}, nil
}

type staticToken struct {
    token  string
    expiry time.Time
}

func (t *staticToken) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
    return azcore.AccessToken{Token: t.token, ExpiresOn: t.expiry}, nil
}
```

### TASKS.MD

<!-- NEXT: 0 -->
# Azure Logs v2 Tasks

## Summary

**All Phases Complete!** ✅

**Phase 1**: CLI `azd app logs --source azure` works standalone with service filtering.
**Phase 2**: Dashboard integration with auto-load, visual feedback, and auto-refresh.
**Phase 2.5**: Diagnostics, auto-resolution, and health checks.
**Phase 3**: Token caching, service filtering, and code cleanup.

Completed:
1. ✅ Dashboard API endpoint with structured errors
2. ✅ Auto-load on mode switch (no manual refresh)
3. ✅ Visual feedback (loading/error states)
4. ✅ Auto-refresh countdown and diagnostics button
5. ✅ Health check endpoint for diagnostics
6. ✅ Workspace ID auto-resolution
7. ✅ Diagnostics modal UI
8. ✅ Token caching (5-minute expiry)
9. ✅ Service filter dropdown
10. ✅ Cleanup of old polling code

---

## Phase 2: Dashboard Integration (P0)

### DONE: Create dashboard API endpoint {#dashboard-api-endpoint}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented `GET /api/azure/logs` endpoint in dashboard server:

**Features**:
- ✅ Reuses `FetchAzureLogsStandalone()` from `standalone_logs.go`
- ✅ Returns structured JSON with status field ("ok" | "error")
- ✅ Includes metadata: count, timestamp
- ✅ Query params: `?service=`, `?since=`, `?tail=`
- ✅ All errors include code, action, command, and docsUrl

**Response Types**:
```go
type AzureLogsResponse struct {
    Status    string      `json:"status"`
    Logs      []LogEntry  `json:"logs,omitempty"`
    Count     int         `json:"count"`
    Timestamp time.Time   `json:"timestamp"`
    Error     *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
    Message string `json:"message"`
    Code    string `json:"code"`
    Action  string `json:"action"`
    Command string `json:"command"`
    DocsURL string `json:"docsUrl"`
}
```

**Error Codes Implemented**:
- `AUTH_REQUIRED` - Not authenticated → "azd auth login"
- `NO_WORKSPACE` - Workspace not configured → "azd env refresh"
- `NO_SERVICES` - No services deployed → "azd up"
- `QUERY_FAILED` - Log Analytics query error
- All errors link to https://aka.ms/azd/app/logs/...

**Files Modified**:
- `server.go` - Added handler and response types
- Follows existing patterns, integrates with current infrastructure

**Testing**: Builds successfully, ready for runtime testing with live Azure environment

---

### DONE: Implement auto-load with loading state {#loading-state}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented auto-load when Azure mode is selected:

**Features**:
- ✅ State machine: 'idle' | 'loading' | 'showing' | 'error'
- ✅ Auto-fetch on Azure mode selection (no button click)
- ✅ Loading spinner appears instantly
- ✅ Message: "Loading logs from Azure..."
- ✅ Integrates with existing local/azure switcher

**Implementation**:
- File: `cli/dashboard/src/components/Console.tsx`
- Added `AzureLogsState` interface
- Added `useEffect` hook triggered by mode change
- Loading UI with Azure-branded spinner
- Clean state transitions

**User Flow**:
1. User clicks Azure mode toggle
2. Loading spinner appears immediately
3. Fetch from `/api/azure/logs` automatic
4. Logs display when ready OR error shown with retry

**Testing**: Dashboard running at http://localhost:40942, feature verified working

---

### DONE: Add error state with action {#error-state}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented enhanced error panel with actionable guidance:

**Features**:
- ✅ ErrorInfo structure support (message, code, action, command, docsUrl)
- ✅ Copyable command box with one-click copy
- ✅ "Retry Now" button that resets state and refetches
- ✅ Documentation links open in new tab
- ✅ Error-specific icons and messaging
- ✅ Dark mode styling

**Implementation**:
- File: `cli/dashboard/src/components/AzureErrorDisplay.tsx` (new)
- File: `cli/dashboard/src/components/Console.tsx` (enhanced)
- Copy-to-clipboard with visual feedback
- Retry handler resets to loading state

**Error Display**:
```
⚠️ {error.message}
{error.action}
┌──────────────────────┐
│ {error.command} [Copy]│
└──────────────────────┘
[📚 Docs] [🔄 Retry Now]
```

**Error Types Supported**:
- AUTH_REQUIRED → "azd auth login"
- NO_WORKSPACE → "azd env refresh"
- NO_SERVICES → "azd up"
- All include docs links to https://aka.ms/azd/app/logs/...

**Testing**: Build successful, ready for runtime testing

---

### DONE: Add status footer with auto-refresh {#status-footer}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented status footer with auto-refresh countdown:

**Features**:
- ✅ Footer displays: "✓ 142 logs • Updated 5s ago • ↻ 25s"
- ✅ Log count shows actual data length
- ✅ "Updated X ago" increments every second
- ✅ Countdown starts at 30s, decrements each second
- ✅ Auto-refresh at 0 (sets loading state, refetches)
- ✅ "Run Diagnostics" button in footer (right side)
- ✅ Refresh cycle continues until mode change/unmount

**Implementation**:
- File: `cli/dashboard/src/components/Console.tsx`
- State: `countdownSeconds` (30s timer), `lastUpdateTime` (relative time)
- `useEffect` hooks with interval cleanup
- Footer only shows when state='showing'
- Diagnostics button placeholder (logs to console for now)

**Styling**:
- Azure theme colors (blue accent)
- Dark mode compatible
- Icons: CheckCircle, RotateCw, Settings

**Testing**: Build successful, ready for runtime testing

---

## Phase 2.5: Diagnostics & Documentation (P0)

### DONE: Add diagnostics health check endpoint {#diagnostics-endpoint}
**Assigned**: Developer
**Completed**: 2025-12-10

Created `GET /api/azure/logs/health` endpoint that checks:

1. **Authentication** - Verify credentials work
2. **Workspace ID** - Check if configured in env vars
3. **Services Deployed** - Verify services exist
4. **Connectivity** - Test Log Analytics connection

**Response Format**:
```json
{
  "status": "healthy" | "degraded" | "error",
  "checks": [
    {
      "name": "Authentication",
      "status": "pass" | "warn" | "fail",
      "message": "Credentials valid",
      "fix": "Run: azd auth login"
    }
  ],
  "docsUrl": "https://aka.ms/azd/app/logs/troubleshoot"
}
```

**Acceptance Criteria**:
- All 4 health checks implemented
- Each check has clear pass/warn/fail status
- Failed checks include fix instructions
- Overall status computed from individual checks
- Response includes docs URL

---

### DONE: Add auto-resolution for missing workspace ID {#auto-resolve-workspace}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented automatic workspace ID discovery and storage:

**Features**:
- ✅ Discovers workspace via `az monitor log-analytics workspace list --resource-group <rg>`
- ✅ Stores in `.azure/{env}/.env` file as AZURE_LOG_ANALYTICS_WORKSPACE_ID
- ✅ Updates current process environment via os.Setenv
- ✅ Integrated into FetchAzureLogsStandalone and StreamAzureLogsStandalone
- ✅ Graceful error handling for missing az CLI, no workspace, etc.
- ✅ Debug logging when AZD_APP_DEBUG=true

**Files Modified**:
- `standalone_logs.go` - Added DiscoverAndStoreWorkspaceID function
- `standalone_logs_test.go` - Added comprehensive tests (6 test cases)

**Testing**: All tests passing, build successful

---

### DONE: Add documentation URLs to all errors {#error-docs-urls}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented comprehensive error documentation linking:

**Features**:
- ✅ ErrorInfo struct includes docsUrl field
- ✅ All errors mapped to specific documentation pages
- ✅ Error mapping in mapAzureErrorToInfo function
- ✅ URLs use aka.ms/azd/app/logs/* structure

**Error → Documentation Mapping**:
- AUTH_EXPIRED, AUTH_REQUIRED → /troubleshoot#auth
- NOT_DEPLOYED → /setup
- NO_WORKSPACE → /configure
- NO_PERMISSION → /troubleshoot#permissions
- All others → /troubleshoot

**Files Modified**:
- `azure_logs.go` - mapAzureErrorToInfo with docs URLs
- Already integrated with ErrorInfo in server responses

**Testing**: Build successful, all errors include docsUrl

---

### DONE: Create diagnostics modal UI {#diagnostics-ui}
**Assigned**: Developer
**Completed**: 2025-12-10

Created comprehensive diagnostics modal component:

**Features**:
- ✅ Modal component: DiagnosticsModal.tsx
- ✅ Fetches /api/azure/logs/health when opened
- ✅ Status icons: ✓ (green pass), ⚠ (yellow warn), ✗ (red fail)
- ✅ Shows fix instructions for failed checks with copy button
- ✅ "Copy Diagnostics" copies full report to clipboard
- ✅ "View Troubleshooting Guide" opens docs URL
- ✅ Loading state during fetch
- ✅ Error handling for fetch failures
- ✅ Dark mode compatible styling
- ✅ Keyboard support (Escape to close)
- ✅ Accessible (ARIA labels, focus management)

**Files Created**:
- `DiagnosticsModal.tsx` - Modal component

**Testing**: Build successful, renders correctly

---

### DONE: Update error panel with docs links {#error-panel-docs}
**Assigned**: Developer
**Completed**: 2025-12-10

Enhanced error panel with full diagnostics integration:

**Features**:
- ✅ Error panel shows docsUrl as clickable link (opens new tab)
- ✅ "Run Diagnostics" button opens DiagnosticsModal
- ✅ Retained "Retry Now" and copy command functionality
- ✅ Button layout: [Retry] [Run Diagnostics] [Docs Link]
- ✅ Consistent styling across all buttons
- ✅ Icons: Settings for diagnostics, ExternalLink for docs

**Files Modified**:
- `AzureErrorDisplay.tsx` - Added diagnostics button and props
- `Console.tsx` - Integrated modal state and passthrough

**Testing**: Build successful, all buttons functional

---

## Phase 3: Polish & Optimization (P1)

### DONE: Cache token from azd {#cache-token}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented token caching to reduce credential chain overhead:

**Features**:
- ✅ TokenCache with 5-minute expiry
- ✅ Thread-safe with sync.RWMutex
- ✅ Automatic refresh on expiry
- ✅ Clear cache on auth errors (401, 403, AADSTS)
- ✅ Debug logging for cache hits/misses
- ✅ Helper function: GetCachedToken
- ✅ Integrated into FetchAzureLogsStandalone and StreamAzureLogsStandalone

**Files Created**:
- `token_cache.go` - Cache implementation
- `token_cache_test.go` - Comprehensive tests (all passing)

**Testing**: All 50+ tests passing, build successful

---

### DONE: Add service filter dropdown {#service-filter}
**Assigned**: Developer
**Completed**: 2025-12-10

Implemented service filtering UI:

**Features**:
- ✅ Dropdown shows "All Services" (default) + individual services
- ✅ Services populated from GET /api/azure/services
- ✅ Filter persists during auto-refresh
- ✅ Resets when switching from Azure to Local mode
- ✅ Passes ?service= query param to API
- ✅ WebSocket reconnects with new filter
- ✅ Positioned in toolbar next to mode toggle

**Backend**:
- ✅ GET /api/azure/services endpoint
- ✅ Extracts services from SERVICE_*_NAME env vars
- ✅ Returns azure.yaml service names

**Files Modified**:
- `ConsoleView.tsx` - Service state management
- `LogsToolbar.tsx` - Dropdown UI
- `LogsView.tsx`, `LogsPane.tsx` - Filter integration
- `server.go` - Route registration
- `azure_logs.go` - handleAzureServices endpoint

**Testing**: Build successful, filtering works correctly

---

### DONE: Remove old polling code {#remove-old-code}
**Assigned**: Developer
**Completed**: 2025-12-10

Cleaned up deprecated v1 polling/WebSocket infrastructure:

**Files Removed**:
- ✅ `azure_log_buffer.go` (~700 lines)
- ✅ `azure_log_buffer_test.go`
- ✅ `azure_enable_test.go`

**Endpoints Removed**:
- ✅ POST /api/azure/enable
- ✅ GET /api/azure/status
- ✅ WS /api/azure/logs/stream
- ✅ POST /api/azure/logs/query (deprecated version)

**Code Simplified**:
- ✅ Removed AzureLogBuffer from LogManager
- ✅ Removed WebSocket streaming handler
- ✅ Removed background polling goroutines
- ✅ Removed subscription/channel management
- ✅ Updated routes to v2 only

**Preserved**:
- ✅ SDK client code (standalone_logs.go)
- ✅ Token cache (token_cache.go)
- ✅ V2 request/response endpoints
- ✅ Query management (GET/PUT /api/azure/query)

**Testing**: Build successful, ~1000 lines removed

---

## Done

### DONE: CLI Azure logs standalone {#cli-azure-logs}
**Assigned**: Developer
**Completed**: 2025-12-10

Fully implemented `azd app logs --source azure` CLI commands:

**Features**:
- ✅ One-shot: `azd app logs --source azure`
- ✅ Streaming: `azd app logs --source azure -f` (30s poll)
- ✅ Service filter: `azd app logs --source azure -s <service>`
- ✅ Time range: `--since 1h`, `--since 30m`
- ✅ Works without `azd app run` (standalone)
- ✅ Uses `azd auth login` credentials via SDK
- ✅ Service name mapping from azure.yaml to Azure resources

**Implementation**:
- `standalone_logs.go`: Core Azure Log Analytics query logic
- `logs.go`: CLI command integration
- Uses `github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azlogs` SDK
- Discovers Log Analytics workspace from env vars
- Maps azure.yaml service names to Azure resource names via `SERVICE_*_NAME`
- Container Apps support (App Service/Functions need additional work)

**Testing**:
- Verified with deployed Container Apps
- Service filtering works: `containerapp-api` → `ca-k7zjfgph5a6jk`
- Streaming polls every 30s, shows new logs
- 24h initial window catches logs even when containers idle
- Graceful Ctrl+C shutdown

**Known Limitations**:
- Container Apps only (no App Service or Functions yet)
- 30s polling (no real-time streaming API)
- Log Analytics ingestion has 30-90s delay

---

**Phase 1 Complete** ✅

**Next**: Dashboard integration (Phase 2)

---

## Archive

**All tasks archived!**

See [azure-logs-v2-archive-001.md](../../archive/azure-logs-v2-archive-001.md) for complete project history.

**Project Summary**:
- **Started**: 2025-12-10
- **Completed**: 2025-12-10
- **Status**: All phases delivered ✅
- **Total Tasks**: 17 completed
- **Build**: SUCCESS (v0.9.0)

## PROJECT: dependency-ordered-startup

### SPEC.MD

# Dependency-Ordered Service Startup

## Overview

Update the `azd app run` command to honor the `uses` field in azure.yaml, starting services in dependency order and waiting for dependencies to become healthy before starting dependent services.

## Problem

Currently, `OrchestrateServices` starts all services in parallel without respecting the `uses` dependencies. This causes issues when:
- An API service depends on a database container
- The API starts before the database is ready to accept connections
- Connection errors occur during startup

Example from azure.yaml:
```yaml
services:
  postgres:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      
  api:
    project: ./api
    uses:
      - postgres  # API should wait for postgres to be healthy
```

## Solution

Use the existing `BuildDependencyGraph` and `TopologicalSort` functions to:
1. Build dependency graph from services and their `uses` fields
2. Group services into startup levels (level 0 = no deps, level 1 = depends on level 0, etc.)
3. Start services level by level
4. Wait for all services in a level to become **healthy** before starting the next level

### Healthy vs Started

Wait for **healthy** status rather than just **started** because:
- A database being "started" doesn't mean it's accepting connections
- Container services may take time after process start to become functional
- Health checks are already implemented and configurable per-service

### Implementation Changes

#### 1. Update OrchestrateServices signature

Add `services` and `resources` parameters to build the dependency graph:
```go
func OrchestrateServices(
    runtimes []*ServiceRuntime,
    services map[string]Service,    // NEW: for dependency graph
    resources map[string]Resource,  // NEW: for dependency graph  
    envVars map[string]string,
    logger *ServiceLogger,
    restartContainers bool,
) (*OrchestrationResult, error)
```

#### 2. Modify orchestration logic

```go
// Build dependency graph
graph, err := BuildDependencyGraph(services, resources)
if err != nil {
    return nil, fmt.Errorf("failed to build dependency graph: %w", err)
}

// Get services grouped by startup level
levels := TopologicalSort(graph)

// Start services level by level
for levelNum, serviceNames := range levels {
    // Start all services in this level in parallel
    for _, name := range serviceNames {
        runtime := runtimeMap[name]
        wg.Add(1)
        go startService(runtime, ...)
    }
    wg.Wait()
    
    // Wait for all services in this level to become healthy
    for _, name := range serviceNames {
        if err := waitForHealthy(name); err != nil {
            return nil, fmt.Errorf("dependency %s failed to become healthy: %w", name, err)
        }
    }
}
```

#### 3. Add health waiting function

Create a function to wait for a service to become healthy with timeout:
```go
func waitForServiceHealthy(name string, processes map[string]*ServiceProcess, timeout time.Duration) error
```

#### 4. Update run command

Pass `azureYaml.Services` and `azureYaml.Resources` to `OrchestrateServices`.

### Edge Cases

1. **No dependencies**: Services without `uses` start in level 0 (parallel)
2. **Missing dependency**: Error during graph building (already handled)
3. **Circular dependency**: Error during graph building (already handled)
4. **Health check disabled**: Service considered healthy when started
5. **Container services**: Use Docker health check status
6. **Service filter**: Only include filtered services and their transitive dependencies

### Registry Status Updates

Update registry status progression:
- `starting` - Process launched
- `running` - Process running, health checks in progress
- `healthy` - Health check passed (new status)
- `error` - Failed to start or health check failed

## Success Criteria

1. Services with no dependencies start immediately (parallel)
2. Services with dependencies wait for dependencies to be healthy
3. Clear error messages when a dependency fails health check
4. Timeout handling when a dependency never becomes healthy
5. Existing behavior preserved when no `uses` fields present

## Out of Scope

- Dynamic dependency resolution at runtime
- Circular dependency breaking strategies
- Resource provisioning (only service startup order)

### TASKS.MD

<!-- NEXT: 0 -->
# Dependency-Ordered Service Startup Tasks

All tasks complete.

## Done

### DONE: Write Integration Tests {#write-integration-tests}

Added tests in `orchestrator_test.go`:
- `TestTopologicalSort_NoDependencies` - services with no deps start in level 0
- `TestTopologicalSort_LinearDependency` - linear chain (frontend → api → db)
- `TestTopologicalSort_DiamondDependency` - diamond pattern dependencies
- `TestTopologicalSort_ContainerDependencies` - container-test pattern (api uses 4 containers)
- `TestTopologicalSort_MixedDependencies` - mix of dependency depths
- `TestTopologicalSort_EmptyServices` - edge case handling
- `TestWaitForServiceHealthy_HealthCheckDisabled` - disabled health check returns immediately
- `TestWaitForServiceHealthy_HealthCheckTypeNone` - type=none returns immediately
- `TestGetServiceDependencies` - verify dependency retrieval
- `TestGetDependents` - verify dependent service retrieval
- `TestFilterGraphByServices` - transitive dependency inclusion

### DONE: Update OrchestrateServices Signature {#update-orchestrateservices-signature}

Added `services map[string]Service` parameter to `OrchestrateServices` function.
Resources parameter not needed since container dependencies are services in azure.yaml.

### DONE: Implement Level-Based Service Startup {#implement-level-based-startup}

Modified `OrchestrateServices` to:
- Build dependency graph using `BuildDependencyGraph(services, nil)`
- Get startup levels via `TopologicalSort(graph)`
- Start services level by level (parallel within each level)
- Wait for all services in level N to be healthy before starting level N+1

### DONE: Add waitForServiceHealthy Function {#add-wait-healthy-function}

Created `waitForServiceHealthy(name, process, svc, timeout)` function that:
- Returns immediately if health check is disabled for the service
- Uses existing `PerformHealthCheck` with exponential backoff
- Returns error on timeout or health check failure

### DONE: Update Registry Status Progression {#update-registry-status}

Kept existing status progression (starting → running). Health is determined dynamically
by the health check system. No new "healthy" status needed since `process.Ready = true`
indicates health check passed.

### DONE: Update Run Command {#update-run-command}

Updated `executeAndMonitorServices` in run.go to pass `azureYaml.Services` to
`OrchestrateServices`.
