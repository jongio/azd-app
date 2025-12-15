# azd-app Archive #003
Archived: 2025-01-21

## Completed Tasks

### 8 Document findings and recommended fixes (security, logic, types, errors, tests, perf)
- **CRITICAL Fixes (blocking user experience):**
  - **(DONE)** CLI Azure-not-configured error referenced obsolete `logs.azure.enabled` config → Updated to `logs.analytics` with example structure (fixed in TODO 5).
  - **(DONE)** Ctrl+Shift+M mode switch shortcut bypassed backend `/api/mode` API, causing UI/backend desync → Fixed to call API and update global mode state (fixed in TODO 3).
  - **(DONE)** Unified LogsView did not respect Azure time range presets or support Azure polling vs realtime → Added Azure mode support matching LogsPane behavior (fixed in TODO 3).
- **HIGH Priority (degraded UX, should fix before ship):**
  - **Go Azure logs standalone:** Service name not propagated from CLI to Log Analytics queries, so output labels depend on Azure resource fields and may not align with `--service` filters → Propagate logical service name through query context or map back from azure.yaml to avoid mislabeling (affects CLI `azd app logs --source azure` when dashboard absent).
  - **Go Azure logs standalone:** Fetch swallows per-resource query failures and returns success with empty results when all queries fail (only warns) → Bubble an error if every resource type fails and include actionable guidance when zero rows returned (affects CLI Azure logs error messaging).
  - **Go Azure logs standalone streaming:** Writes `[HH:MM:SS] Last polled` to stderr on every poll even outside debug, spamming CLI output → Gate behind debug flag or remove (affects CLI `azd app logs --source azure --follow` readability).
  - **Go Azure logs standalone streaming:** Always starts with 24h window regardless of requested tail/since, potentially expensive and ignoring user intent → Align polling window with CLI tail/since or make configurable (affects CLI `azd app logs --source azure --since 15m` behavior).
- **MEDIUM Priority (polish, nice to have):**
  - **(DONE)** Docs showed unsupported `--start`/`--end` CLI flags for Azure logs → Removed from docs (fixed in TODO 5).
  - **(DONE)** Inconsistent ingestion delay wording across docs/CLI → Standardized to `1-5 minute` range (fixed in TODO 5).
  - **(DONE)** LogsPane error states did not surface fetch/WS errors in empty-state UI → Added error display (fixed in TODO 3).
  - **(DONE)** LogsPane per-line copy buttons not keyboard accessible → Made visible on focus (fixed in TODO 3).
  - **Test Gap 1 (minor):** No vitest unit test confirming only 4 Azure presets (15m, 30m, 6h, 24h) available and 1h excluded → Add unit test for time range selector if time allows (E2E covers this).
  - **Test Gap 2 (low priority):** No test covering `--source=all` when only local or Azure is available → May produce confusing output (document or add test).
  - **Test Gap 3 (minor):** No E2E test toggling Azure presets and verifying fetch URL `since=<preset>` updates → Add E2E test if time allows (unit tests cover this).
- **LOW Priority (deferred/informational):**
  - LogsPane render cost is per-line `stripEmbeddedTimestamp` + `convertAnsiToHtml` on each render → Acceptable at current limits (1000 lines) but worth monitoring if limits increase.
  - Custom KQL and historical time range UI removed; specs/design docs still reference them → User-facing docs cleaned up; spec/design files are historical reference, no action needed.
- **Recommended Fix Owners:**
  - Go Azure logs backend (HIGH priority items) → @developer (standalone service name propagation, error bubbling, streaming stderr/window fixes).
  - Test gaps (MEDIUM priority) → @tester (vitest unit test for preset exclusion, E2E test for preset toggling, `--source=all` edge case).

### 7 Verify key commands/tests locally (dashboard vitest focused suites; Go unit/integration as applicable)
- **Dashboard vitest:** Ran `pnpm vitest run src/components/logspane.test.tsx src/components/consoleview.test.tsx` → **24/24 passed** (9 LogsPane tests, 15 ConsoleView tests).
- **Go logs executor:** Ran `go test -v ./src/cmd/app/commands -run "TestLogs|TestAzure"` → **All tests passed** (logs constants, command structure, executor parse/validate/execute, Azure standalone fallback, follow logs via dashboard/in-memory, orchestration).
- **Go Azure logs backend:** Ran `go test -v ./src/internal/dashboard -run "Azure"` → **All tests passed** (handleAzureLogs defaults/bounds, service filter passthrough, error mapping, health status checks).
- **Result:** All existing tests for Azure logs behaviors pass cleanly; no regressions introduced by recent changes.

### 6 Review tests coverage and gaps: vitest suites, Go unit/integration, E2E logs UX
- **Dashboard vitest (37 files, key coverage validated):**
  - `logspane.test.tsx` covers Azure preset defaults (15m), mode switching, deduplication/timestamp stripping, local vs Azure fetch separation, loading/empty states.
  - `consoleview.test.tsx` covers mode toggle behavior and passing `logMode` prop correctly to panes.
  - **Gap 1 (minor):** No explicit test confirming that only 4 presets (15m, 30m, 6h, 24h) are available and 1h is excluded in the Azure time range selector. E2E covers this but unit test would catch selector regressions early.
