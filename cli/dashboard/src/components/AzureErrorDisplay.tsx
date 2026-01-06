/**
 * AzureErrorDisplay - Specific error handling UI for Azure log operations
 * Provides actionable guidance based on error type.
 * Design spec: cli/docs/design/components/azure-error-states.md
 */
import * as React from 'react'
import { 
  KeyRound, 
  ShieldOff, 
  Search, 
  Clock, 
  Wifi, 
  Database, 
  AlertTriangle,
  XCircle,
  Copy,
  Check,
  ExternalLink,
  RefreshCw,
  Monitor,
  Settings,
  BookOpen,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ErrorInfo } from '@/types'

// =============================================================================
// Types
// =============================================================================

export type AzureErrorType = 
  | 'auth'
  | 'permission'
  | 'not-found'
  | 'rate-limit'
  | 'network'
  | 'workspace'
  | 'query'
  | 'generic'

export type SetupStep = 'workspace' | 'auth' | 'diagnostic-settings' | 'verification'

export interface AzureErrorDisplayProps {
  /** Parsed error type */
  errorType?: AzureErrorType
  /** Original error message */
  message?: string
  /** Service name context */
  serviceName?: string
  /** Retry callback */
  onRetry?: () => void
  /** Switch to local logs callback */
  onViewLocal?: () => void
  /** Reset query callback (for query errors) */
  onResetQuery?: () => void
  /** Retry-After seconds (for rate limits) */
  retryAfter?: number
  /** Compact inline variant */
  compact?: boolean
  /** Additional class names */
  className?: string
  /** ErrorInfo from API - structured error with actionable guidance */
  errorInfo?: ErrorInfo
  /** Callback to open diagnostics modal */
  onRunDiagnostics?: () => void
  /** Callback to open setup guide at specific step */
  onOpenSetupGuide?: (step: SetupStep) => void
}

// =============================================================================
// Error Config
// =============================================================================

interface ErrorConfig {
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: (serviceName?: string) => string
  colorClasses: {
    icon: string
    border: string
    bg: string
  }
  command?: string
  codeSnippet?: string[]
  externalLink?: { url: string; label: string }
  primaryAction?: string
  secondaryAction?: 'local' | 'report' | 'reset-query'
}

const errorConfigs: Record<AzureErrorType, ErrorConfig> = {
  auth: {
    icon: KeyRound,
    title: 'Authentication Required',
    description: () => 'Sign in to Azure to view cloud logs.',
    colorClasses: {
      icon: 'text-amber-500',
      border: 'border-amber-200 dark:border-amber-800',
      bg: 'bg-amber-50 dark:bg-amber-950/30',
    },
    command: 'azd auth login',
    primaryAction: 'Retry Connection',
  },
  permission: {
    icon: ShieldOff,
    title: 'Permission Denied',
    description: () => "Your account doesn't have access to query logs.",
    colorClasses: {
      icon: 'text-rose-500',
      border: 'border-rose-200 dark:border-rose-800',
      bg: 'bg-rose-50 dark:bg-rose-950/30',
    },
    externalLink: {
      url: 'https://learn.microsoft.com/azure/azure-monitor/logs/manage-access',
      label: 'View Azure RBAC Docs',
    },
    primaryAction: 'Retry',
  },
  'not-found': {
    icon: Search,
    title: 'Resource Not Found',
    description: (serviceName) => 
      serviceName 
        ? `Service "${serviceName}" not found in Azure.`
        : 'The requested resource was not found in Azure.',
    colorClasses: {
      icon: 'text-slate-500',
      border: 'border-slate-200 dark:border-slate-700',
      bg: 'bg-slate-50 dark:bg-slate-800/50',
    },
    command: 'azd provision',
    primaryAction: 'Retry',
    secondaryAction: 'local',
  },
  'rate-limit': {
    icon: Clock,
    title: 'Rate Limited',
    description: () => 'Too many requests to Azure. Please wait.',
    colorClasses: {
      icon: 'text-amber-500',
      border: 'border-amber-200 dark:border-amber-800',
      bg: 'bg-amber-50 dark:bg-amber-950/30',
    },
    primaryAction: 'Retry Now',
  },
  network: {
    icon: Wifi,
    title: 'Connection Failed',
    description: () => 'Unable to reach Azure services.',
    colorClasses: {
      icon: 'text-sky-500',
      border: 'border-sky-200 dark:border-sky-800',
      bg: 'bg-sky-50 dark:bg-sky-950/30',
    },
    primaryAction: 'Retry',
    secondaryAction: 'local',
  },
  workspace: {
    icon: Database,
    title: 'Log Analytics Not Configured',
    description: () => 'No Log Analytics workspace found for this resource.',
    colorClasses: {
      icon: 'text-violet-500',
      border: 'border-violet-200 dark:border-violet-800',
      bg: 'bg-violet-50 dark:bg-violet-950/30',
    },
    codeSnippet: [
      'logs:',
      '  azure:',
      '    workspace: "your-workspace-id"',
    ],
    externalLink: {
      url: 'https://learn.microsoft.com/azure/azure-monitor/logs/log-analytics-workspace-overview',
      label: 'View Setup Guide',
    },
  },
  query: {
    icon: AlertTriangle,
    title: 'Query Error',
    description: () => 'Invalid query syntax.',
    colorClasses: {
      icon: 'text-amber-500',
      border: 'border-amber-200 dark:border-amber-800',
      bg: 'bg-amber-50 dark:bg-amber-950/30',
    },
    primaryAction: 'Retry',
    secondaryAction: 'reset-query',
  },
  generic: {
    icon: XCircle,
    title: 'Something went wrong',
    description: () => 'An unexpected error occurred.',
    colorClasses: {
      icon: 'text-red-500',
      border: 'border-red-200 dark:border-red-800',
      bg: 'bg-red-50 dark:bg-red-950/30',
    },
    primaryAction: 'Retry',
    secondaryAction: 'report',
  },
}

