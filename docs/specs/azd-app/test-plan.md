# azd-app logs testing plan

## Objectives
- Validate dashboard logs UX changes: removed services dropdown, refreshed timeframe presets (no custom 1h), bounded refresh interval control, diagnostics entry, and local-service override.
- Prevent regressions in log querying, diagnostics loading, and service health visuals across Azure and local modes.
- Raise confidence through layered coverage (unit, integration, e2e) with clear owners and gates.

## Scope
- Dashboard UI: toolbar controls, FiltersBar pills with health colors, LogsPane countdown/refresh logic, diagnostics button visibility.
- Backend dashboard handlers: azure_logs defaults, ServiceInfo host overrides, diagnostics data path, refresh interval bounds enforcement.
- Modes: Azure mode, local mode, and mixed configurations with host=local services.
- Out of scope: authentication flows, non-log dashboard panels, analytics schema changes beyond defaults.

## Environments
- Local dev: cli/dashboard pnpm test -- --run; Playwright using existing dev server per repo conventions.
- Integration: mage build/test; existing azure-logs-test project; azure.yaml-driven azd app run where required.
- Browsers: Chromium (required), validate in WebKit/Firefox if Playwright matrix already configured.

## Coverage and ownership
- Unit (Developer): Vitest in cli/dashboard for refresh clamping and persistence, diagnostics button visibility (Azure mode only), effectiveLogMode override for host=local, optional timeRange handling in LogsPane, health-based pill color states.
- Integration (Developer): Go tests around azure_logs handlers for missing services filter defaults, ServiceInfo host=local override, diagnostics data load path, refresh interval bounds enforcement.
- E2E (Tester): Playwright flows validating toolbar controls (no services dropdown, no 1h preset), refresh interval bounds with validation/clamping, diagnostics button visibility and navigation, local-only services always showing local logs regardless of mode toggle, health pill colors matching health states.
- Reporting (DevOps/Tester): capture coverage for dashboard (Vitest) and Go packages; publish summary; set alerts on regressions.

## Test scenarios
- Toolbar controls
  - Services dropdown absent; no broken queries or params.
  - Timeframe presets exclude custom 1h; preset selection updates logs.
  - Refresh interval control clamps to 5s-5m; validation or clamping feedback shown; presets selectable.
  - Countdown reflects next refresh and hides when collapsed or paused.
- Diagnostics
  - Diagnostics button visible only in Azure mode; hidden in local-only mode.
  - Navigation opens diagnostics screen; data load succeeds; error state renders guidance.
- Local-service override
  - Services with host=local always fetch local logs even when global mode set to Azure.
  - Mixed services respect per-service host; Azure services still use Azure mode.
- Health-based pills
  - Pills show health colors and tooltips per health status; selected and unselected states meet defined color mapping.
  - Health wiring matches LogsPane indicators.

## Data and fixtures
- Use existing integration project azure-logs-test; extend fixtures for host=local services and health states.
- Provide sample diagnostics responses for success and error states.
- Ensure time-range defaults match backend expectations when services filter absent.

## Tooling and commands
- mage build; mage test (backend)
- pnpm test -- --run (dashboard unit)
- pnpm exec playwright test (dashboard e2e)

## Exit criteria
- New unit and integration tests added for each critical scenario above.
- Playwright suite covers toolbar controls, diagnostics, local-service override, and health pills; passes in CI matrix.
- Coverage does not regress; any decreases justified and documented.
- All CI pipelines green; manual QA not required unless new failures emerge.

## Open items
- Decide whether refresh interval persistence is session-only or stored; align tests once finalized.
- Confirm which routes expose diagnostics entry; add e2e coverage accordingly.
