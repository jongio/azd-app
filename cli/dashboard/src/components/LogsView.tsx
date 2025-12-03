import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search, Download, Trash2, Pause, Play, ArrowDown } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import type { Service } from '@/types'
import {
  MAX_LOGS_IN_MEMORY,
  INITIAL_LOG_TAIL,
  SCROLL_THRESHOLD_PX,
  LOG_LEVELS,
  convertAnsiToHtml,
  isErrorLine,
  isWarningLine,
  getServiceColor,
} from '@/lib/log-utils'

interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

interface LogsViewProps {
  selectedServices?: Set<string>
  levelFilter?: Set<'info' | 'warning' | 'error'>
  /** External pause control from parent (e.g., ModernLogsView toolbar) */
  isPaused?: boolean
  /** External auto-scroll control from parent */
  autoScrollEnabled?: boolean
  /** External search term from parent */
  globalSearchTerm?: string
  /** Trigger to clear all logs (increment to trigger) */
  clearAllTrigger?: number
  /** Hide internal controls when parent provides them */
  hideControls?: boolean
}

export function LogsView({ 
  selectedServices, 
  levelFilter,
  isPaused: externalIsPaused,
  autoScrollEnabled: externalAutoScroll,
  globalSearchTerm,
  clearAllTrigger = 0,
  hideControls = false,
}: LogsViewProps = {}) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [services, setServices] = useState<string[]>([])
  const [selectedService, setSelectedService] = useState<string>('all')
  const [internalSearchTerm, setInternalSearchTerm] = useState('')
  const [internalIsPaused, setInternalIsPaused] = useState(false)
  const [isUserScrolling, setIsUserScrolling] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const isPausedRef = useRef(false)
  
  // Use external values when provided (controlled mode from ModernLogsView), otherwise internal
  const isPaused = externalIsPaused ?? internalIsPaused
  const autoScroll = externalAutoScroll ?? true
  const searchTerm = globalSearchTerm ?? internalSearchTerm
  
  // Keep ref in sync for WebSocket callback
  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

  // Fetch services list
  useEffect(() => {
    const fetchServices = async () => {
      try {
        const res = await fetch('/api/services')
        if (!res.ok) {
          throw new Error(`HTTP error! status: ${res.status}`)
        }
        const data = await res.json() as Service[]
        const serviceNames = data.map((s) => s.name)
        setServices(serviceNames)
      } catch (err) {
        console.error('Failed to fetch services:', err)
      }
    }
    void fetchServices()
  }, [])

  const fetchLogs = useCallback(async () => {
    const url = selectedService === 'all'
      ? `/api/logs?tail=${INITIAL_LOG_TAIL}`
      : `/api/logs?service=${selectedService}&tail=${INITIAL_LOG_TAIL}`

    try {
      const res = await fetch(url)
      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`)
      }
      const data = await res.json() as LogEntry[]
      setLogs(data ?? [])
    } catch (err) {
      console.error('Failed to fetch logs:', err)
      setLogs([])
    }
  }, [selectedService])

  const setupWebSocket = useCallback(() => {
    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = selectedService === 'all'
      ? `${protocol}//${window.location.host}/api/logs/stream`
      : `${protocol}//${window.location.host}/api/logs/stream?service=${selectedService}`

    const ws = new WebSocket(url)

    ws.onopen = () => {
      // WebSocket connected
    }

    ws.onmessage = (event: MessageEvent<string>) => {
      // Check pause state from ref to get current value (not stale closure)
      if (isPausedRef.current) {
        return
      }
      try {
        const entry = JSON.parse(event.data) as LogEntry
        setLogs(prev => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
      } catch (err) {
        console.error('Failed to parse log entry:', err)
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      // WebSocket closed
    }

    wsRef.current = ws
  }, [selectedService]) // Removed isPaused - WebSocket shouldn't reconnect on pause toggle

  // Fetch initial logs and setup WebSocket
  useEffect(() => {
    void fetchLogs()
    void setupWebSocket()

    return () => {
      wsRef.current?.close()
    }
  }, [fetchLogs, setupWebSocket, selectedService])

  // Auto-scroll to bottom - scroll the container, not the page
  useEffect(() => {
    if (autoScroll && !isPaused && !isUserScrolling && logsContainerRef.current) {
      const container = logsContainerRef.current
      container.scrollTop = container.scrollHeight
    }
  }, [logs, isPaused, isUserScrolling, autoScroll])

  // Clear logs when global clear is triggered
  useEffect(() => {
    if (clearAllTrigger > 0) {
      setLogs([])
    }
  }, [clearAllTrigger])

  // Detect manual scrolling - only affects internal state in uncontrolled mode
  const handleScroll = () => {
    // Skip scroll detection when externally controlled
    if (externalIsPaused !== undefined) return
    
    const container = logsContainerRef.current
    if (!container) return

    const { scrollTop, scrollHeight, clientHeight } = container
    const isAtBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < SCROLL_THRESHOLD_PX

    if (!isAtBottom && !isUserScrolling) {
      // User scrolled up - pause auto-scroll
      setIsUserScrolling(true)
      setInternalIsPaused(true)
    } else if (isAtBottom && isUserScrolling) {
      // User scrolled back to bottom - resume auto-scroll
      setIsUserScrolling(false)
      setInternalIsPaused(false)
    }
  }

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      // Filter by search term
      const matchesSearch = log && log.message && log.message.toLowerCase().includes(searchTerm.toLowerCase())
      if (!matchesSearch) return false

      // Filter by selected services from multi-pane view
      if (selectedServices && selectedServices.size > 0) {
        if (!selectedServices.has(log.service)) return false
      }

      // Filter by dropdown service selection
      if (selectedService !== 'all') {
        if (log.service !== selectedService) return false
      }

      // Filter by log level
      if (levelFilter && levelFilter.size > 0) {
        const isError = log.level === LOG_LEVELS.ERROR || log.isStderr || isErrorLine(log.message)
        const isWarning = log.level === LOG_LEVELS.WARNING || isWarningLine(log.message)
        const isInfo = !isError && !isWarning
        
        if (isError && !levelFilter.has('error')) return false
        if (isWarning && !isError && !levelFilter.has('warning')) return false
        if (isInfo && !levelFilter.has('info')) return false
      }

      return true
    })
  }, [logs, searchTerm, selectedServices, selectedService, levelFilter])

  const exportLogs = useCallback(() => {
    const content = filteredLogs
      .map(log => `[${log.timestamp ?? ''}] [${log.service ?? ''}] ${log.message ?? ''}`)
      .join('\n')

    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `logs-${Date.now()}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }, [filteredLogs])

  const clearLogs = useCallback(() => {
    if (window.confirm(`Clear all ${logs.length} log entries? This cannot be undone.`)) {
      setLogs([])
    }
  }, [logs.length])

  const togglePause = useCallback(() => {
    const newPausedState = !internalIsPaused
    setInternalIsPaused(newPausedState)
    
    // If resuming, scroll to bottom
    if (!newPausedState) {
      setIsUserScrolling(false)
      setTimeout(() => {
        if (logsContainerRef.current) {
          logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight
        }
      }, 100)
    }
  }, [internalIsPaused])

  const scrollToBottom = useCallback(() => {
    setIsUserScrolling(false)
    setInternalIsPaused(false)
    if (logsContainerRef.current) {
      logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight
    }
  }, [])

  const getLogColor = useCallback((log: LogEntry) => {
    // Check message content first for errors/warnings
    if (isErrorLine(log.message)) return 'text-red-400'
    if (isWarningLine(log.message)) return 'text-yellow-400'
    
    // Check log level and stderr
    if (log.isStderr || log.level === LOG_LEVELS.ERROR) return 'text-red-400'
    if (log.level === LOG_LEVELS.WARNING) return 'text-yellow-400'
    if (log.level === LOG_LEVELS.INFO) return 'text-foreground-tertiary'
    
    return 'text-foreground'
  }, [])

  return (
    <div className={cn(hideControls ? "flex flex-col h-full" : "space-y-4")}>
      {/* Controls - only show when not controlled by parent */}
      {!hideControls && (
        <div className="flex gap-4 items-center flex-wrap">
          <Select 
            value={selectedService} 
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setSelectedService(e.target.value)}
            className="min-w-[150px]"
          >
            <option value="all">All Services</option>
            {services.map((service) => (
              <option key={service} value={service}>{service}</option>
            ))}
          </Select>

          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-3 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search logs..."
              value={internalSearchTerm}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setInternalSearchTerm(e.target.value)}
              className="pl-10"
            />
          </div>

          <Button
            variant="outline"
            size="icon"
            onClick={togglePause}
            title={internalIsPaused ? 'Resume' : 'Pause'}
          >
            {internalIsPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          </Button>

          <Button variant="outline" size="icon" onClick={exportLogs} title="Export logs">
            <Download className="w-4 h-4" />
          </Button>

          <Button variant="outline" size="icon" onClick={clearLogs} title="Clear logs">
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      )}

      {/* Log Display */}
      <div 
        ref={logsContainerRef}
        onScroll={handleScroll}
        className={cn(
          "bg-card border rounded-lg p-4 overflow-y-auto font-mono text-sm",
          hideControls ? "flex-1" : "h-[600px]"
        )}
      >
        {filteredLogs.length === 0 ? (
          <div className="text-center text-muted-foreground py-12">
            {logs.length === 0 ? 'No logs to display' : 'No logs match your search'}
          </div>
        ) : (
          <div className="space-y-0.5">
            {filteredLogs.map((log, idx) => (
              <div key={idx} className={getLogColor(log)}>
                <span className="text-muted-foreground text-xs">
                  [{formatLogTimestamp(String(log?.timestamp ?? ''))}]
                </span>
                {' '}
                <span className={getServiceColor(log?.service ?? 'unknown')}>
                  [{log?.service ?? 'unknown'}]
                </span>
                {' '}
                <span 
                  dangerouslySetInnerHTML={{ 
                    __html: convertAnsiToHtml(log?.message ?? '') 
                  }} 
                />
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>

      {/* Status Bar - only show when not controlled */}
      {!hideControls && (
        <div className="text-sm text-muted-foreground flex justify-between items-center">
          <span>
            Showing {filteredLogs.length} of {logs.length} log entries
          </span>
          <div className="flex items-center gap-4">
            {internalIsPaused && (
              <>
                <span className="text-yellow-600 font-medium">⏸ Paused - scroll stopped</span>
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={scrollToBottom}
                  className="flex items-center gap-2"
              >
                <ArrowDown className="w-4 h-4" />
                Jump to Bottom
              </Button>
            </>
          )}
        </div>
      </div>
      )}
    </div>
  )
}
