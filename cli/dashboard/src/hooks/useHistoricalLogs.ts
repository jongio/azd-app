/**
 * useHistoricalLogs - Hook for querying historical Azure logs via the
 * Connect AzureService.
 *
 * Wire migration note: previously POSTed JSON to `/api/azure/logs/query`,
 * which was a planned legacy endpoint that never actually shipped on
 * the Go side (the route was wired but returned 404). The hook is
 * preserved because the Connect proto exposes the equivalent capability
 * via `getAzureLogs` and a future panel will consume it.
 *
 * Pagination intentionally degraded: the proto `GetAzureLogs` RPC does
 * not surface a server-side offset/cursor (the legacy plan was to add
 * one but the dashboard never relied on it). To keep the hook surface
 * stable we treat the unary response as the full result set: `total`
 * equals `entries.length`, `hasMore` is always `false`, and `loadMore`
 * is a no-op. Callers retain the same options shape so the
 * `HistoricalLogPanel` (currently unwired) can light up later without
 * another type churn. Custom KQL is also dropped because the unary RPC
 * does not currently accept it; the field remains in the options for
 * forward compatibility.
 */
import { useCallback, useMemo, useRef, useState } from 'react'
import { ConnectError, type PromiseClient, type Transport } from '@connectrpc/connect'
import { protoInt64 } from '@bufbuild/protobuf'

import type { ParsedAzureError } from '@/types'
import { createParsedAzureError } from '@/lib/azure-errors'
import { useBackendConnection } from '@/hooks/useBackendConnection'
import { createAzureClient } from '@/lib/connectClient'
import type { AzureService } from '@/gen/proto/azdapp/v1/azure_connect.js'
import {
  GetAzureLogsRequest,
  type GetAzureLogsResponse,
} from '@/gen/proto/azdapp/v1/azure_pb.js'
import { LogLevel, LogStream, type LogEntry as ProtoLogEntry } from '@/gen/proto/azdapp/v1/common_pb.js'

// =============================================================================
// Types (preserved from legacy hook surface)
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
  serviceName: string
  pageSize?: number
  /** Test transport injection - production omits */
  transport?: Transport
}

export interface UseHistoricalLogsReturn {
  logs: HistoricalLogEntry[]
  total: number
  hasMore: boolean
  isLoading: boolean
  error: string | null
  azureError: ParsedAzureError | null
  executionTime: number | null
  executeQuery: (timeRange: TimeRange, customKql?: string) => Promise<void>
  loadMore: () => Promise<void>
  clearResults: () => void
  resetQuery: () => void
  offset: number
}

// =============================================================================
// Constants
// =============================================================================

const DEFAULT_PAGE_SIZE = 100
const SECONDS_PER_MINUTE = 60
const SECONDS_PER_HOUR = 60 * 60
const SECONDS_PER_DAY = 60 * 60 * 24

// =============================================================================
// Time range helpers (exported for component reuse)
// =============================================================================

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

  return 'PT30M'
}

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
    const formatDate = (d: Date) =>
      d.toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      })
    return `${formatDate(timeRange.start)} - ${formatDate(timeRange.end)}`
  }

  return 'custom range'
}

/**
 * Convert a TimeRange (or the equivalent ISO 8601 duration the legacy
 * REST sent) into seconds for the proto `since_seconds` field. Anything
 * unparseable falls back to 30 minutes - matches the legacy server
 * default when `timespan` was empty.
 */
function timeRangeToSeconds(timeRange: TimeRange): number {
  if (timeRange.preset === 'custom' && timeRange.start && timeRange.end) {
    const durationMs = Math.max(0, timeRange.end.getTime() - timeRange.start.getTime())
    return Math.ceil(durationMs / 1000)
  }
  switch (timeRange.preset) {
    case '15m':
      return 15 * SECONDS_PER_MINUTE
    case '30m':
      return 30 * SECONDS_PER_MINUTE
    case '6h':
      return 6 * SECONDS_PER_HOUR
    case '24h':
      return SECONDS_PER_DAY
    default:
      return 30 * SECONDS_PER_MINUTE
  }
}

// =============================================================================
// Proto -> dashboard mappers
// =============================================================================

function levelToNumeric(level: LogLevel): number {
  // Mirrors `protoLevelToNumeric` in useSharedLogStream so historical
  // and realtime entries render with consistent badges.
  switch (level) {
    case LogLevel.ERROR:
    case LogLevel.FATAL:
      return 3
    case LogLevel.WARN:
      return 2
    case LogLevel.INFO:
    case LogLevel.TRACE:
    case LogLevel.DEBUG:
    case LogLevel.UNSPECIFIED:
    default:
      return 1
  }
}

