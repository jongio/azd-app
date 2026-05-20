import type { Page } from '@playwright/test'
import type { Service, HealthCheckResult, HealthSummary } from '../../src/types'
import {
  mockConnectUnary,
  mockConnectServerStream,
  encodeStreamEnvelopeNoEnd,
} from './connect-mock'

// =============================================================================
// Service Fixtures
// =============================================================================

/**
 * Create a service fixture with customizable options
 */
export function createServiceFixture(options: {
  name: string
  status?: string
  health?: string
  port?: number
  language?: string
  framework?: string
  serviceType?: 'http' | 'tcp' | 'process'
  serviceMode?: 'watch' | 'build' | 'daemon' | 'task'
  error?: string
  host?: 'local'
  azure?: { url: string; resourceName: string }
}): Service {
  const port = options.port ?? 3000 + Math.floor(Math.random() * 5000)
  const isProcess = options.serviceType === 'process'

  return {
    name: options.name,
    host: options.host,
    language: options.language ?? 'TypeScript',
    framework: options.framework ?? 'Express',
    project: `./src/${options.name}`,
    local: {
      status: options.status ?? 'running',
      health: options.health ?? 'healthy',
      port: isProcess ? undefined : port,
      url: isProcess ? undefined : `http://localhost:${port}`,
      pid: 10000 + Math.floor(Math.random() * 50000),
      startTime: new Date().toISOString(),
      serviceType: options.serviceType ?? 'http',
      serviceMode: options.serviceMode,
    },
    azure: options.azure,
    error: options.error,
  } as Service
}

/**
 * Pre-built service fixtures for common scenarios
 */
export const mockServices = {
  // HTTP Services
  healthyApi: createServiceFixture({
    name: 'api',
    status: 'running',
    health: 'healthy',
    port: 3001,
    language: 'TypeScript',
    framework: 'Express',
  }),
  healthyWeb: createServiceFixture({
    name: 'web',
    status: 'running',
    health: 'healthy',
    port: 3000,
    language: 'TypeScript',
    framework: 'React',
  }),
  degradedService: createServiceFixture({
    name: 'slow-api',
    status: 'running',
    health: 'degraded',
    port: 3002,
  }),
  unhealthyService: createServiceFixture({
    name: 'failing-api',
    status: 'running',
    health: 'unhealthy',
    port: 3003,
    error: 'Connection refused',
  }),
  stoppedService: createServiceFixture({
    name: 'stopped-api',
    status: 'stopped',
    health: 'unknown',
    port: 3004,
  }),
  startingService: createServiceFixture({
    name: 'starting-api',
    status: 'starting',
    health: 'unknown',
    port: 3005,
  }),
  // TCP Services
  database: createServiceFixture({
    name: 'database',
    status: 'running',
    health: 'healthy',
    port: 5432,
    serviceType: 'tcp',
    language: 'SQL',
    framework: 'PostgreSQL',
  }),
  // Process Services
  watchService: createServiceFixture({
    name: 'typescript-watcher',
    status: 'watching',
    health: 'healthy',
    serviceType: 'process',
    serviceMode: 'watch',
    framework: 'TypeScript',
  }),
  buildService: createServiceFixture({
    name: 'compiler',
    status: 'building',
    health: 'unknown',
    serviceType: 'process',
    serviceMode: 'build',
  }),
  daemonService: createServiceFixture({
    name: 'mcp-server',
    status: 'running',
    health: 'healthy',
    serviceType: 'process',
    serviceMode: 'daemon',
  }),
  taskService: createServiceFixture({
    name: 'migration',
    status: 'completed',
    health: 'healthy',
    serviceType: 'process',
    serviceMode: 'task',
  }),
  failedBuild: createServiceFixture({
    name: 'failed-build',
    status: 'failed',
    health: 'unhealthy',
    serviceType: 'process',
    serviceMode: 'build',
    error: 'Compilation failed',
  }),
  // Azure Services
  azureService: createServiceFixture({
    name: 'prod-api',
    status: 'running',
    health: 'healthy',
    port: 8080,
    azure: {
      url: 'https://api.azurewebsites.net',
      resourceName: 'api-prod',
    },
  }),
}

// =============================================================================
// Health Check Fixtures
// =============================================================================

export function createHealthCheckFixture(
  serviceName: string,
  status: 'healthy' | 'degraded' | 'unhealthy' | 'unknown',
  options: {
    checkType?: 'http' | 'tcp' | 'process'
    responseTime?: number
    error?: string
    port?: number
  } = {}
): HealthCheckResult {
  return {
    serviceName,
    status,
    checkType: options.checkType ?? 'http',
    endpoint: options.checkType === 'process' ? undefined : `http://localhost:${options.port ?? 3000}/health`,
    responseTime: options.responseTime ?? (status === 'degraded' ? 3000_000_000 : 50_000_000),
    statusCode: status === 'unhealthy' ? undefined : 200,
    error: options.error,
    timestamp: new Date().toISOString(),
    port: options.port ?? 3000,
    pid: 12345,
    uptime: 3600_000_000_000,
  } as HealthCheckResult
}

