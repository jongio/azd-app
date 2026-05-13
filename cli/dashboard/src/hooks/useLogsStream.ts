import { useEffect, useMemo, useRef, useCallback } from 'react'
import { ConnectError, Code, type Transport } from '@connectrpc/connect'
import { protoInt64 } from '@bufbuild/protobuf'

import type { LogEntry } from '@/components/LogsPane'
import type { LogMode } from '@/components/ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import { MAX_LOGS_IN_MEMORY } from '@/lib/log-utils'
import { useBackendConnection } from '@/hooks/useBackendConnection'
import { useSharedLogStream } from '@/hooks/useSharedLogStream'
import { createAzureClient, createLogsClient } from '@/lib/connectClient'
import { GetAzureLogsRequest } from '@/gen/proto/azdapp/v1/azure_pb.js'
import { GetLogsRequest } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { protoLogEntryToView } from '@/lib/log-proto'

/** Default tail passed to GetLogs/GetAzureLogs on the initial historical fetch. */
const INITIAL_FETCH_TAIL = 500

/**
 * Map a dashboard Azure time-range preset to the proto
 * `since_seconds` field. Matches the Go handler's duration parsing so
 * the wire value is equivalent to the legacy `?since=1h` REST query.
 */
function azureSinceSeconds(preset: AzureTimeRange['preset'] | undefined): number {
  switch (preset) {
    case '15m': return 15 * 60
    case '30m': return 30 * 60
    case '6h': return 6 * 60 * 60
    case '24h': return 24 * 60 * 60
    default: return 30 * 60
  }
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
  /**
   * Optional Connect transport override for tests. Production code
   * never passes this; specs inject an in-memory router transport via
   * `createRouterTransport`. Threaded through to both Logs and Azure
   * clients AND the shared-log-stream hook so every RPC in this path
   * hits the same in-memory backend.
   */
  transport?: Transport
}

/**
 * Hook for managing log streaming with simplified architecture.
 *
 * Flow:
 * 1. Initial fetch: Connect-RPC unary call for historical logs (one-
 *    time per fetchKey). `LogsService.GetLogs` for local mode,
 *    `AzureService.GetAzureLogs` for Azure. Replaces the legacy
 *    `GET /api/logs` and `GET /api/azure/logs` endpoints.
 * 2. Live updates: `useSharedLogStream` drives the server-streaming
 *    RPCs (StreamLocalLogs / StreamAzureLogs), which replaced the
 *    `/api/logs/stream` and `/api/azure/logs/stream` WebSockets.
 */
