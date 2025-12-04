import { useState, useEffect, useMemo } from 'react'
import { useServicesContext } from '@/contexts/ServicesContext'
import { useHealthStream } from '@/hooks/useHealthStream'
import { App as DashboardApp } from '@/components/App'
import type { HealthCheckResult } from '@/types'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const { services } = useServicesContext()
  
  // Real-time health monitoring
  const { 
    healthReport, 
    summary: healthSummary, 
    connected: healthConnected,
    error: healthError,
    reconnect: healthReconnect,
    getServiceHealth 
  } = useHealthStream()

  // Build health map for modern components
  const healthMap = useMemo(() => {
    const map = new Map<string, HealthCheckResult>()
    for (const service of services) {
      const health = getServiceHealth(service.name)
      if (health) {
        map.set(service.name, health)
      }
    }
    return map
  }, [services, getServiceHealth])

  useEffect(() => {
    const fetchProjectName = async () => {
      try {
        const res = await fetch('/api/project')
        if (!res.ok) {
          throw new Error(`HTTP error! status: ${res.status}`)
        }
        const data = await res.json() as { name: string }
        setProjectName(data.name)
        document.title = `${data.name}`
      } catch (err) {
        console.error('Failed to fetch project name:', err)
      }
    }
    void fetchProjectName()
  }, [])

  return (
    <DashboardApp
      projectName={projectName || 'Project'}
      services={services}
      connected={healthConnected}
      healthSummary={healthSummary ?? { 
        total: 0, 
        healthy: 0, 
        degraded: 0, 
        unhealthy: 0, 
        starting: 0,
        stopped: 0,
        unknown: 0,
        overall: 'unknown' as const 
      }}
      healthReport={healthReport}
      healthMap={healthMap}
      healthError={healthError}
      healthReconnect={healthReconnect}
    />
  )
}

export default App
