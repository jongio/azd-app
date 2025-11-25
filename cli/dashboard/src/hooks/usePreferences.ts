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

export function usePreferences() {
  const [preferences, setPreferences] = useState<UserPreferences>(DEFAULT_PREFERENCES)
  const [isLoading, setIsLoading] = useState(true)

  const loadPreferences = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/logs/preferences')
      if (response.ok) {
        const data = await response.json() as UserPreferences
        setPreferences({ ...DEFAULT_PREFERENCES, ...data })
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
