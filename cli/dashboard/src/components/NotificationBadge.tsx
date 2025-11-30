import { useEffect, useState, useRef, useSyncExternalStore, useCallback } from 'react'
import { cn } from '@/lib/utils'

export interface NotificationBadgeProps {
  count: number
  variant?: 'default' | 'critical' | 'warning'
  size?: 'sm' | 'md' | 'lg'
  max?: number
  showZero?: boolean
  pulse?: boolean
  className?: string
}

// Use a module-level Map to track pulse state per component instance
const pulseStates = new Map<string, { isPulsing: boolean; timeoutId: ReturnType<typeof setTimeout> | null }>()
const pulseListeners = new Map<string, Set<() => void>>()

function getPulseState(id: string): boolean {
  return pulseStates.get(id)?.isPulsing ?? false
}

function subscribeToPulse(id: string, listener: () => void) {
  if (!pulseListeners.has(id)) {
    pulseListeners.set(id, new Set())
  }
  pulseListeners.get(id)!.add(listener)
  return () => {
    pulseListeners.get(id)?.delete(listener)
  }
}

function triggerPulse(id: string) {
  const state = pulseStates.get(id) ?? { isPulsing: false, timeoutId: null }
  if (state.timeoutId) clearTimeout(state.timeoutId)
  
  state.isPulsing = true
  pulseStates.set(id, state)
  pulseListeners.get(id)?.forEach(l => l())
  
  state.timeoutId = setTimeout(() => {
    state.isPulsing = false
    pulseStates.set(id, state)
    pulseListeners.get(id)?.forEach(l => l())
  }, 1200)
}

let instanceCounter = 0

export function NotificationBadge({
  count,
  variant = 'default',
  size = 'md',
  max = 99,
  showZero = false,
  pulse = false,
  className
}: NotificationBadgeProps) {
  const [announcement, setAnnouncement] = useState('')
  const prevCountRef = useRef(count)
  
  // Stable instance ID for this component
  const [instanceId] = useState(() => `badge-${++instanceCounter}`)
  
  // Create stable callbacks
  const subscribe = useCallback((listener: () => void) => subscribeToPulse(instanceId, listener), [instanceId])
  const getSnapshot = useCallback(() => getPulseState(instanceId), [instanceId])
  
  // Use external store for pulse state
  const isPulsing = useSyncExternalStore(subscribe, getSnapshot)
  
  // Handle pulse animation on count increase
  useEffect(() => {
    if (count > prevCountRef.current && count > 0) {
      triggerPulse(instanceId)
    }
    prevCountRef.current = count
  }, [count, instanceId])
  
  // Cleanup on unmount
  useEffect(() => {
    return () => {
      const state = pulseStates.get(instanceId)
      if (state?.timeoutId) clearTimeout(state.timeoutId)
      pulseStates.delete(instanceId)
      pulseListeners.delete(instanceId)
    }
  }, [instanceId])
  
  const shouldPulse = pulse || isPulsing

  // Debounced screen reader announcement
  useEffect(() => {
    if (count > 0) {
      const timer = setTimeout(() => {
        const severityText = variant === 'critical' ? 'critical ' : variant === 'warning' ? 'warning ' : ''
        setAnnouncement(`${count} ${severityText}unread notification${count === 1 ? '' : 's'}`)
      }, 2000)
      return () => clearTimeout(timer)
    }
  }, [count, variant])

  // Don't render if count is 0 and showZero is false
  if (count === 0 && !showZero) {
    return null
  }

  // Clamp negative counts to 0
  const displayCount = Math.max(0, count)
  const countText = displayCount > max ? `${max}+` : displayCount.toString()

  const sizeClasses = {
    sm: 'h-4 min-w-[16px] px-1 text-[10px]',
    md: 'h-5 min-w-[20px] px-1.5 text-[11px]',
    lg: 'h-6 min-w-[24px] px-2 text-xs'
  }

  const variantClasses = {
    default: 'bg-[hsl(210,100%,50%)] dark:bg-[hsl(210,90%,55%)] text-white',
    critical: 'bg-[hsl(0,84%,60%)] dark:bg-[hsl(0,70%,60%)] text-white',
    warning: 'bg-[hsl(45,100%,50%)] dark:bg-[hsl(45,90%,55%)] text-[hsl(45,10%,15%)] dark:text-[hsl(45,5%,10%)]'
  }

  return (
    <>
      <span
        className={cn(
          'inline-flex items-center justify-center rounded-full font-semibold leading-none transition-transform',
          sizeClasses[size],
          variantClasses[variant],
          shouldPulse && 'animate-notification-pulse',
          className
        )}
        role="status"
        aria-label={`${displayCount} unread notifications`}
        aria-live="polite"
        aria-atomic="true"
      >
        {countText}
      </span>
      
      {/* Screen reader announcement region */}
      {announcement && (
        <span className="sr-only" role="status" aria-live="polite">
          {announcement}
        </span>
      )}
    </>
  )
}
