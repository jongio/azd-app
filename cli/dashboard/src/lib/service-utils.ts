import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle, CircleDot, Circle, AlertTriangle, type LucideIcon } from 'lucide-react'
import type { Service, HealthCheckResult, HealthStatus, HealthSummary } from '@/types'

/**
 * Status display configuration for a service
 */
export interface StatusDisplay {
  text: string
  color: string
  textColor: string
  badgeVariant: 'success' | 'warning' | 'destructive' | 'secondary'
  icon: LucideIcon
}

/**
 * Status indicator configuration (icon, color, animation)
 * Used for status icons/dots throughout the UI
 */
export interface StatusIndicator {
  icon: string
  color: string
  animate: string
}

/**
 * Get status indicator for a service status
 * Returns icon, color class, and animation class for status dots/icons
 */
export function getStatusIndicator(status?: string): StatusIndicator {
  const indicators: Record<string, StatusIndicator> = {
    running: { icon: '●', color: 'text-green-500', animate: 'animate-pulse' },
    ready: { icon: '●', color: 'text-green-500', animate: '' },
    starting: { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    restarting: { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    stopping: { icon: '◑', color: 'text-yellow-500', animate: '' },
    stopped: { icon: '◉', color: 'text-gray-400', animate: '' },
    error: { icon: '⚠', color: 'text-red-500', animate: 'animate-pulse' },
    'not-running': { icon: '○', color: 'text-gray-500', animate: '' },
  }
  return indicators[status || 'not-running'] || indicators['not-running']
}

/**
 * Status counts for summary displays in headers/sidebars
 */
export interface StatusCounts {
  running: number
  warn: number
  error: number
  stopped: number
  total: number
}

/**
 * Calculate status counts from services array
 * Process status (local.status) takes priority over health status
 */
export function calculateStatusCounts(
  services: Service[],
  healthSummary?: HealthSummary | null,
  hasActiveErrors?: boolean
): StatusCounts {
  const counts: StatusCounts = {
    running: 0,
    warn: 0,
    error: 0,
    stopped: 0,
    total: services.length,
  }

  // First count stopped services from process status (most accurate source)
  const stoppedFromServices = services.filter(s => 
    (s.local?.status || s.status) === 'stopped'
  ).length
  counts.stopped = stoppedFromServices

  // If we have health summary, use it for running/warn/error (but adjust for stopped services)
  if (healthSummary) {
    // Adjust unhealthy count - stopped services show as unhealthy in health checks
    counts.error = Math.max(0, healthSummary.unhealthy - stoppedFromServices)
    counts.warn = healthSummary.degraded + healthSummary.unknown
    counts.running = healthSummary.healthy
    
    // Add starting services to warn count
    if (healthSummary.starting) {
      counts.warn += healthSummary.starting
    }
    // When we have healthSummary, it provides accurate status - don't use hasActiveErrors
  } else {
    // Calculate from services when health summary is not available
    for (const service of services) {
      const status = service.local?.status || service.status || 'not-running'
      const health = service.local?.health || service.health
      
      // Skip stopped services - already counted
      if (status === 'stopped') continue
      
      if (status === 'not-running' || status === 'error' || health === 'unhealthy') {
        counts.error++
      } else if (health === 'degraded' || health === 'unknown' || status === 'starting' || status === 'stopping') {
        counts.warn++
      } else {
        // healthy/running services
        counts.running++
      }
    }
    
    // Only use hasActiveErrors when we don't have healthSummary
    // If there are active log errors but no service-level errors, show in warn
    if (hasActiveErrors && counts.error === 0) {
      if (counts.running > 0) {
        counts.warn += counts.running
        counts.running = 0
      }
    }
  }

  return counts
}

/**
 * Get overall status indicator for sidebar/header based on service states
 * Returns the most critical status indicator
 */
export function getOverallStatusIndicator(services: Service[]): StatusIndicator {
  const counts = calculateStatusCounts(services)
  
  // Priority: error > starting > running > stopped > not-running
  if (counts.error > 0) {
    return getStatusIndicator('error')
  }
  // Check for starting services via warn count (starting adds to warn)
  const hasStarting = services.some(s => (s.local?.status || s.status) === 'starting')
  if (hasStarting) {
    return getStatusIndicator('starting')
  }
  if (counts.running > 0) {
    return getStatusIndicator('running')
  }
  if (counts.stopped > 0) {
    return getStatusIndicator('stopped')
  }
  return getStatusIndicator('not-running')
}

/**
 * Badge configuration for status display
 */
export interface StatusBadgeConfig {
  color: string
  icon: string
  label: string
}

/**
 * Get badge styling configuration for a process status
 * Used for Badge components in tables and cards
 */
export function getStatusBadgeConfig(status?: string): StatusBadgeConfig {
  const configs: Record<string, StatusBadgeConfig> = {
    running: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '●', label: 'Running' },
    ready: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '●', label: 'Ready' },
    starting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◐', label: 'Starting' },
    restarting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◐', label: 'Restarting' },
    stopping: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◑', label: 'Stopping' },
    stopped: { color: 'bg-gray-400/10 text-gray-400 border-gray-400/20', icon: '◉', label: 'Stopped' },
    error: { color: 'bg-red-500/10 text-red-500 border-red-500/20', icon: '⚠', label: 'Error' },
    'not-running': { color: 'bg-gray-500/10 text-gray-500 border-gray-500/20', icon: '○', label: 'Not Running' },
  }
  return configs[status || 'not-running'] || configs['not-running']
}

