import { CheckCircle, XCircle, Clock, AlertCircle, StopCircle } from 'lucide-react'

interface StatusCellProps {
  status: 'starting' | 'ready' | 'running' | 'stopping' | 'stopped' | 'error' | 'not-running'
  health: 'healthy' | 'unhealthy' | 'unknown'
}

export function StatusCell({ status, health }: StatusCellProps) {
  const getStatusDisplay = (status: string, health: string) => {
    // Running only if status is running/ready AND health is healthy
    if ((status === 'ready' || status === 'running') && health === 'healthy') {
      return {
        text: 'Running',
        color: 'bg-green-500',
        textColor: 'text-green-400',
        icon: <CheckCircle className="w-4 h-4" />
      }
    }
    
    // Unhealthy state
    if ((status === 'ready' || status === 'running') && health === 'unhealthy') {
      return {
        text: 'Unhealthy',
        color: 'bg-red-500',
        textColor: 'text-red-400',
        icon: <XCircle className="w-4 h-4" />
      }
    }
    
    // Starting
    if (status === 'starting') {
      return {
        text: 'Starting',
        color: 'bg-yellow-500',
        textColor: 'text-yellow-400',
        icon: <Clock className="w-4 h-4 animate-spin" />
      }
    }
    
    // Error
    if (status === 'error') {
      return {
        text: 'Error',
        color: 'bg-red-500',
        textColor: 'text-red-400',
        icon: <XCircle className="w-4 h-4" />
      }
    }
    
    // Stopping
    if (status === 'stopping') {
      return {
        text: 'Stopping',
        color: 'bg-gray-500',
        textColor: 'text-gray-400',
        icon: <StopCircle className="w-4 h-4 animate-pulse" />
      }
    }
    
    // Stopped or not-running
    if (status === 'stopped' || status === 'not-running') {
      return {
        text: 'Stopped',
        color: 'bg-gray-500',
        textColor: 'text-gray-400',
        icon: <StopCircle className="w-4 h-4" />
      }
    }
    
    // Unknown
    return {
      text: 'Unknown',
      color: 'bg-gray-500',
      textColor: 'text-gray-400',
      icon: <AlertCircle className="w-4 h-4" />
    }
  }

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
