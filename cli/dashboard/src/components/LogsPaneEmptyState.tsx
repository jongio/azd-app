import type { ReactNode } from 'react'
import { XCircle, Loader2 } from 'lucide-react'
import type { LogMode } from './ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import { 
  formatAzureRangeTimestamp, 
  formatAzureTimeRangePreset, 
  suggestAzureTimeRangePreset,
  getAzureTimeRangeBounds 
} from '@/hooks/useAzureTimeRange'
import { NoLogsPrompt } from './NoLogsPrompt'

export interface LogsPaneEmptyStateProps {
  errorMessage: string | null
  isLoading: boolean
  isWaiting: boolean
  logMode: LogMode
  timeRange: AzureTimeRange
  hasLogs: boolean
  loadingMessage?: string
  canRetry?: boolean
  onRetry?: () => void
  serviceName?: string
  onOpenDiagnostics?: () => void
}

export function LogsPaneEmptyState({
  errorMessage,
  isLoading,
  isWaiting,
  logMode,
  timeRange,
  hasLogs,
  loadingMessage,
  canRetry,
  onRetry,
  serviceName,
  onOpenDiagnostics,
}: Readonly<LogsPaneEmptyStateProps>): ReactNode {
  if (errorMessage) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="flex items-center gap-2 text-sm font-medium text-red-600 dark:text-red-400 mb-2" role="alert">
          <XCircle className="w-4 h-4" aria-hidden="true" />
          <span>{logMode === 'azure' ? 'Failed to load Azure logs' : 'Failed to load local logs'}</span>
        </div>
        <div className="text-sm text-muted-foreground max-w-sm">
          {errorMessage}
        </div>
      </div>
    )
  }

  // Show loading spinner if either loading or waiting (prevents flashing)
  if (isLoading || isWaiting) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
        <Loader2 className="w-5 h-5 animate-spin mb-2" />
        <div className="text-sm">
          {loadingMessage || (logMode === 'azure' ? 'Fetching Azure logs...' : 'Fetching local logs...')}
        </div>
      </div>
    )
  }

  // Show retry option if service never started
  if (canRetry && !hasLogs && logMode === 'local') {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="text-slate-500 dark:text-slate-400 mb-4">
          <svg className="w-12 h-12 mx-auto mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-sm font-medium">
            No logs available {serviceName && `for ${serviceName}`}
          </p>
          <p className="text-xs mt-1">
            Service may not be running locally or hasn't started yet
          </p>
        </div>
        <button
          onClick={onRetry}
          className="px-4 py-2 text-sm bg-slate-800 hover:bg-slate-700 dark:bg-slate-700 dark:hover:bg-slate-600 text-slate-200 rounded-md transition-colors flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Retry
        </button>
      </div>
    )
  }

  if (hasLogs) {
    return (
      <div className="text-center text-muted-foreground py-12">
        No logs match your search
      </div>
    )
  }

  if (logMode === 'azure') {
    const now = new Date()
    const current = formatAzureTimeRangePreset(timeRange.preset)
    const suggestion = suggestAzureTimeRangePreset(timeRange.preset)
    const bounds = getAzureTimeRangeBounds(timeRange, now)

    // Show NoLogsPrompt for services with 0 logs (potential configuration issue)
    if (!hasLogs && serviceName) {
      return (
        <NoLogsPrompt
          serviceName={serviceName}
          onOpenDiagnostics={onOpenDiagnostics}
        />
      )
    }

    // Show time range suggestion when logs exist but not in current range
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="text-sm font-medium text-foreground mb-2">No logs in the selected time range</div>
        {bounds && (
          <div className="text-sm text-muted-foreground max-w-sm mb-2">
            <span className="font-mono">{formatAzureRangeTimestamp(bounds.start)}</span>
            {' → '}
            <span className="font-mono">{formatAzureRangeTimestamp(bounds.end)}</span>
          </div>
        )}
        <div className="text-sm text-muted-foreground max-w-sm">
          Try changing from {current} to {suggestion}.
        </div>
      </div>
    )
  }

  return (
    <div className="text-center text-muted-foreground py-12">
      No logs to display
    </div>
  )
}
