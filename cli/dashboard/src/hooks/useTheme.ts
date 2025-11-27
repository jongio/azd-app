import { useState, useEffect } from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'dashboard-theme'

function isValidTheme(value: string | null): value is Theme {
  return value === 'light' || value === 'dark'
}

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    // Initialize from localStorage immediately (SSR-safe)
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (isValidTheme(stored)) {
        return stored
      }
    }
    return 'light'
  })
  const [isMounted, setIsMounted] = useState(false)

  useEffect(() => {
    // Apply theme to document on mount
    document.documentElement.setAttribute('data-theme', theme)
    setIsMounted(true)
  }, [theme])

  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme)
    document.documentElement.setAttribute('data-theme', newTheme)
    localStorage.setItem(STORAGE_KEY, newTheme)
  }

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light'
    setTheme(newTheme)
  }

  return {
    theme,
    setTheme,
    toggleTheme,
    isMounted,
  }
}
