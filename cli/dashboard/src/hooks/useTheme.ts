import { useState, useEffect, useCallback } from 'react'

export type Theme = 'light' | 'dark'

const DEFAULT_THEME: Theme = 'light'
// Keep localStorage key for fast initial render (prevents flash), but API is source of truth
const STORAGE_KEY = 'dashboard-theme'

function isValidTheme(value: string | null): value is Theme {
  return value === 'light' || value === 'dark'
}

/**
 * Get theme from localStorage for immediate render (prevents flash)
 * This is a cache - the API is the source of truth
 */
function getCachedTheme(): Theme {
  if (typeof window === 'undefined') return DEFAULT_THEME
  const stored = localStorage.getItem(STORAGE_KEY)
  const theme = isValidTheme(stored) ? stored : DEFAULT_THEME
  // Apply dark class immediately on load to prevent flash
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  return theme
}

function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
  // Also toggle the 'dark' class for Tailwind's dark: prefix
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  // Cache in localStorage for fast initial render
  try {
    localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    // localStorage may not be available
  }
}

interface UseThemeOptions {
  /** Theme value from API preferences */
  apiTheme?: Theme
  /** Callback when theme changes (to save via API) */
  onThemeChange?: (theme: Theme) => void
}

export function useTheme(options: UseThemeOptions = {}) {
  const { apiTheme, onThemeChange } = options
  const [theme, setThemeState] = useState<Theme>(getCachedTheme)

  // Sync with API theme when it changes
  // This pattern is intentional: we sync local state with API state as external source of truth
  useEffect(() => {
    if (apiTheme && apiTheme !== theme) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- API is external source of truth
      setThemeState(apiTheme)
      applyTheme(apiTheme)
    }
  }, [apiTheme, theme])

  // Apply theme to document on mount and theme changes
  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  const setTheme = useCallback((newTheme: Theme) => {
    setThemeState(newTheme)
    applyTheme(newTheme)
    // Notify parent to save via API
    onThemeChange?.(newTheme)
  }, [onThemeChange])

  const toggleTheme = useCallback(() => {
    setTheme(theme === 'light' ? 'dark' : 'light')
  }, [theme, setTheme])

  return {
    theme,
    setTheme,
    toggleTheme,
  }
}
