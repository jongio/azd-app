/**
 * Tests for useLogStream against an in-memory Connect router transport.
 *
 * The hook now consumes `LogsService.streamLocalLogs` (Connect server-
 * stream) plus the legacy REST `GET /api/logs` for the historical
 * tail. The REST call is mocked via `globalThis.fetch`; the streaming
 * call is mocked via `createRouterTransport`. This mirrors the pattern
 * established in useHealthStream.test.ts.
 */
import { renderHook, act, waitFor } from '@testing-library/react'
import { createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { Timestamp } from '@bufbuild/protobuf'

import { useLogStream } from './useLogStream'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_connect.js'
import { StreamLocalLogsResponse, DroppedNotice } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogEntry as ProtoLogEntry, LogLevel, LogStream } from '@/gen/proto/azdapp/v1/common_pb.js'

type EmittedEvent =
  | { kind: 'entry'; entry: ProtoLogEntry }
  | { kind: 'dropped'; count: number }
  | { kind: 'end' }

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  emitEntry: (entry: ProtoLogEntry) => void
  emitDropped: (count: number) => void
  closeStream: () => void
  getInvocations: () => number
  getLastSignal: () => AbortSignal | undefined
  getLastServiceName: () => string | undefined
}

function buildHarness(): Harness {
  const queue: EmittedEvent[] = []
  let signal: (() => void) | null = null
  let closed = false
  let invocations = 0
  let lastSignal: AbortSignal | undefined
  let lastServiceName: string | undefined

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

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(LogsService, {
      /* eslint-disable @typescript-eslint/require-await -- stub methods unused by hook */
      async *streamLocalLogs(req, ctx) {
        invocations += 1
        lastSignal = ctx.signal
        lastServiceName = req.serviceName
        while (true) {
          if (queue.length === 0) {
            await wait()
          }
          const next = queue.shift()
          if (!next) continue
          if (next.kind === 'end') return
          if (next.kind === 'entry') {
            yield new StreamLocalLogsResponse({
              event: { case: 'entry', value: next.entry },
            })
            continue
          }
          if (next.kind === 'dropped') {
            yield new StreamLocalLogsResponse({
              event: {
                case: 'dropped',
                value: new DroppedNotice({ count: BigInt(next.count), at: new Timestamp() }),
              },
            })
          }
        }
      },
      async getLogs() {
        return { entries: [], lastId: 0n } as never
      },
      async listClassifications() {
        return { classifications: [] } as never
      },
      async addClassification() {
        return { classifications: [] } as never
      },
      async deleteClassification() {
        return { classifications: [] } as never
      },
      async getPreferences() {
        return { preferences: undefined } as never
      },
      async savePreferences() {
        return { preferences: undefined } as never
      },
    })
  })

  return {
    transport,
    emitEntry: (entry) => push({ kind: 'entry', entry }),
    emitDropped: (count) => push({ kind: 'dropped', count }),
    closeStream: () => {
      closed = true
      push({ kind: 'end' })
    },
    getInvocations: () => invocations,
    getLastSignal: () => lastSignal,
    getLastServiceName: () => lastServiceName,
  }
}

const sampleEntry = (overrides?: Partial<ProtoLogEntry>): ProtoLogEntry =>
  new ProtoLogEntry({
    id: '1',
    timestamp: new Timestamp(),
    service: 'web',
    level: LogLevel.INFO,
    stream: LogStream.STDOUT,
    message: 'hello',
    ...overrides,
  })

