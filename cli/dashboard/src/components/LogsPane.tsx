import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Copy, AlertTriangle, Info, XCircle, Check, ChevronDown, ChevronRight, Heart, HeartPulse, HeartCrack, ExternalLink, CircleDot, PanelRight, CheckCircle, CircleOff, Loader2, RotateCw, HelpCircle, Eye, Hammer, CheckSquare, CircleX, Monitor, Cloud } from 'lucide-react'
import { formatLogTimestamp, getLogPaneVisualStatus, normalizeHealthStatus, type VisualStatus } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import { useCodespaceEnv } from '@/hooks/useCodespaceEnv'
import { getEffectiveServiceUrl } from '@/lib/codespace-utils'
import type { HealthStatus, Service } from '@/types'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import { ServiceActions } from './ServiceActions'
import type { LogMode } from './ModeToggle'
import {
  MAX_LOGS_IN_MEMORY,
  LOG_LEVELS,
  convertAnsiToHtml,
  isErrorLine as baseIsErrorLine,
  isWarningLine as baseIsWarningLine,
  stripEmbeddedTimestamp,
} from '@/lib/log-utils'
import { useAzurePollingRefreshTrigger } from '@/hooks/useAzurePollingRefreshTrigger'

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

type PaneLogLevel = 'info' | 'warning' | 'error'

type ClassificationLevel = 'info' | 'warning' | 'error'

type AzureTimeRange = {
  preset: '15m' | '30m' | '6h' | '24h'
  end?: Date
}

const DEFAULT_AZURE_TIME_RANGE: AzureTimeRange = { preset: '15m' }

type AzureTimeRangeBounds = {
  start: Date
  end: Date
}

const LOADING_INDICATOR_DELAY_MS = 150
const LOADING_INDICATOR_MIN_VISIBLE_MS = 250

function useSmoothedLoadingIndicator(isActive: boolean): boolean {
  const [isVisible, setIsVisible] = useState(false)
  const delayTimeoutRef = useRef<ReturnType<typeof globalThis.setTimeout> | null>(null)
  const hideTimeoutRef = useRef<ReturnType<typeof globalThis.setTimeout> | null>(null)
  const shownAtRef = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (delayTimeoutRef.current !== null) {
        globalThis.clearTimeout(delayTimeoutRef.current)
        delayTimeoutRef.current = null
      }
      if (hideTimeoutRef.current !== null) {
        globalThis.clearTimeout(hideTimeoutRef.current)
        hideTimeoutRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    if (delayTimeoutRef.current !== null) {
      globalThis.clearTimeout(delayTimeoutRef.current)
      delayTimeoutRef.current = null
    }

    if (hideTimeoutRef.current !== null) {
      globalThis.clearTimeout(hideTimeoutRef.current)
      hideTimeoutRef.current = null
    }

    if (isActive) {
      if (isVisible) {
        return
      }

      delayTimeoutRef.current = globalThis.setTimeout(() => {
        delayTimeoutRef.current = null
        shownAtRef.current = Date.now()
        setIsVisible(true)
      }, LOADING_INDICATOR_DELAY_MS)

      return
    }

    if (!isVisible) {
      return
    }

    const shownAt = shownAtRef.current ?? Date.now()
    const elapsedMs = Date.now() - shownAt
    const remainingMs = LOADING_INDICATOR_MIN_VISIBLE_MS - elapsedMs
    if (remainingMs <= 0) {
      shownAtRef.current = null
      queueMicrotask(() => setIsVisible(false))
      return
    }

    hideTimeoutRef.current = globalThis.setTimeout(() => {
      hideTimeoutRef.current = null
      shownAtRef.current = null
      setIsVisible(false)
    }, remainingMs)
  }, [isActive, isVisible])

  return isVisible
}

function getAzureTimeRangeBounds(timeRange: AzureTimeRange, now: Date): AzureTimeRangeBounds | null {
  const end = timeRange.end ?? now
  const durationMs = {
    '15m': 15 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
  }[timeRange.preset]

  return { start: new Date(end.getTime() - durationMs), end }
}

function formatAzureRangeTimestamp(value: Date): string {
  // Deterministic and timezone-explicit.
  return value.toISOString().replace('T', ' ').replace(/\.\d{3}Z$/, 'Z')
}

