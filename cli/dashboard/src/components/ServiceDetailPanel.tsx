/**
 * ServiceDetailPanel - Slide-over detail panel with modern styling
 * Follows design spec: cli/dashboard/design/components/detail-panel.md
 */
import * as React from 'react'
import { 
  X, 
  ExternalLink, 
  Eye, 
  EyeOff, 
  Copy, 
  Check, 
  Lock,
  Server,
  Globe,
  Cloud,
  Settings2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useClipboard } from '@/hooks/useClipboard'
import { StatusBadge, type EffectiveStatus } from './StatusIndicator'
import { ServiceActions } from '@/components/ServiceActions'
import { useServiceOperations, type OperationState } from '@/hooks/useServiceOperations'
import type { Service, HealthCheckResult } from '@/types'
import { 
  formatUptime, 
  formatResponseTime,
  getCheckTypeDisplay,
  getEffectiveStatus as getEffectiveStatusFromUtils,
  getStatusDisplay,
  isProcessService,
  getServiceModeBadgeConfig,
  getServiceTypeLabel,
} from '@/lib/service-utils'

// =============================================================================
// Types
// =============================================================================

export interface ServiceDetailPanelProps {
  /** Service to display */
  service: Service | null
  /** Whether panel is open */
  isOpen: boolean
  /** Close handler */
  onClose: () => void
  /** Health status */
  healthStatus?: HealthCheckResult
  /** Additional class names */
  className?: string
}

type TabId = 'overview' | 'local' | 'azure' | 'environment'

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Convert service status + health to EffectiveStatus for UI components.
 * Uses the centralized getEffectiveStatus from service-utils for consistency,
 * then converts to the EffectiveStatus type used by components.
 */
function getEffectiveStatusForUI(
  service: Service, 
  healthStatus?: HealthCheckResult,
  operationState?: OperationState
): EffectiveStatus {
  // Use centralized getEffectiveStatus from service-utils (same as classic view)
  const { status, health } = getEffectiveStatusFromUtils(service, operationState)
  
  // Get the display info which normalizes status/health combinations
  const statusDisplay = getStatusDisplay(status, health)
  
  // Use real-time health from health stream if available and not in operation
  // Backend may send 'starting' but normalizeHealthStatus already handles it
  const effectiveHealth = (operationState === 'idle' || !operationState)
    ? (healthStatus?.status ?? health)
    : health
  
  // Map to EffectiveStatus based on display text (which is already normalized)
  const displayText = statusDisplay.text.toLowerCase()
  
  // Process service specific statuses
  if (status === 'watching') return 'watching'
  if (status === 'building') return 'building'
  if (status === 'built') return 'built'
  if (status === 'completed') return 'completed'
  if (status === 'failed') return 'failed'
  
  // Standard status mappings (lifecycle state takes priority)
  if (displayText === 'stopped') return 'stopped'
  if (displayText === 'stopping') return 'stopping'
  if (displayText === 'starting' || displayText === 'restarting') return 'starting'
  if (displayText === 'restarting') return 'restarting'
  if (displayText === 'error') return 'error'
  if (displayText === 'not running') return 'not-running'
  
  // Health-based statuses (when service is running)
  // Note: 'starting' health is normalized to 'unknown', so it won't match here
  if (effectiveHealth === 'healthy' || displayText === 'running') return 'healthy'
  if (effectiveHealth === 'degraded' || displayText === 'degraded') return 'degraded'
  if (effectiveHealth === 'unhealthy' || displayText === 'unhealthy') return 'unhealthy'
  
  return 'unknown'
}

function hasAzureDeployment(service: Service): boolean {
  return !!(service.azure?.url || service.azure?.resourceName || service.azure?.resourceType)
}

function formatTimestamp(dateStr: string | undefined): string {
  if (!dateStr) return '-'
  try {
    return new Date(dateStr).toLocaleString()
  } catch {
    return dateStr
  }
}

function buildAzurePortalUrl(service: Service): string | null {
  if (!service.azure?.subscriptionId || !service.azure?.resourceGroup || !service.azure?.resourceName) {
    return null
  }
  const resourceType = service.azure.resourceType || 'containerapp'
  const provider = resourceType === 'containerapp' 
    ? 'Microsoft.App/containerApps'
    : 'Microsoft.Web/sites'
  
  return `https://portal.azure.com/#@/resource/subscriptions/${service.azure.subscriptionId}/resourceGroups/${service.azure.resourceGroup}/providers/${provider}/${service.azure.resourceName}`
}

