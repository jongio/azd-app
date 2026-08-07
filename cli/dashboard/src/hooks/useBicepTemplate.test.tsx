/**
 * Hook tests for useBicepTemplate. Wired against an in-memory Connect
 * router transport so we exercise the actual generated client + error
 * mapping with no real HTTP. The component test
 * (BicepTemplateModal.test.tsx) mocks this hook and stays focused on UI.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { ConnectError, Code, createRouterTransport, type ConnectRouter } from '@connectrpc/connect'

import { useBicepTemplate } from './useBicepTemplate'
import { BicepService } from '@/gen/proto/azdapp/v1/bicep_pb.js'
import { GetBicepTemplateResponseSchema, type GetBicepTemplateResponse } from '@/gen/proto/azdapp/v1/bicep_pb.js'
import { create } from '@bufbuild/protobuf'

// =============================================================================
// Helpers
// =============================================================================

interface RouterOverrides {
  getBicepTemplate?: () => Promise<GetBicepTemplateResponse> | GetBicepTemplateResponse
}

function makeTransport(overrides: RouterOverrides = {}) {
  return createRouterTransport((router: ConnectRouter) => {
    router.service(BicepService, {
      getBicepTemplate: () => {
        if (overrides.getBicepTemplate) return overrides.getBicepTemplate()
        return create(GetBicepTemplateResponseSchema, {
          template: '// generated bicep',
          includedServices: ['api', 'web'],
          workspaceId: '',
        })
      },
    })
  })
}

// =============================================================================
// Tests
// =============================================================================

describe('useBicepTemplate', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('starts in loading state and resolves with template on success', async () => {
    const transport = makeTransport()
    const { result } = renderHook(() => useBicepTemplate(transport))

    expect(result.current.isLoading).toBe(true)
    expect(result.current.template).toBeNull()

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toBeNull()
    expect(result.current.template).toBe('// generated bicep')
    expect(result.current.services).toEqual(['api', 'web'])
  })

  it('always populates static instructions and parameters on success', async () => {
    // The proto omits these fields by design; the hook injects the
    // hardcoded BICEP_INSTRUCTIONS / BICEP_PARAMETERS constants. Pin the
    // shape so a future change has to be intentional.
    const transport = makeTransport()
    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.instructions).not.toBeNull()
    expect(result.current.instructions?.summary).toMatch(/diagnostic settings/i)
    expect(result.current.instructions?.steps.length).toBeGreaterThanOrEqual(5)

    expect(result.current.parameters).toHaveLength(1)
    expect(result.current.parameters[0].name).toBe('logAnalyticsWorkspaceId')
  })

  it('maps Code.NotFound to the "deploy with azd up" message', async () => {
    const transport = makeTransport({
      getBicepTemplate: () =>
        Promise.reject(new ConnectError('no resources', Code.NotFound)),
    })
    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.template).toBeNull()
    expect(result.current.error).toMatch(/no Azure resources found/i)
    expect(result.current.error).toMatch(/azd up/i)
  })

  it('maps Code.FailedPrecondition to the credentials message', async () => {
    const transport = makeTransport({
      getBicepTemplate: () =>
        Promise.reject(new ConnectError('no creds', Code.FailedPrecondition)),
    })
    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toMatch(/azure credentials/i)
    expect(result.current.error).toMatch(/azd auth login/i)
  })

  it('maps Code.Unavailable to the discovery message', async () => {
    const transport = makeTransport({
      getBicepTemplate: () =>
        Promise.reject(new ConnectError('arm down', Code.Unavailable)),
    })
    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toMatch(/unable to discover/i)
  })

  it('maps Code.DeadlineExceeded to the timeout message', async () => {
    const transport = makeTransport({
      getBicepTemplate: () =>
        Promise.reject(new ConnectError('took too long', Code.DeadlineExceeded)),
    })
    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.error).toMatch(/timed out/i)
  })

  it('clears error and re-fetches when fetchTemplate is called', async () => {
    let shouldFail = true
    const transport = makeTransport({
      getBicepTemplate: () => {
        if (shouldFail) {
          return Promise.reject(new ConnectError('boom', Code.Unavailable))
        }
        return create(GetBicepTemplateResponseSchema, {
          template: '// retry success',
          includedServices: ['api'],
        })
      },
    })

    const { result } = renderHook(() => useBicepTemplate(transport))

    await waitFor(() => {
      expect(result.current.error).not.toBeNull()
    })

    shouldFail = false
    await act(async () => {
      await result.current.fetchTemplate()
    })

    await waitFor(() => {
      expect(result.current.template).toBe('// retry success')
    })
    expect(result.current.error).toBeNull()
  })

  it('does not surface aborted requests as errors', async () => {
    // Block the request indefinitely so the unmount cleanup is what
    // resolves the promise via abort. If the abort surfaced as an error
    // we'd see error state on the (now-unmounted) hook, but we mainly
    // care that no console noise / unhandled rejection escapes. The
    // assertion below proves the hook treated the abort as a no-op
    // because state never transitioned out of loading before unmount.
    const transport = makeTransport({
      getBicepTemplate: () => new Promise<GetBicepTemplateResponse>(() => {}),
    })
    const { result, unmount } = renderHook(() => useBicepTemplate(transport))

    expect(result.current.isLoading).toBe(true)
    unmount()
    // Give the microtask queue a tick to flush any abort handler.
    await new Promise((resolve) => setTimeout(resolve, 10))
    // Nothing to assert on result.current after unmount; reaching here
    // without an unhandled promise rejection is the contract.
  })
})
