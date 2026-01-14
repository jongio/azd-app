/**
 * Health Check Types
 * Type definitions for health check configuration
 */

export type HealthCheckType = 'http' | 'tcp' | 'process' | 'output' | 'none'

export interface HealthCheckFormData {
  /** Type of health check */
  type: HealthCheckType
  
  /** HTTP URL (for http type) */
  url?: string
  
  /** TCP port (for tcp type) */
  port?: number
  
  /** Process/command (for process type) */
  command?: string
  
  /** Expected output pattern (for output type) */
  pattern?: string
  
  /** Time between health checks (e.g., "30s", "1m") */
  interval?: string
  
  /** Maximum time for health check to complete (e.g., "5s", "10s") */
  timeout?: string
  
  /** Number of consecutive failures before marking unhealthy */
  retries?: number
  
  /** Grace period for container initialization (e.g., "0s", "40s") */
  startPeriod?: string
  
  /** Time between health checks during start period (e.g., "5s") */
  startInterval?: string
}

export interface HealthCheckConfig {
  /** Health check test command or URL */
  test?: string | string[]
  
  /** Type of health check */
  type?: HealthCheckType
  
  /** HTTP path (when type=http) */
  path?: string
  
  /** Regex pattern (when type=output) */
  pattern?: string
  
  /** Check interval (e.g., "30s") */
  interval?: string
  
  /** Timeout duration (e.g., "5s") */
  timeout?: string
  
  /** Number of retries before unhealthy */
  retries?: number
  
  /** Start period before checks begin (e.g., "10s") */
  start_period?: string
  
  /** Start interval during start period */
  start_interval?: string
  
  /** Disable health check */
  disable?: boolean
}

/**
 * Convert form data to health check config for azure.yaml
 */
export function formDataToHealthCheck(data: HealthCheckFormData): HealthCheckConfig | null {
  if (data.type === 'none') {
    return { disable: true }
  }

  const config: HealthCheckConfig = {
    type: data.type,
  }

  // Add type-specific fields
  if (data.type === 'http' && data.url) {
    config.test = data.url
    config.path = new URL(data.url).pathname
  } else if (data.type === 'tcp' && data.port) {
    config.test = `tcp://localhost:${data.port}`
  } else if (data.type === 'process' && data.command) {
    config.test = data.command
  } else if (data.type === 'output' && data.pattern) {
    config.pattern = data.pattern
  }

  // Add optional timing fields
  if (data.interval) config.interval = data.interval
  if (data.timeout) config.timeout = data.timeout
  if (data.retries) config.retries = data.retries
  if (data.startPeriod) config.start_period = data.startPeriod
  if (data.startInterval) config.start_interval = data.startInterval

  return config
}

/**
 * Convert health check config to form data
 */
export function healthCheckToFormData(config: HealthCheckConfig | undefined): HealthCheckFormData {
  if (!config || config.disable) {
    return {
      type: 'none',
      interval: '30s',
      timeout: '5s',
      retries: 3,
      startPeriod: '0s',
      startInterval: '5s',
    }
  }

  const formData: HealthCheckFormData = {
    type: config.type || 'http',
    interval: config.interval || '30s',
    timeout: config.timeout || '5s',
    retries: config.retries || 3,
    startPeriod: config.start_period || '0s',
    startInterval: config.start_interval || '5s',
  }

  // Extract type-specific fields
  if (config.type === 'http' && typeof config.test === 'string' && config.test.startsWith('http')) {
    formData.url = config.test
  } else if (config.type === 'tcp' && typeof config.test === 'string' && config.test.startsWith('tcp://')) {
    const port = config.test.replace('tcp://localhost:', '')
    formData.port = parseInt(port, 10)
  } else if (config.type === 'process' && config.test) {
    formData.command = typeof config.test === 'string' ? config.test : config.test.join(' ')
  } else if (config.type === 'output' && config.pattern) {
    formData.pattern = config.pattern
  }

  return formData
}

/**
 * Get default health check suggestions based on service type
 */
export interface ServiceInfo {
  host?: string
  ports?: string[]
  image?: string
  language?: string
}

export function getDefaultHealthCheck(service: ServiceInfo): HealthCheckFormData {
  const firstPort = service.ports?.[0]?.split(':')[0]
  const port = firstPort ? parseInt(firstPort, 10) : 8080

  // Determine default type based on service
  let url = `http://localhost:${port}/health`
  
  // Check for specific service types
  if (service.image?.includes('postgres')) {
    return {
      type: 'tcp',
      port: 5432,
      interval: '10s',
      timeout: '5s',
      retries: 3,
      startPeriod: '10s',
      startInterval: '5s',
    }
  }
  
  if (service.image?.includes('redis')) {
    return {
      type: 'tcp',
      port: 6379,
      interval: '10s',
      timeout: '5s',
      retries: 3,
      startPeriod: '5s',
      startInterval: '5s',
    }
  }
  
  if (service.image?.includes('mongo')) {
    return {
      type: 'tcp',
      port: 27017,
      interval: '10s',
      timeout: '5s',
      retries: 3,
      startPeriod: '10s',
      startInterval: '5s',
    }
  }

  // HTTP defaults for web services
  if (service.language === 'node' || service.image?.includes('node')) {
    url = `http://localhost:${port}/health`
  } else if (service.language === 'python' || service.image?.includes('python')) {
    url = `http://localhost:${port}/health`
  } else if (service.language === 'dotnet' || service.image?.includes('dotnet')) {
    url = `http://localhost:${port}/healthz`
  } else if (service.language === 'java' || service.image?.includes('java')) {
    url = `http://localhost:${port}/actuator/health`
  }

  return {
    type: 'http',
    url,
    interval: '30s',
    timeout: '5s',
    retries: 3,
    startPeriod: '0s',
    startInterval: '5s',
  }
}

/**
 * Validate duration format (e.g., "30s", "1m", "2h")
 */
export function validateDuration(value: string): boolean {
  return /^\d+[smh]$/.test(value)
}

/**
 * Validate URL format
 */
export function validateUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

/**
 * Validate port number
 */
export function validatePort(value: number): boolean {
  return Number.isInteger(value) && value >= 1 && value <= 65535
}

