import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useHealthStream } from './useHealthStream'
import type { HealthReportEvent, HealthChangeEvent } from '@/types'

// Mock EventSource
class MockEventSource {
  url: string
  readyState: number = 0
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  private listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map()

  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSED = 2

  constructor(url: string) {
    this.url = url
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockEventSource.OPEN
      this.onopen?.()
    }, 0)
  }

  addEventListener(type: string, listener: (event: MessageEvent) => void): void {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, [])
    }
    this.listeners.get(type)?.push(listener)
  }

  removeEventListener(type: string, listener: (event: MessageEvent) => void): void {
    const handlers = this.listeners.get(type)
    if (handlers) {
      const index = handlers.indexOf(listener)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  dispatchEvent(type: string, data: unknown): void {
    const event = { data: JSON.stringify(data) } as MessageEvent
    if (type === 'message' && this.onmessage) {
      this.onmessage(event)
    }
    const handlers = this.listeners.get(type)
    if (handlers) {
      handlers.forEach(handler => handler(event))
    }
  }

  close(): void {
    this.readyState = MockEventSource.CLOSED
  }

  simulateError(): void {
    this.onerror?.()
  }
}

// Store mock instances for test access
let mockEventSourceInstance: MockEventSource | null = null
let mockEventSourceCallCount = 0
let lastEventSourceUrl: string | null = null

// Create a factory function that creates MockEventSource instances
class EventSourceFactory {
  constructor(url: string) {
    mockEventSourceInstance = new MockEventSource(url)
    mockEventSourceCallCount++
    lastEventSourceUrl = url
    return mockEventSourceInstance as unknown as EventSource
  }
}

vi.stubGlobal('EventSource', EventSourceFactory)

describe('useHealthStream', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockEventSourceInstance = null
    mockEventSourceCallCount = 0
    lastEventSourceUrl = null
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should initialize with default values', () => {
    const { result } = renderHook(() => useHealthStream())

    expect(result.current.healthReport).toBeNull()
    expect(result.current.changes).toEqual([])
    expect(result.current.connected).toBe(false)
    expect(result.current.error).toBeNull()
    expect(result.current.lastUpdate).toBeNull()
    expect(result.current.summary).toBeNull()
  })

  it('should connect to SSE endpoint on mount', () => {
    renderHook(() => useHealthStream())

    // Advance timers to allow connection
    act(() => {
      vi.advanceTimersByTime(10)
    })

    expect(lastEventSourceUrl).toBe('/api/health/stream?interval=5s')
    expect(mockEventSourceInstance).not.toBeNull()
  })

  it('should set connected to true on open', () => {
    const { result } = renderHook(() => useHealthStream())

    // Advance timers to trigger connection
    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Manually trigger the open callback since fake timers don't work with setTimeout inside EventSource
    act(() => {
      mockEventSourceInstance?.onopen?.()
    })

    expect(result.current.connected).toBe(true)
  })

  it('should build URL with custom interval', () => {
    renderHook(() => useHealthStream({ interval: 10 }))

    act(() => {
      vi.advanceTimersByTime(10)
    })

    expect(lastEventSourceUrl).toBe('/api/health/stream?interval=10s')
  })

  it('should build URL with service filter', () => {
    renderHook(() => useHealthStream({ services: ['api', 'web'] }))

    act(() => {
      vi.advanceTimersByTime(10)
    })

    expect(lastEventSourceUrl).toBe('/api/health/stream?interval=5s&service=api%2Cweb')
  })

  it('should not connect when disabled', () => {
    renderHook(() => useHealthStream({ enabled: false }))

    act(() => {
      vi.advanceTimersByTime(10)
    })

    expect(mockEventSourceCallCount).toBe(0)
  })

  it('should handle health report event', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    const healthReport: HealthReportEvent = {
      type: 'health',
      timestamp: '2024-11-27T10:30:00Z',
      services: [
        {
          serviceName: 'api',
          status: 'healthy',
          checkType: 'http',
          responseTime: 45000000,
          timestamp: '2024-11-27T10:30:00Z',
        },
      ],
      summary: {
        total: 1,
        healthy: 1,
        degraded: 0,
        unhealthy: 0,
        starting: 0,
        stopped: 0,
        unknown: 0,
        overall: 'healthy',
      },
    }

    act(() => {
      mockEventSourceInstance?.dispatchEvent('message', healthReport)
    })

    expect(result.current.healthReport).toEqual(healthReport)
    expect(result.current.summary).toEqual(healthReport.summary)
    expect(result.current.lastUpdate).not.toBeNull()
  })

  it('should handle health change event', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    const changeEvent: HealthChangeEvent = {
      type: 'health-change',
      timestamp: '2024-11-27T10:30:00Z',
      service: 'api',
      oldStatus: 'healthy',
      newStatus: 'unhealthy',
      reason: 'connection refused',
    }

    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', changeEvent)
    })

    expect(result.current.changes).toHaveLength(1)
    expect(result.current.changes[0]).toEqual(changeEvent)
  })

  it('should handle heartbeat event', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    const initialUpdate = result.current.lastUpdate

    act(() => {
      mockEventSourceInstance?.dispatchEvent('heartbeat', { type: 'heartbeat', timestamp: '2024-11-27T10:30:00Z' })
    })

    expect(result.current.lastUpdate).not.toEqual(initialUpdate)
  })

  it('should get service health by name', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    const healthReport: HealthReportEvent = {
      type: 'health',
      timestamp: '2024-11-27T10:30:00Z',
      services: [
        {
          serviceName: 'api',
          status: 'healthy',
          checkType: 'http',
          responseTime: 45000000,
          timestamp: '2024-11-27T10:30:00Z',
        },
        {
          serviceName: 'web',
          status: 'unhealthy',
          checkType: 'tcp',
          responseTime: 0,
          timestamp: '2024-11-27T10:30:00Z',
          error: 'connection refused',
        },
      ],
      summary: {
        total: 2,
        healthy: 1,
        degraded: 0,
        unhealthy: 1,
        starting: 0,
        stopped: 0,
        unknown: 0,
        overall: 'unhealthy',
      },
    }

    act(() => {
      mockEventSourceInstance?.dispatchEvent('message', healthReport)
    })

    const apiHealth = result.current.getServiceHealth('api')
    expect(apiHealth?.serviceName).toBe('api')
    expect(apiHealth?.status).toBe('healthy')

    const webHealth = result.current.getServiceHealth('web')
    expect(webHealth?.serviceName).toBe('web')
    expect(webHealth?.status).toBe('unhealthy')

    const unknownHealth = result.current.getServiceHealth('unknown')
    expect(unknownHealth).toBeUndefined()
  })

  it('should limit stored changes', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Dispatch 60 change events (more than MAX_CHANGES_TO_KEEP = 50)
    for (let i = 0; i < 60; i++) {
      act(() => {
        mockEventSourceInstance?.dispatchEvent('health-change', {
          type: 'health-change',
          timestamp: new Date().toISOString(),
          service: `service-${i}`,
          oldStatus: 'healthy',
          newStatus: 'unhealthy',
        })
      })
    }

    expect(result.current.changes.length).toBeLessThanOrEqual(50)
  })

  it('should handle connection error and attempt reconnection', () => {
    const { result } = renderHook(() => useHealthStream({ reconnectDelay: 1000, maxReconnectAttempts: 3 }))

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Simulate error
    act(() => {
      mockEventSourceInstance?.simulateError()
    })

    expect(result.current.connected).toBe(false)
    expect(result.current.error).toContain('Reconnecting')

    // Advance timer to trigger reconnect
    act(() => {
      vi.advanceTimersByTime(2000)
    })

    // Should have attempted reconnection
    expect(mockEventSourceCallCount).toBe(2)
  })

  it('should cleanup on unmount', () => {
    const { unmount } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    const instance = mockEventSourceInstance
    expect(instance).not.toBeNull()

    unmount()

    expect(instance?.readyState).toBe(MockEventSource.CLOSED)
  })

  it('should allow manual reconnection', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Simulate disconnect
    act(() => {
      mockEventSourceInstance?.close()
    })

    // Manual reconnect
    act(() => {
      result.current.reconnect()
    })

    act(() => {
      vi.advanceTimersByTime(10)
    })

    expect(mockEventSourceCallCount).toBe(2)
  })

  it('should parse malformed JSON gracefully', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Send malformed data
    act(() => {
      if (mockEventSourceInstance?.onmessage) {
        mockEventSourceInstance.onmessage({ data: 'not-valid-json' } as MessageEvent)
      }
    })

    // Should not crash, healthReport should remain null
    expect(result.current.healthReport).toBeNull()
    expect(consoleSpy).toHaveBeenCalled()

    consoleSpy.mockRestore()
  })

  it('should get latest change for a service', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Add multiple changes for different services
    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', {
        type: 'health-change',
        timestamp: '2024-11-27T10:30:00Z',
        service: 'api',
        oldStatus: 'healthy',
        newStatus: 'unhealthy',
      })
    })

    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', {
        type: 'health-change',
        timestamp: '2024-11-27T10:31:00Z',
        service: 'web',
        oldStatus: 'healthy',
        newStatus: 'unhealthy',
      })
    })

    const apiChange = result.current.getLatestChange('api')
    expect(apiChange?.service).toBe('api')
    expect(apiChange?.newStatus).toBe('unhealthy')

    const unknownChange = result.current.getLatestChange('unknown')
    expect(unknownChange).toBeUndefined()
  })

  it('should detect service recovery', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Service goes unhealthy
    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', {
        type: 'health-change',
        timestamp: '2024-11-27T10:30:00Z',
        service: 'api',
        oldStatus: 'healthy',
        newStatus: 'unhealthy',
      })
    })

    expect(result.current.hasRecovered('api')).toBe(false)

    // Service recovers
    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', {
        type: 'health-change',
        timestamp: '2024-11-27T10:31:00Z',
        service: 'api',
        oldStatus: 'unhealthy',
        newStatus: 'healthy',
      })
    })

    expect(result.current.hasRecovered('api')).toBe(true)
  })

  it('should clear changes', () => {
    const { result } = renderHook(() => useHealthStream())

    act(() => {
      vi.advanceTimersByTime(10)
    })

    // Add a change
    act(() => {
      mockEventSourceInstance?.dispatchEvent('health-change', {
        type: 'health-change',
        timestamp: '2024-11-27T10:30:00Z',
        service: 'api',
        oldStatus: 'healthy',
        newStatus: 'unhealthy',
      })
    })

    expect(result.current.changes).toHaveLength(1)

    // Clear changes
    act(() => {
      result.current.clearChanges()
    })

    expect(result.current.changes).toHaveLength(0)
  })
})
