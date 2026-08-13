/**
 * useHealthStream: subscribes to the HealthService.StreamHealth Connect
 * server-streaming RPC and exposes the same React hook surface the
 * dashboard already consumes.
 *
 * Wire migration note: replaces the previous EventSource hook against
 * `/api/health/stream`. The hook contract (UseHealthStreamReturn) is
 * preserved exactly so App.tsx keeps working unchanged. Only the wire
 * moves: instead of named SSE events (`message`, `health-change`,
 * `heartbeat`) we consume a `HealthEvent` oneof and translate each
 * variant back into the legacy event shape so downstream UI never sees
 * a proto type. Doing that translation in this single place, rather
 * than threading proto types through to App.tsx, keeps the migration
 * contained to the transport layer.
 *
 * The `summary` field on legacy `HealthReportEvent` was computed by the
 * Go SSE handler and shipped over the wire. The Connect-era proto
 * deliberately omits it because a server that emits `results` already
 * supplies everything needed to derive the summary, and shipping it
 * twice means two sources of truth that can drift. The hook computes
 * the summary client-side from the latest results, matching the legacy
 * contract.
 *
 * Reconnection: on stream error or disconnect, the hook schedules an
 * exponential-backoff retry (3s, 6s, 12s, 24s, 48s) up to
 * `maxReconnectAttempts`. A manual `reconnect()` resets the attempt
 * counter and restarts the stream immediately. The behaviour mirrors
 * the legacy SSE hook so on-screen messaging ("Reconnecting in Ns...",
 * "Backend connection lost") stays identical.
 */
import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { Code, ConnectError, type Client, type Transport } from '@connectrpc/connect'

import { createHealthClient } from '@/lib/connectClient'
import type { HealthService } from '@/gen/proto/azdapp/v1/health_pb.js'
import { StreamHealthRequestSchema, type HealthCheckResult as ProtoHealthCheckResult, type HealthChange as ProtoHealthChange } from '@/gen/proto/azdapp/v1/health_pb.js'
import { HealthState } from '@/gen/proto/azdapp/v1/common_pb.js'
import { create, protoInt64 } from '@bufbuild/protobuf'
import type { Timestamp } from '@bufbuild/protobuf/wkt'
import type { HealthReportEvent, HealthChangeEvent, HealthCheckResult, HealthSummary, HealthStatus } from '@/types'
import { WEBSOCKET_CONSTANTS, HEALTH_CONSTANTS } from '@/lib/constants'

/** Configuration options for the health stream hook */
export interface UseHealthStreamOptions {
  /** Whether to enable health streaming (default: true) */
  enabled?: boolean
  /** Interval between health checks in seconds (default: 5). Server clamps to 1..60. */
  interval?: number
  /** Optional service filter */
  services?: string[]
  /** Reconnect delay in ms after disconnect (default: 3000) */
  reconnectDelay?: number
  /** Maximum reconnect attempts (default: 5) */
  maxReconnectAttempts?: number
  /**
   * Override the underlying Connect transport. Production code never
   * passes this; tests inject `createRouterTransport(...)` so the real
   * client code path runs against an in-memory service handler.
   */
  transport?: Transport
}

/** Return type for the health stream hook */
export interface UseHealthStreamReturn {
  /** Latest health report */
  healthReport: HealthReportEvent | null
  /** Recent health changes (newest first) */
  changes: HealthChangeEvent[]
  /** Whether connected to the health stream */
  connected: boolean
  /** Error message if any */
  error: string | null
  /** Last update timestamp */
  lastUpdate: Date | null
  /** Health summary for quick access */
  summary: HealthSummary | null
  /** Get health result for a specific service */
  getServiceHealth: (serviceName: string) => HealthCheckResult | undefined
  /** Check if a service has recovered (was unhealthy, now healthy) */
  hasRecovered: (serviceName: string) => boolean
  /** Get the most recent change for a service */
  getLatestChange: (serviceName: string) => HealthChangeEvent | undefined
  /** Clear change history */
  clearChanges: () => void
  /** Manually trigger reconnection */
  reconnect: () => void
}