// =============================================================================
// Error to Setup Step Mapping
// =============================================================================

/**
 * Maps error types to the corresponding setup guide step
 * Returns null for errors that don't require setup guide (rate-limit, network, etc.)
 */
function getSetupStepForError(errorType: AzureErrorType): SetupStep | null {
  const errorToStep: Record<AzureErrorType, SetupStep | null> = {
    'workspace': 'workspace',
    'auth': 'auth',
    'permission': 'auth',
    'not-found': 'diagnostic-settings',
    'query': 'verification',
    'rate-limit': null,
    'network': null,
    'generic': null,
  }
  return errorToStep[errorType]
}

// =============================================================================
// Helper Components
// =============================================================================

interface CommandCopyProps {
  command: string
}

function CommandCopy({ command }: CommandCopyProps) {
  const [copied, setCopied] = React.useState(false)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="flex items-center gap-2 mt-3">
      <code className={cn(
        'flex-1 px-3 py-2 rounded-md font-mono text-sm',
        'bg-slate-100 dark:bg-slate-800',
        'text-slate-700 dark:text-slate-300',
        'border border-slate-200 dark:border-slate-700',
      )}>
        {command}
      </code>
      <button
        type="button"
        onClick={() => void handleCopy()}
        className={cn(
          'p-2 rounded-md transition-colors',
          'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500',
        )}
        aria-label={copied ? 'Copied' : 'Copy command'}
      >
        {copied ? (
          <Check className="w-4 h-4 text-green-500" />
        ) : (
          <Copy className="w-4 h-4" />
        )}
      </button>
      <span className="sr-only" aria-live="polite">
        {copied ? 'Copied to clipboard' : ''}
      </span>
    </div>
  )
}

// Helper to map error code to error type
function mapErrorCodeToType(code: string): AzureErrorType {
  const normalized = code.toUpperCase()
  if (normalized.includes('AUTH')) return 'auth'
  if (normalized.includes('PERMISSION') || normalized.includes('FORBIDDEN')) return 'permission'
  if (normalized.includes('NOT_DEPLOYED') || normalized.includes('NOT_FOUND')) return 'not-found'
  if (normalized.includes('RATE') || normalized.includes('THROTTLE')) return 'rate-limit'
  if (normalized.includes('NETWORK') || normalized.includes('TIMEOUT')) return 'network'
  if (normalized.includes('WORKSPACE')) return 'workspace'
  if (normalized.includes('QUERY')) return 'query'
  return 'generic'
}

interface CodeSnippetProps {
  lines: string[]
}

function CodeSnippet({ lines }: CodeSnippetProps) {
  return (
    <pre className={cn(
      'mt-3 px-3 py-2 rounded-md font-mono text-sm overflow-x-auto',
      'bg-slate-100 dark:bg-slate-800',
      'text-slate-700 dark:text-slate-300',
      'border border-slate-200 dark:border-slate-700',
    )}>
      {lines.map((line, idx) => (
        <div key={idx}>{line}</div>
      ))}
    </pre>
  )
}

