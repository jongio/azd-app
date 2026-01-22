/**
 * Theme Provider for Azure YAML Editor
 * Task 21: Visual Design and Styling
 * 
 * Manages theme state (light/dark mode) with:
 * - System preference detection (prefers-color-scheme)
 * - Manual override capability
 * - LocalStorage persistence
 * - Smooth transitions between modes
 */

/* eslint-disable react-refresh/only-export-components */
import * as React from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

interface ThemeContextValue {
  /** Current theme setting (includes 'system' option) */
  theme: Theme
  /** Resolved theme (actual light/dark value) */
  resolvedTheme: ResolvedTheme
  /** Set theme (light/dark/system) */
  setTheme: (theme: Theme) => void
  /** Toggle between light and dark */
  toggleTheme: () => void
}

const ThemeContext = React.createContext<ThemeContextValue | undefined>(undefined)

const STORAGE_KEY = 'azure-yaml-editor-theme'

/**
 * Get system color scheme preference
 */
function getSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

/**
 * Get initial theme from localStorage or system preference
 */
function getInitialTheme(storageKey: string = STORAGE_KEY): Theme {
  if (typeof window === 'undefined') return 'system'
  
  try {
    const stored = localStorage.getItem(storageKey) as Theme | null
    if (stored && ['light', 'dark', 'system'].includes(stored)) {
      return stored
    }
  } catch {
    // Silently fail - theme is not critical
  }
  
  return 'system'
}

/**
 * Resolve theme to actual light/dark value
 */
function resolveTheme(theme: Theme): ResolvedTheme {
  if (theme === 'system') {
    return getSystemTheme()
  }
  return theme
}

export interface ThemeProviderProps {
  children: React.ReactNode
  /** Default theme if no preference is stored */
  defaultTheme?: Theme
  /** Storage key for localStorage (default: 'azure-yaml-editor-theme') */
  storageKey?: string
}

/**
 * Theme Provider Component
 * 
 * Wraps the editor application and provides theme context.
 * 
 * @example
 * ```tsx
 * function App() {
 *   return (
 *     <ThemeProvider defaultTheme="system">
 *       <YamlEditor />
 *     </ThemeProvider>
 *   )
 * }
 * ```
 */
export function ThemeProvider({
  children,
  storageKey = STORAGE_KEY,
}: ThemeProviderProps) {
  const [theme, setThemeState] = React.useState<Theme>(() => getInitialTheme(storageKey))
  const [resolvedTheme, setResolvedTheme] = React.useState<ResolvedTheme>(() =>
    resolveTheme(getInitialTheme(storageKey))
  )

  // Apply theme to document
  const applyTheme = React.useCallback((newResolvedTheme: ResolvedTheme) => {
    const root = document.documentElement
    root.classList.remove('light', 'dark')
    root.classList.add(newResolvedTheme)
    root.setAttribute('data-theme', newResolvedTheme)
    root.style.colorScheme = newResolvedTheme
  }, [])

  // Listen for system theme changes
  React.useEffect(() => {
    if (theme !== 'system') return

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    
    const handleChange = (e: MediaQueryListEvent) => {
      const newResolvedTheme = e.matches ? 'dark' : 'light'
      setResolvedTheme(newResolvedTheme)
      applyTheme(newResolvedTheme)
    }

    // Modern browsers
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handleChange)
      return () => mediaQuery.removeEventListener('change', handleChange)
    } else {
      // Older browsers - deprecated API
      mediaQuery.addListener(handleChange)
      return () => mediaQuery.removeListener(handleChange)
    }
  }, [theme, applyTheme])

  // Set theme and persist to localStorage
  const setTheme = React.useCallback(
    (newTheme: Theme) => {
      setThemeState(newTheme)
      
      const newResolvedTheme = resolveTheme(newTheme)
      setResolvedTheme(newResolvedTheme)
      applyTheme(newResolvedTheme)
      
      // Persist to localStorage
      try {
        localStorage.setItem(storageKey, newTheme)
      } catch {
        // Silent failure - theme will still work, just won't persist
      }
    },
    [storageKey, applyTheme]
  )

  // Toggle between light and dark (ignoring system preference)
  const toggleTheme = React.useCallback(() => {
    const newTheme = resolvedTheme === 'light' ? 'dark' : 'light'
    setTheme(newTheme)
  }, [resolvedTheme, setTheme])

  // Initialize theme on mount
  React.useEffect(() => {
    applyTheme(resolvedTheme)
  }, [applyTheme, resolvedTheme])

  const value = React.useMemo(
    () => ({
      theme,
      resolvedTheme,
      setTheme,
      toggleTheme,
    }),
    [theme, resolvedTheme, setTheme, toggleTheme]
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

/**
 * Hook to access theme context
 * 
 * @example
 * ```tsx
 * function ThemeToggle() {
 *   const { resolvedTheme, toggleTheme } = useTheme()
 *   return (
 *     <button onClick={toggleTheme}>
 *       {resolvedTheme === 'light' ? '🌙' : '☀️'}
 *     </button>
 *   )
 * }
 * ```
 */
export function useTheme() {
  const context = React.useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

/**
 * Theme Toggle Button Component
 * 
 * Pre-built button for toggling theme.
 */
export interface ThemeToggleProps {
  className?: string
  showLabel?: boolean
}

export function ThemeToggle({ className, showLabel = false }: ThemeToggleProps) {
  const { resolvedTheme, toggleTheme } = useTheme()

  return (
    <button
      onClick={toggleTheme}
      className={className}
      aria-label={`Switch to ${resolvedTheme === 'light' ? 'dark' : 'light'} mode`}
      title={`Switch to ${resolvedTheme === 'light' ? 'dark' : 'light'} mode`}
    >
      {resolvedTheme === 'light' ? (
        <>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
          </svg>
          {showLabel && <span className="ml-2">Dark Mode</span>}
        </>
      ) : (
        <>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <circle cx="12" cy="12" r="5" />
            <line x1="12" y1="1" x2="12" y2="3" />
            <line x1="12" y1="21" x2="12" y2="23" />
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
            <line x1="1" y1="12" x2="3" y2="12" />
            <line x1="21" y1="12" x2="23" y2="12" />
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
          </svg>
          {showLabel && <span className="ml-2">Light Mode</span>}
        </>
      )}
    </button>
  )
}

