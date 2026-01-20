/**
 * Hooks Configuration Modal
 * Modal dialog for configuring Azure YAML lifecycle hooks with support for:
 * - All lifecycle events (preprovision, postprovision, predeploy, postdeploy, etc.)
 * - Enable/disable toggle per hook
 * - Platform-specific overrides (Windows/POSIX)
 * - Multiple shell types
 * - Continue on error and interactive mode
 */

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { Webhook, AlertTriangle, CheckCircle2, Code, Settings2 } from 'lucide-react'
import {
  Dialog,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogContent,
  DialogFooter,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import type {
  HookFormData,
  HookConfig,
  LifecycleEvent,
  ShellType,
} from '@/lib/editor/hooks-types'
import {
  hookToFormData,
  formDataToHook,
  getDefaultHookFormData,
  getLifecycleEventMeta,
  getPlatformCoverageWarning,
  LIFECYCLE_EVENTS,
} from '@/lib/editor/hooks-types'

export interface HooksConfigModalProps {
  /** Whether modal is open */
  isOpen: boolean
  
  /** Callback to close modal */
  onClose: () => void
  
  /** Callback when hook is saved */
  onSave: (event: LifecycleEvent, hook: HookConfig | null) => void | Promise<void>
  
  /** Current lifecycle event being edited (for editing mode) */
  initialEvent?: LifecycleEvent
  
  /** Current hook configuration (for editing mode) */
  initialConfig?: HookConfig
}

const SHELL_TYPES: { value: ShellType; label: string; description: string }[] = [
  { value: 'sh', label: 'sh', description: 'Bourne shell (POSIX)' },
  { value: 'bash', label: 'bash', description: 'Bash shell' },
  { value: 'pwsh', label: 'pwsh', description: 'PowerShell Core' },
  { value: 'powershell', label: 'powershell', description: 'Windows PowerShell' },
  { value: 'cmd', label: 'cmd', description: 'Windows Command Prompt' },
]

/**
 * Hooks Configuration Modal Component
 */
export function HooksConfigModal({
  isOpen,
  onClose,
  onSave,
  initialEvent,
  initialConfig,
}: HooksConfigModalProps) {
  const [isSubmitting, setIsSubmitting] = React.useState(false)
  const [platformCoverageWarning, setPlatformCoverageWarning] = React.useState<string | null>(null)

  const isEditing = !!initialEvent && !!initialConfig

  // Initialize form with default values
  const defaultValues = React.useMemo(() => {
    if (initialEvent && initialConfig) {
      return hookToFormData(initialEvent, initialConfig)
    }
    return getDefaultHookFormData(initialEvent || 'preprovision')
  }, [initialEvent, initialConfig])

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
    reset,
  } = useForm<HookFormData>({
    defaultValues,
  })

  const selectedEvent = watch('event')
  const enabled = watch('enabled')
  const hasWindowsOverride = watch('platforms.windows')
  const hasPosixOverride = watch('platforms.posix')
  const run = watch('run')
  const windowsRun = watch('platforms.windowsRun')
  const posixRun = watch('platforms.posixRun')

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (isOpen) {
      reset(defaultValues)
    }
  }, [isOpen, defaultValues, reset])

  // Update platform coverage warning
  React.useEffect(() => {
    if (!enabled) {
      setPlatformCoverageWarning(null)
      return
    }

    const config: HookConfig = {
      run,
      windows: hasWindowsOverride && windowsRun ? { run: windowsRun } : undefined,
      posix: hasPosixOverride && posixRun ? { run: posixRun } : undefined,
    }

    const warning = getPlatformCoverageWarning(config)
    setPlatformCoverageWarning(warning)
  }, [enabled, run, hasWindowsOverride, hasPosixOverride, windowsRun, posixRun])

  // Handle form submission
  const onSubmit = async (data: HookFormData) => {
    try {
      setIsSubmitting(true)
      const config = formDataToHook(data)
      await onSave(data.event, config)
      onClose()
    } catch {
      alert('Failed to save hook. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  const eventMeta = getLifecycleEventMeta(selectedEvent)

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="3xl">
      <DialogHeader onClose={onClose}>
        <div className="flex items-center gap-2">
          <Webhook className="w-5 h-5 text-cyan-600" />
          <DialogTitle>
            {isEditing ? 'Edit Lifecycle Hook' : 'Add Lifecycle Hook'}
          </DialogTitle>
        </div>
        <DialogDescription>
          Configure custom scripts to run before or after Azure DevCenter lifecycle events
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={handleSubmit(onSubmit)}>
        <DialogContent>
          <div className="space-y-6">
            {/* Lifecycle Event Selector */}
            {!isEditing && (
              <div>
                <label
                  htmlFor="hook-event"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Lifecycle Event <span className="text-red-500">*</span>
                </label>
                <select
                  id="hook-event"
                  {...register('event', { required: 'Lifecycle event is required' })}
                  className={cn(
                    'w-full px-3 py-2 rounded-lg border',
                    'bg-white dark:bg-slate-900',
                    'border-slate-300 dark:border-slate-700',
                    'text-slate-900 dark:text-slate-100',
                    'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500',
                    errors.event && 'border-red-500'
                  )}
                >
                  {LIFECYCLE_EVENTS.map((event) => (
                    <option key={event.event} value={event.event}>
                      {event.displayName} ({event.category}) - {event.description}
                    </option>
                  ))}
                </select>
                {errors.event && (
                  <p className="mt-1 text-xs text-red-500">{errors.event.message}</p>
                )}
              </div>
            )}

            {/* Event Display (editing mode) */}
            {isEditing && (
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Lifecycle Event
                </label>
                <div className="px-4 py-3 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
                  <div className="font-semibold text-slate-900 dark:text-slate-100">
                    {eventMeta.displayName}
                  </div>
                  <div className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    {eventMeta.description}
                  </div>
                </div>
              </div>
            )}

            {/* Enable/Disable Toggle */}
            <div className="flex items-center gap-3 p-4 rounded-lg bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-700">
              <input
                id="hook-enabled"
                type="checkbox"
                {...register('enabled')}
                className="w-4 h-4 rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
              />
              <label htmlFor="hook-enabled" className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Enable this hook
              </label>
            </div>

            {/* Hook Configuration (only shown when enabled) */}
            {enabled && (
              <>
                {/* Base Configuration */}
                <div className="space-y-4 p-4 rounded-lg border border-slate-200 dark:border-slate-700">
                  <div className="flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-300">
                    <Code className="w-4 h-4" />
                    <span>Base Configuration</span>
                  </div>

                  {/* Script Command */}
                  <div>
                    <label
                      htmlFor="hook-run"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Script Command {(!hasWindowsOverride && !hasPosixOverride) && <span className="text-red-500">*</span>}
                    </label>
                    <Input
                      id="hook-run"
                      type="text"
                      {...register('run', {
                        required: (!hasWindowsOverride && !hasPosixOverride) ? 'Script command is required' : false,
                      })}
                      placeholder="./scripts/setup.sh or echo 'Hello'"
                      className={cn(errors.run && 'border-red-500')}
                      disabled={hasWindowsOverride && hasPosixOverride}
                    />
                    {errors.run ? (
                      <p className="mt-1 text-xs text-red-500">{errors.run.message}</p>
                    ) : (
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Path to script or inline command to execute
                      </p>
                    )}
                  </div>

                  {/* Shell Selection */}
                  <div>
                    <label
                      htmlFor="hook-shell"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Shell
                    </label>
                    <select
                      id="hook-shell"
                      {...register('shell')}
                      className="w-full px-3 py-2 rounded-lg border bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-cyan-500"
                      disabled={hasWindowsOverride && hasPosixOverride}
                    >
                      {SHELL_TYPES.map((shell) => (
                        <option key={shell.value} value={shell.value}>
                          {shell.label} - {shell.description}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Continue on Error */}
                  <div className="flex items-center gap-2">
                    <input
                      id="hook-continue-on-error"
                      type="checkbox"
                      {...register('continueOnError')}
                      className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                      disabled={hasWindowsOverride && hasPosixOverride}
                    />
                    <label htmlFor="hook-continue-on-error" className="text-sm text-slate-700 dark:text-slate-300">
                      Continue on error (don't fail if script exits with error)
                    </label>
                  </div>

                  {/* Interactive Mode */}
                  <div className="flex items-center gap-2">
                    <input
                      id="hook-interactive"
                      type="checkbox"
                      {...register('interactive')}
                      className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                      disabled={hasWindowsOverride && hasPosixOverride}
                    />
                    <label htmlFor="hook-interactive" className="text-sm text-slate-700 dark:text-slate-300">
                      Interactive mode (bind to stdin/stdout/stderr)
                    </label>
                  </div>
                </div>

                {/* Platform-Specific Overrides */}
                <div className="space-y-4">
                  <div className="flex items-center gap-2 text-sm font-semibold text-slate-700 dark:text-slate-300">
                    <Settings2 className="w-4 h-4" />
                    <span>Platform-Specific Overrides</span>
                  </div>

                  {/* Windows Override */}
                  <div className="space-y-3 p-4 rounded-lg border border-slate-200 dark:border-slate-700">
                    <div className="flex items-center gap-2">
                      <input
                        id="hook-windows"
                        type="checkbox"
                        {...register('platforms.windows')}
                        className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                      />
                      <label htmlFor="hook-windows" className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        Windows Override
                      </label>
                    </div>

                    {hasWindowsOverride && (
                      <div className="ml-6 space-y-3">
                        <Input
                          type="text"
                          {...register('platforms.windowsRun')}
                          placeholder=".\\scripts\\setup.ps1"
                          className="text-sm"
                        />
                        <select
                          {...register('platforms.windowsShell')}
                          className="w-full px-3 py-2 rounded-lg border bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-700 text-sm"
                        >
                          {SHELL_TYPES.filter(s => ['pwsh', 'powershell', 'cmd'].includes(s.value)).map((shell) => (
                            <option key={shell.value} value={shell.value}>
                              {shell.label}
                            </option>
                          ))}
                        </select>
                        <div className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            {...register('platforms.windowsContinueOnError')}
                            className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                          />
                          <label className="text-xs text-slate-600 dark:text-slate-400">
                            Continue on error
                          </label>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* POSIX Override */}
                  <div className="space-y-3 p-4 rounded-lg border border-slate-200 dark:border-slate-700">
                    <div className="flex items-center gap-2">
                      <input
                        id="hook-posix"
                        type="checkbox"
                        {...register('platforms.posix')}
                        className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                      />
                      <label htmlFor="hook-posix" className="text-sm font-medium text-slate-700 dark:text-slate-300">
                        POSIX Override (Linux/macOS)
                      </label>
                    </div>

                    {hasPosixOverride && (
                      <div className="ml-6 space-y-3">
                        <Input
                          type="text"
                          {...register('platforms.posixRun')}
                          placeholder="./scripts/setup.sh"
                          className="text-sm"
                        />
                        <select
                          {...register('platforms.posixShell')}
                          className="w-full px-3 py-2 rounded-lg border bg-white dark:bg-slate-900 border-slate-300 dark:border-slate-700 text-sm"
                        >
                          {SHELL_TYPES.filter(s => ['sh', 'bash'].includes(s.value)).map((shell) => (
                            <option key={shell.value} value={shell.value}>
                              {shell.label}
                            </option>
                          ))}
                        </select>
                        <div className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            {...register('platforms.posixContinueOnError')}
                            className="rounded border-slate-300 text-cyan-600 focus:ring-cyan-500"
                          />
                          <label className="text-xs text-slate-600 dark:text-slate-400">
                            Continue on error
                          </label>
                        </div>
                      </div>
                    )}
                  </div>
                </div>

                {/* Platform Coverage Warning */}
                {platformCoverageWarning && (
                  <div className="rounded-lg bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-900 p-4">
                    <div className="flex gap-2">
                      <AlertTriangle className="w-4 h-4 text-yellow-600 dark:text-yellow-400 shrink-0 mt-0.5" />
                      <p className="text-sm text-yellow-900 dark:text-yellow-100">
                        {platformCoverageWarning}
                      </p>
                    </div>
                  </div>
                )}

                {/* Success Message (cross-platform coverage) */}
                {!platformCoverageWarning && (run || (hasWindowsOverride && hasPosixOverride)) && (
                  <div className="rounded-lg bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-900 p-4">
                    <div className="flex gap-2">
                      <CheckCircle2 className="w-4 h-4 text-green-600 dark:text-green-400 shrink-0 mt-0.5" />
                      <p className="text-sm text-green-900 dark:text-green-100">
                        This hook has cross-platform coverage
                      </p>
                    </div>
                  </div>
                )}

                {/* Info Panel */}
                <div className="rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 p-4">
                  <div className="flex gap-2">
                    <Webhook className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
                    <div className="text-sm text-blue-900 dark:text-blue-100">
                      <p className="font-medium mb-1">Hook Execution Tips</p>
                      <ul className="text-xs space-y-1 text-blue-800 dark:text-blue-200">
                        <li>• Hooks run automatically at the specified lifecycle stage</li>
                        <li>• Use absolute paths or paths relative to azure.yaml</li>
                        <li>• Platform overrides allow different scripts for Windows vs Linux/macOS</li>
                        <li>• Environment variables available: AZD_APP_PROJECT_DIR, AZD_APP_PROJECT_NAME</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </DialogContent>

        <DialogFooter>
          <div className="flex items-center justify-end gap-3 w-full">
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
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className={cn(
                'px-6 py-2 rounded-lg text-sm font-semibold shadow-sm',
                'bg-cyan-600 text-white hover:bg-cyan-700',
                'focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2',
                'disabled:opacity-50 disabled:cursor-not-allowed',
                'transition-colors duration-150'
              )}
            >
              {isSubmitting ? 'Saving...' : 'Save Hook'}
            </button>
          </div>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
