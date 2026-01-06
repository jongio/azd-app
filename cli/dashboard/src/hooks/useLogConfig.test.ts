/**
 * Tests for useLogConfig hooks
 * Validates table fetching and log configuration management
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import {
  useAvailableTables,
  useLogConfig,
  type TablesResponse,
  type LogConfig,
} from './useLogConfig'

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

describe('useLogConfig', () => {
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

  describe('useAvailableTables', () => {
    const createMockTablesResponse = (overrides?: Partial<TablesResponse>): TablesResponse => ({
      tables: [
        { name: 'AppServiceConsoleLogs', category: 'application', description: 'Console logs' },
        { name: 'AppServiceHTTPLogs', category: 'application', description: 'HTTP logs' },
      ],
      recommended: ['AppServiceConsoleLogs'],
      workspace: 'workspace-abc123',
      categories: [
        { name: 'application', displayName: 'Application Logs', tables: ['AppServiceConsoleLogs', 'AppServiceHTTPLogs'] },
      ],
      ...overrides,
    })

    describe('initialization', () => {
      it('should initialize with empty state', () => {
        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        expect(result.current.tables).toEqual([])
        expect(result.current.categories).toEqual([])
        expect(result.current.recommended).toEqual([])
        expect(result.current.workspace).toBe('')
        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should auto-fetch on mount when autoFetch is true', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockTablesResponse()),
        })
        globalThis.fetch = mockFetch

        renderHook(() => useAvailableTables({ autoFetch: true }))

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('/api/azure/tables?resourceType=containerapp')
          )
        })
      })

      it('should not auto-fetch when autoFetch is false', async () => {
        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        renderHook(() => useAvailableTables({ autoFetch: false }))

        await new Promise((resolve) => setTimeout(resolve, 50))
        expect(mockFetch).not.toHaveBeenCalled()
      })
    })

    describe('fetchTables', () => {
      it('should fetch tables successfully', async () => {
        const mockResponse = createMockTablesResponse()
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(mockResponse),
        })

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.tables).toEqual(mockResponse.tables)
        expect(result.current.categories).toEqual(mockResponse.categories)
        expect(result.current.recommended).toEqual(mockResponse.recommended)
        expect(result.current.workspace).toBe(mockResponse.workspace)
        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should include resourceType in query params', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockTablesResponse()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useAvailableTables({ resourceType: 'webapp', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/azure/tables?resourceType=webapp')
        )
      })

      it('should set loading state during fetch', async () => {
        let resolvePromise: (value: unknown) => void = () => {}
        const fetchPromise = new Promise((resolve) => {
          resolvePromise = resolve
        })

        globalThis.fetch = vi.fn().mockReturnValue(fetchPromise)

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        act(() => {
          void result.current.fetchTables()
        })

        expect(result.current.isLoading).toBe(true)

        resolvePromise({
          ok: true,
          json: () => Promise.resolve(createMockTablesResponse()),
        })

        await waitFor(() => {
          expect(result.current.isLoading).toBe(false)
        })
      })

      it('should handle fetch errors', async () => {
        globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'))

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.error).toBe('Network error')
        expect(result.current.tables).toEqual([])
        expect(result.current.categories).toEqual([])
      })

      it('should handle non-OK responses', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: false,
          status: 500,
        })

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.error).toContain('Failed to fetch tables: 500')
      })

      it('should normalize missing arrays to empty arrays', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve({ workspace: 'test' }), // Missing arrays
        })

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.tables).toEqual([])
        expect(result.current.categories).toEqual([])
        expect(result.current.recommended).toEqual([])
        expect(result.current.workspace).toBe('test')
      })

      it('should normalize non-array values to empty arrays', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () =>
            Promise.resolve({
              tables: 'not-an-array',
              categories: null,
              recommended: undefined,
              workspace: 'test',
            }),
        })

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.tables).toEqual([])
        expect(result.current.categories).toEqual([])
        expect(result.current.recommended).toEqual([])
      })

      it('should normalize non-string workspace to empty string', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () =>
            Promise.resolve({
              tables: [],
              categories: [],
              recommended: [],
              workspace: null,
            }),
        })

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(result.current.workspace).toBe('')
      })
    })

    describe('backend connection handling', () => {
      it('should not fetch when backend disconnected', async () => {
        mockUseBackendConnection.mockReturnValue({ connected: false })

        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        const { result } = renderHook(() => useAvailableTables({ autoFetch: false }))

        await act(async () => {
          await result.current.fetchTables()
        })

        expect(mockFetch).not.toHaveBeenCalled()
      })
    })

    describe('resourceType updates', () => {
      it('should refetch when resourceType changes and autoFetch is true', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockTablesResponse()),
        })
        globalThis.fetch = mockFetch

        const { rerender } = renderHook(
          ({ resourceType }) => useAvailableTables({ resourceType, autoFetch: true }),
          { initialProps: { resourceType: 'containerapp' } }
        )

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('resourceType=containerapp')
          )
        })

        mockFetch.mockClear()

        rerender({ resourceType: 'webapp' })

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('resourceType=webapp')
          )
        })
      })
    })
  })

  describe('useLogConfig', () => {
    const createMockConfig = (overrides?: Partial<LogConfig>): LogConfig => ({
      service: 'test-service',
      mode: 'tables',
      tables: ['AppServiceConsoleLogs'],
      resourceType: 'containerapp',
      ...overrides,
    })

    describe('initialization', () => {
      it('should initialize with null config', () => {
        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        expect(result.current.config).toBeNull()
        expect(result.current.isLoading).toBe(false)
        expect(result.current.isSaving).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should auto-fetch on mount when autoFetch is true', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })
        globalThis.fetch = mockFetch

        renderHook(() => useLogConfig({ serviceName: 'test-service', autoFetch: true }))

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('/api/azure/logs/config?service=test-service')
          )
        })
      })

      it('should not auto-fetch when autoFetch is false', async () => {
        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        renderHook(() => useLogConfig({ serviceName: 'test-service', autoFetch: false }))

        await new Promise((resolve) => setTimeout(resolve, 50))
        expect(mockFetch).not.toHaveBeenCalled()
      })
    })

    describe('fetchConfig', () => {
      it('should fetch config successfully', async () => {
        const mockConfig = createMockConfig()
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(mockConfig),
        })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(result.current.config).toEqual(mockConfig)
        expect(result.current.isLoading).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should include service name in query params', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'my-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/azure/logs/config?service=my-service')
        )
      })

      it('should set loading state during fetch', async () => {
        let resolvePromise: (value: unknown) => void = () => {}
        const fetchPromise = new Promise((resolve) => {
          resolvePromise = resolve
        })

        globalThis.fetch = vi.fn().mockReturnValue(fetchPromise)

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        act(() => {
          void result.current.fetchConfig()
        })

        expect(result.current.isLoading).toBe(true)

        resolvePromise({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })

        await waitFor(() => {
          expect(result.current.isLoading).toBe(false)
        })
      })

      it('should handle fetch errors', async () => {
        globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network error'))

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(result.current.error).toBe('Network error')
        expect(result.current.config).toBeNull()
      })

      it('should handle non-OK responses', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: false,
          status: 404,
        })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(result.current.error).toContain('Failed to fetch config: 404')
      })

      it('should not fetch when backend disconnected', async () => {
        mockUseBackendConnection.mockReturnValue({ connected: false })

        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(mockFetch).not.toHaveBeenCalled()
      })

      it('should not fetch when serviceName is empty', async () => {
        const mockFetch = vi.fn()
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: '', autoFetch: false })
        )

        await act(async () => {
          await result.current.fetchConfig()
        })

        expect(mockFetch).not.toHaveBeenCalled()
      })
    })

    describe('saveConfig', () => {
      it('should save config with tables successfully', async () => {
        const mockConfig = createMockConfig({ tables: ['AppServiceConsoleLogs', 'AppServiceHTTPLogs'] })
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(mockConfig),
        })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = false
        await act(async () => {
          success = await result.current.saveConfig({
            tables: ['AppServiceConsoleLogs', 'AppServiceHTTPLogs'],
          })
        })

        expect(success).toBe(true)
        expect(result.current.config).toEqual(mockConfig)
        expect(result.current.isSaving).toBe(false)
        expect(result.current.error).toBeNull()
      })

      it('should save config with custom query successfully', async () => {
        const mockConfig = createMockConfig({ mode: 'custom', query: '| where Level == "Error"' })
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(mockConfig),
        })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = false
        await act(async () => {
          success = await result.current.saveConfig({
            query: '| where Level == "Error"',
          })
        })

        expect(success).toBe(true)
        expect(result.current.config?.mode).toBe('custom')
        expect(result.current.config?.query).toBe('| where Level == "Error"')
      })

      it('should send correct request payload for tables', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(mockFetch).toHaveBeenCalledWith(
          '/api/azure/logs/config',
          expect.objectContaining({
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              service: 'test-service',
              mode: 'tables',
              tables: ['AppServiceConsoleLogs'],
              query: undefined,
            }),
          })
        )
      })

      it('should send correct request payload for query', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })
        globalThis.fetch = mockFetch

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        await act(async () => {
          await result.current.saveConfig({ query: '| where Level == "Error"' })
        })

        const body = JSON.parse((mockFetch.mock.calls[0][1] as RequestInit).body as string) as MockRequestBody
        expect(body.query).toBe('| where Level == "Error"')
        expect(body.tables).toBeUndefined()
      })

      it('should set saving state during save', async () => {
        let resolvePromise: (value: unknown) => void = () => {}
        const fetchPromise = new Promise((resolve) => {
          resolvePromise = resolve
        })

        globalThis.fetch = vi.fn().mockReturnValue(fetchPromise)

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        act(() => {
          void result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(result.current.isSaving).toBe(true)

        resolvePromise({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })

        await waitFor(() => {
          expect(result.current.isSaving).toBe(false)
        })
      })

      it('should return false on save error', async () => {
        globalThis.fetch = vi.fn().mockRejectedValue(new Error('Save failed'))

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(success).toBe(false)
        expect(result.current.error).toBe('Save failed')
      })

      it('should handle non-OK response', async () => {
        globalThis.fetch = vi.fn().mockResolvedValue({
          ok: false,
          status: 400,
          text: () => Promise.resolve('Invalid configuration'),
        })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(success).toBe(false)
        expect(result.current.error).toContain('Invalid configuration')
      })

      it('should reject when backend disconnected', async () => {
        mockUseBackendConnection.mockReturnValue({ connected: false })

        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(success).toBe(false)
        expect(result.current.error).toBe('Backend connection lost')
      })

      it('should reject when serviceName is empty', async () => {
        const { result } = renderHook(() =>
          useLogConfig({ serviceName: '', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({ tables: ['AppServiceConsoleLogs'] })
        })

        expect(success).toBe(false)
        expect(result.current.error).toBe('Service name is required')
      })

      it('should reject when neither tables nor query provided', async () => {
        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({})
        })

        expect(success).toBe(false)
        expect(result.current.error).toBe('Either tables or query is required')
      })

      it('should reject when tables array is empty', async () => {
        const { result } = renderHook(() =>
          useLogConfig({ serviceName: 'test-service', autoFetch: false })
        )

        let success = true
        await act(async () => {
          success = await result.current.saveConfig({ tables: [] })
        })

        expect(success).toBe(false)
        expect(result.current.error).toBe('Either tables or query is required')
      })
    })

    describe('serviceName changes', () => {
      it('should refetch when serviceName changes and autoFetch is true', async () => {
        const mockFetch = vi.fn().mockResolvedValue({
          ok: true,
          json: () => Promise.resolve(createMockConfig()),
        })
        globalThis.fetch = mockFetch

        const { rerender } = renderHook(
          ({ serviceName }) => useLogConfig({ serviceName, autoFetch: true }),
          { initialProps: { serviceName: 'service-1' } }
        )

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('service=service-1')
          )
        })

        mockFetch.mockClear()

        rerender({ serviceName: 'service-2' })

        await waitFor(() => {
          expect(mockFetch).toHaveBeenCalledWith(
            expect.stringContaining('service=service-2')
          )
        })
      })
    })
  })
})
