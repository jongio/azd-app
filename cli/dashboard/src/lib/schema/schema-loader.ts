/**
 * Schema Loader - Loads and caches azure.yaml JSON Schema
 * 
 * Responsibilities:
 * - Load schema from bundled version (packaged with dashboard)
 * - Optionally fetch from remote URL for development/testing
 * - Handle errors gracefully
 * - Cache schema in memory
 * 
 * Strategy:
 * - PRIMARY: Use bundled schema (packaged with dashboard build)
 *   This ensures version consistency and no network dependency
 * - FALLBACK: In development, optionally try remote URL if bundled fails
 */

const SCHEMA_URL = 'https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json'

// Bundled schema is the primary source - packaged with the dashboard build
import bundledSchema from './bundled-schema.json'

function isSchemaRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

export interface SchemaLoadResult {
  success: boolean
  schema: Record<string, unknown> | null
  source: 'bundled' | 'remote'
  error?: string
}

/**
 * Load JSON Schema - prioritizes bundled version for consistency
 * 
 * In production: Always uses bundled schema (packaged with build)
 * In development: Uses bundled schema, with optional remote fallback if needed
 */
export async function loadSchema(): Promise<SchemaLoadResult> {
  // PRIMARY: Use bundled schema (packaged with dashboard)
  // This ensures we always know what version is being used
  const bundledSchemaSource = bundledSchema as unknown
  const bundled: Record<string, unknown> | null = isSchemaRecord(bundledSchemaSource)
    ? bundledSchemaSource
    : null

  if (bundled) {
    return {
      success: true,
      schema: bundled,
      source: 'bundled',
    }
  }

  // FALLBACK: Only if bundled schema is invalid/missing, try remote
  // This should rarely happen in production since schema is packaged
  
  try {
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

    const schemaResponse = (await response.json()) as unknown

    if (!isSchemaRecord(schemaResponse)) {
      throw new Error('Invalid schema format received from remote source')
    }

    return {
      success: true,
      schema: schemaResponse,
      source: 'remote',
    }
  } catch (error) {
    return {
      success: false,
      schema: null,
      source: 'remote',
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
