/* eslint-disable react-refresh/only-export-components */

import * as React from "react"
import { cn } from "@/lib/utils"

interface ToastProps {
  message: string
  type?: 'success' | 'error' | 'info'
  duration?: number
  onClose: () => void
}

function Toast({ message, type = 'success', duration = 3000, onClose }: ToastProps) {
  React.useEffect(() => {
    const timer = setTimeout(onClose, duration)
    return () => clearTimeout(timer)
  }, [duration, onClose])

  const bgColor = {
    success: 'bg-green-600',
    error: 'bg-red-600',
    info: 'bg-blue-600'
  }[type]

  return (
    <div 
      className={cn(
        "fixed bottom-4 right-4 px-4 py-3 rounded-lg shadow-lg text-white z-50",
        "animate-in fade-in slide-in-from-bottom-2 duration-300",
        bgColor
      )}
    >
      {message}
    </div>
  )
}

// Named export to satisfy Fast Refresh
export { Toast }

// Re-export hook from separate file
export { useToast } from '@/hooks/useToast'

