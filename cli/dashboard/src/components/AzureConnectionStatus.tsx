/**
 * AzureConnectionStatus - Azure connection status indicator
 * Shows the current Azure log streaming connection status with error details
 */
import * as React from 'react'
import { Cloud, CloudOff, Loader2, AlertCircle, RefreshCw, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { parseAzureError } from '@/lib/azure-errors'
import AzureErrorDisplay, { type SetupStep } from './AzureErrorDisplay'

// =============================================================================
// Types
// =============================================================================

export type AzureConnectionState = 'connected' | 'disconnected' | 'connecting' | 'error' | 'disabled'

export interface AzureConnectionStatusProps {
  /** Connection state */
  status: AzureConnectionState
  /** Number of discovered Azure resources */
  resourceCount?: number
  /** Error message if status is 'error' */
  errorMessage?: string
  /** Whether to show detailed status */
  showDetails?: boolean
  /** Callback to retry connection */
  onRetry?: () => void
  /** Callback to open setup guide at specific step */
  onOpenSetupGuide?: (step: SetupStep) => void
  /** Additional class names */
  className?: string
}

// =============================================================================
// Status Config
// =============================================================================

const statusConfig: Record<AzureConnectionState, {
  icon: React.ComponentType<{ className?: string }>
  label: string
  description: string
  colorClass: string
  bgClass: string
}> = {
  connected: {
    icon: Cloud,
    label: 'Connected',
    description: 'Streaming logs from Azure',
    colorClass: 'text-green-600 dark:text-green-400',
    bgClass: 'bg-green-100 dark:bg-green-900/30',
  },
  disconnected: {
    icon: CloudOff,
    label: 'Disconnected',
    description: 'Azure logs not streaming',
    colorClass: 'text-yellow-600 dark:text-yellow-400',
    bgClass: 'bg-yellow-100 dark:bg-yellow-900/30',
  },
  connecting: {
    icon: Loader2,
    label: 'Connecting',
    description: 'Establishing Azure connection',
    colorClass: 'text-blue-600 dark:text-blue-400',
    bgClass: 'bg-blue-100 dark:bg-blue-900/30',
  },
  error: {
    icon: AlertCircle,
    label: 'Error',
    description: 'Failed to connect to Azure',
    colorClass: 'text-red-600 dark:text-red-400',
    bgClass: 'bg-red-100 dark:bg-red-900/30',
  },
  disabled: {
    icon: CloudOff,
    label: 'Not Configured',
    description: 'Add logs.analytics section to azure.yaml',
    colorClass: 'text-slate-500 dark:text-slate-400',
    bgClass: 'bg-slate-100 dark:bg-slate-800',
  },
}

// =============================================================================
// Error Popover Component
// =============================================================================

interface ErrorPopoverProps {
  errorMessage: string
  onRetry?: () => void
  onClose: () => void
  onOpenSetupGuide?: (step: SetupStep) => void
}

function ErrorPopover({ errorMessage, onRetry, onClose, onOpenSetupGuide }: ErrorPopoverProps) {
  const errorType = parseAzureError(errorMessage)
  const popoverRef = React.useRef<HTMLDivElement>(null)

  // Close on click outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(event.target as Node)) {
        onClose()
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [onClose])

  // Close on Escape
  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div
      ref={popoverRef}
      className={cn(
        'absolute right-0 top-full mt-2 z-50',
        'w-80 rounded-lg shadow-xl',
        'bg-white dark:bg-slate-900',
        'border border-slate-200 dark:border-slate-700',
        'animate-fade-in',
      )}
      role="dialog"
      aria-label="Azure connection error details"
    >
      {/* Close button */}
      <button
        type="button"
        onClick={onClose}
        className={cn(
          'absolute top-2 right-2 p-1 rounded-md',
          'text-slate-400 hover:text-slate-600 dark:hover:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'transition-colors',
        )}
        aria-label="Close"
      >
        <X className="w-4 h-4" />
      </button>

      {/* Error display */}
      <div className="p-4">
        <AzureErrorDisplay
          errorType={errorType}
          message={errorMessage}
          onRetry={onRetry}
          compact={false}
          onOpenSetupGuide={onOpenSetupGuide}
        />
      </div>
    </div>
  )
}

// =============================================================================
// AzureConnectionStatus Component
// =============================================================================

