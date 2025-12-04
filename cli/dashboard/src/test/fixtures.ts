/**
 * Test fixtures for comprehensive dashboard testing
 * Provides factories for all service types, modes, states, and health conditions
 */
import type {
  Service,
  HealthCheckResult,
  HealthSummary,
  HealthReportEvent,
  ServiceType,
  ServiceMode,
  ServiceStatus,
  HealthStatus,
  LocalServiceInfo,
  AzureServiceInfo,
  HealthDetails,
} from '@/types'

// =============================================================================
// Factory Utilities
// =============================================================================

let idCounter = 0

function generateId(): string {
  return `test-${++idCounter}`
}

function generatePort(): number {
  return 3000 + Math.floor(Math.random() * 5000)
}

function generatePid(): number {
  return 10000 + Math.floor(Math.random() * 50000)
}

// =============================================================================
// Service Factories
// =============================================================================

export interface CreateServiceOptions {
  name?: string
  language?: string
  framework?: string
  project?: string
  serviceType?: ServiceType
  serviceMode?: ServiceMode
  status?: ServiceStatus
  health?: HealthStatus
  port?: number
  pid?: number
  url?: string
  error?: string
  startTime?: string
  lastChecked?: string
  healthDetails?: Partial<HealthDetails>
  azure?: Partial<AzureServiceInfo>
  environmentVariables?: Record<string, string>
}

/**
 * Create a service with sensible defaults
 */
export function createService(options: CreateServiceOptions = {}): Service {
  const name = options.name ?? `service-${generateId()}`
  const port = options.port ?? generatePort()
  const serviceType = options.serviceType ?? 'http'
  const isProcess = serviceType === 'process'

  const local: LocalServiceInfo = {
    status: options.status ?? 'running',
    health: options.health ?? 'healthy',
    port: isProcess ? undefined : port,
    pid: options.pid ?? generatePid(),
    url: isProcess ? undefined : (options.url ?? `http://localhost:${port}`),
    startTime: options.startTime ?? new Date().toISOString(),
    lastChecked: options.lastChecked ?? new Date().toISOString(),
    serviceType,
    serviceMode: options.serviceMode,
    healthDetails: options.healthDetails ? {
      checkType: options.healthDetails.checkType ?? (isProcess ? 'process' : 'http'),
      endpoint: options.healthDetails.endpoint,
      responseTime: options.healthDetails.responseTime,
      statusCode: options.healthDetails.statusCode,
      uptime: options.healthDetails.uptime,
      lastError: options.healthDetails.lastError,
      details: options.healthDetails.details,
    } : undefined,
  }

  return {
    name,
    language: options.language ?? 'TypeScript',
    framework: options.framework ?? 'Express',
    project: options.project ?? `/src/${name}`,
    local,
    azure: options.azure ? {
      url: options.azure.url,
      resourceName: options.azure.resourceName,
      imageName: options.azure.imageName,
      resourceType: options.azure.resourceType,
      resourceGroup: options.azure.resourceGroup,
      location: options.azure.location,
      subscriptionId: options.azure.subscriptionId,
    } : undefined,
    environmentVariables: options.environmentVariables,
    error: options.error,
  }
}

// =============================================================================
// Preset Service Factories by Type
// =============================================================================

/**
 * Create an HTTP service (web server, API)
 */
export function createHttpService(options: Omit<CreateServiceOptions, 'serviceType'> = {}): Service {
  return createService({
    ...options,
    serviceType: 'http',
    name: options.name ?? 'api',
    framework: options.framework ?? 'Express',
  })
}

/**
 * Create a TCP service (database, message queue)
 */
export function createTcpService(options: Omit<CreateServiceOptions, 'serviceType'> = {}): Service {
  return createService({
    ...options,
    serviceType: 'tcp',
    name: options.name ?? 'database',
    framework: options.framework ?? 'PostgreSQL',
    language: options.language ?? 'SQL',
  })
}

/**
 * Create a process service (build tool, watcher)
 */
export function createProcessService(options: Omit<CreateServiceOptions, 'serviceType'> = {}): Service {
  return createService({
    ...options,
    serviceType: 'process',
    serviceMode: options.serviceMode ?? 'watch',
    name: options.name ?? 'builder',
    framework: options.framework ?? 'TypeScript',
    port: undefined,
    url: undefined,
  })
}

// =============================================================================
// Preset Service Factories by Mode
// =============================================================================

/**
 * Create a watch mode process service (tsc --watch, nodemon)
 */
export function createWatchService(options: Omit<CreateServiceOptions, 'serviceType' | 'serviceMode'> = {}): Service {
  return createProcessService({
    ...options,
    serviceMode: 'watch',
    name: options.name ?? 'watcher',
    status: options.status ?? 'watching',
  })
}

