import { useState, useEffect, useCallback } from 'react'

export interface UserPreferences {
  version: string
  ui: {
    gridColumns: number
    viewMode: 'grid' | 'unified'
    selectedServices: string[]
  }
  behavior: {
    autoScroll: boolean
    pauseOnScroll: boolean
    timestampFormat: string
  }
  copy: {
    defaultFormat: 'plaintext' | 'json' | 'markdown' | 'csv'
    includeTimestamp: boolean
    includeService: boolean
  }
}

const DEFAULT_PREFERENCES: UserPreferences = {
  version: '1.0',
  ui: {
    gridColumns: 2,
    viewMode: 'grid',
    selectedServices: []
  },
  behavior: {
    autoScroll: true,
    pauseOnScroll: true,
    timestampFormat: 'hh:mm:ss.sss'
  },
  copy: {
    defaultFormat: 'plaintext',
    includeTimestamp: true,
    includeService: true
  }
}

/**
 * Type guard to check if a value is a valid view mode.
 */
function isValidViewMode(value: unknown): value is 'grid' | 'unified' {
  return value === 'grid' || value === 'unified'
}

/**
 * Type guard to check if a value is a valid copy format.
 */
function isValidCopyFormat(value: unknown): value is 'plaintext' | 'json' | 'markdown' | 'csv' {
  return value === 'plaintext' || value === 'json' || value === 'markdown' || value === 'csv'
}

/**
 * Validates and sanitizes preferences data from the API.
 * Returns a valid UserPreferences object with defaults for any invalid/missing fields.
 */
function validatePreferences(data: unknown): UserPreferences {
  if (typeof data !== 'object' || data === null) {
    return DEFAULT_PREFERENCES
  }

  const raw = data as Record<string, unknown>

  // Validate UI preferences
  const rawUI = typeof raw.ui === 'object' && raw.ui !== null 
    ? raw.ui as Record<string, unknown> 
    : {}
  
  const ui: UserPreferences['ui'] = {
    gridColumns: typeof rawUI.gridColumns === 'number' && rawUI.gridColumns >= 1 && rawUI.gridColumns <= 6
      ? rawUI.gridColumns
      : DEFAULT_PREFERENCES.ui.gridColumns,
    viewMode: isValidViewMode(rawUI.viewMode)
      ? rawUI.viewMode
      : DEFAULT_PREFERENCES.ui.viewMode,
    selectedServices: Array.isArray(rawUI.selectedServices) && 
      rawUI.selectedServices.every((s): s is string => typeof s === 'string')
      ? rawUI.selectedServices
      : DEFAULT_PREFERENCES.ui.selectedServices
  }

  // Validate behavior preferences
  const rawBehavior = typeof raw.behavior === 'object' && raw.behavior !== null
    ? raw.behavior as Record<string, unknown>
    : {}
  
  const behavior: UserPreferences['behavior'] = {
    autoScroll: typeof rawBehavior.autoScroll === 'boolean'
      ? rawBehavior.autoScroll
      : DEFAULT_PREFERENCES.behavior.autoScroll,
    pauseOnScroll: typeof rawBehavior.pauseOnScroll === 'boolean'
      ? rawBehavior.pauseOnScroll
      : DEFAULT_PREFERENCES.behavior.pauseOnScroll,
    timestampFormat: typeof rawBehavior.timestampFormat === 'string'
      ? rawBehavior.timestampFormat
      : DEFAULT_PREFERENCES.behavior.timestampFormat
  }

  // Validate copy preferences
  const rawCopy = typeof raw.copy === 'object' && raw.copy !== null
    ? raw.copy as Record<string, unknown>
    : {}
  
  const copy: UserPreferences['copy'] = {
    defaultFormat: isValidCopyFormat(rawCopy.defaultFormat)
      ? rawCopy.defaultFormat
      : DEFAULT_PREFERENCES.copy.defaultFormat,
    includeTimestamp: typeof rawCopy.includeTimestamp === 'boolean'
      ? rawCopy.includeTimestamp
      : DEFAULT_PREFERENCES.copy.includeTimestamp,
    includeService: typeof rawCopy.includeService === 'boolean'
      ? rawCopy.includeService
      : DEFAULT_PREFERENCES.copy.includeService
  }

  return {
    version: typeof raw.version === 'string' ? raw.version : DEFAULT_PREFERENCES.version,
    ui,
    behavior,
    copy
  }
}

export function usePreferences() {
  const [preferences, setPreferences] = useState<UserPreferences>(DEFAULT_PREFERENCES)
  const [isLoading, setIsLoading] = useState(true)

  const loadPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/logs/preferences')
      if (response.ok) {
        const data: unknown = await response.json()
        const validatedPrefs = validatePreferences(data)
        setPreferences(validatedPrefs)
      } else {
        setPreferences(DEFAULT_PREFERENCES)
      }
    } catch (err) {
      console.error('Failed to load preferences:', err)
      setPreferences(DEFAULT_PREFERENCES)
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadPreferences()
  }, [loadPreferences])

  const savePreferences = useCallback(async (updates: Partial<UserPreferences>) => {
    try {
      const updated = { ...preferences, ...updates }
      setPreferences(updated)

      await fetch('/api/logs/preferences', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated)
      })
    } catch (err) {
      console.error('Failed to save preferences:', err)
    }
  }, [preferences])

  const updateUI = useCallback((updates: Partial<UserPreferences['ui']>) => {
    const updated = { ...preferences, ui: { ...preferences.ui, ...updates } }
    void savePreferences(updated)
  }, [preferences, savePreferences])

  return {
    preferences,
    isLoading,
    savePreferences,
    updateUI,
    reload: loadPreferences
  }
}
