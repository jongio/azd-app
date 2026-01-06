/**
 * HistoricalLogPanel - Slide-in panel for querying historical Azure logs
 * Provides time range selection, KQL query input, pagination, and export.
 * Follows design spec: cli/docs/design/components/historical-log-panel.md
 */
import * as React from 'react'
import { 
  X, 
  Download, 
  ChevronDown, 
  Loader2, 
  Inbox,
  Cloud,
  Settings2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { 
  useHistoricalLogs, 
  formatTimeRangeDisplay,
  type TimeRange,
  type TimeRangePreset,
  type HistoricalLogEntry,
} from '@/hooks/useHistoricalLogs'
import { TimeRangeSelector } from './TimeRangeSelector'
import { KqlQueryInput } from './KqlQueryInput'
import { AzureErrorDisplay } from './AzureErrorDisplay'
import { LogConfigPanel } from './LogConfigPanel'
import { formatLogTimestamp } from '@/lib/service-utils'
import { 
  convertAnsiToHtml, 
  isErrorLine, 
  isWarningLine,
  getServiceColor,
} from '@/lib/log-utils'

// =============================================================================
// Types
// =============================================================================

export interface HistoricalLogPanelProps {
  /** Service name to query logs for */
  serviceName: string
  /** Whether panel is visible */
  isOpen: boolean
  /** Close panel callback */
  onClose: () => void
  /** Default time range preset */
  defaultTimeRange?: TimeRangePreset
  /** Whether Azure is connected */
  azureConnected: boolean
  /** Additional class names */
  className?: string
}

type ExportFormat = 'json' | 'text' | 'csv'

// =============================================================================
// Constants
// =============================================================================

const PANEL_WIDTH = 480
const PAGE_SIZE = 100

// =============================================================================
// Helper Functions
// =============================================================================

function getLogColor(log: HistoricalLogEntry): string {
  // Check message content first for errors/warnings
  if (isErrorLine(log.message)) return 'text-red-400'
  if (isWarningLine(log.message)) return 'text-yellow-400'
  
  // Check log level and stderr
  if (log.isStderr || log.level === 3) return 'text-red-400'
  if (log.level === 2) return 'text-yellow-400'
  
  return 'text-slate-300'
}

function exportLogs(logs: HistoricalLogEntry[], serviceName: string, format: ExportFormat): void {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  let content: string
  let mimeType: string
  let extension: string

  switch (format) {
    case 'json':
      content = JSON.stringify(logs, null, 2)
      mimeType = 'application/json'
      extension = 'json'
      break
    
    case 'csv': {
      // CSV header
      const csvHeader = 'Timestamp,Service,Level,Message'
      const csvRows = logs.map(log => {
        // Escape quotes and wrap in quotes
        const message = `"${(log.message ?? '').replace(/"/g, '""')}"`
        return `${log.timestamp ?? ''},${log.service ?? ''},${log.level ?? ''},${message}`
      })
      content = [csvHeader, ...csvRows].join('\n')
      mimeType = 'text/csv'
      extension = 'csv'
      break
    }
    
    case 'text':
    default:
      content = logs
        .map(log => `[${log.timestamp ?? ''}] [${log.service ?? ''}] ${log.message ?? ''}`)
        .join('\n')
      mimeType = 'text/plain'
      extension = 'txt'
      break
  }

  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${serviceName}-logs-${timestamp}.${extension}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// =============================================================================
// Export Menu Component
// =============================================================================

interface ExportMenuProps {
  logs: HistoricalLogEntry[]
  serviceName: string
  disabled: boolean
}

function ExportMenu({ logs, serviceName, disabled }: ExportMenuProps) {
  const [isOpen, setIsOpen] = React.useState(false)
  const menuRef = React.useRef<HTMLDivElement>(null)

  // Close on click outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleExport = (format: ExportFormat) => {
    exportLogs(logs, serviceName, format)
    setIsOpen(false)
  }

  return (
    <div ref={menuRef} className="relative">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled || logs.length === 0}
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
          'text-slate-600 dark:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-700',
          'transition-colors duration-200',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500',
          'disabled:opacity-50 disabled:cursor-not-allowed',
        )}
        aria-expanded={isOpen}
        aria-haspopup="menu"
      >
        <Download className="w-4 h-4" />
        Export
        <ChevronDown className={cn('w-3 h-3 transition-transform', isOpen && 'rotate-180')} />
      </button>

      {isOpen && (
        <div 
          className={cn(
            'absolute right-0 top-full mt-1 z-50',
            'w-36 rounded-md shadow-lg',
            'bg-white dark:bg-slate-800',
            'border border-slate-200 dark:border-slate-700',
            'py-1',
          )}
          role="menu"
        >
          {[
            { format: 'json' as ExportFormat, label: 'JSON' },
            { format: 'text' as ExportFormat, label: 'Plain Text' },
            { format: 'csv' as ExportFormat, label: 'CSV' },
          ].map(({ format, label }) => (
            <button
              key={format}
              type="button"
              onClick={() => handleExport(format)}
              className={cn(
                'w-full px-4 py-2 text-left text-sm',
                'text-slate-700 dark:text-slate-200',
                'hover:bg-slate-100 dark:hover:bg-slate-700',
                'focus:outline-none focus:bg-slate-100 dark:focus:bg-slate-700',
              )}
              role="menuitem"
            >
              {label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// =============================================================================
// HistoricalLogPanel Component
// =============================================================================

export function HistoricalLogPanel({
  serviceName,
  isOpen,
  onClose,
  defaultTimeRange = '30m',
  azureConnected,
  className,
}: HistoricalLogPanelProps) {
  const panelRef = React.useRef<HTMLDivElement>(null)
  const logsContainerRef = React.useRef<HTMLDivElement>(null)

  // Time range state
  const [timeRange, setTimeRange] = React.useState<TimeRange>({ preset: defaultTimeRange })
  
  // KQL query state
  const [kqlQuery, setKqlQuery] = React.useState('')
  const [kqlCollapsed, setKqlCollapsed] = React.useState(true)
  
  // Config panel state
  const [configPanelOpen, setConfigPanelOpen] = React.useState(false)

  // Historical logs hook
  const {
    logs,
    total,
    hasMore,
    isLoading,
    error,
    azureError,
    executionTime,
    executeQuery,
    loadMore,
    clearResults,
    resetQuery,
  } = useHistoricalLogs({
    serviceName,
    pageSize: PAGE_SIZE,
  })

  // Close on Escape
  useEscapeKey(onClose, isOpen)

  // Focus management
  React.useEffect(() => {
    if (isOpen && panelRef.current) {
      const closeButton = panelRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Reset state when panel opens with new service
  React.useEffect(() => {
    if (isOpen) {
      setTimeRange({ preset: defaultTimeRange })
      setKqlQuery('')
      setKqlCollapsed(true)
      clearResults()
    }
  }, [isOpen, serviceName, defaultTimeRange, clearResults])

  // Execute query when time range preset changes (not for custom)
  const handleTimeRangeChange = React.useCallback((newTimeRange: TimeRange) => {
    setTimeRange(newTimeRange)
    
    // Auto-execute for preset changes (not custom - requires explicit apply)
    if (newTimeRange.preset !== 'custom' && azureConnected) {
      void executeQuery(newTimeRange, kqlQuery || undefined)
    } else if (newTimeRange.preset === 'custom' && newTimeRange.start && newTimeRange.end && azureConnected) {
      // For custom, execute when both dates are set (after Apply)
      void executeQuery(newTimeRange, kqlQuery || undefined)
    }
  }, [azureConnected, executeQuery, kqlQuery])

  // Handle Run Query from KQL input
  const handleRunQuery = React.useCallback(() => {
    if (azureConnected) {
      void executeQuery(timeRange, kqlQuery || undefined)
    }
  }, [azureConnected, executeQuery, timeRange, kqlQuery])

  // Handle Load More
  const handleLoadMore = React.useCallback(() => {
    void loadMore()
  }, [loadMore])

  // Handle retry after error
  const handleRetry = React.useCallback(() => {
    void executeQuery(timeRange, kqlQuery || undefined)
  }, [executeQuery, timeRange, kqlQuery])

  // Handle reset query (for query syntax errors)
  const handleResetQuery = React.useCallback(() => {
    setKqlQuery('')
    resetQuery()
  }, [resetQuery])

  // Handle switch to local logs
  const handleViewLocal = React.useCallback(() => {
    onClose()
  }, [onClose])

  // Handle config panel
  const handleOpenConfigPanel = React.useCallback(() => {
    setConfigPanelOpen(true)
  }, [])

  const handleCloseConfigPanel = React.useCallback(() => {
    setConfigPanelOpen(false)
  }, [])

  const handleConfigSaved = React.useCallback(() => {
    // Re-execute query with potentially new config
    if (azureConnected) {
      void executeQuery(timeRange, undefined) // Clear custom KQL, use new config
      setKqlQuery('') // Reset KQL input
    }
  }, [azureConnected, executeQuery, timeRange])

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="historical-panel-title"
        style={{ width: PANEL_WIDTH }}
        className={cn(
          'fixed right-0 top-0 z-50 h-screen',
          'bg-white dark:bg-slate-900',
          'border-l border-slate-200 dark:border-slate-700',
          'shadow-2xl',
          'flex flex-col',
          'animate-slide-in-right',
          className
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3 p-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-9 h-9 rounded-lg flex items-center justify-center bg-cyan-100 dark:bg-cyan-500/20 shrink-0">
              <Cloud className="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <div className="min-w-0">
              <h2
                id="historical-panel-title"
                className="text-lg font-semibold text-slate-900 dark:text-slate-100 truncate"
              >
                Historical Logs
              </h2>
              <p className="text-sm text-slate-500 dark:text-slate-400 truncate">
                {serviceName}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            {/* Configure Button - always available since local services can also write to Log Analytics */}
            <button
              type="button"
              onClick={handleOpenConfigPanel}
              className="p-2 rounded-lg text-slate-400 hover:text-cyan-600 dark:hover:text-cyan-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
              aria-label="Configure log sources"
              title="Configure log sources"
            >
              <Settings2 className="w-5 h-5" />
            </button>
            {/* Close Button */}
            <button
              type="button"
              data-close-button
              onClick={onClose}
              className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
              aria-label="Close panel"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Azure Not Connected Overlay */}
        {!azureConnected && (
          <div className="flex-1 flex items-center justify-center p-8">
            <div className="text-center">
              <Cloud className="w-12 h-12 mx-auto mb-4 text-slate-300 dark:text-slate-600" />
              <h3 className="text-lg font-medium text-slate-700 dark:text-slate-300 mb-2">
                Azure Not Connected
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 max-w-xs">
                Connect to Azure to view historical logs from Log Analytics.
              </p>
            </div>
          </div>
        )}

        {/* Main Content (when Azure is connected) */}
        {azureConnected && (
          <>
            {/* Query Controls */}
            <div className="p-4 space-y-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
              {/* Time Range Selector */}
              <TimeRangeSelector
                value={timeRange}
                onChange={handleTimeRangeChange}
                disabled={isLoading}
              />

              {/* KQL Query Input */}
              <KqlQueryInput
                value={kqlQuery}
                onChange={setKqlQuery}
                onRunQuery={handleRunQuery}
                isCollapsed={kqlCollapsed}
                onCollapsedChange={setKqlCollapsed}
                disabled={isLoading}
              />
            </div>

            {/* Results Section */}
            <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
              {/* Results Header */}
              <div className="flex items-center justify-between px-4 py-2 bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700 shrink-0">
                <div className="text-sm text-slate-600 dark:text-slate-300">
                  {isLoading && logs.length === 0 ? (
                    <span className="flex items-center gap-2">
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Querying...
                    </span>
                  ) : logs.length > 0 ? (
                    <span>
                      <strong>{logs.length}</strong>
                      {total > logs.length && <span> of {total}</span>}
                      {' logs '}
                      <span className="text-slate-400 dark:text-slate-500">
                        ({formatTimeRangeDisplay(timeRange)})
                      </span>
                    </span>
                  ) : (
                    <span className="text-slate-400 dark:text-slate-500">
                      Select a time range to query logs
                    </span>
                  )}
                </div>
                <ExportMenu logs={logs} serviceName={serviceName} disabled={isLoading} />
              </div>

              {/* Results Content */}
              <div 
                ref={logsContainerRef}
                className="flex-1 overflow-y-auto p-4 font-mono text-sm"
              >
                {/* Loading State (initial) */}
                {isLoading && logs.length === 0 && (
                  <div className="flex flex-col items-center justify-center py-12">
                    <Loader2 className="w-8 h-8 animate-spin text-cyan-500 mb-4" />
                    <p className="text-slate-500 dark:text-slate-400">
                      Querying Azure Log Analytics...
                    </p>
                    {executionTime && (
                      <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                        {executionTime}ms
                      </p>
                    )}
                  </div>
                )}

                {/* Error State - Using specific Azure error display */}
                {error && !isLoading && azureError && (
                  <AzureErrorDisplay
                    errorType={azureError.type}
                    message={error}
                    serviceName={serviceName}
                    onRetry={handleRetry}
                    onViewLocal={handleViewLocal}
                    onResetQuery={azureError.type === 'query' ? handleResetQuery : undefined}
                    retryAfter={azureError.retryAfter}
                  />
                )}

                {/* Empty State */}
                {!isLoading && !error && logs.length === 0 && total === 0 && timeRange.preset !== 'custom' && (
                  <div className="flex flex-col items-center justify-center py-12 text-center">
                    <Inbox className="w-10 h-10 text-slate-300 dark:text-slate-600 mb-4" />
                    <h3 className="text-base font-medium text-slate-700 dark:text-slate-300 mb-2">
                      No logs found
                    </h3>
                    <p className="text-sm text-slate-500 dark:text-slate-400 max-w-xs">
                      No logs matching your query in the selected time range.
                      Try expanding the time range or adjusting your query filters.
                    </p>
                  </div>
                )}

                {/* Log Entries */}
                {logs.length > 0 && (
                  <div className="space-y-0.5">
                    {logs.map((log, idx) => (
                      <div key={`${log.timestamp}-${idx}`} className={getLogColor(log)}>
                        <span className="text-slate-500 dark:text-slate-500 text-xs">
                          [{formatLogTimestamp(String(log?.timestamp ?? ''))}]
                        </span>
                        {' '}
                        <span className={getServiceColor(log?.service ?? serviceName)}>
                          [{log?.service ?? serviceName}]
                        </span>
                        {' '}
                        <span 
                          className="text-foreground"
                          dangerouslySetInnerHTML={{ 
                            __html: convertAnsiToHtml(log?.message ?? '') 
                          }} 
                        />
                      </div>
                    ))}
                  </div>
                )}

                {/* Load More Button */}
                {hasMore && !isLoading && (
                  <div className="flex justify-center pt-4">
                    <button
                      type="button"
                      onClick={handleLoadMore}
                      className={cn(
                        'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
                        'bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700',
                        'text-slate-700 dark:text-slate-200',
                        'transition-colors duration-200',
                      )}
                    >
                      Load More ({PAGE_SIZE})
                    </button>
                  </div>
                )}

                {/* Loading More Indicator */}
                {isLoading && logs.length > 0 && (
                  <div className="flex justify-center pt-4">
                    <div className="flex items-center gap-2 text-slate-500 dark:text-slate-400">
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Loading more...
                    </div>
                  </div>
                )}
              </div>

              {/* Execution Time Footer */}
              {executionTime && logs.length > 0 && (
                <div className="px-4 py-2 text-xs text-slate-400 dark:text-slate-500 border-t border-slate-200 dark:border-slate-700 shrink-0">
                  Query executed in {executionTime}ms
                </div>
              )}
            </div>
          </>
        )}
      </div>

      {/* Log Config Panel */}
      <LogConfigPanel
        serviceName={serviceName}
        isOpen={configPanelOpen}
        onClose={handleCloseConfigPanel}
        onSave={handleConfigSaved}
      />
    </>
  )
}

export default HistoricalLogPanel
