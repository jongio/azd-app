/**
 * useBicepTemplate - Hook to fetch the unified Bicep template for diagnostic
 * settings via the Connect-RPC BicepService.
 *
 * Migration note: replaced raw `fetch('/api/azure/bicep-template')` with the
 * generated Connect client. The proto response intentionally omits the
 * `instructions` and `parameters` fields (they were static strings on the
 * server side); we re-introduce them as compile-time constants here so the
 * existing modal UI continues to render unchanged. If those strings ever
 * become user-tunable, promote them back into the proto in a new RPC field
 * rather than reviving the parallel REST shape.
 */
import * as React from 'react'
import { Code, ConnectError } from '@connectrpc/connect'
import type { Client, Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { GetBicepTemplateRequestSchema } from '@/gen/proto/azdapp/v1/bicep_pb.js'
import type { BicepService } from '@/gen/proto/azdapp/v1/bicep_pb.js'
import { createBicepClient } from '@/lib/connectClient'

// =============================================================================
// Types
// =============================================================================

/**
 * Parameter definition for the Bicep template
 */
export interface BicepTemplateParameter {
  name: string
  description: string
  example: string
}

/**
 * Integration instructions for the Bicep template
 */
export interface BicepTemplateInstructions {
  summary: string
  steps: string[]
}

/**
 * Hook state. Mirrors the prior REST shape so call sites (BicepTemplateModal)
 * are unaffected by the transport swap.
 */
export interface UseBicepTemplateResult {
  /** Loading state */
  isLoading: boolean
  /** Error message if fetch failed */
  error: string | null
  /** Bicep template code */
  template: string | null
  /** List of services included in template */
  services: string[]
  /** Integration instructions (always populated when template is non-null) */
  instructions: BicepTemplateInstructions | null
  /** Template parameters (always populated when template is non-null) */
  parameters: BicepTemplateParameter[]
  /** Fetch the template (called automatically on mount, can be called manually to retry) */
  fetchTemplate: () => Promise<void>
}

// =============================================================================
// Static instructions and parameters
// =============================================================================
//
// These were returned verbatim by the legacy REST handler from the Go side
// (cli/src/internal/azure/bicep.go::buildInstructions / buildParameters).
// Embedding them client-side keeps the proto narrow and avoids a server
// round-trip for content that never varies. If a future generator emits
// service-specific guidance we can layer additional dynamic fields onto the
// proto response without changing the call sites.

const BICEP_INSTRUCTIONS: BicepTemplateInstructions = {
  summary: 'Add this module to your Bicep infrastructure to enable diagnostic settings',
  steps: [
    'Save this template as infra/modules/diagnostic-settings.bicep in your project',
    'Ensure your main.bicep has a Log Analytics workspace resource or parameter',
    'Add module reference in main.bicep after your service resources',
    'Pass the required parameters (workspace ID and resource names)',
    "Run 'azd up' to deploy the diagnostic settings",
  ],
}

const BICEP_PARAMETERS: BicepTemplateParameter[] = [
  {
    name: 'logAnalyticsWorkspaceId',
    description: 'Resource ID of the Log Analytics Workspace where logs will be sent',
    example:
      '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg/providers/Microsoft.OperationalInsights/workspaces/my-workspace',
  },
]

// =============================================================================
// Error mapping
// =============================================================================

/**
 * Map a ConnectError to a user-facing message. We translate the small set of
 * codes the Bicep handler emits (FailedPrecondition, NotFound, Unavailable,
 * DeadlineExceeded) into the same wording the legacy REST handler produced
 * so users don't see different strings after the migration.
 */
function bicepErrorMessage(err: unknown): string {
  if (err instanceof ConnectError) {
    switch (err.code) {
      case Code.FailedPrecondition:
        return 'Azure credentials are not available. Sign in with `az login` and try again.'
      case Code.NotFound:
        return "No Azure resources found. Deploy your application with 'azd up' first."
      case Code.Unavailable:
        return 'Unable to discover Azure resources. Ensure your environment is deployed.'
      case Code.DeadlineExceeded:
        return 'Request timed out while generating Bicep template. Try again.'
      default:
        return err.rawMessage || 'Failed to generate Bicep template'
    }
  }
  if (err instanceof Error) {
    return err.message
  }
  return 'Failed to fetch Bicep template'
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to fetch the unified Bicep template via Connect-RPC.
 *
 * @param transport optional Connect transport. Production omits this and
 *   uses the singleton transport from `connectClient`. Tests pass an
 *   in-memory `createRouterTransport` so no real HTTP is involved.
 *
 * @example
 * ```tsx
 * const { isLoading, template, services, instructions } = useBicepTemplate()
 *
 * if (isLoading) return <Spinner />
 * if (template) {
 *   return <CodeBlock code={template} language="bicep" />
 * }
 * ```
 */
export function useBicepTemplate(transport?: Transport): UseBicepTemplateResult {
  const [isLoading, setIsLoading] = React.useState(true)
  const [error, setError] = React.useState<string | null>(null)
  const [template, setTemplate] = React.useState<string | null>(null)
  const [services, setServices] = React.useState<string[]>([])
  const [instructions, setInstructions] = React.useState<BicepTemplateInstructions | null>(null)
  const [parameters, setParameters] = React.useState<BicepTemplateParameter[]>([])

  // Memoise the client against the transport identity so a stable transport
  // (the production singleton, or a test's `useMemo`-wrapped router) yields a
  // stable client. Without this, every render rebuilds the client and the
  // fetchTemplate identity churns, defeating downstream React.useCallback
  // memoization.
  const client: Client<typeof BicepService> = React.useMemo(
    () => createBicepClient(transport),
    [transport]
  )

  // AbortController lets us cancel an in-flight request when the component
  // unmounts or the caller fires a fresh fetch before the previous one
  // resolves. Connect honors AbortSignal natively via the second-arg options.
  const abortControllerRef = React.useRef<AbortController | null>(null)

  const fetchTemplate = React.useCallback(async () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    setIsLoading(true)
    setError(null)

    try {
      const response = await client.getBicepTemplate(
        create(GetBicepTemplateRequestSchema, { serviceNames: [] }),
        { signal: controller.signal }
      )

      // The handler returns instructions + parameters as constants client-side
      // (proto omits them, see top-of-file note); we always populate them when
      // the call succeeds so consumers don't have to special-case nulls.
      setTemplate(response.template || null)
      setServices(response.includedServices)
      setInstructions(BICEP_INSTRUCTIONS)
      setParameters(BICEP_PARAMETERS)
      setError(null)
    } catch (err) {
      // Connect raises ConnectError with code Canceled when the signal aborts;
      // treat that the same as the REST hook treated DOMException AbortError.
      if (
        err instanceof ConnectError &&
        (err.code === Code.Canceled || controller.signal.aborted)
      ) {
        return
      }
      if (err instanceof DOMException && err.name === 'AbortError') {
        return
      }

      setError(bicepErrorMessage(err))
      setTemplate(null)
      setServices([])
      setInstructions(null)
      setParameters([])
    } finally {
      setIsLoading(false)
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [client])

  // Initial fetch on mount (and whenever the client identity changes, which
  // in production never happens after first render).
  /* eslint-disable react-hooks/set-state-in-effect -- async fetch; setState happens asynchronously */
  React.useEffect(() => {
    void fetchTemplate()
  }, [fetchTemplate])

  // Cleanup: abort any in-flight request so a fast unmount doesn't leak the
  // pending HTTP call or update state on an unmounted component.
  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])
  /* eslint-enable react-hooks/set-state-in-effect */

  return {
    isLoading,
    error,
    template,
    services,
    instructions,
    parameters,
    fetchTemplate,
  }
}
