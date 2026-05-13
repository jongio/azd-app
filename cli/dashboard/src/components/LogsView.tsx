import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { create, protoInt64 } from '@bufbuild/protobuf'
import { Select } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search, Download, Trash2, Pause, Play, ArrowDown, Monitor, Cloud, Loader2 } from 'lucide-react'
import { formatLogTimestamp } from '@/lib/service-utils'
import { cn } from '@/lib/utils'
import { useCodespaceEnv } from '@/hooks/useCodespaceEnv'
import { useSharedLogStream } from '@/hooks/useSharedLogStream'
import { useServicesContext } from '@/contexts/ServicesContext'
import { createAzureClient, createLogsClient } from '@/lib/connectClient'
import { GetAzureLogsRequestSchema } from '@/gen/proto/azdapp/v1/azure_pb.js'
import { GetLogsRequestSchema } from '@/gen/proto/azdapp/v1/logs_pb.js'
import { protoLogEntryToView, type DashboardLogEntry } from '@/lib/log-proto'
import type { LogMode } from './ModeToggle'
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

type LogEntry = DashboardLogEntry

// =============================================================================
// Azure Connect mappers (mirror the WS-era JSON shape)
// =============================================================================

const SECONDS_PER_MINUTE = 60
const SECONDS_PER_HOUR = 3600

/**
 * Map the Azure timeframe preset to seconds for the proto
 * `since_seconds` field. Mirrors the previous REST `?since=` parsing
 * server-side - the dashboard now does the parsing client-side so the
 * proto wire stays a single integer.
 */
function azureTimeRangeToSeconds(preset: '15m' | '30m' | '6h' | '24h'): number {
  switch (preset) {
    case '15m':
      return 15 * SECONDS_PER_MINUTE
    case '30m':
      return 30 * SECONDS_PER_MINUTE
    case '6h':
      return 6 * SECONDS_PER_HOUR
    case '24h':
      return 24 * SECONDS_PER_HOUR
    default:
      return 30 * SECONDS_PER_MINUTE
  }
}

interface LogsViewProps {
  /** Service names from parent with real-time WebSocket updates */
  services?: string[]
  selectedServices?: Set<string>
  levelFilter?: Set<'info' | 'warning' | 'error'>
  /** External pause control from parent (e.g., ConsoleView toolbar) */
  isPaused?: boolean
  /** External auto-scroll control from parent */
  autoScrollEnabled?: boolean
  /** External search term from parent */
  globalSearchTerm?: string
  /** Trigger to clear all logs (increment to trigger) */
  clearAllTrigger?: number
  /** Hide internal controls when parent provides them */
  hideControls?: boolean
  /** Current log source mode (local or azure) */
  logMode?: LogMode
  /** Whether mode is currently being switched */
  isModeSwitching?: boolean
  /** Azure service filter (only used in Azure mode) */
  azureServiceFilter?: string

  /** Azure timeframe preset (only used in Azure mode) */
  timeRange?: { preset: '15m' | '30m' | '6h' | '24h' }

  /** Azure polling refresh interval in milliseconds (only used when azureRealtime is false) */
  syncInterval?: number

  /** Whether to use WebSocket realtime streaming for Azure logs */
  azureRealtime?: boolean
}

