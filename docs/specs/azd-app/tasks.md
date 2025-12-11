<!-- NEXT: #verify-logs-ux-changes -->
# azd-app Tasks

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
- ✅ Removed Azure services dropdown from toolbar
- ✅ Removed custom 1h option from timeframe picker
- ✅ Added refresh interval control (5s-5m) with presets: 5s, 10s, 30s, 1m, 5m
- ✅ Added Diagnostics button in toolbar (visible only in Azure mode)
- ✅ Build successful, all 645 tests passing

## DONE: Backend/state updates for logs defaults {#backend-state-updates-for-logs-defaults}
- Developer: clean query params/state relying on service filters, set safe defaults, enforce refresh interval validation in state, and ensure diagnostics data loads without errors.
- ✅ Added sync interval validation with bounds enforcement (5s-5m)
- ✅ Implemented handleSyncIntervalChange with clamping logic
- ✅ Removed unused service filter state variables from ConsoleView
- ✅ Backend service filters preserved for per-service queries (valid use case)
- ✅ DiagnosticsModal properly connected and functional
- ✅ Build successful, all 645 tests passing

## DONE: Implement local service override {#implement-local-service-override}
- Developer: services with `host: local` must always show local logs regardless of global Azure/local mode. Update log display logic to check service host configuration and force local logs for local-only services. Ensure mode toggle doesn't affect these services. Test with mixed local/Azure service configurations.
- ✅ Added `host` field to TypeScript Service interface to match backend ServiceInfo
- ✅ Implemented effectiveLogMode logic in ConsoleView to override logMode when service.host === 'local'
- ✅ Made timeRange prop optional in LogsPane with default value (only needed for Azure logs)
- ✅ Build successful, all 645 tests passing

## DONE: Design health-based color mapping {#design-health-color-mapping}
- Designer: define color mapping for service filter pills based on health status (red=unhealthy, yellow=degraded, green=healthy). Specify exact color values (hex/tailwind), pill states (selected/unselected), accessibility (contrast ratios), and how colors apply to icon, text, background, and ring. Ensure colors match log pane health indicators. Deliver component spec with all states and WCAG AA compliance.
- ✅ Defined health-based color scheme: green (healthy), amber/yellow (degraded/unknown), red (unhealthy)
- ✅ Colors match LogsPane health indicators: border-green-500, border-amber-500, border-red-500
- ✅ Selected state: bg-{color}-100 dark:bg-{color}-500/20 text-{color}-700 dark:text-{color}-300 ring-1 ring-{color}-500
- ✅ Unselected state: text-{color}-600 dark:text-{color}-400 hover:bg-{color}-50 dark:hover:bg-{color}-500/10
- ✅ WCAG AA contrast compliance maintained for all states

## DONE: Implement health-based colors {#implement-health-based-colors}
- Developer: apply health-based color mapping from Designer spec to service filter pills. Update getServiceIconAndColor to accept health status parameter. Wire health data from backend to FiltersBar. Ensure colors match log pane health display. Test all health states across both selected and unselected pill states.
- ✅ Added getServiceIconAndColor helper function accepting HealthStatus parameter
- ✅ Implemented health-based color schemes matching log pane indicators
- ✅ Added healthReport prop to FiltersBarProps
- ✅ Wired health data from healthReport to service pills
- ✅ Replaced checkbox UI with icon+text pill buttons
- ✅ Service icons: Globe, Server, Cpu, Zap, Box, Database, Package (default)
- ✅ Pills show health status in tooltip: "{service name} - {health status}"
- ✅ Build successful, all 645 tests passing

## TODO: Verify logs UX changes {#verify-logs-ux-changes}
- Tester: cover new controls, refresh bounds, diagnostics screen presence/loading with unit and e2e tests.

## DONE: Add timeframe + polling UI {#add-timeframe-+-polling-ui}
- Implemented timespan selector (15m,30m,1h,6h,24h) in LogsPane
- Implemented sync interval control (10s,30s,1m,5m)
- Wired to query params and auto-refresh logic in dashboard logs views

## DONE: Update schema docs {#update-schema-docs}
- Documented analytics-based schema: global workspace/polling/defaultTimespan
- Documented service-level tables/query overrides
- Updated examples in cli/docs/features/azure-logs.md

## DONE: Run build and tests {#run-build-and-tests}
- Compiled Go backend successfully (v0.9.0)
- Dashboard tests: 645 passed
- Fixed AnalyticsConfigGlobal field access errors

## DONE: Migrate azure_logs.go handlers {#migrate-azure_logs-go-handlers}
- Updated backend to use ServiceLogsConfig and AnalyticsConfigService/Global
- File: cli/src/internal/dashboard/azure_logs.go
