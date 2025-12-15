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
  Info,
  AlertTriangle,
  XCircle,
  CheckCircle,
  Circle,
  Loader2,
  Heart,
  HeartPulse,
  HeartCrack,
  HelpCircle,
  Globe,
  Server,
  Database,
  Box,
  Cpu,
  Zap,
  Package,
  Activity,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { normalizeHealthStatus } from '@/lib/service-utils'
import { LogsPane, type LogEntry } from '@/components/LogsPane'
import { LogsPaneGrid } from '@/components/LogsPaneGrid'
import { LogsView } from '@/components/LogsView'
import { SettingsDialog } from './SettingsDialog'
import { DiagnosticsModal } from './DiagnosticsModal'
import { ModeToggle, type LogMode } from './ModeToggle'
import { Select } from '@/components/ui/select'
import { usePreferences } from '@/hooks/usePreferences'
import { useToast } from '@/components/ui/toast'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { useServicesContext } from '@/contexts/ServicesContext'
import type { Service, HealthReportEvent, HealthStatus } from '@/types'

const MIN_SYNC_INTERVAL = 5000
const MAX_SYNC_INTERVAL = 300000

function clampSyncInterval(value: number): number {
  if (!Number.isFinite(value)) return MIN_SYNC_INTERVAL
  return Math.min(MAX_SYNC_INTERVAL, Math.max(MIN_SYNC_INTERVAL, value))
}

// =============================================================================
// Types
// =============================================================================

type TimeRangePreset = '15m' | '30m' | '6h' | '24h'

type LogLevel = 'info' | 'warning' | 'error'

type ConsoleServiceSelectionMode = 'all' | 'custom'

type SavedConsoleFiltersV1 = {
  version: 1
  serviceSelectionMode: ConsoleServiceSelectionMode
  selectedServices?: string[]
  levelFilter: LogLevel[]
  stateFilter: FilterableLifecycleState[]
  healthFilter: HealthStatus[]
}

type AzureConnectionStatus = 'connected' | 'disconnected' | 'connecting' | 'disabled'

interface ModeApiResponse {
  mode?: LogMode
  azureEnabled?: boolean
  azureStatus?: AzureConnectionStatus
  azureRealtime?: boolean
  connectionMessage?: string
}

function isLogMode(value: unknown): value is LogMode {
  return value === 'local' || value === 'azure'
}

function isAzureConnectionStatus(value: unknown): value is AzureConnectionStatus {
  return value === 'connected' || value === 'disconnected' || value === 'connecting' || value === 'disabled'
}

function parseModeApiResponse(value: unknown): ModeApiResponse {
  if (typeof value !== 'object' || value === null) {
    return {}
  }

  const record = value as Record<string, unknown>

  const mode = isLogMode(record.mode) ? record.mode : undefined
  const azureEnabled = typeof record.azureEnabled === 'boolean' ? record.azureEnabled : undefined
  const azureStatus = isAzureConnectionStatus(record.azureStatus) ? record.azureStatus : undefined
  const azureRealtime = typeof record.azureRealtime === 'boolean' ? record.azureRealtime : undefined
  const connectionMessage = typeof record.connectionMessage === 'string' ? record.connectionMessage : undefined

  return { mode, azureEnabled, azureStatus, azureRealtime, connectionMessage }
}

function getSavedSyncInterval(): number {
  if (globalThis.localStorage === undefined) {
    return 30000
  }

  try {
    const saved = Number(globalThis.localStorage.getItem('logs-sync-interval'))
    return clampSyncInterval(Number.isFinite(saved) ? saved : 30000)
  } catch {
    return 30000
  }
}

function setSavedSyncInterval(interval: number): void {
  if (globalThis.localStorage === undefined) {
    return
  }

  try {
    globalThis.localStorage.setItem('logs-sync-interval', String(interval))
  } catch {
    // Ignore persistence failures
  }
}

const CONSOLE_FILTERS_STORAGE_KEY = 'console-filters-v1'

function isLogLevel(value: unknown): value is LogLevel {
  return value === 'info' || value === 'warning' || value === 'error'
}

function isFilterableLifecycleState(value: unknown): value is FilterableLifecycleState {
  return value === 'running' || value === 'stopped' || value === 'starting'
}

