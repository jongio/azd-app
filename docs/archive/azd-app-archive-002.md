# azd-app Archive #002
Archived: 2025-12-15

## Completed Tasks

## DONE: 4 Fix containerapp-api logs {#4-fix-containerapp-logs}
- Confirmed logs appear when timeframe is adjusted - no backend fix needed.

## DONE: 5 Implement service filters UI redesign {#5-implement-service-filters-ui}
- Added getServiceIconAndColor helper function with contextual icons
- Icons: Globe (web/frontend), Server (api/backend), Database (db), Box (container), Cpu (worker), Zap (functions), Package (default)
- Replaced checkboxes with pill buttons (icon + text) in FiltersBar
- 8-color cycling palette (emerald, purple, blue, rose, cyan, violet, amber, teal)
- Max-width 150px with text truncate and title tooltip
- Selected state: colored bg/text/ring, Unselected: transparent with hover
- Maintains all existing filter toggle behavior
- Build successful, all 645 tests passing

## DONE: 6 Move timeframe/refresh to both modes {#6-timeframe-refresh-both-modes}
- Removed 'Azure mode only' conditional from timeframe picker in toolbar
- Timeframe control (15m, 1h, 6h, 24h) now available for both local and cloud modes
- Refresh interval already available for both modes (no change needed)
- Build successful, all 645 tests passing

## DONE: 7 Add countdown timer to LogsPane {#7-logspane-countdown-timer}
- Added syncInterval prop to LogsPaneProps interface
- Added secondsUntilRefresh state with countdown logic
- Implemented countdown effect that updates every second
- Added footer component showing "Next refresh in Xs"
- Only displays when not collapsed, syncInterval set, not paused, and countdown > 0
- Uses RotateCw icon and muted styling for subtle appearance
- Passed syncInterval prop from ConsoleView to LogsPane
- Build successful, all 645 tests passing

## DONE: 8 Design logs UI simplification {#8-design-logs-ui-simplification}
- Designer: remove services dropdown, drop custom 1h window option, add refresh interval control with 5s-5m bounds, and restore diagnostics screen entry points; deliver component specs with states, validation, and responsive guidance.

## DONE: 9 Implement logs UI changes {#9-implement-logs-ui-changes}
- Developer: apply designer spec to dashboard, remove services filter dependencies, adjust timeframe picker, add refresh interval control with bounds/presets and persistence decision, and reinstate diagnostics screen visibility and navigation.
- Implemented effectiveLogMode logic in ConsoleView to override logMode when service.host === 'local'
