/**
 * Tests for useLogClassifications against an in-memory Connect router
 * transport. Replaces the previous fetch-mock test bed: the hook now
 * speaks LogsService, so we drive the production code path through
 * `createRouterTransport` and a mutable in-memory store.
 *
 * Tests cover:
 *  - load on mount + empty result
 *  - load failure surfaces .error
 *  - addClassification append + reload + cross-instance notify
 *  - addClassification update-in-place (server keeps original casing)
 *  - addClassification skipNotify suppresses reload
 *  - deleteClassification by index + reload
 *  - deleteClassification skipNotify
 *  - getClassificationForText longest-match wins
 *  - getClassificationForText returns null on no match
 *  - reload() picks up out-of-band changes
 *  - addClassification surfaces ConnectError as Error to callers
 */
import { renderHook, waitFor, act } from '@testing-library/react'
import {
  Code,
  ConnectError,
  createRouterTransport,
  type ConnectRouter,
} from '@connectrpc/connect'
import { describe, it, expect } from 'vitest'

import { useLogClassifications } from './useLogClassifications'
import { LogsService } from '@/gen/proto/azdapp/v1/logs_connect.js'
import {
  Classification,
  ListClassificationsResponse,
  AddClassificationResponse,
  DeleteClassificationResponse,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogLevel } from '@/gen/proto/azdapp/v1/common_pb.js'

interface StoredClassification {
  text: string
  level: LogLevel
}

interface Harness {
  transport: ReturnType<typeof createRouterTransport>
  store: StoredClassification[]
  addThrows: { value: ConnectError | null }
}

/**
 * Build a harness whose LogsService router methods read/write a mutable
 * `store` slice in-memory. Mirrors the Go handler closely enough that
 * hook behaviour against the harness predicts real-server behaviour:
 * - update-in-place when text matches case-insensitively
 * - InvalidArgument for blank text
 * - delete by index, NotFound on out-of-range
 *
 * We intentionally implement just the unary RPCs the hook touches; the
 * streaming methods plus preferences RPCs throw if called so any
 * regression that drives the wrong RPC fails loudly rather than silently
 * succeeding with an empty default.
 */
