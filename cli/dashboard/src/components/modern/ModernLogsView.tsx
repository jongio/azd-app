/**
 * ModernLogsView - Console/logs view with dark theme and multi-pane layout
 * Follows design spec: cli/dashboard/design/modern/components/console-view.md
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
  Download,
  Settings,
  ChevronDown,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { LogsPane, type LogEntry } from '@/components/LogsPane'
import { LogsPaneGrid } from '@/components/LogsPaneGrid'
import { LogsView } from '@/components/LogsView'
import { ModernSettingsDialog } from './ModernSettingsDialog'
import { usePreferences } from '@/hooks/usePreferences'
import { useToast } from '@/components/ui/toast'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import type { Service, HealthReportEvent, HealthStatus } from '@/types'

// =============================================================================
// Types
// =============================================================================

export interface ModernLogsViewProps {
  /** Callback when fullscreen changes */
  onFullscreenChange?: (isFullscreen: boolean) => void
  /** Health report for status updates */
  healthReport?: HealthReportEvent | null
  /** Callback when clicking on a service (to open detail panel) */
  onServiceClick?: (service: Service) => void
}

type ViewMode = 'grid' | 'unified'

// =============================================================================
// ModernLogsToolbar Component
// =============================================================================

interface ModernLogsToolbarProps {
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
  onExportLogs: (format: 'plaintext' | 'json' | 'csv' | 'markdown') => void
  onOpenSettings: () => void
  gridColumns: number
  onGridColumnsChange: (columns: number) => void
  onStartAll: () => void
  onStopAll: () => void
  onRestartAll: () => void
  isBulkOperationInProgress: boolean
}

function ModernLogsToolbar({
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
  onExportLogs,
  onOpenSettings,
  gridColumns,
  onGridColumnsChange,
  onStartAll,
  onStopAll,
  onRestartAll,
  isBulkOperationInProgress,
}: ModernLogsToolbarProps) {
  const [isExportMenuOpen, setIsExportMenuOpen] = React.useState(false)
  const exportMenuRef = React.useRef<HTMLDivElement>(null)

  // Close export menu when clicking outside
  React.useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (exportMenuRef.current && !exportMenuRef.current.contains(e.target as Node)) {
        setIsExportMenuOpen(false)
      }
    }
    if (isExportMenuOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isExportMenuOpen])
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
        {/* Grid columns */}
        {viewMode === 'grid' && (
          <div className="flex items-center gap-0.5 p-1 bg-slate-100 dark:bg-slate-800/50 rounded-md">
            {[1, 2, 3, 4, 5, 6].map((cols) => (
              <button
                key={cols}
                type="button"
                onClick={() => onGridColumnsChange(cols)}
                className={cn(
                  'w-6 h-6 flex items-center justify-center rounded text-xs font-semibold transition-colors',
                  gridColumns === cols
                    ? 'bg-cyan-500/20 text-cyan-600 dark:text-cyan-400'
                    : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'
                )}
              >
                {cols}
              </button>
            ))}
          </div>
        )}

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

        {/* Export Dropdown */}
        <div className="relative" ref={exportMenuRef}>
          <button
            type="button"
            onClick={() => setIsExportMenuOpen(!isExportMenuOpen)}
            className="flex items-center gap-1 px-3 py-1.5 rounded-md text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border border-transparent hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            title="Export logs"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export</span>
            <ChevronDown className={cn("w-3 h-3 transition-transform", isExportMenuOpen && "rotate-180")} />
          </button>
          {isExportMenuOpen && (
            <div className="absolute right-0 top-full mt-1 w-40 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg py-1 z-50">
              {[
                { format: 'plaintext' as const, label: 'Plain Text (.txt)' },
                { format: 'json' as const, label: 'JSON (.json)' },
                { format: 'csv' as const, label: 'CSV (.csv)' },
                { format: 'markdown' as const, label: 'Markdown (.md)' },
              ].map(({ format, label }) => (
                <button
                  key={format}
                  type="button"
                  onClick={() => {
                    onExportLogs(format)
                    setIsExportMenuOpen(false)
                  }}
                  className="w-full px-3 py-2 text-left text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  {label}
                </button>
              ))}
            </div>
          )}
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
// ModernFiltersBar Component
// =============================================================================

interface ModernFiltersBarProps {
  services: Service[]
  selectedServices: Set<string>
  onToggleService: (name: string) => void
  levelFilter: Set<'info' | 'warning' | 'error'>
  onToggleLevel: (level: 'info' | 'warning' | 'error') => void
  healthFilter: Set<HealthStatus>
  onToggleHealth: (status: HealthStatus) => void
}

