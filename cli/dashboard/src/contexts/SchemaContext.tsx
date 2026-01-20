/**
 * Schema Context - Provides schema state and actions to components
 * 
 * Manages:
 * - Schema loading (remote + fallback)
 * - Schema parsing
 * - In-memory caching
 * - Loading states
 */

import { createContext, useContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { loadSchema, type SchemaLoadResult } from '@/lib/schema/schema-loader'
import { parseSchema, type ParsedSchema } from '@/lib/schema/schema-parser'

interface SchemaContextValue {
  // State
  schema: ParsedSchema | null
  rawSchema: Record<string, unknown> | null
  isLoading: boolean
  error: string | null
  source: 'remote' | 'bundled' | null

  // Actions
  refreshSchema: () => Promise<void>
}

const SchemaContext = createContext<SchemaContextValue | undefined>(undefined)

export interface SchemaProviderProps {
  children: ReactNode
}

/**
 * Schema Provider - Loads and caches schema on mount
 */
export function SchemaProvider({ children }: SchemaProviderProps) {
  const [schema, setSchema] = useState<ParsedSchema | null>(null)
  const [rawSchema, setRawSchema] = useState<Record<string, unknown> | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [source, setSource] = useState<'remote' | 'bundled' | null>(null)

  const loadAndParseSchema = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const result: SchemaLoadResult = await loadSchema()

      if (!result.success || !result.schema) {
        throw new Error(result.error || 'Failed to load schema')
      }

      setRawSchema(result.schema)
      setSource(result.source)

      // Parse schema into internal model
      const parsed = parseSchema(result.schema)
      setSchema(parsed)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Load schema on mount
  useEffect(() => {
    void loadAndParseSchema()
  }, [loadAndParseSchema])

  const value: SchemaContextValue = {
    schema,
    rawSchema,
    isLoading,
    error,
    source,
    refreshSchema: loadAndParseSchema,
  }

  return <SchemaContext.Provider value={value}>{children}</SchemaContext.Provider>
}

/**
 * Hook to access schema context
 */
export function useSchema(): SchemaContextValue {
  const context = useContext(SchemaContext)
  
  if (!context) {
    throw new Error('useSchema must be used within SchemaProvider')
  }

  return context
}
