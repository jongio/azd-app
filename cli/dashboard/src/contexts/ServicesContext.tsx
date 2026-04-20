import { createContext, useContext, useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from 'react'
import { ConnectError, Code, type Transport } from '@connectrpc/connect'

import { createServicesClient, createLifecycleClient } from '@/lib/connectClient'
import { protoServiceToService } from '@/lib/protoServiceTranslator'
import type { Service } from '@/types'

/**
 * Event type emitted by the server whenever the services list changes
 * (add/update/remove). Must match the Go constant
 * `broadcast.TypeServicesChanged` in
 * `cli/src/internal/dashboard/broadcast/events.go`.
 *
 * The server sends a full bulk snapshot under `payload.services`, so
 * consumers always replace the services array on each event; there
 * is no "add/update/remove" variant on the wire.
 */
const EVENT_SERVICES_CHANGED = 'services-changed'

/** Initial reconnect delay; doubles up to RECONNECT_MAX_MS on each failure. */
const RECONNECT_INITIAL_MS = 500
const RECONNECT_MAX_MS = 30_000

// Mock data for development when backend isn't running
const MOCK_SERVICES: Service[] = [
  {
    name: 'api',
    local: {
      status: 'ready',
      health: 'healthy',
      pid: 12345,
      port: 5000,
      url: 'http://localhost:5000',
      startTime: new Date().toISOString(),
      lastChecked: new Date().toISOString()
    },
    language: 'python',
    framework: 'flask',
    project: '/Users/dev/projects/fullstack'
  },
  {
    name: 'web',
    local: {
      status: 'ready',
      health: 'healthy',
      pid: 12346,
      port: 5001,
      url: 'http://localhost:5001',
      startTime: new Date().toISOString(),
      lastChecked: new Date().toISOString()
    },
    language: 'node',
    framework: 'express',
    project: '/Users/dev/projects/fullstack'
  }
]

/**
 * Context value type for services data
 */
interface ServicesContextValue {
  /** List of all services with real-time updates */
  services: Service[]
  /** Service names for convenience */
  serviceNames: string[]
  /** Whether services are loading */
  loading: boolean
  /** Error message if any */
  error: string | null
  /** Whether connected to WebSocket */
  connected: boolean
  /** Manually refetch services */
  refetch: () => Promise<void>
  /** Get a service by name */
  getService: (name: string) => Service | undefined
}

const ServicesContext = createContext<ServicesContextValue | null>(null)

interface ServicesProviderProps {
  children: ReactNode
  /**
   * Optional Connect transport override. Production code never passes
   * this -- it lets vitest specs inject `createRouterTransport` against
   * an in-memory ServicesService stub instead of monkey-patching fetch.
   */
  transport?: Transport
}

/**
 * Provider for services context.
 * Wraps the application to share services data across all components with real-time WebSocket updates.
 */
export function ServicesProvider({ children, transport }: ServicesProviderProps) {
  const [services, setServices] = useState<Service[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [useMock, setUseMock] = useState(false)

  // Memoize the client per-transport so re-renders don't churn it.
  // Default-arg path resolves to the singleton transport from connectClient.
  const client = useMemo(() => createServicesClient(transport), [transport])
  const lifecycleClient = useMemo(() => createLifecycleClient(transport), [transport])
  const useMockRef = useRef(false)

  const fetchServices = useCallback(async () => {
    try {
      const resp = await client.getServices({})
      setServices(resp.services.map(protoServiceToService))
      setError(null)
      setUseMock(false)
      useMockRef.current = false
    } catch {
      console.warn('Backend not available, using mock data')
      setServices(MOCK_SERVICES)
      setUseMock(true)
      useMockRef.current = true
      setError(null) // Don't show error when using mock data
    } finally {
      setLoading(false)
    }
  }, [client])

  useEffect(() => {
    void fetchServices()

    // Subscribe to the server-streaming Connect RPC that replaces the
    // legacy /api/ws fanout. The server emits a `services-changed`
    // event carrying a full bulk snapshot (`payload.services`) every
    // time the internal registry mutates -- the client-side `update`,
    // `add`, and `remove` branches that existed during the WebSocket
    // era were dead code (the server never emitted them) and have
    // been removed to match the wire contract exactly.
    const abort = new AbortController()
    let cancelled = false

    const run = async () => {
      let backoff = RECONNECT_INITIAL_MS
      while (!cancelled) {
        // Skip the stream entirely when we're in mock mode: the
        // backend isn't running, so dialing StreamBroadcast would
        // just flood the console with reconnect errors.
        if (useMockRef.current) {
          setConnected(false)
          await new Promise((resolve) => setTimeout(resolve, RECONNECT_MAX_MS))
          continue
        }

        try {
          const stream = lifecycleClient.streamBroadcast(
            { eventTypes: [EVENT_SERVICES_CHANGED] },
            { signal: abort.signal },
          )
          setConnected(true)
          backoff = RECONNECT_INITIAL_MS
          for await (const msg of stream) {
            if (cancelled) break
            const ev = msg.event
            if (!ev || ev.type !== EVENT_SERVICES_CHANGED) continue
            const payload = ev.payload?.toJson() as
              | { services?: Service[] }
              | undefined
            if (payload?.services) {
              setServices(payload.services)
            }
          }
        } catch (err) {
          if (cancelled) return
          // Signal-triggered aborts (component unmount) are expected
          // -- swallow them without logging to avoid scary messages
          // in the console during ordinary navigation.
          if (err instanceof ConnectError && err.code === Code.Canceled) {
            return
          }
          if (err instanceof Error && err.name === 'AbortError') {
            return
          }
          console.warn('services broadcast stream interrupted, reconnecting:', err)
        } finally {
          setConnected(false)
        }

        if (cancelled) break
        await new Promise((resolve) => setTimeout(resolve, backoff))
        backoff = Math.min(backoff * 2, RECONNECT_MAX_MS)
      }
    }

    void run()

    return () => {
      cancelled = true
      abort.abort()
    }
  }, [fetchServices, lifecycleClient])

  // Memoize service names for convenience
  const serviceNames = useMemo(() => services.map(s => s.name), [services])

  // Helper to get a service by name
  const getService = useCallback((name: string) => {
    return services.find(s => s.name === name)
  }, [services])

  const value: ServicesContextValue = useMemo(() => ({
    services,
    serviceNames,
    loading,
    error,
    connected: connected || useMock,
    refetch: fetchServices,
    getService,
  }), [services, serviceNames, loading, error, connected, useMock, fetchServices, getService])

  return (
    <ServicesContext.Provider value={value}>
      {children}
    </ServicesContext.Provider>
  )
}

/**
 * Hook to access services context.
 * Must be used within a ServicesProvider.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useServicesContext(): ServicesContextValue {
  const context = useContext(ServicesContext)
  if (!context) {
    throw new Error('useServicesContext must be used within a ServicesProvider')
  }
  return context
}

/**
 * Re-export the old useServices hook for backward compatibility.
 * This allows gradual migration - components can use either approach.
 * @deprecated Use useServicesContext() instead for new components
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useServices() {
  const { services, loading, error, connected, refetch } = useServicesContext()
  return { services, loading, error, connected, refetch }
}
