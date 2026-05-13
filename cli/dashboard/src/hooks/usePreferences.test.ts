/**
 * Tests for usePreferences against an in-memory Connect router transport.
 *
 * Validates the round-trip from proto Preferences <-> dashboard
 * UserPreferences shape, defaults fallback, optimistic update on save,
 * and theme/UI mutation helpers. Uses createRouterTransport so the hook
 * runs the production code path against a mutable in-memory store.
 */
import { renderHook, waitFor, act } from '@testing-library/react'
import {
  Code,
  ConnectError,
  createRouterTransport,
  type ConnectRouter,
} from '@connectrpc/connect'
import { describe, it, expect } from 'vitest'

import { usePreferences } from './usePreferences'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_pb.js'
import {
  PreferencesSchema,
  type Preferences,
  UIPreferencesSchema,
  BehaviorPreferencesSchema,
  CopyPreferencesSchema,
  GetPreferencesResponseSchema,
  SavePreferencesResponseSchema,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { create } from '@bufbuild/protobuf'

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  state: { value: Preferences | undefined }
}

function buildHarness(initial?: Preferences): Harness {
  const state: { value: Preferences | undefined } = { value: initial }

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(LogsService, {
      // eslint-disable-next-line @typescript-eslint/require-await
      async getLogs() {
        throw new ConnectError('n/a', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await, require-yield
      async *streamLocalLogs() {
        throw new ConnectError('n/a', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async listClassifications() {
        throw new ConnectError('n/a', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async addClassification() {
        throw new ConnectError('n/a', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async deleteClassification() {
        throw new ConnectError('n/a', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async getPreferences() {
        return create(GetPreferencesResponseSchema, { preferences: state.value })
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async savePreferences(req) {
        // Echo back what was saved (mirrors the real handler that
        // protojson-roundtrips the input).
        state.value = req.preferences
        return create(SavePreferencesResponseSchema, { preferences: req.preferences })
      },
    })
  })

  return { transport, state }
}

describe('usePreferences (Connect)', () => {
  it('returns defaults when server has no stored preferences', async () => {
    const { transport } = buildHarness()
    const { result } = renderHook(() => usePreferences(transport))

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.preferences.version).toBe('1.0')
    expect(result.current.preferences.theme).toBe('light')
    expect(result.current.preferences.ui.gridColumns).toBe(2)
    expect(result.current.preferences.behavior.autoScroll).toBe(true)
    expect(result.current.preferences.copy.defaultFormat).toBe('plaintext')
  })

  it('decodes a stored Preferences message including theme and gridAutoFit', async () => {
    const stored = create(PreferencesSchema, {
      version: '1.0',
      theme: 'dark',
      ui: create(UIPreferencesSchema, {
        gridColumns: 4,
        gridAutoFit: true,
        viewMode: 'unified',
        selectedServices: ['api'],
      }),
      behavior: create(BehaviorPreferencesSchema, {
        autoScroll: false,
        pauseOnScroll: true,
        timestampFormat: 'iso',
      }),
      copy: create(CopyPreferencesSchema, {
        defaultFormat: 'json',
        includeTimestamp: false,
        includeService: true,
      }),
    })
    const { transport } = buildHarness(stored)
    const { result } = renderHook(() => usePreferences(transport))

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.preferences.theme).toBe('dark')
    expect(result.current.preferences.ui.gridColumns).toBe(4)
    expect(result.current.preferences.ui.gridAutoFit).toBe(true)
    expect(result.current.preferences.ui.viewMode).toBe('unified')
    expect(result.current.preferences.ui.selectedServices).toEqual(['api'])
    expect(result.current.preferences.behavior.autoScroll).toBe(false)
    expect(result.current.preferences.copy.defaultFormat).toBe('json')
  })

  it('falls back to defaults when stored fields are out of range', async () => {
    // gridColumns=99 is out of [1,6]; viewMode=foo is invalid; theme=neon
    // is invalid. All three should fall back to defaults rather than
    // propagate junk into the UI.
    const stored = create(PreferencesSchema, {
      version: '1.0',
      theme: 'neon',
      ui: create(UIPreferencesSchema, {
        gridColumns: 99,
        viewMode: 'foo',
      }),
      copy: create(CopyPreferencesSchema, { defaultFormat: 'binary' }),
    })
    const { transport } = buildHarness(stored)
    const { result } = renderHook(() => usePreferences(transport))

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.preferences.theme).toBe('light')
    expect(result.current.preferences.ui.gridColumns).toBe(2)
    expect(result.current.preferences.ui.viewMode).toBe('grid')
    expect(result.current.preferences.copy.defaultFormat).toBe('plaintext')
  })

  it('savePreferences merges updates and round-trips through the server', async () => {
    const { transport, state } = buildHarness()
    const { result } = renderHook(() => usePreferences(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.savePreferences({ theme: 'dark' })
    })

    expect(result.current.preferences.theme).toBe('dark')
    // Other fields preserved.
    expect(result.current.preferences.ui.gridColumns).toBe(2)
    // Server received the merged Preferences.
    expect(state.value?.theme).toBe('dark')
  })

  it('updateUI mutates only the ui slice', async () => {
    const { transport, state } = buildHarness()
    const { result } = renderHook(() => usePreferences(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.updateUI({ gridColumns: 5, viewMode: 'unified' })
    })

    await waitFor(() => {
      expect(result.current.preferences.ui.gridColumns).toBe(5)
      expect(result.current.preferences.ui.viewMode).toBe('unified')
    })
    // Other slices unchanged.
    expect(result.current.preferences.theme).toBe('light')
    expect(state.value?.ui?.gridColumns).toBe(5)
  })

  it('setTheme is a thin wrapper over savePreferences', async () => {
    const { transport, state } = buildHarness()
    const { result } = renderHook(() => usePreferences(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    act(() => {
      result.current.setTheme('dark')
    })

    await waitFor(() => {
      expect(result.current.preferences.theme).toBe('dark')
    })
    expect(state.value?.theme).toBe('dark')
  })

  it('reload() picks up out-of-band server changes', async () => {
    const { transport, state } = buildHarness()
    const { result } = renderHook(() => usePreferences(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    state.value = create(PreferencesSchema, {
      version: '1.0',
      theme: 'dark',
      ui: create(UIPreferencesSchema, { gridColumns: 6, viewMode: 'grid' }),
    })

    await act(async () => {
      await result.current.reload()
    })

    expect(result.current.preferences.theme).toBe('dark')
    expect(result.current.preferences.ui.gridColumns).toBe(6)
  })

  it('keeps optimistic state when save fails', async () => {
    let throwOnce = true
    const transport = createRouterTransport((router: ConnectRouter) => {
      router.service(LogsService, {
        // eslint-disable-next-line @typescript-eslint/require-await
        async getLogs() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await, require-yield
        async *streamLocalLogs() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async listClassifications() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async addClassification() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async deleteClassification() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async getPreferences() {
          return create(GetPreferencesResponseSchema, { preferences: undefined })
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async savePreferences() {
          if (throwOnce) {
            throwOnce = false
            throw new ConnectError('disk full', Code.ResourceExhausted)
          }
          return create(SavePreferencesResponseSchema, {})
        },
      })
    })

    const { result } = renderHook(() => usePreferences(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.savePreferences({ theme: 'dark' })
    })

    // Optimistic update preserved despite the server error.
    expect(result.current.preferences.theme).toBe('dark')
  })
})
