import type { ReactNode } from 'react'
import { Copy, AlertTriangle, Info, XCircle, Check } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import { convertAnsiToHtml, stripEmbeddedTimestamp } from '@/lib/log-utils'
import type { LogEntry } from '@/components/LogsPane'
import type { PaneLogLevel } from '@/hooks/useLogFiltering'
import type { LogMode } from './ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import { LogsPaneEmptyState } from './LogsPaneEmptyState'

export interface LogsPaneContentProps {
  isCollapsed: boolean
  logsContainerRef: React.RefObject<HTMLDivElement | null>
  setIsHovering: (value: boolean) => void
  filteredLogs: LogEntry[]
  logs: LogEntry[]
  logMode: LogMode
  codespaceConfig: { isCodespace: boolean; domain?: string; name?: string }
  isLoading: boolean
  isWaiting: boolean
  errorMessage: string | null
  timeRange: AzureTimeRange
  getPaneLogLevel: (log: LogEntry) => PaneLogLevel
  copiedLineIndex: number | null
  handleCopyLine: (log: LogEntry, index?: number) => void
  logsEndRef: React.RefObject<HTMLDivElement | null>
}

export function LogsPaneContent({
  isCollapsed,
  logsContainerRef,
  setIsHovering,
  filteredLogs,
  logs,
  logMode,
  codespaceConfig,
  isLoading,
  isWaiting,
  errorMessage,
  timeRange,
  getPaneLogLevel,
  copiedLineIndex,
  handleCopyLine,
  logsEndRef,
}: Readonly<LogsPaneContentProps>): ReactNode {
  if (isCollapsed) {
    return null
  }

  return (
    <div
      ref={logsContainerRef}
      className="flex-1 overflow-y-auto bg-card p-4 font-mono text-sm"
      role="log"
      aria-live="polite"
      aria-atomic="false"
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {filteredLogs.length === 0 ? (
        <LogsPaneEmptyState
          errorMessage={errorMessage}
          isLoading={isLoading}
          isWaiting={isWaiting}
          logMode={logMode}
          timeRange={timeRange}
          hasLogs={logs.length > 0}
        />
      ) : (
        <div className="space-y-0.5">
          {filteredLogs.map((log, idx) => {
            const logLevel = getPaneLogLevel(log)
            const logKey = `${log.timestamp}-${log.service}-${idx}`
            const formattedTimestamp = formatLogTimestamp(log.timestamp ?? '')
            const cleanedMessage = stripEmbeddedTimestamp(log.message ?? '')
            const serviceLabel = log.service ? ` | ${log.service}` : ''

            return (
              <div
                key={logKey}
                className="relative group flex items-start gap-1 hover:bg-muted/50 px-1 -mx-1 rounded"
              >
                {logLevel === 'error' && (
                  <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0 mt-0.5" aria-label="Error" />
                )}
                {logLevel === 'warning' && (
                  <AlertTriangle className="w-3.5 h-3.5 text-yellow-500 shrink-0 mt-0.5" aria-label="Warning" />
                )}
                {logLevel === 'info' && (
                  <Info className="w-3.5 h-3.5 text-blue-500 shrink-0 mt-0.5" aria-label="Info" />
                )}
                
                <div className="flex-1 min-w-0 select-text">
                  <span className="text-muted-foreground text-xs">
                    [{formattedTimestamp}{serviceLabel}]
                  </span>
                  {' '}
                  <span dangerouslySetInnerHTML={{ __html: convertAnsiToHtml(
                    cleanedMessage,
                    codespaceConfig
                  ) }} />
                </div>
                
                <button
                  type="button"
                  onClick={() => handleCopyLine(log, idx)}
                  className="opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100 shrink-0 p-1 hover:bg-muted rounded transition-opacity"
                  title="Copy log line"
                  aria-label="Copy this log line"
                >
                  {copiedLineIndex === idx ? (
                    <Check className="w-3.5 h-3.5 text-green-500" />
                  ) : (
                    <Copy className="w-3.5 h-3.5 text-muted-foreground hover:text-foreground" />
                  )}
                </button>
                {copiedLineIndex === idx && (
                  <span className="absolute right-8 top-0 text-xs text-green-500 bg-background px-1 rounded shadow">Copied!</span>
                )}
              </div>
            )
          })}
          <div ref={logsEndRef} />
        </div>
      )}
    </div>
  )
}
