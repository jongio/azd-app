import * as React from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  LayoutGrid,
  List,
  Search,
  Pause,
  Play,
  ChevronsDown,
  RotateCw,
  Square,
  Trash2,
  Download,
  Settings,
  Maximize2,
  Minimize2,
  Columns,
  Minus,
  Plus,
} from 'lucide-react'
import { cn } from '@/lib/utils'

export interface LogsToolbarProps {
  /** Current view mode */
  viewMode: 'grid' | 'unified'
  /** Callback when view mode changes */
  onViewModeChange: (mode: 'grid' | 'unified') => void
  /** Whether fullscreen mode is active */
  isFullscreen: boolean
  /** Callback when fullscreen changes */
  onFullscreenChange: (isFullscreen: boolean) => void
  /** Whether log stream is paused */
  isPaused: boolean
  /** Callback when pause state changes */
  onPauseChange: (isPaused: boolean) => void
  /** Whether auto-scroll is enabled */
  autoScrollEnabled: boolean
  /** Callback when auto-scroll changes */
  onAutoScrollChange: (enabled: boolean) => void
  /** Current search term */
  searchTerm: string
  /** Callback when search term changes */
  onSearchChange: (term: string) => void
  /** Callback to clear all logs */
  onClearAll: () => void
  /** Callback to export all logs */
  onExportAll: () => void
  /** Callback to open settings */
  onOpenSettings: () => void
  /** Callback to start all services */
  onStartAll: () => Promise<unknown>
  /** Callback to stop all services */
  onStopAll: () => Promise<unknown>
  /** Callback to restart all services */
  onRestartAll: () => Promise<unknown>
  /** Check if bulk operation is in progress */
  isBulkOperationInProgress: () => boolean
  /** Current bulk operation type */
  bulkOperation?: 'start' | 'stop' | 'restart' | null
  /** Current grid columns count */
  gridColumns: number
  /** Callback when grid columns changes */
  onGridColumnsChange: (columns: number) => void
}

/**
 * Redesigned toolbar for the logs multi-pane view.
 * All controls are directly accessible in the toolbar, organized into logical groups.
 * 
 * Layout Groups:
 * 1. View Controls: [Grid][Unified] | [Columns: -/+]
 * 2. Search: [Search...]
 * 3. Stream Controls: [Pause][Scroll]
 * 4. Service Actions: [Start All][Restart All][Stop All]
 * 5. Log Actions: [Clear][Export]
 * 6. View Actions: [Fullscreen][Settings]
 */
