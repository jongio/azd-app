import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Copy, AlertTriangle, Info, XCircle, Check, ChevronDown, ChevronRight, Heart, HeartPulse, HeartCrack, ExternalLink, CircleDot, PanelRight, CheckCircle, CircleOff, Loader2, RotateCw, HelpCircle, Eye, Hammer, CheckSquare, CircleX } from 'lucide-react'
import { formatLogTimestamp, getLogPaneVisualStatus, normalizeHealthStatus, type VisualStatus } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import type { HealthStatus, Service } from '@/types'
import { useLogClassifications } from '@/hooks/useLogClassifications'
import { ServiceActions } from './ServiceActions'
import {
  MAX_LOGS_IN_MEMORY,
  LOG_LEVELS,
  convertAnsiToHtml,
  isErrorLine as baseIsErrorLine,
  isWarningLine as baseIsWarningLine,
} from '@/lib/log-utils'

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
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
  onShowDetails
}: LogsPaneProps) {
  const [logs, setLogs] = useState<LogEntry[]>([])
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
  
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const isPausedRef = useRef(isPaused)
  
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

  // Clear logs when global clear is triggered
  useEffect(() => {
    if (clearAllTrigger > 0) {
      setLogs([])
    }
  }, [clearAllTrigger])

  // Fetch initial logs and setup WebSocket
  useEffect(() => {
    const fetchLogs = async () => {
      try {
        const res = await fetch(`/api/logs?service=${serviceName}&tail=500`)
        if (res.ok) {
          const data = await res.json() as LogEntry[]
          setLogs(data ?? [])
        }
      } catch (err) {
        console.error(`Failed to fetch logs for ${serviceName}:`, err)
      }
    }

    void fetchLogs()

    // Setup WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/logs/stream?service=${serviceName}`)

    ws.onmessage = (event) => {
      // Check pause state from ref to get current value (not stale closure)
      if (isPausedRef.current) {
        // When paused, don't add new logs
        return
      }
      try {
        const entry = JSON.parse(event.data as string) as LogEntry
        setLogs(prev => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
      } catch (err) {
        console.error('Failed to parse log entry:', err)
      }
    }

    ws.onerror = (err) => {
      console.error(`WebSocket error for ${serviceName}:`, err)
    }

    wsRef.current = ws

    return () => {
      // Ensure clean disconnection
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close(1000, 'Component unmounting')
      }
      wsRef.current = null
    }
  }, [serviceName]) // Removed isPaused - WebSocket shouldn't reconnect on pause toggle

  // Auto-scroll - scroll the container, not the page
  useEffect(() => {
    if (autoScrollEnabled && !isPaused && logsContainerRef.current) {
      const container = logsContainerRef.current
      container.scrollTop = container.scrollHeight
    }
  }, [logs, autoScrollEnabled, isPaused])

  const handleTextSelection = useCallback(() => {
    const selection = window.getSelection()
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

  const handleClassifySelection = useCallback(async (level: 'info' | 'warning' | 'error') => {
    if (selectedText) {
      try {
        await addClassification(selectedText, level)
        setSelectedText('')
        setSelectionPosition(null)
        window.getSelection()?.removeAllRanges()
        
        // Show confirmation feedback
        setShowClassificationConfirmation(true)
        setTimeout(() => {
          setShowClassificationConfirmation(false)
        }, 2000)
      } catch (err) {
        console.error('Failed to add classification:', err)
      }
    }
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

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      if (!log || !log.message) return false
      
      // Text search filter
      if (!log.message.toLowerCase().includes(globalSearchTerm.toLowerCase())) return false
      
      // Level filter
      const overrideLevel = getClassificationForText(log.message)
      const isError = overrideLevel === 'error' || (!overrideLevel && (isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR))
      const isWarning = overrideLevel === 'warning' || (!overrideLevel && !isError && (isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING))
      const logLevel = isError ? 'error' : isWarning ? 'warning' : 'info'
      
      return levelFilter.has(logLevel)
    })
  }, [logs, globalSearchTerm, levelFilter, getClassificationForText, isErrorLine, isWarningLine])

  const handleCopyPane = () => {
    onCopy(filteredLogs)
  }

  const handleCopyLine = (log: LogEntry, index?: number) => {
    const text = `[${log.timestamp}] [${log.service}] ${log.message}`
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

  return (
    <div 
      className={cn("flex flex-col border-4 rounded-lg overflow-hidden transition-all duration-200", borderClass)}
      style={{ 
        height: isCollapsed ? 'fit-content' : '100%',
        minHeight: isCollapsed ? undefined : '150px',
        maxHeight: '100%',
      }}
      role="region"
      aria-label={`Logs for ${serviceName}`}
    >
      {/* Header */}
      <div 
        className={cn(
          "flex items-center justify-between px-4 py-2 border-b cursor-pointer select-none transition-colors duration-200",
          headerBgClass
        )}
        onClick={toggleCollapsed}
      >
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation()
              toggleCollapsed()
            }}
            className="p-0.5 hover:bg-muted rounded transition-colors"
            title={isCollapsed ? "Expand pane" : "Collapse pane"}
            aria-label={isCollapsed ? "Expand pane" : "Collapse pane"}
          >
            {isCollapsed ? (
              <ChevronRight className="w-4 h-4 text-muted-foreground" />
            ) : (
              <ChevronDown className="w-4 h-4 text-muted-foreground" />
            )}
          </button>
          <h3 className="font-semibold">
            {serviceName}{port && <span className="text-muted-foreground font-mono">:{port}</span>}
          </h3>
          {/* Running State Badge - shows lifecycle state (running/stopped/starting/etc) - icon only */}
          <span 
            className={cn(
              "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
              // Running states - green
              (processStatus === 'running' || processStatus === 'watching' || processStatus === 'ready') && "bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30",
              // Stopped/completed states - gray
              (processStatus === 'stopped' || processStatus === 'not-started' || processStatus === 'not-running') && "bg-gray-500/10 text-gray-600 dark:text-gray-400 border border-gray-500/30",
              // Completed build states - green (success)
              (processStatus === 'built' || processStatus === 'completed') && "bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30",
              // Transitional states - blue
              (processStatus === 'starting' || processStatus === 'building') && "bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30",
              // Stopping state - yellow
              processStatus === 'stopping' && "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30",
              // Restarting - blue
              processStatus === 'restarting' && "bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30",
              // Failed/error - red
              (processStatus === 'failed' || processStatus === 'error') && "bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30",
              // Unknown
              !processStatus && "bg-muted text-muted-foreground border border-border"
            )}
            title={`Process state: ${processStatus || 'unknown'}`}
          >
            {/* Running states */}
            {(processStatus === 'running' || processStatus === 'ready') && <CheckCircle className="w-3 h-3 shrink-0" />}
            {processStatus === 'watching' && <Eye className="w-3 h-3 shrink-0" />}
            {/* Stopped states */}
            {(processStatus === 'stopped' || processStatus === 'not-started' || processStatus === 'not-running') && <CircleOff className="w-3 h-3 shrink-0" />}
            {/* Build states */}
            {processStatus === 'building' && <Hammer className="w-3 h-3 shrink-0 animate-pulse" />}
            {(processStatus === 'built' || processStatus === 'completed') && <CheckSquare className="w-3 h-3 shrink-0" />}
            {/* Transitional states */}
            {processStatus === 'starting' && <Loader2 className="w-3 h-3 shrink-0 animate-spin" />}
            {processStatus === 'stopping' && <Loader2 className="w-3 h-3 shrink-0 animate-spin" />}
            {processStatus === 'restarting' && <RotateCw className="w-3 h-3 shrink-0 animate-spin" />}
            {/* Error states */}
            {(processStatus === 'failed' || processStatus === 'error') && <CircleX className="w-3 h-3 shrink-0" />}
            {/* Unknown */}
            {!processStatus && <CircleDot className="w-3 h-3 shrink-0" />}
          </span>
          {/* Health Status Badge - from real-time health checks - icon only */}
          {normalizedHealth && (
            <span 
              className={cn(
                "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
                normalizedHealth === 'healthy' && "bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30",
                normalizedHealth === 'degraded' && "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30",
                normalizedHealth === 'unhealthy' && "bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30",
                normalizedHealth === 'unknown' && "bg-muted text-muted-foreground border border-border"
              )}
              title={`Service health: ${normalizedHealth} (from health checks)`}
            >
              {normalizedHealth === 'healthy' ? (
                <Heart className="w-3 h-3 shrink-0 animate-heartbeat" />
              ) : normalizedHealth === 'degraded' ? (
                <HeartPulse className="w-3 h-3 shrink-0 animate-caution-pulse" />
              ) : normalizedHealth === 'unhealthy' ? (
                <HeartCrack className="w-3 h-3 shrink-0 animate-status-flash" />
              ) : (
                <HelpCircle className="w-3 h-3 shrink-0" />
              )}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
          {/* Service lifecycle controls */}
          {service && (
            <div className="mr-2 border-r pr-2 border-border">
              <ServiceActions service={service} variant="compact" />
            </div>
          )}
          {url && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
              title="Open in new tab"
              aria-label="Open service in new tab"
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

      {/* Log Display - only show when not collapsed */}
      {!isCollapsed && (
        <div
          ref={logsContainerRef}
          className="flex-1 overflow-y-auto bg-card p-4 font-mono text-sm"
          role="log"
          aria-live="polite"
          aria-atomic="false"
        >
        {filteredLogs.length === 0 ? (
          <div className="text-center text-muted-foreground py-12">
            {logs.length === 0 ? 'No logs to display' : 'No logs match your search'}
          </div>
        ) : (
          <div className="space-y-0.5">
            {filteredLogs.map((log, idx) => {
              // Determine log level with override check
              const overrideLevel = getClassificationForText(log.message)
              const isError = overrideLevel === 'error' || (!overrideLevel && (isErrorLine(log.message) || log.level === LOG_LEVELS.ERROR))
              const isWarning = overrideLevel === 'warning' || (!overrideLevel && !isError && (isWarningLine(log.message) || log.level === LOG_LEVELS.WARNING))
              const logLevel = isError ? 'error' : isWarning ? 'warning' : 'info'

              return (
                <div
                  key={idx}
                  className="relative group flex items-start gap-1 hover:bg-muted/50 px-1 -mx-1 rounded"
                  onContextMenu={(e) => {
                    e.preventDefault()
                    handleCopyLine(log)
                  }}
                >
                  {/* Log level icon */}
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
                      [{formatLogTimestamp(log.timestamp ?? '')}]
                    </span>
                    {' '}
                    <span dangerouslySetInnerHTML={{ __html: convertAnsiToHtml(log.message ?? '') }} />
                  </div>
                  
                  {/* Copy button - appears on hover */}
                  <button
                    onClick={() => handleCopyLine(log, idx)}
                    className="opacity-0 group-hover:opacity-100 shrink-0 p-1 hover:bg-muted rounded transition-opacity"
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
      )}

      {/* Classification popup */}
      {selectionPosition && selectedText && (
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
            onClick={() => void handleClassifySelection('info')}
            className="h-8 px-2 bg-blue-500 hover:bg-blue-600"
            title="Classify as Info"
          >
            <Info className="w-4 h-4 text-white" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => void handleClassifySelection('warning')}
            className="h-8 px-2 bg-yellow-500 hover:bg-yellow-600"
            title="Classify as Warning"
          >
            <AlertTriangle className="w-4 h-4 text-white" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => void handleClassifySelection('error')}
            className="h-8 px-2 bg-red-500 hover:bg-red-600"
            title="Classify as Error"
          >
            <XCircle className="w-4 h-4 text-white" />
          </Button>
        </div>
      )}
      
      {showClassificationConfirmation && (
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
      )}

    </div>
  )
}
