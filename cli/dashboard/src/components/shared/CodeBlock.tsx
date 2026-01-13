/**
 * CodeBlock - Reusable code block component with copy functionality
 * Displays syntax-highlighted code with a copy-to-clipboard button
 */
import * as React from 'react'
import { Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTimeout } from '@/hooks/useTimeout'

export interface CodeBlockProps {
  /** Code to display */
  code: string
  /** Programming language for syntax highlighting hint */
  language?: string
  /** Optional callback when code is copied */
  onCopy?: () => void
  /** Additional class names */
  className?: string
}

/**
 * CodeBlock component for displaying copyable code snippets.
 * Features:
 * - Syntax highlighting class hint
 * - Copy to clipboard with visual feedback
 * - Accessible keyboard navigation
 * - Dark mode optimized
 */
export function CodeBlock({ 
  code, 
  language = 'text', 
  onCopy,
  className 
}: Readonly<CodeBlockProps>) {
  const [copied, setCopied] = React.useState(false)
  const { setTimeout } = useTimeout()

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
    onCopy?.()
  }

  return (
    <div className={cn('relative group', className)}>
      <pre
        className={cn(
          'mt-3 p-4 rounded-lg font-mono text-xs overflow-x-auto',
          'bg-slate-900 dark:bg-black',
          'text-slate-100 dark:text-slate-200',
          'border border-slate-700 dark:border-slate-800',
        )}
      >
        <code className={`language-${language}`}>{code}</code>
      </pre>
      <button
        type="button"
        onClick={() => void handleCopy()}
        className={cn(
          'absolute top-2 right-2',
          'inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium',
          'bg-slate-800 dark:bg-slate-900 text-slate-300',
          'border border-slate-600 dark:border-slate-700',
          'hover:bg-slate-700 dark:hover:bg-slate-800',
          'opacity-0 group-hover:opacity-100',
          'transition-all duration-200',
          'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:opacity-100',
        )}
        aria-label={copied ? 'Copied' : 'Copy code'}
      >
        {copied ? (
          <>
            <Check className="w-3.5 h-3.5 text-emerald-400" />
            Copied
          </>
        ) : (
          <>
            <Copy className="w-3.5 h-3.5" />
            Copy
          </>
        )}
      </button>
    </div>
  )
}

export default CodeBlock
