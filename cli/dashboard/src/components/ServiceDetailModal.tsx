import { X, Server, Cloud, Settings2, ExternalLink, Copy } from 'lucide-react'
import { useState } from 'react'
import type { Service } from '@/types'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface ServiceDetailModalProps {
  service: Service
  isOpen: boolean
  onClose: () => void
}

export function ServiceDetailModal({ service, isOpen, onClose }: ServiceDetailModalProps) {
  const [activeTab, setActiveTab] = useState('overview')
  const [copiedField, setCopiedField] = useState<string | null>(null)

  if (!isOpen) return null

  const copyToClipboard = (text: string, field: string) => {
    navigator.clipboard.writeText(text)
    setCopiedField(field)
    setTimeout(() => setCopiedField(null), 2000)
  }

  const status = service.local?.status || service.status || 'not-running'
  const health = service.local?.health || service.health || 'unknown'
  const isHealthy = (status === 'ready' || status === 'running') && health === 'healthy'

  return (
    <div 
      className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div 
        className="glass max-w-4xl w-full rounded-2xl border border-white/10 shadow-2xl overflow-hidden max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10 bg-[#0d0d0d]">
          <div className="flex items-center gap-3">
            <div className={`p-2.5 rounded-xl ${
              isHealthy ? 'bg-success/20' : 'bg-muted/20'
            }`}>
              <Server className={`w-5 h-5 ${isHealthy ? 'text-success' : 'text-muted-foreground'}`} />
            </div>
            <div>
              <h2 className="text-xl font-semibold text-foreground">{service.name}</h2>
              <p className="text-sm text-muted-foreground mt-0.5">
                {service.framework} • {service.language}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Badge variant={isHealthy ? 'default' : 'secondary'}>
              {status}
            </Badge>
            <button
              onClick={onClose}
              className="p-2 hover:bg-white/5 rounded-lg transition-colors"
            >
              <X className="w-5 h-5 text-muted-foreground" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col">
          <TabsList className="px-6 pt-4">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="local">Local</TabsTrigger>
            {service.azure && <TabsTrigger value="azure">Azure</TabsTrigger>}
            <TabsTrigger value="environment">Environment</TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto p-6">
            <TabsContent value="overview" className="mt-0">
              <div className="space-y-4">
                {/* Local Info */}
                {service.local && (
                  <div className="glass p-4 rounded-xl border border-white/10">
                    <h3 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-2">
                      <Server className="w-4 h-4 text-primary" />
                      Local Development
                    </h3>
                    <div className="grid grid-cols-2 gap-3">
                      <InfoField label="Status" value={status} />
                      <InfoField label="Health" value={health} />
                      {service.local.url && (
                        <InfoField 
                          label="URL" 
                          value={service.local.url}
                          copyable
                          onCopy={() => copyToClipboard(service.local!.url!, 'local-url')}
                          copied={copiedField === 'local-url'}
                        />
                      )}
                      {service.local.port && <InfoField label="Port" value={service.local.port.toString()} />}
                      {service.local.pid && <InfoField label="PID" value={service.local.pid.toString()} />}
                      {service.local.startTime && <InfoField label="Started" value={new Date(service.local.startTime).toLocaleString()} />}
                    </div>
                  </div>
                )}

                {/* Azure Info */}
                {service.azure && (
                  <div className="glass p-4 rounded-xl border border-blue-500/20 bg-blue-500/5">
                    <h3 className="text-sm font-semibold text-blue-300 mb-3 flex items-center gap-2">
                      <Cloud className="w-4 h-4 text-blue-400" />
                      Azure Deployment
                    </h3>
                    <div className="grid grid-cols-2 gap-3">
                      {service.azure.resourceName && <InfoField label="Resource" value={service.azure.resourceName} variant="azure" />}
                      {service.azure.resourceType && <InfoField label="Type" value={service.azure.resourceType} variant="azure" />}
                      {service.azure.resourceGroup && <InfoField label="Resource Group" value={service.azure.resourceGroup} variant="azure" />}
                      {service.azure.location && <InfoField label="Location" value={service.azure.location} variant="azure" />}
                      {service.azure.url && (
                        <InfoField 
                          label="Azure URL" 
                          value={service.azure.url}
                          variant="azure"
                          copyable
                          onCopy={() => copyToClipboard(service.azure!.url!, 'azure-url')}
                          copied={copiedField === 'azure-url'}
                        />
                      )}
                    </div>
                  </div>
                )}
              </div>
            </TabsContent>

            <TabsContent value="local" className="mt-0">
              <div className="space-y-4">
                <div className="glass p-4 rounded-xl border border-white/10">
                  <h3 className="text-sm font-semibold text-foreground mb-3">Local Service Details</h3>
                  {service.local ? (
                    <div className="space-y-3">
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
              <div className="space-y-4">
                {service.azure ? (
                  <>
                    <div className="glass p-4 rounded-xl border border-blue-500/20 bg-blue-500/5">
                      <h3 className="text-sm font-semibold text-blue-300 mb-3">Azure Resource Information</h3>
                      <div className="space-y-3">
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
                        className="flex items-center justify-center gap-2 p-3 rounded-xl glass border border-blue-500/20 hover:border-blue-500/50 transition-all bg-blue-500/5 text-blue-300 hover:text-blue-200"
                      >
                        <ExternalLink className="w-4 h-4" />
                        Open Azure Service
                      </a>
                    )}
                  </>
                ) : (
                  <div className="glass p-8 rounded-xl border border-white/10 text-center">
                    <Cloud className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
                    <p className="text-sm text-muted-foreground">Service not deployed to Azure</p>
                  </div>
                )}
              </div>
            </TabsContent>

            <TabsContent value="environment" className="mt-0">
              <div className="glass p-4 rounded-xl border border-white/10">
                <h3 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-2">
                  <Settings2 className="w-4 h-4 text-primary" />
                  Environment Variables
                </h3>
                {service.environmentVariables && Object.keys(service.environmentVariables).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(service.environmentVariables).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between p-2 rounded-lg hover:bg-white/5">
                        <div className="flex-1 min-w-0">
                          <p className="text-xs font-mono text-foreground truncate">{key}</p>
                          <p className="text-xs font-mono text-muted-foreground truncate">{value}</p>
                        </div>
                        <button
                          onClick={() => copyToClipboard(value, `env-${key}`)}
                          className="ml-2 p-1.5 hover:bg-white/10 rounded transition-all"
                          title="Copy value"
                        >
                          {copiedField === `env-${key}` ? (
                            <span className="text-xs text-success">✓</span>
                          ) : (
                            <Copy className="w-3.5 h-3.5 text-muted-foreground" />
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
    </div>
  )
}

interface InfoFieldProps {
  label: string
  value: string
  variant?: 'default' | 'azure'
  copyable?: boolean
  onCopy?: () => void
  copied?: boolean
}

function InfoField({ label, value, variant = 'default', copyable, onCopy, copied }: InfoFieldProps) {
  const textColor = variant === 'azure' ? 'text-blue-200' : 'text-foreground'
  
  return (
    <div className="space-y-1">
      <p className={`text-xs ${variant === 'azure' ? 'text-blue-400/70' : 'text-muted-foreground'}`}>{label}</p>
      <div className="flex items-center gap-2">
        <p className={`text-sm font-medium ${textColor} truncate`}>{value}</p>
        {copyable && (
          <button
            onClick={onCopy}
            className="p-1 hover:bg-white/10 rounded transition-all shrink-0"
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
    <div className="flex items-center justify-between py-2 border-b border-white/5 last:border-b-0">
      <span className={`text-sm ${labelColor}`}>{label}</span>
      <div className="flex items-center gap-2">
        <span className={`text-sm font-medium ${textColor} font-mono`}>{value}</span>
        {copyable && (
          <button
            onClick={copyToClipboard}
            className="p-1 hover:bg-white/10 rounded transition-all"
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