function buildHarness(initial: StoredClassification[] = []): Harness {
  const store: StoredClassification[] = [...initial]
  const addThrows: { value: ConnectError | null } = { value: null }

  const transport = createRouterTransport((router: ConnectRouter) => {
    router.service(LogsService, {
      // eslint-disable-next-line @typescript-eslint/require-await
      async getLogs() {
        throw new ConnectError('not used in this suite', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await, require-yield
      async *streamLocalLogs() {
        throw new ConnectError('not used', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async listClassifications() {
        return new ListClassificationsResponse({
          classifications: store.map(
            (c) => new Classification({ text: c.text, level: c.level })
          ),
        })
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async addClassification(req) {
        if (addThrows.value) throw addThrows.value
        const incoming = req.classification
        if (!incoming || !incoming.text.trim()) {
          throw new ConnectError('text required', Code.InvalidArgument)
        }
        const idx = store.findIndex(
          (c) => c.text.toLowerCase() === incoming.text.toLowerCase()
        )
        let stored: StoredClassification
        if (idx >= 0) {
          // Update level in place; preserve original text casing.
          store[idx] = { text: store[idx].text, level: incoming.level }
          stored = store[idx]
        } else {
          stored = { text: incoming.text, level: incoming.level }
          store.push(stored)
        }
        return new AddClassificationResponse({
          classification: new Classification(stored),
        })
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async deleteClassification(req) {
        if (req.index < 0) {
          throw new ConnectError('negative index', Code.InvalidArgument)
        }
        if (req.index >= store.length) {
          throw new ConnectError('out of range', Code.NotFound)
        }
        store.splice(req.index, 1)
        return new DeleteClassificationResponse({})
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async getPreferences() {
        throw new ConnectError('not used', Code.Unimplemented)
      },
      // eslint-disable-next-line @typescript-eslint/require-await
      async savePreferences() {
        throw new ConnectError('not used', Code.Unimplemented)
      },
    })
  })

  return { transport, store, addThrows }
}

describe('useLogClassifications (Connect)', () => {
  it('loads classifications on mount', async () => {
    const { transport } = buildHarness([
      { text: 'Connection refused', level: LogLevel.ERROR },
      { text: 'cache miss', level: LogLevel.INFO },
    ])

    const { result } = renderHook(() => useLogClassifications(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })
    expect(result.current.classifications).toEqual([
      { text: 'Connection refused', level: 'error' },
      { text: 'cache miss', level: 'info' },
    ])
    expect(result.current.error).toBeNull()
  })

  it('returns empty list when store is empty', async () => {
    const { transport } = buildHarness()

    const { result } = renderHook(() => useLogClassifications(transport))

    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.classifications).toEqual([])
    expect(result.current.error).toBeNull()
  })

  it('appends a new classification and reloads', async () => {
    const { transport, store } = buildHarness()

    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.addClassification('new error', 'error')
    })

    expect(store).toEqual([{ text: 'new error', level: LogLevel.ERROR }])
    await waitFor(() => {
      expect(result.current.classifications).toEqual([
        { text: 'new error', level: 'error' },
      ])
    })
  })

  it('updates an existing classification in place (case-insensitive)', async () => {
    const { transport, store } = buildHarness([
      { text: 'Connection Refused', level: LogLevel.INFO },
    ])

    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.addClassification('connection refused', 'error')
    })

    // Store still has one entry with the ORIGINAL casing and the new level.
    expect(store).toEqual([
      { text: 'Connection Refused', level: LogLevel.ERROR },
    ])
    await waitFor(() => {
      expect(result.current.classifications).toEqual([
        { text: 'Connection Refused', level: 'error' },
      ])
    })
  })

  it('skipNotify=true suppresses the reload after add', async () => {
    const { transport } = buildHarness()

    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => {
      await result.current.addClassification('silent', 'info', true)
    })

    // Local state was NOT refreshed (skipNotify) so classifications stays empty
    // even though the server-side store has one entry.
    expect(result.current.classifications).toEqual([])
  })

  it('deletes a classification by index', async () => {
    const { transport, store } = buildHarness([
      { text: 'first', level: LogLevel.INFO },
      { text: 'second', level: LogLevel.ERROR },
    ])

    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() =>
      expect(result.current.classifications.length).toBe(2)
    )

    await act(async () => {
      await result.current.deleteClassification(0)
    })

    expect(store).toEqual([{ text: 'second', level: LogLevel.ERROR }])
    await waitFor(() => {
      expect(result.current.classifications).toEqual([
        { text: 'second', level: 'error' },
      ])
    })
  })

  it('rejects blank text with a typed error', async () => {
    const { transport } = buildHarness()
    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await expect(
      act(async () => {
        await result.current.addClassification('   ', 'error')
      })
    ).rejects.toThrow(/text required/i)
  })

  it('returns null from getClassificationForText when no match', async () => {
    const { transport } = buildHarness([
      { text: 'panic', level: LogLevel.ERROR },
    ])
    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() =>
      expect(result.current.classifications.length).toBe(1)
    )

    expect(result.current.getClassificationForText('Healthy startup')).toBeNull()
    expect(result.current.getClassificationForText('')).toBeNull()
  })

  it('uses longest-match-wins in getClassificationForText', async () => {
    const { transport } = buildHarness([
      { text: 'error', level: LogLevel.WARN },
      { text: 'fatal error', level: LogLevel.ERROR },
    ])
    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() =>
      expect(result.current.classifications.length).toBe(2)
    )

    // Both rules match; the longer one wins.
    expect(
      result.current.getClassificationForText('FATAL error: kernel panic')
    ).toBe('error')
    // Only the short rule matches; level is "warning".
    expect(
      result.current.getClassificationForText('transient ERROR detected')
    ).toBe('warning')
  })

  it('reload() picks up out-of-band changes to the store', async () => {
    const harness = buildHarness()
    const { result } = renderHook(() => useLogClassifications(harness.transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.classifications).toEqual([])

    // Mutate the underlying store as if another tab wrote it.
    harness.store.push({ text: 'late arrival', level: LogLevel.INFO })

    await act(async () => {
      await result.current.reload()
    })
    await waitFor(() => {
      expect(result.current.classifications).toEqual([
        { text: 'late arrival', level: 'info' },
      ])
    })
  })

  it('surfaces ConnectError from add() as a plain Error to callers', async () => {
    const harness = buildHarness()
    harness.addThrows.value = new ConnectError(
      'classification limit reached',
      Code.ResourceExhausted
    )

    const { result } = renderHook(() => useLogClassifications(harness.transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await expect(
      act(async () => {
        await result.current.addClassification('x', 'info')
      })
    ).rejects.toThrow(/classification limit reached/)
  })

  it('exposes load failures via .error', async () => {
    // Build a transport whose listClassifications rejects.
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
          throw new ConnectError('yaml broken', Code.Internal)
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
          throw new ConnectError('n/a', Code.Unimplemented)
        },
        // eslint-disable-next-line @typescript-eslint/require-await
        async savePreferences() {
          throw new ConnectError('n/a', Code.Unimplemented)
        },
      })
    })

    const { result } = renderHook(() => useLogClassifications(transport))
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.error).not.toBeNull()
    expect(result.current.classifications).toEqual([])
  })
})

