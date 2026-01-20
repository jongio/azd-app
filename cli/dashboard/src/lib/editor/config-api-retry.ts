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
        onRetry: (_attempt) => {
          // Silent retry
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
        onRetry: (_attempt) => {
          // Silent retry
        },
      }
    )

    return { success: true }
  } catch (error) {
    return {
      success: false,
      error:
        error && typeof error === 'object' && 'status' in error && (error as { status: number }).status === 403
          ? createFileSystemError('Permission denied. Cannot save configuration.', error as unknown as Error)
          : createNetworkError('Failed to save configuration', error as unknown as Error, false),
    }
  }
}

/**
 * Load schema with retry
 */
const SCHEMA_CACHE_KEY = 'azd-schema-cache'
const SCHEMA_CACHE_TTL_MS = 60 * 60 * 1000 // 1 hour

function readCachedSchema(): Record<string, unknown> | null {
  try {
    const cached = localStorage.getItem(SCHEMA_CACHE_KEY)
    if (!cached) return null

    const parsed = JSON.parse(cached) as { schema: Record<string, unknown>; timestamp: number }
    if (!parsed?.schema || typeof parsed.timestamp !== 'number') {
      return null
    }

    const isFresh = Date.now() - parsed.timestamp < SCHEMA_CACHE_TTL_MS
    return isFresh ? parsed.schema : null
  } catch {
    return null
  }
}

function writeCachedSchema(schema: Record<string, unknown>) {
  try {
    localStorage.setItem(
      SCHEMA_CACHE_KEY,
      JSON.stringify({ schema, timestamp: Date.now() })
    )
  } catch {
    // Best effort cache; ignore failures (e.g., private mode)
  }
}

export async function loadSchemaWithRetry(): Promise<{ schema: Record<string, unknown> | null; error?: EditorError }> {
  const cachedSchema = readCachedSchema()
  if (cachedSchema) {
    return { schema: cachedSchema }
  }

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

    // Accept either wrapped { schema } shape or raw schema object
    const schema = data && typeof data === 'object' && 'schema' in data ? (data as SchemaResponse).schema : (data as Record<string, unknown>)

    if (!schema || typeof schema !== 'object') {
      throw new Error('Invalid schema response structure')
    }

    writeCachedSchema(schema)
    return { schema }
  } catch (error) {
    return {
      schema: cachedSchema ?? null,
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


