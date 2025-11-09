import { Copy, Check, ExternalLink } from 'lucide-react'
import { useState } from 'react'

interface InfoFieldProps {
  label: string
  value: string | number | undefined
  copyable?: boolean
  link?: boolean
  className?: string
}

/**
 * Reusable info field component for displaying key-value pairs with optional copy and link functionality
 */
export function InfoField({ label, value, copyable = false, link = false, className = '' }: InfoFieldProps) {
  const [copied, setCopied] = useState(false)

  if (!value) return null

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(String(value))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (error) {
      console.error('Failed to copy:', error)
    }
  }

  const displayValue = String(value)
  const isUrl = link || (typeof value === 'string' && value.startsWith('http'))

  return (
    <div className={`flex items-start justify-between gap-2 ${className}`}>
      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground mb-0.5">{label}</p>
        {isUrl ? (
          <a
            href={displayValue}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-primary hover:underline flex items-center gap-1 group"
          >
            <span className="truncate">{displayValue}</span>
            <ExternalLink className="w-3 h-3 shrink-0 opacity-50 group-hover:opacity-100 transition-opacity" />
          </a>
        ) : (
          <p className="text-sm text-foreground font-mono break-all">{displayValue}</p>
        )}
      </div>
      {copyable && (
        <button
          onClick={handleCopy}
          className="shrink-0 p-1.5 hover:bg-white/5 rounded-md transition-colors"
          aria-label={`Copy ${label}`}
          title={`Copy ${label}`}
        >
          {copied ? (
            <Check className="w-3.5 h-3.5 text-success" />
          ) : (
            <Copy className="w-3.5 h-3.5 text-muted-foreground" />
          )}
        </button>
      )}
    </div>
  )
}
