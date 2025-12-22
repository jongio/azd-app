/**
 * useAzureConnectionStatus - Manages Azure connection status and mode switching
 */
import * as React from 'react'
import type { LogMode } from '@/components/ModeToggle'

// =============================================================================
// Types
// =============================================================================

export type AzureConnectionStatus = 'connected' | 'degraded' | 'disconnected' | 'connecting' | 'disabled'

interface ModeApiResponse {
  mode?: LogMode
  azureEnabled?: boolean
  azureStatus?: AzureConnectionStatus
  azureRealtime?: boolean
  connectionMessage?: string
}

// =============================================================================
// Helpers
// =============================================================================

function isLogMode(value: unknown): value is LogMode {
  return value === 'local' || value === 'azure'
}

function isAzureConnectionStatus(value: unknown): value is AzureConnectionStatus {
  return value === 'connected' || value === 'degraded' || value === 'disconnected' || value === 'connecting' || value === 'disabled'
}

function parseModeApiResponse(value: unknown): ModeApiResponse {
  if (typeof value !== 'object' || value === null) {
    return {}
  }

  const record = value as Record<string, unknown>

  const mode = isLogMode(record.mode) ? record.mode : undefined
  const azureEnabled = typeof record.azureEnabled === 'boolean' ? record.azureEnabled : undefined
  const azureStatus = isAzureConnectionStatus(record.azureStatus) ? record.azureStatus : undefined
  const azureRealtime = typeof record.azureRealtime === 'boolean' ? record.azureRealtime : undefined
  const connectionMessage = typeof record.connectionMessage === 'string' ? record.connectionMessage : undefined

  return { mode, azureEnabled, azureStatus, azureRealtime, connectionMessage }
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
}

export function useAzureConnectionStatus(
  options?: UseAzureConnectionStatusOptions
): UseAzureConnectionStatusResult {
  const [logMode, setLogMode] = React.useState<LogMode>('local')
  const [isModeSwitching, setIsModeSwitching] = React.useState(false)
  const [azureEnabled, setAzureEnabled] = React.useState(false)
  const [azureStatus, setAzureStatus] = React.useState<AzureConnectionStatus>('disabled')
  const [azureConnectionMessage, setAzureConnectionMessage] = React.useState<string | undefined>(undefined)
  
  // Track in-flight requests to prevent concurrent fetches
  const abortControllerRef = React.useRef<AbortController | null>(null)
  // Track mode switch cleanup timeout
  const modeSwitchTimeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchAzureStatus = React.useCallback(async () => {
    // Guard: prevent concurrent requests
    if (abortControllerRef.current) {
      return // Already fetching, skip this request
    }

    const controller = new AbortController()
    abortControllerRef.current = controller

    try {
      const res = await fetch('/api/mode', { signal: controller.signal })
      if (res.ok) {
        let raw: unknown
        try {
          raw = await res.json()
        } catch (error_) {
          console.warn('[useAzureConnectionStatus] Failed to parse mode response:', error_)
          return
        }
        
        const data = parseModeApiResponse(raw)

        // Set the current mode from backend (important for initial page load)
        if (data.mode) {
          setLogMode(data.mode)
        }

        const enabled = data.azureEnabled ?? false
        setAzureEnabled(enabled)
        setAzureConnectionMessage(data.connectionMessage)

        // Notify about default realtime toggle from config
        options?.onAzureRealtimeConfig?.(data.azureRealtime)

        if (enabled) {
          setAzureStatus(data.azureStatus ?? 'disconnected')
        } else {
          setAzureStatus('disabled')
        }
      } else {
        // Non-OK response - backend is up but returned error
        const statusText = res.statusText || 'Unknown error'
        console.warn(`[useAzureConnectionStatus] Failed to fetch mode: ${res.status} ${statusText}`)
      }
    } catch (err) {
      // Ignore abort errors (from cleanup or concurrent request prevention)
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }
      // Network error - backend is likely down, don't spam console
      // Status will remain as-is (disabled or last known state)
    } finally {
      // Clear the abort controller if this is still our request
      if (abortControllerRef.current === controller) {
        abortControllerRef.current = null
      }
    }
  }, [options])

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
      // Validate input
      if (!isLogMode(newMode)) {
        console.error(`[useAzureConnectionStatus] Invalid mode: ${String(newMode)}`)
        return
      }

      // Get current mode synchronously
      let currentModeSnapshot: LogMode | null = null
      setLogMode((current) => {
        currentModeSnapshot = current
        return current
      })

      if (!currentModeSnapshot || newMode === currentModeSnapshot) {
        return // No change needed
      }

      // Start async mode switch operation
      setIsModeSwitching(true)

      // Clear any pending timeout
      if (modeSwitchTimeoutRef.current) {
        clearTimeout(modeSwitchTimeoutRef.current)
        modeSwitchTimeoutRef.current = null
      }

      try {
        // Call backend API to switch mode - this starts/stops Azure polling
        const res = await fetch('/api/mode', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: newMode }),
        })

        if (res.ok) {
          // Success - update mode
          setLogMode(newMode)
          // Refresh Azure status after mode change
          await fetchAzureStatus()
        } else {
          const errorText = await res.text()
          const statusText = res.statusText || 'Unknown error'
          console.error(
            `[useAzureConnectionStatus] Failed to switch mode to '${newMode}': ${res.status} ${statusText}`,
            errorText
          )
          // Keep the previous mode on error
        }
      } catch (err) {
        console.error(`[useAzureConnectionStatus] Error switching mode to '${newMode}':`, err)
        // Keep the previous mode on error
      } finally {
        // Clear switching state after a short delay to let panes reconnect
        modeSwitchTimeoutRef.current = setTimeout(() => {
          setIsModeSwitching(false)
          modeSwitchTimeoutRef.current = null
        }, 1500)
      }
    },
    [fetchAzureStatus]
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
