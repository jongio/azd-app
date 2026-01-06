import type { ReactNode } from 'react'
import { useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Copy, ExternalLink, ChevronDown, ChevronRight, PanelRight, CheckCircle, CircleOff, Loader2, RotateCw, Eye, Hammer, CheckSquare, CircleX, CircleDot, Settings2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { normalizeHealthStatus } from '@/lib/service-utils'
import type { Service, HealthCheckResult } from '@/types'
import { ServiceActions } from './ServiceActions'
import { HealthTooltip } from './HealthTooltip'
import type { LogMode } from './ModeToggle'

function getProcessBadge(processStatus?: string): { className: string; icon: ReactNode; title: string } {
  if (!processStatus) {
    return {
      className: 'bg-muted text-muted-foreground border border-border',
      icon: <CircleDot className="w-3 h-3 shrink-0" />,
      title: 'Process state: unknown',
    }
  }

  switch (processStatus) {
    case 'running':
    case 'ready':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <CheckCircle className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'watching':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <Eye className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'stopped':
    case 'not-started':
    case 'not-running':
      return {
        className: 'bg-gray-500/10 text-gray-600 dark:text-gray-400 border border-gray-500/30',
        icon: <CircleOff className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'building':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <Hammer className="w-3 h-3 shrink-0 animate-pulse" />,
        title: `Process state: ${processStatus}`,
      }
    case 'built':
    case 'completed':
      return {
        className: 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30',
        icon: <CheckSquare className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    case 'starting':
    case 'stopping':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <Loader2 className="w-3 h-3 shrink-0 animate-spin" />,
        title: `Process state: ${processStatus}`,
      }
    case 'restarting':
      return {
        className: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/30',
        icon: <RotateCw className="w-3 h-3 shrink-0 animate-spin" />,
        title: `Process state: ${processStatus}`,
      }
    case 'failed':
    case 'error':
      return {
        className: 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30',
        icon: <CircleX className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
    default:
      return {
        className: 'bg-muted text-muted-foreground border border-border',
        icon: <CircleDot className="w-3 h-3 shrink-0" />,
        title: `Process state: ${processStatus}`,
      }
  }
}

function getHealthBadgeClass(normalizedHealth: ReturnType<typeof normalizeHealthStatus>): string {
  switch (normalizedHealth) {
    case 'healthy':
      return 'bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/30'
    case 'degraded':
      return 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/30'
    case 'unhealthy':
      return 'bg-red-500/10 text-red-600 dark:text-red-400 border border-red-500/30'
    case 'unknown':
    default:
      return 'bg-muted text-muted-foreground border border-border'
  }
}

export interface LogsPaneHeaderProps {
  serviceName: string
  port?: number
  isCollapsed: boolean
  toggleCollapsed: () => void
  headerBgClass: string
  processStatus?: string
  normalizedHealth?: ReturnType<typeof normalizeHealthStatus>
  healthIcon: ReactNode
  service?: Service
  healthCheckResult?: HealthCheckResult
  effectiveUrl?: string
  logMode: LogMode
  azureUrl?: string
  onShowDetails?: () => void
  onOpenConfigPanel?: () => void
  handleCopyPane: () => void
}

export function LogsPaneHeader({
  serviceName,
  port,
  isCollapsed,
  toggleCollapsed,
  headerBgClass,
  processStatus,
  normalizedHealth,
  healthIcon,
  healthCheckResult,
  service,
  effectiveUrl,
  logMode,
  azureUrl,
  onShowDetails,
  onOpenConfigPanel,
  handleCopyPane,
}: Readonly<LogsPaneHeaderProps>) {
  const processBadge = useMemo(() => getProcessBadge(processStatus), [processStatus])
  const healthBadgeClass = normalizedHealth ? getHealthBadgeClass(normalizedHealth) : ''

  return (
    <div className={cn("flex items-center justify-between px-4 py-2 border-b transition-colors duration-200", headerBgClass)}>
      <button
        type="button"
        className="flex items-center gap-2 flex-1 min-w-0 cursor-pointer select-none"
        onClick={toggleCollapsed}
        aria-label={isCollapsed ? `Expand logs pane for ${serviceName}` : `Collapse logs pane for ${serviceName}`}
        aria-expanded={!isCollapsed}
      >
        <span className="p-0.5 hover:bg-muted rounded transition-colors" aria-hidden="true">
          {isCollapsed ? (
            <ChevronRight className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          )}
        </span>
        <h3 className="font-semibold truncate" data-testid="logs-pane-header-title">
          {serviceName}
          {port && <span className="text-muted-foreground font-mono">:{port}</span>}
        </h3>

        <span 
          className={cn(
            "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
            processBadge.className
          )}
          title={processBadge.title}
        >
          {processBadge.icon}
        </span>

        {normalizedHealth && (
          service && healthCheckResult ? (
            <HealthTooltip healthStatus={healthCheckResult} service={service}>
              <span 
                className={cn(
                  "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
                  healthBadgeClass
                )}
                title={`Service health: ${normalizedHealth} (from health checks)`}
              >
                {healthIcon}
              </span>
            </HealthTooltip>
          ) : (
            <span 
              className={cn(
                "inline-flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
                healthBadgeClass
              )}
              title={`Service health: ${normalizedHealth} (from health checks)`}
            >
              {healthIcon}
            </span>
          )
        )}
      </button>

      <div className="flex items-center gap-2">
        {service && (
          <div className="mr-2 border-r pr-2 border-border">
            <ServiceActions service={service} variant="compact" />
          </div>
        )}
        {logMode === 'azure' && onOpenConfigPanel && (
          <Button
            variant="outline"
            size="sm"
            onClick={onOpenConfigPanel}
            title="Configure log sources"
            aria-label="Configure log sources for this service"
          >
            <Settings2 className="w-4 h-4" />
          </Button>
        )}
        {effectiveUrl && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => globalThis.open(effectiveUrl, '_blank', 'noopener,noreferrer')}
            title={effectiveUrl}
            aria-label={logMode === 'azure' && azureUrl ? 'Open Azure endpoint in new tab' : 'Open local service in new tab'}
          >
            <ExternalLink className="w-4 h-4" />
          </Button>
        )}
        {onShowDetails && (
          <Button
            variant="outline"
            size="sm"
            onClick={onShowDetails}
            title="Show service details"
            aria-label="Show service details panel"
          >
            <PanelRight className="w-4 h-4" />
          </Button>
        )}
        <Button
          variant="outline"
          size="sm"
          onClick={handleCopyPane}
          title="Copy all logs"
          aria-label="Copy logs to clipboard"
        >
          <Copy className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}
