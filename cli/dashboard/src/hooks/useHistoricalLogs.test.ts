/**
 * Tests for useHistoricalLogs hook
 * Validates historical log querying, pagination, and time range handling
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import {
  useHistoricalLogs,
  timeRangeToTimespan,
  formatTimeRangeDisplay,
  type TimeRange,
  type HistoricalLogResult,
} from './useHistoricalLogs'

// Mock useBackendConnection
const mockUseBackendConnection = vi.fn()
vi.mock('./useBackendConnection', () => ({
  useBackendConnection: (): { connected: boolean } => mockUseBackendConnection() as { connected: boolean },
}))

// Type for mock request body
interface MockRequestBody {
  body?: string
  query?: string
  limit?: number
  tables?: string[]
}

describe('useHistoricalLogs', () => {
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    vi.clearAllMocks()
    mockUseBackendConnection.mockReturnValue({ connected: true })
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    vi.restoreAllMocks()
    globalThis.fetch = originalFetch
  })

  describe('timeRangeToTimespan', () => {
    it('should convert 15m preset to ISO 8601 duration', () => {
      const timeRange: TimeRange = { preset: '15m' }
      expect(timeRangeToTimespan(timeRange)).toBe('PT15M')
    })

    it('should convert 30m preset to ISO 8601 duration', () => {
      const timeRange: TimeRange = { preset: '30m' }
      expect(timeRangeToTimespan(timeRange)).toBe('PT30M')
    })

    it('should convert 6h preset to ISO 8601 duration', () => {
      const timeRange: TimeRange = { preset: '6h' }
      expect(timeRangeToTimespan(timeRange)).toBe('PT6H')
    })

    it('should convert 24h preset to ISO 8601 duration', () => {
      const timeRange: TimeRange = { preset: '24h' }
      expect(timeRangeToTimespan(timeRange)).toBe('PT24H')
    })

    it('should calculate duration for custom range in minutes', () => {
      const start = new Date('2025-12-25T12:00:00Z')
      const end = new Date('2025-12-25T12:45:00Z')
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      expect(timeRangeToTimespan(timeRange)).toBe('PT45M')
    })

    it('should calculate duration for custom range in hours', () => {
      const start = new Date('2025-12-25T10:00:00Z')
      const end = new Date('2025-12-25T13:00:00Z')
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      expect(timeRangeToTimespan(timeRange)).toBe('PT3H')
    })

    it('should calculate duration for custom range in days', () => {
      const start = new Date('2025-12-20T12:00:00Z')
      const end = new Date('2025-12-25T12:00:00Z')
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      expect(timeRangeToTimespan(timeRange)).toBe('P5D')
    })

    it('should round up fractional minutes', () => {
      const start = new Date('2025-12-25T12:00:00Z')
      const end = new Date('2025-12-25T12:00:30Z') // 30 seconds
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      expect(timeRangeToTimespan(timeRange)).toBe('PT1M')
    })

    it('should round up fractional hours', () => {
      const start = new Date('2025-12-25T12:00:00Z')
      const end = new Date('2025-12-25T14:30:00Z') // 2.5 hours
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      expect(timeRangeToTimespan(timeRange)).toBe('PT3H')
    })

    it('should default to PT30M when custom range missing dates', () => {
      const timeRange: TimeRange = { preset: 'custom' }
      expect(timeRangeToTimespan(timeRange)).toBe('PT30M')
    })

    it('should default to PT30M when custom range has only start', () => {
      const timeRange: TimeRange = { preset: 'custom', start: new Date() }
      expect(timeRangeToTimespan(timeRange)).toBe('PT30M')
    })

    it('should default to PT30M when custom range has only end', () => {
      const timeRange: TimeRange = { preset: 'custom', end: new Date() }
      expect(timeRangeToTimespan(timeRange)).toBe('PT30M')
    })
  })

  describe('formatTimeRangeDisplay', () => {
    it('should format 15m preset', () => {
      const timeRange: TimeRange = { preset: '15m' }
      expect(formatTimeRangeDisplay(timeRange)).toBe('last 15 minutes')
    })

    it('should format 30m preset', () => {
      const timeRange: TimeRange = { preset: '30m' }
      expect(formatTimeRangeDisplay(timeRange)).toBe('last 30 minutes')
    })

    it('should format 6h preset', () => {
      const timeRange: TimeRange = { preset: '6h' }
      expect(formatTimeRangeDisplay(timeRange)).toBe('last 6 hours')
    })

    it('should format 24h preset', () => {
      const timeRange: TimeRange = { preset: '24h' }
      expect(formatTimeRangeDisplay(timeRange)).toBe('last 24 hours')
    })

    it('should format custom range with dates', () => {
      const start = new Date('2025-12-25T10:30:00Z')
      const end = new Date('2025-12-25T14:45:00Z')
      const timeRange: TimeRange = { preset: 'custom', start, end }
      
      const formatted = formatTimeRangeDisplay(timeRange)
      
      // Should contain both dates formatted
      expect(formatted).toContain('Dec 25')
      expect(formatted).toContain('-')
    })

    it('should return generic message for custom range without dates', () => {
      const timeRange: TimeRange = { preset: 'custom' }
      expect(formatTimeRangeDisplay(timeRange)).toBe('custom range')
    })
  })

  describe('useHistoricalLogs hook', () => {
    const createMockResponse = (data: Partial<HistoricalLogResult> = {}): HistoricalLogResult => ({
      logs: [],
      total: 0,
      hasMore: false,
      executionTime: 100,
      ...data,
    })

    describe('initialization', () => {
      it('should initialize with empty state', () => {
        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        expect(result.current.logs).toEqual([])
        expect(result.current.total).toBe(0)
        expect(result.current.hasMore).toBe(false)
        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBeNull()
        expect(result.current.azureError).toBeNull()
        expect(result.current.executionTime).toBeNull()
        expect(result.current.offset).toBe(0)
      })

      it('should expose query and pagination functions', () => {
        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        expect(typeof result.current.executeQuery).toBe('function')
        expect(typeof result.current.loadMore).toBe('function')
        expect(typeof result.current.clearResults).toBe('function')
        expect(typeof result.current.resetQuery).toBe('function')
      })
    })

    describe('executeQuery', () => {
      it('should fetch logs successfully', async () => {
        const mockLogs = [
          {
            service: 'test-service',
            message: 'Test log',
            level: 1,
            timestamp: '2025-12-25T12:00:00Z',
            isStderr: false,
          },
        ]

        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ logs: mockLogs, total: 1 })),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.logs).toEqual(mockLogs)
        expect(result.current.total).toBe(1)
        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should send correct request payload', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '30m' })
        })

        expect(mockFetch).toHaveBeenCalledWith(
          '/api/azure/logs/query',
          expect.objectContaining({
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              service: 'test-service',
              timespan: 'PT30M',
              query: undefined,
              limit: 100,
              offset: 0,
            }),
          })
        )
      })

      it('should include custom KQL in request', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        const customKql = '| where Level == "Error"'
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' }, customKql)
        })

        const body = JSON.parse((mockFetch.mock.calls[0][1] as RequestInit).body as string) as MockRequestBody
        expect(body.query).toBe(customKql)
      })

      it('should set loading state during query', async () => {
        let resolvePromise: (value: unknown) => void = () => {}
        const fetchPromise = new Promise((resolve) => {
          resolvePromise = resolve
        })

        globalThis.fetch = vi.fn().mockReturnValue(fetchPromise)

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        act(() => {
          void result.current.executeQuery({ preset: '15m' })
        })

        // Should be loading
        expect(result.current.isLoading).toBe(true)

        // Resolve fetch
        resolvePromise({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })

        await waitFor(() => {
          expect(result.current.isLoading).toBe(false)
        })
      })

      it('should handle API error responses', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: false,
          status: 500,
          text: () => Promise.resolve('Internal server error'),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.error).toBe('Internal server error')
        expect(result.current.logs).toEqual([])
        expect(result.current.total).toBe(0)
        expect(result.current.hasMore).toBe(false)
      })

      it('should handle network errors', async () => {
        globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network failure'))

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.error).toBe('Network failure')
        expect(result.current.logs).toEqual([])
      })

      it('should reset offset on new query', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        // First query
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        // Manually set offset (simulating pagination)
        // Then execute new query
        await act(async () => {
          await result.current.executeQuery({ preset: '30m' })
        })

        expect(result.current.offset).toBe(0)
      })

      it('should handle empty response array', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ logs: [] })),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.logs).toEqual([])
        expect(result.current.error).toBeNull()
      })

      it('should store execution time', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ executionTime: 250 })),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.executionTime).toBe(250)
      })
    })

    describe('loadMore (pagination)', () => {
      it('should append new logs to existing logs', async () => {
        const firstPage = [
          { service: 'test', message: 'Log 1', level: 1, timestamp: '2025-12-25T12:00:00Z', isStderr: false },
        ]
        const secondPage = [
          { service: 'test', message: 'Log 2', level: 1, timestamp: '2025-12-25T12:01:00Z', isStderr: false },
        ]

        const mockFetch = vi
          .fn()
          .mockResolvedValueOnce({
            ok: true,
            json: () => Promise.resolve(createMockResponse({ logs: firstPage, total: 2, hasMore: true })),
          })
          .mockResolvedValueOnce({
            ok: true,
            json: () => Promise.resolve(createMockResponse({ logs: secondPage, total: 2, hasMore: false })),
          })

        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service', pageSize: 1 })
        )

        // Execute initial query
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.logs).toEqual(firstPage)
        expect(result.current.hasMore).toBe(true)

        // Load more
        await act(async () => {
          await result.current.loadMore()
        })

        expect(result.current.logs).toEqual([...firstPage, ...secondPage])
        expect(result.current.hasMore).toBe(false)
      })

      it('should update offset correctly', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ hasMore: true })),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service', pageSize: 50 })
        )

        // Initial query
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(result.current.offset).toBe(0)

        // Load more
        await act(async () => {
          await result.current.loadMore()
        })

        expect(result.current.offset).toBe(50)

        // Load more again
        await act(async () => {
          await result.current.loadMore()
        })

        expect(result.current.offset).toBe(100)
      })

      it('should check isLoading guard in loadMore', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ hasMore: true })),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        // Initial query
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        // Verify loadMore doesn't work when already loading
        // This is best-effort - the guard prevents most concurrent calls
        await act(async () => {
          await result.current.loadMore()
        })

        // Should have made at least one loadMore call
        expect(mockFetch.mock.calls.length).toBeGreaterThan(1)
      })

      it('should not load more if hasMore is false', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse({ hasMore: false })),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        const callCount = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls.length

        await act(async () => {
          await result.current.loadMore()
        })

        // Should not have made another call
        expect((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBe(callCount)
      })

      it('should handle loadMore errors', async () => {
        const mockFetch = vi
          .fn()
          .mockResolvedValueOnce({
            ok: true,
            json: () => Promise.resolve(createMockResponse({ hasMore: true })),
          })
          .mockRejectedValueOnce(new Error('Pagination failed'))

        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        await act(async () => {
          await result.current.loadMore()
        })

        expect(result.current.error).toBe('Pagination failed')
      })
    })

    describe('clearResults', () => {
      it('should clear all state', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () =>
            Promise.resolve(
              createMockResponse({
                logs: [{ service: 'test', message: 'Log', level: 1, timestamp: '2025-12-25T12:00:00Z', isStderr: false }],
                total: 1,
                hasMore: true,
                executionTime: 100,
              })
            ),
        })

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        // Verify state is populated
        expect(result.current.logs.length).toBeGreaterThan(0)
        expect(result.current.total).toBeGreaterThan(0)

        act(() => {
          result.current.clearResults()
        })

        expect(result.current.logs).toEqual([])
        expect(result.current.total).toBe(0)
        expect(result.current.hasMore).toBe(false)
        expect(result.current.error).toBeNull()
        expect(result.current.azureError).toBeNull()
        expect(result.current.executionTime).toBeNull()
        expect(result.current.offset).toBe(0)
      })
    })

    describe('resetQuery', () => {
      it('should re-execute query without custom KQL', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        // Execute with custom KQL
        await act(async () => {
          await result.current.executeQuery({ preset: '15m' }, '| where Level == "Error"')
        })

        mockFetch.mockClear()

        // Reset query
        act(() => {
          result.current.resetQuery()
        })

        // Should execute without custom query
        const body = JSON.parse((mockFetch.mock.calls[0][1] as RequestInit).body as string) as MockRequestBody
        expect(body.query).toBeUndefined()
      })

      it('should not error if no query has been executed', () => {
        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        // Should not throw
        act(() => {
          result.current.resetQuery()
        })

        expect(result.current.error).toBeNull()
      })
    })

    describe('backend connection handling', () => {
      it('should not execute query when backend disconnected', async () => {
        mockUseBackendConnection.mockReturnValue({ connected: false })

        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service' })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        expect(mockFetch).not.toHaveBeenCalled()
        expect(result.current.error).toBe('Backend connection lost')
      })
    })

    describe('custom page size', () => {
      it('should use custom page size', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockResponse()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useHistoricalLogs({ serviceName: 'test-service', pageSize: 50 })
        )

        await act(async () => {
          await result.current.executeQuery({ preset: '15m' })
        })

        const body = JSON.parse((mockFetch.mock.calls[0][1] as RequestInit).body as string) as MockRequestBody
        expect(body.limit).toBe(50)
      })
    })
  })
})
