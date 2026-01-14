/**
 * Paste YAML Tab Component
 * Allows pasting YAML content directly
 */

import * as React from 'react'
import { cn } from '@/lib/utils'
import { ClipboardPaste } from 'lucide-react'

export interface PasteYamlTabProps {
  value: string
  onChange: (value: string) => void
}

/**
 * Paste YAML Tab Component
 */
export function PasteYamlTab({ value, onChange }: PasteYamlTabProps) {
  const [lineCount, setLineCount] = React.useState(0)

  React.useEffect(() => {
    setLineCount(value.split('\n').length)
  }, [value])

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText()
      onChange(text)
    } catch (error) {
      console.error('Failed to read clipboard:', error)
      alert('Failed to read clipboard. Please paste manually.')
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          Paste your azure.yaml content below
        </p>
        <button
          type="button"
          onClick={handlePaste}
          className={cn(
            'inline-flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-semibold',
            'text-cyan-700 dark:text-cyan-300',
            'border border-cyan-200 dark:border-cyan-700',
            'hover:bg-cyan-50 dark:hover:bg-cyan-900/20',
            'transition-colors duration-150'
          )}
        >
          <ClipboardPaste className="w-3.5 h-3.5" />
          Paste from Clipboard
        </button>
      </div>

      <div className="relative">
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="name: my-app&#10;services:&#10;  api:&#10;    host: containerapp&#10;    language: node&#10;    project: ./src/api"
          className={cn(
            'w-full h-80 p-4 rounded-lg border font-mono text-sm',
            'bg-white dark:bg-slate-900',
            'border-slate-300 dark:border-slate-700',
            'text-slate-900 dark:text-slate-100',
            'placeholder:text-slate-400',
            'focus:outline-none focus:ring-2 focus:ring-cyan-500',
            'resize-none',
            'transition-colors duration-150'
          )}
        />

        {/* Line Count */}
        <div className="absolute bottom-2 right-2 px-2 py-1 rounded bg-slate-200 dark:bg-slate-700 text-xs text-slate-600 dark:text-slate-400">
          {lineCount} lines
        </div>
      </div>

      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          className={cn(
            'text-sm text-red-600 dark:text-red-400 hover:underline font-medium'
          )}
        >
          Clear
        </button>
      )}
    </div>
  )
}
