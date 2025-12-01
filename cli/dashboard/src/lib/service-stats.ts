import type { Service } from '@/types'

/**
 * Count running services (status is 'running' or 'ready')
 */
export function countRunningServices(services: Service[]): number {
  return services.filter(s =>
    s.local?.status === 'running' || s.local?.status === 'ready'
  ).length
}

/**
 * Count healthy services
 */
export function countHealthyServices(services: Service[]): number {
  return services.filter(s =>
    s.local?.health === 'healthy'
  ).length
}

/**
 * Count services with errors
 */
export function countErrorServices(services: Service[]): number {
  return services.filter(s =>
    s.local?.status === 'error' || s.local?.health === 'unhealthy'
  ).length
}

/**
 * Get singular or plural form based on count
 */
export function pluralize(count: number, singular: string, plural: string): string {
  return count === 1 ? singular : plural
}
