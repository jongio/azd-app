import { useState, useEffect, useCallback } from 'react'
import { LogsPane, type LogEntry } from './LogsPane'
import { LogsPaneGrid } from './LogsPaneGrid'
import { LogsToolbar } from './LogsToolbar'
import { SettingsModal } from './SettingsModal'
import { LogsView } from './LogsView'
import { usePreferences } from '@/hooks/usePreferences'
import { useToast } from '@/components/ui/toast'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { getStorageItem, setStorageItem, isRecordOfBooleans, createStringArrayValidator } from '@/lib/storage-utils'
import type { Service, HealthReportEvent, HealthStatus } from '@/types'

// Storage keys
const HEALTH_FILTER_KEY = 'logs-health-status-filter'
const COLLAPSED_PANES_KEY = 'logs-pane-collapsed-states'

// Valid health statuses
const VALID_HEALTH_STATUSES = ['healthy', 'degraded', 'unhealthy', 'starting', 'unknown'] as const satisfies readonly HealthStatus[]

// Validator for health filter array
const isValidHealthFilterArray = createStringArrayValidator(VALID_HEALTH_STATUSES)

interface LogsMultiPaneViewProps {
  onFullscreenChange?: (isFullscreen: boolean) => void
  healthReport?: HealthReportEvent | null
  onServiceClick?: (service: Service) => void  // Callback to open service details panel
}

