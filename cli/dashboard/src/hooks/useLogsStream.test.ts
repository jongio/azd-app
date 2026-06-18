/**
 * Tests for useLogsStream - the orchestrator hook that wires the
 * historical-fetch path together with the live shared stream.
 *
 * After the WebSocket -> Connect migration, the initial-fetch path
 * uses Connect-RPC unary calls (`LogsService.GetLogs` /
 * `AzureService.GetAzureLogs`) instead of the legacy REST endpoints,
 * and live updates flow through `useSharedLogStream` (mocked here;
 * transport behaviour is covered by useSharedLogStream.test.ts).
 *
 * Tests drive the Connect path via `createRouterTransport` and pass
 * the transport through the hook's new `transport` param so both
 * unary fetch and (mocked) stream subscriber see the same in-memory
 * backend.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  ConnectError,
  Code,
  createRouterTransport,
  type ConnectRouter,
  type ServiceImpl,
  type Transport,
} from '@connectrpc/connect'

import { useLogsStream } from './useLogsStream'
import type { LogMode } from '@/components/ModeToggle'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'
import {
  type GetLogsRequest,
  GetLogsResponseSchema,
  type GetLogsResponse,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import {
  type GetAzureLogsRequest,
  GetAzureLogsResponseSchema,
  type GetAzureLogsResponse,
} from '@/gen/proto/azdapp/v1/azure_pb.js'
import { create } from '@bufbuild/protobuf'

const { mockBackend } = vi.hoisted(() => ({
  mockBackend: { connected: true },
}))

vi.mock('@/hooks/useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: mockBackend.connected }),
}))

type SharedLogStreamArgs = {
  serviceName: string
  enabled: boolean
  mode: 'local' | 'azure'
  onLogEntry: (entry: { service: string; message: string; level: number; timestamp: string; isStderr: boolean }) => void
  since?: string
}

type SharedLogStreamReturn = {
  connectionState: 'disconnected' | 'connecting' | 'connected' | 'error'
  droppedCount: number
}

const sharedLogStreamMock = vi.fn(
  (_opts: SharedLogStreamArgs): SharedLogStreamReturn => ({
    connectionState: 'disconnected',
    droppedCount: 0,
  }),
)

vi.mock('@/hooks/useSharedLogStream', () => ({
  useSharedLogStream: (opts: SharedLogStreamArgs) => sharedLogStreamMock(opts),
}))

/**
 * Build a Connect router transport with overridable unary handlers
 * for GetLogs / GetAzureLogs. Other methods on the two services are
 * stubbed with Unimplemented rejections - the hook never touches
 * them, and casting through `unknown` avoids hand-writing stubs for
 * every RPC just to keep the `ServiceImpl` type happy.
 */
interface TransportOverrides {
  getLogs?: (req: GetLogsRequest) => GetLogsResponse | Promise<GetLogsResponse>
  getAzureLogs?: (req: GetAzureLogsRequest) => GetAzureLogsResponse | Promise<GetAzureLogsResponse>
}

function makeTransport(overrides: TransportOverrides = {}): Transport {
  return createRouterTransport((router: ConnectRouter) => {
    const notUsed = () =>
      Promise.reject(new ConnectError('unused in these tests', Code.Unimplemented))

    router.service(LogsService, {
      getLogs(req: GetLogsRequest): Promise<GetLogsResponse> {
        if (overrides.getLogs) return Promise.resolve(overrides.getLogs(req))
        return Promise.resolve(create(GetLogsResponseSchema, { entries: [] }))
      },
      streamLocalLogs: notUsed,
      listClassifications: notUsed,
      addClassification: notUsed,
      deleteClassification: notUsed,
      getPreferences: notUsed,
      savePreferences: notUsed,
    } as unknown as ServiceImpl<typeof LogsService>)

    router.service(AzureService, {
      getAzureLogs(req: GetAzureLogsRequest): Promise<GetAzureLogsResponse> {
        if (overrides.getAzureLogs) return Promise.resolve(overrides.getAzureLogs(req))
        return Promise.resolve(create(GetAzureLogsResponseSchema, { entries: [] }))
      },
      streamAzureLogs: notUsed,
      enableAzureLogging: notUsed,
      getAzureServices: notUsed,
      getAzureLogsHealth: notUsed,
      getAzureSetupState: notUsed,
      verifyAzureLogs: notUsed,
      checkDiagnosticSettings: notUsed,
      getAzureDiagnostics: notUsed,
      verifyWorkspace: notUsed,
      getAzureLogConfig: notUsed,
      saveAzureLogConfig: notUsed,
      listAzureTables: notUsed,
      getServiceQuery: notUsed,
      saveServiceQuery: notUsed,
    } as unknown as ServiceImpl<typeof AzureService>)
  })
}

