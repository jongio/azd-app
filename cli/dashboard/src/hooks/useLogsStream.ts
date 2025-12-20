import { useEffect, useRef, useCallback } from 'react'
import type { LogEntry } from '@/components/LogsPane'
import type { LogMode } from '@/components/ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import { MAX_LOGS_IN_MEMORY } from '@/lib/log-utils'
import { useBackendConnection } from '@/hooks/useBackendConnection'
import { useSharedLogStream } from '@/hooks/useSharedLogStream'

function buildLogsFetchUrl(
  endpoint: string,
  serviceName: string,
  logMode: LogMode,
  timeRange: AzureTimeRange | undefined
): string {
  const params = new URLSearchParams({ service: serviceName, tail: '500' })

  if (logMode === 'azure' && timeRange) {
    params.set('since', timeRange.preset)
  }

  return `${endpoint}?${params.toString()}`
}

function parseLogsPayload(logMode: LogMode, data: unknown): LogEntry[] {
  if (!Array.isArray(data) && (typeof data !== 'object' || data === null)) {
    return []
  }

  if (Array.isArray(data)) {
    return data as LogEntry[]
  }

  if (logMode === 'azure') {
    const azurePayload = data as { logs?: LogEntry[] }
    return azurePayload.logs ?? []
  }

  return []
}

export interface UseLogsStreamParams {
  serviceName: string
  fetchKey: string
  logMode: LogMode
  timeRange: AzureTimeRange
  azureRealtime: boolean
  isPausedRef: { current: boolean }
  setLogs: React.Dispatch<React.SetStateAction<LogEntry[]>>
  setErrorMessage: React.Dispatch<React.SetStateAction<string | null>>
  onFetchSettled?: () => void
}

/**
 * Hook for managing log streaming with simplified architecture.
 * 
 * Flow:
 * 1. Initial fetch: HTTP GET for historical logs (one-time per fetchKey)
 * 2. Live updates: WebSocket streaming (continuous)
 *    - Local: Process stdout/stderr streaming
 *    - Azure: Backend handles polling (Log Analytics) or streaming (Container Apps)
 * 
 * The frontend doesn't distinguish between backend polling vs streaming - 
 * the WebSocket connection abstracts this complexity.
 */
