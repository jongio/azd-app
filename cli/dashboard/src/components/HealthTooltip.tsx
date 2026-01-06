/**
 * HealthTooltip - Tooltip wrapper for displaying health diagnostics
 * 
 * Wraps health status icons with a tooltip that shows detailed diagnostic
 * information including error details, suggested actions, and copy functionality.
 */

import * as React from 'react'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'
import { cn } from '@/lib/utils'
import { HealthTooltipContent } from './HealthTooltipContent'
import { buildHealthDiagnostic } from '@/lib/health-diagnostics'
import { useToast } from '@/hooks/useToast'
import type { HealthCheckResult, Service } from '@/types'

export interface HealthTooltipProps {
  /** Health check result data */
  readonly healthStatus: HealthCheckResult
  /** Service information */
  readonly service: Service
  /** Children element (the trigger) */
  readonly children: React.ReactNode
  /** Additional class names */
  readonly className?: string
}

export function HealthTooltip({ 
  healthStatus, 
  service, 
  children,
  className 
}: Readonly<HealthTooltipProps>) {
  const [open, setOpen] = React.useState(false)
  const { showToast } = useToast()
  
  // Build diagnostic information (memoized)
  const diagnostic = React.useMemo(() => 
    buildHealthDiagnostic(healthStatus, service), 
    [healthStatus, service]
  )

  // Copy diagnostic report to clipboard
  const handleCopy = React.useCallback(async () => {
    try {
      await navigator.clipboard.writeText(diagnostic.formattedReport)
      showToast('Diagnostics copied to clipboard', 'success')
    } catch (error) {
      showToast('Failed to copy diagnostics', 'error')
      console.error('Failed to copy diagnostics:', error)
    }
  }, [diagnostic.formattedReport, showToast])

  return (
    <TooltipPrimitive.Provider delayDuration={400}>
      <TooltipPrimitive.Root open={open} onOpenChange={setOpen}>
        <TooltipPrimitive.Trigger asChild>
          {children}
        </TooltipPrimitive.Trigger>
        <TooltipPrimitive.Portal>
          <TooltipPrimitive.Content
            side="top"
            sideOffset={8}
            className={cn(
              'z-50 max-w-md overflow-hidden rounded-xl border-2 bg-white shadow-xl',
              'dark:bg-slate-800 dark:border-slate-700',
              'animate-in fade-in-0 zoom-in-95',
              'data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95',
              'data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2',
              'data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
              className
            )}
          >
            <HealthTooltipContent 
              diagnostic={diagnostic} 
              onCopy={() => { void handleCopy() }}
            />
            <TooltipPrimitive.Arrow 
              className="fill-white dark:fill-slate-800" 
              width={12} 
              height={6} 
            />
          </TooltipPrimitive.Content>
        </TooltipPrimitive.Portal>
      </TooltipPrimitive.Root>
    </TooltipPrimitive.Provider>
  )
}
