/**
 * DiagnosticsModal - Azure logs health diagnostics modal
 * Displays health check results and provides troubleshooting guidance
 */
import * as React from 'react'
import { X, CheckCircle, AlertCircle, XCircle, Copy, Check, ExternalLink, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'

// =============================================================================
// Types
// =============================================================================

export interface DiagnosticsModalProps {
  isOpen: boolean
  onClose: () => void
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

function StatusIcon({ status, className }: StatusIconProps) {
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

function CopyButton({ text, label = 'Copy command' }: CopyButtonProps) {
  const [copied, setCopied] = React.useState(false)

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

// =============================================================================
// DiagnosticsModal Component
// =============================================================================

export function DiagnosticsModal({ isOpen, onClose }: DiagnosticsModalProps) {
  const dialogRef = React.useRef<HTMLDivElement>(null)
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)
  const [data, setData] = React.useState<HealthCheckResponse | null>(null)

  useEscapeKey(onClose, isOpen)

  // Fetch health check data when modal opens
  React.useEffect(() => {
    if (!isOpen) return

    const abortController = new AbortController()

    const fetchHealthCheck = async () => {
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
        setError(err instanceof Error ? err.message : 'Failed to fetch health check')
      } finally {
        setLoading(false)
      }
    }

    void fetchHealthCheck()

    return () => {
      abortController.abort()
    }
  }, [isOpen])

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

  if (!isOpen) {
    return null
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
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
              {data.checks.map((check, index) => (
                <div
                  key={index}
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
        {data && !loading && (
          <div className="flex items-center justify-between px-6 py-4 border-t border-slate-200 dark:border-slate-700 shrink-0">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => void handleCopyDiagnostics()}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
                  'text-slate-700 dark:text-slate-200',
                  'hover:bg-slate-100 dark:hover:bg-slate-800',
                  'transition-colors duration-200',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                )}
              >
                <Copy className="w-4 h-4" />
                Copy Diagnostics
              </button>
              <a
                href={data.docsUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
                  'text-slate-700 dark:text-slate-200',
                  'hover:bg-slate-100 dark:hover:bg-slate-800',
                  'transition-colors duration-200',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                )}
              >
                View Troubleshooting Guide
                <ExternalLink className="w-4 h-4" />
              </a>
            </div>
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-medium',
                'text-slate-700 dark:text-slate-200',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'transition-colors duration-200',
              )}
            >
              Close
            </button>
          </div>
        )}
      </div>
    </>
  )
}

export default DiagnosticsModal