function ModernFiltersBar({
  services,
  selectedServices,
  onToggleService,
  levelFilter,
  onToggleLevel,
  healthFilter,
  onToggleHealth,
}: ModernFiltersBarProps) {
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
              checked={healthFilter.has('starting')}
              onChange={() => onToggleHealth('starting')}
              className="w-3.5 h-3.5 rounded border-slate-400 dark:border-slate-600 bg-white dark:bg-slate-800 text-sky-500 focus:ring-sky-500/30 focus:ring-offset-white dark:focus:ring-offset-slate-900"
            />
            <span className="text-xs text-sky-600 dark:text-sky-400">Starting</span>
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
// ModernLogsView Component
// =============================================================================

export function ModernLogsView({
  onFullscreenChange,
  healthReport,
  onServiceClick,
}: ModernLogsViewProps) {
  const [services, setServices] = React.useState<Service[]>([])
  const [selectedServices, setSelectedServices] = React.useState<Set<string>>(new Set())
  const [isPaused, setIsPaused] = React.useState(false)
  const [isFullscreen, setIsFullscreen] = React.useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = React.useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = React.useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = React.useState(true)
  const [clearAllTrigger, setClearAllTrigger] = React.useState(0)
  const [allLogs, _setAllLogs] = React.useState<LogEntry[]>([])
  const [levelFilter, setLevelFilter] = React.useState<Set<'info' | 'warning' | 'error'>>(
    new Set(['info', 'warning', 'error'])
  )
  const [healthFilter, setHealthFilter] = React.useState<Set<HealthStatus>>(
    new Set(['healthy', 'degraded', 'unhealthy', 'starting', 'unknown'])
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
  const gridColumns = Math.max(1, Math.min(6, preferences.ui.gridColumns))

  // Notify parent of fullscreen changes
  React.useEffect(() => {
    onFullscreenChange?.(isFullscreen)
  }, [isFullscreen, onFullscreenChange])

  // Fetch services
  React.useEffect(() => {
    const fetchServices = async () => {
      try {
        const res = await fetch('/api/services')
        if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)
        const data = (await res.json()) as Service[]
        setServices(data)
        if (selectedServices.size === 0) {
          setSelectedServices(new Set(data.map((s) => s.name)))
        }
      } catch (err) {
        console.error('Failed to fetch services:', err)
      }
    }
    void fetchServices()
  }, [selectedServices.size])

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

  // Export logs with format selection
  const handleExportLogs = React.useCallback((format: 'plaintext' | 'json' | 'csv' | 'markdown') => {
    // Collect logs from all selected services (in a real implementation, 
    // this would aggregate from all LogsPane instances)
    const logs = allLogs.filter(log => selectedServices.has(log.service))
    
    if (logs.length === 0) {
      showToast('No logs to export', 'info')
      return
    }

    let content = ''
    let extension = 'txt'
    let mimeType = 'text/plain'

    switch (format) {
      case 'json':
        content = JSON.stringify(logs, null, 2)
        extension = 'json'
        mimeType = 'application/json'
        break
      case 'csv':
        content = 'Service,Timestamp,Level,Message\n' +
          logs.map(log => `"${log.service}","${log.timestamp}",${log.level},"${log.message.replace(/"/g, '""')}"`).join('\n')
        extension = 'csv'
        mimeType = 'text/csv'
        break
      case 'markdown':
        content = `# Logs Export\n\nExported at: ${new Date().toISOString()}\n\n` +
          logs.map(log => `**[${log.timestamp}]** \`${log.service}\` ${log.message}`).join('\n\n')
        extension = 'md'
        mimeType = 'text/markdown'
        break
      default: // plaintext
        content = logs.map(log => `[${log.timestamp}] [${log.service}] ${log.message}`).join('\n')
        extension = 'txt'
        mimeType = 'text/plain'
    }

    const blob = new Blob([content], { type: mimeType })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `logs-${Date.now()}.${extension}`
    a.click()
    URL.revokeObjectURL(url)
    
    showToast(`Exported ${logs.length} logs as ${format.toUpperCase()}`, 'success')
  }, [allLogs, selectedServices, showToast])

  // Filter and sort services
  const selectedServicesList = Array.from(selectedServices).sort((a, b) =>
    a.toLowerCase().localeCompare(b.toLowerCase())
  )

  const filteredServicesList = selectedServicesList.filter((serviceName) => {
    const service = services.find((s) => s.name === serviceName)
    const processStatus = service?.local?.status
    if (processStatus === 'stopped') return true
    const serviceHealth =
      healthReport?.services.find((s) => s.serviceName === serviceName)?.status ?? 'unknown'
    return healthFilter.has(serviceHealth)
  })

  const effectiveColumns = typeof window !== 'undefined' && window.innerWidth < 600 ? 1 : gridColumns

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
      <ModernLogsToolbar
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
        onExportLogs={handleExportLogs}
        onOpenSettings={() => setIsSettingsOpen(true)}
        gridColumns={gridColumns}
        onGridColumnsChange={(columns) => updateUI({ gridColumns: columns })}
        onStartAll={() => void startAll()}
        onStopAll={() => void stopAll()}
        onRestartAll={() => void restartAll()}
        isBulkOperationInProgress={isBulkOperationInProgress()}
      />

      {/* Filters */}
      <ModernFiltersBar
        services={services}
        selectedServices={selectedServices}
        onToggleService={handleToggleService}
        levelFilter={levelFilter}
        onToggleLevel={toggleLevelFilter}
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
                <p className="text-sm mt-2">Try adjusting your service or health status filters</p>
              </div>
            </div>
          ) : (
            <LogsPaneGrid columns={effectiveColumns} collapsedPanes={collapsedPanes}>
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
          <LogsView selectedServices={selectedServices} levelFilter={levelFilter} />
        )}
      </div>

      {/* Settings Dialog */}
      <ModernSettingsDialog
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />
    </div>
  )
}