export function LogsToolbar({
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
  onExportAll,
  onOpenSettings,
  onStartAll,
  onStopAll,
  onRestartAll,
  isBulkOperationInProgress,
  bulkOperation,
  gridColumns,
  onGridColumnsChange,
}: LogsToolbarProps) {
  const handleStartAll = React.useCallback(() => {
    void onStartAll()
  }, [onStartAll])

  const handleStopAll = React.useCallback(() => {
    void onStopAll()
  }, [onStopAll])

  const handleRestartAll = React.useCallback(() => {
    void onRestartAll()
  }, [onRestartAll])

  const isOperationInProgress = isBulkOperationInProgress()

  return (
    <div
      role="toolbar"
      aria-label="Log viewer controls"
      className="flex flex-wrap gap-3 items-center p-3 bg-card border-b shrink-0"
    >
      {/* Group 1: View Mode & Grid Columns */}
      <div className="flex items-center gap-2">
        {/* View Mode Toggle */}
        <div
          role="group"
          aria-label="View mode"
          className="flex gap-0.5 border rounded-lg p-0.5"
        >
          <Button
            variant={viewMode === 'grid' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => onViewModeChange('grid')}
            aria-pressed={viewMode === 'grid'}
            title="Grid View (Ctrl+Shift+L)"
            className="h-7 px-2"
          >
            <LayoutGrid className="w-4 h-4" />
            <span className="hidden sm:inline ml-1.5">Grid</span>
          </Button>
          <Button
            variant={viewMode === 'unified' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => onViewModeChange('unified')}
            aria-pressed={viewMode === 'unified'}
            title="Unified View (Ctrl+Shift+L)"
            className="h-7 px-2"
          >
            <List className="w-4 h-4" />
            <span className="hidden sm:inline ml-1.5">Unified</span>
          </Button>
        </div>

        {/* Grid Columns Control - Only show in grid mode */}
        {viewMode === 'grid' && (
          <div
            role="group"
            aria-label="Grid columns"
            className="flex items-center gap-1 border rounded-lg p-0.5"
          >
            <Columns className="w-4 h-4 ml-1.5 text-muted-foreground" />
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onGridColumnsChange(Math.max(1, gridColumns - 1))}
              disabled={gridColumns <= 1}
              aria-label="Decrease columns"
              title="Decrease columns"
              className="h-7 w-7"
            >
              <Minus className="!w-4 !h-4" strokeWidth={2.5} />
            </Button>
            <span className="w-5 text-center text-sm font-medium" aria-live="polite">
              {gridColumns}
            </span>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onGridColumnsChange(Math.min(6, gridColumns + 1))}
              disabled={gridColumns >= 6}
              aria-label="Increase columns"
              title="Increase columns"
              className="h-7 w-7"
            >
              <Plus className="!w-4 !h-4" strokeWidth={2.5} />
            </Button>
          </div>
        )}
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-border hidden sm:block" />

      {/* Group 2: Search */}
      <div className="relative min-w-[160px] flex-1 max-w-xs">
        <Search className="absolute left-2 top-2 w-4 h-4 text-muted-foreground" />
        <Input
          placeholder="Search logs..."
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-8 h-8"
          aria-label="Search logs"
        />
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-border hidden sm:block" />

      {/* Group 3: Stream Controls */}
      <div className="flex items-center gap-1">
        <Button
          variant={isPaused ? 'default' : 'outline'}
          size="sm"
          onClick={() => onPauseChange(!isPaused)}
          aria-pressed={isPaused}
          aria-label={isPaused ? 'Resume log stream' : 'Pause log stream'}
          title={`${isPaused ? 'Resume' : 'Pause'} (Space)`}
          className="h-8"
        >
          {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          <span className="hidden sm:inline ml-1.5">{isPaused ? 'Resume' : 'Pause'}</span>
        </Button>

        <Button
          variant={autoScrollEnabled ? 'default' : 'outline'}
          size="sm"
          onClick={() => onAutoScrollChange(!autoScrollEnabled)}
          aria-pressed={autoScrollEnabled}
          aria-label={autoScrollEnabled ? 'Disable auto-scroll' : 'Enable auto-scroll'}
          title={autoScrollEnabled ? 'Stop auto-scroll' : 'Enable auto-scroll'}
          className="h-8"
        >
          <ChevronsDown className="w-4 h-4" />
          <span className="hidden sm:inline ml-1.5">Scroll</span>
        </Button>
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-border hidden sm:block" />

      {/* Group 4: Service Actions */}
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          onClick={handleStartAll}
          disabled={isOperationInProgress}
          title="Start All (Ctrl+Shift+S)"
          className="h-8 text-green-600 hover:text-green-700 hover:bg-green-50 dark:hover:bg-green-950"
        >
          <Play className="w-4 h-4" />
          <span className="hidden md:inline ml-1.5">Start All</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handleRestartAll}
          disabled={isOperationInProgress}
          title="Restart All (Ctrl+Shift+R)"
          className={cn(
            "h-8 text-yellow-600 hover:text-yellow-700 hover:bg-yellow-50 dark:hover:bg-yellow-950",
            bulkOperation === 'restart' && "opacity-70"
          )}
        >
          <RotateCw className={cn("w-4 h-4", bulkOperation === 'restart' && "animate-spin")} />
          <span className="hidden md:inline ml-1.5">Restart All</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handleStopAll}
          disabled={isOperationInProgress}
          title="Stop All (Ctrl+Shift+X)"
          className="h-8 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950"
        >
          <Square className="w-4 h-4" />
          <span className="hidden md:inline ml-1.5">Stop All</span>
        </Button>
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-border hidden sm:block" />

      {/* Group 5: Log Actions */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={onClearAll}
          title="Clear All Logs"
          className="h-8"
        >
          <Trash2 className="w-4 h-4" />
          <span className="hidden lg:inline ml-1.5">Clear</span>
        </Button>

        {!isFullscreen && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onExportAll}
            title="Export All Logs"
            className="h-8"
          >
            <Download className="w-4 h-4" />
            <span className="hidden lg:inline ml-1.5">Export</span>
          </Button>
        )}
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-border hidden sm:block" />

      {/* Group 6: View Actions */}
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onFullscreenChange(!isFullscreen)}
          title={isFullscreen ? 'Exit Fullscreen (F11)' : 'Enter Fullscreen (F11)'}
          className="h-8"
        >
          {isFullscreen ? <Minimize2 className="w-4 h-4" /> : <Maximize2 className="w-4 h-4" />}
        </Button>

        <Button
          variant="ghost"
          size="sm"
          onClick={onOpenSettings}
          title="Settings (Ctrl+,)"
          className="h-8"
        >
          <Settings className="w-4 h-4" />
        </Button>
      </div>

      {/* Status Indicators - Right aligned */}
      {(isPaused || !autoScrollEnabled) && (
        <div className="flex items-center gap-3 ml-auto">
          {isPaused && (
            <div className="text-xs text-yellow-600 font-medium px-2 py-1 bg-yellow-50 dark:bg-yellow-950 rounded" role="status" aria-live="polite">
              ⏸ Paused
            </div>
          )}
          {!autoScrollEnabled && (
            <div className="text-xs text-blue-600 font-medium px-2 py-1 bg-blue-50 dark:bg-blue-950 rounded" role="status" aria-live="polite">
              ↑ Manual scroll
            </div>
          )}
        </div>
      )}
    </div>
  )
}