// =============================================================================
// Scenario Builders
// =============================================================================

export interface TestScenario {
  services: ReadonlyArray<Service>
  healthChecks: ReadonlyArray<HealthCheckResult>
  healthSummary: HealthSummary
}

export const scenarios = {
  /** Standard mixed services scenario */
  standard: (): TestScenario => ({
    services: [mockServices.healthyApi, mockServices.healthyWeb],
    healthChecks: [
      createHealthCheckFixture('api', 'healthy', { port: 3001 }),
      createHealthCheckFixture('web', 'healthy', { port: 3000 }),
    ],
    healthSummary: {
      total: 2, healthy: 2, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy'
    },
  }),
  
  /** All services healthy */
  allHealthy: (): TestScenario => ({
    services: [
      mockServices.healthyApi,
      mockServices.healthyWeb,
      mockServices.database,
    ],
    healthChecks: [
      createHealthCheckFixture('api', 'healthy'),
      createHealthCheckFixture('web', 'healthy'),
      createHealthCheckFixture('database', 'healthy', { checkType: 'tcp', port: 5432 }),
    ],
    healthSummary: {
      total: 3, healthy: 3, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy'
    },
  }),
  
  /** Mixed health states */
  mixedHealth: (): TestScenario => ({
    services: [
      mockServices.healthyApi,
      mockServices.degradedService,
      mockServices.unhealthyService,
      mockServices.stoppedService,
    ],
    healthChecks: [
      createHealthCheckFixture('api', 'healthy'),
      createHealthCheckFixture('slow-api', 'degraded'),
      createHealthCheckFixture('failing-api', 'unhealthy', { error: 'Connection refused' }),
    ],
    healthSummary: {
      total: 4, healthy: 1, degraded: 1, unhealthy: 1, starting: 0, stopped: 1, unknown: 0, overall: 'unhealthy'
    },
  }),
  
  /** All process services */
  processServices: (): TestScenario => ({
    services: [
      mockServices.watchService,
      mockServices.buildService,
      mockServices.daemonService,
      mockServices.taskService,
      mockServices.failedBuild,
    ],
    healthChecks: [
      createHealthCheckFixture('typescript-watcher', 'healthy', { checkType: 'process' }),
      createHealthCheckFixture('compiler', 'unknown', { checkType: 'process' }),
      createHealthCheckFixture('mcp-server', 'healthy', { checkType: 'process' }),
      createHealthCheckFixture('migration', 'healthy', { checkType: 'process' }),
      createHealthCheckFixture('failed-build', 'unhealthy', { checkType: 'process', error: 'Build failed' }),
    ],
    healthSummary: {
      total: 5, healthy: 3, degraded: 0, unhealthy: 1, starting: 0, stopped: 0, unknown: 1, overall: 'unhealthy'
    },
  }),
  
  /** All services unhealthy/error */
  allErrors: (): TestScenario => ({
    services: [
      mockServices.unhealthyService,
      mockServices.failedBuild,
    ],
    healthChecks: [
      createHealthCheckFixture('failing-api', 'unhealthy', { error: 'Connection refused' }),
      createHealthCheckFixture('failed-build', 'unhealthy', { checkType: 'process', error: 'Build failed' }),
    ],
    healthSummary: {
      total: 2, healthy: 0, degraded: 0, unhealthy: 2, starting: 0, stopped: 0, unknown: 0, overall: 'unhealthy'
    },
  }),
  
  /** No services (empty state) */
  empty: (): TestScenario => ({
    services: [],
    healthChecks: [],
    healthSummary: {
      total: 0, healthy: 0, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy'
    },
  }),
  
  /** Azure deployment scenario */
  azureDeployment: (): TestScenario => ({
    services: [mockServices.azureService, mockServices.healthyWeb],
    healthChecks: [
      createHealthCheckFixture('prod-api', 'healthy', { port: 8080 }),
      createHealthCheckFixture('web', 'healthy'),
    ],
    healthSummary: {
      total: 2, healthy: 2, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy'
    },
  }),
  
  /** Starting services scenario */
  starting: (): TestScenario => ({
    services: [
      mockServices.startingService,
      mockServices.healthyApi,
    ],
    healthChecks: [
      createHealthCheckFixture('api', 'healthy'),
    ],
    healthSummary: {
      total: 2, healthy: 1, degraded: 0, unhealthy: 0, starting: 1, stopped: 0, unknown: 0, overall: 'healthy'
    },
  }),
  
  /** Many services (stress test) */
  manyServices: (): TestScenario => {
    const services: Service[] = []
    const healthChecks: HealthCheckResult[] = []
    for (let i = 0; i < 20; i++) {
      services.push(createServiceFixture({
        name: `service-${i}`,
        status: 'running',
        health: 'healthy',
        port: 4000 + i,
      }))
      healthChecks.push(createHealthCheckFixture(`service-${i}`, 'healthy', { port: 4000 + i }))
    }
    return {
      services,
      healthChecks,
      healthSummary: {
        total: 20, healthy: 20, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'healthy'
      },
    }
  },

  /** Unhealthy services scenario */
  unhealthyServices: (): TestScenario => ({
    services: [
      createServiceFixture({
        name: 'api',
        status: 'running',
        health: 'unhealthy',
        port: 3001,
        error: 'HTTP 503: Service Unavailable',
      }),
    ],
    healthChecks: [
      {
        serviceName: 'api',
        status: 'unhealthy',
        checkType: 'http',
        endpoint: 'http://localhost:3001/health',
        responseTime: 45_000_000,
        statusCode: 503,
        error: 'HTTP 503: Service Unavailable',
        errorDetails: 'Database connection pool exhausted',
        consecutiveFailures: 3,
        timestamp: new Date().toISOString(),
        port: 3001,
        pid: 12345,
        uptime: 947_000_000_000,
        details: {
          suggestion: 'Service temporarily unavailable. Check if dependencies are running.',
        },
      } as HealthCheckResult,
    ],
    healthSummary: {
      total: 1, healthy: 0, degraded: 0, unhealthy: 1, starting: 0, stopped: 0, unknown: 0, overall: 'unhealthy'
    },
  }),

  /** Degraded services scenario */
  degradedServices: (): TestScenario => ({
    services: [
      createServiceFixture({
        name: 'api',
        status: 'running',
        health: 'degraded',
        port: 3001,
      }),
    ],
    healthChecks: [
      {
        serviceName: 'api',
        status: 'degraded',
        checkType: 'http',
        endpoint: 'http://localhost:3001/health',
        responseTime: 1_234_000_000,
        statusCode: 200,
        timestamp: new Date().toISOString(),
        port: 3001,
        pid: 12345,
        uptime: 947_000_000_000,
        details: {
          warning: 'Response time exceeds threshold (>1000ms)',
        },
      } as HealthCheckResult,
    ],
    healthSummary: {
      total: 1, healthy: 0, degraded: 1, unhealthy: 0, starting: 0, stopped: 0, unknown: 0, overall: 'degraded'
    },
  }),

  /** Unknown health services scenario */
  unknownHealthServices: (): TestScenario => ({
    services: [
      createServiceFixture({
        name: 'api',
        status: 'running',
        health: 'unknown',
        port: 3001,
      }),
    ],
    healthChecks: [
      {
        serviceName: 'api',
        status: 'unknown',
        checkType: 'http',
        timestamp: new Date().toISOString(),
        port: 3001,
        pid: 12345,
        uptime: 135_000_000_000,
      } as HealthCheckResult,
    ],
    healthSummary: {
      total: 1, healthy: 0, degraded: 0, unhealthy: 0, starting: 0, stopped: 0, unknown: 1, overall: 'unknown'
    },
  }),
}

