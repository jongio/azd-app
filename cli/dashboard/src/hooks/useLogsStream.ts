import { useEffect, useRef, useMemo } from 'react'
import type { LogEntry } from '@/components/LogsPane'
import type { LogMode } from '@/components/ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import { MAX_LOGS_IN_MEMORY } from '@/lib/log-utils'

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
  refreshTrigger: number
  isPausedRef: { current: boolean }
  setLogs: React.Dispatch<React.SetStateAction<LogEntry[]>>
  setErrorMessage: React.Dispatch<React.SetStateAction<string | null>>
  onFetchSettled?: () => void
}

export function useLogsStream(params: UseLogsStreamParams): void {
  const { serviceName, fetchKey, logMode, timeRange, azureRealtime, refreshTrigger, isPausedRef, setLogs, setErrorMessage, onFetchSettled } = params
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    // Choose endpoint based on mode
    const logsEndpoint = logMode === 'azure' ? '/api/azure/logs' : '/api/logs'
    const streamEndpoint = logMode === 'azure' ? '/api/azure/logs/stream' : '/api/logs/stream'

    let cancelled = false
    const fetchLogs = async () => {
      try {
        setErrorMessage(null)
        const url = buildLogsFetchUrl(logsEndpoint, serviceName, logMode, timeRange)
        const res = await fetch(url)
        if (!res.ok) {
          const message = (await res.text()) || `HTTP ${res.status}`
          setErrorMessage(message)
          return
        }

        const data: unknown = await res.json()
        const nextLogs = parseLogsPayload(logMode, data)
        setLogs(nextLogs)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Request failed'
        setErrorMessage(message)
        console.error(`[LogsPane:${serviceName}] Failed to fetch ${logMode} logs:`, err)
      } finally {
        if (!cancelled) {
          onFetchSettled?.()
        }
      }
    }

    void fetchLogs()

    const closeSocket = () => {
      if (!wsRef.current) return
      const currentWs = wsRef.current
      if (currentWs.readyState === WebSocket.OPEN || currentWs.readyState === WebSocket.CONNECTING) {
        currentWs.close(1000, 'Mode change or unmount')
      }
      wsRef.current = null
    }

    // Azure logs can either poll via REST or stream via WebSocket (realtime)
    if (logMode === 'azure') {
      if (!azureRealtime) {
        // Azure polling mode: fetchLogs is called on mount and when refreshTrigger changes
        return () => {
          cancelled = true
          closeSocket()
        }
      }

      const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const ws = new WebSocket(
        `${protocol}//${globalThis.location.host}${streamEndpoint}?service=${serviceName}&realtime=true`
      )

      ws.onmessage = (event) => {
        if (isPausedRef.current) return
        try {
          const entry = JSON.parse(event.data as string) as LogEntry
          setLogs((prev) => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
        } catch (err) {
          console.error('Failed to parse Azure log entry:', err)
        }
      }

      ws.onerror = (err) => {
        if (ws.readyState !== WebSocket.CLOSED) {
          setErrorMessage('WebSocket connection error')
          console.error(`WebSocket error for ${serviceName} (${logMode}):`, err)
        }
      }

      wsRef.current = ws
      return () => {
        cancelled = true
        closeSocket()
      }
    }

    // For local logs, use WebSocket streaming
    const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${globalThis.location.host}${streamEndpoint}?service=${serviceName}`)

    ws.onmessage = (event) => {
      if (isPausedRef.current) return
      try {
        const entry = JSON.parse(event.data as string) as LogEntry
        setLogs((prev) => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
      } catch (err) {
        console.error('Failed to parse log entry:', err)
      }
    }

    ws.onerror = (err) => {
      if (ws.readyState !== WebSocket.CLOSED) {
        setErrorMessage('WebSocket connection error')
        console.error(`WebSocket error for ${serviceName} (${logMode}):`, err)
      }
    }

    wsRef.current = ws
    return () => {
      cancelled = true
      closeSocket()
    }
  }, [serviceName, fetchKey, refreshTrigger, isPausedRef, setLogs, onFetchSettled, azureRealtime, logMode, timeRange])
}
