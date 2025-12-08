import type { Page } from '@playwright/test'
import type { Service, HealthCheckResult, HealthSummary } from '../../src/types'

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
  azure?: { url: string; resourceName: string }
}): Service {
  const port = options.port ?? 3000 + Math.floor(Math.random() * 5000)
  const isProcess = options.serviceType === 'process'

  return {
    name: options.name,
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
  services: Service[]
  healthChecks: HealthCheckResult[]
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
} = {}) {
  const { projectName = 'test-project', logs = [] } = options
  const scenario = options.scenario ?? scenarios.standard()

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

  // Logs preferences (must be before /api/logs*)
  await page.route('/api/logs/preferences*', async route => {
    if (route.request().method() === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          version: '1.0',
          theme: 'light',
          ui: { gridColumns: 2, viewMode: 'grid', gridAutoFit: true, selectedServices: [] },
          behavior: { autoScroll: true, pauseOnScroll: true, timestampFormat: 'hh:mm:ss.sss' },
          copy: { defaultFormat: 'plaintext', includeTimestamp: true, includeService: true },
        }),
      })
    }
  })

  // Logs
  await page.route('/api/logs*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(logs),
    })
  })

  // Preferences
  await page.route('/api/preferences*', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        version: '1.0',
        theme: 'light',
        ui: { gridColumns: 2, viewMode: 'grid', gridAutoFit: true, selectedServices: [] },
        behavior: { autoScroll: true, pauseOnScroll: true, timestampFormat: 'hh:mm:ss.sss' },
        copy: { defaultFormat: 'plaintext', includeTimestamp: true, includeService: true },
      }),
    })
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
} = {}) {
  const { scenario, projectName = 'test-project', clearStorage = true } = options
  
  // Clear storage
  if (clearStorage) {
    await page.addInitScript(() => localStorage.clear())
  }
  
  // Setup mocks
  await mockEventSource(page, scenario)
  await mockWebSocket(page)
  await mockApiRoutes(page, { scenario, projectName })
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