export function useLogsStream(params: UseLogsStreamParams): void {
  const { serviceName, fetchKey, logMode, timeRange, azureRealtime, isPausedRef, setLogs, setErrorMessage, onFetchSettled } = params
  const { connected } = useBackendConnection()
  const currentLogModeRef = useRef<LogMode>(logMode)
  const currentFetchKeyRef = useRef<string>(fetchKey)
  const fetchCountForKeyRef = useRef<number>(0)
  const errorCountRef = useRef<number>(0)
  const lastErrorTimeRef = useRef<number>(0)
  const backoffDelayRef = useRef<number>(1000)
  const lastFetchTimeRef = useRef<number>(0)
  const abortControllerRef = useRef<AbortController | null>(null)

  useEffect(() => {
    // Update current log mode ref
    currentLogModeRef.current = logMode
  }, [logMode])
  
  useEffect(() => {
    // Reset error count and backoff when mode or service changes
    errorCountRef.current = 0
    lastErrorTimeRef.current = 0
    backoffDelayRef.current = 1000
    lastFetchTimeRef.current = 0
    // Cancel any in-flight request when mode/service changes
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
  }, [logMode, serviceName])

  useEffect(() => {
    // Don't attempt to fetch logs if backend is disconnected
    if (!connected) {
      // Only show error on first disconnect
      if (errorCountRef.current === 0) {
        setErrorMessage('Backend connection lost')
      }
      onFetchSettled?.()
      return
    }

    // Reset error count and backoff when connection is restored
    if (connected && errorCountRef.current > 0) {
      errorCountRef.current = 0
      lastErrorTimeRef.current = 0
      backoffDelayRef.current = 1000
      setErrorMessage(null)
    }

    // Reset fetch count if fetchKey has changed
    if (currentFetchKeyRef.current !== fetchKey) {
      currentFetchKeyRef.current = fetchKey
      fetchCountForKeyRef.current = 0
    }

    // Choose endpoint based on mode
    const logsEndpoint = logMode === 'azure' ? '/api/azure/logs' : '/api/logs'
    const capturedLogMode = logMode // Capture for closure
    const capturedFetchKey = fetchKey // Capture for closure

    let cancelled = false
    
    const fetchLogs = async () => {
      // Prevent concurrent fetches - only one in-flight request at a time
      if (abortControllerRef.current) {
        return
      }
      
      // Implement backoff: don't fetch if we're in a backoff period
      const now = Date.now()
      const timeSinceLastFetch = now - lastFetchTimeRef.current
      
      if (errorCountRef.current > 0 && timeSinceLastFetch < backoffDelayRef.current) {
        // Still in backoff period, schedule next fetch
        const remainingBackoff = backoffDelayRef.current - timeSinceLastFetch
        setTimeout(() => {
          if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
            void fetchLogs()
          }
        }, remainingBackoff)
        return
      }
      
      // Create abort controller for this fetch
      const controller = new AbortController()
      abortControllerRef.current = controller
      lastFetchTimeRef.current = now
      
      // Increment fetch count for this key
      fetchCountForKeyRef.current++
      const isFirstFetch = fetchCountForKeyRef.current === 1
      
      try {
        setErrorMessage(null)
        const url = buildLogsFetchUrl(logsEndpoint, serviceName, logMode, timeRange)
        // Add timeout to fail fast when backend is down
        const timeoutId = setTimeout(() => controller.abort(), 5000) // 5s timeout
        
        const res = await fetch(url, { signal: controller.signal })
        clearTimeout(timeoutId)
        
        if (!res.ok) {
          const message = (await res.text()) || `HTTP ${res.status}`
          setErrorMessage(message)
          return
        }

        const data: unknown = await res.json()
        const nextLogs = parseLogsPayload(logMode, data)
        
        // Success - reset error count and backoff
        if (errorCountRef.current > 0) {
          errorCountRef.current = 0
          backoffDelayRef.current = 1000
          setErrorMessage(null)
        }
        
        // Only set logs if we're still in the same mode and fetchKey
        if (currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
          // Always update logs with whatever the API returns
          setLogs(nextLogs)
        }
      } catch (err) {
        // Ignore abort errors from cleanup/mode change
        if (err instanceof Error && err.name === 'AbortError' && cancelled) {
          return
        }
        
        // Handle timeout aborts
        if (err instanceof Error && err.name === 'AbortError') {
          if (!cancelled) {
            errorCountRef.current++
            // Use exponential backoff for timeouts
            backoffDelayRef.current = Math.min(backoffDelayRef.current * 2, 30000)
            
            if (errorCountRef.current === 1) {
              setErrorMessage('Request timed out')
              console.warn(`[LogsPane:${serviceName}] Request timed out, will retry with backoff`)
            }
            
            // Schedule retry with backoff
            setTimeout(() => {
              if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
                void fetchLogs()
              }
            }, backoffDelayRef.current)
          }
          // Don't return early - let finally block execute
        } else {
          const message = err instanceof Error ? err.message : 'Request failed'
          
          // Track consecutive errors to detect backend issues
          const now = Date.now()
          errorCountRef.current++
          
          // Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s
          backoffDelayRef.current = Math.min(backoffDelayRef.current * 2, 30000)
          
          // Log errors occasionally to avoid console spam
          // First error: log immediately with full details
          // Subsequent errors: log only every 30 seconds with count
          if (errorCountRef.current === 1) {
            console.warn(`[LogsPane:${serviceName}] Failed to fetch ${logMode} logs:`, message)
            setErrorMessage(message)
            lastErrorTimeRef.current = now
          } else if (now - lastErrorTimeRef.current > 30000) {
            console.warn(`[LogsPane:${serviceName}] Still failing to fetch ${logMode} logs (${errorCountRef.current} consecutive errors, backoff: ${backoffDelayRef.current}ms)`)
            lastErrorTimeRef.current = now
          }
          // Only update error message on first error to avoid UI flashing
          if (errorCountRef.current === 1) {
            setErrorMessage(message)
          }
          
          // Schedule retry with backoff
          setTimeout(() => {
            if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
              void fetchLogs()
            }
          }, backoffDelayRef.current)
        }
      } finally {
        // Clear abort controller to allow next fetch
        if (abortControllerRef.current === controller) {
          abortControllerRef.current = null
        }
        
        // Always call onFetchSettled for the first fetch (success or error)
        // This ensures loading indicator is hidden even when backend is down
        // Subsequent fetches don't call it to prevent flashing during background polling
        if (!cancelled && isFirstFetch) {
          onFetchSettled?.()
        }
      }
    }

    // Only fetch initial logs via HTTP (one-time per fetchKey)
    // WebSocket handles all subsequent updates (both local and Azure)
    if (fetchCountForKeyRef.current === 0) {
      void fetchLogs()
    } else {
      // Not first fetch - WebSocket is handling updates
      onFetchSettled?.()
    }

    // Cleanup function
    return () => {
      cancelled = true
      // Abort any in-flight fetch to prevent state updates after unmount
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
        abortControllerRef.current = null
      }
    }
  }, [connected, serviceName, fetchKey, isPausedRef, setLogs, setErrorMessage, onFetchSettled, azureRealtime, logMode, timeRange])

  // Use shared WebSocket for both local and Azure realtime logs (multiplexed)
  // This prevents resource exhaustion from multiple connections
  const handleSharedLogEntry = useCallback((entry: LogEntry) => {
    if (isPausedRef.current) return
    // Use functional update to avoid dependency on setLogs
    setLogs((prev) => {
      const updated = [...prev, entry]
      // Trim to max size
      return updated.length > MAX_LOGS_IN_MEMORY 
        ? updated.slice(updated.length - MAX_LOGS_IN_MEMORY) 
        : updated
    })
  }, [isPausedRef, setLogs])

  const shouldUseSharedStream = 
    connected && 
    (logMode === 'local' || (logMode === 'azure' && azureRealtime))

  useSharedLogStream({
    serviceName,
    enabled: shouldUseSharedStream,
    mode: logMode === 'azure' ? 'azure' : 'local',
    onLogEntry: handleSharedLogEntry,
  })
}
