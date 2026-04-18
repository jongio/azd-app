/**
 * Tests for ServiceOperationsContext against an in-memory Connect router
 * transport. Mirrors useServices.test.tsx pattern -- production code path
 * runs unchanged with an injected transport.
 */
import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  Code,
  ConnectError,
  createRouterTransport,
  type ConnectRouter,
} from '@connectrpc/connect'

import {
  useServiceOperations,
  ServiceOperationsProvider,
} from './useServiceOperations'
import { ServicesService } from '@/gen/proto/azdapp/v1/services_connect.js'
import {
  GetServicesResponse,
  StartServiceResponse,
  StopServiceResponse,
  RestartServiceResponse,
} from '@/gen/proto/azdapp/v1/services_pb.js'
import { OperationResult } from '@/gen/proto/azdapp/v1/common_pb.js'
import type { Service } from '@/types'
import type { ReactNode } from 'react'

interface CallLog {
  op: 'start' | 'stop' | 'restart'
  serviceName: string
}

interface RouterOverrides {
  startResult?: OperationResult
  stopResult?: OperationResult
  restartResult?: OperationResult
  startError?: ConnectError
  stopError?: ConnectError
  restartError?: ConnectError
  /** Optional pre-resolution hook used by latency tests. */
  startBefore?: () => Promise<void>
}

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  calls: CallLog[]
}

function makeTransport(overrides: RouterOverrides = {}): Harness {
  const calls: CallLog[] = []
  const ok = (msg: string) => new OperationResult({ success: true, message: msg })

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(ServicesService, {
      getServices: () => Promise.resolve(new GetServicesResponse()),
      async startService(req) {
        calls.push({ op: 'start', serviceName: req.serviceName })
        if (overrides.startBefore) await overrides.startBefore()
        if (overrides.startError) throw overrides.startError
        return new StartServiceResponse({
          result: overrides.startResult ?? ok('1 service(s) started, 0 failed'),
        })
      },
      stopService: (req) => {
        calls.push({ op: 'stop', serviceName: req.serviceName })
        if (overrides.stopError) return Promise.reject(overrides.stopError)
        return Promise.resolve(
          new StopServiceResponse({
            result: overrides.stopResult ?? ok('1 service(s) stopped, 0 failed'),
          }),
        )
      },
      restartService: (req) => {
        calls.push({ op: 'restart', serviceName: req.serviceName })
        if (overrides.restartError) return Promise.reject(overrides.restartError)
        return Promise.resolve(
          new RestartServiceResponse({
            result: overrides.restartResult ?? ok('1 service(s) restarted, 0 failed'),
          }),
        )
      },
    })
  })

  return { transport, calls }
}

function makeWrapper(transport: ReturnType<typeof createRouterTransport>) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <ServiceOperationsProvider transport={transport}>
        {children}
      </ServiceOperationsProvider>
    )
  }
}

