import { useState, useEffect, useCallback } from 'react'
import { useServices } from '@/hooks/useServices'
import { useHealthStream } from '@/hooks/useHealthStream'
import { ServiceCard } from '@/components/ServiceCard'
import { ServiceTable } from '@/components/ServiceTable'
import { LogsMultiPaneView } from '@/components/LogsMultiPaneView'
import { Sidebar } from '@/components/Sidebar'
import { ThemeToggle } from '@/components/ThemeToggle'
import { ServiceStatusCard } from '@/components/ServiceStatusCard'
import { EnvironmentPanel } from '@/components/EnvironmentPanel'
import { PerformanceMetrics } from '@/components/views/PerformanceMetrics'
// ServiceDependencies feature removed
import { ServiceDetailPanel } from '@/components/panels/ServiceDetailPanel'
import { KeyboardShortcuts } from '@/components/modals/KeyboardShortcuts'
import { shouldHandleShortcut, keyToView } from '@/lib/shortcuts-utils'
import type { Service, HealthCheckResult } from '@/types'
import { AlertCircle, Search, Filter, Github, HelpCircle, Settings, RefreshCw, Wifi, WifiOff } from 'lucide-react'
import { useServiceErrors } from '@/hooks/useServiceErrors'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const [activeView, setActiveView] = useState<string>('resources')
  const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
    const saved = localStorage.getItem('dashboard-view-preference')
    return (saved === 'cards' || saved === 'table') ? saved : 'table'
  })
  const [isLogsFullscreen, setIsLogsFullscreen] = useState(false)
  const [selectedService, setSelectedService] = useState<Service | null>(null)
  const [isDetailPanelOpen, setIsDetailPanelOpen] = useState(false)
  const [isShortcutsModalOpen, setIsShortcutsModalOpen] = useState(false)
  const { services, loading, error } = useServices()
  const { hasActiveErrors } = useServiceErrors()
  
  // Real-time health monitoring
  const { 
    healthReport, 
    summary: healthSummary, 
    connected: healthConnected,
    error: healthError,
    reconnect: healthReconnect,
    getServiceHealth 
  } = useHealthStream()

  // Helper to get health status for a specific service
  const getServiceHealthStatus = useCallback((serviceName: string): HealthCheckResult | undefined => {
    return getServiceHealth(serviceName)
  }, [getServiceHealth])

  // Handle service click to open detail panel
  const handleServiceClick = useCallback((service: Service) => {
    setSelectedService(service)
    setIsDetailPanelOpen(true)
  }, [])

  // Handle detail panel close
  const handleDetailPanelClose = useCallback(() => {
    setIsDetailPanelOpen(false)
  }, [])

  // Handle keyboard shortcuts modal
  const handleShortcutsModalOpen = useCallback(() => {
    setIsShortcutsModalOpen(true)
  }, [])

  const handleShortcutsModalClose = useCallback(() => {
    setIsShortcutsModalOpen(false)
  }, [])

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Don't handle shortcuts when in input/textarea
      if (!shouldHandleShortcut(event)) return
      
      // Don't handle shortcuts when modal is open (except Escape)
      if (isShortcutsModalOpen && event.key !== 'Escape') return
      
      const key = event.key
      
      // ? - Show keyboard shortcuts
      if (key === '?') {
        event.preventDefault()
        setIsShortcutsModalOpen(true)
        return
      }
      
      // Navigation shortcuts (1-6)
      const view = keyToView[key as keyof typeof keyToView]
      if (view) {
        event.preventDefault()
        setActiveView(view)
        return
      }
      
      // T - Toggle table/grid view
      if (key.toLowerCase() === 't' && activeView === 'resources') {
        event.preventDefault()
        setViewMode(prev => prev === 'table' ? 'cards' : 'table')
        return
      }
      
      // R - Refresh (already handled by browser, but could be used for manual refresh)
      // C - Clear console logs (would need to be connected to LogsMultiPaneView)
      // E - Export logs (would need to be connected to LogsMultiPaneView)
      // / or Ctrl+F - Focus search (would need to connect to search input)
    }
    
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isShortcutsModalOpen, activeView])

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
            <h2 className="text-xl font-semibold text-foreground">Resources</h2>
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
              <ServiceTable 
                services={services} 
                onViewLogs={() => setActiveView('console')}
                onServiceClick={handleServiceClick}
                healthReport={healthReport}
              />
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {services.map((service: Service) => (
                  <ServiceCard 
                    key={service.name} 
                    service={service}
                    healthStatus={getServiceHealthStatus(service.name)}
                    onClick={() => handleServiceClick(service)}
                  />
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
              <h2 className="text-xl font-semibold text-foreground">Console</h2>
            </div>
          )}
          <LogsMultiPaneView 
            onFullscreenChange={setIsLogsFullscreen}
            healthReport={healthReport}
            onServiceClick={handleServiceClick}
          />
        </>
      )
    }

    if (activeView === 'environment') {
      return (
        <>
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-foreground">Environment</h2>
          </div>
          <EnvironmentPanel services={services} />
        </>
      )
    }

    if (activeView === 'metrics') {
      return (
        <>
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-foreground">Metrics</h2>
          </div>
          <PerformanceMetrics services={services} healthReport={healthReport} />
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
    <div className="flex h-screen bg-background-tertiary relative">
      {/* Connection Lost Overlay */}
      {healthError && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background backdrop-blur-sm transition-opacity duration-300">
          <div className="flex flex-col items-center gap-6 p-8 rounded-2xl bg-card border border-border shadow-2xl max-w-md mx-4 animate-fade-in-up">
            {/* Animated Icon with contextual animation */}
            <div className="relative w-32 h-32 flex items-center justify-center">
              {healthError.includes('Failed to connect') ? (
                <>
                  {/* Disconnected state - subtle pulsing rings */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-28 h-28 rounded-full border-2 border-red-500/20 animate-pulse" />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-20 h-20 rounded-full bg-red-500/10 animate-pulse" />
                  </div>
                  <div className="relative z-10 w-16 h-16 rounded-full bg-red-500/15 flex items-center justify-center shadow-lg">
                    <WifiOff className="w-8 h-8 text-red-500" />
                  </div>
                </>
              ) : (
                <>
                  {/* Reconnecting state - smooth expanding waves */}
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-32 h-32 rounded-full bg-amber-500/15 animate-signal-wave" />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-28 h-28 rounded-full bg-amber-500/20 animate-signal-wave-delayed" />
                  </div>
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="w-24 h-24 rounded-full bg-amber-500/25 animate-signal-wave-delayed-2" />
                  </div>
                  <div className="relative z-10 w-16 h-16 rounded-full bg-amber-500/20 flex items-center justify-center animate-breathe animate-glow-pulse">
                    <Wifi className="w-8 h-8 text-amber-500" />
                  </div>
                </>
              )}
            </div>
            
            {/* Message */}
            <div className="text-center">
              <h2 className="text-xl font-semibold text-foreground mb-2">
                {healthError.includes('Failed to connect') ? 'Connection Lost' : 'Reconnecting...'}
              </h2>
              <p className="text-sm text-muted-foreground">{healthError}</p>
            </div>
            
            {/* Retry Button */}
            {healthError.includes('Failed to connect') && (
              <button 
                onClick={healthReconnect}
                className="flex items-center gap-2 px-6 py-3 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg font-medium transition-all shadow-lg hover:scale-105 active:scale-95"
              >
                <RefreshCw className="w-4 h-4" />
                Retry Connection
              </button>
            )}
          </div>
        </div>
      )}
      
      {!isLogsFullscreen && <Sidebar activeView={activeView} onViewChange={setActiveView} hasActiveErrors={hasActiveErrors} healthSummary={healthSummary} services={services} />}
      <div className="flex-1 flex flex-col overflow-hidden">
        {!isLogsFullscreen && (
          <header className="h-16 border-b border-border flex items-center justify-between px-6 bg-background">
            <div className="flex items-center gap-4">
              <h1 className="text-2xl font-bold text-foreground">{projectName || 'testhost'}</h1>
            </div>
            <div className="flex items-center gap-2">
              <ServiceStatusCard 
                services={services} 
                hasActiveErrors={hasActiveErrors} 
                loading={loading}
                onClick={() => setActiveView('console')}
                healthSummary={healthSummary}
                healthConnected={healthConnected}
              />
              <div className="w-px h-5 bg-border mx-1" />
              <ThemeToggle />
              <button className="p-2 hover:bg-secondary rounded-md transition-colors">
                <Github className="w-4 h-4 text-foreground-secondary hover:text-foreground" />
              </button>
              <button 
                onClick={handleShortcutsModalOpen}
                className="p-2 hover:bg-secondary rounded-md transition-colors"
                aria-label="Keyboard shortcuts"
                title="Keyboard shortcuts (?)"
              >
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
      
      {/* Service Detail Panel */}
      <ServiceDetailPanel
        service={selectedService}
        isOpen={isDetailPanelOpen}
        onClose={handleDetailPanelClose}
        healthStatus={selectedService ? getServiceHealthStatus(selectedService.name) : undefined}
      />
      
      {/* Keyboard Shortcuts Modal */}
      <KeyboardShortcuts
        isOpen={isShortcutsModalOpen}
        onClose={handleShortcutsModalClose}
      />
    </div>
  )
}

export default App
