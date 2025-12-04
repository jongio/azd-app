import { createContext, useContext, useState, useEffect, useCallback, useMemo, type ReactNode } from 'react'
import type { Service } from '@/types'

const API_BASE = ''

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
}

/**
 * Provider for services context.
 * Wraps the application to share services data across all components with real-time WebSocket updates.
 */
export function ServicesProvider({ children }: ServicesProviderProps) {
  const [services, setServices] = useState<Service[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [useMock, setUseMock] = useState(false)

  const fetchServices = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/services`)
      if (!response.ok) throw new Error('Failed to fetch services')
      const data = await response.json() as Service[] | null
      setServices(data ?? [])
      setError(null)
      setUseMock(false)
    } catch {
      console.warn('Backend not available, using mock data')
      setServices(MOCK_SERVICES)
      setUseMock(true)
      setError(null) // Don't show error when using mock data
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchServices()

    // Set up WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)
    let isMounted = true

    ws.onopen = () => {
      if (isMounted) {
        setConnected(true)
      }
    }

    ws.onmessage = (event: MessageEvent<string>) => {
      if (!isMounted) return
      try {
        const update = JSON.parse(event.data) as { type: string; service?: Service; services?: Service[] }
        if (update.type === 'services' && update.services) {
          // Bulk update: replace all services
          setServices(update.services)
        } else if ((update.type === 'update' || update.type === 'add') && update.service) {
          setServices(prev => {
            const index = prev.findIndex(
              s => s.name === update.service!.name
            )
            if (index >= 0) {
              const updated = [...prev]
              updated[index] = update.service!
              return updated
            }
            return [...prev, update.service!]
          })
        } else if (update.type === 'remove' && update.service) {
          setServices(prev =>
            prev.filter(
              s => s.name !== update.service!.name
            )
          )
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err)
      }
    }

    ws.onerror = () => {
      if (isMounted) {
        setConnected(false)
        console.warn('WebSocket not available (this is normal in dev mode)')
      }
    }

    ws.onclose = () => {
      if (isMounted) {
        setConnected(false)
      }
    }

    return () => {
      isMounted = false
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close(1000, 'Component unmounting')
      }
    }
  }, [fetchServices])

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
