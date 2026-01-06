/**
 * SetupVerification - Step 4 (FINAL) of Azure Logs Setup Guide
 * Verifies log connectivity and displays setup completion status
 */
import * as React from 'react'
import { CheckCircle, AlertTriangle, RefreshCw, Loader2, Sparkles, ArrowLeft } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useWorkspaceVerification } from '@/hooks/useWorkspaceVerification'

// =============================================================================
// Types
// =============================================================================

/**
 * Props for SetupVerification component (final step).
 * 
 * @property onValidationChange - Callback when step validation status changes (enables/disables Complete button)
 * @property onComplete - Optional callback when user clicks "Complete Setup" (typically switches to Azure logs view)
 * @property onNavigateToStep - Optional callback to navigate to a specific step (e.g., back to diagnostic settings)
 */
export interface SetupVerificationProps {
  onValidationChange: (isValid: boolean) => void
  onComplete?: () => void
  onNavigateToStep?: (step: string) => void
}

// =============================================================================
// Helper Components
// =============================================================================

interface ServiceResultCardProps {
  serviceName: string
  logCount: number
  lastLogTime?: string
  status: 'ok' | 'no-logs' | 'error'
  message?: string
  error?: string
}

function ServiceResultCard({ serviceName, logCount, lastLogTime, status, message, error }: Readonly<ServiceResultCardProps>) {
  const statusConfig = {
    ok: {
      Icon: CheckCircle,
      iconClassName: 'text-emerald-600 dark:text-emerald-400',
      bgClassName: 'bg-emerald-50 dark:bg-emerald-900/20',
      borderClassName: 'border-emerald-200 dark:border-emerald-800',
    },
    'no-logs': {
      Icon: AlertTriangle,
      iconClassName: 'text-orange-600 dark:text-orange-400',
      bgClassName: 'bg-orange-50 dark:bg-orange-900/20',
      borderClassName: 'border-orange-200 dark:border-orange-800',
    },
    error: {
      Icon: AlertTriangle,
      iconClassName: 'text-red-600 dark:text-red-400',
      bgClassName: 'bg-red-50 dark:bg-red-900/20',
      borderClassName: 'border-red-200 dark:border-red-800',
    },
  }[status]

  const { Icon, iconClassName, bgClassName, borderClassName } = statusConfig

  return (
    <div className={cn('rounded-lg border p-4', bgClassName, borderClassName)}>
      <div className="flex items-start gap-3">
        <Icon className={cn('w-5 h-5 shrink-0 mt-0.5', iconClassName)} />
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-1">
            {serviceName}
          </h4>
          
          {status === 'ok' && (
            <>
              <p className="text-sm text-slate-700 dark:text-slate-300">
                {logCount} log {logCount === 1 ? 'entry' : 'entries'} found in last 15 minutes
              </p>
              {lastLogTime && (
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-1">
                  Last log: {new Date(lastLogTime).toLocaleString()}
                </p>
              )}
            </>
          )}

          {status === 'no-logs' && (
            <>
              <p className="text-sm text-slate-700 dark:text-slate-300">
                No logs found in last 15 minutes
              </p>
              {message && (
                <p className="text-xs text-slate-600 dark:text-slate-400 mt-2">
                  {message}
                </p>
              )}
            </>
          )}

          {status === 'error' && (
            <>
              <p className="text-sm text-red-700 dark:text-red-300 font-medium">
                {error || message || 'Verification failed'}
              </p>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

// =============================================================================
// SetupVerification Component
// =============================================================================

export function SetupVerification({ 
  onValidationChange, 
  onComplete, 
  onNavigateToStep 
}: Readonly<SetupVerificationProps>) {
  const {
    isVerifying,
    error,
    status,
    results,
    guidance,
    servicesWithLogs,
    totalServices,
    allVerified,
    partiallyVerified,
    verify,
  } = useWorkspaceVerification()

  // Determine if step is valid (can proceed)
  // Valid if: not in error state, and either all/partial success
  const isValid = status === 'success' || status === 'partial'

  React.useEffect(() => {
    onValidationChange(isValid)
  }, [isValid, onValidationChange])

  const handleStartVerification = () => {
    void verify()
  }

  const handleRetry = () => {
    void verify()
  }

  const handleViewLogs = () => {
    onComplete?.()
  }

  const handleBackToDiagnosticSettings = () => {
    onNavigateToStep?.('diagnostic-settings')
  }

  // Idle state - waiting for user to start
  if (status === 'idle') {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
            Verification
          </h3>
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Test your workspace connection and log flow
          </p>
        </div>

        <div className="rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-6 text-center">
          <p className="text-sm text-slate-700 dark:text-slate-300 mb-4">
            Ready to verify your Azure logs setup. This will query your workspace for recent logs and verify that diagnostic settings are working.
          </p>
          <button
            type="button"
            onClick={handleStartVerification}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold shadow-sm',
              'bg-cyan-600 text-white hover:bg-cyan-700',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'dark:focus:ring-offset-slate-900',
              'transition-colors duration-150',
            )}
          >
            Start Verification
          </button>
        </div>
      </div>
    )
  }

  // Verifying state
  if (isVerifying) {
    return (
      <div className="p-8 flex flex-col items-center justify-center gap-3">
        <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        <p className="text-sm text-slate-600 dark:text-slate-400">Testing connection to Log Analytics workspace...</p>
        <p className="text-xs text-slate-500 dark:text-slate-500">This may take a few seconds...</p>
      </div>
    )
  }

  // Error state
  if (error || status === 'error') {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
            Verification
          </h3>
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Test your workspace connection and log flow
          </p>
        </div>

        <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-800 dark:text-red-300 mb-2">
                Verification failed
              </p>
              <p className="text-sm text-red-700 dark:text-red-400">
                {error || 'Could not verify workspace connection'}
              </p>

              {/* Show results if we have them even in error state */}
              {Object.keys(results).length > 0 && (
                <div className="mt-4 space-y-2">
                  {Object.entries(results).map(([serviceName, result]) => (
                    <ServiceResultCard
                      key={serviceName}
                      serviceName={serviceName}
                      logCount={result.logCount}
                      lastLogTime={result.lastLogTime}
                      status={result.status}
                      message={result.message}
                      error={result.error}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          <button
            type="button"
            onClick={handleRetry}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
              'border border-slate-300 dark:border-slate-600',
              'text-slate-700 dark:text-slate-300',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'transition-colors duration-150',
            )}
          >
            <RefreshCw className="w-4 h-4" />
            Retry
          </button>
          {onNavigateToStep && (
            <button
              type="button"
              onClick={handleBackToDiagnosticSettings}
              className={cn(
                'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
                'text-slate-600 dark:text-slate-400',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                'transition-colors duration-150',
              )}
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Diagnostic Settings
            </button>
          )}
        </div>
      </div>
    )
  }

  // Success state (all services verified)
  if (status === 'success' && allVerified) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
            Verification
          </h3>
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Test your workspace connection and log flow
          </p>
        </div>

        {/* Success Summary */}
        <div className="rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/20 p-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
                All {totalServices} {totalServices === 1 ? 'service' : 'services'} verified
              </p>
              <p className="text-xs text-emerald-700 dark:text-emerald-400 mt-1">
                Your Azure logs are flowing correctly
              </p>
            </div>
          </div>
        </div>

        {/* Service Results */}
        <div>
          <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Services</h4>
          <div className="space-y-3">
            {Object.entries(results).map(([serviceName, result]) => (
              <ServiceResultCard
                key={serviceName}
                serviceName={serviceName}
                logCount={result.logCount}
                lastLogTime={result.lastLogTime}
                status={result.status}
                message={result.message}
                error={result.error}
              />
            ))}
          </div>
        </div>

        {/* Guidance */}
        {guidance.length > 0 && (
          <div className="rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-4">
            <ul className="text-sm text-slate-700 dark:text-slate-300 space-y-1">
              {guidance.map((item, idx) => (
                <li key={idx}>• {item}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Complete Setup Success */}
        <div className="rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/20 p-6">
          <div className="flex items-start gap-4">
            <div className="flex-shrink-0">
              <div className="w-12 h-12 rounded-full bg-emerald-100 dark:bg-emerald-900/40 flex items-center justify-center">
                <Sparkles className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
              </div>
            </div>
            <div className="flex-1">
              <h4 className="text-base font-semibold text-emerald-900 dark:text-emerald-300 mb-2">
                Setup Complete! 🎉
              </h4>
              <p className="text-sm text-emerald-800 dark:text-emerald-400 mb-4">
                Your Azure logs integration is fully configured and logs are flowing. You can now view and analyze logs from your services.
              </p>
              <div className="flex items-center gap-3 flex-wrap">
                <button
                  type="button"
                  onClick={handleViewLogs}
                  className={cn(
                    'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold shadow-sm',
                    'bg-emerald-600 text-white hover:bg-emerald-700',
                    'focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2',
                    'dark:focus:ring-offset-slate-900',
                    'transition-colors duration-150',
                  )}
                >
                  View Logs
                </button>
                <button
                  type="button"
                  onClick={handleRetry}
                  className={cn(
                    'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
                    'border border-slate-300 dark:border-slate-600',
                    'text-slate-700 dark:text-slate-300',
                    'hover:bg-slate-100 dark:hover:bg-slate-800',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                    'transition-colors duration-150',
                  )}
                >
                  <RefreshCw className="w-4 h-4" />
                  Recheck
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Partial success state (some services have logs, some don't)
  if (status === 'partial' || (status === 'success' && partiallyVerified)) {
    return (
      <div className="p-6 space-y-6">
        <div>
          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
            Verification
          </h3>
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Test your workspace connection and log flow
          </p>
        </div>

        {/* Partial Summary */}
        <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-4">
          <div className="flex items-start gap-3">
            <AlertTriangle className="w-5 h-5 text-orange-600 dark:text-orange-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-orange-800 dark:text-orange-300">
                {servicesWithLogs} of {totalServices} {totalServices === 1 ? 'service' : 'services'} verified
              </p>
              <p className="text-xs text-orange-700 dark:text-orange-400 mt-1">
                Some services may not have generated logs yet
              </p>
            </div>
          </div>
        </div>

        {/* Service Results */}
        <div>
          <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Services</h4>
          <div className="space-y-3">
            {Object.entries(results).map(([serviceName, result]) => (
              <ServiceResultCard
                key={serviceName}
                serviceName={serviceName}
                logCount={result.logCount}
                lastLogTime={result.lastLogTime}
                status={result.status}
                message={result.message}
                error={result.error}
              />
            ))}
          </div>
        </div>

        {/* Guidance */}
        {guidance.length > 0 && (
          <div className="rounded-lg bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-4">
            <ul className="text-sm text-slate-700 dark:text-slate-300 space-y-1">
              {guidance.map((item, idx) => (
                <li key={idx}>• {item}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex items-center gap-3 flex-wrap">
          <button
            type="button"
            onClick={handleRetry}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
              'border border-slate-300 dark:border-slate-600',
              'text-slate-700 dark:text-slate-300',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'transition-colors duration-150',
            )}
          >
            <RefreshCw className="w-4 h-4" />
            Retry
          </button>
          {servicesWithLogs > 0 && (
            <button
              type="button"
              onClick={handleViewLogs}
              className={cn(
                'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'dark:focus:ring-offset-slate-900',
                'transition-colors duration-150',
              )}
            >
              View Logs Anyway
            </button>
          )}
          <button
            type="button"
            onClick={handleViewLogs}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
              'text-slate-600 dark:text-slate-400',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'transition-colors duration-150',
            )}
          >
            Complete Setup
          </button>
        </div>
      </div>
    )
  }

  // Default fallback (shouldn't reach here)
  return (
    <div className="p-6">
      <p className="text-sm text-slate-600 dark:text-slate-400">Unknown verification state</p>
    </div>
  )
}

export default SetupVerification
