/**
 * useProject — fetches azure.yaml-derived project metadata via the
 * ProjectService Connect handler.
 *
 * Wire migration note: replaces a one-shot `fetch('/api/project')` that
 * lived inline in App.tsx. Extracting it as a hook lets future consumers
 * (settings panes, breadcrumbs) share the same fetched value without
 * re-issuing the RPC, and gives tests a clean seam via the optional
 * `transport` prop without monkey-patching `globalThis.fetch`.
 *
 * Lifetime: the project name is essentially immutable for the dashboard
 * session (azure.yaml changes require a server restart today), so this
 * hook fetches once on mount and never refetches. If a future feature
 * needs reactive updates, layer that on; don't change this hook's
 * single-shot semantics silently.
 */
import { useEffect, useMemo, useRef, useState } from 'react'
import { Code, ConnectError, type Client, type Transport } from '@connectrpc/connect'

import { createProjectClient } from '@/lib/connectClient'
import type { ProjectService } from '@/gen/proto/azdapp/v1/project_pb.js'
import { create } from '@bufbuild/protobuf'
import { GetProjectRequestSchema } from '@/gen/proto/azdapp/v1/project_pb.js'

export interface UseProjectReturn {
  /** azure.yaml `name`. Empty string until the first response lands. */
  name: string
  /** Absolute project directory reported by the server. */
  dir: string
  /** True until the first GetProject response (success or failure) lands. */
  loading: boolean
  /** Last error message, if any. Cleared on successful fetch. */
  error: string | null
}

export interface UseProjectOptions {
  /**
   * Override the underlying Connect transport. Production code never
   * passes this; tests inject `createRouterTransport(...)` so the real
   * client code path runs against an in-memory service handler.
   */
  transport?: Transport
}

export function useProject(options?: UseProjectOptions): UseProjectReturn {
  const transport = options?.transport
  const client = useMemo<Client<typeof ProjectService>>(
    () => createProjectClient(transport),
    [transport]
  )

  const [name, setName] = useState('')
  const [dir, setDir] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const mountedRef = useRef(true)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    mountedRef.current = true
    const controller = new AbortController()
    abortRef.current = controller

    void (async () => {
      try {
        const resp = await client.getProject(create(GetProjectRequestSchema), {
          signal: controller.signal,
        })
        if (!mountedRef.current) return
        setName(resp.name)
        setDir(resp.dir)
        setError(null)
      } catch (err) {
        if (controller.signal.aborted) return
        if (err instanceof ConnectError && err.code === Code.Canceled) return
        if (!mountedRef.current) return
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        if (mountedRef.current) {
          setLoading(false)
        }
      }
    })()

    return () => {
      mountedRef.current = false
      controller.abort()
      if (abortRef.current === controller) {
        abortRef.current = null
      }
    }
  }, [client])

  return { name, dir, loading, error }
}
