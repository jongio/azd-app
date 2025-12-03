import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle, CircleDot, Circle, AlertTriangle, Eye, Hammer, type LucideIcon } from 'lucide-react'
import type { Service, HealthCheckResult, HealthStatus, HealthSummary, ServiceType, ServiceMode, LifecycleState } from '@/types'

// Re-export OperationState from context for backward compatibility
export type { OperationState } from '@/contexts/ServiceOperationsContext'

/**
 * Normalize health status from backend
 * Backend may send 'starting' during grace period - we treat this as 'unknown'
 * since 'starting' is a lifecycle state, not a health status
 */
export function normalizeHealthStatus(status: string | undefined): HealthStatus {
  if (!status) return 'unknown'
  // 'starting' from backend means health checks haven't passed yet
  // This is actually unknown health, not a health condition
  if (status === 'starting') return 'unknown'
  // Validate it's a known health status
  if (['healthy', 'degraded', 'unhealthy', 'unknown'].includes(status)) {
    return status as HealthStatus
  }
  return 'unknown'
}

/**
 * Normalize lifecycle state from backend
 * Maps various status values to clean LifecycleState
 */
export function normalizeLifecycleState(status: string | undefined): LifecycleState {
  if (!status) return 'not-started'
  
  // Map legacy/alias values to clean lifecycle states
  const stateMap: Record<string, LifecycleState> = {
    'ready': 'running',
    'error': 'failed',
    'watching': 'running',
    'building': 'running',
    'built': 'completed',
    'not-running': 'not-started',
  }
  
  const mapped = stateMap[status]
  if (mapped) return mapped
  
  // Check if it's already a valid lifecycle state
  const validStates: LifecycleState[] = [
    'not-started', 'starting', 'running', 'stopping', 
    'stopped', 'restarting', 'completed', 'failed'
  ]
  if (validStates.includes(status as LifecycleState)) {
    return status as LifecycleState
  }
  
  return 'not-started'
}

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
  const indicators = {
    running: { icon: '●', color: 'text-green-500', animate: 'animate-pulse' },
    ready: { icon: '●', color: 'text-green-500', animate: '' },
    starting: { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    restarting: { icon: '◐', color: 'text-yellow-500', animate: 'animate-spin' },
    stopping: { icon: '◑', color: 'text-yellow-500', animate: '' },
    stopped: { icon: '◉', color: 'text-gray-400', animate: '' },
    error: { icon: '⚠', color: 'text-red-500', animate: 'animate-pulse' },
    'not-running': { icon: '○', color: 'text-gray-500', animate: '' },
    // Process service specific statuses
    watching: { icon: '👁', color: 'text-green-500', animate: '' },
    building: { icon: '🔨', color: 'text-yellow-500', animate: 'animate-pulse' },
    built: { icon: '✓', color: 'text-green-500', animate: '' },
    failed: { icon: '✗', color: 'text-red-500', animate: '' },
    completed: { icon: '✓', color: 'text-green-500', animate: '' },
  } as const satisfies Record<string, StatusIndicator>
  return indicators[status as keyof typeof indicators] ?? indicators['not-running']
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
  const configs = {
    running: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '●', label: 'Running' },
    ready: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '●', label: 'Ready' },
    starting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◐', label: 'Starting' },
    restarting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◐', label: 'Restarting' },
    stopping: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '◑', label: 'Stopping' },
    stopped: { color: 'bg-gray-400/10 text-gray-400 border-gray-400/20', icon: '◉', label: 'Stopped' },
    error: { color: 'bg-red-500/10 text-red-500 border-red-500/20', icon: '⚠', label: 'Error' },
    'not-running': { color: 'bg-gray-500/10 text-gray-500 border-gray-500/20', icon: '○', label: 'Not Running' },
    // Process service specific statuses
    watching: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '👁', label: 'Watching' },
    building: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', icon: '🔨', label: 'Building' },
    built: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '✓', label: 'Built' },
    failed: { color: 'bg-red-500/10 text-red-500 border-red-500/20', icon: '✗', label: 'Failed' },
    completed: { color: 'bg-green-500/10 text-green-500 border-green-500/20', icon: '✓', label: 'Completed' },
  } as const satisfies Record<string, StatusBadgeConfig>
  return configs[status as keyof typeof configs] ?? configs['not-running']
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
  const configs = {
    healthy: { color: 'bg-green-500/10 text-green-500 border-green-500/20', label: 'Healthy' },
    degraded: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', label: 'Degraded' },
    unhealthy: { color: 'bg-red-500/10 text-red-500 border-red-500/20', label: 'Unhealthy' },
    starting: { color: 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20', label: 'Starting' },
    unknown: { color: 'bg-gray-500/10 text-gray-500 border-gray-500/20', label: 'Unknown' },
  } as const satisfies Record<string, HealthBadgeConfig>
  return configs[health as keyof typeof configs] ?? configs['unknown']
}

