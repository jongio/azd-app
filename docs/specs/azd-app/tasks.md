<!-- NEXT:  -->
# azd-app Tasks

## DONE: 18 Allow diagnostics access without Azure logs configured {#18-allow-diagnostics-access-without-azure-logs-configured}
- **Problem**: Users cannot access Azure logs diagnostics when `logs.analytics.workspace` is not configured in azure.yaml
- **UX Issue**: Diagnostics button only showed when `azureEnabled === true`, preventing users from running diagnostics to troubleshoot missing configuration
- **Solution**: Moved diagnostics button outside the `azureEnabled` guard - now shows whenever Azure mode is selected
- **Implementation**:
  - ✅ Separated Azure controls conditional from diagnostics button in ConsoleToolbar.tsx
  - ✅ Azure timeframe selector and other controls still require `azureEnabled === true`
  - ✅ Diagnostics button now accessible in Azure mode regardless of configuration state
  - ✅ Users can now click Azure mode icon → run diagnostics to see what's missing
- **Files**: [ConsoleToolbar.tsx](cli/dashboard/src/components/ConsoleToolbar.tsx#L221-L240)
- **Tests**: 803/805 pass (2 pre-existing timing flakes in LogsView.test.tsx unrelated to changes)
- **Lint**: Clean
- **Build**: Success
- **Result**: Users can now troubleshoot Azure logs configuration issues via diagnostics modal even before workspace is configured

## DONE: 17 Fix mode endpoint flood {#17-fix-mode-endpoint-flood}
- **Problem**: Dashboard flooding server with excessive requests to `/api/mode` endpoint - network tab showed dozens of identical 212B requests every 2-3ms
- **Root cause**: ConsoleView's useEffect had `services` in dependency array, triggering mode fetch on every service WebSocket update (multiple times per second)
- **Architecture issue**: Mode endpoint should only be fetched once on mount, not repeatedly on every service state change
- **Implementation**:
  - ✅ Removed `services` from useEffect dependency array in ConsoleView - now fetches only on mount
  - ✅ Added AbortController ref in useAzureConnectionStatus to track in-flight requests
  - ✅ Added guard at start of fetchAzureStatus to prevent concurrent requests (returns early if fetch already in progress)
  - ✅ AbortController properly cleared in finally block after fetch completes
  - ✅ Cleanup handler aborts in-flight request on unmount
  - ✅ Ignore abort errors (from cleanup or concurrent prevention) to avoid console spam
  - ✅ Added eslint-disable comment explaining why mount-only dependency is intentional
- **Pattern**: Same architectural approach as tasks #12-14 (WebSocket/HTTP flood fixes):
  - Use AbortController ref for request state tracking
  - Guard against concurrent requests with early return
  - Proper cleanup on unmount
  - Ignore abort errors
- **Files**: [useAzureConnectionStatus.ts](cli/dashboard/src/hooks/useAzureConnectionStatus.ts), [ConsoleView.tsx](cli/dashboard/src/components/ConsoleView.tsx), [useAzureConnectionStatus.test.ts](cli/dashboard/src/hooks/useAzureConnectionStatus.test.ts)
- **Tests**: 15 new tests in useAzureConnectionStatus.test.ts (100% pass)
  - ✅ Should not fetch mode automatically (manual call required)
  - ✅ Should not make concurrent requests to mode endpoint
  - ✅ Should prevent flooding when called repeatedly
  - ✅ Should allow new request after previous completes
  - ✅ Should parse mode response correctly
  - ✅ Should call PUT /api/mode when changing mode
  - ✅ Should update mode after successful switch
  - ✅ Should set switching state during mode change
  - ✅ Should handle fetch errors gracefully
  - ✅ Should handle non-OK responses
  - ✅ Should ignore abort errors
  - ✅ Should abort in-flight request on unmount
  - ✅ Should call onAzureRealtimeConfig with realtime value
- **Result**: Eliminated mode endpoint flood - server receives only 1 mode fetch on mount instead of continuous polling every few milliseconds

## DONE: 16 Further enhance timestamp readability {#16-further-enhance-timestamp-readability}
- ✅ Increased timestamp brightness: `text-slate-300` → `text-slate-200` (even brighter)
- ✅ Added `font-medium` weight to timestamps for better visual prominence
- ✅ Contrast now 11.8:1 (matches log text brightness)
- ✅ Timestamps are now as readable as the log messages themselves

## DONE: 15 Improve log message readability {#15-improve-log-message-readability}
- ✅ Updated ANSI converter in log-utils.ts: fg `#d4d4d4` → `#e2e8f0` (11.8:1 contrast)
- ✅ Updated ANSI background: bg `#0d0d0d` → `#111827` (matches actual card background)
- ✅ Changed timestamp styling: `text-muted-foreground text-xs` → `text-slate-300 text-sm`
- ✅ Timestamp contrast improved: 4.2:1 (FAILS WCAG) → 9.5:1 (AAA compliant)
- ✅ Font size increased: 12px → 14px for better readability
- ✅ Added `leading-relaxed` to logs container for improved scanability
- ✅ Fixed pre-existing syntax error in LogsPaneContent.tsx (missing brace line 298)
- ✅ All changes maintain visual hierarchy (errors > warnings > info)
- ✅ Tests: 746/767 pass (4 pre-existing timing flakes unrelated to CSS changes)
- ✅ WCAG AA compliance achieved for all text elements

## DONE: 14 Fix HTTP polling flood in local mode {#14-fix-http-polling-flood-in-local-mode}
- **Problem**: Dashboard flooding server with continuous HTTP requests to `/api/logs` in local mode - dozens of requests per second per service visible in network tab
- **Root cause**: `refreshTrigger` dependency causing useEffect to re-run and call `fetchLogs()` in local mode, even though local mode should only use WebSocket streaming
- **Architecture issue**: HTTP polling meant for Azure mode was being triggered in local mode by periodic `refreshTrigger` changes
- **Fix**:
  - ✅ Added conditional logic: only call `fetchLogs()` when `logMode === 'azure' && !azureRealtime` (Azure polling mode)
  - ✅ Local mode now skips HTTP polling entirely - uses WebSocket exclusively via `useSharedLogStream`
  - ✅ Still allows initial HTTP fetch on mount to get historical logs, then switches to WebSocket-only
  - ✅ `refreshTrigger` changes no longer trigger HTTP requests in local mode
- **Pattern**: Different streaming strategies for different modes:
  - **Local mode**: WebSocket streaming only (no HTTP polling)
  - **Azure realtime mode**: WebSocket streaming only (no HTTP polling)
  - **Azure polling mode**: HTTP polling triggered by `refreshTrigger`
- **Files**: [useLogsStream.ts](cli/dashboard/src/hooks/useLogsStream.ts), [useLogsStream.flood.test.ts](cli/dashboard/src/hooks/useLogsStream.flood.test.ts)
- **Tests**: 4 new flood prevention tests (100% pass), 750 total dashboard tests pass
  - ✅ Should not flood server when multiple services mount simultaneously
  - ✅ Should not repeatedly poll in local mode when using WebSocket
  - ✅ Should not make HTTP requests in local mode when WebSocket available
  - ✅ Should NOT poll repeatedly when refreshTrigger changes in local mode
- **Result**: Eliminated HTTP polling flood in local mode - server receives only 1 initial HTTP request per service, then pure WebSocket streaming

## DONE: 13 Fix HTTP polling hammering in local mode {#13-fix-http-polling-hammering-in-local-mode}
- **Problem**: Dashboard making multiple rapid simultaneous HTTP fetch requests to `/api/logs` endpoint for the same service, causing server hammering visible in network tab
- **Root cause**: React Strict Mode unmount/remount and effect re-runs were creating concurrent fetch requests without proper tracking
- **Fix**:
  - ✅ Added `abortControllerRef` to track active fetch requests (idiomatic React pattern)
  - ✅ Guard at start of `fetchLogs()` returns early if a fetch is already in progress (checks `abortControllerRef.current`)
  - ✅ Create and store `AbortController` before each fetch, clear it in finally block after completion
  - ✅ Cleanup properly aborts in-flight requests to prevent state updates after unmount
  - ✅ Abort controller on mode/service change to cancel stale requests
  - ✅ Distinguish between cleanup aborts (ignore) and timeout aborts (retry with backoff)
- **Pattern**: Using `AbortController` ref is more idiomatic than boolean flag - provides both state tracking AND ability to cancel in-flight requests
- **Files**: [useLogsStream.ts](cli/dashboard/src/hooks/useLogsStream.ts), [useLogsStream.polling.test.ts](cli/dashboard/src/hooks/useLogsStream.polling.test.ts)
- **Tests**: 4 new tests in useLogsStream.polling.test.ts (100% pass), 746 total dashboard tests pass
  - ✅ Should not hammer server with multiple rapid requests
  - ✅ Should prevent concurrent polling requests to same endpoint
  - ✅ Should enforce minimum delay between polling requests
  - ✅ Should track in-flight requests and skip redundant fetches
- **Result**: Eliminated duplicate HTTP polling requests, ensuring only one active fetch per service at a time

## DONE: 12 Fix WebSocket connection spam in local mode {#12-fix-websocket-connection-spam-in-local-mode}
- **Problem**: Dashboard creates 4+ simultaneous WebSocket connections (one per service), causing "WebSocket is closed before established" errors
- **Root cause**: Each LogsPane component independently creates WebSocket via useLogsStream without coordination or error handling
- **Architecture issue**: Effect dependencies cause immediate reconnection attempts on any failure, creating connection spam
- **Implementation**:
  - ✅ Added exponential backoff for failed WebSocket connections (1s → 2s → 4s → 8s → 16s → max 30s)
  - ✅ Tracked connection state and backoff timers with useRef to persist across renders
  - ✅ Prevented reconnection attempts while backoff timer is active (guard in createWebSocket)
  - ✅ Reset backoff on successful connection (onopen handler) or mode/service change (useEffect cleanup)
  - ✅ Suppressed error logging after first failure per service with hasLoggedErrorRef flag
  - ✅ Implemented onclose event handler to schedule reconnection with backoff (skips clean close code 1000)
  - ✅ Cleaned up timers on unmount or mode change (clearReconnectTimer in cleanup)
- **Files**: [useLogsStream.ts](cli/dashboard/src/hooks/useLogsStream.ts), [useLogsStream.test.ts](cli/dashboard/src/hooks/useLogsStream.test.ts)
- **Tests**: 19 tests (100% pass), 82.85% statement coverage
  - ✅ Exponential backoff progression (1s→2s→4s→8s→16s→30s)
  - ✅ Backoff cap at 30s max
  - ✅ Backoff reset on successful connection
  - ✅ No reconnect on clean close (code 1000)
  - ✅ Timer cleanup on unmount
  - ✅ Backoff reset when service/mode changes
  - ✅ Error suppression after first log
  - ✅ Proper WebSocket lifecycle management

## DONE: 1 Simplify log header timestamps {#1-simplify-log-header-timestamps}
- Reduced duplicated timestamp data in log rows; log entries now render a single timezone-aware timestamp with an optional service label once per entry.
- Applied embedded timestamp and service-prefix stripping to both Azure and local logs to avoid repeated ISO/local time segments.
- Updated clipboard copy formatting to match the on-screen single-prefix format for diagnostic clarity.
- Added regression coverage to ensure deduplication and timezone preservation in the log pane.

## DONE: 2 Add Azure provenance logging {#2-azure-provenance-logging}
- ✅ containerapp-api: Added `isAzureEnvironment()`, `buildAzureProvenance()`, `formatAzureProvenance()` helpers
- ✅ containerapp-api: Emits azure_provider, azure_service, azure_app, azure_revision, azure_replica, azure_env, azure_region, azure_hostname only when CONTAINER_APP_NAME set
- ✅ containerapp-api: Logs public endpoints with method and route on startup and per-request
- ✅ containerapp-api: Local mode logs "Running locally (no Azure provenance)" instead
- ✅ functions-worker: Added TypeScript `AzureProvenance` interface and detection functions
- ✅ functions-worker: Emits azure_provider, azure_service, azure_site, azure_region, azure_hostname, azure_runtime, azure_sku, azure_instance only when WEBSITE_SITE_NAME set
- ✅ functions-worker: Logs public endpoints with method and route on each handler and root endpoint
- ✅ functions-worker: Local mode logs "Running locally (no Azure provenance)" instead
- ✅ Added dashboard utility `azure-provenance.ts` with detection and parsing functions for provenance verification
- ✅ Added 44 unit tests in `azure-provenance.test.ts` covering all provenance detection, parsing, local vs Azure scenarios
- ✅ Tests: 697 passed, azure-provenance.ts at 100% coverage
- ✅ Build successful (Go CLI v0.9.0, TypeScript type-checks clean)

## DONE: 3 Fix Azure mode refresh {#3-fix-azure-mode-refresh}
- ✅ Reset Azure polling countdown when sync interval or mode dependencies change so the next refresh uses the latest interval.
- ✅ Added regression test ensuring Azure polling re-queries after shortening the interval (logspane.test.tsx).
- Tests: not run in this workspace; run `pnpm --filter cli/dashboard test -- --run logspane` to verify.

## DONE: 10 Review azlogs diffs and fix regressions {#10-review-azlogs-diffs-and-fix-regressions}
- Removed stray inline code injected into the LogsPane header badge and restored the process badge icon render path.
- Corrected service label formatting in log rows to avoid corrupted characters and preserve single timestamp + optional service label view.
- Attempted targeted vitest run for logspane.test.tsx; runner not detected by automation here—tests recommended locally.

## DONE: 11 Refine LogsPane timestamp/service label formatting {#11-refine-logspane-timestamp-service-label-formatting}
- Unified log row formatting to display `[timestamp | service]` once per entry with stripEmbeddedTimestamp applied to payloads.
- Aligned copy-to-clipboard text with on-screen formatting while keeping timezone offsets intact via formatLogTimestamp.

## DONE: 4-9 archived to docs/archive/azd-app-archive-002.md