export function LogsMultiPaneView({ onFullscreenChange, healthReport, onServiceClick }: LogsMultiPaneViewProps = {}) {
  const [services, setServices] = useState<Service[]>([])
  const [selectedServices, setSelectedServices] = useState<Set<string>>(new Set())
  const [isPaused, setIsPaused] = useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = useState(true)
  const [clearAllTrigger, setClearAllTrigger] = useState(0)
  const [levelFilter, setLevelFilter] = useState<Set<'info' | 'warning' | 'error'>>(new Set(['info', 'warning', 'error']))
  const [healthFilter, setHealthFilter] = useState<Set<HealthStatus>>(() => {
    const stored = getStorageItem<HealthStatus[]>(HEALTH_FILTER_KEY, [...VALID_HEALTH_STATUSES], isValidHealthFilterArray)
    return new Set(stored.length > 0 ? stored : VALID_HEALTH_STATUSES)
  })
  const [collapsedPanes, setCollapsedPanes] = useState<Record<string, boolean>>(() => {
    return getStorageItem(COLLAPSED_PANES_KEY, {}, isRecordOfBooleans)
  })
  
  const { preferences, updateUI } = usePreferences()
  const { showToast, ToastContainer } = useToast()
  const { 
    startAll, 
    stopAll, 
    restartAll, 
    isBulkOperationInProgress, 
    bulkOperation,
    lastResult,
    error: operationError 
  } = useServiceOperations()

  // Show toast for bulk operation results
  useEffect(() => {
    if (lastResult) {
      if (lastResult.success) {
        showToast(lastResult.message, 'success')
      } else {
        showToast(lastResult.message, 'error')
      }
    }
  }, [lastResult, showToast])

  // Show toast for operation errors
  useEffect(() => {
    if (operationError) {
      showToast(operationError, 'error')
    }
  }, [operationError, showToast])

  // Persist collapsed state
  useEffect(() => {
    setStorageItem(COLLAPSED_PANES_KEY, collapsedPanes)
  }, [collapsedPanes])

  // Persist health filter state
  useEffect(() => {
    setStorageItem(HEALTH_FILTER_KEY, Array.from(healthFilter))
  }, [healthFilter])

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
  // Clamp gridColumns to valid range (1-6) for safety
  const gridColumns = Math.max(1, Math.min(6, preferences.ui.gridColumns))

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

  const toggleHealthFilter = (status: HealthStatus) => {
    setHealthFilter(prev => {
      const newSet = new Set(prev)
      if (newSet.has(status)) {
        newSet.delete(status)
      } else {
        newSet.add(status)
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

  // Sort services alphabetically (case-insensitive)
  const selectedServicesList = Array.from(selectedServices).sort((a, b) => 
    a.toLowerCase().localeCompare(b.toLowerCase())
  )

  // Filter services by health status (stopped services always show since they have no health status)
  const filteredServicesList = selectedServicesList.filter(serviceName => {
    const service = services.find(s => s.name === serviceName)
    const processStatus = service?.local?.status
    
    // Stopped services always show (they don't have health status)
    if (processStatus === 'stopped') return true
    
    const serviceHealth = healthReport?.services.find(
      s => s.serviceName === serviceName
    )?.status ?? 'unknown'
    return healthFilter.has(serviceHealth)
  })

  // Responsive column calculation
  const effectiveColumns = window.innerWidth < 600 ? 1 : gridColumns

  return (
    <div className={isFullscreen ? "fixed inset-0 z-50 flex flex-col overflow-hidden bg-background" : "h-screen flex flex-col overflow-hidden"}>
      <ToastContainer />
      
      {/* Global Toolbar */}
      <LogsToolbar
        viewMode={viewMode}
        onViewModeChange={(mode) => updateUI({ viewMode: mode })}
        isFullscreen={isFullscreen}
        onFullscreenChange={setIsFullscreen}
        isPaused={isPaused}
        onPauseChange={setIsPaused}
        autoScrollEnabled={autoScrollEnabled}
        onAutoScrollChange={setAutoScrollEnabled}
        searchTerm={globalSearchTerm}
        onSearchChange={setGlobalSearchTerm}
        onClearAll={handleClearAll}
        onExportAll={handleExportAll}
        onOpenSettings={() => setIsSettingsOpen(true)}
        onStartAll={startAll}
        onStopAll={stopAll}
        onRestartAll={restartAll}
        isBulkOperationInProgress={isBulkOperationInProgress}
        bulkOperation={bulkOperation}
        gridColumns={gridColumns}
        onGridColumnsChange={(columns) => updateUI({ gridColumns: columns })}
      />

      {/* Filters */}
      <div className="p-4 bg-background dark:bg-card dark:border-b shrink-0">
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
          {/* Divider */}
          <div className="w-px bg-border self-stretch" />
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
          {/* Divider */}
          <div className="w-px bg-border self-stretch" />
          {/* Health Status Filter */}
          <div role="group" aria-labelledby="health-status-filter-heading">
            <h3 id="health-status-filter-heading" className="font-medium mb-3">Health Status</h3>
            <div className="flex flex-wrap gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={healthFilter.has('healthy')}
                  onChange={() => toggleHealthFilter('healthy')}
                  aria-label="Show healthy services"
                  className="w-4 h-4 accent-green-500"
                />
                <span className="text-sm text-green-500">Healthy</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={healthFilter.has('degraded')}
                  onChange={() => toggleHealthFilter('degraded')}
                  aria-label="Show degraded services"
                  className="w-4 h-4 accent-yellow-500"
                />
                <span className="text-sm text-yellow-500">Degraded</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={healthFilter.has('unhealthy')}
                  onChange={() => toggleHealthFilter('unhealthy')}
                  aria-label="Show unhealthy services"
                  className="w-4 h-4 accent-red-500"
                />
                <span className="text-sm text-red-500">Unhealthy</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={healthFilter.has('starting')}
                  onChange={() => toggleHealthFilter('starting')}
                  aria-label="Show starting services"
                  className="w-4 h-4 accent-blue-500"
                />
                <span className="text-sm text-blue-500">Starting</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={healthFilter.has('unknown')}
                  onChange={() => toggleHealthFilter('unknown')}
                  aria-label="Show unknown services"
                  className="w-4 h-4 accent-gray-500"
                />
                <span className="text-sm text-gray-500 dark:text-gray-400">Unknown</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* View Content */}
      <div className="flex-1 overflow-hidden">
        {viewMode === 'grid' ? (
          filteredServicesList.length === 0 ? (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              <div className="text-center">
                <p className="text-lg font-medium">No services match the current filters</p>
                <p className="text-sm mt-2">Try adjusting your service or health status filters</p>
              </div>
            </div>
          ) : (
            <LogsPaneGrid columns={effectiveColumns} collapsedPanes={collapsedPanes}>
              {filteredServicesList.map(serviceName => {
                const service = services.find(s => s.name === serviceName)
                const serviceHealthStatus = healthReport?.services.find(s => s.serviceName === serviceName)?.status
                return (
                  <LogsPane
                    key={serviceName}
                    serviceName={serviceName}
                    port={service?.local?.port}
                    url={service?.local?.url}
                    service={service}
                    onCopy={handleCopyPane}
                    isPaused={isPaused}
                    globalSearchTerm={globalSearchTerm}
                    autoScrollEnabled={autoScrollEnabled}
                    clearAllTrigger={clearAllTrigger}
                    levelFilter={levelFilter}
                    isCollapsed={collapsedPanes[serviceName] ?? false}
                    onToggleCollapse={() => togglePaneCollapse(serviceName)}
                    serviceHealth={serviceHealthStatus}
                    onShowDetails={service && onServiceClick ? () => onServiceClick(service) : undefined}
                  />
                )
              })}
            </LogsPaneGrid>
          )
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
