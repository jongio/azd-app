/**
 * HealthTooltipContent - Content layout for health diagnostic tooltip
 * 
 * Displays detailed health information including:
 * - Status header with icon
 * - Check details (type, endpoint, timing)
 * - Error section (if applicable)
 * - Service info (uptime, port, pid)
 * - Suggested actions list
 * - Copy button
 */

import { 
  CheckCircle, 
  XCircle, 
  AlertTriangle, 
  HelpCircle,
  Globe,
  Plug,
  Cpu,
  Zap,
  Clock,
  Server,
  Copy,
  Terminal,
  ExternalLink,
  AlertCircle,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatResponseTime, formatUptime } from '@/lib/service-formatters'
import { getCheckTypeDisplay } from '@/lib/service-health'
import type { HealthDiagnostic, HealthStatus } from '@/types'

export interface HealthTooltipContentProps {
  /** Diagnostic information */
  diagnostic: HealthDiagnostic
  /** Copy handler */
  onCopy: () => void
}

interface StatusConfig {
  icon: LucideIcon
  color: string
  bgColor: string
  borderColor: string
  label: string
}

export function HealthTooltipContent({ diagnostic, onCopy }: Readonly<HealthTooltipContentProps>) {
  const { healthStatus, service, suggestedActions } = diagnostic
  const status: HealthStatus = healthStatus.status

  // Determine status-specific styling
  const statusConfig: StatusConfig = {
    healthy: {
      icon: CheckCircle,
      color: 'text-emerald-600 dark:text-emerald-400',
      bgColor: 'bg-emerald-50 dark:bg-emerald-500/10',
      borderColor: 'border-emerald-200 dark:border-emerald-500/30',
      label: 'Healthy',
    },
    degraded: {
      icon: AlertTriangle,
      color: 'text-amber-600 dark:text-amber-400',
      bgColor: 'bg-amber-50 dark:bg-amber-500/10',
      borderColor: 'border-amber-200 dark:border-amber-500/30',
      label: 'Degraded',
    },
    unhealthy: {
      icon: XCircle,
      color: 'text-rose-600 dark:text-rose-400',
      bgColor: 'bg-rose-50 dark:bg-rose-500/10',
      borderColor: 'border-rose-200 dark:border-rose-500/30',
      label: 'Unhealthy',
    },
    unknown: {
      icon: HelpCircle,
      color: 'text-slate-500 dark:text-slate-400',
      bgColor: 'bg-slate-50 dark:bg-slate-500/10',
      borderColor: 'border-slate-200 dark:border-slate-500/30',
      label: 'Unknown',
    },
  }[status]

  const StatusIcon = statusConfig.icon

  // Get check type icon
  const getCheckIcon = () => {
    const checkType = healthStatus.checkType
    if (checkType === 'http') return Globe
    if (checkType === 'tcp') return Plug
    if (checkType === 'process') return Cpu
    return Server
  }

  const renderCheckIcon = () => {
    const Icon = getCheckIcon()
    return <Icon className="w-3.5 h-3.5 text-slate-500" />
  }

  return (
    <div className="p-4 space-y-3 max-h-125">
      {/* Status Header */}
      <div className={cn(
        'flex items-center gap-2 p-3 rounded-lg border',
        statusConfig.bgColor,
        statusConfig.borderColor
      )}>
        <StatusIcon className={cn('w-5 h-5', statusConfig.color)} />
        <div className="flex-1">
          <p className={cn('text-sm font-semibold', statusConfig.color)}>
            Service Health: {statusConfig.label}
          </p>
          <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
            {service.name}
          </p>
        </div>
      </div>

      {/* Check Details */}
      <div className="space-y-2">
        <div className="flex items-center gap-1.5">
          {renderCheckIcon()}
          <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
            Check Details
          </span>
        </div>
        
        <div className="pl-5 space-y-1.5 text-xs">
          <div className="flex justify-between">
            <span className="text-slate-500 dark:text-slate-400">Type:</span>
            <span className="font-medium text-slate-700 dark:text-slate-300">
              {getCheckTypeDisplay(healthStatus.checkType).toUpperCase()}
            </span>
          </div>
          
          {healthStatus.endpoint && (
            <div className="flex flex-col gap-0.5">
              <span className="text-slate-500 dark:text-slate-400">Endpoint:</span>
              <code className="text-[10px] font-mono bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded break-all">
                {healthStatus.endpoint}
              </code>
            </div>
          )}
          
          {healthStatus.statusCode && (
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Status:</span>
              <span className={cn(
                'font-medium',
                healthStatus.statusCode >= 200 && healthStatus.statusCode < 300
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-rose-600 dark:text-rose-400'
              )}>
                {healthStatus.statusCode}
              </span>
            </div>
          )}
          
          <div className="flex justify-between items-center">
            <div className="flex items-center gap-1">
              <Zap className="w-3 h-3 text-amber-500" />
              <span className="text-slate-500 dark:text-slate-400">Response Time:</span>
            </div>
            <span className="font-mono font-medium text-slate-700 dark:text-slate-300">
              {formatResponseTime(healthStatus.responseTime)}
            </span>
          </div>

          {healthStatus.consecutiveFailures && healthStatus.consecutiveFailures > 0 && (
            <div className="flex justify-between items-center">
              <div className="flex items-center gap-1">
                <AlertCircle className="w-3 h-3 text-rose-500" />
                <span className="text-slate-500 dark:text-slate-400">Consecutive Failures:</span>
              </div>
              <span className="font-semibold text-rose-600 dark:text-rose-400">
                {healthStatus.consecutiveFailures}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Error Section */}
      {healthStatus.error && (
        <div className="space-y-2">
          <div className="flex items-center gap-1.5">
            <XCircle className="w-3.5 h-3.5 text-rose-500" />
            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
              Error Details
            </span>
          </div>
          
          <div className="pl-5 space-y-2">
            <div className={cn(
              'p-2 rounded-lg border text-xs',
              'bg-rose-50 dark:bg-rose-500/10',
              'border-rose-200 dark:border-rose-500/30',
              'text-rose-700 dark:text-rose-300'
            )}>
              <p className="font-medium wrap-break-word">{healthStatus.error}</p>
              {healthStatus.errorDetails && (
                <p className="mt-1 text-rose-600 dark:text-rose-400 wrap-break-word">
                  {healthStatus.errorDetails}
                </p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Service Info */}
      <div className="space-y-2">
        <div className="flex items-center gap-1.5">
          <Server className="w-3.5 h-3.5 text-slate-500" />
          <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
            Service Info
          </span>
        </div>
        
        <div className="pl-5 space-y-1.5 text-xs">
          <div className="flex justify-between items-center">
            <div className="flex items-center gap-1">
              <Clock className="w-3 h-3 text-emerald-500" />
              <span className="text-slate-500 dark:text-slate-400">Uptime:</span>
            </div>
            <span className="font-mono font-medium text-slate-700 dark:text-slate-300">
              {formatUptime(healthStatus.uptime)}
            </span>
          </div>
          
          {healthStatus.port && healthStatus.port > 0 && (
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Port:</span>
              <span className="font-mono font-medium text-slate-700 dark:text-slate-300">
                {healthStatus.port}
              </span>
            </div>
          )}
          
          {healthStatus.pid && (
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">PID:</span>
              <span className="font-mono font-medium text-slate-700 dark:text-slate-300">
                {healthStatus.pid}
              </span>
            </div>
          )}

          {service.local?.serviceType && (
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Type:</span>
              <span className="font-medium text-slate-700 dark:text-slate-300 capitalize">
                {service.local.serviceType}
              </span>
            </div>
          )}

          {service.local?.serviceMode && (
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Mode:</span>
              <span className="font-medium text-slate-700 dark:text-slate-300 capitalize">
                {service.local.serviceMode}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Suggested Actions */}
      {suggestedActions.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-1.5">
            <Terminal className="w-3.5 h-3.5 text-cyan-500" />
            <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
              Suggested Actions
            </span>
          </div>
          
          <div className="pl-5 space-y-1.5">
            {suggestedActions.slice(0, 5).map((action, index) => (
              <div 
                key={`action-${index}-${action.label.slice(0, 20)}`}
                className="text-xs text-slate-600 dark:text-slate-400 group"
              >
                <div className="flex items-start gap-1.5">
                  <span className="text-slate-400 dark:text-slate-500 mt-0.5">•</span>
                  <div className="flex-1 space-y-1">
                    <p>{action.label}</p>
                    {action.command && (
                      <code className="block text-[10px] font-mono bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded break-all">
                        {action.command}
                      </code>
                    )}
                    {action.docsUrl && (
                      <a
                        href={action.docsUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-cyan-600 dark:text-cyan-400 hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <ExternalLink className="w-3 h-3" />
                        <span>Documentation</span>
                      </a>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Copy Button */}
      <div className="pt-2 border-t border-slate-200 dark:border-slate-700">
        <button
          type="button"
          onClick={onCopy}
          className={cn(
            'w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg',
            'text-xs font-medium transition-colors',
            'bg-slate-100 hover:bg-slate-200 dark:bg-slate-700 dark:hover:bg-slate-600',
            'text-slate-700 dark:text-slate-300',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            'dark:focus:ring-offset-slate-800'
          )}
        >
          <Copy className="w-3.5 h-3.5" />
          Copy Diagnostics
        </button>
      </div>

      {/* Footer */}
      <div className="text-center text-[10px] text-slate-400 dark:text-slate-500">
        Last checked: {new Date(healthStatus.timestamp).toLocaleTimeString()}
      </div>
    </div>
  )
}
