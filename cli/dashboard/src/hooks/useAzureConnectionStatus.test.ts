/**
 * Tests for useAzureConnectionStatus.
 *
 * The hook moved off raw fetch onto the ModeService Connect client, so
 * these tests inject an in-memory router transport instead of mocking
 * globalThis.fetch. The production code path (client → transport →
 * handler) runs unchanged; only the wire is stubbed.
 *
 * No vi.fn() mock of fetch and no mocking of the generated client live
 * here on purpose. If a future test reaches for either, that's a
 * signal the test seam is wrong, not the test.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { Code, ConnectError, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { useAzureConnectionStatus } from './useAzureConnectionStatus'
import type { LogMode } from '@/components/ModeToggle'
import { ModeService } from '@/gen/proto/azdapp/v1/mode_pb.js'
import {
  GetModeResponseSchema,
  type GetModeResponse,
  LogMode as ProtoLogMode,
  SetModeResponseSchema,
  type SetModeResponse,
} from '@/gen/proto/azdapp/v1/mode_pb.js'

interface RouterOverrides {
  getMode?: () => Promise<GetModeResponse> | GetModeResponse
  setMode?: (mode: ProtoLogMode) => Promise<SetModeResponse> | SetModeResponse
}

/**
 * Build an in-memory router serving ModeService. Each test passes the
 * scenario it cares about; unimplemented methods raise CodeUnimplemented
 * automatically: exactly what we want when a test should not hit a
 * given RPC.
 */
function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(ModeService, {
      async getMode() {
        if (overrides.getMode) {
          return overrides.getMode()
        }
        return create(GetModeResponseSchema, {
          mode: ProtoLogMode.LOCAL,
          azureEnabled: false,
          azureStatus: 'disabled',
          azureRealtime: false,
          connectionMessage: '',
        })
      },
      async setMode(req) {
        if (overrides.setMode) {
          return overrides.setMode(req.mode)
        }
        return create(SetModeResponseSchema, {
          mode: req.mode,
          azureEnabled: req.mode === ProtoLogMode.AZURE,
          azureStatus: req.mode === ProtoLogMode.AZURE ? 'connected' : 'disabled',
          azureRealtime: false,
          connectionMessage: '',
        })
      },
    })
  })
}