describe('useLogStream', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches initial logs via REST and streams new entries via Connect', async () => {
    const initialEntry = {
      service: 'web',
      message: 'initial',
      level: 1,
      timestamp: new Date().toISOString(),
      isStderr: false,
    }
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([initialEntry]),
    })

    const harness = buildHarness()
    const { result } = renderHook(() =>
      useLogStream({ serviceName: 'web', transport: harness.transport }),
    )

    await waitFor(() => expect(result.current.logs).toHaveLength(1))
    expect(result.current.logs[0].message).toBe('initial')

    await waitFor(() => expect(harness.getInvocations()).toBe(1))
    expect(harness.getLastServiceName()).toBe('web')

    await act(async () => {
      harness.emitEntry(sampleEntry({ message: 'live-1' }))
      await Promise.resolve()
    })

    await waitFor(() => expect(result.current.logs).toHaveLength(2))
    expect(result.current.logs[1].message).toBe('live-1')
    await waitFor(() => expect(result.current.isConnected).toBe(true))
  })

  it('subscribes to all services when serviceName is omitted', async () => {
    const harness = buildHarness()
    renderHook(() => useLogStream({ transport: harness.transport }))
    await waitFor(() => expect(harness.getInvocations()).toBe(1))
    expect(harness.getLastServiceName()).toBe('')
  })

  it('subscribes to all services when serviceName is "all"', async () => {
    const harness = buildHarness()
    renderHook(() => useLogStream({ serviceName: 'all', transport: harness.transport }))
    await waitFor(() => expect(harness.getInvocations()).toBe(1))
    expect(harness.getLastServiceName()).toBe('')
  })

  it('accumulates droppedCount on DroppedNotice events', async () => {
    const harness = buildHarness()
    const { result } = renderHook(() =>
      useLogStream({ serviceName: 'web', transport: harness.transport }),
    )

    await waitFor(() => expect(harness.getInvocations()).toBe(1))

    await act(async () => {
      harness.emitDropped(3)
      harness.emitDropped(5)
      await Promise.resolve()
    })

    await waitFor(() => expect(result.current.droppedCount).toBe(8))
  })

  it('aborts the stream on unmount', async () => {
    const harness = buildHarness()
    const { unmount } = renderHook(() =>
      useLogStream({ serviceName: 'web', transport: harness.transport }),
    )

    await waitFor(() => expect(harness.getInvocations()).toBe(1))
    const signal = harness.getLastSignal()
    expect(signal).toBeDefined()
    expect(signal!.aborted).toBe(false)

    unmount()
    await waitFor(() => expect(signal!.aborted).toBe(true))
  })

  it('drops live entries while paused but keeps the stream open', async () => {
    const harness = buildHarness()
    const { result, rerender } = renderHook(
      (props: { isPaused: boolean }) =>
        useLogStream({
          serviceName: 'web',
          transport: harness.transport,
          isPaused: props.isPaused,
        }),
      { initialProps: { isPaused: true } },
    )

    await waitFor(() => expect(harness.getInvocations()).toBe(1))

    await act(async () => {
      harness.emitEntry(sampleEntry({ message: 'while-paused' }))
      await Promise.resolve()
    })

    expect(result.current.logs).toHaveLength(0)

    // Unpause and emit again - new entry should land.
    rerender({ isPaused: false })
    await act(async () => {
      harness.emitEntry(sampleEntry({ message: 'after-resume' }))
      await Promise.resolve()
    })
    await waitFor(() => expect(result.current.logs).toHaveLength(1))
    expect(result.current.logs[0].message).toBe('after-resume')
  })

  it('clears logs when onClearTrigger increments', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve([
          {
            service: 'web',
            message: 'a',
            level: 1,
            timestamp: new Date().toISOString(),
            isStderr: false,
          },
        ]),
    })
    const harness = buildHarness()
    const { result, rerender } = renderHook(
      (props: { trigger: number }) =>
        useLogStream({
          serviceName: 'web',
          transport: harness.transport,
          onClearTrigger: props.trigger,
        }),
      { initialProps: { trigger: 0 } },
    )

    await waitFor(() => expect(result.current.logs).toHaveLength(1))

    rerender({ trigger: 1 })
    await waitFor(() => expect(result.current.logs).toHaveLength(0))
  })
})