describe('useServiceOperations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  const createMockService = (
    name: string,
    status:
      | 'starting'
      | 'stopping'
      | 'error'
      | 'ready'
      | 'running'
      | 'stopped'
      | 'not-running',
  ): Service => ({
    name,
    local: {
      status,
      health: 'healthy',
      pid: 1234,
      port: 3000,
      url: 'http://localhost:3000',
      startTime: new Date().toISOString(),
      lastChecked: new Date().toISOString(),
    },
    language: 'node',
    framework: 'express',
    project: '/test/project',
  })

  // -------------------------------------------------------------------------
  // Pure state queries (no transport interaction)
  // -------------------------------------------------------------------------

  describe('state queries', () => {
    it('getOperationState returns idle for unknown service', () => {
      const { transport } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })
      expect(result.current.getOperationState('unknown-service')).toBe('idle')
    })

    it('isOperationInProgress returns false initially', () => {
      const { transport } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })
      expect(result.current.isOperationInProgress('test-service')).toBe(false)
    })

    it('isBulkOperationInProgress returns false initially', () => {
      const { transport } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })
      expect(result.current.isBulkOperationInProgress()).toBe(false)
    })
  })

  describe('getAvailableActions', () => {
    it('returns start for stopped service', () => {
      const { transport } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })
      const actions = result.current.getAvailableActions(createMockService('test', 'stopped'))
      expect(actions).toContain('start')
      expect(actions).not.toContain('stop')
      expect(actions).not.toContain('restart')
    })

    it('returns start for not-running service', () => {
      const { transport } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })
      const actions = result.current.getAvailableActions(createMockService('test', 'not-running'))
      expect(actions).toContain('start')
    })
  })

  // -------------------------------------------------------------------------
  // Single-service operations
  // -------------------------------------------------------------------------

  describe('startService', () => {
    it('routes to Connect StartService and returns true on success', async () => {
      const { transport, calls } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let success = false
      await act(async () => {
        success = await result.current.startService('test-service')
      })

      expect(success).toBe(true)
      expect(calls).toEqual([{ op: 'start', serviceName: 'test-service' }])
    })

    it('returns false and surfaces the message on Connect error', async () => {
      const { transport } = makeTransport({
        startError: new ConnectError('Service not found', Code.NotFound),
      })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let success = true
      await act(async () => {
        success = await result.current.startService('nonexistent')
      })

      expect(success).toBe(false)
      expect(result.current.error).toMatch(/Service not found/)
    })

    it('returns false on transport-layer failure', async () => {
      const { transport } = makeTransport({
        startError: new ConnectError('Network error', Code.Unavailable),
      })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let success = true
      await act(async () => {
        success = await result.current.startService('test-service')
      })

      expect(success).toBe(false)
      expect(result.current.error).toMatch(/Network error/)
    })
  })

  describe('stopService', () => {
    it('routes to Connect StopService with the supplied name', async () => {
      const { transport, calls } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      await act(async () => {
        await result.current.stopService('test-service')
      })

      expect(calls).toEqual([{ op: 'stop', serviceName: 'test-service' }])
    })
  })

  describe('restartService', () => {
    it('routes to Connect RestartService with the supplied name', async () => {
      const { transport, calls } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      await act(async () => {
        await result.current.restartService('test-service')
      })

      expect(calls).toEqual([{ op: 'restart', serviceName: 'test-service' }])
    })
  })

  // -------------------------------------------------------------------------
  // Bulk operations
  // -------------------------------------------------------------------------

  describe('startAll', () => {
    it('invokes Connect StartService with empty serviceName for bulk', async () => {
      const bulkResult = new OperationResult({
        success: true,
        message: '2 service(s) started, 0 failed',
      })
      const { transport, calls } = makeTransport({ startResult: bulkResult })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let bulkRet: Awaited<ReturnType<typeof result.current.startAll>> | undefined
      await act(async () => {
        bulkRet = await result.current.startAll()
      })

      expect(calls).toEqual([{ op: 'start', serviceName: '' }])
      expect(bulkRet).toMatchObject({
        success: true,
        message: '2 service(s) started, 0 failed',
        services: [],
        successCount: 1,
        failureCount: 0,
      })
      expect(result.current.lastResult?.success).toBe(true)
    })

    it('returns null and records error on Connect failure', async () => {
      const { transport } = makeTransport({
        startError: new ConnectError('Failed', Code.Internal),
      })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let bulkRet: unknown = 'initial'
      await act(async () => {
        bulkRet = await result.current.startAll()
      })

      expect(bulkRet).toBeNull()
      expect(result.current.error).toMatch(/Failed/)
    })
  })

  describe('stopAll', () => {
    it('invokes Connect StopService with empty serviceName for bulk', async () => {
      const { transport, calls } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      await act(async () => {
        await result.current.stopAll()
      })

      expect(calls).toEqual([{ op: 'stop', serviceName: '' }])
    })
  })

  describe('restartAll', () => {
    it('invokes Connect RestartService with empty serviceName for bulk', async () => {
      const { transport, calls } = makeTransport()
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      await act(async () => {
        await result.current.restartAll()
      })

      expect(calls).toEqual([{ op: 'restart', serviceName: '' }])
    })
  })

  // -------------------------------------------------------------------------
  // Cross-cutting behavior
  // -------------------------------------------------------------------------

  describe('clearError', () => {
    it('clears the error state', async () => {
      const { transport } = makeTransport({
        startError: new ConnectError('Test error', Code.Internal),
      })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      await act(async () => {
        await result.current.startService('test')
      })
      expect(result.current.error).toMatch(/Test error/)

      act(() => {
        result.current.clearError()
      })
      expect(result.current.error).toBeNull()
    })
  })

  describe('operation state tracking', () => {
    it('tracks operation in progress during the Connect call', async () => {
      let release: () => void = () => undefined
      const gate = new Promise<void>((resolve) => {
        release = resolve
      })
      const { transport } = makeTransport({ startBefore: () => gate })
      const { result } = renderHook(() => useServiceOperations(), {
        wrapper: makeWrapper(transport),
      })

      let opPromise!: Promise<boolean>
      act(() => {
        opPromise = result.current.startService('test-service')
      })

      await waitFor(() => {
        expect(result.current.isOperationInProgress('test-service')).toBe(true)
      })

      await act(async () => {
        release()
        await opPromise
      })

      expect(result.current.isOperationInProgress('test-service')).toBe(false)
    })
  })
})
