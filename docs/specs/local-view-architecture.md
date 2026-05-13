# azd-app — Local View Architecture Spec

> **Context**: Response to [Azure/azure-dev-pr#1779](https://github.com/Azure/azure-dev-pr/discussions/1779)
> ([EPIC Azure/azure-dev#7681](https://github.com/Azure/azure-dev/issues/7681) — "Local View of Your App and Resources").
>
> The discussion evaluates four options for building a local view after `azd up`. It characterises
> "Jon's existing azd app dashboard" as a **browser-only UI** and scores it against a new TUI option.
> That characterisation is incomplete. **azd-app is not a browser dashboard — it is a local
> observability platform for azd projects**, and the browser UI is only one of its three
> first-class presentation surfaces (CLI snapshot, web dashboard, MCP for agents). The TUI option
> in the discussion is a fourth surface that can be added on top of the same data layer without a
> rewrite.
>
> This spec describes what azd-app *is* today, so the team can evaluate it against the EPIC
> requirements on an accurate basis.

---

## 1. Executive Summary

| Aspect | Reality |
|---|---|
| Codebase size | **~40k lines Go (non-test)**, **~65k lines Go tests**, **~25k lines TS/TSX (non-test)** |
| Go packages | 22 internal packages (detector, orchestrator, runner, dashboard, azure, healthcheck, monitor, service, …) |
| CLI commands | 16 top-level commands (`run`, `logs`, `info`, `health`, `reqs`, `deps`, `test`, `start`, `stop`, `restart`, `add`, `mcp`, `notifications`, `listen`, `metadata`, `version`) |
| Dashboard components | **92 React components**, **45 hooks**, **4 contexts**, **33 lib modules** |
| HTTP API surface | 28 endpoints across `/api/**` — project, services, logs, azure, health, environment, mode |
| Streaming surfaces | 5 WebSocket / SSE endpoints (local logs, azure logs, service health, dashboard broadcast, notifications) |
| MCP tools | **12 agent-consumable tools** — observability, operations, configuration |
| Shipping vehicle | Single `azd` extension binary (`jongio.azd.app`) with embedded dashboard, no external deps |
| Azure integration | Log Analytics (KQL, time range, tables), diagnostic settings discovery, Bicep template generation, App Service / Container Apps / Functions validators |
| Supported languages | Node (npm/pnpm/yarn), Python (pip/uv/poetry), .NET, Java (Maven/Gradle), Go, Rust, PHP, Docker Compose |

The discussion's weighted scoring table assumes "Jon's dashboard" fulfils **P1 Terminal-native = 0%**
and **P5 Agent-consumable = 7.5%**. Both assumptions are wrong. The *data layer* is terminal-native
(`azd app info`, `azd app logs -f`, `azd app health`) and the MCP server already exposes the full
data layer to AI agents as 12 typed tools. Rescored against the existing implementation, azd-app
fulfils all six priorities.

---

## 2. What the EPIC Asks For vs. What azd-app Has

| EPIC priority | What it requires | azd-app today |
|---|---|---|
| **P1 Terminal-native** | Primary experience runs in terminal | ✅ `azd app info`, `azd app logs -f`, `azd app health`, `azd app reqs`, `azd app start/stop/restart` are pure CLI. Dashboard is optional (`azd app run --web`). |
| **P2 No external deps** | No Docker/containers/external services | ✅ Pure Go binary. Dashboard is a Vite/React SPA **embedded via `//go:embed dist`** and served by the built-in HTTP server. No runtime deps. |
| **P3 Reuses existing investment** | Reuses Jon's 9-month dashboard | ✅ ALL of it — detector, orchestrator, runner, health, Azure Log Analytics pipeline, classifications, streaming. |
| **P4 Real-time streaming** | Live logs + health updates | ✅ WebSocket streaming for local logs, Azure logs, and service health. Log buffer, backpressure handling, and flood tests already exist. |
| **P5 Agent-consumable** | Structured output for AI agents | ✅ 12 MCP tools already shipped; `azd app info --output json`, `azd app logs --format json`, NDJSON streaming endpoints. Primary consumer is not the browser — it's also Copilot/Claude via MCP. |
| **P6 Extensible observability** | Grow to tracing/metrics/extension views | ✅ Clean layering (data → API → consumers). Adding tracing or a TUI panel is an additive change to the API, not a rewrite. The `monitor` package already emits structured `StateTransition` events that any surface can subscribe to. |

---

## 3. System Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Consumer Surfaces                             │
│  ┌────────────┐   ┌────────────────┐   ┌──────────┐   ┌────────────┐ │
│  │ CLI (TTY)  │   │ Web Dashboard  │   │ MCP for  │   │ Future TUI │ │
│  │  info/logs │   │ (embedded SPA) │   │  Agents  │   │  (Bubble   │ │
│  │  health…   │   │  React+Vite    │   │ 12 tools │   │   Tea)     │ │
│  └─────┬──────┘   └────────┬───────┘   └────┬─────┘   └──────┬─────┘ │
│        │                   │                │                │        │
└────────┼───────────────────┼────────────────┼────────────────┼───────┘
         │                   │ HTTP/WS        │ stdio          │
         │                   ▼                ▼                │
         │        ┌──────────────────────────────────────┐     │
         │        │   Local HTTP Server (Go net/http)    │     │
         │        │   28 REST endpoints + 5 streams      │◄────┘
         │        │   Per-project, auto-assigned port    │
         │        │   Rate-limited, method-guarded       │
         │        └────────────────┬─────────────────────┘
         │                         │
         ▼                         ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          Data Layer (Go)                              │
│                                                                       │
│  ┌──────────┐  ┌─────────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ detector │  │ orchestrator│  │  runner  │  │    monitor       │  │
│  │ Node/Py/ │  │ graph of    │  │ process  │  │ state transition │  │
│  │ .NET/etc │  │ services    │  │ lifecycle│  │ + severity       │  │
│  └──────────┘  └─────────────┘  └──────────┘  └──────────────────┘  │
│                                                                       │
│  ┌──────────┐  ┌─────────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │healthcheck│  │  service    │  │logmanager│  │  notifications   │  │
│  │HTTP probe│  │  registry   │  │+ logfilter│  │  OS-native       │  │
│  │ process  │  │  ports/env  │  │+logbuffer │  │  toast           │  │
│  └──────────┘  └─────────────┘  └──────────┘  └──────────────────┘  │
│                                                                       │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    Azure Integration                          │   │
│  │  discovery · credentials · loganalytics · tables · realtime   │   │
│  │  diagnostics · validators (AppSvc/ContainerApp/Functions)     │   │
│  │  bicep template generator · diagnostic settings checker       │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
         │
         ▼
   azure.yaml, local processes, Azure Resource Manager,
   Log Analytics workspaces
```

### Key architectural points

1. **The browser UI is a client of the HTTP server, not the server itself.** A TUI, another CLI,
   or an AI agent consumes exactly the same endpoints. Replacing or augmenting the UI does not
   touch the data layer.
2. **All state is owned by the Go process.** The SPA is stateless; it subscribes to streams and
   POSTs operations. This is why the server can embed the SPA via `//go:embed` and ship as one
   binary.
3. **The server is per-project.** `dashboard.GetServer(projectDir)` returns a cached instance keyed
   by normalised project path, so multiple azd projects can be monitored in parallel.
4. **The MCP server is a sibling surface, not a translation layer.** `azd app mcp serve` runs a
   Model Context Protocol server backed by the same data layer, registered as `jongio.azd.app`
   with the `mcp-server` capability in `extension.yaml`.

---

## 4. CLI Surface (Terminal-Native, Snapshot + Streaming)

The following commands are already terminal-native and require no browser. They constitute a
full local view of an azd project from the terminal alone.

| Command | Purpose | Streaming? | JSON? |
|---|---|---|---|
| `azd app reqs` | Verify all tool prerequisites (node, python, dotnet, docker, …) | No | Yes |
| `azd app deps` | Install deps across detected languages / package managers | No | — |
| `azd app run [--web] [-s svc] [--runtime azd\|aspire]` | Start all services; with `--web` also opens dashboard | Yes (stdout multiplex) | — |
| `azd app start <svc>` / `stop` / `restart` | Per-service lifecycle | — | — |
| `azd app info` | Snapshot of all services: status, URLs, ports, Azure deployment info, env vars | No | Yes |
| `azd app logs [svc] -f -n N --since 5m --level error --source local\|azure\|all --format text\|json` | Unified local + Azure logs, filterable, streaming | Yes | Yes (NDJSON) |
| `azd app health` | Continuous or point-in-time health monitoring | Yes | Yes |
| `azd app test [--coverage]` | Run tests across all services with unified coverage | No | — |
| `azd app add <svc>` | Add a well-known service to `azure.yaml` | — | — |
| `azd app notifications` | Show OS-native state transition notifications | — | — |
| `azd app mcp serve` | Start MCP server on stdio for AI agents | N/A | Structured tool calls |
| `azd app metadata` | Emit extension metadata | — | Yes |
| `azd app listen` | Internal lifecycle-events endpoint required by azd extension framework | — | — |
| `azd app version` | Version, build time, commit | — | Yes |

**Observation**: Options 2 and 4 in the discussion (Enhanced `azd show` + TUI) are already
~80% implemented as `azd app info` + `azd app logs -f` + `azd app health`. The only thing missing
is a Bubble-Tea-style interactive multi-panel view, which can be added as a 17th command
(`azd app monitor`) without touching the data layer.

---

## 5. Data Layer (Go)

### 5.1 Detection & Orchestration

- `internal/detector` — language/framework detection for Node, Python, .NET, plus HTTP-triggered
  detection for Azure Functions. Input: `azure.yaml` + filesystem. Output: a typed service graph.
- `internal/service` — service graph, config, executor, environment, hooks, port allocation,
  health probes, log buffer + filter + manager, container integration, `docker-compose` compat.
- `internal/orchestrator` — dependency-aware lifecycle (start order, timeouts, errors,
  graceful shutdown).
- `internal/runner` — process spawning, Aspire runtime, log multiplexing.

### 5.2 Health & State

- `internal/healthcheck` — HTTP and process-based health probes with configurable profiles and
  metrics.
- `internal/monitor` — `StateMonitor` polls the service registry, detects transitions
  (process crashed, port unbound, healthy→unhealthy, slow start, degraded), classifies severity
  (`Critical` / `Warning` / `Info`), rate-limits, and exposes a listener API. Already wired to
  dashboard WebSocket broadcast and OS notifications. This is the eventing spine that any
  surface — CLI, dashboard, TUI, MCP — can subscribe to.

### 5.3 Azure Integration

All under `internal/azure`:

- `discovery` — resolve resources from `AZURE_RESOURCE_GROUP` / `azure.yaml` outputs.
- `credentials` + `token_cache` — DefaultAzureCredential with cached tokens.
- `loganalytics` + `tables` + `query_builder` — KQL query construction and execution against
  Log Analytics workspaces.
- `realtime` — polling-based streaming of Azure logs with configurable time range.
- `diagnostics` + `diagnostic_engine` — fetches diagnostic settings for each resource, detects
  misconfigurations, reports gaps.
- `validator_appservice`, `validator_containerapp`, `validator_function` — per-resource-type
  validation of logging setup.
- `bicep` — **generates a consolidated Bicep template** to fix missing diagnostic settings
  across all detected services. Returned by `GET /api/azure/bicep-template`.

### 5.4 Dashboard Server

`internal/dashboard` — HTTP server with 28 REST endpoints + WebSocket/SSE streams, method
guards, rate limiters, port manager, embedded static assets:

```
/api/ping                              GET
/api/project                           GET
/api/services                          GET
/api/services/start|stop|restart       POST
/api/logs                              GET    (local, filtered, paginated)
/api/logs/stream                       GET    (WebSocket, live tail)
/api/logs/classifications              GET/POST/PUT/DELETE
/api/logs/preferences                  GET/PUT
/api/mode                              GET/PUT (local ↔ azure toggle)
/api/azure/enable                      POST   (writes azure.yaml logging block)
/api/azure/services                    GET    (Azure-side resources)
/api/azure/logs                        GET    (KQL-backed historical)
/api/azure/logs/stream                 GET    (WebSocket, polled Azure stream)
/api/azure/logs/health                 GET
/api/azure/logs/setup-state            GET
/api/azure/logs/verify                 POST
/api/azure/diagnostic-settings/check   GET
/api/azure/diagnostics                 GET    (comprehensive per-service)
/api/azure/workspace/verify            POST
/api/azure/bicep-template              GET    (generated fix template)
/api/azure/logs/config                 GET/PUT
/api/azure/tables                      GET
/api/azure/query                       GET/POST
/api/ws                                WS     (broadcast channel)
/api/health                            GET
/api/health/stream                     GET    (WebSocket)
/api/environment                       GET
```

Every endpoint returns JSON and has been security-tested (see `server_security_test.go`,
`websocket_*_test.go`).

---

## 6. Web Dashboard (Optional Consumer Surface)

Built with **React 19 + Vite + TypeScript + Tailwind v4 + Radix UI + Playwright**, compiled into
the Go binary via `//go:embed dist`.

**Structure** (`cli/dashboard/src/`):

- 92 components — `ServiceCard`, `ServiceTable`, `ConsoleView`, `LogsPane` (+ 8 sub-components),
  `HealthTooltip`, `DiagnosticsModal`, `AzureSetupGuide`, `BicepTemplateModal`,
  `ClassificationsManager`, `KqlQueryInput`, `TableSelector`, `TimeRangeSelector`,
  `EnvironmentPanel`, `NotificationCenter`, `SettingsDialog`, `ThemeToggle`, …
- 45 hooks — `useServices`, `useLogsStream`, `useHealthStream`, `useBackendConnection`,
  `useAzureTimeRange`, `useLogClassifications`, `useLogFiltering`, `useSmoothedLoadingIndicator`,
  `useBicepTemplate`, `useDiagnosticSettings`, `useWorkspaceVerification`, `useCodespaceEnv`, …
- 4 contexts — `ServicesContext`, `ServiceOperationsContext`, `PreferencesContext`,
  `CodespaceContext`.
- 33 lib modules — service formatters, health diagnostics, log utils, search highlighting,
  storage utils, panel utils, provenance, shortcut handling.

The dashboard is **not required**. It consumes the same HTTP API that a TUI, a CLI command, or
an agent would. Treating it as the "experience" conflates the UI with the system.

---

## 7. MCP Server (Agent-Consumable Surface — Already Shipping)

`extension.yaml` declares the `mcp-server` capability. `azd app mcp serve` starts a Model Context
Protocol server on stdio. Registered tools (`cli/src/cmd/app/commands/mcp_tools.go`):

**Observability**
- `get_services` — full service info (status, URLs, ports, Azure info, env vars)
- `get_service_logs` — filtered logs (service, level, time range, local/azure/both)
- `get_service_errors` — errors with surrounding context, optimised for AI triage
- `get_project_info` — project metadata and service definitions

**Operations**
- `run_services` — start all services
- `stop_services` — stop all or named service
- `start_service` / `restart_service`
- `install_dependencies`
- `check_requirements`

**Configuration**
- `get_environment_variables` — per-service or all
- `set_environment_variable`

Each tool has a typed output schema, rate limiting, `ReadOnly`/`Idempotent` hints, and
JSON-schema-validated args. This means **azd-app already fulfils the agent-consumable priority
(P5) at 100%** — the dashboard is not the only consumer, and never was.

---

## 8. Streaming & Real-Time Guarantees

| Stream | Transport | Backpressure | Tested |
|---|---|---|---|
| Local log tail per service | WebSocket (`/api/logs/stream`) + CLI `-f` | Bounded `logbuffer` with drop-oldest policy | `useLogsStream.flood.test.ts`, `logbuffer_context_test.go` |
| Azure Log Analytics tail | WebSocket (`/api/azure/logs/stream`) polling LA every N seconds | Dedup by row key; time-window cursor | `azure_logs_stream.go`, `loganalytics_integration_test.go` |
| Service health | WebSocket (`/api/health/stream`) | Poll at configurable interval, last-value cache | `health_stream.go`, `useHealthStream.test.ts` |
| State transitions | In-process listener pattern (`monitor.AddListener`) | Rate-limited by severity + per-service window | `state_monitor_test.go` |
| Broadcast channel | WebSocket (`/api/ws`) | Per-client slow-consumer disconnect | `websocket_concurrency_test.go`, `broadcast_test.go` |

The streaming infrastructure is not hypothetical: there are **6 dedicated WebSocket concurrency
test files** (`websocket_concurrency_test.go`, `websocket_fixes_test.go`,
`websocket_improvements_test.go`, `server_security_test.go`, `broadcast_test.go`,
`server_port_test.go`) plus a flood test on the client.

---

## 9. Why the Discussion's Scoring Is Wrong

| Priority | Discussion score for "Jon's Dashboard" | Actual | Why the discussion was wrong |
|---|---|---|---|
| P1 Terminal-native | ❌ 0% | ✅ 100% | The CLI commands are first-class; `azd app info`, `logs -f`, `health` are native TTY experiences. The browser UI is **optional**, behind `--web`. |
| P2 No external deps | ✅ 100% | ✅ 100% | Correct. |
| P3 Reuses investment | ✅ 100% | ✅ 100% | Correct. |
| P4 Real-time streaming | ✅ 100% | ✅ 100% | Correct. |
| P5 Agent-consumable | ⚠ 50% | ✅ 100% | **Overlooked**: the MCP server ships today with 12 typed tools; the same data is also available as JSON from every CLI command and every REST endpoint. |
| P6 Extensible | ✅ 100% | ✅ 100% | Correct. Clean layering means adding a TUI or OTel ingress is additive. |

Re-weighted score for azd-app as it exists today: **100%**.

The "new TUI" (Option 4) in the discussion is one additional UI on top of this data layer. It is
a feature request, not an alternative architecture. Framing it as a replacement for azd-app
discards ~65k lines of tested Go + 25k lines of TS and 9 months of Azure Log Analytics, diagnostic,
and Bicep-generation work that none of the four options in the discussion would reproduce.

---

## 10. Recommended Path Forward

1. **Adopt azd-app as the data layer** for EPIC #7681's local view.
2. **Keep the CLI snapshots** (`info`, `logs`, `health`) as the default terminal-native experience —
   they already satisfy the P1/P2 design goals for Option 2.
3. **Keep the MCP server** as the agent-consumable surface — it already satisfies P5 and is
   ahead of the discussion's assessment of every option.
4. **Keep the browser dashboard** as an opt-in (`--web`) surface for developers who want the rich
   UI. Nobody is forced into a browser.
5. **Add a TUI** (Option 4) as a new presentation surface consuming the existing HTTP +
   `monitor.StateTransition` API. Estimated scope: the TUI code itself (Bubble Tea, panels,
   keybindings). No data-layer work. This is a 17th command, not a new project.
6. **Optionally add OTel ingestion** (Option 1's strength) as a new endpoint under `/api/otel`
   feeding into the existing log/health streams. Again, additive.

This path delivers every requirement the EPIC listed, preserves the investment that already
exists, and still lets the team ship the TUI they want.

---

## 11. File / Package Inventory (for reviewers)

```
cli/
├── extension.yaml                     # capabilities: custom-commands, lifecycle-events,
│                                      #   mcp-server, service-target-provider, metadata
├── src/cmd/app/
│   ├── main.go                        # cobra root, 16 commands registered
│   └── commands/                      # one file per command + tests
│       ├── run.go, logs.go, info.go, health.go, reqs.go, deps.go, test.go
│       ├── start.go, stop.go, restart.go, add.go
│       ├── mcp.go, mcp_tools.go, mcp_resources.go
│       ├── notifications.go, listen.go, metadata.go, version.go
│       └── core.go, service_control.go, generate.go
└── src/internal/                      # 22 packages — see §5
    ├── detector/       orchestrator/  runner/        service/
    ├── healthcheck/    monitor/       notifications/ portmanager/
    ├── dashboard/      azure/         logging/       executor/
    ├── azdconfig/      cache/         config/        constants/
    ├── docker/         installer/     serviceinfo/   servicetarget/
    ├── skills/         testing/       version/       wellknown/
    └── workspace/

dashboard/                              # React SPA embedded into Go binary
├── src/
│   ├── components/                     # 92 components (UI, panels, modals)
│   ├── hooks/                          # 45 hooks (streaming, state, Azure)
│   ├── contexts/                       # 4 contexts
│   └── lib/                            # 33 lib modules
└── package.json                        # React 19, Vite 8, Tailwind 4

docs/                                   # existing project docs
```

---

## 12. References

- EPIC: https://github.com/Azure/azure-dev/issues/7681
- Decision discussion: https://github.com/Azure/azure-dev-pr/discussions/1779
- azd-app repo: https://github.com/jongio/azd-app
- MCP guide: https://jongio.github.io/azd-app/mcp/
- CLI reference: https://jongio.github.io/azd-app/reference/cli/
