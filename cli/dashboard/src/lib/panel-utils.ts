/**
 * Helper functions for ServiceDetailPanel component
 */
import type { Service } from '@/types'

/**
 * Format uptime from timestamp to human-readable string
 */
export function formatUptime(startTime?: string): string {
  if (!startTime) return 'N/A'
  const start = new Date(startTime)
  const now = new Date()
  const diff = now.getTime() - start.getTime()

  if (diff < 0) return 'N/A'

  const hours = Math.floor(diff / (1000 * 60 * 60))
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60))
  const seconds = Math.floor((diff % (1000 * 60)) / 1000)

  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

/**
 * Format timestamp to locale string
 */
export function formatTimestamp(timestamp?: string): string {
  if (!timestamp) return 'N/A'
  const date = new Date(timestamp)
  if (isNaN(date.getTime())) return 'N/A'
  return date.toLocaleString()
}

/**
 * Get text color for service status
 */
export function getStatusColor(status?: string): string {
  const colors: Record<string, string> = {
    running: 'text-green-500',
    ready: 'text-green-500',
    starting: 'text-yellow-500',
    stopping: 'text-yellow-500',
    stopped: 'text-gray-500',
    error: 'text-red-500',
    'not-running': 'text-gray-500',
  }
  return colors[status || 'not-running'] || 'text-gray-500'
}

/**
 * Get text color for health status
 */
export function getHealthColor(health?: string): string {
  const colors: Record<string, string> = {
    healthy: 'text-green-500',
    degraded: 'text-yellow-500',
    unhealthy: 'text-red-500',
    unknown: 'text-gray-500',
  }
  return colors[health || 'unknown'] || 'text-gray-500'
}

/**
 * Build Azure Portal URL for a service
 */
export function buildAzurePortalUrl(service: Service): string | null {
  const azure = service.azure
  if (!azure?.subscriptionId || !azure?.resourceGroup || !azure?.resourceName) {
    return null
  }
  
  // Determine provider based on resource type
  const resourceType = azure.resourceType?.toLowerCase()
  let provider = 'Microsoft.App/containerApps'
  
  if (resourceType === 'appservice' || resourceType === 'webapp') {
    provider = 'Microsoft.Web/sites'
  } else if (resourceType === 'function') {
    provider = 'Microsoft.Web/sites'
  } else if (resourceType === 'aks') {
    provider = 'Microsoft.ContainerService/managedClusters'
  }

  return `https://portal.azure.com/#@/resource/subscriptions/${azure.subscriptionId}/resourceGroups/${azure.resourceGroup}/providers/${provider}/${azure.resourceName}`
}

/**
 * Check if a key represents a sensitive value
 */
export function isSensitiveKey(key: string): boolean {
  const sensitivePatterns = [
    /password/i,
    /secret/i,
    /key/i,
    /token/i,
    /api[-_]?key/i,
    /auth/i,
    /credential/i,
    /private/i,
    /connection[-_]?string/i,
  ]
  return sensitivePatterns.some(pattern => pattern.test(key))
}

/**
 * Mask a sensitive value
 */
export function maskValue(value: string): string {
  if (value.length <= 4) return '•'.repeat(value.length)
  return '•'.repeat(Math.min(value.length, 16))
}

/**
 * Format resource type to display name
 */
export function formatResourceType(resourceType?: string): string {
  if (!resourceType) return 'Unknown'
  
  const typeMap: Record<string, string> = {
    containerapp: 'Container App',
    appservice: 'App Service',
    webapp: 'Web App',
    function: 'Function App',
    aks: 'Kubernetes Service',
    staticwebapp: 'Static Web App',
  }
  
  return typeMap[resourceType.toLowerCase()] || resourceType
}

/**
 * Get status display with indicator
 */
export function getStatusDisplay(status?: string): { text: string; indicator: string } {
  const displays: Record<string, { text: string; indicator: string }> = {
    running: { text: 'Running', indicator: '●' },
    ready: { text: 'Ready', indicator: '●' },
    starting: { text: 'Starting', indicator: '◐' },
    stopping: { text: 'Stopping', indicator: '◑' },
    stopped: { text: 'Stopped', indicator: '○' },
    error: { text: 'Error', indicator: '⚠' },
    'not-running': { text: 'Not Running', indicator: '○' },
  }
  return displays[status || 'not-running'] || displays['not-running']
}

/**
 * Get health display with indicator
 */
export function getHealthDisplay(health?: string): { text: string; indicator: string } {
  const displays: Record<string, { text: string; indicator: string }> = {
    healthy: { text: 'Healthy', indicator: '●' },
    degraded: { text: 'Degraded', indicator: '◐' },
    unhealthy: { text: 'Unhealthy', indicator: '●' },
    unknown: { text: 'Unknown', indicator: '○' },
  }
  return displays[health || 'unknown'] || displays['unknown']
}

/**
 * Format health check type
 */
export function formatCheckType(checkType?: string): string {
  if (!checkType) return 'Unknown'
  
  const typeMap: Record<string, string> = {
    http: 'HTTP',
    port: 'Port',
    process: 'Process',
    tcp: 'TCP',
  }
  
  return typeMap[checkType.toLowerCase()] || checkType
}

/**
 * Check if service has Azure deployment
 */
export function hasAzureDeployment(service: Service): boolean {
  return !!(
    service.azure?.resourceName ||
    service.azure?.url
  )
}
