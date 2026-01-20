/**
 * Health Check Modal
 * Modal dialog for configuring service health checks with support for different check types:
 * - http: URL-based health check (GET request)
 * - tcp: Port-based health check (TCP connection)
 * - process: Process-based check (command execution)
 * - output: Output-based check (command with output validation)
 * - none: Disable health checks
 */

import * as React from 'react'
import { useForm } from 'react-hook-form'
import { Activity, Globe, Network, Terminal, FileOutput, Ban } from 'lucide-react'
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
  HealthCheckFormData,
  HealthCheckType,
  ServiceInfo,
  HealthCheckConfig,
} from '@/lib/editor/healthcheck-types'
import {
  formDataToHealthCheck,
  healthCheckToFormData,
  getDefaultHealthCheck,
  validateDuration,
  validateUrl,
  validatePort,
} from '@/lib/editor/healthcheck-types'

export interface HealthCheckModalProps {
  /** Whether modal is open */
  isOpen: boolean
  
  /** Callback to close modal */
  onClose: () => void
  
  /** Callback when health check is saved */
  onSave: (healthCheck: HealthCheckConfig | null) => void | Promise<void>
  
  /** Current health check configuration (for editing) */
  initialConfig?: HealthCheckConfig
  
  /** Service information for default suggestions */
  serviceInfo?: ServiceInfo
}

const HEALTH_CHECK_TYPES = [
  { value: 'http', label: 'HTTP', icon: Globe, description: 'Check HTTP endpoint' },
  { value: 'tcp', label: 'TCP', icon: Network, description: 'Check TCP port' },
  { value: 'process', label: 'Process', icon: Terminal, description: 'Check process/command' },
  { value: 'output', label: 'Output', icon: FileOutput, description: 'Monitor stdout pattern' },
  { value: 'none', label: 'None', icon: Ban, description: 'Disable health checks' },
] as const

/**
 * Health Check Modal Component
 */
