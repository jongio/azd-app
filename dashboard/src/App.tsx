import { useState, useEffect } from 'react'
import { useServices } from '@/hooks/useServices'
import { ServiceCard } from '@/components/ServiceCard'
import { AlertCircle, Wifi, WifiOff } from 'lucide-react'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const { services, loading, error, connected } = useServices()

  // Fetch project name from backend
  useEffect(() => {
    fetch('/api/project')
      .then(res => res.json())
      .then(data => setProjectName(data.name))
      .catch(err => console.error('Failed to fetch project name:', err))
  }, [])

  // Group services by project
  const servicesByProject = services.reduce((acc, service) => {
    if (!acc[service.projectDir]) {
      acc[service.projectDir] = []
    }
    acc[service.projectDir].push(service)
    return acc
  }, {} as Record<string, typeof services>)

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b bg-card">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold">
                {projectName || 'AZD App'} Dashboard
              </h1>
              <p className="text-muted-foreground mt-1">
                Monitor your running services
              </p>
            </div>
            <div className="flex items-center gap-2 text-sm">
              {connected ? (
                <>
                  <div className="relative">
                    <Wifi className="w-4 h-4 text-green-500" />
                    <span className="absolute top-0 right-0 w-2 h-2 bg-green-500 rounded-full animate-ping"></span>
                    <span className="absolute top-0 right-0 w-2 h-2 bg-green-500 rounded-full"></span>
                  </div>
                  <span className="text-green-600 font-medium">Connected</span>
                </>
              ) : (
                <>
                  <WifiOff className="w-4 h-4 text-red-500" />
                  <span className="text-red-600 font-medium">Disconnected</span>
                </>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
          </div>
        ) : error ? (
          <div className="flex items-center gap-2 p-4 bg-destructive/10 border border-destructive rounded-lg">
            <AlertCircle className="w-5 h-5 text-destructive" />
            <p className="text-destructive">{error}</p>
          </div>
        ) : services.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-muted-foreground text-lg">
              No services are currently running
            </p>
            <p className="text-muted-foreground text-sm mt-2">
              Run <code className="bg-muted px-2 py-1 rounded">azd app run</code> to start services
            </p>
          </div>
        ) : (
          <div className="space-y-8">
            {Object.entries(servicesByProject).map(([projectDir, projectServices]) => (
              <div key={projectDir}>
                <h2 className="text-xl font-semibold mb-4">
                  {projectDir}
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {projectServices.map((service) => (
                    <ServiceCard key={`${service.projectDir}:${service.name}`} service={service} />
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}

export default App
