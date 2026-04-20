/**
 * Tests for the ServicesContext provider against an in-memory Connect
 * router transport.
 *
 * After the WebSocket -> LifecycleService.StreamBroadcast migration the
 * provider has two RPC dependencies:
 *   1. ServicesService.GetServices  - initial snapshot
 *   2. LifecycleService.StreamBroadcast - live `services-changed` events
 *
 * Tests drive both via `createRouterTransport`, which wires the
 * provider's transport hook directly to in-memory handlers. The
 * legacy WebSocket-era "add / update / remove" message variants are
 * gone: the server only ever emitted `services-changed` bulk
 * snapshots, and the dashboard's wire contract now reflects that
 * reality. Those tests have been dropped rather than rewritten.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { ConnectError, Code, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { Struct } from '@bufbuild/protobuf'
import type { ReactNode } from 'react'

import { useServicesContext, ServicesProvider } from '@/contexts/ServicesContext'
import { mockServices, serviceToProto } from '@/test/mocks'
import { ServicesService } from '@/gen/proto/azdapp/v1/services_connect.js'
import { GetServicesResponse } from '@/gen/proto/azdapp/v1/services_pb.js'
import { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_connect.js'
import {
  BroadcastEvent,
  StreamBroadcastResponse,
} from '@/gen/proto/azdapp/v1/lifecycle_pb.js'

interface RouterOverrides {
  getServices?: () => Promise<GetServicesResponse> | GetServicesResponse
  streamController?: StreamController
}

/**
 * A controllable in-memory stream used by the LifecycleService router
 * handler. Tests push `BroadcastEvent` messages at will and the
 * handler yields them as `StreamBroadcastResponse` frames. `close()`
 * ends the stream cleanly so reconnect-backoff tests can observe the
 * provider looping.
 */
interface StreamController {
  push(event: BroadcastEvent): void
  close(): void
  /** Count of times the provider has dialed the stream. */
  connections(): number
  /** Internal: bound by `makeTransport` onto the LifecycleService router. */
  _handler(): AsyncGenerator<StreamBroadcastResponse>
}

function createStreamController(): StreamController {
  const queue: BroadcastEvent[] = []
  let resolveNext: (() => void) | null = null
  let closed = false
  let connections = 0

  const push: StreamController['push'] = (event) => {
    queue.push(event)
    const resolver = resolveNext
    resolveNext = null
    resolver?.()
  }
  const close: StreamController['close'] = () => {
    closed = true
    const resolver = resolveNext
    resolveNext = null
    resolver?.()
  }
  const connections_: StreamController['connections'] = () => connections

  // Attached to the router below. Async generator is re-entered each
  // time the provider dials the stream, so `connections` increments
  // naturally.
  ;(push as StreamController['push'] & { handler?: unknown }).handler = null

  const controller: StreamController = {
    push,
    close,
    connections: connections_,
    async *_handler() {
      connections++
      while (!closed) {
        while (queue.length > 0) {
          const event = queue.shift()
          if (event) yield new StreamBroadcastResponse({ event })
        }
        if (closed) return
        await new Promise<void>((resolve) => {
          resolveNext = resolve
        })
      }
    },
  }

  return controller
}

async function* defaultStreamHandler(): AsyncGenerator<StreamBroadcastResponse> {
  // Keep the stream open until the provider aborts on unmount.
  // `await new Promise` lets the Connect runtime observe abort.
  while (true) {
    await new Promise<void>((resolve) => setTimeout(resolve, 50))
    // Unreachable yield silences require-yield; the Connect runtime
    // tears the generator down via abort before this point.
    yield new StreamBroadcastResponse()
  }
}

function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(ServicesService, {
      getServices(): Promise<GetServicesResponse> {
        if (overrides.getServices) return Promise.resolve(overrides.getServices())
        return Promise.resolve(
          new GetServicesResponse({
            services: mockServices.map(serviceToProto),
          }),
        )
      },
      startService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      stopService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      restartService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
    })

    router.service(LifecycleService, {
      ping: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      getEnvironment: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      streamBroadcast: overrides.streamController
        ? overrides.streamController._handler.bind(overrides.streamController)
        : defaultStreamHandler,
    })
  })
}

