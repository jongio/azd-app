---
status: implemented
created: 2025-01-09
updated: 2025-01-10
category: architecture
deciders: jongio
---

# ADR 0001: Connect-RPC as the unified transport for azd-app

## Status

Implemented. All four PRs have landed: PR 1 (foundation/proto), PR 2 (Connect handlers mounted in parallel), Stage 3 (CLI/MCP converged on typed Connect client), Stage 3.5 (dashboard TS migrated to Connect-ES), and PR 4 (REST + WebSocket surface deleted). The dashboard, CLI cobra commands, and MCP tools all talk to the same Connect handlers. `handleCheckRequirements` remains a subprocess path; no `RequirementsService` exists in the proto yet, so the MCP `reqs` tool still shells out to `azd app reqs`; adding that service is a follow-up ADR.

## Context

`azd-app` exposes the same domain operations to four consumers:

1. **Dashboard SPA** (`cli/dashboard/`): browser, fetches via REST + EventSource.
2. **CLI (cobra commands)**: in-process Go calls today, but the inventory of "what can the dashboard do that the CLI can't" keeps growing.
3. **MCP server**: re-implements business logic to expose tools, drifting from the dashboard surface.
4. **Future TUI / external automation**: would require a third hand-rolled client.

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

- `LifecycleService`: Ping, GetEnvironment, StreamBroadcast
- `ProjectService`: GetProject
- `ServicesService`: GetServices, Start/Stop/RestartService
- `LogsService`: GetLogs, StreamLocalLogs, classifications CRUD, preferences GET/SAVE
- `HealthService`: GetHealth, StreamHealth, StreamStateTransitions
- `ModeService`: GetMode, SetMode
- `AzureService`: 14 unary + 1 streaming RPC (`StreamAzureLogs`); the stream takes a `bool realtime` flag and the server picks polling or realtime transparently. Responses are framed in a `oneof { LogEntry entry; StreamStatus status; AzureDroppedNotice dropped; }` envelope so clients see entries, mode/health transitions, and overflow notices on a single wire stream.
- `BicepService`: GetBicepTemplate

### Stream back-pressure (locked in proto comments)

The five server-streaming RPCs each codify a back-pressure policy. These match what the existing implementation does today and what `local-view-architecture.md` §8 specifies:

| Stream | Policy | Rationale |
|---|---|---|
| `LogsService.StreamLocalLogs` | Drop-oldest, bounded ring | Matches existing `logbuffer`. Local logs are high-volume; latest matters most. |
| `AzureService.StreamAzureLogs` (polling) | Block producer with backoff | Azure Log Analytics queries are billed and rate-limited. Cannot drop. |
| `AzureService.StreamAzureLogs` (realtime) | Drop-oldest, emit `AzureDroppedNotice` | Realtime push has no producer-side rate limit; matches the local-log ring policy. |
| `HealthService.StreamHealth` | Last-value-wins | Only the most recent snapshot is meaningful; intermediate states are noise. |
| `LifecycleService.StreamBroadcast` | Drop-oldest, disconnect slow consumer | UI hints are best-effort. Slow consumers shed load. |
| `HealthService.StreamStateTransitions` | Block producer, 256-event bounded buffer | CRITICAL state changes cannot drop. Producer is rate-limited at source. |

### Service interface extraction: deferred

PR 2 will wire Connect handlers in parallel with the existing REST handlers. The handlers will call the same underlying types the REST handlers call today:

- `azdconfig.ConfigClient` is already an interface ✓
- `service.LogManager`, `monitor.StateMonitor`, `azure.LogAnalyticsClient`, `azure.DiagnosticSettingsChecker` are concrete structs

Extracting interfaces around the concrete types is an orthogonal testability concern. It is **not required** for the transport swap. PR 2 calls concrete types directly (mirroring REST). A later PR (3 or post-4) extracts interfaces if the MCP/cobra clients need them; PR 3 adds a typed Connect client that talks to the dashboard over localhost HTTP, so interface extraction only matters for unit tests that want to stand in a fake handler.

### Struct usage inventory

Two responses use `google.protobuf.Struct` instead of typed messages, intentionally. Every other response - including all 14 typed Azure unary RPCs and the streaming envelope - is fully typed.

| RPC | Reason for `Struct` | Promotion trigger |
|---|---|---|
| `AzureService.GetAzureSetupState` | The setup state aggregates ~12 sub-objects (Workspace, Authentication, Services, Issues, NextSteps, ...) whose shapes are still drifting as the Azure provider model firms up. A typed message would force a breaking proto change every time a probe is added. | When the shape goes a full release cycle without churn. |
| `AzureService.GetAzureDiagnostics` | Aggregates heterogeneous probe results into a single response keyed by probe name. The diagnostic catalog is still expanding (workspace, table, retention, RBAC, query-permission probes have all landed in the last quarter). | When the probe catalog stabilizes. |

Both are tracked in the migration plan and revisited every release. Adding a third Struct-typed response requires an ADR amendment.

### AzureService proto rewrite (Stage 2)