// =============================================================================
// Mock Setup Functions
// =============================================================================

/**
 * Mock EventSource for SSE health stream
 */
export async function mockEventSource(page: Page, scenario?: TestScenario) {
  const healthData = scenario ?? scenarios.standard()
  
  await page.addInitScript((data) => {
    class MockEventSource {
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      static readonly CLOSED = 2
      readonly CONNECTING = 0
      readonly OPEN = 1
      readonly CLOSED = 2
      readyState = 1
      url: string
      withCredentials = false
      onopen: ((ev: Event) => void) | null = null
      onmessage: ((ev: MessageEvent) => void) | null = null
      onerror: ((ev: Event) => void) | null = null
      private listeners: Record<string, ((ev: Event) => void)[]> = {}

      constructor(url: string) {
        this.url = url
        setTimeout(() => {
          if (this.onopen) this.onopen(new Event('open'))
          this.sendHealthEvent()
        }, 10)
        
        // Send periodic health updates
        setInterval(() => this.sendHealthEvent(), 5000)
      }
      
      private sendHealthEvent() {
        const event = {
          type: 'health',
          timestamp: new Date().toISOString(),
          services: data.healthChecks,
          summary: data.healthSummary,
        }
        
        if (this.onmessage) {
          this.onmessage(new MessageEvent('message', { data: JSON.stringify(event) }))
        }
        
        // Also dispatch to 'health' listeners
        this.listeners['health']?.forEach(listener => {
          listener(new MessageEvent('health', { data: JSON.stringify(event) }))
        })
      }

      close() { this.readyState = 2 }
      
      addEventListener(type: string, listener: (ev: Event) => void) {
        this.listeners[type] = this.listeners[type] || []
        this.listeners[type].push(listener)
      }
      
      removeEventListener(type: string, listener: (ev: Event) => void) {
        if (this.listeners[type]) {
          this.listeners[type] = this.listeners[type].filter(l => l !== listener)
        }
      }
      
      dispatchEvent() { return false }
    }

    ;(window as unknown as { EventSource: typeof MockEventSource }).EventSource = MockEventSource
  }, healthData)
}

/**
 * Mock all API routes needed for dashboard
 */