/**
 * Create a build mode process service (tsc, go build)
 */
export function createBuildService(options: Omit<CreateServiceOptions, 'serviceType' | 'serviceMode'> = {}): Service {
  return createProcessService({
    ...options,
    serviceMode: 'build',
    name: options.name ?? 'compiler',
    status: options.status ?? 'building',
  })
}

/**
 * Create a daemon mode process service (MCP server, worker)
 */
export function createDaemonService(options: Omit<CreateServiceOptions, 'serviceType' | 'serviceMode'> = {}): Service {
  return createProcessService({
    ...options,
    serviceMode: 'daemon',
    name: options.name ?? 'worker',
    status: options.status ?? 'running',
  })
}

/**
 * Create a task mode process service (migrations, scripts)
 */
export function createTaskService(options: Omit<CreateServiceOptions, 'serviceType' | 'serviceMode'> = {}): Service {
  return createProcessService({
    ...options,
    serviceMode: 'task',
    name: options.name ?? 'migration',
    status: options.status ?? 'completed',
  })
}

// =============================================================================
// Preset Service Factories by Lifecycle State
// =============================================================================

export function createNotStartedService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'not-started',
    health: 'unknown',
    pid: undefined,
    startTime: undefined,
  })
}

export function createStartingService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'starting',
    health: 'unknown',
  })
}

export function createRunningService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'running',
    health: options.health ?? 'healthy',
  })
}

export function createStoppingService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'stopping',
    health: 'unknown',
  })
}

export function createStoppedService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'stopped',
    health: 'unknown',
    pid: undefined,
  })
}

export function createRestartingService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'restarting',
    health: 'unknown',
  })
}

export function createFailedService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'failed',
    health: 'unhealthy',
    error: options.error ?? 'Process exited with code 1',
  })
}

export function createCompletedService(options: CreateServiceOptions = {}): Service {
  return createProcessService({
    ...options,
    serviceMode: 'build',
    status: 'completed',
    health: 'healthy',
  })
}

// =============================================================================
// Preset Service Factories by Health Status
// =============================================================================

export function createHealthyService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'running',
    health: 'healthy',
    healthDetails: {
      checkType: 'http',
      responseTime: 50,
      statusCode: 200,
      uptime: 3600,
      ...options.healthDetails,
    },
  })
}

export function createDegradedService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'running',
    health: 'degraded',
    healthDetails: {
      checkType: 'http',
      responseTime: 2500, // Slow response
      statusCode: 200,
      uptime: 3600,
      ...options.healthDetails,
    },
  })
}

export function createUnhealthyService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'running',
    health: 'unhealthy',
    error: options.error ?? 'Health check failed',
    healthDetails: {
      checkType: 'http',
      lastError: 'Connection refused',
      consecutiveFailures: 3,
      ...options.healthDetails,
    },
  })
}

export function createUnknownHealthService(options: CreateServiceOptions = {}): Service {
  return createService({
    ...options,
    status: 'running',
    health: 'unknown',
  })
}

// =============================================================================
// Health Check Result Factories
// =============================================================================

export interface CreateHealthCheckOptions {
  serviceName?: string
  status?: HealthStatus
  checkType?: 'http' | 'tcp' | 'process'
  endpoint?: string
  responseTime?: number // nanoseconds
  statusCode?: number
  error?: string
  port?: number
  pid?: number
  uptime?: number // nanoseconds
  serviceType?: ServiceType
  serviceMode?: ServiceMode
}

export function createHealthCheckResult(options: CreateHealthCheckOptions = {}): HealthCheckResult {
  return {
    serviceName: options.serviceName ?? 'test-service',
    status: options.status ?? 'healthy',
    checkType: options.checkType ?? 'http',
    endpoint: options.endpoint ?? 'http://localhost:3000/health',
    responseTime: options.responseTime ?? 50_000_000, // 50ms in nanoseconds
    statusCode: options.statusCode ?? 200,
    error: options.error,
    timestamp: new Date().toISOString(),
    port: options.port ?? 3000,
    pid: options.pid ?? 12345,
    uptime: options.uptime ?? 3600_000_000_000, // 1 hour in nanoseconds
    serviceType: options.serviceType ?? 'http',
    serviceMode: options.serviceMode,
  }
}

export function createHealthyCheckResult(serviceName: string, options: Omit<CreateHealthCheckOptions, 'serviceName' | 'status'> = {}): HealthCheckResult {
  return createHealthCheckResult({
    ...options,
    serviceName,
    status: 'healthy',
  })
}

export function createUnhealthyCheckResult(serviceName: string, options: Omit<CreateHealthCheckOptions, 'serviceName' | 'status'> = {}): HealthCheckResult {
  return createHealthCheckResult({
    ...options,
    serviceName,
    status: 'unhealthy',
    error: options.error ?? 'Health check failed',
    responseTime: 0,
    statusCode: undefined,
  })
}

