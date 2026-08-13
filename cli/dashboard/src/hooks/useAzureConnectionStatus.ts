/**
 * useAzureConnectionStatus - Manages Azure connection status and mode switching.
 *
 * Wire migration: this hook used to talk to GET/PUT /api/mode through
 * raw fetch. It now uses the ModeService Connect client. The public
 * surface (return shape, options, behavior) is preserved on purpose so
 * existing consumers (ConsoleView, ModeToggle) need no changes.
 *
 * Test seam: an optional `transport` option lets tests inject an
 * in-memory router transport. Production code never passes one and
 * gets the singleton shared with every other hook.
 */
import * as React from 'react'
import { Code, ConnectError, type Client, type Transport } from '@connectrpc/connect'
import { create } from '@bufbuild/protobuf'

import type { LogMode } from '@/components/ModeToggle'
import { createModeClient } from '@/lib/connectClient'
import type { ModeService } from '@/gen/proto/azdapp/v1/mode_pb.js'
import {
  GetModeRequestSchema,
  type GetModeResponse,
  LogMode as ProtoLogMode,
  SetModeRequestSchema,
  type SetModeResponse,
} from '@/gen/proto/azdapp/v1/mode_pb.js'

// =============================================================================
// Types
// =============================================================================

export type AzureConnectionStatus = 'connected' | 'degraded' | 'disconnected' | 'connecting' | 'disabled'

// =============================================================================
// Helpers
// =============================================================================

function isLogMode(value: unknown): value is LogMode {
  return value === 'local' || value === 'azure'
}

function isAzureConnectionStatus(value: string): value is AzureConnectionStatus {
  return (
    value === 'connected' ||
    value === 'degraded' ||
    value === 'disconnected' ||
    value === 'connecting' ||
    value === 'disabled'
  )
}

/**
 * Map proto LogMode → string LogMode used by the rest of the dashboard.
 * UNSPECIFIED is treated as undefined (not a valid mode); the hook then
 * leaves the existing local state alone.
 */
function protoToLogMode(m: ProtoLogMode): LogMode | undefined {
  switch (m) {
    case ProtoLogMode.LOCAL:
      return 'local'
    case ProtoLogMode.AZURE:
      return 'azure'
    default:
      return undefined
  }
}

/**
 * Map string LogMode → proto LogMode for SetMode requests. Invalid
 * inputs map to UNSPECIFIED, which the server rejects with
 * InvalidArgument; that signal bubbles back as a ConnectError and the
 * caller logs/rolls back exactly like the legacy 400 path.
 */
function logModeToProto(m: LogMode): ProtoLogMode {
  switch (m) {
    case 'local':
      return ProtoLogMode.LOCAL
    case 'azure':
      return ProtoLogMode.AZURE
    default:
      return ProtoLogMode.UNSPECIFIED
  }
}

/**
 * Project a Get/SetMode response onto the local state shape. Keeps the
 * two response handlers in sync: both messages share the same fields,
 * so a divergence here would silently desync the UI.
 */
interface NormalizedModeSnapshot {
  mode?: LogMode
  azureEnabled: boolean
  azureStatus: AzureConnectionStatus
  azureRealtime: boolean
  connectionMessage: string | undefined
}

function normalizeModeResponse(resp: GetModeResponse | SetModeResponse): NormalizedModeSnapshot {
  const enabled = resp.azureEnabled
  // Server is the source of truth for status; if it sends an unknown
  // string we fall back to "disconnected" (enabled-but-broken) or
  // "disabled" (disabled), matching the legacy parseModeApiResponse
  // contract that swallowed bad values.
  let azureStatus: AzureConnectionStatus
  if (isAzureConnectionStatus(resp.azureStatus)) {
    azureStatus = resp.azureStatus
  } else {
    azureStatus = enabled ? 'disconnected' : 'disabled'
  }
  return {
    mode: protoToLogMode(resp.mode),
    azureEnabled: enabled,
    azureStatus: enabled ? azureStatus : 'disabled',
    azureRealtime: resp.azureRealtime,
    connectionMessage: resp.connectionMessage === '' ? undefined : resp.connectionMessage,
  }
}

// =============================================================================
// Hook
// =============================================================================

export interface UseAzureConnectionStatusResult {
  logMode: LogMode
  isModeSwitching: boolean
  azureEnabled: boolean
  azureStatus: AzureConnectionStatus
  azureConnectionMessage: string | undefined
  fetchAzureStatus: () => Promise<void>
  handleLogModeChange: (newMode: LogMode) => Promise<void>
}

export interface UseAzureConnectionStatusOptions {
  onAzureRealtimeConfig?: (azureRealtime: boolean | undefined) => void
  /** Test seam, inject a Connect transport (e.g. createRouterTransport). */
  transport?: Transport
}

