/**
 * useWorkspaceVerification - Hook to verify Log Analytics workspace
 * connection and per-service log flow via the Connect AzureService.
 *
 * Wire migration note: replaces the legacy POST against
 * `/api/azure/workspace/verify`. The hook return surface is preserved
 * verbatim so the Setup wizard's verification step continues to work
 * unchanged. Only the wire moves: proto enums collapse back to the
 * lowercase string union the dashboard renders, and the legacy
 * `message` field (which lived on the JSON body) is read off the
 * proto Struct `details.message` so per-service tooltip text continues
 * to render.
 */
import * as React from 'react'
import { ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'
import { timestampDate } from '@bufbuild/protobuf/wkt'

import { createAzureClient } from '@/lib/connectClient'
import type { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'
import {
  type ServiceVerificationResult as ProtoServiceVerificationResult,
  ServiceVerificationStatus as ProtoServiceVerificationStatus,
  VerifyWorkspaceRequestSchema,
  WorkspaceVerificationStatus as ProtoWorkspaceVerificationStatus,
} from '@/gen/proto/azdapp/v1/azure_pb.js'

// =============================================================================
// Types (preserved from legacy hook surface)
// =============================================================================

/**
 * Status of verification for a single service. Preserves the legacy
 * union; `DIAGNOSTIC_NOT_CONFIGURED` (a newer proto enum value) maps
 * to 'error' so the existing wizard rendering still applies.
 */
export type ServiceVerificationStatus = 'ok' | 'no-logs' | 'error'

export interface ServiceVerificationResult {
  serviceName: string
  logCount: number
  lastLogTime?: string
  status: ServiceVerificationStatus
  message?: string
  error?: string
}

/**
 * Overall verification status. 'idle' and 'verifying' are client-only
 * (the proto only reports terminal states - SUCCESS / PARTIAL / ERROR).
 */
export type VerificationStatus = 'idle' | 'verifying' | 'success' | 'partial' | 'error'

export interface WorkspaceVerificationResponse {
  status: VerificationStatus
  workspace: {
    id: string
    name: string
  }
  results: Record<string, ServiceVerificationResult>
  guidance: string[]
}

export interface WorkspaceVerificationRequest {
  services: string[]
  timespan?: string
}

export interface UseWorkspaceVerificationResult {
  isVerifying: boolean
  error: string | null
  status: VerificationStatus
  workspace: { id: string; name: string } | null
  results: Record<string, ServiceVerificationResult>
  guidance: string[]
  servicesWithLogs: number
  totalServices: number
  allVerified: boolean
  partiallyVerified: boolean
  verify: (services?: string[]) => Promise<void>
}

export interface UseWorkspaceVerificationOptions {
  /**
   * Override the underlying Connect transport. Production code never
   * passes this; tests inject `createRouterTransport(...)` so the real
   * client code path runs against an in-memory service handler.
   */
  transport?: Transport
}

// =============================================================================
// Mappers
// =============================================================================

function workspaceStatusToString(status: ProtoWorkspaceVerificationStatus): VerificationStatus {
  switch (status) {
    case ProtoWorkspaceVerificationStatus.SUCCESS:
      return 'success'
    case ProtoWorkspaceVerificationStatus.PARTIAL:
      return 'partial'
    case ProtoWorkspaceVerificationStatus.ERROR:
    case ProtoWorkspaceVerificationStatus.UNSPECIFIED:
    default:
      return 'error'
  }
}

function serviceStatusToString(status: ProtoServiceVerificationStatus): ServiceVerificationStatus {
  switch (status) {
    case ProtoServiceVerificationStatus.OK:
      return 'ok'
    case ProtoServiceVerificationStatus.NO_LOGS:
      return 'no-logs'
    case ProtoServiceVerificationStatus.ERROR:
    case ProtoServiceVerificationStatus.DIAGNOSTIC_NOT_CONFIGURED:
    case ProtoServiceVerificationStatus.UNSPECIFIED:
    default:
      // Collapse DIAGNOSTIC_NOT_CONFIGURED into 'error' so the legacy
      // wizard renderer surfaces it as a problem rather than dropping
      // it. The richer detail still lives in `error`/`message`.
      return 'error'
  }
}

/**
 * Pull the legacy `message` string out of the proto Struct.
 *
 * The pre-migration JSON shape was:
 *   { status, serviceName, logCount, lastLogTime, message?, error? }
 *
 * The proto carries `message` inside `details: google.protobuf.Struct`
 * to keep the schema narrow. We toJson() the Struct (which yields a
 * plain JS value) and pluck `message` if present.
 */
function readMessageFromDetails(r: ProtoServiceVerificationResult): string | undefined {
  if (!r.details) return undefined
  const obj = r.details
  if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
    const msg = (obj as Record<string, unknown>).message
    if (typeof msg === 'string' && msg.length > 0) return msg
  }
  return undefined
}

function protoToServiceResult(
  serviceName: string,
  r: ProtoServiceVerificationResult,
): ServiceVerificationResult {
  const out: ServiceVerificationResult = {
    serviceName,
    logCount: r.rowsReturned,
    status: serviceStatusToString(r.status),
  }
  if (r.queriedAt) {
    // Proto Timestamp -> ISO string. toDate() handles seconds+nanos.
    out.lastLogTime = timestampDate(r.queriedAt).toISOString()
  }
  const msg = readMessageFromDetails(r)
  if (msg) out.message = msg
  if (r.error) out.error = r.error
  return out
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to verify workspace connection and log flow from Azure services.
 *
 * Calls AzureService.VerifyWorkspace and queries the workspace for
 * recent logs from each service. Results expose per-service status,
 * workspace info, and free-form remediation guidance.
 *
 * @example
 * ```tsx
 * const { isVerifying, status, results, verify } = useWorkspaceVerification()
 * if (status === 'idle') return <button onClick={() => verify()}>Verify</button>
 * if (isVerifying) return <Spinner />
 * if (status === 'success') return <Success message="All services verified" />
 * ```
 */
export function useWorkspaceVerification(
  options: UseWorkspaceVerificationOptions = {},
): UseWorkspaceVerificationResult {
  const { transport } = options
  const client = React.useMemo<Client<typeof AzureService>>(
    () => createAzureClient(transport),
    [transport],
  )

  const [isVerifying, setIsVerifying] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [status, setStatus] = React.useState<VerificationStatus>('idle')
  const [workspace, setWorkspace] = React.useState<{ id: string; name: string } | null>(null)
  const [results, setResults] = React.useState<Record<string, ServiceVerificationResult>>({})
  const [guidance, setGuidance] = React.useState<string[]>([])

  const abortControllerRef = React.useRef<AbortController | null>(null)

  const verify = React.useCallback(
    async (services?: string[]) => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }

      const controller = new AbortController()
      abortControllerRef.current = controller

      setIsVerifying(true)
      setStatus('verifying')
      setError(null)

      try {
        const resp = await client.verifyWorkspace(
          create(VerifyWorkspaceRequestSchema, {
            services: services ?? [],
            // Default 'PT15M' matches legacy behaviour; empty string
            // triggers server default but explicit is friendlier when
            // grepping HTTP logs for the verifier query.
            timespan: 'PT15M',
          }),
          { signal: controller.signal },
        )

        if (controller.signal.aborted) return

        const nextResults: Record<string, ServiceVerificationResult> = {}
        for (const [name, r] of Object.entries(resp.results)) {
          nextResults[name] = protoToServiceResult(name, r)
        }

        setStatus(workspaceStatusToString(resp.status))
        setWorkspace(
          resp.workspace ? { id: resp.workspace.id, name: resp.workspace.name } : null,
        )
        setResults(nextResults)
        setGuidance(resp.guidance)
        setError(null)
      } catch (err) {
        if (controller.signal.aborted) return
        const message =
          err instanceof ConnectError
            ? err.rawMessage || err.message
            : err instanceof Error
            ? err.message
            : 'Failed to verify workspace'
        setError(message)
        setStatus('error')
        setWorkspace(null)
        setResults({})
        setGuidance([])
      } finally {
        if (!controller.signal.aborted) {
          setIsVerifying(false)
        }
        if (abortControllerRef.current === controller) {
          abortControllerRef.current = null
        }
      }
    },
    [client],
  )

  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])

  // Derived counters - small enough that recomputing every render is
  // cheaper than tracking with useMemo + dependency arrays.
  const resultList = Object.values(results)
  const servicesWithLogs = resultList.filter((r) => r.status === 'ok' && r.logCount > 0).length
  const totalServices = resultList.length
  const allVerified = totalServices > 0 && servicesWithLogs === totalServices
  const partiallyVerified = servicesWithLogs > 0 && servicesWithLogs < totalServices

  return {
    isVerifying,
    error,
    status,
    workspace,
    results,
    guidance,
    servicesWithLogs,
    totalServices,
    allVerified,
    partiallyVerified,
    verify,
  }
}