function isHealthStatus(value: unknown): value is HealthStatus {
  return value === 'healthy' || value === 'degraded' || value === 'unhealthy' || value === 'unknown'
}

function loadSavedConsoleFilters(): SavedConsoleFiltersV1 | null {
  if (globalThis.localStorage === undefined) {
    return null
  }

  try {
    const raw = globalThis.localStorage.getItem(CONSOLE_FILTERS_STORAGE_KEY)
    if (!raw) {
      return null
    }

    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) {
      return null
    }

    const record = parsed as Record<string, unknown>
    if (record.version !== 1) {
      return null
    }

    const serviceSelectionMode = record.serviceSelectionMode === 'custom' ? 'custom' : 'all'
    const selectedServices = Array.isArray(record.selectedServices)
      ? record.selectedServices.filter((v): v is string => typeof v === 'string')
      : undefined

    const levelFilter = Array.isArray(record.levelFilter)
      ? record.levelFilter.filter(isLogLevel)
      : []
    const stateFilter = Array.isArray(record.stateFilter)
      ? record.stateFilter.filter(isFilterableLifecycleState)
      : []
    const healthFilter = Array.isArray(record.healthFilter)
      ? record.healthFilter.filter(isHealthStatus)
      : []

    return {
      version: 1,
      serviceSelectionMode,
      selectedServices,
      levelFilter,
      stateFilter,
      healthFilter,
    }
  } catch {
    return null
  }
}

function saveConsoleFilters(value: SavedConsoleFiltersV1): void {
  if (globalThis.localStorage === undefined) {
    return
  }

  try {
    globalThis.localStorage.setItem(CONSOLE_FILTERS_STORAGE_KEY, JSON.stringify(value))
  } catch {
    // Ignore persistence failures
  }
}

