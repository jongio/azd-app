import { renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useLogsStream } from './useLogsStream'
import type { LogEntry } from '@/components/LogsPane'
import type { LogMode } from '@/components/ModeToggle'
import { resetManagers } from './useSharedLogStream'

// Mock useBackendConnection
vi.mock('./useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true }),
}))

describe('useLogsStream', () => {
  interface MockWebSocketInstance {
    url: string
    onopen: ((ev: Event) => void) | null
    onmessage: ((ev: MessageEvent) => void) | null
    onerror: ((ev: Event) => void) | null
    onclose: ((ev: CloseEvent) => void) | null
    readyState: number
    close: ReturnType<typeof vi.fn>
    send: ReturnType<typeof vi.fn>
  }
  
  let webSocketInstances: MockWebSocketInstance[] = []
  let originalWebSocket: typeof WebSocket

  beforeEach(() => {
    vi.useFakeTimers()
    webSocketInstances = []
    
    // Mock WebSocket constructor
    originalWebSocket = globalThis.WebSocket
    
    // Create a proper mock class with spy on close
    const MockWebSocket = vi.fn().mockImplementation(function(this: MockWebSocketInstance, url: string) {
      const instance: MockWebSocketInstance = {
        url,
        onopen: null,
        onmessage: null,
        onerror: null,
        onclose: null,
        readyState: 0, // CONNECTING
        close: vi.fn(function(this: MockWebSocketInstance, code?: number) {
          this.readyState = 3 // CLOSED
          if (this.onclose) {
            this.onclose({ code: code ?? 1000 } as CloseEvent)
          }
        }),
        send: vi.fn(),
      }
      
      webSocketInstances.push(instance)
      Object.assign(this, instance)
      return instance
    })
    
    globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
    
    // Mock fetch
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
      text: () => Promise.resolve(''),
    })
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    globalThis.WebSocket = originalWebSocket
    vi.restoreAllMocks()
    resetManagers()
  })

  const createParams = (overrides?: Partial<Parameters<typeof useLogsStream>[0]>) => ({
    serviceName: 'test-service',
    fetchKey: 'local:stream',
    logMode: 'local' as LogMode,
    timeRange: { preset: '15m' as const },
    azureRealtime: false,
    refreshTrigger: 0,
    isPausedRef: { current: false },
    lastClearTimeRef: { current: Date.now() - 1000 }, // Initialize to 1s in the past
    setLogs: vi.fn(),
    setErrorMessage: vi.fn(),
    onFetchSettled: vi.fn(),
    ...overrides,
  })

  describe('WebSocket connection management', () => {
    it('creates WebSocket on mount in local mode', () => {
      const params = createParams()
      renderHook(() => useLogsStream(params))
      
      expect(globalThis.WebSocket).toHaveBeenCalledWith(
        expect.stringContaining('/api/logs/stream')
      )
      expect(webSocketInstances).toHaveLength(1)
    })

    it('creates WebSocket with azure endpoint in azure realtime mode', () => {
      const params = createParams({
        logMode: 'azure',
        azureRealtime: true,
        fetchKey: 'azure:15m::realtime',
      })
      renderHook(() => useLogsStream(params))
      
      expect(globalThis.WebSocket).toHaveBeenCalledWith(
        expect.stringContaining('/api/azure/logs/stream?realtime=true')
      )
    })

    it('does not create WebSocket in azure polling mode', () => {
      const params = createParams({
        logMode: 'azure',
        azureRealtime: false,
        fetchKey: 'azure:15m::poll',
      })
      renderHook(() => useLogsStream(params))
      
      expect(globalThis.WebSocket).not.toHaveBeenCalled()
    })

  })

  describe('WebSocket message handling', () => {
    it('ignores messages when paused', () => {
      const setLogs = vi.fn()
      const isPausedRef = { current: true }
      const params = createParams({ setLogs, isPausedRef })
      renderHook(() => useLogsStream(params))
      
      const ws = webSocketInstances[0]
      const logEntry: LogEntry = {
        service: 'test-service',
        message: 'Test log',
        level: 1,
        timestamp: new Date().toISOString(),
        isStderr: false,
      }
      
      if (ws.onmessage) {
        ws.onmessage({ data: JSON.stringify(logEntry) } as MessageEvent)
      }
      
      expect(setLogs).not.toHaveBeenCalled()
    })
  })

  describe('onFetchSettled callback', () => {
    it('calls onFetchSettled only after first fetch completes', async () => {
      const onFetchSettled = vi.fn()
      const params = createParams({ onFetchSettled })
      
      renderHook(() => useLogsStream(params))
      
      // Should not be called synchronously on mount
      expect(onFetchSettled).not.toHaveBeenCalled()
      
      // Wait for fetch to complete
      await vi.runAllTimersAsync()
      
      // Should be called after first fetch (may be called multiple times due to retries/WebSocket reconnects)
      expect(onFetchSettled).toHaveBeenCalled()
    })

    it('does not call onFetchSettled immediately when fetchKey changes', async () => {
      const onFetchSettled = vi.fn()
      const params = createParams({ onFetchSettled, fetchKey: 'local:stream' })
      
      const { rerender } = renderHook((props) => useLogsStream(props), {
        initialProps: params,
      })
      
      // Wait for initial fetch
      await vi.runAllTimersAsync()
      expect(onFetchSettled).toHaveBeenCalled()
      
      const initialCallCount = onFetchSettled.mock.calls.length
      
      // Change fetchKey (e.g., switching time range in Azure mode)
      rerender(createParams({ 
        onFetchSettled, 
        fetchKey: 'azure:30m::poll',
        logMode: 'azure',
      }))
      
      // Should NOT be called immediately after fetchKey change (still same count)
      expect(onFetchSettled).toHaveBeenCalledTimes(initialCallCount)
      
      // Wait for new fetch to complete
      await vi.runAllTimersAsync()
      
      // Should be called at least once more after the new fetch completes
      expect(onFetchSettled.mock.calls.length).toBeGreaterThan(initialCallCount)
    })
  })
})
