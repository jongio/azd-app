/**
 * ModeToggle - Log source mode toggle component
 * Switches between local and Azure log sources
 * 
 * @see cli/docs/design/components/log-source-switcher.md
 */
import * as React from 'react'
import { Monitor, Loader2 } from 'lucide-react'
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
  azureStatus?: 'connected' | 'disconnected' | 'connecting' | 'disabled' | 'degraded'
  /** Message explaining connection issue (shown in tooltip when disconnected) */
  connectionMessage?: string
  /** Loading state during mode switch */
  isLoading?: boolean
  /** Size variant */
  size?: 'compact' | 'standard' | 'large'
  /** Show text labels */
  showLabels?: boolean
  /** Show connection status indicator */
  showStatus?: boolean
  /** Callback when mode changes */
  onModeChange?: (mode: LogMode) => void
  /** Callback to open setup guide (called when Azure clicked while disabled) */
  onOpenSetupGuide?: () => void
  /** Additional class names */
  className?: string
}

// =============================================================================
// Azure Icon Component
// =============================================================================

interface AzureIconProps {
  className?: string
}

/**
 * Azure cloud icon with brand styling
 */
function AzureIcon({ className }: Readonly<AzureIconProps>) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      {/* Cloud shape */}
      <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z" />
    </svg>
  )
}

// =============================================================================
// Size Configuration
// =============================================================================

const sizeConfig = {
  compact: {
    container: 'p-1 rounded-lg',
    button: 'p-2 rounded-md',
    icon: 'w-4 h-4',
    label: 'hidden',
    gap: 'gap-0.5',
    statusDot: 'w-1.5 h-1.5',
  },
  standard: {
    container: 'p-1 rounded-lg',
    button: 'px-3 py-1.5 rounded-md gap-1.5',
    icon: 'w-4 h-4',
    label: 'text-sm font-medium',
    gap: 'gap-1',
    statusDot: 'w-1.5 h-1.5',
  },
  large: {
    container: 'p-1.5 rounded-xl',
    button: 'px-4 py-2 rounded-lg gap-2',
    icon: 'w-5 h-5',
    label: 'text-sm font-medium',
    gap: 'gap-1.5',
    statusDot: 'w-2 h-2',
  },
}

interface StatusIndicatorProps {
  showStatus: boolean
  azureEnabled: boolean
  status: ModeToggleProps['azureStatus']
  connectionMessage?: string
  statusDotClassName: string
}

function StatusIndicator({
  showStatus,
  azureEnabled,
  status,
  connectionMessage,
  statusDotClassName,
}: Readonly<StatusIndicatorProps>) {
  if (!showStatus) return null

  // Show connection message even when Azure is disabled (e.g., not configured)
  if (connectionMessage && (!azureEnabled || status === 'disconnected')) {
    return (
      <span className="relative group" title={connectionMessage}>
        <span
          className={cn(
            'flex items-center justify-center rounded-full bg-amber-500',
            statusDotClassName,
          )}
          aria-hidden="true"
        >
          <span className="text-white text-[6px] font-bold">!</span>
        </span>
        <span
          className={cn(
            'absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1',
            'bg-slate-900 dark:bg-slate-700 text-white text-xs rounded whitespace-nowrap',
            'opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity',
            'z-50 shadow-lg',
          )}
        >
          {connectionMessage}
          <span className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-900 dark:border-t-slate-700" />
        </span>
      </span>
    )
  }

  return null
}

// =============================================================================
// ModeToggle Component
// =============================================================================