const DEFAULT_INTERVAL = HEALTH_CONSTANTS.DEFAULT_INTERVAL
const DEFAULT_RECONNECT_DELAY = WEBSOCKET_CONSTANTS.DEFAULT_RECONNECT_DELAY
const DEFAULT_MAX_RECONNECT_ATTEMPTS = WEBSOCKET_CONSTANTS.MAX_RECONNECT_ATTEMPTS
const MAX_CHANGES_TO_KEEP = HEALTH_CONSTANTS.MAX_CHANGES_TO_KEEP

// =============================================================================
// Proto -> dashboard mappers
// =============================================================================

/**
 * Map proto HealthState enum to the dashboard HealthStatus string.
 * Both UNSPECIFIED and any out-of-range value collapse to 'unknown' so
 * UI never branches on a sentinel and a future enum addition won't crash
 * the dashboard for clients on an older bundle.
 */
function healthStateToStatus(state: HealthState): HealthStatus {
  switch (state) {
    case HealthState.HEALTHY:
      return 'healthy'
    case HealthState.UNHEALTHY:
      return 'unhealthy'
    case HealthState.DEGRADED:
      return 'degraded'
    case HealthState.UNKNOWN:
    case HealthState.UNSPECIFIED:
    default:
      return 'unknown'
  }
}

/**
 * Convert google.protobuf.Timestamp (seconds + nanos as bigint) to ISO
 * string. Falls back to "now" when the server omitted the field: the
 * legacy SSE handler always populated it, so omission is a server bug
 * rather than a normal path, but a missing timestamp must never crash
 * the UI.
 */
function timestampToIso(ts: Timestamp | undefined): string {
  if (!ts) return new Date().toISOString()
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0)
  const nanos = typeof ts.nanos === 'number' ? ts.nanos : 0
  return new Date(seconds * 1000 + Math.floor(nanos / 1e6)).toISOString()
}

/**
 * Build a dashboard HealthCheckResult from a proto one.
 *
 * Field-by-field justification:
 * - serviceName/status/timestamp/details: direct round-trip.
 * - checkType: proto omits this (the dashboard concept does not exist in
 *   the contract); 'http' is the dominant case in this codebase and
 *   downstream UI uses it only as a hint label, never for logic.
 * - responseTime: legacy field is nanoseconds; proto carries milliseconds
 *   (latencyMs) since that's the only granularity the upstream
 *   healthcheck package surfaces. Convert ms -> ns to keep the existing
 *   UI formatters happy.
 * - error: surfaced from proto `message` only when the state is
 *   UNHEALTHY/DEGRADED, mirroring legacy semantics where `error` was
 *   absent on healthy results.
 */
function protoToHealthCheckResult(r: ProtoHealthCheckResult): HealthCheckResult {
  const status = healthStateToStatus(r.state)
  const isErrorState = status === 'unhealthy' || status === 'degraded'
  const result: HealthCheckResult = {
    serviceName: r.serviceName,
    status,
    checkType: 'http',
    responseTime: Number(r.latencyMs) * 1_000_000,
    timestamp: timestampToIso(r.checkedAt),
  }
  if (r.message && isErrorState) {
    result.error = r.message
  }
  if (r.details && Object.keys(r.details).length > 0) {
    // proto delivers a flat string map; pass it straight through. The
    // diagnostic UI already accepts arbitrary string values here.
    result.details = { ...r.details }
  }
  return result
}

/**
 * Compute a HealthSummary from a list of results. The proto stream
 * intentionally does not carry the summary (single source of truth at
 * the consumer); deriving it here matches the contract that the legacy
 * SSE handler computed server-side.
 *
 * `stopped` has no representation in the proto HealthState enum and the
 * upstream healthcheck package doesn't track it either; defaulting to 0
 * matches the actual data shape we now have.
 */
