/**
 * Tests for useCodespaceEnv against a real Connect client wired to an
 * in-memory router transport (`createRouterTransport`). The hook code
 * path under test is identical to production; only the wire is stubbed.
 *
 * No `vi.fn()` mock of `fetch` and no monkey-patching of the generated
 * client live here on purpose. If a future test reaches for those, that's
 * a signal the abstraction is wrong, not the test.
 */
import { act, renderHook, waitFor } from '@testing-library/react'
import { ConnectError, Code, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { useCodespaceEnv } from './useCodespaceEnv'
import { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_connect.js'
import {
  CodespaceInfo,
  GetEnvironmentResponse,
} from '@/gen/proto/azdapp/v1/lifecycle_pb.js'

interface RouterOverrides {
  getEnvironment?: () => Promise<GetEnvironmentResponse> | GetEnvironmentResponse
}

/**
 * Build an in-memory router transport that serves LifecycleService.
 * Tests pass a `getEnvironment` impl matching the scenario under test;
 * unimplemented methods raise CodeUnimplemented automatically, which is
 * the right behavior for this hook (it only calls GetEnvironment).
 */
function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(LifecycleService, {
      async getEnvironment() {
        if (overrides.getEnvironment) {
          return overrides.getEnvironment()
        }
        return new GetEnvironmentResponse({
          codespace: new CodespaceInfo({
            enabled: false,
            name: '',
            domain: '',
            isVsCodeDesktop: false,
          }),
          environmentName: '',
        })
      },
    })
  })
}

describe('useCodespaceEnv (Connect)', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })
  afterEach(() => {
    sessionStorage.clear()
  })

  it('reports defaults when not in a Codespace', async () => {
    const transport = makeTransport()
    const { result } = renderHook(() => useCodespaceEnv({ transport }))

    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.isCodespace).toBe(false)
    expect(result.current.config?.enabled).toBe(false)
    expect(result.current.environmentName).toBeUndefined()
    expect(result.current.error).toBeNull()
  })

  it('surfaces Codespace info from the GetEnvironment response', async () => {
    const transport = makeTransport({
      getEnvironment: () =>
        new GetEnvironmentResponse({
          codespace: new CodespaceInfo({
            enabled: true,
            name: 'silver-space-xyzzy',
            domain: 'app.github.dev',
            isVsCodeDesktop: false,
          }),
          environmentName: 'dev',
        }),
    })

    const { result } = renderHook(() => useCodespaceEnv({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.isCodespace).toBe(true)
    expect(result.current.config).toEqual({
      enabled: true,
      name: 'silver-space-xyzzy',
      domain: 'app.github.dev',
      isVsCodeDesktop: false,
    })
    expect(result.current.environmentName).toBe('dev')
  })

  it('caches the response in sessionStorage and skips fetch on remount', async () => {
    let callCount = 0
    const transport = makeTransport({
      getEnvironment: () => {
        callCount += 1
        return new GetEnvironmentResponse({
          codespace: new CodespaceInfo({
            enabled: true,
            name: 'cached-codespace',
            domain: 'app.github.dev',
          }),
          environmentName: 'cached-env',
        })
      },
    })

    const first = renderHook(() => useCodespaceEnv({ transport }))
    await waitFor(() => expect(first.result.current.loading).toBe(false))
    expect(callCount).toBe(1)
    first.unmount()

    // Second hook instance must hydrate from cache without a second wire call.
    const second = renderHook(() => useCodespaceEnv({ transport }))
    expect(second.result.current.loading).toBe(false)
    expect(second.result.current.config?.name).toBe('cached-codespace')
    expect(second.result.current.environmentName).toBe('cached-env')
    expect(callCount).toBe(1)
  })

  it('refresh() clears cache and re-fetches', async () => {
    let callCount = 0
    const transport = makeTransport({
      getEnvironment: () => {
        callCount += 1
        return new GetEnvironmentResponse({
          codespace: new CodespaceInfo({
            enabled: callCount === 1,
            name: callCount === 1 ? 'first' : 'second',
            domain: 'app.github.dev',
          }),
          environmentName: callCount === 1 ? 'env-a' : 'env-b',
        })
      },
    })

    const { result } = renderHook(() => useCodespaceEnv({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.config?.name).toBe('first')

    act(() => {
      result.current.refresh()
    })
    await waitFor(() => expect(result.current.config?.name).toBe('second'))
    expect(result.current.environmentName).toBe('env-b')
    expect(callCount).toBe(2)
  })

  it('records an error message when the RPC fails', async () => {
    const transport = makeTransport({
      getEnvironment: () => {
        throw new ConnectError('lifecycle exploded', Code.Internal)
      },
    })

    const { result } = renderHook(() => useCodespaceEnv({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.error).toMatch(/lifecycle exploded/)
    // Config stays null because we have no prior cached value to fall back on.
    expect(result.current.config).toBeNull()
  })

  it('treats empty environmentName as undefined to match legacy semantics', async () => {
    const transport = makeTransport({
      getEnvironment: () =>
        new GetEnvironmentResponse({
          codespace: new CodespaceInfo({ enabled: false }),
          environmentName: '',
        }),
    })

    const { result } = renderHook(() => useCodespaceEnv({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.environmentName).toBeUndefined()
  })
})