export function createDegradedCheckResult(serviceName: string, options: Omit<CreateHealthCheckOptions, 'serviceName' | 'status'> = {}): HealthCheckResult {
  return createHealthCheckResult({
    ...options,
    serviceName,
    status: 'degraded',
    responseTime: options.responseTime ?? 3000_000_000, // 3s - slow
  })
}

// =============================================================================
// Health Summary Factories
// =============================================================================

export interface CreateHealthSummaryOptions {
  total?: number
  healthy?: number
  degraded?: number
  unhealthy?: number
  starting?: number
  stopped?: number
  unknown?: number
  overall?: HealthStatus
}

export function createHealthSummary(options: CreateHealthSummaryOptions = {}): HealthSummary {
  const total = options.total ?? 4
  const healthy = options.healthy ?? 2
  const degraded = options.degraded ?? 1
  const unhealthy = options.unhealthy ?? 0
  const starting = options.starting ?? 1
  const stopped = options.stopped ?? 0
  const unknown = options.unknown ?? 0

  // Calculate overall status
  let overall: HealthStatus = options.overall ?? 'healthy'
  if (!options.overall) {
    if (unhealthy > 0) overall = 'unhealthy'
    else if (degraded > 0) overall = 'degraded'
    else if (unknown > 0 || starting > 0) overall = 'unknown'
  }

  return {
    total,
    healthy,
    degraded,
    unhealthy,
    starting,
    stopped,
    unknown,
    overall,
  }
}

export function createAllHealthySummary(total: number): HealthSummary {
  return createHealthSummary({
    total,
    healthy: total,
    degraded: 0,
    unhealthy: 0,
    starting: 0,
    stopped: 0,
    unknown: 0,
    overall: 'healthy',
  })
}

export function createMixedHealthSummary(): HealthSummary {
  return createHealthSummary({
    total: 5,
    healthy: 2,
    degraded: 1,
    unhealthy: 1,
    starting: 1,
    stopped: 0,
    unknown: 0,
    overall: 'unhealthy',
  })
}

// =============================================================================
// Health Report Event Factories
// =============================================================================

export function createHealthReportEvent(
  services: HealthCheckResult[],
  summary?: HealthSummary
): HealthReportEvent {
  // Auto-calculate summary from services if not provided
  const calculatedSummary = summary ?? {
    total: services.length,
    healthy: services.filter(s => s.status === 'healthy').length,
    degraded: services.filter(s => s.status === 'degraded').length,
    unhealthy: services.filter(s => s.status === 'unhealthy').length,
    starting: 0,
    stopped: 0,
    unknown: services.filter(s => s.status === 'unknown').length,
    overall: services.some(s => s.status === 'unhealthy') ? 'unhealthy' as const :
             services.some(s => s.status === 'degraded') ? 'degraded' as const :
             services.some(s => s.status === 'unknown') ? 'unknown' as const :
             'healthy' as const,
  }

  return {
    type: 'health',
    timestamp: new Date().toISOString(),
    services,
    summary: calculatedSummary,
  }
}

// =============================================================================
// Complete Test Scenarios
// =============================================================================

/**
 * Standard test scenario with mixed services
 */
export const standardScenario = {
  services: [
    createHttpService({ name: 'api', status: 'running', health: 'healthy' }),
    createHttpService({ name: 'web', status: 'running', health: 'healthy' }),
    createTcpService({ name: 'database', status: 'running', health: 'healthy' }),
    createWatchService({ name: 'typescript-watcher', status: 'watching' }),
  ],
  get healthChecks() {
    return this.services.map(s => createHealthyCheckResult(s.name))
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks)
  },
}

/**
 * All services healthy scenario
 */
export const allHealthyScenario = {
  services: [
    createHealthyService({ name: 'api' }),
    createHealthyService({ name: 'web' }),
    createHealthyService({ name: 'worker' }),
  ],
  get healthChecks() {
    return this.services.map(s => createHealthyCheckResult(s.name))
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks)
  },
}

/**
 * Mixed health scenario with various states
 */
export const mixedHealthScenario = {
  services: [
    createHealthyService({ name: 'api' }),
    createDegradedService({ name: 'web' }),
    createUnhealthyService({ name: 'worker' }),
    createStartingService({ name: 'database' }),
    createStoppedService({ name: 'cache' }),
  ],
  get healthChecks() {
    return [
      createHealthyCheckResult('api'),
      createDegradedCheckResult('web'),
      createUnhealthyCheckResult('worker'),
      createHealthCheckResult({ serviceName: 'database', status: 'unknown' }),
    ]
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks, createMixedHealthSummary())
  },
}