function computeSummary(results: HealthCheckResult[], protoResults: ProtoHealthCheckResult[]): HealthSummary {
  const summary: HealthSummary = {
    total: results.length,
    healthy: 0,
    degraded: 0,
    unhealthy: 0,
    starting: 0,
    stopped: 0,
    unknown: 0,
    overall: 'unknown',
  }
  for (const r of results) {
    switch (r.status) {
      case 'healthy':
        summary.healthy++
        break
      case 'degraded':
        summary.degraded++
        break
      case 'unhealthy':
        summary.unhealthy++
        break
      default:
        summary.unknown++
    }
  }
  // STARTING has no HealthStatus equivalent (it's a lifecycle state, not
  // a health state), so count it directly off the proto enum. Such
  // entries also fall into 'unknown' above, which matches legacy
  // behaviour where starting services rendered with the unknown badge.
  for (const r of protoResults) {
    if (r.state === HealthState.STARTING) summary.starting++
  }
  // Overall precedence: any unhealthy -> unhealthy; else any degraded ->
  // degraded; else all healthy -> healthy; empty/unknown -> unknown.
  // 'starting' is intentionally not an overall HealthStatus value.
  if (summary.unhealthy > 0) summary.overall = 'unhealthy'
  else if (summary.degraded > 0) summary.overall = 'degraded'
  else if (summary.healthy > 0 && summary.healthy === summary.total) summary.overall = 'healthy'
  else summary.overall = 'unknown'
  return summary
}