export async function mockApiRoutes(page: Page, options: {
  scenario?: TestScenario
  projectName?: string
  logs?: Array<{ service: string; message: string; level: number; timestamp: string; isStderr: boolean }>
  azure?: {
    enabled?: boolean
    status?: 'connected' | 'disconnected' | 'connecting' | 'disabled'
    mode?: 'local' | 'azure'
    connectionMessage?: string
  }
  azureLogs?: Array<{ service: string; message: string; level: number; timestamp: string; isStderr: boolean }>
} = {}) {
  const { projectName = 'test-project', logs = [] } = options
  const scenario = options.scenario ?? scenarios.standard()
  const azureEnabled = options.azure?.enabled ?? false
  const azureStatus = options.azure?.status ?? (azureEnabled ? 'connected' : 'disabled')
  const connectionMessage = options.azure?.connectionMessage
  let currentMode: 'local' | 'azure' = options.azure?.mode ?? 'local'
  const azureLogs = options.azureLogs ?? []

  // Project info
  await page.route('/api/project', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ name: projectName }),
    })
  })

  // Services list
  await page.route('/api/services', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(scenario.services),
    })
  })

  // Service operations (start/stop/restart)
  await page.route('/api/services/*/start', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"success": true}' })
  })
  await page.route('/api/services/*/stop', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"success": true}' })
  })
  await page.route('/api/services/*/restart', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"success": true}' })
  })
  await page.route('/api/services/start-all', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"started": 2, "failed": 0}' })
  })
  await page.route('/api/services/stop-all', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"stopped": 2, "failed": 0}' })
  })
  await page.route('/api/services/restart-all', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{"restarted": 2, "failed": 0}' })
  })

  // Shared preferences state (persisted across page reloads within same test)
  let currentPreferences = {
    version: '1.0',
    theme: 'light' as 'light' | 'dark',
    ui: { gridColumns: 2, viewMode: 'grid' as 'grid' | 'unified', gridAutoFit: true, selectedServices: [] as string[] },
    behavior: { autoScroll: true, pauseOnScroll: true, timestampFormat: 'hh:mm:ss.sss' },
    copy: { defaultFormat: 'plaintext', includeTimestamp: true, includeService: true },
  }

  // Logs preferences (must be before /api/logs*)
  await page.route('/api/logs/preferences*', async route => {
    if (route.request().method() === 'POST') {
      // Save the updated preferences
      try {
        const body = route.request().postDataJSON() as Partial<typeof currentPreferences> | null
        if (body) {
          currentPreferences = { ...currentPreferences, ...body }
        }
      } catch {
        // Ignore parse errors
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(currentPreferences) })
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(currentPreferences),
      })
    }
  })

  // Mode (local/azure) and Azure availability status
  await page.route('/api/mode', async route => {
    const method = route.request().method()

    if (method === 'PUT') {
      try {
        const body = route.request().postDataJSON() as { mode?: 'local' | 'azure' }
        if (body?.mode === 'local' || body?.mode === 'azure') {
          currentMode = body.mode
        }
      } catch {
        // Ignore invalid JSON
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          mode: currentMode,
          azureEnabled,
          azureStatus,
          connectionMessage,
        }),
      })
      return
    }

    // GET
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        mode: currentMode,
        azureEnabled,
        azureStatus,
        connectionMessage,
      }),
    })
  })

  // Logs
  await page.route('/api/logs*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(logs),
    })
  })

  // Azure logs endpoints used by the Console/LogsPane and diagnostics
  await page.route('/api/azure/logs*', async route => {
    const url = new URL(route.request().url())
    const service = url.searchParams.get('service')
    const filtered = service ? azureLogs.filter(l => l.service === service) : azureLogs
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'ok', logs: filtered }),
    })
  })

  await page.route('/api/azure/logs/health*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status: 'healthy', checks: [] }),
    })
  })

  await page.route('/api/azure/tables*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tables: [] }),
    })
  })

  await page.route('/api/azure/query*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ query: '// mocked', resourceType: 'mock' }),
    })
  })

  // Note: Preferences are now handled by /api/logs/preferences above, not /api/preferences
  // This route is kept for backward compatibility but delegates to the same storage
  await page.route('/api/preferences*', async route => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(currentPreferences),
      })
    } else if (route.request().method() === 'POST' || route.request().method() === 'PUT') {
      try {
        const body = route.request().postDataJSON() as Partial<typeof currentPreferences> | null
        if (body) {
          currentPreferences = { ...currentPreferences, ...body }
        }
      } catch {
        // Ignore parse errors
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(currentPreferences),
      })
    } else {
      await route.continue()
    }
  })

  // Classifications
  await page.route('/api/classifications*', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  })

  // Environment
  await page.route('/api/environment*', async route => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
  })
}

// =============================================================================
// Connect-RPC route mocks
// =============================================================================

/**
 * Connect proto ServiceStatus enum values. Must stay in sync with
 * `src/gen/proto/azdapp/v1/common_pb.ts`; we duplicate them here instead
 * of importing the generated module to keep the e2e helpers independent
 * of the proto build (they're loaded by Playwright, not Vite).
 */
const PROTO_SERVICE_STATUS = {
  UNSPECIFIED: 0,
  STOPPED: 1,
  STARTING: 2,
  READY: 3,
  DEGRADED: 4,
  ERROR: 5,
  STOPPING: 6,
} as const

