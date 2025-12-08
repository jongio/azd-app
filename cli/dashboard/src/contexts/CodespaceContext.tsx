/**
 * Context for sharing Codespace environment across components.
 */
import { createContext, useContext, type ReactNode } from 'react'
import { useCodespaceEnv, type UseCodespaceEnvReturn } from '@/hooks/useCodespaceEnv'

const CodespaceContext = createContext<UseCodespaceEnvReturn | null>(null)

export interface CodespaceProviderProps {
  children: ReactNode
}

/**
 * Provider component for sharing Codespace environment across the app.
 * Wrap your app with this to avoid multiple API calls.
 */
export function CodespaceProvider({ children }: CodespaceProviderProps) {
  const value = useCodespaceEnv()
  return (
    <CodespaceContext.Provider value={value}>
      {children}
    </CodespaceContext.Provider>
  )
}

/**
 * Hook to consume Codespace context.
 * Must be used within a CodespaceProvider.
 */
// eslint-disable-next-line react-refresh/only-export-components -- Hooks are conventionally co-located with their providers
export function useCodespaceContext(): UseCodespaceEnvReturn {
  const context = useContext(CodespaceContext)
  if (!context) {
    throw new Error('useCodespaceContext must be used within a CodespaceProvider')
  }
  return context
}
