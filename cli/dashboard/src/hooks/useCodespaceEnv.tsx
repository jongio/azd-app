/**
 * Hook for detecting and providing Codespace environment configuration.
 * 
 * Fetches environment info from the backend API and caches it for the session.
 * Used to transform localhost URLs to Codespace-forwarded URLs.
 */
import { useState, useEffect, useCallback } from 'react'
import type { CodespaceConfig, EnvironmentInfo } from '@/lib/codespace-utils'

// =============================================================================
// Types
// =============================================================================

export interface UseCodespaceEnvReturn {
  /** Whether currently in a GitHub Codespace */
  isCodespace: boolean
  /** Codespace configuration (null if not in Codespace or loading) */
  config: CodespaceConfig | null
  /** Whether the environment info is still loading */
  loading: boolean
  /** Error message if fetch failed */
  error: string | null
  /** Manually refresh environment info */
  refresh: () => void
}

// =============================================================================
// Constants
// =============================================================================

const API_BASE = ''

/** Cache key for sessionStorage */
const CACHE_KEY = 'azd-codespace-env'

/** Cache TTL in milliseconds (5 minutes) */
const CACHE_TTL_MS = 5 * 60 * 1000

// =============================================================================
// Cache Helpers
// =============================================================================

interface CachedEnvInfo {
  data: EnvironmentInfo
  timestamp: number
}

function getCachedEnv(): EnvironmentInfo | null {
  try {
    const cached = sessionStorage.getItem(CACHE_KEY)
    if (!cached) return null

    const parsed = JSON.parse(cached) as CachedEnvInfo
    const age = Date.now() - parsed.timestamp

    // Return cached data if still fresh
    if (age < CACHE_TTL_MS) {
      return parsed.data
    }

    // Clear stale cache
    sessionStorage.removeItem(CACHE_KEY)
  } catch {
    // Ignore cache errors
  }
  return null
}

function setCachedEnv(data: EnvironmentInfo): void {
  try {
    const cached: CachedEnvInfo = {
      data,
      timestamp: Date.now(),
    }
    sessionStorage.setItem(CACHE_KEY, JSON.stringify(cached))
  } catch {
    // Ignore cache errors (e.g., storage full)
  }
}

// =============================================================================
// Hook Implementation
// =============================================================================

/**
 * Hook for detecting Codespace environment and providing configuration.
 * 
 * @example
 * const { isCodespace, config } = useCodespaceEnv()
 * 
 * const url = isCodespace
 *   ? transformLocalhostUrl('http://localhost:3000', config)
 *   : 'http://localhost:3000'
 */
export function useCodespaceEnv(): UseCodespaceEnvReturn {
  const [config, setConfig] = useState<CodespaceConfig | null>(() => {
    // Initialize from cache if available
    const cached = getCachedEnv()
    return cached?.codespace ?? null
  })
  const [loading, setLoading] = useState<boolean>(() => {
    // Skip loading if we have cached data
    return getCachedEnv() === null
  })
  const [error, setError] = useState<string | null>(null)

  const fetchEnvironment = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      const response = await fetch(`${API_BASE}/api/environment`)
      
      if (!response.ok) {
        throw new Error(`Failed to fetch environment: ${response.statusText}`)
      }

      const data = await response.json() as EnvironmentInfo
      
      // Cache the result
      setCachedEnv(data)
      
      setConfig(data.codespace)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setError(message)
      // Don't clear config on error - keep using cached value if available
    } finally {
      setLoading(false)
    }
  }, [])

  // Fetch on mount if not cached
  useEffect(() => {
    const cached = getCachedEnv()
    if (!cached) {
      void fetchEnvironment()
    }
  }, [fetchEnvironment])

  const refresh = useCallback(() => {
    // Clear cache and refetch
    try {
      sessionStorage.removeItem(CACHE_KEY)
    } catch {
      // Ignore
    }
    void fetchEnvironment()
  }, [fetchEnvironment])

  return {
    isCodespace: config?.enabled ?? false,
    config,
    loading,
    error,
    refresh,
  }
}
