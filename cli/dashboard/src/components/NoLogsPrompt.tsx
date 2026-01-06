/**
 * NoLogsPrompt - Prompt shown in log pane when service has 0 Azure logs
 * Part of azure-logs-diagnostics feature
 * Provides explanatory text and link to open diagnostics modal
 */
import type { ReactNode } from 'react'
import { AlertTriangle, Wrench } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface NoLogsPromptProps {
  /** Name of the service with no logs */
  serviceName: string
  /** Callback to open diagnostics modal */
  onOpenDiagnostics?: () => void
}

/**
 * NoLogsPrompt Component
 * 
 * Displays when an Azure service has 0 logs in the selected time range.
 * Provides clear explanation of possible reasons and link to diagnostics.
 * 
 * @param serviceName - Name of the service
 * @param onOpenDiagnostics - Callback to open diagnostics modal
 */
export function NoLogsPrompt({
  serviceName,
  onOpenDiagnostics,
}: Readonly<NoLogsPromptProps>): ReactNode {
  return (
    <div 
      className="flex flex-col items-center justify-center py-12 text-center"
      role="status"
      aria-label={`No logs available for ${serviceName}`}
    >
      {/* Warning Icon */}
      <AlertTriangle 
        className="w-12 h-12 text-amber-500 dark:text-amber-400 mb-4" 
        aria-hidden="true"
      />
      
      {/* Service Name */}
      <h3 className="text-base font-semibold text-foreground mb-2">
        No logs for {serviceName}
      </h3>
      
      {/* Explanation */}
      <p className="text-sm text-muted-foreground max-w-md mb-6 leading-relaxed">
        This could be because diagnostic settings are not configured, 
        there's a delay in log ingestion (2-5 minutes), 
        or the service hasn't generated any activity yet.
      </p>
      
      {/* Diagnostic Link */}
      {onOpenDiagnostics && (
        <button
          type="button"
          onClick={onOpenDiagnostics}
          className={cn(
            'inline-flex items-center gap-2 px-4 py-2.5 rounded-md',
            'text-sm font-medium',
            'bg-cyan-600 hover:bg-cyan-700 dark:bg-cyan-700 dark:hover:bg-cyan-600',
            'text-white',
            'transition-colors duration-200',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
            'shadow-sm hover:shadow-md'
          )}
          aria-label="View diagnostic details to troubleshoot"
        >
          <Wrench className="w-4 h-4" aria-hidden="true" />
          View Diagnostic Details
        </button>
      )}
    </div>
  )
}