export function LogsView({ 
  services: servicesProp,
  selectedServices, 
  levelFilter,
  isPaused: externalIsPaused,
  autoScrollEnabled: externalAutoScroll,
  globalSearchTerm,
  clearAllTrigger = 0,
  hideControls = false,
  logMode = 'local',
  isModeSwitching = false,
  azureServiceFilter = '',
  timeRange,
  syncInterval,
  azureRealtime = false,
}: LogsViewProps = {}) {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [selectedService, setSelectedService] = useState<string>('all')
  const [internalSearchTerm, setInternalSearchTerm] = useState('')
  const [internalIsPaused, setInternalIsPaused] = useState(false)
  const [isUserScrolling, setIsUserScrolling] = useState(false)
  const [isHovering, setIsHovering] = useState(false)
  const [hasFetched, setHasFetched] = useState(false)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const isPausedRef = useRef(false)
  const lastClearTimeRef = useRef<number>(Date.now() - 1000) // Initialize to 1s in the past

  // Get Codespace config for URL transformation in logs
  const { config: codespaceConfig } = useCodespaceEnv()

  // Services are sourced from context (live-updated via the
  // LifecycleService.StreamBroadcast subscription in ServicesProvider)
  // unless the parent supplies an explicit list. The legacy path
  // that re-fetched `/api/services` inside this component has been
  // removed; the provider's stream is the single source of truth.
  const { serviceNames: contextServiceNames } = useServicesContext()
  const services = servicesProp ?? contextServiceNames

  // Use external values when provided (controlled mode from parent), otherwise internal
  const isPaused = externalIsPaused ?? internalIsPaused
  const autoScroll = externalAutoScroll ?? true
  const searchTerm = globalSearchTerm ?? internalSearchTerm

  // Keep ref in sync so the shared-log-stream callback (which runs
  // outside the normal render cycle) sees the current pause state.
  useEffect(() => {
    isPausedRef.current = isPaused
  }, [isPaused])

  const fetchLogs = useCallback(async () => {
    const serviceValue = (logMode === 'azure' && azureServiceFilter)
      ? azureServiceFilter
      : selectedService

    if (logMode === 'azure') {
      // Connect RPC replaces the legacy GET /api/azure/logs. Empty
      // service name maps to "all" (server returns the union); proto
      // requires a non-empty value, so we serialise 'all' as empty and
      // rely on server-side handling matching the legacy REST contract.
      try {
        const sinceSeconds = azureTimeRangeToSeconds(timeRange?.preset ?? '15m')
        const client = createAzureClient()
        const resp = await client.getAzureLogs(
          create(GetAzureLogsRequestSchema, {
            service: serviceValue !== 'all' && serviceValue !== '' ? serviceValue : '',
            sinceSeconds: protoInt64.parse(sinceSeconds),
            tail: INITIAL_LOG_TAIL,
          }),
        )
        setLogs(resp.entries.map(protoLogEntryToView))
      } catch (err) {
        console.error('Failed to fetch azure logs:', err)
        setLogs([])
      } finally {
        setHasFetched(true)
      }
      return
    }

    // Local mode: Connect RPC replaces the legacy GET /api/logs.
    // Empty `serviceName` asks the server for the merged tail across
    // every service (GetAll); a specific service name returns just
    // that service's ring. A missing service surfaces as Connect
    // `NotFound`, which is equivalent to the legacy 404 and handled
    // by `useLogsStream`; here in LogsView we just fall through to
    // the empty-logs state so the pane renders cleanly.
    try {
      const client = createLogsClient()
      const resp = await client.getLogs(
        create(GetLogsRequestSchema, {
          serviceName: serviceValue !== 'all' && serviceValue !== '' ? serviceValue : '',
          tail: INITIAL_LOG_TAIL,
        }),
      )
      setLogs(resp.entries.map(protoLogEntryToView))
    } catch (err) {
      console.error('Failed to fetch local logs:', err)
      setLogs([])
    } finally {
      setHasFetched(true)
    }
  }, [selectedService, logMode, azureServiceFilter, timeRange?.preset])

  // Unified realtime entry handler: both local and Azure streams drop
  // through the same pause + clear-debounce gate so the two code paths
  // can't diverge. Kept as a `useCallback` so the useSharedLogStream
  // effect stays stable across renders.
  const handleRealtimeEntry = useCallback((entry: LogEntry) => {
    if (isPausedRef.current) return
    if (Date.now() - lastClearTimeRef.current < 100) return
    setLogs((prev) => [...prev, entry].slice(-MAX_LOGS_IN_MEMORY))
  }, [])

  // Azure realtime requires a concrete service name (the streamer
  // attaches to a per-resource event-hub); for local mode the server
  // multiplexes every service onto a single stream so empty or 'all'
  // are both valid.
  const streamServiceName = useMemo(() => {
    if (logMode === 'azure') {
      const value = azureServiceFilter || selectedService
      return value && value !== 'all' ? value : ''
    }
    return selectedService && selectedService !== 'all' ? selectedService : ''
  }, [logMode, azureServiceFilter, selectedService])

  // Enable the shared stream whenever we're actively rendering logs
  // for the current mode. Azure honours the `azureRealtime` toggle
  // (polling mode still uses the periodic fetch below); local mode
  // is always streaming since that's the only path after the WS
  // migration. Mode-switching suspends the stream to avoid
  // interleaving entries from the outgoing and incoming modes.
  const streamEnabled = !isModeSwitching && (
    logMode === 'local' ||
    (logMode === 'azure' && azureRealtime && streamServiceName !== '')
  )

  useSharedLogStream({
    serviceName: streamServiceName || 'all',
    enabled: streamEnabled,
    mode: logMode === 'azure' ? 'azure' : 'local',
    onLogEntry: handleRealtimeEntry,
    since: logMode === 'azure' ? timeRange?.preset : undefined,
  })

  // Fetch initial logs whenever the effective fetch key changes.
  // Live updates flow through useSharedLogStream above.
  useEffect(() => {
    void fetchLogs()
  }, [fetchLogs])

  // Azure polling (non-realtime): periodically refetch logs.
  useEffect(() => {
    if (logMode !== 'azure') return
    if (azureRealtime) return
    if (!syncInterval || syncInterval <= 0) return
    if (isPaused) return

    const intervalMs = Math.max(1000, syncInterval)
    const id = globalThis.setInterval(() => {
      void fetchLogs()
    }, intervalMs)

    return () => {
      globalThis.clearInterval(id)
    }
  }, [logMode, azureRealtime, syncInterval, isPaused, fetchLogs])

  // Clear logs when mode/filter/timeframe changes
  useEffect(() => {
    lastClearTimeRef.current = Date.now()
    setLogs([]) // Clear logs when switching modes or changing filter
    setHasFetched(false) // Reset loading state
  }, [logMode, azureServiceFilter, timeRange?.preset])

  // Auto-scroll to bottom - scroll the container, not the page
  // Pause auto-scroll when user is hovering over the logs
  useEffect(() => {
    if (autoScroll && !isPaused && !isUserScrolling && !isHovering && logsContainerRef.current) {
      const container = logsContainerRef.current
      container.scrollTop = container.scrollHeight
    }
  }, [logs, isPaused, isUserScrolling, autoScroll, isHovering])

  // Clear logs when global clear is triggered
  useEffect(() => {
    if (clearAllTrigger > 0) {
      // Record clear time to ignore WebSocket messages for a brief period
      lastClearTimeRef.current = Date.now()
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
      lastClearTimeRef.current = Date.now()
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

      {/* Log Source Mode Indicator */}
      <div 
        className={cn(
          "flex items-center gap-2 px-3 py-1.5 text-xs font-medium border-b rounded-t-lg transition-colors",
          logMode === 'azure' 
            ? "bg-azure-50 dark:bg-azure-900/30 text-azure-700 dark:text-azure-300 border-azure-200 dark:border-azure-700"
            : "bg-slate-50 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700"
        )}
      >
        {isModeSwitching ? (
          <>
            <Loader2 className="w-3.5 h-3.5 animate-spin" />
            <span>Switching to {logMode === 'azure' ? 'Azure' : 'Local'} logs...</span>
          </>
        ) : logMode === 'azure' ? (
          <>
            <Cloud className="w-3.5 h-3.5" />
            <span>Viewing Azure Logs</span>
            <span className="text-azure-500 dark:text-azure-400">•</span>
            <span className="text-azure-500/70 dark:text-azure-400/70">Live from Azure resources</span>
          </>
        ) : (
          <>
            <Monitor className="w-3.5 h-3.5" />
            <span>Viewing Local Logs</span>
            <span className="text-slate-400 dark:text-slate-500">•</span>
            <span className="text-slate-500/70 dark:text-slate-400/70">From local development server</span>
          </>
        )}
      </div>

      {/* Log Display */}
      <div 
        ref={logsContainerRef}
        onScroll={handleScroll}
        onMouseEnter={() => setIsHovering(true)}
        onMouseLeave={() => setIsHovering(false)}
        className={cn(
          "bg-card border rounded-b-lg p-4 overflow-y-auto font-mono text-sm",
          hideControls ? "flex-1" : "h-[600px]"
        )}
        role="log"
        aria-live="polite"
        aria-atomic="false"
      >
        {(isModeSwitching || !hasFetched) && filteredLogs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
            <Loader2 className="w-5 h-5 animate-spin mb-2" />
            <div className="text-sm">
              {logMode === 'azure' ? 'Fetching Azure logs...' : 'Fetching local logs...'}
            </div>
          </div>
        ) : filteredLogs.length === 0 ? (
          <div className="text-center text-muted-foreground py-12">
            {logs.length === 0 ? 'No logs to display' : 'No logs match your search'}
          </div>
        ) : (
          <div className="space-y-0.5 pb-4">
            {filteredLogs.map((log, idx) => {
              const key = `${log?.timestamp ?? ''}-${log?.service ?? 'unknown'}-${idx}`
              return (
              <div key={key} className={cn(getLogColor(log), "select-text")}>
                <span className="text-muted-foreground text-xs">
                  [{formatLogTimestamp(String(log?.timestamp ?? ''))}]
                </span>
                {' '}
                <span className={getServiceColor(log?.service ?? 'unknown')}>
                  [{log?.service ?? 'unknown'}]
                </span>
                {' '}
                <span 
                  className="text-foreground"
                  dangerouslySetInnerHTML={{ 
                    __html: convertAnsiToHtml(log?.message ?? '', codespaceConfig) 
                  }} 
                />
              </div>
              )
            })}
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
