import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { formatLogTimestamp, getLogPaneVisualStatus, getEffectiveAzureUrl, getEffectiveLocalUrl, normalizeHealthStatus } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import { useCodespaceEnv } from '@/hooks/useCodespaceEnv'
import { getEffectiveServiceUrl } from '@/lib/codespace-utils'
import type { HealthStatus, Service, HealthCheckResult } from '@/types'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import type { LogMode } from './ModeToggle'
import { stripEmbeddedTimestamp } from '@/lib/log-utils'
import { LogsPaneHeader } from './LogsPaneHeader'
import { LogsPaneContent } from './LogsPaneContent'
import { LogsPaneModeBar } from './LogsPaneAzureControls'
import { LogsPaneClassificationOverlay, LogsPaneClassificationToast } from './LogsPaneClassifications'
import { LogsPaneRefreshFooter } from './LogsPaneRefreshFooter'
import { LogConfigPanel } from './LogConfigPanel'
import { useLogFiltering } from '@/hooks/useLogFiltering'
import { useAzureTimeRange } from '@/hooks/useAzureTimeRange'
import { useLogScrolling } from '@/hooks/useLogScrolling'
import { useSmoothedLoadingIndicator } from '@/hooks/useSmoothedLoadingIndicator'
import { useLogsStream } from '@/hooks/useLogsStream'
import { getHealthIcon, getPaneStyleClasses } from '@/lib/logs-pane-utils'
import { UI_CONSTANTS } from '@/lib/constants'
import { useTimeout } from '@/hooks/useTimeout'

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
  sequence?: number  // Optional sequence number for gap detection
}