function formatAzureTimeRangePreset(preset: AzureTimeRange['preset']): string {
  switch (preset) {
    case '15m':
      return '15 minutes'
    case '30m':
      return '30 minutes'
    case '6h':
      return '6 hours'
    case '24h':
      return '24 hours'
  }
}

function suggestAzureTimeRangePreset(preset: AzureTimeRange['preset']): string {
  if (preset === '15m' || preset === '30m' || preset === '6h') {
    return '24 hours'
  }

  return 'a wider range'
}

function LogsPaneEmptyState({
  errorMessage,
  isLoading,
  isWaiting,
  logMode,
  timeRange,
  hasLogs,
}: Readonly<{
  errorMessage: string | null
  isLoading: boolean
  isWaiting: boolean
  logMode: LogMode
  timeRange: AzureTimeRange
  hasLogs: boolean
}>): ReactNode {
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

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
        <Loader2 className="w-5 h-5 animate-spin mb-2" />
        <div className="text-sm">
          {logMode === 'azure' ? 'Fetching Azure logs...' : 'Fetching local logs...'}
        </div>
      </div>
    )
  }

  if (isWaiting) {
    const label = logMode === 'azure' ? 'Fetching Azure logs...' : 'Fetching local logs...'
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
        <span className="sr-only">{label}</span>
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

function getHealthIcon(normalizedHealth: ReturnType<typeof normalizeHealthStatus> | undefined): ReactNode {
  if (!normalizedHealth) {
    return null
  }

  switch (normalizedHealth) {
    case 'healthy':
      return <Heart className="w-3 h-3 shrink-0 animate-heartbeat" />
    case 'degraded':
      return <HeartPulse className="w-3 h-3 shrink-0 animate-caution-pulse" />
    case 'unhealthy':
      return <HeartCrack className="w-3 h-3 shrink-0 animate-status-flash" />
    default:
      return <HelpCircle className="w-3 h-3 shrink-0" />
  }
}

function getPaneStyleClasses(visualStatus: VisualStatus): { borderClass: string; headerBgClass: string } {
  const borderClass = {
    error: 'border-red-500',
    warning: 'border-amber-500',
    healthy: 'border-green-500',
    stopped: 'border-gray-400',
    info: 'border-border'
  }[visualStatus]

  const headerBgClass = {
    error: 'log-header-error',
    warning: 'log-header-warning',
    healthy: 'log-header-healthy',
    stopped: 'bg-muted',
    info: 'bg-card'
  }[visualStatus]

  return { borderClass, headerBgClass }
}

function getProcessBadge(processStatus?: string): { className: string; icon: ReactNode; title: string } {
  if (!processStatus) {
    return {
      className: 'bg-muted text-muted-foreground border border-border',
      icon: <CircleDot className="w-3 h-3 shrink-0" />,
      title: 'Process state: unknown',
    }
  }

  switch (processStatus) {
    case 'running':
    case 'ready':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <CheckCircle className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'watching':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <Eye className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'stopped':
    case 'not-started':
    case 'not-running':
      return {
        className: 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border border-gray-500/30',
        icon: <CircleOff className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'building':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <Hammer className="w-3 h-3 shrink-0 animate-pulse" />,
        title: `Process state: ${processStatus}`,
      }
    case 'built':
    case 'completed':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <CheckSquare className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'starting':
    case 'stopping':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <Loader2 className="w-3 h-3 shrink-0 animate-spin" />,
        title: `Process state: ${processStatus}`,
      }
    case 'restarting':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <RotateCw className="w-3 h-3 shrink-0 animate-spin" />,
        title: `Process state: ${processStatus}`,
      }
    case 'failed':
    case 'error':
      return {
        className: 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30',
        icon: <CircleX className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    default:
      return {
        className: 'bg-muted text-muted-foreground border border-border',
        icon: <CircleDot className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
  }
}

function getHealthBadgeClass(normalizedHealth: ReturnType<typeof normalizeHealthStatus>): string {
  switch (normalizedHealth) {
    case 'healthy':
      return 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30'
    case 'degraded':
      return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30'
    case 'unhealthy':
      return 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30'
    case 'unknown':
    default:
      return 'bg-muted text-muted-foreground border border-border'
  }
}

function LogsPaneHeader({
  serviceName,
  port,
  isCollapsed,
  toggleCollapsed,
  headerBgClass,
  processStatus,
  normalizedHealth,
  healthIcon,
  service,
  effectiveUrl,
  logMode,
  azureUrl,
  onShowDetails,
  handleCopyPane,
}: Readonly<{
  serviceName: string
  port?: number
  isCollapsed: boolean
  toggleCollapsed: () => void
  headerBgClass: string
  processStatus?: string
  normalizedHealth?: ReturnType<typeof normalizeHealthStatus>
  healthIcon: ReactNode
  service?: Service
  effectiveUrl?: string
  logMode: LogMode
  azureUrl?: string
  onShowDetails?: () => void
  handleCopyPane: () => void
}>): ReactNode {
  const processBadge = useMemo(() => getProcessBadge(processStatus), [processStatus])
  const healthBadgeClass = normalizedHealth ? getHealthBadgeClass(normalizedHealth) : ''

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-b transition-colors duration-200", headerBgClass)}>
      <button
        type="button"
        className="flex items-center gap-2 flex-1 min-w-0 cursor-pointer select-none"
        onClick={toggleCollapsed}
        aria-label={isCollapsed ? `Expand logs pane for ${serviceName}` : `Collapse logs pane for ${serviceName}`}
        aria-expanded={!isCollapsed}
      >
        <span className="p-0.5 hover:bg-muted rounded transition-colors" aria-hidden="true">
          {isCollapsed ? (
            <ChevronRight className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          )}
        </span>
        <h3 className="font-semibold truncate" data-testid="logs-pane-header-title">
          {serviceName}
          {port && <span className="text-muted-foreground font-mono">:{port}</span>}
        </h3>

        <span 
          className={cn(
            "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
            processBadge.className
          )}
          title={processBadge.title}
        >
          {processBadge.icon}
        </span>

        {/* Health Status Badge */}
        {normalizedHealth && (
          <span 
            className={cn(
              "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
              healthBadgeClass
            )}
            title={`Service health: ${normalizedHealth} (from health checks)`}
          >
            {healthIcon}
          </span>
        )}
      </button>

      <div className="flex items-center gap-2">
        {service && (
          <div className="mr-2 border-r pr-2 border-border">
            <ServiceActions service={service} variant="compact" />
          </div>
        )}
        {effectiveUrl && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => globalThis.open(effectiveUrl, '_blank', 'noopener,noreferrer')}
            title={effectiveUrl}
            aria-label={logMode === 'azure' && azureUrl ? 'Open Azure endpoint in new tab' : 'Open local service in new tab'}
          >
            <ExternalLink className="w-4 h-4" />
          </Button>
        )}
        {onShowDetails && (
          <Button
            variant="outline"
            size="sm"
            onClick={onShowDetails}
            title="Show service details"
            aria-label="Show service details panel"
          >
            <PanelRight className="w-4 h-4" />
          </Button>
        )}
        <Button
          variant="outline"
          size="sm"
          onClick={handleCopyPane}
          title="Copy all logs"
          aria-label="Copy logs to clipboard"
        >
          <Copy className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}

