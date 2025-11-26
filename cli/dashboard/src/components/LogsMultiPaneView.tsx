import { useState, useEffect, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { LayoutGrid, List, Settings, Pause, Play, Download, Maximize2, Minimize2, Search, Trash2, ChevronsDown } from 'lucide-react'
import { LogsPane, type LogEntry } from './LogsPane'
import { LogsPaneGrid } from './LogsPaneGrid'
import { SettingsModal } from './SettingsModal'
import { LogsView } from './LogsView'
import { usePreferences } from '@/hooks/usePreferences'
import { useLogPatterns } from '@/hooks/useLogPatterns'
import { useToast } from '@/components/ui/toast'
import type { Service } from '@/types'

interface LogsMultiPaneViewProps {
  onFullscreenChange?: (isFullscreen: boolean) => void
}

export function LogsMultiPaneView({ onFullscreenChange }: LogsMultiPaneViewProps = {}) {
  const [services, setServices] = useState<Service[]>([])
  const [selectedServices, setSelectedServices] = useState<Set<string>>(new Set())
  const [isPaused, setIsPaused] = useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = useState(true)
  const [clearAllTrigger, setClearAllTrigger] = useState(0)
  const [levelFilter, setLevelFilter] = useState<Set<'info' | 'warning' | 'error'>>(new Set(['info', 'warning', 'error']))
  const [collapsedPanes, setCollapsedPanes] = useState<Record<string, boolean>>(() => {
    try {
      const saved = localStorage.getItem('logs-pane-collapsed-states')
      return saved ? JSON.parse(saved) : {}
    } catch {
      return {}
    }
  })
  
  const { preferences, updateUI } = usePreferences()
  const { patterns } = useLogPatterns()
  const { showToast, ToastContainer } = useToast()

  // Persist collapsed state
  useEffect(() => {
    localStorage.setItem('logs-pane-collapsed-states', JSON.stringify(collapsedPanes))
  }, [collapsedPanes])

  const togglePaneCollapse = useCallback((serviceName: string) => {
    setCollapsedPanes(prev => ({
      ...prev,
      [serviceName]: !prev[serviceName]
    }))
  }, [])

  // Notify parent component when fullscreen changes
  useEffect(() => {
    onFullscreenChange?.(isFullscreen)
  }, [isFullscreen, onFullscreenChange])

  const viewMode = preferences.ui.viewMode
  const gridColumns = preferences.ui.gridColumns

  // Fetch services
  useEffect(() => {
    const fetchServices = async () => {
      try {
        const res = await fetch('/api/services')
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)
        const data = await res.json() as Service[]
        setServices(data)
        
        // Initialize selected services if empty
        if (selectedServices.size === 0) {
          setSelectedServices(new Set(data.map(s => s.name)))
        }
      } catch (err) {
        console.error('Failed to fetch services:', err)
      }
    }
    void fetchServices()
  }, [selectedServices.size])

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Space: toggle pause
      if (e.code === 'Space' && !e.ctrlKey && !e.shiftKey && !e.altKey) {
        const target = e.target as HTMLElement
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault()
          setIsPaused(prev => !prev)
        }
      }
      
      // Ctrl+Shift+L: toggle view mode
      if (e.ctrlKey && e.shiftKey && e.code === 'KeyL') {
        e.preventDefault()
        updateUI({ viewMode: viewMode === 'grid' ? 'unified' : 'grid' })
      }
      
      // F11 or Ctrl+Shift+F: toggle fullscreen
      if (e.key === 'F11' || (e.ctrlKey && e.shiftKey && e.code === 'KeyF')) {
        e.preventDefault()
        setIsFullscreen(prev => !prev)
      }
      
      // Escape: exit fullscreen
      if (e.key === 'Escape' && isFullscreen) {
        setIsFullscreen(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [viewMode, updateUI, isFullscreen])

  const handleToggleService = (serviceName: string) => {
    setSelectedServices(prev => {
      const next = new Set(prev)
      if (next.has(serviceName)) {
        next.delete(serviceName)
      } else {
        next.add(serviceName)
      }
      return next
    })
  }

  const toggleLevelFilter = (level: 'info' | 'warning' | 'error') => {
    setLevelFilter(prev => {
      const newSet = new Set(prev)
      if (newSet.has(level)) {
        newSet.delete(level)
      } else {
        newSet.add(level)
      }
      return newSet
    })
  }

  const handleCopyPane = useCallback((logs: LogEntry[]) => {
    const format = preferences.copy.defaultFormat
    let content = ''

    switch (format) {
      case 'json':
        content = JSON.stringify(logs, null, 2)
        break
      case 'csv':
        content = 'Service,Timestamp,Level,Message\n' +
          logs.map(log => `"${log.service}","${log.timestamp}",${log.level},"${log.message.replace(/"/g, '""')}"`).join('\n')
        break
      case 'markdown':
        content = logs.map(log => `**[${log.timestamp}]** \`${log.service}\` ${log.message}`).join('\n\n')
        break
      default: // plaintext
        content = logs.map(log => `[${log.timestamp}] [${log.service}] ${log.message}`).join('\n')
    }

    void navigator.clipboard.writeText(content)
    showToast(`Copied ${logs.length} lines to clipboard`, 'success')
  }, [preferences.copy.defaultFormat, showToast])

  const handleExportAll = useCallback(() => {
    // This would need to aggregate all pane logs
    showToast('Export all feature coming soon', 'info')
  }, [showToast])

  const handleClearAll = useCallback(() => {
    setClearAllTrigger(prev => prev + 1)
    showToast('All logs cleared', 'success')
  }, [showToast])

  const handleToggleAutoScroll = useCallback(() => {
    setAutoScrollEnabled(prev => !prev)
  }, [])

  // Sort services alphabetically (case-insensitive)
  const selectedServicesList = Array.from(selectedServices).sort((a, b) => 
    a.toLowerCase().localeCompare(b.toLowerCase())
  )

  // Responsive column calculation
  const effectiveColumns = window.innerWidth < 600 ? 1 : gridColumns

  return (
    <div className={isFullscreen ? "fixed inset-0 z-50 flex flex-col overflow-hidden bg-background" : "h-screen flex flex-col overflow-hidden"}>
      <ToastContainer />
      
      {/* Global Toolbar */}
      <div className="flex flex-wrap gap-4 items-center justify-between p-4 bg-card border-b flex-shrink-0">
        <div className="flex gap-2 items-center flex-wrap">
          {/* View Mode Toggle */}
          {!isFullscreen && (
            <div className="flex gap-1 border rounded-lg p-1">
              <Button
                variant={viewMode === 'grid' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => updateUI({ viewMode: 'grid' })}
                title="Grid View (Ctrl+Shift+L)"
              >
                <LayoutGrid className="w-4 h-4 mr-2" />
                Grid
              </Button>
              <Button
                variant={viewMode === 'unified' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => updateUI({ viewMode: 'unified' })}
                title="Unified View (Ctrl+Shift+L)"
              >
                <List className="w-4 h-4 mr-2" />
                Unified
              </Button>
            </div>
          )}

          {/* Global Search */}
          <div className="relative min-w-[200px]">
            <Search className="absolute left-2 top-2.5 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search all logs..."
              value={globalSearchTerm}
              onChange={(e) => setGlobalSearchTerm(e.target.value)}
              className="pl-8 h-9"
            />
          </div>

          {/* Pause/Resume All */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsPaused(!isPaused)}
            title={`${isPaused ? 'Resume' : 'Pause'} all logs (Space)`}
          >
            {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          </Button>

          {/* Stop Auto-Scroll All */}
          <Button
            variant={autoScrollEnabled ? "default" : "outline"}
            size="sm"
            onClick={handleToggleAutoScroll}
            title={autoScrollEnabled ? "Stop auto-scroll (all panes)" : "Enable auto-scroll (all panes)"}
          >
            <ChevronsDown className="w-4 h-4" />
          </Button>

          {/* Clear All Logs */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleClearAll}
            title="Clear all logs"
          >
            <Trash2 className="w-4 h-4" />
          </Button>

          {/* Settings */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsSettingsOpen(true)}
            title="Settings"
          >
            <Settings className="w-4 h-4" />
          </Button>

          {/* Export All */}
          {!isFullscreen && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleExportAll}
              title="Export All"
            >
              <Download className="w-4 h-4" />
            </Button>
          )}

          {/* Fullscreen Toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsFullscreen(!isFullscreen)}
            title={`${isFullscreen ? 'Exit Fullscreen (Esc)' : 'Enter Fullscreen (F11)'}`}
          >
            {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
          </Button>
        </div>

        <div className="flex items-center gap-4">
          {isPaused && (
            <div className="text-sm text-yellow-600 font-medium">
              ⏸ Paused
            </div>
          )}
          {!autoScrollEnabled && (
            <div className="text-sm text-blue-600 font-medium">
              🛑 Auto-scroll stopped
            </div>
          )}
        </div>
      </div>

      {/* Filters - hidden in fullscreen mode */}
      {!isFullscreen && (
        <div className="p-4 bg-card border-b flex-shrink-0">
          <div className="flex flex-wrap gap-8">
            {/* Services Filter */}
            <div>
              <h3 className="font-medium mb-3">Services</h3>
              <div className="flex flex-wrap gap-2">
                {[...services].sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase())).map(service => (
                  <label key={service.name} className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedServices.has(service.name)}
                      onChange={() => handleToggleService(service.name)}
                      className="w-4 h-4"
                    />
                    <span className="text-sm">{service.name}</span>
                  </label>
                ))}
              </div>
            </div>
            {/* Level Filter */}
            <div>
              <h3 className="font-medium mb-3">Log Levels</h3>
              <div className="flex flex-wrap gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={levelFilter.has('info')}
                    onChange={() => toggleLevelFilter('info')}
                    className="w-4 h-4 accent-blue-500"
                  />
                  <span className="text-sm text-blue-500">Info</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={levelFilter.has('warning')}
                    onChange={() => toggleLevelFilter('warning')}
                  className="w-4 h-4 accent-yellow-500"
                />
                <span className="text-sm text-yellow-500">Warning</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={levelFilter.has('error')}
                  onChange={() => toggleLevelFilter('error')}
                  className="w-4 h-4 accent-red-500"
                />
                <span className="text-sm text-red-500">Error</span>
              </label>
            </div>
          </div>
        </div>
      </div>
      )}

      {/* View Content */}
      <div className="flex-1 overflow-hidden">
        {viewMode === 'grid' ? (
          <LogsPaneGrid columns={effectiveColumns} collapsedPanes={collapsedPanes}>
            {selectedServicesList.map(serviceName => (
              <LogsPane
                key={serviceName}
                serviceName={serviceName}
                patterns={patterns}
                onCopy={handleCopyPane}
                isPaused={isPaused}
                globalSearchTerm={globalSearchTerm}
                autoScrollEnabled={autoScrollEnabled}
                clearAllTrigger={clearAllTrigger}
                levelFilter={levelFilter}
                isCollapsed={collapsedPanes[serviceName] ?? false}
                onToggleCollapse={() => togglePaneCollapse(serviceName)}
              />
            ))}
          </LogsPaneGrid>
        ) : (
          <LogsView selectedServices={selectedServices} levelFilter={levelFilter} />
        )}
      </div>

      {/* Settings Modal */}
      <SettingsModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />
    </div>
  )
}
