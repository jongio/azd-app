/**
 * Toast Notification System - Simple toast notifications without external dependencies
 */

import { useState, useCallback, useEffect } from 'react'
import type { ToastNotification } from '../../../lib/errors'

let toastIdCounter = 0

function generateToastId(): string {
  return `toast-${++toastIdCounter}-${Date.now()}`
}

/**
 * Toast store (singleton)
 */
class ToastStore {
  private listeners: Set<(toasts: ToastNotification[]) => void> = new Set()
  private toasts: ToastNotification[] = []

  subscribe(listener: (toasts: ToastNotification[]) => void) {
    this.listeners.add(listener)
    return () => {
      this.listeners.delete(listener)
    }
  }

  getToasts() {
    return this.toasts
  }

  addToast(toast: Omit<ToastNotification, 'id'>) {
    const newToast: ToastNotification = {
      ...toast,
      id: generateToastId(),
      duration: toast.duration ?? 5000, // Default 5s
    }

    this.toasts = [...this.toasts, newToast]
    this.notify()

    // Auto-dismiss if duration is set
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        this.removeToast(newToast.id)
      }, newToast.duration)
    }

    return newToast.id
  }

  removeToast(id: string) {
    this.toasts = this.toasts.filter((t) => t.id !== id)
    this.notify()
  }

  clearAll() {
    this.toasts = []
    this.notify()
  }

  private notify() {
    this.listeners.forEach((listener) => listener(this.toasts))
  }
}

const toastStore = new ToastStore()

/**
 * Hook to use toasts
 */
export function useToast() {
  const [toasts, setToasts] = useState<ToastNotification[]>(toastStore.getToasts())

  useEffect(() => {
    return toastStore.subscribe(setToasts)
  }, [])

  const showToast = useCallback((toast: Omit<ToastNotification, 'id'>) => {
    return toastStore.addToast(toast)
  }, [])

  const dismissToast = useCallback((id: string) => {
    toastStore.removeToast(id)
  }, [])

  const clearAllToasts = useCallback(() => {
    toastStore.clearAll()
  }, [])

  // Convenience methods
  const success = useCallback(
    (message: string, description?: string) => {
      return showToast({ type: 'success', message, description })
    },
    [showToast]
  )

  const error = useCallback(
    (message: string, description?: string) => {
      return showToast({ type: 'error', message, description })
    },
    [showToast]
  )

  const warning = useCallback(
    (message: string, description?: string) => {
      return showToast({ type: 'warning', message, description })
    },
    [showToast]
  )

  const info = useCallback(
    (message: string, description?: string) => {
      return showToast({ type: 'info', message, description })
    },
    [showToast]
  )

  return {
    toasts,
    showToast,
    dismissToast,
    clearAllToasts,
    success,
    error,
    warning,
    info,
  }
}
