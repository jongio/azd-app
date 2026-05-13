/**
 * Tests for the shared Connect transport / client factories.
 *
 * Scope: factory wiring, default-transport singleton behavior, and the
 * test-only override hook. The actual wire (connect-web fetch transport)
 * is verified end-to-end in per-hook tests against `createRouterTransport`.
 */
import { createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { afterEach, describe, expect, it } from 'vitest'

import {
  __setDefaultTransportForTesting,
  createHealthClient,
  createLifecycleClient,
  createModeClient,
  createProjectClient,
  getDefaultTransport,
} from './connectClient'
import { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_connect.js'
import {
  CodespaceInfo,
  GetEnvironmentResponse,
} from '@/gen/proto/azdapp/v1/lifecycle_pb.js'

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
            new GetEnvironmentResponse({
              codespace: new CodespaceInfo({ enabled: true, name: 'router-cs' }),
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
})
