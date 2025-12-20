/**
 * useConsoleSyncSettings - Manages Azure realtime settings
 */
import * as React from 'react'

// =============================================================================
// Storage Helpers
// =============================================================================

function getSavedAzureRealtime(): boolean {
  if (globalThis.localStorage === undefined) {
    return false
  }

  try {
    return globalThis.localStorage.getItem('azure-logs-realtime') === 'true'
  } catch {
    return false
  }
}

// =============================================================================
// Hook
// =============================================================================

export interface UseConsoleSyncSettingsResult {
  azureRealtime: boolean
  setAzureRealtime: (enabled: boolean) => void
  maybeInitializeAzureRealtimeFromConfig: (azureRealtimeFromConfig: boolean | undefined) => void
}

export function useConsoleSyncSettings(): UseConsoleSyncSettingsResult {
  const [azureRealtime, setAzureRealtime] = React.useState<boolean>(() => getSavedAzureRealtime())
  const azureRealtimeInitializedRef = React.useRef(false)

  const maybeInitializeAzureRealtimeFromConfig = React.useCallback((azureRealtimeFromConfig: boolean | undefined) => {
    if (azureRealtimeInitializedRef.current) {
      return
    }

    azureRealtimeInitializedRef.current = true

    try {
      const hasSavedPreference = globalThis.localStorage?.getItem('azure-logs-realtime') !== null
      if (!hasSavedPreference && typeof azureRealtimeFromConfig === 'boolean') {
        setAzureRealtime(azureRealtimeFromConfig)
      }
    } catch {
      // Ignore localStorage errors
    }
  }, [])

  return {
    azureRealtime,
    setAzureRealtime,
    maybeInitializeAzureRealtimeFromConfig,
  }
}
