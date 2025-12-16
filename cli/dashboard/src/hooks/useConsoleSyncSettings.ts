/**
 * useConsoleSyncSettings - Manages Azure sync interval and realtime settings
 */
import * as React from 'react'
import { UI_CONSTANTS } from '@/lib/constants'

// =============================================================================
// Constants
// =============================================================================

const MIN_SYNC_INTERVAL = UI_CONSTANTS.MIN_SYNC_INTERVAL
const MAX_SYNC_INTERVAL = UI_CONSTANTS.MAX_SYNC_INTERVAL

// =============================================================================
// Storage Helpers
// =============================================================================

function clampSyncInterval(value: number): number {
  if (!Number.isFinite(value)) return MIN_SYNC_INTERVAL
  return Math.min(MAX_SYNC_INTERVAL, Math.max(MIN_SYNC_INTERVAL, value))
}

function getSavedSyncInterval(): number {
  if (globalThis.localStorage === undefined) {
    return 30000
  }

  try {
    const saved = Number(globalThis.localStorage.getItem('logs-sync-interval'))
    return clampSyncInterval(Number.isFinite(saved) ? saved : 30000)
  } catch {
    return 30000
  }
}

function setSavedSyncInterval(interval: number): void {
  if (globalThis.localStorage === undefined) {
    return
  }

  try {
    globalThis.localStorage.setItem('logs-sync-interval', String(interval))
  } catch {
    // Ignore persistence failures
  }
}

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
  syncInterval: number
  setSyncInterval: (interval: number) => void
  azureRealtime: boolean
  setAzureRealtime: (enabled: boolean) => void
  maybeInitializeAzureRealtimeFromConfig: (azureRealtimeFromConfig: boolean | undefined) => void
}

export function useConsoleSyncSettings(): UseConsoleSyncSettingsResult {
  const [syncInterval, setSyncIntervalState] = React.useState<number>(() => getSavedSyncInterval())
  const [azureRealtime, setAzureRealtime] = React.useState<boolean>(() => getSavedAzureRealtime())
  const azureRealtimeInitializedRef = React.useRef(false)

  const setSyncInterval = React.useCallback((value: number) => {
    const clamped = clampSyncInterval(value)
    setSyncIntervalState(clamped)
    setSavedSyncInterval(clamped)
  }, [])

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
    syncInterval,
    setSyncInterval,
    azureRealtime,
    setAzureRealtime,
    maybeInitializeAzureRealtimeFromConfig,
  }
}
