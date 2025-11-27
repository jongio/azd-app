import { useState, useEffect } from 'react'
import { useServices } from '@/hooks/useServices'
import { ServiceCard } from '@/components/ServiceCard'
import { ServiceTable } from '@/components/ServiceTable'
import { LogsMultiPaneView } from '@/components/LogsMultiPaneView'
import { Sidebar } from '@/components/Sidebar'
import { ThemeToggle } from '@/components/ThemeToggle'
import { ServiceStatusCard } from '@/components/ServiceStatusCard'
import type { Service } from '@/types'
import { AlertCircle, Search, Filter, Github, HelpCircle, Settings } from 'lucide-react'
import { useServiceErrors } from '@/hooks/useServiceErrors'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const [activeView, setActiveView] = useState<string>('resources')
  const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
    const saved = localStorage.getItem('dashboard-view-preference')
    return (saved === 'cards' || saved === 'table') ? saved : 'table'
  })
  const [isLogsFullscreen, setIsLogsFullscreen] = useState(false)
  const { services, loading, error } = useServices()
  const { hasActiveErrors } = useServiceErrors()

  // Helper function to scroll main element to top smoothly
  const scrollMainToTop = () => {
    const mainElement = document.querySelector('main')
    if (mainElement) {
      mainElement.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // Scroll to top when view changes (only for non-console views)
  useEffect(() => {
    if (activeView !== 'console') {
      scrollMainToTop()
    }
  }, [activeView])

  // Scroll to top when view mode changes (only for non-console views in resources)
  useEffect(() => {
    if (activeView === 'resources') {
      scrollMainToTop()
    }
  }, [viewMode, activeView])

  useEffect(() => {
    localStorage.setItem('dashboard-view-preference', viewMode)
  }, [viewMode])

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

  const renderContent = () => {
    if (activeView === 'resources') {
      return (
        <>
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-2xl font-semibold text-foreground">Resources</h1>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-foreground-tertiary" />
                <input
                  type="text"
                  placeholder="Filter..."
                  className="pl-9 pr-4 py-2 bg-input-background border border-input-border rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 w-64 text-input-foreground placeholder:text-input-placeholder"
                />
              </div>
              <button className="p-2 hover:bg-secondary rounded-md transition-colors">
                <Filter className="w-4 h-4 text-foreground-tertiary hover:text-foreground-secondary" />
              </button>
            </div>
          </div>

          <div className="flex items-center gap-1 mb-6 border-b border-border">
            <button
              onClick={() => setViewMode('table')}
              className={`px-4 py-2 text-sm font-medium transition-colors relative ${viewMode === 'table' ? 'text-foreground' : 'text-foreground-tertiary hover:text-foreground-secondary'}`}
            >
              Table
              {viewMode === 'table' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary"></div>
              )}
            </button>
            <button
              onClick={() => setViewMode('cards')}
              className={`px-4 py-2 text-sm font-medium transition-colors relative ${viewMode === 'cards' ? 'text-foreground' : 'text-foreground-tertiary hover:text-foreground-secondary'}`}
            >
              Grid
              {viewMode === 'cards' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary"></div>
              )}
            </button>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
            </div>
          ) : error ? (
            <div className="bg-destructive/10 border border-destructive/30 p-4 rounded-lg flex items-center gap-3">
              <AlertCircle className="w-5 h-5 text-destructive" />
              <div>
                <p className="text-destructive font-medium">Error Loading Services</p>
                <p className="text-destructive/80 text-sm mt-1">{error}</p>
              </div>
            </div>
          ) : services.length === 0 ? (
            <div className="bg-card border border-card-border p-12 rounded-lg text-center">
              <h3 className="text-xl font-semibold mb-2">No Services Running</h3>
              <p className="text-muted-foreground mb-4">
                Get started by launching your development services
              </p>
              <code className="bg-muted px-3 py-2 rounded text-primary inline-block text-sm">
                azd app run
              </code>
            </div>
          ) : (
            viewMode === 'table' ? (
              <ServiceTable services={services} onViewLogs={() => setActiveView('console')} />
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {services.map((service: Service) => (
                  <ServiceCard key={service.name} service={service} />
                ))}
              </div>
            )
          )}
        </>
      )
    }

    if (activeView === 'console') {
      return (
        <>
          {!isLogsFullscreen && (
            <div className="flex items-center justify-between mb-6">
              <h1 className="text-2xl font-semibold text-foreground">Console</h1>
            </div>
          )}
          <LogsMultiPaneView onFullscreenChange={setIsLogsFullscreen} />
        </>
      )
    }

    return (
      <div className="flex items-center justify-center py-20">
        <div className="text-center">
          <h2 className="text-xl font-semibold mb-2">Coming Soon</h2>
          <p className="text-muted-foreground">This view is not yet implemented</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-screen bg-background-tertiary">
      {!isLogsFullscreen && <Sidebar activeView={activeView} onViewChange={setActiveView} hasActiveErrors={hasActiveErrors} />}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!isLogsFullscreen && (
          <header className="h-14 border-b border-border flex items-center justify-between px-6 bg-background">
            <div className="flex items-center gap-4">
              <h1 className="text-sm font-semibold text-foreground">{projectName || 'testhost'}</h1>
            </div>
            <div className="flex items-center gap-2">
              <ServiceStatusCard 
                services={services} 
                hasActiveErrors={hasActiveErrors} 
                loading={loading}
                onClick={() => setActiveView('console')}
              />
              <div className="w-px h-5 bg-border mx-1" />
              <ThemeToggle />
              <button className="p-2 hover:bg-secondary rounded-md transition-colors">
                <Github className="w-4 h-4 text-foreground-secondary hover:text-foreground" />
              </button>
              <button className="p-2 hover:bg-secondary rounded-md transition-colors">
                <HelpCircle className="w-4 h-4 text-foreground-secondary hover:text-foreground" />
              </button>
              <button className="p-2 hover:bg-secondary rounded-md transition-colors">
                <Settings className="w-4 h-4 text-foreground-secondary hover:text-foreground" />
              </button>
            </div>
          </header>
        )}
        <main className={isLogsFullscreen ? "flex-1 overflow-auto bg-background-tertiary" : "flex-1 overflow-auto p-6 bg-background-tertiary"}>
          {renderContent()}
        </main>
      </div>
    </div>
  )
}

export default App
