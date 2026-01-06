/**
 * useWorkspaceVerification - Hook to verify workspace connection and log flow
 */
import * as React from 'react'

// =============================================================================
// Types
// =============================================================================

/**
 * Status of verification for a single service
 */
export type ServiceVerificationStatus = 'ok' | 'no-logs' | 'error'

/**
 * Result of verifying logs for a single service
 */
export interface ServiceVerificationResult {
  serviceName: string
  logCount: number
  lastLogTime?: string
  status: ServiceVerificationStatus
  message?: string
  error?: string
}

/**
 * Overall verification status
 */
export type VerificationStatus = 'idle' | 'verifying' | 'success' | 'partial' | 'error'

/**
 * API response from /api/azure/workspace/verify
 */
export interface WorkspaceVerificationResponse {
  status: VerificationStatus
  workspace: {
    id: string
    name: string
  }
  results: Record<string, ServiceVerificationResult>
  guidance: string[]
}

/**
 * Request payload for workspace verification
 */
export interface WorkspaceVerificationRequest {
  services: string[]
  timespan?: string // Default: "PT15M" (last 15 minutes)
}

/**
 * Hook state
 */
export interface UseWorkspaceVerificationResult {
  /** Loading state: true during initial or manual verification */
  isVerifying: boolean
  /** Error message if verification failed */
  error: string | null
  /** Overall verification status */
  status: VerificationStatus
  /** Workspace info from response */
  workspace: { id: string; name: string } | null
  /** Map of service name -> verification result */
  results: Record<string, ServiceVerificationResult>
  /** Guidance messages from API */
  guidance: string[]
  /** Number of services with logs */
  servicesWithLogs: number
  /** Total number of services checked */
  totalServices: number
  /** All services have logs */
  allVerified: boolean
  /** At least one service has logs */
  partiallyVerified: boolean
  /** Start verification (optionally with specific services) */
  verify: (services?: string[]) => Promise<void>
}

// =============================================================================
// Hook
// =============================================================================

/**
 * Hook to verify workspace connection and log flow from Azure services.
 * 
 * Makes an API call to /api/azure/workspace/verify to query the workspace
 * for recent logs from each service and provides detailed verification results.
 * 
 * @example
 * ```tsx
 * const { isVerifying, status, results, verify } = useWorkspaceVerification()
 * 
 * if (status === 'idle') {
 *   return <button onClick={() => verify()}>Start Verification</button>
 * }
 * 
 * if (isVerifying) return <Spinner />
 * 
 * if (status === 'success') {
 *   return <Success message="All services verified" />
 * }
 * ```
 */
export function useWorkspaceVerification(): UseWorkspaceVerificationResult {
  const [isVerifying, setIsVerifying] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [status, setStatus] = React.useState<VerificationStatus>('idle')
  const [workspace, setWorkspace] = React.useState<{ id: string; name: string } | null>(null)
  const [results, setResults] = React.useState<Record<string, ServiceVerificationResult>>({})
  const [guidance, setGuidance] = React.useState<string[]>([])

  // Abort controller for cleanup
  const abortControllerRef = React.useRef<AbortController | null>(null)

  // Verify workspace connection
  const verify = React.useCallback(async (services?: string[]) => {
    // Cancel any in-flight request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    setIsVerifying(true)
    setStatus('verifying')
    setError(null)

    try {
      const requestBody: WorkspaceVerificationRequest = {
        services: services || [], // Empty array means verify all services
        timespan: 'PT15M', // Last 15 minutes
      }

      const response = await fetch('/api/azure/workspace/verify', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
        signal: controller.signal,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`API returned ${response.status}: ${errorText || response.statusText}`)
      }

      const data = (await response.json()) as WorkspaceVerificationResponse

      setStatus(data.status)
      setWorkspace(data.workspace || null)
      setResults(data.results || {})
      setGuidance(data.guidance || [])
      setError(null)
    } catch (err) {
      // Ignore abort errors
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }

      const errorMessage = err instanceof Error ? err.message : 'Failed to verify workspace'
      setError(errorMessage)
      setStatus('error')
      setWorkspace(null)
      setResults({})
      setGuidance([])
    } finally {
      setIsVerifying(false)

      // Clear the abort controller if this is still our request
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [])

  // Cleanup on unmount
  React.useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [])

  // Calculate derived state
  const resultList = Object.values(results)
  const servicesWithLogs = resultList.filter(r => r.status === 'ok' && r.logCount > 0).length
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
