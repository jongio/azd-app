# Log Streaming Architecture Simplification

## Overview

Simplify the dashboard log display architecture by removing redundant frontend HTTP polling and leveraging the backend's WebSocket-based streaming for all real-time updates.

## Problem Statement

Current architecture has unnecessary complexity:
- **Dual transport coordination**: Frontend manages both HTTP polling AND WebSocket streaming
- **Redundant polling**: Frontend polls Azure logs every N seconds via HTTP, even though backend WebSocket already handles polling internally
- **Complex state management**: Multiple refs, triggers, and countdown timers to coordinate polling
- **Code duplication**: ~200 lines of polling logic that duplicate backend functionality

## Current Architecture (Before)

```
Frontend Log Fetching (Azure Mode)
├── Initial HTTP fetch (historical logs)
├── HTTP polling every 30s (frontend-initiated)
│   └── useAzurePollingRefreshTrigger (91 lines)
├── WebSocket streaming (backend-initiated)
│   └── Backend polls Log Analytics every 5s
└── Complex coordination logic in useLogsStream
```

**Problems:**
1. Frontend HTTP polling is wasteful - backend already polls via WebSocket
2. Two sources of truth for "refresh interval" (frontend timer + backend timer)
3. Complex synchronization between HTTP and WebSocket transports
4. Confusion about when polling vs streaming is used

## Simplified Architecture (After)

```
Frontend Log Fetching (All Modes)
├── Initial HTTP fetch (one-time, historical logs)
│   └── On mount, mode change, or time range change
└── WebSocket streaming (continuous, all updates)
    ├── Local: Native process log streaming
    ├── Azure Container Apps: Native Azure streaming API
    └── Azure Log Analytics: Backend polls every 5s, pushes diffs
```

**Benefits:**
1. ✅ Single transport from frontend perspective (WebSocket after initial fetch)
2. ✅ Backend decides polling strategy (transparent to frontend)
3. ✅ Simpler state management (no polling triggers/timers)
4. ✅ ~200 fewer lines of frontend code
5. ✅ Consistent behavior across local and Azure modes

## Backend Capabilities (No Changes Required)

The backend already supports this pattern:

### Log Analytics (Query-Based)
```go
// Backend WebSocket handler (azure_logs_stream.go:88-115)
func streamAzureLogsViaPolling(ctx, serviceName, conn, since) {
    ticker := time.NewTicker(5 * time.Second)
    for {
        // Backend polls Log Analytics every 5s
        fetchAndSendAzureLogs(ctx, projectDir, serviceName, since, conn)
    }
}
```

### Container Apps (Real Streaming)
```go
// Backend WebSocket handler (azure_logs_stream.go:117-200)
func streamAzureLogsRealtime(ctx, serviceName, conn) {
    // Uses Azure Container Apps native streaming API
    streamer.Start(ctx, logsCh)
    // Forwards events to WebSocket
}
```

**Key Insight:** Backend WebSocket abstracts the polling/streaming difference. Frontend just receives logs over WebSocket regardless of backend implementation.

## Implementation Plan

### Phase 1: Remove Frontend Polling

#### 1.1 Delete `useAzurePollingRefreshTrigger`
- **File**: `cli/dashboard/src/hooks/useAzurePollingRefreshTrigger.ts` (91 lines)
- **Reason**: Frontend no longer needs to trigger periodic refreshes

#### 1.2 Simplify `useLogsStream`
- **File**: `cli/dashboard/src/hooks/useLogsStream.ts`
- **Changes**:
  - Remove `refreshTrigger` parameter
  - Remove `shouldPollViaHttp` logic
  - Simplify to: initial fetch + WebSocket streaming
  - Remove complex coordination logic (lines 237-270)

**Before:**
```typescript
const shouldPollViaHttp = logMode === 'azure' && !azureRealtime
if (shouldPollViaHttp || fetchCountForKeyRef.current === 0) {
  void fetchLogs()  // Repeated HTTP polling
}
```

**After:**
```typescript
// Always fetch initial logs once
if (fetchCountForKeyRef.current === 0) {
  void fetchLogs()  // One-time HTTP fetch
}
// WebSocket handles all subsequent updates
```

#### 1.3 Update Component Props
- **LogsPane**: Remove `syncInterval`, `refreshTrigger` props
- **ConsoleView**: Remove `syncInterval` state
- **LogsPaneRefreshFooter**: Delete or replace with connection status

### Phase 2: Clean Up UI

#### 2.1 Remove Polling UI Elements
- Delete countdown timer: "Auto-refresh in 27s"
- Replace with connection status: "🟢 Live" / "🔴 Disconnected"
- Remove `useConsoleSyncSettings` hook (if only used for syncInterval)