function LogsPaneModeBar({
  isCollapsed,
  logMode,
  modeIndicator,
}: Readonly<{
  isCollapsed: boolean
  logMode: LogMode
  modeIndicator: ReactNode
}>): ReactNode {
  if (isCollapsed) {
    return null
  }

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
    </div>
  )
}

function LogsPaneLogArea({
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
}: Readonly<{
  isCollapsed: boolean
  logsContainerRef: React.RefObject<HTMLDivElement | null>
  setIsHovering: (value: boolean) => void
  filteredLogs: LogEntry[]
  logs: LogEntry[]
  logMode: LogMode
  codespaceConfig: ReturnType<typeof useCodespaceEnv>['config']
  isLoading: boolean
  isWaiting: boolean
  errorMessage: string | null
  timeRange: AzureTimeRange
  getPaneLogLevel: (log: LogEntry) => PaneLogLevel
  copiedLineIndex: number | null
  handleCopyLine: (log: LogEntry, index?: number) => void
  logsEndRef: React.RefObject<HTMLDivElement | null>
}>): ReactNode {
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

function LogsPaneClassificationOverlay({
  selectionPosition,
  selectedText,
  handleClassifySelection,
}: Readonly<{
  selectionPosition: { x: number; y: number } | null
  selectedText: string
  handleClassifySelection: (level: ClassificationLevel) => void
}>): ReactNode {
  if (!selectionPosition || !selectedText) {
    return null
  }

  return (
    <div
      className="classification-popup fixed z-50 flex gap-1 bg-popover border rounded-md shadow-lg p-1"
      style={{
        left: `${selectionPosition.x}px`,
        top: `${selectionPosition.y}px`,
        transform: 'translate(-50%, -100%)'
      }}
    >
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('info')}
        className="h-8 px-2 bg-blue-500 hover:bg-blue-600"
        title="Classify as Info"
      >
        <Info className="w-4 h-4 text-white" />
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('warning')}
        className="h-8 px-2 bg-yellow-500 hover:bg-yellow-600"
        title="Classify as Warning"
      >
        <AlertTriangle className="w-4 h-4 text-white" />
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={() => handleClassifySelection('error')}
        className="h-8 px-2 bg-red-500 hover:bg-red-600"
        title="Classify as Error"
      >
        <XCircle className="w-4 h-4 text-white" />
      </Button>
    </div>
  )
}

