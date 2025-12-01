import { useState, useCallback } from 'react'

export function useClipboard() {
  const [copiedField, setCopiedField] = useState<string | null>(null)

  const copyToClipboard = useCallback(async (text: string, fieldName: string) => {
    await navigator.clipboard.writeText(text)
    setCopiedField(fieldName)
    setTimeout(() => setCopiedField(null), 2000)
  }, [])

  return { copiedField, copyToClipboard }
}
