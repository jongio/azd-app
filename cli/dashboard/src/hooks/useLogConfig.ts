/**
 * useLogConfig - Hooks for Log Analytics table selection and configuration
 * Provides state management for available tables and service log configuration.
 */
import { useState, useCallback, useEffect } from 'react'
import { useBackendConnection } from '@/hooks/useBackendConnection'

// =============================================================================
// Types
// =============================================================================

export interface TableInfo {
  name: string
  category: string
  description: string
  columns?: string[]
  recommended?: boolean
}

export interface TableCategory {
  name: string
  displayName: string
  tables: string[]
}

export interface TablesResponse {
  tables: TableInfo[]
  recommended: string[]
  workspace: string
  categories: TableCategory[]
}

export interface LogConfig {
  service: string
  mode: 'tables' | 'custom'
  tables?: string[]
  query?: string
  resourceType?: string
}

export interface UseAvailableTablesOptions {
  /** Resource type to filter/recommend tables */
  resourceType?: string
  /** Auto-fetch on mount */
  autoFetch?: boolean
}

export interface UseAvailableTablesReturn {
  /** Available tables from Log Analytics */
  tables: TableInfo[]
  /** Table categories for grouping */
  categories: TableCategory[]
  /** Recommended tables for the resource type */
  recommended: string[]
  /** Workspace ID (truncated) */
  workspace: string
  /** Loading state */
  isLoading: boolean
  /** Error message */
  error: string | null
  /** Fetch/refresh tables */
  fetchTables: () => Promise<void>
}

export interface UseLogConfigOptions {
  /** Service name */
  serviceName: string
  /** Auto-fetch config on mount */
  autoFetch?: boolean
}

export interface UseLogConfigReturn {
  /** Current configuration */
  config: LogConfig | null
  /** Loading state */
  isLoading: boolean
  /** Saving state */
  isSaving: boolean
  /** Error message */
  error: string | null
  /** Fetch config from server */
  fetchConfig: () => Promise<void>
  /** Save config to server - provide tables OR query, mode is inferred */
  saveConfig: (options: { tables?: string[]; query?: string }) => Promise<boolean>
}

// =============================================================================
// useAvailableTables Hook
// =============================================================================

/**
 * Hook for fetching available Log Analytics tables.
 */
export function useAvailableTables({
  resourceType = 'containerapp',
  autoFetch = true,
}: UseAvailableTablesOptions = {}): UseAvailableTablesReturn {
  const { connected } = useBackendConnection()
  const [tables, setTables] = useState<TableInfo[]>([])
  const [categories, setCategories] = useState<TableCategory[]>([])
  const [recommended, setRecommended] = useState<string[]>([])
  const [workspace, setWorkspace] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchTables = useCallback(async () => {
    if (!connected) {
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams()
      if (resourceType) {
        params.set('resourceType', resourceType)
      }

      const response = await fetch(`/api/azure/tables?${params.toString()}`)

      if (!response.ok) {
        throw new Error(`Failed to fetch tables: ${response.status}`)
      }

      const data = await response.json() as TablesResponse

      // Defensive: backend/network issues can return unexpected shapes.
      // Normalize to arrays to avoid runtime errors in components using `.filter`, `.map`, etc.
      const tables = Array.isArray(data.tables) ? data.tables : []
      const categories = Array.isArray(data.categories) ? data.categories : []
      const recommended = Array.isArray(data.recommended) ? data.recommended : []

      setTables(tables)
      setCategories(categories)
      setRecommended(recommended)
      setWorkspace(typeof data.workspace === 'string' ? data.workspace : '')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch tables'
      setError(message)
      setTables([])
      setCategories([])
      setRecommended([])
    } finally {
      setIsLoading(false)
    }
  }, [connected, resourceType])

  // Auto-fetch on mount
  useEffect(() => {
    if (autoFetch) {
      void fetchTables()
    }
  }, [autoFetch, fetchTables])

  return {
    tables,
    categories,
    recommended,
    workspace,
    isLoading,
    error,
    fetchTables,
  }
}

// =============================================================================
// useLogConfig Hook
// =============================================================================

/**
 * Hook for managing log configuration per service.
 */
export function useLogConfig({
  serviceName,
  autoFetch = true,
}: UseLogConfigOptions): UseLogConfigReturn {
  const { connected } = useBackendConnection()
  const [config, setConfig] = useState<LogConfig | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = useCallback(async () => {
    if (!connected || !serviceName) {
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams({ service: serviceName })
      const response = await fetch(`/api/azure/logs/config?${params.toString()}`)

      if (!response.ok) {
        throw new Error(`Failed to fetch config: ${response.status}`)
      }

      const data = await response.json() as LogConfig
      setConfig(data)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch config'
      setError(message)
      setConfig(null)
    } finally {
      setIsLoading(false)
    }
  }, [connected, serviceName])

  const saveConfig = useCallback(async (
    options: { tables?: string[]; query?: string }
  ): Promise<boolean> => {
    if (!connected) {
      setError('Backend connection lost')
      return false
    }
    
    if (!serviceName) {
      setError('Service name is required')
      return false
    }

    const { tables, query } = options

    // Validate: must have either tables or query
    if (!query && (!tables || tables.length === 0)) {
      setError('Either tables or query is required')
      return false
    }

    setIsSaving(true)
    setError(null)

    try {
      const response = await fetch('/api/azure/logs/config', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          service: serviceName,
          mode: query ? 'custom' : 'tables',
          tables: query ? undefined : tables,  // Only include tables if no query
          query: query || undefined,           // Query takes precedence
        }),
      })

      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || `Failed to save config: ${response.status}`)
      }

      const data = await response.json() as LogConfig
      setConfig(data)
      return true
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to save config'
      setError(message)
      return false
    } finally {
      setIsSaving(false)
    }
  }, [connected, serviceName])

  // Auto-fetch on mount or service change
  useEffect(() => {
    if (autoFetch && serviceName) {
      void fetchConfig()
    }
  }, [autoFetch, serviceName, fetchConfig])

  return {
    config,
    isLoading,
    isSaving,
    error,
    fetchConfig,
    saveConfig,
  }
}

export default useLogConfig
