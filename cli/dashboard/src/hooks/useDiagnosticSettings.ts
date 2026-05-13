/**
 * useDiagnosticSettings - Hook to check diagnostic settings status for
 * Azure services via the Connect AzureService.
 *
 * Wire migration note: replaces the previous REST GET against
 * `/api/azure/diagnostic-settings/check`. The hook return shape
 * (services map keyed by service name with status/resourceId/etc.) is
 * preserved exactly so consumer components keep compiling. Only the
 * wire moves: the proto `DiagnosticSettingsStatus` enum is mapped
 * back to the legacy lowercase strings ('configured' | 'not-configured'
 * | 'error') here so downstream UI never sees a proto type.
 */
import * as React from 'react'
import { ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { createAzureClient } from '@/lib/connectClient'
import type { AzureService } from '@/gen/proto/azdapp/v1/azure_pb.js'
import {
  CheckDiagnosticSettingsRequestSchema,
  type DiagnosticSettingResult as ProtoDiagnosticSettingResult,
  DiagnosticSettingsStatus as ProtoDiagnosticSettingsStatus,
} from '@/gen/proto/azdapp/v1/azure_pb.js'

// =============================================================================
// Types
// =============================================================================

/**
 * Status of diagnostic settings for a single service.
 */
export type DiagnosticSettingsStatus = 'configured' | 'not-configured' | 'error'

/**
 * Result of checking diagnostic settings for a single service.
 */
export interface ServiceDiagnosticStatus {
  status: DiagnosticSettingsStatus
  resourceId?: string
  diagnosticSettingName?: string
  error?: string
  workspaceId?: string
}

/**
 * Aggregated check result keyed by service name. Kept as an exported
 * type so tests and other hooks can construct fixture data without
 * reaching into the proto layer.
 */
export interface DiagnosticSettingsResponse {
  workspaceId: string
  services: Record<string, ServiceDiagnosticStatus>
}

/**
 * Hook state.
 */
export interface UseDiagnosticSettingsResult {
  /** Loading state: true on initial fetch */
  isLoading: boolean
  /** Refreshing state: true during manual refresh */
  isRefreshing: boolean
  /** Error message if check failed */
  error: string | null
  /** Workspace ID from the response */
  workspaceId: string | null
  /** Map of service name -> diagnostic status */
  services: Record<string, ServiceDiagnosticStatus>
  /** Manually trigger a recheck */
  recheck: () => Promise<void>
  /** All services configured */
  allConfigured: boolean
  /** Number of services configured */
  configuredCount: number
  /** Total number of services */
  totalCount: number
}

export interface UseDiagnosticSettingsOptions {
  /**
   * Override the underlying Connect transport. Production code never
   * passes this; tests inject `createRouterTransport(...)` so the real
   * client code path runs against an in-memory service handler.
   */
  transport?: Transport
}

// =============================================================================
// Proto -> dashboard mappers
// =============================================================================

/**
 * Map proto enum to the legacy lowercase string the dashboard renders.
 * UNSPECIFIED collapses to 'error' rather than silently dropping, so a
 * future proto enum addition surfaces as a visible problem instead of
 * a phantom "not configured".
 */
function statusToString(status: ProtoDiagnosticSettingsStatus): DiagnosticSettingsStatus {
  switch (status) {
    case ProtoDiagnosticSettingsStatus.CONFIGURED:
      return 'configured'
    case ProtoDiagnosticSettingsStatus.NOT_CONFIGURED:
      return 'not-configured'
    case ProtoDiagnosticSettingsStatus.ERROR:
    case ProtoDiagnosticSettingsStatus.UNSPECIFIED:
    default:
      return 'error'
  }
}

function protoToStatus(r: ProtoDiagnosticSettingResult): ServiceDiagnosticStatus {
  // Only include optional fields when populated. The legacy REST handler
  // omitted empty strings and the dashboard UI checks for `truthy` to
  // decide whether to render diagnostic resource links.
  const out: ServiceDiagnosticStatus = {
    status: statusToString(r.status),
  }
  if (r.resourceId) out.resourceId = r.resourceId
  if (r.diagnosticSettingName) out.diagnosticSettingName = r.diagnosticSettingName
  if (r.error) out.error = r.error
  if (r.workspaceId) out.workspaceId = r.workspaceId
  return out
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to check diagnostic settings status for all Azure services.
 *
 * Calls AzureService.CheckDiagnosticSettings and returns aggregated
 * status keyed by service name. Cancellation is plumbed through Connect
 * via `AbortController.signal`, matching the legacy fetch-cancel
 * semantics.
 *
 * @example
 * ```tsx
 * const { isLoading, services, allConfigured, recheck } = useDiagnosticSettings()
 * if (isLoading) return <Spinner />
 * if (allConfigured) return <Success message="All services configured" />
 * ```
 */
export function useDiagnosticSettings(
  options: UseDiagnosticSettingsOptions = {},
): UseDiagnosticSettingsResult {
  const { transport } = options
  const client = React.useMemo<Client<typeof AzureService>>(
    () => createAzureClient(transport),
    [transport],
  )

  const [isLoading, setIsLoading] = React.useState(true)
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [workspaceId, setWorkspaceId] = React.useState<string | null>(null)
  const [services, setServices] = React.useState<Record<string, ServiceDiagnosticStatus>>({})

  // Track the in-flight controller so a manual recheck can pre-empt an
  // initial fetch without leaving stale state behind.
  const abortControllerRef = React.useRef<AbortController | null>(null)

  const fetchDiagnosticSettings = React.useCallback(async (isManualRefresh = false) => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    if (isManualRefresh) {
      setIsRefreshing(true)
    } else {
      setIsLoading(true)
    }

    try {
      const resp = await client.checkDiagnosticSettings(
        create(CheckDiagnosticSettingsRequestSchema),
        { signal: controller.signal },
      )

      // Bail if a newer request superseded us before the response landed.
      if (controller.signal.aborted) return

      const next: Record<string, ServiceDiagnosticStatus> = {}
      for (const [name, result] of Object.entries(resp.services)) {
        next[name] = protoToStatus(result)
      }

      setWorkspaceId(resp.workspaceId || null)
      setServices(next)
      setError(null)
    } catch (err) {
      if (controller.signal.aborted) return
      // ConnectError surfaces a `.rawMessage` plus a status code; the
      // legacy hook just stringified the fetch error so we mirror that
      // surface (consumer renders `error` directly in the alert banner).
      const message =
        err instanceof ConnectError
          ? err.rawMessage || err.message
          : err instanceof Error
          ? err.message
          : 'Failed to check diagnostic settings'
      setError(message)
      setServices({})
      setWorkspaceId(null)
    } finally {
      if (!controller.signal.aborted) {
        setIsLoading(false)
        setIsRefreshing(false)
      }
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [client])

  /* eslint-disable react-hooks/set-state-in-effect -- async fetch; setState happens asynchronously */
  React.useEffect(() => {
    void fetchDiagnosticSettings()
  }, [fetchDiagnosticSettings])

  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])
  /* eslint-enable react-hooks/set-state-in-effect */

  const recheck = React.useCallback(async () => {
    await fetchDiagnosticSettings(true)
  }, [fetchDiagnosticSettings])

  // Derived state - cheap enough to recompute every render so we skip
  // useMemo and avoid the indirection.
  const serviceList = Object.values(services)
  const configuredCount = serviceList.filter((s) => s.status === 'configured').length
  const totalCount = serviceList.length
  const allConfigured = totalCount > 0 && configuredCount === totalCount

  return {
    isLoading,
    isRefreshing,
    error,
    workspaceId,
    services,
    recheck,
    allConfigured,
    configuredCount,
    totalCount,
  }
}
