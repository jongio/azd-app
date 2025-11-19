import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle, type LucideIcon } from 'lucide-react'
import type { Service } from '@/types'

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

  // Starting
  if (status === 'starting') {
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
