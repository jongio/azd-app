---
status: accepted
created: 2025-01-09
updated: 2025-01-09
category: architecture
deciders: jongio
---

# ADR 0001: Connect-RPC as the unified transport for azd-app

## Status

Accepted (PR 1 of 4 — foundation only, no behavior change).

## Context

`azd-app` exposes the same domain operations to four consumers:

1. **Dashboard SPA** (`cli/dashboard/`) — browser, fetches via REST + EventSource.
2. **CLI (cobra commands)** — in-process Go calls today, but the inventory of "what can the dashboard do that the CLI can't" keeps growing.
3. **MCP server** — re-implements business logic to expose tools, drifting from the dashboard surface.
4. **Future TUI / external automation** — would require a third hand-rolled client.

The current REST + WebSocket layer in `cli/src/internal/dashboard/` (28 endpoints + 5 streams) is hand-marshalled JSON with no shared schema. Each new endpoint requires:

- Hand-written Go handler (validation, marshalling, error mapping).
- Hand-written TypeScript fetcher + types in the dashboard.
- A separate MCP tool wrapper that re-derives the contract.
- No compile-time guarantee that any of the three agree.

We need a single, schema-first contract that all consumers share, that works in browsers (no raw HTTP/2 trailer requirement), and that supports server-streaming for the five long-lived feeds documented in `docs/specs/local-view-architecture.md` §8.

## Decision

Adopt **Connect-RPC** as the unified transport, with the following stack pinned:

| Side | Package | Version | Role |
|---|---|---|---|
| Go | `connectrpc.com/connect` | v1.19.1 | Server + in-process client runtime |
| Go | `google.golang.org/protobuf` | (transitive) | Message runtime |
| Tooling | `buf` | v1.68.2 | Lint, build, breaking-change detection |
| Tooling | `protoc-gen-go` | v1.36.11 | Go message codegen |
| Tooling | `protoc-gen-connect-go` | v1.19.1 | Go service stub codegen |
| TS | `@bufbuild/protobuf` | 1.10.1 | TS message runtime |
| TS | `@bufbuild/protoc-gen-es` | 1.10.1 | TS message codegen |
| TS | `@connectrpc/protoc-gen-connect-es` | 1.7.0 | TS service stub codegen |
| TS | `@connectrpc/connect` | 1.7.0 | Browser/Node runtime |
| TS | `@connectrpc/connect-web` | 1.7.0 | `fetch`-based web transport |

**Why the v1 line on the JS/TS side?** v2 of `@bufbuild/protoc-gen-es` absorbed Connect codegen and removed `protoc-gen-connect-es`. The v2 single-plugin model is a forward direction we may adopt later, but PR 1 locks the v1 dual-plugin combo so the four PRs in this series ship against a frozen toolchain. Upgrading is a future ADR.

The schema lives at `proto/azdapp/v1/` and is generated into:

- Go: `cli/src/gen/proto/azdapp/v1/` (messages) and `.../azdappv1connect/` (service stubs).
- TS: `cli/dashboard/src/gen/proto/azdapp/v1/` (messages and stubs side-by-side, per Connect-ES convention).

Generated code is checked in. Codegen is invoked by `scripts/proto-gen.ps1` (and by mage in later PRs).

### Endpoint inventory: 28 REST routes → 31 RPCs across 8 services

The local-view spec counts 28 REST endpoints by URL path. The proto schema has 31 unary RPCs because three router endpoints multiplex on HTTP method:

| REST route | Methods | Resulting RPCs |
|---|---|---|
| `/api/logs/classifications` | GET, POST, DELETE | `GetClassifications`, `CreateClassification`, `DeleteClassification` |
| `/api/logs/preferences` | GET, POST | `GetPreferences`, `SavePreferences` |
| `/api/mode` | GET, POST | `GetMode`, `SetMode` |

Each method becomes a distinct RPC with its own request/response types. This is required by buf's `STANDARD` lint (`RPC_REQUEST_RESPONSE_UNIQUE`) and is the right shape: independent evolution per operation, no method-discriminated union.

Service split (8 services):

- `LifecycleService` — Ping, GetEnvironment, StreamBroadcast
- `ProjectService` — GetProject
- `ServicesService` — GetServices, Start/Stop/RestartService
- `LogsService` — GetLogs, StreamLocalLogs, classifications CRUD, preferences GET/SAVE
- `HealthService` — GetHealth, StreamHealth, StreamStateTransitions
- `ModeService` — GetMode, SetMode
- `AzureService` — 14 unary + StreamAzureLogs
- `BicepService` — GetBicepTemplate

### Stream back-pressure (locked in proto comments)

