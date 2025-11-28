import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle, AlertTriangle, type LucideIcon } from 'lucide-react'
import type { Service, HealthCheckResult } from '@/types'

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
 * Get the effective status from a service, preferring local status
 */
export function getEffectiveStatus(service: Service): {
  status: string
  health: string
} {
  return {
    status: service.local?.status || service.status || 'not-running',
    health: service.local?.health || service.health || 'unknown'
  }
}

/**
 * Get status display configuration based on status and health
 */
export function getStatusDisplay(status: string, health: string): StatusDisplay {
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

  // Degraded state (new)
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

  // Starting (either status or health is starting)
  if (status === 'starting' || health === 'starting') {
    return {
      text: 'Starting',
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

  // Stopping
  if (status === 'stopping') {
    return {
      text: 'Stopping',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      badgeVariant: 'secondary',
      icon: StopCircle
    }
  }

  // Stopped or not-running
  if (status === 'stopped' || status === 'not-running') {
    return {
      text: 'Stopped',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      badgeVariant: 'secondary',
      icon: StopCircle
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
