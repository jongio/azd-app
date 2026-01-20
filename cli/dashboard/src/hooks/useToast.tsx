/**
 * Toast hook - Custom hook for displaying toast notifications
 * Separated from Toast component for Fast Refresh compatibility
 */

import * as React from "react"
import { Toast } from "@/components/ui/toast"

interface ToastState {
  message: string
  type: 'success' | 'error' | 'info'
}

export function useToast() {
  const [toast, setToast] = React.useState<ToastState | null>(null)

  const showToast = React.useCallback((message: string, type: 'success' | 'error' | 'info' = 'info') => {
    setToast({ message, type })
  }, [])

  const hideToast = React.useCallback(() => {
    setToast(null)
  }, [])

  const ToastContainer = React.useCallback(() => {
    if (!toast) return null
    return <Toast message={toast.message} type={toast.type} onClose={hideToast} />
  }, [toast, hideToast])

  return { showToast, ToastContainer }
}
