/**
 * Tests for useAzureConnectionStatus hook
 * Validates mode endpoint request handling and flood prevention
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useAzureConnectionStatus } from './useAzureConnectionStatus'
import type { LogMode } from '@/components/ModeToggle'

// Mock fetch
const createMockFetchResponse = <T,>(data: T, ok = true, status = 200) => {
  return Promise.resolve({
    ok,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response)
}

// Helper for delayed response (reduces function nesting in tests)
const createDelayedResponse = (data: unknown, delay: number): Promise<Response> => {
  return new Promise<Response>((resolve) => {
    setTimeout(() => resolve(createMockFetchResponse(data)), delay)
  })
}

// Helper for never-resolving promise (for cleanup tests)
const createNeverResolvingPromise = (): Promise<Response> => {
  return new Promise<Response>(() => {
    // Intentionally never resolves
  })
}

describe('useAzureConnectionStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial fetch', () => {
    it('should not fetch mode automatically (manual call required)', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      renderHook(() => useAzureConnectionStatus())

      // Should not fetch automatically
      await new Promise((resolve) => setTimeout(resolve, 50))
      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('should fetch mode when fetchAzureStatus is called', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(mockFetch).toHaveBeenCalledWith('/api/mode', expect.objectContaining({
        signal: expect.any(AbortSignal) as unknown
      }) as RequestInit)
    })

    it('should parse mode response correctly', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
          azureRealtime: true,
          connectionMessage: 'Connected to Azure',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      await waitFor(() => {
        expect(result.current.logMode).toBe('azure')
        expect(result.current.azureEnabled).toBe(true)
        expect(result.current.azureStatus).toBe('connected')
        expect(result.current.azureConnectionMessage).toBe('Connected to Azure')
      })
    })
  })

  describe('concurrent request prevention', () => {
    it('should not make concurrent requests to mode endpoint', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      // Fire multiple rapid requests
      await act(async () => {
        const promises = [
          result.current.fetchAzureStatus(),
          result.current.fetchAzureStatus(),
          result.current.fetchAzureStatus(),
          result.current.fetchAzureStatus(),
        ]
        await Promise.all(promises)
      })

      // Should only make 1 request (others skipped due to in-flight guard)
      expect(mockFetch).toHaveBeenCalledTimes(1)
    })

    it('should prevent flooding when called repeatedly', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      // Simulate rapid polling (like services update triggers)
      // Fire all calls concurrently without waiting
      await act(async () => {
        const promises = []
        for (let i = 0; i < 10; i++) {
          promises.push(result.current.fetchAzureStatus())
        }
        await Promise.all(promises)
      })

      // Should make very few requests (most are skipped by guard)
      // Realistically 1-2 requests max (one completes before next starts)
      expect(mockFetch.mock.calls.length).toBeLessThanOrEqual(2)
    })

    it('should allow new request after previous completes', async () => {
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      // First request
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(mockFetch).toHaveBeenCalledTimes(1)

      // Second request after first completes
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(mockFetch).toHaveBeenCalledTimes(2)
    })
  })

  describe('mode switching', () => {
    it('should call PUT /api/mode when changing mode', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({ success: true })
        }
        return createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      act(() => {
        void result.current.handleLogModeChange('azure')
      })

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          '/api/mode',
          expect.objectContaining({
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mode: 'azure' }),
          })
        )
      })
    })

    it('should update mode after successful switch', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({ success: true })
        }
        return createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      act(() => {
        void result.current.handleLogModeChange('azure')
      })

      await waitFor(() => {
        expect(result.current.logMode).toBe('azure')
      })
    })

    it('should set switching state during mode change', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createDelayedResponse({ success: true }, 100)
        }
        return createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      act(() => {
        void result.current.handleLogModeChange('azure')
      })

      // Should be switching immediately
      expect(result.current.isModeSwitching).toBe(true)

      await waitFor(
        () => {
          expect(result.current.isModeSwitching).toBe(false)
        },
        { timeout: 2000 }
      )
    })
  })

  describe('error handling', () => {
    it('should handle fetch errors gracefully', async () => {
      const mockFetch = vi.fn(() => Promise.reject(new Error('Network error')))
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      // Should not throw, status should remain as initial
      expect(result.current.azureStatus).toBe('disabled')
      expect(consoleSpy).not.toHaveBeenCalled() // Network errors are silent

      consoleSpy.mockRestore()
    })

    it('should handle non-OK responses', async () => {
      const mockFetch = vi.fn(() => createMockFetchResponse({}, false, 500))
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[useAzureConnectionStatus] Failed to fetch mode: 500')
      )

      consoleSpy.mockRestore()
    })

    it('should ignore abort errors', async () => {
      const mockFetch = vi.fn(() => Promise.reject(new DOMException('Aborted', 'AbortError')))
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      // Should not log abort errors
      expect(consoleSpy).not.toHaveBeenCalled()
      expect(consoleErrorSpy).not.toHaveBeenCalled()

      consoleSpy.mockRestore()
      consoleErrorSpy.mockRestore()
    })
  })

  describe('cleanup', () => {
    it('should abort in-flight request on unmount', async () => {
      let abortCalled = false
      const markAbortCalled = () => { abortCalled = true }
      
      const mockFetch = vi.fn((_url, options) => {
        const signal = (options as RequestInit)?.signal
        signal?.addEventListener('abort', markAbortCalled)
        return createNeverResolvingPromise()
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result, unmount } = renderHook(() => useAzureConnectionStatus())

      act(() => {
        void result.current.fetchAzureStatus()
      })

      // Unmount while request is in flight
      unmount()

      await new Promise((resolve) => setTimeout(resolve, 50))

      expect(abortCalled).toBe(true)
    })
  })

  describe('realtime config callback', () => {
    it('should call onAzureRealtimeConfig with realtime value', async () => {
      const onAzureRealtimeConfig = vi.fn()
      const mockFetch = vi.fn(() =>
        createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
          azureRealtime: true,
        })
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() =>
        useAzureConnectionStatus({ onAzureRealtimeConfig })
      )

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      await waitFor(() => {
        expect(onAzureRealtimeConfig).toHaveBeenCalledWith(true)
      })
    })
  })

  describe('input validation', () => {
    it('should reject invalid mode values', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const mockFetch = vi.fn(() => createMockFetchResponse({ success: true }))
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.handleLogModeChange('invalid' as LogMode)
      })

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[useAzureConnectionStatus] Invalid mode: invalid')
      )
      expect(mockFetch).not.toHaveBeenCalled()

      consoleSpy.mockRestore()
    })

    it('should accept valid local mode', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({ success: true })
        }
        return createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.handleLogModeChange('local')
      })

      await waitFor(() => {
        expect(result.current.logMode).toBe('local')
      })
    })
  })

  describe('JSON parsing errors', () => {
    it('should handle malformed JSON gracefully', async () => {
      const mockFetch = vi.fn(() =>
        Promise.resolve({
          ok: true,
          status: 200,
          json: () => Promise.reject(new SyntaxError('Unexpected token')),
        } as Response)
      )
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[useAzureConnectionStatus] Failed to parse mode response:'),
        expect.any(SyntaxError)
      )

      consoleSpy.mockRestore()
    })
  })

  describe('timeout cleanup', () => {
    it('should clear mode switch timeout on unmount', () => {
      vi.useFakeTimers()

      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({ success: true })
        }
        return createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result, unmount } = renderHook(() => useAzureConnectionStatus())

      // Start mode change
      act(() => {
        void result.current.handleLogModeChange('azure')
      })

      // Unmount before timeout completes
      unmount()

      // Advance time - should not throw or cause errors
      act(() => {
        vi.advanceTimersByTime(2000)
      })

      // No assertions needed - just verify no errors thrown

      vi.useRealTimers()
    })

    it('should clear previous timeout when switching modes rapidly', async () => {
      vi.useFakeTimers()

      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({ success: true })
        }
        return createMockFetchResponse({
          mode: 'azure',
          azureEnabled: true,
          azureStatus: 'connected',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const { result } = renderHook(() => useAzureConnectionStatus())

      // Switch mode multiple times rapidly
      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      await act(async () => {
        await result.current.handleLogModeChange('local')
      })

      // Advance time past timeout
      act(() => {
        vi.advanceTimersByTime(2000)
      })

      // Should be false (only one timeout should fire)
      expect(result.current.isModeSwitching).toBe(false)

      vi.useRealTimers()
    })
  })

  describe('mode switch error recovery', () => {
    it('should revert mode on API error', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return createMockFetchResponse({}, false, 500)
        }
        return createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      // Initial state
      expect(result.current.logMode).toBe('local')

      // Try to switch (will fail)
      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      // Mode should remain local (reverted)
      await waitFor(() => {
        expect(result.current.logMode).toBe('local')
      })

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[useAzureConnectionStatus] Failed to switch mode'),
        expect.any(String)
      )

      consoleSpy.mockRestore()
    })

    it('should revert mode on network error', async () => {
      const mockFetch = vi.fn((_url: string, options?: RequestInit) => {
        if (options?.method === 'PUT') {
          return Promise.reject(new Error('Network failure'))
        }
        return createMockFetchResponse({
          mode: 'local',
          azureEnabled: false,
          azureStatus: 'disabled',
        })
      })
      globalThis.fetch = mockFetch as unknown as typeof fetch

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const { result } = renderHook(() => useAzureConnectionStatus())

      // Initial state
      expect(result.current.logMode).toBe('local')

      // Try to switch (will fail)
      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      // Mode should remain local (reverted)
      await waitFor(() => {
        expect(result.current.logMode).toBe('local')
      })

      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('[useAzureConnectionStatus] Error switching mode'),
        expect.any(Error)
      )

      consoleSpy.mockRestore()
    })
  })
})
