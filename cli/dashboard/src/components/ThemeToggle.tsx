/**
 * ThemeToggle - Theme toggle component with modern styling
 * Implements light/dark mode switching with smooth transitions
 */
import * as React from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme, type Theme } from '@/hooks/useTheme'
import { usePreferencesContext } from '@/contexts/PreferencesContext'
import { cn } from '@/lib/utils'

// =============================================================================
// Types
// =============================================================================

export interface ThemeToggleProps {
  /** Additional class names */
  className?: string
  /** Callback when theme changes */
  onThemeChange?: (theme: Theme) => void
}

// =============================================================================
// ThemeToggle Component
// =============================================================================

export function ThemeToggle({ className, onThemeChange }: ThemeToggleProps) {
  const { preferences, setTheme: saveThemeToAPI } = usePreferencesContext()
  const { theme, toggleTheme } = useTheme({
    apiTheme: preferences.theme,
    onThemeChange: saveThemeToAPI
  })
  const [announcement, setAnnouncement] = React.useState('')
  const timeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)

  // Cleanup timer on unmount
  React.useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleToggle = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light'
    toggleTheme()
    
    // Announce to screen readers
    setAnnouncement(`${newTheme === 'light' ? 'Light' : 'Dark'} mode enabled`)
    
    // Clear any existing timeout before setting a new one
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }
    timeoutRef.current = setTimeout(() => setAnnouncement(''), 1000)
    
    // Callback
    onThemeChange?.(newTheme)
  }

  const isDark = theme === 'dark'
  const label = isDark ? 'Switch to light mode' : 'Switch to dark mode'

  return (
    <>
      <button
        type="button"
        onClick={handleToggle}
        aria-label={label}
        title={label}
        className={cn(
          'relative p-2 rounded-lg',
          'transition-all duration-200 ease-out',
          'text-slate-500 dark:text-slate-400',
          'hover:text-slate-700 dark:hover:text-slate-200',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'focus-visible:outline-none focus-visible:ring-2',
          'focus-visible:ring-cyan-500 focus-visible:ring-offset-2',
          'dark:focus-visible:ring-offset-slate-900',
          'active:scale-95',
          className
        )}
      >
        <div className="relative w-[18px] h-[18px]">
          {/* Sun Icon */}
          <Sun 
            className={cn(
              'absolute inset-0 w-[18px] h-[18px]',
              'transition-all duration-300',
              isDark 
                ? 'opacity-100 rotate-0 scale-100' 
                : 'opacity-0 rotate-90 scale-0'
            )} 
          />
          {/* Moon Icon */}
          <Moon 
            className={cn(
              'absolute inset-0 w-[18px] h-[18px]',
              'transition-all duration-300',
              isDark 
                ? 'opacity-0 -rotate-90 scale-0' 
                : 'opacity-100 rotate-0 scale-100'
            )} 
          />
        </div>
      </button>
      
      {/* Screen reader announcements */}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </>
  )
}
