import { useState, useEffect, useRef } from 'react'
import { LOG_LEVELS, isErrorLine } from '@/lib/log-utils'

interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

/**
 * Hook to track active errors across all services by monitoring WebSocket log streams.
 * Returns true if any service has error-level logs in the last 30 seconds.
 */
export function useServiceErrors() {
  const [hasActiveErrors, setHasActiveErrors] = useState(false)
  const [services, setServices] = useState<string[]>([])
  const errorTimestampsRef = useRef<Map<string, number>>(new Map())
  const websocketsRef = useRef<Map<string, WebSocket>>(new Map())

  // Fetch services
  useEffect(() => {
    const fetchServices = async () => {
      try {
        const res = await fetch('/api/services')
        if (!res.ok) return
        const data = await res.json() as Array<{ name: string }>
        const serviceNames = data.map(s => s.name)
        setServices(serviceNames)
      } catch (err) {
        console.error('Failed to fetch services for error tracking:', err)
      }
    }
    void fetchServices()
  }, [])

  // Setup WebSocket connections for each service
  useEffect(() => {
    if (services.length === 0) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const websockets = websocketsRef.current

    // Clean up old websockets
    websockets.forEach(ws => ws.close())
    websockets.clear()

    // Create new websockets for each service
    services.forEach(serviceName => {
      const ws = new WebSocket(`${protocol}//${window.location.host}/api/logs/stream?service=${serviceName}`)

      ws.onmessage = (event) => {
        try {
          const entry = JSON.parse(event.data as string) as LogEntry
          
          // Check if this is an error log using centralized detection
          const hasError = entry.level === LOG_LEVELS.ERROR || isErrorLine(entry.message)
          
          if (hasError) {
            errorTimestampsRef.current.set(`${serviceName}-${Date.now()}`, Date.now())
          }
        } catch (err) {
          console.error('Failed to parse log entry for error tracking:', err)
        }
      }

      websockets.set(serviceName, ws)
    })

    return () => {
      websockets.forEach(ws => ws.close())
      websockets.clear()
    }
  }, [services])

  // Periodically check for active errors (within last 30 seconds)
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now()
      const thirtySecondsAgo = now - 30000
      
      // Remove old error timestamps
      const entries = Array.from(errorTimestampsRef.current.entries())
      entries.forEach(([key, timestamp]) => {
        if (timestamp < thirtySecondsAgo) {
          errorTimestampsRef.current.delete(key)
        }
      })
      
      // Update hasActiveErrors based on remaining errors
      setHasActiveErrors(errorTimestampsRef.current.size > 0)
    }, 1000) // Check every second

    return () => clearInterval(interval)
  }, [])

  return { hasActiveErrors }
}