function readStringField(fields: Record<string, unknown> | null, key: string): string | undefined {
  if (!fields) return undefined
  const v = fields[key]
  return typeof v === 'string' && v.length > 0 ? v : undefined
}

function protoEntryToHistorical(entry: ProtoLogEntry): HistoricalLogEntry {
  const out: HistoricalLogEntry = {
    service: entry.service,
    message: entry.message,
    level: levelToNumeric(entry.level),
    timestamp: entry.timestamp ? entry.timestamp.toDate().toISOString() : '',
    isStderr: entry.stream === LogStream.STDERR,
  }
  if (entry.fields) {
    const json = entry.fields.toJson()
    const obj =
      json && typeof json === 'object' && !Array.isArray(json)
        ? (json as Record<string, unknown>)
        : null
    const resourceId = readStringField(obj, 'resourceId') ?? readStringField(obj, 'resource_id')
    const operationName =
      readStringField(obj, 'operationName') ?? readStringField(obj, 'operation_name')
    if (resourceId) out.resourceId = resourceId
    if (operationName) out.operationName = operationName
  }
  return out
}

function buildResult(resp: GetAzureLogsResponse, executionTimeMs: number): HistoricalLogResult {
  return {
    logs: resp.entries.map(protoEntryToHistorical),
    total: resp.entries.length,
    // Proto `getAzureLogs` is a single-shot unary RPC - there is no
    // server-side cursor. Reflect that honestly so consumers don't render
    // a phantom "load more" link.
    hasMore: false,
    executionTime: executionTimeMs,
  }
}

// =============================================================================
// Hook
// =============================================================================

export function useHistoricalLogs({
  serviceName,
  pageSize = DEFAULT_PAGE_SIZE,
  transport,
}: UseHistoricalLogsOptions): UseHistoricalLogsReturn {
  const { connected } = useBackendConnection()
  const client = useMemo<PromiseClient<typeof AzureService>>(
    () => createAzureClient(transport),
    [transport],
  )

  const [logs, setLogs] = useState<HistoricalLogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [azureError, setAzureError] = useState<ParsedAzureError | null>(null)
  const [executionTime, setExecutionTime] = useState<number | null>(null)

  // The proto RPC is unary and does not paginate, so `offset` stays 0.
  // It remains in the return for API stability.
  const offset = 0

  const currentQueryRef = useRef<{
    timeRange: TimeRange
    customKql?: string
  } | null>(null)

  const executeQuery = useCallback(
    async (timeRange: TimeRange, customKql?: string) => {
      if (!connected) {
        setError('Backend connection lost')
        return
      }
      if (!serviceName) {
        setError('Service name is required')
        return
      }

      setIsLoading(true)
      setError(null)
      setAzureError(null)
      currentQueryRef.current = { timeRange, customKql }

      const startedAt = Date.now()
      try {
        const resp = await client.getAzureLogs(
          new GetAzureLogsRequest({
            service: serviceName,
            sinceSeconds: protoInt64.parse(timeRangeToSeconds(timeRange)),
            tail: pageSize,
          }),
        )
        const result = buildResult(resp, Date.now() - startedAt)
        setLogs(result.logs)
        setTotal(result.total)
        setHasMore(result.hasMore)
        setExecutionTime(result.executionTime)
      } catch (err) {
        const message =
          err instanceof ConnectError
            ? err.rawMessage || err.message
            : err instanceof Error
            ? err.message
            : 'Query failed'
        // `createParsedAzureError` accepts an optional Response; without
        // one it still classifies based on message content (e.g. rate
        // limit detection by phrase), which is what we want here since
        // ConnectError doesn't carry a Response.
        setError(message)
        setAzureError(createParsedAzureError(message))
        setLogs([])
        setTotal(0)
        setHasMore(false)
      } finally {
        setIsLoading(false)
      }
    },
    [client, connected, serviceName, pageSize],
  )

  /**
   * Pagination intentionally not supported through the proto API. Kept
   * as a no-op so the existing consumer prop wiring continues to type
   * check; calling it after the first page is a no-op rather than an
   * error to avoid spurious failures from list virtualisation
   * components that fire it on scroll.
   */
  const loadMore = useCallback(async () => {
    /* no-op: proto getAzureLogs is unary; see file header */
  }, [])

  const clearResults = useCallback(() => {
    setLogs([])
    setTotal(0)
    setHasMore(false)
    setError(null)
    setAzureError(null)
    setExecutionTime(null)
    currentQueryRef.current = null
  }, [])

  const resetQuery = useCallback(() => {
    if (currentQueryRef.current) {
      const { timeRange } = currentQueryRef.current
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
