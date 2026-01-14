/**
 * API Response Caching
 * Provides caching layer for API responses using SWR
 * 
 * Features:
 * - Stale-while-revalidate strategy
 * - LocalStorage persistence for schema
 * - Automatic revalidation
 * - Cache invalidation
 */

import useSWR, { SWRConfiguration, mutate } from 'swr'
import { loadConfig, listBackups } from '@/lib/editor/config-api'
import { fetchWellKnownServices, fetchWellKnownService } from '@/lib/api/wellknown'

/**
 * Default SWR configuration
 */
const DEFAULT_CONFIG: SWRConfiguration = {
  revalidateOnFocus: false,
  revalidateOnReconnect: true,
  shouldRetryOnError: true,
  errorRetryCount: 3,
  errorRetryInterval: 1000,
}

/**
 * Schema cache configuration (indefinite cache)
 */
const SCHEMA_CACHE_CONFIG: SWRConfiguration = {
  ...DEFAULT_CONFIG,
  revalidateOnMount: false,
  revalidateOnFocus: false,
  revalidateOnReconnect: false,
  dedupingInterval: Infinity, // Never refetch
}

/**
 * Well-known services cache (5 minute cache)
 */
const WELLKNOWN_CACHE_CONFIG: SWRConfiguration = {
  ...DEFAULT_CONFIG,
  dedupingInterval: 5 * 60 * 1000, // 5 minutes
  focusThrottleInterval: 5 * 60 * 1000,
}

/**
 * Config cache (frequently updated)
 */
const CONFIG_CACHE_CONFIG: SWRConfiguration = {
  ...DEFAULT_CONFIG,
  dedupingInterval: 1000, // 1 second
  refreshInterval: 0, // No auto-refresh (only manual)
}

/**
 * Generic fetcher for SWR
 */
async function fetcher<T>(url: string): Promise<T> {
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`Failed to fetch ${url}: ${response.statusText}`)
  }
  return response.json()
}

/**
 * Hook: Cached configuration loading
 */
export function useCachedConfig() {
  return useSWR('/api/editor/config', loadConfig, CONFIG_CACHE_CONFIG)
}

/**
 * Hook: Cached schema loading with localStorage persistence
 */
export function useCachedSchema() {
  const SCHEMA_URL = 'https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json'
  const STORAGE_KEY = 'azd-editor-schema-v1.1'

  const { data, error, isLoading } = useSWR(
    SCHEMA_URL,
    async (url: string) => {
      // Try localStorage first
      const cached = localStorage.getItem(STORAGE_KEY)
      if (cached) {
        try {
          const parsed = JSON.parse(cached)
          // Return cached immediately, but trigger background revalidation
          setTimeout(() => {
            fetcher(url).then((data) => {
              localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
              mutate(SCHEMA_URL, data, false)
            })
          }, 0)
          return parsed
        } catch {
          // Invalid cache, fetch fresh
        }
      }

      // Fetch from network
      const data = await fetcher(url)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
      return data
    },
    SCHEMA_CACHE_CONFIG
  )

  return { schema: data, error, isLoading }
}

/**
 * Hook: Cached well-known services
 */
export function useCachedWellKnownServices() {
  return useSWR('/api/editor/wellknown', fetchWellKnownServices, WELLKNOWN_CACHE_CONFIG)
}

/**
 * Hook: Cached specific well-known service
 */
export function useCachedWellKnownService(name: string | null) {
  return useSWR(
    name ? `/api/editor/wellknown/${name}` : null,
    () => (name ? fetchWellKnownService(name) : null),
    WELLKNOWN_CACHE_CONFIG
  )
}

/**
 * Hook: Cached backups list
 */
export function useCachedBackups() {
  return useSWR('/api/editor/backups', listBackups, DEFAULT_CONFIG)
}

/**
 * Invalidate config cache (after save)
 */
export function invalidateConfigCache() {
  return mutate('/api/editor/config')
}

/**
 * Invalidate backups cache (after backup operation)
 */
export function invalidateBackupsCache() {
  return mutate('/api/editor/backups')
}

/**
 * Invalidate well-known services cache
 */
export function invalidateWellKnownCache() {
  return mutate('/api/editor/wellknown')
}

/**
 * Clear all caches
 */
export function clearAllCaches() {
  mutate(() => true, undefined, { revalidate: false })
  localStorage.removeItem('azd-editor-schema-v1.1')
}

/**
 * Preload API endpoints for faster initial load
 */
export async function preloadCaches() {
  // Preload schema
  const SCHEMA_URL = 'https://raw.githubusercontent.com/jongio/azd-app/main/schemas/v1.1/azure.yaml.json'
  mutate(SCHEMA_URL, fetcher(SCHEMA_URL))

  // Preload well-known services
  mutate('/api/editor/wellknown', fetchWellKnownServices())

  // Preload config
  mutate('/api/editor/config', loadConfig())
}
