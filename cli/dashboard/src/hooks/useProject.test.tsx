/**
 * Tests for useProject against an in-memory Connect router transport.
 * Mirrors useCodespaceEnv.test.tsx: no fetch mocking, no client mocking;
 * the production hook code path runs unchanged with an injected transport.
 */
import { renderHook, waitFor } from '@testing-library/react'
import { Code, ConnectError, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'
import { describe, expect, it } from 'vitest'

import { useProject } from './useProject'
import { ProjectService } from '@/gen/proto/azdapp/v1/project_pb.js'
import { GetProjectResponseSchema, type GetProjectResponse } from '@/gen/proto/azdapp/v1/project_pb.js'
import { create } from '@bufbuild/protobuf'

interface RouterOverrides {
  getProject?: () => Promise<GetProjectResponse> | GetProjectResponse
}

function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(ProjectService, {
      async getProject() {
        if (overrides.getProject) {
          return overrides.getProject()
        }
        return create(GetProjectResponseSchema, { name: '', dir: '' })
      },
    })
  })
}

describe('useProject (Connect)', () => {
  it('starts in loading state and resolves to the server-reported name and dir', async () => {
    const transport = makeTransport({
      getProject: () =>
        create(GetProjectResponseSchema, {
          name: 'fullstack-demo',
          dir: '/abs/projects/fullstack',
        }),
    })

    const { result } = renderHook(() => useProject({ transport }))
    expect(result.current.loading).toBe(true)
    expect(result.current.name).toBe('')

    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.name).toBe('fullstack-demo')
    expect(result.current.dir).toBe('/abs/projects/fullstack')
    expect(result.current.error).toBeNull()
  })

  it('records an error message when GetProject fails', async () => {
    const transport = makeTransport({
      getProject: () => {
        throw new ConnectError('azure.yaml malformed', Code.Internal)
      },
    })

    const { result } = renderHook(() => useProject({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.error).toMatch(/azure\.yaml malformed/)
    expect(result.current.name).toBe('')
    expect(result.current.dir).toBe('')
  })

  it('preserves an empty project name from the server (no placeholder)', async () => {
    // The hook must propagate the proto zero value so callers (e.g.,
    // App.tsx) can apply their own fallback ("Project") rather than
    // having one baked in here.
    const transport = makeTransport({
      getProject: () => create(GetProjectResponseSchema, { name: '', dir: '/abs/projects/anon' }),
    })

    const { result } = renderHook(() => useProject({ transport }))
    await waitFor(() => expect(result.current.loading).toBe(false))

    expect(result.current.name).toBe('')
    expect(result.current.dir).toBe('/abs/projects/anon')
    expect(result.current.error).toBeNull()
  })

  it('does not record an error after unmount when a slow response arrives', async () => {
    let resolveLater: (resp: GetProjectResponse) => void = () => undefined
    const pending = new Promise<GetProjectResponse>((resolve) => {
      resolveLater = resolve
    })
    const transport = makeTransport({ getProject: () => pending })

    const { result, unmount } = renderHook(() => useProject({ transport }))
    expect(result.current.loading).toBe(true)
    unmount()

    // Resolve after unmount; the hook must swallow the response without
    // touching state. We can't observe state directly post-unmount, but
    // we can confirm no unhandled rejection bubbles up.
    resolveLater(create(GetProjectResponseSchema, { name: 'late', dir: '/late' }))
    await new Promise((r) => setTimeout(r, 10))
  })
})
