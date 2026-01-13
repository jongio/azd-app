/**
 * DiagnosticsModal - Azure logs health diagnostics modal
 * Displays health check results and provides troubleshooting guidance
 */
import * as React from 'react'
import { X, CheckCircle, AlertCircle, XCircle, Copy, Check, ExternalLink, Loader2, RefreshCw, Wrench } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useTimeout } from '@/hooks/useTimeout'
import type { SetupStep } from './AzureSetupGuide'

// =============================================================================
// Types
// =============================================================================

export interface DiagnosticsModalProps {
  isOpen: boolean
  onClose: () => void
  onOpenSetupGuide?: (step: SetupStep) => void
}

interface HealthCheckResponse {
  status: 'healthy' | 'degraded' | 'error'
  checks: HealthCheck[]
  docsUrl: string
  timestamp: string
}

interface HealthCheck {
  name: string
  status: 'pass' | 'warn' | 'fail'
  message: string
  fix?: string
}

// =============================================================================
// Helper Components
// =============================================================================

interface StatusIconProps {
  status: 'pass' | 'warn' | 'fail'
  className?: string
}

function StatusIcon({ status, className }: Readonly<StatusIconProps>) {
  const config = {
    pass: { Icon: CheckCircle, color: 'text-emerald-500' },
    warn: { Icon: AlertCircle, color: 'text-amber-500' },
    fail: { Icon: XCircle, color: 'text-red-500' },
  }[status]

  const { Icon, color } = config

  return <Icon className={cn('w-5 h-5', color, className)} />
}

interface CopyButtonProps {
  text: string
  label?: string
}

function CopyButton({ text, label = 'Copy command' }: Readonly<CopyButtonProps>) {
  const [copied, setCopied] = React.useState(false)
  const { setTimeout } = useTimeout()

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <button
      type="button"
      onClick={() => void handleCopy()}
      className={cn(
        'inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium',
        'text-slate-600 dark:text-slate-300',
        'hover:bg-slate-100 dark:hover:bg-slate-800',
        'transition-colors duration-200',
        'focus:outline-none focus:ring-2 focus:ring-cyan-500',
      )}
      aria-label={copied ? 'Copied' : label}
    >
      {copied ? (
        <>
          <Check className="w-3.5 h-3.5 text-emerald-500" />
          Copied
        </>
      ) : (
        <>
          <Copy className="w-3.5 h-3.5" />
          Copy
        </>
      )}
    </button>
  )
}

