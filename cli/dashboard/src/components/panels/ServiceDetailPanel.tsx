/**
 * ServiceDetailPanel - Slide-in panel displaying comprehensive service details
 */
import * as React from 'react'
import { X, ExternalLink, Eye, EyeOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useClipboard } from '@/hooks/useClipboard'
import { InfoField } from '@/components/ui/InfoField'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import type { Service, HealthCheckResult } from '@/types'
import {
  formatUptime,
  formatTimestamp,
  getStatusColor,
  getHealthColor,
  buildAzurePortalUrl,
  isSensitiveKey,
  maskValue,
  formatResourceType,
  getStatusDisplay,
  getHealthDisplay,
  formatCheckType,
  hasAzureDeployment,
} from '@/lib/panel-utils'

// =============================================================================
// Types
// =============================================================================

export interface ServiceDetailPanelProps {
  /** Service to display details for */
  service: Service | null
  /** Whether the panel is open */
  isOpen: boolean
  /** Callback when panel should close */
  onClose: () => void
  /** Health check result for real-time status */
  healthStatus?: HealthCheckResult
  /** Additional class names */
  className?: string
  /** Data test ID for testing */
  'data-testid'?: string
}

type DetailTab = 'overview' | 'local' | 'azure' | 'environment'

// =============================================================================
// Section Card Component
// =============================================================================

interface SectionCardProps {
  title: string
  children: React.ReactNode
  'data-testid'?: string
}

function SectionCard({ title, children, 'data-testid': testId }: SectionCardProps) {
  return (
    <div
      className="p-4 rounded-lg border border-border bg-card mb-4"
      data-testid={testId}
    >
      <h4 className="text-sm font-medium text-muted-foreground uppercase tracking-wider mb-3">
        {title}
      </h4>
      {children}
    </div>
  )
}

// =============================================================================
// Overview Tab
// =============================================================================

interface OverviewTabProps {
  service: Service
  healthStatus?: HealthCheckResult
}

function OverviewTab({ service, healthStatus }: OverviewTabProps) {
  const statusDisplay = getStatusDisplay(service.local?.status || service.status)
  const healthDisplay = getHealthDisplay(service.local?.health || service.health)
  // Build local URL, filtering out port 0 URLs (e.g., localhost:0)
  const rawUrl = service.local?.url || (service.local?.port && service.local.port > 0 ? `http://localhost:${service.local.port}` : null)
  const localUrl = rawUrl && !rawUrl.match(/:0\/?$/) ? rawUrl : null
  const isAzureDeployed = hasAzureDeployment(service)
  const azurePortalUrl = buildAzurePortalUrl(service)

  return (
    <div data-testid="overview-tab-content">
      {/* Local Development Section */}
      <SectionCard title="Local Development" data-testid="local-development-section">
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-sm text-muted-foreground">Status</span>
            <span className={cn('text-sm font-medium', getStatusColor(service.local?.status || service.status))}>
              {statusDisplay.indicator} {statusDisplay.text}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-sm text-muted-foreground">Health</span>
            <span className={cn('text-sm font-medium', getHealthColor(healthStatus?.status || service.local?.health || service.health))}>
              {healthDisplay.indicator} {healthDisplay.text}
            </span>
          </div>
          {localUrl && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">URL</span>
              <a
                href={localUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-primary hover:underline flex items-center gap-1"
              >
                {localUrl}
                <ExternalLink className="h-3 w-3" />
              </a>
            </div>
          )}
          {service.local?.port && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Port</span>
              <span className="text-sm font-medium text-foreground">{service.local.port}</span>
            </div>
          )}
        </div>
      </SectionCard>

      {/* Azure Deployment Section */}
      <SectionCard title="Azure Deployment" data-testid="azure-deployment-section">
        {isAzureDeployed ? (
          <div className="space-y-2">
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Status</span>
              <span className="text-sm font-medium text-green-500">● Deployed</span>
            </div>
            {service.azure?.resourceName && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-muted-foreground">Resource</span>
                <span className="text-sm font-medium text-foreground">{service.azure.resourceName}</span>
              </div>
            )}
            {service.azure?.url && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-muted-foreground">Endpoint</span>
                <a
                  href={service.azure.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-primary hover:underline flex items-center gap-1"
                >
                  {service.azure.url}
                  <ExternalLink className="h-3 w-3" />
                </a>
              </div>
            )}
            {azurePortalUrl && (
              <div className="mt-3">
                <a
                  href={azurePortalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 text-sm text-primary hover:underline"
                >
                  Open in Azure Portal
                  <ExternalLink className="h-3 w-3" />
                </a>
              </div>
            )}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">Not deployed to Azure</p>
        )}
      </SectionCard>
    </div>
  )
}

