import { useState, useEffect, useCallback, useRef } from 'react'
import type { Notification } from '@/components/NotificationStack'
import type { NotificationHistoryItem } from '@/components/NotificationCenter'
import { getStorageItem, setStorageItem, removeStorageItem } from '@/lib/storage-utils'

const STORAGE_KEY = 'azd-notification-history'
const MAX_HISTORY = 100

/** Type guard for notification history items */
function isNotificationHistoryArray(value: unknown): value is Array<Omit<NotificationHistoryItem, 'timestamp'> & { timestamp: string }> {
  if (!Array.isArray(value)) return false
  return value.every(item => {
    if (typeof item !== 'object' || item === null) return false
    const obj = item as Record<string, unknown>
    return (
      typeof obj.id === 'string' &&
      typeof obj.serviceName === 'string' &&
      typeof obj.message === 'string' &&
      typeof obj.timestamp === 'string'
    )
  })
}

/** Load history from localStorage */
function loadHistoryFromStorage(): NotificationHistoryItem[] {
  const stored = getStorageItem<Array<Omit<NotificationHistoryItem, 'timestamp'> & { timestamp: string }>>(
    STORAGE_KEY, 
    [], 
    isNotificationHistoryArray
  )
  return stored.map((item) => ({
    ...item,
    timestamp: new Date(item.timestamp)
  }))
}

export function useNotifications() {
  const [toastNotifications, setToastNotifications] = useState<Notification[]>([])
  // Initialize history from localStorage using lazy initializer
  const [history, setHistory] = useState<NotificationHistoryItem[]>(loadHistoryFromStorage)
  const [isCenterOpen, setIsCenterOpen] = useState(false)
  
  // Track pending dismiss timeouts for cleanup
  const dismissTimeoutsRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map())

  // Cleanup all pending timeouts on unmount
  useEffect(() => {
    const timeouts = dismissTimeoutsRef.current
    return () => {
      timeouts.forEach((timeout) => clearTimeout(timeout))
      timeouts.clear()
    }
  }, [])

  // Save history to localStorage
  useEffect(() => {
    if (history.length > 0) {
      setStorageItem(STORAGE_KEY, history.slice(0, MAX_HISTORY))
    }
  }, [history])

  const addNotification = useCallback((notification: Omit<Notification, 'id'>) => {
    const id = `notif-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`
    const fullNotification: Notification = {
      ...notification,
      id
    }

    // Add to toast stack
    setToastNotifications(prev => [...prev, fullNotification])

    // Add to history
    const historyItem: NotificationHistoryItem = {
      id,
      serviceName: notification.title,
      message: notification.message,
      severity: notification.severity,
      timestamp: notification.timestamp,
      read: false,
      acknowledged: false
    }
    setHistory(prev => [historyItem, ...prev])

    return id
  }, [])

  const dismissToast = useCallback((id: string) => {
    // Clear any existing timeout for this notification
    const existingTimeout = dismissTimeoutsRef.current.get(id)
    if (existingTimeout) {
      clearTimeout(existingTimeout)
      dismissTimeoutsRef.current.delete(id)
    }

    setToastNotifications(prev => prev.map(n => 
      n.id === id ? { ...n, dismissed: true } : n
    ))
    
    const timeout = setTimeout(() => {
      setToastNotifications(prev => prev.filter(n => n.id !== id))
      dismissTimeoutsRef.current.delete(id)
    }, 300)
    
    dismissTimeoutsRef.current.set(id, timeout)
  }, [])

  const markAsRead = useCallback((id: string) => {
    setHistory(prev => prev.map(item =>
      item.id === id ? { ...item, read: true } : item
    ))
  }, [])

  const markAllAsRead = useCallback(() => {
    setHistory(prev => prev.map(item => ({ ...item, read: true })))
  }, [])

  const clearAll = useCallback(() => {
    // Clear without browser confirm - let the UI handle confirmation dialogs
    setHistory([])
    removeStorageItem(STORAGE_KEY)
  }, [])

  const handleNotificationClick = useCallback((id: string) => {
    const notification = history.find(n => n.id === id)
    if (notification) {
      markAsRead(id)
      // Navigate to service - will be implemented with router
      // TODO: Implement navigation to service
    }
  }, [history, markAsRead])

  const unreadCount = history.filter(n => !n.read).length

  return {
    toastNotifications,
    history,
    isCenterOpen,
    unreadCount,
    addNotification,
    dismissToast,
    markAsRead,
    markAllAsRead,
    clearAll,
    handleNotificationClick,
    openCenter: () => setIsCenterOpen(true),
    closeCenter: () => setIsCenterOpen(false),
    toggleCenter: () => setIsCenterOpen(prev => !prev)
  }
}
