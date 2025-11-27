import { Play, Square, RotateCw } from 'lucide-react'
import { useState } from 'react'
import type { Service } from '@/types'
import { getEffectiveStatus } from '@/lib/service-utils'

interface ServiceActionsProps {
  service: Service
  variant?: 'default' | 'compact'
  onActionComplete?: () => void
}

export function ServiceActions({ service, variant = 'default', onActionComplete }: ServiceActionsProps) {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { status } = getEffectiveStatus(service)

  const canStart = status === 'stopped' || status === 'not-running' || status === 'error'
  const canStop = status === 'running' || status === 'ready' || status === 'starting'
  const canRestart = status === 'running' || status === 'ready'

  const handleAction = async (action: 'start' | 'stop' | 'restart') => {
    setIsLoading(true)
    setError(null)

    try {
      const response = await fetch(`/api/services/${action}?service=${encodeURIComponent(service.name)}`, {
        method: 'POST',
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' })) as { error?: string }
        throw new Error(errorData.error ?? `Failed to ${action} service`)
      }

      onActionComplete?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${action} service`)
      console.error(`Error ${action}ing service:`, err)
    } finally {
      setIsLoading(false)
    }
  }

  const handleClick = (action: 'start' | 'stop' | 'restart') => {
    void handleAction(action)
  }

  if (variant === 'compact') {
    return (
      <div className="flex items-center gap-1">
        {canStart && (
          <button
            onClick={() => handleClick('start')}
            disabled={isLoading}
            className="p-1.5 rounded hover:bg-success/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Start service"
          >
            <Play className="w-3.5 h-3.5 text-success group-hover:text-success/80" />
          </button>
        )}
        {canRestart && (
          <button
            onClick={() => handleClick('restart')}
            disabled={isLoading}
            className="p-1.5 rounded hover:bg-warning/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Restart service"
          >
            <RotateCw className={`w-3.5 h-3.5 text-warning group-hover:text-warning/80 ${isLoading ? 'animate-spin' : ''}`} />
          </button>
        )}
        {canStop && (
          <button
            onClick={() => handleClick('stop')}
            disabled={isLoading}
            className="p-1.5 rounded hover:bg-destructive/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Stop service"
          >
            <Square className="w-3.5 h-3.5 text-destructive group-hover:text-destructive/80" />
          </button>
        )}
        {error && (
          <div className="text-xs text-destructive ml-2">{error}</div>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        {canStart && (
          <button
            onClick={() => handleClick('start')}
            disabled={isLoading}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-success/10 hover:bg-success/20 text-success font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Play className="w-4 h-4" />
            Start
          </button>
        )}
        {canRestart && (
          <button
            onClick={() => handleClick('restart')}
            disabled={isLoading}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-warning/10 hover:bg-warning/20 text-warning font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <RotateCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            Restart
          </button>
        )}
        {canStop && (
          <button
            onClick={() => handleClick('stop')}
            disabled={isLoading}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-destructive/10 hover:bg-destructive/20 text-destructive font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Square className="w-4 h-4" />
            Stop
          </button>
        )}
      </div>
      {error && (
        <div className="text-xs text-destructive">{error}</div>
      )}
    </div>
  )
}
