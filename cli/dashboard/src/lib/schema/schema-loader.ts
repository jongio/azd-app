/**
 * Schema Loader - Loads and caches azure.yaml JSON Schema
 * 
 * Responsibilities:
 * - Fetch schema from remote URL with fallback to bundled version
 * - Handle network errors gracefully
 * - Cache schema in memory
 */

const SCHEMA_URL = 'https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json'

// Schema will be bundled as a fallback
import bundledSchema from './bundled-schema.json'

export interface SchemaLoadResult {
  success: boolean
  schema: Record<string, unknown> | null
  source: 'remote' | 'bundled'
  error?: string
}

/**
 * Load JSON Schema from remote URL with local fallback
 */
export async function loadSchema(): Promise<SchemaLoadResult> {
  try {
    // Try to fetch from remote URL
    const response = await fetch(SCHEMA_URL, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
      // Set timeout to avoid hanging
      signal: AbortSignal.timeout(5000),
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    const schema = await response.json()
    
    return {
      success: true,
      schema,
      source: 'remote',
    }
  } catch (error) {
    // Fallback to bundled schema on any error
    console.warn('Failed to load schema from remote URL, using bundled fallback:', error)
    
    return {
      success: true,
      schema: bundledSchema,
      source: 'bundled',
      error: error instanceof Error ? error.message : String(error),
    }
  }
}

/**
 * Get bundled schema directly (synchronous)
 */
export function getBundledSchema(): Record<string, unknown> {
  return bundledSchema
}
