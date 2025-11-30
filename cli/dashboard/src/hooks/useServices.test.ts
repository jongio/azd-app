import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useServices } from '@/hooks/useServices'
import { mockServices, createMockFetchResponse, createMockWebSocketMessage } from '@/test/mocks'

interface MockWebSocket {
  url: string
  onopen: ((event: Event) => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onerror: ((event: Event) => void) | null
  onclose: ((event: CloseEvent) => void) | null
  close: ReturnType<typeof vi.fn>
}

describe('useServices', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should fetch services on mount', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const { result } = renderHook(() => useServices())

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(mockFetch).toHaveBeenCalledWith('/api/services')
    expect(result.current.services).toEqual(mockServices)
    expect(result.current.error).toBeNull()
  })

  it('should handle fetch errors and use mock data', async () => {
    const mockFetch = vi.fn(() => Promise.reject(new Error('Network error')))
    globalThis.fetch = mockFetch as unknown as typeof fetch
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBeNull() // No error shown when using mock data
    expect(result.current.services.length).toBeGreaterThan(0) // Should have mock services
    expect(consoleSpy).toHaveBeenCalledWith('Backend not available, using mock data')

    consoleSpy.mockRestore()
  })

  it('should handle empty service list', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse([]))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.services).toEqual([])
  })

  it('should set up WebSocket connection', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // WebSocket should be created (checked via constructor call)
    expect(result.current.connected).toBe(true)
  })

  it('should handle WebSocket service updates', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    // Create a custom WebSocket mock class that we can control
    const wsRef: { current: MockWebSocket | null } = { current: null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
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

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // Simulate receiving an update message
    const updatedService = {
      ...mockServices[0],
      local: { ...mockServices[0].local, status: 'stopping' as const },
    }

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(
          createMockWebSocketMessage({
            type: 'update',
            service: updatedService,
          })
        )
      })
    }

    await waitFor(() => {
      const apiService = result.current.services.find(s => s.name === 'api')
      expect(apiService?.local?.status).toBe('stopping')
    })
  })

  it('should handle WebSocket service addition', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const wsRef: { current: MockWebSocket | null } = { current: null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
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

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const initialCount = result.current.services.length

    // Add a new service
    const newService = {
      name: 'new-service',
      language: 'rust',
      framework: 'actix',
      local: {
        status: 'ready' as const,
        health: 'healthy' as const,
        port: 8080,
      },
    }

    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(
          createMockWebSocketMessage({
            type: 'add',
            service: newService,
          })
        )
      })
    }

    await waitFor(() => {
      expect(result.current.services.length).toBe(initialCount + 1)
      expect(result.current.services.find(s => s.name === 'new-service')).toBeDefined()
    })
  })

  it('should handle WebSocket service removal', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const wsRef: { current: MockWebSocket | null } = { current: null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
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

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const initialCount = result.current.services.length

    // Remove a service
    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(
          createMockWebSocketMessage({
            type: 'remove',
            service: mockServices[0],
          })
        )
      })
    }

    await waitFor(() => {
      expect(result.current.services.length).toBe(initialCount - 1)
      expect(result.current.services.find(s => s.name === mockServices[0].name)).toBeUndefined()
    })
  })

  it('should handle malformed WebSocket messages', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const wsRef: { current: MockWebSocket | null } = { current: null }
    class WebSocketMock {
      url: string
      onopen: ((event: Event) => void) | null = null
      onmessage: ((event: MessageEvent) => void) | null = null
      onerror: ((event: Event) => void) | null = null
      onclose: ((event: CloseEvent) => void) | null = null
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

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    const initialServices = [...result.current.services]

    // Send malformed message
    if (wsRef.current?.onmessage) {
      const handler = wsRef.current.onmessage
      act(() => {
        handler(new MessageEvent('message', { data: 'not-valid-json' }))
      })
    }

    await waitFor(() => {
      expect(consoleErrorSpy).toHaveBeenCalledWith(
        'Failed to parse WebSocket message:',
        expect.any(Error)
      )
    })

    // Services should remain unchanged
    expect(result.current.services).toEqual(initialServices)

    consoleErrorSpy.mockRestore()
  })

  it('should provide refetch function', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const closeMock = vi.fn()
    // Mock WebSocket as a class for vitest 4.x compatibility
    class WebSocketMock {
      onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
      onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
      onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
      onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
      close = closeMock
      send = vi.fn()
      constructor(_url: string) {
        // no-op
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    const { result } = renderHook(() => useServices())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(mockFetch).toHaveBeenCalledTimes(1)

    // Call refetch
    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(2)
    })
  })

  it('should close WebSocket on unmount', async () => {
    const mockFetch = vi.fn(() => createMockFetchResponse(mockServices))
    globalThis.fetch = mockFetch as unknown as typeof fetch

    const closeMock = vi.fn()
    // Mock WebSocket as a class for vitest 4.x compatibility
    class WebSocketMock {
      onopen: ((this: WebSocket, ev: Event) => unknown) | null = null
      onmessage: ((this: WebSocket, ev: MessageEvent) => unknown) | null = null
      onerror: ((this: WebSocket, ev: Event) => unknown) | null = null
      onclose: ((this: WebSocket, ev: CloseEvent) => unknown) | null = null
      close = closeMock
      send = vi.fn()
      constructor(_url: string) {
        // no-op
      }
    }
    globalThis.WebSocket = WebSocketMock as unknown as typeof WebSocket

    const { unmount } = renderHook(() => useServices())

    await waitFor(() => {
      // Give time for WebSocket to be created
      expect(true).toBe(true)
    })

    unmount()

    expect(closeMock).toHaveBeenCalled()
  })
})
