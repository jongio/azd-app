import type { ReactNode } from 'react'
import { Heart, HeartPulse, HeartCrack, HelpCircle } from 'lucide-react'
import { normalizeHealthStatus, type VisualStatus } from '@/lib/service-utils'

export function getHealthIcon(normalizedHealth: ReturnType<typeof normalizeHealthStatus> | undefined): ReactNode {
  if (!normalizedHealth) {
    return null
  }

  switch (normalizedHealth) {
    case 'healthy':
      return <Heart className="w-3 h-3 shrink-0 animate-heartbeat" />
    case 'degraded':
      return <HeartPulse className="w-3 h-3 shrink-0 animate-caution-pulse" />
    case 'unhealthy':
      return <HeartCrack className="w-3 h-3 shrink-0 animate-status-flash" />
    default:
      return <HelpCircle className="w-3 h-3 shrink-0" />
  }
}

export function getPaneStyleClasses(visualStatus: VisualStatus): { borderClass: string; headerBgClass: string } {
  const borderClass = {
    error: 'border-red-500',
    warning: 'border-amber-500',
    healthy: 'border-green-500',
    stopped: 'border-gray-400',
    info: 'border-border'
  }[visualStatus]

  const headerBgClass = {
    error: 'log-header-error',
    warning: 'log-header-warning',
    healthy: 'log-header-healthy',
    stopped: 'bg-muted',
    info: 'bg-card'
  }[visualStatus]

  return { borderClass, headerBgClass }
}