type ClassificationLevel = 'info' | 'warning' | 'error'



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
  healthCheckResult?: HealthCheckResult  // Full health check result with diagnostics
  onShowDetails?: () => void      // Callback to open service details panel
  logMode?: LogMode               // Current log source mode (local or azure)
  isModeSwitching?: boolean       // Whether mode is currently being switched
  timeRange?: { preset: '15m' | '30m' | '6h' | '24h'; end?: Date }  // Optional, only used for Azure logs
  azureRealtime?: boolean         // Whether to use WebSocket realtime streaming for Azure logs
  onOpenDiagnostics?: () => void  // Callback to open diagnostics modal
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
  healthCheckResult,
  onShowDetails,
  logMode = 'local',
  isModeSwitching = false,
  timeRange,
  azureRealtime = false,
  onOpenDiagnostics,
}: Readonly<LogsPaneProps>) {
  const { timeRange: resolvedTimeRange } = useAzureTimeRange(timeRange)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [hasFetchedForKey, setHasFetchedForKey] = useState(false)
  const [isLoadingLogs, setIsLoadingLogs] = useState(false)
  const [loadingMessage, setLoadingMessage] = useState('')
  const [canRetry, setCanRetry] = useState(false)
  
  // Using isLoadingLogs for detailed loading state tracking - value managed by useLogsStream
  // isLoadingLogs is managed by hook but not used in this component
  
  const fetchKey = useMemo(() => {
    if (logMode !== 'azure') {
      return `${logMode}:stream`
    }
    const end = resolvedTimeRange.end ? resolvedTimeRange.end.toISOString() : ''
    return `azure:${resolvedTimeRange.preset}:${end}:${azureRealtime ? 'realtime' : 'poll'}`
  }, [logMode, resolvedTimeRange.preset, resolvedTimeRange.end, azureRealtime])

  useEffect(() => {
    setHasFetchedForKey(false)
    setErrorMessage(null)
    setIsLoadingLogs(false)
    setLoadingMessage('')
    setCanRetry(false)
  }, [fetchKey])
  
  const handleFetchSettled = useCallback(() => {
    setHasFetchedForKey(true)
  }, [])

  const [selectedText, setSelectedText] = useState<string>('')
  const [selectionPosition, setSelectionPosition] = useState<{ x: number; y: number } | null>(null)
  const [showClassificationConfirmation, setShowClassificationConfirmation] = useState(false)
  const [copiedLineIndex, setCopiedLineIndex] = useState<number | null>(null)
  const { setTimeout } = useTimeout()
  
  const [internalIsCollapsed, setInternalIsCollapsed] = useState<boolean>(() => {
    const saved = localStorage.getItem(`logs-pane-collapsed-${serviceName}`)
    return saved === 'true'
  })
  
  const isCollapsed = controlledIsCollapsed ?? internalIsCollapsed
  const [configPanelOpen, setConfigPanelOpen] = useState(false)
  const { config: codespaceConfig } = useCodespaceEnv()

  const effectiveLocal = getEffectiveLocalUrl(
    service?.local ?? (url ? { url, status: 'not-started', health: 'unknown' } : undefined)
  )
  const localPort = port ?? service?.local?.port
  const transformedLocalUrl = getEffectiveServiceUrl(effectiveLocal.url, localPort, codespaceConfig)

  const effectiveAzure = getEffectiveAzureUrl(service?.azure)

  const effectiveUrl = logMode === 'azure'
    ? effectiveAzure.url ?? transformedLocalUrl
    : transformedLocalUrl ?? effectiveAzure.url

  const effectiveUrlSource: 'azure' | 'local' | undefined = logMode === 'azure'
    ? effectiveAzure.url
      ? 'azure'
      : transformedLocalUrl
        ? 'local'
        : undefined
    : transformedLocalUrl
      ? 'local'
      : effectiveAzure.url
        ? 'azure'
        : undefined
  
  const isPausedRef = useRef(isPaused)
  const lastClearTimeRef = useRef<number>(Date.now() - 1000) // Initialize to 1s in the past
  const { addClassification } = useLogClassifications()

  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

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

  const handleOpenConfigPanel = useCallback(() => {
    setConfigPanelOpen(true)
  }, [])

  const handleCloseConfigPanel = useCallback(() => {
    setConfigPanelOpen(false)
  }, [])

  const handleConfigSaved = useCallback(() => {
    setConfigPanelOpen(false)
    // Trigger log refresh by updating fetchKey
    setHasFetchedForKey(false)
  }, [])

  useEffect(() => {
    if (clearAllTrigger > 0) {
      // Record clear time to ignore WebSocket messages for a brief period
      lastClearTimeRef.current = Date.now()
      setLogs([])
    }
  }, [clearAllTrigger])

  useEffect(() => {
    // Record clear time when mode changes to prevent stale logs from appearing
    lastClearTimeRef.current = Date.now()
    setLogs([])
    setErrorMessage(null)
  }, [logMode])

  const { retry: retryLogs } = useLogsStream({
    serviceName,
    fetchKey,
    logMode,
    timeRange: resolvedTimeRange,
    azureRealtime,
    isPausedRef,
    lastClearTimeRef,
    setLogs,
    setErrorMessage,
    onFetchSettled: handleFetchSettled,
    setIsLoading: setIsLoadingLogs,
    setLoadingMessage,
    setCanRetry,
  })

  const isWaitingForFirstFetch = isModeSwitching || !hasFetchedForKey
  // Show loading indicator immediately for first fetch to provide instant feedback
  const showLoadingIndicator = useSmoothedLoadingIndicator(isWaitingForFirstFetch, { immediate: true })
  
  // Use isLoadingLogs for showing custom retry messages, fall back to showLoadingIndicator for general loading
  const effectiveIsLoading = isLoadingLogs || showLoadingIndicator

  const { filteredLogs, getPaneLogLevel, paneStatus } = useLogFiltering(
    logs,
    globalSearchTerm,
    levelFilter
  )

  const { logsContainerRef, logsEndRef, setIsHovering } = useLogScrolling(
    logs,
    autoScrollEnabled,
    isPaused
  )

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
    if (!selectedText) return

    addClassification(selectedText, level)
      .then(() => {
        setSelectedText('')
        setSelectionPosition(null)
        globalThis.getSelection()?.removeAllRanges()
        setShowClassificationConfirmation(true)
        // Use window.setTimeout to avoid bringing the hook's `setTimeout` into
        // this callback's dependency array (prevents ESLint missing-deps warning)
        window.setTimeout(() => setShowClassificationConfirmation(false), 2000)
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
  }, [handleTextSelection, logsContainerRef])

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
      setTimeout(() => setCopiedLineIndex(null), UI_CONSTANTS.COPY_FEEDBACK_DURATION_MS)
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

  // No extra cleanup needed; timers are handled by useTimeout

  const processStatus = service?.local?.status
  const normalizedHealth = serviceHealth ? normalizeHealthStatus(serviceHealth) : undefined
  const visualStatus = getLogPaneVisualStatus(normalizedHealth, paneStatus as 'info' | 'warning' | 'error', processStatus)
  const { borderClass, headerBgClass } = useMemo(() => getPaneStyleClasses(visualStatus), [visualStatus])
  const healthIcon = useMemo(() => getHealthIcon(normalizedHealth), [normalizedHealth])

  return (
    <section
      className={cn("flex flex-col border-4 rounded-lg overflow-hidden transition-all duration-200", borderClass)}
      style={{ 
        height: isCollapsed ? 'fit-content' : '100%',
        minHeight: isCollapsed ? undefined : `${UI_CONSTANTS.MIN_PANE_HEIGHT_PX}px`,
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
        healthCheckResult={healthCheckResult}
        service={service}
        effectiveUrl={effectiveUrl ?? undefined}
        effectiveUrlSource={effectiveUrlSource}
        logMode={logMode}
        onShowDetails={onShowDetails}
        handleCopyPane={handleCopyPane}
        onOpenConfigPanel={handleOpenConfigPanel}
      />

      <LogsPaneModeBar
        isCollapsed={isCollapsed}
        logMode={logMode}
        isModeSwitching={isModeSwitching}
        onOpenDiagnostics={onOpenDiagnostics}
      />

      <LogsPaneContent
        isCollapsed={isCollapsed}
        logsContainerRef={logsContainerRef}
        setIsHovering={setIsHovering}
        filteredLogs={filteredLogs}
        logs={logs}
        logMode={logMode}
        codespaceConfig={codespaceConfig ?? { enabled: false, name: '', domain: '' }}
        isLoading={effectiveIsLoading}
        isWaiting={isWaitingForFirstFetch}
        errorMessage={errorMessage}
        timeRange={resolvedTimeRange}
        getPaneLogLevel={getPaneLogLevel}
        copiedLineIndex={copiedLineIndex}
        handleCopyLine={handleCopyLine}
        logsEndRef={logsEndRef}
        loadingMessage={loadingMessage}
        canRetry={canRetry}
        onRetry={retryLogs}
        serviceName={serviceName}
        globalSearchTerm={globalSearchTerm}
        onOpenDiagnostics={onOpenDiagnostics}
      />

      <LogsPaneClassificationOverlay
        selectionPosition={selectionPosition}
        selectedText={selectedText}
        handleClassifySelection={handleClassifySelection}
      />

      <LogsPaneClassificationToast show={showClassificationConfirmation} />

      <LogsPaneRefreshFooter
        isCollapsed={isCollapsed}
        isPaused={isPaused}
        logMode={logMode}
      />

      <LogConfigPanel
        serviceName={serviceName}
        isOpen={configPanelOpen}
        onClose={handleCloseConfigPanel}
        onSave={handleConfigSaved}
      />
    </section>
  )
}
