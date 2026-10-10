import { useEffect, useMemo } from 'react'
import { useServicesContext } from '@/contexts/ServicesContext'
import { useHealthStream } from '@/hooks/useHealthStream'
import { useCodespaceEnv } from '@/hooks/useCodespaceEnv'
import { useProject } from '@/hooks/useProject'
import { App as DashboardApp } from '@/components/App'
import { BackendConnectionContext } from '@/hooks/useBackendConnection'
import type { HealthCheckResult } from '@/types'

function App() {
  const { name: projectName } = useProject()
  const { services } = useServicesContext()
  
  // Environment info (includes Azure environment name)
  const { environmentName } = useCodespaceEnv()
  
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

  // Update document title whenever the project name resolves. Keeping
  // this side-effect in App.tsx (vs. inside useProject) preserves the
  // hook's purity: useProject is reused-safe and shouldn't mutate the
  // browser document just by being called.
  useEffect(() => {
    if (projectName) {
      document.title = projectName
    }
  }, [projectName])

  // Provide global connection state to all components
  const connectionState = useMemo(() => ({
    connected: healthConnected,
    error: healthError,
  }), [healthConnected, healthError])

  return (
    <BackendConnectionContext.Provider value={connectionState}>
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
        environmentName={environmentName}
      />
    </BackendConnectionContext.Provider>
  )
}

export default App
