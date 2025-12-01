import { useState, useEffect, useCallback, useRef } from 'react'
import type { HealthReportEvent, HealthChangeEvent, HealthCheckResult, HealthSummary } from '@/types'

/** Configuration options for the health stream hook */
export interface UseHealthStreamOptions {
  /** Whether to enable health streaming (default: true) */
  enabled?: boolean
  /** Interval between health checks in seconds (default: 5) */
  interval?: number
  /** Optional service filter */
  services?: string[]
  /** Reconnect delay in ms after disconnect (default: 3000) */
  reconnectDelay?: number
  /** Maximum reconnect attempts (default: 5) */
  maxReconnectAttempts?: number
}

/** Return type for the health stream hook */
export interface UseHealthStreamReturn {
  /** Latest health report */
  healthReport: HealthReportEvent | null
  /** Recent health changes (newest first) */
  changes: HealthChangeEvent[]
  /** Whether connected to SSE stream */
  connected: boolean
  /** Error message if any */
  error: string | null
  /** Last update timestamp */
  lastUpdate: Date | null
  /** Health summary for quick access */
  summary: HealthSummary | null
  /** Get health result for a specific service */
  getServiceHealth: (serviceName: string) => HealthCheckResult | undefined
  /** Check if a service has recovered (was unhealthy, now healthy) */
  hasRecovered: (serviceName: string) => boolean
  /** Get the most recent change for a service */
  getLatestChange: (serviceName: string) => HealthChangeEvent | undefined
  /** Clear change history */
  clearChanges: () => void
  /** Manually trigger reconnection */
  reconnect: () => void
}

const API_BASE = ''
const DEFAULT_INTERVAL = 5
const DEFAULT_RECONNECT_DELAY = 3000
const DEFAULT_MAX_RECONNECT_ATTEMPTS = 5
const MAX_CHANGES_TO_KEEP = 50

/**
 * Hook for consuming health check data from the SSE stream.
 * Automatically connects to /api/health/stream and handles reconnection.
 */
export function useHealthStream(options: UseHealthStreamOptions = {}): UseHealthStreamReturn {
  const {
    enabled = true,
    interval = DEFAULT_INTERVAL,
    services,
    reconnectDelay = DEFAULT_RECONNECT_DELAY,
    maxReconnectAttempts = DEFAULT_MAX_RECONNECT_ATTEMPTS,
  } = options

  const [healthReport, setHealthReport] = useState<HealthReportEvent | null>(null)
  const [changes, setChanges] = useState<HealthChangeEvent[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null)

  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const countdownIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Build SSE URL with parameters
  const buildUrl = useCallback(() => {
    const params = new URLSearchParams()
    params.set('interval', `${interval}s`)
    if (services && services.length > 0) {
      params.set('service', services.join(','))
    }
    return `${API_BASE}/api/health/stream?${params.toString()}`
  }, [interval, services])

  // Handle incoming health event
  const handleHealthEvent = useCallback((event: MessageEvent<string>) => {
    try {
      const data = JSON.parse(event.data) as HealthReportEvent
      setHealthReport(data)
      setLastUpdate(new Date())
      setError(null)
    } catch (err) {
      console.error('Failed to parse health event:', err)
    }
  }, [])

  // Handle health change event
  const handleChangeEvent = useCallback((event: MessageEvent<string>) => {
    try {
      const data = JSON.parse(event.data) as HealthChangeEvent
      setChanges(prev => {
        const updated = [data, ...prev]
        // Keep only recent changes
        return updated.slice(0, MAX_CHANGES_TO_KEEP)
      })
      setLastUpdate(new Date())
    } catch (err) {
      console.error('Failed to parse health change event:', err)
    }
  }, [])

  // Handle heartbeat event
  const handleHeartbeatEvent = useCallback(() => {
    setLastUpdate(new Date())
  }, [])

  // Cleanup function
  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (countdownIntervalRef.current) {
      clearInterval(countdownIntervalRef.current)
      countdownIntervalRef.current = null
    }
  }, [])

  // Connect to SSE stream
  const connect = useCallback(() => {
    cleanup()

    if (!enabled) {
      return
    }

    const url = buildUrl()
    const eventSource = new EventSource(url)
    eventSourceRef.current = eventSource

    eventSource.onopen = () => {
      setConnected(true)
      setError(null)
      reconnectAttemptsRef.current = 0
    }

    // Default message handler (for 'health' events without explicit event type)
    eventSource.onmessage = handleHealthEvent

    // Named event handlers
    eventSource.addEventListener('health-change', handleChangeEvent as EventListener)
    eventSource.addEventListener('heartbeat', handleHeartbeatEvent as EventListener)

    eventSource.onerror = () => {
      setConnected(false)

      // Clear any existing countdown
      if (countdownIntervalRef.current) {
        clearInterval(countdownIntervalRef.current)
        countdownIntervalRef.current = null
      }

      // Attempt reconnection
      if (reconnectAttemptsRef.current < maxReconnectAttempts) {
        reconnectAttemptsRef.current++
        const delay = reconnectDelay * Math.pow(2, reconnectAttemptsRef.current - 1)
        let remainingSeconds = Math.ceil(delay / 1000)
        
        // Set initial countdown message
        setError(`Connection lost. Reconnecting in ${remainingSeconds}s...`)
        
        // Update countdown every second
        countdownIntervalRef.current = setInterval(() => {
          remainingSeconds--
          if (remainingSeconds > 0) {
            setError(`Connection lost. Reconnecting in ${remainingSeconds}s...`)
          }
        }, 1000)

        reconnectTimeoutRef.current = setTimeout(() => {
          // Clear countdown interval when reconnecting
          if (countdownIntervalRef.current) {
            clearInterval(countdownIntervalRef.current)
            countdownIntervalRef.current = null
          }
          connect()
        }, delay)
      } else {
        setError('Failed to connect to health stream. Please refresh the page.')
      }
    }
  }, [
    enabled,
    buildUrl,
    cleanup,
    handleHealthEvent,
    handleChangeEvent,
    handleHeartbeatEvent,
    maxReconnectAttempts,
    reconnectDelay,
  ])

  // Manual reconnect function
  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0
    connect()
  }, [connect])

  // Get health for a specific service
  const getServiceHealth = useCallback(
    (serviceName: string): HealthCheckResult | undefined => {
      return healthReport?.services.find(s => s.serviceName === serviceName)
    },
    [healthReport]
  )

  // Get the most recent change for a service
  const getLatestChange = useCallback(
    (serviceName: string): HealthChangeEvent | undefined => {
      return changes.find(c => c.service === serviceName)
    },
    [changes]
  )

  // Check if a service has recovered (most recent change was to healthy)
  const hasRecovered = useCallback(
    (serviceName: string): boolean => {
      const latestChange = getLatestChange(serviceName)
      return latestChange?.newStatus === 'healthy' && latestChange?.oldStatus !== 'healthy'
    },
    [getLatestChange]
  )

  // Clear change history
  const clearChanges = useCallback(() => {
    setChanges([])
  }, [])

  // Connect on mount, cleanup on unmount
  useEffect(() => {
    connect()
    return cleanup
  }, [connect, cleanup])

  // Summary for quick access
  const summary = healthReport?.summary ?? null

  return {
    healthReport,
    changes,
    connected,
    error,
    lastUpdate,
    summary,
    getServiceHealth,
    hasRecovered,
    getLatestChange,
    clearChanges,
    reconnect,
  } satisfies UseHealthStreamReturn
}
