import type { Service } from '@/types'

/**
 * Get the status of a service from either local or top-level status
 */
export function getServiceStatus(service: Service): string {
  return service.local?.status || service.status || 'not-running'
}

/**
 * Get the health of a service from either local or top-level health
 */
export function getServiceHealth(service: Service): string {
  return service.local?.health || service.health || 'unknown'
}

/**
 * Check if a service is healthy (running and healthy)
 */
export function isServiceHealthy(service: Service): boolean {
  const status = getServiceStatus(service)
  const health = getServiceHealth(service)
  return (status === 'ready' || status === 'running') && health === 'healthy'
}

/**
 * Get the appropriate color class for a service status
 */
export function getStatusColor(service: Service): string {
  const status = getServiceStatus(service)
  const health = getServiceHealth(service)
  
  if ((status === 'ready' || status === 'running') && health === 'healthy') {
    return 'bg-success'
  }
  if (status === 'starting') {
    return 'bg-warning'
  }
  if (status === 'error' || health === 'unhealthy') {
    return 'bg-destructive'
  }
  return 'bg-muted-foreground'
}

/**
 * Get a human-readable status label
 */
export function getStatusLabel(service: Service): string {
  const status = getServiceStatus(service)
  const health = getServiceHealth(service)
  
  if (health === 'unhealthy') return 'Unhealthy'
  if (health === 'healthy' && (status === 'ready' || status === 'running')) return 'Healthy'
  if (status === 'starting') return 'Starting'
  if (status === 'error') return 'Error'
  if (status === 'not-running') return 'Not Running'
  return 'Unknown'
}

/**
 * Format uptime in seconds to human-readable format
 */
export function formatUptime(seconds: number | undefined): string {
  if (!seconds) return 'N/A'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  
  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`
  }
  return `${secs}s`
}

/**
 * Validate Azure resource identifiers to prevent command injection
 */
export function isValidAzureIdentifier(identifier: string): boolean {
  // Check for dangerous characters that could be used for command injection
  const dangerousChars = /[;|&$`\n\r]/
  return !dangerousChars.test(identifier)
}

/**
 * Group services by a specific property (e.g., language, framework)
 */
export function groupServicesBy<K extends keyof Service>(
  services: Service[],
  property: K
): Record<string, Service[]> {
  return services.reduce((acc, service) => {
    const key = String(service[property] || 'Unknown')
    if (!acc[key]) acc[key] = []
    acc[key].push(service)
    return acc
  }, {} as Record<string, Service[]>)
}