The five server-streaming RPCs each codify a back-pressure policy. These match what the existing implementation does today and what `local-view-architecture.md` §8 specifies:

| Stream | Policy | Rationale |
|---|---|---|
| `LogsService.StreamLocalLogs` | Drop-oldest, bounded ring | Matches existing `logbuffer`. Local logs are high-volume; latest matters most. |
| `AzureService.StreamAzureLogs` | Block producer with backoff | Azure Log Analytics queries are billed and rate-limited. Cannot drop. |
| `HealthService.StreamHealth` | Last-value-wins | Only the most recent snapshot is meaningful; intermediate states are noise. |
| `LifecycleService.StreamBroadcast` | Drop-oldest, disconnect slow consumer | UI hints are best-effort. Slow consumers shed load. |
| `HealthService.StreamStateTransitions` | Block producer, 256-event bounded buffer | CRITICAL state changes cannot drop. Producer is rate-limited at source. |

### Service interface extraction — deferred

PR 2 will wire Connect handlers in parallel with the existing REST handlers. The handlers will call the same underlying types the REST handlers call today:

- `azdconfig.ConfigClient` is already an interface ✓
- `service.LogManager`, `monitor.StateMonitor`, `azure.LogAnalyticsClient`, `azure.DiagnosticSettingsChecker` are concrete structs

Extracting interfaces around the concrete types is an orthogonal testability concern. It is **not required** for the transport swap. PR 2 calls concrete types directly (mirroring REST). A later PR (3 or post-4) extracts interfaces if the in-process MCP/cobra refactor needs them.

### Well-known type usage

Two responses use `google.protobuf.Struct` instead of typed messages, intentionally:

- `AzureService.GetSetupState` — the setup state shape is in flux as PR 2 firms up the Azure provider model. Locking it now would force a breaking change.
- `AzureService.GetComprehensiveDiagnostics` — aggregates heterogeneous probe results. A typed union is premature until the diagnostic catalog stabilizes.

Both are flagged in PR 2 to be promoted to typed messages once the shape settles.

## Alternatives considered

**Raw gRPC.** Browser-hostile. gRPC-Web requires a proxy, no native fetch path, no streaming-from-server without trailers. Connect speaks gRPC + gRPC-Web + Connect protocol on the same handler, so we keep gRPC compatibility for free and add a browser-friendly path.

**tRPC.** TypeScript-only. Forces us to keep a separate Go contract or generate Go from TS — the wrong direction for a Go-rooted project.

**OpenAPI + generated clients.** More codegen surface, weaker streaming story (SSE is a bolt-on, not first-class), and the typed contract lives in YAML/JSON instead of a real schema language. proto + buf gives us breaking-change detection, lint, and a single source of truth across four consumers.

**Status quo (hand-rolled REST/WebSocket).** This is what we're replacing. Drift across consumers is the explicit problem.

## Migration plan

| PR | Scope | Behavior change? |
|---|---|---|
| **PR 1 (this PR)** | proto schema, codegen, generated stubs, ADR | None — no handlers wired |
| PR 2 | Connect handlers mounted in parallel with existing REST. Dashboard reads via Connect-ES client. REST handlers untouched. | Dashboard reads via Connect; REST still works for legacy callers |
| PR 3 | MCP and cobra commands call Connect services in-process via `connect.Client` against the same handler set. | MCP/CLI converge on the proto contract |
| PR 4 | Delete REST handlers and the dashboard's REST fetchers. | REST surface removed |

## Consequences

**Positive:**

- One schema, four consumers. Compile-time agreement on every field.
- Streaming is first-class. Back-pressure policy lives next to the RPC definition.
- buf `breaking` checks catch incompatible schema changes before they ship.
- MCP and CLI stop re-implementing dashboard logic.

**Negative:**

- Toolchain footprint grows: buf + 4 codegen plugins. Mitigated by `scripts/proto-gen.ps1` and (PR 2) mage targets.
- Generated code is checked in. Reviewers must understand "do not edit" boundaries. Mitigated by directory naming (`gen/`) and codegen comments.
- Two transports coexist during PRs 2-3. Documented and time-boxed.

**Neutral:**

- v1 toolchain pin is intentional and reviewed. v2 migration is a future, deliberate change.

## References

- `docs/specs/local-view-architecture.md` (the spec being implemented; §8 documents stream characteristics)
- `proto/azdapp/v1/*.proto` (the schema; back-pressure policies in stream RPC comments)
- `buf.gen.yaml`, `proto/buf.yaml` (codegen + lint config)
- Connect-Go: <https://connectrpc.com/docs/go/getting-started>
- Connect-ES (v1): <https://github.com/connectrpc/connect-es/tree/v1.7.0>
