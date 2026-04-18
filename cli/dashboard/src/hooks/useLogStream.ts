/**
 * useLogStream — standalone log-stream hook (no shared multiplexer).
 *
 * Wire migration note: the streaming portion has moved from the
 * legacy `/api/logs/stream` WebSocket onto `LogsService.StreamLocalLogs`
 * (Connect server-streaming). The initial REST GET against `/api/logs`
 * stays - the unary `LogsService.GetLogs` migration is a separate
 * commit because it touches the static-cache and pagination story.
 *
 * Public surface preserved exactly: same options, same return
 * `{ logs, isConnected, clearLogs, refetch }`. New additive fields:
 *   - `droppedCount`: total log entries the server reported as dropped
 *     on the active stream (drop-OLDEST back-pressure). UI can render
 *     a "lost N lines" banner.
 *   - `transport`: optional `Transport` for tests; production omits it
 *     and the hook builds its own client.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Code, ConnectError, type Transport } from '@connectrpc/connect'

import { MAX_LOGS_IN_MEMORY } from '@/lib/log-utils'
import { WEBSOCKET_CONSTANTS } from '@/lib/constants'
import { createLogsClient } from '@/lib/connectClient'
import { protoToLogEntry } from '@/hooks/useSharedLogStream'
import { StreamLocalLogsRequest } from '@/gen/proto/azdapp/v1/logs_pb.js'

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

interface UseLogStreamOptions {
  /** Service name to filter logs. If 'all' or undefined, returns logs from all services. */
  serviceName?: string
  /** Number of historical logs to fetch initially. Defaults to 500. */
  initialTail?: number
  /** Whether to pause streaming. Defaults to false. */
  isPaused?: boolean
  /** Trigger to clear logs externally; non-zero new value clears state. */
  onClearTrigger?: number
  /**
   * Test seam. Production code never passes a transport; tests inject
   * `createRouterTransport(...)`. Same convention as `useHealthStream`.
   */
  transport?: Transport
}

const INITIAL_RETRY_DELAY_MS = WEBSOCKET_CONSTANTS.WS_INITIAL_RETRY_DELAY_MS
const MAX_RETRY_DELAY_MS = WEBSOCKET_CONSTANTS.WS_MAX_RETRY_DELAY_MS
const MAX_RETRIES = WEBSOCKET_CONSTANTS.WS_MAX_RETRIES

/**
 * Hook for streaming a single service's logs (or all services). Owns
 * its own connection lifecycle, exponential backoff and memory cap.
 */
