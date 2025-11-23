import { getStatusDisplay } from '@/lib/service-utils'

interface StatusCellProps {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: 'healthy' | 'unhealthy' | 'unknown'
}

export function StatusCell({ status, health }: StatusCellProps) {
  const statusDisplay = getStatusDisplay(status, health)

  return (
    <div className="flex items-center gap-2">
      <div className={`w-2 h-2 rounded-full ${statusDisplay.color}`}></div>
      <span className={`font-medium ${statusDisplay.textColor}`}>
        {statusDisplay.text}
      </span>
    </div>
  )
}
