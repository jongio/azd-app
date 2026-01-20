/**
 * Connection Strings Panel
 * Displays connection strings for a service after addition
 */

import * as React from 'react'
import { Copy, Check, ExternalLink } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { WellKnownService } from '@/lib/editor/wellknown-types'
import { useTimeout } from '@/hooks/useTimeout'

interface ConnectionStringsPanelProps {
  service: WellKnownService
  className?: string
}

export function ConnectionStringsPanel({ service, className }: ConnectionStringsPanelProps) {
  const { setTimeout, clearAllTimeouts } = useTimeout()
  const [copiedKey, setCopiedKey] = React.useState<string | null>(null)
  const getWriteText = React.useCallback(() => {
    const testWrite = (globalThis as any).__initialWriteText as Clipboard['writeText'] | undefined
    return testWrite ?? navigator.clipboard?.writeText ?? null
  }, [])

  const handleCopy = React.useCallback((key: string, value: string) => {
    const writeText = getWriteText()

    if (!writeText) {
      return
    }

    const copyPromise = writeText.call(navigator.clipboard, value)

    setCopiedKey(key)
    setTimeout(() => setCopiedKey(null), 2000)

    void copyPromise.catch(() => {
      // Silent failure - copy functionality is non-critical
    })
  }, [getWriteText, setTimeout])

  React.useEffect(() => clearAllTimeouts, [clearAllTimeouts])

  if (!service.connectionStrings || Object.keys(service.connectionStrings).length === 0) {
    return null
  }

  return (
    <div className={cn('space-y-4', className)}>
      {/* Header */}
      <div className="flex items-center gap-2">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          Connection Strings
        </h3>
        <span className="text-xs px-2 py-0.5 rounded-full bg-cyan-100 dark:bg-cyan-900 text-cyan-700 dark:text-cyan-300">
          {service.displayName}
        </span>
      </div>

      {/* Connection Strings List */}
      <div className="space-y-3">
        {Object.entries(service.connectionStrings).map(([key, value]) => (
          <div 
            key={key}
            className={cn(
              'bg-slate-50 dark:bg-slate-800 rounded-lg p-4',
              'border border-slate-200 dark:border-slate-700'
            )}
          >
            {/* Connection String Label */}
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-slate-700 dark:text-slate-300 capitalize">
                {key === 'default' ? 'Connection String' : `${key} Connection String`}
              </span>
              <button
                onClick={() => handleCopy(key, value)}
                className={cn(
                  'inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium',
                  'transition-colors duration-150',
                  'focus:outline-none focus:ring-2 focus:ring-cyan-500',
                  copiedKey === key
                    ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300'
                    : 'bg-slate-200 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-300 dark:hover:bg-slate-600'
                )}
                aria-label={`Copy ${key} connection string`}
              >
                {copiedKey === key ? (
                  <>
                    <Check className="w-3 h-3" aria-hidden="true" />
                    <span>Copied!</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" aria-hidden="true" />
                    <span>Copy</span>
                  </>
                )}
              </button>
            </div>

            {/* Connection String Value */}
            <div className="relative">
              <code className={cn(
                'block text-xs text-slate-900 dark:text-slate-100',
                'bg-white dark:bg-slate-900 rounded p-2',
                'border border-slate-200 dark:border-slate-700',
                'break-all font-mono',
                'max-h-24 overflow-y-auto'
              )}>
                {value}
              </code>
            </div>
          </div>
        ))}
      </div>

      {/* Documentation Link */}
      {service.docsUrl && (
        <div className="pt-2">
          <a
            href={service.docsUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              'inline-flex items-center gap-2 text-sm text-cyan-600 dark:text-cyan-400',
              'hover:underline focus:outline-none focus:ring-2 focus:ring-cyan-500 rounded'
            )}
          >
            <ExternalLink className="w-4 h-4" aria-hidden="true" />
            <span>View {service.displayName} Documentation</span>
          </a>
        </div>
      )}

      {/* Usage Hint */}
      <div className={cn(
        'bg-blue-50 dark:bg-blue-950/30 rounded-lg p-3',
        'border border-blue-200 dark:border-blue-800'
      )}>
        <p className="text-sm text-blue-900 dark:text-blue-100">
          <strong className="font-semibold">💡 Tip:</strong> Use these connection strings in your application's environment variables to connect to {service.displayName}.
        </p>
      </div>
    </div>
  )
}
