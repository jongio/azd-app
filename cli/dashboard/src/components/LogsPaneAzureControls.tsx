import type { ReactNode } from 'react'
import { Monitor, Cloud, Loader2, Activity } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { LogMode } from './ModeToggle'

function getModeIndicator(logMode: LogMode, isModeSwitching: boolean): ReactNode {
  if (isModeSwitching) {
    return (
      <>
        <Loader2 className="w-3.5 h-3.5 animate-spin" />
        <span>Switching to {logMode === 'azure' ? 'Azure' : 'Local'} logs...</span>
      </>
    )
  }

  if (logMode === 'azure') {
    return (
      <>
        <Cloud className="w-3.5 h-3.5" />
        <span>Viewing Azure Logs</span>
        <span className="text-azure-500 dark:text-azure-400">•</span>
        <span className="text-azure-500/70 dark:text-azure-400/70">Live from Azure resources</span>
      </>
    )
  }

  return (
    <>
      <Monitor className="w-3.5 h-3.5" />
      <span>Viewing Local Logs</span>
      <span className="text-slate-400 dark:text-slate-500">•</span>
      <span className="text-slate-500/70 dark:text-slate-400/70">From local development server</span>
    </>
  )
}

export interface LogsPaneModeBarProps {
  isCollapsed: boolean
  logMode: LogMode
  isModeSwitching: boolean
  onOpenDiagnostics?: () => void
}

export function LogsPaneModeBar({
  isCollapsed,
  logMode,
  isModeSwitching,
  onOpenDiagnostics,
}: Readonly<LogsPaneModeBarProps>): ReactNode {
  if (isCollapsed) {
    return null
  }

  const modeIndicator = getModeIndicator(logMode, isModeSwitching)

  return (
    <div 
      className={cn(
        "flex items-center justify-between gap-2 px-3 py-1.5 text-xs font-medium border-b transition-colors",
        logMode === 'azure' 
          ? "bg-azure-50 dark:bg-azure-900/30 text-azure-700 dark:text-azure-300 border-azure-200 dark:border-azure-700"
          : "bg-slate-50 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700"
      )}
    >
      <div className="flex items-center gap-2">
        {modeIndicator}
      </div>
      
      {/* Diagnostics button - only show in Azure mode */}
      {logMode === 'azure' && onOpenDiagnostics && (
        <button
          type="button"
          onClick={onOpenDiagnostics}
          className={cn(
            "flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors",
            "bg-azure-100 dark:bg-azure-500/20 text-azure-700 dark:text-azure-300",
            "hover:bg-azure-200 dark:hover:bg-azure-500/30",
            "border border-azure-300 dark:border-azure-700",
            "focus:outline-none focus:ring-2 focus:ring-azure-500 focus:ring-offset-1",
            "focus:ring-offset-azure-50 dark:focus:ring-offset-azure-900/30"
          )}
          title="Run Azure logs diagnostics"
          aria-label="Open Azure logs diagnostics"
        >
          <Activity className="w-3.5 h-3.5" aria-hidden="true" />
          <span>Diagnostics</span>
        </button>
      )}
    </div>
  )
}
