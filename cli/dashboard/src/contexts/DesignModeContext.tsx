import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'

export type DesignMode = 'classic' | 'modern'

interface DesignModeContextValue {
  designMode: DesignMode
  setDesignMode: (mode: DesignMode) => void
  isClassic: boolean
  isModern: boolean
}

const DesignModeContext = createContext<DesignModeContextValue | undefined>(undefined)

const STORAGE_KEY = 'dashboard-design-mode'
const URL_PARAM = 'design'
const DEFAULT_MODE: DesignMode = 'modern'

/**
 * Parse design mode from URL parameter
 * @returns The design mode from URL or null if not present/invalid
 */
function getDesignModeFromURL(): DesignMode | null {
  const params = new URLSearchParams(window.location.search)
  const mode = params.get(URL_PARAM)
  if (mode === 'classic' || mode === 'modern') {
    return mode
  }
  return null
}

/**
 * Get design mode from localStorage
 * @returns The stored design mode or null if not present/invalid
 */
function getDesignModeFromStorage(): DesignMode | null {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'classic' || stored === 'modern') {
      return stored
    }
  } catch {
    // localStorage may not be available
  }
  return null
}

/**
 * Save design mode to localStorage
 */
function saveDesignModeToStorage(mode: DesignMode): void {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // localStorage may not be available
  }
}

/**
 * Determine the initial design mode
 * Priority: URL parameter > localStorage > default ('modern')
 */
function getInitialDesignMode(): DesignMode {
  // URL parameter takes precedence
  const urlMode = getDesignModeFromURL()
  if (urlMode) {
    // Save URL preference to localStorage for persistence
    saveDesignModeToStorage(urlMode)
    return urlMode
  }
  
  // Fall back to localStorage
  const storedMode = getDesignModeFromStorage()
  if (storedMode) {
    return storedMode
  }
  
  // Default to 'modern'
  return DEFAULT_MODE
}

interface DesignModeProviderProps {
  children: ReactNode
}

export function DesignModeProvider({ children }: DesignModeProviderProps) {
  const [designMode, setDesignModeState] = useState<DesignMode>(getInitialDesignMode)

  // Update design mode and persist to localStorage
  const setDesignMode = useCallback((mode: DesignMode) => {
    setDesignModeState(mode)
    saveDesignModeToStorage(mode)
  }, [])

  // Sync with URL parameter changes (e.g., browser back/forward)
  useEffect(() => {
    const handlePopState = () => {
      const urlMode = getDesignModeFromURL()
      if (urlMode && urlMode !== designMode) {
        setDesignModeState(urlMode)
        saveDesignModeToStorage(urlMode)
      }
    }

    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [designMode])

  const value: DesignModeContextValue = {
    designMode,
    setDesignMode,
    isClassic: designMode === 'classic',
    isModern: designMode === 'modern',
  }

  return (
    <DesignModeContext.Provider value={value}>
      {children}
    </DesignModeContext.Provider>
  )
}

/**
 * Hook to access the design mode context
 * @throws Error if used outside of DesignModeProvider
 */
export function useDesignModeContext(): DesignModeContextValue {
  const context = useContext(DesignModeContext)
  if (context === undefined) {
    throw new Error('useDesignModeContext must be used within a DesignModeProvider')
  }
  return context
}

export { DesignModeContext }