function getSavedAzureRealtime(): boolean {
  if (globalThis.localStorage === undefined) {
    return false
  }

  try {
    return globalThis.localStorage.getItem('azure-logs-realtime') === 'true'
  } catch {
    return false
  }
}

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
  logMode: LogMode
  onLogModeChange: (mode: LogMode) => void
  azureEnabled: boolean
  azureStatus: AzureConnectionStatus
  azureConnectionMessage?: string
  // Azure log controls
  timeRange: { preset: TimeRangePreset }
  onTimeRangeChange: (preset: TimeRangePreset) => void
  syncInterval: number
  onSyncIntervalChange: (interval: number) => void
  onRunDiagnostics: () => void
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
  logMode,
  onLogModeChange,
  azureEnabled,
  azureStatus,
  azureConnectionMessage,
  timeRange,
  onTimeRangeChange,
  syncInterval,
  onSyncIntervalChange,
  onRunDiagnostics,
}: Readonly<LogsToolbarProps>) {
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
        {/* Log Source Toggle (Local/Azure) */}
        <ModeToggle
          mode={logMode}
          onModeChange={onLogModeChange}
          azureEnabled={azureEnabled}
          azureStatus={azureStatus}
          connectionMessage={azureConnectionMessage}
          size="compact"
          showLabels={false}
          showStatus={true}
        />

        {/* Azure Log Controls - Show when Azure mode is active */}
        {logMode === 'azure' && azureEnabled && (
          <>
            {/* Timeframe selector */}
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-slate-600 dark:text-slate-400">Timeframe:</span>
              <Select
                value={timeRange.preset}
                onChange={(e) => onTimeRangeChange(e.target.value as TimeRangePreset)}
                className="h-7 w-24 text-xs bg-white dark:bg-slate-800 border-slate-300 dark:border-slate-700 px-2 py-0"
              >
                <option value="15m">15 min</option>
                <option value="30m">30 min</option>
                <option value="6h">6 hours</option>
                <option value="24h">24 hours</option>
              </Select>
            </div>
            
            {/* Refresh interval selector */}
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-slate-600 dark:text-slate-400">Refresh:</span>
              <Select
                value={String(syncInterval)}
                onChange={(e) => onSyncIntervalChange(Number(e.target.value))}
                className="h-7 w-20 text-xs bg-white dark:bg-slate-800 border-slate-300 dark:border-slate-700 px-2 py-0"
              >
                <option value="5000">5s</option>
                <option value="10000">10s</option>
                <option value="30000">30s</option>
                <option value="60000">1m</option>
                <option value="300000">5m</option>
              </Select>
            </div>

            {/* Realtime/polling toggle is temporarily removed; tracked in docs/specs/azure-logs/tasks.md. */}
            
            {/* Diagnostics button */}
            <button
              type="button"
              onClick={onRunDiagnostics}
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium bg-azure-100 dark:bg-azure-500/20 text-azure-700 dark:text-azure-300 hover:bg-azure-200 dark:hover:bg-azure-500/30 transition-colors border border-azure-300 dark:border-azure-700"
              title="Run Azure logs diagnostics"
            >
              <Activity className="w-3.5 h-3.5" />
              <span>Diagnostics</span>
            </button>
          </>
        )}

        {/* Divider */}
        <div className="w-px h-6 bg-slate-300 dark:bg-slate-700" />

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
// Helper Functions  
// =============================================================================

interface ServiceIconColor {
  icon: typeof Globe
  colorScheme: {
    selected: string
    unselected: string
  }
}

function getServiceIconAndColor(serviceName: string, health: HealthStatus = 'unknown'): ServiceIconColor {
  const lowerName = serviceName.toLowerCase()
  
  // Determine icon based on service name patterns
  let icon = Package // default
  if (lowerName.includes('web') || lowerName.includes('frontend') || lowerName.includes('ui') || lowerName.includes('app')) {
    icon = Globe
  } else if (lowerName.includes('api') || lowerName.includes('backend') || lowerName.includes('server')) {
    icon = Server
  } else if (lowerName.includes('worker') || lowerName.includes('queue') || lowerName.includes('background')) {
    icon = Cpu
  } else if (lowerName.includes('function') || lowerName.includes('func')) {
    icon = Zap
  } else if (lowerName.includes('container')) {
    icon = Box
  } else if (lowerName.includes('db') || lowerName.includes('database') || lowerName.includes('postgres') || lowerName.includes('redis') || lowerName.includes('mongo') || lowerName.includes('mysql')) {
    icon = Database
  }
  
  // Health-based colors matching log pane indicators
  // Red (unhealthy), Yellow (degraded/unknown), Green (healthy)
  const healthColorSchemes: Record<HealthStatus, { selected: string; unselected: string }> = {
    healthy: {
      selected: 'bg-green-100 dark:bg-green-500/20 text-green-700 dark:text-green-300 ring-1 ring-green-500',
      unselected: 'text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-500/10'
    },
    degraded: {
      selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-500',
      unselected: 'text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-500/10'
    },
    unhealthy: {
      selected: 'bg-red-100 dark:bg-red-500/20 text-red-700 dark:text-red-300 ring-1 ring-red-500',
      unselected: 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-500/10'
    },
    unknown: {
      selected: 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-500',
      unselected: 'text-amber-600 dark:text-amber-400 hover:bg-amber-50 dark:hover:bg-amber-500/10'
    }
  }
  
  return {
    icon,
    colorScheme: healthColorSchemes[health]
  }
}

// =============================================================================
// FiltersBar Component
// =============================================================================

type FilterableLifecycleState = 'running' | 'stopped' | 'starting'

interface FiltersBarProps {
  services: Service[]
  selectedServices: Set<string>
  onToggleService: (name: string) => void
  levelFilter: Set<LogLevel>
  onToggleLevel: (level: LogLevel) => void
  stateFilter: Set<FilterableLifecycleState>
  onToggleState: (state: FilterableLifecycleState) => void
  healthFilter: Set<HealthStatus>
  onToggleHealth: (status: HealthStatus) => void
  healthReport?: HealthReportEvent | null
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
  healthReport,
}: Readonly<FiltersBarProps>) {
  const sortedServices = React.useMemo(() => {
    return [...services].sort((a, b) => a.name.localeCompare(b.name))
  }, [services])

  return (
    <div className="flex flex-wrap gap-6 p-4 bg-slate-100 dark:bg-slate-800 border-b border-slate-300 dark:border-slate-700 shrink-0">
      {/* Services */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Services</span>
        <div className="flex flex-wrap gap-2">
          {sortedServices.map((service) => {
            // Get health status from health report
            const serviceHealth = healthReport?.services.find(
              (s) => s.serviceName === service.name
            )?.status ?? 'unknown'
            const normalizedHealth = normalizeHealthStatus(serviceHealth)
            
            const { icon: IconComponent, colorScheme } = getServiceIconAndColor(service.name, normalizedHealth)
            const isSelected = selectedServices.has(service.name)
            
            return (
              <button
                key={service.name}
                type="button"
                onClick={() => onToggleService(service.name)}
                className={cn(
                  'flex items-center gap-1.5 px-2.5 py-1.5 rounded-md transition-all max-w-[150px]',
                  isSelected
                    ? colorScheme.selected
                    : cn('bg-transparent', colorScheme.unselected)
                )}
                aria-label={`Toggle ${service.name}`}
                title={`${service.name} - ${normalizedHealth}`}
              >
                <IconComponent className="w-3.5 h-3.5 shrink-0" />
                <span className="text-xs font-medium truncate">{service.name}</span>
              </button>
            )
          })}
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Log Levels */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Log Levels</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleLevel('info')}
            aria-label="Info"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('info')
                ? 'bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300 ring-1 ring-sky-300 dark:ring-sky-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Info logs"
          >
            <Info className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-sky-700/95 text-white text-xs px-2 py-1">Info</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleLevel('warning')}
            aria-label="Warning"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('warning')
                ? 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Warning logs"
          >
            <AlertTriangle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-amber-700/95 text-white text-xs px-2 py-1">Warning</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleLevel('error')}
            aria-label="Error"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              levelFilter.has('error')
                ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Error logs"
          >
            <XCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-rose-700/95 text-white text-xs px-2 py-1">Error</span>
          </button>
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* State Filter */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">State</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleState('running')}
            aria-label="Running"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('running')
                ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Running services"
          >
            <CheckCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-emerald-700/95 text-white text-xs px-2 py-1">Running</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleState('stopped')}
            aria-label="Stopped"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('stopped')
                ? 'bg-slate-200 dark:bg-slate-600/40 text-slate-700 dark:text-slate-300 ring-1 ring-slate-300 dark:ring-slate-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Stopped services"
          >
            <Circle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-slate-700/95 text-white text-xs px-2 py-1">Stopped</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleState('starting')}
            aria-label="Starting"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              stateFilter.has('starting')
                ? 'bg-sky-100 dark:bg-sky-500/20 text-sky-700 dark:text-sky-300 ring-1 ring-sky-300 dark:ring-sky-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Starting services"
          >
            <Loader2 className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-sky-700/95 text-white text-xs px-2 py-1">Starting</span>
          </button>
        </div>
      </div>

      <div className="w-px bg-slate-300 dark:bg-slate-700 self-stretch" />

      {/* Health Status */}
      <div className="flex flex-col gap-2">
        <span className="text-xs font-medium text-slate-500">Health Status</span>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => onToggleHealth('healthy')}
            aria-label="Healthy"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('healthy')
                ? 'bg-emerald-100 dark:bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-300 dark:ring-emerald-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Healthy services"
          >
            <Heart className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-emerald-700/95 text-white text-xs px-2 py-1">Healthy</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('degraded')}
            aria-label="Degraded"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('degraded')
                ? 'bg-amber-100 dark:bg-amber-500/20 text-amber-700 dark:text-amber-300 ring-1 ring-amber-300 dark:ring-amber-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Degraded services"
          >
            <HeartPulse className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-amber-700/95 text-white text-xs px-2 py-1">Degraded</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('unhealthy')}
            aria-label="Unhealthy"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('unhealthy')
                ? 'bg-rose-100 dark:bg-rose-500/20 text-rose-700 dark:text-rose-300 ring-1 ring-rose-300 dark:ring-rose-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Unhealthy services"
          >
            <HeartCrack className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-rose-700/95 text-white text-xs px-2 py-1">Unhealthy</span>
          </button>
          <button
            type="button"
            onClick={() => onToggleHealth('unknown')}
            aria-label="Unknown"
            className={cn(
              'relative flex items-center justify-center w-9 h-9 rounded-md transition-all',
              healthFilter.has('unknown')
                ? 'bg-slate-200 dark:bg-slate-600/40 text-slate-700 dark:text-slate-300 ring-1 ring-slate-300 dark:ring-slate-500/50'
                : 'bg-transparent text-slate-400 hover:bg-slate-200/60 dark:hover:bg-slate-700/60'
            )}
            title="Toggle Unknown services"
          >
            <HelpCircle className="w-4 h-4" />
            <span className="sr-only group-hover:not-sr-only absolute -top-8 left-1/2 transform -translate-x-1/2 whitespace-nowrap rounded bg-slate-700/95 text-white text-xs px-2 py-1">Unknown</span>
          </button>
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
}: Readonly<ConsoleViewProps>) {
  const { services } = useServicesContext()
  const savedFilters = React.useMemo(() => loadSavedConsoleFilters(), [])
  const [serviceSelectionMode, setServiceSelectionMode] = React.useState<ConsoleServiceSelectionMode>(
    () => savedFilters?.serviceSelectionMode ?? 'all'
  )
  const [selectedServices, setSelectedServices] = React.useState<Set<string>>(
    () => new Set(serviceSelectionMode === 'custom' ? (savedFilters?.selectedServices ?? []) : [])
  )
  const [isPaused, setIsPaused] = React.useState(false)
  const [isFullscreen, setIsFullscreen] = React.useState(false)
  const [isSettingsOpen, setIsSettingsOpen] = React.useState(false)
  const [globalSearchTerm, setGlobalSearchTerm] = React.useState('')
  const [autoScrollEnabled, setAutoScrollEnabled] = React.useState(true)
  const [clearAllTrigger, setClearAllTrigger] = React.useState(0)
  const [levelFilter, setLevelFilter] = React.useState<Set<LogLevel>>(
    () => new Set(savedFilters?.levelFilter?.length ? savedFilters.levelFilter : ['info', 'warning', 'error'])
  )
  const [stateFilter, setStateFilter] = React.useState<Set<FilterableLifecycleState>>(
    () => new Set(savedFilters?.stateFilter?.length ? savedFilters.stateFilter : ['running', 'stopped', 'starting'])
  )
  const [healthFilter, setHealthFilter] = React.useState<Set<HealthStatus>>(
    () => new Set(savedFilters?.healthFilter?.length ? savedFilters.healthFilter : ['healthy', 'degraded', 'unhealthy', 'unknown'])
  )
  const [collapsedPanes, setCollapsedPanes] = React.useState<Record<string, boolean>>({})
  
  // Log source mode state (local vs azure)
  const [logMode, setLogMode] = React.useState<LogMode>('local')
  const [isModeSwitching, setIsModeSwitching] = React.useState(false)
  
  // Azure status from API
  const [azureEnabled, setAzureEnabled] = React.useState(false)
  const [azureStatus, setAzureStatus] = React.useState<AzureConnectionStatus>('disabled')
  const [azureConnectionMessage, setAzureConnectionMessage] = React.useState<string | undefined>(undefined)
  
  // Azure logs settings
  const [timeRange, setTimeRange] = React.useState<{ preset: TimeRangePreset }>({ preset: '15m' })
  const [syncInterval, setSyncInterval] = React.useState<number>(() => getSavedSyncInterval())
  const [azureRealtime, setAzureRealtime] = React.useState<boolean>(() => getSavedAzureRealtime())
  const [showDiagnostics, setShowDiagnostics] = React.useState(false)

  const azureRealtimeInitializedRef = React.useRef(false)

  const maybeInitializeAzureRealtimeFromConfig = React.useCallback((azureRealtimeFromConfig: boolean | undefined) => {
    if (azureRealtimeInitializedRef.current) {
      return
    }

    azureRealtimeInitializedRef.current = true

    try {
      const hasSavedPreference = globalThis.localStorage?.getItem('azure-logs-realtime') !== null
      if (!hasSavedPreference && typeof azureRealtimeFromConfig === 'boolean') {
        setAzureRealtime(azureRealtimeFromConfig)
      }
    } catch {
      // Ignore localStorage errors
    }
  }, [])

  // Fetch Azure status from API
  const fetchAzureStatus = React.useCallback(async () => {
    try {
      const res = await fetch('/api/mode')
      if (res.ok) {
        const raw: unknown = await res.json()
        const data = parseModeApiResponse(raw)

        // Set the current mode from backend (important for initial page load)
        if (data.mode) {
          setLogMode(data.mode)
        }

        const enabled = data.azureEnabled ?? false
        setAzureEnabled(enabled)
        setAzureConnectionMessage(data.connectionMessage)

        // Default realtime toggle from config unless user has explicitly set a preference.
        maybeInitializeAzureRealtimeFromConfig(data.azureRealtime)

        if (enabled) {
          setAzureStatus(data.azureStatus ?? 'disconnected')
        } else {
          setAzureStatus('disabled')
        }
      }
    } catch {
      // Ignore errors - status will remain disabled
    }
  }, [maybeInitializeAzureRealtimeFromConfig])

  // Handle mode change with loading indicator
  const handleLogModeChange = React.useCallback((newMode: LogMode) => {
    void (async () => {
      if (newMode === logMode) {
        return
      }

      setIsModeSwitching(true)

      try {
        // Call backend API to switch mode - this starts/stops Azure polling
        const res = await fetch('/api/mode', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: newMode }),
        })

        if (res.ok) {
          setLogMode(newMode)
          // Refresh Azure status after mode change
          await fetchAzureStatus()
        } else {
          const errorText = await res.text()
          console.error('[ConsoleView] Failed to switch mode:', errorText)
        }
      } catch (err) {
        console.error('Error switching mode:', err)
      } finally {
        // Clear switching state after a short delay to let panes reconnect
        setTimeout(() => setIsModeSwitching(false), 1500)
      }
    })()
  }, [logMode, fetchAzureStatus])

  // Fetch Azure status on mount and when services change
  React.useEffect(() => {
    void fetchAzureStatus()
  }, [fetchAzureStatus, services])

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

  React.useEffect(() => {
    if (services.length > 0) {
      const currentServiceNames = new Set(services.map((s) => s.name))

      if (serviceSelectionMode === 'all') {
        setSelectedServices(currentServiceNames)
        return
      }

      // Custom selection: preserve user's choices, but drop services that no longer exist.
      setSelectedServices((prev) => {
        const next = new Set(Array.from(prev).filter((name) => currentServiceNames.has(name)))
        return next.size > 0 ? next : currentServiceNames
      })
    }
  }, [services, serviceSelectionMode])

  // Persist filter state to localStorage.
  React.useEffect(() => {
    const currentServiceNames = new Set(services.map((s) => s.name))

    const isAllSelected =
      currentServiceNames.size > 0 &&
      selectedServices.size === currentServiceNames.size &&
      Array.from(selectedServices).every((name) => currentServiceNames.has(name))

    saveConsoleFilters({
      version: 1,
      serviceSelectionMode: isAllSelected ? 'all' : 'custom',
      selectedServices: isAllSelected ? undefined : Array.from(selectedServices).sort((a, b) => a.localeCompare(b)),
      levelFilter: Array.from(levelFilter),
      stateFilter: Array.from(stateFilter),
      healthFilter: Array.from(healthFilter),
    })
  }, [services, selectedServices, levelFilter, stateFilter, healthFilter])

  // Keyboard shortcuts
  React.useEffect(() => {
    const isEditableTarget = (target: EventTarget | null): boolean => {
      const el = target as HTMLElement | null
      if (!el) return false
      if (el.isContentEditable) return true
      return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA'
    }

    const isSpaceToggle = (e: KeyboardEvent): boolean => {
      return e.code === 'Space' && !e.ctrlKey && !e.shiftKey && !e.altKey && !e.metaKey
    }

    const isToggleViewMode = (e: KeyboardEvent): boolean => {
      return e.ctrlKey && e.shiftKey && e.code === 'KeyL'
    }

    const isToggleLogModeShortcut = (e: KeyboardEvent): boolean => {
      return e.ctrlKey && e.shiftKey && e.code === 'KeyM'
    }

    const isToggleFullscreen = (e: KeyboardEvent): boolean => {
      return e.key === 'F11' || (e.ctrlKey && e.shiftKey && e.code === 'KeyF')
    }

    const isExitFullscreen = (e: KeyboardEvent): boolean => {
      return e.key === 'Escape' && isFullscreen
    }

    const handleKeyDown = (e: KeyboardEvent) => {
      if (isSpaceToggle(e)) {
        if (!isEditableTarget(e.target)) {
          e.preventDefault()
          setIsPaused((prev) => !prev)
        }
        return
      }

      if (isToggleViewMode(e)) {
        e.preventDefault()
        updateUI({ viewMode: viewMode === 'grid' ? 'unified' : 'grid' })
        return
      }

      if (isToggleLogModeShortcut(e)) {
        e.preventDefault()
        if (azureEnabled) {
          handleLogModeChange(logMode === 'local' ? 'azure' : 'local')
        }
        return
      }

      if (isToggleFullscreen(e)) {
        e.preventDefault()
        setIsFullscreen((prev) => !prev)
        return
      }

      if (isExitFullscreen(e)) {
        setIsFullscreen(false)
      }
    }

    globalThis.addEventListener('keydown', handleKeyDown)
    return () => globalThis.removeEventListener('keydown', handleKeyDown)
  }, [viewMode, updateUI, isFullscreen, azureEnabled, logMode, handleLogModeChange])

  const handleToggleService = (serviceName: string) => {
    setServiceSelectionMode('custom')
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

  const toggleLevelFilter = (level: LogLevel) => {
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

  const handleSyncIntervalChange = React.useCallback((value: number) => {
    const clamped = clampSyncInterval(value)
    setSyncInterval(clamped)
    setSavedSyncInterval(clamped)
  }, [])

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
          logs.map(log => `"${log.service}","${log.timestamp}",${log.level},"${log.message.replaceAll('"', '""')}"`).join('\n')
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

  // Pane visibility is controlled only by explicit service selection.
  // State/health filters must not hide panes (see docs/specs/log-pane-visibility/spec.md).
  const paneServicesList = selectedServicesList

  let content: React.ReactNode

  if (viewMode === 'grid') {
    if (paneServicesList.length === 0) {
      content = (
        <div className="flex items-center justify-center h-full text-slate-500">
          <div className="text-center">
            <p className="text-lg font-medium">No services selected</p>
            <p className="text-sm mt-2">Select one or more services to view their logs</p>
          </div>
        </div>
      )
    } else {
      content = (
        <LogsPaneGrid columns={2} collapsedPanes={collapsedPanes} autoFit={true}>
          {paneServicesList.map((serviceName) => {
            const service = services.find((s) => s.name === serviceName)
            const serviceHealthStatus = healthReport?.services.find(
              (s) => s.serviceName === serviceName
            )?.status
            // Services with host: local always show local logs regardless of global mode
            const effectiveLogMode = service?.host === 'local' ? 'local' : logMode
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
                logMode={effectiveLogMode}
                isModeSwitching={isModeSwitching}
                timeRange={effectiveLogMode === 'azure' ? timeRange : undefined}
                syncInterval={syncInterval}
                azureRealtime={azureRealtime}
              />
            )
          })}
        </LogsPaneGrid>
      )
    }
  } else {
    content = (
      <LogsView
        selectedServices={selectedServices}
        levelFilter={levelFilter}
        isPaused={isPaused}
        autoScrollEnabled={autoScrollEnabled}
        globalSearchTerm={globalSearchTerm}
        clearAllTrigger={clearAllTrigger}
        hideControls={true}
        logMode={logMode}
        isModeSwitching={isModeSwitching}
        timeRange={logMode === 'azure' ? timeRange : undefined}
        syncInterval={syncInterval}
        azureRealtime={azureRealtime}
      />
    )
  }

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
        logMode={logMode}
        onLogModeChange={handleLogModeChange}
        azureEnabled={azureEnabled}
        azureStatus={azureStatus}
        azureConnectionMessage={azureConnectionMessage}
        timeRange={timeRange}
        onTimeRangeChange={(preset) => setTimeRange(prev => (prev.preset === preset ? prev : { preset }))}
        syncInterval={syncInterval}
        onSyncIntervalChange={handleSyncIntervalChange}
        onRunDiagnostics={() => setShowDiagnostics(true)}
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
        healthReport={healthReport}
      />

      {/* Content - Constrain to remaining viewport height */}
      <div className="flex-1 overflow-hidden min-h-0">
        {content}
      </div>

      {/* Diagnostics Modal */}
      <DiagnosticsModal
        isOpen={showDiagnostics}
        onClose={() => setShowDiagnostics(false)}
      />

      {/* Settings Dialog */}
      <SettingsDialog
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
      />
    </div>
  )
}
