import { useState, useEffect, useRef, useCallback } from 'react'
import { MAX_LOGS_IN_MEMORY } from '@/lib/log-utils'

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

interface UseLogStreamOptions {
  /** Service name to filter logs. If 'all' or undefined, returns logs from all services */
  serviceName?: string
  /** Number of historical logs to fetch initially. Defaults to 500 */
  initialTail?: number
  /** Whether to pause streaming. Defaults to false */
  isPaused?: boolean
  /** Callback when logs are cleared externally */
  onClearTrigger?: number
}

// WebSocket reconnection constants
const WS_INITIAL_RETRY_DELAY_MS = 1000
const WS_MAX_RETRY_DELAY_MS = 30000
const WS_MAX_RETRIES = 10

/**
 * Shared hook for streaming logs from the backend via WebSocket.
 * Consolidates WebSocket logic previously duplicated across LogsView, LogsPane, and useServiceErrors.
 * 
 * Features:
 * - Fetches initial logs from REST API
 * - Streams new logs via WebSocket
 * - Handles connection lifecycle and cleanup
 * - Respects pause state for buffering
 * - Limits memory usage with MAX_LOGS_IN_MEMORY
 * - Automatic reconnection with exponential backoff
 */
export function useLogStream({
  serviceName,
  initialTail = 500,
  isPaused = false,
  onClearTrigger = 0
}: UseLogStreamOptions = {}) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const isPausedRef = useRef(isPaused)
  const retryCountRef = useRef(0)
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isMountedRef = useRef(true)

  // Keep isPaused ref in sync without causing reconnects
  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

  // Clear logs when trigger changes
  useEffect(() => {
    if (onClearTrigger > 0) {
      setLogs([])
    }
  }, [onClearTrigger])

  // Fetch initial logs
  const fetchLogs = useCallback(async () => {
    const serviceParam = serviceName && serviceName !== 'all' ? `service=${serviceName}&` : ''
    const url = `/api/logs?${serviceParam}tail=${initialTail}`

    try {
      const res = await fetch(url)
      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`)
      }
      const data = await res.json() as LogEntry[]
      setLogs(data || [])
    } catch (err) {
      console.error('Failed to fetch logs:', err)
      setLogs([])
    }
  }, [serviceName, initialTail])

  // Schedule reconnection with exponential backoff
  const scheduleReconnect = useCallback((setupFn: () => void) => {
    if (!isMountedRef.current) return
    if (retryCountRef.current >= WS_MAX_RETRIES) {
      console.error(`WebSocket: Max retries (${WS_MAX_RETRIES}) exceeded, giving up`)
      return
    }

    // Calculate delay with exponential backoff: 1s, 2s, 4s, 8s, ... up to 30s
    const delay = Math.min(
      WS_INITIAL_RETRY_DELAY_MS * Math.pow(2, retryCountRef.current),
      WS_MAX_RETRY_DELAY_MS
    )
    
    console.warn(`WebSocket: Reconnecting in ${delay}ms (attempt ${retryCountRef.current + 1}/${WS_MAX_RETRIES})`)
    
    retryTimeoutRef.current = setTimeout(() => {
      if (isMountedRef.current) {
        retryCountRef.current++
        setupFn()
      }
    }, delay)
  }, [])

  // Setup WebSocket connection
  const setupWebSocket = useCallback(() => {
    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const serviceParam = serviceName && serviceName !== 'all' ? `?service=${serviceName}` : ''
    const url = `${protocol}//${window.location.host}/api/logs/stream${serviceParam}`

    const ws = new WebSocket(url)

    ws.onopen = () => {
      setIsConnected(true)
      // Reset retry count on successful connection
      retryCountRef.current = 0
    }

    ws.onmessage = (event: MessageEvent<string>) => {
      // Check pause state from ref to avoid stale closure
      if (!isPausedRef.current) {
        try {
          const entry = JSON.parse(event.data) as LogEntry
          setLogs(prev => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
        } catch (err) {
          console.error('Failed to parse log entry:', err)
        }
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      setIsConnected(false)
    }

    ws.onclose = (event) => {
      setIsConnected(false)
      
      // Don't reconnect if this was a clean close (e.g., component unmounting)
      // Code 1000 = normal closure, 1001 = going away (page navigation)
      if (event.code !== 1000 && event.code !== 1001 && isMountedRef.current) {
        scheduleReconnect(setupWebSocket)
      }
    }

    wsRef.current = ws
  }, [serviceName, scheduleReconnect])

  // Initialize connection
  useEffect(() => {
    isMountedRef.current = true
    void fetchLogs()
    setupWebSocket()

    return () => {
      isMountedRef.current = false
      
      // Clear any pending reconnection
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current)
        retryTimeoutRef.current = null
      }
      
      if (wsRef.current) {
        if (wsRef.current.readyState === WebSocket.OPEN || 
            wsRef.current.readyState === WebSocket.CONNECTING) {
          wsRef.current.close(1000, 'Component unmounting')
        }
        wsRef.current = null
      }
    }
  }, [fetchLogs, setupWebSocket])

  const clearLogs = useCallback(() => {
    setLogs([])
  }, [])

  return {
    logs,
    isConnected,
    clearLogs,
    refetch: fetchLogs
  }
}
