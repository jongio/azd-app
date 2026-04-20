/**
 * Flood-prevention tests for useLogsStream.
 *
 * Post-Connect migration, the per-service fetch count is observed
 * through a router-transport handler (not globalThis.fetch) since the
 * hook now dials `LogsService.GetLogs` via a Connect transport. Each
 * test still measures the same invariant - the hook must not flood
 * the backend when many services mount simultaneously, and must not
 * fall back to polling in local mode now that the WebSocket has been
 * replaced with a server-streaming Connect RPC.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import {
  ConnectError,
  Code,
  createRouterTransport,
  type ConnectRouter,
  type ServiceImpl,
  type Transport,
} from '@connectrpc/connect'

import { useLogsStream } from './useLogsStream'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_connect.js'
import { AzureService } from '@/gen/proto/azdapp/v1/azure_connect.js'
import {
  GetLogsRequest,
  GetLogsResponse,
} from '@/gen/proto/azdapp/v1/logs_pb.js'

vi.mock('@/hooks/useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true }),
}))

vi.mock('@/hooks/useSharedLogStream', () => ({
  useSharedLogStream: () => ({ connectionState: 'disconnected', droppedCount: 0 }),
}))

/**
 * Build a transport that counts per-service GetLogs calls. Returning
 * `{entries: []}` is deliberate - the hook treats empty results as a
 * possibly-slow-start service and schedules retries (500ms, 1s), so
 * an empty response is the worst-case for flood detection.
 */
function makeCountingTransport(): {
  transport: Transport
  callsByService: Map<string, number>
  totalCalls: () => number
} {
  const callsByService = new Map<string, number>()
  let totalCalls = 0

  const transport = createRouterTransport((router: ConnectRouter) => {
    const notUsed = () =>
      Promise.reject(new ConnectError('unused in these tests', Code.Unimplemented))

    router.service(LogsService, {
      getLogs(req: GetLogsRequest): Promise<GetLogsResponse> {
        totalCalls++
        const key = req.serviceName || 'all'
        callsByService.set(key, (callsByService.get(key) ?? 0) + 1)
        return Promise.resolve(new GetLogsResponse({ entries: [] }))
      },
      streamLocalLogs: notUsed,
      listClassifications: notUsed,
      addClassification: notUsed,
      deleteClassification: notUsed,
      getPreferences: notUsed,
      savePreferences: notUsed,
    } as unknown as ServiceImpl<typeof LogsService>)

    // AzureService stubs - the flood tests run in local mode so Azure
    // methods are never invoked, but `router.service` still needs the
    // full impl registered so `createAzureClient(transport)` inside
    // the hook can resolve.
    router.service(AzureService, {
      getAzureLogs: notUsed,
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

  return { transport, callsByService, totalCalls: () => totalCalls }
}

describe('useLogsStream flood prevention', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('should not flood the server when multiple services mount simultaneously', async () => {
    const { transport, callsByService, totalCalls } = makeCountingTransport()

    const services = ['appservice-web', 'azurite', 'containerapp-api', 'functions-worker']

    // Render hooks for all services simultaneously (like on app mount).
    services.forEach((serviceName) => {
      const setLogs = vi.fn()
      const setErrorMessage = vi.fn()
      const isPausedRef = { current: false }
      const lastClearTimeRef = { current: 0 }

      renderHook(() =>
        useLogsStream({
          serviceName,
          fetchKey: 'local:stream',
          logMode: 'local',
          timeRange: { preset: '15m' },
          azureRealtime: false,
          isPausedRef,
          lastClearTimeRef,
          setLogs,
          setErrorMessage,
          onFetchSettled: vi.fn(),
          transport,
        }),
      )
    })

    // Run all timers to execute the initial fetch and any empty-result
    // retry cascades.
    await vi.runAllTimersAsync()

    // Each service should be called at most 6 times:
    //   - 1 initial fetch
    //   - 2 empty-result retries (500ms, 1s)
    //   - Possibly doubled by React Strict Mode
    services.forEach((service) => {
      const count = callsByService.get(service) ?? 0
      expect(count).toBeLessThanOrEqual(6)
    })

    // Global ceiling: 4 services * 6 max.
    expect(totalCalls()).toBeLessThanOrEqual(24)
  })

  it('should not repeatedly poll in local mode when using the streaming RPC', async () => {
    const { transport, totalCalls } = makeCountingTransport()

    const setLogs = vi.fn()
    const setErrorMessage = vi.fn()
    const isPausedRef = { current: false }
    const lastClearTimeRef = { current: 0 }

    renderHook(() =>
      useLogsStream({
        serviceName: 'api',
        fetchKey: 'local:stream',
        logMode: 'local',
        timeRange: { preset: '15m' },
        azureRealtime: false,
        isPausedRef,
        lastClearTimeRef,
        setLogs,
        setErrorMessage,
        onFetchSettled: vi.fn(),
        transport,
      }),
    )

    // Run timers for the initial fetch + its empty-result retries.
    await vi.runAllTimersAsync()

    const initialCallCount = totalCalls()
    expect(initialCallCount).toBeGreaterThan(0)

    // Advance 30 seconds; local mode must not fall back to polling
    // now that live updates are driven by StreamLocalLogs.
    await vi.advanceTimersByTimeAsync(30000)

    expect(totalCalls()).toBe(initialCallCount)
  })
})