export function useAzureConnectionStatus(
  options?: UseAzureConnectionStatusOptions
): UseAzureConnectionStatusResult {
  const transport = options?.transport
  const onAzureRealtimeConfig = options?.onAzureRealtimeConfig

  const client = React.useMemo<Client<typeof ModeService>>(
    () => createModeClient(transport),
    [transport]
  )

  const [logMode, setLogMode] = React.useState<LogMode>('local')
  const [isModeSwitching, setIsModeSwitching] = React.useState(false)
  const [azureEnabled, setAzureEnabled] = React.useState(false)
  const [azureStatus, setAzureStatus] = React.useState<AzureConnectionStatus>('disabled')
  const [azureConnectionMessage, setAzureConnectionMessage] = React.useState<string | undefined>(undefined)

  // Track in-flight requests to prevent concurrent fetches. The
  // AbortController is kept for symmetry with the legacy hook (so
  // cleanup on unmount aborts the call); Connect honours the signal.
  const abortControllerRef = React.useRef<AbortController | null>(null)
  const modeSwitchTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchAzureStatus = React.useCallback(async () => {
    if (abortControllerRef.current) {
      return // Already fetching, skip this request
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    try {
      const resp = await client.getMode(create(GetModeRequestSchema), { signal: controller.signal })
      const snap = normalizeModeResponse(resp)

      if (snap.mode) {
        setLogMode(snap.mode)
      }
      setAzureEnabled(snap.azureEnabled)
      setAzureConnectionMessage(snap.connectionMessage)
      onAzureRealtimeConfig?.(snap.azureRealtime)
      setAzureStatus(snap.azureEnabled ? snap.azureStatus : 'disabled')
    } catch (err) {
      if (controller.signal.aborted) return
      if (err instanceof ConnectError && err.code === Code.Canceled) return
      // Network/server error - log without spamming. Status stays as-is
      // so a transient failure doesn't visually flip the toggle.
      const message = err instanceof Error ? err.message : String(err)
      console.warn(`[useAzureConnectionStatus] Failed to fetch mode: ${message}`)
    } finally {
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [client, onAzureRealtimeConfig])

  // Cleanup: abort any in-flight request and clear timeout on unmount
  React.useEffect(() => {
    return () => {
      abortControllerRef.current?.abort()
      abortControllerRef.current = null
      if (modeSwitchTimeoutRef.current) {
        clearTimeout(modeSwitchTimeoutRef.current)
        modeSwitchTimeoutRef.current = null
      }
    }
  }, [])

  const handleLogModeChange = React.useCallback(
    async (newMode: LogMode) => {
      if (!isLogMode(newMode)) {
        console.error(`[useAzureConnectionStatus] Invalid mode: ${String(newMode)}`)
        return
      }

      // Snapshot current mode synchronously so we don't kick off a
      // SetMode call when nothing would change.
      let currentModeSnapshot: LogMode | null = null
      setLogMode((current) => {
        currentModeSnapshot = current
        return current
      })

      if (!currentModeSnapshot || newMode === currentModeSnapshot) {
        return
      }

      setIsModeSwitching(true)

      if (modeSwitchTimeoutRef.current) {
        clearTimeout(modeSwitchTimeoutRef.current)
        modeSwitchTimeoutRef.current = null
      }

      try {
        const resp = await client.setMode(
          create(SetModeRequestSchema, { mode: logModeToProto(newMode) })
        )
        const snap = normalizeModeResponse(resp)

        // Trust the server's echo of the new mode rather than the
        // requested one; if the server clamps to something else (it
        // doesn't today, but the contract allows it) we follow.
        setLogMode(snap.mode ?? newMode)
        setAzureEnabled(snap.azureEnabled)
        setAzureConnectionMessage(snap.connectionMessage)
        onAzureRealtimeConfig?.(snap.azureRealtime)
        setAzureStatus(snap.azureEnabled ? snap.azureStatus : 'disabled')
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        console.error(
          `[useAzureConnectionStatus] Failed to switch mode to '${newMode}': ${message}`
        )
        // Keep the previous mode on error
      } finally {
        // Clear switching state after a short delay to let panes reconnect
        modeSwitchTimeoutRef.current = setTimeout(() => {
          setIsModeSwitching(false)
          modeSwitchTimeoutRef.current = null
        }, 1500)
      }
    },
    [client, onAzureRealtimeConfig]
  )

  return {
    logMode,
    isModeSwitching,
    azureEnabled,
    azureStatus,
    azureConnectionMessage,
    fetchAzureStatus,
    handleLogModeChange,
  }
}