const SENSITIVE_PATTERNS = [
  /secret/i, /password/i, /key/i, /token/i, /auth/i, /credential/i,
  /private/i, /cert/i, /api[-_]?key/i, /access[-_]?key/i
]

function isSensitiveKey(key: string): boolean {
  return SENSITIVE_PATTERNS.some(pattern => pattern.test(key))
}

// =============================================================================
// Section Card Component
// =============================================================================

interface SectionCardProps {
  title: string
  children: React.ReactNode
  action?: React.ReactNode
}

function SectionCard({ title, children, action }: SectionCardProps) {
  return (
    <div className="mb-4 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2.5 bg-slate-100 dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <h4 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
          {title}
        </h4>
        {action}
      </div>
      <div className="p-4">{children}</div>
    </div>
  )
}

// =============================================================================
// Info Row Component
// =============================================================================

interface InfoRowProps {
  label: string
  value: React.ReactNode
  copyable?: boolean
  onCopy?: () => void
  copied?: boolean
}

function InfoRow({ label, value, copyable, onCopy, copied }: InfoRowProps) {
  return (
    <div className="flex justify-between items-start py-2 border-b border-slate-200 dark:border-slate-700 last:border-b-0 group">
      <span className="text-sm text-slate-500 dark:text-slate-400 shrink-0">{label}</span>
      <div className="flex items-center gap-2 text-right">
        <span className="text-sm font-medium text-slate-900 dark:text-slate-100 wrap-break-word max-w-[250px]">
          {value}
        </span>
        {copyable && onCopy && (
          <button
            type="button"
            onClick={onCopy}
            className={cn(
              'p-1 rounded opacity-0 group-hover:opacity-100 transition-opacity',
              'hover:bg-slate-200 dark:hover:bg-slate-700',
              copied && 'text-emerald-600 dark:text-emerald-400 opacity-100'
            )}
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
        )}
      </div>
    </div>
  )
}

// =============================================================================
// Tab Components
// =============================================================================

interface OverviewTabProps {
  service: Service
  healthStatus?: HealthCheckResult
  operationState?: OperationState
}

