/**
 * Tests for useSharedLogStream's local (Connect) path.
 *
 * Drives the production code through a `createRouterTransport` shim
 * that backs the `LogsService.streamLocalLogs` handler with a queue,
 * matching the canonical pattern from useHealthStream.test.ts. The
 * Azure (legacy WebSocket) path is intentionally NOT exercised here:
 * its wire format is unchanged by this batch and its old WS-based
 * tests (LogsView, integration suite) keep covering it until the
 * AzureService migration lands.
 */
import { renderHook, act, waitFor } from '@testing-library/react'
import { Code, ConnectError, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { describe, it, expect, afterEach } from 'vitest'
import { Timestamp } from '@bufbuild/protobuf'

import { useSharedLogStream, resetManagers, protoToLogEntry } from './useSharedLogStream'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_connect.js'
import {
  StreamLocalLogsResponse,
  DroppedNotice,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogEntry as ProtoLogEntry, LogLevel, LogStream } from '@/gen/proto/azdapp/v1/common_pb.js'

type EmittedEvent =
  | { kind: 'entry'; entry: ProtoLogEntry }
  | { kind: 'dropped'; count: number }
  | { kind: 'error'; err: Error }
  | { kind: 'end' }

interface StreamController {
  emitEntry: (entry: ProtoLogEntry) => void
  emitDropped: (count: number) => void
  errorStream: (err: Error) => void
  closeStream: () => void
  getInvocations: () => number
  getLastSignal: () => AbortSignal | undefined
}

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  controller: StreamController
}

