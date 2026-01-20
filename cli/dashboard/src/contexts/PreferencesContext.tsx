/* eslint-disable react-refresh/only-export-components */

import { createContext, useContext, type ReactNode } from 'react'
import { usePreferences, type Theme, type UsePreferencesReturn } from '@/hooks/usePreferences'

type PreferencesContextValue = UsePreferencesReturn

const PreferencesContext = createContext<PreferencesContextValue | undefined>(undefined)

interface PreferencesProviderProps {
  children: ReactNode
}

/**
 * Provides user preferences from the API
 * This ensures all components have access to API-backed preferences
 */
export function PreferencesProvider({ children }: PreferencesProviderProps) {
  const preferencesHook = usePreferences()

  return (
    <PreferencesContext.Provider value={preferencesHook}>
      {children}
    </PreferencesContext.Provider>
  )
}

/**
 * Hook to access the preferences context
 * @throws Error if used outside of PreferencesProvider
 */
function usePreferencesContext(): PreferencesContextValue {
  const context = useContext(PreferencesContext)
  if (context === undefined) {
    throw new Error('usePreferencesContext must be used within a PreferencesProvider')
  }
  return context
}

// Named exports to satisfy Fast Refresh
export { PreferencesProvider, usePreferencesContext }

// Re-export types for convenience
export type { Theme }
