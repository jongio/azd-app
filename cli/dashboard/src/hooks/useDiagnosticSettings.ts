/**
 * useDiagnosticSettings - Hook to check diagnostic settings status for Azure services
 */
import * as React from 'react'

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
 * API response from /api/azure/diagnostic-settings/check
 */
export interface DiagnosticSettingsResponse {
  workspaceId: string
  services: Record<string, ServiceDiagnosticStatus>
}

/**
 * Hook state
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

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to check diagnostic settings status for all Azure services.
 * 
 * Makes a single API call to /api/azure/diagnostic-settings/check to get
 * aggregated status for all services.
 * 
 * @example
 * ```tsx
 * const { isLoading, services, allConfigured, recheck } = useDiagnosticSettings()
 * 
 * if (isLoading) return <Spinner />
 * 
 * if (allConfigured) {
 *   return <Success message="All services configured" />
 * }
 * ```
 */
export function useDiagnosticSettings(): UseDiagnosticSettingsResult {
  const [isLoading, setIsLoading] = React.useState(true)
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [workspaceId, setWorkspaceId] = React.useState<string | null>(null)
  const [services, setServices] = React.useState<Record<string, ServiceDiagnosticStatus>>({})

  // Abort controller for cleanup
  const abortControllerRef = React.useRef<AbortController | null>(null)

  // Fetch diagnostic settings status
  const fetchDiagnosticSettings = React.useCallback(async (isManualRefresh = false) => {
    // Cancel any in-flight request
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
      const response = await fetch('/api/azure/diagnostic-settings/check', {
        signal: controller.signal,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`API returned ${response.status}: ${errorText || response.statusText}`)
      }

      const data = (await response.json()) as DiagnosticSettingsResponse
      
      setWorkspaceId(data.workspaceId || null)
      setServices(data.services || {})
      setError(null)
    } catch (err) {
      // Ignore abort errors
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }

      const errorMessage = err instanceof Error ? err.message : 'Failed to check diagnostic settings'
      setError(errorMessage)
      setServices({})
      setWorkspaceId(null)
    } finally {
      setIsLoading(false)
      setIsRefreshing(false)
      
      // Clear the abort controller if this is still our request
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [])

  // Initial fetch on mount
  React.useEffect(() => {
    void fetchDiagnosticSettings()
  }, [fetchDiagnosticSettings])

  // Cleanup on unmount
  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])

  // Manual recheck function
  const recheck = React.useCallback(async () => {
    await fetchDiagnosticSettings(true)
  }, [fetchDiagnosticSettings])

  // Calculate derived state
  const serviceList = Object.values(services)
  const configuredCount = serviceList.filter(s => s.status === 'configured').length
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