export function ModeToggle({ 
  mode, 
  azureEnabled = false,
  azureStatus = 'disabled',
  connectionMessage,
  isLoading = false,
  size = 'standard',
  showLabels = true,
  showStatus = true,
  onModeChange,
  onOpenSetupGuide,
  className 
}: Readonly<ModeToggleProps>) {
  const [announcement, setAnnouncement] = React.useState('')
  const timeoutRef = React.useRef<ReturnType<typeof setTimeout> | null>(null)
  const config = sizeConfig[size]

  // Cleanup timer on unmount
  React.useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleModeChange = (newMode: LogMode) => {
    if (isLoading || mode === newMode) {
      return
    }
    
    // If switching to Azure but it's not enabled, open setup guide instead
    if (newMode === 'azure' && !azureEnabled) {
      onOpenSetupGuide?.()
      return
    }
    
    // Allow switching modes
    onModeChange?.(newMode)
    
    // Announce to screen readers
    const modeLabel = newMode === 'local' ? 'Local' : 'Azure'
    setAnnouncement(`Switched to ${modeLabel} logs`)
    
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    timeoutRef.current = setTimeout(() => setAnnouncement(''), 1000)
  }

  // Handle keyboard navigation within radio group
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      e.preventDefault()
      const newMode = mode === 'local' ? 'azure' : 'local'
      handleModeChange(newMode)
    }
  }

  const isLocal = mode === 'local'
  
  // Get Azure icon color based on connection status
  const getAzureIconColor = (status: typeof azureStatus) => {
    if (!azureEnabled) return ''
    if (status === 'connected') return 'text-emerald-500 dark:text-emerald-400'
    if (status === 'connecting') return 'text-amber-500 dark:text-amber-400 animate-pulse'
    if (status === 'disconnected') return 'text-amber-500 dark:text-amber-400'
    return ''
  }

  // Determine Azure button title
  const getAzureTitle = (): string | undefined => {
    if (azureEnabled) {
      return size === 'compact' ? 'Azure logs' : undefined
    }
    return 'Click to set up Azure logs'
  }

  return (
    <>
      <div 
        className={cn(
          'flex items-center bg-slate-100 dark:bg-slate-800',
          config.container,
          config.gap,
          className
        )}
        role="group"
        aria-label="Log source"
        onKeyDown={handleKeyDown}
        tabIndex={-1}
      >
        {/* Local Mode Button */}
        <button
          type="button"
          aria-pressed={isLocal}
          aria-label="View local logs"
          onClick={() => handleModeChange('local')}
          disabled={isLoading}
          title={size === 'compact' ? 'Local logs' : undefined}
          className={cn(
            'flex items-center justify-center',
            'transition-all duration-150 ease-out',
            config.button,
            isLocal ? [
              'bg-white dark:bg-slate-700',
              'text-slate-900 dark:text-slate-100',
              'shadow-sm',
            ] : [
              'text-slate-500 dark:text-slate-400',
              'hover:text-slate-700 dark:hover:text-slate-300',
              'hover:bg-slate-50 dark:hover:bg-slate-700/50',
            ],
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-cyan-500 focus-visible:ring-offset-1',
            'focus-visible:ring-offset-slate-100 dark:focus-visible:ring-offset-slate-800',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'active:scale-95',
          )}
        >
          <Monitor className={config.icon} aria-hidden="true" />
          {showLabels && size !== 'compact' && (
            <span className={config.label}>Local</span>
          )}
        </button>

        {/* Azure Mode Button */}
        <button
          type="button"
          aria-pressed={isLocal === false}
          aria-label="View Azure logs"
          onClick={() => handleModeChange('azure')}
          disabled={isLoading}
          title={getAzureTitle()}
          className={cn(
            'flex items-center justify-center',
            'transition-all duration-150 ease-out',
            config.button,
            isLocal === false ? [
              'bg-azure-100 dark:bg-azure-500/20',
              'text-azure-600 dark:text-azure-400',
              'shadow-sm',
            ] : [
              'text-slate-500 dark:text-slate-400',
              'hover:text-azure-600 dark:hover:text-azure-400',
              'hover:bg-azure-50 dark:hover:bg-azure-500/10',
            ],
            'focus-visible:outline-none focus-visible:ring-2',
            'focus-visible:ring-azure-500 focus-visible:ring-offset-1',
            'focus-visible:ring-offset-slate-100 dark:focus-visible:ring-offset-slate-800',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'active:scale-95',
          )}
        >
          {isLoading && isLocal === false ? (
            <Loader2 className={cn(config.icon, 'animate-spin')} aria-hidden="true" />
          ) : (
            <AzureIcon className={cn(config.icon, showStatus && getAzureIconColor(azureStatus))} />
          )}
          {showLabels && size !== 'compact' && (
            <span className={config.label}>Azure</span>
          )}
          <StatusIndicator
            showStatus={showStatus}
            azureEnabled={azureEnabled}
            status={azureStatus}
            connectionMessage={connectionMessage}
            statusDotClassName={config.statusDot}
          />
        </button>
      </div>
      
      {/* Screen reader announcements */}
      <output aria-live="polite" className="sr-only">
        {announcement}
      </output>
    </>
  )
}

export default ModeToggle
