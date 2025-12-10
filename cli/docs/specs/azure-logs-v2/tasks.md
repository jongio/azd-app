<!-- NEXT: #simplify-credential-init -->
# Azure Logs v2 Tasks

## Summary

Fix Azure logs reliability by:
1. Keeping `azlogs` SDK (already works)
2. Simplifying credential chain (use `azd auth token`)
3. Adding clear visual feedback (loading/error/countdown states)
4. Auto-load on tab open (no manual refresh)

---

## Phase 1: Backend Reliability (P0)

### TODO: Simplify credential initialization {#simplify-credential-init}
**Assigned**: Developer
**Priority**: P0

Create `cli/src/internal/azure/azdcredential.go`:

- `GetCredentialFromAzd(projectDir) (azcore.TokenCredential, error)`
- Use `azd auth token --scope https://api.loganalytics.io/.default`
- Return `staticToken` credential that SDK can use
- Clear error with action when not authenticated

Replace `DefaultAzureCredential` usage with this simpler approach.

**Acceptance Criteria**:
- Credential works when user has run `azd auth login`
- Error clearly says "Run `azd auth login`" when not authenticated
- Token cached and reused (not fetched on every request)
- Unit tests for success and failure paths

---

### TODO: Add lazy initialization to Azure client {#lazy-init}
**Assigned**: Developer
**Priority**: P0

Modify `handleAzureLogs` endpoint:
- Initialize Azure client on first request (not at startup)
- Return structured error with action if init fails
- Cache initialized client for reuse

This eliminates silent failures during startup.

**Acceptance Criteria**:
- First request triggers initialization
- Failed init returns JSON error with fix command
- Subsequent requests reuse cached client
- No startup delays/failures

---

### TODO: Return structured errors from API {#structured-errors}
**Assigned**: Developer  
**Priority**: P0

Update `GET /api/azure/logs` response format:

```go
type AzureLogsResponse struct {
    Status    string      `json:"status"`    // "ok" | "error"
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
}
```

Map all Azure errors to actionable guidance.

**Acceptance Criteria**:
- Every error response includes `action` and `command`
- Auth errors say "Run `azd auth login`"
- Not-deployed errors say "Run `azd up`"
- Permission errors explain what role is needed

---

## Phase 2: Frontend Feedback (P0)

### TODO: Add loading state to Azure logs panel {#loading-state}
**Assigned**: Developer
**Priority**: P0

When Azure tab opens:
- Immediately show loading spinner
- Text: "Loading logs from Azure..."
- Auto-fetch logs (no button click)

**Acceptance Criteria**:
- Loading spinner appears instantly on tab switch
- Fetch starts automatically
- Loading clears when data arrives or error occurs

---

### TODO: Add error state with action {#error-state}
**Assigned**: Developer
**Priority**: P0

When fetch fails:
- Show error message prominently
- Show the fix command in a copyable box
- Add "Retry" button
- Retry should show loading state again

**Acceptance Criteria**:
- Error panel is clearly visible
- Command can be copied with one click
- Retry button works
- Different errors show different actions

---

### TODO: Add status footer with countdown {#status-footer}
**Assigned**: Developer
**Priority**: P0

After successful fetch, show footer:
- "✓ 142 logs • Updated 5s ago • Next refresh in 25s"
- Countdown updates every second
- When countdown hits 0, auto-refresh (show loading)

**Acceptance Criteria**:
- Footer shows log count
- "Updated X ago" updates in real time
- Countdown visible and accurate
- Auto-refresh happens at 0

---

## Phase 3: Polish (P1)

### TODO: Cache token from azd {#cache-token}
**Assigned**: Developer
**Priority**: P1

Don't call `azd auth token` on every request:
- Cache token for 5 minutes
- Refresh before expiry
- Clear cache on auth error

---

### TODO: Add service filter dropdown {#service-filter}
**Assigned**: Developer
**Priority**: P1

Let user filter logs by service:
- Dropdown with discovered services
- "All Services" option (default)
- Persist selection in session

---

### TODO: Remove old polling code {#remove-old-code}
**Assigned**: Developer
**Priority**: P1

Clean up v1 implementation:
- Remove `azure_log_buffer.go` background polling
- Remove WebSocket streaming for Azure logs
- Simplify to request/response model
- Keep SDK client code

---

## Done

(none yet)

---

## Archive

See [archive](../../archive/) for completed task history.
