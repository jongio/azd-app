/**
 * Tests for the ServicesContext provider against an in-memory Connect
 * router transport. WebSocket-driven tests still mock `globalThis.WebSocket`
 * because the streaming migration is deferred to a later batch.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { ConnectError, Code, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'

import { useServicesContext, ServicesProvider } from '@/contexts/ServicesContext'
import { mockServices, createMockWebSocketMessage, serviceToProto } from '@/test/mocks'
import { ServicesService } from '@/gen/proto/azdapp/v1/services_connect.js'
import { GetServicesResponse } from '@/gen/proto/azdapp/v1/services_pb.js'
import type { ReactNode } from 'react'

interface MockWebSocket {
  url: string
  onopen: ((event: Event) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  close: ReturnType<typeof vi.fn>
}

interface RouterOverrides {
  getServices?: () => Promise<GetServicesResponse> | GetServicesResponse
}

function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(ServicesService, {
      async getServices() {
        if (overrides.getServices) return overrides.getServices()
        return new GetServicesResponse({
          services: mockServices.map(serviceToProto),
        })
      },
      startService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      stopService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
      restartService: () =>
        Promise.reject(new ConnectError('not used in these tests', Code.Unimplemented)),
    })
  })
}

function makeWrapper(transport: ReturnType<typeof makeTransport>) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <ServicesProvider transport={transport}>{children}</ServicesProvider>
  }
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

  it('reports connected=true once the WebSocket opens', async () => {
    installWebSocketMock()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    await waitFor(() => expect(result.current.connected).toBe(true))
  })

  it('handles WebSocket service updates', async () => {
    const wsRef = installWebSocketMock()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))

    const updatedService = {
      ...mockServices[0],
      local: { ...mockServices[0].local, status: 'stopping' as const },
    }

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(createMockWebSocketMessage({ type: 'update', service: updatedService }))
      })
    }

    await waitFor(() => {
      const apiService = result.current.services.find(s => s.name === 'api')
      expect(apiService?.local?.status).toBe('stopping')
    })
  })

  it('handles WebSocket bulk services updates', async () => {
    const wsRef = installWebSocketMock()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))

    const updatedServices = mockServices.map(s => ({
      ...s,
      local: { ...s.local, status: 'stopped' as const, health: 'unknown' as const },
    }))

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(createMockWebSocketMessage({ type: 'services', services: updatedServices }))
      })
    }

    await waitFor(() => {
      result.current.services.forEach(service => {
        expect(service.local?.status).toBe('stopped')
      })
    })
  })

  it('handles WebSocket service addition', async () => {
    const wsRef = installWebSocketMock()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    const initialCount = result.current.services.length

    const newService = {
      name: 'new-service',
      language: 'rust',
      framework: 'actix',
      local: { status: 'ready' as const, health: 'healthy' as const, port: 8080 },
    }

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(createMockWebSocketMessage({ type: 'add', service: newService }))
      })
    }

    await waitFor(() => {
      expect(result.current.services).toHaveLength(initialCount + 1)
      expect(result.current.services.find(s => s.name === 'new-service')).toBeDefined()
    })
  })

  it('handles WebSocket service removal', async () => {
    const wsRef = installWebSocketMock()
    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    const initialCount = result.current.services.length

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(createMockWebSocketMessage({ type: 'remove', service: mockServices[0] }))
      })
    }

    await waitFor(() => {
      expect(result.current.services).toHaveLength(initialCount - 1)
      expect(result.current.services.find(s => s.name === mockServices[0].name)).toBeUndefined()
    })
  })

  it('logs malformed WebSocket messages without mutating state', async () => {
    const wsRef = installWebSocketMock()
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const { result } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(result.current.loading).toBe(false))
    const initialServices = [...result.current.services]

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(new MessageEvent('message', { data: 'not-valid-json' }))
      })
    }

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to parse WebSocket message:',
        expect.any(Error),
      )
    })

    expect(result.current.services).toEqual(initialServices)
    consoleErrorSpy.mockRestore()
  })

  it('refetches via Connect when refetch() is invoked', async () => {
    installWebSocketMock()
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

  it('closes the WebSocket on unmount', async () => {
    const closeMock = vi.fn()
    class WebSocketMock {
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      static readonly CLOSING = 2
      static readonly CLOSED = 3
      onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
      onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
      onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
      onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
      // The provider only calls close() when readyState is OPEN or CONNECTING.
      readyState = 1
      close = closeMock
      send = vi.fn()
      constructor(_url: string) {
        // no-op
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    const { unmount } = renderHook(() => useServicesContext(), {
      wrapper: makeWrapper(makeTransport()),
    })
    await waitFor(() => expect(true).toBe(true))
    unmount()
    expect(closeMock).toHaveBeenCalled()
  })
})

// installWebSocketMock plants a class WebSocketMock that captures the
// last instance via the returned ref and dispatches an `open` event on
// next-tick so consumers see `connected=true`.
function installWebSocketMock(): { current: MockWebSocket | null } {
  const wsRef: { current: MockWebSocket | null } = { current: null }
  class WebSocketMock {
    static readonly CONNECTING = 0
    static readonly OPEN = 1
    static readonly CLOSING = 2
    static readonly CLOSED = 3
    url: string
    onopen: ((event: Event) => void) | null = null
    onmessage: ((event: MessageEvent) => void) | null = null
    onerror: ((event: Event) => void) | null = null
    onclose: ((event: CloseEvent) => void) | null = null
    readyState = 1
    close = vi.fn()
    constructor(url: string) {
      this.url = url
      wsRef.current = this
      setTimeout(() => {
        this.onopen?.(new Event('open'))
      }, 0)
    }
  }
  globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket
  return wsRef
}
