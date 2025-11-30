import { useEffect, useState, useRef, useCallback } from 'react'
import { AlertCircle, AlertTriangle, Info, X } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface NotificationToastProps {
  id: string
  title: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date
  onDismiss: (id: string) => void
  onClick?: (id: string) => void
  autoDismiss?: boolean
  dismissTimeout?: number
  className?: string
}

export function NotificationToast({
  id,
  title,
  message,
  severity,
  timestamp,
  onDismiss,
  onClick,
  autoDismiss = true,
  dismissTimeout,
  className
}: NotificationToastProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [isPaused, setIsPaused] = useState(false)
  const [timeRemaining, setTimeRemaining] = useState(100)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const startTimeRef = useRef<number | null>(null)
  const pausedTimeRef = useRef<number>(0)
  
  // Initialize startTime on first render effect
  useEffect(() => {
    if (startTimeRef.current === null) {
      startTimeRef.current = Date.now()
    }
  }, [])

  // Default timeouts based on severity
  const timeout = dismissTimeout ?? (severity === 'critical' ? 10000 : 5000)

  // Relative time formatting
  const getRelativeTime = (date: Date): string => {
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    
    if (diff < 60000) return 'Just now'
    if (diff < 3600000) return `${Math.floor(diff / 60000)} minutes ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} hours ago`
    return `${Math.floor(diff / 86400000)} days ago`
  }

  // Icon mapping
  const icons = {
    critical: AlertCircle,
    warning: AlertTriangle,
    info: Info
  }
  const Icon = icons[severity]

  // Start entry animation
  useEffect(() => {
    requestAnimationFrame(() => setIsVisible(true))
  }, [])

  const handleDismiss = useCallback(() => {
    setIsVisible(false)
    setTimeout(() => onDismiss(id), 300) // Match exit animation duration
  }, [id, onDismiss])

  // Auto-dismiss timer
  useEffect(() => {
    if (!autoDismiss || isPaused || startTimeRef.current === null) return

    const startTime = startTimeRef.current
    const elapsed = Date.now() - startTime - pausedTimeRef.current
    const remaining = timeout - elapsed

    if (remaining <= 0) {
      // Use setTimeout to avoid synchronous setState in effect
      const immediateTimer = setTimeout(() => handleDismiss(), 0)
      return () => clearTimeout(immediateTimer)
    }

    // Update progress bar
    const progressInterval = setInterval(() => {
      if (startTimeRef.current === null) return
      const elapsed = Date.now() - startTimeRef.current - pausedTimeRef.current
      const progress = Math.max(0, 100 - (elapsed / timeout) * 100)
      setTimeRemaining(progress)
      
      if (progress <= 0) {
        clearInterval(progressInterval)
      }
    }, 16) // ~60fps

    // Auto-dismiss timer
    timerRef.current = setTimeout(() => {
      handleDismiss()
    }, remaining)

    return () => {
      clearTimeout(timerRef.current!)
      clearInterval(progressInterval)
    }
  }, [autoDismiss, isPaused, timeout, handleDismiss])

  const handleClick = () => {
    if (onClick) {
      onClick(id)
      handleDismiss()
    }
  }

  const handleMouseEnter = () => {
    setIsPaused(true)
    if (startTimeRef.current !== null) {
      pausedTimeRef.current = Date.now() - startTimeRef.current
    }
  }

  const handleMouseLeave = () => {
    setIsPaused(false)
    startTimeRef.current = Date.now() - pausedTimeRef.current
  }

  const severityStyles = {
    critical: {
      light: 'bg-[hsl(0,84%,95%)] border-[hsl(0,84%,60%)] text-[hsl(0,10%,15%)]',
      dark: 'dark:bg-[hsl(0,50%,15%)] dark:border-[hsl(0,60%,50%)] dark:text-[hsl(0,5%,95%)]',
      icon: 'text-[hsl(0,84%,40%)] dark:text-[hsl(0,70%,60%)]',
      progress: 'bg-[hsl(0,84%,60%)]'
    },
    warning: {
      light: 'bg-[hsl(45,100%,95%)] border-[hsl(45,100%,50%)] text-[hsl(45,10%,15%)]',
      dark: 'dark:bg-[hsl(45,50%,15%)] dark:border-[hsl(45,80%,50%)] dark:text-[hsl(45,5%,95%)]',
      icon: 'text-[hsl(45,100%,35%)] dark:text-[hsl(45,90%,60%)]',
      progress: 'bg-[hsl(45,100%,50%)]'
    },
    info: {
      light: 'bg-[hsl(210,100%,95%)] border-[hsl(210,100%,50%)] text-[hsl(210,10%,15%)]',
      dark: 'dark:bg-[hsl(210,50%,15%)] dark:border-[hsl(210,80%,50%)] dark:text-[hsl(210,5%,95%)]',
      icon: 'text-[hsl(210,100%,35%)] dark:text-[hsl(210,90%,60%)]',
      progress: 'bg-[hsl(210,100%,50%)]'
    }
  }

  const styles = severityStyles[severity]

  return (
    <div
      role="alert"
      aria-live="assertive"
      aria-atomic="true"
      aria-labelledby={`toast-title-${id}`}
      aria-describedby={`toast-message-${id}`}
      className={cn(
        'w-full sm:w-[360px] min-h-[80px] max-w-[420px] rounded-lg border shadow-lg transition-all duration-300',
        styles.light,
        styles.dark,
        isVisible ? 'opacity-100 translate-x-0' : 'opacity-0 translate-x-full',
        isPaused ? 'shadow-xl' : '',
        onClick && 'cursor-pointer hover:shadow-xl hover:scale-[1.01] active:scale-[0.98]',
        className
      )}
      onClick={handleClick}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <div className="p-4 flex gap-3 items-start">
        {/* Icon */}
        <div className="flex-shrink-0">
          <Icon className={cn('w-[18px] h-[18px]', styles.icon)} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div 
            id={`toast-title-${id}`}
            className="font-semibold text-sm leading-tight mb-1"
          >
            {title}
          </div>
          <div
            id={`toast-message-${id}`}
            className="text-[13px] leading-relaxed mb-1"
          >
            {message}
          </div>
          <div className="text-[11px] opacity-70">
            {getRelativeTime(timestamp)}
          </div>
        </div>

        {/* Close button */}
        <button
          onClick={(e) => {
            e.stopPropagation()
            handleDismiss()
          }}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              e.stopPropagation()
              handleDismiss()
            }
          }}
          aria-label="Dismiss notification"
          className="flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity p-1 rounded hover:bg-black/5 dark:hover:bg-white/5"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Progress bar */}
      {autoDismiss && (
        <div className="h-[2px] bg-black/10 dark:bg-white/10">
          <div
            className={cn('h-full opacity-60 transition-all duration-75 linear', styles.progress)}
            style={{ width: `${timeRemaining}%` }}
          />
        </div>
      )}
    </div>
  )
}
