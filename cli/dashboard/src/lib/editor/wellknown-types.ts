/**
 * Well-Known Services Types
 * Types for well-known service definitions from the backend API
 */

export interface WellKnownService {
  /** Unique service identifier (e.g., "azurite", "cosmos", "redis") */
  name: string
  
  /** Display name for UI (e.g., "Azurite (Azure Storage Emulator)") */
  displayName: string
  
  /** Service description */
  description: string
  
  /** Service category for grouping */
  category: 'storage' | 'database' | 'cache' | 'messaging' | 'search' | 'other'
  
  /** Icon identifier or emoji */
  icon?: string
  
  /** Host type (e.g., "containerapp") */
  host: string
  
  /** Docker image */
  image: string
  
  /** Port mappings */
  ports?: string[]
  
  /** Environment variables */
  environment?: Record<string, string>
  
  /** Health check configuration */
  healthcheck?: HealthCheckConfig
  
  /** Connection strings */
  connectionStrings?: Record<string, string>
  
  /** Additional documentation URL */
  docsUrl?: string
}

export interface HealthCheckConfig {
  /** Health check test command or URL */
  test: string | string[]
  
  /** Check interval (e.g., "30s") */
  interval?: string
  
  /** Timeout duration (e.g., "5s") */
  timeout?: string
  
  /** Number of retries before unhealthy */
  retries?: number
  
  /** Start period before checks begin (e.g., "10s") */
  startPeriod?: string
}

export interface ServiceFormData {
  /** Service name */
  name: string
  
  /** Host type */
  host: string
  
  /** Project path (for application services) */
  project?: string
  
  /** Language (for application services) */
  language?: string
  
  /** Docker image (for container services) */
  image?: string
  
  /** Port mappings */
  ports?: string[]
  
  /** Environment variables */
  environment?: Record<string, string>
  
  /** Health check configuration */
  healthcheck?: HealthCheckConfig
  
  /** Service type */
  type?: string
  
  /** Service mode */
  mode?: string
}
