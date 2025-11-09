import { useEffect } from 'react'

/**
 * Hook to handle Escape key press for closing modals/panels
 * @param isOpen - Whether the modal/panel is currently open
 * @param onClose - Callback to close the modal/panel
 * @param respectPreventDefault - Whether to respect preventDefault on the event
 */
export function useEscapeKey(
  isOpen: boolean,
  onClose: () => void,
  respectPreventDefault = true
) {
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        if (respectPreventDefault && e.defaultPrevented) {
          return
        }
        onClose()
      }
    }
    
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [isOpen, onClose, respectPreventDefault])
}
