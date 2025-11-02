import { useState, useEffect } from 'react'
import { useServices } from '@/hooks/useServices'
import { ServiceCard } from '@/components/ServiceCard'
import { LogsView } from '@/components/LogsView'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { AlertCircle, Wifi, WifiOff, Zap, Activity } from 'lucide-react'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const [activeTab, setActiveTab] = useState<string>('services')
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

  const runningServices = services.filter(s => s.status === 'running' || s.status === 'ready')
  const healthyServices = services.filter(s => s.health === 'healthy')

  return (
    <div className="min-h-screen bg-background">
      {/* Animated Header with Gradient */}
      <header className="border-b border-white/10 glass sticky top-0 z-50 backdrop-blur-xl">
        <div className="container mx-auto px-6 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="relative">
                <div className="absolute inset-0 bg-linear-to-r from-primary to-accent blur-xl opacity-50 animate-pulse"></div>
                <div className="relative bg-linear-to-br from-primary to-accent p-3 rounded-2xl shadow-lg">
                  <Zap className="w-6 h-6 text-white" />
                </div>
              </div>
              <div>
                <h1 className="text-4xl font-bold gradient-text">
                  {projectName || 'AZD App'}
                </h1>
                <p className="text-muted-foreground mt-1 text-sm flex items-center gap-2">
                  <Activity className="w-4 h-4" />
                  Development Environment Dashboard
                </p>
              </div>
            </div>
            
            <div className="flex items-center gap-6">
              {/* Stats */}
              <div className="hidden md:flex items-center gap-6 text-sm">
                <div className="text-center">
                  <div className="text-2xl font-bold text-primary">{services.length}</div>
                  <div className="text-muted-foreground text-xs">Total</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-success">{runningServices.length}</div>
                  <div className="text-muted-foreground text-xs">Running</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-accent">{healthyServices.length}</div>
                  <div className="text-muted-foreground text-xs">Healthy</div>
                </div>
              </div>
              
              {/* Connection Status */}
              <div className="flex items-center gap-2 px-4 py-2 rounded-full glass">
                {connected ? (
                  <>
                    <div className="relative">
                      <Wifi className="w-4 h-4 text-success" />
                      <span className="absolute -top-1 -right-1 w-2 h-2 bg-success rounded-full animate-ping"></span>
                      <span className="absolute -top-1 -right-1 w-2 h-2 bg-success rounded-full"></span>
                    </div>
                    <span className="text-success font-medium text-sm">Live</span>
                  </>
                ) : (
                  <>
                    <WifiOff className="w-4 h-4 text-destructive" />
                    <span className="text-destructive font-medium text-sm">Offline</span>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-6 py-10">
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="glass border-white/10 mb-8 p-1">
            <TabsTrigger value="services" className="data-[state=active]:bg-linear-to-r data-[state=active]:from-primary data-[state=active]:to-accent data-[state=active]:text-white transition-all-smooth">
              Services
            </TabsTrigger>
            <TabsTrigger value="logs" className="data-[state=active]:bg-linear-to-r data-[state=active]:from-primary data-[state=active]:to-accent data-[state=active]:text-white transition-all-smooth">
              Logs
            </TabsTrigger>
          </TabsList>

          <TabsContent value="services" className="mt-0">
            {loading ? (
              <div className="flex flex-col items-center justify-center py-20">
                <div className="relative">
                  <div className="w-16 h-16 border-4 border-primary/20 rounded-full"></div>
                  <div className="absolute top-0 left-0 w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin"></div>
                </div>
                <p className="text-muted-foreground mt-4">Loading services...</p>
              </div>
            ) : error ? (
              <div className="glass border-destructive/50 p-6 rounded-2xl flex items-center gap-3">
                <AlertCircle className="w-6 h-6 text-destructive" />
                <div>
                  <p className="text-destructive font-semibold">Error Loading Services</p>
                  <p className="text-destructive/80 text-sm mt-1">{error}</p>
                </div>
              </div>
            ) : services.length === 0 ? (
              <div className="glass p-12 rounded-2xl text-center">
                <div className="max-w-md mx-auto">
                  <div className="bg-linear-to-br from-primary/20 to-accent/20 w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-6">
                    <Activity className="w-10 h-10 text-primary" />
                  </div>
                  <h3 className="text-2xl font-bold mb-3">No Services Running</h3>
                  <p className="text-muted-foreground mb-6">
                    Get started by launching your development services
                  </p>
                  <code className="glass px-4 py-3 rounded-lg text-primary inline-block border border-primary/30">
                    azd app run
                  </code>
                </div>
              </div>
            ) : (
              <div className="space-y-10">
                {Object.entries(servicesByProject).map(([projectDir, projectServices]) => (
                  <div key={projectDir} className="space-y-4">
                    <div className="flex items-center gap-3">
                      <div className="h-px flex-1 bg-linear-to-r from-transparent via-primary/50 to-transparent"></div>
                      <h2 className="text-lg font-semibold text-muted-foreground px-4">
                        {projectDir}
                      </h2>
                      <div className="h-px flex-1 bg-linear-to-r from-transparent via-primary/50 to-transparent"></div>
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                      {projectServices.map((service) => (
                        <ServiceCard key={`${service.projectDir}:${service.name}`} service={service} />
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </TabsContent>

          <TabsContent value="logs">
            <LogsView />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}

export default App
