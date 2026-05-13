/**
 * Hook for detecting and providing Codespace environment configuration.
 *
 * Fetches environment info from the dashboard's LifecycleService Connect
 * handler and caches it for the session. Used to transform localhost URLs
 * to Codespace-forwarded URLs.
 *
 * Wire migration note: this hook used to call GET /api/environment with
 * raw `fetch`. It now uses the generated Connect client. The cached
 * payload shape (sessionStorage `azd-codespace-env`) is preserved as
 * `EnvironmentInfo` so a stale cache from a previous build remains
 * readable across the rollout window.
 */
import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { Code, ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import { createLifecycleClient } from '@/lib/connectClient'
import type { CodespaceConfig, EnvironmentInfo } from '@/lib/codespace-utils'
import type { LifecycleService } from '@/gen/proto/azdapp/v1/lifecycle_pb.js'
import { GetEnvironmentRequestSchema } from '@/gen/proto/azdapp/v1/lifecycle_pb.js'

// =============================================================================
// Types
// =============================================================================

export interface UseCodespaceEnvReturn {
  /** Whether currently in a GitHub Codespace */
  isCodespace: boolean
  /** Codespace configuration (null if not in Codespace or loading) */
  config: CodespaceConfig | null
  /** Azure environment name (if available) */
  environmentName?: string
  /** Whether the environment info is still loading */
  loading: boolean
  /** Error message if fetch failed */
  error: string | null
  /** Manually refresh environment info */
  refresh: () => void
}

export interface UseCodespaceEnvOptions {
  /**
   * Override the underlying Connect transport. Production code never
   * passes this; it exists exclusively as a unit-test seam so specs can
   * inject `createRouterTransport(...)` and exercise the real client
   * code path against an in-memory service handler.
   */
  transport?: Transport
}

// =============================================================================
// Constants
// =============================================================================

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
export function useCodespaceEnv(options?: UseCodespaceEnvOptions): UseCodespaceEnvReturn {
  // The client is constructed once per (hook instance, transport) pair.
  // Memoising on `transport` keeps tests deterministic when a spec swaps
  // the in-memory transport between renders.
  const transport = options?.transport
  const client = useMemo<Client<typeof LifecycleService>>(
    () => createLifecycleClient(transport),
    [transport]
  )

  const [config, setConfig] = useState<CodespaceConfig | null>(() => {
    // Initialize from cache if available
    const cached = getCachedEnv()
    return cached?.codespace ?? null
  })
  const [environmentName, setEnvironmentName] = useState<string | undefined>(() => {
    // Initialize from cache if available
    const cached = getCachedEnv()
    return cached?.environmentName
  })
  const [loading, setLoading] = useState<boolean>(() => {
    // Skip loading if we have cached data
    return getCachedEnv() === null
  })
  const [error, setError] = useState<string | null>(null)

  // Track in-flight requests so unmount can abort and so concurrent
  // refresh() calls don't race to setState on a dead component.
  const abortRef = useRef<AbortController | null>(null)
  const mountedRef = useRef(true)

  const fetchEnvironment = useCallback(async () => {
    // Cancel any in-flight request from a prior call. Connect honours
    // AbortSignal and surfaces it as a CodeCanceled ConnectError, which
    // we explicitly swallow below.
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    try {
      setLoading(true)
      setError(null)

      const resp = await client.getEnvironment(create(GetEnvironmentRequestSchema), {
        signal: controller.signal,
      })

      const codespace = resp.codespace
      const data: EnvironmentInfo = {
        codespace: {
          enabled: codespace?.enabled ?? false,
          name: codespace?.name ?? '',
          domain: codespace?.domain ?? '',
          isVsCodeDesktop: codespace?.isVsCodeDesktop ?? false,
        },
        // proto3 returns `""` for unset string; preserve the legacy REST
        // semantic of `undefined when empty` so downstream code that does
        // `if (environmentName)` keeps working unchanged.
        environmentName: resp.environmentName !== '' ? resp.environmentName : undefined,
      }

      // Cache the result
      setCachedEnv(data)

      if (!mountedRef.current) return
      setConfig(data.codespace)
      setEnvironmentName(data.environmentName)
    } catch (err) {
      // Aborted requests are not errors at the hook contract; they fire
      // on unmount or when a newer fetch supersedes us.
      if (controller.signal.aborted) return
      if (err instanceof ConnectError && err.code === Code.Canceled) return

      const message = err instanceof Error ? err.message : 'Unknown error'
      if (!mountedRef.current) return
      setError(message)
      // Don't clear config on error - keep using cached value if available
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
      }
      if (mountedRef.current) {
        setLoading(false)
      }
    }
  }, [client])

  // Fetch on mount if not cached
  useEffect(() => {
    mountedRef.current = true
    const cached = getCachedEnv()
    if (!cached) {
      void fetchEnvironment()
    }
    return () => {
      mountedRef.current = false
      abortRef.current?.abort()
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
    environmentName,
    loading,
    error,
    refresh,
  }
}
