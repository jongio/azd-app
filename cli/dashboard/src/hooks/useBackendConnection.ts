import { createContext, useContext } from 'react'

/**
 * Backend connection state shared across the application.
 * This tracks whether the dashboard has connectivity to the backend server.
 */
export interface BackendConnectionState {
  /** Whether the backend connection is active */
  connected: boolean
  /** Error message if disconnected */
  error: string | null
}

/**
 * Context for sharing backend connection state across the application.
 * Used to prevent log fetching and other API calls when the backend is unavailable.
 */
export const BackendConnectionContext = createContext<BackendConnectionState>({
  connected: true,
  error: null,
})

/**
 * Hook to access the global backend connection state.
 * Returns whether the dashboard is connected to the backend server.
 * 
 * Components should check this before making API calls or establishing WebSocket connections.
 */
export function useBackendConnection(): BackendConnectionState {
  return useContext(BackendConnectionContext)
}
