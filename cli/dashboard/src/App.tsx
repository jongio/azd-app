import { useState, useEffect } from 'react'
import { useServices } from '@/hooks/useServices'
import { ServiceCard } from '@/components/ServiceCard'
import { ServiceTable } from '@/components/ServiceTable'
import { LogsView } from '@/components/LogsView'
import { Sidebar } from '@/components/Sidebar'
import type { Service } from '@/types'
import { AlertCircle, Search, Filter, Github, HelpCircle, Settings } from 'lucide-react'

function App() {
  const [projectName, setProjectName] = useState<string>('')
  const [activeView, setActiveView] = useState<string>('resources')
  const [viewMode, setViewMode] = useState<'cards' | 'table'>(() => {
    const saved = localStorage.getItem('dashboard-view-preference')
    return (saved === 'cards' || saved === 'table') ? saved : 'table'
  })
  const { services, loading, error } = useServices()

  // Scroll to top when view changes
  useEffect(() => {
    const mainElement = document.querySelector('main')
    if (mainElement) {
      mainElement.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }, [activeView])

  // Scroll to top when view mode changes
  useEffect(() => {
    const mainElement = document.querySelector('main')
    if (mainElement) {
      mainElement.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }, [viewMode])

  useEffect(() => {
    localStorage.setItem('dashboard-view-preference', viewMode)
  }, [viewMode])

  useEffect(() => {
    fetch('/api/project')
      .then(res => res.json())
      .then(data => {
        setProjectName(data.name)
        document.title = `${data.name}`
      })
      .catch(err => console.error('Failed to fetch project name:', err))
  }, [])

  const renderContent = () => {
    if (activeView === 'resources') {
      return (
        <>
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-2xl font-semibold text-foreground">Resources</h1>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
                <input
                  type="text"
                  placeholder="Filter..."
                  className="pl-9 pr-4 py-2 bg-[#0d0d0d] border border-[#2a2a2a] rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 w-64 text-foreground"
                />
              </div>
              <button className="p-2 hover:bg-white/5 rounded-md transition-colors">
                <Filter className="w-4 h-4 text-gray-500 hover:text-gray-300" />
              </button>
            </div>
          </div>

          <div className="flex items-center gap-1 mb-6 border-b border-[#2a2a2a]">
            <button
              onClick={() => setViewMode('table')}
              className={`px-4 py-2 text-sm font-medium transition-colors relative ${viewMode === 'table' ? 'text-foreground' : 'text-gray-500 hover:text-gray-300'}`}
            >
              Table
              {viewMode === 'table' && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary"></div>
              )}
            </button>
            <button
              onClick={() => setViewMode('cards')}
              className={`px-4 py-2 text-sm font-medium transition-colors relative ${viewMode === 'cards' ? 'text-foreground' : 'text-gray-500 hover:text-gray-300'}`}
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
            <div className="bg-[#1a1a1a] border border-white/10 p-12 rounded-lg text-center">
              <h3 className="text-xl font-semibold mb-2">No Services Running</h3>
              <p className="text-muted-foreground mb-4">
                Get started by launching your development services
              </p>
              <code className="bg-black/30 px-3 py-2 rounded text-primary inline-block text-sm">
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
          <div className="flex items-center justify-between mb-6">
            <h1 className="text-2xl font-semibold text-foreground">Console</h1>
          </div>
          <LogsView />
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
    <div className="flex h-screen bg-[#1a1a1a]">
      <Sidebar activeView={activeView} onViewChange={setActiveView} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="h-14 border-b border-[#2a2a2a] flex items-center justify-between px-6 bg-[#0d0d0d]">
          <div className="flex items-center gap-2">
            <h1 className="text-sm font-semibold">{projectName || 'testhost'}</h1>
          </div>
          <div className="flex items-center gap-2">
            <button className="p-2 hover:bg-white/5 rounded-md transition-colors">
              <Github className="w-4 h-4 text-gray-400 hover:text-gray-300" />
            </button>
            <button className="p-2 hover:bg-white/5 rounded-md transition-colors">
              <HelpCircle className="w-4 h-4 text-gray-400 hover:text-gray-300" />
            </button>
            <button className="p-2 hover:bg-white/5 rounded-md transition-colors">
              <Settings className="w-4 h-4 text-gray-400 hover:text-gray-300" />
            </button>
          </div>
        </header>
        <main className="flex-1 overflow-auto p-6 bg-[#1a1a1a]">
          {renderContent()}
        </main>
      </div>
    </div>
  )
}

export default App
