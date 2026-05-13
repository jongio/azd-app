/**
 * Tests for useHealthStream against an in-memory Connect router transport.
 *
 * The previous incarnation mocked `EventSource`; the hook now consumes a
 * Connect server-stream (`HealthService.StreamHealth`), so we drive the
 * production code path with `createRouterTransport` and a queue-backed
 * async generator. Each test gets a small `StreamController` with
 * `emitReport`/`emitChange`/`emitHeartbeat`/`close`, mirroring the way
 * the real server emits HealthEvent variants.
 */
import { renderHook, waitFor, act } from '@testing-library/react'
import { Code, ConnectError, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { describe, it, expect } from 'vitest'

import { useHealthStream } from './useHealthStream'
import { HealthService } from '@/gen/proto/azdapp/v1/health_pb.js'
import {
  HealthCheckResultSchema,
  type HealthCheckResult as ProtoHealthCheckResult,
  HealthChangeSchema,
  type HealthChange as ProtoHealthChange,
  HealthEventSchema,
  HealthReportSchema,
  type HealthReport,
  StreamHealthResponseSchema,
} from '@/gen/proto/azdapp/v1/health_pb.js'
import { HealthState } from '@/gen/proto/azdapp/v1/common_pb.js'
import { create } from '@bufbuild/protobuf'
import { type Timestamp, TimestampSchema } from '@bufbuild/protobuf/wkt'

type EmittedEvent =
  | { kind: 'report'; report: HealthReport }
  | { kind: 'change'; change: ProtoHealthChange }
  | { kind: 'heartbeat' }
  | { kind: 'error'; err: Error }
  | { kind: 'end' }

interface StreamController {
  emitReport: (results: ProtoHealthCheckResult[], generatedAt?: Timestamp) => void
  emitChange: (change: ProtoHealthChange) => void
  emitHeartbeat: () => void
  errorStream: (err: Error) => void
  closeStream: () => void
}

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  controller: StreamController
}

/**
 * Build a transport whose StreamHealth handler yields events queued via
 * the returned controller. The queue uses an inner promise per pending
 * event so the handler can suspend until tests push something — this is
 * the same shape `cli/src/internal/rpc/health.go` produces (one async
 * source -> one yield per source event) but in TS test land.
 */
function buildHarness(): Harness {
  const queue: EmittedEvent[] = []
  let signal: (() => void) | null = null
  let closed = false

  const wait = () =>
    new Promise<void>((resolve) => {
      signal = resolve
    })

  const push = (ev: EmittedEvent) => {
    if (closed) return
    queue.push(ev)
    const s = signal
    signal = null
    s?.()
  }

  const controller: StreamController = {
    emitReport: (results, generatedAt) => {
      const report = create(HealthReportSchema, { results, generatedAt: generatedAt ?? create(TimestampSchema) })
      push({ kind: 'report', report })
    },
    emitChange: (change) => push({ kind: 'change', change }),
    emitHeartbeat: () => push({ kind: 'heartbeat' }),
    errorStream: (err) => push({ kind: 'error', err }),
    closeStream: () => {
      closed = true
      push({ kind: 'end' })
    },
  }

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(HealthService, {
      // eslint-disable-next-line @typescript-eslint/require-await
      async getHealth() {
        // Not used by the hook; kept for completeness so the router has
        // a definition for every method on the service.
        return { results: [] } as never
      },
      async *streamHealth() {
        while (true) {
          if (queue.length === 0) {
            await wait()
          }
          const next = queue.shift()
          if (!next) continue
          if (next.kind === 'end') return
          if (next.kind === 'error') throw next.err
          if (next.kind === 'report') {
            yield create(StreamHealthResponseSchema, {
              event: create(HealthEventSchema, { event: { case: 'report', value: next.report } }),
            })
            continue
          }
          if (next.kind === 'change') {
            yield create(StreamHealthResponseSchema, {
              event: create(HealthEventSchema, { event: { case: 'change', value: next.change } }),
            })
            continue
          }
          if (next.kind === 'heartbeat') {
            yield create(StreamHealthResponseSchema, {
              event: create(HealthEventSchema, { event: { case: 'heartbeat', value: {} } }),
            })
          }
        }
      },
      async *streamStateTransitions() {
        // Hook does not subscribe to this; provide a no-op so the
        // service definition remains complete.
      },
    })
  })

  return { transport, controller }
}

