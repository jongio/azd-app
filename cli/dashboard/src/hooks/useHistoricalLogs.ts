/**
 * useHistoricalLogs - Hook for querying historical Azure logs
 * Handles pagination, time range conversion, and API communication.
 */
import { useState, useCallback, useRef } from 'react'
import type { ParsedAzureError } from '@/types'
import { createParsedAzureError } from '@/lib/azure-errors'
import { useBackendConnection } from '@/hooks/useBackendConnection'

// =============================================================================
// Types
// =============================================================================

export type TimeRangePreset = '15m' | '30m' | '6h' | '24h' | 'custom'

export interface TimeRange {
  preset: TimeRangePreset
  start?: Date
  end?: Date
}

export interface HistoricalLogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
  /** Azure-specific fields */
  resourceId?: string
  operationName?: string
}

export interface HistoricalLogQuery {
  serviceName: string
  timeRange: TimeRange
  customKql?: string
  limit: number
  offset: number
}

export interface HistoricalLogResult {
  logs: HistoricalLogEntry[]
  total: number
  hasMore: boolean
  executionTime: number
}

export interface UseHistoricalLogsOptions {
  /** Service name to query logs for */
  serviceName: string
  /** Number of logs per page */
  pageSize?: number
}

export interface UseHistoricalLogsReturn {
  /** Fetched log entries */
  logs: HistoricalLogEntry[]
  /** Total count of matching logs */
  total: number
  /** Whether more logs are available */
  hasMore: boolean
  /** Whether a query is in progress */
  isLoading: boolean
  /** Error message if query failed */
  error: string | null
  /** Parsed Azure error with type and metadata */
  azureError: ParsedAzureError | null
  /** Last query execution time in ms */
  executionTime: number | null
  /** Execute a log query */
  executeQuery: (timeRange: TimeRange, customKql?: string) => Promise<void>
  /** Load more logs (pagination) */
  loadMore: () => Promise<void>
  /** Clear all results */
  clearResults: () => void
  /** Reset to default query (clears custom KQL) */
  resetQuery: () => void
  /** Current offset for pagination */
  offset: number
}

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_PAGE_SIZE = 100

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Converts a TimeRange to ISO 8601 duration string for the API.
 * Custom ranges calculate the duration between start and end.
 */
export function timeRangeToTimespan(timeRange: TimeRange): string {
  if (timeRange.preset !== 'custom') {
    switch (timeRange.preset) {
      case '15m':
        return 'PT15M'
      case '30m':
        return 'PT30M'
      case '6h':
        return 'PT6H'
      case '24h':
        return 'PT24H'
      default:
        return 'PT30M'
    }
  }

  // For custom range, calculate duration
  if (timeRange.start && timeRange.end) {
    const durationMs = timeRange.end.getTime() - timeRange.start.getTime()
    const durationMinutes = Math.ceil(durationMs / 60000)
    
    if (durationMinutes < 60) {
      return `PT${durationMinutes}M`
    } else if (durationMinutes < 1440) {
      const hours = Math.ceil(durationMinutes / 60)
      return `PT${hours}H`
    } else {
      const days = Math.ceil(durationMinutes / 1440)
      return `P${days}D`
    }
  }

  return 'PT30M' // Default fallback
}

/**
 * Formats a time range for display in the results header.
 */
export function formatTimeRangeDisplay(timeRange: TimeRange): string {
  if (timeRange.preset !== 'custom') {
    switch (timeRange.preset) {
      case '15m':
        return 'last 15 minutes'
      case '30m':
        return 'last 30 minutes'
      case '6h':
        return 'last 6 hours'
      case '24h':
        return 'last 24 hours'
      default:
        return timeRange.preset
    }
  }

  if (timeRange.start && timeRange.end) {
    const formatDate = (d: Date) => d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
    return `${formatDate(timeRange.start)} - ${formatDate(timeRange.end)}`
  }

  return 'custom range'
}

// =============================================================================
// Hook Implementation
// =============================================================================