#### 2.2 Simplify LogsPaneRefreshFooter
```typescript
// OLD: Countdown timer
<span>Next refresh: {secondsUntilRefresh}s</span>

// NEW: Connection status
<span className="flex items-center gap-2">
  {wsConnected ? (
    <><span className="w-2 h-2 rounded-full bg-green-500" /> Live</>
  ) : (
    <><span className="w-2 h-2 rounded-full bg-red-500" /> Disconnected</>
  )}
</span>
```

### Phase 3: Remove Legacy Code

#### 3.1 Delete `LogsView.tsx`
- **File**: `cli/dashboard/src/components/LogsView.tsx` (336 lines)
- **Reason**: Duplicate of `LogsPane` functionality, predates hook-based architecture
- **Migration**: Replace any usages with `LogsPaneGrid` + `hideControls` prop

#### 3.2 Clean Up Tests
- Update tests that reference `refreshTrigger`
- Remove polling-specific test files if applicable
- Update snapshots

### Phase 4: Documentation

#### 4.1 Add Inline Documentation
Document the simplified flow in `useLogsStream`:
```typescript
/**
 * Manages log streaming for a service.
 * 
 * Flow:
 * 1. Initial fetch: HTTP GET for historical logs (one-time)
 * 2. Live updates: WebSocket streaming (continuous)
 *    - Local: Process stdout/stderr streaming
 *    - Azure: Backend handles polling (Log Analytics) or streaming (Container Apps)
 * 
 * Frontend doesn't distinguish between polling and streaming - 
 * backend WebSocket abstracts this complexity.
 */
```

## Parameters to Remove

### `syncInterval`
- ❌ LogsPane prop
- ❌ ConsoleView state
- ❌ LogsPaneRefreshFooter prop

### `refreshTrigger`
- ❌ useLogsStream parameter
- ❌ All test references

### `azureRealtime` - KEEP
- ✅ Keep this parameter
- **Reason**: Tells backend whether to attempt native streaming (Container Apps) or use polling (Log Analytics)
- Backend uses this to choose implementation strategy

## Files to Modify

### Delete
- `cli/dashboard/src/hooks/useAzurePollingRefreshTrigger.ts`
- `cli/dashboard/src/components/LogsView.tsx`
- `cli/dashboard/src/hooks/useLogsStream.polling.test.ts` (if exists)

### Modify
- `cli/dashboard/src/hooks/useLogsStream.ts` - Simplify logic
- `cli/dashboard/src/components/LogsPane.tsx` - Remove props
- `cli/dashboard/src/components/ConsoleView.tsx` - Remove state
- `cli/dashboard/src/components/LogsPaneRefreshFooter.tsx` - Simplify UI
- `cli/dashboard/src/hooks/useConsoleSyncSettings.ts` - Review usage
- Test files referencing removed params

## Backward Compatibility

### Breaking Changes
None - this is internal refactoring. The public behavior (logs appear in real-time) remains identical.

### User-Visible Changes
- **Before**: "Auto-refresh in 27s" countdown
- **After**: "🟢 Live" connection status

This is actually an improvement - users see connection status instead of confusing countdown.

## Testing Strategy

### Manual Testing
1. **Local logs**: Verify real-time streaming works
2. **Azure Container Apps**: Verify native streaming works (if `azureRealtime=true`)
3. **Azure Log Analytics**: Verify backend polling works (logs appear every 5s)
4. **Mode switching**: Verify switching between local/azure works
5. **Time range changes**: Verify re-fetching historical logs works
6. **Connection loss**: Verify reconnection behavior

### Automated Testing
1. Update existing tests to remove `refreshTrigger` references
2. Add tests for simplified `useLogsStream` flow
3. Verify WebSocket connection lifecycle
4. Test initial fetch + streaming coordination

## Success Criteria

- ✅ Logs appear in real-time for all modes (local, Azure)
- ✅ No duplicate HTTP polling from frontend
- ✅ Connection status shows instead of countdown timer
- ✅ Code is simpler (remove ~200 lines)
- ✅ All tests pass
- ✅ No user-facing regressions

## Rollback Plan

If issues arise:
1. Revert commits (changes are isolated to frontend)
2. Backend behavior unchanged, so no backend rollback needed
3. Git history preserves previous polling implementation

## Timeline

- Spec review: 1 day
- Implementation: 2-3 days
- Testing: 1 day
- Total: 4-5 days

## Related Issues

This simplification addresses architectural issues identified in the log display review:
- Duplicate code between LogsView and LogsPane
- Overuse of refs for state management
- Complex HTTP + WebSocket coordination
- Unclear responsibility boundaries