const PROTO_HEALTH_STATE = {
  UNSPECIFIED: 0,
  HEALTHY: 1,
  UNHEALTHY: 2,
  UNKNOWN: 3,
  STARTING: 4,
  DEGRADED: 5,
} as const

const PROTO_LOG_MODE = {
  UNSPECIFIED: 0,
  LOCAL: 1,
  AZURE: 2,
} as const

/**
 * Translate a Service fixture into the proto ServiceInfo JSON shape the
 * Connect transport produces on the wire. This is the forward inverse
 * of `protoServiceToService`: whatever fields get packed here must
 * survive a round trip through the translator and still render under
 * the dashboard's domain-string contract.
 *
 * Notable encoding choices:
 * - Enums travel as integers. connect-web's JSON codec accepts both
 *   integer and string forms; integers dodge the short-name-vs-prefixed
 *   ambiguity (READY vs SERVICE_STATUS_READY).
 * - Overrides for non-enum health ('degraded'), port, url, and azure
 *   sub-fields land in the metadata Struct because the translator reads
 *   them from metadata exclusively (see azureFromProto + autoUrl
 *   precedence).
 * - Timestamps use google.protobuf.Timestamp JSON ({seconds, nanos}).
 *   seconds is a string because protoInt64 decodes strings for int64.
 */
function fixtureStatusToProto(status: string | undefined): number {
  switch (status) {
    case 'ready':
    case 'running':
    case 'watching':
    case 'building':
    case 'built':
    case 'completed':
      return PROTO_SERVICE_STATUS.READY
    case 'starting':
      return PROTO_SERVICE_STATUS.STARTING
    case 'stopping':
      return PROTO_SERVICE_STATUS.STOPPING
    case 'stopped':
    case 'not-running':
    case 'not-started':
      return PROTO_SERVICE_STATUS.STOPPED
    case 'error':
    case 'failed':
      return PROTO_SERVICE_STATUS.ERROR
    default:
      return PROTO_SERVICE_STATUS.UNSPECIFIED
  }
}

function fixtureHealthToProto(health: string | undefined): { state: number; override?: string } {
  switch (health) {
    case 'healthy':
      return { state: PROTO_HEALTH_STATE.HEALTHY }
    case 'unhealthy':
      return { state: PROTO_HEALTH_STATE.UNHEALTHY }
    case 'starting':
      return { state: PROTO_HEALTH_STATE.STARTING }
    case 'degraded':
      // Translator recognises 'degraded' only when it arrives via
      // metadata.health override because the proto enum's semantic
      // 'DEGRADED' originally meant "service ok, health check slow" --
      // not the richer product surface the dashboard ships. Mirror the
      // Go side's packing.
      return { state: PROTO_HEALTH_STATE.UNSPECIFIED, override: 'degraded' }
    case 'unknown':
      return { state: PROTO_HEALTH_STATE.UNKNOWN }
    default:
      return { state: PROTO_HEALTH_STATE.UNSPECIFIED }
  }
}

function isoToTimestamp(iso: string | undefined): string | undefined {
  if (!iso) return undefined
  const ms = Date.parse(iso)
  if (Number.isNaN(ms)) return undefined
  // proto3 JSON encodes google.protobuf.Timestamp as an RFC 3339 string
  // with nanosecond precision and a trailing 'Z'. Date#toISOString only
  // emits milliseconds, which protobuf-es accepts; pad to 9 digits so the
  // shape matches what the Go server emits.
  return new Date(ms).toISOString()
}

function serviceFixtureToProto(service: Service): Record<string, unknown> {
  const local = service.local
  const azure = service.azure
  const healthPacked = fixtureHealthToProto(local?.health as string | undefined)

  // Metadata fields the translator reads. Keys match the shape
  // `structToObject` + `metaString` expect; nested azure is a nested
  // object, everything else is a flat string.
  const metadata: Record<string, unknown> = {}
  if (healthPacked.override) metadata.health = healthPacked.override
  if (local?.lastChecked) metadata.lastChecked = local.lastChecked
  if (local?.serviceType) metadata.serviceType = local.serviceType
  if (local?.serviceMode) metadata.serviceMode = local.serviceMode

  // Azure metadata (url / customUrl / customDomain / imageName) flows
  // through `azureMeta`, not the top-level metadata.
  const azureMeta: Record<string, unknown> = {}
  if (azure?.url) azureMeta.url = azure.url
  if (azure?.customUrl) azureMeta.customUrl = azure.customUrl
  if (azure?.customDomain) azureMeta.customDomain = azure.customDomain
  if (azure?.customDomainSource) azureMeta.customDomainSource = azure.customDomainSource
  if (azure?.imageName) azureMeta.imageName = azure.imageName
  if (Object.keys(azureMeta).length > 0) metadata.azure = azureMeta

  const proto: Record<string, unknown> = {
    name: service.name,
    status: fixtureStatusToProto(local?.status as string | undefined),
    health: healthPacked.state,
  }
  if (service.host) proto.kind = service.host
  if (service.language) proto.language = service.language
  if (service.framework) proto.framework = service.framework
  if (service.project) proto.projectDir = service.project
  if (local?.url) proto.url = local.url
  if (local?.port) proto.port = local.port
  if (local?.pid) proto.pid = local.pid
  const startTime = isoToTimestamp(local?.startTime)
  if (startTime) proto.startTime = startTime

  if (Object.keys(metadata).length > 0) {
    // google.protobuf.Struct JSON == the JSON object itself. connect-web
    // re-wraps it via fromJson, which matches how the translator reads
    // `info.metadata.toJson()`.
    proto.metadata = metadata
  }

  if (azure) {
    const azureProto: Record<string, unknown> = {}
    if (azure.resourceName) azureProto.resourceId = azure.resourceName
    if (azure.resourceType) azureProto.resourceType = azure.resourceType
    if (azure.resourceGroup) azureProto.resourceGroup = azure.resourceGroup
    if (azure.subscriptionId) azureProto.subscriptionId = azure.subscriptionId
    if (azure.location) azureProto.region = azure.location
    if (azure.logAnalyticsId) azureProto.workspaceId = azure.logAnalyticsId
    if (Object.keys(azureProto).length > 0) proto.azure = azureProto
  }

  return proto
}

