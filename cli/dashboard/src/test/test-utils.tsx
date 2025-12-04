/* eslint-disable react-refresh/only-export-components */
import { render, type RenderOptions, type RenderResult } from '@testing-library/react'
import { ServiceOperationsProvider } from '@/contexts/ServiceOperationsContext'
import { ServicesProvider } from '@/contexts/ServicesContext'
import { PreferencesProvider } from '@/contexts/PreferencesContext'
import type { ReactNode, ReactElement, FC } from 'react'
import type { Service, HealthCheckResult, HealthSummary, HealthReportEvent } from '@/types'
import { vi, expect } from 'vitest'

// =============================================================================
// Mock Providers for Controlled Testing
// =============================================================================

interface MockServicesContextValue {
  services: Service[]
  serviceNames: string[]
  loading: boolean
  error: string | null
  connected: boolean
  refetch: () => Promise<void>
  getService: (name: string) => Service | undefined
}

interface MockServiceOperationsValue {
  startService: (name: string) => Promise<boolean>
  stopService: (name: string) => Promise<boolean>
  restartService: (name: string) => Promise<boolean>
  startAll: () => Promise<{ started: number; failed: number } | null>
  stopAll: () => Promise<{ stopped: number; failed: number } | null>
  restartAll: () => Promise<{ restarted: number; failed: number } | null>
  getOperationState: (name: string) => 'idle' | 'starting' | 'stopping' | 'restarting'
  getEffectiveOperationState: (name: string) => 'idle' | 'starting' | 'stopping' | 'restarting'
  isServiceOperating: (name: string) => boolean
  isBulkOperationInProgress: () => boolean
  error: string | null
  clearError: () => void
}

interface MockHealthStreamValue {
  healthReport: HealthReportEvent | null
  changes: unknown[]
  connected: boolean
  error: string | null
  lastUpdate: Date | null
  summary: HealthSummary | null
  getServiceHealth: (serviceName: string) => HealthCheckResult | undefined
  hasRecovered: (serviceName: string) => boolean
  getLatestChange: (serviceName: string) => unknown
  clearChanges: () => void
  reconnect: () => void
}

// =============================================================================
// Provider Wrapper Options
// =============================================================================

export interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  /** Initial services data */
  services?: Service[]
  /** Health check results map */
  healthMap?: Map<string, HealthCheckResult>
  /** Health summary */
  healthSummary?: HealthSummary
  /** Health report event */
  healthReport?: HealthReportEvent | null
  /** Whether connected to backend */
  connected?: boolean
  /** Loading state */
  loading?: boolean
  /** Error message */
  error?: string | null
  /** Custom wrapper (will be nested inside providers) */
  wrapper?: FC<{ children: ReactNode }>
}

// =============================================================================
// Provider Wrappers
// =============================================================================

interface AllProvidersProps {
  children: ReactNode
}

/**
 * Full provider stack for integration tests
 */
function AllProviders({ children }: AllProvidersProps) {
  return (
    <ServicesProvider>
      <PreferencesProvider>
        <ServiceOperationsProvider>
          {children}
        </ServiceOperationsProvider>
      </PreferencesProvider>
    </ServicesProvider>
  )
}

/**
 * Minimal providers for component tests (no network calls)
 */
function MinimalProviders({ children }: AllProvidersProps) {
  return (
    <PreferencesProvider>
      {children}
    </PreferencesProvider>
  )
}

// =============================================================================
// Custom Render Functions
// =============================================================================

/**
 * Render with full provider stack - use for integration tests
 */
function customRender(
  ui: ReactElement,
  options?: CustomRenderOptions
): RenderResult {
  const { wrapper: CustomWrapper, ...renderOptions } = options ?? {}
  
  const Wrapper: FC<{ children: ReactNode }> = ({ children }) => {
    const wrapped = <AllProviders>{children}</AllProviders>
    return CustomWrapper ? <CustomWrapper>{wrapped}</CustomWrapper> : wrapped
  }
  
  return render(ui, { wrapper: Wrapper, ...renderOptions })
}

