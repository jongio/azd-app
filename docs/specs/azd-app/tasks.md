<!-- NEXT: #fix-azure-mode-refresh -->
# azd-app Tasks

## IN PROGRESS: Fix Azure mode refresh {#fix-azure-mode-refresh}
- Auto-refresh in Azure mode must actually re-query Azure logs when the countdown reaches zero; ensure polling is not paused or blocked by mode toggles.
- Verify refresh honors the selected sync interval bounds (5s-5m) and resumes correctly after manual refresh or tab visibility changes.
- Add automated coverage (unit or e2e) proving at least one refresh cycle happens in Azure mode within the configured interval.

## TODO: Simplify log header timestamps {#simplify-log-header-timestamps}
- Reduce duplicated timestamp data in log rows; display a single timestamp format plus source/service once per entry while retaining timezone clarity.
- Keep necessary Azure metadata (service name, level, message) visible without repeating full ISO timestamps multiple times.
- Remove repeated date/time segments like `[2025-12-13T05:45:49.1071934-08:00] [appservice-web] [2025-12-13 05:45:49]` so each entry shows only one clear timestamp.
- Update tests/snapshots to reflect the streamlined header format without losing ordering or diagnostic fidelity.

## DONE: Add Azure provenance logging {#azure-provenance-logging}
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

## DONE: Fix containerapp-api logs {#fix-containerapp-logs}
- Confirmed logs appear when timeframe is adjusted - no backend fix needed.

## DONE: Implement service filters UI redesign {#implement-service-filters-ui}
- ✅ Added getServiceIconAndColor helper function with contextual icons
- ✅ Icons: Globe (web/frontend), Server (api/backend), Database (db), Box (container), Cpu (worker), Zap (functions), Package (default)
- ✅ Replaced checkboxes with pill buttons (icon + text) in FiltersBar
- ✅ 8-color cycling palette (emerald, purple, blue, rose, cyan, violet, amber, teal)
- ✅ Max-width 150px with text truncate and title tooltip
- ✅ Selected state: colored bg/text/ring, Unselected: transparent with hover
- ✅ Maintains all existing filter toggle behavior
- ✅ Build successful, all 645 tests passing

## DONE: Move timeframe/refresh to both modes {#timeframe-refresh-both-modes}
- ✅ Removed 'Azure mode only' conditional from timeframe picker in toolbar
- ✅ Timeframe control (15m, 1h, 6h, 24h) now available for both local and cloud modes
- ✅ Refresh interval already available for both modes (no change needed)
- ✅ Build successful, all 645 tests passing

## DONE: Add countdown timer to LogsPane {#logspane-countdown-timer}
- ✅ Added syncInterval prop to LogsPaneProps interface
- ✅ Added secondsUntilRefresh state with countdown logic
- ✅ Implemented countdown effect that updates every second
- ✅ Added footer component showing "Next refresh in Xs"
- ✅ Only displays when not collapsed, syncInterval set, not paused, and countdown > 0
- ✅ Uses RotateCw icon and muted styling for subtle appearance
- ✅ Passed syncInterval prop from ConsoleView to LogsPane
- ✅ Build successful, all 645 tests passing

## DONE: Design logs UI simplification {#design-logs-ui-simplification}
- Designer: remove services dropdown, drop custom 1h window option, add refresh interval control with 5s-5m bounds, and restore diagnostics screen entry points; deliver component specs with states, validation, and responsive guidance.

## DONE: Implement logs UI changes {#implement-logs-ui-changes}
- Developer: apply designer spec to dashboard, remove services filter dependencies, adjust timeframe picker, add refresh interval control with bounds/presets and persistence decision, and reinstate diagnostics screen visibility and navigation.
- ✅ Implemented effectiveLogMode logic in ConsoleView to override logMode when service.host === 'local'