// =============================================================================
// Local Tab
// =============================================================================

interface LocalTabProps {
  service: Service
  healthStatus?: HealthCheckResult
}

function LocalTab({ service, healthStatus }: LocalTabProps) {
  const statusDisplay = getStatusDisplay(service.local?.status || service.status)
  const healthDisplay = getHealthDisplay(healthStatus?.status || service.local?.health || service.health)
  // Build local URL, filtering out port 0 URLs (e.g., localhost:0)
  const rawUrl = service.local?.url || (service.local?.port && service.local.port > 0 ? `http://localhost:${service.local.port}` : null)
  const localUrl = rawUrl && !rawUrl.match(/:0\/?$/) ? rawUrl : null

  return (
    <div data-testid="local-tab-content">
      {/* Service Details */}
      <SectionCard title="Service Details" data-testid="service-details-section">
        <div className="space-y-2">
          <InfoField
            label="Name"
            value={service.name}
            copyable
          />
          {service.language && (
            <InfoField
              label="Language"
              value={service.language}
              copyable
            />
          )}
          {service.framework && (
            <InfoField
              label="Framework"
              value={service.framework}
              copyable
            />
          )}
          {service.project && (
            <InfoField
              label="Project"
              value={service.project}
              copyable
            />
          )}
        </div>
      </SectionCard>

      {/* Runtime */}
      <SectionCard title="Runtime" data-testid="runtime-section">
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <span className="text-sm text-muted-foreground">Status</span>
            <span className={cn('text-sm font-medium', getStatusColor(service.local?.status || service.status))}>
              {statusDisplay.indicator} {statusDisplay.text}
            </span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-sm text-muted-foreground">Health</span>
            <span className={cn('text-sm font-medium', getHealthColor(healthStatus?.status || service.local?.health || service.health))}>
              {healthDisplay.indicator} {healthDisplay.text}
            </span>
          </div>
          {service.local?.pid && (
            <InfoField
              label="PID"
              value={String(service.local.pid)}
              copyable
            />
          )}
          {service.local?.port && (
            <InfoField
              label="Port"
              value={String(service.local.port)}
              copyable
            />
          )}
          {localUrl && (
            <InfoField
              label="URL"
              value={localUrl}
              copyable
            />
          )}
        </div>
      </SectionCard>

      {/* Timing */}
      <SectionCard title="Timing" data-testid="timing-section">
        <div className="space-y-2">
          {service.local?.startTime && (
            <InfoField
              label="Started"
              value={formatTimestamp(service.local.startTime)}
              copyable
            />
          )}
          {service.local?.startTime && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Uptime</span>
              <span className="text-sm font-medium text-foreground">
                {formatUptime(service.local.startTime)}
              </span>
            </div>
          )}
          {service.local?.lastChecked && (
            <InfoField
              label="Last Checked"
              value={formatTimestamp(service.local.lastChecked)}
              copyable
            />
          )}
          {healthStatus?.responseTime !== undefined && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Response Time</span>
              <span className="text-sm font-medium text-foreground">
                {Math.round(healthStatus.responseTime / 1000000)}ms
              </span>
            </div>
          )}
        </div>
      </SectionCard>

      {/* Health Details */}
      {(healthStatus || service.local?.healthDetails) && (
        <SectionCard title="Health Details" data-testid="health-details-section">
          <div className="space-y-2">
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Check Type</span>
              <span className="text-sm font-medium text-foreground">
                {formatCheckType(healthStatus?.checkType || service.local?.healthDetails?.checkType)}
              </span>
            </div>
            {(healthStatus?.endpoint || service.local?.healthDetails?.endpoint) && (
              <InfoField
                label="Endpoint"
                value={healthStatus?.endpoint || service.local?.healthDetails?.endpoint || ''}
                copyable
              />
            )}
            {(healthStatus?.statusCode || service.local?.healthDetails?.statusCode) && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-muted-foreground">Status Code</span>
                <span className="text-sm font-medium text-foreground">
                  {healthStatus?.statusCode || service.local?.healthDetails?.statusCode}
                </span>
              </div>
            )}
            {service.local?.healthDetails?.consecutiveFailures !== undefined && (
              <div className="flex justify-between items-center">
                <span className="text-sm text-muted-foreground">Failures</span>
                <span className="text-sm font-medium text-foreground">
                  {service.local.healthDetails.consecutiveFailures}
                </span>
              </div>
            )}
          </div>
        </SectionCard>
      )}
    </div>
  )
}