export function useLogsStream(
  params: UseLogsStreamParams,
): { retry: () => void; droppedCount: number } {
  const {
    serviceName, fetchKey, logMode, timeRange, azureRealtime, isPausedRef, lastClearTimeRef,
    setLogs, setErrorMessage, onFetchSettled,
    setIsLoading, setLoadingMessage, setCanRetry, onRetry,
    transport,
  } = params
  const { connected } = useBackendConnection()
  const currentLogModeRef = useRef<LogMode>(logMode)
  const currentFetchKeyRef = useRef<string>(fetchKey)
  const fetchCountForKeyRef = useRef<number>(0)
  const errorCountRef = useRef<number>(0)
  const lastErrorTimeRef = useRef<number>(0)
  const backoffDelayRef = useRef<number>(1000)
  const notFoundCountRef = useRef<number>(0) // Track NotFound responses for services not started yet
  const maxNotFoundRetries = 3 // Max retries for 404s (3s, 6s, 12s = ~21s total wait)
  const retryTriggerRef = useRef<number>(0) // Trigger for manual retry
  const lastFetchTimeRef = useRef<number>(0)
  const abortControllerRef = useRef<AbortController | null>(null)
  const emptyResultCountRef = useRef<number>(0)
  const maxEmptyRetries = 2 // Retry up to 2 times when getting empty results (500ms, 1s)

  // Memoise the per-transport Connect clients so re-renders don't
  // churn them. Default-arg path resolves to the singleton transport
  // in connectClient.ts.
  const logsClient = useMemo(() => createLogsClient(transport), [transport])
  const azureClient = useMemo(() => createAzureClient(transport), [transport])

  useEffect(() => {
    // Update current log mode ref
    currentLogModeRef.current = logMode
  }, [logMode])
  
  useEffect(() => {
    // Reset error count and backoff when mode or service changes
    errorCountRef.current = 0
    lastErrorTimeRef.current = 0
    notFoundCountRef.current = 0 // Reset NotFound counter
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
        // Add timeout - longer for Azure logs which can be slower.
        // The Connect client honours `AbortController.signal`, so a
        // `controller.abort()` propagates as `ConnectError` with
        // `Code.Canceled`.
        const timeoutMs = logMode === 'azure' ? 30000 : 10000
        const timeoutId = setTimeout(() => controller.abort(), timeoutMs)

        let nextLogs: LogEntry[]
        if (logMode === 'azure') {
          const req = new GetAzureLogsRequest({
            // Empty string means "all services merged" - matches the
            // legacy REST contract and the Go handler's fallback.
            service: serviceName === 'all' ? '' : serviceName,
            sinceSeconds: protoInt64.parse(azureSinceSeconds(timeRange?.preset)),
            tail: INITIAL_FETCH_TAIL,
          })
          const resp = await azureClient.getAzureLogs(req, { signal: controller.signal })
          nextLogs = resp.entries.map(protoLogEntryToView)
        } else {
          const req = new GetLogsRequest({
            serviceName: serviceName === 'all' ? '' : serviceName,
            tail: INITIAL_FETCH_TAIL,
          })
          const resp = await logsClient.getLogs(req, { signal: controller.signal })
          nextLogs = resp.entries.map(protoLogEntryToView)
        }
        clearTimeout(timeoutId)

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
        // `ConnectError` replaces the legacy fetch error surface.
        // Map each Connect code onto the original REST handling:
        //   - Canceled  -> abort (cleanup or timeout)
        //   - NotFound  -> 404 retry path (local-only services)
        //   - other     -> generic error path with exponential backoff
        const connectErr = err instanceof ConnectError ? err : null

        // Ignore aborts triggered by cleanup / mode change.
        if (connectErr?.code === Code.Canceled && cancelled) {
          return
        }
        if (err instanceof Error && err.name === 'AbortError' && cancelled) {
          return
        }

        // Special handling for NotFound - local service might not be
        // started yet. Matches the legacy 404 retry semantics.
        if (connectErr?.code === Code.NotFound && logMode === 'local') {
          notFoundCountRef.current++

          // Give up after max retries - service is likely Azure-only
          // and won't have local logs. User can manually retry.
          if (notFoundCountRef.current >= maxNotFoundRetries) {
            setCanRetry?.(true)
            setIsLoading?.(false)
            setLoadingMessage?.('')
            return
          }

          // Shorter delays for the first few retries: 1s, 2s, 4s.
          const notFoundDelay = 1000 * Math.pow(2, notFoundCountRef.current - 1)
          setLoadingMessage?.(`Waiting for ${serviceName}... (${notFoundDelay / 1000}s)`)

          setTimeout(() => {
            if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
              void fetchLogs()
            }
          }, notFoundDelay)
          return
        }

        // Handle timeout-triggered aborts (controller.abort from the
        // timeout setTimeout). These surface as Canceled when
        // `cancelled` is false (i.e. we triggered it ourselves).
        const isTimeout = connectErr?.code === Code.Canceled
          || (err instanceof Error && err.name === 'AbortError')
        if (isTimeout) {
          if (!cancelled) {
            errorCountRef.current++
            // Exponential backoff for timeouts, capped at 30s.
            backoffDelayRef.current = Math.min(backoffDelayRef.current * 2, 30000)

            if (errorCountRef.current === 1) {
              setErrorMessage('Request timed out')
              console.warn(`[LogsPane:${serviceName}] Request timed out, will retry with backoff`)
            }

            // Schedule retry with backoff.
            setTimeout(() => {
              if (!cancelled && currentLogModeRef.current === capturedLogMode && currentFetchKeyRef.current === capturedFetchKey) {
                void fetchLogs()
              }
            }, backoffDelayRef.current)
          }
        } else {
          const message = connectErr?.rawMessage
            ?? (err instanceof Error ? err.message : 'Request failed')

          // Track consecutive errors to detect backend issues.
          const errNow = Date.now()
          errorCountRef.current++

          // Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s.
          backoffDelayRef.current = Math.min(backoffDelayRef.current * 2, 30000)

          // Log errors occasionally to avoid console spam.
          if (errorCountRef.current === 1) {
            console.warn(`[LogsPane:${serviceName}] Failed to fetch ${logMode} logs:`, message)
            setErrorMessage(message)
            lastErrorTimeRef.current = errNow
          } else if (errNow - lastErrorTimeRef.current > 30000) {
            console.warn(`[LogsPane:${serviceName}] Still failing to fetch ${logMode} logs (${errorCountRef.current} consecutive errors, backoff: ${backoffDelayRef.current}ms)`)
            lastErrorTimeRef.current = errNow
          }

          // Schedule retry with backoff.
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
        // but NOT if we returned early due to retry. Subsequent
        // fetches don't call it to prevent flashing during background
        // polling.
        if (!cancelled && isFirstFetch) {
          onFetchSettled?.()
        }
      }
    }

    // Only fetch initial logs via unary RPC (one-time per fetchKey).
    // Subsequent updates come from `useSharedLogStream` below.
    const shouldFetch = isNewFetchKey || fetchCountForKeyRef.current === 0
    if (shouldFetch) {
      void fetchLogs()
    } else {
      // Not first fetch for this key - shared stream is handling updates.
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
  }, [serviceName, fetchKey, isPausedRef, setLogs, setErrorMessage, onFetchSettled, azureRealtime, logMode, timeRange, connected, setIsLoading, setLoadingMessage, setCanRetry, logsClient, azureClient])

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

  const { droppedCount } = useSharedLogStream({
    serviceName,
    enabled: shouldUseSharedStream,
    mode: logMode === 'azure' ? 'azure' : 'local',
    onLogEntry: handleSharedLogEntry,
    transport,
  })

  return { retry, droppedCount }
}
