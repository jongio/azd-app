/**
 * ConsoleToolbar - Main toolbar for console with controls and actions
 */
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
  Activity,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ModeToggle, type LogMode } from './ModeToggle'
import { Select } from '@/components/ui/select'
import type { AzureConnectionStatus } from '@/hooks/useAzureConnectionStatus'
import type { SetupStep } from './AzureSetupGuide'

// =============================================================================
// Types
// =============================================================================

export type ViewMode = 'grid' | 'unified'
export type TimeRangePreset = '15m' | '30m' | '6h' | '24h'

export interface ConsoleToolbarProps {
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
  azureRealtime: boolean
  onAzureRealtimeChange: (enabled: boolean) => void
  onRunDiagnostics: () => void
  onOpenSetupGuide: () => void
  onOpenSetupGuideWithStep?: (step: SetupStep) => void
}

// =============================================================================
// Component
// =============================================================================

export function ConsoleToolbar({
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
  onRunDiagnostics,
  onOpenSetupGuide,
  onOpenSetupGuideWithStep: _onOpenSetupGuideWithStep,
}: Readonly<ConsoleToolbarProps>) {
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
          <span>Scroll</span>
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
          onOpenSetupGuide={onOpenSetupGuide}
        />

        {/* Azure Log Controls - Show when Azure mode is active and configured */}
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

            {/* Realtime/polling toggle is temporarily removed; tracked in docs/specs/azure-logs/tasks.md. */}
          </>
        )}

        {/* Diagnostics button - Always show in Azure mode (even when not configured) */}
        {logMode === 'azure' && (
          <button
            type="button"
            onClick={onRunDiagnostics}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium bg-azure-100 dark:bg-azure-500/20 text-azure-700 dark:text-azure-300 hover:bg-azure-200 dark:hover:bg-azure-500/30 transition-colors border border-azure-300 dark:border-azure-700"
            title="Run Azure logs diagnostics"
          >
            <Activity className="w-3.5 h-3.5" />
            <span>Diagnostics</span>
          </button>
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