export interface MockConnectOptions {
  scenario?: TestScenario
  projectName?: string
  projectDir?: string
  codespace?: {
    enabled: boolean
    name?: string
    domain?: string
    isVsCodeDesktop?: boolean
  }
  environmentName?: string
  azure?: {
    enabled?: boolean
    status?: 'connected' | 'disconnected' | 'connecting' | 'disabled'
    mode?: 'local' | 'azure'
    connectionMessage?: string
  }
}

/**
 * Register Connect-RPC mocks that cover every service call the dashboard
 * performs on page load plus the main interactive paths (mode toggle,
 * service start/stop/restart, preferences save). Call this alongside
 * `mockApiRoutes`; the two don't overlap (different URL paths) so they
 * compose cleanly.
 *
 * Stream mocks intentionally emit a single heartbeat or end-only
 * envelope rather than a live feed. The dashboard's reconnect logic
 * handles the "stream closed" case by scheduling backoff, which the
 * App.tsx overlay gate keeps silent, so tests stay interactive without
 * needing real streaming infrastructure.
 */
export async function mockConnectRoutes(page: Page, options: MockConnectOptions = {}) {
  const scenario = options.scenario ?? scenarios.standard()
  const projectName = options.projectName ?? 'test-project'
  const projectDir = options.projectDir ?? '/test'
  const azureEnabled = options.azure?.enabled ?? false
  const azureStatus = options.azure?.status ?? (azureEnabled ? 'connected' : 'disabled')
  const connectionMessage = options.azure?.connectionMessage ?? ''
  let currentMode: 'local' | 'azure' = options.azure?.mode ?? 'local'

  const codespace = {
    enabled: options.codespace?.enabled ?? false,
    name: options.codespace?.name ?? '',
    domain: options.codespace?.domain ?? '',
    isVsCodeDesktop: options.codespace?.isVsCodeDesktop ?? false,
  }

  // -- ProjectService ---------------------------------------------------
  await mockConnectUnary(page, 'ProjectService', 'GetProject', () => ({
    name: projectName,
    dir: projectDir,
  }))

  // -- ServicesService --------------------------------------------------
  await mockConnectUnary(page, 'ServicesService', 'GetServices', () => ({
    services: scenario.services.map(serviceFixtureToProto),
  }))
  const ack = () => ({ success: true })
  await mockConnectUnary(page, 'ServicesService', 'StartService', ack)
  await mockConnectUnary(page, 'ServicesService', 'StopService', ack)
  await mockConnectUnary(page, 'ServicesService', 'RestartService', ack)

  // -- LifecycleService -------------------------------------------------
  await mockConnectUnary(page, 'LifecycleService', 'Ping', () => ({
    status: 'ok',
    version: 'test',
    serverTime: isoToTimestamp(new Date().toISOString()),
  }))
  await mockConnectUnary(page, 'LifecycleService', 'GetEnvironment', () => ({
    codespace,
    environmentName: options.environmentName ?? '',
  }))
  // Broadcast stream: no events needed on load, so emit an end trailer
  // immediately and let useBroadcast (if wired) treat the stream as
  // empty.
  await mockConnectServerStream(page, 'LifecycleService', 'StreamBroadcast', () => [])

  // -- ModeService ------------------------------------------------------
  const modeSnapshot = () => ({
    mode: currentMode === 'azure' ? PROTO_LOG_MODE.AZURE : PROTO_LOG_MODE.LOCAL,
    azureEnabled,
    azureStatus,
    azureRealtime: false,
    connectionMessage,
  })
  await mockConnectUnary(page, 'ModeService', 'GetMode', modeSnapshot)
  await mockConnectUnary<{ mode?: number | string }, ReturnType<typeof modeSnapshot>>(
    page,
    'ModeService',
    'SetMode',
    (req) => {
      // proto3 JSON accepts either integer or string enum values.
      if (req.mode === PROTO_LOG_MODE.AZURE || req.mode === 'AZURE' || req.mode === 'LOG_MODE_AZURE') {
        currentMode = 'azure'
      } else if (req.mode === PROTO_LOG_MODE.LOCAL || req.mode === 'LOCAL' || req.mode === 'LOG_MODE_LOCAL') {
        currentMode = 'local'
      }
      return modeSnapshot()
    },
  )

  // -- HealthService ----------------------------------------------------
  // StreamHealthResponse wire shape:
  // { event: { report: { results, generatedAt } } }
  // The oneof field surfaces as a plain JSON key per proto3; connect-web
  // re-packs it into the `{case, value}` discriminant post-parse.
  const healthResults = scenario.healthChecks.map((hc) => ({
    serviceName: hc.serviceName,
    state: (() => {
      switch (hc.status) {
        case 'healthy': return PROTO_HEALTH_STATE.HEALTHY
        case 'unhealthy': return PROTO_HEALTH_STATE.UNHEALTHY
        case 'degraded': return PROTO_HEALTH_STATE.DEGRADED
        case 'unknown': return PROTO_HEALTH_STATE.UNKNOWN
        default: return PROTO_HEALTH_STATE.UNSPECIFIED
      }
    })(),
    message: hc.error ?? '',
    checkedAt: isoToTimestamp(hc.timestamp),
    latencyMs: String(Math.max(0, Math.floor((hc.responseTime ?? 0) / 1_000_000))),
  }))
  const streamReportFrame = {
    event: {
      report: {
        results: healthResults,
        generatedAt: isoToTimestamp(new Date().toISOString()),
      },
    },
  }
  await mockConnectUnary(page, 'HealthService', 'GetHealth', () => ({
    results: healthResults,
    generatedAt: isoToTimestamp(new Date().toISOString()),
  }))
  // StreamHealth is special: we CANNOT emit the end-stream envelope or
  // close the connection, because `useHealthStream` flips `connected` to
  // false on stream close and schedules a reconnect. React batches state
  // updates, so a setConnected(true) + setConnected(false) within the
  // same tick collapses to false — tests never observe the "connected"
  // state and downstream hooks gated on it (useLogsStream for Azure)
  // never fire their first fetch.
  //
  // Patch window.fetch before the page loads to serve StreamHealth as a
  // never-closing ReadableStream that emits exactly one data envelope.
  // Everything else falls through to Playwright's normal routing.
  const healthFrameBytes = Array.from(encodeStreamEnvelopeNoEnd(streamReportFrame))
  await page.addInitScript(
    ({ frameBytes, matchUrl }) => {
      const origFetch = window.fetch.bind(window)
      window.fetch = (input, init) => {
        const url = typeof input === 'string' ? input : (input as Request).url
        if (url.includes(matchUrl)) {
          const stream = new ReadableStream<Uint8Array>({
            start(controller) {
              controller.enqueue(new Uint8Array(frameBytes))
              // Intentionally no controller.close(): keeps the response
              // hanging so `connected` stays true for the test window.
            },
          })
          return Promise.resolve(
            new Response(stream, {
              status: 200,
              headers: { 'content-type': 'application/connect+json' },
            }),
          )
        }
        return origFetch(input, init)
      }
    },
    { frameBytes: healthFrameBytes, matchUrl: 'azdapp.v1.HealthService/StreamHealth' },
  )
  await mockConnectServerStream(page, 'HealthService', 'StreamStateTransitions', () => [])

  // -- LogsService ------------------------------------------------------
  let currentPreferences = {
    version: '1.0',
    theme: 'light',
    ui: { gridColumns: 2, gridAutoFit: true, viewMode: 'grid', selectedServices: [] as string[] },
    behavior: { autoScroll: true, pauseOnScroll: true, timestampFormat: 'hh:mm:ss.sss' },
    copy: { defaultFormat: 'plaintext', includeTimestamp: true, includeService: true },
  }
  await mockConnectUnary(page, 'LogsService', 'GetPreferences', () => ({
    preferences: currentPreferences,
  }))
  await mockConnectUnary<{ preferences?: typeof currentPreferences }, { preferences: typeof currentPreferences }>(
    page,
    'LogsService',
    'SavePreferences',
    (req) => {
      if (req.preferences) {
        currentPreferences = { ...currentPreferences, ...req.preferences }
      }
      return { preferences: currentPreferences }
    },
  )
  await mockConnectUnary(page, 'LogsService', 'ListClassifications', () => ({
    classifications: [],
  }))
  await mockConnectUnary(page, 'LogsService', 'AddClassification', (req: { classification?: unknown }) => ({
    classification: req.classification ?? { text: '', level: 0 },
  }))
  await mockConnectUnary(page, 'LogsService', 'DeleteClassification', () => ({}))
  await mockConnectUnary(page, 'LogsService', 'GetLogs', () => ({
    entries: [],
    tailClamped: false,
  }))
  await mockConnectServerStream(page, 'LogsService', 'StreamLocalLogs', () => [])

  // -- AzureService -----------------------------------------------------
  await mockConnectUnary(page, 'AzureService', 'GetAzureLogs', () => ({
    entries: [],
  }))
  await mockConnectUnary(page, 'AzureService', 'GetAzureLogsHealth', () => ({
    status: 'healthy',
    checks: [],
  }))
  await mockConnectUnary(page, 'AzureService', 'GetAzureServices', () => ({
    services: [],
  }))
  await mockConnectUnary(page, 'AzureService', 'EnableAzureLogging', () => ({
    success: true,
  }))
  await mockConnectServerStream(page, 'AzureService', 'StreamAzureLogs', () => [])

  // -- BicepService -----------------------------------------------------
  await mockConnectUnary(page, 'BicepService', 'GetBicepTemplate', () => ({
    template: '// mocked',
    includedServices: scenario.services.map((s) => s.name),
    workspaceId: '',
    generatedAt: isoToTimestamp(new Date().toISOString()),
  }))
}