function makeWrapper(transport: ReturnType<typeof makeTransport>) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <ServicesProvider transport={transport}>{children}</ServicesProvider>
  }
}

function servicesChangedEvent(services: unknown[]): BroadcastEvent {
  // Server serialises the bulk snapshot as `{services: [...]}` under
  // Struct. Tests construct the JSON the same way so the
  // toJson()-then-cast dance in ServicesContext reads the fixtures
  // faithfully.
  return new BroadcastEvent({
    type: 'services-changed',
    payload: Struct.fromJson({ services: services as never }),
  })
}

describe('useServicesContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetches services on mount via Connect', async () => {
    const transport = makeTransport()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(transport),
    })

    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.services).toHaveLength(mockServices.length)
    expect(result.current.services[0].name).toBe(mockServices[0].name)
    expect(result.current.services[0].local?.status).toBe('ready')
    expect(result.current.error).toBeNull()
  })

  it('falls back to mock data when the RPC fails', async () => {
    const transport = makeTransport({
      getServices: () => {
        throw new ConnectError('backend down', Code.Unavailable)
      },
    })
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(transport),
    })

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.error).toBeNull()
    expect(result.current.services.length).toBeGreaterThan(0)
    expect(consoleSpy).toHaveBeenCalledWith('Backend not available, using mock data')

    // Mock mode reports connected=true because `connected || useMock`.
    expect(result.current.connected).toBe(true)

    consoleSpy.mockRestore()
  })

  it('handles an empty service list', async () => {
    const transport = makeTransport({
      getServices: () => new GetServicesResponse({ services: [] }),
    })
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(transport),
    })

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.services).toEqual([])
  })

  it('reports connected=true once the broadcast stream opens', async () => {
    const stream = createStreamController()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport({ streamController: stream })),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    await waitFor(() => expect(result.current.connected).toBe(true))
  })

  it('applies a bulk services-changed snapshot from the stream', async () => {
    const stream = createStreamController()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport({ streamController: stream })),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    await waitFor(() => expect(result.current.connected).toBe(true))

    const updatedServices = mockServices.map((s) => ({
      ...s,
      local: { ...s.local, status: 'stopped', health: 'unknown' },
    }))

    act(() => {
      stream.push(servicesChangedEvent(updatedServices))
    })

    await waitFor(() => {
      result.current.services.forEach((service) => {
        expect(service.local?.status).toBe('stopped')
      })
    })
  })

  it('ignores broadcast events that are not services-changed', async () => {
    const stream = createStreamController()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport({ streamController: stream })),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    await waitFor(() => expect(result.current.connected).toBe(true))

    const snapshotBefore = result.current.services

    act(() => {
      stream.push(
        new BroadcastEvent({
          type: 'mode-toggled',
          payload: Struct.fromJson({ mode: 'azure' }),
        }),
      )
    })

    // Give the stream reader a microtask tick to observe the event.
    await new Promise((resolve) => setTimeout(resolve, 50))

    expect(result.current.services).toBe(snapshotBefore)
  })

  it('refetches via Connect when refetch() is invoked', async () => {
    let callCount = 0
    const transport = makeTransport({
      getServices: () => {
        callCount++
        return new GetServicesResponse({ services: mockServices.map(serviceToProto) })
      },
    })

    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(transport),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(callCount).toBe(1)

    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => expect(callCount).toBe(2))
  })

  it('cleanly aborts the broadcast stream on unmount', async () => {
    const stream = createStreamController()
    const { result, unmount } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport({ streamController: stream })),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    await waitFor(() => expect(stream.connections()).toBe(1))

    // No assertion beyond "this doesn't throw". The provider's cleanup
    // calls abort.abort(), which surfaces as a Canceled ConnectError
    // the run loop swallows -- the test would error if the provider
    // mishandled the abort.
    unmount()
  })
})