/**
 * Process services scenario
 */
export const processServicesScenario = {
  services: [
    createWatchService({ name: 'typescript', status: 'watching' }),
    createBuildService({ name: 'compiler', status: 'building' }),
    createDaemonService({ name: 'mcp-server', status: 'running' }),
    createTaskService({ name: 'migration', status: 'completed' }),
    createProcessService({ name: 'failed-build', serviceMode: 'build', status: 'failed', error: 'Build failed' }),
  ],
  get healthChecks() {
    return this.services.map(s => createHealthCheckResult({
      serviceName: s.name,
      status: s.local?.health ?? 'unknown',
      checkType: 'process',
      serviceType: 'process',
      serviceMode: s.local?.serviceMode,
    }))
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks)
  },
}

/**
 * Error scenario - all services unhealthy
 */
export const errorScenario = {
  services: [
    createUnhealthyService({ name: 'api', error: 'Connection refused' }),
    createFailedService({ name: 'worker', error: 'Process crashed' }),
    createStoppedService({ name: 'database' }),
  ],
  get healthChecks() {
    return [
      createUnhealthyCheckResult('api', { error: 'Connection refused' }),
      createUnhealthyCheckResult('worker', { error: 'Process crashed' }),
    ]
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks, createHealthSummary({
      total: 3,
      healthy: 0,
      unhealthy: 2,
      stopped: 1,
      overall: 'unhealthy',
    }))
  },
}

/**
 * Empty scenario - no services
 */
export const emptyScenario = {
  services: [] as Service[],
  healthChecks: [] as HealthCheckResult[],
  healthReport: createHealthReportEvent([], createHealthSummary({ total: 0, healthy: 0 })),
}

/**
 * Azure deployment scenario
 */
export const azureScenario = {
  services: [
    createHttpService({
      name: 'api',
      azure: {
        url: 'https://api.azurewebsites.net',
        resourceName: 'api-prod',
        resourceType: 'containerapp',
        resourceGroup: 'rg-prod',
        location: 'eastus',
      },
    }),
    createHttpService({
      name: 'web',
      azure: {
        url: 'https://web.azurewebsites.net',
        resourceName: 'web-prod',
        resourceType: 'staticwebapp',
      },
    }),
  ],
  get healthChecks() {
    return this.services.map(s => createHealthyCheckResult(s.name))
  },
  get healthReport() {
    return createHealthReportEvent(this.healthChecks)
  },
}

// =============================================================================
// Log Entry Factories
// =============================================================================

export interface LogEntry {
  service: string
  message: string
  level: number // 0 = info, 2 = warning, 3 = error
  timestamp: string
  isStderr: boolean
}

export function createLogEntry(options: Partial<LogEntry> = {}): LogEntry {
  return {
    service: options.service ?? 'api',
    message: options.message ?? 'Test log message',
    level: options.level ?? 0,
    timestamp: options.timestamp ?? new Date().toISOString(),
    isStderr: options.isStderr ?? false,
  }
}

export function createInfoLog(service: string, message: string): LogEntry {
  return createLogEntry({ service, message, level: 0, isStderr: false })
}

export function createWarningLog(service: string, message: string): LogEntry {
  return createLogEntry({ service, message, level: 2, isStderr: false })
}

export function createErrorLog(service: string, message: string): LogEntry {
  return createLogEntry({ service, message, level: 3, isStderr: true })
}

/**
 * Generate a sequence of realistic logs
 */
export function createLogSequence(service: string, count: number = 10): LogEntry[] {
  const messages = [
    { msg: 'Server starting...', level: 0 },
    { msg: 'Loading configuration...', level: 0 },
    { msg: 'Database connection established', level: 0 },
    { msg: 'Warning: Cache miss for key "session:123"', level: 2 },
    { msg: 'Request: GET /api/users - 200 OK', level: 0 },
    { msg: 'Request: POST /api/data - 201 Created', level: 0 },
    { msg: 'Warning: Slow query detected (>100ms)', level: 2 },
    { msg: 'Error: Connection timeout', level: 3 },
    { msg: 'Request: GET /api/health - 200 OK', level: 0 },
    { msg: 'Graceful shutdown initiated', level: 0 },
  ]

  const baseTime = new Date()
  return Array.from({ length: count }, (_, i) => {
    const { msg, level } = messages[i % messages.length]
    const timestamp = new Date(baseTime.getTime() + i * 1000).toISOString()
    return createLogEntry({
      service,
      message: msg,
      level,
      timestamp,
      isStderr: level === 3,
    })
  })
}

// =============================================================================
// Reset Utilities
// =============================================================================

export function resetIdCounter(): void {
  idCounter = 0
}
