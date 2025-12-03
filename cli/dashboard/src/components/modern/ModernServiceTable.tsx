/**
 * ModernServiceTable - Table view with modern styling
 * Follows design spec: cli/dashboard/design/modern/components/service-table.md
 */
import * as React from 'react'
import { 
  Server, 
  ExternalLink, 
  FileText,
  ChevronUp,
  ChevronDown,
  Package,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ModernStatusBadge, type EffectiveStatus } from './ModernStatusIndicator'
import { ServiceActions } from '@/components/ServiceActions'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import type { Service, HealthReportEvent, HealthCheckResult } from '@/types'
import { formatRelativeTime } from '@/lib/service-utils'

// =============================================================================
// Types
// =============================================================================

export interface ModernServiceTableProps {
  /** Services to display */
  services: Service[]
  /** Health report for status updates */
  healthReport?: HealthReportEvent | null
  /** Callback when a row is clicked */
  onServiceClick?: (service: Service) => void
  /** Callback to view logs for a service */
  onViewLogs?: (serviceName: string) => void
  /** Additional class names */
  className?: string
}

type SortField = 'name' | 'status' | 'startTime'
type SortDirection = 'asc' | 'desc'

// =============================================================================
// Helper Functions
// =============================================================================

function getEffectiveStatus(
  service: Service, 
  healthStatus?: HealthCheckResult,
  operationState?: string
): EffectiveStatus {
  if (operationState && operationState !== 'idle') {
    return operationState as EffectiveStatus
  }
  
  const processStatus = service.local?.status ?? service.status ?? 'not-running'
  
  if (processStatus === 'stopped') return 'stopped'
  if (processStatus === 'stopping') return 'stopping'
  if (processStatus === 'starting') return 'starting'
  if (processStatus === 'error') return 'error'
  if (processStatus === 'not-running') return 'not-running'
  
  const health = healthStatus?.status ?? service.local?.health ?? service.health ?? 'unknown'
  if (health === 'healthy') return 'healthy'
  if (health === 'degraded') return 'degraded'
  if (health === 'unhealthy') return 'unhealthy'
  if (health === 'starting') return 'starting'
  
  return 'unknown'
}

// =============================================================================
// ModernServiceTableRow Component
// =============================================================================

interface ModernServiceTableRowProps {
  service: Service
  healthStatus?: HealthCheckResult
  onClick?: () => void
  onViewLogs?: () => void
}

function ModernServiceTableRow({ 
  service, 
  healthStatus, 
  onClick,
  onViewLogs,
}: ModernServiceTableRowProps) {
  const { getOperationState } = useServiceOperations()
  const operationState = getOperationState(service.name)
  const effectiveStatus = getEffectiveStatus(service, healthStatus, operationState)
  
  const localUrl = service.local?.url && !service.local.url.match(/:0\/?$/) ? service.local.url : null
  const azureUrl = service.azure?.url
  const startTime = service.local?.startTime ?? service.startTime

  return (
    <tr
      className={cn(
        'group border-b border-slate-100 dark:border-slate-800',
        'hover:bg-slate-50 dark:hover:bg-slate-800/50',
        'transition-colors duration-100',
        onClick && 'cursor-pointer'
      )}
      onClick={onClick}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      } : undefined}
    >
      {/* Name */}
      <td className="py-3 px-4">
        <div className="flex items-center gap-3">
          <div className={cn(
            'w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0',
            effectiveStatus === 'healthy' 
              ? 'bg-emerald-100 dark:bg-emerald-500/20'
              : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
              ? 'bg-rose-100 dark:bg-rose-500/20'
              : 'bg-slate-100 dark:bg-slate-700'
          )}>
            <Server className={cn(
              'w-4 h-4',
              effectiveStatus === 'healthy' 
                ? 'text-emerald-600 dark:text-emerald-400'
                : effectiveStatus === 'error' || effectiveStatus === 'unhealthy'
                ? 'text-rose-600 dark:text-rose-400'
                : 'text-slate-500 dark:text-slate-400'
            )} />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-slate-900 dark:text-slate-100 truncate">
              {service.name}
            </p>
            {(service.language || service.framework) && (
              <p className="text-xs text-slate-500 dark:text-slate-400 truncate">
                {service.language}
                {service.framework && ` • ${service.framework}`}
              </p>
            )}
          </div>
        </div>
      </td>

      {/* Status */}
      <td className="py-3 px-4">
        <ModernStatusBadge status={effectiveStatus} />
      </td>

      {/* Start Time */}
      <td className="py-3 px-4">
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {startTime ? formatRelativeTime(startTime) : '-'}
        </span>
      </td>

      {/* Source */}
      <td className="py-3 px-4 max-w-[200px]">
        <p className="text-xs text-slate-500 dark:text-slate-400 truncate direction-rtl text-left">
          {service.project || '-'}
        </p>
      </td>

      {/* Local URL */}
      <td className="py-3 px-4">
        {localUrl ? (
          <a
            href={localUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-mono bg-slate-100 dark:bg-slate-700 rounded text-slate-600 dark:text-slate-300 hover:text-cyan-600 dark:hover:text-cyan-400 hover:bg-cyan-50 dark:hover:bg-cyan-500/10 transition-colors max-w-[180px]"
          >
            <span className="truncate">{localUrl}</span>
            <ExternalLink className="w-3 h-3 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" />
          </a>
        ) : (
          <span className="text-xs text-slate-400 dark:text-slate-500 italic">No URL</span>
        )}
      </td>

      {/* Azure URL */}
      <td className="py-3 px-4">
        {azureUrl ? (
          <a
            href={azureUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center gap-1.5 px-2 py-1 text-xs font-mono bg-cyan-50 dark:bg-cyan-500/10 border border-cyan-200 dark:border-cyan-500/30 rounded text-cyan-700 dark:text-cyan-300 hover:border-cyan-400 dark:hover:border-cyan-500 transition-colors max-w-[180px]"
          >
            <span className="truncate">{azureUrl}</span>
            <ExternalLink className="w-3 h-3 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" />
          </a>
        ) : (
          <span className="text-xs text-slate-400 dark:text-slate-500 italic">Not deployed</span>
        )}
      </td>

      {/* Actions */}
      <td className="py-3 px-4 text-right">
        <div 
          className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity"
          onClick={(e) => e.stopPropagation()}
        >
          <ServiceActions service={service} variant="compact" />
          {onViewLogs && (
            <button
              type="button"
              onClick={onViewLogs}
              className="p-1.5 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
              title="View logs"
            >
              <FileText className="w-4 h-4" />
            </button>
          )}
        </div>
      </td>
    </tr>
  )
}

