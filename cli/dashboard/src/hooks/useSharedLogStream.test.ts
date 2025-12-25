/**
 * Tests for useSharedLogStream hook and SharedLogStreamManager
 * Validates WebSocket connection sharing, subscription management, and lifecycle
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { useSharedLogStream, resetManagers } from './useSharedLogStream'
import type { LogEntry } from '@/components/LogsPane'

// Mock useBackendConnection
vi.mock('./useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true }),
}))

describe('useSharedLogStream', () => {
  interface MockWebSocketInstance {
    url: string
    addEventListener: ReturnType<typeof vi.fn>
    removeEventListener: ReturnType<typeof vi.fn>
    send: ReturnType<typeof vi.fn>
    close: ReturnType<typeof vi.fn>
    readyState: number
    onopen: ((ev: Event) => void) | null
    onmessage: ((ev: MessageEvent) => void) | null
    onerror: ((ev: Event) => void) | null
    onclose: ((ev: CloseEvent) => void) | null
  }

  let webSocketInstances: MockWebSocketInstance[] = []
  let originalWebSocket: typeof WebSocket

  beforeEach(() => {
    vi.useFakeTimers()
    webSocketInstances = []

    // Mock WebSocket
    originalWebSocket = globalThis.WebSocket

    const MockWebSocket = vi.fn().mockImplementation(function (this: MockWebSocketInstance, url: string) {
      const listeners = new Map<string, Set<(event: Event | MessageEvent | CloseEvent) => void>>()

      const instance: MockWebSocketInstance = {
        url,
        readyState: 0, // CONNECTING
        onopen: null,
        onmessage: null,
        onerror: null,
        onclose: null,
        addEventListener: vi.fn((event: string, handler: (e: Event | MessageEvent | CloseEvent) => void) => {
          if (!listeners.has(event)) {
            listeners.set(event, new Set())
          }
          listeners.get(event)!.add(handler)
        }),
        removeEventListener: vi.fn((event: string, handler: (e: Event | MessageEvent | CloseEvent) => void) => {
          listeners.get(event)?.delete(handler)
        }),
        send: vi.fn(),
        close: vi.fn(function (this: MockWebSocketInstance, code?: number) {
          this.readyState = 3 // CLOSED
          const closeEvent = { code: code ?? 1000 } as CloseEvent
          listeners.get('close')?.forEach((handler) => handler(closeEvent))
        }),
      }

      // Helper to simulate events
      ;(instance as any)._trigger = (event: string, data?: any) => {
        if (event === 'open') {
          instance.readyState = 1 // OPEN
          listeners.get('open')?.forEach((handler) => handler({} as Event))
        } else if (event === 'message') {
          listeners.get('message')?.forEach((handler) =>
            handler({ data: JSON.stringify(data) } as MessageEvent)
          )
        } else if (event === 'close') {
          instance.readyState = 3 // CLOSED
          listeners.get('close')?.forEach((handler) => handler({ code: data?.code ?? 1000 } as CloseEvent))
        } else if (event === 'error') {
          listeners.get('error')?.forEach((handler) => handler({} as Event))
        }
      }

      webSocketInstances.push(instance)
      Object.assign(this, instance)
      return instance
    })

    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
    ;(globalThis.WebSocket as any).CONNECTING = 0
    ;(globalThis.WebSocket as any).OPEN = 1
    ;(globalThis.WebSocket as any).CLOSING = 2
    ;(globalThis.WebSocket as any).CLOSED = 3

    // Mock location
    Object.defineProperty(globalThis, 'location', {
      value: {
        protocol: 'http:',
        host: 'localhost:3000',
      },
      writable: true,
    })
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    globalThis.WebSocket = originalWebSocket
    vi.restoreAllMocks()
    resetManagers()
  })

  const createParams = (overrides?: Partial<Parameters<typeof useSharedLogStream>[0]>) => ({
    serviceName: 'test-service',
    enabled: true,
    mode: 'local' as const,
    onLogEntry: vi.fn(),
    ...overrides,
  })

  describe('connection management', () => {
    it('should create WebSocket when enabled', () => {
      renderHook(() => useSharedLogStream(createParams()))

      // Run timers to allow async initialization
      act(() => {
        vi.runAllTimers()
      })

      expect(globalThis.WebSocket).toHaveBeenCalled()
      expect(webSocketInstances).toHaveLength(1)
    })

    it('should not create WebSocket when disabled', () => {
      renderHook(() => useSharedLogStream(createParams({ enabled: false })))

      act(() => {
        vi.runAllTimers()
      })

      expect(globalThis.WebSocket).not.toHaveBeenCalled()
    })

    it('should use local endpoint for local mode', () => {
      renderHook(() => useSharedLogStream(createParams({ mode: 'local' })))

      act(() => {
        vi.runAllTimers()
      })

      expect(globalThis.WebSocket).toHaveBeenCalledWith(
        expect.stringContaining('ws://localhost:3000/api/logs/stream')
      )
    })

    it('should use azure endpoint for azure mode', () => {
      renderHook(() => useSharedLogStream(createParams({ mode: 'azure' })))

      act(() => {
        vi.runAllTimers()
      })

      expect(globalThis.WebSocket).toHaveBeenCalledWith(
        expect.stringContaining('ws://localhost:3000/api/azure/logs/stream?realtime=true')
      )
    })

    it('should share connection across multiple subscribers to same mode', () => {
      const params1 = createParams({ serviceName: 'service-1' })
      const params2 = createParams({ serviceName: 'service-2' })

      renderHook(() => useSharedLogStream(params1))
      renderHook(() => useSharedLogStream(params2))

      act(() => {
        vi.runAllTimers()
      })

      // Should only create one WebSocket
      expect(webSocketInstances).toHaveLength(1)
    })

    it('should create separate connections for different modes', () => {
      const localParams = createParams({ mode: 'local' })
      const azureParams = createParams({ mode: 'azure' })

      renderHook(() => useSharedLogStream(localParams))
      renderHook(() => useSharedLogStream(azureParams))

      act(() => {
        vi.runAllTimers()
      })

      // Should create two WebSockets (one per mode)
      expect(webSocketInstances).toHaveLength(2)
    })
  })

  describe('connection lifecycle', () => {
    it('should set connection state to connecting initially', () => {
      const { result } = renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      // After initialization, should be connecting
      expect(result.current.connectionState).toBe('connecting')
    })

    it('should set connection state to connected on open', async () => {
      const { result } = renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      // Simulate open
      act(() => {
        ;(ws as any)._trigger('open')
      })

      await waitFor(() => {
        expect(result.current.connectionState).toBe('connected')
      })
    })

    it('should set connection state to disconnected when disabled', () => {
      const { result, rerender } = renderHook(
        ({ enabled }) => useSharedLogStream(createParams({ enabled })),
        { initialProps: { enabled: true } }
      )

      act(() => {
        vi.runAllTimers()
      })

      // Disable
      rerender({ enabled: false })

      expect(result.current.connectionState).toBe('disconnected')
    })

    it('should close connection when no more subscribers', () => {
      const { unmount } = renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      unmount()

      // Wait for debounced disconnect
      act(() => {
        vi.advanceTimersByTime(200)
      })

      expect(ws.close).toHaveBeenCalled()
    })

    it('should not close connection if other subscribers exist', () => {
      const { unmount: unmount1 } = renderHook(() =>
        useSharedLogStream(createParams({ serviceName: 'service-1' }))
      )
      renderHook(() => useSharedLogStream(createParams({ serviceName: 'service-2' })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      unmount1()

      act(() => {
        vi.advanceTimersByTime(200)
      })

      // Should not close (service-2 still subscribed)
      expect(ws.close).not.toHaveBeenCalled()
    })
  })

  describe('message handling', () => {
    it('should call onLogEntry when message received', async () => {
      const onLogEntry = vi.fn()
      renderHook(() => useSharedLogStream(createParams({ onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]
      const logEntry: LogEntry = {
        service: 'test-service',
        message: 'Test log',
        level: 1,
        timestamp: '2025-12-25T12:00:00Z',
        isStderr: false,
      }

      // Simulate open then message
      act(() => {
        ;(ws as any)._trigger('open')
        ;(ws as any)._trigger('message', logEntry)
      })

      await waitFor(() => {
        expect(onLogEntry).toHaveBeenCalledWith(logEntry)
      })
    })

    it('should only deliver messages for matching service', async () => {
      const onLogEntry = vi.fn()
      renderHook(() => useSharedLogStream(createParams({ serviceName: 'service-1', onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]
      const log1: LogEntry = {
        service: 'service-1',
        message: 'For service 1',
        level: 1,
        timestamp: '2025-12-25T12:00:00Z',
        isStderr: false,
      }
      const log2: LogEntry = {
        service: 'service-2',
        message: 'For service 2',
        level: 1,
        timestamp: '2025-12-25T12:00:00Z',
        isStderr: false,
      }

      act(() => {
        ;(ws as any)._trigger('open')
        ;(ws as any)._trigger('message', log1)
        ;(ws as any)._trigger('message', log2)
      })

      await waitFor(() => {
        expect(onLogEntry).toHaveBeenCalledWith(log1)
      })

      // Should not receive service-2 logs
      expect(onLogEntry).toHaveBeenCalledTimes(1)
    })

    it('should handle batched messages (array)', async () => {
      const onLogEntry = vi.fn()
      renderHook(() => useSharedLogStream(createParams({ onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]
      const logs: LogEntry[] = [
        {
          service: 'test-service',
          message: 'Log 1',
          level: 1,
          timestamp: '2025-12-25T12:00:00Z',
          isStderr: false,
        },
        {
          service: 'test-service',
          message: 'Log 2',
          level: 1,
          timestamp: '2025-12-25T12:00:01Z',
          isStderr: false,
        },
      ]

      act(() => {
        ;(ws as any)._trigger('open')
        ;(ws as any)._trigger('message', logs)
      })

      await waitFor(() => {
        expect(onLogEntry).toHaveBeenCalledTimes(2)
      })
    })

    it('should ignore status messages', async () => {
      const onLogEntry = vi.fn()
      renderHook(() => useSharedLogStream(createParams({ onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
        ;(ws as any)._trigger('message', { type: 'status', status: 'healthy' })
      })

      await new Promise((resolve) => setTimeout(resolve, 100))

      expect(onLogEntry).not.toHaveBeenCalled()
    })

    it('should handle invalid JSON gracefully', async () => {
      const onLogEntry = vi.fn()
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      renderHook(() => useSharedLogStream(createParams({ onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
        // Manually trigger with invalid JSON
        const listeners = (ws.addEventListener as any).mock.calls
          .filter(([event]: [string]) => event === 'message')
          .map(([, handler]: [string, Function]) => handler)
        listeners.forEach((handler: Function) => {
          handler({ data: 'invalid json' })
        })
      })

      await new Promise((resolve) => setTimeout(resolve, 50))

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[SharedLogStream] Failed to parse message'),
        expect.any(Error)
      )

      consoleSpy.mockRestore()
    })
  })

  describe('reconnection', () => {
    it('should attempt reconnection on close', () => {
      renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      // Simulate close
      act(() => {
        ;(ws as any)._trigger('close', { code: 1006 })
      })

      const initialCount = webSocketInstances.length

      // Advance past reconnect delay
      act(() => {
        vi.advanceTimersByTime(2000)
      })

      // Should have attempted reconnection
      expect(webSocketInstances.length).toBeGreaterThan(initialCount)
    })

    it('should use exponential backoff for reconnections', () => {
      renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      // First connection
      const ws1 = webSocketInstances[0]

      // Close
      act(() => {
        ;(ws1 as any)._trigger('close', { code: 1006 })
      })

      // First reconnect (short delay)
      act(() => {
        vi.advanceTimersByTime(1500) // ~1s backoff
      })

      expect(webSocketInstances.length).toBe(2)

      // Close again
      const ws2 = webSocketInstances[1]
      act(() => {
        ;(ws2 as any)._trigger('close', { code: 1006 })
      })

      // Second reconnect (longer delay)
      act(() => {
        vi.advanceTimersByTime(2500) // ~2s backoff
      })

      expect(webSocketInstances.length).toBe(3)
    })

    it('should stop reconnecting after max attempts', () => {
      const { result } = renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      // Simulate multiple failures
      for (let i = 0; i < 10; i++) {
        const ws = webSocketInstances[webSocketInstances.length - 1]
        act(() => {
          ;(ws as any)._trigger('close', { code: 1006 })
          vi.advanceTimersByTime(35000) // Max backoff
        })
      }

      // Should stop attempting and set error state
      expect(result.current.connectionState).toBe('error')
    })

    it('should not reconnect on clean close (code 1000)', () => {
      renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('close', { code: 1000 })
      })

      const initialCount = webSocketInstances.length

      act(() => {
        vi.advanceTimersByTime(5000)
      })

      // Should not reconnect
      expect(webSocketInstances.length).toBe(initialCount)
    })
  })

  describe('heartbeat', () => {
    it('should start heartbeat on connection', () => {
      renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
      })

      // Heartbeat should be active (tested by advancing time)
      // If no messages received within timeout, connection should close
      act(() => {
        vi.advanceTimersByTime(35000) // Past heartbeat interval + timeout
      })

      // Should have closed due to heartbeat timeout
      expect(ws.close).toHaveBeenCalled()
    })

    it('should reset heartbeat timeout on message', async () => {
      renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
      })

      // Advance partway through heartbeat
      act(() => {
        vi.advanceTimersByTime(20000)
      })

      // Receive a message (resets timeout)
      act(() => {
        ;(ws as any)._trigger('message', {
          service: 'test-service',
          message: 'Keepalive',
          level: 1,
          timestamp: '2025-12-25T12:00:00Z',
          isStderr: false,
        })
      })

      // Advance more time (should not timeout now)
      act(() => {
        vi.advanceTimersByTime(20000)
      })

      // Should still be open
      expect(ws.close).not.toHaveBeenCalled()
    })
  })

  describe('cleanup', () => {
    it('should clean up on unmount', () => {
      const { unmount } = renderHook(() => useSharedLogStream(createParams()))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      unmount()

      act(() => {
        vi.advanceTimersByTime(200)
      })

      expect(ws.close).toHaveBeenCalled()
    })

    it('should not call onLogEntry after unmount', async () => {
      const onLogEntry = vi.fn()
      const { unmount } = renderHook(() => useSharedLogStream(createParams({ onLogEntry })))

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
      })

      unmount()

      // Try to send message after unmount
      act(() => {
        ;(ws as any)._trigger('message', {
          service: 'test-service',
          message: 'After unmount',
          level: 1,
          timestamp: '2025-12-25T12:00:00Z',
          isStderr: false,
        })
      })

      await new Promise((resolve) => setTimeout(resolve, 50))

      expect(onLogEntry).not.toHaveBeenCalled()
    })
  })

  describe('callback updates', () => {
    it('should use updated callback without reconnecting', async () => {
      const callback1 = vi.fn()
      const callback2 = vi.fn()

      const { rerender } = renderHook(
        ({ onLogEntry }) => useSharedLogStream(createParams({ onLogEntry })),
        { initialProps: { onLogEntry: callback1 } }
      )

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
      })

      const wsCount = webSocketInstances.length

      // Update callback
      rerender({ onLogEntry: callback2 })

      // Send message
      const logEntry: LogEntry = {
        service: 'test-service',
        message: 'Test',
        level: 1,
        timestamp: '2025-12-25T12:00:00Z',
        isStderr: false,
      }

      act(() => {
        ;(ws as any)._trigger('message', logEntry)
      })

      await waitFor(() => {
        expect(callback2).toHaveBeenCalledWith(logEntry)
      })

      expect(callback1).not.toHaveBeenCalled()
      expect(webSocketInstances.length).toBe(wsCount) // No reconnection
    })
  })

  describe('service switching', () => {
    it('should handle service name changes', async () => {
      const onLogEntry = vi.fn()
      const { rerender } = renderHook(
        ({ serviceName }) => useSharedLogStream(createParams({ serviceName, onLogEntry })),
        { initialProps: { serviceName: 'service-1' } }
      )

      act(() => {
        vi.runAllTimers()
      })

      const ws = webSocketInstances[0]

      act(() => {
        ;(ws as any)._trigger('open')
      })

      // Switch service
      rerender({ serviceName: 'service-2' })

      // Send message for new service
      const logEntry: LogEntry = {
        service: 'service-2',
        message: 'For service 2',
        level: 1,
        timestamp: '2025-12-25T12:00:00Z',
        isStderr: false,
      }

      act(() => {
        ;(ws as any)._trigger('message', logEntry)
      })

      await waitFor(() => {
        expect(onLogEntry).toHaveBeenCalledWith(logEntry)
      })
    })
  })

  describe('mode switching', () => {
    it('should create new connection when mode changes', () => {
      const { rerender } = renderHook(
        ({ mode }) => useSharedLogStream(createParams({ mode })),
        { initialProps: { mode: 'local' as const } }
      )

      act(() => {
        vi.runAllTimers()
      })

      expect(webSocketInstances).toHaveLength(1)
      expect(webSocketInstances[0].url).toContain('/api/logs/stream')

      // Switch to azure mode
      rerender({ mode: 'azure' as const })

      act(() => {
        vi.runAllTimers()
      })

      expect(webSocketInstances).toHaveLength(2)
      expect(webSocketInstances[1].url).toContain('/api/azure/logs/stream')
    })
  })
})
