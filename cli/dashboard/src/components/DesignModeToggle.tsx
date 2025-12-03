import { useState, useRef, useEffect } from 'react'
import { Palette, Sparkles, Monitor } from 'lucide-react'
import { useDesignMode, type DesignMode } from '@/hooks/useDesignMode'
import { cn } from '@/lib/utils'

interface DesignModeToggleProps {
  className?: string
  /** Use compact style (icon only) vs full dropdown */
  variant?: 'icon' | 'dropdown'
}

/**
 * Update URL with design mode parameter without page reload
 */
function updateURLWithDesignMode(mode: DesignMode): void {
  const url = new URL(window.location.href)
  url.searchParams.set('design', mode)
  window.history.replaceState({}, '', url.toString())
}

/**
 * Toggle control for switching between Classic and Modern design modes
 */
export function DesignModeToggle({ className, variant = 'icon' }: DesignModeToggleProps) {
  const { designMode, setDesignMode, isModern } = useDesignMode()
  const [isDropdownOpen, setIsDropdownOpen] = useState(false)
  const [announcement, setAnnouncement] = useState('')
  const dropdownRef = useRef<HTMLDivElement>(null)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false)
      }
    }

    if (isDropdownOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isDropdownOpen])

  // Close dropdown on Escape
  useEffect(() => {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isDropdownOpen) {
        setIsDropdownOpen(false)
      }
    }

    if (isDropdownOpen) {
      document.addEventListener('keydown', handleEscape)
      return () => document.removeEventListener('keydown', handleEscape)
    }
  }, [isDropdownOpen])

  const handleModeChange = (mode: DesignMode) => {
    setDesignMode(mode)
    updateURLWithDesignMode(mode)
    setIsDropdownOpen(false)

    // Announce to screen readers
    const modeLabel = mode === 'modern' ? 'Modern' : 'Classic'
    setAnnouncement(`${modeLabel} design mode enabled`)

    // Clear any existing timeout before setting a new one
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }
    timeoutRef.current = setTimeout(() => setAnnouncement(''), 1000)
  }

  const handleToggle = () => {
    const newMode = isModern ? 'classic' : 'modern'
    handleModeChange(newMode)
  }

  const tooltip = isModern ? 'Switch to Classic design' : 'Switch to Modern design'

  // Simple icon toggle variant
  if (variant === 'icon') {
    return (
      <>
        <button
          type="button"
          onClick={handleToggle}
          aria-label={tooltip}
          title={tooltip}
          className={cn(
            'p-2 rounded-md transition-colors',
            'hover:bg-secondary',
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-primary/50 focus-visible:ring-offset-2',
            'active:scale-95',
            className
          )}
        >
          {isModern ? (
            <Sparkles className="w-4 h-4 text-foreground-secondary hover:text-foreground transition-colors" />
          ) : (
            <Monitor className="w-4 h-4 text-foreground-secondary hover:text-foreground transition-colors" />
          )}
        </button>

        {/* Screen reader announcements */}
        <div role="status" aria-live="polite" className="sr-only">
          {announcement}
        </div>
      </>
    )
  }

  // Dropdown variant with both options visible
  return (
    <div ref={dropdownRef} className={cn('relative', className)}>
      <button
        type="button"
        onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        aria-expanded={isDropdownOpen}
        aria-haspopup="listbox"
        aria-label="Design mode"
        title="Change design mode"
        className={cn(
          'flex items-center gap-2 px-3 py-2 rounded-md transition-colors',
          'hover:bg-secondary',
          'focus-visible:outline-none focus-visible:ring-2',
          'focus-visible:ring-primary/50 focus-visible:ring-offset-2',
          'text-sm font-medium',
          'text-foreground-secondary hover:text-foreground'
        )}
      >
        <Palette className="w-4 h-4" />
        <span className="hidden sm:inline capitalize">{designMode}</span>
        <svg
          className={cn(
            'w-3 h-3 transition-transform duration-200',
            isDropdownOpen && 'rotate-180'
          )}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Dropdown Menu */}
      {isDropdownOpen && (
        <div
          role="listbox"
          aria-label="Design mode options"
          className={cn(
            'absolute right-0 top-full mt-1 z-50',
            'w-44 py-1 rounded-lg shadow-lg',
            'bg-popover border border-border',
            'animate-in fade-in-0 zoom-in-95 slide-in-from-top-2',
            'duration-200'
          )}
        >
          <button
            type="button"
            role="option"
            aria-selected={!isModern}
            onClick={() => handleModeChange('classic')}
            className={cn(
              'w-full flex items-center gap-3 px-3 py-2 text-sm',
              'transition-colors',
              !isModern
                ? 'bg-accent text-accent-foreground'
                : 'text-popover-foreground hover:bg-accent hover:text-accent-foreground'
            )}
          >
            <Monitor className="w-4 h-4" />
            <span>Classic</span>
            {!isModern && (
              <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            )}
          </button>
          <button
            type="button"
            role="option"
            aria-selected={isModern}
            onClick={() => handleModeChange('modern')}
            className={cn(
              'w-full flex items-center gap-3 px-3 py-2 text-sm',
              'transition-colors',
              isModern
                ? 'bg-accent text-accent-foreground'
                : 'text-popover-foreground hover:bg-accent hover:text-accent-foreground'
            )}
          >
            <Sparkles className="w-4 h-4" />
            <span>Modern</span>
            {isModern && (
              <svg className="w-4 h-4 ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            )}
          </button>
        </div>
      )}

      {/* Screen reader announcements */}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </div>
  )
}

/**
 * Modern-styled variant of DesignModeToggle for the modern header
 */
export function ModernDesignModeToggle({ className }: { className?: string }) {
  const { setDesignMode, isModern } = useDesignMode()
  const [announcement, setAnnouncement] = useState('')
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleToggle = () => {
    const newMode = isModern ? 'classic' : 'modern'
    setDesignMode(newMode)
    updateURLWithDesignMode(newMode)

    // Announce to screen readers
    const modeLabel = newMode === 'modern' ? 'Modern' : 'Classic'
    setAnnouncement(`${modeLabel} design mode enabled`)

    // Clear any existing timeout before setting a new one
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }
    timeoutRef.current = setTimeout(() => setAnnouncement(''), 1000)
  }

  const tooltip = isModern ? 'Switch to Classic design' : 'Switch to Modern design'

  return (
    <>
      <button
        type="button"
        onClick={handleToggle}
        aria-label={tooltip}
        title={tooltip}
        className={cn(
          'p-2 rounded-lg transition-colors',
          'text-slate-500 dark:text-slate-400',
          'hover:text-slate-700 dark:hover:text-slate-200',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          className
        )}
      >
        {isModern ? (
          <Sparkles className="w-[18px] h-[18px]" />
        ) : (
          <Monitor className="w-[18px] h-[18px]" />
        )}
      </button>

      {/* Screen reader announcements */}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </>
  )
}
