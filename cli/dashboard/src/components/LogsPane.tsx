import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Copy, AlertTriangle, Info, XCircle, Check, ChevronDown, ChevronRight } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import type { LogPattern } from '@/hooks/useLogPatterns'
import { useLogClassification } from '@/hooks/useLogClassification'
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
  patterns: LogPattern[]
  onCopy: (logs: LogEntry[]) => void
  isPaused: boolean
  globalSearchTerm?: string
  autoScrollEnabled?: boolean
  clearAllTrigger?: number
  levelFilter?: Set<'info' | 'warning' | 'error'>
}

export function LogsPane({ serviceName, patterns, onCopy, isPaused, globalSearchTerm = '', autoScrollEnabled = true, clearAllTrigger = 0, levelFilter = new Set(['info', 'warning', 'error'] as const) }: LogsPaneProps) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [selectedText, setSelectedText] = useState<string>('')
  const [selectionPosition, setSelectionPosition] = useState<{ x: number; y: number } | null>(null)
  const [showClassificationConfirmation, setShowClassificationConfirmation] = useState(false)
  const [copiedLineIndex, setCopiedLineIndex] = useState<number | null>(null)
  const [isCollapsed, setIsCollapsed] = useState<boolean>(() => {
    const saved = localStorage.getItem(`logs-pane-collapsed-${serviceName}`)
    return saved === 'true'
  })
  
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  
  const { addOverride, getClassificationForText } = useLogClassification()

  // Toggle collapse state and persist
  const toggleCollapsed = useCallback(() => {
    setIsCollapsed(prev => {
      const newValue = !prev
      localStorage.setItem(`logs-pane-collapsed-${serviceName}`, String(newValue))
      return newValue
    })
  }, [serviceName])

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
          setLogs(data || [])
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
      // Note: isPaused is captured at effect creation time
      // For real-time pause behavior, we use a ref or different approach
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
        await addOverride(selectedText, level)
        setSelectedText('')
        setSelectionPosition(null)
        window.getSelection()?.removeAllRanges()
        
        // Show confirmation feedback
        setShowClassificationConfirmation(true)
        setTimeout(() => {
          setShowClassificationConfirmation(false)
        }, 2000)
      } catch (err) {
        console.error('Failed to add classification override:', err)
      }
    }
  }, [selectedText, addOverride])

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
    // Check if any part of the message has a classification override
    const overrideLevel = getClassificationForText(message)
    if (overrideLevel === 'error') return true
    if (overrideLevel === 'info' || overrideLevel === 'warning') return false

    // Check global patterns
    for (const pattern of patterns) {
      if (pattern.enabled && pattern.source === 'user') {
        try {
          const regex = new RegExp(pattern.regex, 'i')
          if (regex.test(message)) return false
        } catch {
          // Invalid regex, skip
        }
      }
    }

    // Use centralized error detection
    return baseIsErrorLine(message)
  }, [patterns, getClassificationForText])

  const isWarningLine = useCallback((message: string) => {
    // Check if any part of the message has a classification override
    const overrideLevel = getClassificationForText(message)
    if (overrideLevel === 'warning') return true
    if (overrideLevel === 'info' || overrideLevel === 'error') return false
    
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

  const borderClass = {
    error: 'border-red-500 animate-pulse',
    warning: 'border-yellow-500',
    info: 'border-gray-300'
  }[paneStatus]

  return (
    <div 
      className={cn("flex flex-col border-4 rounded-lg overflow-hidden", isCollapsed ? "h-auto" : "h-full", borderClass)}
      role="region"
      aria-label={`Logs for ${serviceName}`}
    >
      {/* Header */}
      <div 
        className="flex items-center justify-between px-4 py-2 bg-card border-b cursor-pointer select-none"
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
          <h3 className="font-semibold">{serviceName}</h3>
          <span className={cn(
            "px-2 py-0.5 text-xs rounded-full font-medium",
            paneStatus === 'error' && "bg-destructive/10 text-destructive border border-destructive/30",
            paneStatus === 'warning' && "bg-warning/10 text-warning border border-warning/30",
            paneStatus === 'info' && "bg-muted text-muted-foreground border border-border"
          )}>
            {paneStatus}
          </span>
          <span className="text-xs text-muted-foreground">
            {filteredLogs.length} / {logs.length} logs
          </span>
        </div>
        <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
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
                      [{formatLogTimestamp(log?.timestamp || '')}]
                    </span>
                    {' '}
                    <span dangerouslySetInnerHTML={{ __html: convertAnsiToHtml(log?.message || '') }} />
                  </div>
                  
                  {/* Copy button - appears on hover */}
                  <button
                    onClick={() => handleCopyLine(log, idx)}
                    className="opacity-0 group-hover:opacity-100 shrink-0 p-1 hover:bg-muted rounded transition-opacity cursor-pointer"
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
