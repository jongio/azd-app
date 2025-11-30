import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useServiceOperations } from './useServiceOperations'
import type { Service } from '@/types'

// Mock fetch globally
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

describe('useServiceOperations', () => {
  beforeEach(() => {
    mockFetch.mockReset()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  const createMockService = (name: string, status: 'starting' | 'stopping' | 'error' | 'ready' | 'running' | 'stopped' | 'not-running'): Service => ({
    name,
    local: {
      status,
      health: 'healthy',
      pid: 1234,
      port: 3000,
      url: `http://localhost:3000`,
      startTime: new Date().toISOString(),
      lastChecked: new Date().toISOString(),
    },
    language: 'node',
    framework: 'express',
    project: '/test/project',
  })

  describe('getOperationState', () => {
    it('returns idle for unknown service', () => {
      const { result } = renderHook(() => useServiceOperations())
      expect(result.current.getOperationState('unknown-service')).toBe('idle')
    })
  })

  describe('isOperationInProgress', () => {
    it('returns false for idle service', () => {
      const { result } = renderHook(() => useServiceOperations())
      expect(result.current.isOperationInProgress('test-service')).toBe(false)
    })
  })

  describe('isBulkOperationInProgress', () => {
    it('returns false initially', () => {
      const { result } = renderHook(() => useServiceOperations())
      expect(result.current.isBulkOperationInProgress()).toBe(false)
    })
  })

  describe('getAvailableActions', () => {
    it('returns start for stopped service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'stopped')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('start')
      expect(actions).not.toContain('stop')
      expect(actions).not.toContain('restart')
    })

    it('returns start for not-running service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'not-running')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('start')
    })

    it('returns stop and restart for error service with PID (process alive)', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'error')
      // createMockService includes pid: 1234, so process is alive
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('restart')
      expect(actions).toContain('stop')
      expect(actions).not.toContain('start')
    })

    it('returns start for error service without PID (process dead)', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service: Service = {
        name: 'test',
        local: {
          status: 'error',
          health: 'unhealthy',
          // No pid - process is dead
          port: 3000,
          url: 'http://localhost:3000',
        },
        language: 'node',
        framework: 'express',
        project: '/test/project',
      }
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('start')
      expect(actions).not.toContain('stop')
      expect(actions).not.toContain('restart')
    })

    it('returns restart and stop for running service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'running')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('restart')
      expect(actions).toContain('stop')
      expect(actions).not.toContain('start')
    })

    it('returns restart and stop for ready service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'ready')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('restart')
      expect(actions).toContain('stop')
    })

    it('returns stop for starting service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'starting')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toContain('stop')
      expect(actions).not.toContain('start')
      expect(actions).not.toContain('restart')
    })

    it('returns empty for stopping service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'stopping')
      const actions = result.current.getAvailableActions(service)
      expect(actions).toHaveLength(0)
    })
  })

  describe('canPerformAction', () => {
    it('returns true for valid action on stopped service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'stopped')
      expect(result.current.canPerformAction(service, 'start')).toBe(true)
      expect(result.current.canPerformAction(service, 'stop')).toBe(false)
    })

    it('returns true for valid action on running service', () => {
      const { result } = renderHook(() => useServiceOperations())
      const service = createMockService('test', 'running')
      expect(result.current.canPerformAction(service, 'stop')).toBe(true)
      expect(result.current.canPerformAction(service, 'restart')).toBe(true)
      expect(result.current.canPerformAction(service, 'start')).toBe(false)
    })
  })

  describe('startService', () => {
    it('calls API and returns true on success', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      })

      const { result } = renderHook(() => useServiceOperations())

      let success: boolean = false
      await act(async () => {
        success = await result.current.startService('test-service')
      })

      expect(success).toBe(true)
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/services/start?service=test-service',
        { method: 'POST' }
      )
    })

    it('returns false on API error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: 'Service not found' }),
      })

      const { result } = renderHook(() => useServiceOperations())

      let success: boolean = true
      await act(async () => {
        success = await result.current.startService('nonexistent')
      })

      expect(success).toBe(false)
      expect(result.current.error).toBe('Service not found')
    })

    it('returns false on network error', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      const { result } = renderHook(() => useServiceOperations())

      let success: boolean = true
      await act(async () => {
        success = await result.current.startService('test-service')
      })

      expect(success).toBe(false)
      expect(result.current.error).toBe('Network error')
    })
  })

  describe('stopService', () => {
    it('calls API with correct endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      })

      const { result } = renderHook(() => useServiceOperations())

      await act(async () => {
        await result.current.stopService('test-service')
      })

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/services/stop?service=test-service',
        { method: 'POST' }
      )
    })
  })

  describe('restartService', () => {
    it('calls API with correct endpoint', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      })

      const { result } = renderHook(() => useServiceOperations())

      await act(async () => {
        await result.current.restartService('test-service')
      })

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/services/restart?service=test-service',
        { method: 'POST' }
      )
    })
  })

  describe('startAll', () => {
    it('calls bulk API without service param', async () => {
      const mockResult = {
        success: true,
        message: '2 service(s) started, 0 failed',
        services: [] as { name: string; success: boolean; error?: string; duration?: string }[],
        successCount: 2,
        failureCount: 0,
        duration: '100ms',
      }
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResult),
      })

      const { result } = renderHook(() => useServiceOperations())

      let bulkResult: typeof mockResult | null = null
      await act(async () => {
        bulkResult = await result.current.startAll()
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/services/start', { method: 'POST' })
      expect(bulkResult).toEqual(mockResult)
      expect(result.current.lastResult).toEqual(mockResult)
    })

    it('returns null on error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: 'Failed' }),
      })

      const { result } = renderHook(() => useServiceOperations())

      let bulkResult: unknown = 'initial'
      await act(async () => {
        bulkResult = await result.current.startAll()
      })

      expect(bulkResult).toBeNull()
      expect(result.current.error).toBe('Failed')
    })
  })

  describe('stopAll', () => {
    it('calls bulk stop API', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          success: true,
          message: '2 service(s) stopped',
          services: [],
          successCount: 2,
          failureCount: 0,
          duration: '50ms',
        }),
      })

      const { result } = renderHook(() => useServiceOperations())

      await act(async () => {
        await result.current.stopAll()
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/services/stop', { method: 'POST' })
    })
  })

  describe('restartAll', () => {
    it('calls bulk restart API', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          success: true,
          message: '2 service(s) restarted',
          services: [],
          successCount: 2,
          failureCount: 0,
          duration: '200ms',
        }),
      })

      const { result } = renderHook(() => useServiceOperations())

      await act(async () => {
        await result.current.restartAll()
      })

      expect(mockFetch).toHaveBeenCalledWith('/api/services/restart', { method: 'POST' })
    })
  })

  describe('clearError', () => {
    it('clears the error state', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: 'Test error' }),
      })

      const { result } = renderHook(() => useServiceOperations())

      await act(async () => {
        await result.current.startService('test')
      })

      expect(result.current.error).toBe('Test error')

      act(() => {
        result.current.clearError()
      })

      expect(result.current.error).toBeNull()
    })
  })

  describe('operation state tracking', () => {
    it('tracks operation in progress during API call', async () => {
      let resolvePromise: () => void
      const delayedPromise = new Promise<void>((resolve) => {
        resolvePromise = resolve
      })

      mockFetch.mockImplementationOnce(() => {
        return delayedPromise.then(() => ({
          ok: true,
          json: () => Promise.resolve({ success: true }),
        }))
      })

      const { result } = renderHook(() => useServiceOperations())

      // Start the operation
      let operationPromise: Promise<boolean>
      act(() => {
        operationPromise = result.current.startService('test-service')
      })

      // Check state during operation - need to wait for state to update
      await waitFor(() => {
        expect(result.current.isOperationInProgress('test-service')).toBe(true)
      })

      // Complete the operation
      await act(async () => {
        resolvePromise!()
        await operationPromise
      })

      // Check state after completion
      expect(result.current.isOperationInProgress('test-service')).toBe(false)
    })
  })
})