/**
 * Mock WebSocket for real-time updates
 */
export async function mockWebSocket(page: Page) {
  await page.addInitScript(() => {
    class MockWebSocket {
      static readonly CONNECTING = 0
      static readonly OPEN = 1
      static readonly CLOSING = 2
      static readonly CLOSED = 3
      readonly CONNECTING = 0
      readonly OPEN = 1
      readonly CLOSING = 2
      readonly CLOSED = 3
      readyState = 1
      url: string
      protocol = ''
      extensions = ''
      bufferedAmount = 0
      binaryType: BinaryType = 'blob'
      onopen: ((ev: Event) => void) | null = null
      onmessage: ((ev: MessageEvent) => void) | null = null
      onerror: ((ev: Event) => void) | null = null
      onclose: ((ev: CloseEvent) => void) | null = null

      constructor(url: string) {
        this.url = url
        setTimeout(() => {
          if (this.onopen) this.onopen(new Event('open'))
        }, 10)
      }

      close() { this.readyState = 3 }
      send(_data: string | ArrayBufferLike | Blob | ArrayBufferView) {}
      addEventListener(_type: string, _listener: EventListenerOrEventListenerObject, _options?: boolean | AddEventListenerOptions) {}
      removeEventListener(_type: string, _listener: EventListenerOrEventListenerObject, _options?: boolean | EventListenerOptions) {}
      dispatchEvent(_event: Event) { return false }
    }

    ;(window as unknown as { WebSocket: typeof MockWebSocket }).WebSocket = MockWebSocket as unknown as typeof WebSocket
  })
}

