/**
 * ConsoleView - Console/logs view with dark theme and multi-pane layout
 * Follows design spec: cli/dashboard/design/components/console-view.md
 */
import * as React from 'react'
import { 
  Search, 
  Pause, 
  Play, 
  Trash2, 
  Maximize2,
  Minimize2,
  X,
  Grid3X3,
  List,
  RefreshCw,
  StopCircle,
  PlayCircle,
  Settings,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { LogsPane, type LogEntry } from '@/components/LogsPane'
import { LogsPaneGrid } from '@/components/LogsPaneGrid'
import { LogsView } from '@/components/LogsView'
import { SettingsDialog } from './SettingsDialog'
import { usePreferences } from '@/hooks/usePreferences'
import { useToast } from '@/components/ui/toast'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { useServicesContext } from '@/contexts/ServicesContext'
import type { Service, HealthReportEvent, HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ConsoleViewProps {
  /** Callback when fullscreen changes */
  onFullscreenChange?: (isFullscreen: boolean) => void
  /** Health report for status updates */
  healthReport?: HealthReportEvent | null
  /** Callback when clicking on a service (to open detail panel) */
  onServiceClick?: (service: Service) => void
}

type ViewMode = 'grid' | 'unified'

// =============================================================================
// LogsToolbar Component
// =============================================================================

interface LogsToolbarProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  isFullscreen: boolean
  onFullscreenChange: (isFullscreen: boolean) => void
  isPaused: boolean
  onPauseChange: (paused: boolean) => void
  autoScrollEnabled: boolean
  onAutoScrollChange: (enabled: boolean) => void
  searchTerm: string
  onSearchChange: (term: string) => void
  onClearAll: () => void
  onOpenSettings: () => void
  onStartAll: () => void
  onStopAll: () => void
  onRestartAll: () => void
  isBulkOperationInProgress: boolean
}

function LogsToolbar({
  viewMode,
  onViewModeChange,
  isFullscreen,
  onFullscreenChange,
  isPaused,
  onPauseChange,
  autoScrollEnabled,
  onAutoScrollChange,
  searchTerm,
  onSearchChange,
  onClearAll,
  onOpenSettings,
  onStartAll,
  onStopAll,
  onRestartAll,
  isBulkOperationInProgress,
}: LogsToolbarProps) {
  return (
    <div className="flex items-center gap-4 p-3 bg-slate-200 dark:bg-slate-900 border-b border-slate-300 dark:border-slate-700 shrink-0">
      {/* Left section - Actions */}
      <div className="flex items-center gap-2">
        {/* Pause/Play */}
        <button
          type="button"
          onClick={() => onPauseChange(!isPaused)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
            isPaused
              ? 'bg-amber-500/20 text-amber-600 dark:text-amber-400 border border-amber-500/30'
              : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-transparent hover:bg-slate-50 dark:hover:bg-slate-700'
          )}
        >
          {isPaused ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
          <span>{isPaused ? 'Resume' : 'Pause'}</span>
        </button>

        {/* Clear All */}
        <button
          type="button"
          onClick={onClearAll}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-transparent hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
        >
          <Trash2 className="w-3.5 h-3.5" />
          <span>Clear</span>
        </button>

        {/* Auto-scroll toggle */}
        <button
          type="button"
          onClick={() => onAutoScrollChange(!autoScrollEnabled)}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
            autoScrollEnabled
              ? 'bg-cyan-500/20 text-cyan-600 dark:text-cyan-400 border border-cyan-500/30'
              : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-transparent hover:bg-slate-50 dark:hover:bg-slate-700'
          )}
          title={autoScrollEnabled ? 'Disable auto-scroll to bottom' : 'Enable auto-scroll to bottom'}
        >
          <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
          </svg>
          <span>{autoScrollEnabled ? 'Auto-scroll' : 'Scroll'}</span>
        </button>

        {/* Divider */}
        <div className="w-px h-6 bg-slate-300 dark:bg-slate-700" />

        {/* Bulk Service Operations */}
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={onStartAll}
            disabled={isBulkOperationInProgress}
            className="p-1.5 rounded-md text-emerald-500 dark:text-emerald-400 hover:bg-emerald-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Start All"
          >
            <PlayCircle className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={onStopAll}
            disabled={isBulkOperationInProgress}
            className="p-1.5 rounded-md text-slate-500 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Stop All"
          >
            <StopCircle className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={onRestartAll}
            disabled={isBulkOperationInProgress}
            className="p-1.5 rounded-md text-sky-500 dark:text-sky-400 hover:bg-sky-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Restart All"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Center section - Search */}
      <div className="flex-1 max-w-md">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-500" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search logs..."
            className="w-full pl-9 pr-9 py-1.5 bg-white dark:bg-slate-800/50 border border-slate-300 dark:border-slate-700 rounded-md text-sm text-slate-800 dark:text-slate-200 placeholder:text-slate-400 dark:placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/50 focus:border-cyan-500/50"
          />
          {searchTerm && (
            <button
              type="button"
              onClick={() => onSearchChange('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 transition-colors"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Right section - View controls */}
      <div className="flex items-center gap-2">
        {/* View Mode Toggle */}
        <div className="flex items-center gap-0.5 p-1 bg-slate-100 dark:bg-slate-800/50 rounded-md">
          <button
            type="button"
            onClick={() => onViewModeChange('grid')}
            className={cn(
              'p-1.5 rounded transition-colors',
              viewMode === 'grid'
                ? 'bg-cyan-500/20 text-cyan-600 dark:text-cyan-400'
                : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'
            )}
            title="Grid view"
          >
            <Grid3X3 className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={() => onViewModeChange('unified')}
            className={cn(
              'p-1.5 rounded transition-colors',
              viewMode === 'unified'
                ? 'bg-cyan-500/20 text-cyan-600 dark:text-cyan-400'
                : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'
            )}
            title="Unified view"
          >
            <List className="w-4 h-4" />
          </button>
        </div>

        {/* Settings */}
        <button
          type="button"
          onClick={onOpenSettings}
          className="p-2 rounded-md text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
          title="Settings"
        >
          <Settings className="w-4 h-4" />
        </button>

        {/* Fullscreen */}
        <button
          type="button"
          onClick={() => onFullscreenChange(!isFullscreen)}
          className="p-2 rounded-md text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
          title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
        >
          {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
        </button>
      </div>
    </div>
  )
}

// =============================================================================
// FiltersBar Component
// =============================================================================

type FilterableLifecycleState = 'running' | 'stopped' | 'starting'

interface FiltersBarProps {
  services: Service[]
  selectedServices: Set<string>
  onToggleService: (name: string) => void
  levelFilter: Set<'info' | 'warning' | 'error'>
  onToggleLevel: (level: 'info' | 'warning' | 'error') => void
  stateFilter: Set<FilterableLifecycleState>
  onToggleState: (state: FilterableLifecycleState) => void
  healthFilter: Set<HealthStatus>
  onToggleHealth: (status: HealthStatus) => void
}

function FiltersBar({
  services,
  selectedServices,
  onToggleService,
  levelFilter,
  onToggleLevel,
  stateFilter,
  onToggleState,
  healthFilter,
  onToggleHealth,
}: FiltersBarProps) {
  return (
    <div className="flex flex-wrap gap-6 p-4 bg-slate-100 dark:bg-slate-800 border-b border-slate-300 dark:border-slate-700 shrink-0">
      {/* Services */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Services</span>
        <div className="flex flex-wrap gap-2">
          {services.sort((a, b) => a.name.localeCompare(b.name)).map((service) => (
            <label key={service.name} className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={selectedServices.has(service.name)}
                onChange={() => onToggleService(service.name)}
                className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-cyan-500 focus:ring-cyan-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
              />
              <span className="text-xs text-slate-700 dark:text-slate-300">{service.name}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Log Levels */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Log Levels</span>
        <div className="flex gap-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={levelFilter.has('info')}
              onChange={() => onToggleLevel('info')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-sky-500 focus:ring-sky-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-sky-600 dark:text-sky-400">Info</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={levelFilter.has('warning')}
              onChange={() => onToggleLevel('warning')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-amber-500 focus:ring-amber-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-amber-600 dark:text-amber-400">Warning</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={levelFilter.has('error')}
              onChange={() => onToggleLevel('error')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-rose-500 focus:ring-rose-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-rose-600 dark:text-rose-400">Error</span>
          </label>
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* State Filter */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">State</span>
        <div className="flex gap-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={stateFilter.has('running')}
              onChange={() => onToggleState('running')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-emerald-500 focus:ring-emerald-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-emerald-600 dark:text-emerald-400">Running</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={stateFilter.has('stopped')}
              onChange={() => onToggleState('stopped')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-500 focus:ring-slate-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-slate-600 dark:text-slate-400">Stopped</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={stateFilter.has('starting')}
              onChange={() => onToggleState('starting')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-sky-500 focus:ring-sky-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-sky-600 dark:text-sky-400">Starting</span>
          </label>
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Health Status */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Health Status</span>
        <div className="flex gap-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={healthFilter.has('healthy')}
              onChange={() => onToggleHealth('healthy')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-emerald-500 focus:ring-emerald-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-emerald-600 dark:text-emerald-400">Healthy</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={healthFilter.has('degraded')}
              onChange={() => onToggleHealth('degraded')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-amber-500 focus:ring-amber-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-amber-600 dark:text-amber-400">Degraded</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={healthFilter.has('unhealthy')}
              onChange={() => onToggleHealth('unhealthy')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-rose-500 focus:ring-rose-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-rose-600 dark:text-rose-400">Unhealthy</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={healthFilter.has('unknown')}
              onChange={() => onToggleHealth('unknown')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-500 focus:ring-slate-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-slate-600 dark:text-slate-400">Unknown</span>
          </label>
        </div>
      </div>
    </div>
  )
}

// =============================================================================
// ConsoleView Component
// =============================================================================

export function ConsoleView({
  onFullscreenChange,
  healthReport,
  onServiceClick,
}: ConsoleViewProps) {
  const { services } = useServicesContext()
  const [selectedServices, setSelectedServices] = React.useState<Set<string>>(new Set())
  const [isPaused, setIsPaused] = React.useState(false)
  const [isFullscreen, setIsFullscreen] = React.useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = React.useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = React.useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = React.useState(true)
  const [clearAllTrigger, setClearAllTrigger] = React.useState(0)
  const [levelFilter, setLevelFilter] = React.useState<Set<'info' | 'warning' | 'error'>>(
    new Set(['info', 'warning', 'error'])
  )
  const [stateFilter, setStateFilter] = React.useState<Set<FilterableLifecycleState>>(
    new Set(['running', 'stopped', 'starting'])
  )
  const [healthFilter, setHealthFilter] = React.useState<Set<HealthStatus>>(
    new Set(['healthy', 'degraded', 'unhealthy', 'unknown'])
  )
  const [collapsedPanes, setCollapsedPanes] = React.useState<Record<string, boolean>>({})

  const { preferences, updateUI } = usePreferences()
  const { showToast, ToastContainer } = useToast()
  const {
    startAll,
    stopAll,
    restartAll,
    isBulkOperationInProgress,
  } = useServiceOperations()

  const viewMode = preferences.ui.viewMode

  // Notify parent of fullscreen changes
  React.useEffect(() => {
    onFullscreenChange?.(isFullscreen)
  }, [isFullscreen, onFullscreenChange])

  // Initialize selected services when services change
  React.useEffect(() => {
    if (services.length > 0 && selectedServices.size === 0) {
      setSelectedServices(new Set(services.map((s) => s.name)))
    }
  }, [services, selectedServices.size])

  // Keyboard shortcuts
  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.code === 'Space' && !e.ctrlKey && !e.shiftKey && !e.altKey) {
        const target = e.target as HTMLElement
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') {
          e.preventDefault()
          setIsPaused((prev) => !prev)
        }
      }
      if (e.ctrlKey && e.shiftKey && e.code === 'KeyL') {
        e.preventDefault()
        updateUI({ viewMode: viewMode === 'grid' ? 'unified' : 'grid' })
      }
      if (e.key === 'F11' || (e.ctrlKey && e.shiftKey && e.code === 'KeyF')) {
        e.preventDefault()
        setIsFullscreen((prev) => !prev)
      }
      if (e.key === 'Escape' && isFullscreen) {
        setIsFullscreen(false)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [viewMode, updateUI, isFullscreen])

  const handleToggleService = (serviceName: string) => {
    setSelectedServices((prev) => {
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
    setLevelFilter((prev) => {
      const next = new Set(prev)
      if (next.has(level)) {
        next.delete(level)
      } else {
        next.add(level)
      }
      return next
    })
  }

  const toggleStateFilter = (state: FilterableLifecycleState) => {
    setStateFilter((prev) => {
      const next = new Set(prev)
      if (next.has(state)) {
        next.delete(state)
      } else {
        next.add(state)
      }
      return next
    })
  }

  const toggleHealthFilter = (status: HealthStatus) => {
    setHealthFilter((prev) => {
      const next = new Set(prev)
      if (next.has(status)) {
        next.delete(status)
      } else {
        next.add(status)
      }
      return next
    })
  }

  const handleClearAll = () => {
    setClearAllTrigger((prev) => prev + 1)
    showToast('All logs cleared', 'success')
  }

  const togglePaneCollapse = (serviceName: string) => {
    setCollapsedPanes((prev) => ({
      ...prev,
      [serviceName]: !prev[serviceName],
    }))
  }

  const handleCopyPane = React.useCallback((logs: LogEntry[]) => {
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
  }, [showToast, preferences.copy.defaultFormat])

  // Filter and sort services
  const selectedServicesList = Array.from(selectedServices).sort((a, b) =>
    a.toLowerCase().localeCompare(b.toLowerCase())
  )

  const filteredServicesList = selectedServicesList.filter((serviceName) => {
    const service = services.find((s) => s.name === serviceName)
    const processStatus = service?.local?.status
    
    // Map process status to filterable state
    // Transitional states (starting, stopping, restarting) map to 'starting' category
    let mappedState: FilterableLifecycleState = 'stopped'
    if (processStatus === 'running' || processStatus === 'ready' || processStatus === 'watching') {
      mappedState = 'running'
    } else if (processStatus === 'starting' || processStatus === 'restarting' || processStatus === 'stopping' || processStatus === 'building') {
      // All transitional states show under "Starting" filter
      mappedState = 'starting'
    } else if (processStatus === 'stopped' || processStatus === 'not-started' || processStatus === 'not-running' || processStatus === 'completed' || processStatus === 'built' || processStatus === 'failed' || !processStatus) {
      mappedState = 'stopped'
    }
    
    // Check state filter
    if (!stateFilter.has(mappedState)) return false
    
    // Stopped/transitional services skip health filter (health not meaningful during transitions)
    if (mappedState === 'stopped' || mappedState === 'starting') return true
    
    // Check health filter for running services
    const serviceHealth =
      healthReport?.services.find((s) => s.serviceName === serviceName)?.status ?? 'unknown'
    return healthFilter.has(serviceHealth)
  })

  return (
    <div
      className={cn(
        'flex flex-col overflow-hidden',
        // Console uses theme-aware colors
        'bg-slate-100 dark:bg-slate-900 text-slate-800 dark:text-slate-200',
        isFullscreen ? 'fixed inset-0 z-50' : 'h-full'
      )}
    >
      <ToastContainer />

      {/* Toolbar */}
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
        onOpenSettings={() => setIsSettingsOpen(true)}
        onStartAll={() => void startAll()}
        onStopAll={() => void stopAll()}
        onRestartAll={() => void restartAll()}
        isBulkOperationInProgress={isBulkOperationInProgress()}
      />

      {/* Filters */}
      <FiltersBar
        services={services}
        selectedServices={selectedServices}
        onToggleService={handleToggleService}
        levelFilter={levelFilter}
        onToggleLevel={toggleLevelFilter}
        stateFilter={stateFilter}
        onToggleState={toggleStateFilter}
        healthFilter={healthFilter}
        onToggleHealth={toggleHealthFilter}
      />

      {/* Content - Constrain to remaining viewport height */}
      <div className="flex-1 overflow-hidden min-h-0">
        {viewMode === 'grid' ? (
          filteredServicesList.length === 0 ? (
            <div className="flex items-center justify-center h-full text-slate-500">
              <div className="text-center">
                <p className="text-lg font-medium">No services match the current filters</p>
                <p className="text-sm mt-2">Try adjusting your service, state, or health status filters</p>
              </div>
            </div>
          ) : (
            <LogsPaneGrid columns={2} collapsedPanes={collapsedPanes} autoFit={true}>
              {filteredServicesList.map((serviceName) => {
                const service = services.find((s) => s.name === serviceName)
                const serviceHealthStatus = healthReport?.services.find(
                  (s) => s.serviceName === serviceName
                )?.status
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
                    onShowDetails={
                      service && onServiceClick ? () => onServiceClick(service) : undefined
                    }
                  />
                )
              })}
            </LogsPaneGrid>
          )
        ) : (
          <LogsView 
            selectedServices={selectedServices} 
            levelFilter={levelFilter}
            isPaused={isPaused}
            autoScrollEnabled={autoScrollEnabled}
            globalSearchTerm={globalSearchTerm}
            clearAllTrigger={clearAllTrigger}
            hideControls={true}
          />
        )}
      </div>

      {/* Settings Dialog */}
      <SettingsDialog
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />
    </div>
  )
}