describe('useAzureConnectionStatus (Connect)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial render', () => {
    it('does not auto-fetch, fetchAzureStatus must be called explicitly', async () => {
      let calls = 0
      const transport = makeTransport({
        getMode: () => {
          calls += 1
          return create(GetModeResponseSchema, { mode: ProtoLogMode.LOCAL })
        },
      })

      renderHook(() => useAzureConnectionStatus({ transport }))

      // Give microtasks + a tick a chance to fire any spurious fetches.
      await new Promise((r) => setTimeout(r, 50))
      expect(calls).toBe(0)
    })
  })

  describe('fetchAzureStatus', () => {
    it('populates state from a successful GetMode response', async () => {
      const transport = makeTransport({
        getMode: () =>
          create(GetModeResponseSchema, {
            mode: ProtoLogMode.AZURE,
            azureEnabled: true,
            azureStatus: 'connected',
            azureRealtime: true,
            connectionMessage: '',
          }),
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(result.current.logMode).toBe('azure')
      expect(result.current.azureEnabled).toBe(true)
      expect(result.current.azureStatus).toBe('connected')
      expect(result.current.azureConnectionMessage).toBeUndefined()
    })

    it('treats empty connectionMessage as undefined to match legacy semantics', async () => {
      const transport = makeTransport({
        getMode: () =>
          create(GetModeResponseSchema, {
            mode: ProtoLogMode.LOCAL,
            azureEnabled: false,
            azureStatus: 'disabled',
            connectionMessage: '',
          }),
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(result.current.azureConnectionMessage).toBeUndefined()
    })

    it('forces status to "disabled" when azureEnabled is false even if server says otherwise', async () => {
      // Defensive normalisation: a server bug claiming status=connected
      // while enabled=false would otherwise mislead the toggle UI.
      const transport = makeTransport({
        getMode: () =>
          create(GetModeResponseSchema, {
            mode: ProtoLogMode.LOCAL,
            azureEnabled: false,
            azureStatus: 'connected',
            connectionMessage: 'Azure logging not configured in azure.yaml',
          }),
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(result.current.azureEnabled).toBe(false)
      expect(result.current.azureStatus).toBe('disabled')
      expect(result.current.azureConnectionMessage).toBe('Azure logging not configured in azure.yaml')
    })

    it('dedupes concurrent fetchAzureStatus calls', async () => {
      let calls = 0
      let resolve: (resp: GetModeResponse) => void = () => undefined
      const pending = new Promise<GetModeResponse>((r) => { resolve = r })

      const transport = makeTransport({
        getMode: () => {
          calls += 1
          return pending
        },
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      // Fire the first call and let the transport land in the handler
      // before we issue the dedup-victims; createRouterTransport is
      // async end-to-end, so calls=0 until the first request reaches
      // the in-memory router.
      let firstSettled: Promise<void> = Promise.resolve()
      act(() => {
        firstSettled = result.current.fetchAzureStatus()
      })
      await waitFor(() => expect(calls).toBe(1))

      // Now issue concurrent calls; they must all bail out at the
      // abortControllerRef guard before reaching the transport.
      act(() => {
        void result.current.fetchAzureStatus()
        void result.current.fetchAzureStatus()
      })
      // Give microtasks a chance; calls must stay at 1.
      await Promise.resolve()
      expect(calls).toBe(1)

      // Settle the in-flight call; a fresh fetch is now allowed.
      await act(async () => {
        resolve(create(GetModeResponseSchema, { mode: ProtoLogMode.LOCAL }))
        await firstSettled
      })

      await act(async () => {
        await result.current.fetchAzureStatus()
      })
      expect(calls).toBe(2)
    })
  })

  describe('mode switching', () => {
    it('issues SetMode and updates state on success', async () => {
      let setModeMode: ProtoLogMode | undefined
      const transport = makeTransport({
        setMode: (mode) => {
          setModeMode = mode
          return create(SetModeResponseSchema, {
            mode,
            azureEnabled: true,
            azureStatus: 'connected',
            azureRealtime: false,
            connectionMessage: '',
          })
        },
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      expect(setModeMode).toBe(ProtoLogMode.AZURE)
      expect(result.current.logMode).toBe('azure')
      expect(result.current.azureEnabled).toBe(true)
      expect(result.current.azureStatus).toBe('connected')
    })

    it('does nothing when the requested mode equals the current mode', async () => {
      let setCalls = 0
      const transport = makeTransport({
        setMode: (mode) => {
          setCalls += 1
          return create(SetModeResponseSchema, { mode })
        },
      })

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      // Initial state is 'local'; switching to 'local' should be a no-op.
      await act(async () => {
        await result.current.handleLogModeChange('local')
      })

      expect(setCalls).toBe(0)
      expect(result.current.isModeSwitching).toBe(false)
    })

    it('keeps the previous mode when SetMode fails (FailedPrecondition)', async () => {
      const transport = makeTransport({
        setMode: () => {
          throw new ConnectError(
            'Azure logging not configured. Add logs.analytics section to azure.yaml',
            Code.FailedPrecondition
          )
        },
      })

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      expect(result.current.logMode).toBe('local')
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining(`Failed to switch mode to 'azure'`)
      )
    })

    it('toggles isModeSwitching during the change and clears it after the timeout', async () => {
      vi.useFakeTimers()
      try {
        let resolveSet: (resp: SetModeResponse) => void = () => undefined
        const pending = new Promise<SetModeResponse>((r) => { resolveSet = r })

        const transport = makeTransport({
          setMode: (mode) => {
            void mode
            return pending
          },
        })

        const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

        act(() => {
          void result.current.handleLogModeChange('azure')
        })

        // Switching flips on synchronously.
        expect(result.current.isModeSwitching).toBe(true)

        // Resolve the SetMode promise; switching stays true until the
        // 1500ms cleanup timeout fires.
        await act(async () => {
          resolveSet(create(SetModeResponseSchema, {
            mode: ProtoLogMode.AZURE,
            azureEnabled: true,
            azureStatus: 'connected',
          }))
          await Promise.resolve()
        })

        expect(result.current.isModeSwitching).toBe(true)

        await act(async () => {
          vi.advanceTimersByTime(1500)
          await Promise.resolve()
        })

        expect(result.current.isModeSwitching).toBe(false)
      } finally {
        vi.useRealTimers()
      }
    })

    it('rejects an invalid mode value before any RPC fires', async () => {
      let setCalls = 0
      const transport = makeTransport({
        setMode: (mode) => {
          setCalls += 1
          return create(SetModeResponseSchema, { mode })
        },
      })
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))

      await act(async () => {
        await result.current.handleLogModeChange('bogus' as LogMode)
      })

      expect(setCalls).toBe(0)
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Invalid mode: bogus')
      )
    })
  })

  describe('error handling', () => {
    it('logs a warning on GetMode failure but leaves status untouched', async () => {
      const transport = makeTransport({
        getMode: () => {
          throw new ConnectError('boom', Code.Internal)
        },
      })
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined)

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(result.current.azureStatus).toBe('disabled')
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining('Failed to fetch mode')
      )
    })

    it('swallows Code.Canceled silently (no warn, no state change)', async () => {
      const transport = makeTransport({
        getMode: () => {
          throw new ConnectError('canceled by client', Code.Canceled)
        },
      })
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => undefined)

      const { result } = renderHook(() => useAzureConnectionStatus({ transport }))
      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      expect(consoleSpy).not.toHaveBeenCalled()
      expect(result.current.azureStatus).toBe('disabled')
    })
  })

  describe('realtime config callback', () => {
    it('forwards azureRealtime from GetMode to onAzureRealtimeConfig', async () => {
      const onAzureRealtimeConfig = vi.fn()
      const transport = makeTransport({
        getMode: () =>
          create(GetModeResponseSchema, {
            mode: ProtoLogMode.AZURE,
            azureEnabled: true,
            azureStatus: 'connected',
            azureRealtime: true,
          }),
      })

      const { result } = renderHook(() =>
        useAzureConnectionStatus({ transport, onAzureRealtimeConfig })
      )

      await act(async () => {
        await result.current.fetchAzureStatus()
      })

      await waitFor(() => {
        expect(onAzureRealtimeConfig).toHaveBeenCalledWith(true)
      })
    })

    it('also forwards azureRealtime from a successful SetMode', async () => {
      const onAzureRealtimeConfig = vi.fn()
      const transport = makeTransport({
        setMode: (mode) =>
          create(SetModeResponseSchema, {
            mode,
            azureEnabled: true,
            azureStatus: 'connected',
            azureRealtime: true,
          }),
      })

      const { result } = renderHook(() =>
        useAzureConnectionStatus({ transport, onAzureRealtimeConfig })
      )

      await act(async () => {
        await result.current.handleLogModeChange('azure')
      })

      expect(onAzureRealtimeConfig).toHaveBeenCalledWith(true)
    })
  })

  describe('cleanup', () => {
    it('clears the mode-switch timeout on unmount without throwing', () => {
      vi.useFakeTimers()
      try {
        const transport = makeTransport({
          setMode: (mode) => create(SetModeResponseSchema, { mode }),
        })

        const { result, unmount } = renderHook(() => useAzureConnectionStatus({ transport }))

        act(() => {
          void result.current.handleLogModeChange('azure')
        })

        unmount()

        // Advancing past the cleanup timeout must be a no-op (no
        // setState-after-unmount warnings, no exceptions).
        act(() => {
          vi.advanceTimersByTime(2000)
        })
      } finally {
        vi.useRealTimers()
      }
    })
  })
})