export function useLogStream({
  serviceName,
  initialTail = 500,
  isPaused = false,
  onClearTrigger = 0,
  transport,
}: UseLogStreamOptions = {}) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [droppedCount, setDroppedCount] = useState(0)

  const isPausedRef = useRef(isPaused)
  const isMountedRef = useRef(true)
  const controllerRef = useRef<AbortController | null>(null)
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const retryCountRef = useRef(0)
  const connectRef = useRef<() => void>(() => {})

  const client = useMemo(() => createLogsClient(transport), [transport])

  // Keep isPaused ref in sync without forcing reconnects.
  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

  // External clear trigger.
  useEffect(() => {
    if (onClearTrigger > 0) {
      setLogs([])
    }
  }, [onClearTrigger])

  const fetchLogs = useCallback(async () => {
    const serviceParam = serviceName && serviceName !== 'all' ? `service=${serviceName}&` : ''
    const url = `/api/logs?${serviceParam}tail=${initialTail}`

    try {
      const res = await fetch(url)
      if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)
      const data = (await res.json()) as LogEntry[]
      if (!isMountedRef.current) return
      setLogs(data ?? [])
    } catch (err) {
      console.error('Failed to fetch logs:', err)
      if (isMountedRef.current) setLogs([])
    }
  }, [serviceName, initialTail])

  // Build the connect closure once, then store in a ref so the
  // setTimeout-based retry path always invokes the latest function
  // (matches useHealthStream's connectRef pattern).
  const connect = useCallback(() => {
    if (!isMountedRef.current) return

    // Tear down previous controller before issuing a fresh one so an
    // old in-flight stream can't race with the new one.
    if (controllerRef.current) {
      controllerRef.current.abort()
    }

    const controller = new AbortController()
    controllerRef.current = controller

    let firstMessageSeen = false

    void (async () => {
      try {
        const req = new StreamLocalLogsRequest({
          serviceName: serviceName && serviceName !== 'all' ? serviceName : '',
          // Backfill 0 because the REST GET above already populated
          // `logs` with the historical tail; backfill here would
          // duplicate those entries.
          backfill: 0,
        })

        for await (const resp of client.streamLocalLogs(req, { signal: controller.signal })) {
          if (!isMountedRef.current || controller.signal.aborted) break

          if (!firstMessageSeen) {
            firstMessageSeen = true
            setIsConnected(true)
            retryCountRef.current = 0
          }

          const event = resp.event
          if (!event) continue
          if (event.case === 'entry') {
            const proto = event.value
            if (!proto || !proto.service) continue
            // Honour pause without dropping the stream: the Connect
            // back-pressure design is "drop oldest server-side", so
            // the UI keeps the wire alive and silently ignores
            // entries while paused.
            if (isPausedRef.current) continue
            const entry = protoToLogEntry(proto)
            setLogs((prev) => {
              const next = prev.concat(entry)
              return next.length > MAX_LOGS_IN_MEMORY ? next.slice(-MAX_LOGS_IN_MEMORY) : next
            })
          } else if (event.case === 'dropped') {
            const count = Number(event.value?.count ?? 0)
            if (count > 0) {
              setDroppedCount((prev) => prev + count)
            }
          }
        }

        // Stream ended cleanly. The server only ends StreamLocalLogs
        // on context cancellation or process shutdown; treat as
        // transient and reschedule with backoff.
        if (isMountedRef.current && !controller.signal.aborted) {
          setIsConnected(false)
          scheduleReconnect()
        }
      } catch (err) {
        if (controller.signal.aborted) return
        if (err instanceof ConnectError && err.code === Code.Canceled) return
        if (!isMountedRef.current) return
        if (retryCountRef.current === 0) {
          console.warn(
            'LogsService.StreamLocalLogs error:',
            err instanceof Error ? err.message : 'Unknown error',
          )
        }
        setIsConnected(false)
        scheduleReconnect()
      } finally {
        if (controllerRef.current === controller) {
          controllerRef.current = null
        }
      }
    })()
  // eslint-disable-next-line react-hooks/exhaustive-deps -- scheduleReconnect is stable (depends only on refs)
  }, [client, serviceName])

  const scheduleReconnect = useCallback(() => {
    if (!isMountedRef.current) return
    if (retryTimeoutRef.current) return
    if (retryCountRef.current >= MAX_RETRIES) {
      console.error(`useLogStream: Max retries (${MAX_RETRIES}) exceeded, giving up`)
      return
    }

    const delay = Math.min(
      INITIAL_RETRY_DELAY_MS * Math.pow(2, retryCountRef.current),
      MAX_RETRY_DELAY_MS,
    )
    console.warn(
      `useLogStream: Reconnecting in ${delay}ms (attempt ${retryCountRef.current + 1}/${MAX_RETRIES})`,
    )

    retryTimeoutRef.current = setTimeout(() => {
      retryTimeoutRef.current = null
      if (!isMountedRef.current) return
      retryCountRef.current++
      connectRef.current()
    }, delay)
  }, [])

  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  // Initialize: fetch + open stream.
  useEffect(() => {
    isMountedRef.current = true
    void fetchLogs()
    connect()

    return () => {
      isMountedRef.current = false
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current)
        retryTimeoutRef.current = null
      }
      if (controllerRef.current) {
        controllerRef.current.abort()
        controllerRef.current = null
      }
    }
  }, [fetchLogs, connect])

  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  return {
    logs,
    isConnected,
    /** Cumulative drop count from the server (drop-oldest back-pressure). */
    droppedCount,
    clearLogs,
    refetch: fetchLogs,
  }
}
