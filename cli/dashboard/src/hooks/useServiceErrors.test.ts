/**
 * Tests for useServiceErrors against an in-memory Connect router transport.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Code, ConnectError, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { renderHook, act } from '@testing-library/react'
import { create } from '@bufbuild/protobuf'
import { TimestampSchema } from '@bufbuild/protobuf/wkt'

import { useServiceErrors } from './useServiceErrors'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { StreamLocalLogsResponseSchema } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogEntrySchema, type LogEntry as ProtoLogEntry, LogLevel, LogStream } from '@/gen/proto/azdapp/v1/common_pb.js'

vi.mock('@/hooks/useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true }),
}))

interface ServiceStream {
  serviceName: string
  signal: AbortSignal
  emit: (entry: ProtoLogEntry) => void
  close: () => void
  error: (err: Error) => void
}

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  getStreams: () => ServiceStream[]
}

function buildHarness(): Harness {
  const streams: ServiceStream[] = []

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(LogsService, {
      /* eslint-disable @typescript-eslint/require-await -- stub methods unused by hook */
      async *streamLocalLogs(req, ctx) {
        const queue: Array<
          { kind: 'entry'; entry: ProtoLogEntry } | { kind: 'end' } | { kind: 'error'; err: Error }
        > = []
        let signal: (() => void) | null = null
        let closed = false

        const wait = () =>
          new Promise<void>((resolve) => {
            signal = resolve
          })

        const stream: ServiceStream = {
          serviceName: req.serviceName,
          signal: ctx.signal,
          emit: (entry) => {
            if (closed) return
            queue.push({ kind: 'entry', entry })
            const s = signal
            signal = null
            s?.()
          },
          close: () => {
            closed = true
            queue.push({ kind: 'end' })
            const s = signal
            signal = null
            s?.()
          },
          error: (err) => {
            queue.push({ kind: 'error', err })
            const s = signal
            signal = null
            s?.()
          },
        }
        streams.push(stream)

        while (true) {
          if (queue.length === 0) {
            await wait()
          }
          const next = queue.shift()
          if (!next) continue
          if (next.kind === 'end') return
          if (next.kind === 'error') throw next.err
          yield create(StreamLocalLogsResponseSchema, {
            event: { case: 'entry', value: next.entry },
          })
        }
      },
      async getLogs() {
        return { entries: [], lastId: 0n } as never
      },
      async listClassifications() {
        return { classifications: [] }
      },
      async addClassification() {
        return { classifications: [] } as never
      },
      async deleteClassification() {
        return { classifications: [] } as never
      },
      async getPreferences() {
        return { preferences: undefined }
      },
      async savePreferences() {
        return { preferences: undefined }
      },
    })
  })

  return { transport, getStreams: () => streams }
}

const errorEntry = (service: string, message = 'BOOM!'): ProtoLogEntry =>
  create(LogEntrySchema, {
    id: '1',
    timestamp: create(TimestampSchema),
    service,
    level: LogLevel.ERROR,
    stream: LogStream.STDERR,
    message,
  })

const infoEntry = (service: string, message = 'fine'): ProtoLogEntry =>
  create(LogEntrySchema, {
    id: '1',
    timestamp: create(TimestampSchema),
    service,
    level: LogLevel.INFO,
    stream: LogStream.STDOUT,
    message,
  })

describe('useServiceErrors', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('opens one Connect stream per service', async () => {
    const harness = buildHarness()
    renderHook(() => useServiceErrors(['web', 'api', 'worker'], { transport: harness.transport }))

    await vi.waitFor(() => {
      expect(harness.getStreams().map((s) => s.serviceName).sort()).toEqual([
        'api',
        'web',
        'worker',
      ])
    })
  })

  it('flips hasActiveErrors to true when any stream reports an error entry', async () => {
    const harness = buildHarness()
    const { result } = renderHook(() =>
      useServiceErrors(['web', 'api'], { transport: harness.transport }),
    )

    await vi.waitFor(() => expect(harness.getStreams()).toHaveLength(2))

    expect(result.current.hasActiveErrors).toBe(false)

    await act(async () => {
      harness.getStreams()[0].emit(errorEntry('web'))
      await Promise.resolve()
    })

    // Poll cycle is 1s; advance and let the interval fire.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(result.current.hasActiveErrors).toBe(true)
  })

  it('keeps hasActiveErrors false for non-error entries', async () => {
    const harness = buildHarness()
    const { result } = renderHook(() =>
      useServiceErrors(['web'], { transport: harness.transport }),
    )

    await vi.waitFor(() => expect(harness.getStreams()).toHaveLength(1))

    await act(async () => {
      harness.getStreams()[0].emit(infoEntry('web', 'happy path'))
      await Promise.resolve()
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })

    expect(result.current.hasActiveErrors).toBe(false)
  })

  it('clears hasActiveErrors after the 30s window expires', async () => {
    const harness = buildHarness()
    const { result } = renderHook(() =>
      useServiceErrors(['web'], { transport: harness.transport }),
    )

    await vi.waitFor(() => expect(harness.getStreams()).toHaveLength(1))

    await act(async () => {
      harness.getStreams()[0].emit(errorEntry('web'))
      await Promise.resolve()
    })
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1000)
    })
    expect(result.current.hasActiveErrors).toBe(true)

    // Advance past the window.
    await act(async () => {
      await vi.advanceTimersByTimeAsync(31_000)
    })
    expect(result.current.hasActiveErrors).toBe(false)
  })

  it('aborts every per-service stream on unmount', async () => {
    const harness = buildHarness()
    const { unmount } = renderHook(() =>
      useServiceErrors(['web', 'api'], { transport: harness.transport }),
    )

    await vi.waitFor(() => expect(harness.getStreams()).toHaveLength(2))
    const signals = harness.getStreams().map((s) => s.signal)
    signals.forEach((s) => expect(s.aborted).toBe(false))

    unmount()
    await vi.waitFor(() => signals.forEach((s) => expect(s.aborted).toBe(true)))
  })

  it('silently swallows NotFound errors (service may not be ready)', async () => {
    const harness = buildHarness()
    const consoleErr = vi.spyOn(console, 'error').mockImplementation(() => {})
    const consoleWarn = vi.spyOn(console, 'warn').mockImplementation(() => {})

    renderHook(() => useServiceErrors(['cold-service'], { transport: harness.transport }))

    await vi.waitFor(() => expect(harness.getStreams()).toHaveLength(1))

    await act(async () => {
      harness.getStreams()[0].error(new ConnectError('not found', Code.NotFound))
      await Promise.resolve()
    })

    // Hook never logs for NotFound and never throws into React.
    expect(consoleErr).not.toHaveBeenCalled()
    expect(consoleWarn).not.toHaveBeenCalled()
  })
})

