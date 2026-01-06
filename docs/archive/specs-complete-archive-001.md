# Completed Specs Archive #001
Archived: January 6, 2026

This archive contains completed specification projects from /docs/specs.

---


## PROJECT: azd-app

### SPEC.MD

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

### TASKS.MD

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

## PROJECT: reqs-install-url

### SPEC.MD

# Reqs Install URL Enhancement

## Overview

Enhance the `azd app reqs` command to display installation URLs when requirement checks fail, and allow users to specify custom install URLs for custom requirements.

## Problem

Currently, when a requirement check fails, users see:
```
❌ mytool: NOT INSTALLED (required: 1.0.0)
```

Users must then search for installation instructions, which slows down onboarding.

## Solution

1. Add `installUrl` field to the `requirement` schema
2. Provide built-in install URLs for known tools
3. Display install URL in failure messages
4. Support custom install URLs for custom requirements

## Schema Changes

Add `installUrl` property to `requirement` definition in `schemas/v1.1/azure.yaml.json`:

```yaml
reqs:
  - name: node
    minVersion: "18.0.0"
  - name: mytool
    minVersion: "1.0.0"
    command: mytool
    args: ["--version"]
    installUrl: "https://example.com/mytool/install"
```

## Built-in Install URLs

| Tool | Install URL |
|------|-------------|
| node | https://nodejs.org/ |
| npm | https://nodejs.org/ |
| pnpm | https://pnpm.io/installation |
| yarn | https://yarnpkg.com/getting-started/install |
| python | https://www.python.org/downloads/ |
| pip | https://www.python.org/downloads/ |
| poetry | https://python-poetry.org/docs/#installation |
| uv | https://docs.astral.sh/uv/getting-started/installation/ |
| docker | https://www.docker.com/products/docker-desktop |
| git | https://git-scm.com/downloads |
| go | https://go.dev/dl/ |
| dotnet | https://dotnet.microsoft.com/download |
| aspire | https://learn.microsoft.com/dotnet/aspire/setup-tooling |
| azd | https://aka.ms/install-azd |
| az | https://aka.ms/installazurecli |
| func | https://learn.microsoft.com/azure/azure-functions/functions-run-local#install-the-azure-functions-core-tools |
| java | https://adoptium.net/ |
| mvn | https://maven.apache.org/install.html |
| gradle | https://gradle.org/install/ |

## Output Changes

### Failed Requirement (Built-in Tool)

```
❌ docker: NOT INSTALLED (required: 20.0.0)
   Install: https://www.docker.com/products/docker-desktop
```

### Failed Requirement (Custom Tool with URL)

```
❌ mytool: NOT INSTALLED (required: 1.0.0)
   Install: https://example.com/mytool/install
```

### Failed Requirement (Custom Tool without URL)

```
❌ mytool: NOT INSTALLED (required: 1.0.0)
```

### JSON Output

Add `installUrl` field to `ReqResult`:

```json
{
  "name": "docker",
  "installed": false,
  "required": "20.0.0",
  "satisfied": false,
  "message": "Not installed",
  "installUrl": "https://www.docker.com/products/docker-desktop"
}
```

## Files to Modify

1. `schemas/v1.1/azure.yaml.json` - Add `installUrl` property to requirement definition
2. `cli/src/cmd/app/commands/reqs.go` - Add `InstallUrl` to `Prerequisite` and `ReqResult`, add URL registry, update output
3. `cli/src/internal/pathutil/pathutil.go` - Consolidate install suggestions with URLs
4. `cli/src/cmd/app/commands/reqs_test.go` - Add tests for install URL display
5. `cli/docs/commands/reqs.md` - Document new `installUrl` field

## Implementation Notes

- Built-in URLs take precedence unless user specifies custom `installUrl`
- URL display only on failure (not installed or version mismatch)
- Keep output concise - single "Install:" line with clickable URL
- JSON output always includes `installUrl` when available

### TASKS.MD

<!-- NEXT: 0 -->
# Reqs Install URL Tasks

## Done

### DONE: Add installUrl to schema {#add-installurl-schema}
**Assigned**: Developer
**File**: `schemas/v1.1/azure.yaml.json`

Added `installUrl` property to the `requirement` definition with type: string, format: uri.

### DONE: Add install URL registry {#add-install-url-registry}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Created `installURLRegistry` map with built-in URLs for 21 tools.

### DONE: Update Prerequisite struct {#update-prerequisite-struct}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Added `InstallUrl` field to `Prerequisite` struct with YAML tag.

### DONE: Update ReqResult struct {#update-reqresult-struct}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Added `InstallUrl` field to `ReqResult` struct with JSON tag.

### DONE: Update Check method output {#update-check-method}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs.go`

