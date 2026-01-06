/**
 * DiagnosticSettingsStep - Step 3 of Azure Logs Setup Guide
 * Shows aggregated diagnostic settings status for all Azure services
 */
import * as React from 'react'
import { CheckCircle, AlertTriangle, RefreshCw, Loader2, Circle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useDiagnosticSettings } from '@/hooks/useDiagnosticSettings'
import type { ServiceDiagnosticStatus } from '@/hooks/useDiagnosticSettings'
import { BicepTemplateModal } from './BicepTemplateModal'

// =============================================================================
// Types
// =============================================================================

/**
 * Props for DiagnosticSettingsStep component.
 * 
 * @property onValidationChange - Callback when step validation status changes (enables/disables Next button)
 */
export interface DiagnosticSettingsStepProps {
  onValidationChange: (isValid: boolean) => void
}

/**
 * Map of Azure resource types to human-readable names
 */
const RESOURCE_TYPE_NAMES: Record<string, string> = {
  'Microsoft.App/containerApps': 'Container Apps',
  'Microsoft.Web/sites': 'App Service',
  'Microsoft.Web/sites/functions': 'Azure Functions',
  'Microsoft.ContainerRegistry/registries': 'Container Registry',
  'Microsoft.Storage/storageAccounts': 'Storage Account',
  'Microsoft.KeyVault/vaults': 'Key Vault',
}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Get human-readable resource type name
 */
function getResourceTypeName(resourceType: string | undefined): string {
  if (!resourceType) return 'Unknown Service'
  return RESOURCE_TYPE_NAMES[resourceType] || resourceType
}

/**
 * Get status summary message based on configuration state
 */
function getStatusSummaryMessage(allConfigured: boolean, configuredCount: number, totalCount: number): string {
  if (allConfigured) {
    const serviceText = totalCount === 1 ? 'service is' : 'services are'
    return `All ${totalCount} ${serviceText} configured`
  }
  
  if (configuredCount === 0) {
    const serviceText = totalCount === 1 ? 'service needs' : 'services need'
    return `${totalCount} ${serviceText} configuration`
  }
  
  const serviceText = totalCount === 1 ? 'service needs' : 'services need'
  return `${totalCount - configuredCount} of ${totalCount} ${serviceText} configuration`
}

/**
 * Get CSS classes for status summary box
 */
function getStatusSummaryClasses(allConfigured: boolean): string {
  return cn(
    'rounded-lg border p-4',
    allConfigured
      ? 'border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-950/30'
      : 'border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-950/30'
  )
}

/**
 * Get CSS classes for status summary text
 */
function getStatusSummaryTextClasses(allConfigured: boolean): string {
  return cn(
    'text-sm font-medium',
    allConfigured
      ? 'text-emerald-800 dark:text-emerald-300'
      : 'text-orange-800 dark:text-orange-300'
  )
}

/**
 * Get CSS classes for status summary subtitle
 */
function getStatusSummarySubtitleClasses(allConfigured: boolean): string {
  return cn(
    'text-sm mt-1',
    allConfigured
      ? 'text-emerald-700 dark:text-emerald-400'
      : 'text-orange-700 dark:text-orange-400'
  )
}

/**
 * Extract resource type from resource ID
 * Example: /subscriptions/.../Microsoft.App/containerApps/my-app -> Microsoft.App/containerApps
 */
function extractResourceType(resourceId: string | undefined): string | undefined {
  if (!resourceId) return undefined
  
  // Match pattern: providers/{resourceType}/{resourceName}
  const pattern = /providers\/(Microsoft\.[^/]+\/[^/]+)/
  const match = pattern.exec(resourceId)
  return match?.[1]
}

// =============================================================================
// Helper Components
// =============================================================================

interface ServiceListItemProps {
  serviceName: string
  status: ServiceDiagnosticStatus
}

/**
 * Single service item in the list - shows name, type, and status icon
 */
