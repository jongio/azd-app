import { useState, useEffect, useCallback } from 'react'
import type { ClassificationOverride } from '@/types'

export function useLogClassification() {
  const [overrides, setOverrides] = useState<ClassificationOverride[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [useMock, setUseMock] = useState(false)

  const loadOverrides = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/logs/patterns')
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      const data = await response.json() as { patterns?: unknown[], overrides?: ClassificationOverride[] } | null
      
      if (data === null || data === undefined) {
        setOverrides([])
        setUseMock(true)
        return
      }
      
      const loadedOverrides: ClassificationOverride[] = data.overrides || []
      setOverrides(loadedOverrides)
      setUseMock(false)
    } catch (err) {
      console.error('Failed to load classification overrides:', err)
      setOverrides([])
      setUseMock(true)
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadOverrides()
  }, [loadOverrides])

  const addOverride = useCallback(async (text: string, level: 'info' | 'warning' | 'error'): Promise<ClassificationOverride> => {
    const newOverride: ClassificationOverride = {
      id: `override-${Date.now()}`,
      text,
      level,
      createdAt: new Date().toISOString()
    }

    try {
      // If using mock data, just add locally
      if (useMock) {
        setOverrides(currentOverrides => {
          const updated: ClassificationOverride[] = [...currentOverrides, newOverride]
          return updated
        })
        return newOverride
      }

      const response = await fetch('/api/logs/patterns/overrides', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newOverride)
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      await loadOverrides()
      return newOverride
    } catch (err) {
      console.error('Failed to add classification override:', err)
      throw err
    }
  }, [loadOverrides, useMock])

  const deleteOverride = useCallback(async (id: string) => {
    try {
      // If using mock data, just delete locally
      if (useMock) {
        setOverrides(currentOverrides => {
          const updated: ClassificationOverride[] = currentOverrides.filter((item: ClassificationOverride) => item.id !== id)
          return updated
        })
        return
      }

      const response = await fetch(`/api/logs/patterns/overrides/${id}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      await loadOverrides()
    } catch (err) {
      console.error('Failed to delete classification override:', err)
      throw err
    }
  }, [loadOverrides, useMock])

  const getClassificationForText = useCallback((text: string): 'info' | 'warning' | 'error' | null => {
    // Find the most specific (longest) matching override
    const typedOverrides: ClassificationOverride[] = overrides
    const matches: ClassificationOverride[] = typedOverrides
      .filter((item: ClassificationOverride) => text.includes(item.text))
      .sort((a: ClassificationOverride, b: ClassificationOverride) => b.text.length - a.text.length)
    
    if (matches.length > 0) {
      const firstMatch: ClassificationOverride | undefined = matches[0]
      return firstMatch ? firstMatch.level : null
    }
    return null
  }, [overrides])

  return {
    overrides,
    isLoading,
    addOverride,
    deleteOverride,
    getClassificationForText,
    reload: loadOverrides
  }
}