export function HealthCheckModal({
  isOpen,
  onClose,
  onSave,
  initialConfig,
  serviceInfo,
}: HealthCheckModalProps) {
  const [isSubmitting, setIsSubmitting] = React.useState(false)

  // Initialize form with default values
  const defaultValues = React.useMemo(() => {
    if (initialConfig) {
      return healthCheckToFormData(initialConfig)
    }
    if (serviceInfo) {
      return getDefaultHealthCheck(serviceInfo)
    }
    return {
      type: 'http' as HealthCheckType,
      url: 'http://localhost:8080/health',
      interval: '30s',
      timeout: '5s',
      retries: 3,
      startPeriod: '0s',
      startInterval: '5s',
    }
  }, [initialConfig, serviceInfo])

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
    reset,
  } = useForm<HealthCheckFormData>({
    defaultValues,
  })

  // Watch for type changes to show/hide fields
  const selectedType = watch('type')

  // Reset form when modal opens/closes
  React.useEffect(() => {
    if (isOpen) {
      reset(defaultValues)
    }
  }, [isOpen, defaultValues, reset])

  // Handle form submission
  const onSubmit = async (data: HealthCheckFormData) => {
    try {
      setIsSubmitting(true)
      const config = formDataToHealthCheck(data)
      await onSave(config)
      onClose()
    } catch {
      alert('Failed to save health check. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle type selection
  const handleTypeChange = (type: HealthCheckType) => {
    setValue('type', type)
    
    // Auto-fill defaults based on service info
    if (serviceInfo && type !== 'none') {
      const defaults = getDefaultHealthCheck({ ...serviceInfo })
      if (type === 'http' && defaults.url) {
        setValue('url', defaults.url)
      } else if (type === 'tcp' && defaults.port) {
        setValue('port', defaults.port)
      }
    }
  }

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="2xl">
      <DialogHeader onClose={onClose}>
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5 text-cyan-600" />
          <DialogTitle>Configure Health Check</DialogTitle>
        </div>
        <DialogDescription>
          Set up health monitoring for your service to ensure it's running correctly
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={handleSubmit(onSubmit)}>
        <DialogContent>
          <div className="space-y-6">
            {/* Health Check Type Selector */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Health Check Type <span className="text-red-500">*</span>
              </label>
              <div className="grid grid-cols-2 gap-3">
                {HEALTH_CHECK_TYPES.map((type) => {
                  const Icon = type.icon
                  const isSelected = selectedType === type.value
                  
                  return (
                    <button
                      key={type.value}
                      type="button"
                      onClick={() => handleTypeChange(type.value as HealthCheckType)}
                      className={cn(
                        'flex items-start gap-3 p-4 rounded-lg border-2 transition-all',
                        'text-left hover:shadow-md',
                        isSelected
                          ? 'border-cyan-500 bg-cyan-50 dark:bg-cyan-950/20'
                          : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
                      )}
                    >
                      <Icon
                        className={cn(
                          'w-5 h-5 mt-0.5 shrink-0',
                          isSelected ? 'text-cyan-600' : 'text-slate-400'
                        )}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="font-semibold text-sm text-slate-900 dark:text-slate-100">
                          {type.label}
                        </div>
                        <div className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                          {type.description}
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Type-Specific Fields */}
            {selectedType === 'http' && (
              <div>
                <label
                  htmlFor="health-check-url"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Health Check URL <span className="text-red-500">*</span>
                </label>
                <Input
                  id="health-check-url"
                  type="text"
                  {...register('url', {
                    required: 'URL is required for HTTP health checks',
                    validate: (value) =>
                      !value || validateUrl(value) || 'Invalid URL format (must start with http:// or https://)',
                  })}
                  placeholder="http://localhost:8080/health"
                  className={cn(errors.url && 'border-red-500')}
                  aria-invalid={errors.url ? 'true' : 'false'}
                  aria-describedby={errors.url ? 'health-check-url-error' : undefined}
                />
                {errors.url && (
                  <p id="health-check-url-error" className="mt-1 text-xs text-red-500">
                    {errors.url.message}
                  </p>
                )}
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Common endpoints: /health, /healthz, /api/health, /actuator/health
                </p>
              </div>
            )}

            {selectedType === 'tcp' && (
              <div>
                <label
                  htmlFor="health-check-port"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  TCP Port <span className="text-red-500">*</span>
                </label>
                <Input
                  id="health-check-port"
                  type="number"
                  min="1"
                  max="65535"
                  {...register('port', {
                    required: 'Port is required for TCP health checks',
                    valueAsNumber: true,
                    validate: (value) =>
                      !value || validatePort(value) || 'Port must be between 1 and 65535',
                  })}
                  placeholder="8080"
                  className={cn(errors.port && 'border-red-500')}
                  aria-invalid={errors.port ? 'true' : 'false'}
                  aria-describedby={errors.port ? 'health-check-port-error' : undefined}
                />
                {errors.port && (
                  <p id="health-check-port-error" className="mt-1 text-xs text-red-500">
                    {errors.port.message}
                  </p>
                )}
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Port number to check for TCP connectivity (1-65535)
                </p>
              </div>
            )}

            {selectedType === 'process' && (
              <div>
                <label
                  htmlFor="health-check-command"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Command <span className="text-red-500">*</span>
                </label>
                <Input
                  id="health-check-command"
                  type="text"
                  {...register('command', {
                    required: 'Command is required for process health checks',
                  })}
                  placeholder="./check-health.sh"
                  className={cn(errors.command && 'border-red-500')}
                  aria-invalid={errors.command ? 'true' : 'false'}
                  aria-describedby={errors.command ? 'health-check-command-error' : undefined}
                />
                {errors.command && (
                  <p id="health-check-command-error" className="mt-1 text-xs text-red-500">
                    {errors.command.message}
                  </p>
                )}
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Script or executable to run for health check (exit code 0 = healthy)
                </p>
              </div>
            )}

            {selectedType === 'output' && (
              <div>
                <label
                  htmlFor="health-check-pattern"
                  className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                >
                  Expected Output Pattern <span className="text-red-500">*</span>
                </label>
                <Input
                  id="health-check-pattern"
                  type="text"
                  {...register('pattern', {
                    required: 'Pattern is required for output health checks',
                  })}
                  placeholder="Server started|Listening on port"
                  className={cn(errors.pattern && 'border-red-500')}
                  aria-invalid={errors.pattern ? 'true' : 'false'}
                  aria-describedby={errors.pattern ? 'health-check-pattern-error' : undefined}
                />
                {errors.pattern && (
                  <p id="health-check-pattern-error" className="mt-1 text-xs text-red-500">
                    {errors.pattern.message}
                  </p>
                )}
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  Regex pattern to match in stdout (useful for watch mode services)
                </p>
              </div>
            )}

            {/* Duration Fields (for all types except none) */}
            {selectedType !== 'none' && (
              <>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label
                      htmlFor="health-check-interval"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Interval
                    </label>
                    <Input
                      id="health-check-interval"
                      type="text"
                      {...register('interval', {
                        validate: (value) =>
                          !value || validateDuration(value) || 'Invalid duration format (use: 30s, 1m, 2h)',
                      })}
                      placeholder="30s"
                      className={cn(errors.interval && 'border-red-500')}
                      aria-invalid={errors.interval ? 'true' : 'false'}
                      aria-describedby={errors.interval ? 'health-check-interval-error' : 'health-check-interval-help'}
                    />
                    {errors.interval ? (
                      <p id="health-check-interval-error" className="mt-1 text-xs text-red-500">
                        {errors.interval.message}
                      </p>
                    ) : (
                      <p id="health-check-interval-help" className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Time between checks (e.g., 30s, 1m)
                      </p>
                    )}
                  </div>

                  <div>
                    <label
                      htmlFor="health-check-timeout"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Timeout
                    </label>
                    <Input
                      id="health-check-timeout"
                      type="text"
                      {...register('timeout', {
                        validate: (value) =>
                          !value || validateDuration(value) || 'Invalid duration format (use: 5s, 10s, 1m)',
                      })}
                      placeholder="5s"
                      className={cn(errors.timeout && 'border-red-500')}
                      aria-invalid={errors.timeout ? 'true' : 'false'}
                      aria-describedby={errors.timeout ? 'health-check-timeout-error' : 'health-check-timeout-help'}
                    />
                    {errors.timeout ? (
                      <p id="health-check-timeout-error" className="mt-1 text-xs text-red-500">
                        {errors.timeout.message}
                      </p>
                    ) : (
                      <p id="health-check-timeout-help" className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Max wait time (e.g., 5s, 10s)
                      </p>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label
                      htmlFor="health-check-retries"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Retries
                    </label>
                    <Input
                      id="health-check-retries"
                      type="number"
                      min="1"
                      max="10"
                      {...register('retries', {
                        valueAsNumber: true,
                        min: { value: 1, message: 'Must be at least 1' },
                        max: { value: 10, message: 'Must be at most 10' },
                      })}
                      placeholder="3"
                      className={cn(errors.retries && 'border-red-500')}
                    />
                    {errors.retries && (
                      <p className="mt-1 text-xs text-red-500">{errors.retries.message}</p>
                    )}
                  </div>

                  <div>
                    <label
                      htmlFor="health-check-start-period"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Start Period
                    </label>
                    <Input
                      id="health-check-start-period"
                      type="text"
                      {...register('startPeriod', {
                        validate: (value) =>
                          !value || validateDuration(value) || 'Invalid duration format',
                      })}
                      placeholder="0s"
                      className={cn(errors.startPeriod && 'border-red-500')}
                    />
                    {errors.startPeriod && (
                      <p className="mt-1 text-xs text-red-500">{errors.startPeriod.message}</p>
                    )}
                  </div>

                  <div>
                    <label
                      htmlFor="health-check-start-interval"
                      className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1"
                    >
                      Start Interval
                    </label>
                    <Input
                      id="health-check-start-interval"
                      type="text"
                      {...register('startInterval', {
                        validate: (value) =>
                          !value || validateDuration(value) || 'Invalid duration format',
                      })}
                      placeholder="5s"
                      className={cn(errors.startInterval && 'border-red-500')}
                    />
                    {errors.startInterval && (
                      <p className="mt-1 text-xs text-red-500">{errors.startInterval.message}</p>
                    )}
                  </div>
                </div>
              </>
            )}

            {/* Info Panel */}
            {selectedType !== 'none' && (
              <div className="rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 p-4">
                <div className="flex gap-2">
                  <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
                  <div className="text-sm text-blue-900 dark:text-blue-100">
                    <p className="font-medium mb-1">Health Check Tips</p>
                    <ul className="text-xs space-y-1 text-blue-800 dark:text-blue-200">
                      <li>• Interval: How often to check (default: 30s)</li>
                      <li>• Timeout: Max wait for response (default: 5s)</li>
                      <li>• Retries: Failures before unhealthy (default: 3)</li>
                      <li>• Start Period: Grace period on startup (default: 0s)</li>
                    </ul>
                  </div>
                </div>
              </div>
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
              {isSubmitting ? 'Saving...' : 'Save Health Check'}
            </button>
          </div>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
