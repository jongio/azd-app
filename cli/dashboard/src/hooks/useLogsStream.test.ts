/**
 * Tests for useLogsStream — the orchestrator hook that wires the
 * historical-fetch path together with the live shared stream.
 *
 * After the WebSocket -> Connect migration, all transport-specific
 * concerns moved into useSharedLogStream (and its sibling Azure
 * manager). This file now exercises the orchestration logic in
 * isolation by mocking useSharedLogStream; transport behaviour is
 * covered separately by useSharedLogStream.test.ts.
 */
import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

import { useLogsStream } from './useLogsStream'
import type { LogMode } from '@/components/ModeToggle'

vi.mock('@/hooks/useBackendConnection', () => ({
  useBackendConnection: () => ({ connected: true }),
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

describe('useLogsStream (orchestration)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    sharedLogStreamMock.mockClear()
    sharedLogStreamMock.mockImplementation(() => ({
      connectionState: 'disconnected',
      droppedCount: 0,
    }))
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
      text: () => Promise.resolve(''),
    })
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