// =============================================================================
// Azure Tab
// =============================================================================

interface AzureTabProps {
  service: Service
}

function AzureTab({ service }: AzureTabProps) {
  const isDeployed = hasAzureDeployment(service)
  const azurePortalUrl = buildAzurePortalUrl(service)

  if (!isDeployed) {
    return (
      <div data-testid="azure-tab-content">
        <SectionCard title="Not Deployed" data-testid="not-deployed-section">
          <div className="text-center py-4">
            <p className="text-sm text-muted-foreground mb-2">
              This service hasn't been deployed to Azure yet.
            </p>
            <p className="text-sm text-muted-foreground">
              Run <code className="bg-muted px-1.5 py-0.5 rounded">azd deploy</code> to deploy your services.
            </p>
          </div>
        </SectionCard>
      </div>
    )
  }

  return (
    <div data-testid="azure-tab-content">
      {/* Resource */}
      <SectionCard title="Resource" data-testid="resource-section">
        <div className="space-y-2">
          {service.azure?.resourceName && (
            <InfoField
              label="Resource Name"
              value={service.azure.resourceName}
              copyable
            />
          )}
          {service.azure?.resourceType && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Resource Type</span>
              <span className="text-sm font-medium text-foreground">
                {formatResourceType(service.azure.resourceType)}
              </span>
            </div>
          )}
          {service.azure?.imageName && (
            <InfoField
              label="Image"
              value={service.azure.imageName}
              copyable
            />
          )}
          {service.azure?.url && (
            <InfoField
              label="Endpoint"
              value={service.azure.url}
              copyable
            />
          )}
        </div>
      </SectionCard>

      {/* Azure Metadata */}
      <SectionCard title="Azure Metadata" data-testid="azure-metadata-section">
        <div className="space-y-2">
          {service.azure?.subscriptionId && (
            <InfoField
              label="Subscription"
              value={service.azure.subscriptionId}
              copyable
            />
          )}
          {service.azure?.resourceGroup && (
            <InfoField
              label="Resource Group"
              value={service.azure.resourceGroup}
              copyable
            />
          )}
          {service.azure?.location && (
            <div className="flex justify-between items-center">
              <span className="text-sm text-muted-foreground">Location</span>
              <span className="text-sm font-medium text-foreground">{service.azure.location}</span>
            </div>
          )}
          {service.azure?.containerAppEnvId && (
            <InfoField
              label="Environment ID"
              value={service.azure.containerAppEnvId}
              copyable
            />
          )}
          {service.azure?.logAnalyticsId && (
            <InfoField
              label="Log Analytics"
              value={service.azure.logAnalyticsId}
              copyable
            />
          )}
        </div>
      </SectionCard>

      {/* Actions */}
      {azurePortalUrl && (
        <SectionCard title="Actions" data-testid="azure-actions-section">
          <div className="flex flex-wrap gap-2">
            <a
              href={azurePortalUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-3 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
            >
              Open in Azure Portal
              <ExternalLink className="h-3 w-3" />
            </a>
          </div>
        </SectionCard>
      )}
    </div>
  )
}

// =============================================================================
// Environment Tab
// =============================================================================

interface EnvironmentTabProps {
  service: Service
  copiedField: string | null
  onCopy: (text: string, fieldName: string) => void
}

