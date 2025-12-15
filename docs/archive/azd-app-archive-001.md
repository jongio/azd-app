# azd-app Archive #001
Archived: 2025-12-14

## Completed Tasks

## DONE: Fix containerapp-api logs {#fix-containerapp-logs}
- Confirmed logs appear when timeframe is adjusted - no backend fix needed.

## DONE: Implement service filters UI redesign {#implement-service-filters-ui}
- Added getServiceIconAndColor helper function with contextual icons
- Icons: Globe (web/frontend), Server (api/backend), Database (db), Box (container), Cpu (worker), Zap (functions), Package (default)
- Replaced checkboxes with pill buttons (icon + text) in FiltersBar
- 8-color cycling palette (emerald, purple, blue, rose, cyan, violet, amber, teal)
- Max-width 150px with text truncate and title tooltip
- Selected state: colored bg/text/ring, Unselected: transparent with hover
- Maintains all existing filter toggle behavior
- Build successful, all tests passing

## DONE: Move timeframe/refresh to both modes {#timeframe-refresh-both-modes}
- Removed 'Azure mode only' conditional from timeframe picker in toolbar
- Timeframe control (15m, 1h, 6h, 24h) now available for both local and cloud modes
- Refresh interval already available for both modes (no change needed)
- Build successful, all tests passing

## DONE: Add countdown timer to LogsPane {#logspane-countdown-timer}
- Added syncInterval prop to LogsPaneProps interface
- Added secondsUntilRefresh state with countdown logic
- Implemented countdown effect that updates every second
- Added footer component showing "Next refresh in Xs"
- Only displays when not collapsed, syncInterval set, not paused, and countdown > 0
- Uses RotateCw icon and muted styling for subtle appearance
- Passed syncInterval prop from ConsoleView to LogsPane
- Build successful, all tests passing

## DONE: Design logs UI simplification {#design-logs-ui-simplification}
- Designer: remove services dropdown, drop custom 1h window option, add refresh interval control with 5s-5m bounds, and restore diagnostics screen entry points; deliver component specs with states, validation, and responsive guidance.

## DONE: Implement logs UI changes {#implement-logs-ui-changes}
- Developer: apply designer spec to dashboard, remove services filter dependencies, adjust timeframe picker, add refresh interval control with bounds/presets and persistence decision, and reinstate diagnostics screen visibility and navigation.
- Removed Azure services dropdown from toolbar
- Removed custom 1h option from timeframe picker
- Added refresh interval control (5s-5m) with presets: 5s, 10s, 30s, 1m, 5m
- Added Diagnostics button in toolbar (visible only in Azure mode)
- Build successful, all tests passing

## DONE: Backend/state updates for logs defaults {#backend-state-updates-for-logs-defaults}
- Developer: clean query params/state relying on service filters, set safe defaults, enforce refresh interval validation in state, and ensure diagnostics data loads without errors.
- Added sync interval validation with bounds enforcement (5s-5m)
- Implemented handleSyncIntervalChange with clamping logic
- Removed unused service filter state variables from ConsoleView
- Backend service filters preserved for per-service queries (valid use case)
- DiagnosticsModal properly connected and functional
- Build successful, all tests passing

## DONE: Implement local service override {#implement-local-service-override}
- Developer: services with host: local must always show local logs regardless of global Azure/local mode.
- Added host field to TypeScript Service interface to match backend ServiceInfo
- Implemented effectiveLogMode logic in ConsoleView to override logMode when service.host === 'local'
- Made timeRange prop optional in LogsPane with default value (only needed for Azure logs)
- Build successful, all tests passing

## DONE: Design health-based color mapping {#design-health-color-mapping}
- Designer: define color mapping for service filter pills based on health status (red=unhealthy, yellow=degraded, green=healthy), including accessibility guidance.
- Defined health-based color scheme: green (healthy), amber/yellow (degraded/unknown), red (unhealthy)
- Colors match LogsPane health indicators

## DONE: Implement health-based colors {#implement-health-based-colors}
- Developer: apply health-based color mapping from Designer spec to service filter pills.
- Wired health data from backend to service pills
- Build successful, all tests passing

## DONE: Draft logs test plan {#draft-logs-test-plan}
- Created test plan mapping UX requirements to unit, integration, and e2e coverage with owners, environments, pass/fail, and regression gates.

## DONE: Add dashboard unit tests {#add-dashboard-unit-tests}
- Added Vitest coverage for refresh interval clamping/persistence, diagnostics visibility in Azure mode, effectiveLogMode override for host=local, health-based pill colors, and default/custom timeRange handling in LogsPane.

## DONE: Add backend integration tests {#add-backend-integration-tests}
- Added dashboard Go integration coverage for /api/azure/logs defaults/bounds/service filter pass-through and /api/azure/logs/health diagnostics responses using injectable Azure wrappers.

## DONE: Add e2e coverage for logs UX {#add-e2e-logs-ux}
- Added Playwright coverage for logs UX requirements: services dropdown removal, timeframe presets (no 1h option), refresh interval clamping (5s–5m), diagnostics visibility in Azure mode, and host=local override behavior.

## DONE: Coverage and reporting {#coverage-and-reporting}
- Dashboard coverage: pnpm test:coverage (script stabilized for Windows file locking).
- Backend coverage: go test ./... and targeted dashboard handler tests.

## DONE: Verify logs UX changes {#verify-logs-ux-changes}
- Verified: dashboard unit tests pass, dashboard Playwright e2e passes, and cli Go tests pass.

## DONE: Add timeframe + polling UI {#add-timeframe-+-polling-ui}
- Implemented timespan selector in LogsPane
- Implemented sync interval control
- Wired to query params and auto-refresh logic in dashboard logs views

## DONE: Update schema docs {#update-schema-docs}
- Documented analytics-based schema: global workspace/polling/defaultTimespan
- Documented service-level tables/query overrides
- Updated examples in cli/docs/features/azure-logs.md

## DONE: Run build and tests {#run-build-and-tests}
- Compiled Go backend successfully
- Dashboard tests passed
- Fixed AnalyticsConfigGlobal field access errors

## DONE: Migrate azure_logs.go handlers {#migrate-azure_logs-go-handlers}
- Updated backend to use ServiceLogsConfig and AnalyticsConfigService/Global
- File: cli/src/internal/dashboard/azure_logs.go
