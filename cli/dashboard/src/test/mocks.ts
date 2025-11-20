import type { Service } from '@/types'

export const createMockService = (overrides?: Partial<Service>): Service => ({
  name: 'api',
  language: 'python',
  framework: 'flask',
  project: '/Users/dev/projects/fullstack',
  local: {
    status: 'ready',
    health: 'healthy',
    pid: 12345,
    port: 5000,
    url: 'http://localhost:5000',
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString(),
  },
  ...overrides,
})

export const mockServices: Service[] = [
  createMockService({
    name: 'api',
    language: 'python',
    framework: 'flask',
    project: '/Users/dev/projects/fullstack',
    local: {
      status: 'ready',
      health: 'healthy',
      pid: 12345,
      port: 5000,
      url: 'http://localhost:5000',
      startTime: new Date('2024-01-01T10:00:00Z').toISOString(),
      lastChecked: new Date().toISOString(),
    },
  }),
  createMockService({
    name: 'web',
    language: 'node',
    framework: 'express',
    project: '/Users/dev/projects/fullstack',
    local: {
      status: 'ready',
      health: 'healthy',
      pid: 12346,
      port: 5001,
      url: 'http://localhost:5001',
      startTime: new Date('2024-01-01T10:00:00Z').toISOString(),
      lastChecked: new Date().toISOString(),
    },
  }),
  createMockService({
    name: 'database',
    language: 'go',
    framework: 'postgres',
    project: '/Users/dev/projects/fullstack',
    local: {
      status: 'starting',
      health: 'unknown',
      pid: 12347,
      port: 5432,
      startTime: new Date('2024-01-01T10:05:00Z').toISOString(),
      lastChecked: new Date().toISOString(),
    },
  }),
]

export const mockServiceWithError = createMockService({
  name: 'error-service',
  local: {
    status: 'error',
    health: 'unhealthy',
    pid: 99999,
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString(),
  },
  error: 'Failed to start: Port already in use',
})

export const mockServiceStopped = createMockService({
  name: 'stopped-service',
  local: {
    status: 'stopped',
    health: 'unknown',
  },
})

export const mockServiceWithAzure = createMockService({
  name: 'azure-service',
  local: {
    status: 'ready',
    health: 'healthy',
    pid: 55555,
    port: 8080,
    url: 'http://localhost:8080',
    startTime: new Date().toISOString(),
    lastChecked: new Date().toISOString(),
  },
  azure: {
    url: 'https://my-app.azurewebsites.net',
    resourceName: 'my-app',
    imageName: 'myregistry.azurecr.io/my-app:latest',
  },
})

export interface LogEntry {
  service: string
  message: string
  level: number
  timestamp: string
  isStderr: boolean
}

export const createMockLogEntry = (overrides?: Partial<LogEntry>): LogEntry => ({
  service: 'api',
  message: 'Application started successfully',
  level: 0,
  timestamp: new Date().toISOString(),
  isStderr: false,
  ...overrides,
})

export const mockLogs: LogEntry[] = [
  createMockLogEntry({
    service: 'api',
    message: 'Starting Flask application',
    timestamp: new Date('2024-01-01T10:00:00Z').toISOString(),
  }),
  createMockLogEntry({
    service: 'api',
    message: 'Application started successfully',
    timestamp: new Date('2024-01-01T10:00:01Z').toISOString(),
  }),
  createMockLogEntry({
    service: 'web',
    message: 'Express server listening on port 5001',
    timestamp: new Date('2024-01-01T10:00:02Z').toISOString(),
  }),
  createMockLogEntry({
    service: 'api',
    message: 'Warning: Cache miss for key "user:123"',
    level: 2,
    timestamp: new Date('2024-01-01T10:00:03Z').toISOString(),
  }),
  createMockLogEntry({
    service: 'database',
    message: 'Error: Connection timeout',
    level: 3,
    isStderr: true,
    timestamp: new Date('2024-01-01T10:00:04Z').toISOString(),
  }),
]

export const mockLogsWithAnsi: LogEntry[] = [
  createMockLogEntry({
    service: 'api',
    message: '\x1b[32mStarting Flask application\x1b[0m',
    timestamp: new Date('2024-01-01T10:00:00Z').toISOString(),
  }),
  createMockLogEntry({
    service: 'api',
    message: '\x1b[31mError: Something went wrong\x1b[0m',
    level: 3,
    isStderr: true,
    timestamp: new Date('2024-01-01T10:00:01Z').toISOString(),
  }),
]

export const mockProjectInfo = {
  name: 'My Test Project',
}

// Helper to create mock fetch responses
export const createMockFetchResponse = <T>(data: T, ok = true) => {
  return Promise.resolve({
    ok,
    json: () => Promise.resolve(data),
  } as Response)
}

// Helper to create mock WebSocket message
export const createMockWebSocketMessage = (data: unknown) => {
  return {
    data: JSON.stringify(data),
  } as MessageEvent
}