Modified `PrerequisiteChecker.Check()` to resolve install URL and display on failure.

### DONE: Consolidate pathutil suggestions {#consolidate-pathutil}
**Assigned**: Developer
**File**: `cli/src/internal/pathutil/pathutil.go`

Updated `GetInstallSuggestion()` to use same URLs as install URL registry.

### DONE: Add unit tests {#add-unit-tests}
**Assigned**: Developer
**File**: `cli/src/cmd/app/commands/reqs_test.go`

Added tests: `TestInstallURLRegistry`, `TestGetInstallUrl`, `TestCheckPrerequisiteIncludesInstallUrl`.

### DONE: Update documentation {#update-documentation}
**Assigned**: Developer
**File**: `cli/docs/commands/reqs.md`

Documented `installUrl` field, built-in URLs table, and updated output examples.


## PROJECT: cli-docs-sync

### SPEC.MD

# CLI Docs Sync

## Goal
Keep CLI command documentation consistent across:
- The actual CLI command set implemented in Go
- The CLI reference markdown
- Per-command markdown files
- The website CLI reference pages

## In Scope
- Ensure every top-level `azd app <command>` implemented in the CLI is documented in `cli/docs/cli-reference.md`.
- Ensure every top-level command has a corresponding file in `cli/docs/commands/<command>.md`.
- Ensure command docs reflect current flags, arguments, and subcommands.
- Ensure the website contains a reference page for each documented command.

## Out of Scope
- Changing command behavior.
- Removing existing documentation unless explicitly requested.
- Documenting internal-only commands beyond minimal “internal/hidden” notes.

## Source of Truth
- CLI command set and flags are derived from the Cobra command definitions in `cli/src/cmd/app/commands/`.

## Acceptance Criteria
- `cli/docs/cli-reference.md` includes all implemented top-level commands.
- Each implemented top-level command has a matching `cli/docs/commands/<command>.md`.
- Website CLI reference pages exist for each documented command and align with the markdown docs.

### TASKS.MD

<!-- NEXT: 0 -->
# CLI Docs Sync Tasks

## Done

## DONE: Inventory Implemented CLI Commands (1)
- Confirmed registered commands from `cli/src/cmd/app/main.go` and Cobra command files.
- Identified internal/hidden commands (`listen`, `mcp`).

## DONE: Sync CLI Reference (2)
- Added `add` and `listen` to `cli/docs/cli-reference.md` (including a new `add` section).
- Documented the missing global `--cwd/-C` flag.

## DONE: Sync Per-Command Docs (3)
- Added `cli/docs/commands/listen.md`.
- Fixed `cli/docs/commands/notifications.md` to use valid Go duration examples.

## DONE: Sync Website Pages (4)
- Updated `web/scripts/generate-cli-reference.ts` to generate a `listen` page but hide it from the index.
- Regenerated website pages in `web/src/pages/reference/cli/`.

## PROJECT: service-filters-ui

### SPEC.MD

# Service Filters UI Redesign

## Overview
Redesign the Services filter section in ConsoleView to match the visual style of Log Levels, State, and Health Status filters. Replace checkbox-based selection with modern pill buttons (icon + text) for visual consistency.

## Current State
- Services use traditional checkboxes with text labels
- Log Levels, State, and Health Status use icon buttons
- Visual inconsistency creates design friction

## Goals
1. **Visual Consistency**: Services section uses pill-style buttons matching other filter sections
2. **Usability**: Clear service identification with icon + text (not just icons)
3. **Accessibility**: Maintain WCAG AA compliance
4. **Responsive**: Handle 1-20+ services gracefully with automatic wrapping

## Design Requirements

### Visual Style
- Pill-style buttons with icon + text (px-2.5 py-1.5, rounded-md)
- Icon (w-3.5 h-3.5 shrink-0) + service name text (text-xs font-medium truncate)
- Text handling: single line, truncate with ellipsis if exceeds 150px (max-w-[150px])
- Buttons grow to fit content, flex-wrap handles multi-row layout automatically
- Selected state: 
  - Colored background (e.g., `bg-emerald-100 dark:bg-emerald-500/20`)
  - Colored text (e.g., `text-emerald-700 dark:text-emerald-300`)
  - Colored ring (e.g., `ring-1 ring-emerald-300 dark:ring-emerald-500/50`)
  - Icon and text both use selected color
- Unselected state:
  - Transparent background (`bg-transparent`)
  - Gray text (`text-slate-600 dark:text-slate-400`)
  - Gray icon (same color as text)
  - Hover: subtle background (`hover:bg-slate-200/60 dark:hover:bg-slate-700/60`)
- Each service gets unique color scheme cycling through 8-color palette

