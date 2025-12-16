import type { Service, ServiceType, ServiceMode } from '@/types'
import { calculateStatusCounts } from './service-status'

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

/**
 * Service type badge configuration
 */
export interface ServiceTypeBadgeConfig {
  color: string
  label: string
  icon: string
}

/**
 * Service mode badge configuration
 */
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
    container: { color: 'bg-sky-500/10 text-sky-500 border-sky-500/20', label: 'Container', icon: '📦' },
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
    container: 'Container',
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
 * Check if a service type is a container service
 */
export function isContainerService(serviceType?: ServiceType): boolean {
  return serviceType === 'container'
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