function ServiceListItem({ serviceName, status }: Readonly<ServiceListItemProps>) {
  const resourceType = extractResourceType(status.resourceId)
  const resourceTypeName = getResourceTypeName(resourceType)
  
  return (
    <div className="flex items-center gap-3 py-2.5 border-b border-slate-200 dark:border-slate-700 last:border-0">
      {/* Status Icon */}
      {status.status === 'configured' && (
        <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0" />
      )}
      {status.status === 'not-configured' && (
        <Circle className="w-5 h-5 text-slate-400 dark:text-slate-600 shrink-0" />
      )}
      {status.status === 'error' && (
        <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 shrink-0" />
      )}
      
      {/* Service Info */}
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-slate-900 dark:text-slate-100 truncate">
          {serviceName}
        </div>
        <div className="text-sm text-slate-600 dark:text-slate-400">
          {resourceTypeName}
        </div>
      </div>
    </div>
  )
}

// =============================================================================
// Main Component
// =============================================================================

export function DiagnosticSettingsStep({ onValidationChange }: Readonly<DiagnosticSettingsStepProps>) {
  const {
    isLoading,
    isRefreshing,
    error,
    services,
    recheck,
    allConfigured,
    configuredCount,
    totalCount,
  } = useDiagnosticSettings()

  // Modal state
  const [isBicepModalOpen, setIsBicepModalOpen] = React.useState(false)

  // Update validation state whenever status changes
  React.useEffect(() => {
    onValidationChange(allConfigured)
  }, [allConfigured, onValidationChange])

  // Loading State
  if (isLoading) {
    return (
      <div className="p-8 flex flex-col items-center justify-center gap-3">
        <Loader2 className="w-8 h-8 animate-spin text-cyan-500" />
        <p className="text-sm text-slate-600 dark:text-slate-400">Checking diagnostic settings...</p>
      </div>
    )
  }

  // Error State
  if (error) {
    return (
      <div className="p-6">
        <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4">
          <div className="flex items-start gap-3 mb-3">
            <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-800 dark:text-red-300">
                Could not check diagnostic settings
              </p>
              <p className="text-sm text-red-700 dark:text-red-400 mt-1">{error}</p>
            </div>
          </div>
          
          <div className="mt-3 space-y-2">
            <p className="text-sm font-semibold text-red-800 dark:text-red-300">Troubleshooting:</p>
            <ul className="text-sm text-red-700 dark:text-red-400 space-y-1 ml-4 list-disc">
              <li>Ensure you have Reader role on resources</li>
              <li>Verify <code className="px-1.5 py-0.5 rounded bg-red-100 dark:bg-red-900 text-red-900 dark:text-red-100 font-mono text-xs">azd auth login</code> is current</li>
              <li>Check network connectivity</li>
            </ul>
          </div>

          <div className="mt-4 flex gap-2">
            <button
              type="button"
              onClick={() => void recheck()}
              disabled={isRefreshing}
              className={cn(
                'inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
                'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300',
                'hover:bg-red-200 dark:hover:bg-red-800',
                'focus:outline-none focus:ring-2 focus:ring-red-500',
                'disabled:opacity-50 disabled:cursor-not-allowed',
              )}
            >
              <RefreshCw className={cn('w-3.5 h-3.5', isRefreshing && 'animate-spin')} />
              Retry
            </button>
            <button
              type="button"
              onClick={() => onValidationChange(true)}
              className={cn(
                'inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
                'text-red-700 dark:text-red-300',
                'hover:bg-red-100 dark:hover:bg-red-900',
                'focus:outline-none focus:ring-2 focus:ring-red-500',
              )}
            >
              Skip This Step →
            </button>
          </div>
        </div>
      </div>
    )
  }

  // No Services Found
  if (totalCount === 0) {
    return (
      <div className="p-6">
        <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50 p-6 text-center">
          <AlertTriangle className="w-8 h-8 text-amber-500 mx-auto mb-3" />
          <p className="text-sm font-medium text-slate-900 dark:text-slate-100">No services found</p>
          <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
            No Azure services were discovered in your project. Deploy your infrastructure first.
          </p>
          <button
            type="button"
            onClick={() => void recheck()}
            disabled={isRefreshing}
            className={cn(
              'mt-4 inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium',
              'text-slate-700 dark:text-slate-300',
              'border border-slate-200 dark:border-slate-700',
              'hover:bg-slate-100 dark:hover:bg-slate-800',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500',
              'disabled:opacity-50 disabled:cursor-not-allowed',
            )}
          >
            <RefreshCw className={cn('w-3.5 h-3.5', isRefreshing && 'animate-spin')} />
            Recheck
          </button>
        </div>
      </div>
    )
  }

  // Convert services object to array for rendering
  const serviceList = Object.entries(services).map(([name, status]) => ({
    name,
    status,
  }))

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div>
        <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100 mb-1">
          Diagnostic Settings
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Configure logging for your Azure services
        </p>
      </div>

      {/* Status Summary */}
      <div className={getStatusSummaryClasses(allConfigured)}>
        <div className="flex items-start gap-3">
          {allConfigured ? (
            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
          ) : (
            <AlertTriangle className="w-5 h-5 text-orange-600 dark:text-orange-400 shrink-0 mt-0.5" />
          )}
          <div className="flex-1">
            <p className={getStatusSummaryTextClasses(allConfigured)}>
              {getStatusSummaryMessage(allConfigured, configuredCount, totalCount)}
            </p>
            <p className={getStatusSummarySubtitleClasses(allConfigured)}>
              {allConfigured
                ? 'Your diagnostic settings are ready'
                : 'Diagnostic settings required for logs'}
            </p>
          </div>
        </div>
      </div>

      {/* Services List */}
      <div>
        <h4 className="text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">Services</h4>
        <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 px-4">
          {serviceList.map(({ name, status }) => (
            <ServiceListItem key={name} serviceName={name} status={status} />
          ))}
        </div>
      </div>

      {/* Instructions (if not all configured) */}
      {!allConfigured && (
        <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-950/20 p-4">
          <p className="text-sm font-semibold text-orange-800 dark:text-orange-300 mb-2">How to fix:</p>
          <ol className="text-sm text-orange-700 dark:text-orange-400 space-y-1.5 ml-4 list-decimal">
            <li>Click <strong>"Show Bicep Template"</strong> below</li>
            <li>Copy the template to your infrastructure files</li>
            <li>Run <code className="px-1.5 py-0.5 rounded bg-orange-100 dark:bg-orange-900 text-orange-900 dark:text-orange-100 font-mono text-xs">azd up</code> to deploy the changes</li>
            <li>Click <strong>"Recheck"</strong> to verify the configuration</li>
          </ol>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex flex-wrap items-center gap-3">
        {!allConfigured && (
          <button
            type="button"
            onClick={() => setIsBicepModalOpen(true)}
            className={cn(
              'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
              'bg-cyan-600 text-white hover:bg-cyan-700',
              'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
              'transition-colors duration-150',
            )}
          >
            Show Bicep Template →
          </button>
        )}
        
        <button
          type="button"
          onClick={() => void recheck()}
          disabled={isRefreshing}
          className={cn(
            'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium',
            'text-slate-700 dark:text-slate-300',
            'border border-slate-200 dark:border-slate-700',
            'hover:bg-slate-100 dark:hover:bg-slate-800',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            'transition-colors duration-150',
          )}
        >
          <RefreshCw className={cn('w-4 h-4', isRefreshing && 'animate-spin')} />
          Recheck
        </button>
      </div>

      {/* Bicep Template Modal */}
      <BicepTemplateModal
        isOpen={isBicepModalOpen}
        onClose={() => setIsBicepModalOpen(false)}
        services={Object.keys(services)}
      />

      {/* Success Message */}
      {allConfigured && (
        <div className="rounded-lg border border-emerald-200 dark:border-emerald-800 bg-emerald-50 dark:bg-emerald-900/20 p-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-emerald-800 dark:text-emerald-300">
                All services configured!
              </p>
              <p className="text-sm text-emerald-700 dark:text-emerald-400 mt-1">
                Diagnostic settings are enabled for all {totalCount} {totalCount === 1 ? 'service' : 'services'}. Click "Next" to verify connectivity.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default DiagnosticSettingsStep
