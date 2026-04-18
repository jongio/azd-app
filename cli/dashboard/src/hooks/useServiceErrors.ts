/**
 * useServiceErrors — fan out N parallel Connect log streams (one per
 * service) and surface a boolean for whether ANY service has produced
 * an error-level entry in the last 30 seconds.
 *
 * Wire migration note: replaces the per-service `/api/logs/stream`
 * WebSocket fan-out with `LogsService.StreamLocalLogs` (one stream
 * per service, `service_name` set). Public surface preserved exactly:
 * same signature, same `{ hasActiveErrors }` return.
 *
 * Reconnect policy intentionally matches the legacy WS hook: we do
 * NOT reconnect on stream failure. The pre-migration version silently
 * swallowed `ws.onerror`/`ws.onclose` because services come and go
 * (deploy/restart) and a noisy reconnect storm wasn't worth the
 * marginal improvement in error-detection latency. AbortController
 * tears down all streams on unmount or on `serviceNames` change.
 */
import { useEffect, useRef, useState } from 'react'
import { Code, ConnectError, type Transport } from '@connectrpc/connect'

import { LOG_LEVELS, isErrorLine } from '@/lib/log-utils'
import { useBackendConnection } from '@/hooks/useBackendConnection'
import { createLogsClient } from '@/lib/connectClient'
import { protoToLogEntry } from '@/hooks/useSharedLogStream'
import { StreamLocalLogsRequest } from '@/gen/proto/azdapp/v1/logs_pb.js'

interface UseServiceErrorsOptions {
  /** Test-only Connect transport override; production omits. */
  transport?: Transport
}

const ERROR_WINDOW_MS = 30_000
const POLL_INTERVAL_MS = 1_000

/**
 * Track active errors across the given services. Returns
 * `{ hasActiveErrors }` which is true when at least one service has
 * emitted an error-level log line within the last 30 seconds.
 *
 * `serviceNames` should be a stable reference (memoised by the caller)
 * to avoid tearing down/re-establishing streams on every render.
 */
export function useServiceErrors(
  serviceNames: string[],
  options: UseServiceErrorsOptions = {},
) {
  const { connected } = useBackendConnection()
  const [hasActiveErrors, setHasActiveErrors] = useState(false)
  const errorTimestampsRef = useRef<Map<string, number>>(new Map())

  useEffect(() => {
    if (!connected || serviceNames.length === 0) return

    const client = createLogsClient(options.transport)
    const controllers = serviceNames.map(() => new AbortController())

    serviceNames.forEach((serviceName, idx) => {
      const controller = controllers[idx]
      void (async () => {
        try {
          const req = new StreamLocalLogsRequest({ serviceName, backfill: 0 })
          for await (const resp of client.streamLocalLogs(req, { signal: controller.signal })) {
            if (controller.signal.aborted) break
            const event = resp.event
            if (!event || event.case !== 'entry') continue
            const proto = event.value
            if (!proto) continue
            const entry = protoToLogEntry(proto)
            const isError = entry.level === LOG_LEVELS.ERROR || isErrorLine(entry.message)
            if (isError) {
              errorTimestampsRef.current.set(`${serviceName}-${Date.now()}`, Date.now())
            }
          }
        } catch (err) {
          if (controller.signal.aborted) return
          if (err instanceof ConnectError && err.code === Code.Canceled) return
          if (err instanceof ConnectError && err.code === Code.NotFound) {
            // Service may not be ready yet; legacy hook silently
            // swallowed the same condition (ws.onerror noop) so we
            // preserve that behaviour rather than spamming the
            // console while services warm up.
            return
          }
          // Match legacy: silent. Errors reaching this branch are
          // typically transient (service restart, network blip) and
          // resolved by the next render that recreates the streams.
        }
      })()
    })

    return () => {
      controllers.forEach((c) => c.abort())
    }
  }, [connected, serviceNames, options.transport])

  // Periodically prune old error timestamps and update the boolean.
  useEffect(() => {
    const interval = setInterval(() => {
      const cutoff = Date.now() - ERROR_WINDOW_MS
      const entries = Array.from(errorTimestampsRef.current.entries())
      entries.forEach(([key, timestamp]) => {
        if (timestamp < cutoff) {
          errorTimestampsRef.current.delete(key)
        }
      })
      setHasActiveErrors(errorTimestampsRef.current.size > 0)
    }, POLL_INTERVAL_MS)

    return () => clearInterval(interval)
  }, [])

  return { hasActiveErrors }
}