function makeResult(serviceName: string, state: HealthState, message?: string): ProtoHealthCheckResult {
  return create(HealthCheckResultSchema, {
    serviceName,
    state,
    message: message ?? '',
    checkedAt: create(TimestampSchema),
    latencyMs: BigInt(45),
  })
}

describe('useHealthStream (Connect)', () => {
  it('initializes with empty defaults before the first message arrives', () => {
    const { transport } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    expect(result.current.healthReport).toBeNull()
    expect(result.current.changes).toEqual([])
    expect(result.current.connected).toBe(false)
    expect(result.current.error).toBeNull()
    expect(result.current.lastUpdate).toBeNull()
    expect(result.current.summary).toBeNull()
  })

  it('does not open a stream when disabled', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ enabled: false, transport }))

    // Pushing a report would surface immediately if the hook had
    // subscribed; absence of state mutation proves it didn't.
    controller.emitReport([makeResult('api', HealthState.HEALTHY)])
    await new Promise((r) => setTimeout(r, 20))

    expect(result.current.healthReport).toBeNull()
    expect(result.current.connected).toBe(false)
  })

  it('maps an initial HealthReport into the legacy HealthReportEvent shape', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    controller.emitReport([
      makeResult('api', HealthState.HEALTHY),
      makeResult('web', HealthState.UNHEALTHY, 'connection refused'),
    ])

    await waitFor(() => expect(result.current.connected).toBe(true))
    await waitFor(() => expect(result.current.healthReport).not.toBeNull())

    const report = result.current.healthReport!
    expect(report.type).toBe('health')
    expect(report.services).toHaveLength(2)

    const api = report.services.find((s) => s.serviceName === 'api')!
    expect(api.status).toBe('healthy')
    expect(api.checkType).toBe('http')
    // ms -> ns conversion for the legacy responseTime contract.
    expect(api.responseTime).toBe(45 * 1_000_000)
    expect(api.error).toBeUndefined()

    const web = report.services.find((s) => s.serviceName === 'web')!
    expect(web.status).toBe('unhealthy')
    expect(web.error).toBe('connection refused')

    // Summary is computed client-side; verify both the per-bucket counts
    // and the precedence rule that any unhealthy wins overall.
    expect(report.summary).toEqual({
      total: 2,
      healthy: 1,
      degraded: 0,
      unhealthy: 1,
      starting: 0,
      stopped: 0,
      unknown: 0,
      overall: 'unhealthy',
    })
    expect(result.current.summary).toEqual(report.summary)
    expect(result.current.lastUpdate).not.toBeNull()
  })

  it('counts STARTING services into summary.starting while keeping their HealthStatus as unknown', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    controller.emitReport([
      makeResult('api', HealthState.HEALTHY),
      makeResult('worker', HealthState.STARTING),
    ])

    await waitFor(() => expect(result.current.healthReport).not.toBeNull())

    const worker = result.current.getServiceHealth('worker')!
    // No 'starting' value exists on HealthStatus; the proto STARTING
    // collapses to 'unknown' for per-service status, but the summary
    // still tracks it in the dedicated counter.
    expect(worker.status).toBe('unknown')
    expect(result.current.summary?.starting).toBe(1)
    expect(result.current.summary?.unknown).toBe(1)
    // Healthy + unknown but no failures => overall stays 'unknown'
    // because not all services are healthy.
    expect(result.current.summary?.overall).toBe('unknown')
  })

  it('appends health-change events newest-first and exposes them via helpers', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    // Trigger first message so connected/error stabilise.
    controller.emitReport([makeResult('api', HealthState.HEALTHY)])
    await waitFor(() => expect(result.current.connected).toBe(true))

    controller.emitChange(
      create(HealthChangeSchema, {
        serviceName: 'api',
        previousState: HealthState.HEALTHY,
        currentState: HealthState.UNHEALTHY,
        message: 'connection refused',
        changedAt: create(TimestampSchema),
      })
    )
    controller.emitChange(
      create(HealthChangeSchema, {
        serviceName: 'api',
        previousState: HealthState.UNHEALTHY,
        currentState: HealthState.HEALTHY,
        changedAt: create(TimestampSchema),
      })
    )

    await waitFor(() => expect(result.current.changes).toHaveLength(2))

    // Newest first.
    expect(result.current.changes[0].oldStatus).toBe('unhealthy')
    expect(result.current.changes[0].newStatus).toBe('healthy')
    expect(result.current.changes[1].reason).toBe('connection refused')

    expect(result.current.getLatestChange('api')?.newStatus).toBe('healthy')
    expect(result.current.getLatestChange('absent')).toBeUndefined()
    expect(result.current.hasRecovered('api')).toBe(true)
  })

  it('caps stored changes at the MAX_CHANGES_TO_KEEP boundary', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    controller.emitReport([makeResult('svc', HealthState.HEALTHY)])
    await waitFor(() => expect(result.current.connected).toBe(true))

    for (let i = 0; i < 60; i++) {
      controller.emitChange(
        create(HealthChangeSchema, {
          serviceName: `svc-${i}`,
          previousState: HealthState.HEALTHY,
          currentState: HealthState.UNHEALTHY,
          changedAt: create(TimestampSchema),
        })
      )
    }

    await waitFor(() => expect(result.current.changes.length).toBe(50))
    // Newest first, so service 59 should be at index 0 and service 10
    // at the tail (older ones got dropped).
    expect(result.current.changes[0].service).toBe('svc-59')
    expect(result.current.changes.at(-1)?.service).toBe('svc-10')
  })

  it('updates lastUpdate on heartbeat without altering report or changes', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    controller.emitReport([makeResult('api', HealthState.HEALTHY)])
    await waitFor(() => expect(result.current.lastUpdate).not.toBeNull())
    const firstUpdate = result.current.lastUpdate

    // Wait at least 1ms so the new Date() is observably distinct.
    await new Promise((r) => setTimeout(r, 5))
    controller.emitHeartbeat()

    await waitFor(() => expect(result.current.lastUpdate?.getTime()).not.toBe(firstUpdate?.getTime()))
    // Heartbeat must not push fake change/report data.
    expect(result.current.changes).toHaveLength(0)
  })

  it('clearChanges empties the change buffer without touching the latest report', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() => useHealthStream({ transport }))

    controller.emitReport([makeResult('api', HealthState.HEALTHY)])
    controller.emitChange(
      create(HealthChangeSchema, {
        serviceName: 'api',
        previousState: HealthState.HEALTHY,
        currentState: HealthState.UNHEALTHY,
        changedAt: create(TimestampSchema),
      })
    )
    await waitFor(() => expect(result.current.changes).toHaveLength(1))

    act(() => {
      result.current.clearChanges()
    })

    expect(result.current.changes).toHaveLength(0)
    expect(result.current.healthReport).not.toBeNull()
  })

  it('flips connected to false and surfaces a reconnect message after a stream error', async () => {
    const { transport, controller } = buildHarness()
    const { result } = renderHook(() =>
      useHealthStream({ transport, reconnectDelay: 60_000, maxReconnectAttempts: 3 })
    )

    controller.emitReport([makeResult('api', HealthState.HEALTHY)])
    await waitFor(() => expect(result.current.connected).toBe(true))

    controller.errorStream(new ConnectError('upstream gone', Code.Unavailable))

    await waitFor(() => expect(result.current.connected).toBe(false))
    await waitFor(() => expect(result.current.error).toMatch(/Reconnecting in/))
  })
})
