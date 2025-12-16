import type { Service, HealthCheckResult, HealthStatus, HealthSummary, LifecycleState } from '@/types'
import type { OperationState } from '@/contexts/ServiceOperationsContext'

// Re-export OperationState from context for backward compatibility
export type { OperationState } from '@/contexts/ServiceOperationsContext'

// Re-export StatusDisplay and related functions from service-status-display
export { getStatusDisplay, getLogPaneVisualStatus, type StatusDisplay, type VisualStatus } from './service-status-display'

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
 * Get effective lifecycle state and health from a service.
 * Returns status (lifecycle) and health (health status).
 * Operation state takes priority for optimistic UI updates.
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
 * Effective display status for modern UI components.
 * This is the SINGLE SOURCE OF TRUTH for how a service should be displayed.
 * 
 * Maps lifecycle state + health status to a unified display state.
 */
export type EffectiveDisplayStatus = 
  // Lifecycle states (take priority)
  | 'stopped'
  | 'stopping'
  | 'starting'
  | 'restarting'
  | 'not-running'
  // Process service specific states
  | 'watching'
  | 'building'
  | 'built'
  | 'completed'
  | 'failed'
  | 'error'
  // Health-based states (when running)
  | 'healthy'
  | 'degraded'
  | 'unhealthy'
  | 'unknown'
  // Backwards compatibility alias
  | 'running'

/**
 * Get unified display status for a service (SINGLE SOURCE OF TRUTH).
 * Priority: operation state > process lifecycle > process-specific > health
 */
export function getServiceDisplayStatus(
  service: Service,
  healthStatus?: HealthCheckResult,
  operationState?: OperationState
): EffectiveDisplayStatus {
  // Use centralized getEffectiveStatus for lifecycle + health
  const { status, health } = getEffectiveStatus(service, operationState)
  
  // Use real-time health from health stream if available and not in operation
  const effectiveHealth = (operationState === 'idle' || !operationState)
    ? normalizeHealthStatus(healthStatus?.status ?? health)
    : 'unknown'
  
  // 1. Operation states take absolute priority (user-initiated actions)
  if (operationState && operationState !== 'idle') {
    return operationState as EffectiveDisplayStatus
  }
  
  // 2. Process-specific statuses (from backend)
  if (status === 'watching') return 'watching'
  if (status === 'building') return 'building'
  if (status === 'built') return 'built'
  if (status === 'completed') return 'completed'
  if (status === 'failed') return 'failed'
  if (status === 'error') return 'error'
  
  // 3. Lifecycle states take priority over health
  if (status === 'stopped') return 'stopped'
  if (status === 'stopping') return 'stopping'
  if (status === 'starting') return 'starting'
  if (status === 'restarting') return 'restarting'
  if (status === 'not-running' || status === 'not-started') return 'not-running'
  
  // 4. When running/ready, use health status
  if (status === 'running' || status === 'ready') {
    if (effectiveHealth === 'healthy') return 'healthy'
    if (effectiveHealth === 'degraded') return 'degraded'
    if (effectiveHealth === 'unhealthy') return 'unhealthy'
    // Running with unknown health shows as healthy (process is running)
    return 'healthy'
  }
  
  // Fallback
  return 'unknown'
}

/**
 * Check if a service is healthy (running and healthy)
 */
export function isServiceHealthy(status: string, health: string): boolean {
  return (status === 'ready' || status === 'running') && health === 'healthy'
}
