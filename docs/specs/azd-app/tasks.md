<!-- NEXT: 12 -->
# azd-app Tasks

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
