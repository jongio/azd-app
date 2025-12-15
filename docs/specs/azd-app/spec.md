# azd-app UX updates

## Context
The dashboard logs experience currently exposes a services dropdown, a custom 1-hour window control, and fixed refresh intervals. Users asked to simplify the controls, constrain refresh cadence, and restore the log setup diagnostics surface.

## Goals
- Simplify log filtering by removing the services selector where it is not needed.
- Remove the custom "1 hour" window option from the timeframe picker.
- Offer a refresh interval control with safe bounds (5s min, 5m max) and sensible presets.
- Reintroduce the log setup diagnostics screen to help users configure and validate log ingestion.

## Non-Goals
- Changing backend ingestion or analytics schema.
- Redesigning other dashboard panels or navigation.
- Altering authentication, role-based access, or permissions.

## Requirements
- Services dropdown: remove from the dashboard logs view; any dependent query parameters or state should be cleaned up or defaulted to "all".
- Timeframe picker: remove the custom 1-hour window option; keep existing preset ranges (e.g., 15m, 30m, 6h, 24h) unless they conflict with this change.
- Refresh interval: add a user-selectable control; enforce minimum of 5 seconds and maximum of 5 minutes; provide preset options spanning that range; validate input to stay within bounds.
- Diagnostics screen: reinstate the log setup/diagnostic screen previously removed; ensure navigation entry points are visible and stateful data loads without errors.
- Local service override: services with `host: local` must always display local logs regardless of the global Azure/local mode toggle; the mode selector should not affect local-only services.

## UX and Validation Notes
- Controls must fail safely: if a user enters a value outside the allowed refresh range, clamp or show inline validation.
- Removing controls must not leave dead query parameters or broken routing; defaults should produce a valid log query.
- Diagnostics screen should present clear status and guidance for misconfigured log pipelines.
- Service filter pills must use health-based colors matching the corresponding log pane: green (healthy), yellow (degraded), red (unhealthy).

## Log header timestamp simplification

### Surface
- Logs pane header title for a selected service (expanded and collapsed states).

### Behavior
- The visible header title must not include an embedded timestamp.
	- Preferred: show only the service name or the concept "Logs for {service}".
- If "last refresh" time is needed for debugging in Azure mode, expose it behind an affordance (for example an info icon opening a small panel), not as inline header text.
- Local mode must not display a last-refresh timestamp in the header.

### Accessibility
- Any info affordance must be keyboard operable and labeled (e.g., aria-label "Log details").
- Truncation must not hide critical information without an accessible path to discover it.

### Tests/Acceptance
- Unit tests must assert the header title region does not contain a datetime-like pattern.
- If an Azure-only details affordance is implemented, tests must cover its presence and keyboard operability.

## Open Questions
- Which routes or tabs should expose the diagnostics screen entry point?
- Do any analytics queries currently require a service filter that needs a backend default?
- Should refresh interval changes persist across sessions (local storage) or reset per visit?