function OverviewTab({ service, healthStatus, operationState }: OverviewTabProps) {
  const effectiveStatus = getEffectiveStatusForUI(service, healthStatus, operationState)
  const localUrl = service.local?.url && !service.local.url.match(/:0\/?$/) ? service.local.url : null
  const isDeployed = hasAzureDeployment(service)
  const azurePortalUrl = buildAzurePortalUrl(service)
  
  // Process service detection
  const serviceType = service.local?.serviceType
  const serviceMode = service.local?.serviceMode
  const isProcess = isProcessService(serviceType)
  const modeBadgeConfig = serviceMode ? getServiceModeBadgeConfig(serviceMode) : null

  return (
    <div>
      {/* Service Type (for process services) */}
      {isProcess && (
        <SectionCard title="Service Type">
          <div className="space-y-0">
            <InfoRow label="Type" value={getServiceTypeLabel(serviceType)} />
            {serviceMode && (
              <InfoRow 
                label="Mode" 
                value={
                  <span className="flex items-center gap-2">
                    {modeBadgeConfig && (
                      <span className={cn(
                        'inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold',
                        modeBadgeConfig.color
                      )}>
                        {modeBadgeConfig.label}
                      </span>
                    )}
                    <span className="text-slate-500 dark:text-slate-400 text-xs">
                      {modeBadgeConfig?.description}
                    </span>
                  </span>
                } 
              />
            )}
          </div>
        </SectionCard>
      )}

      {/* Local Development */}
      <SectionCard title="Local Development">
        <div className="space-y-0">
          <InfoRow 
            label="Status" 
            value={<StatusBadge status={effectiveStatus} />} 
          />
          {localUrl && (
            <InfoRow 
              label="URL" 
              value={
                <a 
                  href={localUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-cyan-600 dark:text-cyan-400 hover:underline flex items-center gap-1"
                >
                  {localUrl}
                  <ExternalLink className="w-3 h-3" />
                </a>
              } 
            />
          )}
          {!isProcess && service.local?.port && service.local.port > 0 && (
            <InfoRow label="Port" value={service.local.port} />
          )}
        </div>
      </SectionCard>

      {/* Azure Deployment */}
      <SectionCard title="Azure Deployment">
        {isDeployed ? (
          <div className="space-y-0">
            <InfoRow 
              label="Status" 
              value={<span className="text-emerald-600 dark:text-emerald-400">● Deployed</span>} 
            />
            {service.azure?.resourceName && (
              <InfoRow label="Resource" value={service.azure.resourceName} />
            )}
            {service.azure?.url && (
              <InfoRow 
                label="Endpoint" 
                value={
                  <a 
                    href={service.azure.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-cyan-600 dark:text-cyan-400 hover:underline flex items-center gap-1 break-all"
                  >
                    {service.azure.url}
                    <ExternalLink className="w-3 h-3 shrink-0" />
                  </a>
                } 
              />
            )}
            {azurePortalUrl && (
              <div className="mt-4">
                <a
                  href={azurePortalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-700 text-white rounded-lg text-sm font-medium transition-colors"
                >
                  Open in Azure Portal
                  <ExternalLink className="w-4 h-4" />
                </a>
              </div>
            )}
          </div>
        ) : (
          <p className="text-sm text-slate-500 dark:text-slate-400">Not deployed to Azure</p>
        )}
      </SectionCard>
    </div>
  )
}

interface LocalTabProps {
  service: Service
  healthStatus?: HealthCheckResult
  copiedField: string | null
  onCopy: (text: string, field: string) => void
  operationState?: OperationState
}

function LocalTab({ service, healthStatus, copiedField, onCopy, operationState }: LocalTabProps) {
  const effectiveStatus = getEffectiveStatusForUI(service, healthStatus, operationState)
  const localUrl = service.local?.url && !service.local.url.match(/:0\/?$/) ? service.local.url : null

  return (
    <div>
      {/* Service Details */}
      <SectionCard title="Service Details">
        <div className="space-y-0">
          <InfoRow 
            label="Name" 
            value={service.name} 
            copyable 
            onCopy={() => onCopy(service.name, 'name')}
            copied={copiedField === 'name'}
          />
          {service.language && (
            <InfoRow label="Language" value={service.language} />
          )}
          {service.framework && (
            <InfoRow label="Framework" value={service.framework} />
          )}
          {service.project && (
            <InfoRow 
              label="Project" 
              value={service.project} 
              copyable
              onCopy={() => onCopy(service.project!, 'project')}
              copied={copiedField === 'project'}
            />
          )}
        </div>
      </SectionCard>

      {/* Runtime */}
      <SectionCard title="Runtime">
        <div className="space-y-0">
          <InfoRow 
            label="Status" 
            value={<StatusBadge status={effectiveStatus} />} 
          />
          {service.local?.pid && (
            <InfoRow label="PID" value={service.local.pid} />
          )}
          {service.local?.port && service.local.port > 0 && (
            <InfoRow label="Port" value={service.local.port} />
          )}
          {localUrl && (
            <InfoRow 
              label="URL" 
              value={localUrl}
              copyable
              onCopy={() => onCopy(localUrl, 'url')}
              copied={copiedField === 'url'}
            />
          )}
        </div>
      </SectionCard>

      {/* Timing */}
      <SectionCard title="Timing">
        <div className="space-y-0">
          {service.local?.startTime && (
            <>
              <InfoRow label="Started" value={formatTimestamp(service.local.startTime)} />
              <InfoRow label="Uptime" value={formatUptime(new Date(service.local.startTime).getTime() * 1_000_000)} />
            </>
          )}
          {healthStatus?.responseTime !== undefined && (
            <InfoRow 
              label="Response Time" 
              value={formatResponseTime(healthStatus.responseTime)} 
            />
          )}
        </div>
      </SectionCard>

      {/* Health Details */}
      {healthStatus && (
        <SectionCard title="Health Details">
          <div className="space-y-0">
            <InfoRow label="Check Type" value={getCheckTypeDisplay(healthStatus.checkType)} />
            {healthStatus.endpoint && (
              <InfoRow label="Endpoint" value={healthStatus.endpoint} />
            )}
            {healthStatus.statusCode && (
              <InfoRow label="Status Code" value={healthStatus.statusCode} />
            )}
          </div>
        </SectionCard>
      )}
    </div>
  )
}

interface AzureTabProps {
  service: Service
}

function AzureTab({ service }: AzureTabProps) {
  const isDeployed = hasAzureDeployment(service)
  const azurePortalUrl = buildAzurePortalUrl(service)

  if (!isDeployed) {
    return (
      <SectionCard title="Not Deployed">
        <div className="text-center py-6">
          <Cloud className="w-10 h-10 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
          <p className="text-sm text-slate-500 dark:text-slate-400 mb-2">
            This service hasn't been deployed to Azure yet.
          </p>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            Run <code className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-700 rounded text-xs font-mono">azd deploy</code> to deploy.
          </p>
        </div>
      </SectionCard>
    )
  }

  return (
    <div>
      {/* Resource */}
      <SectionCard title="Resource">
        <div className="space-y-0">
          {service.azure?.resourceName && (
            <InfoRow label="Resource Name" value={service.azure.resourceName} />
          )}
          {service.azure?.resourceType && (
            <InfoRow label="Resource Type" value={service.azure.resourceType} />
          )}
          {service.azure?.imageName && (
            <InfoRow label="Image" value={service.azure.imageName} />
          )}
          {service.azure?.url && (
            <InfoRow 
              label="Endpoint" 
              value={
                <a 
                  href={service.azure.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-cyan-600 dark:text-cyan-400 hover:underline flex items-center gap-1 break-all"
                >
                  {service.azure.url}
                  <ExternalLink className="w-3 h-3 shrink-0" />
                </a>
              } 
            />
          )}
        </div>
      </SectionCard>

      {/* Azure Metadata */}
      {(service.azure?.subscriptionId || service.azure?.resourceGroup || service.azure?.location) && (
        <SectionCard title="Azure Metadata">
          <div className="space-y-0">
            {service.azure?.subscriptionId && (
              <InfoRow label="Subscription" value={service.azure.subscriptionId} />
            )}
            {service.azure?.resourceGroup && (
              <InfoRow label="Resource Group" value={service.azure.resourceGroup} />
            )}
            {service.azure?.location && (
              <InfoRow label="Location" value={service.azure.location} />
            )}
          </div>
        </SectionCard>
      )}

      {/* Actions */}
      {azurePortalUrl && (
        <div className="mt-4">
          <a
            href={azurePortalUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-4 py-2 bg-cyan-600 hover:bg-cyan-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            Open in Azure Portal
            <ExternalLink className="w-4 h-4" />
          </a>
        </div>
      )}
    </div>
  )
}

interface EnvironmentTabProps {
  service: Service
  copiedField: string | null
  onCopy: (text: string, field: string) => void
}

function EnvironmentTab({ service, copiedField, onCopy }: EnvironmentTabProps) {
  const [showValues, setShowValues] = React.useState(false)
  const envVars = service.environmentVariables || {}
  const envEntries = Object.entries(envVars).sort(([a], [b]) => a.localeCompare(b))

  if (envEntries.length === 0) {
    return (
      <SectionCard title="Environment Variables">
        <p className="text-sm text-slate-500 dark:text-slate-400 text-center py-4">
          No environment variables configured
        </p>
      </SectionCard>
    )
  }

  return (
    <SectionCard 
      title={`Environment Variables (${envEntries.length})`}
      action={
        <button
          type="button"
          onClick={() => setShowValues(!showValues)}
          className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
        >
          {showValues ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
          {showValues ? 'Hide' : 'Show'}
        </button>
      }
    >
      <div className="space-y-3">
        {envEntries.map(([key, value]) => {
          const isSensitive = isSensitiveKey(key)
          const displayValue = isSensitive && !showValues ? '••••••••••••' : value
          const copied = copiedField === `env-${key}`

          return (
            <div key={key} className="space-y-1">
              <div className="flex items-center gap-1">
                {isSensitive && (
                  <Lock className="w-3 h-3 text-amber-500" />
                )}
                <span className="text-xs text-slate-500 dark:text-slate-400 font-mono">
                  {key}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={displayValue}
                  readOnly
                  className="flex-1 px-2 py-1.5 text-sm font-mono bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded text-slate-600 dark:text-slate-300"
                />
                <button
                  type="button"
                  onClick={() => onCopy(value, `env-${key}`)}
                  className={cn(
                    'p-1.5 rounded hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors',
                    copied && 'text-emerald-600 dark:text-emerald-400'
                  )}
                >
                  {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </SectionCard>
  )
}

// =============================================================================
// ServiceDetailPanel Component
// =============================================================================

export function ServiceDetailPanel({
  service,
  isOpen,
  onClose,
  healthStatus,
  className,
}: ServiceDetailPanelProps) {
  const [activeTab, setActiveTab] = React.useState<TabId>('overview')
  const panelRef = React.useRef<HTMLDivElement>(null)
  const { copiedField, copyToClipboard } = useClipboard()
  const { getOperationState, isBulkOperationInProgress, bulkOperation } = useServiceOperations()
  const individualOpState = service ? getOperationState(service.name) : undefined
  
  // During bulk operations, derive operation state from bulk operation type
  // This ensures services show transitional states (stopping/starting) during bulk ops
  const operationState: OperationState | undefined = individualOpState !== 'idle' && individualOpState !== undefined
    ? individualOpState 
    : isBulkOperationInProgress() && bulkOperation
    ? (bulkOperation === 'stop' ? 'stopping' : bulkOperation === 'start' ? 'starting' : 'restarting')
    : individualOpState

  useEscapeKey(onClose, isOpen)

  // Reset tab when service changes
  React.useEffect(() => {
    if (service) {
      setActiveTab('overview')
    }
  }, [service])

  // Focus management
  React.useEffect(() => {
    if (isOpen && panelRef.current) {
      const closeButton = panelRef.current.querySelector<HTMLButtonElement>('[data-close-button]')
      closeButton?.focus()
    }
  }, [isOpen])

  if (!isOpen || !service) {
    return null
  }

  const handleCopy = (text: string, field: string) => {
    void copyToClipboard(text, field)
  }

  const effectiveStatus = getEffectiveStatusForUI(service, healthStatus, operationState)

  const tabs: { id: TabId; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { id: 'overview', label: 'Overview', icon: Server },
    { id: 'local', label: 'Local', icon: Globe },
    { id: 'azure', label: 'Azure', icon: Cloud },
    { id: 'environment', label: 'Environment', icon: Settings2 },
  ]

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="panel-title"
        className={cn(
          'fixed right-0 top-0 z-50 h-screen w-full max-w-[520px]',
          'bg-white dark:bg-slate-900',
          'border-l border-slate-200 dark:border-slate-700',
          'shadow-2xl',
          'flex flex-col',
          'animate-slide-in-right',
          className
        )}
      >
        {/* Header */}
        <div className="flex items-start gap-3 p-5 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div className={cn(
            'w-11 h-11 rounded-xl flex items-center justify-center shrink-0',
            effectiveStatus === 'healthy' 
              ? 'bg-emerald-100 dark:bg-emerald-500/20'
              : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
              ? 'bg-rose-100 dark:bg-rose-500/20'
              : 'bg-slate-100 dark:bg-slate-700'
          )}>
            <Server className={cn(
              'w-5 h-5',
              effectiveStatus === 'healthy' 
                ? 'text-emerald-600 dark:text-emerald-400'
                : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
                ? 'text-rose-600 dark:text-rose-400'
                : 'text-slate-500 dark:text-slate-400'
            )} />
          </div>
          <div className="flex-1 min-w-0">
            <h2
              id="panel-title"
              className="text-xl font-semibold text-slate-900 dark:text-slate-100 truncate"
            >
              {service.name}
            </h2>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {service.framework || service.language || 'Service'}
            </p>
          </div>
          <button
            type="button"
            data-close-button
            onClick={onClose}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close panel"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex gap-6 px-5 border-b border-slate-200 dark:border-slate-700 shrink-0">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={activeTab === tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'relative py-3 text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'text-slate-900 dark:text-slate-100'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
              )}
            >
              {tab.label}
              {activeTab === tab.id && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-cyan-500 dark:bg-cyan-400 rounded-full" />
              )}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-5">
          {activeTab === 'overview' && (
            <OverviewTab service={service} healthStatus={healthStatus} operationState={operationState} />
          )}
          {activeTab === 'local' && (
            <LocalTab 
              service={service} 
              healthStatus={healthStatus}
              copiedField={copiedField}
              onCopy={handleCopy}
              operationState={operationState}
            />
          )}
          {activeTab === 'azure' && (
            <AzureTab service={service} />
          )}
          {activeTab === 'environment' && (
            <EnvironmentTab 
              service={service}
              copiedField={copiedField}
              onCopy={handleCopy}
            />
          )}
        </div>

        {/* Actions Footer */}
        <div className="flex items-center gap-2 p-4 border-t border-slate-200 dark:border-slate-700 shrink-0">
          <ServiceActions service={service} variant="default" />
        </div>
      </div>
    </>
  )
}
