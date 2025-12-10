# Azure Logs v2: Simplified & Reliable Design

## Executive Summary

The current Azure logs implementation has proven unreliable due to:
1. **Log Analytics Ingestion Delay**: 30-90 second latency makes "real-time" polling feel broken
2. **Complex Initialization Chain**: Multiple dependencies (credentials → discovery → workspace → client) with many failure points
3. **KQL Query Fragility**: Queries depend on exact service naming that often mismatches between azure.yaml and Azure resources
4. **Silent Failures**: When something goes wrong, user sees nothing - no feedback

This v2 spec proposes a **simplified, transparent approach** that works reliably with clear status feedback.

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

## Error Handling Strategy

Map Azure errors to actionable guidance:

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
        }
        
    case strings.Contains(errStr, "ResourceNotFound") || strings.Contains(errStr, "404"):
        return &ErrorInfo{
            Message: "Azure resources not found",
            Code:    "NOT_DEPLOYED",
            Action:  "Deploy your app first:",
            Command: "azd up",
        }
        
    case strings.Contains(errStr, "AuthorizationFailed") || strings.Contains(errStr, "403"):
        return &ErrorInfo{
            Message: "Missing permissions on Log Analytics workspace",
            Code:    "NO_PERMISSION",
            Action:  "Grant 'Log Analytics Reader' role in Azure Portal",
            Command: "",
        }
        
    case strings.Contains(errStr, "WorkspaceNotFound"):
        return &ErrorInfo{
            Message: "Log Analytics workspace not configured",
            Code:    "NO_WORKSPACE",
            Action:  "Configure diagnostic settings in Azure Portal",
            Command: "",
        }
        
    default:
        return &ErrorInfo{
            Message: errStr,
            Code:    "UNKNOWN",
            Action:  "Check Azure Portal for details",
            Command: "",
        }
    }
}
```

## UI Changes

### Azure Logs Tab Design

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
│  📊 100 logs fetched from Container App logs • 2.3s              │
│  [ ] Auto-refresh every 30s                                      │
└──────────────────────────────────────────────────────────────────┘
```

### Error State Design

```
┌──────────────────────────────────────────────────────────────────┐
│  Services: [api ▼]  Time Range: [Last 30m ▼]  [🔄 Refresh]      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ⚠️  Authentication Expired                                      │
│                                                                  │
│  Your Azure login session has expired.                           │
│                                                                  │
│  To fix, run in your terminal:                                   │
│  ┌──────────────────────────────────┐                           │
│  │  az login                        │  [Copy]                    │
│  └──────────────────────────────────┘                           │
│                                                                  │
│  Then click Refresh to try again.                                │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
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
