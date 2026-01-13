import { useState, useCallback } from 'react'
import { useTimeout } from './useTimeout'

export function useClipboard() {
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const { setTimeout } = useTimeout()

  const copyToClipboard = useCallback(async (text: string, fieldName: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedField(fieldName)
    setTimeout(() => setCopiedField(null), 2000)
  }, [setTimeout])

  return { copiedField, copyToClipboard }
}