export function AzureConnectionStatus({
  status,
  resourceCount = 0,
  errorMessage,
  showDetails = false,
  onRetry,
  onOpenSetupGuide,
  className,
}: AzureConnectionStatusProps) {
  const [showErrorPopover, setShowErrorPopover] = React.useState(false)
  const config = statusConfig[status]
  const Icon = config.icon
  const isAnimated = status === 'connecting'
  const isClickable = status === 'error' && errorMessage

  const handleClick = () => {
    if (isClickable) {
      setShowErrorPopover((prev) => !prev)
    }
  }

  const handleRetry = () => {
    setShowErrorPopover(false)
    onRetry?.()
  }

  return (
    <div className={cn('relative', className)}>
      <div
        className={cn(
          'flex items-center gap-2',
          isClickable && 'cursor-pointer',
        )}
        role="status"
        aria-label={`Azure connection: ${config.label}`}
        onClick={handleClick}
        onKeyDown={(e) => {
          if (isClickable && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            handleClick()
          }
        }}
        tabIndex={isClickable ? 0 : undefined}
      >
        {/* Icon with background */}
        <div
          className={cn(
            'flex items-center justify-center w-8 h-8 rounded-lg',
            config.bgClass,
            isClickable && 'hover:opacity-80 transition-opacity',
          )}
        >
          <Icon 
            className={cn(
              'w-4 h-4',
              config.colorClass,
              isAnimated && 'animate-spin'
            )}
          />
        </div>

        {/* Status text */}
        {showDetails && (
          <div className="flex flex-col min-w-0">
            <span className={cn('text-sm font-medium', config.colorClass)}>
              {config.label}
              {isClickable && (
                <span className="text-xs ml-1 opacity-60">(click for details)</span>
              )}
            </span>
            <span className="text-xs text-slate-500 dark:text-slate-400 truncate">
              {status === 'error' && errorMessage 
                ? errorMessage 
                : status === 'connected' && resourceCount > 0
                  ? `${resourceCount} resource${resourceCount !== 1 ? 's' : ''}`
                  : config.description
              }
            </span>
          </div>
        )}

        {/* Retry button for error state (non-detailed view) */}
        {status === 'error' && !showDetails && onRetry && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onRetry()
            }}
            className={cn(
              'p-1.5 rounded-md transition-colors',
              'text-red-500 hover:text-red-600 dark:hover:text-red-400',
              'hover:bg-red-100 dark:hover:bg-red-900/30',
              'focus:outline-none focus:ring-2 focus:ring-red-500',
            )}
            title="Retry connection"
            aria-label="Retry Azure connection"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
        )}
      </div>

      {/* Error details popover */}
      {showErrorPopover && errorMessage && (
        <ErrorPopover
          errorMessage={errorMessage}
          onRetry={handleRetry}
          onClose={() => setShowErrorPopover(false)}
          onOpenSetupGuide={onOpenSetupGuide}
        />
      )}
    </div>
  )
}

// =============================================================================
// Compact Variant
// =============================================================================

export interface AzureStatusBadgeProps {
  status: AzureConnectionState
  errorMessage?: string
  onRetry?: () => void
  onOpenSetupGuide?: (step: SetupStep) => void
  className?: string
}

/**
 * Compact badge variant for use in headers or tight spaces
 */
export function AzureStatusBadge({ status, errorMessage, onRetry, onOpenSetupGuide, className }: AzureStatusBadgeProps) {
  const [showErrorPopover, setShowErrorPopover] = React.useState(false)
  const config = statusConfig[status]
  const Icon = config.icon
  const isAnimated = status === 'connecting'
  const isClickable = status === 'error' && errorMessage

  const handleClick = () => {
    if (isClickable) {
      setShowErrorPopover((prev) => !prev)
    }
  }

  const handleRetry = () => {
    setShowErrorPopover(false)
    onRetry?.()
  }

  return (
    <div className="relative">
      <div
        className={cn(
          'inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium',
          config.bgClass,
          config.colorClass,
          isClickable && 'cursor-pointer hover:opacity-80 transition-opacity',
          className
        )}
        role="status"
        aria-label={`Azure: ${config.label}`}
        onClick={handleClick}
        onKeyDown={(e) => {
          if (isClickable && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault()
            handleClick()
          }
        }}
        tabIndex={isClickable ? 0 : undefined}
      >
        <Icon 
          className={cn(
            'w-3 h-3',
            isAnimated && 'animate-spin'
          )}
        />
        <span>{config.label}</span>
      </div>

      {/* Error details popover */}
      {showErrorPopover && errorMessage && (
        <ErrorPopover
          errorMessage={errorMessage}
          onRetry={handleRetry}
          onClose={() => setShowErrorPopover(false)}
          onOpenSetupGuide={onOpenSetupGuide}
        />
      )}
    </div>
  )
}

export default AzureConnectionStatus
