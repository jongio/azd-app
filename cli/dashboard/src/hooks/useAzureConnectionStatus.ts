/**
 * useAzureConnectionStatus - Manages Azure connection status and mode switching
 */
import * as React from 'react'
import type { LogMode } from '@/components/ModeToggle'

// =============================================================================
// Types
// =============================================================================

export type AzureConnectionStatus = 'connected' | 'disconnected' | 'connecting' | 'disabled'

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
  return value === 'connected' || value === 'disconnected' || value === 'connecting' || value === 'disabled'
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
  setLogMode: (mode: LogMode) => void
  isModeSwitching: boolean
  azureEnabled: boolean
  azureStatus: AzureConnectionStatus
  azureConnectionMessage: string | undefined
  fetchAzureStatus: () => Promise<void>
  handleLogModeChange: (newMode: LogMode) => void
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

  const fetchAzureStatus = React.useCallback(async () => {
    try {
      const res = await fetch('/api/mode')
      if (res.ok) {
        const raw: unknown = await res.json()
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
      }
    } catch {
      // Ignore errors - status will remain disabled
    }
  }, [options])

  const handleLogModeChange = React.useCallback((newMode: LogMode) => {
    void (async () => {
      if (newMode === logMode) {
        return
      }

      setIsModeSwitching(true)

      try {
        // Call backend API to switch mode - this starts/stops Azure polling
        const res = await fetch('/api/mode', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ mode: newMode }),
        })

        if (res.ok) {
          setLogMode(newMode)
          // Refresh Azure status after mode change
          await fetchAzureStatus()
        } else {
          const errorText = await res.text()
          console.error('[useAzureConnectionStatus] Failed to switch mode:', errorText)
        }
      } catch (err) {
        console.error('Error switching mode:', err)
      } finally {
        // Clear switching state after a short delay to let panes reconnect
        setTimeout(() => setIsModeSwitching(false), 1500)
      }
    })()
  }, [logMode, fetchAzureStatus])

  return {
    logMode,
    setLogMode,
    isModeSwitching,
    azureEnabled,
    azureStatus,
    azureConnectionMessage,
    fetchAzureStatus,
    handleLogModeChange,
  }
}
