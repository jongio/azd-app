/**
 * AzureConnectionStatus - Azure connection status indicator
 * Shows the current Azure log streaming connection status
 */
import * as React from 'react'
import { Cloud, CloudOff, Loader2, AlertCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

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
    label: 'Disabled',
    description: 'Azure logging not configured',
    colorClass: 'text-slate-500 dark:text-slate-400',
    bgClass: 'bg-slate-100 dark:bg-slate-800',
  },
}

// =============================================================================
// AzureConnectionStatus Component
// =============================================================================

export function AzureConnectionStatus({
  status,
  resourceCount = 0,
  errorMessage,
  showDetails = false,
  className,
}: AzureConnectionStatusProps) {
  const config = statusConfig[status]
  const Icon = config.icon
  const isAnimated = status === 'connecting'

  return (
    <div
      className={cn(
        'flex items-center gap-2',
        className
      )}
      role="status"
      aria-label={`Azure connection: ${config.label}`}
    >
      {/* Icon with background */}
      <div
        className={cn(
          'flex items-center justify-center w-8 h-8 rounded-lg',
          config.bgClass
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
    </div>
  )
}

// =============================================================================
// Compact Variant
// =============================================================================

export interface AzureStatusBadgeProps {
  status: AzureConnectionState
  className?: string
}

/**
 * Compact badge variant for use in headers or tight spaces
 */
export function AzureStatusBadge({ status, className }: AzureStatusBadgeProps) {
  const config = statusConfig[status]
  const Icon = config.icon
  const isAnimated = status === 'connecting'

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium',
        config.bgClass,
        config.colorClass,
        className
      )}
      role="status"
      aria-label={`Azure: ${config.label}`}
    >
      <Icon 
        className={cn(
          'w-3 h-3',
          isAnimated && 'animate-spin'
        )}
      />
      <span>{config.label}</span>
    </div>
  )
}

export default AzureConnectionStatus
