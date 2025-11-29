import { useState, useEffect, useCallback } from 'react'

/**
 * Log classification stored in azure.yaml
 */
export interface LogClassification {
  text: string   // Text to match (case-insensitive)
  level: 'info' | 'warning' | 'error'
}

interface ClassificationsResponse {
  classifications: LogClassification[]
}

// Custom event name for classification changes
const CLASSIFICATIONS_CHANGED_EVENT = 'classificationsChanged'

/**
 * Notify all hook instances that classifications have changed
 */
function notifyClassificationsChanged() {
  window.dispatchEvent(new CustomEvent(CLASSIFICATIONS_CHANGED_EVENT))
}

/**
 * Hook for managing log classifications stored in azure.yaml.
 * 
 * Classifications allow users to override the default log level detection
 * by specifying text patterns and their desired classification level.
 * 
 * Example: "Connection refused" -> error
 *          "cache miss" -> info (downgrade from warning)
 */
export function useLogClassifications() {
  const [classifications, setClassifications] = useState<LogClassification[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const loadClassifications = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      
      const response = await fetch('/api/logs/classifications')
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const data = await response.json() as ClassificationsResponse
      setClassifications(data.classifications || [])
    } catch (err) {
      console.error('Failed to load classifications:', err)
      setError(err instanceof Error ? err : new Error('Failed to load classifications'))
      setClassifications([])
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Load on mount and listen for changes from other hook instances
  useEffect(() => {
    void loadClassifications()

    // Listen for changes from other instances
    const handleChange = () => {
      void loadClassifications()
    }
    
    window.addEventListener(CLASSIFICATIONS_CHANGED_EVENT, handleChange)
    return () => {
      window.removeEventListener(CLASSIFICATIONS_CHANGED_EVENT, handleChange)
    }
  }, [loadClassifications])

  /**
   * Add a new classification.
   * If the text already exists, updates the level instead.
   * @param skipNotify - If true, skip reload and notification (for batch operations)
   */
  const addClassification = useCallback(async (
    text: string, 
    level: 'info' | 'warning' | 'error',
    skipNotify = false
  ): Promise<LogClassification> => {
    const classification: LogClassification = { text, level }
    
    try {
      const response = await fetch('/api/logs/classifications', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(classification)
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(errorText || `HTTP error! status: ${response.status}`)
      }

      // Reload and notify other instances (unless in batch mode)
      if (!skipNotify) {
        await loadClassifications()
        notifyClassificationsChanged()
      }
      
      return classification
    } catch (err) {
      console.error('Failed to add classification:', err)
      throw err
    }
  }, [loadClassifications])

  /**
   * Delete a classification by index.
   * @param skipNotify - If true, skip reload and notification (for batch operations)
   */
  const deleteClassification = useCallback(async (index: number, skipNotify = false): Promise<void> => {
    try {
      const response = await fetch(`/api/logs/classifications/${index}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      // Reload and notify other instances (unless in batch mode)
      if (!skipNotify) {
        await loadClassifications()
        notifyClassificationsChanged()
      }
    } catch (err) {
      console.error('Failed to delete classification:', err)
      throw err
    }
  }, [loadClassifications])

  /**
   * Get the classification level for a given log message.
   * Uses longest-match-wins strategy when multiple classifications match.
   * 
   * @param text The log message to check
   * @returns The classification level, or null if no classification matches
   */
  const getClassificationForText = useCallback((text: string): 'info' | 'warning' | 'error' | null => {
    if (!text || classifications.length === 0) {
      return null
    }

    const lowerText = text.toLowerCase()
    
    // Find all matching classifications
    const matches = classifications.filter(c => 
      lowerText.includes(c.text.toLowerCase())
    )

    if (matches.length === 0) {
      return null
    }

    // Use longest match (most specific)
    const longestMatch = matches.reduce((prev, curr) => 
      curr.text.length > prev.text.length ? curr : prev
    )

    return longestMatch.level
  }, [classifications])

  return {
    classifications,
    isLoading,
    error,
    addClassification,
    deleteClassification,
    getClassificationForText,
    reload: loadClassifications
  }
}
