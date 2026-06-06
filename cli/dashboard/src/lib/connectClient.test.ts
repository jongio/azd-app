/**
 * Tests for the shared Connect transport / client factories.
 *
 * Scope: factory wiring, default-transport singleton behavior, and the
 * test-only override hook. The actual wire (connect-web fetch transport)
 * is verified end-to-end in per-hook tests against `createRouterTransport`.
 */
import { createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  __setDefaultTransportForTesting,
  createHealthClient,
  createLifecycleClient,
  createModeClient,
  createProjectClient,
  getDefaultTransport,
} from './connectClient'
import { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_pb.js'
import {
  CodespaceInfoSchema,
  GetEnvironmentResponseSchema,
} from '@/gen/proto/azdapp/v1/lifecycle_pb.js'
import { create } from '@bufbuild/protobuf'

afterEach(() => {
  __setDefaultTransportForTesting(null)
})

describe('connectClient factories', () => {
  it('returns a Connect client for each service', () => {
    const transport = createRouterTransport(() => {
      // No services registered; we only need a transport instance.
    })

    const lifecycle = createLifecycleClient(transport)
    const mode = createModeClient(transport)
    const project = createProjectClient(transport)
    const health = createHealthClient(transport)

    expect(typeof lifecycle.getEnvironment).toBe('function')
    expect(typeof lifecycle.streamBroadcast).toBe('function')
    expect(typeof mode.getMode).toBe('function')
    expect(typeof mode.setMode).toBe('function')
    expect(typeof project.getProject).toBe('function')
    expect(typeof health.getHealth).toBe('function')
    expect(typeof health.streamHealth).toBe('function')
    expect(typeof health.streamStateTransitions).toBe('function')
  })

  it('routes calls through the supplied transport', async () => {
    let received = 0
    const transport = createRouterTransport((router: ConnectRouter) => {
      router.service(LifecycleService, {
        getEnvironment() {
          received += 1
          return Promise.resolve(
            create(GetEnvironmentResponseSchema, {
              codespace: create(CodespaceInfoSchema, { enabled: true, name: 'router-cs' }),
              environmentName: 'env-from-router',
            })
          )
        },
      })
    })

    const client = createLifecycleClient(transport)
    const resp = await client.getEnvironment({})

    expect(received).toBe(1)
    expect(resp.environmentName).toBe('env-from-router')
    expect(resp.codespace?.name).toBe('router-cs')
  })

  it('memoises the default transport across getDefaultTransport calls', () => {
    __setDefaultTransportForTesting(null) // ensure fresh
    const a = getDefaultTransport()
    const b = getDefaultTransport()
    expect(a).toBe(b)
  })

  it('__setDefaultTransportForTesting overrides and resets the singleton', () => {
    const stub = createRouterTransport(() => {})
    __setDefaultTransportForTesting(stub)
    expect(getDefaultTransport()).toBe(stub)

    __setDefaultTransportForTesting(null)
    const fresh = getDefaultTransport()
    expect(fresh).not.toBe(stub)
  })

  // ── SEC-023: closure-scoped session token ──────────────────────────────────

  it('reads the session-token meta tag exactly once at transport construction, not per request', () => {
    // Insert a real meta tag so jsdom's querySelector finds it.
    const meta = document.createElement('meta')
    meta.setAttribute('name', 'azd-session-token')
    meta.setAttribute('content', 'sec023-closure-test-token')
    document.head.appendChild(meta)

    const querySpy = vi.spyOn(document, 'querySelector')
    try {
      __setDefaultTransportForTesting(null) // ensure fresh construction
      getDefaultTransport()

      // The token selector must have been queried exactly once — at
      // construction time — and must not be re-queried per request.
      const tokenQueryCount = querySpy.mock.calls.filter(
        ([selector]) => selector === 'meta[name="azd-session-token"]'
      ).length
      expect(tokenQueryCount).toBe(1)
    } finally {
      querySpy.mockRestore()
      document.head.removeChild(meta)
    }
  })

  it('does not expose the session token value via the window object', () => {
    // Use a canary string that would be conspicuous if it leaked to window.
    const canary = 'sec023-canary-should-not-appear-on-window'
    const meta = document.createElement('meta')
    meta.setAttribute('name', 'azd-session-token')
    meta.setAttribute('content', canary)
    document.head.appendChild(meta)

    try {
      __setDefaultTransportForTesting(null)
      getDefaultTransport()

      // The raw token string must not be reachable as a window property.
      expect((window as unknown as Record<string, unknown>)[canary]).toBeUndefined()
      // Nor should there be an 'azd-session-token' key on window.
      expect((window as unknown as Record<string, unknown>)['azd-session-token']).toBeUndefined()
    } finally {
      document.head.removeChild(meta)
    }
  })
})
