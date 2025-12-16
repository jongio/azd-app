import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle, CircleDot, Circle, AlertTriangle, Eye, Hammer, type LucideIcon } from 'lucide-react'
import type { HealthStatus } from '@/types'

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
  // Note: normalizeHealthStatus() converts backend 'starting' to 'unknown'
  if ((status === 'ready' || status === 'running') && health === 'unknown') {
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
  // Process status takes priority - terminal states show specific visual status
  // Stopped/not-running states
  if (processStatus === 'stopped' || processStatus === 'not-started' || processStatus === 'not-running') {
    return 'stopped'
  }
  // Completed build/task states show as healthy (success)
  if (processStatus === 'built' || processStatus === 'completed') {
    return 'healthy'
  }
  // Failed state shows as error
  if (processStatus === 'failed' || processStatus === 'error') {
    return 'error'
  }
  // Running/watching states use health status if available
  if (processStatus === 'running' || processStatus === 'watching' || processStatus === 'ready') {
    if (serviceHealth) {
      if (serviceHealth === 'unhealthy') return 'error'
      if (serviceHealth === 'degraded') return 'warning'
      if (serviceHealth === 'healthy') return 'healthy'
    }
    // Running with unknown health shows as healthy
    return 'healthy'
  }
  
  // Transitional states (starting, stopping, building, restarting) use fallback/health
  if (serviceHealth) {
    if (serviceHealth === 'unhealthy') return 'error'
    if (serviceHealth === 'degraded') return 'warning'
    if (serviceHealth === 'healthy') return 'healthy'
    // 'unknown' falls through to fallbackStatus
  }
  return fallbackStatus
}
