import type { Service } from '@/types'

/**
 * Count active services (status is 'running' or 'ready')
 */
export function countActiveServices(services: Service[]): number {
  return services.filter(s =>
    s.local?.status === 'running' || s.local?.status === 'ready'
  ).length
}

/**
 * Count unique active ports
 */
export function countActivePorts(services: Service[]): number {
  const ports = services
    .filter(s => s.local?.port != null)
    .map(s => s.local!.port!)
  return new Set(ports).size
}

/**
 * Calculate average uptime in seconds across all running services
 */
export function calculateAverageUptime(services: Service[]): number {
  const now = Date.now()
  const uptimes = services
    .filter(s => s.local?.startTime)
    .map(s => now - new Date(s.local!.startTime!).getTime())

  if (uptimes.length === 0) return 0
  return Math.floor(uptimes.reduce((a, b) => a + b, 0) / uptimes.length / 1000)
}

/**
 * Calculate health score as percentage (0-100)
 */
export function calculateHealthScore(services: Service[]): number {
  if (services.length === 0) return 100
  const healthy = services.filter(s => s.local?.health === 'healthy').length
  return Math.round((healthy / services.length) * 100)
}

/**
 * Format duration in seconds to human-readable string
 */
export function formatDuration(seconds: number): string {
  if (seconds === 0 || seconds < 0) return '-'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
  }
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`
}

/**
 * Format response time in milliseconds to string
 */
export function formatResponseTime(ms: number | null | undefined): string {
  if (ms == null) return '-'
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/**
 * Get response time variant for color coding
 */
export function getResponseTimeVariant(ms: number | null | undefined): 'success' | 'warning' | 'error' | 'default' {
  if (ms == null) return 'default'
  if (ms < 100) return 'success'
  if (ms < 500) return 'warning'
  return 'error'
}

/**
 * Get health score variant for color coding
 */
export function getHealthScoreVariant(score: number): 'success' | 'warning' | 'error' {
  if (score >= 80) return 'success'
  if (score >= 50) return 'warning'
  return 'error'
}

/**
 * Get uptime for a single service in seconds
 */
export function getServiceUptime(service: Service): number | null {
  if (!service.local?.startTime) return null
  const now = Date.now()
  return Math.floor((now - new Date(service.local.startTime).getTime()) / 1000)
}
