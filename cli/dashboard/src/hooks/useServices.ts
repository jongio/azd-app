import { useState, useEffect, useCallback } from 'react'
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

export function useServices() {
  const [services, setServices] = useState<Service[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [connected, setConnected] = useState(false)
  const [useMock, setUseMock] = useState(false)

  const fetchServices = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/services`)
      if (!response.ok) throw new Error('Failed to fetch services')
      const data = await response.json()
      setServices(data || [])
      setError(null)
      setUseMock(false)
    } catch (err) {
      console.log('Backend not available, using mock data')
      setServices(MOCK_SERVICES)
      setUseMock(true)
      setError(null) // Don't show error when using mock data
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchServices()

    // Set up WebSocket connection
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)

    ws.onopen = () => {
      setConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data)
        if (update.type === 'update' || update.type === 'add') {
          setServices(prev => {
            const index = prev.findIndex(
              s => s.name === update.service.name
            )
            if (index >= 0) {
              const updated = [...prev]
              updated[index] = update.service
              return updated
            }
            return [...prev, update.service]
          })
        } else if (update.type === 'remove') {
          setServices(prev =>
            prev.filter(
              s => s.name !== update.service.name
            )
          )
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err)
      }
    }

    ws.onerror = () => {
      setConnected(false)
      console.log('WebSocket not available (this is normal in dev mode)')
    }

    ws.onclose = () => {
      setConnected(false)
    }

    return () => {
      ws.close()
    }
  }, [fetchServices])

  return { services, loading, error, connected: connected || useMock, refetch: fetchServices }
}
