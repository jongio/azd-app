/**
 * Config API with Retry Integration
 */

import { retryWithBackoff, createNetworkError, createFileSystemError } from '../errors'
import type { EditorError } from '../errors'

const API_BASE = '/api/editor'

/**
 * Azure YAML Configuration structure
 */
export interface AzureYamlConfig {
  name: string
  services?: Record<string, unknown>
  resources?: Record<string, unknown>
  hooks?: Record<string, unknown>
  pipeline?: Record<string, unknown>
  metadata?: Record<string, unknown>
  [key: string]: unknown
}

/**
 * Schema response structure
 */
export interface SchemaResponse {
  schema: Record<string, unknown>
}

/**
 * Well-known services response
 */
export interface WellKnownServicesResponse {
  services: Array<Record<string, unknown>>
}

/**
 * Load configuration with retry
 */
export async function loadConfigWithRetry(): Promise<{ config: AzureYamlConfig | null; error?: EditorError }> {
  try {
    const response = await retryWithBackoff(
      async () => {
        const res = await fetch(`${API_BASE}/config`)
        if (!res.ok) {
          throw Object.assign(new Error(`HTTP ${res.status}`), { status: res.status })
        }
        return res
      },
      {
        maxRetries: 3,
        initialDelay: 1000,
        onRetry: (attempt) => {
          // Only log in development
          if (process.env.NODE_ENV !== 'production') {
            console.warn(`Retrying config load (${attempt}/3)`)
          }
        },
      }
    )

    const data = await response.json()
    
    // Basic validation - config must have a name
    if (!data || typeof data !== 'object' || !('name' in data)) {
      throw new Error('Invalid configuration structure: missing name field')
    }
    
    return { config: data as AzureYamlConfig }
  } catch (error) {
    return {
      config: null,
      error: createNetworkError('Failed to load configuration', error as Error, false),
    }
  }
}

/**
 * Save configuration with retry (requires user confirmation for retries)
 */
export async function saveConfigWithRetry(
  config: AzureYamlConfig,
): Promise<{ success: boolean; error?: EditorError }> {
  try {
    await retryWithBackoff(
      async () => {
        const res = await fetch(`${API_BASE}/config`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(config),
        })
        if (!res.ok) {
          throw Object.assign(new Error(`HTTP ${res.status}`), { status: res.status })
        }
        return res
      },
      {
        maxRetries: 3,
        initialDelay: 1000,
        shouldRetry: (error) => {
          // Default: only retry on network errors, not client errors
          if (error && typeof error === 'object' && 'status' in error) {
            const status = (error as { status: number }).status
            return status >= 500
          }
          return true
        },
        onRetry: (attempt) => {
          // Only log in development
          if (process.env.NODE_ENV !== 'production') {
            console.warn(`Retrying config save (${attempt}/3)`)
          }
        },
      }
    )

    return { success: true }
  } catch (error) {
    return {
      success: false,
      error:
        error && typeof error === 'object' && 'status' in error && (error as any).status === 403
          ? createFileSystemError('Permission denied. Cannot save configuration.', error as unknown as Error)
          : createNetworkError('Failed to save configuration', error as unknown as Error, false),
    }
  }
}

/**
 * Load schema with retry
 */
export async function loadSchemaWithRetry(): Promise<{ schema: SchemaResponse | null; error?: EditorError }> {
  try {
    const response = await retryWithBackoff(
      async () => {
        const res = await fetch(`${API_BASE}/schema`)
        if (!res.ok) {
          throw Object.assign(new Error(`HTTP ${res.status}`), { status: res.status })
        }
        return res
      },
      {
        maxRetries: 3,
        initialDelay: 1000,
      }
    )

    const data = await response.json()
    
    // Validate schema structure
    if (!data || typeof data !== 'object' || !('schema' in data)) {
      throw new Error('Invalid schema response structure')
    }
    
    return { schema: data as SchemaResponse }
  } catch (error) {
    return {
      schema: null,
      error: createNetworkError('Failed to load schema', error as Error, false),
    }
  }
}

/**
 * Load well-known services with retry
 */
export async function loadWellKnownWithRetry(): Promise<{ services: Array<Record<string, unknown>>; error?: EditorError }> {
  try {
    const response = await retryWithBackoff(
      async () => {
        const res = await fetch(`${API_BASE}/wellknown`)
        if (!res.ok) {
          throw Object.assign(new Error(`HTTP ${res.status}`), { status: res.status })
        }
        return res
      },
      {
        maxRetries: 3,
        initialDelay: 1000,
      }
    )

    const data = await response.json()
    
    // Validate wellknown structure
    if (!data || typeof data !== 'object') {
      throw new Error('Invalid well-known services response')
    }
    
    const services = (data as WellKnownServicesResponse).services || []
    return { services }
  } catch (error) {
    return {
      services: [],
      error: createNetworkError('Failed to load well-known services', error as Error, false),
    }
  }
}


