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
  lastClearTimeRef: { current: number }
  setLogs: React.Dispatch<React.SetStateAction<LogEntry[]>>
  setErrorMessage: React.Dispatch<React.SetStateAction<string | null>>
  onFetchSettled?: () => void
  setIsLoading?: React.Dispatch<React.SetStateAction<boolean>>
  setLoadingMessage?: React.Dispatch<React.SetStateAction<string>>
  setCanRetry?: React.Dispatch<React.SetStateAction<boolean>>
  onRetry?: () => void
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
export function useLogsStream(params: UseLogsStreamParams): { retry: () => void } {
  const { 
    serviceName, fetchKey, logMode, timeRange, azureRealtime, isPausedRef, lastClearTimeRef,
    setLogs, setErrorMessage, onFetchSettled,
    setIsLoading, setLoadingMessage, setCanRetry, onRetry
  } = params
  const { connected } = useBackendConnection()
  const currentLogModeRef = useRef<LogMode>(logMode)
  const currentFetchKeyRef = useRef<string>(fetchKey)
  const fetchCountForKeyRef = useRef<number>(0)
  const errorCountRef = useRef<number>(0)
  const lastErrorTimeRef = useRef<number>(0)
  const backoffDelayRef = useRef<number>(1000)
  const notFoundCountRef = useRef<number>(0) // Track 404 responses for services not started yet
  const maxNotFoundRetries = 3 // Max retries for 404s (3s, 6s, 12s = ~21s total wait)
  const retryTriggerRef = useRef<number>(0) // Trigger for manual retry
  const lastFetchTimeRef = useRef<number>(0)
  const abortControllerRef = useRef<AbortController | null>(null)
  const emptyResultCountRef = useRef<number>(0)
  const maxEmptyRetries = 2 // Retry up to 2 times when getting empty results (500ms, 1s)

  useEffect(() => {
    // Update current log mode ref
    currentLogModeRef.current = logMode
  }, [logMode])
  
  useEffect(() => {
    // Reset error count and backoff when mode or service changes
    errorCountRef.current = 0
    lastErrorTimeRef.current = 0
    notFoundCountRef.current = 0 // Reset 404 counter
    backoffDelayRef.current = 1000
    lastFetchTimeRef.current = 0
    emptyResultCountRef.current = 0 // Reset empty result counter
    // Cancel any in-flight request when mode/service changes
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
      abortControllerRef.current = null
    }
  }, [logMode, serviceName])

  useEffect(() => {
    // Choose endpoint based on mode
    const logsEndpoint = logMode === 'azure' ? '/api/azure/logs' : '/api/logs'
    const capturedLogMode = logMode // Capture for closure
    const capturedFetchKey = fetchKey // Capture for closure
    
    // Reset fetch count if fetchKey has changed
    // IMPORTANT: Do this check inside the effect to ensure proper cleanup
    const isNewFetchKey = currentFetchKeyRef.current !== fetchKey
    if (isNewFetchKey) {
      currentFetchKeyRef.current = fetchKey
      fetchCountForKeyRef.current = 0
      emptyResultCountRef.current = 0 // Reset empty result counter on key change
      // Also clear any previous error when switching modes/filters
      setErrorMessage(null)
      errorCountRef.current = 0
      backoffDelayRef.current = 1000
    }

    let cancelled = false
    
    const fetchLogs = async () => {
      // Skip Azure logs fetch if backend not connected - avoids failed requests on dashboard load
      if (logMode === 'azure' && !connected) {
        return
      }
      
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
      
      // Show loading state on first fetch
      if (isFirstFetch) {
        setIsLoading?.(true)
        setLoadingMessage?.(notFoundCountRef.current > 0 
          ? `Waiting for ${serviceName} to start... (attempt ${notFoundCountRef.current + 1})`
          : `Loading logs for ${serviceName}...`
        )
      }
      
      try {
        setErrorMessage(null)
        const url = buildLogsFetchUrl(logsEndpoint, serviceName, logMode, timeRange)
        // Add timeout - longer for Azure logs which can be slower
        const timeoutMs = logMode === 'azure' ? 30000 : 10000 // 30s for Azure, 10s for local
        const timeoutId = setTimeout(() => controller.abort(), timeoutMs)
        
        const res = await fetch(url, { signal: controller.signal })
        clearTimeout(timeoutId)
        
        if (!res.ok) {
          // Special handling for 404 - service might not be started yet (local-only services)
          if (res.status === 404 && logMode === 'local') {
            notFoundCountRef.current++
            
            // Give up after max retries - service is likely Azure-only and won't have local logs
            if (notFoundCountRef.current >= maxNotFoundRetries) {
              // Mark that we gave up - user can manually retry
              setCanRetry?.(true)
              setIsLoading?.(false)
              setLoadingMessage?.('')
              return
            }
            
            // For first few retries, use shorter delays - service might still be starting
            // 1s, 2s, 4s pattern instead of 3s, 6s, 12s
            const notFoundDelay = 1000 * Math.pow(2, notFoundCountRef.current - 1)
            setLoadingMessage?.(`Waiting for ${serviceName}... (${notFoundDelay / 1000}s)`)
            
            setTimeout(() => {
              if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
                void fetchLogs()
              }
            }, notFoundDelay)
            return
          }
          
          const message = (await res.text()) || `HTTP ${res.status}`
          setErrorMessage(message)
          setIsLoading?.(false)
          return
        }

        const data: unknown = await res.json()
        const nextLogs = parseLogsPayload(logMode, data)
        
        // Success - clear retry state
        setCanRetry?.(false)
        setIsLoading?.(false)
        setLoadingMessage?.('')
        
        // Success - reset error count and backoff
        if (errorCountRef.current > 0) {
          errorCountRef.current = 0
          backoffDelayRef.current = 1000
          setErrorMessage(null)
        }
        
        // Only set logs if we're still in the same mode and fetchKey
        if (currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
          // Check if we should retry due to empty results
          const shouldRetryEmpty = nextLogs.length === 0 && emptyResultCountRef.current < maxEmptyRetries && isFirstFetch
          
          // Always update logs with whatever the API returns
          setLogs(nextLogs)
          
          // If we got empty results and this is one of the first fetches, retry with faster interval
          // This handles the case where the service just started and logs aren't available yet
          if (shouldRetryEmpty) {
            emptyResultCountRef.current++
            const retryDelay = 500 * emptyResultCountRef.current // 500ms, 1s, 1.5s
            
            setTimeout(() => {
              if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
                // Reset fetch count to retry
                fetchCountForKeyRef.current = 0
                void fetchLogs()
              }
            }, retryDelay)
            
            // Don't call onFetchSettled yet - we're going to retry
            // This keeps the loading indicator visible
            return
          } else if (nextLogs.length > 0) {
            // Got data - reset empty result counter
            emptyResultCountRef.current = 0
          }
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
        
        // Call onFetchSettled for the first fetch (success or error)
        // But NOT if we returned early due to retry
        // This ensures loading indicator is hidden even when backend is down
        // Subsequent fetches don't call it to prevent flashing during background polling
        if (!cancelled && isFirstFetch) {
          onFetchSettled?.()
        }
      }
    }

    // Only fetch initial logs via HTTP (one-time per fetchKey)
    // WebSocket handles all subsequent updates (both local and Azure)
    // IMPORTANT: Only skip fetch if we're on the same key AND have already fetched
    // Don't call onFetchSettled for a new key until we've actually started fetching
    const shouldFetch = isNewFetchKey || fetchCountForKeyRef.current === 0
    if (shouldFetch) {
      void fetchLogs()
    } else {
      // Not first fetch for this key - WebSocket is handling updates
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
  }, [serviceName, fetchKey, isPausedRef, setLogs, setErrorMessage, onFetchSettled, azureRealtime, logMode, timeRange, connected, setIsLoading, setLoadingMessage, setCanRetry])

  // Manual retry function - resets counters and triggers re-fetch
  const retry = useCallback(() => {
    notFoundCountRef.current = 0
    fetchCountForKeyRef.current = 0
    setCanRetry?.(false)
    setLoadingMessage?.('')
    // Trigger re-fetch by incrementing retryTriggerRef
    retryTriggerRef.current++
    onRetry?.()
  }, [setCanRetry, setLoadingMessage, onRetry])

  // Use shared WebSocket for both local and Azure realtime logs (multiplexed)
  // This prevents resource exhaustion from multiple connections
  const handleSharedLogEntry = useCallback((entry: LogEntry) => {
    if (isPausedRef.current) return
    // Ignore messages received within 100ms of a clear operation
    // This prevents race conditions where in-flight messages appear after clear
    if (Date.now() - lastClearTimeRef.current < 500) {
      return
    }
    // Use functional update to avoid dependency on setLogs
    setLogs((prev) => {
      const updated = [...prev, entry]
      // Trim to max size
      return updated.length > MAX_LOGS_IN_MEMORY 
        ? updated.slice(updated.length - MAX_LOGS_IN_MEMORY) 
        : updated
    })
  }, [isPausedRef, lastClearTimeRef, setLogs])

  const shouldUseSharedStream = 
    connected && 
    (logMode === 'local' || (logMode === 'azure' && azureRealtime))

  useSharedLogStream({
    serviceName,
    enabled: shouldUseSharedStream,
    mode: logMode === 'azure' ? 'azure' : 'local',
    onLogEntry: handleSharedLogEntry,
  })
  
  return { retry }
}
