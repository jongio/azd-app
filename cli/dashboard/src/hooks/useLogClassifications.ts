import { useState, useEffect, useCallback, useMemo } from 'react'
import { Code, ConnectError, type Transport } from '@connectrpc/connect'

import { createLogsClient } from '@/lib/connectClient'
import {
  Classification as PbClassification,
} from '@/gen/proto/azdapp/v1/logs_pb.js'
import { LogLevel } from '@/gen/proto/azdapp/v1/common_pb.js'

/**
 * Log classification stored in azure.yaml. Public shape preserved across
 * the Connect migration: callers (LogsView, useSharedLogStream, the test
 * harness) match on the literal level strings, so the proto LogLevel
 * enum is converted at the hook boundary rather than leaked.
 */
export interface LogClassification {
  text: string   // Text to match (case-insensitive)
  level: 'info' | 'warning' | 'error'
}

// Custom event name for classification changes. Kept as a window-scoped
// CustomEvent so multiple useLogClassifications instances stay in sync
// without a shared store - the same in-process pub/sub the legacy hook
// used. The Connect migration only swaps the wire; cross-instance sync
// still has to flow through the DOM.
const CLASSIFICATIONS_CHANGED_EVENT = 'classificationsChanged'

function notifyClassificationsChanged() {
  window.dispatchEvent(new CustomEvent(CLASSIFICATIONS_CHANGED_EVENT))
}

/**
 * Translate the proto LogLevel enum into the legacy literal strings the
 * dashboard already uses. WARN maps to "warning" because the legacy
 * REST contract used the long form and downstream filters key on it.
 * Anything that isn't INFO/WARN/ERROR is treated as INFO for safety
 * (UNSPECIFIED, TRACE, DEBUG, FATAL aren't valid classification levels
 * server-side, so receiving one is a server bug rather than a hot path
 * we should crash on).
 */
function pbLevelToLiteral(level: LogLevel): 'info' | 'warning' | 'error' {
  switch (level) {
    case LogLevel.ERROR:
    case LogLevel.FATAL:
      return 'error'
    case LogLevel.WARN:
      return 'warning'
    case LogLevel.INFO:
    case LogLevel.TRACE:
    case LogLevel.DEBUG:
    case LogLevel.UNSPECIFIED:
    default:
      return 'info'
  }
}

function literalToPbLevel(level: 'info' | 'warning' | 'error'): LogLevel {
  switch (level) {
    case 'error':
      return LogLevel.ERROR
    case 'warning':
      return LogLevel.WARN
    case 'info':
    default:
      return LogLevel.INFO
  }
}

/**
 * Hook for managing log classifications stored in azure.yaml.
 *
 * Classifications allow users to override the default log-level detection
 * by specifying text patterns and their desired classification level.
 *
 * Example: "Connection refused" -> error
 *          "cache miss"         -> info (downgrade from warning)
 *
 * The optional `transport` argument is for tests that wire an in-memory
 * `createRouterTransport` against a stub LogsService handler. Production
 * callers pass nothing and get the singleton browser transport from
 * connectClient.ts.
 */
export function useLogClassifications(transport?: Transport) {
  const [classifications, setClassifications] = useState<LogClassification[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  // Memoise the client so referential identity is stable across renders
  // when the transport is. Without this, every render would rebuild the
  // client and the loadClassifications useCallback would re-fire.
  const client = useMemo(() => createLogsClient(transport), [transport])

  const loadClassifications = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const resp = await client.listClassifications({})
      setClassifications(
        resp.classifications.map(c => ({
          text: c.text,
          level: pbLevelToLiteral(c.level),
        }))
      )
    } catch (err) {
      console.error('Failed to load classifications:', err)
      setError(err instanceof Error ? err : new Error('Failed to load classifications'))
      setClassifications([])
    } finally {
      setIsLoading(false)
    }
  }, [client])

  // Load on mount and listen for changes from other hook instances.
  useEffect(() => {
    void loadClassifications()
    const handleChange = () => {
      void loadClassifications()
    }
    window.addEventListener(CLASSIFICATIONS_CHANGED_EVENT, handleChange)
    return () => {
      window.removeEventListener(CLASSIFICATIONS_CHANGED_EVENT, handleChange)
    }
  }, [loadClassifications])

  /**
   * Add a new classification. If the text already exists, updates the
   * level instead (case-insensitive match, server-side).
   * @param skipNotify - true skips reload + cross-instance notification
   *                     for batch operations (matches legacy contract).
   */
  const addClassification = useCallback(async (
    text: string,
    level: 'info' | 'warning' | 'error',
    skipNotify = false
  ): Promise<LogClassification> => {
    try {
      const resp = await client.addClassification({
        classification: new PbClassification({
          text,
          level: literalToPbLevel(level),
        }),
      })
      const stored = resp.classification
      const result: LogClassification = stored
        ? { text: stored.text, level: pbLevelToLiteral(stored.level) }
        : { text, level }

      if (!skipNotify) {
        await loadClassifications()
        notifyClassificationsChanged()
      }
      return result
    } catch (err) {
      console.error('Failed to add classification:', err)
      // ConnectError surfaces a useful .message; rethrow as Error so
      // existing toast-on-throw call sites keep their copy.
      throw err instanceof ConnectError
        ? new Error(err.rawMessage || err.message)
        : err
    }
  }, [client, loadClassifications])

  /**
   * Delete a classification by its current index. The index is the
   * position in the most recent ListClassifications response; callers
   * are expected to refresh via reload() / event before issuing
   * positional deletes if they may be racing other writers.
   */
  const deleteClassification = useCallback(async (
    index: number,
    skipNotify = false
  ): Promise<void> => {
    try {
      await client.deleteClassification({ index })
      if (!skipNotify) {
        await loadClassifications()
        notifyClassificationsChanged()
      }
    } catch (err) {
      console.error('Failed to delete classification:', err)
      throw err instanceof ConnectError
        ? new Error(err.rawMessage || err.message)
        : err
    }
  }, [client, loadClassifications])

  /**
   * Get the classification level for a given log message. Uses
   * longest-match-wins when multiple classifications match. Pure on the
   * current `classifications` state - no network. Behaviour is bit-for-
   * bit identical to the legacy hook.
   */
  const getClassificationForText = useCallback(
    (text: string): 'info' | 'warning' | 'error' | null => {
      if (!text || classifications.length === 0) {
        return null
      }
      const lowerText = text.toLowerCase()
      const matches = classifications.filter(c =>
        lowerText.includes(c.text.toLowerCase())
      )
      if (matches.length === 0) {
        return null
      }
      const longestMatch = matches.reduce((prev, curr) =>
        curr.text.length > prev.text.length ? curr : prev
      )
      return longestMatch.level
    },
    [classifications]
  )

  return {
    classifications,
    isLoading,
    error,
    addClassification,
    deleteClassification,
    getClassificationForText,
    reload: loadClassifications,
  }
}

// Re-export Code for tests/callers that want to assert specific Connect
// codes on rejected promises without importing connect-web directly.
export { Code as ConnectCode }
