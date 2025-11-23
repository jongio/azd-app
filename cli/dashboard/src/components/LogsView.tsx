import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search, Download, Trash2, Pause, Play, ArrowDown } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import type { Service } from '@/types'
import Convert from 'ansi-to-html'

// Constants
const MAX_LOGS_IN_MEMORY = 1000
const INITIAL_LOG_TAIL = 500
const SCROLL_THRESHOLD_PX = 10
const LOG_LEVEL_INFO = 1
const LOG_LEVEL_WARNING = 2
const LOG_LEVEL_ERROR = 3

const ansiConverter = new Convert({
  fg: '#FFF',
  bg: '#000',
  newline: false,
  escapeXML: true,
  stream: false
})

interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

export function LogsView() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [services, setServices] = useState<string[]>([])
  const [selectedService, setSelectedService] = useState<string>('all')
  const [searchTerm, setSearchTerm] = useState('')
  const [isPaused, setIsPaused] = useState(false)
  const [isUserScrolling, setIsUserScrolling] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

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

  // Fetch initial logs and setup WebSocket
  useEffect(() => {
    fetchLogs()
    setupWebSocket()

    return () => {
      wsRef.current?.close()
    }
  }, [selectedService])

  // Auto-scroll to bottom
  useEffect(() => {
    if (!isPaused && !isUserScrolling) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, isPaused, isUserScrolling])

  // Detect manual scrolling
  const handleScroll = () => {
    const container = logsContainerRef.current
    if (!container) return

    const { scrollTop, scrollHeight, clientHeight } = container
    const isAtBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < SCROLL_THRESHOLD_PX

    if (!isAtBottom && !isUserScrolling) {
      // User scrolled up - pause auto-scroll
      setIsUserScrolling(true)
      setIsPaused(true)
    } else if (isAtBottom && isUserScrolling) {
      // User scrolled back to bottom - resume auto-scroll
      setIsUserScrolling(false)
      setIsPaused(false)
    }
  }

  const fetchLogs = async () => {
    const url = selectedService === 'all'
      ? `/api/logs?tail=${INITIAL_LOG_TAIL}`
      : `/api/logs?service=${selectedService}&tail=${INITIAL_LOG_TAIL}`

    try {
      const res = await fetch(url)
      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`)
      }
      const data = await res.json()
      setLogs(data || [])
    } catch (err) {
      console.error('Failed to fetch logs:', err)
      setLogs([])
    }
  }

  const setupWebSocket = () => {
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

    ws.onmessage = (event) => {
      if (!isPaused) {
        try {
          const entry = JSON.parse(event.data)
          setLogs(prev => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
        } catch (err) {
          console.error('Failed to parse log entry:', err)
        }
      }
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      // WebSocket closed
    }

    wsRef.current = ws
  }

  const filteredLogs = useMemo(() => {
    return logs.filter(log =>
      log && log.message && log.message.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }, [logs, searchTerm])

  const exportLogs = useCallback(() => {
    const content = filteredLogs
      .map(log => `[${log.timestamp || ''}] [${log.service || ''}] ${log.message || ''}`)
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
    const newPausedState = !isPaused
    setIsPaused(newPausedState)
    
    // If resuming, scroll to bottom
    if (!newPausedState) {
      setIsUserScrolling(false)
      setTimeout(() => {
        logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
      }, 100)
    }
  }, [isPaused])

  const scrollToBottom = useCallback(() => {
    setIsUserScrolling(false)
    setIsPaused(false)
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  // Error/warning detection regex
  const isErrorLine = useCallback((message: string) => {
    const errorPattern = /\b(error|failed|failure|exception|fatal|panic|critical|crash|died)\b/i
    return errorPattern.test(message)
  }, [])

  const isWarningLine = useCallback((message: string) => {
    const warningPattern = /\b(warn|warning|caution|deprecated)\b/i
    return warningPattern.test(message)
  }, [])

  // Assign consistent colors to services (avoiding red)
  const serviceColors = [
    'text-blue-400',
    'text-green-400', 
    'text-purple-400',
    'text-cyan-400',
    'text-pink-400',
    'text-amber-400',
    'text-teal-400',
    'text-indigo-400',
    'text-lime-400',
    'text-fuchsia-400',
    'text-sky-400',
    'text-violet-400',
  ]

  const getServiceColor = useCallback((serviceName: string) => {
    // Generate consistent color index from service name
    const hash = serviceName.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
    return serviceColors[hash % serviceColors.length]
  }, [])

  const getLogColor = useCallback((log: LogEntry) => {
    // Check message content first for errors/warnings
    if (isErrorLine(log.message)) return 'text-red-400'
    if (isWarningLine(log.message)) return 'text-yellow-400'
    
    // Check log level and stderr
    if (log.isStderr || log.level === LOG_LEVEL_ERROR) return 'text-red-400'
    if (log.level === LOG_LEVEL_WARNING) return 'text-yellow-400'
    if (log.level === LOG_LEVEL_INFO) return 'text-gray-400'
    
    return 'text-foreground'
  }, [isErrorLine, isWarningLine])

  const convertAnsiToHtml = useCallback((text: string) => {
    try {
      return ansiConverter.toHtml(text)
    } catch {
      // If conversion fails, return original text
      return text
    }
  }, [])

  return (
    <div className="space-y-4">
      {/* Controls */}
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
            value={searchTerm}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchTerm(e.target.value)}
            className="pl-10"
          />
        </div>

        <Button
          variant="outline"
          size="icon"
          onClick={togglePause}
          title={isPaused ? 'Resume' : 'Pause'}
        >
          {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
        </Button>

        <Button variant="outline" size="icon" onClick={exportLogs} title="Export logs">
          <Download className="w-4 h-4" />
        </Button>

        <Button variant="outline" size="icon" onClick={clearLogs} title="Clear logs">
          <Trash2 className="w-4 h-4" />
        </Button>
      </div>

      {/* Log Display */}
      <div 
        ref={logsContainerRef}
        onScroll={handleScroll}
        className="bg-card border rounded-lg p-4 h-[600px] overflow-y-auto font-mono text-sm"
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
                  [{formatLogTimestamp(log?.timestamp || '')}]
                </span>
                {' '}
                <span className={getServiceColor(log?.service || 'unknown')}>
                  [{log?.service || 'unknown'}]
                </span>
                {' '}
                <span 
                  dangerouslySetInnerHTML={{ 
                    __html: convertAnsiToHtml(log?.message || '') 
                  }} 
                />
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>

      {/* Status Bar */}
      <div className="text-sm text-muted-foreground flex justify-between items-center">
        <span>
          Showing {filteredLogs.length} of {logs.length} log entries
        </span>
        <div className="flex items-center gap-4">
          {isPaused && (
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
    </div>
  )
}
