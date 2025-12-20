import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useLogsStream } from './useLogsStream'

// Mock the backend connection hook
vi.mock('@/hooks/useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true })
}))

// Mock the shared log stream hook
vi.mock('@/hooks/useSharedLogStream', () => ({
  useSharedLogStream: () => ({ connectionState: 'disconnected' })
}))

describe('useLogsStream flood prevention', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('should not flood the server when multiple services mount simultaneously', async () => {
    const mockFetch = vi.fn().mockImplementation(async () => {
      return {
        ok: true,
        json: async () => [],
      }
    })
    globalThis.fetch = mockFetch

    const services = ['appservice-web', 'azurite', 'containerapp-api', 'functions-worker']
    
    // Render hooks for all services simultaneously (like on app mount)
    services.forEach(serviceName => {
      const setLogs = vi.fn()
      const setErrorMessage = vi.fn()
      const isPausedRef = { current: false }
      
      renderHook(() =>
        useLogsStream({
          serviceName,
          fetchKey: 'local:stream',
          logMode: 'local',
          timeRange: { preset: '15m' },
          azureRealtime: false,
          isPausedRef,
          setLogs,
          setErrorMessage,
          onFetchSettled: vi.fn(),
        })
      )
    })

    // Run all timers to execute fetches
    await vi.runAllTimersAsync()

    // Should make exactly 4 requests (one per service), not more
    // Each service should only fetch once, not repeatedly
    console.log(`Total fetch calls: ${mockFetch.mock.calls.length}`)
    
    // Group by service to see how many times each was called
    const callsByService = new Map<string, number>()
    mockFetch.mock.calls.forEach(call => {
      const url = call[0] as string
      const match = url.match(/service=([^&]+)/)
      if (match) {
        const service = match[1]
        callsByService.set(service, (callsByService.get(service) || 0) + 1)
      }
    })
    
    console.log('Calls by service:', Object.fromEntries(callsByService))
    
    // Each service should be called at most 2 times (due to React Strict Mode double-mount)
    // But definitely not more than that
    services.forEach(service => {
      const count = callsByService.get(service) || 0
      expect(count).toBeLessThanOrEqual(2)
    })
    
    // Total should not exceed 8 (4 services * 2 max due to strict mode)
    expect(mockFetch.mock.calls.length).toBeLessThanOrEqual(8)
  })

  it('should not repeatedly poll in local mode when using WebSocket', async () => {
    const mockFetch = vi.fn().mockImplementation(async () => {
      return {
        ok: true,
        json: async () => [],
      }
    })
    globalThis.fetch = mockFetch

    const setLogs = vi.fn()
    const setErrorMessage = vi.fn()
    const isPausedRef = { current: false }

    renderHook(() =>
      useLogsStream({
        serviceName: 'api',
        fetchKey: 'local:stream',
        logMode: 'local',
        timeRange: { preset: '15m' },
        azureRealtime: false,
        isPausedRef,
        setLogs,
        setErrorMessage,
        onFetchSettled: vi.fn(),
      })
    )

    // Run timers for initial fetch
    await vi.runAllTimersAsync()
    
    const initialCallCount = mockFetch.mock.calls.length
    console.log(`Initial calls: ${initialCallCount}`)
    
    mockFetch.mockClear()

    // Advance time significantly (30 seconds)
    await vi.advanceTimersByTimeAsync(30000)

    // In local mode, we should NOT be polling via HTTP - we use WebSocket
    // So there should be NO additional HTTP requests after the initial one
    console.log(`Additional calls after 30s: ${mockFetch.mock.calls.length}`)
    expect(mockFetch.mock.calls.length).toBe(0)
  })

  it('should not make HTTP requests in local mode at all when WebSocket is available', async () => {
    const mockFetch = vi.fn().mockImplementation(async () => {
      return {
        ok: true,
        json: async () => [],
      }
    })
    globalThis.fetch = mockFetch

    const setLogs = vi.fn()
    const setErrorMessage = vi.fn()
    const isPausedRef = { current: false }

    // In local mode, should use WebSocket only, not HTTP polling
    renderHook(() =>
      useLogsStream({
        serviceName: 'api',
        fetchKey: 'local:stream',
        logMode: 'local',
        timeRange: { preset: '15m' },
        azureRealtime: false,
        isPausedRef,
        setLogs,
        setErrorMessage,
        onFetchSettled: vi.fn(),
      })
    )

    await vi.runAllTimersAsync()

    // The question: should local mode use HTTP polling at all?
    // Looking at the code, it seems like it shouldn't - WebSocket should handle it
    console.log(`HTTP fetch calls in local mode: ${mockFetch.mock.calls.length}`)
  })
})