/**
 * Render with minimal providers - use for isolated component tests
 */
function renderWithMinimalProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
): RenderResult {
  return render(ui, { wrapper: MinimalProviders, ...options })
}

// =============================================================================
// Mock Factory Functions
// =============================================================================

/**
 * Create a mock services context value
 */
export function createMockServicesContext(
  services: Service[] = [],
  options: Partial<MockServicesContextValue> = {}
): MockServicesContextValue {
  return {
    services,
    serviceNames: services.map(s => s.name),
    loading: false,
    error: null,
    connected: true,
    refetch: vi.fn().mockResolvedValue(undefined),
    getService: (name: string) => services.find(s => s.name === name),
    ...options,
  }
}

/**
 * Create a mock service operations context value
 */
export function createMockServiceOperations(
  options: Partial<MockServiceOperationsValue> = {}
): MockServiceOperationsValue {
  return {
    startService: vi.fn().mockResolvedValue(true),
    stopService: vi.fn().mockResolvedValue(true),
    restartService: vi.fn().mockResolvedValue(true),
    startAll: vi.fn().mockResolvedValue({ started: 0, failed: 0 }),
    stopAll: vi.fn().mockResolvedValue({ stopped: 0, failed: 0 }),
    restartAll: vi.fn().mockResolvedValue({ restarted: 0, failed: 0 }),
    getOperationState: vi.fn().mockReturnValue('idle'),
    getEffectiveOperationState: vi.fn().mockReturnValue('idle'),
    isServiceOperating: vi.fn().mockReturnValue(false),
    isBulkOperationInProgress: vi.fn().mockReturnValue(false),
    error: null,
    clearError: vi.fn(),
    ...options,
  }
}

/**
 * Create a mock health stream context value
 */
export function createMockHealthStream(
  services: Service[] = [],
  healthChecks: HealthCheckResult[] = [],
  options: Partial<MockHealthStreamValue> = {}
): MockHealthStreamValue {
  const healthMap = new Map(healthChecks.map(h => [h.serviceName, h]))
  
  return {
    healthReport: healthChecks.length > 0 ? {
      type: 'health',
      timestamp: new Date().toISOString(),
      services: healthChecks,
      summary: {
        total: services.length,
        healthy: healthChecks.filter(h => h.status === 'healthy').length,
        degraded: healthChecks.filter(h => h.status === 'degraded').length,
        unhealthy: healthChecks.filter(h => h.status === 'unhealthy').length,
        starting: 0,
        stopped: 0,
        unknown: healthChecks.filter(h => h.status === 'unknown').length,
        overall: 'healthy',
      },
    } : null,
    changes: [],
    connected: true,
    error: null,
    lastUpdate: new Date(),
    summary: null,
    getServiceHealth: (name: string) => healthMap.get(name),
    hasRecovered: vi.fn().mockReturnValue(false),
    getLatestChange: vi.fn().mockReturnValue(undefined),
    clearChanges: vi.fn(),
    reconnect: vi.fn(),
    ...options,
  }
}

// =============================================================================
// Mock API/WebSocket Utilities
// =============================================================================

/**
 * Create a mock fetch function for testing
 */
export function createMockFetch(responses: Record<string, unknown> = {}): typeof fetch {
  return vi.fn().mockImplementation((url: string) => {
    const path = url.startsWith('/') ? url : new URL(url).pathname
    const response = responses[path]
    
    if (response === undefined) {
      return Promise.resolve({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Not found' }),
      } as Response)
    }
    
    if (response instanceof Error) {
      return Promise.reject(response)
    }
    
    return Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve(response),
    } as Response)
  }) as typeof fetch
}

/**
 * Create a mock WebSocket class for testing
 */
