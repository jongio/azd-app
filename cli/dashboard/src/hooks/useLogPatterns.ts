import { useState, useEffect, useCallback } from 'react'
import type { ClassificationOverride } from '@/types'

export interface LogPattern {
  id: string
  name: string
  regex: string
  description: string
  enabled: boolean
  createdAt: string
  source: 'user' | 'app'
}

export interface PatternsFile {
  version: string
  patterns: LogPattern[]
  overrides?: ClassificationOverride[]
}

// Mock data for development when backend isn't running
const MOCK_PATTERNS: LogPattern[] = [
  {
    id: 'pattern-error',
    name: 'Error',
    regex: '\\b(error|fail|exception)\\b',
    description: 'Matches error-related messages',
    enabled: true,
    createdAt: new Date().toISOString(),
    source: 'app'
  },
  {
    id: 'pattern-warning',
    name: 'Warning',
    regex: '\\b(warn|warning|caution)\\b',
    description: 'Matches warning messages',
    enabled: true,
    createdAt: new Date().toISOString(),
    source: 'app'
  }
]

export function useLogPatterns() {
  const [patterns, setPatterns] = useState<LogPattern[]>([])
  const [overrides, setOverrides] = useState<ClassificationOverride[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  const [useMock, setUseMock] = useState(false)

  const loadPatterns = useCallback(async () => {
    try {
      setIsLoading(true)
      const response = await fetch('/api/logs/patterns')
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      const data = await response.json() as PatternsFile | LogPattern[] | null
      
      // Handle null response
      if (data === null || data === undefined) {
        console.warn('Backend returned null, using mock patterns')
        setPatterns(MOCK_PATTERNS)
        setOverrides([])
        setUseMock(true)
        setError(null)
        return
      }
      
      const patterns = Array.isArray(data) ? data : (data.patterns || [])
      const overrides = Array.isArray(data) ? [] : (data.overrides || [])
      setPatterns(patterns)
      setOverrides(overrides)
      setUseMock(false)
      setError(null)
    } catch (err) {
      console.error('Failed to load patterns:', err)
      console.warn('Backend not available, using mock patterns')
      setPatterns(MOCK_PATTERNS)
      setOverrides([])
      setUseMock(true)
      setError(null) // Don't show error when using mock data
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadPatterns()
  }, [loadPatterns])

  const addPattern = useCallback(async (pattern: Omit<LogPattern, 'id' | 'createdAt'>) => {
    try {
      // If using mock data, just add locally
      if (useMock) {
        const newPattern: LogPattern = {
          ...pattern,
          id: `pattern-${Date.now()}`,
          createdAt: new Date().toISOString()
        }
        setPatterns(prev => [...prev, newPattern])
        return newPattern
      }

      const newPattern: LogPattern = {
        ...pattern,
        id: `pattern-${Date.now()}`,
        createdAt: new Date().toISOString()
      }

      const response = await fetch('/api/logs/patterns', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pattern: newPattern, level: pattern.source })
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      await loadPatterns()
      return newPattern
    } catch (err) {
      console.error('Failed to add pattern:', err)
      throw err
    }
  }, [loadPatterns, useMock])

  const updatePattern = useCallback(async (id: string, updates: Partial<LogPattern>) => {
    try {
      // If using mock data, just update locally
      if (useMock) {
        setPatterns(prev => prev.map(p => p.id === id ? { ...p, ...updates } : p))
        return
      }

      const response = await fetch(`/api/logs/patterns/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates)
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      await loadPatterns()
    } catch (err) {
      console.error('Failed to update pattern:', err)
      throw err
    }
  }, [loadPatterns, useMock])

  const deletePattern = useCallback(async (id: string) => {
    try {
      // If using mock data, just delete locally
      if (useMock) {
        setPatterns(prev => prev.filter(p => p.id !== id))
        return
      }

      const response = await fetch(`/api/logs/patterns/${id}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      await loadPatterns()
    } catch (err) {
      console.error('Failed to delete pattern:', err)
      throw err
    }
  }, [loadPatterns, useMock])

  const testPattern = useCallback((regex: string, testString: string): boolean => {
    try {
      const pattern = new RegExp(regex, 'i')
      return pattern.test(testString)
    } catch {
      return false
    }
  }, [])

  return {
    patterns,
    overrides,
    isLoading,
    error,
    addPattern,
    updatePattern,
    deletePattern,
    testPattern,
    reload: loadPatterns
  }
}
