import { X, Server, Cloud, Settings2, ExternalLink, Copy } from 'lucide-react'
import { useState } from 'react'
import type { Service } from '@/types'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEscapeKey } from '@/hooks/useEscapeKey'
import { useClipboard } from '@/hooks/useClipboard'
import { getServiceStatus, getServiceHealth, isServiceHealthy } from '@/lib/serviceUtils'

interface ServiceDetailPanelProps {
  service: Service | null
  isOpen: boolean
  onClose: () => void
}

export function ServiceDetailPanel({ service, isOpen, onClose }: ServiceDetailPanelProps) {
  const [activeTab, setActiveTab] = useState('overview')
  const { copyToClipboard, copiedField } = useClipboard()
  
  // Close on Escape key
  useEscapeKey(isOpen, onClose)

  if (!service) return null

  const status = getServiceStatus(service)
  const health = getServiceHealth(service)
  const isHealthy = isServiceHealthy(service)

  return (
    <>
      {/* Backdrop */}
      <div 
        className={`fixed inset-0 bg-black/30 backdrop-blur-sm z-40 transition-opacity duration-300 ${
          isOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        onClick={onClose}
      />

      {/* Side Panel */}
      <div 
        className={`fixed right-0 top-0 h-full w-[500px] bg-[#1a1a1a] border-l border-white/10 shadow-2xl z-50 flex flex-col transform transition-transform duration-300 ease-in-out ${
          isOpen ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10 bg-[#0d0d0d] shrink-0">
          <div className="flex items-center gap-3 min-w-0 flex-1">
            <div className={`p-2 rounded-lg shrink-0 ${
              isHealthy ? 'bg-success/20' : 'bg-muted/20'
            }`}>
              <Server className={`w-4 h-4 ${isHealthy ? 'text-success' : 'text-muted-foreground'}`} />
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold text-foreground truncate">{service.name}</h2>
              <p className="text-xs text-muted-foreground truncate">
                {service.framework} • {service.language}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Badge variant={isHealthy ? 'default' : 'secondary'} className="text-xs">
              {status}
            </Badge>
            <button
              onClick={onClose}
              className="p-1.5 hover:bg-white/5 rounded-md transition-colors"
              aria-label="Close panel"
            >
              <X className="w-4 h-4 text-muted-foreground" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col min-h-0">
          <TabsList className="px-4 pt-3 shrink-0">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="local">Local</TabsTrigger>
            {service.azure && <TabsTrigger value="azure">Azure</TabsTrigger>}
            <TabsTrigger value="environment">Environment</TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto p-4">
            <TabsContent value="overview" className="mt-0 space-y-3">
              {/* Local Info */}
              {service.local && (
                <div className="glass p-3 rounded-lg border border-white/10">
                  <h3 className="text-xs font-semibold text-foreground mb-2 flex items-center gap-2">
                    <Server className="w-3.5 h-3.5 text-primary" />
                    Local Development
                  </h3>
                  <div className="grid grid-cols-2 gap-2">
                    <InfoField label="Status" value={status} />
                    <InfoField label="Health" value={health} />
                    {service.local.url && (
                      <InfoField 
                        label="URL" 
                        value={service.local.url}
                        copyable
                        onCopy={() => copyToClipboard(service.local!.url!, 'local-url')}
                        copied={copiedField === 'local-url'}
                        fullWidth
                      />
                    )}
                    {service.local.port && <InfoField label="Port" value={service.local.port.toString()} />}
                    {service.local.pid && <InfoField label="PID" value={service.local.pid.toString()} />}
                  </div>
                </div>
              )}

              {/* Azure Info */}
              {service.azure && (
                <div className="glass p-3 rounded-lg border border-blue-500/20 bg-blue-500/5">
                  <h3 className="text-xs font-semibold text-blue-300 mb-2 flex items-center gap-2">
                    <Cloud className="w-3.5 h-3.5 text-blue-400" />
                    Azure Deployment
                  </h3>
                  <div className="grid grid-cols-2 gap-2">
                    {service.azure.resourceName && <InfoField label="Resource" value={service.azure.resourceName} variant="azure" />}
                    {service.azure.resourceType && <InfoField label="Type" value={service.azure.resourceType} variant="azure" />}
                    {service.azure.resourceGroup && <InfoField label="Group" value={service.azure.resourceGroup} variant="azure" />}
                    {service.azure.location && <InfoField label="Location" value={service.azure.location} variant="azure" />}
                    {service.azure.url && (
                      <InfoField 
                        label="Azure URL" 
                        value={service.azure.url}
                        variant="azure"
                        copyable
                        onCopy={() => copyToClipboard(service.azure!.url!, 'azure-url')}
                        copied={copiedField === 'azure-url'}
                        fullWidth
                      />
                    )}
                  </div>
                </div>
              )}
            </TabsContent>

            <TabsContent value="local" className="mt-0">
              <div className="space-y-3">
                <div className="glass p-3 rounded-lg border border-white/10">
                  <h3 className="text-xs font-semibold text-foreground mb-2">Local Service Details</h3>
                  {service.local ? (
                    <div className="space-y-2">
                      <DetailRow label="Status" value={service.local.status} />
                      <DetailRow label="Health" value={service.local.health} />
                      {service.local.url && <DetailRow label="URL" value={service.local.url} copyable />}
                      {service.local.port && <DetailRow label="Port" value={service.local.port.toString()} />}
                      {service.local.pid && <DetailRow label="Process ID" value={service.local.pid.toString()} />}
                      {service.local.startTime && <DetailRow label="Start Time" value={new Date(service.local.startTime).toLocaleString()} />}
                      {service.local.lastChecked && <DetailRow label="Last Checked" value={new Date(service.local.lastChecked).toLocaleString()} />}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">Service not running locally</p>
                  )}
                </div>
              </div>
            </TabsContent>

            <TabsContent value="azure" className="mt-0">
              <div className="space-y-3">
                {service.azure ? (
                  <>
                    <div className="glass p-3 rounded-lg border border-blue-500/20 bg-blue-500/5">
                      <h3 className="text-xs font-semibold text-blue-300 mb-2">Azure Resource Information</h3>
                      <div className="space-y-2">
                        {service.azure.resourceName && <DetailRow label="Resource Name" value={service.azure.resourceName} variant="azure" />}
                        {service.azure.resourceType && <DetailRow label="Resource Type" value={service.azure.resourceType} variant="azure" />}
                        {service.azure.resourceGroup && <DetailRow label="Resource Group" value={service.azure.resourceGroup} variant="azure" copyable />}
                        {service.azure.location && <DetailRow label="Location" value={service.azure.location} variant="azure" />}
                        {service.azure.subscriptionId && <DetailRow label="Subscription ID" value={service.azure.subscriptionId} variant="azure" copyable />}
                        {service.azure.url && <DetailRow label="Endpoint URL" value={service.azure.url} variant="azure" copyable />}
                        {service.azure.imageName && <DetailRow label="Container Image" value={service.azure.imageName} variant="azure" />}
                        {service.azure.containerAppEnvId && <DetailRow label="Environment ID" value={service.azure.containerAppEnvId} variant="azure" copyable />}
                      </div>
                    </div>

                    {service.azure.url && (
                      <a
                        href={service.azure.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="flex items-center justify-center gap-2 p-2.5 rounded-lg glass border border-blue-500/20 hover:border-blue-500/50 transition-all bg-blue-500/5 text-blue-300 hover:text-blue-200 text-sm"
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                        Open Azure Service
                      </a>
                    )}
                  </>
                ) : (
                  <div className="glass p-8 rounded-lg border border-white/10 text-center">
                    <Cloud className="w-10 h-10 text-muted-foreground mx-auto mb-2" />
                    <p className="text-sm text-muted-foreground">Service not deployed to Azure</p>
                  </div>
                )}
              </div>
            </TabsContent>

            <TabsContent value="environment" className="mt-0">
              <div className="glass p-3 rounded-lg border border-white/10">
                <h3 className="text-xs font-semibold text-foreground mb-2 flex items-center gap-2">
                  <Settings2 className="w-3.5 h-3.5 text-primary" />
                  Environment Variables
                </h3>
                {service.environmentVariables && Object.keys(service.environmentVariables).length > 0 ? (
                  <div className="space-y-1.5">
                    {Object.entries(service.environmentVariables).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between p-2 rounded-md hover:bg-white/5 group">
                        <div className="flex-1 min-w-0 mr-2">
                          <p className="text-xs font-mono text-foreground truncate">{key}</p>
                          <p className="text-xs font-mono text-muted-foreground truncate">{value}</p>
                        </div>
                        <button
                          onClick={() => copyToClipboard(value, `env-${key}`)}
                          className="opacity-0 group-hover:opacity-100 p-1 hover:bg-white/10 rounded transition-all"
                          title="Copy value"
                        >
                          {copiedField === `env-${key}` ? (
                            <span className="text-xs text-success">✓</span>
                          ) : (
                            <Copy className="w-3 h-3 text-muted-foreground" />
                          )}
                        </button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No environment variables configured</p>
                )}
              </div>
            </TabsContent>
          </div>
        </Tabs>
      </div>
    </>
  )
}

interface InfoFieldProps {
  label: string
  value: string
  variant?: 'default' | 'azure'
  copyable?: boolean
  onCopy?: () => void
  copied?: boolean
  fullWidth?: boolean
}

function InfoField({ label, value, variant = 'default', copyable, onCopy, copied, fullWidth }: InfoFieldProps) {
  const textColor = variant === 'azure' ? 'text-blue-200' : 'text-foreground'
  
  return (
    <div className={`space-y-0.5 ${fullWidth ? 'col-span-2' : ''}`}>
      <p className={`text-[10px] ${variant === 'azure' ? 'text-blue-400/70' : 'text-muted-foreground'}`}>{label}</p>
      <div className="flex items-center gap-1.5">
        <p className={`text-xs font-medium ${textColor} truncate`} title={value}>{value}</p>
        {copyable && (
          <button
            onClick={onCopy}
            className="p-0.5 hover:bg-white/10 rounded transition-all shrink-0"
            title="Copy"
          >
            {copied ? (
              <span className="text-xs text-success">✓</span>
            ) : (
              <Copy className="w-3 h-3 text-muted-foreground" />
            )}
          </button>
        )}
      </div>
    </div>
  )
}

interface DetailRowProps {
  label: string
  value: string
  variant?: 'default' | 'azure'
  copyable?: boolean
}

function DetailRow({ label, value, variant = 'default', copyable }: DetailRowProps) {
  const [copied, setCopied] = useState(false)
  
  const copyToClipboard = () => {
    navigator.clipboard.writeText(value)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const textColor = variant === 'azure' ? 'text-blue-200' : 'text-foreground'
  const labelColor = variant === 'azure' ? 'text-blue-400/70' : 'text-muted-foreground'

  return (
    <div className="flex items-center justify-between py-1.5 border-b border-white/5 last:border-b-0">
      <span className={`text-xs ${labelColor}`}>{label}</span>
      <div className="flex items-center gap-1.5">
        <span className={`text-xs font-medium ${textColor} font-mono truncate max-w-[200px]`} title={value}>{value}</span>
        {copyable && (
          <button
            onClick={copyToClipboard}
            className="p-0.5 hover:bg-white/10 rounded transition-all"
            title="Copy"
          >
            {copied ? (
              <span className="text-xs text-success">✓</span>
            ) : (
              <Copy className="w-3 h-3 text-muted-foreground" />
            )}
          </button>
        )}
      </div>
    </div>
  )
}
