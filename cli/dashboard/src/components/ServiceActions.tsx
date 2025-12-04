import { Play, Square, RotateCw } from 'lucide-react'
import type { Service } from '@/types'
import { useServiceOperations } from '@/hooks/useServiceOperations'
import { cn } from '@/lib/utils'

interface ServiceActionsProps {
  service: Service
  variant?: 'default' | 'compact'
  showError?: boolean  // Whether to show error inline (default: false for compact)
  onActionComplete?: () => void
}

export function ServiceActions({
  service,
  variant = 'default',
  showError = variant === 'default',
  onActionComplete
}: ServiceActionsProps) {
  const {
    startService,
    stopService,
    restartService,
    isOperationInProgress,
    getOperationState,
    canPerformAction,
    error
  } = useServiceOperations()

  const canStart = canPerformAction(service, 'start')
  const canStop = canPerformAction(service, 'stop')
  const canRestart = canPerformAction(service, 'restart')
  const operationInProgress = isOperationInProgress(service.name)
  const currentOperation = getOperationState(service.name)

  const handleAction = async (action: 'start' | 'stop' | 'restart') => {
    let success = false
    switch (action) {
      case 'start':
        success = await startService(service.name)
        break
      case 'stop':
        success = await stopService(service.name)
        break
      case 'restart':
        success = await restartService(service.name)
        break
    }
    if (success) {
      onActionComplete?.()
    }
  }

  const handleClick = (action: 'start' | 'stop' | 'restart') => {
    void handleAction(action)
  }

  // Show loading state when operation is actually running (not idle)
  const showLoadingState = operationInProgress && currentOperation !== 'idle'

  // Stop propagation to prevent parent click handlers (e.g., opening details panel)
  const stopPropagation = (e: React.MouseEvent) => {
    e.stopPropagation()
  }

  if (variant === 'compact') {
    return (
      <div className="flex items-center gap-1" onClick={stopPropagation}>
        {canStart && (
          <button
            onClick={() => handleClick('start')}
            disabled={operationInProgress}
            className="p-1.5 rounded hover:bg-emerald-500/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Start service"
          >
            <Play className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 group-hover:text-emerald-500" />
          </button>
        )}
        {canRestart && (
          <button
            onClick={() => handleClick('restart')}
            disabled={operationInProgress}
            className="p-1.5 rounded hover:bg-amber-500/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Restart service"
          >
            <RotateCw className={cn(
              "w-3.5 h-3.5 text-amber-600 dark:text-amber-400 group-hover:text-amber-500",
              currentOperation === 'restarting' && "animate-spin"
            )} />
          </button>
        )}
        {canStop && (
          <button
            onClick={() => handleClick('stop')}
            disabled={operationInProgress}
            className="p-1.5 rounded hover:bg-rose-500/10 transition-colors group disabled:opacity-50 disabled:cursor-not-allowed"
            title="Stop service"
          >
            <Square className="w-3.5 h-3.5 text-rose-600 dark:text-rose-400 group-hover:text-rose-500" />
          </button>
        )}
        {showLoadingState && (
          <span className="text-xs text-slate-500 dark:text-slate-400 animate-pulse capitalize ml-1">
            {currentOperation}...
          </span>
        )}
        {showError && error && (
          <div className="text-xs text-rose-600 dark:text-rose-400 ml-2">{error}</div>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2" onClick={stopPropagation}>
      <div className="flex items-center gap-2">
        {canStart && (
          <button
            onClick={() => handleClick('start')}
            disabled={operationInProgress}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Play className="w-4 h-4" />
            Start
          </button>
        )}
        {canRestart && (
          <button
            onClick={() => handleClick('restart')}
            disabled={operationInProgress}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-500/10 hover:bg-amber-500/20 text-amber-600 dark:text-amber-400 font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <RotateCw className={cn("w-4 h-4", currentOperation === 'restarting' && "animate-spin")} />
            Restart
          </button>
        )}
        {canStop && (
          <button
            onClick={() => handleClick('stop')}
            disabled={operationInProgress}
            className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-600 dark:text-rose-400 font-medium text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Square className="w-4 h-4" />
            Stop
          </button>
        )}
      </div>
      {showLoadingState && (
        <div className="text-xs text-slate-500 dark:text-slate-400 animate-pulse capitalize">
          {currentOperation}...
        </div>
      )}
      {showError && error && (
        <div className="text-xs text-rose-600 dark:text-rose-400">{error}</div>
      )}
    </div>
  )
}