describe('useLogsStream (orchestration)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockBackend.connected = true
    sharedLogStreamMock.mockClear()
    sharedLogStreamMock.mockImplementation(() => ({
      connectionState: 'disconnected',
      droppedCount: 0,
    }))
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  const flushTimersInAct = async () => {
    await act(async () => {
      await vi.runAllTimersAsync()
    })
  }

  const createParams = (overrides?: Partial<Parameters<typeof useLogsStream>[0]>) => ({
    serviceName: 'test-service',
    fetchKey: 'local:stream',
    logMode: 'local' as LogMode,
    timeRange: { preset: '15m' as const },
    azureRealtime: false,
    isPausedRef: { current: false },
    lastClearTimeRef: { current: Date.now() - 1000 },
    setLogs: vi.fn(),
    setErrorMessage: vi.fn(),
    onFetchSettled: vi.fn(),
    transport: makeTransport(),
    ...overrides,
  })

  const lastSharedCall = (): SharedLogStreamArgs => {
    const calls = sharedLogStreamMock.mock.calls
    expect(calls.length).toBeGreaterThan(0)
    return calls[calls.length - 1][0]
  }

  describe('shared-stream wiring', () => {
    it('subscribes to the shared local stream when in local mode', () => {
      renderHook(() => useLogsStream(createParams()))
      expect(sharedLogStreamMock).toHaveBeenCalled()
      const arg = lastSharedCall()
      expect(arg.mode).toBe('local')
      expect(arg.enabled).toBe(true)
      expect(arg.serviceName).toBe('test-service')
    })

    it('subscribes to the shared azure stream when in azure realtime mode', () => {
      renderHook(() =>
        useLogsStream(
          createParams({
            logMode: 'azure',
            azureRealtime: true,
            fetchKey: 'azure:15m::realtime',
          }),
        ),
      )
      const arg = lastSharedCall()
      expect(arg.mode).toBe('azure')
      expect(arg.enabled).toBe(true)
    })

    it('does not enable the shared stream in azure polling (non-realtime) mode', () => {
      renderHook(() =>
        useLogsStream(
          createParams({
            logMode: 'azure',
            azureRealtime: false,
            fetchKey: 'azure:15m::poll',
          }),
        ),
      )
      const arg = lastSharedCall()
      expect(arg.enabled).toBe(false)
    })

    it('enables the local stream even when the backend health is not yet connected', () => {
      // Regression: live local logs were gated on the health stream's
      // `connected` signal, so a still-starting service delayed all live
      // logs until the first health probe finished. Local logs run against
      // the same local server that served the page and must stream
      // immediately, independent of health.
      mockBackend.connected = false
      renderHook(() => useLogsStream(createParams()))
      const arg = lastSharedCall()
      expect(arg.mode).toBe('local')
      expect(arg.enabled).toBe(true)
    })

    it('does not enable the azure realtime stream until the backend is connected', () => {
      // Azure realtime stays gated on `connected` because Log Analytics
      // genuinely needs the backend reachable; only local is decoupled.
      mockBackend.connected = false
      renderHook(() =>
        useLogsStream(
          createParams({
            logMode: 'azure',
            azureRealtime: true,
            fetchKey: 'azure:15m::realtime',
          }),
        ),
      )
      const arg = lastSharedCall()
      expect(arg.mode).toBe('azure')
      expect(arg.enabled).toBe(false)
    })

    it('forwards droppedCount from the shared stream', () => {
      sharedLogStreamMock.mockImplementation(() => ({
        connectionState: 'connected',
        droppedCount: 42,
      }))
      const { result } = renderHook(() => useLogsStream(createParams()))
      expect(result.current.droppedCount).toBe(42)
    })
  })

  describe('shared-stream message handling', () => {
    it('routes incoming entries through setLogs when not paused', () => {
      const setLogs = vi.fn()
      const params = createParams({ setLogs })
      renderHook(() => useLogsStream(params))

      const handler = lastSharedCall().onLogEntry
      act(() => {
        handler({
          service: 'test-service',
          message: 'hello',
          level: 1,
          timestamp: new Date().toISOString(),
          isStderr: false,
        })
      })

      expect(setLogs).toHaveBeenCalled()
    })

    it('drops entries when paused', () => {
      const setLogs = vi.fn()
      const params = createParams({ setLogs, isPausedRef: { current: true } })
      renderHook(() => useLogsStream(params))

      const handler = lastSharedCall().onLogEntry
      act(() => {
        handler({
          service: 'test-service',
          message: 'paused',
          level: 1,
          timestamp: new Date().toISOString(),
          isStderr: false,
        })
      })

      expect(setLogs).not.toHaveBeenCalled()
    })
  })

  describe('onFetchSettled callback', () => {
    it('calls onFetchSettled after the initial fetch resolves', async () => {
      const onFetchSettled = vi.fn()
      renderHook(() => useLogsStream(createParams({ onFetchSettled })))

      expect(onFetchSettled).not.toHaveBeenCalled()
      await flushTimersInAct()
      expect(onFetchSettled).toHaveBeenCalled()
    })

    it('does not synchronously call onFetchSettled when fetchKey changes', async () => {
      const onFetchSettled = vi.fn()
      const initial = createParams({ onFetchSettled, fetchKey: 'local:stream' })

      const { rerender } = renderHook((props) => useLogsStream(props), {
        initialProps: initial,
      })

      await flushTimersInAct()
      expect(onFetchSettled).toHaveBeenCalled()
      const initialCalls = onFetchSettled.mock.calls.length

      rerender(
        createParams({
          onFetchSettled,
          fetchKey: 'azure:30m::poll',
          logMode: 'azure',
        }),
      )

      // Should not fire synchronously on prop change; only after the
      // queued fetch resolves.
      expect(onFetchSettled).toHaveBeenCalledTimes(initialCalls)

      await flushTimersInAct()
      expect(onFetchSettled.mock.calls.length).toBeGreaterThan(initialCalls)
    })
  })
})