export function createMockWebSocket() {
  const listeners: Record<string, ((event: Event) => void)[]> = {}
  
  const mockWs = {
    readyState: WebSocket.OPEN,
    url: '',
    onopen: null as ((event: Event) => void) | null,
    onmessage: null as ((event: MessageEvent) => void) | null,
    onerror: null as ((event: Event) => void) | null,
    onclose: null as ((event: CloseEvent) => void) | null,
    close: vi.fn(),
    send: vi.fn(),
    addEventListener: (type: string, listener: (event: Event) => void) => {
      listeners[type] = listeners[type] || []
      listeners[type].push(listener)
    },
    removeEventListener: (type: string, listener: (event: Event) => void) => {
      if (listeners[type]) {
        listeners[type] = listeners[type].filter(l => l !== listener)
      }
    },
    // Test utilities
    _emit: (type: string, data?: unknown) => {
      const event = type === 'message' 
        ? new MessageEvent('message', { data: JSON.stringify(data) })
        : new Event(type)
      
      if (type === 'open' && mockWs.onopen) mockWs.onopen(event)
      if (type === 'message' && mockWs.onmessage) mockWs.onmessage(event as MessageEvent)
      if (type === 'error' && mockWs.onerror) mockWs.onerror(event)
      if (type === 'close' && mockWs.onclose) mockWs.onclose(event as CloseEvent)
      
      listeners[type]?.forEach(l => l(event))
    },
    _simulateOpen: () => mockWs._emit('open'),
    _simulateMessage: (data: unknown) => mockWs._emit('message', data),
    _simulateError: () => mockWs._emit('error'),
    _simulateClose: () => mockWs._emit('close'),
  }
  
  return mockWs
}

/**
 * Create a mock EventSource class for SSE testing
 */
export function createMockEventSource() {
  const listeners: Record<string, ((event: Event) => void)[]> = {}
  
  const mockEs = {
    readyState: EventSource.OPEN,
    url: '',
    onopen: null as ((event: Event) => void) | null,
    onmessage: null as ((event: MessageEvent) => void) | null,
    onerror: null as ((event: Event) => void) | null,
    close: vi.fn(),
    addEventListener: (type: string, listener: (event: Event) => void) => {
      listeners[type] = listeners[type] || []
      listeners[type].push(listener)
    },
    removeEventListener: (type: string, listener: (event: Event) => void) => {
      if (listeners[type]) {
        listeners[type] = listeners[type].filter(l => l !== listener)
      }
    },
    // Test utilities
    _emit: (type: string, data?: unknown) => {
      const event = type === 'error' || type === 'open'
        ? new Event(type)
        : new MessageEvent(type, { data: JSON.stringify(data) })
      
      if (type === 'open' && mockEs.onopen) mockEs.onopen(event)
      if (type === 'message' && mockEs.onmessage) mockEs.onmessage(event as MessageEvent)
      if (type === 'error' && mockEs.onerror) mockEs.onerror(event)
      
      listeners[type]?.forEach(l => l(event))
    },
    _simulateOpen: () => mockEs._emit('open'),
    _simulateMessage: (data: unknown) => mockEs._emit('message', data),
    _simulateHealthEvent: (data: unknown) => {
      listeners['health']?.forEach(l => 
        l(new MessageEvent('health', { data: JSON.stringify(data) }))
      )
    },
    _simulateError: () => mockEs._emit('error'),
  }
  
  return mockEs
}

// =============================================================================
// Assertion Helpers
// =============================================================================

/**
 * Wait for a condition to be true
 */
export async function waitForCondition(
  condition: () => boolean,
  timeout = 5000,
  interval = 50
): Promise<void> {
  const start = Date.now()
  while (!condition()) {
    if (Date.now() - start > timeout) {
      throw new Error('Timeout waiting for condition')
    }
    await new Promise(resolve => setTimeout(resolve, interval))
  }
}

/**
 * Assert element has specific styles
 */
export function expectStyles(
  element: HTMLElement,
  styles: Record<string, string>
): void {
  const computed = window.getComputedStyle(element)
  for (const [prop, value] of Object.entries(styles)) {
    expect(computed.getPropertyValue(prop)).toBe(value)
  }
}

// =============================================================================
// Re-exports
// =============================================================================

// Re-export everything from @testing-library/react
export * from '@testing-library/react'

// Override render with our custom render
export { customRender as render, renderWithMinimalProviders, MinimalProviders, AllProviders }
