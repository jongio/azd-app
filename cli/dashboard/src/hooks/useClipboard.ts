import { useState, useCallback } from 'react'

/**
 * Hook to handle copy to clipboard functionality with feedback
 * @param timeout - How long to show the "copied" feedback (default: 2000ms)
 * @returns Object with copy function and copied field state
 */
export function useClipboard(timeout = 2000) {
  const [copiedField, setCopiedField] = useState<string | null>(null)

  const copyToClipboard = useCallback(
    async (text: string, field: string) => {
      try {
        await navigator.clipboard.writeText(text)
        setCopiedField(field)
        setTimeout(() => setCopiedField(null), timeout)
      } catch (error) {
        console.error('Failed to copy to clipboard:', error)
      }
    },
    [timeout]
  )

  return { copyToClipboard, copiedField }
}
