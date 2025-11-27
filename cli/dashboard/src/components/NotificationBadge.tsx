import { useEffect, useState, useRef } from 'react'
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

export function NotificationBadge({
  count,
  variant = 'default',
  size = 'md',
  max = 99,
  showZero = false,
  pulse = false,
  className
}: NotificationBadgeProps) {
  const [shouldPulse, setShouldPulse] = useState(pulse)
  const [announcement, setAnnouncement] = useState('')
  const prevCountRef = useRef(count)

  // Handle pulse animation on count increase
  useEffect(() => {
    if (count > prevCountRef.current && count > 0) {
      setShouldPulse(true)
      const timer = setTimeout(() => setShouldPulse(false), 1200) // 2 iterations × 600ms
      return () => clearTimeout(timer)
    }
    prevCountRef.current = count
  }, [count])

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