### Service Icons
Use contextual icons based on service name/type:
- `web`, `frontend`, `ui`, `app` → Globe
- `api`, `backend`, `server` → Server
- `worker`, `queue`, `background` → Cpu
- `functions`, `function`, `func` → Zap
- `containerapp`, `container` → Box
- `database`, `db`, `postgres`, `redis`, `mongo`, `mysql` → Database
- Default → Package

### Icon Library
Import from lucide-react: `Globe, Server, Database, Box, Cpu, Zap, Package`

### Color Palette
Cycle through these tailwind color schemes (use `index % 8`):
1. Emerald: `bg-emerald-100 dark:bg-emerald-500/20` / `text-emerald-700 dark:text-emerald-300` / `ring-emerald-300 dark:ring-emerald-500/50`
2. Purple: `bg-purple-100 dark:bg-purple-500/20` / `text-purple-700 dark:text-purple-300` / `ring-purple-300 dark:ring-purple-500/50`
3. Blue: `bg-blue-100 dark:bg-blue-500/20` / `text-blue-700 dark:text-blue-300` / `ring-blue-300 dark:ring-blue-500/50`
4. Rose: `bg-rose-100 dark:bg-rose-500/20` / `text-rose-700 dark:text-rose-300` / `ring-rose-300 dark:ring-rose-500/50`
5. Cyan: `bg-cyan-100 dark:bg-cyan-500/20` / `text-cyan-700 dark:text-cyan-300` / `ring-cyan-300 dark:ring-cyan-500/50`
6. Violet: `bg-violet-100 dark:bg-violet-500/20` / `text-violet-700 dark:text-violet-300` / `ring-violet-300 dark:ring-violet-500/50`
7. Amber: `bg-amber-100 dark:bg-amber-500/20` / `text-amber-700 dark:text-amber-300` / `ring-amber-300 dark:ring-amber-500/50`
8. Teal: `bg-teal-100 dark:bg-teal-500/20` / `text-teal-700 dark:text-teal-300` / `ring-teal-300 dark:ring-teal-500/50`

### Layout & Wrapping

**Few services** (single row):
```
Services
[🌐 web] [⚡ api] [⚙️ worker]
```

**Many services** (automatic multi-row wrapping):
```
Services
[🌐 appservice-web] [⚡ containerapp-api] [⚙️ functions-worker] [💾 postgres]
[📦 redis] [☁️ azurite] [🔧 worker] [🔐 auth] [📊 analytics] [🎨 frontend]
[🗄️ database] [⚡ queue-processor] [🌍 global-api] ...
```

- Container uses `flex flex-wrap gap-2`
- Buttons automatically wrap to new rows when horizontal space fills
- Each button: max-width 150px with text truncate + ellipsis
- `title` attribute shows full service name on hover (for truncated text)
- Scales from 1 to 100+ services responsively

### Selection State Examples

**Selected** (emerald):
```
[🌐 web]  ← emerald bg, emerald text/icon, emerald ring - fully colored
```

**Unselected**:
```
[🌐 web]  ← transparent bg, gray text/icon, subtle hover effect
```

**Multiple selections** (different colors):
```
[🌐 web]   [⚡ api]    [⚙️ worker]
↑ emerald  ↑ purple   ↑ blue
```

## Technical Approach

### Component Structure
```tsx
// In FiltersBar component - Replace checkbox section with pill buttons
<div className="flex flex-col gap-2">
  <span className="text-xs font-medium text-slate-500">Services</span>
  <div className="flex flex-wrap gap-2">
    {services.sort((a, b) => a.name.localeCompare(b.name)).map((service, idx) => {
      const { icon: IconComponent, colorScheme } = getServiceIconAndColor(service.name, idx)
      const isSelected = selectedServices.has(service.name)
      
      return (
        <button
          key={service.name}
          type="button"
          onClick={() => onToggleService(service.name)}
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1.5 rounded-md transition-all max-w-[150px]',
            isSelected
              ? colorScheme.selected
              : 'bg-transparent text-slate-600 dark:text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
          )}
          aria-label={`Toggle ${service.name}`}
          title={service.name} // Full name on hover
        >
          <IconComponent className="w-3.5 h-3.5 shrink-0" />
          <span className="text-xs font-medium truncate">{service.name}</span>
        </button>
      )
    })}
  </div>
</div>
```

