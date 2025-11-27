import * as React from "react"

export function useToast() {
  const [toasts, setToasts] = React.useState<Array<{ id: number; message: string; type: 'success' | 'error' | 'info' }>>([])
  const idRef = React.useRef(0)

  const showToast = React.useCallback((message: string, type: 'success' | 'error' | 'info' = 'success') => {
    const id = idRef.current++
    setToasts(prev => [...prev, { id, message, type }])
  }, [])

  const removeToast = React.useCallback((id: number) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }, [])

  return {
    toasts,
    showToast,
    removeToast
  }
}
