import type { Service, HealthCheckResult } from '@/types'
import { normalizeHealthStatus } from './service-status'

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