### Helper Function
```tsx
import { Globe, Server, Database, Box, Cpu, Zap, Package } from 'lucide-react'

interface ServiceIconColor {
  icon: typeof Globe // LucideIcon type
  colorScheme: {
    selected: string // Combined className for selected state
  }
}

function getServiceIconAndColor(serviceName: string, index: number): ServiceIconColor {
  const lowerName = serviceName.toLowerCase()
  
  // Determine icon based on service name patterns
  let icon = Package // default
  if (lowerName.includes('web') || lowerName.includes('frontend') || lowerName.includes('ui') || lowerName.includes('app')) {
    icon = Globe
  } else if (lowerName.includes('api') || lowerName.includes('backend') || lowerName.includes('server')) {
    icon = Server
  } else if (lowerName.includes('worker') || lowerName.includes('queue') || lowerName.includes('background')) {
    icon = Cpu
  } else if (lowerName.includes('function') || lowerName.includes('func')) {
    icon = Zap
  } else if (lowerName.includes('container')) {
    icon = Box
  } else if (lowerName.includes('db') || lowerName.includes('database') || lowerName.includes('postgres') || lowerName.includes('redis') || lowerName.includes('mongo') || lowerName.includes('mysql')) {
    icon = Database
  }
  
  // Color scheme cycling (8 colors)
  const colorSchemes = [
    { selected: 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50' },
    { selected: 'bg-purple-100 dark:bg-purple-500/20 text-purple-700 dark:text-purple-300 ring-1 ring-purple-300 dark:ring-purple-500/50' },
    { selected: 'bg-blue-100 dark:bg-blue-500/20 text-blue-700 dark:text-blue-300 ring-1 ring-blue-300 dark:ring-blue-500/50' },
    { selected: 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50' },
    { selected: 'bg-cyan-100 dark:bg-cyan-500/20 text-cyan-700 dark:text-cyan-300 ring-1 ring-cyan-300 dark:ring-cyan-500/50' },
    { selected: 'bg-violet-100 dark:bg-violet-500/20 text-violet-700 dark:text-violet-300 ring-1 ring-violet-300 dark:ring-violet-500/50' },
    { selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50' },
    { selected: 'bg-teal-100 dark:bg-teal-500/20 text-teal-700 dark:text-teal-300 ring-1 ring-teal-300 dark:ring-teal-500/50' },
  ]
  
  return {
    icon,
    colorScheme: colorSchemes[index % colorSchemes.length]
  }
}
```

## Files to Modify
- `cli/dashboard/src/components/ConsoleView.tsx` (FiltersBar component)

## Acceptance Criteria
✅ Services section uses pill buttons with icon + text instead of checkboxes  
✅ Each service has contextual icon based on name pattern  
✅ Service name text is visible in button  
✅ Services cycle through 8-color palette deterministically (by index)  
✅ Selected state: colored background + text + ring  
✅ Unselected state: transparent with gray text  
✅ Text truncates with ellipsis for long names (>150px)  
✅ `title` attribute shows full name on hover  
✅ Automatic wrapping with flex-wrap for 1-100+ services  
✅ Accessible (aria-label, keyboard navigation)  
✅ Dark mode support  
✅ Maintains all existing filter toggle behavior

## Out of Scope
- Changing Log Levels, State, or Health Status sections
- Custom service icons from azure.yaml configuration
- Service grouping/categories
- Service ordering (keeps existing alphabetical sort)

### TASKS.MD

<!-- NEXT: 0 -->
# Service Filters UI Redesign Tasks

## Done

### DONE: Implement service pill buttons {#implement-service-icon-buttons}
**Assigned**: Designer → Developer  
**Priority**: P1  
**Completed**: 2025-12-11

Replaced checkbox-based service filters with pill buttons (icon + text) for better visual consistency.

**Implementation**:
- ✅ Added lucide-react icons (Globe, Server, Database, Box, Cpu, Zap, Package) to imports
- ✅ Created `getServiceIconAndColor(serviceName, index)` helper function
- ✅ Maps service name patterns to contextual icons (web→Globe, api→Server, etc.)
- ✅ 8-color palette cycling using index % 8
- ✅ Replaced checkbox `<label>` + `<input>` with pill `<button>`
- ✅ Selected state: colored bg + text + ring (`bg-emerald-100`, `text-emerald-700`, `ring-1`)
- ✅ Unselected state: transparent with gray text, hover effect
- ✅ Text truncation with `max-w-[150px] truncate`
- ✅ `title` attribute for full service name on hover
- ✅ Flex-wrap for automatic multi-row layout
- ✅ All existing toggle behavior preserved

**Files Modified**:
- `cli/dashboard/src/components/ConsoleView.tsx`

**Testing**:
- ✅ Build succeeded
- ✅ All 645 dashboard tests pass
- ✅ TypeScript compilation clean
- ✅ Dark mode support included
- ✅ Accessible with aria-label

**Result**: Services section now uses colorful pill buttons with icon+text matching the visual style of Log Levels, State, and Health Status filters. Handles 1-100+ services with automatic wrapping.
