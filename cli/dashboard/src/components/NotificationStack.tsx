import { useState } from 'react'
import { NotificationToast } from './NotificationToast'
import { cn } from '@/lib/utils'

export interface Notification {
  id: string
  title: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: Date
  dismissed?: boolean
}

export interface NotificationStackProps {
  notifications: Notification[]
  onDismiss: (id: string) => void
  onNotificationClick?: (id: string) => void
  maxVisible?: number
  position?: 'top-right' | 'top-center' | 'bottom-right' | 'bottom-center'
  className?: string
}

export function NotificationStack({
  notifications,
  onDismiss,
  onNotificationClick,
  maxVisible = 3,
  position = 'top-right',
  className
}: NotificationStackProps) {
  const [animatingOut, setAnimatingOut] = useState<Set<string>>(new Set())

  // Filter out dismissed and animating notifications
  const activeNotifications = notifications.filter(
    n => !n.dismissed && !animatingOut.has(n.id)
  )

  // Separate visible and queued
  const visibleNotifications = activeNotifications.slice(0, maxVisible)
  const queuedCount = Math.max(0, activeNotifications.length - maxVisible)

  const handleDismiss = (id: string) => {
    setAnimatingOut(prev => new Set([...prev, id]))
    setTimeout(() => {
      onDismiss(id)
      setAnimatingOut(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }, 300)
  }

  const positionClasses = {
    'top-right': 'top-4 right-4 sm:flex-col-reverse',
    'top-center': 'top-4 left-1/2 -translate-x-1/2 sm:flex-col-reverse',
    'bottom-right': 'bottom-4 right-4 sm:flex-col',
    'bottom-center': 'bottom-4 left-1/2 -translate-x-1/2 sm:flex-col'
  }

  return (
    <div
      role="region"
      aria-label="Notifications"
      aria-live="polite"
      aria-atomic="false"
      className={cn(
        'fixed z-[1000] flex flex-col gap-3 max-w-[420px] px-4 sm:px-0',
        positionClasses[position],
        className
      )}
    >
      {visibleNotifications.map((notification) => (
        <NotificationToast
          key={notification.id}
          id={notification.id}
          title={notification.title}
          message={notification.message}
          severity={notification.severity}
          timestamp={notification.timestamp}
          onDismiss={handleDismiss}
          onClick={onNotificationClick}
        />
      ))}

      {/* Overflow indicator */}
      {queuedCount > 0 && (
        <div
          className="text-center text-xs font-medium py-2 px-3 rounded bg-black/5 dark:bg-white/5 opacity-80"
          role="status"
          aria-live="polite"
        >
          +{queuedCount} more notification{queuedCount === 1 ? '' : 's'}
        </div>
      )}
    </div>
  )
}