/**
 * Badge configuration for health status display
 */
export interface HealthBadgeConfig {
  color: string
  label: string
}

/**
 * Get badge styling configuration for a health status
 * Used for Badge components in tables and cards
 */
export function getHealthBadgeConfig(health?: string): HealthBadgeConfig {
  const configs: Record<string, HealthBadgeConfig> = {
    healthy: { color: 'bg-green-500/10 text-green-500 border-green-500/20', label: 'Healthy' },
    degraded: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', label: 'Degraded' },
    unhealthy: { color: 'bg-red-500/10 text-red-500 border-red-500/20', label: 'Unhealthy' },
    starting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', label: 'Starting' },
    unknown: { color: 'bg-gray-500/10 text-gray-500 border-gray-500/20', label: 'Unknown' },
  }
  return configs[health || 'unknown'] || configs['unknown']
}

/**
 * Operation state type (from useServiceOperations hook)
 */
export type OperationState = 'idle' | 'starting' | 'stopping' | 'restarting'

/**
 * Get the effective status from a service, preferring local status.
 * If an operation state is provided (from useServiceOperations), it takes priority
 * to show optimistic UI updates while operations are in progress.
 */
export function getEffectiveStatus(
  service: Service,
  operationState?: OperationState
): {
  status: string
  health: string
} {
  // If an operation is in progress, use that as the status
  if (operationState && operationState !== 'idle') {
    return {
      status: operationState, // 'starting', 'stopping', or 'restarting' maps to same display status
      health: 'unknown' // Health is unknown during operations
    }
  }
  
  return {
    status: service.local?.status || service.status || 'not-running',
    health: service.local?.health || service.health || 'unknown'
  }
}

/**
 * Get status display configuration based on status and health
 * PRIORITY: Process status (stopped/stopping/starting) takes precedence over health status
 */
export function getStatusDisplay(status: string, health: string): StatusDisplay {
  // PROCESS STATUS TAKES PRIORITY
  // Stopped (intentionally stopped) - check FIRST before health
  if (status === 'stopped') {
    return {
      text: 'Stopped',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      badgeVariant: 'secondary',
      icon: CircleDot
    }
  }

  // Stopping - check before health
  if (status === 'stopping') {
    return {
      text: 'Stopping',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      badgeVariant: 'secondary',
      icon: StopCircle
    }
  }

  // Starting or Restarting - check before health
  if (status === 'starting' || status === 'restarting' || health === 'starting') {
    return {
      text: status === 'restarting' ? 'Restarting' : 'Starting',
      color: 'bg-yellow-500',
      textColor: 'text-yellow-400',
      badgeVariant: 'warning',
      icon: Clock
    }
  }

  // Error
  if (status === 'error') {
    return {
      text: 'Error',
      color: 'bg-red-500',
      textColor: 'text-red-400',
      badgeVariant: 'destructive',
      icon: XCircle
    }
  }

  // Not running (never started)
  if (status === 'not-running') {
    return {
      text: 'Not Running',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      badgeVariant: 'secondary',
      icon: Circle
    }
  }

  // HEALTH STATUS (only when process is running/ready)
  // Running only if status is running/ready AND health is healthy
  if ((status === 'ready' || status === 'running') && health === 'healthy') {
    return {
      text: 'Running',
      color: 'bg-green-500',
      textColor: 'text-green-400',
      badgeVariant: 'success',
      icon: CheckCircle
    }
  }

  // Degraded state
  if ((status === 'ready' || status === 'running') && health === 'degraded') {
    return {
      text: 'Degraded',
      color: 'bg-amber-500',
      textColor: 'text-amber-400',
      badgeVariant: 'warning',
      icon: AlertTriangle
    }
  }

  // Unhealthy state
  if ((status === 'ready' || status === 'running') && health === 'unhealthy') {
    return {
      text: 'Unhealthy',
      color: 'bg-red-500',
      textColor: 'text-red-400',
      badgeVariant: 'destructive',
      icon: XCircle
    }
  }

  // Unknown
  return {
    text: 'Unknown',
    color: 'bg-gray-500',
    textColor: 'text-gray-400',
    badgeVariant: 'secondary',
    icon: AlertCircle
  }
}