The original AzureService proto was generated from a schema sketch and drifted from the legacy REST/WebSocket handlers. Stage 2 starts with a one-shot rewrite that aligns the proto with what the legacy Go handlers (`cli/src/internal/dashboard/azure_*.go`) actually return today, so subsequent commits can wire handlers without amending the contract:

- `QueryAzureLogs` / `SaveAzureQuery` renamed to `GetServiceQuery` / `SaveServiceQuery` - the legacy handler stores a per-service KQL string, not a saved-query library.
- `GetAzureServices` response shrinks to `repeated string services` - the legacy endpoint returns service names only.
- `EnableAzureLogging` request loses `service_names` - the legacy handler takes no body.
- `GetAzureLogConfig` / `SaveAzureLogConfig` add an `AzureLogConfigMode` enum (`UNSPECIFIED` / `TABLES` / `CUSTOM`) plus `tables[]`, `query`, and `resource_type` fields - the legacy config carries all four.
- `ListAzureTables` response gains `recommended[]`, `workspace`, and `categories[]` alongside `tables[]` - the legacy handler returns all four.
- `CheckDiagnosticSettings` response keys per-service results by name (`map<string, DiagnosticSettingsResult>`) and adds the workspace ID at the top level.
- `VerifyWorkspace` request gains `services` + `timespan`; response gains `status`, `workspace`, per-service `results` map, and `guidance[]`.
- New typed envelope: `StreamAzureLogsResponse` is a `oneof { LogEntry entry; StreamStatus status; AzureDroppedNotice dropped; }`. `StreamStatus` carries `{status, mode, consecutive_fails, error, next_retry}` so health-channel JSON frames stop being out-of-band.
- New enums: `AzureCheckStatus`, `AzureOverallStatus`, `DiagnosticSettingsStatus`, `WorkspaceVerificationStatus`, `ServiceVerificationStatus`, `AzureResourceType` - replacing the legacy string-typed status fields.
- `AzureDroppedNotice` (rather than reusing `logs.proto`'s `DroppedNotice`) - lets the Azure overflow vocabulary evolve independently as the realtime streamer matures.

Every RPC in the rewritten proto is doc-commented with a citation to the legacy Go function it mirrors. Subsequent commits in Stage 2 wire the handler and migrate the dashboard hooks. No further proto changes are expected during Stage 2.

## Alternatives considered

**Raw gRPC.** Browser-hostile. gRPC-Web requires a proxy, no native fetch path, no streaming-from-server without trailers. Connect speaks gRPC + gRPC-Web + Connect protocol on the same handler, so we keep gRPC compatibility for free and add a browser-friendly path.

**tRPC.** TypeScript-only. Forces us to keep a separate Go contract or generate Go from TS; the wrong direction for a Go-rooted project.

**OpenAPI + generated clients.** More codegen surface, weaker streaming story (SSE is a bolt-on, not first-class), and the typed contract lives in YAML/JSON instead of a real schema language. proto + buf gives us breaking-change detection, lint, and a single source of truth across four consumers.

**Status quo (hand-rolled REST/WebSocket).** This is what we're replacing. Drift across consumers is the explicit problem.

## Migration plan

| PR | Scope | Behavior change? |
|---|---|---|
| **PR 1** ✅ | proto schema, codegen, generated stubs, ADR | None, no handlers wired |
| PR 2 (Stage 2) ✅ | Connect handlers mounted in parallel with existing REST. Dashboard reads via Connect-ES client. REST handlers untouched. AzureService proto rewrite + handler + 4 sub-store decomposition + dashboard migration land in a 3-commit batch. | Dashboard reads via Connect; REST still works for legacy callers |
| PR 3 (Stage 3) ✅ | CLI cobra commands (`app info`, `app logs`) and MCP tool handlers call a typed Connect client over localhost HTTP against the running dashboard process. (The CLI and dashboard are separate processes, so "in-process" calls are not possible; the Connect client talks to the same Connect handlers the browser uses.) MCP `info`-shaped tools stop spawning `azd app info` subprocesses; `reqs` remains a subprocess until a dedicated RequirementsService exists. No authentication interceptor is added; the dashboard continues to bind to localhost only, matching the trust posture of the REST surface it replaces. | MCP/CLI converge on the proto contract; subprocess round-trip eliminated for info; legacy REST still available. |
| Stage 3.5 ✅ | Remaining TS dashboard REST fetchers cut over to Connect-ES; WebSocket client replaced by Connect `StreamBroadcast` consumer. Landed between Stage 3 and PR 4 so PR 4 could remove the server with no live callers. | Dashboard fully on Connect; zero REST/WS callers remaining |
| PR 4 ✅ | Delete REST handlers, WebSocket plumbing, and the dashboard's REST fetchers. `github.com/coder/websocket` dropped from `cli/go.mod`. `BroadcastServiceUpdate` relocated to `server_broadcast.go`; securityHeaders middleware, port-discovery/lifecycle, and `broadcast.Manager` retained. `handleCheckRequirements` remains a subprocess path (no `RequirementsService` in proto yet). | REST + WebSocket surface removed |

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