function buildHarness(): Harness {
  const queue: EmittedEvent[] = []
  let signal: (() => void) | null = null
  let closed = false
  let invocations = 0
  let lastSignal: AbortSignal | undefined

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
    emitEntry: (entry) => push({ kind: 'entry', entry }),
    emitDropped: (count) => push({ kind: 'dropped', count }),
    errorStream: (err) => push({ kind: 'error', err }),
    closeStream: () => {
      closed = true
      push({ kind: 'end' })
    },
    getInvocations: () => invocations,
    getLastSignal: () => lastSignal,
  }

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(LogsService, {
      /* eslint-disable @typescript-eslint/require-await -- stub methods unused by hook */
      async *streamLocalLogs(_req, ctx) {
        invocations += 1
        lastSignal = ctx.signal
        while (true) {
          if (queue.length === 0) {
            await wait()
          }
          const next = queue.shift()
          if (!next) continue
          if (next.kind === 'end') return
          if (next.kind === 'error') throw next.err
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
      // The Logs service has additional unary methods; provide stubs
      // so the router has a complete service definition. The hook
      // never invokes them, but createRouterTransport requires every
      // method to be implemented.
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

  return { transport, controller }
}

const sampleEntry = (overrides?: Partial<ProtoLogEntry>): ProtoLogEntry => {
  return new ProtoLogEntry({
    id: '1',
    timestamp: new Timestamp(),
    service: 'web',
    level: LogLevel.INFO,
    stream: LogStream.STDOUT,
    source: undefined,
    message: 'hello world',
    ...overrides,
  })
}

describe('useSharedLogStream (Connect local mode)', () => {
  afterEach(() => {
    resetManagers()
  })

  it('subscribes, receives entries, and surfaces droppedCount', async () => {
    const { transport, controller } = buildHarness()
    const received: { service: string; message: string }[] = []

    const { result } = renderHook(() =>
      useSharedLogStream({
        serviceName: 'web',
        enabled: true,
        mode: 'local',
        transport,
        onLogEntry: (entry) => received.push({ service: entry.service, message: entry.message }),
      }),
    )

    await waitFor(() => expect(controller.getInvocations()).toBe(1))

    await act(async () => {
      controller.emitEntry(sampleEntry({ message: 'first' }))
      controller.emitEntry(sampleEntry({ message: 'second' }))
      await Promise.resolve()
    })

    await waitFor(() => expect(received).toHaveLength(2))
    expect(received.map((e) => e.message)).toEqual(['first', 'second'])
    await waitFor(() => expect(result.current.connectionState).toBe('connected'))

    await act(async () => {
      controller.emitDropped(7)
      await Promise.resolve()
    })

    await waitFor(() => expect(result.current.droppedCount).toBe(7))
  })

  it('aborts the upstream stream when the hook unmounts', async () => {
    const { transport, controller } = buildHarness()

    const { unmount } = renderHook(() =>
      useSharedLogStream({
        serviceName: 'web',
        enabled: true,
        mode: 'local',
        transport,
        onLogEntry: () => {},
      }),
    )

    await waitFor(() => expect(controller.getInvocations()).toBe(1))
    const signal = controller.getLastSignal()
    expect(signal).toBeDefined()
    expect(signal!.aborted).toBe(false)

    unmount()

    await waitFor(() => expect(signal!.aborted).toBe(true))
  })

  it('ignores entries while disabled but still tracks state via subscribe', async () => {
    const { transport, controller } = buildHarness()
    const received: string[] = []

    const { result, rerender } = renderHook(
      (props: { enabled: boolean }) =>
        useSharedLogStream({
          serviceName: 'web',
          enabled: props.enabled,
          mode: 'local',
          transport,
          onLogEntry: (entry) => received.push(entry.message),
        }),
      { initialProps: { enabled: false } },
    )

    // No subscribers => no upstream call.
    await Promise.resolve()
    expect(controller.getInvocations()).toBe(0)
    expect(result.current.connectionState).toBe('disconnected')

    rerender({ enabled: true })

    await waitFor(() => expect(controller.getInvocations()).toBe(1))

    await act(async () => {
      controller.emitEntry(sampleEntry({ message: 'enabled' }))
      await Promise.resolve()
    })
    await waitFor(() => expect(received).toEqual(['enabled']))
  })

  it('treats a Code.Canceled error as a clean shutdown (no reconnect storm)', async () => {
    const { transport, controller } = buildHarness()

    const { unmount } = renderHook(() =>
      useSharedLogStream({
        serviceName: 'web',
        enabled: true,
        mode: 'local',
        transport,
        onLogEntry: () => {},
      }),
    )

    await waitFor(() => expect(controller.getInvocations()).toBe(1))

    await act(async () => {
      controller.errorStream(new ConnectError('client cancel', Code.Canceled))
      await Promise.resolve()
    })

    // Server-side cancellation must not trigger another invocation
    // (no infinite reconnect loop).
    await new Promise((r) => setTimeout(r, 20))
    expect(controller.getInvocations()).toBe(1)

    unmount()
  })
})

describe('protoToLogEntry mapping', () => {
  it('maps proto LogLevel onto the dashboard 1/2/3 scale', () => {
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.INFO })).level).toBe(1)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.WARN })).level).toBe(2)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.ERROR })).level).toBe(3)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.FATAL })).level).toBe(3)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.DEBUG })).level).toBe(1)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.TRACE })).level).toBe(1)
    expect(protoToLogEntry(sampleEntry({ level: LogLevel.UNSPECIFIED })).level).toBe(1)
  })

  it('flips isStderr for the STDERR stream variant', () => {
    expect(protoToLogEntry(sampleEntry({ stream: LogStream.STDOUT })).isStderr).toBe(false)
    expect(protoToLogEntry(sampleEntry({ stream: LogStream.STDERR })).isStderr).toBe(true)
  })

  it('emits an ISO timestamp string', () => {
    const ts = new Timestamp({ seconds: 1_700_000_000n, nanos: 500_000_000 })
    const out = protoToLogEntry(sampleEntry({ timestamp: ts }))
    expect(out.timestamp).toBe(new Date(1_700_000_000 * 1000 + 500).toISOString())
  })
})