/**
 * Check if a service is healthy (running and healthy)
 */
export function isServiceHealthy(status: string, health: string): boolean {
  return (status === 'ready' || status === 'running') && health === 'healthy'
}

/**
 * Format a timestamp as relative time (e.g., "5m ago")
 */
export function formatRelativeTime(timeStr?: string): string {
  if (!timeStr) return 'N/A'
  
  try {
    const date = new Date(timeStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (seconds < 60) return `${seconds}s ago`
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    return `${days}d ago`
  } catch {
    return timeStr
  }
}

/**
 * Format a start time for table display (HH:MM:SS)
 */
export function formatStartTime(timeStr?: string): string {
  if (!timeStr) return '-'
  try {
    const date = new Date(timeStr)
    if (isNaN(date.getTime())) {
      return timeStr
    }
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    })
  } catch {
    return timeStr
  }
}

/**
 * Format a timestamp for log display (HH:MM:SS.mmm)
 */
export function formatLogTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp)
    const time = date.toLocaleTimeString('en-US', { 
      hour12: false, 
      hour: '2-digit', 
      minute: '2-digit', 
      second: '2-digit' 
    })
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${time}.${ms}`
  } catch {
    return timestamp
  }
}

/**
 * Format response time from nanoseconds to human-readable string
 */
export function formatResponseTime(nanos?: number): string {
  if (!nanos || nanos <= 0) return '-'
  const ms = nanos / 1_000_000
  if (ms < 1) return '<1ms'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/**
 * Format uptime from nanoseconds to human-readable string
 */
export function formatUptime(nanos?: number): string {
  if (!nanos || nanos <= 0) return '-'
  const seconds = nanos / 1_000_000_000
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = seconds / 60
  if (minutes < 60) return `${Math.round(minutes)}m`
  const hours = minutes / 60
  if (hours < 24) return `${Math.floor(hours)}h ${Math.round(minutes % 60)}m`
  const days = hours / 24
  return `${Math.floor(days)}d ${Math.round(hours % 24)}h`
}

/**
 * Get health check type display text
 */
export function getCheckTypeDisplay(checkType?: string): string {
  switch (checkType) {
    case 'http':
      return 'HTTP'
    case 'port':
      return 'Port'
    case 'process':
      return 'Process'
    default:
      return 'Unknown'
  }
}

/** Visual status type for UI styling */
export type VisualStatus = 'error' | 'warning' | 'info' | 'healthy' | 'stopped'

/**
 * Get the visual status for a log pane based on process status, health status, and log content.
 * Priority: process status (stopped) > health check status > log-based status
 */
export function getLogPaneVisualStatus(
  serviceHealth: HealthStatus | undefined,
  fallbackStatus: 'error' | 'warning' | 'info',
  processStatus?: string
): VisualStatus {
  // Process status takes priority - if service is stopped, show stopped state
  if (processStatus === 'stopped') return 'stopped'
  
  if (serviceHealth) {
    if (serviceHealth === 'unhealthy') return 'error'
    if (serviceHealth === 'degraded' || serviceHealth === 'starting') return 'warning'
    if (serviceHealth === 'healthy') return 'healthy'
  }
  return fallbackStatus
}

/**
 * Merge health check result into service for display
 */
export function mergeHealthIntoService(
  service: Service,
  healthResult?: HealthCheckResult
): Service {
  if (!healthResult) return service

  return {
    ...service,
    local: {
      ...service.local,
      status: service.local?.status ?? 'not-running',
      health: healthResult.status === 'healthy' ? 'healthy' 
            : healthResult.status === 'degraded' ? 'degraded'
            : healthResult.status === 'unhealthy' ? 'unhealthy'
            : healthResult.status === 'starting' ? 'starting'
            : 'unknown',
      lastChecked: healthResult.timestamp,
      healthDetails: {
        checkType: healthResult.checkType,
        endpoint: healthResult.endpoint,
        responseTime: healthResult.responseTime / 1_000_000, // convert to ms
        statusCode: healthResult.statusCode,
        uptime: healthResult.uptime ? healthResult.uptime / 1_000_000_000 : undefined, // convert to seconds
        lastError: healthResult.error,
        details: healthResult.details,
      },
    },
  }
}
