/**
 * Shared Connect-RPC transport and per-service client factories for the
 * dashboard.
 *
 * Why this file exists:
 * - One process-wide transport, configured once. Hooks must never call
 *   `createConnectTransport` themselves; that would fragment configuration
 *   (interceptors, baseUrl, timeouts) across the codebase the moment the
 *   dashboard grows a second cross-cutting concern.
 * - Per-service factories take an optional `Transport` so unit tests can
 *   inject an in-memory router transport (`createRouterTransport`) without
 *   monkey-patching `globalThis.fetch`. Same client code path runs in
 *   production and in tests; only the wire moves.
 * - The default transport is constructed lazily on first use so import-time
 *   side effects stay zero (important for tree-shaking and for tests that
 *   never touch the real wire).
 *
 * Adding a new service: add an `import { FooService } from '@/gen/...'`
 * plus a `createFooClient(transport?)` factory that mirrors the pattern
 * below. Do NOT cache the client at module scope — the caller (typically
 * a hook with a stable transport reference) is responsible for memoising.
 */
import { createClient, type Client, type Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'

import { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_pb.js'
import { ModeService } from '@/gen/proto/azdapp/v1/mode_pb.js'
import { ProjectService } from '@/gen/proto/azdapp/v1/project_pb.js'
import { ServicesService } from '@/gen/proto/azdapp/v1/services_pb.js'
import { BicepService } from '@/gen/proto/azdapp/v1/bicep_pb.js'
import { HealthService } from '@/gen/proto/azdapp/v1/health_pb.js'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'

// =============================================================================
// Default transport
// =============================================================================

/**
 * Resolve the dashboard origin. The dashboard is served from the Go server
 * on the same origin as the API, so a relative `''` base URL would also
 * work in the browser. We normalize to `window.location.origin` so the
 * transport object is uniformly self-describing in logs and so non-browser
 * environments (jsdom in tests, future SSR experiments) get an explicit
 * URL instead of relying on a base-URL heuristic deep inside connect-web.
 */
function defaultBaseUrl(): string {
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin
  }
  // jsdom always provides window.location, but guard for hypothetical
  // non-browser hosts (SSR, workers) so transport construction never throws
  // at module import time.
  return 'http://localhost'
}

let cachedDefaultTransport: Transport | null = null

/**
 * Returns the singleton dashboard transport. Construction is deferred to
 * first call so importing this module from a test file that supplies its
 * own transport never spins up the real one.
 *
 * Interceptor slot: cross-cutting concerns (auth headers, request IDs,
 * telemetry) belong here, NOT in individual hooks. The dashboard today
 * has no auth wrapper around `fetch`; when one lands, add a Connect
 * interceptor here so REST and Connect callers share identical policy.
 */
export function getDefaultTransport(): Transport {
  if (cachedDefaultTransport === null) {
    cachedDefaultTransport = createConnectTransport({
      baseUrl: defaultBaseUrl(),
      // JSON over HTTP/1.1 by default. Matches the server-side Connect
      // handlers (no protobuf-binary opt-in yet) and keeps wire payloads
      // inspectable in DevTools during the migration.
      useBinaryFormat: false,
    })
  }
  return cachedDefaultTransport
}

/**
 * Test-only hook: replace the default transport for the duration of a
 * test. Pass `null` to fall back to the real transport. Production code
 * MUST NOT call this — the only call sites should be vitest specs that
 * wire a `createRouterTransport` against an in-memory service handler.
 *
 * Exposed as a named export rather than a plain assignment because TS
 * forbids module-level `let` mutation across files.
 */
export function __setDefaultTransportForTesting(transport: Transport | null): void {
  cachedDefaultTransport = transport
}

// =============================================================================
// Per-service client factories
// =============================================================================

/**
 * Construct a LifecycleService Connect client. Default-arg behavior mirrors
 * production usage; tests pass an in-memory transport.
 */
export function createLifecycleClient(
  transport: Transport = getDefaultTransport()
): Client<typeof LifecycleService> {
  return createClient(LifecycleService, transport)
}

/**
 * Construct a ProjectService Connect client.
 */
export function createProjectClient(
  transport: Transport = getDefaultTransport()
): Client<typeof ProjectService> {
  return createClient(ProjectService, transport)
}

/**
 * Construct a ModeService Connect client.
 */
export function createModeClient(
  transport: Transport = getDefaultTransport()
): Client<typeof ModeService> {
  return createClient(ModeService, transport)
}

/**
 * Construct a ServicesService Connect client.
 */
export function createServicesClient(
  transport: Transport = getDefaultTransport()
): Client<typeof ServicesService> {
  return createClient(ServicesService, transport)
}

/**
 * Construct a BicepService Connect client.
 */
export function createBicepClient(
  transport: Transport = getDefaultTransport()
): Client<typeof BicepService> {
  return createClient(BicepService, transport)
}

/**
 * Construct a HealthService Connect client. Used by useHealthStream for
 * the server-streaming `streamHealth` and `streamStateTransitions` RPCs
 * plus the unary `getHealth` one-shot probe.
 */
export function createHealthClient(
  transport: Transport = getDefaultTransport()
): Client<typeof HealthService> {
  return createClient(HealthService, transport)
}

/**
 * Construct a LogsService Connect client. Backs `useLogClassifications`
 * and `usePreferences` for the unary RPCs (list/add/delete classifications,
 * get/save preferences) and will back the streaming local-log hooks once
 * the WebSocket→server-stream migration lands in a follow-up commit.
 */
export function createLogsClient(
  transport: Transport = getDefaultTransport()
): Client<typeof LogsService> {
  return createClient(LogsService, transport)
}

/**
 * Construct an AzureService Connect client. Backs all dashboard Azure
 * touchpoints: setup wizard (workspace/auth/diagnostic settings checks,
 * verification), the LogsView's historical fetch + realtime stream, the
 * diagnostics modal, the table picker, and per-service log config.
 *
 * The AzureService surface deliberately preserves the legacy REST shape
 * (one RPC per old endpoint) so the consumer hooks can swap transports
 * one at a time without behavioural drift. The single server-streaming
 * RPC (`streamAzureLogs`) replaces the legacy
 * `/api/azure/logs/stream?realtime=true` WebSocket and ships a typed
 * three-event oneof: log entries, status frames (mode + connection
 * health), and back-pressure drop notices.
 */
export function createAzureClient(
  transport: Transport = getDefaultTransport()
): Client<typeof AzureService> {
  return createClient(AzureService, transport)
}