// =============================================================================
// Empty State Component
// =============================================================================

function TableEmptyState() {
  return (
    <div className="text-center py-16 px-8">
      <Package className="w-12 h-12 mx-auto mb-4 text-slate-300 dark:text-slate-600" />
      <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100 mb-2">
        No Services Running
      </h3>
      <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
        Get started by launching your development services
      </p>
      <code className="inline-block px-4 py-2 bg-slate-100 dark:bg-slate-800 rounded-lg text-sm font-mono text-cyan-600 dark:text-cyan-400">
        azd app run
      </code>
    </div>
  )
}

// =============================================================================
// ModernServiceTable Component
// =============================================================================

export function ModernServiceTable({
  services,
  healthReport,
  onServiceClick,
  onViewLogs,
  className,
}: ModernServiceTableProps) {
  const [sortField, setSortField] = React.useState<SortField>('name')
  const [sortDirection, setSortDirection] = React.useState<SortDirection>('asc')

  const getServiceHealth = (serviceName: string): HealthCheckResult | undefined => {
    return healthReport?.services.find(s => s.serviceName === serviceName)
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(prev => prev === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('asc')
    }
  }

  const sortedServices = React.useMemo(() => {
    return [...services].sort((a, b) => {
      let comparison = 0
      
      switch (sortField) {
        case 'name':
          comparison = a.name.toLowerCase().localeCompare(b.name.toLowerCase())
          break
        case 'status': {
          const statusA = a.local?.status ?? a.status ?? 'unknown'
          const statusB = b.local?.status ?? b.status ?? 'unknown'
          comparison = statusA.localeCompare(statusB)
          break
        }
        case 'startTime': {
          const timeA = a.local?.startTime ?? a.startTime ?? ''
          const timeB = b.local?.startTime ?? b.startTime ?? ''
          comparison = timeA.localeCompare(timeB)
          break
        }
      }
      
      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [services, sortField, sortDirection])

  const renderSortIcon = (field: SortField) => {
    if (sortField !== field) return null
    return sortDirection === 'asc' 
      ? <ChevronUp className="w-3.5 h-3.5 text-cyan-600 dark:text-cyan-400" />
      : <ChevronDown className="w-3.5 h-3.5 text-cyan-600 dark:text-cyan-400" />
  }

  return (
    <div className={cn(
      'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-2xl overflow-hidden shadow-sm',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Services</h2>
          <span className="px-2 py-0.5 bg-slate-200 dark:bg-slate-700 rounded-full text-xs font-medium text-slate-600 dark:text-slate-300">
            {services.length}
          </span>
        </div>
      </div>

      {/* Table */}
      {services.length === 0 ? (
        <TableEmptyState />
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
                <th 
                  className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider cursor-pointer select-none hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
                  onClick={() => handleSort('name')}
                >
                  <span className="flex items-center gap-1">
                    Name
                    {renderSortIcon('name')}
                  </span>
                </th>
                <th 
                  className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider cursor-pointer select-none hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
                  onClick={() => handleSort('status')}
                >
                  <span className="flex items-center gap-1">
                    State
                    {renderSortIcon('status')}
                  </span>
                </th>
                <th 
                  className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider cursor-pointer select-none hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
                  onClick={() => handleSort('startTime')}
                >
                  <span className="flex items-center gap-1">
                    Start time
                    {renderSortIcon('startTime')}
                  </span>
                </th>
                <th className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Source
                </th>
                <th className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Local URL
                </th>
                <th className="py-2.5 px-4 text-left font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Azure URL
                </th>
                <th className="py-2.5 px-4 text-right font-semibold text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider w-[100px]">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedServices.map((service) => (
                <ModernServiceTableRow
                  key={service.name}
                  service={service}
                  healthStatus={getServiceHealth(service.name)}
                  onClick={onServiceClick ? () => onServiceClick(service) : undefined}
                  onViewLogs={onViewLogs ? () => onViewLogs(service.name) : undefined}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
