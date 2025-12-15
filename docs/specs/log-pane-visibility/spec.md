# Log Pane Visibility Defaults

## Overview
When the dashboard Console view first loads, every discovered service should render as a log pane. Users should be able to observe each service’s lifecycle and health status transitions in-place over time.

Today, some panes can be hidden implicitly by state/health filtering. This makes services “disappear” even though they still exist, which is confusing and prevents users from seeing the full system at a glance.

## Goals
- Always show a log pane for every service by default.
- Never hide a service’s pane based on lifecycle state or health status.
- Only hide panes when the user explicitly hides a service (service selection).
- Preserve existing status indicators (state/health) so users can still see transitions.

## Non-goals
- Changing how services are discovered or fetched.
- Removing state/health filters from the UI.
- Changing the semantics of log-level filtering.

## Functional Requirements

### Pane Visibility Rules
1. **Default behavior**: On first load, the grid view renders a pane for every service returned by the services context.
2. **Explicit hide only**: A service pane may be hidden only if the user explicitly deselects that service in the Services selector.
3. **No implicit hide**: State and Health filters must not affect which panes are rendered.
4. **Persistence**: Explicit service hides/selections continue to persist across sessions as they do today.

### Filters Semantics
- **Services selector**: Controls pane visibility (this is the only control that may hide panes).
- **State filter**: Affects only state-related UI affordances (e.g., indicators, summaries, highlighting) but not pane visibility.
- **Health filter**: Affects only health-related UI affordances (e.g., indicators, summaries, highlighting) but not pane visibility.

## Acceptance Criteria
- With no user interaction, all services appear as panes on initial dashboard load.
- Services remain visible even when their lifecycle state changes (starting/running/stopped) or health changes (healthy/degraded/unhealthy/unknown).
- Toggling State/Health filters does not remove panes from the grid.
- Explicitly deselecting a service hides its pane; reselecting shows it again.
- Dashboard unit tests cover the “state/health filters do not hide panes” behavior.
