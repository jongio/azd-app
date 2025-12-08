/**
 * ModeToggle - Log source mode toggle component
 * Switches between local and Azure log sources
 */
import * as React from 'react'
import { Monitor, Cloud, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

// =============================================================================
// Types
// =============================================================================

export type LogMode = 'local' | 'azure'

export interface ModeToggleProps {
  /** Current log source mode */
  mode: LogMode
  /** Whether Azure logging is enabled/available */
  azureEnabled?: boolean
  /** Azure connection status */
  azureStatus?: 'connected' | 'disconnected' | 'disabled'
  /** Loading state during mode switch */
  isLoading?: boolean
  /** Callback when mode changes */
  onModeChange?: (mode: LogMode) => void
  /** Additional class names */
  className?: string
}

// =============================================================================
// ModeToggle Component
// =============================================================================

export function ModeToggle({ 
  mode, 
  azureEnabled = false,
  azureStatus = 'disabled',
  isLoading = false,
  onModeChange,
  className 
}: ModeToggleProps) {
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
    if (isLoading) return
    
    const newMode: LogMode = mode === 'local' ? 'azure' : 'local'
    
    // Check if Azure is available before switching to it
    if (newMode === 'azure' && !azureEnabled) {
      setAnnouncement('Azure logging is not configured')
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
      timeoutRef.current = setTimeout(() => setAnnouncement(''), 2000)
      return
    }
    
    onModeChange?.(newMode)
    
    // Announce to screen readers
    const modeLabel = newMode === 'local' ? 'Local' : 'Azure'
    setAnnouncement(`Switched to ${modeLabel} logs`)
    
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    timeoutRef.current = setTimeout(() => setAnnouncement(''), 1000)
  }

  const isLocal = mode === 'local'
  const label = isLocal 
    ? 'Switch to Azure logs' 
    : 'Switch to local logs'
  
  // Determine if Azure button should be disabled
  const azureDisabled = !azureEnabled || azureStatus === 'disabled'

  return (
    <>
      <div 
        className={cn(
          'flex items-center gap-1 p-1 rounded-lg',
          'bg-slate-100 dark:bg-slate-800',
          className
        )}
        role="radiogroup"
        aria-label="Log source"
      >
        {/* Local Mode Button */}
        <button
          type="button"
          role="radio"
          aria-checked={isLocal}
          onClick={() => mode !== 'local' && onModeChange?.('local')}
          disabled={isLoading}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
            'transition-all duration-200 ease-out',
            isLocal ? [
              'bg-white dark:bg-slate-700',
              'text-slate-900 dark:text-slate-100',
              'shadow-sm',
            ] : [
              'text-slate-500 dark:text-slate-400',
              'hover:text-slate-700 dark:hover:text-slate-300',
            ],
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-cyan-500 focus-visible:ring-offset-1',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          <Monitor className="w-4 h-4" />
          <span>Local</span>
        </button>

        {/* Azure Mode Button */}
        <button
          type="button"
          role="radio"
          aria-checked={!isLocal}
          onClick={handleToggle}
          disabled={isLoading || azureDisabled}
          title={azureDisabled ? 'Azure logging not configured' : label}
          className={cn(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
            'transition-all duration-200 ease-out',
            !isLocal ? [
              'bg-white dark:bg-slate-700',
              'text-slate-900 dark:text-slate-100',
              'shadow-sm',
            ] : [
              'text-slate-500 dark:text-slate-400',
              'hover:text-slate-700 dark:hover:text-slate-300',
            ],
            azureDisabled && 'opacity-50 cursor-not-allowed',
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-cyan-500 focus-visible:ring-offset-1',
            'disabled:opacity-50 disabled:cursor-not-allowed',
          )}
        >
          {isLoading ? (
            <Loader2 className="w-4 h-4 animate-spin" />
          ) : (
            <Cloud className="w-4 h-4" />
          )}
          <span>Azure</span>
          {/* Status indicator dot */}
          {azureEnabled && (
            <span 
              className={cn(
                'w-1.5 h-1.5 rounded-full ml-0.5',
                azureStatus === 'connected' && 'bg-green-500',
                azureStatus === 'disconnected' && 'bg-yellow-500',
                azureStatus === 'disabled' && 'bg-slate-400',
              )}
              aria-hidden="true"
            />
          )}
        </button>
      </div>
      
      {/* Screen reader announcements */}
      <div role="status" aria-live="polite" className="sr-only">
        {announcement}
      </div>
    </>
  )
}

export default ModeToggle
