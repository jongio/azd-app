export interface ClassificationOverride {
  id: string
  text: string
  level: 'info' | 'warning' | 'error'
  createdAt: string
}

/** Type of health check performed */
export type HealthCheckType = 'http' | 'port' | 'process'

/** Health status values */
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'starting' | 'unknown'

/** Detailed health check information */
export interface HealthDetails {
  checkType: HealthCheckType
  endpoint?: string
  responseTime?: number  // milliseconds
  statusCode?: number
  uptime?: number  // seconds
  lastError?: string
  consecutiveFailures?: number
  details?: Record<string, unknown>
}

export interface LocalServiceInfo {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: 'healthy' | 'degraded' | 'unhealthy' | 'unknown'
  url?: string
  port?: number
  pid?: number
  startTime?: string
  lastChecked?: string
  healthDetails?: HealthDetails
}

export interface AzureServiceInfo {
  url?: string
  resourceName?: string
  imageName?: string
  resourceType?: string  // containerapp, appservice, function, etc.
  resourceGroup?: string
  location?: string
  subscriptionId?: string
  logAnalyticsId?: string
  containerAppEnvId?: string
}

export interface Service {
  name: string
  language?: string
  framework?: string
  project?: string
  local?: LocalServiceInfo
  azure?: AzureServiceInfo
  environmentVariables?: Record<string, string>
  // Legacy fields for compatibility during transition
  status?: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health?: 'healthy' | 'unhealthy' | 'unknown'
  startTime?: string
  lastChecked?: string
  error?: string
}

export interface ServiceUpdate {
  type: 'update' | 'add' | 'remove'
  service: Service
}

// ============================================================================
// Health Streaming Types (SSE)
// ============================================================================

/** Health event types for SSE streaming */
export type HealthEventType = 'health' | 'health-change' | 'heartbeat'

/** Base health event */
export interface HealthEvent {
  type: HealthEventType
  timestamp: string
}

/** Health check result for a single service */
export interface HealthCheckResult {
  serviceName: string
  status: HealthStatus
  checkType: HealthCheckType
  endpoint?: string
  responseTime: number  // nanoseconds from Go, convert to ms
  statusCode?: number
  error?: string
  timestamp: string
  details?: Record<string, unknown>
  port?: number
  pid?: number
  uptime?: number  // nanoseconds from Go
}

/** Summary of overall health status */
export interface HealthSummary {
  total: number
  healthy: number
  degraded: number
  unhealthy: number
  unknown: number
  overall: HealthStatus
}

/** Full health report event */
export interface HealthReportEvent extends HealthEvent {
  type: 'health'
  services: HealthCheckResult[]
  summary: HealthSummary
}

/** Health change notification event */
export interface HealthChangeEvent extends HealthEvent {
  type: 'health-change'
  service: string
  oldStatus: string
  newStatus: string
  reason?: string
}

/** Heartbeat keep-alive event */
export interface HeartbeatEvent extends HealthEvent {
  type: 'heartbeat'
}

/** Union type for all health events */
export type AnyHealthEvent = HealthReportEvent | HealthChangeEvent | HeartbeatEvent