function LogsPaneClassificationToast({
  show,
}: Readonly<{ show: boolean }>): ReactNode {
  if (!show) {
    return null
  }

  return (
    <div
      className="fixed z-50 bg-green-500 text-white px-4 py-2 rounded-md shadow-lg flex items-center gap-2"
      style={{
        top: '20px',
        right: '20px'
      }}
    >
      <Check className="w-4 h-4" />
      <span>Classification saved</span>
    </div>
  )
}

function LogsPaneRefreshFooter({
  isCollapsed,
  syncInterval,
  isPaused,
  secondsUntilRefresh,
}: Readonly<{
  isCollapsed: boolean
  syncInterval?: number
  isPaused: boolean
  secondsUntilRefresh: number
}>): ReactNode {
  if (isCollapsed || !syncInterval || isPaused || secondsUntilRefresh <= 0) {
    return null
  }

  return (
    <div className="flex items-center justify-center gap-2 px-3 py-1.5 text-xs border-t border-border bg-muted/30">
      <RotateCw className="w-3 h-3 text-muted-foreground" />
      <span className="text-muted-foreground">
        Next refresh in <span className="font-medium text-foreground">{secondsUntilRefresh}s</span>
      </span>
    </div>
  )
}

function useLogsStream(params: {
  serviceName: string
  fetchKey: string
  logMode: LogMode
  timeRange: AzureTimeRange
  azureRealtime: boolean
  refreshTrigger: number
  isPausedRef: { current: boolean }
  setLogs: React.Dispatch<React.SetStateAction<LogEntry[]>>
  setErrorMessage: React.Dispatch<React.SetStateAction<string | null>>
  onFetchSettled?: () => void
}): void {
  const { serviceName, fetchKey, logMode, timeRange, azureRealtime, refreshTrigger, isPausedRef, setLogs, setErrorMessage, onFetchSettled } = params
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    // Choose endpoint based on mode
    const logsEndpoint = logMode === 'azure' ? '/api/azure/logs' : '/api/logs'
    const streamEndpoint = logMode === 'azure' ? '/api/azure/logs/stream' : '/api/logs/stream'

    let cancelled = false
    const fetchLogs = async () => {
      try {
        setErrorMessage(null)
        const url = buildLogsFetchUrl(logsEndpoint, serviceName, logMode, timeRange)
        const res = await fetch(url)
        if (!res.ok) {
          const message = (await res.text()) || `HTTP ${res.status}`
          setErrorMessage(message)
          return
        }

        const data: unknown = await res.json()
        const nextLogs = parseLogsPayload(logMode, data)
        setLogs(nextLogs)
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Request failed'
        setErrorMessage(message)
        console.error(`[LogsPane:${serviceName}] Failed to fetch ${logMode} logs:`, err)
      } finally {
        if (!cancelled) {
          onFetchSettled?.()
        }
      }
    }

    void fetchLogs()

    const closeSocket = () => {
      if (!wsRef.current) return
      const currentWs = wsRef.current
      if (currentWs.readyState === WebSocket.OPEN || currentWs.readyState === WebSocket.CONNECTING) {
        currentWs.close(1000, 'Mode change or unmount')
      }
      wsRef.current = null
    }

    // Azure logs can either poll via REST or stream via WebSocket (realtime)
    if (logMode === 'azure') {
      if (!azureRealtime) {
        // Azure polling mode: fetchLogs is called on mount and when refreshTrigger changes
        return () => {
          cancelled = true
          closeSocket()
        }
      }

      const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const ws = new WebSocket(
        `${protocol}//${globalThis.location.host}${streamEndpoint}?service=${serviceName}&realtime=true`
      )

      ws.onmessage = (event) => {
        if (isPausedRef.current) return
        try {
          const entry = JSON.parse(event.data as string) as LogEntry
          setLogs((prev) => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
        } catch (err) {
          console.error('Failed to parse Azure log entry:', err)
        }
      }

      ws.onerror = (err) => {
        if (ws.readyState !== WebSocket.CLOSED) {
          setErrorMessage('WebSocket connection error')
          console.error(`WebSocket error for ${serviceName} (${logMode}):`, err)
        }
      }

      wsRef.current = ws
      return () => {
        cancelled = true
        closeSocket()
      }
    }

    // For local logs, use WebSocket streaming
    const protocol = globalThis.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${globalThis.location.host}${streamEndpoint}?service=${serviceName}`)

    ws.onmessage = (event) => {
      if (isPausedRef.current) return
      try {
        const entry = JSON.parse(event.data as string) as LogEntry
        setLogs((prev) => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
      } catch (err) {
        console.error('Failed to parse log entry:', err)
      }
    }

    ws.onerror = (err) => {
      if (ws.readyState !== WebSocket.CLOSED) {
        setErrorMessage('WebSocket connection error')
        console.error(`WebSocket error for ${serviceName} (${logMode}):`, err)
      }
    }

    wsRef.current = ws
    return () => {
      cancelled = true
      closeSocket()
    }
  }, [serviceName, fetchKey, refreshTrigger, isPausedRef, setLogs, onFetchSettled, azureRealtime, logMode, timeRange])
}

function buildLogsFetchUrl(
  endpoint: string,
  serviceName: string,
  logMode: LogMode,
  timeRange: AzureTimeRange | undefined
): string {
  const params = new URLSearchParams({ service: serviceName, tail: '500' })

  if (logMode === 'azure' && timeRange) {
    params.set('since', timeRange.preset)
  }

  return `${endpoint}?${params.toString()}`
}

function parseLogsPayload(logMode: LogMode, data: unknown): LogEntry[] {
  if (!Array.isArray(data) && (typeof data !== 'object' || data === null)) {
    return []
  }

  if (Array.isArray(data)) {
    return data as LogEntry[]
  }

  if (logMode === 'azure') {
    const azurePayload = data as { logs?: LogEntry[] }
    return azurePayload.logs ?? []
  }

  return []
}

interface LogsPaneProps {
  serviceName: string
  port?: number
  url?: string                    // Service URL for "open in new tab" button
  service?: Service               // Full service object for lifecycle controls
  onCopy: (logs: LogEntry[]) => void
  isPaused: boolean
  globalSearchTerm?: string
  autoScrollEnabled?: boolean
  clearAllTrigger?: number
  levelFilter?: Set<'info' | 'warning' | 'error'>
  isCollapsed?: boolean           // NEW: controlled collapse state
  onToggleCollapse?: () => void   // NEW: collapse toggle callback
  serviceHealth?: HealthStatus  // Real-time health from stream
  onShowDetails?: () => void      // Callback to open service details panel
  logMode?: LogMode               // Current log source mode (local or azure)
  isModeSwitching?: boolean       // Whether mode is currently being switched
  timeRange?: { preset: '15m' | '30m' | '6h' | '24h'; end?: Date }  // Optional, only used for Azure logs
  syncInterval?: number           // Auto-refresh interval in milliseconds
  azureRealtime?: boolean         // Whether to use WebSocket realtime streaming for Azure logs
}

export function LogsPane({ 
  serviceName, 
  port,
  url,
  service,
  onCopy, 
  isPaused, 
  globalSearchTerm = '', 
  autoScrollEnabled = true, 
  clearAllTrigger = 0, 
  levelFilter = new Set(['info', 'warning', 'error'] as const),
  isCollapsed: controlledIsCollapsed,
  onToggleCollapse,
  serviceHealth,
  onShowDetails,
  logMode = 'local',
  isModeSwitching = false,
  timeRange,
  syncInterval,
  azureRealtime = false,
}: Readonly<LogsPaneProps>) {
  const resolvedTimeRange = timeRange ?? DEFAULT_AZURE_TIME_RANGE
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [hasFetchedForKey, setHasFetchedForKey] = useState(false)
  const fetchKey = useMemo(() => {
    // Local logs are not scoped by timeframe; keeping them independent avoids unnecessary reconnects
    // (and prevents a caller accidentally passing a constantly-changing end timestamp).
    if (logMode !== 'azure') {
      return `${logMode}:stream`
    }

    const end = resolvedTimeRange.end ? resolvedTimeRange.end.toISOString() : ''
    return `azure:${resolvedTimeRange.preset}:${end}:${azureRealtime ? 'realtime' : 'poll'}`
  }, [logMode, resolvedTimeRange.preset, resolvedTimeRange.end, azureRealtime])

  useEffect(() => {
    setHasFetchedForKey(false)
    setErrorMessage(null)
  }, [fetchKey])

  const [selectedText, setSelectedText] = useState<string>('')
  const [selectionPosition, setSelectionPosition] = useState<{ x: number; y: number } | null>(null)
  const [showClassificationConfirmation, setShowClassificationConfirmation] = useState(false)
  const [copiedLineIndex, setCopiedLineIndex] = useState<number | null>(null)
  // Internal state as fallback for uncontrolled mode
  const [internalIsCollapsed, setInternalIsCollapsed] = useState<boolean>(() => {
    const saved = localStorage.getItem(`logs-pane-collapsed-${serviceName}`)
    return saved === 'true'
  })
  
  // Use controlled state if provided, otherwise internal
  const isCollapsed = controlledIsCollapsed ?? internalIsCollapsed
  
  // Get Codespace config for URL transformation
  const { config: codespaceConfig } = useCodespaceEnv()
  
  // Transform URL for Codespace environment (local mode)
  const effectiveLocalUrl = getEffectiveServiceUrl(url, port, codespaceConfig)
  
  // Get Azure URL from service info
  const azureUrl = service?.azure?.url
  
  // Use Azure URL when in Azure mode, otherwise local URL
  const effectiveUrl = logMode === 'azure' && azureUrl ? azureUrl : effectiveLocalUrl
  
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const isPausedRef = useRef(isPaused)
  const [isHovering, setIsHovering] = useState(false)

  const { secondsUntilRefresh, refreshTrigger } = useAzurePollingRefreshTrigger({
    syncInterval,
    isPaused,
    logMode,
    azureRealtime,
  })
  
  const { addClassification, getClassificationForText } = useLogClassifications()

  // Keep isPaused ref in sync for WebSocket callback
  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

  // Toggle function - use callback if provided, otherwise internal
  const toggleCollapsed = useCallback(() => {
    if (onToggleCollapse) {
      onToggleCollapse()
    } else {
      setInternalIsCollapsed(prev => {
        const newValue = !prev
        localStorage.setItem(`logs-pane-collapsed-${serviceName}`, String(newValue))
        return newValue
      })
    }
  }, [serviceName, onToggleCollapse])

  // Clear logs when global clear is triggered or mode changes
  useEffect(() => {
    if (clearAllTrigger > 0) {
      setLogs([])
    }
  }, [clearAllTrigger])

  // Clear logs and refetch when mode changes
  useEffect(() => {
    setLogs([]) // Clear logs when switching modes
    setErrorMessage(null)
  }, [logMode])

  // Fetch initial logs and setup streaming - reconnect when mode or timeframe changes
  // For Azure polling mode, refreshTrigger triggers re-fetches when countdown expires
  useLogsStream({
    serviceName,
    fetchKey,
    logMode,
    timeRange: resolvedTimeRange,
    azureRealtime,
    refreshTrigger,
    isPausedRef,
    setLogs,
    setErrorMessage,
    onFetchSettled: () => {
      setHasFetchedForKey(true)
    },
  })

  const isWaitingForFirstFetch = isModeSwitching || !hasFetchedForKey
  const showLoadingIndicator = useSmoothedLoadingIndicator(isWaitingForFirstFetch)

  // Auto-scroll - scroll the container, not the page
  // Pause auto-scroll when user is hovering over the logs
  useEffect(() => {
    if (autoScrollEnabled && !isPaused && !isHovering && logsContainerRef.current) {
      const container = logsContainerRef.current
      container.scrollTop = container.scrollHeight
    }
  }, [logs, autoScrollEnabled, isPaused, isHovering])

  const handleTextSelection = useCallback(() => {
    const selection = globalThis.getSelection()
    const text = selection?.toString().trim()
    
    if (text && text.length > 0) {
      const range = selection?.getRangeAt(0)
      const rect = range?.getBoundingClientRect()
      
      if (rect) {
        setSelectedText(text)
        setSelectionPosition({
          x: rect.left + rect.width / 2,
          y: rect.top - 10
        })
      }
    } else {
      setSelectedText('')
      setSelectionPosition(null)
    }
  }, [])

  const handleClassifySelection = useCallback((level: ClassificationLevel) => {
    if (!selectedText) {
      return
    }

    addClassification(selectedText, level)
      .then(() => {
        setSelectedText('')
        setSelectionPosition(null)
        globalThis.getSelection()?.removeAllRanges()

        setShowClassificationConfirmation(true)
        setTimeout(() => {
          setShowClassificationConfirmation(false)
        }, 2000)
      })
      .catch((err: unknown) => {
        console.error('Failed to save classification:', err)
      })
  }, [selectedText, addClassification])

  useEffect(() => {
    const container = logsContainerRef.current
    if (!container) return

    container.addEventListener('mouseup', handleTextSelection)
    container.addEventListener('touchend', handleTextSelection)

    return () => {
      container.removeEventListener('mouseup', handleTextSelection)
      container.removeEventListener('touchend', handleTextSelection)
    }
  }, [handleTextSelection])

  const isErrorLine = useCallback((message: string) => {
    // Check if any part of the message has a classification
    const classificationLevel = getClassificationForText(message)
    if (classificationLevel === 'error') return true
    if (classificationLevel === 'info' || classificationLevel === 'warning') return false

    // Use centralized error detection
    return baseIsErrorLine(message)
  }, [getClassificationForText])

  const isWarningLine = useCallback((message: string) => {
    // Check if any part of the message has a classification
    const classificationLevel = getClassificationForText(message)
    if (classificationLevel === 'warning') return true
    if (classificationLevel === 'info' || classificationLevel === 'error') return false
    
    // Use centralized warning detection
    return baseIsWarningLine(message)
  }, [getClassificationForText])

  const paneStatus = useMemo(() => {
    const hasError = logs.some(log => 
      isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR
    )
    const hasWarning = logs.some(log => isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING)

    if (hasError) return 'error'
    if (hasWarning) return 'warning'
    return 'info'
  }, [logs, isErrorLine, isWarningLine])

  const getPaneLogLevel = useCallback((log: LogEntry): PaneLogLevel => {
    const overrideLevel = getClassificationForText(log.message)
    const isError =
      overrideLevel === 'error' ||
      (!overrideLevel && (isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR))

    if (isError) {
      return 'error'
    }

    const isWarning =
      overrideLevel === 'warning' ||
      (!overrideLevel && (isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING))

    if (isWarning) {
      return 'warning'
    }

    return 'info'
  }, [getClassificationForText, isErrorLine, isWarningLine])

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (!log?.message) return false
      
      // Text search filter
      if (!log.message.toLowerCase().includes(globalSearchTerm.toLowerCase())) return false
      
      // Level filter
      const logLevel = getPaneLogLevel(log)
      return levelFilter.has(logLevel)
    })
  }, [logs, globalSearchTerm, levelFilter, getPaneLogLevel])

  const handleCopyPane = () => {
    onCopy(filteredLogs)
  }

  const handleCopyLine = (log: LogEntry, index?: number) => {
    const formattedTimestamp = formatLogTimestamp(log.timestamp ?? '')
    const cleanedMessage = stripEmbeddedTimestamp(log.message ?? '')
    const serviceLabel = log.service ? ` | ${log.service}` : ''
    const text = `[${formattedTimestamp}${serviceLabel}] ${cleanedMessage}`
    void navigator.clipboard.writeText(text)
    if (index !== undefined) {
      setCopiedLineIndex(index)
      setTimeout(() => setCopiedLineIndex(null), 1500)
    }
  }

  const handleClickOutside = useCallback((e: MouseEvent) => {
    const target = e.target as HTMLElement
    if (selectionPosition && !target.closest('.classification-popup')) {
      setSelectedText('')
      setSelectionPosition(null)
    }
  }, [selectionPosition])

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [handleClickOutside])

  // Get process status from service
  const processStatus = service?.local?.status

  // Normalize health status - backend may send 'starting' which we treat as 'unknown'
  const normalizedHealth = serviceHealth ? normalizeHealthStatus(serviceHealth) : undefined

  // Border and header colors should follow service health (if available), not log content
  // Process status (stopped) takes priority over health status
  const visualStatus: VisualStatus = getLogPaneVisualStatus(normalizedHealth, paneStatus, processStatus)

  const { borderClass, headerBgClass } = useMemo(() => getPaneStyleClasses(visualStatus), [visualStatus])
  const healthIcon = useMemo(() => getHealthIcon(normalizedHealth), [normalizedHealth])

  const modeIndicator = useMemo(() => getModeIndicator(logMode, isModeSwitching), [logMode, isModeSwitching])

  return (
    <section
      className={cn("flex flex-col border-4 rounded-lg overflow-hidden transition-all duration-200", borderClass)}
      style={{ 
        height: isCollapsed ? 'fit-content' : '100%',
        minHeight: isCollapsed ? undefined : '150px',
        maxHeight: '100%',
      }}
      aria-label={`Logs for ${serviceName}`}
    >
      <LogsPaneHeader
        serviceName={serviceName}
        port={port}
        isCollapsed={isCollapsed}
        toggleCollapsed={toggleCollapsed}
        headerBgClass={headerBgClass}
        processStatus={processStatus}
        normalizedHealth={normalizedHealth}
        healthIcon={healthIcon}
        service={service}
        effectiveUrl={effectiveUrl ?? undefined}
        logMode={logMode}
        azureUrl={azureUrl}
        onShowDetails={onShowDetails}
        handleCopyPane={handleCopyPane}
      />

      <LogsPaneModeBar
        isCollapsed={isCollapsed}
        logMode={logMode}
        modeIndicator={modeIndicator}
      />

      <LogsPaneLogArea
        isCollapsed={isCollapsed}
        logsContainerRef={logsContainerRef}
        setIsHovering={setIsHovering}
        filteredLogs={filteredLogs}
        logs={logs}
        logMode={logMode}
        codespaceConfig={codespaceConfig}
        isLoading={showLoadingIndicator}
        isWaiting={isWaitingForFirstFetch}
        errorMessage={errorMessage}
        timeRange={resolvedTimeRange}
        getPaneLogLevel={getPaneLogLevel}
        copiedLineIndex={copiedLineIndex}
        handleCopyLine={handleCopyLine}
        logsEndRef={logsEndRef}
      />

      <LogsPaneClassificationOverlay
        selectionPosition={selectionPosition}
        selectedText={selectedText}
        handleClassifySelection={handleClassifySelection}
      />

      <LogsPaneClassificationToast show={showClassificationConfirmation} />

      <LogsPaneRefreshFooter
        isCollapsed={isCollapsed}
        syncInterval={syncInterval}
        isPaused={isPaused}
        secondsUntilRefresh={secondsUntilRefresh}
      />
    </section>
  )
}
