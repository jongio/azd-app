/**
 * Export Modal Component
 * Modal dialog for exporting configurations in various formats
 */

import * as React from 'react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { stringifyYaml } from '@/lib/editor/yaml-utils'
import { convertToTemplate, detectSecurityWarnings, downloadFile, copyToClipboard } from '@/lib/editor/import-export-utils'
import type { ExportFormat, ExportOptions, SecurityWarning } from '@/lib/editor/import-export-types'
import { Download, Copy, Check, AlertTriangle, FileText, FileJson, FileCode } from 'lucide-react'

export interface ExportModalProps {
  /** Whether modal is open */
  isOpen: boolean
  /** Callback to close modal */
  onClose: () => void
  /** Configuration to export */
  config: Record<string, unknown>
}

/**
 * Export Modal Component
 */
export function ExportModal({
  isOpen,
  onClose,
  config,
}: ExportModalProps) {
  const [format, setFormat] = React.useState<ExportFormat>('yaml')
  const [options, setOptions] = React.useState<ExportOptions>({
    format: 'yaml',
    includeComments: true,
    minify: false,
    includeSecrets: false,
    templateMode: false,
  })
  const [showSecurityWarning, setShowSecurityWarning] = React.useState(false)
  const [securityWarnings, setSecurityWarnings] = React.useState<SecurityWarning[]>([])
  const [copied, setCopied] = React.useState(false)

  // Reset state when modal closes
  React.useEffect(() => {
    if (!isOpen) {
      setFormat('yaml')
      setOptions({
        format: 'yaml',
        includeComments: true,
        minify: false,
        includeSecrets: false,
        templateMode: false,
      })
      setShowSecurityWarning(false)
      setSecurityWarnings([])
      setCopied(false)
    }
  }, [isOpen])

  // Update format in options when format changes
  React.useEffect(() => {
    setOptions(prev => ({ ...prev, format }))
  }, [format])

  // Detect security warnings when options change
  React.useEffect(() => {
    const warnings = detectSecurityWarnings(config, options.includeSecrets || false)
    setSecurityWarnings(warnings)
  }, [config, options.includeSecrets])

  // Generate export content
  const exportContent = React.useMemo(() => {
    let processedConfig = { ...config }

    // Convert to template if requested
    if (options.templateMode) {
      processedConfig = convertToTemplate(processedConfig) as Record<string, unknown>
    }

    // Generate content based on format
    switch (format) {
      case 'yaml':
        return stringifyYaml(processedConfig, {
          indent: options.minify ? 0 : 2,
          sortKeys: false,
        })

      case 'json':
        return options.minify
          ? JSON.stringify(processedConfig)
          : JSON.stringify(processedConfig, null, 2)

      case 'template':
        return stringifyYaml(convertToTemplate(processedConfig), {
          indent: 2,
          sortKeys: false,
        })

      default:
        return ''
    }
  }, [config, format, options])

  // Get file extension
  const getFileExtension = (): string => {
    switch (format) {
      case 'yaml':
      case 'template':
        return 'yaml'
      case 'json':
        return 'json'
      default:
        return 'txt'
    }
  }

  // Get MIME type
  const getMimeType = (): string => {
    switch (format) {
      case 'yaml':
      case 'template':
        return 'text/yaml'
      case 'json':
        return 'application/json'
      default:
        return 'text/plain'
    }
  }

  // Handle download
  const handleDownload = () => {
    // Show security warning if including secrets
    if (options.includeSecrets && securityWarnings.some(w => w.requiresConfirmation)) {
      setShowSecurityWarning(true)
      return
    }

    const filename = `azure.${options.templateMode ? 'template' : 'yaml'}.${getFileExtension()}`
    downloadFile(exportContent, filename, getMimeType())
  }

  // Handle copy to clipboard
  const handleCopy = async () => {
    const success = await copyToClipboard(exportContent)
    if (success) {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  // Handle security warning confirmation
  const handleSecurityConfirm = () => {
    setShowSecurityWarning(false)
    const filename = `azure.${options.templateMode ? 'template' : 'yaml'}.${getFileExtension()}`
    downloadFile(exportContent, filename, getMimeType())
  }

  return (
    <>
      <Dialog isOpen={isOpen && !showSecurityWarning} onClose={onClose} maxWidth="3xl">
        <DialogHeader onClose={onClose}>
          <DialogTitle>Export Configuration</DialogTitle>
          <DialogDescription>
            Export your azure.yaml configuration in various formats
          </DialogDescription>
        </DialogHeader>

        <DialogContent>
          <div className="space-y-6">
            {/* Format Selection */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">
                Format
              </label>
              <div className="grid grid-cols-3 gap-3">
                {/* YAML Format */}
                <button
                  type="button"
                  onClick={() => setFormat('yaml')}
                  className={cn(
                    'p-4 rounded-lg border-2 text-left transition-all duration-150',
                    format === 'yaml'
                      ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                      : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                  )}
                >
                  <div className="flex items-center gap-3 mb-2">
                    <FileText className={cn('w-5 h-5', format === 'yaml' ? 'text-cyan-600' : 'text-slate-500')} />
                    <span className={cn('font-semibold text-sm', format === 'yaml' ? 'text-cyan-900 dark:text-cyan-100' : 'text-slate-900 dark:text-slate-100')}>
                      YAML
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">
                    Standard azure.yaml format
                  </p>
                </button>

                {/* JSON Format */}
                <button
                  type="button"
                  onClick={() => setFormat('json')}
                  className={cn(
                    'p-4 rounded-lg border-2 text-left transition-all duration-150',
                    format === 'json'
                      ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                      : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                  )}
                >
                  <div className="flex items-center gap-3 mb-2">
                    <FileJson className={cn('w-5 h-5', format === 'json' ? 'text-cyan-600' : 'text-slate-500')} />
                    <span className={cn('font-semibold text-sm', format === 'json' ? 'text-cyan-900 dark:text-cyan-100' : 'text-slate-900 dark:text-slate-100')}>
                      JSON
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">
                    JSON representation
                  </p>
                </button>

                {/* Template Format */}
                <button
                  type="button"
                  onClick={() => setFormat('template')}
                  className={cn(
                    'p-4 rounded-lg border-2 text-left transition-all duration-150',
                    format === 'template'
                      ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-900/20'
                      : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                  )}
                >
                  <div className="flex items-center gap-3 mb-2">
                    <FileCode className={cn('w-5 h-5', format === 'template' ? 'text-cyan-600' : 'text-slate-500')} />
                    <span className={cn('font-semibold text-sm', format === 'template' ? 'text-cyan-900 dark:text-cyan-100' : 'text-slate-900 dark:text-slate-100')}>
                      Template
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 dark:text-slate-400">
                    Reusable template
                  </p>
                </button>
              </div>
            </div>

            {/* Export Options */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 dark:text-slate-100 mb-3">
                Options
              </label>
              <div className="space-y-3">
                {/* Include Comments (YAML only) */}
                {format === 'yaml' && (
                  <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors">
                    <input
                      type="checkbox"
                      checked={options.includeComments}
                      onChange={(e) => setOptions(prev => ({ ...prev, includeComments: e.target.checked }))}
                      className="w-4 h-4 text-cyan-600 border-slate-300 rounded focus:ring-cyan-500"
                    />
                    <div className="flex-1">
                      <div className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        Include comments
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">
                        Add descriptive comments to YAML output
                      </div>
                    </div>
                  </label>
                )}

                {/* Minify (JSON only) */}
                {format === 'json' && (
                  <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors">
                    <input
                      type="checkbox"
                      checked={options.minify}
                      onChange={(e) => setOptions(prev => ({ ...prev, minify: e.target.checked }))}
                      className="w-4 h-4 text-cyan-600 border-slate-300 rounded focus:ring-cyan-500"
                    />
                    <div className="flex-1">
                      <div className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        Minify
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">
                        Remove whitespace and formatting
                      </div>
                    </div>
                  </label>
                )}

                {/* Include Secrets */}
                <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors">
                  <input
                    type="checkbox"
                    checked={options.includeSecrets}
                    onChange={(e) => setOptions(prev => ({ ...prev, includeSecrets: e.target.checked }))}
                    className="w-4 h-4 text-amber-600 border-slate-300 rounded focus:ring-amber-500"
                  />
                  <div className="flex-1">
                    <div className="text-sm font-medium text-slate-900 dark:text-slate-100 flex items-center gap-2">
                      Include secrets
                      {securityWarnings.length > 0 && (
                        <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400" />
                      )}
                    </div>
                    <div className="text-xs text-slate-600 dark:text-slate-400">
                      Export secret values (security risk)
                    </div>
                  </div>
                </label>

                {/* Template Mode */}
                {format !== 'template' && (
                  <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors">
                    <input
                      type="checkbox"
                      checked={options.templateMode}
                      onChange={(e) => setOptions(prev => ({ ...prev, templateMode: e.target.checked }))}
                      className="w-4 h-4 text-cyan-600 border-slate-300 rounded focus:ring-cyan-500"
                    />
                    <div className="flex-1">
                      <div className="text-sm font-medium text-slate-900 dark:text-slate-100">
                        Template mode
                      </div>
                      <div className="text-xs text-slate-600 dark:text-slate-400">
                        Replace values with placeholders (e.g., $&#123;VALUE&#125;)
                      </div>
                    </div>
                  </label>
                )}
              </div>
            </div>

            {/* Security Warnings */}
            {securityWarnings.length > 0 && options.includeSecrets && (
              <div className="p-4 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
                  <div>
                    <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100 mb-2">
                      Security Warning
                    </h4>
                    {securityWarnings.map((warning, i) => (
                      <p key={i} className="text-sm text-amber-700 dark:text-amber-300">
                        {warning.message}
                      </p>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Preview */}
            <div>
              <label className="block text-sm font-semibold text-slate-900 dark:text-slate-100 mb-2">
                Preview
              </label>
              <div className="relative">
                <pre className="p-4 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-xs font-mono overflow-x-auto max-h-64 text-slate-900 dark:text-slate-100">
                  {exportContent}
                </pre>
                <div className="absolute top-2 right-2 text-xs text-slate-500 dark:text-slate-400">
                  {exportContent.length} characters
                </div>
              </div>
            </div>
          </div>
        </DialogContent>

        <DialogFooter>
          <div className="flex items-center justify-end w-full gap-3">
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'transition-colors duration-150'
              )}
            >
              Close
            </button>
            <button
              type="button"
              onClick={handleCopy}
              className={cn(
                'inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold',
                'text-slate-700 dark:text-slate-300',
                'border border-slate-200 dark:border-slate-700',
                'hover:bg-slate-100 dark:hover:bg-slate-800',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'transition-colors duration-150'
              )}
            >
              {copied ? (
                <>
                  <Check className="w-4 h-4 text-green-600" />
                  Copied!
                </>
              ) : (
                <>
                  <Copy className="w-4 h-4" />
                  Copy
                </>
              )}
            </button>
            <button
              type="button"
              onClick={handleDownload}
              className={cn(
                'inline-flex items-center gap-2 px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'transition-colors duration-150'
              )}
            >
              <Download className="w-4 h-4" />
              Download
            </button>
          </div>
        </DialogFooter>
      </Dialog>

      {/* Security Warning Modal */}
      {showSecurityWarning && (
        <Dialog isOpen={showSecurityWarning} onClose={() => setShowSecurityWarning(false)} maxWidth="md">
          <DialogHeader onClose={() => setShowSecurityWarning(false)}>
            <DialogTitle>Security Warning</DialogTitle>
          </DialogHeader>

          <DialogContent>
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-6 h-6 text-amber-600 dark:text-amber-400 flex-shrink-0" />
              <div className="space-y-3">
                <p className="text-sm text-slate-700 dark:text-slate-300">
                  You are about to export a configuration that includes <strong>secret values</strong>.
                </p>
                <p className="text-sm text-slate-700 dark:text-slate-300">
                  Anyone with access to the exported file will be able to read these secrets. Make sure to:
                </p>
                <ul className="text-sm text-slate-700 dark:text-slate-300 list-disc list-inside space-y-1">
                  <li>Store the file securely</li>
                  <li>Never commit it to version control</li>
                  <li>Share it only through secure channels</li>
                  <li>Delete it when no longer needed</li>
                </ul>
                <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                  Do you want to proceed?
                </p>
              </div>
            </div>
          </DialogContent>

          <DialogFooter>
            <div className="flex items-center justify-end w-full gap-3">
              <button
                type="button"
                onClick={() => setShowSecurityWarning(false)}
                className={cn(
                  'px-4 py-2 rounded-lg text-sm font-semibold',
                  'text-slate-700 dark:text-slate-300',
                  'border border-slate-200 dark:border-slate-700',
                  'hover:bg-slate-100 dark:hover:bg-slate-800',
                  'transition-colors duration-150'
                )}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSecurityConfirm}
                className={cn(
                  'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                  'bg-amber-600 text-white hover:bg-amber-700',
                  'transition-colors duration-150'
                )}
              >
                Export Anyway
              </button>
            </div>
          </DialogFooter>
        </Dialog>
      )}
    </>
  )
}
