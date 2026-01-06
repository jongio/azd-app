import type { ReactNode } from 'react'
import { Copy, AlertTriangle, Info, XCircle, Check } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import { convertAnsiToHtml, stripEmbeddedTimestamp } from '@/lib/log-utils'
import { highlightSearchTermInHtml } from '@/lib/search-highlight'
import type { LogEntry } from '@/components/LogsPane'
import type { PaneLogLevel } from '@/hooks/useLogFiltering'
import type { LogMode } from './ModeToggle'
import type { AzureTimeRange } from '@/hooks/useAzureTimeRange'
import type { CodespaceConfig } from '@/lib/codespace-utils'
import { LogsPaneEmptyState } from './LogsPaneEmptyState'

export interface LogsPaneContentProps {
  isCollapsed: boolean
  logsContainerRef: React.RefObject<HTMLDivElement | null>
  setIsHovering: (value: boolean) => void
  filteredLogs: LogEntry[]
  logs: LogEntry[]
  logMode: LogMode
  codespaceConfig: CodespaceConfig
  isLoading: boolean
  isWaiting: boolean
  errorMessage: string | null
  timeRange: AzureTimeRange
  getPaneLogLevel: (log: LogEntry) => PaneLogLevel
  copiedLineIndex: number | null
  handleCopyLine: (log: LogEntry, index?: number) => void
  logsEndRef: React.RefObject<HTMLDivElement | null>
  loadingMessage?: string
  canRetry?: boolean
  onRetry?: () => void
  serviceName?: string
  globalSearchTerm?: string
  onOpenDiagnostics?: () => void
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
  loadingMessage,
  canRetry,
  onRetry,
  serviceName,
  globalSearchTerm = '',
  onOpenDiagnostics,
}: Readonly<LogsPaneContentProps>): ReactNode {
  if (isCollapsed) {
    return null
  }

  // Only show empty state when we truly have no logs to display
  // Don't hide existing logs just because we're in a waiting/loading state
  const showEmptyState = filteredLogs.length === 0

  return (
    <div
      ref={logsContainerRef}
      className="flex-1 overflow-y-auto bg-card p-4 font-mono text-sm leading-relaxed"
      role="log"
      aria-live="polite"
      aria-atomic="false"
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      {showEmptyState ? (
        <LogsPaneEmptyState
          errorMessage={errorMessage}
          isLoading={isLoading}
          isWaiting={isWaiting}
          logMode={logMode}
          timeRange={timeRange}
          hasLogs={logs.length > 0}
          loadingMessage={loadingMessage}
          canRetry={canRetry}
          onRetry={onRetry}
          serviceName={serviceName}
          onOpenDiagnostics={onOpenDiagnostics}
        />
      ) : (
        <div className="space-y-0.5">
          {filteredLogs.map((log, idx) => {
            const logLevel = getPaneLogLevel(log)
            const logKey = `${log.timestamp}-${log.service}-${idx}`
            const formattedTimestamp = formatLogTimestamp(log.timestamp ?? '')
            const cleanedMessage = stripEmbeddedTimestamp(log.message ?? '')
            const serviceLabel = log.service ? ` | ${log.service}` : ''

            // Convert ANSI to HTML first, then apply search highlighting
            const htmlMessage = convertAnsiToHtml(cleanedMessage, codespaceConfig)
            const highlightedMessage = highlightSearchTermInHtml(htmlMessage, globalSearchTerm)

            return (
              <div
                key={logKey}
                className="relative group flex items-start gap-1 hover:bg-muted/50 px-1 -mx-1 rounded"
              >
                {logLevel === 'error' && (
                  <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0 mt-0.5 pointer-events-none" aria-label="Error" />
                )}
                {logLevel === 'warning' && (
                  <AlertTriangle className="w-3.5 h-3.5 text-yellow-500 shrink-0 mt-0.5 pointer-events-none" aria-label="Warning" />
                )}
                {logLevel === 'info' && (
                  <Info className="w-3.5 h-3.5 text-blue-500 shrink-0 mt-0.5 pointer-events-none" aria-label="Info" />
                )}
                
                <div className="flex-1 min-w-0 select-text">
                  <span className="text-slate-700 dark:text-slate-200 text-sm font-medium">
                    [{formattedTimestamp}{serviceLabel}]
                  </span>
                  {' '}
                  <span className="text-foreground" dangerouslySetInnerHTML={{ __html: highlightedMessage }} />
                </div>
                
                <button
                  type="button"
                  onClick={() => {
                    // Only trigger copy if there's no text selected
                    const selection = window.getSelection()
                    if (!selection || selection.toString().length === 0) {
                      handleCopyLine(log, idx)
                    }
                  }}
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
                  <span className="absolute right-8 top-0 text-xs text-green-500 bg-background px-1 rounded shadow pointer-events-none">Copied!</span>
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