interface CountdownTimerProps {
  seconds: number
  onComplete?: () => void
}

function CountdownTimer({ seconds, onComplete }: CountdownTimerProps) {
  const [remaining, setRemaining] = React.useState(seconds)

  React.useEffect(() => {
    if (remaining <= 0) {
      onComplete?.()
      return
    }

    const timer = setInterval(() => {
      setRemaining((prev) => {
        if (prev <= 1) {
          clearInterval(timer)
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [remaining, onComplete])

  const progress = ((seconds - remaining) / seconds) * 100

  return (
    <div className="mt-4 space-y-2">
      <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
        <span>Retry in:</span>
        <span className="font-medium">{remaining}s</span>
      </div>
      <div className="h-2 rounded-full bg-slate-200 dark:bg-slate-700 overflow-hidden">
        <div 
          className="h-full bg-amber-500 transition-all duration-1000 ease-linear"
          style={{ width: `${progress}%` }}
        />
      </div>
    </div>
  )
}

// =============================================================================
// AzureErrorDisplay Component
// =============================================================================

export function AzureErrorDisplay({
  errorType,
  message,
  serviceName,
  onRetry,
  onViewLocal,
  onResetQuery,
  retryAfter,
  compact = false,
  className,
  errorInfo,
  onRunDiagnostics,
  onOpenSetupGuide,
}: AzureErrorDisplayProps) {
  // If errorInfo is provided, use it; otherwise fallback to legacy props
  const effectiveErrorType = errorInfo ? mapErrorCodeToType(errorInfo.code) : errorType ?? 'generic'
  const config = errorConfigs[effectiveErrorType]
  const effectiveMessage = errorInfo?.message ?? message ?? config.description(serviceName)
  const actionMessage = errorInfo?.action
  const commandToRun = errorInfo?.command
  const docsUrl = errorInfo?.docsUrl
  
  const Icon = config.icon

  // Permission-specific details
  const permissionDetails = effectiveErrorType === 'permission' && (
    <div className="mt-3 text-sm text-slate-600 dark:text-slate-400">
      <div className="font-medium mb-1">Required permissions:</div>
      <ul className="list-disc list-inside space-y-1">
        <li>Log Analytics Reader on the workspace</li>
        <li>Reader on the resource group</li>
      </ul>
    </div>
  )

  // Query error shows the original error message
  const queryErrorDetails = effectiveErrorType === 'query' && effectiveMessage && (
    <pre className={cn(
      'mt-3 px-3 py-2 rounded-md font-mono text-xs overflow-x-auto',
      'bg-slate-100 dark:bg-slate-800',
      'text-slate-600 dark:text-slate-400',
      'border border-slate-200 dark:border-slate-700',
      'max-h-24',
    )}>
      {effectiveMessage}
    </pre>
  )

  // Secondary action button
  const secondaryButton = config.secondaryAction && (
    config.secondaryAction === 'local' && onViewLocal ? (
      <button
        type="button"
        onClick={onViewLocal}
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
          'text-slate-600 dark:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'transition-colors duration-200',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500',
        )}
      >
        <Monitor className="w-4 h-4" />
        View Local Logs
      </button>
    ) : config.secondaryAction === 'reset-query' && onResetQuery ? (
      <button
        type="button"
        onClick={onResetQuery}
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
          'text-slate-600 dark:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'transition-colors duration-200',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500',
        )}
      >
        Reset to Default Query
      </button>
    ) : config.secondaryAction === 'report' ? (
      <a
        href="https://github.com/Azure/azure-dev/issues/new"
        target="_blank"
        rel="noopener noreferrer"
        className={cn(
          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium',
          'text-slate-600 dark:text-slate-300',
          'hover:bg-slate-100 dark:hover:bg-slate-800',
          'transition-colors duration-200',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500',
        )}
      >
        Report Issue
        <ExternalLink className="w-3.5 h-3.5" />
      </a>
    ) : null
  )

  // Compact variant for inline display
  if (compact) {
    return (
      <div 
        role="alert"
        className={cn(
          'flex items-center gap-3 px-4 py-3 rounded-lg',
          config.colorClasses.bg,
          config.colorClasses.border,
          'border',
          className,
        )}
      >
        <Icon className={cn('w-5 h-5 shrink-0', config.colorClasses.icon)} />
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium text-slate-900 dark:text-slate-100">
            {config.title}
          </div>
          <div className="text-xs text-slate-600 dark:text-slate-400 truncate">
            {effectiveMessage}
          </div>
        </div>
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className={cn(
              'p-1.5 rounded-md transition-colors shrink-0',
              'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300',
              'hover:bg-white/50 dark:hover:bg-slate-800/50',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            )}
            aria-label="Retry"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        )}
      </div>
    )
  }

  return (
    <div 
      role="alert"
      className={cn(
        'flex flex-col items-center justify-center py-8 px-6 text-center',
        className,
      )}
    >
      {/* Icon */}
      <div className={cn(
        'w-12 h-12 rounded-full flex items-center justify-center mb-4',
        config.colorClasses.bg,
        config.colorClasses.border,
        'border',
      )}>
        <Icon className={cn('w-6 h-6', config.colorClasses.icon)} />
      </div>

      {/* Title */}
      <h3 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-2">
        {config.title}
      </h3>

      {/* Description */}
      <p className="text-sm text-slate-600 dark:text-slate-400 max-w-xs mb-4">
        {effectiveMessage}
      </p>

      {/* Action message from ErrorInfo */}
      {actionMessage && (
        <p className="text-sm font-medium text-slate-700 dark:text-slate-300 max-w-md mb-3">
          {actionMessage}
        </p>
      )}

      {/* Additional details */}
      {permissionDetails}
      {queryErrorDetails}

      {/* Command to copy - prioritize ErrorInfo.command over config.command */}
      {(commandToRun || config.command) && (
        <div className="w-full max-w-sm">
          <p className="text-xs text-slate-500 dark:text-slate-500 mb-1">
            Run this command in your terminal:
          </p>
          <CommandCopy command={commandToRun ?? config.command!} />
        </div>
      )}

      {/* Code snippet */}
      {config.codeSnippet && (
        <div className="w-full max-w-sm text-left">
          <p className="text-xs text-slate-500 dark:text-slate-500 mb-1">
            Configure in azure.yaml:
          </p>
          <CodeSnippet lines={config.codeSnippet} />
        </div>
      )}

      {/* Countdown timer for rate limits */}
      {effectiveErrorType === 'rate-limit' && retryAfter && retryAfter > 0 && (
        <div className="w-full max-w-xs">
          <CountdownTimer seconds={retryAfter} onComplete={onRetry} />
        </div>
      )}

      {/* Action buttons */}
      <div className="flex items-center gap-3 mt-6">
        {/* Primary action - Retry button */}
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
              'bg-cyan-500 hover:bg-cyan-600',
              'text-white',
              'transition-colors duration-200',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            )}
          >
            <RefreshCw className="w-4 h-4" />
            {config.primaryAction ?? 'Retry Now'}
          </button>
        )}

        {/* Setup Guide button - for setup-related errors */}
        {onOpenSetupGuide && getSetupStepForError(effectiveErrorType) && (
          <button
            type="button"
            onClick={() => {
              const step = getSetupStepForError(effectiveErrorType)
              if (step) {
                onOpenSetupGuide(step)
              }
            }}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
              'bg-violet-500 hover:bg-violet-600',
              'text-white',
              'transition-colors duration-200',
              'focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2',
            )}
          >
            <BookOpen className="w-4 h-4" />
            Setup Guide
          </button>
        )}

        {/* Run Diagnostics button */}
        {onRunDiagnostics && (
          <button
            type="button"
            onClick={onRunDiagnostics}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium',
              'text-slate-700 dark:text-slate-200',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'transition-colors duration-200',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            )}
          >
            <Settings className="w-4 h-4" />
            Run Diagnostics
          </button>
        )}

        {/* Docs link - prioritize ErrorInfo.docsUrl over config.externalLink */}
        {(docsUrl || config.externalLink) && (
          <a
            href={docsUrl ?? config.externalLink!.url}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              'flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium',
              'text-slate-600 dark:text-slate-300',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'transition-colors duration-200',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            )}
          >
            {docsUrl ? 'View Setup Guide' : config.externalLink!.label}
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
        )}

        {/* Secondary action */}
        {secondaryButton}
      </div>
    </div>
  )
}

export default AzureErrorDisplay