// Import OperationState from context (re-exported at top of file)
import type { OperationState } from '@/contexts/ServiceOperationsContext'

/**
 * Get the effective lifecycle state and health from a service.
 * 
 * IMPORTANT: This returns TWO independent values:
 * - status: The process lifecycle state (starting, running, stopped, etc.)
 * - health: The service health status (healthy, unhealthy, degraded, unknown)
 * 
 * If an operation state is provided (from useServiceOperations), it takes priority
 * for the lifecycle state to show optimistic UI updates.
 */
export function getEffectiveStatus(
  service: Service,
  operationState?: OperationState
): {
  status: string
  health: string
} {
  // If an operation is in progress, use that as the lifecycle state
  if (operationState && operationState !== 'idle') {
    return {
      status: operationState, // 'starting', 'stopping', or 'restarting'
      health: 'unknown' // Health is unknown during operations
    }
  }
  
  // Get raw values from service
  const rawStatus = service.local?.status ?? service.status ?? 'not-running'
  const rawHealth = service.local?.health ?? service.health ?? 'unknown'
  
  return {
    status: rawStatus,
    health: normalizeHealthStatus(rawHealth)
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

  // PROCESS SERVICE SPECIFIC STATUSES
  // Watching - process service is actively watching for file changes
  if (status === 'watching') {
    return {
      text: 'Watching',
      color: 'bg-green-500',
      textColor: 'text-green-400',
      badgeVariant: 'success',
      icon: Eye
    }
  }

  // Building - process service is currently building
  if (status === 'building') {
    return {
      text: 'Building',
      color: 'bg-yellow-500',
      textColor: 'text-yellow-400',
      badgeVariant: 'warning',
      icon: Hammer
    }
  }

  // Built - process service completed build successfully
  if (status === 'built') {
    return {
      text: 'Built',
      color: 'bg-green-500',
      textColor: 'text-green-400',
      badgeVariant: 'success',
      icon: CheckCircle
    }
  }

  // Completed - process service task completed successfully
  if (status === 'completed') {
    return {
      text: 'Completed',
      color: 'bg-green-500',
      textColor: 'text-green-400',
      badgeVariant: 'success',
      icon: CheckCircle
    }
  }

  // Failed - process service build/task failed
  if (status === 'failed') {
    return {
      text: 'Failed',
      color: 'bg-red-500',
      textColor: 'text-red-400',
      badgeVariant: 'destructive',
      icon: XCircle
    }
  }

  // Starting or Restarting - only based on process status, NOT health status
  // Health "starting" means health checks haven't passed yet, not that the process is starting
  if (status === 'starting' || status === 'restarting') {
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

  // Running with unknown health (e.g., health checks still pending/warming up)
  // Still show as Running since the process is confirmed running
  if ((status === 'ready' || status === 'running') && (health === 'unknown' || health === 'starting')) {
    return {
      text: 'Running',
      color: 'bg-green-500',
      textColor: 'text-green-400',
      badgeVariant: 'success',
      icon: CheckCircle
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
    if (Number.isNaN(date.getTime())) {
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
  const displayMap = {
    http: 'HTTP',
    tcp: 'TCP',
    process: 'Process',
  } as const
  return displayMap[checkType as keyof typeof displayMap] ?? 'Unknown'
}

/** Visual status type for UI styling */
export type VisualStatus = 'error' | 'warning' | 'info' | 'healthy' | 'stopped'

/**
 * Get the visual status for a log pane based on process status, health status, and log content.
 * Priority: process status (stopped) > health check status > log-based status
 * 
 * NOTE: serviceHealth should be normalized (no 'starting' - use normalizeHealthStatus first)
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
    if (serviceHealth === 'degraded') return 'warning'
    if (serviceHealth === 'healthy') return 'healthy'
    // 'unknown' falls through to fallbackStatus
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
      health: normalizeHealthStatus(healthResult.status),
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

// ============================================================================
// Service Type and Mode Utilities
// ============================================================================

export interface ServiceTypeBadgeConfig {
  color: string
  label: string
  icon: string
}

export interface ServiceModeBadgeConfig {
  color: string
  label: string
  description: string
}

/**
 * Get display configuration for a service type
 */
export function getServiceTypeBadgeConfig(serviceType?: ServiceType): ServiceTypeBadgeConfig {
  const configs: Record<ServiceType, ServiceTypeBadgeConfig> = {
    http: { color: 'bg-blue-500/10 text-blue-500 border-blue-500/20', label: 'HTTP', icon: '🌐' },
    tcp: { color: 'bg-purple-500/10 text-purple-500 border-purple-500/20', label: 'TCP', icon: '🔌' },
    process: { color: 'bg-cyan-500/10 text-cyan-500 border-cyan-500/20', label: 'Process', icon: '⚙️' },
  }
  return configs[serviceType ?? 'http']
}

/**
 * Get display configuration for a service mode
 */
export function getServiceModeBadgeConfig(serviceMode?: ServiceMode): ServiceModeBadgeConfig {
  const configs: Record<ServiceMode, ServiceModeBadgeConfig> = {
    watch: { color: 'bg-green-500/10 text-green-500 border-green-500/20', label: 'Watch', description: 'Continuously watching for file changes' },
    build: { color: 'bg-orange-500/10 text-orange-500 border-orange-500/20', label: 'Build', description: 'One-time build, exits on completion' },
    daemon: { color: 'bg-indigo-500/10 text-indigo-500 border-indigo-500/20', label: 'Daemon', description: 'Long-running background process' },
    task: { color: 'bg-gray-500/10 text-gray-500 border-gray-500/20', label: 'Task', description: 'On-demand one-time execution' },
  }
  return configs[serviceMode ?? 'watch']
}

/**
 * Get a human-readable label for a service type
 */
export function getServiceTypeLabel(serviceType?: ServiceType): string {
  const labels: Record<ServiceType, string> = {
    http: 'HTTP Service',
    tcp: 'TCP Service',
    process: 'Process Service',
  }
  return labels[serviceType ?? 'http']
}

/**
 * Get a human-readable label for a service mode
 */
export function getServiceModeLabel(serviceMode?: ServiceMode): string {
  const labels: Record<ServiceMode, string> = {
    watch: 'Watch Mode',
    build: 'Build Mode',
    daemon: 'Daemon Mode',
    task: 'Task Mode',
  }
  return labels[serviceMode ?? 'watch']
}

/**
 * Check if a service type is process-based (no network endpoint)
 */
export function isProcessService(serviceType?: ServiceType): boolean {
  return serviceType === 'process'
}

/**
 * Check if a service mode indicates continuous operation
 */
export function isContinuousMode(serviceMode?: ServiceMode): boolean {
  return serviceMode === 'watch' || serviceMode === 'daemon'
}

/**
 * Check if a service mode indicates one-time execution
 */
export function isOneTimeMode(serviceMode?: ServiceMode): boolean {
  return serviceMode === 'build' || serviceMode === 'task'
}
