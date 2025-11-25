import { useState } from 'react'
import { Sun, Moon } from 'lucide-react'
import { useTheme } from '@/hooks/useTheme'
import { cn } from '@/lib/utils'

interface ThemeToggleProps {
  className?: string
  onThemeChange?: (theme: 'light' | 'dark') => void
}

export function ThemeToggle({ className, onThemeChange }: ThemeToggleProps) {
  const { theme, toggleTheme, isMounted } = useTheme()
  const [announcement, setAnnouncement] = useState('')

  const handleToggle = () => {
    toggleTheme()
    
    // Announce to screen readers
    const newTheme = theme === 'light' ? 'dark' : 'light'
    setAnnouncement(`${newTheme === 'light' ? 'Light' : 'Dark'} mode enabled`)
    setTimeout(() => setAnnouncement(''), 1000)
    
    // Callback
    onThemeChange?.(newTheme)
  }

  // Prevent SSR flash
  if (!isMounted) {
    return null
  }

  const Icon = theme === 'light' ? Sun : Moon
  const label = theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'

  return (
    <>
      <button
        type="button"
        onClick={handleToggle}
        aria-label={label}
        aria-pressed={theme === 'dark'}
        className={cn(
          'p-2 rounded-md transition-colors',
          'hover:bg-secondary',
          'focus-visible:outline-none focus-visible:ring-2',
          'focus-visible:ring-primary/50 focus-visible:ring-offset-2',
          'active:scale-95',
          className
        )}
      >
        <Icon className="w-4 h-4 text-foreground-secondary hover:text-foreground transition-colors" />
      </button>
      
      {/* Screen reader announcements */}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </>
  )
}