export function useHistoricalLogs({
  serviceName,
  pageSize = DEFAULT_PAGE_SIZE,
}: UseHistoricalLogsOptions): UseHistoricalLogsReturn {
  const { connected } = useBackendConnection()
  const [logs, setLogs] = useState<HistoricalLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [azureError, setAzureError] = useState<ParsedAzureError | null>(null)
  const [executionTime, setExecutionTime] = useState<number | null>(null)
  const [offset, setOffset] = useState(0)

  // Store current query params for pagination and reset
  const currentQueryRef = useRef<{
    timeRange: TimeRange
    customKql?: string
  } | null>(null)

  const executeQuery = useCallback(async (timeRange: TimeRange, customKql?: string) => {
    if (!connected) {
      setError('Backend connection lost')
      return
    }

    setIsLoading(true)
    setError(null)
    setAzureError(null)
    setOffset(0)

    // Store query params for pagination
    currentQueryRef.current = { timeRange, customKql }

    let response: Response | undefined

    try {
      response = await fetch('/api/azure/logs/query', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          service: serviceName,
          timespan: timeRangeToTimespan(timeRange),
          query: customKql || undefined,
          limit: pageSize,
          offset: 0,
        }),
      })

      if (!response.ok) {
        const errorText = await response.text()
        const message = errorText || `Query failed with status ${response.status}`
        const parsedError = createParsedAzureError(message, response)
        setError(message)
        setAzureError(parsedError)
        setLogs([])
        setTotal(0)
        setHasMore(false)
        return
      }

      const result = await response.json() as HistoricalLogResult
      
      setLogs(result.logs ?? [])
      setTotal(result.total ?? 0)
      setHasMore(result.hasMore ?? false)
      setExecutionTime(result.executionTime ?? null)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Query failed'
      const parsedError = createParsedAzureError(message, response)
      setError(message)
      setAzureError(parsedError)
      setLogs([])
      setTotal(0)
      setHasMore(false)
    } finally {
      setIsLoading(false)
    }
  }, [connected, serviceName, pageSize])

  const loadMore = useCallback(async () => {
    if (!connected || !currentQueryRef.current || isLoading || !hasMore) {
      return
    }

    const { timeRange, customKql } = currentQueryRef.current
    const newOffset = offset + pageSize

    setIsLoading(true)
    setError(null)
    setAzureError(null)

    let response: Response | undefined

    try {
      response = await fetch('/api/azure/logs/query', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          service: serviceName,
          timespan: timeRangeToTimespan(timeRange),
          query: customKql || undefined,
          limit: pageSize,
          offset: newOffset,
        }),
      })

      if (!response.ok) {
        const errorText = await response.text()
        const message = errorText || `Query failed with status ${response.status}`
        const parsedError = createParsedAzureError(message, response)
        setError(message)
        setAzureError(parsedError)
        return
      }

      const result = await response.json() as HistoricalLogResult

      setLogs(prev => [...prev, ...(result.logs ?? [])])
      setTotal(result.total ?? total)
      setHasMore(result.hasMore ?? false)
      setOffset(newOffset)
      setExecutionTime(result.executionTime ?? null)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load more logs'
      const parsedError = createParsedAzureError(message, response)
      setError(message)
      setAzureError(parsedError)
    } finally {
      setIsLoading(false)
    }
  }, [connected, serviceName, pageSize, offset, isLoading, hasMore, total])

  const clearResults = useCallback(() => {
    setLogs([])
    setTotal(0)
    setHasMore(false)
    setError(null)
    setAzureError(null)
    setExecutionTime(null)
    setOffset(0)
    currentQueryRef.current = null
  }, [])

  // Reset to default query (clears custom KQL)
  const resetQuery = useCallback(() => {
    if (currentQueryRef.current) {
      const { timeRange } = currentQueryRef.current
      // Re-execute with no custom query
      void executeQuery(timeRange)
    }
  }, [executeQuery])

  return {
    logs,
    total,
    hasMore,
    isLoading,
    error,
    azureError,
    executionTime,
    executeQuery,
    loadMore,
    clearResults,
    resetQuery,
    offset,
  }
}

export default useHistoricalLogs