- **Go unit tests (50+ Azure logs references, key coverage validated):**
  - `azure_logs_test.go` covers `/api/azure/logs` defaults (1h, 10k tail cap), service filter passthrough, error code mapping (AUTH_REQUIRED → 401), health endpoint checks.
  - `logs_executor_test.go` covers standalone fallback when dashboard is absent (`TestLogsExecutor_AzureStandaloneFallback`), `--source azure` parsing, service filter validation.
  - **Gap 2 (low priority):** No test covering scenario when user requests `--source=all` (both local + Azure simultaneously) and only one is available; executor may produce confusing output.
- **E2E tests (9 Playwright specs, key coverage validated):**
  - `logs-ux.spec.ts` covers Azure preset exclusion (1h absent, 30m present), refresh interval bounds (5s min, 5m max), diagnostics button visibility in Azure mode, local-only services not fetching Azure logs in global Azure mode.
  - `console.spec.ts` covers basic console load, view mode switching, filtering controls.
  - **Gap 3 (minor):** No E2E test covering user toggling Azure presets and verifying that the fetch URL updates with `since=<preset>` correctly. This is covered in unit tests but E2E would catch integration regressions.
- **Coverage for removed features (custom KQL/time ranges):**
  - No legacy tests exist for custom KQL UI or historical time range picker; they were removed before tests were added.
  - Docs still reference custom KQL in spec/design files but user-facing docs have been cleaned up (no action needed for test coverage).
- **Recommendation:** Prioritize Gap 1 (unit test for preset exclusion) if time allows; Gaps 2 and 3 are low priority given existing coverage depth.

### 5 Review CLI surface/docs alignment for new flags (--source, --cwd, --restart-containers) and Azure logs usage
- Confirmed CLI flag wiring:
  - `--cwd/-C` is a persistent root flag (changes working dir before command execution).
  - `azd app run --restart-containers` exists and is documented in run docs.
  - `azd app logs --source` defaults to `local` and supports `local|azure|all`.
- Fixed mismatches:
  - Updated CLI Azure-not-configured guidance to reference `logs.analytics` (was stale `logs.azure.enabled`).
  - Removed unsupported `azd app logs --start/--end` examples from docs.
  - Standardized ingestion-delay wording in user-facing docs/CLI to `1-5 minute` range.

### 4 Review performance in LogsPane/Log panels: memoization, tail limits, rendering churn
- Confirmed `MAX_LOGS_IN_MEMORY=1000` cap and `tail=500` initial fetch; no virtualization but bounded list size.
- Primary render cost is per-line `stripEmbeddedTimestamp` + `convertAnsiToHtml` on each render; acceptable at current limits but worth watching if limits increase.
- Avoided array-index-only keys in `LogsView` rows to reduce reconciliation churn.

### 3 Review dashboard Azure logs UI: mode switching, time ranges, dedup/strip, copy formatting, loading/error/empty states, accessibility
- Fixed Ctrl+Shift+M shortcut to switch modes via `/api/mode` (keeps backend + UI in sync).
- Unified LogsView now respects Azure `since` presets and supports Azure polling vs realtime WS (matches LogsPane behavior).
- LogsPane now surfaces fetch/WS errors in the empty-state and makes per-line copy buttons visible on keyboard focus.
- Ran focused dashboard tests: `pnpm vitest run src/components/logspane.test.tsx src/components/LogsView.test.tsx src/components/LogsPane.footer.test.tsx src/components/LogsPane.header.test.tsx`.

### 2 Review Azure logs backend (Go): auth/token cache, discovery, Log Analytics queries, realtime/polling, paging/tail limits, error handling
- Query paths use empty service name when calling Log Analytics standalone, so output service labels depend on Azure resource fields and may not align with CLI `--service` filters; propagate logical service name or map back from azure.yaml to avoid mislabeling.
- Standalone fetch swallows per-resource query failures and returns success with empty results when all queries fail (only warns), leaving users without actionable errors; bubble an error if every resource type fails and include guidance when zero rows are returned.
- Standalone streaming writes `[HH:MM:SS] Last polled` to stderr on every poll even outside debug, which will spam CLI output; gate behind debug or remove.
- Streaming always starts with a 24h window regardless of requested tail/since, potentially expensive and ignoring user intent; align polling window with CLI tail/since or make configurable.

### 1 Capture summary diff vs main for azlogs scope and entry points
- Entry points touched: dashboard LogsPane (Azure mode/time range presets and fetch URLs), LogConfigPanel (custom KQL removed), component index exports, logspane tests, Go logs command (Azure standalone fallback + defaults), docs/dev/todo deferral note.
- Major changes: Azure time range limited to presets with `since` param only; custom KQL and historical range UI removed/deferred; CLI logs command now supports standalone Azure fetch/stream when dashboard absent and defaults source to local; local logs now error when dashboard missing; new Azure log level mapping and fallback warnings; tests updated for presets and standalone fallback.
- Risks to probe: loss of custom ranges/KQL, Azure auth/scope/paging in fallback, divergence between dashboard vs standalone paths, Azure error messaging and time range suggestion coverage.

### 0 Review plan and prioritize high-risk areas (Go Azure logs backend, dashboard UI)
- Priorities: 1) Azure logs backend (auth, standalone fallback, streaming/polling, tail/paging, errors), 2) Dashboard Azure logs UX (mode/time ranges, dedup/strip, copy/copy formatting, loading/error/empty states, a11y), 3) Performance in LogsPane/log rows (render churn, memoization), 4) CLI surface/docs alignment and test coverage/verification.
- Context: Recent diffs removed custom KQL/time-range UI and added CLI Azure standalone fallback; time range is preset-only. Focus on regressions and missing behaviors around Azure flows.