/**
 * Complete setup for a test with all mocks
 */
export async function setupTest(page: Page, options: {
  scenario?: TestScenario
  projectName?: string
  clearStorage?: boolean
  azure?: {
    enabled?: boolean
    status?: 'connected' | 'disconnected' | 'connecting' | 'disabled'
    mode?: 'local' | 'azure'
    connectionMessage?: string
  }
  codespace?: {
    enabled: boolean
    name?: string
    domain?: string
    isVsCodeDesktop?: boolean
  }
  environmentName?: string
} = {}) {
  const { scenario, projectName = 'test-project', clearStorage = true, azure, codespace, environmentName } = options
  
  // Clear storage
  if (clearStorage) {
    await page.addInitScript(() => localStorage.clear())
  }
  
  // Setup mocks
  await mockEventSource(page, scenario)
  await mockWebSocket(page)
  // Mock the session token endpoint so the Connect interceptor doesn't block
  await page.route('/api/session-token', async route => {
    await route.fulfill({ status: 200, contentType: 'text/plain', body: 'test-token' })
  })
  await mockApiRoutes(page, { scenario, projectName, azure })
  await mockConnectRoutes(page, { scenario, projectName, azure, codespace, environmentName })
}

// =============================================================================
// Test Utilities
// =============================================================================

/**
 * Wait for dashboard to be fully loaded
 */
export async function waitForDashboardReady(page: Page) {
  // Wait for the main header/nav to be present with tabs
  // This indicates the React app has mounted and rendered
  // Use :visible to ensure we wait for a visible tablist (desktop or mobile)
  await page.locator('[role="tablist"]:visible').first().waitFor({
    state: 'visible',
    timeout: 15000,
  })
  // Also wait for the page to stabilize
  await page.waitForLoadState('domcontentloaded')
}

/**
 * Navigate to a specific view
 */
export async function navigateToView(page: Page, view: 'console' | 'resources' | 'environment' | 'metrics') {
  // Note: 'resources' view is labeled 'Services' in the UI
  const viewNames: Record<string, string> = {
    console: 'Console',
    resources: 'Services',
    environment: 'Environment',
    metrics: 'Metrics',
  }
  
  const tab = page.locator(`[role="tab"]:has-text("${viewNames[view]}")`).first()
  if (await tab.isVisible()) {
    await tab.click()
    await page.waitForTimeout(300) // Wait for view transition
  }
}

/**
 * Get service card by name
 */
export function getServiceCard(page: Page, serviceName: string) {
  return page.locator(`article:has-text("${serviceName}")`).first()
}

/**
 * Get service row in table view
 */
export function getServiceRow(page: Page, serviceName: string) {
  return page.locator(`tr:has-text("${serviceName}")`).first()
}