const statusDisplay: Record<HealthCheckResponse['status'], {
  label: string
  Icon: typeof CheckCircle
  badgeClasses: string
}> = {
  healthy: {
    label: 'All checks passing',
    Icon: CheckCircle,
    badgeClasses: 'border-emerald-200 text-emerald-700 bg-emerald-50 dark:bg-emerald-950/30 dark:border-emerald-800 dark:text-emerald-200',
  },
  degraded: {
    label: 'Some checks need attention',
    Icon: AlertCircle,
    badgeClasses: 'border-amber-200 text-amber-700 bg-amber-50 dark:bg-amber-950/30 dark:border-amber-800 dark:text-amber-200',
  },
  error: {
    label: 'Diagnostics blocked',
    Icon: XCircle,
    badgeClasses: 'border-red-200 text-red-700 bg-red-50 dark:bg-red-950/30 dark:border-red-800 dark:text-red-200',
  },
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Determine which setup step failed based on health check results
 * Used for deep linking to the correct step in the setup guide
 */
function determineFailingStep(checks: HealthCheck[]): SetupStep {
  // Check for workspace issues (must come first as it's foundational)
  if (checks.some(c => c.name.toLowerCase().includes('workspace'))) {
    return 'workspace'
  }
  
  // Check for authentication issues
  if (checks.some(c => c.name.toLowerCase().includes('auth') || c.name.toLowerCase().includes('permission'))) {
    return 'auth'
  }
  
  // Check for diagnostic settings issues
  if (checks.some(c => c.name.toLowerCase().includes('diagnostic'))) {
    return 'diagnostic-settings'
  }
  
  // Default to verification step for other issues
  return 'verification'
}

// =============================================================================
// DiagnosticsModal Component
// =============================================================================

export function DiagnosticsModal({ isOpen, onClose, onOpenSetupGuide }: Readonly<DiagnosticsModalProps>) {
  const dialogRef = React.useRef<HTMLDialogElement>(null)
  const abortRef = React.useRef<AbortController | null>(null)
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [data, setData] = React.useState<HealthCheckResponse | null>(null)

  useEscapeKey(onClose, isOpen)

  const runHealthCheck = React.useCallback(async () => {
    abortRef.current?.abort()
    const abortController = new AbortController()
    abortRef.current = abortController

    setLoading(true)
    setError(null)

    try {
      const response = await fetch('/api/azure/logs/health', {
        signal: abortController.signal,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const result = await response.json() as HealthCheckResponse
      setData(result)
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        return
      }
      setData(null)
      setError(err instanceof Error ? err.message : 'Failed to fetch health check')
    } finally {
      if (abortRef.current === abortController) {
        abortRef.current = null
      }
      setLoading(false)
    }
  }, [])

  // Fetch health check data when modal opens
  React.useEffect(() => {
    if (!isOpen) {
      abortRef.current?.abort()
      return
    }

    void runHealthCheck()

    return () => {
      abortRef.current?.abort()
    }
  }, [isOpen, runHealthCheck])

  // Focus management
  React.useEffect(() => {
    if (isOpen && dialogRef.current) {
      const closeButton = dialogRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Copy full diagnostics report
  const handleCopyDiagnostics = async () => {
    if (!data) return

    const statusSymbols = {
      pass: '✓',
      warn: '⚠',
      fail: '✗',
    }

    const lines = [
      `Azure Logs Diagnostics - ${new Date(data.timestamp).toLocaleString()}`,
      `Status: ${data.status}`,
      '',
    ]

    for (const check of data.checks) {
      const symbol = statusSymbols[check.status]
      const fixText = check.fix ? ` (Fix: ${check.fix})` : ''
      lines.push(`${symbol} ${check.name}: ${check.message}${fixText}`)
    }

    await navigator.clipboard.writeText(lines.join('\n'))
  }

  const activeStatus = data ? statusDisplay[data.status] : null

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 dark:bg-black/70 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Dialog */}
      <dialog
        ref={dialogRef}
        open
        aria-labelledby="diagnostics-title"
        className={cn(
          'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
          'w-full max-w-2xl',
          'bg-white dark:bg-slate-900',
          'border border-slate-200 dark:border-slate-700',
          'rounded-2xl shadow-2xl',
          'flex flex-col',
          'max-h-[90vh]',
          'animate-scale-in',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <h2
            id="diagnostics-title"
            className="text-lg font-semibold text-slate-900 dark:text-slate-100"
          >
            Azure Logs Diagnostics
          </h2>
          <button
            type="button"
            data-close-button
            onClick={onClose}
            className="p-2 -mr-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close diagnostics"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {/* Loading State */}
          {loading && (
            <div className="flex flex-col items-center justify-center py-12 text-slate-500 dark:text-slate-400">
              <Loader2 className="w-8 h-8 mb-3 animate-spin" />
              <p className="text-sm">Running health checks...</p>
            </div>
          )}

          {/* Error State */}
          {error && !loading && (
            <div className="flex flex-col items-center justify-center py-12">
              <div className={cn(
                'w-12 h-12 rounded-full flex items-center justify-center mb-4',
                'bg-red-50 dark:bg-red-950/30',
                'border border-red-200 dark:border-red-800',
              )}>
                <XCircle className="w-6 h-6 text-red-500" />
              </div>
              <h3 className="text-lg font-medium text-slate-900 dark:text-slate-100 mb-2">
                Failed to fetch diagnostics
              </h3>
              <p className="text-sm text-slate-600 dark:text-slate-400 max-w-md text-center">
                {error}
              </p>
            </div>
          )}

          {/* Health Check Results */}
          {data && !loading && (
            <div className="space-y-4">
              {data.checks.map((check) => (
                <div
                  key={`${check.name}:${check.status}`}
                  className={cn(
                    'rounded-lg border p-4',
                    'bg-slate-50 dark:bg-slate-800/50',
                    'border-slate-200 dark:border-slate-700',
                  )}
                >
                  <div className="flex items-start gap-3">
                    <StatusIcon status={check.status} className="shrink-0 mt-0.5" />
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-slate-900 dark:text-slate-100 mb-1">
                        {check.name}
                      </div>
                      <div className="text-sm text-slate-600 dark:text-slate-400">
                        {check.message}
                      </div>
                      {check.fix && (
                        <div className={cn(
                          'mt-3 rounded-md p-3',
                          'bg-slate-100 dark:bg-slate-800',
                          'border border-slate-200 dark:border-slate-700',
                        )}>
                          <div className="flex items-start justify-between gap-2">
                            <div className="flex-1 min-w-0">
                              <div className="text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                                Fix
                              </div>
                              <code className="text-xs text-slate-600 dark:text-slate-400 break-all">
                                {check.fix}
                              </code>
                            </div>
                            <CopyButton text={check.fix} />
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        {!loading && (
          <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/60 shrink-0">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap items-center gap-3">
                {activeStatus ? (
                  <div
                    className={cn(
                      'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm font-semibold shadow-sm',
                      activeStatus.badgeClasses,
                    )}
                  >
                    <activeStatus.Icon className="w-4 h-4" />
                    {activeStatus.label}
                  </div>
                ) : (
                  <div className="text-sm font-semibold text-slate-800 dark:text-slate-100">
                    Diagnostics ready to run
                  </div>
                )}
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  {data?.timestamp
                    ? `Updated ${new Date(data.timestamp).toLocaleString()}`
                    : 'Run diagnostics to capture the latest checks.'}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 justify-end">
                {/* Fix Setup button - shown when checks fail and callback is provided */}
                {data && onOpenSetupGuide && data.status !== 'healthy' && (
                  <button
                    type="button"
                    onClick={() => {
                      const failingStep = determineFailingStep(data.checks.filter(c => c.status === 'fail'))
                      onOpenSetupGuide(failingStep)
                    }}
                    className={cn(
                      'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold shadow-sm',
                      'bg-orange-600 text-white hover:bg-orange-700',
                      'focus:outline-none focus:ring-2 focus:ring-orange-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                    )}
                  >
                    <Wrench className="w-4 h-4" />
                    Fix Setup
                  </button>
                )}

                <button
                  type="button"
                  onClick={() => void runHealthCheck()}
                  disabled={loading}
                  className={cn(
                    'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold shadow-sm',
                    'bg-cyan-600 text-white hover:bg-cyan-700',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                    'disabled:opacity-60 disabled:cursor-not-allowed',
                  )}
                >
                  <RefreshCw className="w-4 h-4" />
                  Run Diagnostics
                </button>

                {data && (
                  <>
                    <button
                      type="button"
                      onClick={() => void handleCopyDiagnostics()}
                      className={cn(
                        'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold border shadow-sm',
                        'text-slate-800 dark:text-slate-100 border-slate-200 dark:border-slate-700',
                        'bg-white hover:bg-slate-100 dark:bg-slate-900 dark:hover:bg-slate-800',
                        'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                      )}
                    >
                      <Copy className="w-4 h-4" />
                      Copy Report
                    </button>

                    <a
                      href={data.docsUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={cn(
                        'inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold border shadow-sm',
                        'text-slate-800 dark:text-slate-100 border-slate-200 dark:border-slate-700',
                        'bg-white hover:bg-slate-100 dark:bg-slate-900 dark:hover:bg-slate-800',
                        'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900',
                      )}
                    >
                      Troubleshooting Guide
                      <ExternalLink className="w-4 h-4" />
                    </a>
                  </>
                )}

                <button
                  type="button"
                  onClick={onClose}
                  className={cn(
                    'inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-semibold text-slate-700 dark:text-slate-200',
                    'hover:bg-slate-200/70 dark:hover:bg-slate-800',
                    'transition-colors duration-150',
                  )}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </dialog>
    </>
  )
}

export default DiagnosticsModal
