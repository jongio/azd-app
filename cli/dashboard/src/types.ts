export interface ClassificationOverride {
  id: string
  text: string
  level: 'info' | 'warning' | 'error'
  createdAt: string
}

/** Type of health check performed */
export type HealthCheckType = 'http' | 'tcp' | 'process'

/**
 * Health status - describes service's ability to handle requests
 * This is determined by health checks (HTTP, port, or process checks)
 * 
 * IMPORTANT: Health is INDEPENDENT of lifecycle state
 * - A running service can be unhealthy (process up, health checks fail)
 * - A stopped service has no health status (n/a)
 * 
 * NOTE: 'starting' is NOT a health status - it's a lifecycle state.
 * Services that are starting have 'unknown' health until checked.
 */
export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

/**
 * Process lifecycle state - describes the process's current lifecycle phase
 * This is managed by the service orchestrator
 * 
 * IMPORTANT: Lifecycle is INDEPENDENT of health status
 * - starting: Process is being launched (not yet accepting requests)
 * - running: Process is actively running (may or may not be healthy)
 * - stopped: Process has been intentionally stopped
 */
export type LifecycleState = 
  | 'not-started'  // Never been started
  | 'not-running'  // Alias for not-started (backward compatibility)
  | 'starting'     // Process is being launched
  | 'running'      // Process is actively running
  | 'stopping'     // Process is being terminated
  | 'stopped'      // Process has been intentionally stopped
  | 'restarting'   // Process is being restarted
  | 'completed'    // Process finished successfully (build/task mode)
  | 'failed'       // Process exited with error (build/task mode)

/**
 * Service type - how the service is accessed (protocol level)
 * - http: Serves HTTP/HTTPS traffic (default when ports defined)
 * - tcp: Raw TCP connections (databases, gRPC)
 * - process: No network endpoint (default when no ports)
 * - container: Docker container service (requires Docker)
 */
export type ServiceType = 'http' | 'tcp' | 'process' | 'container'

/**
 * Service run mode - lifecycle behavior for process-type services
 * - watch: Continuous, watches for changes (tsc --watch, nodemon)
 * - build: One-time build, exits on completion (tsc, go build)
 * - daemon: Long-running background process (MCP servers, workers)
 * - task: One-time task run on demand (migrations, scripts)
 */
export type ServiceMode = 'watch' | 'build' | 'daemon' | 'task'

/**
 * Service status - DEPRECATED, use LifecycleState instead
 * Kept for backward compatibility during transition
 */
export type ServiceStatus = LifecycleState | 'ready' | 'error' | 'watching' | 'building' | 'built'

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
  /** Process lifecycle state */
  status: ServiceStatus
  /** Health check status - independent of lifecycle */
  health: HealthStatus
  url?: string
  port?: number
  pid?: number
  startTime?: string
  lastChecked?: string
  healthDetails?: HealthDetails
  serviceType?: ServiceType
  serviceMode?: ServiceMode
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
  status?: ServiceStatus
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
  /** 
   * Health status from the health check
   * Note: Backend may send 'starting' during grace period
   */
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
  serviceType?: ServiceType
  serviceMode?: ServiceMode
}

/** Summary of overall health status */
export interface HealthSummary {
  total: number
  healthy: number
  degraded: number
  unhealthy: number
  starting: number  // Backend sends this - services in startup grace period
  stopped: number   // Services that are stopped (not running)
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
