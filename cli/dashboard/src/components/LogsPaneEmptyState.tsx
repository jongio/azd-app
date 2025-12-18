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

export interface LogsPaneEmptyStateProps {
  errorMessage: string | null
  isLoading: boolean
  isWaiting: boolean
  logMode: LogMode
  timeRange: AzureTimeRange
  hasLogs: boolean
}

export function LogsPaneEmptyState({
  errorMessage,
  isLoading,
  isWaiting,
  logMode,
  timeRange,
  hasLogs,
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
          {logMode === 'azure' ? 'Fetching Azure logs...' : 'Fetching local logs...'}
        </div>
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

    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="text-sm font-medium text-foreground mb-1">No logs in the selected time range</div>
        <div className="text-sm text-muted-foreground max-w-sm">
          {bounds ? (
            <>
              No logs were returned between <span className="font-mono">{formatAzureRangeTimestamp(bounds.start)}</span> and{' '}
              <span className="font-mono">{formatAzureRangeTimestamp(bounds.end)}</span>.
            </>
          ) : (
            <>No logs were returned for the selected timeframe.</>
          )}
        </div>
        <div className="text-sm text-muted-foreground max-w-sm mt-2">
          Current timeframe: {current}. Try changing the timeframe to {suggestion}.
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