function protoChangeToEvent(c: ProtoHealthChange): HealthChangeEvent {
  return {
    type: 'health-change',
    timestamp: timestampToIso(c.changedAt),
    service: c.serviceName,
    oldStatus: healthStateToStatus(c.previousState),
    newStatus: healthStateToStatus(c.currentState),
    reason: c.message || undefined,
  }
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook for consuming health check data from the HealthService.StreamHealth
 * Connect server-streaming RPC. Maintains the same surface as the legacy
 * SSE-based hook so downstream consumers are unaffected.
 */
export function useHealthStream(options: UseHealthStreamOptions = {}): UseHealthStreamReturn {
  const {
    enabled = true,
    interval = DEFAULT_INTERVAL,
    services,
    reconnectDelay = DEFAULT_RECONNECT_DELAY,
    maxReconnectAttempts = DEFAULT_MAX_RECONNECT_ATTEMPTS,
    transport,
  } = options

  // Memoise the client per (transport) reference. Construction is cheap
  // but a new client object per render would invalidate the effect's
  // dep array on every render and reopen the stream, churning sockets.
  const client = useMemo<Client<typeof HealthService>>(
    () => createHealthClient(transport),
    [transport]
  )

  const [healthReport, setHealthReport] = useState<HealthReportEvent | null>(null)
  const [changes, setChanges] = useState<HealthChangeEvent[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)

  // Refs (not state) for transport plumbing: we never need to re-render
  // when the abort controller changes, only when received data does.
  const abortRef = useRef<AbortController | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const countdownIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const mountedRef = useRef(true)

  // Stable references to options that the consume loop captures so the
  // loop doesn't have to be reconstructed on every prop change.
  const intervalRef = useRef(interval)
  const servicesRef = useRef(services)
  const reconnectDelayRef = useRef(reconnectDelay)
  const maxReconnectAttemptsRef = useRef(maxReconnectAttempts)
  useEffect(() => {
    intervalRef.current = interval
    servicesRef.current = services
    reconnectDelayRef.current = reconnectDelay
    maxReconnectAttemptsRef.current = maxReconnectAttempts
  })

  // Forward declaration so connect() can recurse via the reconnect
  // scheduler and the scheduler can call connect().
  const connectRef = useRef<() => void>(() => undefined)

  const cleanup = useCallback(() => {
    if (abortRef.current) {
      abortRef.current.abort()
      abortRef.current = null
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (countdownIntervalRef.current) {
      clearInterval(countdownIntervalRef.current)
      countdownIntervalRef.current = null
    }
  }, [])

  const scheduleReconnect = useCallback(() => {
    if (!mountedRef.current) return
    if (countdownIntervalRef.current) {
      clearInterval(countdownIntervalRef.current)
      countdownIntervalRef.current = null
    }
    if (reconnectAttemptsRef.current >= maxReconnectAttemptsRef.current) {
      setError('Backend connection lost. Click to reconnect.')
      return
    }
    reconnectAttemptsRef.current++
    // Exponential backoff: 3s, 6s, 12s, 24s, 48s with the default delay.
    const delay = reconnectDelayRef.current * Math.pow(2, reconnectAttemptsRef.current - 1)
    let remainingSeconds = Math.ceil(delay / 1000)
    if (reconnectAttemptsRef.current <= 3) {
      setError(`Connection lost. Reconnecting in ${remainingSeconds}s...`)
      countdownIntervalRef.current = setInterval(() => {
        remainingSeconds--
        if (remainingSeconds > 0) {
          setError(`Connection lost. Reconnecting in ${remainingSeconds}s...`)
        }
      }, 1000)
    } else {
      setError('Backend connection lost')
    }
    reconnectTimeoutRef.current = setTimeout(() => {
      if (countdownIntervalRef.current) {
        clearInterval(countdownIntervalRef.current)
        countdownIntervalRef.current = null
      }
      connectRef.current()
    }, delay)
  }, [])

  const connect = useCallback(() => {
    cleanup()
    if (!enabled) return

    const controller = new AbortController()
    abortRef.current = controller

    // Build request once per attempt. We don't memoise it because the
    // construction cost is negligible vs. the lifetime of the stream
    // and pulling it out of the closure would create more state.
    const req = create(StreamHealthRequestSchema, {
      intervalSeconds: protoInt64.parse(intervalRef.current),
      serviceNames: servicesRef.current ?? [],
    })

    void (async () => {
      // Connection state flips to true on the first message, not on the
      // RPC initiation, because Connect over HTTP/1.1 only flushes
      // response headers when the handler sends the first message. Until
      // then we cannot prove the wire is healthy.
      let firstMessageSeen = false

      try {
        for await (const resp of client.streamHealth(req, { signal: controller.signal })) {
          if (!mountedRef.current || controller.signal.aborted) break
          if (!firstMessageSeen) {
            firstMessageSeen = true
            setConnected(true)
            setError(null)
            reconnectAttemptsRef.current = 0
          }

          const ev = resp.event?.event
          if (!ev) continue
          switch (ev.case) {
            case 'report': {
              const protoResults = ev.value.results
              const services = protoResults.map(protoToHealthCheckResult)
              const summary = computeSummary(services, protoResults)
              setHealthReport({
                type: 'health',
                timestamp: timestampToIso(ev.value.generatedAt),
                services,
                summary,
              })
              setLastUpdate(new Date())
              break
            }
            case 'change': {
              const change = protoChangeToEvent(ev.value)
              setChanges((prev) => [change, ...prev].slice(0, MAX_CHANGES_TO_KEEP))
              setLastUpdate(new Date())
              break
            }
            case 'heartbeat': {
              setLastUpdate(new Date())
              break
            }
          }
        }
        // Stream ended cleanly (server closed). Treat like a transient
        // disconnect and reschedule with backoff so the dashboard
        // recovers without a manual reload.
        if (mountedRef.current && !controller.signal.aborted) {
          setConnected(false)
          scheduleReconnect()
        }
      } catch (err) {
        if (controller.signal.aborted) return
        if (err instanceof ConnectError && err.code === Code.Canceled) return
        if (!mountedRef.current) return
        setConnected(false)
        scheduleReconnect()
      } finally {
        if (abortRef.current === controller) {
          abortRef.current = null
        }
      }
    })()
  }, [client, cleanup, enabled, scheduleReconnect])

  // Keep the ref to the latest connect() in sync so scheduleReconnect's
  // setTimeout callback always invokes the current closure (which has
  // captured the current props/transport).
  useEffect(() => {
    connectRef.current = connect
  })

  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0
    connect()
  }, [connect])

  const getServiceHealth = useCallback(
    (serviceName: string): HealthCheckResult | undefined => {
      return healthReport?.services.find((s) => s.serviceName === serviceName)
    },
    [healthReport]
  )

  const getLatestChange = useCallback(
    (serviceName: string): HealthChangeEvent | undefined => {
      return changes.find((c) => c.service === serviceName)
    },
    [changes]
  )

  const hasRecovered = useCallback(
    (serviceName: string): boolean => {
      const latestChange = getLatestChange(serviceName)
      return latestChange?.newStatus === 'healthy' && latestChange?.oldStatus !== 'healthy'
    },
    [getLatestChange]
  )

  const clearChanges = useCallback(() => {
    setChanges([])
  }, [])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      cleanup()
    }
  }, [connect, cleanup])

  const summary = healthReport?.summary ?? null

  return {
    healthReport,
    changes,
    connected,
    error,
    lastUpdate,
    summary,
    getServiceHealth,
    hasRecovered,
    getLatestChange,
    clearChanges,
    reconnect,
  } satisfies UseHealthStreamReturn
}