function EnvironmentTab({ service, copiedField, onCopy }: EnvironmentTabProps) {
  const [showSensitive, setShowSensitive] = React.useState<Record<string, boolean>>({})
  const envVars = service.environmentVariables || {}
  const envEntries = Object.entries(envVars)

  const toggleSensitive = (key: string) => {
    setShowSensitive(prev => ({ ...prev, [key]: !prev[key] }))
  }

  if (envEntries.length === 0) {
    return (
      <div data-testid="environment-tab-content">
        <SectionCard title="Environment Variables" data-testid="env-vars-section">
          <p className="text-sm text-muted-foreground text-center py-4">
            No environment variables configured
          </p>
        </SectionCard>
      </div>
    )
  }

  return (
    <div data-testid="environment-tab-content">
      <SectionCard
        title={`Environment Variables (${envEntries.length})`}
        data-testid="env-vars-section"
      >
        <div className="space-y-2">
          {envEntries.map(([key, value]) => {
            const isSensitive = isSensitiveKey(key)
            const isShown = showSensitive[key]
            const displayValue = isSensitive && !isShown ? maskValue(value) : value

            return (
              <div key={key} className="flex items-center justify-between gap-2">
                <span className="text-sm text-muted-foreground font-mono flex-shrink-0">
                  {key}
                </span>
                <div className="flex items-center gap-1">
                  <span className="text-sm font-medium text-foreground font-mono truncate max-w-[200px]">
                    {displayValue}
                  </span>
                  {isSensitive && (
                    <button
                      type="button"
                      onClick={() => toggleSensitive(key)}
                      className="p-1 hover:bg-secondary rounded-md transition-colors"
                      aria-label={isShown ? `Hide ${key}` : `Show ${key}`}
                    >
                      {isShown ? (
                        <EyeOff className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Eye className="h-4 w-4 text-muted-foreground" />
                      )}
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={() => onCopy(value, `env-${key}`)}
                    className={cn(
                      'p-1 hover:bg-secondary rounded-md transition-colors',
                      copiedField === `env-${key}` && 'text-green-500'
                    )}
                    aria-label={`Copy ${key}`}
                  >
                    {copiedField === `env-${key}` ? (
                      <span className="text-xs">✓</span>
                    ) : (
                      <span className="text-xs">📋</span>
                    )}
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      </SectionCard>
    </div>
  )
}

// =============================================================================
// ServiceDetailPanel
// =============================================================================

export function ServiceDetailPanel({
  service,
  isOpen,
  onClose,
  healthStatus,
  className,
  'data-testid': testId = 'service-detail-panel',
}: ServiceDetailPanelProps) {
  const [activeTab, setActiveTab] = React.useState<DetailTab>('overview')
  const panelRef = React.useRef<HTMLDivElement>(null)
  const { copiedField, copyToClipboard } = useClipboard()

  // Close on escape key
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
      const closeButton = panelRef.current.querySelector<HTMLButtonElement>('[data-testid="close-button"]')
      closeButton?.focus()
    }
  }, [isOpen])

  // Don't render if not open or no service
  if (!isOpen || !service) {
    return null
  }

  const handleCopy = (text: string, fieldName: string) => {
    void copyToClipboard(text, fieldName)
  }

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  const statusDisplay = getStatusDisplay(service.local?.status || service.status)
  const healthDisplay = getHealthDisplay(healthStatus?.status || service.local?.health || service.health)

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm animate-fade-in"
        onClick={handleBackdropClick}
        data-testid="panel-backdrop"
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="panel-title"
        className={cn(
          'fixed right-0 top-0 z-50 h-screen w-[500px] max-w-full',
          'bg-[var(--background)] border-l border-border shadow-xl',
          'animate-slide-in-right',
          className
        )}
        data-testid={testId}
      >
        {/* Header */}
        <div className="flex items-start justify-between p-6 border-b border-border">
          <div className="flex items-center gap-3">
            <span
              className={cn(
                'text-xl',
                getStatusColor(service.local?.status || service.status)
              )}
              aria-hidden="true"
            >
              {statusDisplay.indicator}
            </span>
            <div>
              <h2
                id="panel-title"
                className="text-xl font-semibold text-foreground"
              >
                {service.name}
              </h2>
              <p className="text-sm text-muted-foreground">
                {service.framework || service.language || 'Service'}
                {' • '}
                {statusDisplay.text}
                {' • '}
                {healthDisplay.text}
              </p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            data-testid="close-button"
            aria-label="Close panel"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Tabs */}
        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as DetailTab)}
          className="flex flex-col h-[calc(100vh-80px)]"
        >
          <TabsList className="flex border-b border-border px-6 h-12 bg-transparent rounded-none justify-start gap-4">
            <TabsTrigger
              value="overview"
              className="px-0 pb-3 pt-3 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="local"
              className="px-0 pb-3 pt-3 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent"
            >
              Local
            </TabsTrigger>
            <TabsTrigger
              value="azure"
              className="px-0 pb-3 pt-3 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent"
            >
              Azure
            </TabsTrigger>
            <TabsTrigger
              value="environment"
              className="px-0 pb-3 pt-3 rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent"
            >
              Environment
            </TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto p-6">
            <TabsContent value="overview" className="mt-0">
              <OverviewTab service={service} healthStatus={healthStatus} />
            </TabsContent>
            <TabsContent value="local" className="mt-0">
              <LocalTab
                service={service}
                healthStatus={healthStatus}
              />
            </TabsContent>
            <TabsContent value="azure" className="mt-0">
              <AzureTab
                service={service}
              />
            </TabsContent>
            <TabsContent value="environment" className="mt-0">
              <EnvironmentTab
                service={service}
                copiedField={copiedField}
                onCopy={handleCopy}
              />
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </>
  )
}
